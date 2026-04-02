import { apiRequest } from "../../shared/api/client";
import type { FileListResponse, FileReadResponse } from "./types";

export async function listFiles(path: string) {
  return apiRequest<FileListResponse>(`/api/files/list?path=${encodeURIComponent(path)}`);
}

export async function readFile(path: string) {
  return apiRequest<FileReadResponse>(`/api/files/read?path=${encodeURIComponent(path)}`);
}

export async function writeFile(path: string, content: string) {
  return apiRequest<{ path: string }>("/api/files/write", {
    method: "POST",
    body: JSON.stringify({ path, content }),
  });
}

export async function createDirectory(path: string) {
  return apiRequest<{ path: string }>("/api/files/mkdir", {
    method: "POST",
    body: JSON.stringify({ path }),
  });
}

export async function createFile(path: string, content: string) {
  return apiRequest<{ path: string }>("/api/files/create", {
    method: "POST",
    body: JSON.stringify({ path, content }),
  });
}

export async function moveFile(fromPath: string, toPath: string) {
  return apiRequest<{ path: string }>("/api/files/move", {
    method: "POST",
    body: JSON.stringify({ from_path: fromPath, to_path: toPath }),
  });
}

export async function deleteFile(path: string) {
  return apiRequest<{ path: string }>("/api/files/delete", {
    method: "POST",
    body: JSON.stringify({ path }),
  });
}

export async function uploadFiles(path: string, files: File[]) {
  const formData = new FormData();
  formData.append("path", path);
  files.forEach((file) => formData.append("files", file));
  return apiRequest<{ items: string[] }>("/api/files/upload", {
    method: "POST",
    body: formData,
  });
}
