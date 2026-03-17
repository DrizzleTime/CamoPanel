import {
  DeleteOutlined,
  EyeOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  SyncOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Drawer,
  Empty,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import { useEffect, useMemo, useState } from "react";
import { useShellHeader } from "../components/ShellHeaderContext";
import { apiRequest, bytesToSize } from "../lib/api";
import type {
  DockerContainer,
  DockerImage,
  DockerNetwork,
  DockerSystemInfo,
  Project,
} from "../lib/types";

type LoadingState = {
  containers: boolean;
  images: boolean;
  projects: boolean;
  networks: boolean;
  system: boolean;
};

type ErrorState = {
  containers?: string;
  images?: string;
  projects?: string;
  networks?: string;
  system?: string;
};

const initialLoadingState: LoadingState = {
  containers: true,
  images: true,
  projects: true,
  networks: true,
  system: true,
};

const CONTAINER_TAB_LABELS = {
  containers: "容器",
  images: "镜像",
  projects: "编排",
  networks: "网络",
  system: "系统设置",
} as const;

export function ContainersPage() {
  const [activeTab, setActiveTab] = useState("containers");
  const [loading, setLoading] = useState<LoadingState>(initialLoadingState);
  const [errors, setErrors] = useState<ErrorState>({});
  const [containers, setContainers] = useState<DockerContainer[]>([]);
  const [images, setImages] = useState<DockerImage[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [networks, setNetworks] = useState<DockerNetwork[]>([]);
  const [systemInfo, setSystemInfo] = useState<DockerSystemInfo | null>(null);
  const [activeContainer, setActiveContainer] = useState<DockerContainer | null>(null);
  const [containerLogs, setContainerLogs] = useState("");
  const [containerLogsLoading, setContainerLogsLoading] = useState(false);
  const [activeProject, setActiveProject] = useState<Project | null>(null);
  const [projectLogs, setProjectLogs] = useState("");
  const [projectLogsLoading, setProjectLogsLoading] = useState(false);

  const updateLoading = (key: keyof LoadingState, value: boolean) => {
    setLoading((state) => ({ ...state, [key]: value }));
  };

  const updateError = (key: keyof ErrorState, value?: string) => {
    setErrors((state) => ({ ...state, [key]: value }));
  };

  const loadContainers = async () => {
    updateLoading("containers", true);
    try {
      const response = await apiRequest<{ items: DockerContainer[] }>("/api/docker/containers");
      setContainers(response.items);
      updateError("containers", undefined);
    } catch (error) {
      updateError("containers", error instanceof Error ? error.message : "加载容器失败");
    } finally {
      updateLoading("containers", false);
    }
  };

  const loadImages = async () => {
    updateLoading("images", true);
    try {
      const response = await apiRequest<{ items: DockerImage[] }>("/api/docker/images");
      setImages(response.items);
      updateError("images", undefined);
    } catch (error) {
      updateError("images", error instanceof Error ? error.message : "加载镜像失败");
    } finally {
      updateLoading("images", false);
    }
  };

  const loadProjects = async () => {
    updateLoading("projects", true);
    try {
      const response = await apiRequest<{ items: Project[] }>("/api/projects");
      setProjects(response.items);
      updateError("projects", undefined);
    } catch (error) {
      updateError("projects", error instanceof Error ? error.message : "加载编排失败");
    } finally {
      updateLoading("projects", false);
    }
  };

  const loadNetworks = async () => {
    updateLoading("networks", true);
    try {
      const response = await apiRequest<{ items: DockerNetwork[] }>("/api/docker/networks");
      setNetworks(response.items);
      updateError("networks", undefined);
    } catch (error) {
      updateError("networks", error instanceof Error ? error.message : "加载网络失败");
    } finally {
      updateLoading("networks", false);
    }
  };

  const loadSystemInfo = async () => {
    updateLoading("system", true);
    try {
      const response = await apiRequest<DockerSystemInfo>("/api/docker/system");
      setSystemInfo(response);
      updateError("system", undefined);
    } catch (error) {
      setSystemInfo(null);
      updateError("system", error instanceof Error ? error.message : "加载系统信息失败");
    } finally {
      updateLoading("system", false);
    }
  };

  const loadAll = async () => {
    await Promise.allSettled([
      loadContainers(),
      loadImages(),
      loadProjects(),
      loadNetworks(),
      loadSystemInfo(),
    ]);
  };

  useEffect(() => {
    void loadAll();
  }, []);

  const headerContent = useMemo(
    () => (
      <div className="shell-header-tabs">
        {[
          { key: "containers", label: `${CONTAINER_TAB_LABELS.containers} (${containers.length})` },
          { key: "images", label: `${CONTAINER_TAB_LABELS.images} (${images.length})` },
          { key: "projects", label: `${CONTAINER_TAB_LABELS.projects} (${projects.length})` },
          { key: "networks", label: `${CONTAINER_TAB_LABELS.networks} (${networks.length})` },
          { key: "system", label: CONTAINER_TAB_LABELS.system },
        ].map((item) => (
          <Button
            key={item.key}
            size="small"
            type={activeTab === item.key ? "primary" : "text"}
            className="shell-header-tab"
            onClick={() => setActiveTab(item.key)}
          >
            {item.label}
          </Button>
        ))}
      </div>
    ),
    [activeTab, containers.length, images.length, networks.length, projects.length],
  );

  useShellHeader(headerContent);

  const runProjectAction = async (project: Project, action: string) => {
    await apiRequest(`/api/projects/${project.id}/actions`, {
      method: "POST",
      body: JSON.stringify({ action }),
    });
    message.success(`${actionLabel(action)}已执行`);
    await Promise.allSettled([loadProjects(), loadContainers(), loadSystemInfo()]);
    if (activeProject?.id === project.id && action !== "delete") {
      await openProjectDetails(project);
    }
    if (action === "delete") {
      setActiveProject(null);
    }
  };

  const openContainerLogs = async (container: DockerContainer) => {
    setActiveContainer(container);
    setContainerLogs("");
    setContainerLogsLoading(true);
    try {
      const response = await apiRequest<{ logs: string }>(
        `/api/docker/containers/${container.id}/logs?tail=200`,
      );
      setContainerLogs(response.logs || "暂无日志");
    } catch {
      setContainerLogs("当前无法读取容器日志，通常是 Docker 不可用或容器已经消失。");
    } finally {
      setContainerLogsLoading(false);
    }
  };

  const openProjectDetails = async (project: Project) => {
    setActiveProject(project);
    setProjectLogs("");
    setProjectLogsLoading(true);
    try {
      const response = await apiRequest<{ logs: string }>(`/api/projects/${project.id}/logs`);
      setProjectLogs(response.logs || "暂无日志");
    } catch {
      setProjectLogs("当前无法读取日志，通常是 Docker 未安装或 daemon 未启动。");
    } finally {
      setProjectLogsLoading(false);
    }
  };

  const tabItems = [
    {
      key: "containers",
      label: `容器 (${containers.length})`,
      children: (
        <div className="containers-tab-stack">
          {renderError(errors.containers)}
          <Card className="glass-card">
            <Table<DockerContainer>
              rowKey="id"
              loading={loading.containers}
              dataSource={containers}
              scroll={{ x: 1080 }}
              locale={{ emptyText: <Empty description="当前没有容器" /> }}
              columns={[
                {
                  title: "容器",
                  dataIndex: "name",
                  render: (_, record) => (
                    <Space direction="vertical" size={2}>
                      <Typography.Text strong>{record.name}</Typography.Text>
                      <Typography.Text type="secondary">{shortId(record.id)}</Typography.Text>
                    </Space>
                  ),
                },
                {
                  title: "镜像",
                  dataIndex: "image",
                  render: (value: string) => (
                    <Typography.Text style={{ wordBreak: "break-all" }}>{value}</Typography.Text>
                  ),
                },
                {
                  title: "状态",
                  render: (_, record) => <Tag color={containerStateColor(record.state)}>{record.state}</Tag>,
                },
                {
                  title: "编排",
                  render: (_, record) =>
                    record.project ? <Tag color="blue">{record.project}</Tag> : <Typography.Text type="secondary">独立容器</Typography.Text>,
                },
                {
                  title: "网络",
                  render: (_, record) => renderTagList(record.networks, "未加入网络"),
                },
                {
                  title: "端口",
                  render: (_, record) => renderTagList(record.ports, "未暴露端口"),
                },
                {
                  title: "动作",
                  render: (_, record) => (
                    <Button size="small" icon={<EyeOutlined />} onClick={() => void openContainerLogs(record)}>
                      日志
                    </Button>
                  ),
                },
              ]}
            />
          </Card>
        </div>
      ),
    },
    {
      key: "images",
      label: `镜像 (${images.length})`,
      children: (
        <div className="containers-tab-stack">
          {renderError(errors.images)}
          <Card className="glass-card">
            <Table<DockerImage>
              rowKey="id"
              loading={loading.images}
              dataSource={images}
              scroll={{ x: 920 }}
              locale={{ emptyText: <Empty description="当前没有镜像" /> }}
              columns={[
                {
                  title: "标签",
                  render: (_, record) => renderImageTags(record.repo_tags),
                },
                {
                  title: "镜像 ID",
                  dataIndex: "id",
                  render: (value: string) => <Typography.Text code>{shortId(value)}</Typography.Text>,
                },
                {
                  title: "大小",
                  dataIndex: "size",
                  render: (value: number) => bytesToSize(value),
                },
                {
                  title: "关联容器",
                  dataIndex: "containers",
                  render: (value: number) => String(Math.max(0, value)),
                },
                {
                  title: "创建时间",
                  dataIndex: "created_at",
                  render: (value: string) => formatDateTime(value),
                },
              ]}
            />
          </Card>
        </div>
      ),
    },
    {
      key: "projects",
      label: `编排 (${projects.length})`,
      children: (
        <div className="containers-tab-stack">
          {renderError(errors.projects)}
          <Alert
            showIcon
            type="info"
            message="这里可以直接执行项目的启动、停止、重启、删除和重新部署。"
          />
          <Card className="glass-card">
            <Table<Project>
              rowKey="id"
              loading={loading.projects}
              dataSource={projects}
              scroll={{ x: 980 }}
              locale={{ emptyText: <Empty description="还没有编排项目" /> }}
              onRow={(record) => ({
                onClick: () => void openProjectDetails(record),
              })}
              columns={[
                { title: "编排", dataIndex: "name" },
                { title: "模板", dataIndex: "template_id" },
                {
                  title: "状态",
                  render: (_, record) => <Tag color={projectStatusColor(record.status)}>{record.status}</Tag>,
                },
                {
                  title: "容器数",
                  render: (_, record) => record.runtime.containers.length,
                },
                {
                  title: "动作",
                  render: (_, record) => (
                    <Space onClick={(event) => event.stopPropagation()}>
                      <Button
                        size="small"
                        icon={<PlayCircleOutlined />}
                        onClick={() => void runProjectAction(record, "start")}
                      />
                      <Button
                        size="small"
                        icon={<PauseCircleOutlined />}
                        onClick={() => void runProjectAction(record, "stop")}
                      />
                      <Button
                        size="small"
                        icon={<SyncOutlined />}
                        onClick={() => void runProjectAction(record, "restart")}
                      />
                      <Button size="small" onClick={() => void runProjectAction(record, "redeploy")}>
                        重新部署
                      </Button>
                      <Button
                        danger
                        size="small"
                        icon={<DeleteOutlined />}
                        onClick={() => void runProjectAction(record, "delete")}
                      />
                    </Space>
                  ),
                },
              ]}
            />
          </Card>
        </div>
      ),
    },
    {
      key: "networks",
      label: `网络 (${networks.length})`,
      children: (
        <div className="containers-tab-stack">
          {renderError(errors.networks)}
          <Card className="glass-card">
            <Table<DockerNetwork>
              rowKey="id"
              loading={loading.networks}
              dataSource={networks}
              scroll={{ x: 900 }}
              locale={{ emptyText: <Empty description="当前没有网络" /> }}
              columns={[
                {
                  title: "网络",
                  render: (_, record) => (
                    <Space direction="vertical" size={2}>
                      <Typography.Text strong>{record.name}</Typography.Text>
                      <Typography.Text type="secondary">{shortId(record.id)}</Typography.Text>
                    </Space>
                  ),
                },
                { title: "Driver", dataIndex: "driver" },
                { title: "Scope", dataIndex: "scope" },
                {
                  title: "容器数",
                  dataIndex: "container_count",
                },
                {
                  title: "特性",
                  render: (_, record) =>
                    renderTagList(
                      [
                        record.internal ? "internal" : "",
                        record.attachable ? "attachable" : "",
                        record.ingress ? "ingress" : "",
                      ].filter(Boolean),
                      "默认",
                    ),
                },
                {
                  title: "创建时间",
                  dataIndex: "created_at",
                  render: (value: string) => formatDateTime(value),
                },
              ]}
            />
          </Card>
        </div>
      ),
    },
    {
      key: "system",
      label: "系统设置",
      children: (
        <div className="containers-tab-stack">
          <Alert showIcon type="info" message="系统设置当前只读，先用于查看 Docker 守护进程和宿主机侧的关键状态。" />
          {renderError(errors.system)}
          <div className="containers-summary-grid">
            <Card className="glass-card">
              <Typography.Text type="secondary">容器总数</Typography.Text>
              <Typography.Title level={2} style={{ margin: "8px 0 0" }}>
                {systemInfo?.containers ?? 0}
              </Typography.Title>
              <Typography.Text type="secondary">
                运行中 {systemInfo?.containers_running ?? 0} / 停止 {systemInfo?.containers_stopped ?? 0}
              </Typography.Text>
            </Card>
            <Card className="glass-card">
              <Typography.Text type="secondary">镜像总数</Typography.Text>
              <Typography.Title level={2} style={{ margin: "8px 0 0" }}>
                {systemInfo?.images ?? 0}
              </Typography.Title>
              <Typography.Text type="secondary">Docker Engine {systemInfo?.server_version ?? "-"}</Typography.Text>
            </Card>
            <Card className="glass-card">
              <Typography.Text type="secondary">默认运行时</Typography.Text>
              <Typography.Title level={2} style={{ margin: "8px 0 0" }}>
                {systemInfo?.default_runtime || "-"}
              </Typography.Title>
              <Typography.Text type="secondary">{systemInfo?.driver || "-"}</Typography.Text>
            </Card>
            <Card className="glass-card">
              <Typography.Text type="secondary">可用内存</Typography.Text>
              <Typography.Title level={2} style={{ margin: "8px 0 0" }}>
                {bytesToSize(systemInfo?.mem_total ?? 0)}
              </Typography.Title>
              <Typography.Text type="secondary">{systemInfo?.ncpu ?? 0} 核 CPU</Typography.Text>
            </Card>
          </div>

          <Card className="glass-card" loading={loading.system}>
            {systemInfo ? (
              <Descriptions bordered column={2} size="small">
                <Descriptions.Item label="主机名">{systemInfo.name}</Descriptions.Item>
                <Descriptions.Item label="Docker ID">{shortId(systemInfo.id)}</Descriptions.Item>
                <Descriptions.Item label="系统">{systemInfo.operating_system}</Descriptions.Item>
                <Descriptions.Item label="内核">{systemInfo.kernel_version}</Descriptions.Item>
                <Descriptions.Item label="架构">{systemInfo.architecture}</Descriptions.Item>
                <Descriptions.Item label="日志驱动">{systemInfo.logging_driver}</Descriptions.Item>
                <Descriptions.Item label="存储驱动">{systemInfo.driver}</Descriptions.Item>
                <Descriptions.Item label="Cgroup">
                  {systemInfo.cgroup_driver}
                  {systemInfo.cgroup_version ? ` / ${systemInfo.cgroup_version}` : ""}
                </Descriptions.Item>
                <Descriptions.Item label="Docker 数据目录" span={2}>
                  {systemInfo.docker_root_dir}
                </Descriptions.Item>
                <Descriptions.Item label="运行时" span={2}>
                  {systemInfo.runtimes.length ? renderTagList(systemInfo.runtimes, "无") : "无"}
                </Descriptions.Item>
                <Descriptions.Item label="网络插件" span={2}>
                  {systemInfo.network_plugins.length ? renderTagList(systemInfo.network_plugins, "无") : "无"}
                </Descriptions.Item>
                <Descriptions.Item label="卷插件" span={2}>
                  {systemInfo.volume_plugins.length ? renderTagList(systemInfo.volume_plugins, "无") : "无"}
                </Descriptions.Item>
              </Descriptions>
            ) : (
              <Empty description="当前无法读取 Docker 系统信息" />
            )}
          </Card>

          {systemInfo?.warnings.length ? (
            <Card className="glass-card" title="Docker 警告">
              <div className="containers-warning-list">
                {systemInfo.warnings.map((warning) => (
                  <Alert key={warning} type="warning" showIcon message={warning} />
                ))}
              </div>
            </Card>
          ) : null}
        </div>
      ),
    },
  ];

  const activeTabContent = tabItems.find((item) => item.key === activeTab)?.children ?? null;

  return (
    <div className="page-grid">
      <div className="page-inline-bar">
        <Typography.Text type="secondary">
          在一个页面内查看容器、镜像、编排、网络和 Docker 系统状态。
        </Typography.Text>
        <Button icon={<ReloadOutlined />} onClick={() => void loadAll()}>
          刷新
        </Button>
      </div>

      {activeTabContent}

      <Drawer
        open={!!activeContainer}
        onClose={() => setActiveContainer(null)}
        size={720}
        title={activeContainer?.name}
        extra={<Tag color={containerStateColor(activeContainer?.state)}>{activeContainer?.state || "unknown"}</Tag>}
      >
        {activeContainer ? (
          <Space direction="vertical" size="large" style={{ width: "100%" }}>
            <Descriptions bordered column={1} size="small">
              <Descriptions.Item label="镜像">{activeContainer.image}</Descriptions.Item>
              <Descriptions.Item label="编排">{activeContainer.project || "独立容器"}</Descriptions.Item>
              <Descriptions.Item label="网络">
                {activeContainer.networks.length ? activeContainer.networks.join(", ") : "无"}
              </Descriptions.Item>
              <Descriptions.Item label="端口">
                {activeContainer.ports.length ? activeContainer.ports.join(", ") : "无"}
              </Descriptions.Item>
            </Descriptions>
            <Card title="最近日志" size="small" loading={containerLogsLoading}>
              <div className="mono-box">{containerLogs}</div>
            </Card>
          </Space>
        ) : null}
      </Drawer>

      <Drawer
        open={!!activeProject}
        onClose={() => setActiveProject(null)}
        size={720}
        title={activeProject?.name}
        extra={<Tag color={projectStatusColor(activeProject?.status)}>{activeProject?.status}</Tag>}
      >
        {activeProject ? (
          <Space direction="vertical" size="large" style={{ width: "100%" }}>
            <Descriptions bordered column={1} size="small">
              <Descriptions.Item label="模板">{activeProject.template_id}</Descriptions.Item>
              <Descriptions.Item label="Compose">{activeProject.compose_path}</Descriptions.Item>
              <Descriptions.Item label="最近错误">{activeProject.last_error || "无"}</Descriptions.Item>
            </Descriptions>
            <Card title="容器状态" size="small">
              {activeProject.runtime.containers.length ? (
                <Space direction="vertical" size="middle" style={{ width: "100%" }}>
                  {activeProject.runtime.containers.map((container) => (
                    <Card key={container.id} size="small">
                      <Space direction="vertical">
                        <Typography.Text strong>{container.name}</Typography.Text>
                        <Typography.Text type="secondary">{container.image}</Typography.Text>
                        <Space wrap>
                          <Tag color={containerStateColor(container.state)}>{container.state}</Tag>
                          {container.ports.map((port) => (
                            <Tag key={port}>{port}</Tag>
                          ))}
                        </Space>
                      </Space>
                    </Card>
                  ))}
                </Space>
              ) : (
                <Empty description="未检测到容器" />
              )}
            </Card>
            <Card title="配置快照" size="small">
              <div className="mono-box">{JSON.stringify(activeProject.config, null, 2)}</div>
            </Card>
            <Card title="最近日志" size="small" loading={projectLogsLoading}>
              <div className="mono-box">{projectLogs}</div>
            </Card>
          </Space>
        ) : null}
      </Drawer>
    </div>
  );
}

function renderError(messageText?: string) {
  if (!messageText) {
    return null;
  }

  return <Alert showIcon type="error" message={messageText} />;
}

function renderTagList(values: string[], emptyText: string) {
  if (!values.length) {
    return <Typography.Text type="secondary">{emptyText}</Typography.Text>;
  }

  return (
    <Space wrap size={[6, 6]}>
      {values.map((value) => (
        <Tag key={value}>{value}</Tag>
      ))}
    </Space>
  );
}

function renderImageTags(tags: string[]) {
  const values = tags.length ? tags : ["<none>:<none>"];
  const visible = values.slice(0, 3);
  const remain = values.length - visible.length;

  return (
    <Space wrap size={[6, 6]}>
      {visible.map((tag) => (
        <Tag key={tag} color={tag === "<none>:<none>" ? "default" : "blue"}>
          {tag}
        </Tag>
      ))}
      {remain > 0 ? <Tag>+{remain}</Tag> : null}
    </Space>
  );
}

function containerStateColor(state?: string) {
  switch (state) {
    case "running":
      return "green";
    case "paused":
      return "gold";
    case "exited":
    case "dead":
      return "red";
    default:
      return "default";
  }
}

function projectStatusColor(status?: string) {
  switch (status) {
    case "running":
      return "green";
    case "stopped":
      return "gold";
    case "degraded":
      return "red";
    default:
      return "default";
  }
}

function actionLabel(action: string) {
  switch (action) {
    case "start":
      return "启动";
    case "stop":
      return "停止";
    case "restart":
      return "重启";
    case "redeploy":
      return "重新部署";
    case "delete":
      return "删除";
    default:
      return action;
  }
}

function formatDateTime(value: string) {
  return new Date(value).toLocaleString("zh-CN", {
    hour12: false,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function shortId(value: string) {
  return value.length > 12 ? value.slice(0, 12) : value;
}
