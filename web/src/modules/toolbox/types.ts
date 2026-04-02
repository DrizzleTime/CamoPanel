export interface SwapInfo {
  total: number;
  used: number;
  file: string;
}

export interface SystemConfig {
  hostname: string;
  dns: string[];
  timezone: string;
  swap: SwapInfo;
}

export interface CleanupItem {
  size: number;
  description: string;
}

export interface CleanupScanResult {
  items: Record<string, CleanupItem>;
}

export interface CleanupResult {
  cleaned: number;
}
