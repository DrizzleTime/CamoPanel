package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"camopanel/server/internal/config"
	"github.com/google/uuid"
)

var ErrCopilotDisabled = errors.New("copilot is not configured")

type SessionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CopilotSession struct {
	ID       string           `json:"id"`
	Messages []SessionMessage `json:"messages"`
}

type ProposedAction struct {
	Action      string         `json:"action"`
	TemplateID  string         `json:"template_id,omitempty"`
	ProjectID   string         `json:"project_id,omitempty"`
	ProjectName string         `json:"project_name,omitempty"`
	Summary     string         `json:"summary,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type CopilotReply struct {
	Message        string          `json:"message"`
	ProposedAction *ProposedAction `json:"proposed_action,omitempty"`
}

type ProjectToolData struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	TemplateID      string             `json:"template_id"`
	TemplateVersion string             `json:"template_version"`
	Status          string             `json:"status"`
	LastError       string             `json:"last_error"`
	Containers      []ProjectContainer `json:"containers,omitempty"`
}

type CopilotToolbox interface {
	ListTemplates() []TemplateSpec
	GetTemplate(templateID string) (*LoadedTemplate, error)
	ListProjects(ctx context.Context) ([]ProjectToolData, error)
	GetProject(ctx context.Context, projectID string) (ProjectToolData, error)
	GetProjectLogs(ctx context.Context, projectID string, tail int) (string, error)
	GetHostSummary(ctx context.Context) (HostSummary, error)
}

type CopilotService struct {
	cfg      config.AIConfig
	client   *http.Client
	toolbox  CopilotToolbox
	mu       sync.Mutex
	sessions map[string]*CopilotSession
}

func NewCopilotService(cfg config.AIConfig, toolbox CopilotToolbox) *CopilotService {
	return &CopilotService{
		cfg:      cfg,
		client:   &http.Client{},
		toolbox:  toolbox,
		sessions: map[string]*CopilotSession{},
	}
}

func (s *CopilotService) CreateSession() CopilotSession {
	session := &CopilotSession{
		ID:       uuid.NewString(),
		Messages: []SessionMessage{},
	}
	s.mu.Lock()
	s.sessions[session.ID] = session
	s.mu.Unlock()
	return *session
}

func (s *CopilotService) Reply(ctx context.Context, sessionID, userMessage string) (CopilotReply, error) {
	if s.cfg.BaseURL == "" || s.cfg.APIKey == "" || s.cfg.Model == "" {
		return CopilotReply{}, ErrCopilotDisabled
	}

	session, err := s.session(sessionID)
	if err != nil {
		return CopilotReply{}, err
	}

	conversation := append([]SessionMessage{}, session.Messages...)
	conversation = append(conversation, SessionMessage{Role: "user", Content: userMessage})

	result, history, err := s.runToolLoop(ctx, conversation)
	if err != nil {
		return CopilotReply{}, err
	}

	s.mu.Lock()
	session.Messages = history
	s.mu.Unlock()

	return result, nil
}

func (s *CopilotService) session(sessionID string) (*CopilotSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}
	return session, nil
}

func (s *CopilotService) runToolLoop(ctx context.Context, messages []SessionMessage) (CopilotReply, []SessionMessage, error) {
	history := []chatMessage{{Role: "system", Content: systemPrompt}}
	for _, message := range messages {
		history = append(history, chatMessage{
			Role:    message.Role,
			Content: message.Content,
		})
	}

	persistentHistory := append([]SessionMessage{}, messages...)

	var assistantReply string
	for i := 0; i < 4; i++ {
		response, err := s.chatCompletion(ctx, history)
		if err != nil {
			return CopilotReply{}, nil, err
		}

		message := response.Choices[0].Message
		if len(message.ToolCalls) == 0 {
			assistantReply = message.Content
			history = append(history, chatMessage{Role: "assistant", Content: assistantReply})
			persistentHistory = append(persistentHistory, SessionMessage{Role: "assistant", Content: assistantReply})
			break
		}

		history = append(history, chatMessage{
			Role:      "assistant",
			Content:   message.Content,
			ToolCalls: message.ToolCalls,
		})
		for _, toolCall := range message.ToolCalls {
			result, err := s.handleToolCall(ctx, toolCall)
			if err != nil {
				result = map[string]any{"error": err.Error()}
			}
			raw, _ := json.Marshal(result)
			history = append(history, chatMessage{
				Role:       "tool",
				ToolCallID: toolCall.ID,
				Content:    string(raw),
			})
		}
	}

	if assistantReply == "" {
		return CopilotReply{}, nil, fmt.Errorf("copilot produced no final response")
	}

	reply := CopilotReply{}
	if err := json.Unmarshal([]byte(assistantReply), &reply); err != nil {
		reply.Message = assistantReply
	}

	if reply.Message == "" {
		reply.Message = assistantReply
	}

	return reply, persistentHistory, nil
}

func (s *CopilotService) handleToolCall(ctx context.Context, toolCall chatToolCall) (any, error) {
	var args map[string]any
	if toolCall.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			return nil, fmt.Errorf("invalid tool args: %w", err)
		}
	}

	switch toolCall.Function.Name {
	case "list_templates":
		return s.toolbox.ListTemplates(), nil
	case "get_template_detail":
		templateID, _ := args["template_id"].(string)
		template, err := s.toolbox.GetTemplate(templateID)
		if err != nil {
			return nil, err
		}
		return template.Spec, nil
	case "list_projects":
		return s.toolbox.ListProjects(ctx)
	case "get_project_detail":
		projectID, _ := args["project_id"].(string)
		return s.toolbox.GetProject(ctx, projectID)
	case "get_project_logs":
		projectID, _ := args["project_id"].(string)
		tail := 200
		if value, ok := args["tail"].(float64); ok && value > 0 {
			tail = int(value)
		}
		return s.toolbox.GetProjectLogs(ctx, projectID, tail)
	case "get_host_metrics":
		return s.toolbox.GetHostSummary(ctx)
	default:
		return nil, fmt.Errorf("unknown tool %s", toolCall.Function.Name)
	}
}

func (s *CopilotService) chatCompletion(ctx context.Context, messages []chatMessage) (chatCompletionResponse, error) {
	body := chatCompletionRequest{
		Model:       s.cfg.Model,
		Messages:    messages,
		Tools:       toolSchemas,
		Temperature: 0.2,
	}

	rawBody, err := json.Marshal(body)
	if err != nil {
		return chatCompletionResponse{}, err
	}

	url := strings.TrimRight(s.cfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawBody))
	if err != nil {
		return chatCompletionResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return chatCompletionResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return chatCompletionResponse{}, fmt.Errorf("copilot request failed: %s", strings.TrimSpace(string(raw)))
	}

	var payload chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return chatCompletionResponse{}, err
	}

	if len(payload.Choices) == 0 {
		return chatCompletionResponse{}, fmt.Errorf("copilot returned empty choices")
	}

	return payload, nil
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Tools       []chatTool    `json:"tools,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
}

type chatMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolCalls  []chatToolCall `json:"tool_calls,omitempty"`
}

type chatCompletionResponse struct {
	Choices []chatChoice `json:"choices"`
}

type chatChoice struct {
	Message chatCompletionMessage `json:"message"`
}

type chatCompletionMessage struct {
	Content   string         `json:"content"`
	ToolCalls []chatToolCall `json:"tool_calls"`
}

type chatToolCall struct {
	ID       string               `json:"id"`
	Type     string               `json:"type"`
	Function chatToolCallFunction `json:"function"`
}

type chatToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatTool struct {
	Type     string             `json:"type"`
	Function chatToolDefinition `json:"function"`
}

type chatToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

var toolSchemas = []chatTool{
	{
		Type: "function",
		Function: chatToolDefinition{
			Name:        "list_templates",
			Description: "列出当前所有可部署模板",
			Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		},
	},
	{
		Type: "function",
		Function: chatToolDefinition{
			Name:        "get_template_detail",
			Description: "获取指定模板详情",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"template_id": map[string]any{"type": "string"},
				},
				"required": []string{"template_id"},
			},
		},
	},
	{
		Type: "function",
		Function: chatToolDefinition{
			Name:        "list_projects",
			Description: "列出当前项目",
			Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		},
	},
	{
		Type: "function",
		Function: chatToolDefinition{
			Name:        "get_project_detail",
			Description: "获取项目详情和容器状态",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "string"},
				},
				"required": []string{"project_id"},
			},
		},
	},
	{
		Type: "function",
		Function: chatToolDefinition{
			Name:        "get_project_logs",
			Description: "读取项目最近日志",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "string"},
					"tail":       map[string]any{"type": "number"},
				},
				"required": []string{"project_id"},
			},
		},
	},
	{
		Type: "function",
		Function: chatToolDefinition{
			Name:        "get_host_metrics",
			Description: "获取主机资源概览",
			Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		},
	},
}

const systemPrompt = `你是 CamoPanel 的运维 Copilot。
你只能做只读分析，不能直接执行写操作。
你可以通过工具读取模板、项目、日志和主机信息。
如果你认为应该让用户执行某个动作，必须返回 JSON，格式如下：
{
  "message": "给用户看的自然语言说明",
  "proposed_action": {
    "action": "deploy|start|stop|restart|delete|redeploy",
    "template_id": "部署时需要",
    "project_id": "操作已有项目时需要",
    "project_name": "部署时建议使用的项目名",
    "summary": "一段简短执行摘要",
    "parameters": { "任意参数": "值" }
  }
}
如果只是答疑或诊断，不需要 proposed_action，请返回：
{
  "message": "你的回答"
}
所有输出必须是有效 JSON，不要输出 Markdown 代码块。`
