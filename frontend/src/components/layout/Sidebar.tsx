import { useEffect, useRef } from 'react';
import * as stylex from '@stylexjs/stylex';
import { Icon, type IconName } from '../Icon';
import { Badge } from '../ui/Badge';
import { type Snapshot } from '../../lib/api';

const styles = stylex.create({
  sidebar: {
    width: 236,
    backgroundColor: 'var(--sidebar-bg, #fafbfc)',
    color: '#334155',
    display: 'flex',
    flexDirection: 'column',
    padding: '20px 14px 16px',
    overflowY: 'auto',
    visibility: { default: 'visible', '@media (max-width: 900px)': 'hidden' },
    position: 'fixed',
    inset: '0 auto 0 0',
    zIndex: 20,
    borderRight: '1px solid var(--border-color, #e5e7eb)',
    transform: { default: 'none', '@media (max-width: 900px)': 'translateX(-100%)' },
    transition: 'transform .22s ease, visibility .22s',
  },
  sidebarOpen: {
    transform: 'translateX(0)',
    visibility: 'visible',
  },
  mobileShade: {
    display: { default: 'none', '@media (max-width: 900px)': 'block' },
    position: 'fixed',
    inset: 0,
    backgroundColor: 'rgba(15, 23, 42, 0.45)',
    backdropFilter: 'blur(4px)',
    borderWidth: 0,
    zIndex: 19,
  },
  brand: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: '0 6px 24px',
    color: '#0f172a',
    fontWeight: 700,
    fontSize: 15,
    letterSpacing: '-0.3px',
  },
  brandLeft: {
    display: 'flex',
    alignItems: 'center',
    gap: 9,
  },
  logo: {
    backgroundColor: 'var(--accent, #d85c49)',
    color: '#ffffff',
    width: 30,
    height: 30,
    borderRadius: 9,
    display: 'grid',
    placeItems: 'center',
    fontSize: 14,
    fontWeight: 750,
    boxShadow: '0 2px 5px rgba(15, 23, 42, 0.12)',
  },
  searchBox: {
    display: 'flex',
    alignItems: 'center',
    gap: 7,
    padding: '9px 10px',
    borderRadius: 8,
    backgroundColor: '#ffffff',
    borderWidth: 1,
    borderStyle: 'solid',
    borderColor: { default: '#e2e8f0', ':hover': '#bbc4d1', ':focus-within': 'var(--accent)' },
    color: '#64748b',
    fontSize: 13,
    margin: '0 2px 20px',
    boxShadow: { default: 'none', ':focus-within': '0 0 0 3px var(--accent-light)' },
    transition: 'border-color .16s ease, box-shadow .16s ease',
    cursor: 'text',
  },
  searchInput: {
    borderWidth: 0,
    outline: 'none',
    backgroundColor: 'transparent',
    width: '100%',
    fontSize: 13,
    color: '#0f172a',
    minWidth: 0,
  },
  navLabel: {
    fontSize: 11,
    letterSpacing: '0.6px',
    padding: '8px 12px',
    marginTop: 12,
    marginBottom: 4,
    color: '#64748b',
    fontWeight: 600,
    textTransform: 'uppercase',
  },
  nav: {
    display: 'flex',
    flexDirection: 'column',
    gap: 3,
  },
  navButton: {
    display: 'flex',
    alignItems: 'center',
    gap: 10,
    padding: '10px 12px',
    minHeight: 42,
    borderWidth: 0,
    borderRadius: 7,
    backgroundColor: { default: 'transparent', ':hover:not(:disabled)': 'var(--surface-hover)' },
    color: '#334155',
    textAlign: 'left',
    fontSize: 'var(--font-size)',
    fontWeight: 500,
    transition: 'all .14s ease',
    cursor: 'pointer',
  },
  navActive: {
    backgroundColor: { default: 'var(--accent-light, #fef2f0)', ':hover': 'color-mix(in srgb, var(--accent-light) 85%, var(--accent) 15%)' },
    color: 'var(--accent-text, #b93826)',
    fontWeight: 600,
  },
  bottom: {
    marginTop: 'auto',
    paddingTop: 12,
    borderTop: '1px solid var(--border-color, #e5e7eb)',
    display: 'flex',
    flexDirection: 'column',
    gap: 4,
  },
  userCard: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: '8px 10px',
    borderRadius: 8,
    cursor: 'pointer',
    backgroundColor: { default: 'transparent', ':hover': 'var(--surface-hover)' },
    textAlign: 'left',
    width: '100%',
    transition: 'background-color .15s',
  },
  userMeta: {
    display: 'flex',
    alignItems: 'center',
    gap: 10,
    minWidth: 0,
  },
  userAvatar: {
    width: 32,
    height: 32,
    borderRadius: '50%',
    backgroundColor: 'var(--accent, #d85c49)',
    color: '#ffffff',
    display: 'grid',
    placeItems: 'center',
    fontSize: 12.5,
    fontWeight: 750,
    flexShrink: 0,
    boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
  },
  userName: {
    fontSize: 13,
    fontWeight: 650,
    color: '#0f172a',
    lineHeight: 1.5,
  },
  userSubtitle: {
    fontSize: 11,
    color: '#64748b',
    lineHeight: 1.25,
  },
  version: {
    fontSize: 11,
    color: '#64748b',
    padding: '6px 10px 0',
    display: 'flex',
    justifyContent: 'space-between',
  },
});

