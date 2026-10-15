import * as stylex from '@stylexjs/stylex';
import { Icon } from './Icon';

const styles = stylex.create({
  container: {
    padding: '20px',
    marginTop: 20,
    border: '1px solid var(--border-color, #e5e7eb)',
    borderRadius: 12,
    backgroundColor: '#ffffff',
    boxShadow: 'var(--shadow-card)',
  },
  header: {
    display: 'flex',
    alignItems: 'center',
    gap: 8,
  },
  title: {
    fontSize: 'calc(var(--font-size) + 1px)',
    fontWeight: 600,
    color: '#0f172a',
  },
  bar: {
    display: 'flex',
    height: 8,
    borderRadius: 9999,
    overflow: 'hidden',
    gap: 2,
    margin: '14px 0 12px',
    backgroundColor: '#f1f5f9',
  },
  stoppedSegment: {
    flex: 1,
    backgroundColor: '#cbd5e1',
    borderRadius: 9999,
  },
  legend: {
    display: 'flex',
    alignItems: 'center',
    gap: 16,
    flexWrap: 'wrap',
    fontSize: 12,
    color: '#475569',
  },
  legendItem: {
    display: 'flex',
    alignItems: 'center',
    gap: 7,
  },
  dotRunning: {
    width: 7,
    height: 7,
    backgroundColor: '#10b981',
    borderRadius: '50%',
    display: 'inline-block',
  },
});

export function LifecycleBar({ running, total }: { running: number; total: number }) {
  const stopped = total - running;
  const percentage = total > 0 ? (running / total) * 100 : 0;

  return (
    <section {...stylex.props(styles.container)} aria-label="运行状态分布">
      <div {...stylex.props(styles.header)}>
        <Icon name="grid" size={16} />
        <h2 {...stylex.props(styles.title)}>运行状态分布</h2>
      </div>

      <div
        {...stylex.props(styles.bar)}
        role="img"
        aria-label={`${running} 个运行中，${stopped} 个未运行`}
      >
        <div
          style={{
            width: `${percentage}%`,
            backgroundColor: '#10b981',
            borderRadius: 9999,
          }}
        />
        {stopped > 0 && <div {...stylex.props(styles.stoppedSegment)} />}
      </div>

      <div {...stylex.props(styles.legend)}>
        <span {...stylex.props(styles.legendItem)}>
          <span {...stylex.props(styles.dotRunning)} />
          <span>运行中 {running}</span>
        </span>
        <span {...stylex.props(styles.legendItem)}>
          <Icon name="stop" size={11} />
          <span>未运行 {stopped}</span>
        </span>
        <span style={{ marginLeft: 'auto', color: '#64748b' }}>共 {total} 个容器</span>
      </div>
    </section>
  );
}
