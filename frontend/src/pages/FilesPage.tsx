import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
  type MouseEvent as ReactMouseEvent,
} from "react";
import {
  api,
  ApiError,
  type FileEntry,
  type FileRoots,
  type FileBatch,
} from "../lib/api";
import { Icon, type IconName } from "../components/Icon";
import { Button } from "../components/ui/Button";
import { IconButton } from "../components/ui/IconButton";
import { Input } from "../components/ui/Input";
import { FormField } from "../components/ui/FormField";
import { Checkbox, Switch } from "../components/ui/Checkbox";
import { DropdownMenu, type MenuAction } from "../components/ui/DropdownMenu";
import { ContextMenu } from "../components/ui/ContextMenu";
import { SearchInput } from "../components/ui/SearchInput";
import { SegmentedControl } from "../components/ui/SegmentedControl";
import { Tooltip } from "../components/ui/Tooltip";
import { Alert } from "../components/ui/Alert";
import { Modal, ModalFooter } from "../components/Modal";
import { useOpsFeedback, opsDate } from "../components/ui/OpsUI";
import "./files.css";

type Location = { root: string; path: string };
type Clipboard = { location: Location; entries: FileEntry[]; move: boolean };
type FileDialog =
  | { type: "create"; directory: boolean }
  | { type: "rename"; entry: FileEntry }
  | { type: "permissions"; entry: FileEntry }
  | { type: "properties"; entry: FileEntry }
  | { type: "delete"; entries: FileEntry[] }
  | { type: "archive"; entries: FileEntry[] }
  | { type: "extract"; entry: FileEntry }
  | { type: "editor"; entry: FileEntry; original: string }
  | { type: "upload" };
type UploadItem = {
  file: File;
  progress: number;
  status: "pending" | "uploading" | "done" | "error";
  error?: string;
};
type TreeData = { entries: FileEntry[]; loading: boolean; error?: string };
type SortKey = "name" | "size" | "modified" | "type";
const joinPath = (base: string, name: string) =>
  `${base.replace(/\/$/, "")}/${name}`.replace(
    /^\//,
    base.startsWith("/") ? "/" : "",
  );
const parentPath = (path: string) =>
  path.replace(/\/$/, "").split("/").slice(0, -1).join("/") ||
  (path.startsWith("/") ? "/" : "");
const fileSize = (size: number) =>
  size < 1024
    ? `${size} B`
    : size < 1024 ** 2
      ? `${(size / 1024).toFixed(1)} KB`
      : size < 1024 ** 3
        ? `${(size / 1024 ** 2).toFixed(1)} MB`
        : `${(size / 1024 ** 3).toFixed(2)} GB`;
function fileKind(entry: FileEntry) {
  if (entry.dir)
    return {
      label: entry.symlink ? "目录链接" : "文件夹",
      icon: "folder" as IconName,
      color: "folder",
    };
  if (entry.symlink)
    return { label: "符号链接", icon: "link" as IconName, color: "link" };
  if (entry.type === "special")
    return { label: "特殊文件", icon: "drive" as IconName, color: "neutral" };
  const extension = entry.name.split(".").pop()?.toLowerCase() || "";
  if (/^(zip|gz|tgz|tar|bz2|xz|7z|rar)$/.test(extension))
    return {
      label: `${extension.toUpperCase()} 压缩包`,
      icon: "archive" as IconName,
      color: "archive",
    };
  if (/^(png|jpg|jpeg|gif|svg|webp|ico|avif)$/.test(extension))
    return {
      label: `${extension.toUpperCase()} 图像`,
      icon: "image" as IconName,
      color: "image",
    };
  if (
    /^(js|jsx|ts|tsx|json|html|css|php|py|go|sh|yml|yaml|conf|toml|xml|md)$/.test(
      extension,
    )
  )
    return {
      label: `${extension.toUpperCase()} 文件`,
      icon: "code" as IconName,
      color: "code",
    };
  return {
    label:
      extension && extension !== entry.name.toLowerCase()
        ? `${extension.toUpperCase()} 文件`
        : "文件",
    icon: "fileText" as IconName,
    color: "neutral",
  };
}

function FileGlyph({
  entry,
  large = false,
}: {
  entry: FileEntry;
  large?: boolean;
}) {
  const kind = fileKind(entry);
  return (
    <span
      className={`fe-file-icon fe-icon-${kind.color}${large ? " fe-icon-large" : ""}`}
    >
      <Icon name={kind.icon} size={large ? 32 : 18} />
      {entry.symlink && <span className="fe-link-mark">↗</span>}
    </span>
  );
}
function Tool({
  icon,
  children,
  onClick,
  disabled = false,
  danger = false,
  title,
}: {
  icon: IconName;
  children?: React.ReactNode;
  onClick: () => void;
  disabled?: boolean;
  danger?: boolean;
  title?: string;
}) {
  const content = (
    <>
      <Icon name={icon} size={16} />
      {children}
    </>
  );
  const control = children ? (
    <Button
      variant="ghost"
      size="sm"
      disabled={disabled}
      onClick={onClick}
      aria-label={title}
    >
      {content}
    </Button>
  ) : (
    <IconButton
      variant={danger ? "ghostDanger" : "ghost"}
      disabled={disabled}
      onClick={onClick}
      aria-label={title}
    >
      {content}
    </IconButton>
  );
  return title ? <Tooltip content={title}>{control}</Tooltip> : control;
}

