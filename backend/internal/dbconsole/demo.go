package dbconsole

import (
	"fmt"
	"strings"
	"sync"
)

type Engine struct {
	Kind      string
	Installed bool
	StorePath string
	databases map[string]bool
	users     map[string]string
	dump      string
	keys      []string
}

type Console struct {
	mu      sync.Mutex
	engines map[string]*Engine
}

func NewDemo() *Console {
	c := &Console{engines: map[string]*Engine{}}
	for _, kind := range []string{"mysql", "postgres", "redis"} {
		c.engines[kind] = &Engine{
			Kind: kind, Installed: true, StorePath: "/store?app=" + kind,
			databases: map[string]bool{}, users: map[string]string{}, keys: []string{},
		}
	}
	return c
}

type Info struct {
	Kind      string `json:"kind"`
	Installed bool   `json:"installed"`
	StorePath string `json:"store_path"`
}

func (c *Console) List() []Info {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := []Info{}
	for _, kind := range []string{"mysql", "postgres", "redis"} {
		engine := c.engines[kind]
		out = append(out, Info{Kind: kind, Installed: engine.Installed, StorePath: engine.StorePath})
	}
	return out
}

func (c *Console) Missing(kind string) Info {
	return Info{Kind: kind, Installed: false, StorePath: "/store?app=" + kind}
}

func (c *Console) CreateDatabase(kind, name string) error {
	engine, err := c.sql(kind)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	engine.databases[name] = true
	return nil
}

func (c *Console) CreateUser(kind, name, password string) error {
	engine, err := c.sql(kind)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	engine.users[name] = password
	return nil
}

func (c *Console) Export(kind string) (string, error) {
	engine, err := c.sql(kind)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	var b strings.Builder
	for name := range engine.databases {
		fmt.Fprintf(&b, "CREATE DATABASE %s;\n", name)
	}
	for name := range engine.users {
		fmt.Fprintf(&b, "CREATE USER %s;\n", name)
	}
	engine.dump = b.String()
	return engine.dump, nil
}

func (c *Console) Import(kind, sql string) error {
	engine, err := c.sql(kind)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, line := range strings.Split(sql, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && strings.EqualFold(fields[0], "CREATE") && strings.EqualFold(fields[1], "DATABASE") {
			name := strings.Trim(fields[2], ";")
			engine.databases[name] = true
		}
	}
	return nil
}

func (c *Console) Exec(kind, sql string) (string, error) {
	if _, err := c.sql(kind); err != nil {
		return "", err
	}
	return "ok", nil
}

func (c *Console) RedisScan(prefix string, limit int) ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	engine := c.engines["redis"]
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	var out []string
	for _, key := range engine.keys {
		if prefix == "" || strings.HasPrefix(key, prefix) {
			out = append(out, key)
			if len(out) >= limit {
				break
			}
		}
	}
	if out == nil {
		out = []string{}
	}
	return out, nil
}

func (c *Console) RedisAdd(keys ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.engines["redis"].keys = append(c.engines["redis"].keys, keys...)
}

func (c *Console) RedisDel(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	engine := c.engines["redis"]
	next := engine.keys[:0]
	found := false
	for _, item := range engine.keys {
		if item == key && !found {
			found = true
			continue
		}
		next = append(next, item)
	}
	engine.keys = next
	if !found {
		return fmt.Errorf("键不存在")
	}
	return nil
}

func (c *Console) sql(kind string) (*Engine, error) {
	if kind != "mysql" && kind != "postgres" {
		return nil, fmt.Errorf("不支持的数据库")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	engine := c.engines[kind]
	if engine == nil || !engine.Installed {
		return nil, fmt.Errorf("数据库未安装")
	}
	return engine, nil
}
