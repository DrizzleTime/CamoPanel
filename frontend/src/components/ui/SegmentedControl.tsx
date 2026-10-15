import { type ReactNode } from "react";
import * as stylex from "@stylexjs/stylex";

const styles = stylex.create({
  container: {
    display: "inline-flex",
    alignItems: "center",
    padding: 2,
    borderRadius: "var(--radius-control)",
    backgroundColor: "var(--surface-subtle)",
    borderWidth: 1,
    borderStyle: "solid",
    borderColor: "var(--border-color)",
    maxWidth: "100%",
    overflowX: "auto",
    flexShrink: 0,
  },
  item: {
    padding: { default: "4px 10px", "@media (max-width: 480px)": "4px 8px" },
    minHeight: "calc(var(--control-height-sm) - 6px)",
    fontSize: "calc(var(--font-size) - 1px)",
    fontWeight: 500,
    borderRadius: "var(--radius-sm)",
    borderWidth: 0,
    backgroundColor: {
      default: "transparent",
      ":hover": "var(--surface-hover)",
    },
    color: "var(--text-muted)",
    cursor: "pointer",
    transition: "all .14s ease",
    whiteSpace: "nowrap",
    display: "inline-flex",
    alignItems: "center",
    gap: 6,
    outlineWidth: { default: 0, ":focus-visible": 2 },
    outlineStyle: "solid",
    outlineColor: "var(--accent)",
    outlineOffset: -2,
  },
  itemActive: {
    backgroundColor: { default: "var(--card-bg)", ":hover": "var(--card-bg)" },
    color: "var(--accent-text)",
    fontWeight: 600,
    boxShadow: "0 1px 3px rgba(15, 23, 42, 0.12)",
  },
});

export interface SegmentOption<T extends string = string> {
  value: T;
  label: ReactNode;
  icon?: ReactNode;
}

export function SegmentedControl<T extends string = string>({
  options,
  value,
  onChange,
  label = "显示范围",
}: {
  options: SegmentOption<T>[];
  value: T;
  onChange: (val: T) => void;
  label?: string;
}) {
  return (
    <div
      {...stylex.props(styles.container)}
      role="radiogroup"
      aria-label={label}
    >
      {options.map((opt) => {
        const isActive = opt.value === value;
        return (
          <button
            key={opt.value}
            type="button"
            role="radio"
            aria-checked={isActive}
            tabIndex={isActive ? 0 : -1}
            {...stylex.props(styles.item, isActive && styles.itemActive)}
            onClick={() => onChange(opt.value)}
            onKeyDown={(event) => {
              const index = options.indexOf(opt);
              let next = index;
              if (event.key === "ArrowRight" || event.key === "ArrowDown")
                next = (index + 1) % options.length;
              else if (event.key === "ArrowLeft" || event.key === "ArrowUp")
                next = (index - 1 + options.length) % options.length;
              else if (event.key === "Home") next = 0;
              else if (event.key === "End") next = options.length - 1;
              else return;
              event.preventDefault();
              onChange(options[next].value);
              (
                event.currentTarget.parentElement?.children[
                  next
                ] as HTMLButtonElement
              )?.focus();
            }}
          >
            {opt.icon}
            <span>{opt.label}</span>
          </button>
        );
      })}
    </div>
  );
}
