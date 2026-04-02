import { apiRequest } from "../../shared/api/client";
import type { Project, TemplateSpec } from "../../shared/types";

export async function loadStoreBundle() {
  const [templateResponse, projectResponse] = await Promise.all([
    apiRequest<{ items: TemplateSpec[] }>("/api/templates"),
    apiRequest<{ items: Project[] }>("/api/projects"),
  ]);

  return {
    templates: templateResponse.items,
    projects: projectResponse.items,
  };
}

export async function deployTemplateProject(payload: {
  name: unknown;
  template_id: string;
  parameters: Record<string, unknown>;
}) {
  return apiRequest<{ project: Project }>("/api/projects", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function runStoreProjectAction(projectId: string, action: "delete" | "redeploy") {
  return apiRequest(`/api/projects/${projectId}/actions`, {
    method: "POST",
    body: JSON.stringify({ action }),
  });
}
