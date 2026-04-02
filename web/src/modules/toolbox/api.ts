import { apiRequest } from "../../shared/api/client";
import type { CleanupResult, CleanupScanResult, SystemConfig, SwapInfo } from "./types";

export async function getSystemConfig() {
  return apiRequest<SystemConfig>("/api/system/config");
}

export async function updateHostname(hostname: string) {
  return apiRequest<{ ok: boolean }>("/api/system/hostname", {
    method: "PUT",
    body: JSON.stringify({ hostname }),
  });
}

export async function updateDNS(servers: string[]) {
  return apiRequest<{ ok: boolean }>("/api/system/dns", {
    method: "PUT",
    body: JSON.stringify({ servers }),
  });
}

export async function updateTimezone(timezone: string) {
  return apiRequest<{ ok: boolean }>("/api/system/timezone", {
    method: "PUT",
    body: JSON.stringify({ timezone }),
  });
}

export async function createSwap(sizeMB: number) {
  return apiRequest<SwapInfo>("/api/system/swap/create", {
    method: "POST",
    body: JSON.stringify({ size_mb: sizeMB }),
  });
}

export async function removeSwap() {
  return apiRequest<SwapInfo>("/api/system/swap/remove", {
    method: "POST",
  });
}

export async function scanCleanup() {
  return apiRequest<CleanupScanResult>("/api/system/cleanup/scan");
}

export async function executeCleanup(categories: string[]) {
  return apiRequest<CleanupResult>("/api/system/cleanup/clean", {
    method: "POST",
    body: JSON.stringify({ categories }),
  });
}
