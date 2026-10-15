import { type ComponentPropsWithRef } from "react";
import * as stylex from "@stylexjs/stylex";
import { Icon } from "../Icon";

const styles = stylex.create({
  wrapper: {
    display: "flex",
    gap: 9,
    alignItems: "center",
    borderWidth: 1,
    borderStyle: "solid",
    borderColor: {
      default: "var(--control-border)",
      ":hover": "var(--control-border-hover)",
      ":focus-within": "var(--accent)",
    },
    borderRadius: "var(--radius-control)",
    padding: "var(--control-padding)",
    minHeight: "var(--control-height)",
    width: "100%",
    minWidth: 0,
    flex: "1 1 220px",
    color: "var(--text-muted)",
    backgroundColor: "var(--card-bg)",
    boxShadow: {
      default: "none",
      ":focus-within": "0 0 0 3px var(--accent-light)",
    },
    transition: "border-color .16s ease, box-shadow .16s ease",
  },
  input: {
    borderWidth: 0,
    outline: "none",
    backgroundColor: "transparent",
    width: "100%",
    fontSize: "calc(var(--font-size) - 1px)",
    color: "var(--text-primary)",
    minWidth: 0,
  },
});

export interface SearchInputProps extends ComponentPropsWithRef<"input"> {
  maxWidth?: number | string;
}

export function SearchInput({
  maxWidth = 340,
  className = "",
  ...props
}: SearchInputProps) {
  const wrapper = stylex.props(styles.wrapper);
  return (
    <label
      {...wrapper}
      className={`${wrapper.className} ${className}`}
      style={{ maxWidth }}
    >
      <Icon name="search" size={15} />
      <input
        {...stylex.props(styles.input)}
        type="search"
        aria-label={props.placeholder || "搜索资源"}
        {...props}
      />
    </label>
  );
}
