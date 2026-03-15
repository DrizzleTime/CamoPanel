package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"camopanel/server/internal/model"

	"github.com/google/uuid"
)

type deployApprovalPayload struct {
	Name       string         `json:"name"`
	TemplateID string         `json:"template_id"`
	Parameters map[string]any `json:"parameters"`
}

type projectActionPayload struct {
	ProjectID string `json:"project_id"`
	Action    string `json:"action"`
}

func (a *App) createDeployApproval(actorID, source string, req createProjectRequest) (model.ApprovalRequest, error) {
	normalizedName := normalizeProjectName(req.Name)
	if !projectNamePattern.MatchString(normalizedName) {
		return model.ApprovalRequest{}, fmt.Errorf("项目名只能包含小写字母、数字、下划线和中划线")
	}

	var count int64
	if err := a.db.Model(&model.Project{}).Where("name = ?", normalizedName).Count(&count).Error; err != nil {
		return model.ApprovalRequest{}, err
	}
	if count > 0 {
		return model.ApprovalRequest{}, fmt.Errorf("项目名已存在")
	}

	templateItem, err := a.templates.Get(req.TemplateID)
	if err != nil {
		return model.ApprovalRequest{}, err
	}

	normalized, err := templateItem.ValidateAndNormalize(req.Parameters)
	if err != nil {
		return model.ApprovalRequest{}, err
	}
	if _, err := templateItem.Render(normalized); err != nil {
		return model.ApprovalRequest{}, err
	}

	payload := deployApprovalPayload{
		Name:       normalizedName,
		TemplateID: templateItem.Spec.ID,
		Parameters: normalized,
	}
	return a.saveApproval(actorID, source, model.ApprovalActionDeploy, "project", normalizedName, payload,
		fmt.Sprintf("部署项目 %s（模板 %s）", normalizedName, templateItem.Spec.Name))
}

func (a *App) createProjectActionApproval(actorID, source string, project model.Project, action string) (model.ApprovalRequest, error) {
	switch action {
	case model.ApprovalActionStart, model.ApprovalActionStop, model.ApprovalActionRestart, model.ApprovalActionDelete, model.ApprovalActionRedeploy:
	default:
		return model.ApprovalRequest{}, fmt.Errorf("不支持的动作: %s", action)
	}

	payload := projectActionPayload{ProjectID: project.ID, Action: action}
	summary := fmt.Sprintf("%s 项目 %s", chineseAction(action), project.Name)
	return a.saveApproval(actorID, source, action, "project", project.ID, payload, summary)
}

func (a *App) executeDeploy(ctx context.Context, payload deployApprovalPayload) error {
	var count int64
	if err := a.db.Model(&model.Project{}).Where("name = ?", payload.Name).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("项目名已存在")
	}

	templateItem, err := a.templates.Get(payload.TemplateID)
	if err != nil {
		return err
	}
	normalized, err := templateItem.ValidateAndNormalize(payload.Parameters)
	if err != nil {
		return err
	}
	rendered, err := templateItem.Render(normalized)
	if err != nil {
		return err
	}

	projectID := uuid.NewString()
	projectDir := filepath.Join(a.cfg.ProjectsDir, projectID)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return err
	}

	composePath := filepath.Join(projectDir, "compose.yaml")
	if err := os.WriteFile(composePath, []byte(rendered), 0o644); err != nil {
		return err
	}

	if err := a.executor.Deploy(ctx, payload.Name, composePath); err != nil {
		return err
	}

	configJSON, err := templateItem.ConfigJSON(normalized)
	if err != nil {
		return err
	}

	project := model.Project{
		ID:              projectID,
		Name:            payload.Name,
		TemplateID:      templateItem.Spec.ID,
		TemplateVersion: templateItem.Spec.Version,
		ConfigJSON:      configJSON,
		ComposePath:     composePath,
		Status:          "running",
	}

	if runtimeInfo, runtimeErr := a.executor.InspectProject(ctx, payload.Name); runtimeErr == nil {
		project.Status = runtimeInfo.Status
	}

	if err := a.db.Create(&project).Error; err != nil {
		return err
	}
	return nil
}

func (a *App) executeProjectAction(ctx context.Context, payload projectActionPayload) error {
	project, err := a.findProject(payload.ProjectID)
	if err != nil {
		return err
	}

	switch payload.Action {
	case model.ApprovalActionStart:
		err = a.executor.Start(ctx, project.Name, project.ComposePath)
	case model.ApprovalActionStop:
		err = a.executor.Stop(ctx, project.Name, project.ComposePath)
	case model.ApprovalActionRestart:
		err = a.executor.Restart(ctx, project.Name, project.ComposePath)
	case model.ApprovalActionRedeploy:
		err = a.executor.Redeploy(ctx, project.Name, project.ComposePath)
	case model.ApprovalActionDelete:
		err = a.executor.Delete(ctx, project.Name, project.ComposePath)
	default:
		err = fmt.Errorf("未知项目动作: %s", payload.Action)
	}
	if err != nil {
		project.LastError = err.Error()
		_ = a.db.Save(&project).Error
		return err
	}

	if payload.Action == model.ApprovalActionDelete {
		_ = os.RemoveAll(filepath.Dir(project.ComposePath))
		return a.db.Delete(&project).Error
	}

	project.LastError = ""
	switch payload.Action {
	case model.ApprovalActionStop:
		project.Status = "stopped"
	default:
		project.Status = "running"
	}
	if runtimeInfo, runtimeErr := a.executor.InspectProject(ctx, project.Name); runtimeErr == nil {
		project.Status = runtimeInfo.Status
	}
	return a.db.Save(&project).Error
}
