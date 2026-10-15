import { useEffect, useState } from 'react';
import * as stylex from '@stylexjs/stylex';
import { api } from '../lib/api';
import { Icon } from './Icon';
import { Button } from './ui/Button';

const styles = stylex.create({
  header: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 14,
  },
  desc: {
    fontSize: 12.5,
    color: '#64748b',
  },
  error: {
    backgroundColor: '#fef2f2',
    color: '#b91c1c',
    padding: '11px 15px',
    borderRadius: 8,
    border: '1px solid #fee2e2',
    fontSize: 12.5,
    marginBottom: 12,
  },
  logs: {
    backgroundColor: '#0f172a',
    borderRadius: 9,
    color: '#e2e8f0',
    padding: 16,
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
    fontSize: 12,
    lineHeight: 1.85,
    whiteSpace: 'pre-wrap',
    overflowWrap: 'anywhere',
    maxHeight: '52vh',
    overflowY: 'auto',
    border: '1px solid #1e293b',
  },
});

export function ContainerLogs({ id }: { id: string }) {
  const [logs, setLogs] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [revision, setRevision] = useState(0);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    setError('');
    api
      .logs(id)
      .then(result => {
        if (alive) setLogs(result.logs);
      })
      .catch(err => {
        if (alive) setError(err.message);
      })
      .finally(() => {
        if (alive) setLoading(false);
      });
    return () => {
      alive = false;
    };
  }, [id, revision]);

  return (
    <>
      <div {...stylex.props(styles.header)}>
        <p {...stylex.props(styles.desc)}>最近 200 行 · stdout / stderr</p>
        <Button size="sm" disabled={loading} onClick={() => setRevision(n => n + 1)}>
          <Icon name="refresh" size={14} />
          <span>刷新日志</span>
        </Button>
      </div>
      {error && <p {...stylex.props(styles.error)} role="alert">{error}</p>}
      <pre {...stylex.props(styles.logs)} aria-live="polite">
        {loading ? '正在读取日志…' : logs || '暂无日志输出。'}
      </pre>
    </>
  );
}
