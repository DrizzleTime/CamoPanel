import { apiRequest } from "../../shared/api/client";
import type { Project, TemplateSpec } from "../../shared/types";
import type { Certificate, OpenRestyStatus, Website } from "./types";

export async function loadWebsitesDataBundle() {
  const [status, websiteResponse, certificateResponse, projectResponse, templateResponse] = await Promise.all([
    apiRequest<OpenRestyStatus>("/api/openresty/status"),
    apiRequest<{ items: Website[] }>("/api/websites"),
    apiRequest<{ items: Certificate[] }>("/api/certificates"),
    apiRequest<{ items: Project[] }>("/api/projects"),
    apiRequest<{ items: TemplateSpec[] }>("/api/templates"),
  ]);

  return {
    status,
    websites: websiteResponse.items,
    certificates: certificateResponse.items,
    projects: projectResponse.items,
    templates: templateResponse.items,
  };
}

export async function saveWebsite(payload: Record<string, unknown>, websiteId?: string) {
  if (websiteId) {
    return apiRequest<{ website: Website }>(`/api/websites/${websiteId}`, {
      method: "PUT",
      body: JSON.stringify(payload),
    });
  }

  return apiRequest<{ website: Website }>("/api/websites", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function previewWebsiteConfig(websiteId: string) {
  return apiRequest<{ config: string }>(`/api/websites/${websiteId}/config-preview`);
}

export async function deleteWebsite(websiteId: string) {
  return apiRequest(`/api/websites/${websiteId}`, { method: "DELETE" });
}

export async function createCertificate(payload: { domain: string; email: string }) {
  return apiRequest<{ certificate: Certificate }>("/api/certificates", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function deleteCertificate(certificateId: string) {
  return apiRequest(`/api/certificates/${certificateId}`, { method: "DELETE" });
}

export async function createPhpEnvironment(payload: {
  name: string;
  template_id: string;
  parameters: Record<string, unknown>;
}) {
  return apiRequest<{ project: Project }>("/api/projects", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function runWebsiteEnvironmentAction(projectId: string, action: string) {
  return apiRequest(`/api/projects/${projectId}/actions`, {
    method: "POST",
    body: JSON.stringify({ action }),
  });
}
