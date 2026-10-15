import { Select } from "./ui/Select";
import { useState, type FormEvent } from "react";
import * as stylex from "@stylexjs/stylex";
import { api } from "../lib/api";
import { Icon } from "./Icon";
import { Button } from "./ui/Button";

const styles = stylex.create({
  form: {
    display: "flex",
    flexDirection: "column",
    gap: 16,
    padding: "4px 0",
  },
  desc: {
    fontSize: 12.5,
    color: "#475569",
    lineHeight: 1.6,
  },
  label: {
    display: "flex",
    flexDirection: "column",
    gap: 7,
    fontSize: 12.5,
    fontWeight: 550,
    color: "#334155",
  },
  input: {
    width: "100%",
    minWidth: 0,
    padding: "8px 12px",
    border: "1px solid #cbd5e1",
    borderRadius: 7,
    color: "#0f172a",
    backgroundColor: "#ffffff",
    fontSize: 13,
    outline: { default: "none", ":focus-visible": "2px solid var(--accent)" },
    outlineOffset: 2,
    borderColor: {
      default: "#cbd5e1",
      ":hover": "#94a3b8",
      ":focus": "var(--accent)",
    },
    boxShadow: "0 1px 2px rgba(0, 0, 0, 0.02)",
    transition: "border-color .15s",
  },
  columns: {
    display: "grid",
    gridTemplateColumns: "1fr 1fr",
    gap: 14,
  },
  error: {
    backgroundColor: "#fef2f2",
    color: "#b91c1c",
    padding: "11px 15px",
    borderRadius: 8,
    border: "1px solid #fee2e2",
    fontSize: 12.5,
    lineHeight: 1.55,
  },
  actions: {
    display: "flex",
    justifyContent: "flex-end",
    gap: 9,
    marginTop: 10,
  },
});

