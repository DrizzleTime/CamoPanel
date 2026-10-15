export interface Container {
  Id: string; Names: string[]; Image: string; State: string; Status: string; Created: number;
  Ports: { IP?: string; PrivatePort: number; PublicPort?: number; Type: string }[];
}
export interface Snapshot {
  mode: 'demo' | 'docker'; generated_at: string;
  engine: { Name: string; ServerVersion: string; OperatingSystem: string; NCPU: number; MemTotal: number };
  containers: Container[];
  images: { Id: string; RepoTags: string[] | null; Size: number; Created: number }[];
  networks: { Id: string; Name: string; Driver: string; Scope: string }[];
  volumes: { Name: string; Driver: string; Mountpoint: string }[];
}

export interface PortSpec {
  key: string;
  label: string;
  container_port: number;
  protocol?: string;
  default: number;
  description?: string;
  required?: boolean;
  proxyable?: boolean;
}

export interface ShowIfCondition {
  field: string;
  operator: 'equals' | 'not_equals';
  value: any;
}

export interface ParamOption {
  label: string;
  value: string;
}

export interface ParamSpec {
  key: string;
  label: string;
  type: 'string' | 'password' | 'number' | 'boolean' | 'select';
  default?: any;
  description?: string;
  required?: boolean;
  is_secret?: boolean;
  auto_generate?: boolean;
  options?: ParamOption[];
  show_if?: ShowIfCondition;
}

export interface GatewaySpec {
  is_web_app: boolean;
  primary_port_key?: string;
  default_subpath?: string;
}

export interface EnvParam {
  key: string;
  label: string;
  default: string;
  description: string;
  required: boolean;
  is_secret?: boolean;
}

export interface StoreApp {
  id: string;
  name: string;
  category: string;
  description: string;
  version: string;
  icon: string;
  website: string;
  default_ports: Record<string, number>;
  env_params: EnvParam[];
  ports?: PortSpec[];
  params?: ParamSpec[];
  gateway?: GatewaySpec;
  compose_template: string;
  data_dirs: string[];
  installed: boolean;
}

export interface InstalledApp {
  id: string;
  app_id: string;
  name: string;
  category: string;
  status: 'running' | 'stopped' | 'error';
  app_dir: string;
  compose_path: string;
  ports: Record<string, number>;
  primary_port?: number;
  gateway?: GatewaySpec;
  env: Record<string, string>;
  container_names: string[];
  installed_at: string;
  updated_at: string;
}

export type SiteType = 'static' | 'proxy' | 'runtime';

export interface SiteRedirect {
  from: string;
  to: string;
  code: number;
}

export interface SiteLocation {
  path: string;
  proxy: string;
}

export interface Site {
  id: string;
  name: string;
  primary_domain: string;
  domains: string[];
  type: SiteType;
  runtime?: string;
  root_path: string;
  proxy_target: string;
  ssl_enabled: boolean;
  ssl_provider: string;
  cert_id?: string;
  cert_expiry: string;
  days_remaining?: number;
  force_https: boolean;
  status: 'running' | 'stopped';
  custom_config: string;
  redirects?: SiteRedirect[];
  locations?: SiteLocation[];
  basic_auth_user?: string;
  hotlink?: boolean;
  rate_limit?: number;
  created_at: string;
  updated_at: string;
}

export interface SiteWrite {
  name: string;
  primary_domain: string;
  domains?: string[];
  type: SiteType;
  runtime?: string;
  proxy_target?: string;
  force_https?: boolean;
  custom_config?: string;
  redirects?: SiteRedirect[];
  locations?: SiteLocation[];
  basic_auth_user?: string;
  basic_auth_password?: string;
  hotlink?: boolean;
  rate_limit?: number;
}

export interface SSLInfo {
  enabled: boolean;
  provider: string;
  domains: string[];
  expiry: string;
  days_remaining: number;
  issuer: string;
}

export interface HostMemory {
  total: number;
  used: number;
  available: number;
  percent: number;
}

