import * as stylex from '@stylexjs/stylex';
import { type HostOverview, type HostProcess } from '../lib/api';
import { Icon } from './Icon';
import { Badge } from './ui/Badge';
import { Panel, PanelHeader } from './ui/Panel';
import { StatCard } from './ui/StatCard';
import { resourceTableStyles as table } from './ui/resourceTableStyles';

const styles = stylex.create({
  wrap: {
    marginBottom: 24,
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: {
      default: 'repeat(4, minmax(0, 1fr))',
      '@media (max-width: 1024px)': 'repeat(2, minmax(0, 1fr))',
    },
    gap: { default: 16, '@media (max-width: 600px)': 12 },
    marginBottom: 16,
  },
  split: {
    display: 'grid',
    gridTemplateColumns: { default: '1.3fr 1fr', '@media (max-width: 800px)': '1fr' },
    gap: 16,
    marginBottom: 16,
  },
  body: {
    padding: '8px 20px 16px',
  },
  row: {
    display: 'grid',
    gridTemplateColumns: 'minmax(0, 1.2fr) minmax(72px, 0.8fr) auto',
    gap: 12,
    alignItems: 'center',
    padding: '10px 0',
    borderBottom: '1px solid var(--border-subtle, #eef2f6)',
    fontSize: 'calc(var(--font-size) - 1px)',
  },
  name: {
    minWidth: 0,
    color: '#0f172a',
    fontWeight: 600,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  device: {
    color: '#64748b',
    fontWeight: 450,
    marginLeft: 6,
  },
  meta: {
    color: '#475569',
    textAlign: 'right',
    fontVariantNumeric: 'tabular-nums',
    whiteSpace: 'nowrap',
  },
  track: {
    display: 'block',
    height: 6,
    borderRadius: 99,
    backgroundColor: '#e2e8f0',
    overflow: 'hidden',
  },
  fill: {
    display: 'block',
    height: '100%',
    backgroundColor: 'var(--accent, #d85c49)',
    borderRadius: 99,
  },
  chart: {
    padding: '14px 20px 16px',
  },
  chartLabel: {
    display: 'flex',
    justifyContent: 'space-between',
    color: '#64748b',
    fontSize: 12,
    marginBottom: 8,
  },
  empty: {
    color: '#64748b',
    padding: '16px 0',
    fontSize: 'calc(var(--font-size) - 1px)',
  },
  error: {
    marginBottom: 12,
    color: '#b91c1c',
    backgroundColor: '#fef2f2',
    border: '1px solid #fee2e2',
    borderRadius: 8,
    padding: '10px 12px',
    fontSize: 13,
  },
});

function formatQuantity(n: number) {
  if (!Number.isFinite(n) || n < 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let value = n;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  const digits = unit === 0 ? 0 : 1;
  return `${value.toFixed(digits)} ${units[unit]}`;
}

function formatRate(n: number) {
  return `${formatQuantity(n)}/s`;
}

function formatUptime(seconds: number) {
  const total = Math.max(0, Math.floor(seconds));
  const days = Math.floor(total / 86400);
  const hours = Math.floor((total % 86400) / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  if (days > 0) return `${days} 天 ${hours} 小时`;
  if (hours > 0) return `${hours} 小时 ${minutes} 分`;
  return `${minutes} 分`;
}

function Sparkline({ values }: { values: number[] }) {
  if (values.length < 2) return null;
  const width = 640;
  const height = 72;
  const min = Math.min(...values);
  const max = Math.max(...values);
  const span = Math.max(max - min, 1);
  const points = values
    .map((value, index) => {
      const x = (index / (values.length - 1)) * width;
      const y = height - ((value - min) / span) * (height - 8) - 4;
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(' ');
  return (
    <svg viewBox={`0 0 ${width} ${height}`} width="100%" height="72" role="img" aria-label="CPU 与内存曲线">
      <polyline points={points} fill="none" stroke="#d85c49" strokeWidth="2.5" strokeLinejoin="round" strokeLinecap="round" />
    </svg>
  );
}

export function HostPanel({
  overview,
  processes,
  error,
  loading,
}: {
  overview: HostOverview | null;
  processes: HostProcess[];
  error: string;
  loading: boolean;
}) {
  const cpu = overview ? `${overview.cpu.percent.toFixed(1)}%` : '—';
  const memory = overview ? `${overview.memory.percent.toFixed(1)}%` : '—';
  const swap = overview ? `${overview.swap.percent.toFixed(1)}%` : '—';
  const load = overview ? overview.load.load1.toFixed(2) : '—';
  return (
    <section {...stylex.props(styles.wrap)} aria-label="主机状态">
      {error && <div {...stylex.props(styles.error)} role="alert">{error}{overview ? ' · 仍显示上次采样。' : ''}</div>}
      <div {...stylex.props(styles.grid)}>
        <StatCard
          title="CPU"
          value={cpu}
          icon="cpu"
          foot={overview ? `${overview.cpu.cores} 核` : loading ? '正在采样' : '等待采样'}
          iconColor="#d85c49"
          iconBg="#fef2f0"
        />
        <StatCard
          title="内存"
          value={memory}
          icon="monitor"
          foot={overview ? `${formatQuantity(overview.memory.used)} / ${formatQuantity(overview.memory.total)}` : '等待采样'}
          iconColor="#4f46e5"
          iconBg="#eef2ff"
        />
        <StatCard
          title="交换区"
          value={swap}
          icon="database"
          foot={overview && overview.swap.total > 0 ? `${formatQuantity(overview.swap.used)} / ${formatQuantity(overview.swap.total)}` : '未启用'}
          iconColor="#d97706"
          iconBg="#fffbeb"
        />
        <StatCard
          title="负载"
          value={load}
          icon="activity"
          foot={overview ? `5 分钟 ${overview.load.load5.toFixed(2)} · 15 分钟 ${overview.load.load15.toFixed(2)}` : '等待采样'}
          iconColor="#059669"
          iconBg="#ecfdf5"
        />
      </div>

      <div {...stylex.props(styles.split)}>
        <Panel>
          <PanelHeader
            icon={<Icon name="monitor" size={17} />}
            title="近一小时"
            badge={<Badge>本机</Badge>}
            extra={<span style={{ color: '#64748b', fontSize: 12 }}>{overview ? `已运行 ${formatUptime(overview.uptime_seconds)}` : ''}</span>}
          />
          <div {...stylex.props(styles.chart)}>
            <div {...stylex.props(styles.chartLabel)}>
              <span>CPU 使用率</span>
              <span>{overview ? overview.cpu.percent.toFixed(1) : '—'}%</span>
            </div>
            {overview && overview.history.length > 1 ? (
              <Sparkline values={overview.history.map(point => point.cpu_percent)} />
            ) : (
              <p {...stylex.props(styles.empty)}>采样积累中，刷新后会画出曲线。</p>
            )}
          </div>
        </Panel>
        <Panel>
          <PanelHeader icon={<Icon name="server" size={17} />} title="磁盘与网络" />
          <div {...stylex.props(styles.body)}>
            {(overview?.disks.length ? overview.disks : []).map(disk => (
              <div key={disk.mount} {...stylex.props(styles.row)}>
                <span {...stylex.props(styles.name)}>
                  {disk.mount}
                  <span {...stylex.props(styles.device)}>{disk.device}</span>
                </span>
                <span {...stylex.props(styles.track)} aria-hidden="true">
                  <span {...stylex.props(styles.fill)} style={{ width: `${Math.max(0, Math.min(100, disk.percent))}%` }} />
                </span>
                <span {...stylex.props(styles.meta)}>{disk.percent.toFixed(0)}% · {formatQuantity(disk.used)}</span>
              </div>
            ))}
            {!overview?.disks.length && <p {...stylex.props(styles.empty)}>没有可读的磁盘。</p>}
            {(overview?.networks || []).map(net => (
              <div key={net.name} {...stylex.props(styles.row)}>
                <span {...stylex.props(styles.name)}>{net.name}</span>
                <span {...stylex.props(styles.meta)}>↓ {formatRate(net.rx_per_sec)}</span>
                <span {...stylex.props(styles.meta)}>↑ {formatRate(net.tx_per_sec)}</span>
              </div>
            ))}
            {overview && overview.networks.length === 0 && <p {...stylex.props(styles.empty)}>没有可显示的网卡。</p>}
          </div>
        </Panel>
      </div>

      <Panel>
        <PanelHeader icon={<Icon name="cpu" size={17} />} title="占用最高的进程" badge={<Badge variant="count">{processes.length}</Badge>} />
        <div {...stylex.props(table.tableScroll)} tabIndex={0} aria-label="占用最高的进程">
          <table {...stylex.props(table.table)}>
            <thead>
              <tr>
                <th {...stylex.props(table.th)}>进程</th>
                <th {...stylex.props(table.th)}>PID</th>
                <th {...stylex.props(table.th)}>CPU</th>
                <th {...stylex.props(table.th)}>内存</th>
                <th {...stylex.props(table.th)}>常驻内存</th>
              </tr>
            </thead>
            <tbody>
              {processes.map(proc => (
                <tr key={proc.pid} {...stylex.props(table.row)}>
                  <td {...stylex.props(table.td)}>{proc.name}</td>
                  <td {...stylex.props(table.td)}>{proc.pid}</td>
                  <td {...stylex.props(table.td)}>{proc.cpu_percent.toFixed(1)}%</td>
                  <td {...stylex.props(table.td)}>{proc.mem_percent.toFixed(1)}%</td>
                  <td {...stylex.props(table.td)}>{formatQuantity(proc.rss)}</td>
                </tr>
              ))}
              {!processes.length && (
                <tr>
                  <td {...stylex.props(table.td)} colSpan={5}>{loading ? '正在采样' : '暂无进程数据'}</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </Panel>
    </section>
  );
}
