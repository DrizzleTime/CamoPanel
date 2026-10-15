import * as stylex from '@stylexjs/stylex';
import { Icon } from '../Icon';
import { Button } from '../ui/Button';
import { IconButton } from '../ui/IconButton';

const styles = stylex.create({
  topbar: {
    flexShrink: 0,
    minHeight: { default: 64, '@media (max-width: 600px)': 104 },
    backgroundColor: 'rgba(255, 255, 255, 0.92)',
    backdropFilter: 'blur(12px)',
    borderBottom: '1px solid var(--border-color, #e5e7eb)',
    padding: { default: '12px 28px', '@media (max-width: 600px)': '10px 16px' },
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    flexWrap: { default: 'nowrap', '@media (max-width: 600px)': 'wrap' },
    gap: { default: 14, '@media (max-width: 600px)': 8 },
    position: 'sticky',
    top: 0,
    zIndex: 10,
  },
  row: {
    display: 'flex',
    alignItems: 'center',
    gap: 8,
  },
  mobileMenu: {
    display: { default: 'none', '@media (max-width: 900px)': 'inline-grid' },
  },
  breadcrumb: {
    display: 'flex',
    alignItems: 'center',
    gap: 7,
    fontSize: 13,
    color: '#475569',
    fontWeight: 500,
  },
  breadcrumbCurrent: {
    color: '#0f172a',
    fontWeight: 600,
  },
  topRight: {
    display: 'flex',
    alignItems: 'center',
    gap: 8,
    width: { default: 'auto', '@media (max-width: 600px)': '100%' },
    justifyContent: 'flex-end',
  },
  connection: {
    display: { default: 'flex', '@media (max-width: 1100px)': 'none' },
    alignItems: 'center',
    gap: 7,
    fontSize: 12,
    fontWeight: 500,
    color: '#334155',
    padding: '4px 10px',
  },
  dot: {
    width: 7,
    height: 7,
    backgroundColor: '#10b981',
    borderRadius: '50%',
    display: 'inline-block',
    flexShrink: 0,
  },
  dotError: { backgroundColor: '#d97706' },
  dotOffline: { backgroundColor: '#94a3b8' },
  avatar: {
    width: 36,
    height: 36,
    borderRadius: 8,
    backgroundColor: { default: '#ffffff', ':hover': 'var(--accent-light)' },
    borderWidth: 1,
    borderStyle: 'solid',
    borderColor: { default: '#dce1e8', ':hover': 'var(--accent)' },
    color: '#334155',
    display: 'grid',
    placeItems: 'center',
    fontSize: 13,
    fontWeight: 650,
    cursor: 'pointer',
    transition: 'background-color .16s ease, border-color .16s ease',
    boxShadow: '0 1px 2px rgba(0, 0, 0, 0.04)',
  },
});

export function Topbar({
  title,
  isConnected,
  connectionLabel,
  hasError,
  loading,
  busy,
  onRefresh,
  onCreate,
  showCreate = true,
  onOpenSettings,
  onOpenMobileMenu,
}: {
  title: string;
  isConnected: boolean;
  connectionLabel?: string;
  hasError: boolean;
  loading: boolean;
  busy: boolean;
  onRefresh: () => void;
  onCreate: () => void;
  showCreate?: boolean;
  onOpenSettings: () => void;
  onOpenMobileMenu: () => void;
}) {
  return (
    <header {...stylex.props(styles.topbar)}>
      <div {...stylex.props(styles.row)}>
        <span {...stylex.props(styles.mobileMenu)}>
          <IconButton aria-label="打开导航" onClick={onOpenMobileMenu}>
            <Icon name="menu" size={17} />
          </IconButton>
        </span>
        <div {...stylex.props(styles.breadcrumb)}>
          <Icon name="server" size={15} />
          <span>本地工作空间</span>
          <Icon name="chevron" size={12} />
          <span {...stylex.props(styles.breadcrumbCurrent)}>{title}</span>
        </div>
      </div>

      <div {...stylex.props(styles.topRight)}>
        <span {...stylex.props(styles.connection)}>
          <span {...stylex.props(styles.dot, hasError ? styles.dotError : !isConnected && styles.dotOffline)} />
          {connectionLabel || (hasError ? '同步异常' : isConnected ? '15 秒自动同步' : '等待连接')}
        </span>
        <Button
          size="sm"
          disabled={loading || busy}
          onClick={onRefresh}
          title="立即刷新数据"
        >
          <Icon name="refresh" size={14} />
          <span>{loading ? '同步中…' : '刷新'}</span>
        </Button>
        {showCreate && (
          <Button
            variant="primary"
            size="sm"
            disabled={!isConnected || busy}
            onClick={onCreate}
          >
            <Icon name="plus" size={15} />
            <span>创建容器</span>
          </Button>
        )}
        <button
          {...stylex.props(styles.avatar)}
          aria-label="外观与设置"
          title="外观与设置"
          onClick={onOpenSettings}
        >
          <Icon name="palette" size={16} />
        </button>
      </div>
    </header>
  );
}