export interface HostOverview {
  mode: 'demo' | 'host';
  sampled_at: string;
  uptime_seconds: number;
  cpu: { percent: number; cores: number };
  memory: HostMemory;
  swap: HostMemory;
  load: { load1: number; load5: number; load15: number };
  disks: { mount: string; device: string; total: number; used: number; percent: number }[];
  networks: { name: string; rx_bytes: number; tx_bytes: number; rx_per_sec: number; tx_per_sec: number }[];
  history: { at: string; cpu_percent: number; mem_percent: number }[];
}

export interface HostProcess {
  pid: number;
  name: string;
  cpu_percent: number;
  mem_percent: number;
  rss: number;
}

export interface NginxStatus {
  installed: boolean;
  running: boolean;
  container_id: string;
  http_port: number;
  https_port: number;
  config_path: string;
  last_reload_at: string;
  active_site_count: number;
}

export interface AuthSession { actor: string; role: string }
export interface AuthStatus { configured: boolean; source: 'environment' | 'legacy' | 'token' | 'unconfigured'; totp_enabled: boolean }

export class ApiError extends Error { constructor(message: string, public status: number) { super(message); } }
export async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = sessionStorage.getItem('camo-token');
  const response = await fetch(`/api/v1${path}`, {
    ...options,
    credentials: options.credentials || 'include',
    signal: options.signal || AbortSignal.timeout(55000),
    headers: { ...(options.body instanceof FormData ? {} : { 'Content-Type': 'application/json' }), ...(token ? { Authorization: `Bearer ${token}` } : {}), ...options.headers },
  });
  const data = await response.json().catch(() => null);
  if (!response.ok) {
    if (response.status === 401 && !path.startsWith('/auth/')) window.dispatchEvent(new Event('camo:unauthorized'));
    throw new ApiError(data?.error?.message || `请求失败 (${response.status})`, response.status);
  }
  return data as T;
}

