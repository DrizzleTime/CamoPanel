package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"camopanel/server/internal/model"
	"camopanel/server/internal/services"

	"github.com/gin-gonic/gin"
)

const (
	databaseEngineMySQL    = "mysql"
	databaseEnginePostgres = "postgres"
	databaseEngineRedis    = "redis"
)

var managedDatabaseTemplateIDs = []string{
	databaseEngineMySQL,
	databaseEnginePostgres,
	databaseEngineRedis,
}

type databaseConnectionInfo struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	AdminUsername   string `json:"admin_username,omitempty"`
	AppUsername     string `json:"app_username,omitempty"`
	DefaultDatabase string `json:"default_database,omitempty"`
	PasswordManaged bool   `json:"password_managed"`
}

type databaseInstanceResponse struct {
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	Engine     string                  `json:"engine"`
	Status     string                  `json:"status"`
	LastError  string                  `json:"last_error"`
	Runtime    services.ProjectRuntime `json:"runtime"`
	Connection databaseConnectionInfo  `json:"connection"`
	CreatedAt  string                  `json:"created_at"`
	UpdatedAt  string                  `json:"updated_at"`
}

type databaseNameItem struct {
	Name string `json:"name"`
}

type databaseAccountItem struct {
	Name      string `json:"name"`
	Host      string `json:"host,omitempty"`
	Superuser bool   `json:"superuser,omitempty"`
}

type databaseRedisKeyspaceItem struct {
	Name string `json:"name"`
	Keys int    `json:"keys"`
}

type databaseRedisConfigItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type databaseOverviewResponse struct {
	Instance       databaseInstanceResponse    `json:"instance"`
	Notice         string                      `json:"notice,omitempty"`
	Databases      []databaseNameItem          `json:"databases,omitempty"`
	Accounts       []databaseAccountItem       `json:"accounts,omitempty"`
	RedisKeyspaces []databaseRedisKeyspaceItem `json:"redis_keyspaces,omitempty"`
	RedisConfig    []databaseRedisConfigItem   `json:"redis_config,omitempty"`
	Summary        map[string]string           `json:"summary,omitempty"`
}

type createManagedDatabaseRequest struct {
	Name string `json:"name"`
}

type createDatabaseAccountRequest struct {
	Name         string `json:"name"`
	Password     string `json:"password"`
	DatabaseName string `json:"database_name"`
}

type updateDatabaseAccountPasswordRequest struct {
	Password string `json:"password"`
}

type grantDatabaseAccountRequest struct {
	AccountName  string `json:"account_name"`
	DatabaseName string `json:"database_name"`
}

type updateRedisConfigRequest struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

