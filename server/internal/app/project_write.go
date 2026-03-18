package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"camopanel/server/internal/model"
	"camopanel/server/internal/services"

	"github.com/google/uuid"
)

const (
	managedOpenRestyTemplateID = "openresty"
	managedOpenRestyProjectID  = "openresty"
)

type deployPayload struct {
	Name       string         `json:"name"`
	TemplateID string         `json:"template_id"`
	Parameters map[string]any `json:"parameters"`
}

type projectActionPayload struct {
	ProjectID string `json:"project_id"`
	Action    string `json:"action"`
}

func (a *App) createProject(ctx context.Context, actorID string, req createProjectRequest) (model.Project, error) {
	payload, err := a.prepareDeployPayload(req)
	if err != nil {
		return model.Project{}, err
	}

	project, err := a.executeDeploy(ctx, payload)
	if err != nil {
		return model.Project{}, err
	}

	_ = a.recordAudit(actorID, "project_deploy", "project", project.ID, map[string]any{
		"name":        project.Name,
		"template_id": project.TemplateID,
	})
	return project, nil
}

func (a *App) prepareDeployPayload(req createProjectRequest) (deployPayload, error) {
	normalizedName := normalizeProjectName(req.Name)
	if req.TemplateID == managedOpenRestyTemplateID {
		normalizedName = managedOpenRestyProjectID
	}
	if !projectNamePattern.MatchString(normalizedName) {
		return deployPayload{}, fmt.Errorf("项目名只能包含小写字母、数字、下划线和中划线")
	}

	var count int64
	if err := a.db.Model(&model.Project{}).Where("name = ?", normalizedName).Count(&count).Error; err != nil {
		return deployPayload{}, err
	}
	if count > 0 {
		return deployPayload{}, fmt.Errorf("项目名已存在")
	}
	if req.TemplateID == managedOpenRestyTemplateID {
		if err := a.ensureManagedOpenRestyAvailable(); err != nil {
			return deployPayload{}, err
		}
	}

	templateItem, err := a.templates.Get(req.TemplateID)
	if err != nil {
		return deployPayload{}, err
	}

	normalized, err := templateItem.ValidateAndNormalize(req.Parameters)
	if err != nil {
		return deployPayload{}, err
	}
	if _, err := templateItem.Render(normalized, a.templateRuntime(normalizedName)); err != nil {
		return deployPayload{}, err
	}

	return deployPayload{
		Name:       normalizedName,
		TemplateID: templateItem.Spec.ID,
		Parameters: normalized,
	}, nil
}

func (a *App) runProjectAction(ctx context.Context, actorID string, project model.Project, action string) error {
	switch action {
	case model.ActionStart, model.ActionStop, model.ActionRestart, model.ActionDelete, model.ActionRedeploy:
	default:
		return fmt.Errorf("不支持的动作: %s", action)
	}

	payload := projectActionPayload{ProjectID: project.ID, Action: action}
	if err := a.executeProjectAction(ctx, payload); err != nil {
		return err
	}

	_ = a.recordAudit(actorID, "project_"+action, "project", project.ID, map[string]any{
		"name":   project.Name,
		"action": action,
	})
	return nil
}

