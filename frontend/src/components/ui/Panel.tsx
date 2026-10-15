import { type ReactNode } from "react";
import * as stylex from "@stylexjs/stylex";

const styles = stylex.create({
  panel: {
    backgroundColor: "var(--card-bg)",
    borderWidth: 1,
    borderStyle: "solid",
    borderColor: "var(--border-color, #e5e7eb)",
    borderRadius: "var(--radius-panel)",
    overflow: "hidden",
    boxShadow: "var(--shadow-card)",
  },
  header: {
    padding: "var(--card-padding)",
    minHeight: 62,
    flexWrap: "wrap",
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 12,
    borderBottomWidth: 1,
    borderBottomStyle: "solid",
    borderBottomColor: "var(--border-color, #e5e7eb)",
    backgroundColor: "var(--card-bg)",
  },
  title: {
    fontSize: "calc(var(--font-size) + 1px)",
    fontWeight: 600,
    color: "var(--text-primary)",
    display: "flex",
    alignItems: "center",
    gap: 9,
  },
  heading: { display: "flex", alignItems: "center", gap: 12, minWidth: 0 },
  description: {
    marginTop: 3,
    fontSize: "var(--text-xs)",
    lineHeight: 1.6,
    color: "var(--text-muted)",
  },
});

export function Panel({
  children,
  style,
  className = "",
}: {
  children: ReactNode;
  style?: any;
  className?: string;
}) {
  const styled = stylex.props(styles.panel, style);
  return (
    <section {...styled} className={`${styled.className} ${className}`}>
      {children}
    </section>
  );
}

export function PanelHeader({
  title,
  icon,
  badge,
  extra,
  description,
}: {
  title: ReactNode;
  icon?: ReactNode;
  badge?: ReactNode;
  extra?: ReactNode;
  description?: string;
}) {
  return (
    <div {...stylex.props(styles.header)}>
      <div {...stylex.props(styles.heading)}>
        {icon}
        <div>
          <h2 {...stylex.props(styles.title)}>
            <span>{title}</span>
            {badge}
          </h2>
          {description && (
            <p {...stylex.props(styles.description)}>{description}</p>
          )}
        </div>
      </div>
      {extra}
    </div>
  );
}
