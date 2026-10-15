import { Select } from "../components/ui/Select";
import { useState, useMemo } from 'react';
import * as stylex from '@stylexjs/stylex';
import { resourceTableStyles as tableStyles } from '../components/ui/resourceTableStyles';
import { api, type Site, type NginxStatus, type SSLInfo } from '../lib/api';
import { Icon } from '../components/Icon';
import { Panel, PanelHeader } from '../components/ui/Panel';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { IconButton } from '../components/ui/IconButton';
import { SearchInput } from '../components/ui/SearchInput';
import { StatusBadge } from '../components/ui/StatusBadge';
import { Modal } from '../components/Modal';

const styles = stylex.create({
  gatewayBar: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    flexWrap: 'wrap',
    gap: 12,
    padding: '14px 18px',
    backgroundColor: '#ffffff',
    border: '1px solid #e2e8f0',
    borderRadius: 10,
    marginBottom: 20,
    boxShadow: '0 1px 3px rgba(0,0,0,0.02)',
  },
  gatewayLeft: {
    display: 'flex',
    alignItems: 'center',
    gap: 12,
  },
  gatewayIcon: {
    width: 36,
    height: 36,
    borderRadius: 8,
    backgroundColor: '#eff6ff',
    color: '#2563eb',
    display: 'grid',
    placeItems: 'center',
    flexShrink: 0,
  },
  gatewayTitle: {
    fontSize: 14,
    fontWeight: 650,
    color: '#0f172a',
    display: 'flex',
    alignItems: 'center',
    gap: 8,
  },
  gatewayMeta: {
    fontSize: 12,
    color: '#64748b',
    marginTop: 2,
  },
  gatewayActions: {
    display: 'flex',
    alignItems: 'center',
    gap: 8,
  },
  statsRow: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))',
    gap: 12,
    marginBottom: 20,
  },
  statCard: {
    padding: '12px 16px',
    backgroundColor: '#ffffff',
    border: '1px solid #e2e8f0',
    borderRadius: 8,
  },
  statLabel: {
    fontSize: 12,
    color: '#64748b',
    fontWeight: 500,
  },
  statValue: {
    fontSize: 22,
    fontWeight: 700,
    color: '#0f172a',
    marginTop: 4,
  },
  domainCell: {
    display: 'flex',
    flexDirection: 'column',
    gap: 4,
  },
  primaryDomain: {
    fontSize: 13.5,
    fontWeight: 600,
    color: '#0f172a',
    display: 'flex',
    alignItems: 'center',
    gap: 6,
    textDecoration: 'none',
  },
  aliasList: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: 4,
  },
  aliasBadge: {
    fontSize: 11,
    padding: '1px 6px',
    borderRadius: 4,
    backgroundColor: '#f1f5f9',
    color: '#475569',
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
    marginBottom: 16,
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
    lineHeight: 1.4,
  },
  typeToggleRow: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))',
    gap: 10,
    marginBottom: 16,
  },
  typeOption: {
    display: 'flex',
    alignItems: 'center',
    gap: 10,
    padding: '12px 14px',
    borderRadius: 8,
    border: '1.5px solid #e2e8f0',
    backgroundColor: '#ffffff',
    cursor: 'pointer',
    textAlign: 'left',
    transition: 'all .15s ease',
  },
  typeOptionActive: {
    borderColor: 'var(--accent, #d85c49)',
    backgroundColor: 'var(--accent-light, #fef2f0)',
  },
  codeArea: {
    width: '100%',
    minHeight: 280,
    padding: 12,
    fontFamily: 'monospace',
    fontSize: 12.5,
    lineHeight: 1.5,
    borderRadius: 6,
    border: '1px solid #cbd5e1',
    backgroundColor: '#0f172a',
    color: '#f8fafc',
    outline: 'none',
    whiteSpace: 'pre',
    resize: 'vertical',
  },
  tabsRow: {
    display: 'flex',
    gap: 6,
    borderBottom: '1px solid #e2e8f0',
    paddingBottom: 8,
    marginBottom: 16,
  },
  tabBtn: {
    padding: '6px 12px',
    borderRadius: 6,
    borderWidth: 0,
    fontSize: 12.5,
    fontWeight: 500,
    cursor: 'pointer',
    backgroundColor: 'transparent',
    color: '#64748b',
  },
  tabBtnActive: {
    backgroundColor: 'var(--accent, #d85c49)',
    color: '#ffffff',
    fontWeight: 600,
  },
  sslCard: {
    backgroundColor: '#f8fafc',
    border: '1px solid #e2e8f0',
    borderRadius: 8,
    padding: 12,
    marginBottom: 16,
    fontSize: 12.5,
    lineHeight: 1.6,
  },
});