export type PageId = 'overview' | 'websites' | 'store' | 'certificates' | 'files' | 'databases' | 'backups' | 'schedules' | 'firewall' | 'assistant' | 'containers' | 'images' | 'networks' | 'volumes';

export interface PageMenuItem {
  id: PageId;
  label: string;
  icon: IconName;
}

export type DockerPageId = 'containers' | 'images' | 'networks' | 'volumes';

export interface DockerMenuItem {
  id: DockerPageId;
  label: string;
  icon: IconName;
}

export const WORKSPACE_PAGES: PageMenuItem[] = [
  { id: 'overview', label: '工作台概览', icon: 'grid' },
  { id: 'websites', label: '网站管理', icon: 'globe' },
  { id: 'store', label: '应用商店', icon: 'store' },
];

export const PLATFORM_PAGES: PageMenuItem[] = [
  { id: 'certificates', label: '证书', icon: 'lock' },
  { id: 'files', label: '文件', icon: 'folder' },
  { id: 'databases', label: '数据库', icon: 'database' },
  { id: 'backups', label: '备份', icon: 'download' },
  { id: 'schedules', label: '计划任务', icon: 'clock' },
  { id: 'firewall', label: '防火墙', icon: 'shield' },
  { id: 'assistant', label: 'AI 运维助手', icon: 'sparkle' },
];

export const DOCKER_PAGES: DockerMenuItem[] = [
  { id: 'containers', label: '容器管理', icon: 'box' },
  { id: 'images', label: '本地镜像', icon: 'layers' },
  { id: 'networks', label: '网络环境', icon: 'network' },
  { id: 'volumes', label: '数据持久卷', icon: 'database' },
];

export const PAGES: PageMenuItem[] = [...WORKSPACE_PAGES, ...PLATFORM_PAGES, ...DOCKER_PAGES];

