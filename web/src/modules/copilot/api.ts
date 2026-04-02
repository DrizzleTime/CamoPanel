import { apiRequest } from "../../shared/api/client";
import type { CopilotConfigStatus, CopilotModel, CopilotProvider, CopilotSession } from "./types";

export async function createCopilotSession() {
  return apiRequest<CopilotSession>("/api/copilot/sessions", {
    method: "POST",
  });
}

export async function getCopilotConfigStatus() {
  return apiRequest<CopilotConfigStatus>("/api/copilot/config");
}

export async function listCopilotProviders() {
  return apiRequest<{ items: CopilotProvider[] }>("/api/copilot/providers");
}

export async function sendCopilotMessage(sessionId: string, message: string) {
  return fetch(`/api/copilot/sessions/${sessionId}/messages`, {
    method: "POST",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ message }),
  });
}

export async function saveCopilotProvider(
  values: { name: string; base_url: string; api_key: string; enabled: boolean },
  providerId?: string,
) {
  if (providerId) {
    return apiRequest(`/api/copilot/providers/${providerId}`, {
      method: "PUT",
      body: JSON.stringify({
        ...values,
        type: "openai",
      }),
    });
  }

  return apiRequest("/api/copilot/providers", {
    method: "POST",
    body: JSON.stringify({
      ...values,
      type: "openai",
    }),
  });
}

export async function deleteCopilotProvider(providerId: string) {
  return apiRequest(`/api/copilot/providers/${providerId}`, { method: "DELETE" });
}

export async function saveCopilotModel(
  values: { name: string; enabled: boolean; is_default: boolean },
  providerId: string,
  modelId?: string,
) {
  if (modelId) {
    return apiRequest(`/api/copilot/models/${modelId}`, {
      method: "PUT",
      body: JSON.stringify(values),
    });
  }

  return apiRequest(`/api/copilot/providers/${providerId}/models`, {
    method: "POST",
    body: JSON.stringify(values),
  });
}

export async function deleteCopilotModel(modelId: string) {
  return apiRequest(`/api/copilot/models/${modelId}`, { method: "DELETE" });
}

export async function setCopilotDefaultModel(aiModel: CopilotModel) {
  return apiRequest(`/api/copilot/models/${aiModel.id}`, {
    method: "PUT",
    body: JSON.stringify({
      name: aiModel.name,
      enabled: aiModel.enabled,
      is_default: true,
    }),
  });
}