export function FilesPage({
  refreshKey = 0,
  search,
  onSearchChange,
  onConnectionChange,
  onErrorChange,
}: {
  refreshKey?: number;
  search: string;
  onSearchChange: (value: string) => void;
  onConnectionChange: (connected: boolean) => void;
  onErrorChange: (error: boolean) => void;
}) {
  const feedback = useOpsFeedback();
  const [meta, setMeta] = useState<FileRoots | null>(null);
  const [history, setHistory] = useState<Location[]>([
    { root: "server", path: "" },
  ]);
  const [historyIndex, setHistoryIndex] = useState(0);
  const current = history[historyIndex];
  const [entries, setEntries] = useState<FileEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [reload, setReload] = useState(0);
  const [address, setAddress] = useState("");
  const [editingAddress, setEditingAddress] = useState(false);
  const [selection, setSelection] = useState<Set<string>>(new Set());
  const anchor = useRef("");
  const [clipboard, setClipboard] = useState<Clipboard | null>(null);
  const [showHidden, setShowHidden] = useState(false);
  const [view, setView] = useState<"list" | "grid">("list");
  const [sort, setSort] = useState<{ key: SortKey; ascending: boolean }>({
    key: "name",
    ascending: true,
  });
  const [tree, setTree] = useState<Record<string, TreeData>>({});
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [dialog, setDialog] = useState<FileDialog | null>(null);
  const [value, setValue] = useState("");
  const [editor, setEditor] = useState("");
  const [editorWrap, setEditorWrap] = useState(false);
  const [uploads, setUploads] = useState<UploadItem[]>([]);
  const [overwrite, setOverwrite] = useState(false);
  const [context, setContext] = useState<{
    x: number;
    y: number;
    entries: FileEntry[];
  } | null>(null);
  const [dragging, setDragging] = useState(false);
  const dragDepth = useRef(0);
  const uploadInput = useRef<HTMLInputElement>(null);
  const pathInput = useRef<HTMLInputElement>(null);
  const searchInput = useRef<HTMLInputElement>(null);
  const listingRequest = useRef(0);
  const readOnly = meta?.read_only ?? true;
  const unavailable = !meta || loading || !!loadError;
  const busy = feedback.busy;
  const selectedEntries = entries.filter((entry) => selection.has(entry.path));
  const serverBase = meta?.server_root || "/";
  const scope = meta?.roots.find((root) => root.name === current.root);
  const absolutePath =
    current.root === "server"
      ? current.path || serverBase
      : joinPath(scope?.path || "", current.path);
  const atRoot =
    current.root === "server" ? current.path === serverBase : !current.path;

  useEffect(() => {
    if (meta) return;
    let active = true;
    setLoading(true);
    setLoadError("");
    void api.files
      .roots()
      .then((data) => {
        if (!active) return;
        setMeta(data);
        setHistory([{ root: "server", path: data.server_root }]);
        setHistoryIndex(0);
        onConnectionChange(true);
        onErrorChange(false);
      })
      .catch((error) => {
        if (!active) return;
        setLoadError(
          error instanceof Error ? error.message : "无法连接文件服务",
        );
        setLoading(false);
        onConnectionChange(false);
        onErrorChange(true);
      });
    return () => {
      active = false;
    };
  }, [meta, refreshKey, reload, onConnectionChange, onErrorChange]);

  useEffect(() => {
    setAddress(absolutePath);
    setEditingAddress(false);
  }, [absolutePath]);

  const refreshDirectory = useCallback(
    async (signal?: AbortSignal) => {
      const version = ++listingRequest.current;
      const listing = await api.files.list(current.root, current.path, signal);
      if (version !== listingRequest.current || signal?.aborted) return;
      setEntries(listing.entries || []);
      setLoadError("");
      onConnectionChange(true);
      onErrorChange(false);
      if (current.root === "server")
        setTree((previous) => ({
          ...previous,
          [current.path]: {
            entries: listing.entries.filter((entry) => entry.dir),
            loading: false,
          },
        }));
    },
    [current.root, current.path, onConnectionChange, onErrorChange],
  );

  useEffect(() => {
    if (!meta) return;
    const controller = new AbortController();
    setLoading(true);
    setLoadError("");
    setSelection(new Set());
    anchor.current = "";
    void refreshDirectory(controller.signal)
      .catch((error) => {
        if (controller.signal.aborted) return;
        setEntries([]);
        setLoadError(error instanceof Error ? error.message : "目录读取失败");
        const connected = error instanceof ApiError;
        onConnectionChange(connected);
        onErrorChange(!connected || error.status >= 500);
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [
    meta,
    refreshDirectory,
    refreshKey,
    reload,
    onConnectionChange,
    onErrorChange,
  ]);

  useEffect(() => {
    if (!meta || tree[serverBase]) return;
    let active = true;
    void api.files
      .list("server", serverBase)
      .then((listing) => {
        if (active)
          setTree((previous) => ({
            ...previous,
            [serverBase]: {
              entries: listing.entries.filter((entry) => entry.dir),
              loading: false,
            },
          }));
      })
      .catch(() => {});
    return () => {
      active = false;
    };
  }, [meta, serverBase, tree]);

  const filtered = useMemo(() => {
    const data = entries.filter(
      (entry) =>
        (showHidden || !entry.name.startsWith(".")) &&
        entry.name.toLowerCase().includes(search.toLowerCase()),
    );
    return data.sort((a, b) => {
      if (a.dir !== b.dir) return a.dir ? -1 : 1;
      const delta =
        sort.key === "size"
          ? a.size - b.size
          : sort.key === "modified"
            ? Date.parse(a.modified) - Date.parse(b.modified)
            : sort.key === "type"
              ? fileKind(a).label.localeCompare(fileKind(b).label)
              : a.name.localeCompare(b.name, "zh-CN", { numeric: true });
      return sort.ascending ? delta : -delta;
    });
  }, [entries, showHidden, search, sort]);

  useEffect(() => {
    const visible = new Set(filtered.map((entry) => entry.path));
    setSelection((previous) => {
      const next = new Set([...previous].filter((path) => visible.has(path)));
      return next.size === previous.size ? previous : next;
    });
  }, [filtered]);

  function navigate(location: Location) {
    if (busy) return;
    setContext(null);
    feedback.setError("");
    feedback.setMessage("");
    onSearchChange("");
    if (location.root === current.root && location.path === current.path) {
      setReload((version) => version + 1);
      return;
    }
    setHistory((previous) => [
      ...previous.slice(0, historyIndex + 1),
      location,
    ]);
    setHistoryIndex(historyIndex + 1);
  }
  function historyGo(index: number) {
    if (busy) return;
    setHistoryIndex(index);
    onSearchChange("");
    feedback.setError("");
    feedback.setMessage("");
    setContext(null);
  }
  function goUp() {
    if (!atRoot)
      navigate({ root: current.root, path: parentPath(current.path) });
  }
  function select(
    entry: FileEntry,
    event?: { ctrlKey: boolean; metaKey: boolean; shiftKey: boolean },
    toggle = false,
  ) {
    if (event?.shiftKey && anchor.current) {
      const from = filtered.findIndex((item) => item.path === anchor.current),
        to = filtered.findIndex((item) => item.path === entry.path);
      if (from >= 0 && to >= 0) {
        const range = filtered.slice(
          Math.min(from, to),
          Math.max(from, to) + 1,
        );
        setSelection(
          (previous) =>
            new Set([
              ...(event.ctrlKey || event.metaKey ? previous : []),
              ...range.map((item) => item.path),
            ]),
        );
        return;
      }
    }
    if (toggle || event?.ctrlKey || event?.metaKey)
      setSelection((previous) => {
        const next = new Set(previous);
        if (next.has(entry.path)) next.delete(entry.path);
        else next.add(entry.path);
        return next;
      });
    else setSelection(new Set([entry.path]));
    anchor.current = entry.path;
  }
  async function openEntry(entry: FileEntry) {
    if (entry.dir) {
      navigate({ root: current.root, path: entry.path });
      return;
    }
    if (entry.type === "special") {
      openDialog({ type: "properties", entry });
      return;
    }
    await feedback.run(async () => {
      const result = await api.files.read(current.root, entry.path);
      setEditor(result.content);
      setDialog({ type: "editor", entry, original: result.content });
    });
  }
  function openDialog(next: FileDialog) {
    feedback.setError("");
    setContext(null);
    setValue(
      next.type === "rename"
        ? next.entry.name
        : next.type === "permissions"
          ? next.entry.permissions
          : next.type === "archive"
            ? `${next.entries.length === 1 ? next.entries[0].name : "archive"}.zip`
            : next.type === "extract"
              ? joinPath(current.path, next.entry.name.replace(/\.zip$/i, ""))
              : "",
    );
    setDialog(next);
  }
  function closeDialog() {
    if (busy) return;
    if (
      dialog?.type === "editor" &&
      editor !== dialog.original &&
      !window.confirm("文件内容尚未保存，确认关闭编辑器？")
    )
      return;
    if (dialog?.type === "upload") setUploads([]);
    setDialog(null);
    feedback.setError("");
  }
  async function perform(action: () => Promise<unknown>, message?: string) {
    await feedback.run(async () => {
      try {
        await action();
      } finally {
        await refreshDirectory();
        setExpanded(new Set());
        setTree((previous) =>
          previous[serverBase] ? { [serverBase]: previous[serverBase] } : {},
        );
      }
      setSelection(new Set());
      setDialog(null);
    }, message);
  }
  function checkBatch(result: FileBatch) {
    const failures = result.results.filter((item) => item.error);
    if (failures.length) {
      setSelection(new Set(failures.map((item) => item.path)));
      throw new Error(
        `完成 ${result.succeeded} 项，${failures.length} 项失败：${failures
          .slice(0, 3)
          .map((item) => `${item.path.split("/").pop()}：${item.error}`)
          .join("；")}`,
      );
    }
  }
  function copy(items: FileEntry[], move: boolean) {
    setClipboard({ location: current, entries: items, move });
    setContext(null);
    feedback.setMessage(
      `已${move ? "剪切" : "复制"} ${items.length} 项，前往目标目录后粘贴`,
    );
  }
  async function paste() {
    if (!clipboard || !clipboard.entries.length) return;
    await perform(
      async () => {
        const result = await api.files.transfer(
          clipboard.location.root,
          clipboard.entries.map((item) => item.path),
          current.root,
          current.path,
          clipboard.move,
        );
        checkBatch(result);
        if (clipboard.move) setClipboard(null);
      },
      clipboard.move ? "文件已移动" : "文件已复制",
    );
  }
  function download(items: FileEntry[]) {
    void feedback.run(async () => {
      for (const item of items)
        await api.files.download(current.root, item.path);
    });
    setContext(null);
  }
  function addUploads(files: File[]) {
    if (readOnly || unavailable || busy) return;
    const accepted = files.map((file) => ({
      file,
      progress: 0,
      status: "pending" as const,
    }));
    setUploads((previous) => {
      const names = new Set(previous.map((item) => item.file.name));
      return [
        ...previous,
        ...accepted.filter((item) => !names.has(item.file.name)),
      ];
    });
    setDialog({ type: "upload" });
    setContext(null);
  }
  async function startUploads() {
    await feedback.run(async () => {
      let completed = 0;
      for (let index = 0; index < uploads.length; index++) {
        const item = uploads[index];
        if (item.status === "done") continue;
        const update = (change: Partial<UploadItem>) =>
          setUploads((previous) =>
            previous.map((entry, position) =>
              position === index ? { ...entry, ...change } : entry,
            ),
          );
        if (item.file.size > (meta?.upload_limit || 0)) {
          update({ status: "error", error: "超过单文件 256 MB 限制" });
          continue;
        }
        update({ status: "uploading", error: undefined });
        try {
          await api.files.uploadFile(
            current.root,
            joinPath(current.path, item.file.name),
            item.file,
            overwrite,
            (progress) => update({ progress }),
          );
          update({ status: "done", progress: 100 });
          completed++;
        } catch (error) {
          update({
            status: "error",
            error: error instanceof Error ? error.message : "上传失败",
          });
        }
      }
      await refreshDirectory();
      setTree({});
      setExpanded(new Set());
      feedback.setMessage(`已上传 ${completed} 个文件`);
    });
  }
  function contextMenu(event: ReactMouseEvent, entry?: FileEntry) {
    event.preventDefault();
    if (busy || unavailable) return;
    const items = entry
      ? selection.has(entry.path)
        ? selectedEntries
        : [entry]
      : [];
    if (entry && !selection.has(entry.path))
      setSelection(new Set([entry.path]));
    const anchor = event.currentTarget.getBoundingClientRect();
    setContext({
      x:
        event.detail === 0 && event.type === "click"
          ? anchor.left
          : event.clientX,
      y:
        event.detail === 0 && event.type === "click"
          ? anchor.bottom
          : event.clientY,
      entries: items,
    });
  }
  function keyDown(event: ReactKeyboardEvent) {
    if (
      dialog ||
      busy ||
      context ||
      (event.target as HTMLElement).closest(
        'input, textarea, select, [role="menu"], [role="radiogroup"], [role="combobox"]',
      )
    )
      return;
    if (
      event.key === "Enter" &&
      (event.target as HTMLElement).closest("button") &&
      !(event.target as HTMLElement).closest(".fe-name-cell, .fe-grid-name")
    )
      return;
    const modifier = event.ctrlKey || event.metaKey;
    if (modifier && event.key.toLowerCase() === "a") {
      event.preventDefault();
      setSelection(new Set(filtered.map((entry) => entry.path)));
    } else if (
      modifier &&
      event.key.toLowerCase() === "c" &&
      selectedEntries.length
    ) {
      event.preventDefault();
      copy(selectedEntries, false);
    } else if (
      modifier &&
      event.key.toLowerCase() === "x" &&
      selectedEntries.length &&
      !readOnly
    ) {
      event.preventDefault();
      copy(selectedEntries, true);
    } else if (
      modifier &&
      event.key.toLowerCase() === "v" &&
      clipboard &&
      !readOnly
    ) {
      event.preventDefault();
      void paste();
    } else if (modifier && event.key.toLowerCase() === "l") {
      event.preventDefault();
      setEditingAddress(true);
      requestAnimationFrame(() => pathInput.current?.select());
    } else if (modifier && event.key.toLowerCase() === "f") {
      event.preventDefault();
      searchInput.current?.focus();
    } else if (
      event.key === "F2" &&
      selectedEntries.length === 1 &&
      !readOnly
    ) {
      event.preventDefault();
      openDialog({ type: "rename", entry: selectedEntries[0] });
    } else if (event.key === "Delete" && selectedEntries.length && !readOnly) {
      event.preventDefault();
      openDialog({ type: "delete", entries: selectedEntries });
    } else if (event.key === "Enter" && selectedEntries.length === 1) {
      event.preventDefault();
      void openEntry(selectedEntries[0]);
    } else if (
      (event.altKey && event.key === "ArrowUp") ||
      event.key === "Backspace"
    ) {
      event.preventDefault();
      goUp();
    } else if (event.key === "Escape") setSelection(new Set());
  }
  async function toggleTree(path: string) {
    if (expanded.has(path)) {
      setExpanded((previous) => {
        const next = new Set(previous);
        next.delete(path);
        return next;
      });
      return;
    }
    setExpanded((previous) => new Set([...previous, path]));
    if (tree[path]) return;
    setTree((previous) => ({
      ...previous,
      [path]: { entries: [], loading: true },
    }));
    try {
      const data = await api.files.list("server", path);
      setTree((previous) => ({
        ...previous,
        [path]: {
          entries: data.entries.filter((entry) => entry.dir),
          loading: false,
        },
      }));
    } catch (error) {
      setTree((previous) => ({
        ...previous,
        [path]: {
          entries: [],
          loading: false,
          error: error instanceof Error ? error.message : "读取失败",
        },
      }));
    }
  }
  function treeNode(entry: FileEntry, depth = 0): React.ReactNode {
    if (depth > 6 || (!showHidden && entry.name.startsWith("."))) return null;
    const isExpanded = expanded.has(entry.path),
      data = tree[entry.path];
    return (
      <div key={entry.path}>
        <div
          className={`fe-tree-row${current.root === "server" && current.path === entry.path ? " fe-place-active" : ""}`}
          style={{ paddingLeft: 10 + depth * 13 }}
        >
          <button
            className="fe-tree-toggle"
            aria-label={`${isExpanded ? "收起" : "展开"} ${entry.name}`}
            aria-expanded={isExpanded}
            onClick={() => void toggleTree(entry.path)}
          >
            <Icon name={isExpanded ? "down" : "chevron"} size={11} />
          </button>
          <button
            className="fe-tree-folder"
            title={entry.path}
            disabled={busy}
            onClick={() => navigate({ root: "server", path: entry.path })}
          >
            <Icon name="folder" size={15} />
            <span>{entry.name}</span>
          </button>
        </div>
        {isExpanded && (
          <div>
            {data?.loading ? (
              <p className="fe-tree-note">正在读取…</p>
            ) : data?.error ? (
              <p className="fe-tree-note" title={data.error}>
                无法读取此目录
              </p>
            ) : data?.entries.length ? (
              data.entries.map((item) => treeNode(item, depth + 1))
            ) : (
              <p className="fe-tree-note">没有子目录</p>
            )}
          </div>
        )}
      </div>
    );
  }
  function changeSort(key: SortKey) {
    setSort((previous) => ({
      key,
      ascending: previous.key === key ? !previous.ascending : true,
    }));
  }
  const contextItems = context?.entries || [];
  const contextActions: MenuAction[] = contextItems.length
    ? [
        {
          label: contextItems[0].dir ? "打开文件夹" : "编辑文本",
          icon: contextItems[0].dir ? "folder" : "edit",
          shortcut: "Enter",
          disabled: contextItems.length !== 1,
          onSelect: () => {
            void openEntry(contextItems[0]);
          },
        },
        {
          label: "剪切",
          icon: "cut",
          shortcut: "Ctrl+X",
          separator: true,
          disabled: readOnly,
          onSelect: () => copy(contextItems, true),
        },
        {
          label: "复制",
          icon: "copy",
          shortcut: "Ctrl+C",
          onSelect: () => copy(contextItems, false),
        },
        {
          label: "粘贴",
          icon: "paste",
          shortcut: "Ctrl+V",
          disabled: readOnly || !clipboard,
          onSelect: () => {
            void paste();
          },
        },
        {
          label: "重命名",
          icon: "edit",
          shortcut: "F2",
          disabled: readOnly || contextItems.length !== 1,
          onSelect: () =>
            openDialog({ type: "rename", entry: contextItems[0] }),
        },
        {
          label: "下载",
          icon: "download",
          separator: true,
          disabled: contextItems.some(
            (entry) => entry.dir || entry.type === "special",
          ),
          onSelect: () => download(contextItems),
        },
        {
          label: "压缩为 ZIP",
          icon: "archive",
          disabled: readOnly,
          onSelect: () =>
            openDialog({ type: "archive", entries: contextItems }),
        },
        ...(contextItems.length === 1 && /\.zip$/i.test(contextItems[0].name)
          ? [
              {
                label: "解压到…",
                icon: "folder" as const,
                disabled: readOnly,
                onSelect: () =>
                  openDialog({ type: "extract", entry: contextItems[0] }),
              },
            ]
          : []),
        {
          label: "修改权限",
          icon: "shield",
          separator: true,
          disabled:
            readOnly || contextItems.length !== 1 || contextItems[0]?.symlink,
          onSelect: () =>
            openDialog({ type: "permissions", entry: contextItems[0] }),
        },
        {
          label: "删除",
          icon: "trash",
          shortcut: "Delete",
          danger: true,
          disabled: readOnly,
          onSelect: () => openDialog({ type: "delete", entries: contextItems }),
        },
        {
          label: "属性",
          icon: "info",
          disabled: contextItems.length !== 1,
          onSelect: () =>
            openDialog({ type: "properties", entry: contextItems[0] }),
        },
      ]
    : [
        {
          label: "刷新",
          icon: "refresh",
          onSelect: () => setReload((version) => version + 1),
        },
        {
          label: "粘贴",
          icon: "paste",
          shortcut: "Ctrl+V",
          disabled: readOnly || !clipboard,
          onSelect: () => {
            void paste();
          },
        },
        {
          label: "新建文件夹",
          icon: "folder",
          separator: true,
          disabled: readOnly,
          onSelect: () => openDialog({ type: "create", directory: true }),
        },
        {
          label: "新建文件",
          icon: "fileText",
          disabled: readOnly,
          onSelect: () => openDialog({ type: "create", directory: false }),
        },
        {
          label: "上传文件",
          icon: "upload",
          disabled: readOnly,
          onSelect: () => openDialog({ type: "upload" }),
        },
      ];

  const allSelected =
    filtered.length > 0 && filtered.every((entry) => selection.has(entry.path));
  const partialSelected =
    !allSelected && filtered.some((entry) => selection.has(entry.path));
  const addressBase =
    current.root === "server" ? serverBase : scope?.path || "/";
  const breadcrumbs = (
    current.root === "server"
      ? absolutePath.slice(serverBase === "/" ? 1 : serverBase.length + 1)
      : current.path
  )
    .split("/")
    .filter(Boolean);
  const modalTitle =
    dialog?.type === "create"
      ? dialog.directory
        ? "新建文件夹"
        : "新建文件"
      : dialog?.type === "rename"
        ? "重命名"
        : dialog?.type === "permissions"
          ? "修改权限"
          : dialog?.type === "properties"
            ? "文件属性"
            : dialog?.type === "delete"
              ? "删除所选项目"
              : dialog?.type === "archive"
                ? "压缩为 ZIP"
                : dialog?.type === "extract"
                  ? "解压 ZIP"
                  : dialog?.type === "editor"
                    ? dialog.entry.name
                    : "上传文件";

  return (
    <section
      className="fe-page"
      aria-label="服务器文件管理"
      onKeyDown={keyDown}
    >
      {feedback.error && !dialog && (
        <Alert tone="error" onDismiss={() => feedback.setError("")}>
          {feedback.error}
        </Alert>
      )}
      {feedback.message && (
        <Alert tone="success" onDismiss={() => feedback.setMessage("")}>
          {feedback.message}
        </Alert>
      )}
      <div className="fe-explorer" aria-busy={busy || loading}>
        <div className="fe-address-row">
          <div className="fe-navigation">
            <Tool
              icon="back"
              title="后退"
              onClick={() => historyGo(historyIndex - 1)}
              disabled={historyIndex === 0 || busy}
            />
            <Tool
              icon="chevron"
              title="前进"
              onClick={() => historyGo(historyIndex + 1)}
              disabled={historyIndex === history.length - 1 || busy}
            />
            <Tool
              icon="up"
              title="上一级"
              onClick={goUp}
              disabled={atRoot || unavailable || busy}
            />
            <Tool
              icon="refresh"
              title="刷新目录"
              onClick={() => setReload((version) => version + 1)}
              disabled={loading || busy}
            />
          </div>
          <div
            className={`fe-address${editingAddress ? " fe-address-editing" : ""}`}
          >
            <Icon name="drive" size={16} />
            {editingAddress ? (
              <form
                onSubmit={(event) => {
                  event.preventDefault();
                  const target = address.trim();
                  if (!target) {
                    setEditingAddress(false);
                    return;
                  }
                  if (target.startsWith("/"))
                    navigate({ root: "server", path: target });
                  else
                    navigate({
                      root: current.root,
                      path: joinPath(current.path, target),
                    });
                }}
              >
                <input
                  ref={pathInput}
                  autoFocus
                  aria-label="服务器路径"
                  value={address}
                  onChange={(event) => setAddress(event.target.value)}
                  onBlur={(event) => {
                    if (
                      event.currentTarget.parentElement?.contains(
                        event.relatedTarget as Node,
                      )
                    )
                      return;
                    setAddress(absolutePath);
                    setEditingAddress(false);
                  }}
                  onKeyDown={(event) => {
                    if (event.key === "Escape") {
                      setAddress(absolutePath);
                      setEditingAddress(false);
                    }
                  }}
                />
                <button type="submit" aria-label="打开路径">
                  <Icon name="arrow" size={15} />
                </button>
              </form>
            ) : (
              <>
                <div className="fe-breadcrumbs">
                  <button
                    onClick={() =>
                      navigate({
                        root: current.root,
                        path: current.root === "server" ? serverBase : "",
                      })
                    }
                    disabled={busy}
                    title={addressBase}
                  >
                    {addressBase}
                  </button>
                  {breadcrumbs.map((part, index) => {
                    const relative = breadcrumbs.slice(0, index + 1).join("/");
                    const target =
                      current.root === "server"
                        ? joinPath(serverBase, relative)
                        : relative;
                    return (
                      <span key={target}>
                        <Icon name="chevron" size={11} />
                        <button
                          title={target}
                          disabled={busy}
                          onClick={() =>
                            navigate({ root: current.root, path: target })
                          }
                        >
                          {part}
                        </button>
                      </span>
                    );
                  })}
                </div>
                <button
                  className="fe-edit-address"
                  aria-label="编辑路径"
                  title="编辑路径"
                  onClick={() => setEditingAddress(true)}
                >
                  <Icon name="edit" size={13} />
                </button>
              </>
            )}
          </div>
          <SearchInput
            className="fe-search"
            maxWidth={260}
            ref={searchInput}
            aria-label="搜索当前目录"
            placeholder="搜索当前目录"
            value={search}
            onChange={(event) => onSearchChange(event.target.value)}
          />
        </div>
        <div className="fe-toolbar">
          <div className="fe-toolbar-group">
            <Button
              variant="primary"
              size="sm"
              disabled={readOnly || unavailable || busy}
              onClick={() => {
                setUploads([]);
                openDialog({ type: "upload" });
              }}
            >
              <Icon name="upload" size={15} />
              上传
            </Button>
            <DropdownMenu
              label="新建文件或文件夹"
              trigger={
                <Button size="sm" disabled={readOnly || unavailable || busy}>
                  <Icon name="plus" size={15} />
                  新建
                  <Icon name="down" size={13} />
                </Button>
              }
              items={[
                {
                  label: "文件夹",
                  icon: "folder",
                  onSelect: () =>
                    openDialog({ type: "create", directory: true }),
                },
                {
                  label: "空白文件",
                  icon: "fileText",
                  onSelect: () =>
                    openDialog({ type: "create", directory: false }),
                },
              ]}
            />
          </div>
          <span className="fe-toolbar-divider" />
          <div className="fe-toolbar-group">
            <Tool
              icon="cut"
              title="剪切 (Ctrl+X)"
              disabled={readOnly || !selection.size || busy}
              onClick={() => copy(selectedEntries, true)}
            />
            <Tool
              icon="copy"
              title="复制 (Ctrl+C)"
              disabled={!selection.size || busy}
              onClick={() => copy(selectedEntries, false)}
            />
            <Tool
              icon="paste"
              title="粘贴 (Ctrl+V)"
              disabled={readOnly || !clipboard || busy || unavailable}
              onClick={() => void paste()}
            >
              粘贴
              {clipboard && (
                <span className="fe-count">{clipboard.entries.length}</span>
              )}
            </Tool>
            <Tool
              icon="edit"
              title="重命名 (F2)"
              disabled={readOnly || selection.size !== 1 || busy}
              onClick={() =>
                openDialog({ type: "rename", entry: selectedEntries[0] })
              }
            />
            <Tool
              icon="archive"
              title="压缩为 ZIP"
              disabled={readOnly || !selection.size || busy}
              onClick={() =>
                openDialog({ type: "archive", entries: selectedEntries })
              }
            />
            <Tool
              icon="download"
              title="下载文件"
              disabled={
                !selection.size ||
                selectedEntries.some(
                  (entry) => entry.dir || entry.type === "special",
                ) ||
                busy
              }
              onClick={() => download(selectedEntries)}
            />
            <Tool
              icon="trash"
              title="删除 (Delete)"
              danger
              disabled={readOnly || !selection.size || busy}
              onClick={() =>
                openDialog({ type: "delete", entries: selectedEntries })
              }
            />
          </div>
          <div className="fe-view-tools">
            <Switch
              className="fe-hidden"
              label="显示隐藏文件"
              checked={showHidden}
              onChange={(event) => setShowHidden(event.target.checked)}
            />
            <span className="fe-toolbar-divider" />
            <SegmentedControl
              label="文件视图"
              value={view}
              onChange={setView}
              options={[
                {
                  value: "list",
                  label: <span className="ui-sr-only">详细信息视图</span>,
                  icon: <Icon name="list" size={16} />,
                },
                {
                  value: "grid",
                  label: <span className="ui-sr-only">图标视图</span>,
                  icon: <Icon name="grid" size={16} />,
                },
              ]}
            />
          </div>
        </div>
        <div className="fe-body">
          <aside className="fe-sidebar" aria-label="目录导航">
            <p className="fe-sidebar-label">快速访问</p>
            <button
              className={`fe-place${current.root === "server" && atRoot ? " fe-place-active" : ""}`}
              disabled={!meta || busy}
              onClick={() => navigate({ root: "server", path: serverBase })}
            >
              <Icon name="drive" size={16} />
              <span>服务器根目录</span>
            </button>
            <button
              className="fe-place"
              disabled={!meta || busy}
              onClick={() =>
                navigate({ root: "server", path: meta?.home || serverBase })
              }
            >
              <Icon name="home" size={16} />
              <span>用户目录</span>
            </button>
            {meta?.roots
              .filter((root) => root.name !== "server")
              .map((root) => (
                <button
                  key={root.name}
                  className={`fe-place${current.root === root.name ? " fe-place-active" : ""}`}
                  disabled={busy}
                  title={root.path}
                  onClick={() => navigate({ root: root.name, path: "" })}
                >
                  <Icon
                    name={
                      root.name === "backups"
                        ? "archive"
                        : root.name === "sites"
                          ? "globe"
                          : "box"
                    }
                    size={16}
                  />
                  <span>{root.label}</span>
                </button>
              ))}
            <p className="fe-sidebar-label fe-tree-label">
              目录树<span>{serverBase}</span>
            </p>
            <div className="fe-tree">
              {tree[serverBase]?.entries.map((entry) => treeNode(entry))}
            </div>
            <div className="fe-server-meta">
              <span className="fe-server-dot" />
              <span>{meta?.hostname || "服务器文件系统"}</span>
              <small>{readOnly ? "只读访问" : "按系统权限访问"}</small>
            </div>
          </aside>
          <main
            className={`fe-main${dragging ? " fe-dragging" : ""}`}
            tabIndex={0}
            aria-label="文件列表"
            onContextMenu={(event) => contextMenu(event)}
            onDragEnter={(event) => {
              if (
                !event.dataTransfer.types.includes("Files") ||
                readOnly ||
                unavailable
              )
                return;
              event.preventDefault();
              dragDepth.current++;
              setDragging(true);
            }}
            onDragOver={(event) => {
              if (event.dataTransfer.types.includes("Files") && !readOnly) {
                event.preventDefault();
                event.dataTransfer.dropEffect = "copy";
              }
            }}
            onDragLeave={(event) => {
              event.preventDefault();
              dragDepth.current--;
              if (dragDepth.current <= 0) {
                dragDepth.current = 0;
                setDragging(false);
              }
            }}
            onDrop={(event) => {
              event.preventDefault();
              dragDepth.current = 0;
              setDragging(false);
              addUploads([...event.dataTransfer.files]);
            }}
          >
            {loading ? (
              <div className="fe-loading">
                <span className="fe-spinner" />
                正在读取服务器目录…
              </div>
            ) : loadError ? (
              <div className="fe-empty">
                <span className="fe-empty-glyph">
                  <Icon name="alertCircle" size={28} />
                </span>
                <h2>无法打开此位置</h2>
                <p>{loadError}</p>
                <Button
                  size="sm"
                  onClick={() => setReload((version) => version + 1)}
                >
                  重新读取
                </Button>
              </div>
            ) : filtered.length ? (
              view === "list" ? (
                <div className="fe-table-scroll">
                  <table className="fe-table">
                    <thead>
                      <tr>
                        <th className="fe-select-cell">
                          <Checkbox
                            aria-label="选择当前目录全部项目"
                            checked={allSelected}
                            indeterminate={partialSelected}
                            onChange={() =>
                              setSelection(
                                allSelected
                                  ? new Set()
                                  : new Set(
                                      filtered.map((entry) => entry.path),
                                    ),
                              )
                            }
                          />
                        </th>
                        {(
                          ["name", "modified", "type", "size"] as SortKey[]
                        ).map((key, index) => (
                          <th
                            key={key}
                            aria-sort={
                              sort.key === key
                                ? sort.ascending
                                  ? "ascending"
                                  : "descending"
                                : "none"
                            }
                          >
                            <button onClick={() => changeSort(key)}>
                              {["名称", "修改时间", "类型", "大小"][index]}
                              {sort.key === key && (
                                <Icon
                                  name={sort.ascending ? "up" : "down"}
                                  size={11}
                                />
                              )}
                            </button>
                          </th>
                        ))}
                        <th>权限</th>
                        <th>所有者 / 用户组</th>
                        <th className="fe-more-cell" />
                      </tr>
                    </thead>
                    <tbody>
                      {filtered.map((entry) => (
                        <tr
                          key={entry.path}
                          className={`${selection.has(entry.path) ? "fe-row-selected" : ""} ${clipboard?.move && clipboard.entries.some((item) => item.path === entry.path) ? "fe-row-cut" : ""}`}
                          aria-selected={selection.has(entry.path)}
                          onClick={(event) => select(entry, event)}
                          onDoubleClick={() => void openEntry(entry)}
                          onContextMenu={(event) => {
                            event.stopPropagation();
                            contextMenu(event, entry);
                          }}
                        >
                          <td className="fe-select-cell">
                            <Checkbox
                              aria-label={`选择 ${entry.name}`}
                              checked={selection.has(entry.path)}
                              onClick={(event) => event.stopPropagation()}
                              onChange={() => select(entry, undefined, true)}
                            />
                          </td>
                          <td className="fe-name-cell">
                            <div>
                              <FileGlyph entry={entry} />
                              <button
                                type="button"
                                title={
                                  entry.symlink
                                    ? `${entry.name} → ${entry.link_target}`
                                    : entry.name
                                }
                                onClick={(event) => {
                                  event.stopPropagation();
                                  select(entry, event);
                                }}
                                onDoubleClick={(event) => {
                                  event.stopPropagation();
                                  void openEntry(entry);
                                }}
                              >
                                {entry.name}
                              </button>
                            </div>
                          </td>
                          <td className="fe-muted fe-date-cell">
                            {opsDate(entry.modified)}
                          </td>
                          <td className="fe-muted">{fileKind(entry).label}</td>
                          <td className="fe-muted fe-size-cell">
                            {entry.dir ? "—" : fileSize(entry.size)}
                          </td>
                          <td>
                            <button
                              className="fe-permission"
                              disabled={readOnly || busy || entry.symlink}
                              title={
                                entry.symlink
                                  ? "请修改链接目标的权限"
                                  : entry.mode
                              }
                              onClick={(event) => {
                                event.stopPropagation();
                                openDialog({ type: "permissions", entry });
                              }}
                            >
                              {entry.permissions}
                            </button>
                          </td>
                          <td
                            className="fe-muted fe-owner-cell"
                            title={`${entry.owner} / ${entry.group}`}
                          >
                            {entry.owner}
                            <span> / {entry.group}</span>
                          </td>
                          <td className="fe-more-cell">
                            <IconButton
                              aria-label={`${entry.name} 操作菜单`}
                              onClick={(event) => {
                                event.stopPropagation();
                                contextMenu(event, entry);
                              }}
                            >
                              <Icon name="more" size={18} />
                            </IconButton>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="fe-grid">
                  {filtered.map((entry) => (
                    <div
                      key={entry.path}
                      className={`fe-grid-item${selection.has(entry.path) ? " fe-row-selected" : ""}`}
                      onClick={(event) => select(entry, event)}
                      onDoubleClick={() => void openEntry(entry)}
                      onContextMenu={(event) => {
                        event.stopPropagation();
                        contextMenu(event, entry);
                      }}
                    >
                      <Checkbox
                        aria-label={`选择 ${entry.name}`}
                        checked={selection.has(entry.path)}
                        onClick={(event) => event.stopPropagation()}
                        onChange={() => select(entry, undefined, true)}
                      />
                      <FileGlyph entry={entry} large />
                      <button
                        className="fe-grid-name"
                        onClick={(event) => {
                          event.stopPropagation();
                          select(entry, event);
                        }}
                        onDoubleClick={(event) => {
                          event.stopPropagation();
                          void openEntry(entry);
                        }}
                        title={entry.name}
                      >
                        {entry.name}
                      </button>
                      <small>
                        {entry.dir ? "文件夹" : fileSize(entry.size)}
                      </small>
                    </div>
                  ))}
                </div>
              )
            ) : (
              <div className="fe-empty">
                <span className="fe-empty-glyph">
                  <Icon name={search ? "search" : "folder"} size={30} />
                </span>
                <h2>{search ? "没有匹配的项目" : "此文件夹为空"}</h2>
                <p>
                  {search
                    ? "尝试其他文件名，或清空搜索条件。"
                    : readOnly
                      ? "当前目录没有可显示的文件。"
                      : "拖入文件上传，或通过工具栏新建文件夹。"}
                </p>
                {search && (
                  <Button size="sm" onClick={() => onSearchChange("")}>
                    清空搜索
                  </Button>
                )}
              </div>
            )}
            {dragging && (
              <div className="fe-drop-overlay">
                <Icon name="upload" size={36} />
                <strong>释放以上传到当前目录</strong>
                <span>{absolutePath}</span>
              </div>
            )}
          </main>
        </div>
        <footer className="fe-status">
          <span>
            {loading ? "正在读取…" : `${filtered.length} 个项目`}
            {!loading &&
              filtered.length !== entries.length &&
              `（共 ${entries.length} 个）`}
            {selection.size > 0 && (
              <>
                <span className="fe-status-divider">|</span>已选择{" "}
                {selection.size} 项
                {selectedEntries.every((entry) => !entry.dir) &&
                  ` · ${fileSize(selectedEntries.reduce((total, entry) => total + entry.size, 0))}`}
              </>
            )}
          </span>
          <span>
            {busy && (
              <>
                <span className="fe-spinner" />
                正在处理<span className="fe-status-divider">|</span>
              </>
            )}
            {clipboard && (
              <>
                <Icon name={clipboard.move ? "cut" : "copy"} size={12} />
                {clipboard.entries.length} 项待粘贴
                <span className="fe-status-divider">|</span>
              </>
            )}
            <Icon name="server" size={12} />
            {readOnly ? "只读" : "服务器文件系统"}
          </span>
        </footer>
      </div>
      {context && (
        <ContextMenu
          x={context.x}
          y={context.y}
          label="文件操作"
          onClose={() => setContext(null)}
          items={contextActions}
        />
      )}
      {dialog && (
        <Modal
          structured
          title={modalTitle}
          wide={dialog.type === "editor" || dialog.type === "upload"}
          onClose={closeDialog}
        >
          {feedback.error && (
            <Alert className="fe-modal-notice" tone="error">
              {feedback.error}
            </Alert>
          )}
          {dialog.type === "editor" ? (
            <>
              <div className="fe-editor-bar">
                <span title={dialog.entry.path}>
                  <Icon name="fileText" size={14} />
                  {dialog.entry.path}
                </span>
                <label>
                  <Checkbox
                    checked={editorWrap}
                    onChange={(event) => setEditorWrap(event.target.checked)}
                  />
                  自动换行
                </label>
                <span>{editor !== dialog.original ? "未保存" : "已保存"}</span>
              </div>
              <textarea
                autoFocus
                data-autofocus
                className={`fe-editor${editorWrap ? " fe-editor-wrap" : ""}`}
                aria-label="文件内容"
                spellCheck={false}
                value={editor}
                readOnly={readOnly || busy}
                onChange={(event) => setEditor(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === "Tab" && !readOnly && !busy) {
                    event.preventDefault();
                    const textarea = event.currentTarget;
                    const start = textarea.selectionStart,
                      end = textarea.selectionEnd;
                    setEditor(
                      (previous) =>
                        previous.slice(0, start) + "  " + previous.slice(end),
                    );
                    requestAnimationFrame(() => {
                      textarea.setSelectionRange(start + 2, start + 2);
                    });
                  }
                  if (
                    (event.ctrlKey || event.metaKey) &&
                    event.key.toLowerCase() === "s"
                  ) {
                    event.preventDefault();
                    if (!readOnly && !busy)
                      void feedback.run(async () => {
                        await api.files.write(
                          current.root,
                          dialog.entry.path,
                          editor,
                        );
                        setDialog({ ...dialog, original: editor });
                        await refreshDirectory();
                      }, "文件已保存");
                  }
                }}
              />
              <ModalFooter>
                <span className="fe-muted">
                  UTF-8 · {editor.split("\n").length} 行 ·{" "}
                  {fileSize(new Blob([editor]).size)}
                </span>
                <div>
                  <Button size="sm" disabled={busy} onClick={closeDialog}>
                    关闭
                  </Button>
                  <Button
                    variant="primary"
                    size="sm"
                    disabled={readOnly || busy || editor === dialog.original}
                    onClick={() =>
                      void feedback.run(async () => {
                        await api.files.write(
                          current.root,
                          dialog.entry.path,
                          editor,
                        );
                        setDialog({ ...dialog, original: editor });
                        await refreshDirectory();
                      }, "文件已保存")
                    }
                  >
                    <Icon name="check" size={14} />
                    {busy ? "保存中…" : "保存文件"}
                  </Button>
                </div>
              </ModalFooter>
            </>
          ) : dialog.type === "upload" ? (
            <>
              <div className="fe-dialog-body">
                <p className="fe-muted">
                  上传到 <strong>{absolutePath}</strong>
                </p>
                <button
                  className="fe-upload-zone"
                  disabled={busy}
                  onClick={() => uploadInput.current?.click()}
                >
                  <Icon name="upload" size={30} />
                  <strong>选择要上传的文件</strong>
                  <span>支持多文件和二进制文件，单文件最大 256 MB</span>
                </button>
                <input
                  ref={uploadInput}
                  type="file"
                  multiple
                  hidden
                  onChange={(event) => {
                    if (event.target.files) addUploads([...event.target.files]);
                    event.target.value = "";
                  }}
                />
                <div className="fe-upload-list">
                  {uploads.map((item, index) => (
                    <div
                      className="fe-upload-item"
                      key={`${item.file.name}-${index}`}
                    >
                      <Icon name="fileText" size={17} />
                      <div>
                        <div className="fe-upload-item-title">
                          <strong>{item.file.name}</strong>
                          <span>{fileSize(item.file.size)}</span>
                        </div>
                        <div className="fe-upload-progress">
                          <span style={{ width: `${item.progress}%` }} />
                        </div>
                        {item.error && (
                          <small className="fe-danger">{item.error}</small>
                        )}
                      </div>
                      <span
                        className={`fe-upload-state${item.status === "done" ? " fe-upload-done" : ""}`}
                      >
                        {item.status === "done" ? (
                          <Icon name="check" size={16} />
                        ) : item.status === "uploading" ? (
                          `${item.progress}%`
                        ) : item.status === "error" ? (
                          "失败"
                        ) : (
                          "待上传"
                        )}
                      </span>
                      {!busy && (
                        <button
                          aria-label={`移除 ${item.file.name}`}
                          onClick={() =>
                            setUploads((previous) =>
                              previous.filter(
                                (_, position) => position !== index,
                              ),
                            )
                          }
                        >
                          <Icon name="close" size={13} />
                        </button>
                      )}
                    </div>
                  ))}
                </div>
                <label className="fe-checkbox">
                  <Checkbox
                    disabled={busy}
                    checked={overwrite}
                    onChange={(event) => setOverwrite(event.target.checked)}
                  />
                  覆盖服务器上的同名文件
                </label>
              </div>
              <ModalFooter>
                <span className="fe-muted">{uploads.length} 个文件</span>
                <div>
                  <Button size="sm" disabled={busy} onClick={closeDialog}>
                    关闭
                  </Button>
                  <Button
                    variant="primary"
                    size="sm"
                    disabled={
                      busy || !uploads.some((item) => item.status !== "done")
                    }
                    onClick={() => void startUploads()}
                  >
                    <Icon name="upload" size={14} />
                    {busy ? "上传中…" : "开始上传"}
                  </Button>
                </div>
              </ModalFooter>
            </>
          ) : dialog.type === "properties" ? (
            <>
              <div className="fe-dialog-body">
                <div className="fe-properties-title">
                  <FileGlyph entry={dialog.entry} large />
                  <strong>{dialog.entry.name}</strong>
                </div>
                <dl className="fe-properties">
                  {[
                    ["位置", dialog.entry.path],
                    ["类型", fileKind(dialog.entry).label],
                    [
                      "大小",
                      dialog.entry.dir
                        ? "目录"
                        : `${fileSize(dialog.entry.size)} (${dialog.entry.size.toLocaleString()} 字节)`,
                    ],
                    ["修改时间", opsDate(dialog.entry.modified)],
                    [
                      "权限",
                      `${dialog.entry.permissions} · ${dialog.entry.mode}`,
                    ],
                    [
                      "所有者 / 用户组",
                      `${dialog.entry.owner} / ${dialog.entry.group}`,
                    ],
                    ...(dialog.entry.symlink
                      ? [["链接目标", dialog.entry.link_target || "—"]]
                      : []),
                  ].map(([label, text]) => (
                    <div key={label}>
                      <dt>{label}</dt>
                      <dd>{text}</dd>
                    </div>
                  ))}
                </dl>
              </div>
              <ModalFooter>
                <span />
                <Button size="sm" onClick={closeDialog}>
                  关闭
                </Button>
              </ModalFooter>
            </>
          ) : (
            <form
              onSubmit={(event) => {
                event.preventDefault();
                if (dialog.type === "create")
                  void perform(
                    () =>
                      api.files.create(
                        current.root,
                        joinPath(current.path, value),
                        dialog.directory,
                      ),
                    dialog.directory ? "文件夹已创建" : "文件已创建",
                  );
                else if (dialog.type === "rename")
                  void perform(
                    () =>
                      api.files.rename(
                        current.root,
                        dialog.entry.path,
                        joinPath(parentPath(dialog.entry.path), value),
                      ),
                    "已重命名",
                  );
                else if (dialog.type === "permissions")
                  void perform(
                    () =>
                      api.files.chmod(current.root, dialog.entry.path, value),
                    "文件权限已更新",
                  );
                else if (dialog.type === "delete")
                  void perform(
                    async () =>
                      checkBatch(
                        await api.files.removeMany(
                          current.root,
                          dialog.entries.map((entry) => entry.path),
                        ),
                      ),
                    "所选项目已删除",
                  );
                else if (dialog.type === "archive")
                  void perform(
                    () =>
                      api.files.archive(
                        current.root,
                        dialog.entries.map((entry) => entry.path),
                        joinPath(current.path, value),
                      ),
                    "ZIP 压缩包已创建",
                  );
                else if (dialog.type === "extract")
                  void perform(
                    () =>
                      api.files.unzip(current.root, dialog.entry.path, value),
                    "压缩包已解压",
                  );
              }}
            >
              <fieldset className="fe-dialog-body" disabled={busy}>
                {dialog.type === "delete" ? (
                  <>
                    <div className="fe-delete-summary">
                      <span>
                        <Icon name="trash" size={24} />
                      </span>
                      <div>
                        <strong>
                          确认删除 {dialog.entries.length} 个项目？
                        </strong>
                        <p>目录及其内容将一并删除，此操作无法撤销。</p>
                      </div>
                    </div>
                    <div className="fe-delete-list">
                      {dialog.entries.map((entry) => (
                        <div key={entry.path}>
                          <FileGlyph entry={entry} />
                          <span>{entry.name}</span>
                        </div>
                      ))}
                    </div>
                  </>
                ) : (
                  <>
                    <p className="fe-muted">
                      {dialog.type === "create" || dialog.type === "archive"
                        ? absolutePath
                        : dialog.entry.path}
                    </p>
                    <FormField
                      label={
                        dialog.type === "permissions"
                          ? "权限代码"
                          : dialog.type === "extract"
                            ? "目标目录"
                            : "名称"
                      }
                    >
                      <Input
                        autoFocus
                        required
                        value={value}
                        onChange={(event) => setValue(event.target.value)}
                        pattern={
                          dialog.type === "permissions"
                            ? "[0-7]{3,4}"
                            : dialog.type === "extract"
                              ? undefined
                              : "[^/\\\\]+"
                        }
                        placeholder={
                          dialog.type === "permissions"
                            ? "0644"
                            : dialog.type === "create"
                              ? dialog.directory
                                ? "新文件夹"
                                : "untitled.txt"
                              : undefined
                        }
                      />
                    </FormField>
                    {dialog.type === "permissions" && (
                      <>
                        <div className="fe-permission-grid">
                          <span />
                          <span>读取</span>
                          <span>写入</span>
                          <span>执行</span>
                          {["所有者", "用户组", "其他用户"].map(
                            (label, index) => (
                              <div key={label}>
                                <span>{label}</span>
                                {[4, 2, 1].map((bit) => (
                                  <Checkbox
                                    aria-label={`${label}${bit === 4 ? "读取" : bit === 2 ? "写入" : "执行"}`}
                                    key={bit}
                                    checked={
                                      !!(
                                        parseInt(
                                          value.slice(-3)[index] || "0",
                                          8,
                                        ) & bit
                                      )
                                    }
                                    onChange={(event) => {
                                      const digits = value
                                        .slice(-3)
                                        .padStart(3, "0")
                                        .split("");
                                      const original =
                                        parseInt(digits[index], 8) || 0;
                                      digits[index] = String(
                                        event.target.checked
                                          ? original | bit
                                          : original & ~bit,
                                      );
                                      setValue("0" + digits.join(""));
                                    }}
                                  />
                                ))}
                              </div>
                            ),
                          )}
                        </div>
                        <p className="fe-muted">
                          仅修改当前项目，不递归修改子目录。
                        </p>
                      </>
                    )}
                    {dialog.type === "extract" && (
                      <p className="fe-muted">
                        解压不会覆盖已存在的文件，请选择空目录或新目录。
                      </p>
                    )}
                  </>
                )}
              </fieldset>
              <ModalFooter>
                <span className="fe-muted">{busy ? "正在处理…" : ""}</span>
                <div>
                  <Button size="sm" disabled={busy} onClick={closeDialog}>
                    取消
                  </Button>
                  <Button
                    type="submit"
                    size="sm"
                    variant={dialog.type === "delete" ? "danger" : "primary"}
                    disabled={busy}
                  >
                    {dialog.type === "delete" ? "确认删除" : "确认"}
                  </Button>
                </div>
              </ModalFooter>
            </form>
          )}
        </Modal>
      )}
    </section>
  );
}
