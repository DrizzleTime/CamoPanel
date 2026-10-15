package assistant

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"camopanel/backend/internal/audit"
	"camopanel/backend/internal/certs"
	"camopanel/backend/internal/cron"
	"camopanel/backend/internal/docker"
	"camopanel/backend/internal/host"
	"camopanel/backend/internal/nginx"
	"camopanel/backend/internal/site"
	"camopanel/backend/internal/store"
)

type Settings struct {
	BaseURL string            `json:"base_url"`
	APIKey  string            `json:"api_key,omitempty"`
	Model   string            `json:"model"`
	Headers map[string]string `json:"headers,omitempty"`
}

func (s Settings) Configured() bool {
	return strings.TrimSpace(s.BaseURL) != "" && strings.TrimSpace(s.APIKey) != "" && strings.TrimSpace(s.Model) != ""
}

func Load(dataDir string) (Settings, error) {
	b, err := os.ReadFile(filepath.Join(dataDir, "ai.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return Settings{}, nil
		}
		return Settings{}, err
	}
	var settings Settings
	if len(b) == 0 {
		return settings, nil
	}
	err = json.Unmarshal(b, &settings)
	return settings, err
}

func Save(dataDir string, settings Settings) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	if settings.Headers == nil {
		settings.Headers = map[string]string{}
	}
	b, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dataDir, "ai.json"), b, 0o600)
}

func Public(settings Settings) map[string]any {
	base := settings.BaseURL
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	headers := []map[string]any{}
	for name, value := range settings.Headers {
		headers = append(headers, map[string]any{"name": name, "secret_set": value != ""})
	}
	return map[string]any{
		"base_url":   base,
		"model":      settings.Model,
		"key_set":    settings.APIKey != "",
		"configured": settings.Configured(),
		"headers":    headers,
	}
}

// Endpoint is POST {base}/v1/responses. A base that already ends in /v1 is not doubled.
func Endpoint(base string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if strings.HasSuffix(base, "/v1") {
		return base + "/responses"
	}
	return base + "/v1/responses"
}

type Tools struct {
	Host   host.Sampler
	Docker docker.Service
	Sites  *site.Service
	Certs  *certs.Store
	Store  *store.Service
	Cron   *cron.Scheduler
	Nginx  *nginx.Supervisor
	Audit  *audit.Log
	Actor  string
	Role   string
}

type Reply struct {
	Text    string         `json:"reply"`
	Pending map[string]any `json:"pending_write,omitempty"`
}

type call struct {
	Name      string
	CallID    string
	Arguments string
}

func (t Tools) Chat(ctx context.Context, settings Settings, question string, confirm bool) (Reply, error) {
	if !settings.Configured() {
		text := t.snapshot()
		return Reply{Text: text + "\n请填写供应商、接口地址、API 密钥和模型。"}, nil
	}
	writes := map[string]bool{"restart_container": true, "reload_nginx": true, "run_scheduled_job": true}
	input := any(question)
	var previous string
	var text string
	for round := 0; round < 3; round++ {
		body, err := t.post(ctx, settings, input, previous)
		if err != nil {
			return Reply{}, err
		}
		if body.ID != "" {
			previous = body.ID
		}
		if body.Text != "" {
			text = body.Text
		}
		if len(body.Calls) == 0 {
			if text == "" {
				text = "模型没有返回文本。"
			}
			return Reply{Text: text}, nil
		}
		outputs := []map[string]any{}
		for _, item := range body.Calls {
			if writes[item.Name] {
				if t.Role == "viewer" {
					return Reply{}, fmt.Errorf("只读账号不能执行写入操作")
				}
				if !confirm {
					return Reply{Text: "这个操作需要你在对话里确认后才会执行。", Pending: map[string]any{"name": item.Name, "arguments": item.Arguments}}, nil
				}
				out, err := t.write(ctx, item)
				if err != nil {
					return Reply{}, err
				}
				outputs = append(outputs, map[string]any{"type": "function_call_output", "call_id": item.CallID, "output": out})
				continue
			}
			out, err := t.read(ctx, item)
			if err != nil {
				out = err.Error()
			}
			outputs = append(outputs, map[string]any{"type": "function_call_output", "call_id": item.CallID, "output": out})
		}
		input = outputs
	}
	if text == "" {
		text = "模型没有返回文本。"
	}
	return Reply{Text: text}, nil
}

