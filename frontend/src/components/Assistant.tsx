import { useState } from 'react';
import * as stylex from '@stylexjs/stylex';
import { containerName, formatBytes, type Snapshot } from '../lib/api';
import { Icon } from './Icon';
import { Badge } from './ui/Badge';

const styles = stylex.create({
  container: {
    padding: '20px',
    backgroundColor: '#ffffff',
    border: '1px solid var(--border-color, #e5e7eb)',
    borderRadius: 12,
    boxShadow: 'var(--shadow-card)',
  },
  head: {
    display: 'flex',
    alignItems: 'center',
    gap: 8,
    flexWrap: 'wrap',
    color: 'var(--accent, #d85c49)',
    marginBottom: 9,
  },
  title: {
    fontSize: 'calc(var(--font-size) + 1px)',
    fontWeight: 600,
    color: '#0f172a',
  },
  desc: {
    fontSize: 'calc(var(--font-size) - 1px)',
    lineHeight: 'var(--line-height)',
    color: '#475569',
    marginBottom: 12,
  },
  insight: {
    backgroundColor: '#f8fafc',
    padding: '11px 13px',
    borderRadius: 7,
    border: '1px solid #e2e8f0',
    fontSize: 12.5,
    lineHeight: 1.6,
    color: '#1e293b',
    marginBottom: 10,
    display: 'flex',
    gap: 8,
  },
  questionBtn: {
    width: '100%',
    textAlign: 'left',
    marginTop: 7,
    borderWidth: 1,
    borderStyle: 'solid',
    borderColor: { default: '#e2e8f0', ':hover': 'var(--accent)' },
    borderRadius: 7,
    padding: '10px 12px',
    backgroundColor: { default: '#ffffff', ':hover': 'var(--accent-light)' },
    fontSize: 'calc(var(--font-size) - 1px)',
    color: { default: '#334155', ':hover': 'var(--accent-text)' },
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    cursor: 'pointer',
    transition: 'all .14s',
    fontWeight: 500,
  },
  questionActive: {
    backgroundColor: { default: 'var(--accent-light)', ':hover': 'var(--accent-light)' },
    color: 'var(--accent-text)',
    borderColor: 'var(--accent)',
  },
  answer: {
    whiteSpace: 'pre-wrap',
    overflowWrap: 'anywhere',
    borderTop: '1px solid #f1f5f9',
    paddingTop: 12,
    marginTop: 12,
    fontSize: 12.5,
    color: '#0f172a',
    lineHeight: 1.7,
  },
  footer: {
    fontSize: 11,
    color: '#64748b',
    display: 'flex',
    gap: 5,
    alignItems: 'center',
    marginTop: 14,
  },
});

export function Assistant({ data }: { data: Snapshot | null }) {
  const [question, setQuestion] = useState<'stopped' | 'host' | null>(null);
  const stopped = data?.containers.filter(c => c.State !== 'running') || [];
  const answer = !data
    ? '连接后即可读取运行状态。'
    : question === 'stopped'
    ? stopped.length
      ? `当前有 ${stopped.length} 个非运行容器：\n${stopped
          .map(c => `• ${containerName(c)}：${c.Status}`)
          .join('\n')}\n请结合业务用途确认是否需要启动。`
      : '当前所有容器均处于运行状态。'
    : `主机：${data.engine.Name}\nDocker：${data.engine.ServerVersion}\nCPU：${data.engine.NCPU} 核\n内存：${formatBytes(
        data.engine.MemTotal
      )}\n${data.mode === 'demo' ? '这些是演示环境数据。' : '信息来自当前 Docker 引擎。'}`;

  return (
    <section {...stylex.props(styles.container)} aria-label="运维助手">
      <div {...stylex.props(styles.head)}>
        <Icon name="sparkle" size={18} />
        <h2 {...stylex.props(styles.title)}>运维诊断助手</h2>
        <span style={{ marginLeft: 'auto' }}>
          <Badge>基础诊断</Badge>
        </span>
      </div>

      <p {...stylex.props(styles.desc)}>从当前运行状态出发，让每一次运维决策都有依据。</p>

      <div {...stylex.props(styles.insight)}>
        <Icon name={stopped.length ? 'info' : 'check'} size={16} />
        <p>
          {!data
            ? '等待 Docker 连接，获取环境摘要。'
            : stopped.length
            ? `${stopped.length} 个容器未运行。建议确认它们是否为已完成的任务。`
            : '所有容器均处于运行状态，可查看日志了解应用健康情况。'}
        </p>
      </div>

      <button
        type="button"
        {...stylex.props(styles.questionBtn, question === 'stopped' && styles.questionActive)}
        onClick={() => setQuestion('stopped')}
        aria-pressed={question === 'stopped'}
        aria-controls="assistant-answer"
      >
        <span>哪些容器没有运行？</span>
        <Icon name="arrow" size={14} />
      </button>

      <button
        type="button"
        {...stylex.props(styles.questionBtn, question === 'host' && styles.questionActive)}
        onClick={() => setQuestion('host')}
        aria-pressed={question === 'host'}
        aria-controls="assistant-answer"
      >
        <span>查看当前环境摘要</span>
        <Icon name="arrow" size={14} />
      </button>

      <div id="assistant-answer" aria-live="polite">
        {question && (
          <p {...stylex.props(styles.answer)}>
            {answer}
          </p>
        )}
      </div>

      <p {...stylex.props(styles.footer)}>
        <Icon name="info" size={12} />
        <span>这是当前快照。要让模型回答，请在 AI 运维助手里填写供应商。</span>
      </p>
    </section>
  );
}
