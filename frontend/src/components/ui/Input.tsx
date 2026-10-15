import type { ComponentPropsWithRef } from "react";
import "./ui.css";

export function Input({
  className = "",
  ...props
}: ComponentPropsWithRef<"input">) {
  return (
    <input
      className={`ui-input ${className}`}
      data-autofocus={props.autoFocus || undefined}
      {...props}
    />
  );
}

export function Textarea({
  className = "",
  ...props
}: ComponentPropsWithRef<"textarea">) {
  return (
    <textarea
      className={`ui-input ui-textarea ${className}`}
      data-autofocus={props.autoFocus || undefined}
      {...props}
    />
  );
}