export function CreateContainer({
  images,
  onCreated,
  onClose,
}: {
  images: string[];
  onCreated: () => void;
  onClose: () => void;
}) {
  const [name, setName] = useState("");
  const [image, setImage] = useState(images[0] || "");
  const [hostPort, setHostPort] = useState("");
  const [containerPort, setContainerPort] = useState("");
  const [hostPort2, setHostPort2] = useState("");
  const [containerPort2, setContainerPort2] = useState("");
  const [envText, setEnvText] = useState("");
  const [command, setCommand] = useState("");
  const [restart, setRestart] = useState("");
  const [bind, setBind] = useState("");
  const [confirmMount, setConfirmMount] = useState(false);
  const [pull, setPull] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    if (!!hostPort !== !!containerPort || !!hostPort2 !== !!containerPort2) {
      setError("映射端口须同时填写，或同时留空。");
      return;
    }
    setBusy(true);
    try {
      const ports = [
        hostPort
          ? {
              host_port: Number(hostPort),
              container_port: Number(containerPort),
              protocol: "tcp",
            }
          : null,
        hostPort2
          ? {
              host_port: Number(hostPort2),
              container_port: Number(containerPort2),
              protocol: "tcp",
            }
          : null,
      ].filter(
        (
          item,
        ): item is {
          host_port: number;
          container_port: number;
          protocol: string;
        } => item !== null,
      );
      const env = envText
        .split("\n")
        .map((line) => line.trim())
        .filter(Boolean);
      const commandParts = command.trim() ? command.trim().split(/\s+/) : [];
      const extended =
        ports.length > 1 ||
        env.length > 0 ||
        commandParts.length > 0 ||
        restart !== "" ||
        bind.trim() !== "" ||
        pull;
      if (extended) {
        await api.create({
          name,
          image,
          ports,
          env,
          command: commandParts,
          restart_policy: restart,
          binds: bind.trim() ? [bind.trim()] : [],
          confirm_host_mount: confirmMount,
          pull,
        });
      } else {
        await api.create({
          name,
          image,
          host_port: Number(hostPort) || 0,
          container_port: Number(containerPort) || 0,
        });
      }
      onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建失败");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form {...stylex.props(styles.form)} onSubmit={submit}>
      <p {...stylex.props(styles.desc)}>
        使用本地已有镜像创建容器。容器创建后处于停止状态，可在列表中启动。
      </p>

      {error && (
        <div {...stylex.props(styles.error)} role="alert">
          {error}
        </div>
      )}

      <label {...stylex.props(styles.label)}>
        容器名称
        <input
          {...stylex.props(styles.input)}
          autoFocus
          data-autofocus
          required
          minLength={2}
          maxLength={63}
          pattern="[a-zA-Z0-9][a-zA-Z0-9_.\-]+"
          placeholder="例如 my-web"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </label>

      <label {...stylex.props(styles.label)}>
        本地镜像
        <Select
          required
          aria-label="本地镜像"
          value={image}
          onValueChange={setImage}
          placeholder={images.length ? "选择本地镜像" : "当前没有本地镜像"}
          options={images.map((tag) => ({ value: tag, label: tag }))}
        />
      </label>

      <div {...stylex.props(styles.columns)}>
        <label {...stylex.props(styles.label)}>
          宿主机端口（可选）
          <input
            {...stylex.props(styles.input)}
            type="number"
            min="1"
            max="65535"
            placeholder="8088"
            value={hostPort}
            onChange={(e) => setHostPort(e.target.value)}
          />
        </label>
        <label {...stylex.props(styles.label)}>
          容器端口（可选）
          <input
            {...stylex.props(styles.input)}
            type="number"
            min="1"
            max="65535"
            placeholder="80"
            value={containerPort}
            onChange={(e) => setContainerPort(e.target.value)}
          />
        </label>
      </div>

      <div {...stylex.props(styles.columns)}>
        <label {...stylex.props(styles.label)}>
          第二组宿主机端口
          <input
            {...stylex.props(styles.input)}
            type="number"
            min="1"
            max="65535"
            value={hostPort2}
            onChange={(e) => setHostPort2(e.target.value)}
          />
        </label>
        <label {...stylex.props(styles.label)}>
          第二组容器端口
          <input
            {...stylex.props(styles.input)}
            type="number"
            min="1"
            max="65535"
            value={containerPort2}
            onChange={(e) => setContainerPort2(e.target.value)}
          />
        </label>
      </div>
      <label {...stylex.props(styles.label)}>
        环境变量（每行 KEY=VALUE）
        <textarea
          {...stylex.props(styles.input)}
          rows={2}
          value={envText}
          onChange={(e) => setEnvText(e.target.value)}
        />
      </label>
      <label {...stylex.props(styles.label)}>
        启动命令
        <input
          {...stylex.props(styles.input)}
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          placeholder="nginx -g daemon off;"
        />
      </label>
      <label {...stylex.props(styles.label)}>
        重启策略
        <Select
          aria-label="重启策略"
          value={restart || "unset"}
          onValueChange={(value) => setRestart(value === "unset" ? "" : value)}
          options={[
            { value: "unset", label: "不设置" },
            ...["no", "always", "on-failure", "unless-stopped"].map(
              (value) => ({ value, label: value }),
            ),
          ]}
        />
      </label>
      <label {...stylex.props(styles.label)}>
        挂载（宿主机:容器）
        <input
          {...stylex.props(styles.input)}
          value={bind}
          onChange={(e) => setBind(e.target.value)}
          placeholder="相对数据目录，或确认后的绝对路径"
        />
      </label>
      <label {...stylex.props(styles.desc)}>
        <input
          type="checkbox"
          checked={confirmMount}
          onChange={(e) => setConfirmMount(e.target.checked)}
        />{" "}
        确认挂载数据目录之外的路径
      </label>
      <label {...stylex.props(styles.desc)}>
        <input
          type="checkbox"
          checked={pull}
          onChange={(e) => setPull(e.target.checked)}
        />{" "}
        创建前拉取镜像（默认不隐式拉取）
      </label>
      <p {...stylex.props(styles.desc)} style={{ fontSize: 12 }}>
        只填第一组端口时仍使用原来的单端口请求。端口映射使用 TCP。
      </p>

      <div {...stylex.props(styles.actions)}>
        <Button type="button" onClick={onClose} disabled={busy}>
          取消
        </Button>
        <Button
          type="submit"
          variant="primary"
          disabled={busy || !images.length}
        >
          <Icon name="plus" size={15} />
          <span>{busy ? "创建中…" : "创建容器"}</span>
        </Button>
      </div>
    </form>
  );
}
