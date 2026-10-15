import { useState } from 'react';
import * as stylex from '@stylexjs/stylex';
import { resourceTableStyles as styles } from '../components/ui/resourceTableStyles';
import { api, formatBytes, type Snapshot } from '../lib/api';
import { Icon } from '../components/Icon';
import { Panel, PanelHeader } from '../components/ui/Panel';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { SearchInput } from '../components/ui/SearchInput';

export function ImagesPage({
  images,
  total,
  search,
  onSearchChange,
  loading,
  lastUpdate,
  onChanged,
}: {
  images: Snapshot['images'];
  total: number;
  search: string;
  onSearchChange: (v: string) => void;
  loading: boolean;
  lastUpdate: string;
  onChanged?: () => void;
}) {
  const [notice, setNotice] = useState('');
  return (
    <Panel>
      <PanelHeader
        icon={<Icon name="layers" size={17} />}
        title="本地镜像"
        badge={<Badge>{total}</Badge>}
        extra={<span style={{ fontSize: 12, color: '#64748b' }}>Docker Engine</span>}
      />

      <div {...stylex.props(styles.toolbar)}>
        <SearchInput
          placeholder="搜索镜像名称或 ID…"
          value={search}
          onChange={e => onSearchChange(e.target.value)}
        />
        <Button
          size="sm"
          onClick={() => void api.pruneImages().then(res => {
            setNotice(`已清理 ${res.removed} 个悬空镜像`);
            onChanged?.();
          }).catch(err => setNotice(err instanceof Error ? err.message : '清理失败'))}
        >
          清理悬空镜像
        </Button>
      </div>
      {notice && <p style={{ fontSize: 12.5, color: '#475569', padding: '0 16px' }}>{notice}</p>}

      <div {...stylex.props(styles.tableScroll)} tabIndex={0} role="region" aria-label="资源列表，可横向滚动">
        <table {...stylex.props(styles.table)}>
          <thead>
            <tr>
              {['镜像名称', '镜像 ID', '大小', '创建时间', ''].map(label => (
                <th key={label} {...stylex.props(styles.th)} scope="col">
                  {label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {images.map(i => (
              <tr {...stylex.props(styles.row)} key={i.Id}>
                <td {...stylex.props(styles.td)}>
                  <p {...stylex.props(styles.name)}>{i.RepoTags?.join(', ') || '<none>'}</p>
                </td>
                <td {...stylex.props(styles.td, styles.mono)}>
                  {i.Id.replace('sha256:', '').slice(0, 12)}
                </td>
                <td {...stylex.props(styles.td)}>{formatBytes(i.Size)}</td>
                <td {...stylex.props(styles.td)}>
                  {new Date(i.Created * 1000).toLocaleDateString('zh-CN')}
                </td>
                <td {...stylex.props(styles.td)}>
                  <Button
                    size="sm"
                    variant="danger"
                    onClick={() => void api.deleteImage(i.Id).then(() => onChanged?.()).catch(err => setNotice(err instanceof Error ? err.message : '删除失败'))}
                  >
                    删除
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {images.length === 0 && (
        <div {...stylex.props(styles.empty)}>
          <Icon name={loading ? 'refresh' : 'layers'} size={28} />
          <p>{loading ? '正在读取镜像数据…' : search ? '没有匹配的镜像。' : '当前没有本地镜像。'}</p>
        </div>
      )}

      <div {...stylex.props(styles.footer)}>
        <span>显示 {images.length} / {total} 项</span>
        <span>上次同步 {lastUpdate}</span>
      </div>
    </Panel>
  );
}