export function WebsitesPage({
  sites,
  nginxStatus,
  loading,
  search,
  onSearchChange,
  onRefresh,
}: {
  sites: Site[];
  nginxStatus: NginxStatus | null;
  loading: boolean;
  search: string;
  onSearchChange: (v: string) => void;
  onRefresh: () => void;
}) {
  const [filterType, setFilterType] = useState<string>('all');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Modals state
  const [createOpen, setCreateOpen] = useState(false);
  const [editSite, setEditSite] = useState<Site | null>(null);
  const [sslSite, setSslSite] = useState<Site | null>(null);
  const [fileSite, setFileSite] = useState<Site | null>(null);
  const [configSite, setConfigSite] = useState<Site | null>(null);
  const [deleteSite, setDeleteSite] = useState<Site | null>(null);

  // Form states
  const [formType, setFormType] = useState<'static' | 'proxy' | 'runtime'>('static');
  const [formRuntime, setFormRuntime] = useState('php');
  const [formName, setFormName] = useState('');
  const [formPrimaryDomain, setFormPrimaryDomain] = useState('');
  const [formDomains, setFormDomains] = useState('');
  const [formTarget, setFormTarget] = useState('http://127.0.0.1:8080');
  const [formForceHTTPS, setFormForceHTTPS] = useState(false);
  const [formCustomConfig, setFormCustomConfig] = useState('');
  const [formRedirectFrom, setFormRedirectFrom] = useState('');
  const [formRedirectTo, setFormRedirectTo] = useState('');
  const [formLocationPath, setFormLocationPath] = useState('');
  const [formLocationProxy, setFormLocationProxy] = useState('');
  const [formAuthUser, setFormAuthUser] = useState('');
  const [formAuthPass, setFormAuthPass] = useState('');
  const [formHotlink, setFormHotlink] = useState(false);
  const [formRate, setFormRate] = useState('');

  // SSL form states
  const [sslProvider, setSslProvider] = useState<'letsencrypt' | 'self-signed' | 'custom'>('letsencrypt');
  const [sslEmail, setSslEmail] = useState('');
  const [sslCertPEM, setSslCertPEM] = useState('');
  const [sslKeyPEM, setSslKeyPEM] = useState('');
  const [sslForceHTTPS, setSslForceHTTPS] = useState(true);
  const [sslInfo, setSslInfo] = useState<SSLInfo | null>(null);

  // File editor states
  const [fileContent, setFileContent] = useState('');
  const [configContent, setConfigContent] = useState('');

  // Reset create form
  function openCreate() {
    setFormType('static');
    setFormRuntime('php');
    setFormName('');
    setFormPrimaryDomain('');
    setFormDomains('');
    setFormTarget('http://127.0.0.1:8080');
    setFormForceHTTPS(false);
    setFormCustomConfig('');
    setFormRedirectFrom('');
    setFormRedirectTo('');
    setFormLocationPath('');
    setFormLocationProxy('');
    setFormAuthUser('');
    setFormAuthPass('');
    setFormHotlink(false);
    setFormRate('');
    setError(null);
    setCreateOpen(true);
  }

  function siteRules() {
    return {
      redirects: formRedirectFrom.trim() && formRedirectTo.trim() ? [{ from: formRedirectFrom.trim(), to: formRedirectTo.trim(), code: 301 }] : [],
      locations: formLocationPath.trim() && formLocationProxy.trim() ? [{ path: formLocationPath.trim(), proxy: formLocationProxy.trim() }] : [],
      basic_auth_user: formAuthUser.trim(),
      basic_auth_password: formAuthPass,
      hotlink: formHotlink,
      rate_limit: formRate ? Number(formRate) : 0,
    };
  }

  // Open edit modal
  function openEdit(s: Site) {
    setEditSite(s);
    setFormType(s.type);
    setFormRuntime(s.runtime || 'php');
    setFormName(s.name);
    setFormPrimaryDomain(s.primary_domain);
    setFormDomains((s.domains || []).join(', '));
    setFormTarget(s.proxy_target || 'http://127.0.0.1:8080');
    setFormForceHTTPS(s.force_https);
    setFormCustomConfig(s.custom_config || '');
    setError(null);
  }

  // Open SSL modal
  async function openSSL(s: Site) {
    setSslSite(s);
    setSslProvider('letsencrypt');
    setSslEmail('');
    setSslCertPEM('');
    setSslKeyPEM('');
    setSslForceHTTPS(s.force_https || true);
    setError(null);
    try {
      const info = await api.sites.getSSL(s.id);
      setSslInfo(info);
    } catch {
      setSslInfo(null);
    }
  }

  // Open File modal
  async function openFile(s: Site) {
    setFileSite(s);
    setError(null);
    try {
      const res = await api.sites.getFile(s.id, 'index.html');
      setFileContent(res.content);
    } catch {
      setFileContent('<!DOCTYPE html>\n<html>\n<head><title>My Site</title></head>\n<body><h1>Hello World</h1></body>\n</html>');
    }
  }

  // Open Config modal
  async function openConfig(s: Site) {
    setConfigSite(s);
    setError(null);
    try {
      const res = await api.sites.getConfig(s.id);
      setConfigContent(res.config);
    } catch (e: any) {
      setError(e.message);
    }
  }

  // Submit Create Site
  async function handleCreate() {
    if (!formName.trim() || !formPrimaryDomain.trim()) {
      setError('请填写网站名称和主域名');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const domainsArr = formDomains.split(',').map(d => d.trim()).filter(Boolean);
      await api.sites.create({
        name: formName.trim(),
        primary_domain: formPrimaryDomain.trim(),
        domains: domainsArr,
        type: formType,
        runtime: formType === 'runtime' ? formRuntime : undefined,
        proxy_target: formType === 'proxy' ? formTarget.trim() : undefined,
        force_https: formForceHTTPS,
        custom_config: formCustomConfig.trim() || undefined,
        ...siteRules(),
      });
      setCreateOpen(false);
      onRefresh();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  }

  // Submit Update Site
  async function handleUpdate() {
    if (!editSite) return;
    setBusy(true);
    setError(null);
    try {
      const domainsArr = formDomains.split(',').map(d => d.trim()).filter(Boolean);
      await api.sites.update(editSite.id, {
        name: formName.trim(),
        primary_domain: formPrimaryDomain.trim(),
        domains: domainsArr,
        type: formType,
        runtime: formType === 'runtime' ? formRuntime : undefined,
        proxy_target: formType === 'proxy' ? formTarget.trim() : undefined,
        force_https: formForceHTTPS,
        custom_config: formCustomConfig.trim() || undefined,
        ...siteRules(),
      });
      setEditSite(null);
      onRefresh();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  }

  // Submit SSL Apply
  async function handleApplySSL() {
    if (!sslSite) return;
    setBusy(true);
    setError(null);
    try {
      await api.sites.applySSL(sslSite.id, {
        provider: sslProvider,
        email: sslEmail.trim() || undefined,
        cert_pem: sslProvider === 'custom' ? sslCertPEM : undefined,
        key_pem: sslProvider === 'custom' ? sslKeyPEM : undefined,
        force_https: sslForceHTTPS,
      });
      setSslSite(null);
      onRefresh();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  }

  // Submit Save File
  async function handleSaveFile() {
    if (!fileSite) return;
    setBusy(true);
    setError(null);
    try {
      await api.sites.saveFile(fileSite.id, 'index.html', fileContent);
      setFileSite(null);
      onRefresh();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  }

  // Toggle Site Running/Stopped
  async function handleToggle(s: Site) {
    try {
      await api.sites.toggle(s.id);
      onRefresh();
    } catch (e: any) {
      alert(e.message);
    }
  }

  // Delete Site
  async function handleDelete() {
    if (!deleteSite) return;
    setBusy(true);
    try {
      await api.sites.delete(deleteSite.id);
      setDeleteSite(null);
      onRefresh();
    } catch (e: any) {
      alert(e.message);
    } finally {
      setBusy(false);
    }
  }

  // Nginx Gateway Actions
  async function handleNginxAction(act: 'restart' | 'reload') {
    setBusy(true);
    try {
      await api.nginx.action(act);
      onRefresh();
    } catch (e: any) {
      alert(e.message);
    } finally {
      setBusy(false);
    }
  }

  // Filtered sites
  const filteredSites = useMemo(() => {
    return sites.filter(s => {
      if (filterType === 'static' && s.type !== 'static') return false;
      if (filterType === 'proxy' && s.type !== 'proxy') return false;
      if (filterType === 'runtime' && s.type !== 'runtime') return false;
      if (filterType === 'ssl' && !s.ssl_enabled) return false;
      if (!search) return true;
      const q = search.toLowerCase();
      return (
        s.name.toLowerCase().includes(q) ||
        s.primary_domain.toLowerCase().includes(q) ||
        (s.domains && s.domains.some(d => d.toLowerCase().includes(q))) ||
        (s.proxy_target && s.proxy_target.toLowerCase().includes(q))
      );
    });
  }, [sites, filterType, search]);

  const totalSites = sites.length;
  const runningSites = sites.filter(s => s.status === 'running').length;
  const sslSites = sites.filter(s => s.ssl_enabled).length;
  const staticSites = sites.filter(s => s.type === 'static').length;

  return (
    <div>
      {/* 内置 Nginx 网关状态条 */}
      <div {...stylex.props(styles.gatewayBar)}>
        <div {...stylex.props(styles.gatewayLeft)}>
          <div {...stylex.props(styles.gatewayIcon)}>
            <Icon name="server" size={20} />
          </div>
          <div>
            <div {...stylex.props(styles.gatewayTitle)}>
              <span>内置 Nginx 网站网关</span>
              {nginxStatus?.running ? (
                <StatusBadge state="running" statusText="运行中 (80/443)" />
              ) : (
                <StatusBadge state="exited" statusText="已停止" />
              )}
            </div>
            <div {...stylex.props(styles.gatewayMeta)}>
              HTTP 端口: {nginxStatus?.http_port || 80} · HTTPS 端口: {nginxStatus?.https_port || 443} · 托管站点: {nginxStatus?.active_site_count || 0} 个
            </div>
          </div>
        </div>
        <div {...stylex.props(styles.gatewayActions)}>
          <Button
            size="sm"
            onClick={() => handleNginxAction('reload')}
            disabled={busy}
          >
            <Icon name="refresh" size={14} />
            <span>重载配置</span>
          </Button>
          <Button
            size="sm"
            onClick={() => handleNginxAction('restart')}
            disabled={busy}
          >
            <Icon name="play" size={14} />
            <span>重启网关</span>
          </Button>
          <Button variant="primary" size="sm" onClick={openCreate}>
            <Icon name="plus" size={15} />
            <span>新建网站</span>
          </Button>
        </div>
      </div>

      {/* 统计指标卡片 */}
      <div {...stylex.props(styles.statsRow)}>
        <div {...stylex.props(styles.statCard)}>
          <div {...stylex.props(styles.statLabel)}>总网站数</div>
          <div {...stylex.props(styles.statValue)}>{totalSites}</div>
        </div>
        <div {...stylex.props(styles.statCard)}>
          <div {...stylex.props(styles.statLabel)}>运行中站点</div>
          <div {...stylex.props(styles.statValue)} style={{ color: '#16a34a' }}>{runningSites}</div>
        </div>
        <div {...stylex.props(styles.statCard)}>
          <div {...stylex.props(styles.statLabel)}>已启用 HTTPS</div>
          <div {...stylex.props(styles.statValue)} style={{ color: '#2563eb' }}>{sslSites}</div>
        </div>
        <div {...stylex.props(styles.statCard)}>
          <div {...stylex.props(styles.statLabel)}>静态 / 反代</div>
          <div {...stylex.props(styles.statValue)}>{staticSites} / {totalSites - staticSites}</div>
        </div>
      </div>

      {/* 主面板：站点列表 */}
      <Panel>
        <PanelHeader
          icon={<Icon name="globe" size={17} />}
          title="网站管理"
          badge={<Badge>{filteredSites.length}</Badge>}
          extra={<span style={{ fontSize: 12, color: '#64748b' }}>内置 Nginx 虚拟主机</span>}
        />

        {/* 工具栏 */}
        <div {...stylex.props(tableStyles.toolbar)}>
          <SearchInput
            placeholder="搜索网站名称、主域名、别名或反代目标…"
            value={search}
            onChange={e => onSearchChange(e.target.value)}
          />

          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
            {[
              { id: 'all', label: '全部' },
              { id: 'static', label: '静态站点' },
              { id: 'proxy', label: '反向代理' },
              { id: 'runtime', label: '运行环境' },
              { id: 'ssl', label: '已启用 SSL' },
            ].map(f => (
              <Button
                key={f.id}
                variant={filterType === f.id ? 'primary' : 'default'}
                size="sm"
                onClick={() => setFilterType(f.id)}
              >
                {f.label}
              </Button>
            ))}
          </div>
        </div>

        {/* 站点表格 */}
        <div {...stylex.props(tableStyles.tableScroll)}>
          <table {...stylex.props(tableStyles.table)}>
            <thead>
              <tr>
                <th {...stylex.props(tableStyles.th)}>网站名称 / 域名绑定</th>
                <th {...stylex.props(tableStyles.th)}>类型与目标</th>
                <th {...stylex.props(tableStyles.th)}>SSL 证书</th>
                <th {...stylex.props(tableStyles.th)}>运行状态</th>
                <th {...stylex.props(tableStyles.th)} style={{ textAlign: 'right' }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {filteredSites.length === 0 ? (
                <tr>
                  <td colSpan={5} style={{ padding: '36px 20px', textAlign: 'center', color: '#64748b' }}>
                    {loading ? '加载网站列表中…' : '暂无网站，点击右上角「新建网站」开始添加'}
                  </td>
                </tr>
              ) : (
                filteredSites.map(s => {
                  const protocol = s.ssl_enabled ? 'https://' : 'http://';
                  const primaryURL = `${protocol}${s.primary_domain}`;
                  return (
                    <tr key={s.id} {...stylex.props(tableStyles.row)}>
                      {/* 名称与域名 */}
                      <td {...stylex.props(tableStyles.td)}>
                        <div {...stylex.props(styles.domainCell)}>
                          <a
                            href={primaryURL}
                            target="_blank"
                            rel="noreferrer"
                            {...stylex.props(styles.primaryDomain)}
                            title="在新标签页中访问该网站"
                          >
                            <span>{s.primary_domain}</span>
                            <Icon name="external" size={13} style={{ color: '#94a3b8' }} />
                          </a>
                          <span style={{ fontSize: 12, color: '#64748b' }}>{s.name}</span>
                          {s.domains && s.domains.length > 0 && (
                            <div {...stylex.props(styles.aliasList)}>
                              {s.domains.map(d => (
                                <span key={d} {...stylex.props(styles.aliasBadge)}>
                                  {d}
                                </span>
                              ))}
                            </div>
                          )}
                        </div>
                      </td>

                      {/* 类型与目标 */}
                      <td {...stylex.props(tableStyles.td)}>
                        {s.type === 'static' ? (
                          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                            <Badge variant="outline">静态站点</Badge>
                            <span style={{ fontSize: 12, color: '#64748b' }}>HTML / 资源托管</span>
                          </div>
                        ) : s.type === 'runtime' ? (
                          <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                            <Badge variant="outline">运行环境 · {s.runtime || 'runtime'}</Badge>
                            <span style={{ fontSize: 12, fontFamily: 'monospace', color: '#0369a1' }}>{s.proxy_target}</span>
                          </div>
                        ) : (
                          <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                              <Badge variant="outline">反向代理</Badge>
                            </div>
                            <span style={{ fontSize: 12, fontFamily: 'monospace', color: '#0369a1' }}>
                              {s.proxy_target}
                            </span>
                          </div>
                        )}
                      </td>

                      {/* SSL 状态 */}
                      <td {...stylex.props(tableStyles.td)}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                          {s.ssl_enabled ? (
                            <Badge variant="ok">
                              <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                                <Icon name="lock" size={12} />
                                <span>HTTPS 已开启</span>
                              </span>
                            </Badge>
                          ) : (
                            <Badge variant="count">
                              <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                                <Icon name="unlock" size={12} />
                                <span>未开启</span>
                              </span>
                            </Badge>
                          )}
                          {s.ssl_enabled && <span style={{ fontSize: 12, color: '#475569' }}>剩余 {s.days_remaining ?? 0} 天</span>}
                          <Button
                            size="sm"
                            onClick={() => openSSL(s)}
                            title="证书管理与一键申请"
                          >
                            <Icon name="shield" size={14} />
                            <span>{s.ssl_enabled ? '管理' : '一键申请'}</span>
                          </Button>
                        </div>
                      </td>

                      {/* 运行状态 */}
                      <td {...stylex.props(tableStyles.td)}>
                        <StatusBadge
                          state={s.status === 'running' ? 'running' : 'exited'}
                          statusText={s.status === 'running' ? '运行中' : '已停用'}
                        />
                      </td>

                      {/* 操作 */}
                      <td {...stylex.props(tableStyles.td)} style={{ textAlign: 'right' }}>
                        <div {...stylex.props(styles.actionRow)}>
                          {s.type === 'static' && (
                            <IconButton
                              variant="default"
                              title="编辑静态页面"
                              onClick={() => openFile(s)}
                            >
                              <Icon name="fileText" size={15} />
                            </IconButton>
                          )}
                          <IconButton
                            variant="default"
                            title="预览 Nginx 配置"
                            onClick={() => openConfig(s)}
                          >
                            <Icon name="code" size={15} />
                          </IconButton>
                          <IconButton
                            variant={s.status === 'running' ? 'stop' : 'start'}
                            title={s.status === 'running' ? '停用站点' : '启用站点'}
                            onClick={() => handleToggle(s)}
                          >
                            <Icon name={s.status === 'running' ? 'stop' : 'play'} size={15} />
                          </IconButton>
                          <IconButton
                            variant="default"
                            title="修改配置"
                            onClick={() => openEdit(s)}
                          >
                            <Icon name="settings" size={15} />
                          </IconButton>
                          <IconButton
                            variant="remove"
                            title="删除网站"
                            onClick={() => setDeleteSite(s)}
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
      </Panel>

      {/* 创建 / 编辑网站弹窗 */}
      {(createOpen || !!editSite) && (
        <Modal
          onClose={() => {
            setCreateOpen(false);
            setEditSite(null);
          }}
          title={editSite ? `编辑网站: ${editSite.name}` : '新建网站与域名绑定'}
        >
          <div>
            {error && (
              <div style={{ backgroundColor: '#fef2f2', color: '#b91c1c', padding: '10px 14px', borderRadius: 6, marginBottom: 16, fontSize: 13 }}>
                {error}
              </div>
            )}

            {/* 类型选择 */}
            <div {...stylex.props(styles.typeToggleRow)}>
              <button
                type="button"
                {...stylex.props(styles.typeOption, formType === 'static' && styles.typeOptionActive)}
                onClick={() => setFormType('static')}
              >
                <Icon name="folder" size={20} style={{ color: formType === 'static' ? '#d85c49' : '#64748b' }} />
                <div>
                  <div style={{ fontWeight: 650, fontSize: 13.5 }}>静态站点</div>
                  <div style={{ fontSize: 11.5, color: '#64748b' }}>托管 HTML / CSS / JS 前端页面</div>
                </div>
              </button>
              <button
                type="button"
                {...stylex.props(styles.typeOption, formType === 'proxy' && styles.typeOptionActive)}
                onClick={() => setFormType('proxy')}
              >
                <Icon name="link" size={20} style={{ color: formType === 'proxy' ? '#d85c49' : '#64748b' }} />
                <div>
                  <div style={{ fontWeight: 650, fontSize: 13.5 }}>反向代理</div>
                  <div style={{ fontSize: 11.5, color: '#64748b' }}>将域名流量转发至本地端口或容器</div>
                </div>
              </button>
              <button
                type="button"
                {...stylex.props(styles.typeOption, formType === 'runtime' && styles.typeOptionActive)}
                onClick={() => setFormType('runtime')}
              >
                <Icon name="server" size={20} style={{ color: formType === 'runtime' ? '#d85c49' : '#64748b' }} />
                <div>
                  <div style={{ fontWeight: 650, fontSize: 13.5 }}>运行环境</div>
                  <div style={{ fontSize: 11.5, color: '#64748b' }}>PHP、Node、Java、Go、Python</div>
                </div>
              </button>
            </div>

            {/* 网站名称 */}
            <div {...stylex.props(styles.formGroup)}>
              <label {...stylex.props(styles.formLabel)}>网站名称</label>
              <input
                {...stylex.props(styles.formInput)}
                placeholder="例如: my-blog"
                value={formName}
                onChange={e => setFormName(e.target.value)}
              />
              <span {...stylex.props(styles.formHelp)}>仅限英文字母、数字、点与中划线</span>
            </div>

            {/* 主域名 */}
            <div {...stylex.props(styles.formGroup)}>
              <label {...stylex.props(styles.formLabel)}>主域名绑定 (Primary Domain)</label>
              <input
                {...stylex.props(styles.formInput)}
                placeholder="例如: blog.example.com 或 127.0.0.1"
                value={formPrimaryDomain}
                onChange={e => setFormPrimaryDomain(e.target.value)}
              />
              <span {...stylex.props(styles.formHelp)}>请确保该域名在 DNS 解析中指向当前服务器 IP</span>
            </div>

            {/* 额外绑定域名 */}
            <div {...stylex.props(styles.formGroup)}>
              <label {...stylex.props(styles.formLabel)}>额外绑定别名域名 (可选，英文逗号分隔)</label>
              <input
                {...stylex.props(styles.formInput)}
                placeholder="例如: www.example.com, test.example.com"
                value={formDomains}
                onChange={e => setFormDomains(e.target.value)}
              />
            </div>

            {/* 反向代理目标地址 */}
            {formType === 'proxy' && (
              <div {...stylex.props(styles.formGroup)}>
                <label {...stylex.props(styles.formLabel)}>目标代理地址 (Proxy Pass Target)</label>
                <input
                  {...stylex.props(styles.formInput)}
                  placeholder="例如: http://127.0.0.1:3000 或 http://127.0.0.1:8080"
                  value={formTarget}
                  onChange={e => setFormTarget(e.target.value)}
                />
                <span {...stylex.props(styles.formHelp)}>支持 WebSocket 协议与自适应代理头部</span>
              </div>
            )}

            {formType === 'runtime' && (
              <div {...stylex.props(styles.formGroup)}>
                <label {...stylex.props(styles.formLabel)}>运行环境</label>
                <Select aria-label="运行环境" value={formRuntime} onValueChange={setFormRuntime}
                  options={['php', 'node', 'java', 'go', 'python'].map(value => ({ value, label: value }))} />
                <span {...stylex.props(styles.formHelp)}>站点由对应运行时容器反代，Java、Go 和 Python 与 PHP、Node 走同一条路径</span>
              </div>
            )}

            {/* 强制 HTTPS */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, margin: '14px 0' }}>
              <input
                type="checkbox"
                id="forceHttpsCheck"
                checked={formForceHTTPS}
                onChange={e => setFormForceHTTPS(e.target.checked)}
              />
              <label htmlFor="forceHttpsCheck" style={{ fontSize: 13, cursor: 'pointer', userSelect: 'none' }}>
                开启 HTTP 自动跳转 HTTPS (需在配置证书后生效)
              </label>
            </div>

            {/* 自定义 Nginx 指令 */}
            <div {...stylex.props(styles.formGroup)}>
              <label {...stylex.props(styles.formLabel)}>跳转（从 / 到）</label>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                <input {...stylex.props(styles.formInput)} placeholder="/old" value={formRedirectFrom} onChange={e => setFormRedirectFrom(e.target.value)} />
                <input {...stylex.props(styles.formInput)} placeholder="/new" value={formRedirectTo} onChange={e => setFormRedirectTo(e.target.value)} />
              </div>
            </div>
            <div {...stylex.props(styles.formGroup)}>
              <label {...stylex.props(styles.formLabel)}>额外路径反代</label>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                <input {...stylex.props(styles.formInput)} placeholder="/api" value={formLocationPath} onChange={e => setFormLocationPath(e.target.value)} />
                <input {...stylex.props(styles.formInput)} placeholder="http://127.0.0.1:9000" value={formLocationProxy} onChange={e => setFormLocationProxy(e.target.value)} />
              </div>
            </div>
            <div {...stylex.props(styles.formGroup)}>
              <label {...stylex.props(styles.formLabel)}>目录基本认证</label>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                <input {...stylex.props(styles.formInput)} placeholder="用户名" value={formAuthUser} onChange={e => setFormAuthUser(e.target.value)} />
                <input {...stylex.props(styles.formInput)} type="password" placeholder="密码" value={formAuthPass} onChange={e => setFormAuthPass(e.target.value)} />
              </div>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, margin: '8px 0' }}>
              <input type="checkbox" id="hotlinkCheck" checked={formHotlink} onChange={e => setFormHotlink(e.target.checked)} />
              <label htmlFor="hotlinkCheck" style={{ fontSize: 13 }}>防盗链</label>
              <input {...stylex.props(styles.formInput)} style={{ maxWidth: 120 }} type="number" min="0" placeholder="限流" value={formRate} onChange={e => setFormRate(e.target.value)} />
            </div>
            <div {...stylex.props(styles.formGroup)}>
              <label {...stylex.props(styles.formLabel)}>自定义 Nginx 指令 (可选高级配置)</label>
              <textarea
                {...stylex.props(styles.formInput)}
                rows={3}
                placeholder="例如: client_max_body_size 50m;"
                value={formCustomConfig}
                onChange={e => setFormCustomConfig(e.target.value)}
              />
            </div>

            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 20 }}>
              <Button
                onClick={() => {
                  setCreateOpen(false);
                  setEditSite(null);
                }}
              >
                取消
              </Button>
              <Button
                variant="primary"
                onClick={editSite ? handleUpdate : handleCreate}
                disabled={busy}
              >
                {busy ? '正在应用配置…' : editSite ? '保存修改' : '确认创建'}
              </Button>
            </div>
          </div>
        </Modal>
      )}

      {/* SSL 证书管理与一键申请弹窗 */}
      {!!sslSite && (
        <Modal
          onClose={() => setSslSite(null)}
          title={`SSL 证书一键申请: ${sslSite.primary_domain}`}
        >
          <div>
            {error && (
              <div style={{ backgroundColor: '#fef2f2', color: '#b91c1c', padding: '10px 14px', borderRadius: 6, marginBottom: 16, fontSize: 13 }}>
                {error}
              </div>
            )}

            {sslInfo?.enabled && (
              <div {...stylex.props(styles.sslCard)}>
                <div style={{ fontWeight: 650, color: '#16a34a', display: 'flex', alignItems: 'center', gap: 6 }}>
                  <Icon name="check" size={15} />
                  <span>当前站点已成功绑定有效证书</span>
                </div>
                <div style={{ marginTop: 6, color: '#475569' }}>
                  颁发机构: {sslInfo.issuer || '未知'} · 剩余有效期: {sslInfo.days_remaining} 天 (截止 {sslInfo.expiry})
                </div>
                <div style={{ marginTop: 4, color: '#64748b' }}>
                  保护域名: {sslInfo.domains.join(', ')}
                </div>
              </div>
            )}

            {/* 证书提供商切换 */}
            <div {...stylex.props(styles.tabsRow)}>
              <button
                type="button"
                {...stylex.props(styles.tabBtn, sslProvider === 'letsencrypt' && styles.tabBtnActive)}
                onClick={() => setSslProvider('letsencrypt')}
              >
                Let's Encrypt 一键申请
              </button>
              <button
                type="button"
                {...stylex.props(styles.tabBtn, sslProvider === 'self-signed' && styles.tabBtnActive)}
                onClick={() => setSslProvider('self-signed')}
              >
                自签名测试证书 (免域名验证)
              </button>
              <button
                type="button"
                {...stylex.props(styles.tabBtn, sslProvider === 'custom' && styles.tabBtnActive)}
                onClick={() => setSslProvider('custom')}
              >
                自定义证书上传
              </button>
            </div>

            {sslProvider === 'letsencrypt' && (
              <div>
                <p style={{ fontSize: 13, color: '#475569', lineHeight: 1.5, marginBottom: 12 }}>
                  通过内置 ACME HTTP-01 验证协议自动向 Let's Encrypt 申请受信任的免费 SSL 证书。
                  <strong>请确保您的域名已解析至当前主机 80 端口，且防火墙已放行 80/443。</strong>
                </p>
                <div {...stylex.props(styles.formGroup)}>
                  <label {...stylex.props(styles.formLabel)}>通知邮箱 (用于证书续签与到期提醒)</label>
                  <input
                    {...stylex.props(styles.formInput)}
                    placeholder="admin@example.com"
                    value={sslEmail}
                    onChange={e => setSslEmail(e.target.value)}
                  />
                </div>
              </div>
            )}

            {sslProvider === 'self-signed' && (
              <div>
                <p style={{ fontSize: 13, color: '#475569', lineHeight: 1.5, marginBottom: 12 }}>
                  立即为 <strong>{sslSite.primary_domain}</strong> 生成 365 天自签名 SSL 证书。适合本地开发、局域网测试或未完成公网 DNS 解析的环境。浏览器访问时提示证书不被信任属于正常现象。
                </p>
              </div>
            )}

            {sslProvider === 'custom' && (
              <div>
                <div {...stylex.props(styles.formGroup)}>
                  <label {...stylex.props(styles.formLabel)}>证书内容 (fullchain.pem / CRT)</label>
                  <textarea
                    {...stylex.props(styles.formInput)}
                    rows={4}
                    placeholder="-----BEGIN CERTIFICATE-----..."
                    value={sslCertPEM}
                    onChange={e => setSslCertPEM(e.target.value)}
                  />
                </div>
                <div {...stylex.props(styles.formGroup)}>
                  <label {...stylex.props(styles.formLabel)}>私钥内容 (privkey.pem / KEY)</label>
                  <textarea
                    {...stylex.props(styles.formInput)}
                    rows={4}
                    placeholder="-----BEGIN RSA PRIVATE KEY-----..."
                    value={sslKeyPEM}
                    onChange={e => setSslKeyPEM(e.target.value)}
                  />
                </div>
              </div>
            )}

            <div style={{ display: 'flex', alignItems: 'center', gap: 8, margin: '14px 0' }}>
              <input
                type="checkbox"
                id="sslForceCheck"
                checked={sslForceHTTPS}
                onChange={e => setSslForceHTTPS(e.target.checked)}
              />
              <label htmlFor="sslForceCheck" style={{ fontSize: 13, cursor: 'pointer', userSelect: 'none' }}>
                申请成功后立即开启 HTTP 强制跳转 HTTPS
              </label>
            </div>

            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 20 }}>
              <Button onClick={() => setSslSite(null)}>取消</Button>
              <Button variant="primary" onClick={handleApplySSL} disabled={busy}>
                {busy ? '正在申请与部署证书…' : '立即申请并部署'}
              </Button>
            </div>
          </div>
        </Modal>
      )}

      {/* 静态网页文件在线编辑弹窗 */}
      {!!fileSite && (
        <Modal
          onClose={() => setFileSite(null)}
          title={`在线编辑网页: ${fileSite.primary_domain}/index.html`}
        >
          <div>
            {error && (
              <div style={{ backgroundColor: '#fef2f2', color: '#b91c1c', padding: '10px 14px', borderRadius: 6, marginBottom: 16, fontSize: 13 }}>
                {error}
              </div>
            )}
            <p style={{ fontSize: 12.5, color: '#64748b', marginBottom: 10 }}>
              静态文件直接由内置 Nginx 服务。保存后即刻生效，无需重启 Nginx。
            </p>
            <textarea
              {...stylex.props(styles.codeArea)}
              value={fileContent}
              onChange={e => setFileContent(e.target.value)}
            />
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 16 }}>
              <Button onClick={() => setFileSite(null)}>取消</Button>
              <Button variant="primary" onClick={handleSaveFile} disabled={busy}>
                {busy ? '保存中…' : '保存网页'}
              </Button>
            </div>
          </div>
        </Modal>
      )}

      {/* Nginx 配置预览弹窗 */}
      {!!configSite && (
        <Modal
          onClose={() => setConfigSite(null)}
          title={`Nginx 虚拟主机配置: ${configSite.primary_domain}.conf`}
        >
          <div>
            <p style={{ fontSize: 12.5, color: '#64748b', marginBottom: 10 }}>
              这是由 CamoPanel 自动生成的 Nginx server 块完整配置：
            </p>
            <textarea
              {...stylex.props(styles.codeArea)}
              readOnly
              value={configContent}
            />
            <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 16 }}>
              <Button onClick={() => setConfigSite(null)}>关闭</Button>
            </div>
          </div>
        </Modal>
      )}

      {/* 删除确认弹窗 */}
      {!!deleteSite && (
        <Modal
          onClose={() => setDeleteSite(null)}
          title="确认删除网站"
        >
          <div>
            <p style={{ fontSize: 13.5, color: '#334155', lineHeight: 1.6 }}>
              确定要删除网站 <strong>{deleteSite.primary_domain}</strong> ({deleteSite.name}) 吗？
              将同时移除 Nginx 虚拟主机配置、域名绑定及专属静态文件目录。
            </p>
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 10, marginTop: 24 }}>
              <Button onClick={() => setDeleteSite(null)}>取消</Button>
              <Button variant="danger" onClick={handleDelete} disabled={busy}>
                {busy ? '删除中…' : '确认删除'}
              </Button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
}
