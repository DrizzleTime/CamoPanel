import type { Project } from "../../shared/types";
import type { Website } from "../websites/types";

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
  websites: Website[];
  generated_at: string;
};
