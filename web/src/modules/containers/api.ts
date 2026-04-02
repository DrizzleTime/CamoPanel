import { apiRequest } from "../../shared/api/client";
import type { Project } from "../../shared/types";
import type {
  DockerContainer,
  DockerImage,
  DockerImagePruneResult,
  DockerNetwork,
  DockerSettings,
  DockerSystemInfo,
} from "./types";

export async function listDockerContainers() {
  return apiRequest<{ items: DockerContainer[] }>("/api/docker/containers");
}

export async function listDockerImages() {
  return apiRequest<{ items: DockerImage[] }>("/api/docker/images");
}

export async function listContainerProjects() {
  return apiRequest<{ items: Project[] }>("/api/projects");
}

export async function listDockerNetworks() {
  return apiRequest<{ items: DockerNetwork[] }>("/api/docker/networks");
}

export async function getDockerSystemInfo() {
  return apiRequest<DockerSystemInfo>("/api/docker/system");
}

export async function getDockerSettings() {
  return apiRequest<DockerSettings>("/api/docker/settings");
}

export async function runContainerProjectAction(projectId: string, action: string) {
  return apiRequest(`/api/projects/${projectId}/actions`, {
    method: "POST",
    body: JSON.stringify({ action }),
  });
}

export async function deleteDockerImage(imageId: string) {
  return apiRequest(`/api/docker/images/${encodeURIComponent(imageId)}`, {
    method: "DELETE",
  });
}

export async function pruneDockerImages() {
  return apiRequest<DockerImagePruneResult>("/api/docker/images/prune", {
    method: "POST",
  });
}

export async function createCustomComposeProject(payload: { name: string; compose: string }) {
  return apiRequest<{ project: Project }>("/api/projects/custom", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function updateDockerSettings(payload: { registry_mirrors: string[] }) {
  return apiRequest<DockerSettings>("/api/docker/settings", {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export async function restartDockerService() {
  return apiRequest<{ ok: boolean }>("/api/docker/restart", {
    method: "POST",
  });
}

export async function runDockerContainerAction(containerId: string, action: string) {
  return apiRequest(`/api/docker/containers/${containerId}/actions`, {
    method: "POST",
    body: JSON.stringify({ action }),
  });
}

export async function getDockerContainerLogs(containerId: string) {
  return apiRequest<{ logs: string }>(`/api/docker/containers/${containerId}/logs?tail=200`);
}

export async function getContainerProjectLogs(projectId: string) {
  return apiRequest<{ logs: string }>(`/api/projects/${projectId}/logs`);
}
