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

func (a *App) handleCopilotConfig(c *gin.Context) {
	status, err := a.copilotConfigStatus(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, status)
}

func (a *App) handleCopilotProviders(c *gin.Context) {
	items, err := a.listCopilotProviderResponses()
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (a *App) handleCreateCopilotProvider(c *gin.Context) {
	var req createCopilotProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	item, err := a.createCopilotProvider(currentUser(c).ID, req)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"provider": item})
}

func (a *App) handleUpdateCopilotProvider(c *gin.Context) {
	provider, err := a.findCopilotProvider(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	var req updateCopilotProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	item, err := a.updateCopilotProvider(currentUser(c).ID, provider, req)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"provider": item})
}

func (a *App) handleDeleteCopilotProvider(c *gin.Context) {
	provider, err := a.findCopilotProvider(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	if err := a.deleteCopilotProvider(currentUser(c).ID, provider); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (a *App) handleCreateCopilotModel(c *gin.Context) {
	provider, err := a.findCopilotProvider(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	var req createCopilotModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	item, err := a.createCopilotModel(currentUser(c).ID, provider, req)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"model": item})
}

func (a *App) handleUpdateCopilotModel(c *gin.Context) {
	aiModel, err := a.findCopilotModel(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	var req updateCopilotModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	item, err := a.updateCopilotModel(currentUser(c).ID, aiModel, req)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"model": item})
}

func (a *App) handleDeleteCopilotModel(c *gin.Context) {
	aiModel, err := a.findCopilotModel(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	if err := a.deleteCopilotModel(currentUser(c).ID, aiModel); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
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
