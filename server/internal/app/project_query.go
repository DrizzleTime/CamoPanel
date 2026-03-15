package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"camopanel/server/internal/model"
	"camopanel/server/internal/services"
)

type projectResponse struct {
	ID              string                  `json:"id"`
	Name            string                  `json:"name"`
	TemplateID      string                  `json:"template_id"`
	TemplateVersion string                  `json:"template_version"`
	Config          map[string]any          `json:"config"`
	ComposePath     string                  `json:"compose_path"`
	Status          string                  `json:"status"`
	LastError       string                  `json:"last_error"`
	Runtime         services.ProjectRuntime `json:"runtime"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

func (a *App) listProjectResponses(ctx context.Context) ([]projectResponse, error) {
	var projects []model.Project
	if err := a.db.Order("created_at desc").Find(&projects).Error; err != nil {
		return nil, err
	}

	items := make([]projectResponse, 0, len(projects))
	for _, project := range projects {
		item, err := a.projectToResponse(ctx, project)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (a *App) projectToResponse(ctx context.Context, project model.Project) (projectResponse, error) {
	configMap := map[string]any{}
	_ = json.Unmarshal([]byte(project.ConfigJSON), &configMap)

	runtimeInfo, err := a.executor.InspectProject(ctx, project.Name)
	if err != nil {
		runtimeInfo = services.ProjectRuntime{Status: "docker_unavailable", Containers: []services.ProjectContainer{}}
	}

	if runtimeInfo.Status != "" && runtimeInfo.Status != "docker_unavailable" && project.Status != runtimeInfo.Status {
		project.Status = runtimeInfo.Status
		project.LastError = ""
		_ = a.db.Model(&model.Project{}).
			Where("id = ?", project.ID).
			Updates(map[string]any{"status": project.Status, "last_error": project.LastError}).Error
	}

	return projectResponse{
		ID:              project.ID,
		Name:            project.Name,
		TemplateID:      project.TemplateID,
		TemplateVersion: project.TemplateVersion,
		Config:          configMap,
		ComposePath:     project.ComposePath,
		Status:          project.Status,
		LastError:       project.LastError,
		Runtime:         runtimeInfo,
		CreatedAt:       project.CreatedAt,
		UpdatedAt:       project.UpdatedAt,
	}, nil
}

func (a *App) ListProjects(ctx context.Context) ([]services.ProjectToolData, error) {
	var projects []model.Project
	if err := a.db.Order("created_at desc").Find(&projects).Error; err != nil {
		return nil, err
	}

	items := make([]services.ProjectToolData, 0, len(projects))
	for _, project := range projects {
		item, err := a.projectToolData(ctx, project)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (a *App) GetProject(ctx context.Context, projectID string) (services.ProjectToolData, error) {
	project, err := a.findProject(projectID)
	if err != nil {
		return services.ProjectToolData{}, err
	}
	return a.projectToolData(ctx, project)
}

func (a *App) GetProjectLogs(ctx context.Context, projectID string, tail int) (string, error) {
	project, err := a.findProject(projectID)
	if err != nil {
		return "", err
	}
	return a.executor.ProjectLogs(ctx, project.Name, tail)
}

func (a *App) projectToolData(ctx context.Context, project model.Project) (services.ProjectToolData, error) {
	runtimeInfo, err := a.executor.InspectProject(ctx, project.Name)
	if err != nil {
		runtimeInfo = services.ProjectRuntime{Status: "docker_unavailable", Containers: []services.ProjectContainer{}}
	}

	status := project.Status
	if runtimeInfo.Status != "" && runtimeInfo.Status != "docker_unavailable" {
		status = runtimeInfo.Status
	}

	return services.ProjectToolData{
		ID:              project.ID,
		Name:            project.Name,
		TemplateID:      project.TemplateID,
		TemplateVersion: project.TemplateVersion,
		Status:          status,
		LastError:       project.LastError,
		Containers:      runtimeInfo.Containers,
	}, nil
}

func (a *App) findProject(projectID string) (model.Project, error) {
	var project model.Project
	if err := a.db.First(&project, "id = ?", projectID).Error; err != nil {
		return model.Project{}, fmt.Errorf("项目不存在")
	}
	return project, nil
}
