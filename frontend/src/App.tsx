import { useCallback, useEffect, useRef, useState } from 'react';
import * as stylex from '@stylexjs/stylex';
import { api, containerName, formatBytes, type Container, type HostOverview, type HostProcess, type Snapshot, type AuthSession } from './lib/api';
import { Icon } from './components/Icon';
import { Button } from './components/ui/Button';
import { Badge } from './components/ui/Badge';
import { StatCard } from './components/ui/StatCard';
import { Panel, PanelHeader } from './components/ui/Panel';
import { Modal } from './components/Modal';
import { SettingsModal } from './components/SettingsModal';
import { CreateContainer } from './components/CreateContainer';
import { ContainerLogs } from './components/ContainerLogs';
import { Assistant } from './components/Assistant';
import { HostPanel } from './components/HostPanel';
import { LifecycleBar } from './components/LifecycleBar';
import { Sidebar, PAGES, type PageId } from './components/layout/Sidebar';
import { AuthGate } from './components/AuthGate';
import { Topbar } from './components/layout/Topbar';
import { ContainersPage } from './pages/ContainersPage';
import { ImagesPage } from './pages/ImagesPage';
import { NetworksPage } from './pages/NetworksPage';
import { VolumesPage } from './pages/VolumesPage';
import { WebsitesPage } from './pages/WebsitesPage';
import { AssistantPage, BackupsPage, CertificatesPage, DatabasesPage, FirewallPage, SchedulesPage } from './pages/OpsPages';
import { FilesPage } from './pages/FilesPage';
import { AppStorePage } from './pages/AppStorePage';
import { type Site, type StoreApp, type InstalledApp, type NginxStatus } from './lib/api';

const styles = stylex.create({
  app: {
    display: 'flex',
    minHeight: '100dvh',
    backgroundColor: 'var(--body-bg, #f5f6f8)',
    color: 'var(--text-primary, #0f172a)',
  },
  main: {
    display: 'flex',
    flexDirection: 'column',
    marginLeft: { default: 236, '@media (max-width: 900px)': 0 },
    minWidth: 0,
    flex: 1,
    backgroundColor: 'var(--body-bg, #f5f6f8)',
  },
  content: {
    flex: 1,
    minWidth: 0,
    padding: { default: 20, '@media (max-width: 1200px)': 16, '@media (max-width: 600px)': 12 },
    margin: 0,
  },
  filesMain: {
    height: '100dvh',
    overflow: 'hidden',
  },
  filesContent: {
    display: 'flex',
    flexDirection: 'column',
    minHeight: 0,
  },
  filesWorkspace: {
    flex: 1,
    minHeight: 0,
    alignItems: 'stretch',
  },
  filesColumn: {
    display: 'flex',
    flexDirection: 'column',
    minHeight: 0,
  },
  demoBanner: {
    display: 'flex',
    alignItems: 'center',
    gap: 10,
    backgroundColor: '#fffcf3',
    border: '1px solid #f3e8c9',
    color: '#94652b',
    padding: '11px 14px',
    borderRadius: 8,
    fontSize: 'calc(var(--font-size) - 2px)',
    lineHeight: 1.55,
    marginBottom: 24,
    boxShadow: '0 1px 2px rgba(0, 0, 0, 0.02)',
  },
  error: {
    backgroundColor: '#fef2f2',
    color: '#b91c1c',
    padding: '11px 15px',
    borderRadius: 8,
    marginBottom: 16,
    border: '1px solid #fee2e2',
    fontSize: 12.5,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: 12,
    lineHeight: 1.55,
  },
  statsGrid: {
    display: 'grid',
    gridTemplateColumns: { default: 'repeat(4, minmax(0, 1fr))', '@media (max-width: 1024px)': 'repeat(2, minmax(0, 1fr))' },
    gap: { default: 16, '@media (max-width: 600px)': 12 },
    marginBottom: 24,
  },
  workspace: {
    display: 'grid',
    gridTemplateColumns: { default: 'minmax(0, 1fr) 288px', '@media (max-width: 1439px)': 'minmax(0, 1fr)' },
    gap: 20,
    alignItems: 'start',
  },
  workspaceSolo: {
    display: 'grid',
    gridTemplateColumns: 'minmax(0, 1fr)',
    alignItems: 'start',
  },
  leftColumn: {
    minWidth: 0,
  },
  aside: {
    display: 'grid',
    gridTemplateColumns: { default: '1fr', '@media (min-width: 700px) and (max-width: 1439px)': 'repeat(2, minmax(0, 1fr))' },
    alignItems: 'start',
    gap: 20,
    minWidth: 0,
  },
  hostBody: {
    padding: '20px',
  },
  hostTitle: {
    display: 'flex',
    alignItems: 'center',
    gap: 10,
    marginBottom: 14,
  },
  hostIcon: {
    width: 34,
    height: 34,
    borderRadius: 8,
    backgroundColor: '#f1f5f9',
    display: 'grid',
    placeItems: 'center',
    color: '#475569',
    flexShrink: 0,
    border: '1px solid #e2e8f0',
  },
  hostRow: {
    display: 'flex',
    justifyContent: 'space-between',
    gap: 14,
    color: '#475569',
    fontSize: 'calc(var(--font-size) - 1px)',
    padding: '10px 0',
    borderBottom: '1px solid var(--border-subtle)',
    lineHeight: 1.45,
  },
  hostValue: {
    color: '#0f172a',
    fontWeight: 600,
    textAlign: 'right',
  },
  toast: {
    position: 'fixed',
    bottom: 28,
    left: '50%',
    transform: 'translateX(-50%)',
    zIndex: 50,
    backgroundColor: '#0f172a',
    color: '#f8fafc',
    padding: '10px 18px',
    borderRadius: 8,
    boxShadow: '0 12px 28px -6px rgba(0, 0, 0, 0.28)',
    fontSize: 12.5,
    display: 'flex',
    alignItems: 'center',
    gap: 9,
    maxWidth: '90vw',
  },
  modalActions: {
    display: 'flex',
    justifyContent: 'flex-end',
    gap: 9,
    marginTop: 14,
  },
});

