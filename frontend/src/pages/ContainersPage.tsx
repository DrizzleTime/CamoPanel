import * as stylex from '@stylexjs/stylex';
import { resourceTableStyles as tableStyles } from '../components/ui/resourceTableStyles';
import { containerName, type Container } from '../lib/api';
import { Icon } from '../components/Icon';
import { Panel, PanelHeader } from '../components/ui/Panel';
import { Badge } from '../components/ui/Badge';
import { SearchInput } from '../components/ui/SearchInput';
import { SegmentedControl } from '../components/ui/SegmentedControl';
import { StatusBadge } from '../components/ui/StatusBadge';
import { IconButton } from '../components/ui/IconButton';

const styles = stylex.create({
  nameCell: {
    display: 'flex',
    alignItems: 'center',
    gap: 12,
  },
  containerIcon: {
    width: 36,
    height: 36,
    borderRadius: 8,
    display: 'grid',
    placeItems: 'center',
    flexShrink: 0,
    border: '1px solid #e2e8f0',
  },

  actionRow: {
    display: 'flex',
    alignItems: 'center',
    gap: 6,
  },
});

export function ContainersPage({
  containers,
  total,
  running,
  stopped,
  search,
  onSearchChange,
  filter,
  onFilterChange,
  busy,
  loading,
  lastUpdate,
  onAction,
  onOpenLogs,
  onOpenRemove,
}: {
  containers: Container[];
  total: number;
  running: number;
  stopped: number;
  search: string;
  onSearchChange: (v: string) => void;
  filter: string;
  onFilterChange: (v: string) => void;
  busy: string;
  loading: boolean;
  lastUpdate: string;
  onAction: (c: Container, action: string) => void;
  onOpenLogs: (c: Container) => void;
  onOpenRemove: (c: Container) => void;
}) {
  return (
    <Panel>
      <PanelHeader
        icon={<Icon name="box" size={17} />}
        title="容器列表"
        badge={<Badge>{total}</Badge>}
        extra={<span style={{ fontSize: 12, color: '#64748b' }}>Docker Engine</span>}
      />

      {/* 工具栏：搜索框与状态胶囊段选择器 */}
      <div {...stylex.props(tableStyles.toolbar)}>
        <SearchInput
          placeholder="搜索容器名称、镜像或 ID…"
          value={search}
          onChange={e => onSearchChange(e.target.value)}
        />

        <SegmentedControl
          value={filter}
          onChange={onFilterChange}
          options={[
            { value: 'all', label: `全部 (${total})` },
            { value: 'running', label: `运行中 (${running})` },
            { value: 'stopped', label: `未运行 (${stopped})` },
          ]}
        />
      </div>

      {/* 数据表格 */}
      <div {...stylex.props(tableStyles.tableScroll)} tabIndex={0} role="region" aria-label="容器列表，可横向滚动">
        <table {...stylex.props(tableStyles.table)}>
          <thead>
            <tr>
              {['容器名称', '状态', '镜像', '端口', '操作'].map(label => (
                <th key={label} {...stylex.props(tableStyles.th)} scope="col">
                  {label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {containers.map(c => {
              const isRun = c.State === 'running';
              const isBusy = !!busy;
              return (
                <tr key={c.Id} {...stylex.props(tableStyles.row)} aria-busy={busy === c.Id}>
                  <td {...stylex.props(tableStyles.td)}>
                    <div {...stylex.props(styles.nameCell)}>
                      <span
                        {...stylex.props(styles.containerIcon)}
                        style={{
                          backgroundColor: isRun ? '#ecfdf5' : '#f1f5f9',
                          color: isRun ? '#059669' : '#64748b',
                          borderColor: isRun ? '#a7f3d0' : '#e2e8f0',
                        }}
                      >
                        <Icon name="box" size={16} />
                      </span>
                      <div>
                        <p {...stylex.props(tableStyles.name)}>{containerName(c)}</p>
                        <span {...stylex.props(tableStyles.mono)}>{c.Id.slice(0, 12)}</span>
                      </div>
                    </div>
                  </td>
                  <td {...stylex.props(tableStyles.td)}>
                    <StatusBadge state={c.State} statusText={c.Status} />
                  </td>
                  <td
                    {...stylex.props(tableStyles.td)}
                    title={c.Image}
                    style={{ maxWidth: 220, overflow: 'hidden', textOverflow: 'ellipsis' }}
                  >
                    {c.Image}
                  </td>
                  <td {...stylex.props(tableStyles.td)}>
                    {c.Ports?.length ? (
                      c.Ports.map((p, index) => (
                        <div key={index} {...stylex.props(tableStyles.mono)}>
                          {p.PublicPort ? `${p.IP || '0.0.0.0'}:${p.PublicPort} → ` : ''}
                          {p.PrivatePort}/{p.Type}
                        </div>
                      ))
                    ) : (
                      <span style={{ color: '#94a3b8' }}>—</span>
                    )}
                  </td>
                  <td {...stylex.props(tableStyles.td)}>
                    <div {...stylex.props(styles.actionRow)}>
                      {(isRun || c.State === 'exited' || c.State === 'created') && (
                        <IconButton
                          variant={isRun ? 'stop' : 'start'}
                          disabled={isBusy}
                          title={isRun ? '停止容器' : '启动容器'}
                          aria-label={`${isRun ? '停止' : '启动'} ${containerName(c)}`}
                          onClick={() => onAction(c, isRun ? 'stop' : 'start')}
                        >
                          <Icon name={isRun ? 'stop' : 'play'} size={15} />
                        </IconButton>
                      )}
                      <IconButton
                        variant="restart"
                        disabled={isBusy || !isRun}
                        title="重启容器"
                        aria-label={`重启 ${containerName(c)}`}
                        onClick={() => onAction(c, 'restart')}
                      >
                        <Icon name="refresh" size={15} />
                      </IconButton>
                      <IconButton
                        variant="default"
                        disabled={isBusy || (c.State !== 'running' && c.State !== 'paused')}
                        title={c.State === 'paused' ? '恢复容器' : '暂停容器'}
                        aria-label={`${c.State === 'paused' ? '恢复' : '暂停'} ${containerName(c)}`}
                        onClick={() => onAction(c, c.State === 'paused' ? 'unpause' : 'pause')}
                      >
                        <Icon name={c.State === 'paused' ? 'play' : 'clock'} size={15} />
                      </IconButton>
                      <IconButton
                        variant="default"
                        disabled={isBusy}
                        title="查看日志"
                        aria-label={`查看 ${containerName(c)} 日志`}
                        onClick={() => onOpenLogs(c)}
                      >
                        <Icon name="terminal" size={15} />
                      </IconButton>
                      <IconButton
                        variant="remove"
                        disabled={
                          isBusy ||
                          isRun ||
                          c.State === 'paused' ||
                          c.State === 'restarting'
                        }
                        title="删除停止的容器"
                        aria-label={`删除 ${containerName(c)}`}
                        onClick={() => onOpenRemove(c)}
                      >
                        <Icon name="trash" size={15} />
                      </IconButton>
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {containers.length === 0 && (
        <div {...stylex.props(tableStyles.empty)}>
          <Icon name={loading ? 'refresh' : 'box'} size={28} />
          <p>
            {loading
              ? '正在读取 Docker 容器列表…'
              : search || filter !== 'all'
              ? '没有符合筛选条件的容器。'
              : '当前没有容器。'}
          </p>
        </div>
      )}

      <div {...stylex.props(tableStyles.footer)}>
        <span>显示 {containers.length} / {total} 项</span>
        <span>{busy ? '正在执行操作…' : `上次同步 ${lastUpdate}`}</span>
      </div>
    </Panel>
  );
}
