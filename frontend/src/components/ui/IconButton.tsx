import { type ComponentPropsWithRef, type ReactNode } from "react";
import * as stylex from "@stylexjs/stylex";

const styles = stylex.create({
  base: {
    display: "inline-grid",
    placeItems: "center",
    borderRadius: "var(--radius-control)",
    transition:
      "background-color .16s ease, border-color .16s ease, color .16s ease, transform .16s ease",
    cursor: { default: "pointer", ":disabled": "not-allowed" },
    borderWidth: 1,
    borderStyle: "solid",
    padding: 0,
    outlineWidth: { default: 0, ":focus-visible": 2 },
    outlineStyle: "solid",
    outlineColor: "var(--accent)",
    outlineOffset: 3,
    transform: { default: "none", ":active:not(:disabled)": "translateY(1px)" },
    opacity: { default: 1, ":disabled": 0.35 },
  },
  // 顶栏/工具栏普通图标按钮
  ghost: {
    borderColor: "transparent",
    backgroundColor: {
      default: "transparent",
      ":hover:not(:disabled)": "var(--surface-hover)",
    },
    color: "var(--text-muted)",
    width: "var(--control-height-sm)",
    height: "var(--control-height-sm)",
  },
  ghostDanger: {
    borderColor: "transparent",
    backgroundColor: {
      default: "transparent",
      ":hover:not(:disabled)": "var(--danger-bg)",
    },
    color: {
      default: "var(--text-muted)",
      ":hover:not(:disabled)": "var(--danger)",
    },
    width: "var(--control-height-sm)",
    height: "var(--control-height-sm)",
  },
  active: {
    backgroundColor: "var(--accent-light)",
    color: "var(--accent-text)",
  },
  // 表格微按钮操作
  tableAction: {
    width: "var(--control-height-sm)",
    height: "var(--control-height-sm)",
  },
  default: {
    borderColor: "var(--control-border)",
    backgroundColor: {
      default: "var(--card-bg)",
      ":hover:not(:disabled)": "var(--surface-hover)",
    },
    color: "var(--text-muted)",
  },
  start: {
    borderColor: "#d1fae5",
    backgroundColor: { default: "#ecfdf5", ":hover:not(:disabled)": "#d1fae5" },
    color: "#059669",
  },
  stop: {
    borderColor: "#fef3c7",
    backgroundColor: { default: "#fffbeb", ":hover:not(:disabled)": "#fde68a" },
    color: "#d97706",
  },
  restart: {
    borderColor: "#dbeafe",
    backgroundColor: { default: "#eff6ff", ":hover:not(:disabled)": "#bfdbfe" },
    color: "#2563eb",
  },
  remove: {
    borderColor: "#fee2e2",
    backgroundColor: { default: "#fef2f2", ":hover:not(:disabled)": "#fecaca" },
    color: "#ef4444",
  },
});

export interface IconButtonProps extends ComponentPropsWithRef<"button"> {
  variant?:
    | "ghost"
    | "ghostDanger"
    | "default"
    | "start"
    | "stop"
    | "restart"
    | "remove";
  children: ReactNode;
}

export function IconButton({
  variant = "ghost",
  type = "button",
  children,
  className,
  ...props
}: IconButtonProps) {
  const isTableAction = variant !== "ghost" && variant !== "ghostDanger";
  const styled = stylex.props(
    styles.base,
    isTableAction ? styles.tableAction : styles.ghost,
    styles[variant],
    props["aria-pressed"] === true && styles.active,
  );
  return (
    <button
      type={type}
      {...styled}
      className={[styled.className, className].filter(Boolean).join(" ")}
      {...props}
    >
      {children}
    </button>
  );
}
