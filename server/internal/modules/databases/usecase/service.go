package usecase

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	databasesdomain "camopanel/server/internal/modules/databases/domain"
	platformaudit "camopanel/server/internal/platform/audit"
	platformdocker "camopanel/server/internal/platform/docker"
	"camopanel/server/internal/services"
)

type Repository interface {
	List(ctx context.Context, engine string) ([]databasesdomain.Instance, error)
	FindByID(ctx context.Context, instanceID string) (databasesdomain.Instance, error)
	Save(ctx context.Context, instance databasesdomain.Instance) error
}

type Runtime interface {
	platformdocker.Runtime
}

type ContainerOperator interface {
	platformdocker.ContainerOperator
}

type TemplateCatalog interface {
	Get(id string) (*services.LoadedTemplate, error)
}

type AuditRecorder interface {
	Record(ctx context.Context, entry platformaudit.Entry) error
}

type Config struct {
	BridgeNetworkName  string
	OpenRestyContainer string
	OpenRestyDataDir   string
}

type Service struct {
	repo       Repository
	runtime    Runtime
	containers ContainerOperator
	templates  TemplateCatalog
	audit      AuditRecorder
	cfg        Config
}

func NewService(repo Repository, runtime Runtime, containers ContainerOperator, templates TemplateCatalog, audit AuditRecorder, cfg Config) *Service {
	return &Service{repo: repo, runtime: runtime, containers: containers, templates: templates, audit: audit, cfg: cfg}
}

func (s *Service) ListInstances(ctx context.Context, engine string) ([]databasesdomain.InstanceView, error) {
	if engine != "" && !slices.Contains(databasesdomain.ManagedEngines, engine) {
		return nil, fmt.Errorf("不支持的数据库类型")
	}

	items, err := s.repo.List(ctx, engine)
	if err != nil {
		return nil, err
	}

	result := make([]databasesdomain.InstanceView, 0, len(items))
	for _, item := range items {
		view, err := s.instanceView(ctx, item)
		if err != nil {
			return nil, err
		}
		result = append(result, view)
	}
	return result, nil
}

func (s *Service) GetOverview(ctx context.Context, instanceID string) (databasesdomain.Overview, error) {
	instance, err := s.repo.FindByID(ctx, instanceID)
	if err != nil {
		return databasesdomain.Overview{}, err
	}

	view, err := s.instanceView(ctx, instance)
	if err != nil {
		return databasesdomain.Overview{}, err
	}

	overview := databasesdomain.Overview{Instance: view}
	if view.Runtime.Status != platformdocker.StatusRunning {
		overview.Notice = "实例当前未运行，暂时无法读取数据库内部信息。"
		return overview, nil
	}

	containerName, err := primaryProjectContainer(view.Runtime)
	if err != nil {
		return databasesdomain.Overview{}, err
	}

	switch instance.Engine {
	case databasesdomain.EngineMySQL:
		databases, accounts, err := s.loadMySQLOverview(ctx, containerName, instance.Config)
		if err != nil {
			return databasesdomain.Overview{}, err
		}
		overview.Databases = databases
		overview.Accounts = accounts
	case databasesdomain.EnginePostgres:
		databases, accounts, err := s.loadPostgresOverview(ctx, containerName, instance.Config)
		if err != nil {
			return databasesdomain.Overview{}, err
		}
		overview.Databases = databases
		overview.Accounts = accounts
	case databasesdomain.EngineRedis:
		keyspaces, configItems, summary, err := s.loadRedisOverview(ctx, containerName, instance.Config)
		if err != nil {
			return databasesdomain.Overview{}, err
		}
		overview.RedisKeyspaces = keyspaces
		overview.RedisConfig = configItems
		overview.Summary = summary
	}

	return overview, nil
}

