import type { Project } from "../../shared/types";

export type DatabaseEngine = "mysql" | "postgres" | "redis";

export type DatabaseConnectionInfo = {
  host: string;
  port: number;
  admin_username?: string;
  app_username?: string;
  default_database?: string;
  password_managed: boolean;
};

export type DatabaseInstance = {
  id: string;
  name: string;
  engine: DatabaseEngine;
  status: string;
  last_error: string;
  runtime: Project["runtime"];
  connection: DatabaseConnectionInfo;
  created_at: string;
  updated_at: string;
};

export type DatabaseNameItem = {
  name: string;
};

export type DatabaseAccountItem = {
  name: string;
  host?: string;
  superuser?: boolean;
};

export type DatabaseRedisKeyspaceItem = {
  name: string;
  keys: number;
};

export type DatabaseRedisConfigItem = {
  key: string;
  value: string;
};

export type DatabaseOverview = {
  instance: DatabaseInstance;
  notice?: string;
  databases?: DatabaseNameItem[];
  accounts?: DatabaseAccountItem[];
  redis_keyspaces?: DatabaseRedisKeyspaceItem[];
  redis_config?: DatabaseRedisConfigItem[];
  summary?: Record<string, string>;
};