func (t Tools) snapshot() string {
	if t.Docker == nil {
		return "当前还没有运行快照。"
	}
	snap, err := t.Docker.Snapshot(context.Background())
	if err != nil {
		return "暂时读不到运行快照。"
	}
	stopped := 0
	for _, c := range snap.Containers {
		if c.State != "running" {
			stopped++
		}
	}
	return fmt.Sprintf("主机：%s\nDocker：%s\nCPU：%d 核\n容器 %d 个，其中 %d 个未运行。\n这些是当前面板快照。", snap.Engine.Name, snap.Engine.ServerVersion, snap.Engine.NCPU, len(snap.Containers), stopped)
}

type parsed struct {
	ID    string
	Text  string
	Calls []call
}

func (t Tools) post(ctx context.Context, settings Settings, input any, previous string) (parsed, error) {
	payload := map[string]any{
		"model":  settings.Model,
		"input":  input,
		"stream": true,
		"tools":  toolDefs(),
	}
	if previous != "" {
		payload["previous_response_id"] = previous
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return parsed{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, Endpoint(settings.BaseURL), bytes.NewReader(buf))
	if err != nil {
		return parsed{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, application/json")
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)
	for name, value := range settings.Headers {
		if name != "" {
			req.Header.Set(name, value)
		}
	}
	client := &http.Client{Timeout: 40 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return parsed{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return parsed{}, err
	}
	if resp.StatusCode >= 300 {
		return parsed{}, fmt.Errorf("模型接口返回 %d", resp.StatusCode)
	}
	return parseResponse(raw), nil
}

func parseResponse(raw []byte) parsed {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return parsed{}
	}
	if strings.HasPrefix(text, "data:") || strings.Contains(text, "\ndata:") {
		return parseSSE(text)
	}
	return parseJSON(raw)
}

func parseJSON(raw []byte) parsed {
	var body struct {
		ID     string `json:"id"`
		Output []struct {
			Type    string `json:"type"`
			Name    string `json:"name"`
			CallID  string `json:"call_id"`
			Args    string `json:"arguments"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if json.Unmarshal(raw, &body) != nil {
		return parsed{}
	}
	out := parsed{ID: body.ID}
	var b strings.Builder
	for _, item := range body.Output {
		if item.Type == "function_call" {
			out.Calls = append(out.Calls, call{Name: item.Name, CallID: item.CallID, Arguments: item.Args})
		}
		for _, part := range item.Content {
			if part.Text != "" {
				b.WriteString(part.Text)
			}
		}
	}
	out.Text = b.String()
	return out
}

func parseSSE(text string) parsed {
	out := parsed{}
	var b strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var event map[string]any
		if json.Unmarshal([]byte(data), &event) != nil {
			continue
		}
		if id, ok := event["id"].(string); ok && id != "" {
			out.ID = id
		}
		if response, ok := event["response"].(map[string]any); ok {
			if id, ok := response["id"].(string); ok {
				out.ID = id
			}
		}
		switch event["type"] {
		case "response.output_text.delta":
			if delta, ok := event["delta"].(string); ok {
				b.WriteString(delta)
			}
		case "response.function_call":
			out.Calls = append(out.Calls, callFrom(event))
		case "response.output_item.done":
			if item, ok := event["item"].(map[string]any); ok && item["type"] == "function_call" {
				out.Calls = append(out.Calls, callFrom(item))
			}
		}
	}
	out.Text = b.String()
	return out
}

func callFrom(item map[string]any) call {
	name, _ := item["name"].(string)
	id, _ := item["call_id"].(string)
	args, _ := item["arguments"].(string)
	if id == "" {
		id = name
	}
	return call{Name: name, CallID: id, Arguments: args}
}

func toolDefs() []map[string]any {
	names := []struct{ name, desc string }{
		{"host_overview", "读取主机概览"},
		{"host_processes", "读取进程占用"},
		{"list_sites", "读取网站列表"},
		{"list_certificates", "读取证书列表"},
		{"list_apps", "读取已安装应用"},
		{"list_containers", "读取容器列表"},
		{"log_tail", "读取容器日志末尾"},
		{"list_jobs", "读取计划任务"},
		{"restart_container", "重启一个容器，必须先得到确认"},
		{"reload_nginx", "重载 Nginx，必须先得到确认"},
		{"run_scheduled_job", "运行一个已有计划任务，必须先得到确认"},
	}
	out := make([]map[string]any, 0, len(names))
	for _, item := range names {
		props := map[string]any{}
		if item.name == "log_tail" || item.name == "restart_container" {
			props["id"] = map[string]string{"type": "string"}
		}
		if item.name == "run_scheduled_job" {
			props["id"] = map[string]string{"type": "string"}
		}
		out = append(out, map[string]any{
			"type": "function", "name": item.name, "description": item.desc,
			"parameters": map[string]any{"type": "object", "properties": props},
		})
	}
	return out
}

func (t Tools) read(ctx context.Context, item call) (string, error) {
	var args struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal([]byte(item.Arguments), &args)
	switch item.Name {
	case "host_overview":
		if t.Host == nil {
			return "", fmt.Errorf("没有主机数据")
		}
		view, err := t.Host.Overview(time.Now())
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(view)
		return string(b), nil
	case "host_processes":
		if t.Host == nil {
			return "", fmt.Errorf("没有主机数据")
		}
		list, err := t.Host.Processes(time.Now(), 10, "cpu")
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(list)
		return string(b), nil
	case "list_sites":
		if t.Sites == nil {
			return "[]", nil
		}
		list, err := t.Sites.List(ctx)
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(list)
		return string(b), nil
	case "list_certificates":
		if t.Certs == nil {
			return "[]", nil
		}
		list, err := t.Certs.List()
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(list)
		return string(b), nil
	case "list_apps":
		if t.Store == nil {
			return "[]", nil
		}
		list, err := t.Store.ListInstalled(ctx)
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(list)
		return string(b), nil
	case "list_containers":
		if t.Docker == nil {
			return "[]", nil
		}
		snap, err := t.Docker.Snapshot(ctx)
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(snap.Containers)
		return string(b), nil
	case "log_tail":
		if t.Docker == nil {
			return "", fmt.Errorf("没有容器")
		}
		logs, err := t.Docker.Logs(ctx, args.ID)
		if err != nil {
			return "", err
		}
		lines := strings.Split(logs, "\n")
		if len(lines) > 50 {
			lines = lines[len(lines)-50:]
		}
		return strings.Join(lines, "\n"), nil
	case "list_jobs":
		if t.Cron == nil {
			return "[]", nil
		}
		b, _ := json.Marshal(t.Cron.List())
		return string(b), nil
	default:
		return "", fmt.Errorf("未知的只读工具")
	}
}

func (t Tools) write(ctx context.Context, item call) (string, error) {
	var args struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal([]byte(item.Arguments), &args)
	switch item.Name {
	case "restart_container":
		if err := t.Docker.Action(ctx, args.ID, "restart"); err != nil {
			return "", err
		}
		_ = t.Audit.Append(audit.Record{Actor: t.Actor, Method: "POST", Path: "/api/v1/docker/containers/" + args.ID + "/restart", Status: 200, Detail: "ai"})
		return "restarted", nil
	case "reload_nginx":
		if t.Nginx != nil {
			if err := t.Nginx.Reload(ctx); err != nil {
				return "", err
			}
		}
		_ = t.Audit.Append(audit.Record{Actor: t.Actor, Method: "POST", Path: "/api/v1/nginx/action", Status: 200, Detail: "reload"})
		return "reloaded", nil
	case "run_scheduled_job":
		if t.Cron == nil {
			return "", fmt.Errorf("没有计划任务")
		}
		run, err := t.Cron.RunNow(ctx, args.ID)
		if err != nil {
			return "", err
		}
		_ = t.Audit.Append(audit.Record{Actor: t.Actor, Method: "POST", Path: "/api/v1/schedules/" + args.ID + "/run", Status: 200, Detail: "ai"})
		b, _ := json.Marshal(run)
		return string(b), nil
	default:
		return "", fmt.Errorf("不允许的写入工具")
	}
}