func (s *Service) CreateDatabase(ctx context.Context, actorID, instanceID, name string) error {
	instance, containerName, err := s.runningTarget(ctx, instanceID)
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("请输入数据库名")
	}

	switch instance.Engine {
	case databasesdomain.EngineMySQL:
		err = s.execMySQL(ctx, containerName, instance.Config, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s;", mysqlIdentifier(name)))
	case databasesdomain.EnginePostgres:
		var exists string
		exists, err = s.execPostgresQuery(ctx, containerName, instance.Config, fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname = %s;", sqlString(name)))
		if err == nil && strings.TrimSpace(exists) == "" {
			_, err = s.execPostgresQuery(ctx, containerName, instance.Config, fmt.Sprintf("CREATE DATABASE %s;", postgresIdentifier(name)))
		}
	default:
		err = fmt.Errorf("当前数据库类型不支持创建数据库")
	}
	if err != nil {
		return err
	}

	s.recordAudit(ctx, actorID, "database_create", instance.ID, map[string]any{"engine": instance.Engine, "name": name})
	return nil
}

func (s *Service) DeleteDatabase(ctx context.Context, actorID, instanceID, name string) error {
	instance, containerName, err := s.runningTarget(ctx, instanceID)
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("请输入数据库名")
	}
	if isProtectedDatabaseName(instance.Engine, name) {
		return fmt.Errorf("系统数据库不允许删除")
	}

	switch instance.Engine {
	case databasesdomain.EngineMySQL:
		err = s.execMySQL(ctx, containerName, instance.Config, fmt.Sprintf("DROP DATABASE IF EXISTS %s;", mysqlIdentifier(name)))
	case databasesdomain.EnginePostgres:
		if _, err = s.execPostgresQuery(ctx, containerName, instance.Config, fmt.Sprintf("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = %s AND pid <> pg_backend_pid();", sqlString(name))); err == nil {
			_, err = s.execPostgresQuery(ctx, containerName, instance.Config, fmt.Sprintf("DROP DATABASE IF EXISTS %s;", postgresIdentifier(name)))
		}
	default:
		err = fmt.Errorf("当前数据库类型不支持删除数据库")
	}
	if err != nil {
		return err
	}

	s.recordAudit(ctx, actorID, "database_delete", instance.ID, map[string]any{"engine": instance.Engine, "name": name})
	return nil
}