export const api = {
  overview: async (signal?: AbortSignal) => {
    const data = await request<Snapshot>('/docker/overview', { signal });
    return { ...data, containers: data.containers || [], images: data.images || [], networks: data.networks || [], volumes: data.volumes || [] };
  },
  host: {
    overview: async () => {
      const data = await request<HostOverview>('/host/overview');
      return { ...data, disks: data.disks || [], networks: data.networks || [], history: data.history || [] };
    },
    processes: async (limit = 10, sort: 'cpu' | 'mem' = 'cpu') => {
      const data = await request<{ processes: HostProcess[] }>(`/host/processes?limit=${limit}&sort=${sort}`);
      return data.processes || [];
    },
  },
  action: (id: string, action: string) => request(`/docker/containers/${encodeURIComponent(id)}/${action}`, { method: 'POST' }),
  logs: (id: string) => request<{ logs: string }>(`/docker/containers/${encodeURIComponent(id)}/logs`),
  create: (body: {
    name: string;
    image: string;
    host_port?: number;
    container_port?: number;
    ports?: { host_port: number; container_port: number; protocol?: string }[];
    env?: string[];
    command?: string[];
    restart_policy?: string;
    binds?: string[];
    confirm_host_mount?: boolean;
    pull?: boolean;
  }) => request<{ id: string; message?: string }>('/docker/containers', { method: 'POST', body: JSON.stringify(body) }),
  forceRemove: (id: string) => request<{ message: string; volumes_deleted: boolean }>(`/docker/containers/${encodeURIComponent(id)}/force-remove`, { method: 'POST', body: JSON.stringify({ confirm: true }) }),
  deleteImage: (id: string) => request<{ message: string }>(`/docker/images/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  pruneImages: () => request<{ removed: number }>('/docker/images/prune', { method: 'POST' }),

  // App Store
  store: {
    listCatalog: () => request<StoreApp[]>('/store/apps'),
    getCatalogApp: (id: string) => request<StoreApp>(`/store/apps/${encodeURIComponent(id)}`),
    listInstalled: () => request<InstalledApp[]>('/store/installed'),
    getInstalled: (id: string) => request<InstalledApp>(`/store/installed/${encodeURIComponent(id)}`),
    install: (body: { app_id: string; name: string; ports?: Record<string, number>; env?: Record<string, string>; custom_compose?: string }) =>
      request<InstalledApp>('/store/installed', { method: 'POST', body: JSON.stringify(body) }),
    action: (id: string, action: 'start' | 'stop' | 'restart' | 'uninstall', deleteData = false) =>
      request<{ message: string }>(`/store/installed/${encodeURIComponent(id)}/action`, {
        method: 'POST',
        body: JSON.stringify({ action, delete_data: deleteData }),
      }),
    getCompose: (id: string) => request<{ compose: string }>(`/store/installed/${encodeURIComponent(id)}/compose`),
    updateCompose: (id: string, compose: string) =>
      request<{ message: string }>(`/store/installed/${encodeURIComponent(id)}/compose`, {
        method: 'PUT',
        body: JSON.stringify({ compose }),
      }),
    getLogs: (id: string) => request<{ logs: string }>(`/store/installed/${encodeURIComponent(id)}/logs`),
  },

  // Websites
  sites: {
    list: () => request<Site[]>('/sites'),
    get: (id: string) => request<Site>(`/sites/${encodeURIComponent(id)}`),
    create: (body: SiteWrite) => request<Site>('/sites', { method: 'POST', body: JSON.stringify(body) }),
    update: (id: string, body: SiteWrite) => request<Site>(`/sites/${encodeURIComponent(id)}`, { method: 'PUT', body: JSON.stringify(body) }),
    delete: (id: string) => request<{ message: string }>(`/sites/${encodeURIComponent(id)}`, { method: 'DELETE' }),
    toggle: (id: string) => request<Site>(`/sites/${encodeURIComponent(id)}/toggle`, { method: 'POST' }),
    applySSL: (id: string, body: { provider: 'letsencrypt' | 'self-signed' | 'custom'; email?: string; cert_pem?: string; key_pem?: string; force_https?: boolean }) =>
      request<Site>(`/sites/${encodeURIComponent(id)}/ssl/apply`, { method: 'POST', body: JSON.stringify(body) }),
    getSSL: (id: string) => request<SSLInfo>(`/sites/${encodeURIComponent(id)}/ssl`),
    getConfig: (id: string) => request<{ config: string }>(`/sites/${encodeURIComponent(id)}/config`),
    getFile: (id: string, filename = 'index.html') => request<{ filename: string; content: string }>(`/sites/${encodeURIComponent(id)}/files/${encodeURIComponent(filename)}`),
    saveFile: (id: string, filename: string, content: string) =>
      request<{ message: string }>(`/sites/${encodeURIComponent(id)}/files/${encodeURIComponent(filename)}`, { method: 'PUT', body: JSON.stringify({ content }) }),
  },

  // Nginx
  nginx: {
    status: () => request<NginxStatus>('/nginx/status'),
    action: (action: 'start' | 'stop' | 'restart' | 'reload') =>
      request<{ message: string }>('/nginx/action', { method: 'POST', body: JSON.stringify({ action }) }),
  },

  auth: {
    status: () => request<AuthStatus>('/auth/status', { headers: { Authorization: '' } }),
    session: () => request<AuthSession>('/auth/session', { headers: { Authorization: '' } }),
    bootstrap: (password: string) => request<{ username: string }>('/auth/bootstrap', { method: 'POST', body: JSON.stringify({ password }) }),
    login: (username: string, password: string, code?: string) =>
      request<{ username: string; role: string }>('/auth/login', { method: 'POST', body: JSON.stringify({ username, password, code }), headers: { Authorization: '' } }),
    logout: () => request<{ message: string }>('/auth/logout', { method: 'POST', headers: { Authorization: '' } }),
    password: (current: string, next: string) =>
      request<{ message: string }>('/auth/password', { method: 'POST', body: JSON.stringify({ current, next }) }),
    viewer: (password: string) => request<{ username: string; role: string }>('/auth/viewer', { method: 'POST', body: JSON.stringify({ password }) }),
    totpBegin: () => request<{ secret: string }>('/auth/totp/begin', { method: 'POST' }),
    totpEnable: (code: string) => request<{ enabled: boolean }>('/auth/totp/enable', { method: 'POST', body: JSON.stringify({ code }) }),
  },
  exposure: () => request<{ https: boolean; message: string }>('/settings/exposure'),
  certificates: {
    list: () => request<Certificate[]>('/certificates'),
    selfSigned: (domains: string[]) => request<Certificate>('/certificates/self-signed', { method: 'POST', body: JSON.stringify({ domains }) }),
    upload: (cert_pem: string, key_pem: string, domains: string[]) =>
      request<Certificate>('/certificates/upload', { method: 'POST', body: JSON.stringify({ cert_pem, key_pem, domains }) }),
    bind: (id: string, sites: string[]) => request<Certificate>(`/certificates/${encodeURIComponent(id)}/bind`, { method: 'POST', body: JSON.stringify({ sites }) }),
    dns: () => request<DnsPublic>('/certificates/dns'),
    saveDns: (body: { provider: string; api_base?: string; token?: string; zone?: string }) =>
      request<DnsPublic>('/certificates/dns', { method: 'PUT', body: JSON.stringify(body) }),
    manualTxt: (name: string, value: string) =>
      request<{ name: string; type: string; value: string }>('/certificates/dns/manual', { method: 'POST', body: JSON.stringify({ name, value }) }),
    dnsIssue: (domains: string[]) => request<{ status: string; message: string; certificate: Certificate }>('/certificates/dns/issue', { method: 'POST', body: JSON.stringify({ domains }) }),
  },
  files: {
    roots: () => request<FileRoots>('/files/roots'),
    create: (root: string, path: string, directory: boolean) => request<{ message: string }>('/files/create', { method: 'POST', body: JSON.stringify({ root, path, directory }) }),
    uploadFile: (root: string, path: string, file: File, overwrite: boolean, onProgress: (percent: number) => void) => uploadServerFile(root, path, file, overwrite, onProgress),
    download: (root: string, path: string) => downloadServerFile(root, path),
    transfer: (root: string, paths: string[], dest_root: string, dest: string, move: boolean) => request<FileBatch>('/files/transfer', { method: 'POST', body: JSON.stringify({ root, paths, dest_root, dest, move }) }),
    removeMany: (root: string, paths: string[]) => request<FileBatch>('/files/delete', { method: 'POST', body: JSON.stringify({ root, paths }) }),
    chmod: (root: string, path: string, mode: string) => request<{ message: string }>('/files/permissions', { method: 'POST', body: JSON.stringify({ root, path, mode }) }),
    archive: (root: string, paths: string[], dest: string) => request<{ message: string }>('/files/zip', { method: 'POST', body: JSON.stringify({ root, paths, dest }) }),
    list: (root: string, path = '', signal?: AbortSignal) => request<FileListing>(`/files?root=${encodeURIComponent(root)}&path=${encodeURIComponent(path)}`, { signal }),
    read: (root: string, path: string) => request<{ content: string }>(`/files/content?root=${encodeURIComponent(root)}&path=${encodeURIComponent(path)}`),
    write: (root: string, path: string, content: string) => request<{ message: string }>('/files', { method: 'PUT', body: JSON.stringify({ root, path, content }) }),
    upload: (root: string, path: string, content: string) => request<{ message: string }>('/files/upload', { method: 'POST', body: JSON.stringify({ root, path, content }) }),
    mkdir: (root: string, path: string) => request<{ message: string }>('/files/mkdir', { method: 'POST', body: JSON.stringify({ root, path }) }),
    rename: (root: string, from: string, to: string) => request<{ message: string }>('/files/rename', { method: 'POST', body: JSON.stringify({ root, from, to }) }),
    remove: (root: string, path: string) => request<{ message: string }>(`/files?root=${encodeURIComponent(root)}&path=${encodeURIComponent(path)}`, { method: 'DELETE' }),
    zip: (root: string, path: string, dest: string) => request<{ message: string }>('/files/zip', { method: 'POST', body: JSON.stringify({ root, path, dest }) }),
    unzip: (root: string, zip: string, dest: string) => request<{ message: string }>('/files/unzip', { method: 'POST', body: JSON.stringify({ root, zip, dest }) }),
  },
  databases: {
    list: () => request<DatabaseInfo[]>('/databases'),
    get: (kind: string) => request<DatabaseInfo>(`/databases/${encodeURIComponent(kind)}`),
    createDatabase: (kind: string, name: string) => request<{ name: string }>(`/databases/${encodeURIComponent(kind)}/databases`, { method: 'POST', body: JSON.stringify({ name }) }),
    createUser: (kind: string, name: string, password: string) =>
      request<{ name: string }>(`/databases/${encodeURIComponent(kind)}/users`, { method: 'POST', body: JSON.stringify({ name, password }) }),
    exportSql: (kind: string) => request<{ sql: string }>(`/databases/${encodeURIComponent(kind)}/export`, { method: 'POST' }),
    importSql: (kind: string, sql: string) => request<{ message: string }>(`/databases/${encodeURIComponent(kind)}/import`, { method: 'POST', body: JSON.stringify({ sql }) }),
    exec: (kind: string, sql: string, password: string) =>
      request<{ result: string }>(`/databases/${encodeURIComponent(kind)}/exec`, { method: 'POST', body: JSON.stringify({ sql, password }) }),
    redisKeys: (prefix: string, limit = 50) => request<{ keys: string[] }>(`/databases/redis/keys?prefix=${encodeURIComponent(prefix)}&limit=${limit}`),
    redisAdd: (key: string) => request<{ key: string }>('/databases/redis/keys', { method: 'POST', body: JSON.stringify({ key }) }),
    redisDel: (key: string) => request<{ message: string }>(`/databases/redis/keys?key=${encodeURIComponent(key)}`, { method: 'DELETE' }),
  },
  backups: {
    list: () => request<BackupItem[]>('/backups'),
    site: (id: string) => request<BackupItem>(`/backups/sites/${encodeURIComponent(id)}`, { method: 'POST' }),
    directory: (path: string) => request<BackupItem>('/backups/directories', { method: 'POST', body: JSON.stringify({ path }) }),
    restore: (id: string, confirm: boolean) => request<{ message: string }>(`/backups/${encodeURIComponent(id)}/restore`, { method: 'POST', body: JSON.stringify({ confirm }) }),
    publish: (id: string, body: { endpoint: string; bucket: string; access_key: string; secret: string }) =>
      request<{ remote: string }>(`/backups/${encodeURIComponent(id)}/publish`, { method: 'POST', body: JSON.stringify(body) }),
  },
  schedules: {
    list: () => request<ScheduleJob[]>('/schedules'),
    create: (body: { name: string; type: string; cron: string; timeout_sec?: number; payload?: Record<string, string>; enabled?: boolean }) =>
      request<ScheduleJob>('/schedules', { method: 'POST', body: JSON.stringify(body) }),
    run: (id: string) => request<{ at: string; status: string; output?: string }>(`/schedules/${encodeURIComponent(id)}/run`, { method: 'POST' }),
    enabled: (id: string, enabled: boolean) => request<{ enabled: boolean }>(`/schedules/${encodeURIComponent(id)}/enabled`, { method: 'POST', body: JSON.stringify({ enabled }) }),
    script: (name: string, content: string) => request<{ name: string }>('/schedules/scripts', { method: 'POST', body: JSON.stringify({ name, content }) }),
    reload: () => request<ScheduleJob[]>('/schedules/reload', { method: 'POST' }),
  },
  firewall: {
    status: () => request<FirewallStatus>('/firewall'),
    add: (port: number, policy: string, confirm: boolean) =>
      request<FirewallStatus>('/firewall/rules', { method: 'POST', body: JSON.stringify({ port, policy, confirm }) }),
  },
  alerts: {
    list: () => request<AlertRule[]>('/alerts'),
    add: (body: { kind: string; threshold?: number; webhook?: string; smtp_host?: string }) =>
      request<AlertRule>('/alerts', { method: 'POST', body: JSON.stringify(body) }),
    evaluate: (sample: { disk_percent?: number; cert_days?: number; container_exited?: boolean; site_down?: boolean }) =>
      request<{ sent: number }>('/alerts/evaluate', { method: 'POST', body: JSON.stringify(sample) }),
  },
  terminals: {
    host: () => request<{ id: string; kind: string; account: string }>('/terminals/host', { method: 'POST' }),
    container: (id: string) => request<{ id: string; kind: string; account: string; container: string }>(`/terminals/containers/${encodeURIComponent(id)}`, { method: 'POST' }),
  },
  ai: {
    settings: () => request<AiPublic>('/ai/settings'),
    save: (body: { base_url: string; model: string; api_key?: string; headers?: Record<string, string> }) =>
      request<AiPublic>('/ai/settings', { method: 'PUT', body: JSON.stringify(body) }),
    chat: (message: string, confirm = false) =>
      request<AiReply>('/ai/chat', { method: 'POST', body: JSON.stringify({ message, confirm, stream: true }) }),
  },
};

export interface Certificate {
  id: string;
  domains: string[];
  provider: string;
  expiry: string;
  issuer: string;
  sites: string[];
  attempts?: { at: string; method: string; status: string; error?: string }[];
  created_at: string;
}

export interface DnsPublic {
  provider: string;
  zone?: string;
  api_base?: string;
  token_set: boolean;
}

export interface FileEntry {
  name: string;
  path: string;
  dir: boolean;
  size: number;
  modified: string;
  mode: string;
  permissions: string;
  owner: string;
  group: string;
  type: 'file' | 'directory' | 'symlink' | 'special';
  symlink: boolean;
  link_target?: string;
}

export interface FileListing { entries: FileEntry[]; path: string; parent: string; root: string }
export interface FileRoot { name: string; label: string; path: string }
export interface FileRoots { roots: FileRoot[]; server_root: string; home: string; hostname: string; read_only: boolean; upload_limit: number; text_limit: number }
export interface FileBatch { results: { path: string; dest?: string; error?: string }[]; succeeded: number }

export interface DatabaseInfo {
  kind: string;
  installed: boolean;
  store_path: string;
}

export interface BackupItem {
  id: string;
  kind: string;
  source: string;
  path: string;
  created_at: string;
  checksum: string;
  remote?: string;
}

export interface ScheduleJob {
  id: string;
  name: string;
  type: string;
  cron: string;
  enabled: boolean;
  timeout_sec: number;
  payload?: Record<string, string>;
  missed: boolean;
  runs: { at: string; status: string; output?: string }[] | null;
}

export interface FirewallStatus {
  available: boolean;
  backend?: string;
  message?: string;
  rules: { port: number; policy: string }[] | null;
  panel_port: number;
}

export interface AlertRule {
  id: string;
  kind: string;
  threshold: number;
  webhook?: string;
  smtp_host?: string;
}

export interface AiPublic {
  base_url: string;
  model: string;
  key_set: boolean;
  configured: boolean;
  headers: { name: string; secret_set: boolean }[];
}

export interface AiReply {
  reply: string;
  pending_write?: { name: string; arguments: string };
}

export const containerName = (c: Container) => c.Names?.[0]?.replace(/^\//, '') || c.Id.slice(0, 12);
export const formatBytes = (n: number) => n >= 1024 ** 3 ? `${(n / 1024 ** 3).toFixed(1)} GB` : `${(n / 1024 ** 2).toFixed(1)} MB`;


function uploadServerFile(root: string, path: string, file: File, overwrite: boolean, onProgress: (percent: number) => void): Promise<void> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open('POST', '/api/v1/files/upload');
    xhr.withCredentials = true;
    xhr.timeout = 10 * 60 * 1000;
    const token = sessionStorage.getItem('camo-token');
    if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`);
    const body = new FormData();
    body.append('root', root); body.append('path', path); body.append('overwrite', String(overwrite)); body.append('file', file);
    xhr.upload.onprogress = event => { if (event.lengthComputable) onProgress(Math.round(event.loaded / event.total * 100)); };
    xhr.onload = () => {
      if (xhr.status === 401) window.dispatchEvent(new Event('camo:unauthorized'));
      let response: { error?: { message?: string } } = {};
      try { response = JSON.parse(xhr.responseText); } catch { /* Non-JSON upstream error. */ }
      if (xhr.status >= 200 && xhr.status < 300) resolve();
      else reject(new ApiError(response.error?.message || `上传失败 (${xhr.status})`, xhr.status));
    };
    xhr.onerror = () => reject(new Error('上传失败，请检查网络连接'));
    xhr.ontimeout = () => reject(new Error('上传超时，请重试'));
    xhr.send(body);
  });
}

async function downloadServerFile(root: string, path: string) {
  const { url } = await request<{ url: string }>(`/files/download-link?root=${encodeURIComponent(root)}&path=${encodeURIComponent(path)}`);
  const anchor = document.createElement('a');
  anchor.href = url; anchor.download = path.split('/').pop() || 'download';
  document.body.appendChild(anchor); anchor.click(); anchor.remove();
}
