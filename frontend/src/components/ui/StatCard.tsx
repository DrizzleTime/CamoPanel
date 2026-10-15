import { type ReactNode } from 'react';
import * as stylex from '@stylexjs/stylex';
import { Icon, type IconName } from '../Icon';

const styles = stylex.create({
  card: {
    backgroundColor: '#ffffff',
    border: '1px solid var(--border-color, #e5e7eb)',
    borderRadius: 12,
    padding: 'var(--card-padding)',
    minWidth: 0,
    display: 'flex',
    flexDirection: 'column',
    gap: 14,
    boxShadow: 'var(--shadow-card)',
  },
  head: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    color: '#475569',
    gap: 8,
    fontSize: 'calc(var(--font-size) - 1px)',
    fontWeight: 500,
  },
  iconWrapper: {
    width: 34,
    height: 34,
    borderRadius: 8,
    display: 'grid',
    placeItems: 'center',
    flexShrink: 0,
  },
  number: {
    fontSize: 32,
    fontWeight: 700,
    color: '#0f172a',
    letterSpacing: '-0.8px',
    fontVariantNumeric: 'tabular-nums',
    lineHeight: 1,
  },
  foot: {
    fontSize: 'calc(var(--font-size) - 2px)',
    color: '#64748b',
    display: 'flex',
    alignItems: 'center',
    gap: 5,
    fontWeight: 500,
  },
  greenFoot: {
    color: '#059669',
  },
});

export function StatCard({
  title,
  value,
  icon,
  foot,
  green,
  iconColor = '#334155',
  iconBg = '#f1f5f9',
}: {
  title: string;
  value: number | string;
  icon: IconName;
  foot: ReactNode;
  green?: boolean;
  iconColor?: string;
  iconBg?: string;
}) {
  return (
    <div {...stylex.props(styles.card)}>
      <div {...stylex.props(styles.head)}>
        <span>{title}</span>
        <span {...stylex.props(styles.iconWrapper)} style={{ backgroundColor: iconBg, color: iconColor }}>
          <Icon name={icon} size={16} />
        </span>
      </div>
      <p {...stylex.props(styles.number)}>{value}</p>
      <p {...stylex.props(styles.foot, green && styles.greenFoot)}>
        {green && <Icon name="check" size={13} />}
        {foot}
      </p>
    </div>
  );
}
