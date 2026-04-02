import { apiRequest } from "../../shared/api/client";
import type { DatabaseEngine, DatabaseInstance, DatabaseOverview } from "./types";

type DatabaseAccountPayload = {
  name: string;
  password: string;
  database_name?: string;
};

type DatabaseGrantPayload = {
  name: string;
  database_name: string;
};

export async function listDatabaseInstances(engine: DatabaseEngine) {
  return apiRequest<{ items: DatabaseInstance[] }>(`/api/databases?engine=${engine}`);
}

export async function getDatabaseOverview(instanceId: string) {
  return apiRequest<DatabaseOverview>(`/api/databases/${instanceId}/overview`);
}

export async function runDatabaseInstanceAction(projectId: string, action: string) {
  return apiRequest(`/api/projects/${projectId}/actions`, {
    method: "POST",
    body: JSON.stringify({ action }),
  });
}

export async function createDatabaseSchema(instanceId: string, name: string) {
  return apiRequest(`/api/databases/${instanceId}/databases`, {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

export async function createDatabaseAccount(instanceId: string, payload: DatabaseAccountPayload) {
  return apiRequest(`/api/databases/${instanceId}/accounts`, {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function createDatabaseGrant(instanceId: string, payload: DatabaseGrantPayload) {
  return apiRequest(`/api/databases/${instanceId}/grants`, {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function updateDatabasePassword(instanceId: string, accountName: string, password: string) {
  return apiRequest(`/api/databases/${instanceId}/accounts/${encodeURIComponent(accountName)}/password`, {
    method: "PUT",
    body: JSON.stringify({ password }),
  });
}

export async function deleteDatabaseSchema(instanceId: string, databaseName: string) {
  return apiRequest(`/api/databases/${instanceId}/databases/${encodeURIComponent(databaseName)}`, {
    method: "DELETE",
  });
}

export async function deleteDatabaseAccount(instanceId: string, accountName: string) {
  return apiRequest(`/api/databases/${instanceId}/accounts/${encodeURIComponent(accountName)}`, {
    method: "DELETE",
  });
}

export async function updateRedisConfig(
  instanceId: string,
  values: Record<string, boolean | number | string>,
) {
  return apiRequest(`/api/databases/${instanceId}/redis/config`, {
    method: "PUT",
    body: JSON.stringify(values),
  });
}
