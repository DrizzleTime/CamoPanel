import {
  Children,
  cloneElement,
  isValidElement,
  useId,
  type ReactElement,
  type ReactNode,
} from "react";
import "./ui.css";

export function FormField({
  label,
  hint,
  error,
  children,
  className = "",
}: {
  label: string;
  hint?: ReactNode;
  error?: string;
  children: ReactNode;
  className?: string;
}) {
  const generatedId = useId();
  const child =
    Children.count(children) === 1 && isValidElement(children)
      ? (children as ReactElement<{
          id?: string;
          "aria-describedby"?: string;
          "aria-invalid"?: boolean | "true" | "false";
        }>)
      : null;
  const id = child?.props.id || generatedId;
  const description =
    [
      child?.props["aria-describedby"],
      hint && `${id}-hint`,
      error && `${id}-error`,
    ]
      .filter(Boolean)
      .join(" ") || undefined;
  return (
    <div className={`ui-field ${className}`}>
      <label className="ui-field-label" htmlFor={id}>
        {label}
      </label>
      {child
        ? cloneElement(child, {
            id,
            "aria-describedby": description,
            "aria-invalid": error ? true : child.props["aria-invalid"],
          })
        : children}
      {hint && (
        <small className="ui-field-hint" id={`${id}-hint`}>
          {hint}
        </small>
      )}
      {error && (
        <small className="ui-field-error" id={`${id}-error`} role="alert">
          {error}
        </small>
      )}
    </div>
  );
}
