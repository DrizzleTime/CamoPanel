package app

import (
	"encoding/json"
	"regexp"
	"strings"

	"camopanel/server/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var projectNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

func (a *App) recordAudit(actorID, action, targetType, targetID string, metadata map[string]any) error {
	rawMetadata := "{}"
	if metadata != nil {
		payload, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		rawMetadata = string(payload)
	}

	event := model.AuditEvent{
		ID:           uuid.NewString(),
		ActorID:      actorID,
		Action:       action,
		TargetType:   targetType,
		TargetID:     targetID,
		MetadataJSON: rawMetadata,
	}
	return a.db.Create(&event).Error
}

func writeError(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}

func writeSSE(c *gin.Context, event string, payload any) {
	raw, _ := json.Marshal(payload)
	_, _ = c.Writer.WriteString("event: " + event + "\n")
	_, _ = c.Writer.WriteString("data: " + string(raw) + "\n\n")
	c.Writer.Flush()
}

func chunkText(text string, size int) []string {
	if len(text) <= size {
		return []string{text}
	}

	chunks := []string{}
	runes := []rune(text)
	for len(runes) > size {
		chunks = append(chunks, string(runes[:size]))
		runes = runes[size:]
	}
	if len(runes) > 0 {
		chunks = append(chunks, string(runes))
	}
	return chunks
}

func normalizeProjectName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func chineseAction(action string) string {
	switch action {
	case model.ApprovalActionStart:
		return "启动"
	case model.ApprovalActionStop:
		return "停止"
	case model.ApprovalActionRestart:
		return "重启"
	case model.ApprovalActionDelete:
		return "删除"
	case model.ApprovalActionRedeploy:
		return "重新部署"
	default:
		return action
	}
}

func websiteTypeLabel(websiteType string) string {
	switch websiteType {
	case model.WebsiteTypeStatic:
		return "静态站点"
	case model.WebsiteTypeProxy:
		return "反向代理"
	default:
		return websiteType
	}
}
