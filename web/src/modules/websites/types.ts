export type OpenRestyStatus = {
  certificate_ready: boolean;
  exists: boolean;
  ready: boolean;
  container_name: string;
  container_status: string;
  host_config_dir: string;
  host_site_dir: string;
  message: string;
};

export type Certificate = {
  id: string;
  domain: string;
  email: string;
  provider: string;
  status: string;
  fullchain_path: string;
  private_key_path: string;
  last_error: string;
  expires_at: string;
  website_id?: string;
  website_name?: string;
  created_at: string;
  updated_at: string;
};

export type Website = {
  id: string;
  name: string;
  type: "static" | "php" | "proxy";
  domain: string;
  domains_json?: string;
  site_mode?: "static" | "php" | "proxy";
  root_path: string;
  index_files?: string;
  proxy_pass: string;
  php_project_id?: string;
  php_port?: number;
  rewrite_mode?: "off" | "preset" | "custom";
  rewrite_preset?: string;
  rewrite_rules?: string;
  config_path: string;
  status: string;
  created_at: string;
  updated_at: string;
};

export type WebsiteFormValues = {
  name: string;
  type: "static" | "php" | "proxy";
  domain: string;
  domains: string;
  root_path?: string;
  index_files?: string;
  proxy_pass?: string;
  php_project_id?: string;
  rewrite_mode: "off" | "preset" | "custom";
  rewrite_preset?: string;
  rewrite_rules?: string;
};

export type CertificateFormValues = {
  domain: string;
  email: string;
};

export type EnvironmentFormValues = {
  name: string;
  php_version: "8.1" | "8.2" | "8.3";
  port: number;
};