type Dialog =
  | { type: 'create' }
  | { type: 'logs'; container: Container }
  | { type: 'remove'; container: Container }
  | { type: 'settings' }
  | null;

const DOCKER_CHROME_PAGES = new Set<PageId>(['overview', 'containers', 'images', 'networks', 'volumes']);

export function App() {
  return <AuthGate>{(session, logout) => <Dashboard session={session} onLogout={logout} />}</AuthGate>;
}

function Dashboard({ session, onLogout }: { session: AuthSession; onLogout: () => Promise<void> }) {
  const [page, setPage] = useState<PageId>(() => PAGES.find(item => `/${item.id}` === window.location.pathname)?.id || 'overview');
  const [logoutBusy, setLogoutBusy] = useState(false);
  const [filesConnected, setFilesConnected] = useState(false);
  const [filesError, setFilesError] = useState(false);
  const [filesRefresh, setFilesRefresh] = useState(0);
  const [data, setData] = useState<Snapshot | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState('');
  const [search, setSearch] = useState('');
  const [filter, setFilter] = useState('all');
  const [dialog, setDialog] = useState<Dialog>(null);
  const [toast, setToast] = useState('');
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [token, setToken] = useState(() => sessionStorage.getItem('camo-token') || '');
  const [sites, setSites] = useState<Site[]>([]);
  const [nginxStatus, setNginxStatus] = useState<NginxStatus | null>(null);
  const [catalog, setCatalog] = useState<StoreApp[]>([]);
  const [installedApps, setInstalledApps] = useState<InstalledApp[]>([]);
  const [sitesLoading, setSitesLoading] = useState(false);
  const [storeLoading, setStoreLoading] = useState(false);
  const [hostOverview, setHostOverview] = useState<HostOverview | null>(null);
  const [hostProcesses, setHostProcesses] = useState<HostProcess[]>([]);
  const [hostError, setHostError] = useState('');
  const revision = useRef(0);
  const closeSidebar = useCallback(() => setSidebarOpen(false), []);

  const fetchWebsitesAndNginx = useCallback(async () => {
    setSitesLoading(true);
    try {
      const [sitesList, ngx] = await Promise.all([
        api.sites.list().catch(() => []),
        api.nginx.status().catch(() => null),
      ]);
      setSites(sitesList);
      setNginxStatus(ngx);
    } finally {
      setSitesLoading(false);
    }
  }, []);

  const fetchStoreData = useCallback(async () => {
    setStoreLoading(true);
    try {
      const [cat, inst] = await Promise.all([
        api.store.listCatalog().catch(() => []),
        api.store.listInstalled().catch(() => []),
      ]);
      setCatalog(cat);
      setInstalledApps(inst);
    } finally {
      setStoreLoading(false);
    }
  }, []);

  const refresh = useCallback(async (quiet = false) => {
    const current = ++revision.current;
    if (!quiet) setLoading(true);
    const dockerResult = api.overview().then(
      (value) => ({ ok: true as const, value }),
      (err: unknown) => ({ ok: false as const, err }),
    );
    const hostResult = Promise.all([api.host.overview(), api.host.processes()]).then(
      ([overview, processes]) => ({ ok: true as const, overview, processes }),
      (err: unknown) => ({ ok: false as const, err }),
    );
    try {
      const [docker, host] = await Promise.all([
        dockerResult,
        hostResult,
        fetchWebsitesAndNginx(),
        fetchStoreData(),
      ]);
      if (current !== revision.current) return;
      if (docker.ok) {
        setData(docker.value);
        setError('');
      } else {
        setError(docker.err instanceof Error ? docker.err.message : '无法连接后端');
      }
      if (host.ok) {
        setHostOverview(host.overview);
        setHostProcesses(host.processes);
        setHostError('');
      } else {
        setHostError(host.err instanceof Error ? host.err.message : '无法读取主机状态');
      }
    } finally {
      if (current === revision.current) setLoading(false);
    }
  }, [fetchWebsitesAndNginx, fetchStoreData]);

  useEffect(() => {
    void refresh();
    return () => {
      revision.current++;
    };
  }, [refresh]);

  useEffect(() => {
    const timer = window.setInterval(() => {
      if (!busy && !dialog && !loading) void refresh(true);
    }, 15000);
    return () => clearInterval(timer);
  }, [refresh, busy, dialog, loading]);

  useEffect(() => {
    if (!toast) return;
    const timer = window.setTimeout(() => setToast(''), 4000);
    return () => clearTimeout(timer);
  }, [toast]);

  // 全局快捷键支持：Ctrl/Cmd + K 聚焦搜索
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (dialog) return;
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        const el = document.querySelector(sidebarOpen ? 'aside input[type="search"]' : 'main input[type="search"]') as HTMLInputElement | null;
        el?.focus();
      }
      if (e.key === 'Escape' && e.target instanceof HTMLInputElement && e.target.type === 'search') {
        setSearch('');
      }
    }
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [dialog, sidebarOpen]);

  useEffect(() => {
    const pop = () => { setPage(PAGES.find(item => `/${item.id}` === window.location.pathname)?.id || 'overview'); setSearch(''); setFilter('all'); setSidebarOpen(false); };
    window.addEventListener('popstate', pop);
    return () => window.removeEventListener('popstate', pop);
  }, []);

  async function logout() {
    setLogoutBusy(true);
    try { await onLogout(); }
    catch (err) { setToast(err instanceof Error ? err.message : '退出失败，请重试'); }
    finally { setLogoutBusy(false); }
  }

  function navigate(next: PageId) {
    const path = next === 'overview' ? '/' : `/${next}`;
    if (window.location.pathname !== path) window.history.pushState(null, '', path);
    setPage(next);
    setSearch('');
    setFilter('all');
    setSidebarOpen(false);
  }

  async function action(container: Container, kind: string) {
    setBusy(container.Id);
    setError('');
    try {
      await api.action(container.Id, kind);
      setToast(
        `${containerName(container)} · ${
          kind === 'start'
            ? '已启动'
            : kind === 'stop'
            ? '已停止'
            : kind === 'restart'
            ? '已重启'
            : kind === 'pause'
            ? '已暂停'
            : kind === 'unpause'
            ? '已恢复'
            : '已删除'
        }`
      );
      if (kind === 'remove') setDialog(null);
      await refresh(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : '操作失败');
    } finally {
      setBusy('');
    }
  }

  function handleSaveToken(newToken: string) {
    setToken(newToken);
    if (newToken.trim()) sessionStorage.setItem('camo-token', newToken.trim());
    else sessionStorage.removeItem('camo-token');
    void refresh();
  }

  const running = data?.containers.filter(c => c.State === 'running').length || 0;
  const total = data?.containers.length || 0;
  const stopped = total - running;
  const currentPageInfo = PAGES.find(p => p.id === page)!;
  const showDockerChrome = DOCKER_CHROME_PAGES.has(page);
  const title = page === 'overview' ? 'Docker 工作台' : currentPageInfo.label;
  const query = search.toLowerCase().trim();

  const containers = (data?.containers || []).filter(
    c =>
      `${containerName(c)} ${c.Image} ${c.Id}`.toLowerCase().includes(query) &&
      (filter === 'all' || (filter === 'running' ? c.State === 'running' : c.State !== 'running'))
  );
  const imageTags = data?.images.flatMap(i => (i.RepoTags || []).filter(tag => tag !== '<none>:<none>')) || [];
  const images = (data?.images || []).filter(i => `${i.RepoTags?.join(' ')} ${i.Id}`.toLowerCase().includes(query));
  const networks = (data?.networks || []).filter(n => `${n.Name} ${n.Driver}`.toLowerCase().includes(query));
  const volumes = (data?.volumes || []).filter(v => `${v.Name} ${v.Driver} ${v.Mountpoint}`.toLowerCase().includes(query));
  const lastUpdate = data ? new Date(data.generated_at).toLocaleTimeString('zh-CN', { hour12: false }) : '—';

  return (
    <div {...stylex.props(styles.app)}>
      {/* 模块化侧边栏组件 */}
      <Sidebar
        page={page}
        actor={session.actor}
        role={session.role}
        onLogout={() => void logout()}
        logoutBusy={logoutBusy}
        onNavigate={navigate}
        search={search}
        onSearchChange={setSearch}
        data={data}
        siteCount={sites.length}
        installedAppCount={installedApps.length}
        isOpen={sidebarOpen}
        onClose={closeSidebar}
        onOpenSettings={() => {
          setDialog({ type: 'settings' });
          setSidebarOpen(false);
        }}
      />

      {/* 主界面区域 */}
      <main {...stylex.props(styles.main, page === 'files' && styles.filesMain)} aria-label={title}>
        {/* 模块化顶栏组件 */}
        <Topbar
          title={title}
          connectionLabel={page === 'files' ? filesError ? '文件服务异常' : filesConnected ? '服务器已连接' : '等待文件服务' : undefined}
          isConnected={page === 'files' ? filesConnected : !!data}
          hasError={page === 'files' ? filesError : !!error}
          loading={page === 'files' ? false : loading}
          busy={!!busy}
          onRefresh={() => page === 'files' ? setFilesRefresh(value => value + 1) : void refresh()}
          onCreate={() => setDialog({ type: 'create' })}
          showCreate={showDockerChrome}
          onOpenSettings={() => setDialog({ type: 'settings' })}
          onOpenMobileMenu={() => setSidebarOpen(true)}
        />

        <div {...stylex.props(styles.content, page === 'files' && styles.filesContent)}>
          {data?.mode === 'demo' && page !== 'files' && (
            <div {...stylex.props(styles.demoBanner)}>
              <Icon name="info" size={16} />
              <span>
                演示环境 · 操作仅影响演示数据，服务重启后恢复。真实引擎可在连接设置中查看配置说明。
              </span>
            </div>
          )}

          {error && page !== 'files' && (
            <div {...stylex.props(styles.error)} role="alert">
              <span>{error}{data ? ' · 页面保留上次成功同步的数据。' : ' · 请确认后端已启动。'}</span>
              <Button size="sm" onClick={() => void refresh()}>
                <Icon name="refresh" size={14} />
                <span>重连</span>
              </Button>
            </div>
          )}

          {page === 'overview' && (
            <HostPanel
              overview={hostOverview}
              processes={hostProcesses}
              error={hostError}
              loading={loading && !hostOverview}
            />
          )}

          {showDockerChrome && (
            <section {...stylex.props(styles.statsGrid)} aria-label="Docker 概览">
              <StatCard
                title="容器总数"
                value={data ? total : '—'}
                icon="box"
                foot={`${running} 个运行中 · ${stopped} 个未运行`}
                iconColor="#4f46e5"
                iconBg="#eef2ff"
              />
              <StatCard
                title="运行中的容器"
                value={data ? running : '—'}
                icon="play"
                foot={total ? `占全部容器的 ${Math.round((running / total) * 100)}%` : '等待服务运行'}
                green
                iconColor="#059669"
                iconBg="#ecfdf5"
              />
              <StatCard
                title="本地镜像"
                value={data ? data.images.length : '—'}
                icon="layers"
                foot={
                  data
                    ? `逻辑大小合计 ${formatBytes(data.images.reduce((sum, i) => sum + i.Size, 0))}`
                    : '连接后获取镜像信息'
                }
                iconColor="#2563eb"
                iconBg="#eff6ff"
              />
              <StatCard
                title="数据持久卷"
                value={data ? data.volumes.length : '—'}
                icon="database"
                foot={data ? '本地 Docker 数据卷' : '连接后获取存储信息'}
                iconColor="#d97706"
                iconBg="#fffbeb"
              />
            </section>
          )}

          {/* 页面与侧边辅助栏 */}
          <div {...stylex.props(showDockerChrome ? styles.workspace : styles.workspaceSolo, page === 'files' && styles.filesWorkspace)}>
            <div {...stylex.props(styles.leftColumn, page === 'files' && styles.filesColumn)}>
              {(page === 'overview' || page === 'containers') && (
                <>
                  <ContainersPage
                    containers={containers}
                    total={total}
                    running={running}
                    stopped={stopped}
                    search={search}
                    onSearchChange={setSearch}
                    filter={filter}
                    onFilterChange={setFilter}
                    busy={busy}
                    loading={loading}
                    lastUpdate={lastUpdate}
                    onAction={action}
                    onOpenLogs={c => setDialog({ type: 'logs', container: c })}
                    onOpenRemove={c => setDialog({ type: 'remove', container: c })}
                  />
                  <LifecycleBar running={running} total={total} />
                </>
              )}

              {page === 'websites' && (
                <WebsitesPage
                  sites={sites}
                  nginxStatus={nginxStatus}
                  loading={sitesLoading}
                  search={search}
                  onSearchChange={setSearch}
                  onRefresh={() => {
                    void fetchWebsitesAndNginx();
                    void refresh(true);
                  }}
                />
              )}

              {page === 'store' && (
                <AppStorePage
                  catalog={catalog}
                  installed={installedApps}
                  loading={storeLoading}
                  search={search}
                  onSearchChange={setSearch}
                  onRefresh={() => {
                    void fetchStoreData();
                    void refresh(true);
                  }}
                  onCreateProxy={(appName) => {
                    navigate('websites');
                    setSearch(appName);
                  }}
                />
              )}

              {page === 'images' && (
                <ImagesPage
                  images={images}
                  total={data?.images.length || 0}
                  search={search}
                  onSearchChange={setSearch}
                  loading={loading}
                  lastUpdate={lastUpdate}
                  onChanged={() => void refresh(true)}
                />
              )}

              {page === 'networks' && (
                <NetworksPage
                  networks={networks}
                  total={data?.networks.length || 0}
                  search={search}
                  onSearchChange={setSearch}
                  loading={loading}
                  lastUpdate={lastUpdate}
                />
              )}

              {page === 'volumes' && (
                <VolumesPage
                  volumes={volumes}
                  total={data?.volumes.length || 0}
                  search={search}
                  onSearchChange={setSearch}
                  loading={loading}
                  lastUpdate={lastUpdate}
                />
              )}

              {page === 'certificates' && <CertificatesPage />}
              {page === 'files' && <FilesPage refreshKey={filesRefresh} search={search} onSearchChange={setSearch} onConnectionChange={setFilesConnected} onErrorChange={setFilesError} />}
              {page === 'databases' && <DatabasesPage onOpenStore={() => navigate('store')} />}
              {page === 'backups' && <BackupsPage />}
              {page === 'schedules' && <SchedulesPage />}
              {page === 'firewall' && <FirewallPage />}
              {page === 'assistant' && <AssistantPage />}
            </div>

            {showDockerChrome && (
              <aside {...stylex.props(styles.aside)}>
                <Assistant data={data} />
                <Panel>
                  <PanelHeader
                    icon={<Icon name="server" size={17} />}
                    title="引擎信息"
                    badge={<Badge>{data?.mode === 'demo' ? 'DEMO' : 'LOCAL'}</Badge>}
                  />
                  <div {...stylex.props(styles.hostBody)}>
                    <div {...stylex.props(styles.hostTitle)}>
                      <span {...stylex.props(styles.hostIcon)}>
                        <Icon name="server" size={17} />
                      </span>
                      <div>
                        <p style={{ fontSize: 14, fontWeight: 700, color: '#0f172a' }}>
                          {data?.engine.Name || '等待连接'}
                        </p>
                        <span style={{ fontSize: 12, color: '#64748b' }}>
                          {data?.engine.OperatingSystem || '本地 Docker 引擎'}
                        </span>
                      </div>
                    </div>
                    {[
                      ['Docker 版本', data?.engine.ServerVersion || '—'],
                      ['处理器核心', data ? `${data.engine.NCPU} 核` : '—'],
                      ['内存容量', data ? formatBytes(data.engine.MemTotal) : '—'],
                      ['连接方式', data?.mode === 'demo' ? '内存演示服务' : 'Unix socket'],
                    ].map(([label, value]) => (
                      <div key={label} {...stylex.props(styles.hostRow)}>
                        <span>{label}</span>
                        <span {...stylex.props(styles.hostValue)}>{value}</span>
                      </div>
                    ))}
                  </div>
                </Panel>
              </aside>
            )}
          </div>


        </div>
      </main>

      {/* 底部浮动 Toast */}
      {toast && (
        <div {...stylex.props(styles.toast)} role="status">
          <Icon name="check" size={16} />
          <span>{toast}</span>
        </div>
      )}

      {/* 弹窗中心 */}
      {dialog && (
        <>
          {dialog.type === 'settings' ? (
            <Modal onClose={() => setDialog(null)} bare>
              <SettingsModal
                token={token}
                onSaveToken={handleSaveToken}
                data={data}
                onClose={() => setDialog(null)}
              />
            </Modal>
          ) : (
            <Modal
              title={
                dialog.type === 'create'
                  ? '创建容器'
                  : dialog.type === 'logs'
                  ? `${containerName(dialog.container)} · 最近日志`
                  : '删除容器'
              }
              onClose={() => {
                if (!busy) setDialog(null);
              }}
            >
              {dialog.type === 'create' && (
                <CreateContainer
                  images={imageTags}
                  onClose={() => setDialog(null)}
                  onCreated={() => {
                    setDialog(null);
                    setToast('容器已创建，请在列表中启动。');
                    navigate('containers');
                    void refresh();
                  }}
                />
              )}

              {dialog.type === 'logs' && <ContainerLogs id={dialog.container.Id} />}

              {dialog.type === 'remove' && (
                <>
                  <p style={{ fontSize: 13, color: '#334155', lineHeight: 1.6 }}>
                    确认删除容器 <strong>{containerName(dialog.container)}</strong>？容器的可写层将被删除，此操作无法撤销。已挂载的数据卷会保留。
                  </p>
                  {error && (
                    <p style={{ color: '#b91c1c', fontSize: 12.5, marginTop: 10 }}>{error}</p>
                  )}
                  <div {...stylex.props(styles.modalActions)}>
                    <Button disabled={!!busy} onClick={() => setDialog(null)}>
                      取消
                    </Button>
                    <Button
                      variant="danger"
                      disabled={!!busy}
                      onClick={() => void action(dialog.container, 'remove')}
                    >
                      <Icon name="trash" size={15} />
                      <span>{busy ? '删除中…' : '确认删除'}</span>
                    </Button>
                    <Button
                      variant="danger"
                      disabled={!!busy}
                      onClick={() => {
                        setBusy(dialog.container.Id);
                        void api.forceRemove(dialog.container.Id)
                          .then(() => {
                            setDialog(null);
                            setToast(`${containerName(dialog.container)} · 已强制删除，数据卷仍保留`);
                            return refresh(true);
                          })
                          .catch(err => setError(err instanceof Error ? err.message : '强制删除失败'))
                          .finally(() => setBusy(''));
                      }}
                    >
                      <span>确认强制删除</span>
                    </Button>
                  </div>
                </>
              )}
            </Modal>
          )}
        </>
      )}
    </div>
  );
}
