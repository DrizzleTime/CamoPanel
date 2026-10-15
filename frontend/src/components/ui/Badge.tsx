import { type ReactNode } from 'react';
import * as stylex from '@stylexjs/stylex';

const styles = stylex.create({
  base: {
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    fontWeight: 600,
    lineHeight: 1.2,
    fontVariantNumeric: 'tabular-nums',
    whiteSpace: 'nowrap',
    flexShrink: 0,
  },
  pill: {
    fontSize: 11,
    borderRadius: 5,
    backgroundColor: '#f1f5f9',
    color: '#475569',
    padding: '2px 7px',
  },
  count: {
    fontSize: 11.5,
    backgroundColor: '#f1f5f9',
    color: '#475569',
    padding: '2px 7px',
    borderRadius: 9999,
  },
  countActive: {
    backgroundColor: 'var(--accent-light, #fef2f0)',
    color: 'var(--accent-text, #b93826)',
  },
  kbd: {
    fontSize: 11,
    fontFamily: 'ui-monospace, SFMono-Regular, Consolas, monospace',
    padding: '2px 5px',
    borderRadius: 4,
    backgroundColor: '#f1f5f9',
    border: '1px solid #cbd5e1',
    color: '#475569',
    fontWeight: 500,
  },
  ok: {
    fontSize: 11,
    borderRadius: 5,
    backgroundColor: '#ecfdf5',
    color: '#065f46',
    border: '1px solid #a7f3d0',
    padding: '2px 7px',
  },
  outline: {
    fontSize: 11,
    borderRadius: 5,
    backgroundColor: 'transparent',
    color: '#475569',
    border: '1px solid #cbd5e1',
    padding: '2px 7px',
  },
});

export function Badge({
  variant = 'pill',
  active = false,
  children,
}: {
  variant?: 'pill' | 'count' | 'kbd' | 'ok' | 'outline';
  active?: boolean;
  children: ReactNode;
}) {
  return (
    <span
      {...stylex.props(
        styles.base,
        styles[variant],
        variant === 'count' && active && styles.countActive
      )}
    >
      {children}
    </span>
  );
}
