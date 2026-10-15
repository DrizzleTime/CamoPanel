package docker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

type runSpec struct {
	Env     []string
	Command []string
	Restart string
	Binds   []string
}

type Demo struct {
	mu    sync.Mutex
	data  Snapshot
	logs  map[string]string
	specs map[string]runSpec
}

func NewDemo() *Demo {
	now := time.Now().Unix()
	d := &Demo{data: Snapshot{
		Mode: "demo", Engine: Engine{Name: "camo-demo", ServerVersion: "演示引擎", OperatingSystem: "Ubuntu 24.04 LTS · 演示", NCPU: 4, MemTotal: 8 << 30},
		Containers: []Container{
			{ID: "a1b2c3d4e5f6", Names: []string{"/camo-web"}, Image: "nginx:alpine", State: "running", Status: "Up 2 hours", Created: now - 7200, Ports: []Port{{IP: "127.0.0.1", PrivatePort: 80, PublicPort: 8088, Type: "tcp"}}},
			{ID: "b2c3d4e5f6a1", Names: []string{"/camo-postgres"}, Image: "postgres:17-alpine", State: "running", Status: "Up 2 hours", Created: now - 7200, Ports: []Port{{PrivatePort: 5432, Type: "tcp"}}},
			{ID: "c3d4e5f6a1b2", Names: []string{"/camo-redis"}, Image: "redis:7-alpine", State: "running", Status: "Up 2 hours", Created: now - 7200, Ports: []Port{{PrivatePort: 6379, Type: "tcp"}}},
			{ID: "d4e5f6a1b2c3", Names: []string{"/backup-worker"}, Image: "alpine:3.21", State: "exited", Status: "Exited (0) 30 minutes ago", Created: now - 86400, Ports: []Port{}},
		},
		Images:   []Image{{ID: "sha256:111111111111", RepoTags: []string{"nginx:alpine"}, Size: 52 << 20, Created: now - 86400}, {ID: "sha256:222222222222", RepoTags: []string{"postgres:17-alpine"}, Size: 273 << 20, Created: now - 86400}, {ID: "sha256:333333333333", RepoTags: []string{"redis:7-alpine"}, Size: 41 << 20, Created: now - 86400}, {ID: "sha256:444444444444", RepoTags: []string{"alpine:3.21"}, Size: 8 << 20, Created: now - 86400}, {ID: "sha256:dangling0001", RepoTags: []string{"<none>:<none>"}, Size: 1024, Created: now - 86400}},
		Networks: []Network{{ID: "net111", Name: "bridge", Driver: "bridge", Scope: "local"}, {ID: "net222", Name: "camo-internal", Driver: "bridge", Scope: "local"}, {ID: "net333", Name: "host", Driver: "host", Scope: "local"}},
		Volumes:  []Volume{{Name: "postgres-data", Driver: "local", Mountpoint: "/var/lib/docker/volumes/postgres-data/_data"}, {Name: "redis-data", Driver: "local", Mountpoint: "/var/lib/docker/volumes/redis-data/_data"}},
	}, logs: map[string]string{}, specs: map[string]runSpec{}}
	for _, c := range d.data.Containers {
		d.logs[c.ID] = fmt.Sprintf("%s [demo] %s: container initialized\n", time.Now().UTC().Format(time.RFC3339), c.Names[0][1:])
	}
	return d
}
func (d *Demo) Snapshot(_ context.Context) (Snapshot, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	s := d.data
	s.Containers = append([]Container{}, d.data.Containers...)
	s.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	return s, nil
}
func (d *Demo) Action(_ context.Context, id, action string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i := range d.data.Containers {
		c := &d.data.Containers[i]
		if c.ID != id {
			continue
		}
		if action == "remove" {
			if c.State == "running" {
				return &Error{409, "请先停止容器再删除"}
			}
			d.data.Containers = append(d.data.Containers[:i], d.data.Containers[i+1:]...)
			delete(d.logs, id)
			return nil
		}
		switch action {
		case "stop":
			c.State, c.Status = "exited", "Exited (0) just now"
		case "pause":
			c.State, c.Status = "paused", "Paused"
		default:
			c.State, c.Status = "running", "Up just now"
		}
		d.logs[id] += fmt.Sprintf("%s [demo] action=%s\n", time.Now().UTC().Format(time.RFC3339), action)
		return nil
	}
	return &Error{404, "容器不存在"}
}
func (d *Demo) Create(_ context.Context, r CreateRequest) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, c := range d.data.Containers {
		if c.Names[0] == "/"+r.Name {
			return "", &Error{409, "容器名称已存在"}
		}
	}
	found := false
	for _, image := range d.data.Images {
		for _, tag := range image.RepoTags {
			if tag == r.Image {
				found = true
			}
		}
	}
	if !found {
		return "", &Error{404, "请使用本地已有镜像"}
	}
	ports := publishedPorts(r)
	for _, p := range ports {
		if p.PublicPort == 0 {
			continue
		}
		for _, c := range d.data.Containers {
			for _, existing := range c.Ports {
				if existing.PublicPort == p.PublicPort {
					return "", &Error{409, "演示端口已被占用"}
				}
			}
		}
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	id := hex.EncodeToString(b[:])
	d.data.Containers = append(d.data.Containers, Container{ID: id, Names: []string{"/" + r.Name}, Image: r.Image, State: "created", Status: "Created", Created: time.Now().Unix(), Ports: ports})
	if d.specs == nil {
		d.specs = map[string]runSpec{}
	}
	d.specs[id] = runSpec{Env: append([]string{}, r.Env...), Command: append([]string{}, r.Command...), Restart: r.RestartPolicy, Binds: append([]string{}, r.Binds...)}
	d.logs[id] = fmt.Sprintf("%s [demo] container created; awaiting start\n", time.Now().UTC().Format(time.RFC3339))
	return id, nil
}

