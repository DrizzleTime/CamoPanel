import {
  useEffect,
  useImperativeHandle,
  useRef,
  type ComponentPropsWithRef,
  type ReactNode,
} from "react";
import "./ui.css";

export function Checkbox({
  indeterminate = false,
  className = "",
  ref,
  ...props
}: Omit<ComponentPropsWithRef<"input">, "type"> & { indeterminate?: boolean }) {
  const input = useRef<HTMLInputElement>(null);
  useImperativeHandle(ref, () => input.current!, []);
  useEffect(() => {
    if (input.current) input.current.indeterminate = indeterminate;
  }, [indeterminate]);
  return (
    <input
      {...props}
      ref={input}
      type="checkbox"
      className={`ui-checkbox ${className}`}
    />
  );
}

export function Switch({
  label,
  className = "",
  ...props
}: Omit<ComponentPropsWithRef<"input">, "type" | "role"> & {
  label: ReactNode;
}) {
  return (
    <label className={`ui-switch-label ${className}`}>
      <input {...props} className="ui-switch" type="checkbox" role="switch" />
      <span>{label}</span>
    </label>
  );
}
