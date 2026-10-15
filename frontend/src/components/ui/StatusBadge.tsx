import * as stylex from '@stylexjs/stylex';

const styles = stylex.create({
  badge: {
    display: 'inline-flex',
    alignItems: 'center',
    gap: 6,
    borderRadius: 9999,
    padding: '3px 9px',
    fontSize: 11.5,
    fontWeight: 600,
    color: '#475569',
    backgroundColor: '#f1f5f9',
    border: '1px solid #e2e8f0',
    whiteSpace: 'nowrap',
  },
  running: {
    color: '#065f46',
    backgroundColor: '#ecfdf5',
    borderColor: '#a7f3d0',
  },
  dot: {
    width: 6,
    height: 6,
    borderRadius: '50%',
    backgroundColor: 'currentColor',
    flexShrink: 0,
  },
});

const stateLabel: Record<string, string> = {
  running: '运行中',
  exited: '已停止',
  created: '已创建',
  paused: '已暂停',
  restarting: '重启中',
  dead: '已失效',
  removing: '删除中',
};

export function StatusBadge({ state, statusText }: { state: string; statusText?: string }) {
  const isRunning = state === 'running';
  return (
    <span {...stylex.props(styles.badge, isRunning && styles.running)} title={statusText || state}>
      <span {...stylex.props(styles.dot)} />
      <span>{stateLabel[state] || state}</span>
    </span>
  );
}
