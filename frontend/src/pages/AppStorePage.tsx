import { Select } from "../components/ui/Select";
import { useState, useMemo } from 'react';
import * as stylex from '@stylexjs/stylex';
import { resourceTableStyles as tableStyles } from '../components/ui/resourceTableStyles';
import { api, type StoreApp, type InstalledApp, type ParamSpec, type PortSpec } from '../lib/api';
import { Icon, type IconName } from '../components/Icon';
import { Panel, PanelHeader } from '../components/ui/Panel';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { IconButton } from '../components/ui/IconButton';
import { SearchInput } from '../components/ui/SearchInput';
import { StatusBadge } from '../components/ui/StatusBadge';
import { Modal } from '../components/Modal';

const styles = stylex.create({
  constraintNotice: {
    display: 'flex',
    alignItems: 'flex-start',
    gap: 12,
    padding: '14px 18px',
    backgroundColor: '#f0fdf4',
    border: '1px solid #bbf7d0',
    borderRadius: 10,
    marginBottom: 20,
    fontSize: 13,
    lineHeight: 1.55,
    color: '#166534',
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))',
    gap: 16,
    marginTop: 16,
  },
  card: {
    backgroundColor: '#ffffff',
    border: '1px solid #e2e8f0',
    borderRadius: 10,
    padding: 18,
    display: 'flex',
    flexDirection: 'column',
    transition: 'border-color .15s, box-shadow .15s',
    boxShadow: '0 1px 3px rgba(0,0,0,0.02)',
  },
  cardTop: {
    display: 'flex',
    alignItems: 'flex-start',
    gap: 12,
    marginBottom: 12,
  },
  appIconBox: {
    width: 44,
    height: 44,
    borderRadius: 10,
    backgroundColor: '#f8fafc',
    border: '1px solid #e2e8f0',
    color: 'var(--accent, #d85c49)',
    display: 'grid',
    placeItems: 'center',
    flexShrink: 0,
  },
  cardTitle: {
    fontSize: 15,
    fontWeight: 700,
    color: '#0f172a',
    display: 'flex',
    alignItems: 'center',
    gap: 6,
  },
  cardMeta: {
    fontSize: 11.5,
    color: '#64748b',
    marginTop: 2,
  },
  cardDesc: {
    fontSize: 12.5,
    color: '#475569',
    lineHeight: 1.5,
    flex: 1,
    marginBottom: 14,
  },
  pathBox: {
    backgroundColor: '#f8fafc',
    border: '1px dashed #cbd5e1',
    borderRadius: 6,
    padding: '6px 10px',
    fontSize: 11.5,
    fontFamily: 'monospace',
    color: '#475569',
    marginBottom: 14,
    display: 'flex',
    alignItems: 'center',
    gap: 6,
  },
  cardBottom: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingTop: 12,
    borderTop: '1px solid #f1f5f9',
    marginTop: 'auto',
  },
  actionRow: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'flex-end',
    gap: 6,
  },
  formGroup: {
    display: 'flex',
    flexDirection: 'column',
    gap: 6,
    marginBottom: 14,
  },
  formLabel: {
    fontSize: 13,
    fontWeight: 600,
    color: '#334155',
  },
  formInput: {
    padding: '8px 12px',
    fontSize: 13.5,
    borderRadius: 6,
    border: '1px solid #cbd5e1',
    outline: 'none',
    backgroundColor: '#ffffff',
    color: '#0f172a',
  },
  formHelp: {
    fontSize: 11.5,
    color: '#64748b',
  },
  codeArea: {
    width: '100%',
    minHeight: 240,
    padding: 12,
    fontFamily: 'monospace',
    fontSize: 12,
    lineHeight: 1.5,
    borderRadius: 6,
    border: '1px solid #cbd5e1',
    backgroundColor: '#0f172a',
    color: '#f8fafc',
    outline: 'none',
    whiteSpace: 'pre',
    resize: 'vertical',
  },
});

function generatePassword(length = 16): string {
  const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_#@!';
  let res = '';
  const array = new Uint32Array(length);
  crypto.getRandomValues(array);
  for (let i = 0; i < length; i++) {
    res += chars[array[i] % chars.length];
  }
  return res;
}

