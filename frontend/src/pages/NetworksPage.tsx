import * as stylex from '@stylexjs/stylex';
import { resourceTableStyles as styles } from '../components/ui/resourceTableStyles';
import { type Snapshot } from '../lib/api';
import { Icon } from '../components/Icon';
import { Panel, PanelHeader } from '../components/ui/Panel';
import { Badge } from '../components/ui/Badge';
import { SearchInput } from '../components/ui/SearchInput';

export function NetworksPage({
  networks,
  total,
  search,
  onSearchChange,
  loading,
  lastUpdate,
}: {
  networks: Snapshot['networks'];
  total: number;
  search: string;
  onSearchChange: (v: string) => void;
  loading: boolean;
  lastUpdate: string;
}) {
  return (
    <Panel>
      <PanelHeader
        icon={<Icon name="network" size={17} />}
        title="网络环境"
        badge={<Badge>{total}</Badge>}
        extra={<span style={{ fontSize: 12, color: '#64748b' }}>Docker Engine</span>}
      />

      <div {...stylex.props(styles.toolbar)}>
        <SearchInput
          placeholder="搜索网络名称或驱动…"
          value={search}
          onChange={e => onSearchChange(e.target.value)}
        />
      </div>

      <div {...stylex.props(styles.tableScroll)} tabIndex={0} role="region" aria-label="资源列表，可横向滚动">
        <table {...stylex.props(styles.table)}>
          <thead>
            <tr>
              {['网络名称', '驱动', '作用域', '网络 ID'].map(label => (
                <th key={label} {...stylex.props(styles.th)} scope="col">
                  {label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {networks.map(n => (
              <tr {...stylex.props(styles.row)} key={n.Id}>
                <td {...stylex.props(styles.td)}>
                  <p {...stylex.props(styles.name)}>{n.Name}</p>
                </td>
                <td {...stylex.props(styles.td)}>{n.Driver}</td>
                <td {...stylex.props(styles.td)}>{n.Scope}</td>
                <td {...stylex.props(styles.td, styles.mono)}>{n.Id.slice(0, 12)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {networks.length === 0 && (
        <div {...stylex.props(styles.empty)}>
          <Icon name={loading ? 'refresh' : 'network'} size={28} />
          <p>{loading ? '正在读取网络数据…' : search ? '没有匹配的网络。' : '当前没有 Docker 网络。'}</p>
        </div>
      )}

      <div {...stylex.props(styles.footer)}>
        <span>显示 {networks.length} / {total} 项</span>
        <span>上次同步 {lastUpdate}</span>
      </div>
    </Panel>
  );
}
