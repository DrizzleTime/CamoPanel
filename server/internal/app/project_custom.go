package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"camopanel/server/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	customComposeTemplateID      = "custom-compose"
	customComposeTemplateVersion = "1"
)

type createCustomProjectRequest struct {
	Name    string `json:"name"`
	Compose string `json:"compose"`
}

func (a *App) handleCreateCustomProject(c *gin.Context) {
	var req createCustomProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	project, err := a.createCustomProject(c.Request.Context(), currentUser(c).ID, req)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	item, err := a.projectToResponse(c.Request.Context(), project)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"project": item})
}

func (a *App) createCustomProject(ctx context.Context, actorID string, req createCustomProjectRequest) (model.Project, error) {
	normalizedName := normalizeProjectName(req.Name)
	if !projectNamePattern.MatchString(normalizedName) {
		return model.Project{}, fmt.Errorf("项目名只能包含小写字母、数字、下划线和中划线")
	}

	composeContent := strings.TrimSpace(req.Compose)
	if composeContent == "" {
		return model.Project{}, fmt.Errorf("Compose 内容不能为空")
	}
	composeContent += "\n"

	var count int64
	if err := a.db.Model(&model.Project{}).Where("name = ?", normalizedName).Count(&count).Error; err != nil {
		return model.Project{}, err
	}
	if count > 0 {
		return model.Project{}, fmt.Errorf("项目名已存在")
	}

	projectID := uuid.NewString()
	projectDir := filepath.Join(a.cfg.ProjectsDir, projectID)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return model.Project{}, err
	}

	composePath := filepath.Join(projectDir, "compose.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0o644); err != nil {
		return model.Project{}, err
	}

	if err := a.executor.Deploy(ctx, normalizedName, composePath); err != nil {
		return model.Project{}, err
	}

	configJSON, err := json.Marshal(map[string]any{"mode": customComposeTemplateID})
	if err != nil {
		return model.Project{}, err
	}

	project := model.Project{
		ID:              projectID,
		Name:            normalizedName,
		TemplateID:      customComposeTemplateID,
		TemplateVersion: customComposeTemplateVersion,
		ConfigJSON:      string(configJSON),
		ComposePath:     composePath,
		Status:          "running",
	}
	if runtimeInfo, runtimeErr := a.executor.InspectProject(ctx, normalizedName); runtimeErr == nil {
		project.Status = runtimeInfo.Status
	}
	if err := a.db.Create(&project).Error; err != nil {
		return model.Project{}, err
	}

	_ = a.recordAudit(actorID, "project_deploy", "project", project.ID, map[string]any{
		"name":        project.Name,
		"template_id": project.TemplateID,
	})
	return project, nil
}
