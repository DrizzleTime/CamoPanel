export type User = {
  id: string;
  username: string;
  role: string;
};

export type TemplateParam = {
  name: string;
  label: string;
  description: string;
  type: "string" | "number" | "boolean" | "secret";
  required: boolean;
  default?: string | number | boolean;
  placeholder?: string;
};

export type TemplateSpec = {
  id: string;
  name: string;
  version: string;
  description: string;
  params: TemplateParam[];
  health_hints: string[];
};

export type ProjectContainer = {
  id: string;
  name: string;
  image: string;
  state: string;
  status: string;
  ports: string[];
};

export type Project = {
  id: string;
  name: string;
  template_id: string;
  template_version: string;
  config: Record<string, unknown>;
  compose_path: string;
  status: string;
  last_error: string;
  runtime: {
    status: string;
    containers: ProjectContainer[];
  };
  created_at: string;
  updated_at: string;
};

export type Approval = {
  id: string;
  source: string;
  action: string;
  target_type: string;
  target_id: string;
  payload_json: string;
  summary: string;
  status: string;
  created_by: string;
  approved_by: string;
  error_message: string;
  executed_at?: string;
  created_at: string;
  updated_at: string;
};

export type HostSummary = {
  hostname: string;
  os: string;
  platform: string;
  kernel: string;
  architecture: string;
  cpu_cores: number;
  cpu_percent: number;
  load_1: number;
  load_5: number;
  memory_used: number;
  memory_total: number;
  disk_used: number;
  disk_total: number;
  sampled_at: string;
};

export type HostMetricsPoint = {
  timestamp: string;
  cpu_percent: number;
  load_1: number;
  load_5: number;
  memory_used: number;
  memory_total: number;
  disk_used: number;
  disk_total: number;
  network_rx_rate: number;
  network_tx_rate: number;
  disk_read_rate: number;
  disk_write_rate: number;
};

export type HostMetrics = {
  summary: HostSummary;
  history: HostMetricsPoint[];
  sample_interval_seconds: number;
};

export type DashboardSnapshot = {
  metrics: HostMetrics;
  projects: Project[];
  approvals: Approval[];
  websites: Website[];
  generated_at: string;
};

export type DockerContainer = {
  id: string;
  name: string;
  image: string;
  state: string;
  status: string;
  project: string;
  ports: string[];
  networks: string[];
  created_at: string;
};

export type DockerImage = {
  id: string;
  repo_tags: string[];
  containers: number;
  size: number;
  created_at: string;
};

export type DockerNetwork = {
  id: string;
  name: string;
  driver: string;
  scope: string;
  internal: boolean;
  attachable: boolean;
  ingress: boolean;
  container_count: number;
  created_at: string;
};

export type DockerSystemInfo = {
  id: string;
  name: string;
  server_version: string;
  operating_system: string;
  kernel_version: string;
  architecture: string;
  ncpu: number;
  mem_total: number;
  docker_root_dir: string;
  driver: string;
  logging_driver: string;
  cgroup_driver: string;
  cgroup_version: string;
  default_runtime: string;
  runtimes: string[];
  network_plugins: string[];
  volume_plugins: string[];
  containers: number;
  containers_running: number;
  containers_paused: number;
  containers_stopped: number;
  images: number;
  warnings: string[];
};

export type CopilotSession = {
  id: string;
};

export type OpenRestyStatus = {
  exists: boolean;
  ready: boolean;
  container_name: string;
  container_status: string;
  host_config_dir: string;
  host_site_dir: string;
  message: string;
};

export type Website = {
  id: string;
  name: string;
  type: "static" | "proxy";
  domain: string;
  root_path: string;
  proxy_pass: string;
  config_path: string;
  status: string;
  created_at: string;
  updated_at: string;
};

export type FileEntry = {
  name: string;
  path: string;
  type: "file" | "directory" | "symlink";
  size: number;
  mode: string;
  modified_at: string;
};

export type FileListResponse = {
  current_path: string;
  parent_path: string;
  items: FileEntry[];
};

export type FileReadResponse = {
  path: string;
  name: string;
  size: number;
  mode: string;
  modified_at: string;
  content: string;
  is_binary: boolean;
};
