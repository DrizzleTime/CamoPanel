package app

import (
	"errors"
	"net/http"

	"camopanel/server/internal/services"

	"github.com/gin-gonic/gin"
)

type createProjectRequest struct {
	Name       string         `json:"name"`
	TemplateID string         `json:"template_id"`
	Parameters map[string]any `json:"parameters"`
}

type projectActionRequest struct {
	Action string `json:"action"`
}

func (a *App) handleProjects(c *gin.Context) {
	items, err := a.listProjectResponses(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (a *App) handleCreateProject(c *gin.Context) {
	var req createProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	approval, err := a.createDeployApproval(currentUser(c).ID, "ui", req)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"approval": approval})
}

func (a *App) handleProject(c *gin.Context) {
	project, err := a.findProject(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	item, err := a.projectToResponse(c.Request.Context(), project)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (a *App) handleProjectAction(c *gin.Context) {
	project, err := a.findProject(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	var req projectActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	approval, err := a.createProjectActionApproval(currentUser(c).ID, "ui", project, req.Action)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"approval": approval})
}

func (a *App) handleProjectLogs(c *gin.Context) {
	project, err := a.findProject(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	logs, err := a.executor.ProjectLogs(c.Request.Context(), project.Name, 200)
	if err != nil {
		if errors.Is(err, services.ErrDockerUnavailable) {
			writeError(c, http.StatusBadGateway, "Docker 当前不可用")
			return
		}
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}
