import type { Project } from "../../shared/types";

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

export type DockerImagePruneResult = {
  images_deleted: number;
  space_reclaimed: number;
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

export type DockerSettings = {
  registry_mirrors: string[];
  control_enabled: boolean;
  config_path: string;
  message: string;
};

export type ContainerProject = Project;

export type CustomComposeValues = {
  name: string;
  compose: string;
};
