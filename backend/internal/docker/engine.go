package docker

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// EngineClient talks to the local daemon over its Unix socket. No shell commands.
type EngineClient struct{ client *http.Client }

func NewEngine(host string) (*EngineClient, error) {
	if !strings.HasPrefix(host, "unix://") || strings.TrimPrefix(host, "unix://") == "" {
		return nil, fmt.Errorf("DOCKER_HOST must be a unix:// socket path")
	}
	socket := strings.TrimPrefix(host, "unix://")
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	return &EngineClient{client: &http.Client{Transport: transport, Timeout: 40 * time.Second}}, nil
}

func (e *EngineClient) request(ctx context.Context, method, path string, body any) ([]byte, error) {
	var payload io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		payload = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://docker"+path, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, &Error{503, "无法连接 Docker，请检查服务和 socket 权限"}
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20+1))
	if err != nil {
		return nil, err
	}
	if len(b) > 8<<20 {
		return nil, &Error{502, "Docker 响应超过 8 MiB 限制"}
	}
	if resp.StatusCode >= 400 {
		var message struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(b, &message)
		if message.Message == "" {
			message.Message = http.StatusText(resp.StatusCode)
		}
		return nil, &Error{resp.StatusCode, message.Message}
	}
	return b, nil
}

// Discover the daemon's API version on each operation so restarts/upgrades are handled.
func (e *EngineClient) prefix(ctx context.Context) (string, error) {
	b, err := e.request(ctx, "GET", "/version", nil)
	if err != nil {
		return "", err
	}
	var version struct {
		APIVersion string `json:"ApiVersion"`
	}
	if err := json.Unmarshal(b, &version); err != nil {
		return "", err
	}
	if !regexp.MustCompile(`^\d+\.\d+$`).MatchString(version.APIVersion) {
		return "", fmt.Errorf("invalid Docker API version")
	}
	return "/v" + version.APIVersion, nil
}
func (e *EngineClient) Snapshot(ctx context.Context) (Snapshot, error) {
	prefix, err := e.prefix(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	s := Snapshot{Mode: "docker", Containers: []Container{}, Images: []Image{}, Networks: []Network{}, Volumes: []Volume{}, GeneratedAt: time.Now().UTC().Format(time.RFC3339)}
	var volumes struct {
		Volumes []Volume `json:"Volumes"`
	}
	for _, endpoint := range []struct {
		path   string
		target any
	}{
		{"/info", &s.Engine}, {"/containers/json?all=true", &s.Containers}, {"/images/json", &s.Images}, {"/networks", &s.Networks}, {"/volumes", &volumes},
	} {
		b, err := e.request(ctx, "GET", prefix+endpoint.path, nil)
		if err != nil {
			return Snapshot{}, err
		}
		if err := json.Unmarshal(b, endpoint.target); err != nil {
			return Snapshot{}, err
		}
	}
	if volumes.Volumes != nil {
		s.Volumes = volumes.Volumes
	}
	return s, nil
}
func (e *EngineClient) Action(ctx context.Context, id, action string) error {
	prefix, err := e.prefix(ctx)
	if err != nil {
		return err
	}
	method, path := "POST", "/containers/"+url.PathEscape(id)+"/"+action
	if action == "remove" {
		method, path = "DELETE", "/containers/"+url.PathEscape(id)
	}
	if action == "stop" || action == "restart" {
		path += "?t=10"
	}
	_, err = e.request(ctx, method, prefix+path, nil)
	return err
}
func (e *EngineClient) Create(ctx context.Context, r CreateRequest) (string, error) {
	prefix, err := e.prefix(ctx)
	if err != nil {
		return "", err
	}
	body := map[string]any{"Image": r.Image, "Labels": map[string]string{"io.camopanel.managed": "true"}}
	if len(r.Env) > 0 {
		body["Env"] = r.Env
	}
	if len(r.Command) > 0 {
		body["Cmd"] = r.Command
	}
	exposed := map[string]any{}
	bindings := map[string]any{}
	for _, p := range publishedPorts(r) {
		proto := p.Type
		if proto == "" {
			proto = "tcp"
		}
		key := strconv.Itoa(p.PrivatePort) + "/" + proto
		exposed[key] = map[string]any{}
		if p.PublicPort > 0 {
			ip := p.IP
			if ip == "" {
				ip = "127.0.0.1"
			}
			bindings[key] = []map[string]string{{"HostIp": ip, "HostPort": strconv.Itoa(p.PublicPort)}}
		}
	}
	if len(exposed) > 0 {
		body["ExposedPorts"] = exposed
	}
	hostConfig := map[string]any{}
	if len(bindings) > 0 {
		hostConfig["PortBindings"] = bindings
	}
	if len(r.Binds) > 0 {
		hostConfig["Binds"] = r.Binds
	}
	if r.RestartPolicy != "" {
		hostConfig["RestartPolicy"] = map[string]string{"Name": r.RestartPolicy}
	}
	if len(hostConfig) > 0 {
		body["HostConfig"] = hostConfig
	}
	b, err := e.request(ctx, "POST", prefix+"/containers/create?name="+url.QueryEscape(r.Name), body)
	if err != nil {
		return "", err
	}
	var created struct {
		ID string `json:"Id"`
	}
	err = json.Unmarshal(b, &created)
	return created.ID, err
}
func (e *EngineClient) Logs(ctx context.Context, id string) (string, error) {
	prefix, err := e.prefix(ctx)
	if err != nil {
		return "", err
	}
	b, err := e.request(ctx, "GET", prefix+"/containers/"+url.PathEscape(id)+"/logs?stdout=true&stderr=true&timestamps=true&tail=200", nil)
	if err != nil {
		return "", err
	}
	return decodeLogs(b), nil
}

func (e *EngineClient) FindContainerByName(ctx context.Context, name string) (*Container, error) {
	prefix, err := e.prefix(ctx)
	if err != nil {
		return nil, err
	}
	b, err := e.request(ctx, "GET", prefix+"/containers/json?all=true", nil)
	if err != nil {
		return nil, err
	}
	var containers []Container
	if err := json.Unmarshal(b, &containers); err != nil {
		return nil, err
	}
	trimmed := strings.TrimPrefix(name, "/")
	for _, c := range containers {
		for _, n := range c.Names {
			if strings.TrimPrefix(n, "/") == trimmed {
				return &c, nil
			}
		}
	}
	return nil, nil
}

func (e *EngineClient) RunContainer(ctx context.Context, r RunContainerRequest) (string, error) {
	existing, err := e.FindContainerByName(ctx, r.Name)
	if err != nil {
		return "", err
	}
	if existing != nil {
		if existing.State != "running" {
			_ = e.Action(ctx, existing.ID, "start")
		}
		return existing.ID, nil
	}
	prefix, err := e.prefix(ctx)
	if err != nil {
		return "", err
	}
	labels := map[string]string{"io.camopanel.managed": "true"}
	for k, v := range r.Labels {
		labels[k] = v
	}
	body := map[string]any{
		"Image":  r.Image,
		"Labels": labels,
	}
	if len(r.Env) > 0 {
		body["Env"] = r.Env
	}
	exposedPorts := map[string]any{}
	portBindings := map[string][]map[string]string{}
	for _, p := range r.Ports {
		proto := p.Type
		if proto == "" {
			proto = "tcp"
		}
		portKey := fmt.Sprintf("%d/%s", p.PrivatePort, proto)
		exposedPorts[portKey] = map[string]any{}
		if p.PublicPort > 0 {
			hostIP := p.IP
			if hostIP == "" {
				hostIP = "0.0.0.0"
			}
			portBindings[portKey] = []map[string]string{
				{"HostIp": hostIP, "HostPort": strconv.Itoa(p.PublicPort)},
			}
		}
	}
	if len(exposedPorts) > 0 {
		body["ExposedPorts"] = exposedPorts
	}
	hostConfig := map[string]any{}
	if len(portBindings) > 0 {
		hostConfig["PortBindings"] = portBindings
	}
	if len(r.Binds) > 0 {
		hostConfig["Binds"] = r.Binds
	}
	if r.RestartPolicy != "" {
		hostConfig["RestartPolicy"] = map[string]string{"Name": r.RestartPolicy}
	} else {
		hostConfig["RestartPolicy"] = map[string]string{"Name": "unless-stopped"}
	}
	body["HostConfig"] = hostConfig

	b, err := e.request(ctx, "POST", prefix+"/containers/create?name="+url.QueryEscape(r.Name), body)
	if err != nil {
		return "", err
	}
	var created struct {
		ID string `json:"Id"`
	}
	if err := json.Unmarshal(b, &created); err != nil {
		return "", err
	}
	if err := e.Action(ctx, created.ID, "start"); err != nil {
		return created.ID, err
	}
	return created.ID, nil
}

// Docker multiplexes stdout/stderr with an 8-byte header, except for TTY containers.
func decodeLogs(b []byte) string {
	var out bytes.Buffer
	for len(b) >= 8 && (b[0] == 1 || b[0] == 2) && b[1] == 0 && b[2] == 0 && b[3] == 0 {
		n := int(binary.BigEndian.Uint32(b[4:8]))
		if n > len(b)-8 {
			break
		}
		out.Write(b[8 : 8+n])
		b = b[8+n:]
	}
	out.Write(b)
	return out.String()
}