func (a *App) handleDatabaseInstances(c *gin.Context) {
	items, err := a.listDatabaseInstances(c.Request.Context(), strings.TrimSpace(c.Query("engine")))
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (a *App) handleDatabaseOverview(c *gin.Context) {
	project, err := a.findManagedDatabaseProject(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	overview, err := a.buildDatabaseOverview(c.Request.Context(), project)
	if err != nil {
		writeDatabaseError(c, err)
		return
	}
	c.JSON(http.StatusOK, overview)
}

func (a *App) handleCreateManagedDatabase(c *gin.Context) {
	project, err := a.findManagedDatabaseProject(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	var req createManagedDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	if err := a.createManagedDatabase(c.Request.Context(), project, req.Name); err != nil {
		writeDatabaseError(c, err)
		return
	}

	_ = a.recordAudit(currentUser(c).ID, "database_create", "project", project.ID, map[string]any{
		"engine": project.TemplateID,
		"name":   strings.TrimSpace(req.Name),
	})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *App) handleCreateDatabaseAccount(c *gin.Context) {
	project, err := a.findManagedDatabaseProject(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	var req createDatabaseAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	if err := a.createDatabaseAccount(c.Request.Context(), project, req); err != nil {
		writeDatabaseError(c, err)
		return
	}

	_ = a.recordAudit(currentUser(c).ID, "database_account_create", "project", project.ID, map[string]any{
		"engine":        project.TemplateID,
		"account_name":  strings.TrimSpace(req.Name),
		"database_name": strings.TrimSpace(req.DatabaseName),
	})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *App) handleUpdateDatabaseAccountPassword(c *gin.Context) {
	project, err := a.findManagedDatabaseProject(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	var req updateDatabaseAccountPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	accountName := strings.TrimSpace(c.Param("account"))
	if err := a.updateDatabaseAccountPassword(c.Request.Context(), project, accountName, req.Password); err != nil {
		writeDatabaseError(c, err)
		return
	}

	_ = a.recordAudit(currentUser(c).ID, "database_account_password_update", "project", project.ID, map[string]any{
		"engine":       project.TemplateID,
		"account_name": accountName,
	})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *App) handleGrantDatabaseAccount(c *gin.Context) {
	project, err := a.findManagedDatabaseProject(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	var req grantDatabaseAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	if err := a.grantDatabaseAccount(c.Request.Context(), project, req.AccountName, req.DatabaseName); err != nil {
		writeDatabaseError(c, err)
		return
	}

	_ = a.recordAudit(currentUser(c).ID, "database_grant", "project", project.ID, map[string]any{
		"engine":        project.TemplateID,
		"account_name":  strings.TrimSpace(req.AccountName),
		"database_name": strings.TrimSpace(req.DatabaseName),
	})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *App) handleUpdateRedisConfig(c *gin.Context) {
	project, err := a.findManagedDatabaseProject(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	if project.TemplateID != databaseEngineRedis {
		writeError(c, http.StatusBadRequest, "当前实例不是 Redis")
		return
	}

	var req updateRedisConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	if err := a.updateRedisProjectConfig(c.Request.Context(), project, req.Key, req.Value); err != nil {
		writeDatabaseError(c, err)
		return
	}

	_ = a.recordAudit(currentUser(c).ID, "redis_config_update", "project", project.ID, map[string]any{
		"key": req.Key,
	})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *App) listDatabaseInstances(ctx context.Context, engine string) ([]databaseInstanceResponse, error) {
	if engine != "" && !slices.Contains(managedDatabaseTemplateIDs, engine) {
		return nil, fmt.Errorf("不支持的数据库类型")
	}

	var projects []model.Project
	query := a.db.Order("created_at desc")
	if engine != "" {
		query = query.Where("template_id = ?", engine)
	} else {
		query = query.Where("template_id IN ?", managedDatabaseTemplateIDs)
	}
	if err := query.Find(&projects).Error; err != nil {
		return nil, err
	}

	items := make([]databaseInstanceResponse, 0, len(projects))
	for _, project := range projects {
		item, err := a.databaseInstanceResponse(ctx, project)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (a *App) findManagedDatabaseProject(projectID string) (model.Project, error) {
	project, err := a.findProject(projectID)
	if err != nil {
		return model.Project{}, err
	}
	if !slices.Contains(managedDatabaseTemplateIDs, project.TemplateID) {
		return model.Project{}, fmt.Errorf("当前项目不是受支持的数据库实例")
	}
	return project, nil
}

func (a *App) databaseInstanceResponse(ctx context.Context, project model.Project) (databaseInstanceResponse, error) {
	item, err := a.projectToResponse(ctx, project)
	if err != nil {
		return databaseInstanceResponse{}, err
	}

	return databaseInstanceResponse{
		ID:         item.ID,
		Name:       item.Name,
		Engine:     item.TemplateID,
		Status:     item.Status,
		LastError:  item.LastError,
		Runtime:    item.Runtime,
		Connection: databaseConnectionFromConfig(item.TemplateID, item.Config),
		CreatedAt:  item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  item.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (a *App) buildDatabaseOverview(ctx context.Context, project model.Project) (databaseOverviewResponse, error) {
	instance, err := a.databaseInstanceResponse(ctx, project)
	if err != nil {
		return databaseOverviewResponse{}, err
	}

	overview := databaseOverviewResponse{
		Instance: instance,
	}

	if instance.Runtime.Status != "running" {
		overview.Notice = "实例当前未运行，暂时无法读取数据库内部信息。"
		return overview, nil
	}

	containerName, err := primaryProjectContainer(instance.Runtime)
	if err != nil {
		return databaseOverviewResponse{}, err
	}

	config := map[string]any{}
	_ = json.Unmarshal([]byte(project.ConfigJSON), &config)

	switch project.TemplateID {
	case databaseEngineMySQL:
		databases, accounts, err := a.loadMySQLOverview(ctx, containerName, config)
		if err != nil {
			return databaseOverviewResponse{}, err
		}
		overview.Databases = databases
		overview.Accounts = accounts
	case databaseEnginePostgres:
		databases, accounts, err := a.loadPostgresOverview(ctx, containerName, config)
		if err != nil {
			return databaseOverviewResponse{}, err
		}
		overview.Databases = databases
		overview.Accounts = accounts
	case databaseEngineRedis:
		keyspaces, redisConfig, summary, err := a.loadRedisOverview(ctx, containerName, config)
		if err != nil {
			return databaseOverviewResponse{}, err
		}
		overview.RedisKeyspaces = keyspaces
		overview.RedisConfig = redisConfig
		overview.Summary = summary
	}

	return overview, nil
}

func (a *App) createManagedDatabase(ctx context.Context, project model.Project, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("请输入数据库名")
	}

	containerName, config, err := a.runningDatabaseTarget(ctx, project)
	if err != nil {
		return err
	}

	switch project.TemplateID {
	case databaseEngineMySQL:
		return a.execMySQL(ctx, containerName, config, fmt.Sprintf(
			"CREATE DATABASE IF NOT EXISTS %s;",
			mysqlIdentifier(name),
		))
	case databaseEnginePostgres:
		exists, err := a.execPostgresQuery(ctx, containerName, config, fmt.Sprintf(
			"SELECT 1 FROM pg_database WHERE datname = %s;",
			sqlString(name),
		))
		if err != nil {
			return err
		}
		if strings.TrimSpace(exists) != "" {
			return nil
		}
		_, err = a.execPostgresQuery(ctx, containerName, config, fmt.Sprintf(
			"CREATE DATABASE %s;",
			postgresIdentifier(name),
		))
		return err
	default:
		return fmt.Errorf("当前数据库类型不支持创建数据库")
	}
}

func (a *App) createDatabaseAccount(ctx context.Context, project model.Project, req createDatabaseAccountRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	req.Password = strings.TrimSpace(req.Password)
	req.DatabaseName = strings.TrimSpace(req.DatabaseName)

	if req.Name == "" {
		return fmt.Errorf("请输入账号名")
	}
	if req.Password == "" {
		return fmt.Errorf("请输入密码")
	}

	containerName, config, err := a.runningDatabaseTarget(ctx, project)
	if err != nil {
		return err
	}

	switch project.TemplateID {
	case databaseEngineMySQL:
		if err := a.execMySQL(ctx, containerName, config, fmt.Sprintf(
			"CREATE USER IF NOT EXISTS %s@'%%' IDENTIFIED BY %s;",
			sqlString(req.Name),
			sqlString(req.Password),
		)); err != nil {
			return err
		}
		if req.DatabaseName != "" {
			return a.execMySQL(ctx, containerName, config, fmt.Sprintf(
				"GRANT ALL PRIVILEGES ON %s.* TO %s@'%%'; FLUSH PRIVILEGES;",
				mysqlIdentifier(req.DatabaseName),
				sqlString(req.Name),
			))
		}
		return nil
	case databaseEnginePostgres:
		exists, err := a.execPostgresQuery(ctx, containerName, config, fmt.Sprintf(
			"SELECT 1 FROM pg_roles WHERE rolname = %s;",
			sqlString(req.Name),
		))
		if err != nil {
			return err
		}
		if strings.TrimSpace(exists) == "" {
			if _, err := a.execPostgresQuery(ctx, containerName, config, fmt.Sprintf(
				"CREATE ROLE %s WITH LOGIN PASSWORD %s;",
				postgresIdentifier(req.Name),
				sqlString(req.Password),
			)); err != nil {
				return err
			}
		} else if err := a.updateDatabaseAccountPassword(ctx, project, req.Name, req.Password); err != nil {
			return err
		}
		if req.DatabaseName != "" {
			return a.grantDatabaseAccount(ctx, project, req.Name, req.DatabaseName)
		}
		return nil
	default:
		return fmt.Errorf("当前数据库类型不支持创建账号")
	}
}

func (a *App) updateDatabaseAccountPassword(ctx context.Context, project model.Project, accountName, password string) error {
	accountName = strings.TrimSpace(accountName)
	password = strings.TrimSpace(password)

	if accountName == "" {
		return fmt.Errorf("请输入账号名")
	}
	if password == "" {
		return fmt.Errorf("请输入密码")
	}

	containerName, config, err := a.runningDatabaseTarget(ctx, project)
	if err != nil {
		return err
	}

	switch project.TemplateID {
	case databaseEngineMySQL:
		return a.execMySQL(ctx, containerName, config, fmt.Sprintf(
			"ALTER USER %s@'%%' IDENTIFIED BY %s;",
			sqlString(accountName),
			sqlString(password),
		))
	case databaseEnginePostgres:
		_, err := a.execPostgresQuery(ctx, containerName, config, fmt.Sprintf(
			"ALTER ROLE %s WITH PASSWORD %s;",
			postgresIdentifier(accountName),
			sqlString(password),
		))
		return err
	default:
		return fmt.Errorf("当前数据库类型不支持修改账号密码")
	}
}

func (a *App) grantDatabaseAccount(ctx context.Context, project model.Project, accountName, databaseName string) error {
	accountName = strings.TrimSpace(accountName)
	databaseName = strings.TrimSpace(databaseName)

	if accountName == "" {
		return fmt.Errorf("请输入账号名")
	}
	if databaseName == "" {
		return fmt.Errorf("请输入数据库名")
	}

	containerName, config, err := a.runningDatabaseTarget(ctx, project)
	if err != nil {
		return err
	}

	switch project.TemplateID {
	case databaseEngineMySQL:
		return a.execMySQL(ctx, containerName, config, fmt.Sprintf(
			"GRANT ALL PRIVILEGES ON %s.* TO %s@'%%'; FLUSH PRIVILEGES;",
			mysqlIdentifier(databaseName),
			sqlString(accountName),
		))
	case databaseEnginePostgres:
		_, err := a.execPostgresQuery(ctx, containerName, config, fmt.Sprintf(
			"GRANT ALL PRIVILEGES ON DATABASE %s TO %s;",
			postgresIdentifier(databaseName),
			postgresIdentifier(accountName),
		))
		return err
	default:
		return fmt.Errorf("当前数据库类型不支持授权")
	}
}

func (a *App) updateRedisProjectConfig(ctx context.Context, project model.Project, key string, value any) error {
	key = strings.TrimSpace(key)
	allowedKeys := map[string]bool{
		"password":   true,
		"databases":  true,
		"appendonly": true,
	}
	if !allowedKeys[key] {
		return fmt.Errorf("不支持修改该 Redis 配置项")
	}

	config := map[string]any{}
	_ = json.Unmarshal([]byte(project.ConfigJSON), &config)

	switch key {
	case "password":
		raw := strings.TrimSpace(fmt.Sprint(value))
		if raw == "" {
			return fmt.Errorf("密码不能为空")
		}
		config[key] = raw
	case "databases":
		number, err := normalizeInt(value)
		if err != nil {
			return err
		}
		if number <= 0 {
			return fmt.Errorf("逻辑库数量必须大于 0")
		}
		config[key] = number
	case "appendonly":
		boolean, err := normalizeBool(value)
		if err != nil {
			return err
		}
		config[key] = boolean
	}

	return a.redeployProjectWithConfig(ctx, project, config)
}

func (a *App) loadMySQLOverview(ctx context.Context, containerName string, config map[string]any) ([]databaseNameItem, []databaseAccountItem, error) {
	databaseOutput, err := a.execMySQLQuery(ctx, containerName, config, "SHOW DATABASES;")
	if err != nil {
		return nil, nil, err
	}
	accountOutput, err := a.execMySQLQuery(ctx, containerName, config, "SELECT user, host FROM mysql.user ORDER BY user, host;")
	if err != nil {
		return nil, nil, err
	}

	databases := make([]databaseNameItem, 0)
	for _, line := range splitLines(databaseOutput) {
		databases = append(databases, databaseNameItem{Name: line})
	}

	accounts := make([]databaseAccountItem, 0)
	for _, line := range splitLines(accountOutput) {
		parts := strings.Split(line, "\t")
		account := databaseAccountItem{Name: strings.TrimSpace(parts[0])}
		if len(parts) > 1 {
			account.Host = strings.TrimSpace(parts[1])
		}
		accounts = append(accounts, account)
	}

	return databases, accounts, nil
}

func (a *App) loadPostgresOverview(ctx context.Context, containerName string, config map[string]any) ([]databaseNameItem, []databaseAccountItem, error) {
	databaseOutput, err := a.execPostgresQuery(ctx, containerName, config, "SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname;")
	if err != nil {
		return nil, nil, err
	}
	accountOutput, err := a.execPostgresQuery(ctx, containerName, config, "SELECT rolname, rolsuper FROM pg_roles WHERE rolcanlogin ORDER BY rolname;")
	if err != nil {
		return nil, nil, err
	}

	databases := make([]databaseNameItem, 0)
	for _, line := range splitLines(databaseOutput) {
		databases = append(databases, databaseNameItem{Name: line})
	}

	accounts := make([]databaseAccountItem, 0)
	for _, line := range splitLines(accountOutput) {
		parts := strings.Split(line, "|")
		account := databaseAccountItem{Name: strings.TrimSpace(parts[0])}
		if len(parts) > 1 {
			account.Superuser = strings.EqualFold(strings.TrimSpace(parts[1]), "t")
		}
		accounts = append(accounts, account)
	}

	return databases, accounts, nil
}

func (a *App) loadRedisOverview(ctx context.Context, containerName string, config map[string]any) ([]databaseRedisKeyspaceItem, []databaseRedisConfigItem, map[string]string, error) {
	keyspaceOutput, err := a.execRedisQuery(ctx, containerName, config, "INFO", "keyspace")
	if err != nil {
		return nil, nil, nil, err
	}
	infoOutput, err := a.execRedisQuery(ctx, containerName, config, "INFO", "server")
	if err != nil {
		return nil, nil, nil, err
	}
	memoryOutput, err := a.execRedisQuery(ctx, containerName, config, "INFO", "memory")
	if err != nil {
		return nil, nil, nil, err
	}
	clientOutput, err := a.execRedisQuery(ctx, containerName, config, "INFO", "clients")
	if err != nil {
		return nil, nil, nil, err
	}

	configItems := make([]databaseRedisConfigItem, 0, 3)
	for _, key := range []string{"appendonly", "databases"} {
		value, err := a.execRedisConfigGet(ctx, containerName, config, key)
		if err != nil {
			return nil, nil, nil, err
		}
		configItems = append(configItems, databaseRedisConfigItem{Key: key, Value: value})
	}
	configItems = append(configItems, databaseRedisConfigItem{
		Key:   "password",
		Value: "已托管",
	})

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

	keyspaces := make([]databaseRedisKeyspaceItem, 0)
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
			if !strings.HasPrefix(item, "keys=") {
				continue
			}
			keys, _ = strconv.Atoi(strings.TrimPrefix(item, "keys="))
		}
		keyspaces = append(keyspaces, databaseRedisKeyspaceItem{Name: name, Keys: keys})
	}
	sort.Slice(keyspaces, func(i, j int) bool {
		return keyspaces[i].Name < keyspaces[j].Name
	})

	return keyspaces, configItems, summary, nil
}

func (a *App) runningDatabaseTarget(ctx context.Context, project model.Project) (string, map[string]any, error) {
	runtime, err := a.executor.InspectProject(ctx, project.Name)
	if err != nil {
		return "", nil, err
	}
	if runtime.Status != "running" {
		return "", nil, fmt.Errorf("实例当前未运行")
	}
	containerName, err := primaryProjectContainer(runtime)
	if err != nil {
		return "", nil, err
	}

	config := map[string]any{}
	_ = json.Unmarshal([]byte(project.ConfigJSON), &config)

	return containerName, config, nil
}

func (a *App) execMySQL(ctx context.Context, containerName string, config map[string]any, query string) error {
	_, err := a.execMySQLQuery(ctx, containerName, config, query)
	return err
}

func (a *App) execMySQLQuery(ctx context.Context, containerName string, config map[string]any, query string) (string, error) {
	password := strings.TrimSpace(fmt.Sprint(config["root_password"]))
	if password == "" {
		return "", fmt.Errorf("缺少 MySQL root 密码")
	}
	command := fmt.Sprintf(
		"MYSQL_PWD=%s mysql -uroot --batch --raw --skip-column-names -e %s",
		shellQuote(password),
		shellQuote(query),
	)
	return a.containers.ExecInContainer(ctx, containerName, "sh", "-lc", command)
}

func (a *App) execPostgresQuery(ctx context.Context, containerName string, config map[string]any, query string) (string, error) {
	username := strings.TrimSpace(fmt.Sprint(config["username"]))
	password := strings.TrimSpace(fmt.Sprint(config["password"]))
	if username == "" || password == "" {
		return "", fmt.Errorf("缺少 PostgreSQL 管理账号")
	}
	command := fmt.Sprintf(
		"PGPASSWORD=%s psql -U %s -d postgres -t -A -c %s",
		shellQuote(password),
		shellQuote(username),
		shellQuote(query),
	)
	return a.containers.ExecInContainer(ctx, containerName, "sh", "-lc", command)
}

func (a *App) execRedisQuery(ctx context.Context, containerName string, config map[string]any, args ...string) (string, error) {
	password := strings.TrimSpace(fmt.Sprint(config["password"]))
	if password == "" {
		return "", fmt.Errorf("缺少 Redis 密码")
	}

	commandArgs := []string{"redis-cli", "--no-auth-warning", "-a", password}
	commandArgs = append(commandArgs, args...)
	return a.containers.ExecInContainer(ctx, containerName, commandArgs...)
}

func (a *App) execRedisConfigGet(ctx context.Context, containerName string, config map[string]any, key string) (string, error) {
	output, err := a.execRedisQuery(ctx, containerName, config, "--raw", "CONFIG", "GET", key)
	if err != nil {
		return "", err
	}
	lines := splitLines(output)
	if len(lines) >= 2 {
		return lines[1], nil
	}
	return "", nil
}

func (a *App) redeployProjectWithConfig(ctx context.Context, project model.Project, config map[string]any) error {
	templateItem, err := a.templates.Get(project.TemplateID)
	if err != nil {
		return err
	}

	normalized, err := templateItem.ValidateAndNormalize(config)
	if err != nil {
		return err
	}

	rendered, err := templateItem.Render(normalized, a.templateRuntime(project.Name))
	if err != nil {
		return err
	}
	if err := os.WriteFile(project.ComposePath, []byte(rendered), 0o644); err != nil {
		return err
	}

	if err := a.executor.Redeploy(ctx, project.Name, project.ComposePath); err != nil {
		project.LastError = err.Error()
		_ = a.db.Save(&project).Error
		return err
	}

	configJSON, err := templateItem.ConfigJSON(normalized)
	if err != nil {
		return err
	}

	project.ConfigJSON = configJSON
	project.LastError = ""
	project.Status = "running"
	if runtime, runtimeErr := a.executor.InspectProject(ctx, project.Name); runtimeErr == nil {
		project.Status = runtime.Status
	}
	return a.db.Save(&project).Error
}

func databaseConnectionFromConfig(engine string, config map[string]any) databaseConnectionInfo {
	info := databaseConnectionInfo{
		Host:            "127.0.0.1",
		Port:            configInt(config["port"]),
		PasswordManaged: true,
	}

	switch engine {
	case databaseEngineMySQL:
		info.AdminUsername = "root"
		info.AppUsername = strings.TrimSpace(fmt.Sprint(config["username"]))
		info.DefaultDatabase = strings.TrimSpace(fmt.Sprint(config["database"]))
	case databaseEnginePostgres:
		info.AdminUsername = strings.TrimSpace(fmt.Sprint(config["username"]))
		info.AppUsername = strings.TrimSpace(fmt.Sprint(config["username"]))
		info.DefaultDatabase = strings.TrimSpace(fmt.Sprint(config["database"]))
	case databaseEngineRedis:
		info.DefaultDatabase = fmt.Sprintf("0-%d", max(configInt(config["databases"])-1, 0))
	}

	return info
}

func primaryProjectContainer(runtime services.ProjectRuntime) (string, error) {
	if len(runtime.Containers) == 0 {
		return "", fmt.Errorf("当前项目没有可用容器")
	}
	for _, container := range runtime.Containers {
		if container.State == "running" {
			return container.Name, nil
		}
	}
	return runtime.Containers[0].Name, nil
}

func writeDatabaseError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrDockerUnavailable):
		writeError(c, http.StatusBadGateway, "Docker 当前不可用")
	default:
		writeError(c, http.StatusBadRequest, err.Error())
	}
}

func parseRedisInfo(raw string) map[string]string {
	items := map[string]string{}
	for _, line := range splitLines(raw) {
		if strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		items[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return items
}

func splitLines(raw string) []string {
	lines := []string{}
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lines = append(lines, trimmed)
	}
	return lines
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
