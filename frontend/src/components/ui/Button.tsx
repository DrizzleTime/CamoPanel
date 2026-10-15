import { type ComponentPropsWithRef, type ReactNode } from "react";
import * as stylex from "@stylexjs/stylex";

const styles = stylex.create({
  base: {
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    gap: 8,
    borderRadius: "var(--radius-control)",
    fontWeight: 600,
    lineHeight: 1.4,
    whiteSpace: "nowrap",
    transition:
      "background-color .16s ease, border-color .16s ease, box-shadow .16s ease, transform .16s ease",
    cursor: { default: "pointer", ":disabled": "not-allowed" },
    borderWidth: 1,
    borderStyle: "solid",
    outlineWidth: { default: 0, ":focus-visible": 2 },
    outlineStyle: "solid",
    outlineColor: "var(--accent)",
    outlineOffset: 3,
    transform: { default: "none", ":active:not(:disabled)": "translateY(1px)" },
    opacity: { default: 1, ":disabled": 0.45 },
  },
  sizeSm: {
    padding: "6px 12px",
    minHeight: "var(--control-height-sm)",
    fontSize: "calc(var(--font-size) - 1px)",
  },
  sizeMd: {
    padding: "8px 16px",
    minHeight: "var(--control-height)",
    fontSize: "var(--font-size)",
  },
  default: {
    borderColor: {
      default: "var(--control-border)",
      ":hover:not(:disabled)": "var(--control-border-hover)",
    },
    backgroundColor: {
      default: "var(--card-bg)",
      ":hover:not(:disabled)": "var(--surface-hover)",
    },
    color: "var(--text-secondary)",
    boxShadow: "0 1px 2px rgba(0, 0, 0, 0.04)",
  },
  primary: {
    borderColor: "var(--accent, #d85c49)",
    backgroundColor: {
      default: "var(--accent, #d85c49)",
      ":hover:not(:disabled)": "var(--accent-hover, #c44f3d)",
    },
    color: "#ffffff",
    boxShadow: {
      default: "0 1px 2px rgba(15, 23, 42, .12)",
      ":hover:not(:disabled)": "0 3px 8px rgba(15, 23, 42, .16)",
    },
  },
  ghost: {
    borderColor: "transparent",
    backgroundColor: {
      default: "transparent",
      ":hover:not(:disabled)": "var(--surface-hover)",
    },
    color: "var(--text-secondary)",
  },
  danger: {
    borderColor: "transparent",
    backgroundColor: { default: "#dc2626", ":hover:not(:disabled)": "#b91c1c" },
    color: "#ffffff",
    boxShadow: "0 1px 2px rgba(15, 23, 42, .08)",
  },
});

export interface ButtonProps extends ComponentPropsWithRef<"button"> {
  variant?: "default" | "primary" | "danger" | "ghost";
  size?: "sm" | "md";
  children: ReactNode;
}

export function Button({
  variant = "default",
  size = "md",
  type = "button",
  children,
  className,
  ...props
}: ButtonProps) {
  const styled = stylex.props(
    styles.base,
    size === "sm" ? styles.sizeSm : styles.sizeMd,
    styles[variant],
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