export function Sidebar({
  page,
  actor,
  role,
  onLogout,
  logoutBusy,
  onNavigate,
  search,
  onSearchChange,
  data,
  siteCount,
  installedAppCount,
  isOpen,
  onClose,
  onOpenSettings,
}: {
  page: PageId;
  actor: string;
  role: string;
  onLogout: () => void;
  logoutBusy: boolean;
  onNavigate: (p: PageId) => void;
  search: string;
  onSearchChange: (val: string) => void;
  data: Snapshot | null;
  siteCount?: number;
  installedAppCount?: number;
  isOpen: boolean;
  onClose: () => void;
  onOpenSettings: () => void;
}) {
  const searchInputRef = useRef<HTMLInputElement>(null);
  const sidebarRef = useRef<HTMLElement>(null);

  useEffect(() => {
    if (!isOpen) return;
    const previousFocus = document.activeElement as HTMLElement | null;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    searchInputRef.current?.focus();
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        event.preventDefault();
        onClose();
      }
      if (event.key !== 'Tab') return;
      const focusable = sidebarRef.current?.querySelectorAll<HTMLElement>('input, button:not(:disabled)');
      if (!focusable?.length) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    }
    window.addEventListener('keydown', handleKeyDown);
    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener('keydown', handleKeyDown);
      previousFocus?.focus();
    };
  }, [isOpen, onClose]);

  return (
    <>
      {isOpen && (
        <button
          {...stylex.props(styles.mobileShade)}
          aria-label="关闭导航"
          onClick={onClose}
        />
      )}

      <aside ref={sidebarRef} {...stylex.props(styles.sidebar, isOpen && styles.sidebarOpen)} aria-label="主导航">
        {/* 顶部品牌 */}
        <div {...stylex.props(styles.brand)}>
          <div {...stylex.props(styles.brandLeft)}>
            <span {...stylex.props(styles.logo)}>C</span>
            <span>CamoPanel</span>
          </div>
          <Badge>v0.1.0</Badge>
        </div>

        {/* 侧栏快捷搜索框 */}
        <label {...stylex.props(styles.searchBox)}>
          <Icon name="search" size={15} />
          <input
            ref={searchInputRef}
            {...stylex.props(styles.searchInput)}
            placeholder="搜索资源..."
            type="search"
            aria-label="搜索当前页面资源"
            value={search}
            onChange={e => onSearchChange(e.target.value)}
          />
          <Badge variant="kbd">⌘ K</Badge>
        </label>

        {/* 导航分组：工作空间 */}
        <p {...stylex.props(styles.navLabel)}>工作空间</p>
        <nav {...stylex.props(styles.nav)} aria-label="工作空间">
          {WORKSPACE_PAGES.map(p => {
            let countBadge: React.ReactNode = null;
            if (p.id === 'websites' && siteCount !== undefined) {
              countBadge = <Badge variant="count" active={page === p.id}>{siteCount}</Badge>;
            } else if (p.id === 'store' && installedAppCount !== undefined) {
              countBadge = <Badge variant="count" active={page === p.id}>{installedAppCount}</Badge>;
            }
            return (
              <button
                key={p.id}
                {...stylex.props(styles.navButton, page === p.id && styles.navActive)}
                onClick={() => onNavigate(p.id)}
                aria-current={page === p.id ? 'page' : undefined}
              >
                <Icon name={p.icon} size={17} />
                <span>{p.label}</span>
                {countBadge && <span style={{ marginLeft: 'auto' }}>{countBadge}</span>}
              </button>
            );
          })}
        </nav>

        <p {...stylex.props(styles.navLabel)}>平台</p>
        <nav {...stylex.props(styles.nav)} aria-label="平台">
          {PLATFORM_PAGES.map(p => (
            <button
              key={p.id}
              {...stylex.props(styles.navButton, page === p.id && styles.navActive)}
              onClick={() => onNavigate(p.id)}
              aria-current={page === p.id ? 'page' : undefined}
            >
              <Icon name={p.icon} size={17} />
              <span>{p.label}</span>
            </button>
          ))}
        </nav>

        {/* 导航分组：DOCKER 资源 */}
        <p {...stylex.props(styles.navLabel)}>DOCKER 资源</p>
        <nav {...stylex.props(styles.nav)} aria-label="Docker 资源">
          {DOCKER_PAGES.map(p => (
            <button
              key={p.id}
              {...stylex.props(styles.navButton, page === p.id && styles.navActive)}
              onClick={() => onNavigate(p.id)}
              aria-current={page === p.id ? 'page' : undefined}
            >
              <Icon name={p.icon} size={17} />
              <span>{p.label}</span>
              <span style={{ marginLeft: 'auto' }}>
                <Badge variant="count" active={page === p.id}>
                  {data ? data[p.id].length : '—'}
                </Badge>
              </span>
            </button>
          ))}
        </nav>

        {/* 本地引擎与设置入口 */}
        <div {...stylex.props(styles.bottom)}>
          <button
            type="button"
            {...stylex.props(styles.userCard)}
            onClick={onOpenSettings}
            title="点击打开偏好设置与连接"
          >
            <div {...stylex.props(styles.userMeta)}>
              <span {...stylex.props(styles.userAvatar)}>{Array.from(actor)[0]?.toUpperCase() || 'C'}</span>
              <div>
                <p {...stylex.props(styles.userName)}>{actor}</p>
                <p {...stylex.props(styles.userSubtitle)}>{role === 'viewer' ? '只读账号' : '管理员'} · 工作空间</p>
              </div>
            </div>
            <Icon name="settings" size={16} />
          </button>
          <button type="button" {...stylex.props(styles.navButton)} disabled={logoutBusy} onClick={onLogout}>
            <Icon name="logout" size={16} /><span>{logoutBusy ? '正在退出…' : '退出登录'}</span>
          </button>
          <div {...stylex.props(styles.version)}>
            <span>CamoPanel Web</span>
            <span>v0.1.0</span>
          </div>
        </div>
      </aside>
    </>
  );
}