func (s *Service) CreateAccount(ctx context.Context, actorID, instanceID, name, password, databaseName string) error {
	instance, containerName, err := s.runningTarget(ctx, instanceID)
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	password = strings.TrimSpace(password)
	databaseName = strings.TrimSpace(databaseName)
	if name == "" {
		return fmt.Errorf("请输入账号名")
	}
	if password == "" {
		return fmt.Errorf("请输入密码")
	}

	switch instance.Engine {
	case databasesdomain.EngineMySQL:
		if err := s.execMySQL(ctx, containerName, instance.Config, fmt.Sprintf("CREATE USER IF NOT EXISTS %s@'%%' IDENTIFIED BY %s;", sqlString(name), sqlString(password))); err != nil {
			return err
		}
		if databaseName != "" {
			if err := s.execMySQL(ctx, containerName, instance.Config, fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO %s@'%%'; FLUSH PRIVILEGES;", mysqlIdentifier(databaseName), sqlString(name))); err != nil {
				return err
			}
		}
	case databasesdomain.EnginePostgres:
		exists, err := s.execPostgresQuery(ctx, containerName, instance.Config, fmt.Sprintf("SELECT 1 FROM pg_roles WHERE rolname = %s;", sqlString(name)))
		if err != nil {
			return err
		}
		if strings.TrimSpace(exists) == "" {
			if _, err := s.execPostgresQuery(ctx, containerName, instance.Config, fmt.Sprintf("CREATE ROLE %s WITH LOGIN PASSWORD %s;", postgresIdentifier(name), sqlString(password))); err != nil {
				return err
			}
		} else if err := s.UpdateAccountPassword(ctx, actorID, instanceID, name, password); err != nil {
			return err
		}
		if databaseName != "" {
			if err := s.GrantAccount(ctx, actorID, instanceID, name, databaseName); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("当前数据库类型不支持创建账号")
	}

	s.recordAudit(ctx, actorID, "database_account_create", instance.ID, map[string]any{"engine": instance.Engine, "account_name": name, "database_name": databaseName})
	return nil
}

func (s *Service) DeleteAccount(ctx context.Context, actorID, instanceID, accountName string) error {
	instance, containerName, err := s.runningTarget(ctx, instanceID)
	if err != nil {
		return err
	}
	accountName = strings.TrimSpace(accountName)
	if accountName == "" {
		return fmt.Errorf("请输入账号名")
	}
	if strings.EqualFold(accountName, connectionFromConfig(instance.Engine, instance.Config).AdminUsername) {
		return fmt.Errorf("管理账号不允许删除")
	}

	switch instance.Engine {
	case databasesdomain.EngineMySQL:
		err = s.execMySQL(ctx, containerName, instance.Config, fmt.Sprintf("DROP USER IF EXISTS %s@'%%';", sqlString(accountName)))
	case databasesdomain.EnginePostgres:
		_, err = s.execPostgresQuery(ctx, containerName, instance.Config, fmt.Sprintf("DROP ROLE IF EXISTS %s;", postgresIdentifier(accountName)))
	default:
		err = fmt.Errorf("当前数据库类型不支持删除账号")
	}
	if err != nil {
		return err
	}

	s.recordAudit(ctx, actorID, "database_account_delete", instance.ID, map[string]any{"engine": instance.Engine, "account_name": accountName})
	return nil
}

func (s *Service) UpdateAccountPassword(ctx context.Context, actorID, instanceID, accountName, password string) error {
	instance, containerName, err := s.runningTarget(ctx, instanceID)
	if err != nil {
		return err
	}
	accountName = strings.TrimSpace(accountName)
	password = strings.TrimSpace(password)
	if accountName == "" {
		return fmt.Errorf("请输入账号名")
	}
	if password == "" {
		return fmt.Errorf("请输入密码")
	}

	switch instance.Engine {
	case databasesdomain.EngineMySQL:
		err = s.execMySQL(ctx, containerName, instance.Config, fmt.Sprintf("ALTER USER %s@'%%' IDENTIFIED BY %s;", sqlString(accountName), sqlString(password)))
	case databasesdomain.EnginePostgres:
		_, err = s.execPostgresQuery(ctx, containerName, instance.Config, fmt.Sprintf("ALTER ROLE %s WITH PASSWORD %s;", postgresIdentifier(accountName), sqlString(password)))
	default:
		err = fmt.Errorf("当前数据库类型不支持修改账号密码")
	}
	if err != nil {
		return err
	}

	s.recordAudit(ctx, actorID, "database_account_password_update", instance.ID, map[string]any{"engine": instance.Engine, "account_name": accountName})
	return nil
}

func (s *Service) GrantAccount(ctx context.Context, actorID, instanceID, accountName, databaseName string) error {
	instance, containerName, err := s.runningTarget(ctx, instanceID)
	if err != nil {
		return err
	}
	accountName = strings.TrimSpace(accountName)
	databaseName = strings.TrimSpace(databaseName)
	if accountName == "" {
		return fmt.Errorf("请输入账号名")
	}
	if databaseName == "" {
		return fmt.Errorf("请输入数据库名")
	}

	switch instance.Engine {
	case databasesdomain.EngineMySQL:
		err = s.execMySQL(ctx, containerName, instance.Config, fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO %s@'%%'; FLUSH PRIVILEGES;", mysqlIdentifier(databaseName), sqlString(accountName)))
	case databasesdomain.EnginePostgres:
		_, err = s.execPostgresQuery(ctx, containerName, instance.Config, fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s;", postgresIdentifier(databaseName), postgresIdentifier(accountName)))
	default:
		err = fmt.Errorf("当前数据库类型不支持授权")
	}
	if err != nil {
		return err
	}

	s.recordAudit(ctx, actorID, "database_grant", instance.ID, map[string]any{"engine": instance.Engine, "account_name": accountName, "database_name": databaseName})
	return nil
}

func (s *Service) UpdateRedisConfig(ctx context.Context, actorID, instanceID, key string, value any) error {
	instance, err := s.repo.FindByID(ctx, instanceID)
	if err != nil {
		return err
	}
	if instance.Engine != databasesdomain.EngineRedis {
		return fmt.Errorf("当前实例不是 Redis")
	}

	key = strings.TrimSpace(key)
	switch key {
	case "password":
		raw := strings.TrimSpace(fmt.Sprint(value))
		if raw == "" {
			return fmt.Errorf("密码不能为空")
		}
		instance.Config[key] = raw
	case "databases":
		number, err := normalizeInt(value)
		if err != nil {
			return err
		}
		if number <= 0 {
			return fmt.Errorf("逻辑库数量必须大于 0")
		}
		instance.Config[key] = number
	case "appendonly":
		boolean, err := normalizeBool(value)
		if err != nil {
			return err
		}
		instance.Config[key] = boolean
	default:
		return fmt.Errorf("不支持修改该 Redis 配置项")
	}

	if err := s.redeployInstance(ctx, instance); err != nil {
		return err
	}
	s.recordAudit(ctx, actorID, "redis_config_update", instance.ID, map[string]any{"key": key})
	return nil
}

func (s *Service) instanceView(ctx context.Context, instance databasesdomain.Instance) (databasesdomain.InstanceView, error) {
	runtimeInfo, err := s.runtime.InspectProject(ctx, instance.Name)
	if err != nil {
		if errors.Is(err, platformdocker.ErrUnavailable) {
			runtimeInfo = platformdocker.ProjectRuntime{Status: "docker_unavailable"}
		} else {
			return databasesdomain.InstanceView{}, err
		}
	}

	if runtimeInfo.Status != "" && runtimeInfo.Status != "docker_unavailable" && instance.Status != runtimeInfo.Status {
		instance.Status = runtimeInfo.Status
		instance.LastError = ""
		_ = s.repo.Save(ctx, instance)
	}

	return databasesdomain.InstanceView{
		ID:         instance.ID,
		Name:       instance.Name,
		Engine:     instance.Engine,
		Status:     instance.Status,
		LastError:  instance.LastError,
		Runtime:    runtimeInfo,
		Connection: connectionFromConfig(instance.Engine, instance.Config),
		CreatedAt:  instance.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  instance.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (s *Service) runningTarget(ctx context.Context, instanceID string) (databasesdomain.Instance, string, error) {
	instance, err := s.repo.FindByID(ctx, instanceID)
	if err != nil {
		return databasesdomain.Instance{}, "", err
	}
	runtimeInfo, err := s.runtime.InspectProject(ctx, instance.Name)
	if err != nil {
		return databasesdomain.Instance{}, "", err
	}
	if runtimeInfo.Status != platformdocker.StatusRunning {
		return databasesdomain.Instance{}, "", fmt.Errorf("实例当前未运行")
	}
	containerName, err := primaryProjectContainer(runtimeInfo)
	if err != nil {
		return databasesdomain.Instance{}, "", err
	}
	return instance, containerName, nil
}

func (s *Service) loadMySQLOverview(ctx context.Context, containerName string, config map[string]any) ([]databasesdomain.DatabaseNameItem, []databasesdomain.AccountItem, error) {
	databaseOutput, err := s.execMySQLQuery(ctx, containerName, config, "SHOW DATABASES;")
	if err != nil {
		return nil, nil, err
	}
	accountOutput, err := s.execMySQLQuery(ctx, containerName, config, "SELECT user, host FROM mysql.user ORDER BY user, host;")
	if err != nil {
		return nil, nil, err
	}

	databases := make([]databasesdomain.DatabaseNameItem, 0)
	for _, line := range splitLines(databaseOutput) {
		databases = append(databases, databasesdomain.DatabaseNameItem{Name: line})
	}

	accounts := make([]databasesdomain.AccountItem, 0)
	for _, line := range splitLines(accountOutput) {
		parts := strings.Split(line, "\t")
		item := databasesdomain.AccountItem{Name: strings.TrimSpace(parts[0])}
		if len(parts) > 1 {
			item.Host = strings.TrimSpace(parts[1])
		}
		accounts = append(accounts, item)
	}
	return databases, accounts, nil
}

func (s *Service) loadPostgresOverview(ctx context.Context, containerName string, config map[string]any) ([]databasesdomain.DatabaseNameItem, []databasesdomain.AccountItem, error) {
	databaseOutput, err := s.execPostgresQuery(ctx, containerName, config, "SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname;")
	if err != nil {
		return nil, nil, err
	}
	accountOutput, err := s.execPostgresQuery(ctx, containerName, config, "SELECT rolname, rolsuper FROM pg_roles WHERE rolcanlogin ORDER BY rolname;")
	if err != nil {
		return nil, nil, err
	}

	databases := make([]databasesdomain.DatabaseNameItem, 0)
	for _, line := range splitLines(databaseOutput) {
		databases = append(databases, databasesdomain.DatabaseNameItem{Name: line})
	}

	accounts := make([]databasesdomain.AccountItem, 0)
	for _, line := range splitLines(accountOutput) {
		parts := strings.Split(line, "|")
		item := databasesdomain.AccountItem{Name: strings.TrimSpace(parts[0])}
		if len(parts) > 1 {
			item.Superuser = strings.EqualFold(strings.TrimSpace(parts[1]), "t")
		}
		accounts = append(accounts, item)
	}
	return databases, accounts, nil
}

func (s *Service) loadRedisOverview(ctx context.Context, containerName string, config map[string]any) ([]databasesdomain.RedisKeyspaceItem, []databasesdomain.RedisConfigItem, map[string]string, error) {
	keyspaceOutput, err := s.execRedisQuery(ctx, containerName, config, "INFO", "keyspace")
	if err != nil {
		return nil, nil, nil, err
	}
	infoOutput, err := s.execRedisQuery(ctx, containerName, config, "INFO", "server")
	if err != nil {
		return nil, nil, nil, err
	}
	memoryOutput, err := s.execRedisQuery(ctx, containerName, config, "INFO", "memory")
	if err != nil {
		return nil, nil, nil, err
	}
	clientOutput, err := s.execRedisQuery(ctx, containerName, config, "INFO", "clients")
	if err != nil {
		return nil, nil, nil, err
	}

	configItems := make([]databasesdomain.RedisConfigItem, 0, 3)
	for _, key := range []string{"appendonly", "databases"} {
		value, err := s.execRedisConfigGet(ctx, containerName, config, key)
		if err != nil {
			return nil, nil, nil, err
		}
		configItems = append(configItems, databasesdomain.RedisConfigItem{Key: key, Value: value})
	}
	configItems = append(configItems, databasesdomain.RedisConfigItem{Key: "password", Value: "已托管"})

	summary := map[string]string{}
	for key, value := range parseRedisInfo(infoOutput) {
		if key == "redis_version" || key == "uptime_in_days" {
			summary[key] = value
		}
	}
	for key, value := range parseRedisInfo(memoryOutput) {
		if key == "used_memory_human" {
			summary[key] = value
		}
	}
	for key, value := range parseRedisInfo(clientOutput) {
		if key == "connected_clients" {
			summary[key] = value
		}
	}

	keyspaces := make([]databasesdomain.RedisKeyspaceItem, 0)
	for _, line := range splitLines(keyspaceOutput) {
		if !strings.HasPrefix(line, "db") {
			continue
		}
		name, meta, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		keys := 0
		for _, item := range strings.Split(meta, ",") {
			if strings.HasPrefix(item, "keys=") {
				keys, _ = strconv.Atoi(strings.TrimPrefix(item, "keys="))
			}
		}
		keyspaces = append(keyspaces, databasesdomain.RedisKeyspaceItem{Name: name, Keys: keys})
	}
	sort.Slice(keyspaces, func(i, j int) bool { return keyspaces[i].Name < keyspaces[j].Name })

	return keyspaces, configItems, summary, nil
}

func (s *Service) execMySQL(ctx context.Context, containerName string, config map[string]any, query string) error {
	_, err := s.execMySQLQuery(ctx, containerName, config, query)
	return err
}

func (s *Service) execMySQLQuery(ctx context.Context, containerName string, config map[string]any, query string) (string, error) {
	password := strings.TrimSpace(fmt.Sprint(config["root_password"]))
	if password == "" {
		return "", fmt.Errorf("缺少 MySQL root 密码")
	}
	command := fmt.Sprintf("MYSQL_PWD=%s mysql -uroot --batch --raw --skip-column-names -e %s", shellQuote(password), shellQuote(query))
	return s.containers.ExecInContainer(ctx, containerName, "sh", "-lc", command)
}

func (s *Service) execPostgresQuery(ctx context.Context, containerName string, config map[string]any, query string) (string, error) {
	username := strings.TrimSpace(fmt.Sprint(config["username"]))
	password := strings.TrimSpace(fmt.Sprint(config["password"]))
	if username == "" || password == "" {
		return "", fmt.Errorf("缺少 PostgreSQL 管理账号")
	}
	command := fmt.Sprintf("PGPASSWORD=%s psql -U %s -d postgres -t -A -c %s", shellQuote(password), shellQuote(username), shellQuote(query))
	return s.containers.ExecInContainer(ctx, containerName, "sh", "-lc", command)
}

func (s *Service) execRedisQuery(ctx context.Context, containerName string, config map[string]any, args ...string) (string, error) {
	password := strings.TrimSpace(fmt.Sprint(config["password"]))
	if password == "" {
		return "", fmt.Errorf("缺少 Redis 密码")
	}
	commandArgs := []string{"redis-cli", "--no-auth-warning", "-a", password}
	commandArgs = append(commandArgs, args...)
	return s.containers.ExecInContainer(ctx, containerName, commandArgs...)
}

func (s *Service) execRedisConfigGet(ctx context.Context, containerName string, config map[string]any, key string) (string, error) {
	output, err := s.execRedisQuery(ctx, containerName, config, "--raw", "CONFIG", "GET", key)
	if err != nil {
		return "", err
	}
	lines := splitLines(output)
	if len(lines) >= 2 {
		return lines[1], nil
	}
	return "", nil
}

func (s *Service) redeployInstance(ctx context.Context, instance databasesdomain.Instance) error {
	templateItem, err := s.templates.Get(instance.Engine)
	if err != nil {
		return err
	}

	normalized, err := templateItem.ValidateAndNormalize(instance.Config)
	if err != nil {
		return err
	}
	rendered, err := templateItem.Render(normalized, services.TemplateRuntime{
		ProjectName:          instance.Name,
		BridgeNetworkName:    s.cfg.BridgeNetworkName,
		OpenRestyContainer:   s.cfg.OpenRestyContainer,
		OpenRestyHostConfDir: filepathJoin(s.cfg.OpenRestyDataDir, "conf.d"),
		OpenRestyHostSiteDir: filepathJoin(s.cfg.OpenRestyDataDir, "www"),
		OpenRestyHostCertDir: filepathJoin(s.cfg.OpenRestyDataDir, "certs"),
	})
	if err != nil {
		return err
	}
	if err := os.WriteFile(instance.ComposePath, []byte(rendered), 0o644); err != nil {
		return err
	}
	if err := s.runtime.EnsureNetwork(ctx, s.cfg.BridgeNetworkName, "bridge"); err != nil {
		instance.LastError = err.Error()
		_ = s.repo.Save(ctx, instance)
		return err
	}
	if err := s.runtime.Redeploy(ctx, instance.Name, instance.ComposePath); err != nil {
		instance.LastError = err.Error()
		_ = s.repo.Save(ctx, instance)
		return err
	}
	instance.Config = normalized
	instance.Status = platformdocker.StatusRunning
	instance.LastError = ""
	return s.repo.Save(ctx, instance)
}

func (s *Service) recordAudit(ctx context.Context, actorID, action, instanceID string, metadata map[string]any) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Record(ctx, platformaudit.Entry{
		ActorID:    actorID,
		Action:     action,
		TargetType: "project",
		TargetID:   instanceID,
		Metadata:   metadata,
	})
}

func connectionFromConfig(engine string, config map[string]any) databasesdomain.ConnectionInfo {
	info := databasesdomain.ConnectionInfo{
		Host:            "127.0.0.1",
		Port:            configInt(config["port"]),
		PasswordManaged: true,
	}
	switch engine {
	case databasesdomain.EngineMySQL:
		info.AdminUsername = "root"
		info.AppUsername = strings.TrimSpace(fmt.Sprint(config["username"]))
		info.DefaultDatabase = strings.TrimSpace(fmt.Sprint(config["database"]))
	case databasesdomain.EnginePostgres:
		info.AdminUsername = strings.TrimSpace(fmt.Sprint(config["username"]))
		info.AppUsername = strings.TrimSpace(fmt.Sprint(config["username"]))
		info.DefaultDatabase = strings.TrimSpace(fmt.Sprint(config["database"]))
	case databasesdomain.EngineRedis:
		info.DefaultDatabase = fmt.Sprintf("0-%d", max(configInt(config["databases"])-1, 0))
	}
	return info
}

func primaryProjectContainer(runtime platformdocker.ProjectRuntime) (string, error) {
	if len(runtime.Containers) == 0 {
		return "", fmt.Errorf("当前项目没有可用容器")
	}
	for _, item := range runtime.Containers {
		if item.State == "running" {
			return item.Name, nil
		}
	}
	return runtime.Containers[0].Name, nil
}

func parseRedisInfo(raw string) map[string]string {
	items := map[string]string{}
	for _, line := range splitLines(raw) {
		if strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if ok {
			items[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return items
}

func splitLines(raw string) []string {
	result := []string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func isProtectedDatabaseName(engine, name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	switch engine {
	case databasesdomain.EngineMySQL:
		return slices.Contains([]string{"information_schema", "mysql", "performance_schema", "sys"}, normalized)
	case databasesdomain.EnginePostgres:
		return slices.Contains([]string{"postgres", "template0", "template1"}, normalized)
	default:
		return false
	}
}

func sqlString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func mysqlIdentifier(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

func postgresIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func configInt(value any) int {
	number, err := normalizeInt(value)
	if err != nil {
		return 0
	}
	return number
}

func normalizeInt(value any) (int, error) {
	switch item := value.(type) {
	case int:
		return item, nil
	case int32:
		return int(item), nil
	case int64:
		return int(item), nil
	case float64:
		return int(item), nil
	case string:
		number, err := strconv.Atoi(strings.TrimSpace(item))
		if err != nil {
			return 0, fmt.Errorf("请输入合法数字")
		}
		return number, nil
	default:
		number, err := strconv.Atoi(strings.TrimSpace(fmt.Sprint(value)))
		if err != nil {
			return 0, fmt.Errorf("请输入合法数字")
		}
		return number, nil
	}
}

func normalizeBool(value any) (bool, error) {
	switch item := value.(type) {
	case bool:
		return item, nil
	case string:
		switch strings.ToLower(strings.TrimSpace(item)) {
		case "true", "1", "yes", "on":
			return true, nil
		case "false", "0", "no", "off":
			return false, nil
		}
	}
	return false, fmt.Errorf("请输入合法布尔值")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func filepathJoin(parts ...string) string {
	return strings.Join(parts, string(os.PathSeparator))
}