func publishedPorts(r CreateRequest) []Port {
	if len(r.Ports) > 0 {
		ports := make([]Port, 0, len(r.Ports))
		for _, p := range r.Ports {
			proto := p.Protocol
			if proto == "" {
				proto = "tcp"
			}
			ip := p.HostIP
			if ip == "" {
				ip = "127.0.0.1"
			}
			ports = append(ports, Port{IP: ip, PrivatePort: p.ContainerPort, PublicPort: p.HostPort, Type: proto})
		}
		return ports
	}
	if r.HostPort > 0 {
		return []Port{{IP: "127.0.0.1", PrivatePort: r.ContainerPort, PublicPort: r.HostPort, Type: "tcp"}}
	}
	return []Port{}
}
func (d *Demo) Logs(_ context.Context, id string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	value, ok := d.logs[id]
	if !ok {
		return "", &Error{404, "容器不存在"}
	}
	return value, nil
}

func (d *Demo) FindContainerByName(_ context.Context, name string) (*Container, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	trimmed := strings.TrimPrefix(name, "/")
	for _, c := range d.data.Containers {
		for _, n := range c.Names {
			if strings.TrimPrefix(n, "/") == trimmed {
				copy := c
				return &copy, nil
			}
		}
	}
	return nil, nil
}

func (d *Demo) RunContainer(_ context.Context, r RunContainerRequest) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	trimmed := strings.TrimPrefix(r.Name, "/")
	for i := range d.data.Containers {
		c := &d.data.Containers[i]
		for _, n := range c.Names {
			if strings.TrimPrefix(n, "/") == trimmed {
				c.State = "running"
				c.Status = "Up just now"
				return c.ID, nil
			}
		}
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	id := hex.EncodeToString(b[:])
	d.data.Containers = append(d.data.Containers, Container{
		ID:      id,
		Names:   []string{"/" + trimmed},
		Image:   r.Image,
		State:   "running",
		Status:  "Up just now",
		Created: time.Now().Unix(),
		Ports:   r.Ports,
	})
	hasImage := false
	for _, img := range d.data.Images {
		for _, tag := range img.RepoTags {
			if tag == r.Image {
				hasImage = true
				break
			}
		}
	}
	if !hasImage {
		d.data.Images = append(d.data.Images, Image{
			ID:       "sha256:" + id[:12],
			RepoTags: []string{r.Image},
			Size:     85 << 20,
			Created:  time.Now().Unix() - 3600,
		})
	}
	d.logs[id] = fmt.Sprintf("%s [demo] %s container created and running\n", time.Now().UTC().Format(time.RFC3339), trimmed)
	return id, nil
}
