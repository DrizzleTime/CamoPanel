import * as stylex from '@stylexjs/stylex';
import { resourceTableStyles as styles } from '../components/ui/resourceTableStyles';
import { type Snapshot } from '../lib/api';
import { Icon } from '../components/Icon';
import { Panel, PanelHeader } from '../components/ui/Panel';
import { Badge } from '../components/ui/Badge';
import { SearchInput } from '../components/ui/SearchInput';

export function VolumesPage({
  volumes,
  total,
  search,
  onSearchChange,
  loading,
  lastUpdate,
}: {
  volumes: Snapshot['volumes'];
  total: number;
  search: string;
  onSearchChange: (v: string) => void;
  loading: boolean;
  lastUpdate: string;
}) {
  return (
    <Panel>
      <PanelHeader
        icon={<Icon name="database" size={17} />}
        title="数据持久卷"
        badge={<Badge>{total}</Badge>}
        extra={<span style={{ fontSize: 12, color: '#64748b' }}>Docker Engine</span>}
      />

      <div {...stylex.props(styles.toolbar)}>
        <SearchInput
          placeholder="搜索数据卷名称或挂载位置…"
          value={search}
          onChange={e => onSearchChange(e.target.value)}
        />
      </div>

      <div {...stylex.props(styles.tableScroll)} tabIndex={0} role="region" aria-label="资源列表，可横向滚动">
        <table {...stylex.props(styles.table)}>
          <thead>
            <tr>
              {['数据卷名称', '驱动', '挂载位置'].map(label => (
                <th key={label} {...stylex.props(styles.th)} scope="col">
                  {label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {volumes.map(v => (
              <tr {...stylex.props(styles.row)} key={v.Name}>
                <td {...stylex.props(styles.td)}>
                  <p {...stylex.props(styles.name)}>{v.Name}</p>
                </td>
                <td {...stylex.props(styles.td)}>{v.Driver}</td>
                <td {...stylex.props(styles.td, styles.mono)}>{v.Mountpoint}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {volumes.length === 0 && (
        <div {...stylex.props(styles.empty)}>
          <Icon name={loading ? 'refresh' : 'database'} size={28} />
          <p>{loading ? '正在读取数据卷…' : search ? '没有匹配的数据卷。' : '当前没有数据卷。'}</p>
        </div>
      )}

      <div {...stylex.props(styles.footer)}>
        <span>显示 {volumes.length} / {total} 项</span>
        <span>上次同步 {lastUpdate}</span>
      </div>
    </Panel>
  );
}
