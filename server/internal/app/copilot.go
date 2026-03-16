package app

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type copilotMessageRequest struct {
	Message string `json:"message"`
}

func (a *App) handleCreateCopilotSession(c *gin.Context) {
	session := a.copilot.CreateSession()
	c.JSON(http.StatusCreated, session)
}

func (a *App) handleCopilotMessage(c *gin.Context) {
	var req copilotMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		writeError(c, http.StatusBadRequest, "消息不能为空")
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	reply, err := a.copilot.Reply(c.Request.Context(), c.Param("id"), req.Message)
	if err != nil {
		writeSSE(c, "error", gin.H{"error": err.Error()})
		return
	}

	for _, chunk := range chunkText(reply.Message, 120) {
		writeSSE(c, "chunk", gin.H{"content": chunk})
	}

	writeSSE(c, "done", gin.H{"message": reply.Message})
}
