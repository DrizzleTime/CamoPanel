import type { ReactNode } from "react";
import { Icon } from "../Icon";
import { IconButton } from "./IconButton";
import "./ui.css";

export function Alert({
  children,
  tone = "info",
  onDismiss,
  className = "",
}: {
  children: ReactNode;
  tone?: "info" | "success" | "error" | "warning";
  onDismiss?: () => void;
  className?: string;
}) {
  return (
    <div
      className={`ui-alert ui-alert-${tone} ${className}`}
      role={tone === "error" ? "alert" : "status"}
    >
      <Icon
        name={
          tone === "success"
            ? "check"
            : tone === "error" || tone === "warning"
              ? "alertCircle"
              : "info"
        }
        size={17}
      />
      <div className="ui-alert-content">{children}</div>
      {onDismiss && (
        <IconButton aria-label="关闭提示" onClick={onDismiss}>
          <Icon name="close" size={15} />
        </IconButton>
      )}
    </div>
  );
}
