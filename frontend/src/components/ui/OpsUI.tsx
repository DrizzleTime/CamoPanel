import { useState, type ReactNode } from "react";
import { Icon, type IconName } from "../Icon";
import "./ops.css";
import { Alert } from "./Alert";
import { Panel, PanelHeader } from "./Panel";
export { FormField as OpsField } from "./FormField";

export function useOpsFeedback() {
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);

  async function run(action: () => Promise<unknown>, success?: string) {
    setError("");
    setMessage("");
    setBusy(true);
    try {
      await action();
      if (success !== undefined) setMessage(success);
    } catch (err) {
      setError(err instanceof Error ? err.message : "操作失败，请重试");
    } finally {
      setBusy(false);
    }
  }

  return { error, setError, message, setMessage, busy, run };
}

export function OpsPage({
  title,
  feedback,
  children,
}: {
  title: string;
  feedback: ReturnType<typeof useOpsFeedback>;
  children: ReactNode;
}) {
  return (
    <section className="ops-page" aria-label={title} aria-busy={feedback.busy}>
      {feedback.error && (
        <Alert tone="error" onDismiss={() => feedback.setError("")}>
          {feedback.error}
        </Alert>
      )}
      {feedback.message && (
        <Alert tone="success" onDismiss={() => feedback.setMessage("")}>
          {feedback.message}
        </Alert>
      )}
      {feedback.busy && (
        <div className="ops-working" role="status">
          <span className="ops-spinner" />
          正在处理，请稍候…
        </div>
      )}
      <fieldset className="ops-content" disabled={feedback.busy}>
        {children}
      </fieldset>
    </section>
  );
}

export function OpsCard({
  title,
  description,
  icon,
  extra,
  children,
}: {
  title: string;
  description?: string;
  icon: IconName;
  extra?: ReactNode;
  children: ReactNode;
}) {
  return (
    <Panel className="ops-card">
      <PanelHeader
        title={title}
        description={description}
        extra={extra}
        icon={
          <span className="ops-card-icon">
            <Icon name={icon} size={18} />
          </span>
        }
      />
      <div className="ops-card-body">{children}</div>
    </Panel>
  );
}

export function OpsEmpty({
  icon,
  title,
  description,
}: {
  icon: IconName;
  title: string;
  description: string;
}) {
  return (
    <div className="ops-empty">
      <span className="ops-empty-icon">
        <Icon name={icon} size={25} />
      </span>
      <h3>{title}</h3>
      <p>{description}</p>
    </div>
  );
}

export function OpsStats({
  items,
}: {
  items: { label: string; value: ReactNode; icon: IconName; hint: string }[];
}) {
  return (
    <div className="ops-stats">
      {items.map((item) => (
        <div className="ops-stat" key={item.label}>
          <div className="ops-stat-top">
            <span>{item.label}</span>
            <Icon name={item.icon} size={16} />
          </div>
          <strong>{item.value}</strong>
          <small>{item.hint}</small>
        </div>
      ))}
    </div>
  );
}

export function OpsTag({
  children,
  tone = "neutral",
}: {
  children: ReactNode;
  tone?: "neutral" | "success" | "warning";
}) {
  return <span className={`ops-tag ops-tag-${tone}`}>{children}</span>;
}

export function OpsLoading() {
  return (
    <div className="ops-loading" role="status">
      <span className="ops-spinner" />
      正在加载…
    </div>
  );
}

export function opsDate(value: string) {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime())
    ? value
    : date.toLocaleString("zh-CN", {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
        hour12: false,
      });
}