func (a *App) executeDeploy(ctx context.Context, payload deployPayload) (model.Project, error) {
	var count int64
	if err := a.db.Model(&model.Project{}).Where("name = ?", payload.Name).Count(&count).Error; err != nil {
		return model.Project{}, err
	}
	if count > 0 {
		return model.Project{}, fmt.Errorf("项目名已存在")
	}
	if payload.TemplateID == managedOpenRestyTemplateID {
		if err := a.ensureManagedOpenRestyAvailable(); err != nil {
			return model.Project{}, err
		}
	}

	templateItem, err := a.templates.Get(payload.TemplateID)
	if err != nil {
		return model.Project{}, err
	}
	normalized, err := templateItem.ValidateAndNormalize(payload.Parameters)
	if err != nil {
		return model.Project{}, err
	}
	rendered, err := templateItem.Render(normalized, a.templateRuntime(payload.Name))
	if err != nil {
		return model.Project{}, err
	}

	projectID := uuid.NewString()
	projectDir := filepath.Join(a.cfg.ProjectsDir, projectID)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return model.Project{}, err
	}

	composePath := filepath.Join(projectDir, "compose.yaml")
	if err := os.WriteFile(composePath, []byte(rendered), 0o644); err != nil {
		return model.Project{}, err
	}

	if err := a.ensureProjectBridgeNetwork(ctx, payload.TemplateID); err != nil {
		return model.Project{}, err
	}
	if err := a.executor.Deploy(ctx, payload.Name, composePath); err != nil {
		return model.Project{}, err
	}

	configJSON, err := templateItem.ConfigJSON(normalized)
	if err != nil {
		return model.Project{}, err
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
		return model.Project{}, err
	}
	return project, nil
}

func (a *App) templateRuntime(projectName string) services.TemplateRuntime {
	return services.TemplateRuntime{
		ProjectName:          projectName,
		BridgeNetworkName:    a.cfg.BridgeNetworkName,
		OpenRestyContainer:   a.cfg.OpenRestyContainer,
		OpenRestyHostConfDir: filepath.Join(a.cfg.OpenRestyDataDir, "conf.d"),
		OpenRestyHostSiteDir: filepath.Join(a.cfg.OpenRestyDataDir, "www"),
		OpenRestyHostCertDir: filepath.Join(a.cfg.OpenRestyDataDir, "certs"),
	}
}

func (a *App) ensureProjectBridgeNetwork(ctx context.Context, templateID string) error {
	if templateID == managedOpenRestyTemplateID {
		return nil
	}
	return a.executor.EnsureNetwork(ctx, a.cfg.BridgeNetworkName, "bridge")
}

func (a *App) ensureManagedOpenRestyAvailable() error {
	var count int64
	if err := a.db.Model(&model.Project{}).Where("template_id = ?", managedOpenRestyTemplateID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("固定 OpenResty 已存在，不支持重复部署")
	}
	return nil
}

func (a *App) executeProjectAction(ctx context.Context, payload projectActionPayload) error {
	project, err := a.findProject(payload.ProjectID)
	if err != nil {
		return err
	}

	switch payload.Action {
	case model.ActionStart:
		if err = a.ensureProjectBridgeNetwork(ctx, project.TemplateID); err == nil {
			err = a.executor.Start(ctx, project.Name, project.ComposePath)
		}
	case model.ActionStop:
		err = a.executor.Stop(ctx, project.Name, project.ComposePath)
	case model.ActionRestart:
		if err = a.ensureProjectBridgeNetwork(ctx, project.TemplateID); err == nil {
			err = a.executor.Restart(ctx, project.Name, project.ComposePath)
		}
	case model.ActionRedeploy:
		if err = a.ensureProjectBridgeNetwork(ctx, project.TemplateID); err == nil {
			err = a.executor.Redeploy(ctx, project.Name, project.ComposePath)
		}
	case model.ActionDelete:
		err = a.executor.Delete(ctx, project.Name, project.ComposePath)
	default:
		err = fmt.Errorf("未知项目动作: %s", payload.Action)
	}
	if err != nil {
		project.LastError = err.Error()
		_ = a.db.Save(&project).Error
		return err
	}

	if payload.Action == model.ActionDelete {
		_ = os.RemoveAll(filepath.Dir(project.ComposePath))
		return a.db.Delete(&project).Error
	}

	project.LastError = ""
	switch payload.Action {
	case model.ActionStop:
		project.Status = "stopped"
	default:
		project.Status = "running"
	}
	if runtimeInfo, runtimeErr := a.executor.InspectProject(ctx, project.Name); runtimeErr == nil {
		project.Status = runtimeInfo.Status
	}
	return a.db.Save(&project).Error
}