function isParamVisible(p: ParamSpec, currentEnv: Record<string, string>): boolean {
  if (!p.show_if) return true;
  const currentVal = currentEnv[p.show_if.field];
  const targetVal = String(p.show_if.value);
  if (p.show_if.operator === 'equals') {
    return String(currentVal) === targetVal;
  }
  if (p.show_if.operator === 'not_equals') {
    return String(currentVal) !== targetVal;
  }
  return true;
}

export function AppStorePage({
  catalog,
  installed,
  search,
  onSearchChange,
  onRefresh,
  onCreateProxy,
}: {
  catalog: StoreApp[];
  installed: InstalledApp[];
  loading: boolean;
  search: string;
  onSearchChange: (v: string) => void;
  onRefresh: () => void;
  onCreateProxy?: (appName: string, port: number) => void;
}) {
  const [category, setCategory] = useState<string>('all');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Modals
  const [installApp, setInstallApp] = useState<StoreApp | null>(null);
  const [composeApp, setComposeApp] = useState<InstalledApp | null>(null);
  const [logsApp, setLogsApp] = useState<InstalledApp | null>(null);
  const [uninstallApp, setUninstallApp] = useState<InstalledApp | null>(null);

  // Install Form
  const [installName, setInstallName] = useState('');
  const [installPorts, setInstallPorts] = useState<Record<string, number>>({});
  const [installEnv, setInstallEnv] = useState<Record<string, string>>({});
  const [customComposeOpen, setCustomComposeOpen] = useState(false);
  const [customCompose, setCustomCompose] = useState('');

  // Compose View/Edit
  const [composeContent, setComposeContent] = useState('');
  // Logs View
  const [logsContent, setLogsContent] = useState('');
  // Uninstall
  const [deleteDataDir, setDeleteDataDir] = useState(true);
  const [showSecretMap, setShowSecretMap] = useState<Record<string, boolean>>({});

  // Open Install Modal
  function openInstall(app: StoreApp) {
    setInstallApp(app);
    setInstallName(app.id);

    const initialPorts: Record<string, number> = {};
    if (app.ports && app.ports.length > 0) {
      for (const p of app.ports) {
        initialPorts[p.key] = p.default;
      }
    } else if (app.default_ports) {
      Object.assign(initialPorts, app.default_ports);
    }
    setInstallPorts(initialPorts);

    const initialEnv: Record<string, string> = {};
    if (app.params && app.params.length > 0) {
      for (const p of app.params) {
        if (p.auto_generate || p.default === 'auto') {
          initialEnv[p.key] = generatePassword(16);
        } else if (p.default !== undefined && p.default !== null) {
          initialEnv[p.key] = String(p.default);
        } else {
          initialEnv[p.key] = '';
        }
      }
    } else if (app.env_params) {
      for (const p of app.env_params) {
        if (p.default === 'auto') {
          initialEnv[p.key] = generatePassword(16);
        } else {
          initialEnv[p.key] = p.default;
        }
      }
    }
    setInstallEnv(initialEnv);
    setShowSecretMap({});
    setCustomCompose(app.compose_template);
    setCustomComposeOpen(false);
    setError(null);
  }

  // Open Compose Modal
  async function openCompose(app: InstalledApp) {
    setComposeApp(app);
    setError(null);
    try {
      const res = await api.store.getCompose(app.id);
      setComposeContent(res.compose);
    } catch (e: any) {
      setError(e.message);
    }
  }

  // Open Logs Modal
  async function openLogs(app: InstalledApp) {
    setLogsApp(app);
    setError(null);
    try {
      const res = await api.store.getLogs(app.id);
      setLogsContent(res.logs);
    } catch (e: any) {
      setError(e.message);
    }
  }

  // Submit Install
  async function handleInstall() {
    if (!installApp) return;
    if (!installName.trim()) {
      setError('请输入应用安装名称');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await api.store.install({
        app_id: installApp.id,
        name: installName.trim(),
        ports: installPorts,
        env: installEnv,
        custom_compose: customComposeOpen ? customCompose : undefined,
      });
      setInstallApp(null);
      onRefresh();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  }

  // Action on Installed App
  async function handleAction(app: InstalledApp, act: 'start' | 'stop' | 'restart') {
    setBusy(true);
    try {
      await api.store.action(app.id, act);
      onRefresh();
    } catch (e: any) {
      alert(e.message);
    } finally {
      setBusy(false);
    }
  }

  // Submit Compose Update
  async function handleUpdateCompose() {
    if (!composeApp) return;
    setBusy(true);
    setError(null);
    try {
      await api.store.updateCompose(composeApp.id, composeContent);
      setComposeApp(null);
      onRefresh();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  }

  // Submit Uninstall
  async function handleUninstall() {
    if (!uninstallApp) return;
    setBusy(true);
    try {
      await api.store.action(uninstallApp.id, 'uninstall', deleteDataDir);
      setUninstallApp(null);
      onRefresh();
    } catch (e: any) {
      alert(e.message);
    } finally {
      setBusy(false);
    }
  }

  // Filter Catalog
  const filteredCatalog = useMemo(() => {
    return catalog.filter(app => {
      if (category !== 'all' && category !== 'installed' && app.category !== category) return false;
      if (!search) return true;
      const q = search.toLowerCase();
      return (
        app.name.toLowerCase().includes(q) ||
        app.description.toLowerCase().includes(q) ||
        app.id.toLowerCase().includes(q)
      );
    });
  }, [catalog, category, search]);

  const installedMap = useMemo(() => {
    const map = new Map<string, InstalledApp>();
    for (const inst of installed) {
      map.set(inst.app_id, inst);
      map.set(inst.id, inst);
    }
    return map;
  }, [installed]);

  return (
    <div>
      {/* 规范受控目录提示框 */}
      <div {...stylex.props(styles.constraintNotice)}>
        <Icon name="shield" size={20} style={{ color: '#16a34a', marginTop: 2, flexShrink: 0 }} />
        <div>
          <strong>严格受控安装规范：</strong>
          应用商店基于标准 Docker Compose 驱动。所有安装程序严格约束在独立沙箱目录 <code>data/apps/&lt;应用名&gt;/</code> 下，
          持久化卷固定映射为 <code>./data</code> 与 <code>./config</code>，严禁逃逸至宿主机根目录或 <code>/etc</code>，保障容器安全与彻底干净卸载。
        </div>
      </div>

      <Panel>
        <PanelHeader
          icon={<Icon name="store" size={17} />}
          title="Docker 应用商店"
          badge={<Badge>{catalog.length}</Badge>}
          extra={<span style={{ fontSize: 12, color: '#64748b' }}>Docker Compose 驱动</span>}
        />

        {/* 顶部工具栏与分类过滤器 */}
        <div {...stylex.props(tableStyles.toolbar)}>
          <SearchInput
            placeholder="搜索商店中的应用名称、数据库或工具…"
            value={search}
            onChange={e => onSearchChange(e.target.value)}
          />

          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
            {[
              { id: 'all', label: '全部应用' },
              { id: 'installed', label: `已安装 (${installed.length})` },
              { id: 'web', label: '网站与代理' },
              { id: 'database', label: '数据库' },
              { id: 'devops', label: 'DevOps' },
              { id: 'tools', label: '实用工具' },
            ].map(cat => (
              <Button
                key={cat.id}
                variant={category === cat.id ? 'primary' : 'default'}
                size="sm"
                onClick={() => setCategory(cat.id)}
              >
                {cat.label}
              </Button>
            ))}
          </div>
        </div>

        {/* 已安装列表视图（若选中已安装标签） */}
        {category === 'installed' ? (
          <div {...stylex.props(tableStyles.tableScroll)}>
            <table {...stylex.props(tableStyles.table)}>
              <thead>
                <tr>
                  <th {...stylex.props(tableStyles.th)}>应用实例</th>
                  <th {...stylex.props(tableStyles.th)}>安装约束路径</th>
                  <th {...stylex.props(tableStyles.th)}>映射端口</th>
                  <th {...stylex.props(tableStyles.th)}>运行状态</th>
                  <th {...stylex.props(tableStyles.th)} style={{ textAlign: 'right' }}>操作</th>
                </tr>
              </thead>
              <tbody>
                {installed.length === 0 ? (
                  <tr>
                    <td colSpan={5} style={{ padding: '36px 20px', textAlign: 'center', color: '#64748b' }}>
                      当前暂未安装任何商店应用，可切换至「全部应用」选择并安装
                    </td>
                  </tr>
                ) : (
                  installed.map(inst => {
                    const firstPort = inst.primary_port || (inst.gateway?.primary_port_key ? inst.ports?.[inst.gateway.primary_port_key] : 0) || Object.values(inst.ports || {})[0] || 8080;
                    return (
                      <tr key={inst.id} {...stylex.props(tableStyles.row)}>
                        <td {...stylex.props(tableStyles.td)}>
                          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                            <div {...stylex.props(styles.appIconBox)} style={{ width: 34, height: 34 }}>
                              <Icon name="box" size={17} />
                            </div>
                            <div>
                              <div style={{ fontWeight: 650, fontSize: 13.5, color: '#0f172a' }}>
                                {inst.name}
                              </div>
                              <span style={{ fontSize: 11.5, color: '#64748b' }}>
                                基于 {inst.app_id} 模板
                              </span>
                            </div>
                          </div>
                        </td>
                        <td {...stylex.props(tableStyles.td)}>
                          <span style={{ fontFamily: 'monospace', fontSize: 12, color: '#475569' }}>
                            {inst.app_dir}
                          </span>
                        </td>
                        <td {...stylex.props(tableStyles.td)}>
                          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                            {Object.entries(inst.ports || {}).map(([k, p]) => (
                              <Badge key={k} variant="kbd">{k}: {p}</Badge>
                            ))}
                          </div>
                        </td>
                        <td {...stylex.props(tableStyles.td)}>
                          <StatusBadge
                            state={inst.status === 'running' ? 'running' : 'exited'}
                            statusText={inst.status === 'running' ? '运行中' : '已停止'}
                          />
                        </td>
                        <td {...stylex.props(tableStyles.td)} style={{ textAlign: 'right' }}>
                          <div {...stylex.props(styles.actionRow)}>
                            {onCreateProxy && (
                              <Button
                                size="sm"
                                onClick={() => onCreateProxy(inst.name, firstPort)}
                                title="一键将此应用端口绑定至网站反向代理"
                              >
                                <Icon name="globe" size={13} />
                                <span>网站反代</span>
                              </Button>
                            )}
                            <IconButton
                              variant={inst.status === 'running' ? 'stop' : 'start'}
                              title={inst.status === 'running' ? '停止应用' : '启动应用'}
                              onClick={() => handleAction(inst, inst.status === 'running' ? 'stop' : 'start')}
                            >
                              <Icon name={inst.status === 'running' ? 'stop' : 'play'} size={15} />
                            </IconButton>
                            <IconButton
                              variant="default"
                              title="查看/编辑 Compose"
                              onClick={() => openCompose(inst)}
                            >
                              <Icon name="code" size={15} />
                            </IconButton>
                            <IconButton
                              variant="default"
                              title="应用日志"
                              onClick={() => openLogs(inst)}
                            >
                              <Icon name="terminal" size={15} />
                            </IconButton>
                            <IconButton
                              variant="remove"
                              title="卸载应用"
                              onClick={() => setUninstallApp(inst)}
                            >
                              <Icon name="trash" size={15} />
                            </IconButton>
                          </div>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        ) : (
          /* 卡片网格视图 */
          <div {...stylex.props(styles.grid)}>
            {filteredCatalog.map(app => {
              const inst = installedMap.get(app.id);
              const isInstalled = !!inst || app.installed;
              const iconName: IconName = (app.icon as IconName) || 'box';

              return (
                <div key={app.id} {...stylex.props(styles.card)}>
                  <div {...stylex.props(styles.cardTop)}>
                    <div {...stylex.props(styles.appIconBox)}>
                      <Icon name={iconName} size={22} />
                    </div>
                    <div>
                      <div {...stylex.props(styles.cardTitle)}>
                        <span>{app.name}</span>
                        <Badge variant="outline">{app.version}</Badge>
                      </div>
                      <div {...stylex.props(styles.cardMeta)}>
                        分类: {app.category} · 标准 Compose
                      </div>
                    </div>
                  </div>

                  <div {...stylex.props(styles.cardDesc)}>
                    {app.description}
                  </div>

                  {/* 受控路径说明 */}
                  <div {...stylex.props(styles.pathBox)}>
                    <Icon name="folder" size={13} />
                    <span>受控目录: apps/{app.id}/ (data, config, logs)</span>
                  </div>

                  <div {...stylex.props(styles.cardBottom)}>
                    <div>
                      {isInstalled ? (
                        <Badge variant="ok">已安装</Badge>
                      ) : (
                        <span style={{ fontSize: 12, color: '#64748b' }}>未安装</span>
                      )}
                    </div>
                    <div>
                      {isInstalled && inst ? (
                        <Button
                          size="sm"
                          onClick={() => {
                            setCategory('installed');
                          }}
                        >
                          管理应用
                        </Button>
                      ) : (
                        <Button
                          variant="primary"
                          size="sm"
                          onClick={() => openInstall(app)}
                        >
                          <Icon name="download" size={14} />
                          <span>一键安装</span>
                        </Button>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </Panel>

      {/* 安装应用弹窗 */}
      {!!installApp && (
        <Modal
          onClose={() => setInstallApp(null)}
          title={`安装应用: ${installApp.name}`}
        >
          <div>
            {error && (
              <div style={{ backgroundColor: '#fef2f2', color: '#b91c1c', padding: '10px 14px', borderRadius: 6, marginBottom: 16, fontSize: 13 }}>
                {error}
              </div>
            )}

            <div {...stylex.props(styles.constraintNotice)} style={{ marginBottom: 16 }}>
              <Icon name="shield" size={18} style={{ color: '#16a34a', flexShrink: 0 }} />
              <div>
                该程序将遵循目录规范安装至专属目录：<code>data/apps/{installName || 'app'}</code>。
                所有持久化数据与配置文件受限在应用专属目录内，安全可控。
              </div>
            </div>

            {/* 应用名称 */}
            <div {...stylex.props(styles.formGroup)}>
              <label {...stylex.props(styles.formLabel)}>实例名称 (创建独立子目录与容器前缀)</label>
              <input
                {...stylex.props(styles.formInput)}
                value={installName}
                onChange={e => setInstallName(e.target.value)}
                placeholder="例如: my-wordpress"
              />
            </div>

            {/* 端口配置 */}
            {((installApp.ports && installApp.ports.length > 0) || Object.keys(installApp.default_ports || {}).length > 0) && (
              <div {...stylex.props(styles.formGroup)}>
                <label {...stylex.props(styles.formLabel)}>服务端口映射</label>
                {installApp.ports && installApp.ports.length > 0 ? (
                  installApp.ports.map(p => {
                    const isProxy = p.proxyable || installApp.gateway?.primary_port_key === p.key;
                    return (
                      <div key={p.key} style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 8, flexWrap: 'wrap' }}>
                        <div style={{ width: 140, display: 'flex', flexDirection: 'column' }}>
                          <span style={{ fontSize: 13, fontWeight: 600, color: '#1e293b' }}>{p.label || p.key}</span>
                          <span style={{ fontSize: 11, color: '#64748b', fontFamily: 'monospace' }}>{p.key}</span>
                        </div>
                        <Badge variant="kbd">容器: {p.container_port}/{p.protocol || 'tcp'}</Badge>
                        {isProxy && <Badge variant="ok">可反代</Badge>}
                        <input
                          type="number"
                          {...stylex.props(styles.formInput)}
                          style={{ width: 110 }}
                          value={installPorts[p.key] ?? p.default}
                          onChange={e => setInstallPorts({ ...installPorts, [p.key]: parseInt(e.target.value, 10) || p.default })}
                        />
                        <span style={{ fontSize: 11.5, color: '#64748b' }}>{p.description || '宿主机访问端口'}</span>
                      </div>
                    );
                  })
                ) : (
                  Object.entries(installApp.default_ports || {}).map(([k, defaultVal]) => (
                    <div key={k} style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 6 }}>
                      <span style={{ fontSize: 12.5, width: 120, color: '#475569', fontFamily: 'monospace' }}>{k}</span>
                      <input
                        type="number"
                        {...stylex.props(styles.formInput)}
                        style={{ width: 120 }}
                        value={installPorts[k] ?? defaultVal}
                        onChange={e => setInstallPorts({ ...installPorts, [k]: parseInt(e.target.value, 10) || defaultVal })}
                      />
                      <span style={{ fontSize: 11.5, color: '#64748b' }}>宿主机外部访问端口</span>
                    </div>
                  ))
                )}
              </div>
            )}

            {/* 环境变量与参数配置 */}
            {((installApp.params && installApp.params.length > 0) || (installApp.env_params && installApp.env_params.length > 0)) && (
              <div {...stylex.props(styles.formGroup)}>
                <label {...stylex.props(styles.formLabel)}>应用参数与秘钥</label>
                {installApp.params && installApp.params.length > 0 ? (
                  installApp.params
                    .filter(p => isParamVisible(p, installEnv))
                    .map(p => {
                      if (p.type === 'boolean') {
                        return (
                          <div key={p.key} style={{ marginBottom: 10 }}>
                            <label style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer', fontSize: 13, fontWeight: 500, color: '#1e293b' }}>
                              <input
                                type="checkbox"
                                checked={installEnv[p.key] === 'true'}
                                onChange={e => setInstallEnv({ ...installEnv, [p.key]: String(e.target.checked) })}
                                style={{ width: 16, height: 16, cursor: 'pointer' }}
                              />
                              <span>{p.label} ({p.key})</span>
                            </label>
                            {p.description && <div {...stylex.props(styles.formHelp)} style={{ marginLeft: 24, marginTop: 2 }}>{p.description}</div>}
                          </div>
                        );
                      }

                      if (p.type === 'select') {
                        return (
                          <div key={p.key} style={{ marginBottom: 10 }}>
                            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, marginBottom: 4 }}>
                              <span style={{ fontWeight: 600, color: '#334155' }}>{p.label} ({p.key})</span>
                              {p.required && <span style={{ color: '#dc2626' }}>必填</span>}
                            </div>
                            <Select aria-label={`${p.label} (${p.key})`} required={p.required}
                              value={installEnv[p.key] ?? p.default ?? ''}
                              onValueChange={value => setInstallEnv({ ...installEnv, [p.key]: value })}
                              options={p.options || []} />
                            {p.description && <div {...stylex.props(styles.formHelp)} style={{ marginTop: 2 }}>{p.description}</div>}
                          </div>
                        );
                      }

                      if (p.type === 'password') {
                        const isPlain = !!showSecretMap[p.key];
                        return (
                          <div key={p.key} style={{ marginBottom: 10 }}>
                            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, marginBottom: 4 }}>
                              <span style={{ fontWeight: 600, color: '#334155' }}>{p.label} ({p.key})</span>
                              {p.required && <span style={{ color: '#dc2626' }}>必填</span>}
                            </div>
                            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                              <input
                                type={isPlain ? 'text' : 'password'}
                                {...stylex.props(styles.formInput)}
                                style={{ flex: 1, fontFamily: isPlain ? 'monospace' : undefined }}
                                value={installEnv[p.key] ?? ''}
                                onChange={e => setInstallEnv({ ...installEnv, [p.key]: e.target.value })}
                              />
                              <IconButton
                                variant="default"
                                title={isPlain ? '隐藏密码' : '显示明文'}
                                onClick={() => setShowSecretMap(m => ({ ...m, [p.key]: !m[p.key] }))}
                              >
                                <Icon name={isPlain ? 'unlock' : 'lock'} size={15} />
                              </IconButton>
                              {p.auto_generate && (
                                <Button
                                  size="sm"
                                  variant="default"
                                  onClick={() => {
                                    const pwd = generatePassword(16);
                                    setInstallEnv(env => ({ ...env, [p.key]: pwd }));
                                    setShowSecretMap(m => ({ ...m, [p.key]: true }));
                                  }}
                                  title="生成 16 位强随机密码"
                                >
                                  <Icon name="sparkle" size={13} />
                                  <span>随机生成</span>
                                </Button>
                              )}
                            </div>
                            {p.description && <div {...stylex.props(styles.formHelp)} style={{ marginTop: 2 }}>{p.description}</div>}
                          </div>
                        );
                      }

                      return (
                        <div key={p.key} style={{ marginBottom: 10 }}>
                          <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, marginBottom: 4 }}>
                            <span style={{ fontWeight: 600, color: '#334155' }}>{p.label} ({p.key})</span>
                            {p.required && <span style={{ color: '#dc2626' }}>必填</span>}
                          </div>
                          <input
                            type={p.type === 'number' ? 'number' : 'text'}
                            {...stylex.props(styles.formInput)}
                            value={installEnv[p.key] ?? ''}
                            onChange={e => setInstallEnv({ ...installEnv, [p.key]: e.target.value })}
                          />
                          {p.description && <div {...stylex.props(styles.formHelp)} style={{ marginTop: 2 }}>{p.description}</div>}
                        </div>
                      );
                    })
                ) : (
                  installApp.env_params.map(p => (
                    <div key={p.key} style={{ marginBottom: 8 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, marginBottom: 2 }}>
                        <span>{p.label} ({p.key})</span>
                        {p.required && <span style={{ color: '#dc2626' }}>必填</span>}
                      </div>
                      <input
                        type={p.is_secret ? 'password' : 'text'}
                        {...stylex.props(styles.formInput)}
                        value={installEnv[p.key] ?? p.default}
                        onChange={e => setInstallEnv({ ...installEnv, [p.key]: e.target.value })}
                      />
                      {p.description && <span {...stylex.props(styles.formHelp)}>{p.description}</span>}
                    </div>
                  ))
                )}
              </div>
            )}

            {/* 自定义 Compose 开关 */}
            <div style={{ margin: '14px 0' }}>
              <Button
                size="sm"
                onClick={() => setCustomComposeOpen(!customComposeOpen)}
              >
                <Icon name="code" size={14} />
                <span>{customComposeOpen ? '收起 Compose 配置' : '查看 / 自定义 Compose 文件'}</span>
              </Button>
            </div>

            {customComposeOpen && (
              <div {...stylex.props(styles.formGroup)}>
                <label {...stylex.props(styles.formLabel)}>Docker Compose YAML</label>
                <textarea
                  {...stylex.props(styles.codeArea)}
                  value={customCompose}
                  onChange={e => setCustomCompose(e.target.value)}
                />
                <span {...stylex.props(styles.formHelp)}>
                  挂载路径必须为受控相对路径（如 ./data）或专属应用目录，禁止挂载宿主机敏感系统文件。
                </span>
              </div>
            )}

            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 20 }}>
              <Button onClick={() => setInstallApp(null)}>取消</Button>
              <Button variant="primary" onClick={handleInstall} disabled={busy}>
                {busy ? '正在部署并启动…' : '确认安装并启动'}
              </Button>
            </div>
          </div>
        </Modal>
      )}

      {/* Compose 查看与修改弹窗 */}
      {!!composeApp && (
        <Modal
          onClose={() => setComposeApp(null)}
          title={`Compose 配置: ${composeApp.name}`}
        >
          <div>
            {error && (
              <div style={{ backgroundColor: '#fef2f2', color: '#b91c1c', padding: '10px 14px', borderRadius: 6, marginBottom: 16, fontSize: 13 }}>
                {error}
              </div>
            )}
            <textarea
              {...stylex.props(styles.codeArea)}
              value={composeContent}
              onChange={e => setComposeContent(e.target.value)}
            />
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 16 }}>
              <Button onClick={() => setComposeApp(null)}>取消</Button>
              <Button variant="primary" onClick={handleUpdateCompose} disabled={busy}>
                {busy ? '保存中…' : '保存配置'}
              </Button>
            </div>
          </div>
        </Modal>
      )}

      {/* 日志弹窗 */}
      {!!logsApp && (
        <Modal
          onClose={() => setLogsApp(null)}
          title={`应用控制台日志: ${logsApp.name}`}
        >
          <div>
            <textarea
              {...stylex.props(styles.codeArea)}
              readOnly
              value={logsContent}
              style={{ minHeight: 340 }}
            />
            <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 16 }}>
              <Button onClick={() => setLogsApp(null)}>关闭</Button>
            </div>
          </div>
        </Modal>
      )}

      {/* 卸载确认弹窗 */}
      {!!uninstallApp && (
        <Modal
          onClose={() => setUninstallApp(null)}
          title="确认卸载应用"
        >
          <div>
            <p style={{ fontSize: 13.5, color: '#334155', lineHeight: 1.6 }}>
              确定要卸载应用 <strong>{uninstallApp.name}</strong> 吗？
              将停止并删除该应用关联的 Docker 容器。
            </p>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, margin: '16px 0' }}>
              <input
                type="checkbox"
                id="delDataCheck"
                checked={deleteDataDir}
                onChange={e => setDeleteDataDir(e.target.checked)}
              />
              <label htmlFor="delDataCheck" style={{ fontSize: 13, cursor: 'pointer', userSelect: 'none', color: '#b91c1c' }}>
                彻底删除专属受控数据目录 ({uninstallApp.app_dir})
              </label>
            </div>
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 24 }}>
              <Button onClick={() => setUninstallApp(null)}>取消</Button>
              <Button variant="danger" onClick={handleUninstall} disabled={busy}>
                {busy ? '卸载中…' : '确认卸载'}
              </Button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
}
