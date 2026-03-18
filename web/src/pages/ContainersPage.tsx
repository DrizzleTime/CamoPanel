import {
  DeleteOutlined,
  EyeOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  PlusOutlined,
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
  Form,
  Input,
  Modal,
  Popconfirm,
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
  DockerImagePruneResult,
  DockerNetwork,
  DockerSettings,
  DockerSystemInfo,
  Project,
} from "../lib/types";

type LoadingState = {
  containers: boolean;
  images: boolean;
  projects: boolean;
  networks: boolean;
  system: boolean;
  dockerSettings: boolean;
};

type ErrorState = {
  containers?: string;
  images?: string;
  projects?: string;
  networks?: string;
  system?: string;
  dockerSettings?: string;
};

const initialLoadingState: LoadingState = {
  containers: true,
  images: true,
  projects: true,
  networks: true,
  system: true,
  dockerSettings: true,
};

type CustomComposeValues = {
  name: string;
  compose: string;
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
  const [dockerSettings, setDockerSettings] = useState<DockerSettings | null>(null);
  const [activeContainer, setActiveContainer] = useState<DockerContainer | null>(null);
  const [containerLogs, setContainerLogs] = useState("");
  const [containerLogsLoading, setContainerLogsLoading] = useState(false);
  const [containerActionKey, setContainerActionKey] = useState<string | null>(null);
  const [imageActionKey, setImageActionKey] = useState<string | null>(null);
  const [activeProject, setActiveProject] = useState<Project | null>(null);
  const [projectLogs, setProjectLogs] = useState("");
  const [projectLogsLoading, setProjectLogsLoading] = useState(false);
  const [customComposeOpen, setCustomComposeOpen] = useState(false);
  const [customComposeSubmitting, setCustomComposeSubmitting] = useState(false);
  const [dockerSettingsSubmitting, setDockerSettingsSubmitting] = useState(false);
  const [dockerRestarting, setDockerRestarting] = useState(false);
  const [customComposeForm] = Form.useForm<CustomComposeValues>();

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
      return response.items;
    } catch (error) {
      updateError("containers", error instanceof Error ? error.message : "加载容器失败");
      return [];
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

  const loadDockerSettings = async () => {
    updateLoading("dockerSettings", true);
    try {
      const response = await apiRequest<DockerSettings>("/api/docker/settings");
      setDockerSettings(response);
      updateError("dockerSettings", undefined);
    } catch (error) {
      setDockerSettings(null);
      updateError("dockerSettings", error instanceof Error ? error.message : "加载 Docker 设置失败");
    } finally {
      updateLoading("dockerSettings", false);
    }
  };

  const loadAll = async () => {
    await Promise.allSettled([
      loadContainers(),
      loadImages(),
      loadProjects(),
      loadNetworks(),
      loadSystemInfo(),
      loadDockerSettings(),
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
    try {
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
    } catch (error) {
      message.error(getErrorMessage(error));
    }
  };

  const removeImage = async (image: DockerImage) => {
    const actionKey = `${image.id}:delete`;
    setImageActionKey(actionKey);
    try {
      await apiRequest(`/api/docker/images/${encodeURIComponent(image.id)}`, {
        method: "DELETE",
      });
      message.success("镜像已删除");
      await Promise.allSettled([loadImages(), loadSystemInfo()]);
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setImageActionKey(null);
    }
  };

  const pruneImages = async () => {
    setImageActionKey("prune");
    try {
      const result = await apiRequest<DockerImagePruneResult>("/api/docker/images/prune", {
        method: "POST",
      });
      message.success(`已清理 ${result.images_deleted} 个未使用镜像，回收 ${bytesToSize(result.space_reclaimed)}`);
      await Promise.allSettled([loadImages(), loadSystemInfo()]);
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setImageActionKey(null);
    }
  };

  const submitCustomCompose = async (values: CustomComposeValues) => {
    setCustomComposeSubmitting(true);
    try {
      await apiRequest<{ project: Project }>("/api/projects/custom", {
        method: "POST",
        body: JSON.stringify(values),
      });
      message.success("编排已创建");
      setCustomComposeOpen(false);
      customComposeForm.resetFields();
      await Promise.allSettled([loadProjects(), loadContainers(), loadSystemInfo()]);
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setCustomComposeSubmitting(false);
    }
  };

  const saveDockerSettings = async () => {
    const mirrors = dockerSettingsTextToList(dockerSettings?.registry_mirrors ?? []);
    setDockerSettingsSubmitting(true);
    try {
      const response = await apiRequest<DockerSettings>("/api/docker/settings", {
        method: "PUT",
        body: JSON.stringify({ registry_mirrors: mirrors }),
      });
      setDockerSettings(response);
      message.success("镜像源配置已保存");
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setDockerSettingsSubmitting(false);
    }
  };

  const restartDocker = async () => {
    setDockerRestarting(true);
    try {
      await apiRequest<{ ok: boolean }>("/api/docker/restart", {
        method: "POST",
      });
      message.success("Docker 已重启");
      await Promise.allSettled([loadSystemInfo(), loadDockerSettings(), loadContainers(), loadImages(), loadProjects()]);
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setDockerRestarting(false);
    }
  };

  const runContainerAction = async (container: DockerContainer, action: string) => {
    const actionKey = `${container.id}:${action}`;
    setContainerActionKey(actionKey);
    try {
      await apiRequest(`/api/docker/containers/${container.id}/actions`, {
        method: "POST",
        body: JSON.stringify({ action }),
      });
      message.success(`容器已${actionLabel(action)}`);
      const nextContainers = await loadContainers();
      await Promise.allSettled([loadProjects(), loadSystemInfo()]);

      if (activeContainer?.id !== container.id) {
        return;
      }

      if (action === "delete") {
        setActiveContainer(null);
        return;
      }

      const updatedContainer = nextContainers.find((item) => item.id === container.id);
      if (!updatedContainer) {
        setActiveContainer(null);
        return;
      }

      await openContainerLogs(updatedContainer);
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setContainerActionKey(null);
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
                    <Space wrap>
                      <Button
                        size="small"
                        icon={<PlayCircleOutlined />}
                        disabled={record.state === "running" || containerActionKey !== null}
                        loading={containerActionKey === `${record.id}:start`}
                        onClick={() => void runContainerAction(record, "start")}
                      />
                      <Button
                        size="small"
                        icon={<PauseCircleOutlined />}
                        disabled={isContainerStopped(record.state) || containerActionKey !== null}
                        loading={containerActionKey === `${record.id}:stop`}
                        onClick={() => void runContainerAction(record, "stop")}
                      />
                      <Button
                        size="small"
                        icon={<SyncOutlined />}
                        disabled={containerActionKey !== null}
                        loading={containerActionKey === `${record.id}:restart`}
                        onClick={() => void runContainerAction(record, "restart")}
                      />
                      <Button size="small" icon={<EyeOutlined />} onClick={() => void openContainerLogs(record)}>
                        日志
                      </Button>
                      <Popconfirm
                        title={`确认删除容器 ${record.name}？`}
                        description={
                          record.project
                            ? `该容器属于编排 ${record.project}，删除后项目可能进入降级状态。`
                            : "删除后无法恢复。"
                        }
                        okText="删除"
                        cancelText="取消"
                        okButtonProps={{ danger: true, loading: containerActionKey === `${record.id}:delete` }}
                        onConfirm={() => void runContainerAction(record, "delete")}
                      >
                        <Button danger size="small" icon={<DeleteOutlined />} disabled={containerActionKey !== null} />
                      </Popconfirm>
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
      key: "images",
      label: `镜像 (${images.length})`,
      children: (
        <div className="containers-tab-stack">
          {renderError(errors.images)}
          <div className="page-inline-bar">
            <Typography.Text type="secondary">
              支持删除单个镜像，或一键清理所有未被容器使用的镜像。
            </Typography.Text>
            <Popconfirm
              title="确认清理所有未使用镜像？"
              description="只会清理当前没有被容器使用的镜像。"
              okText="立即清理"
              cancelText="取消"
              okButtonProps={{ danger: true, loading: imageActionKey === "prune" }}
              onConfirm={() => void pruneImages()}
            >
              <Button danger loading={imageActionKey === "prune"}>
                清理未使用镜像
              </Button>
            </Popconfirm>
          </div>
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
                {
                  title: "动作",
                  render: (_, record) => (
                    <Popconfirm
                      title="确认删除这个镜像？"
                      description="如果仍有容器在使用，Docker 会拒绝删除。"
                      okText="删除"
                      cancelText="取消"
                      okButtonProps={{ danger: true, loading: imageActionKey === `${record.id}:delete` }}
                      onConfirm={() => void removeImage(record)}
                    >
                      <Button
                        danger
                        size="small"
                        icon={<DeleteOutlined />}
                        disabled={imageActionKey !== null}
                      >
                        删除
                      </Button>
                    </Popconfirm>
                  ),
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
            message="这里可以管理现有编排，也可以直接粘贴 compose.yaml 新建自定义编排。"
          />
          <div className="page-inline-bar">
            <Typography.Text type="secondary">
              自定义编排会直接保存原始 compose 内容，不走模板参数化。
            </Typography.Text>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setCustomComposeOpen(true)}>
              新建编排
            </Button>
          </div>
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
          <Alert
            showIcon
            type="info"
            message="这里可以查看 Docker 系统信息，并维护 daemon 的 registry-mirrors 与重启 Docker。"
          />
          {renderError(errors.system)}
          {renderError(errors.dockerSettings)}
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

          <Card
            className="glass-card"
            title="Docker 镜像源"
            loading={loading.dockerSettings}
            extra={
              <Space>
                <Button
                  onClick={() => void saveDockerSettings()}
                  loading={dockerSettingsSubmitting}
                  disabled={!dockerSettings?.control_enabled}
                >
                  保存镜像源
                </Button>
                <Popconfirm
                  title="确认重启 Docker？"
                  description="会短暂影响容器管理和镜像拉取。"
                  okText="立即重启"
                  cancelText="取消"
                  okButtonProps={{ danger: true, loading: dockerRestarting }}
                  onConfirm={() => void restartDocker()}
                >
                  <Button danger loading={dockerRestarting} disabled={!dockerSettings?.control_enabled}>
                    重启 Docker
                  </Button>
                </Popconfirm>
              </Space>
            }
          >
            {dockerSettings ? (
              <Space direction="vertical" size="middle" style={{ width: "100%" }}>
                <Typography.Text type="secondary">{dockerSettings.message || "一行一个镜像源 URL。"}</Typography.Text>
                <Descriptions bordered size="small" column={1}>
                  <Descriptions.Item label="配置文件">{dockerSettings.config_path}</Descriptions.Item>
                  <Descriptions.Item label="宿主机控制">
                    {dockerSettings.control_enabled ? (
                      <Tag color="green">已启用</Tag>
                    ) : (
                      <Tag color="red">未启用</Tag>
                    )}
                  </Descriptions.Item>
                </Descriptions>
                <Input.TextArea
                  rows={6}
                  value={dockerSettings.registry_mirrors.join("\n")}
                  onChange={(event) =>
                    setDockerSettings((current) =>
                      current
                        ? {
                            ...current,
                            registry_mirrors: dockerSettingsTextToList(event.target.value.split("\n")),
                          }
                        : current,
                    )
                  }
                  placeholder={"https://mirror-1.example\nhttps://mirror-2.example"}
                  disabled={!dockerSettings.control_enabled}
                />
              </Space>
            ) : (
              <Empty description="当前无法读取 Docker daemon 设置" />
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

      <Modal
        open={customComposeOpen}
        title="新建编排"
        okText="立即创建"
        cancelText="取消"
        onCancel={() => {
          setCustomComposeOpen(false);
          customComposeForm.resetFields();
        }}
        onOk={() => void customComposeForm.submit()}
        confirmLoading={customComposeSubmitting}
        destroyOnClose
      >
        <Form
          form={customComposeForm}
          layout="vertical"
          initialValues={{ name: "", compose: "services:\n  app:\n    image: nginx:alpine\n" }}
          onFinish={submitCustomCompose}
        >
          <Form.Item
            label="编排名"
            name="name"
            rules={[{ required: true, message: "请输入编排名" }]}
            extra="只允许小写字母、数字、下划线和中划线。"
          >
            <Input placeholder="custom-blog" />
          </Form.Item>
          <Form.Item
            label="Compose"
            name="compose"
            rules={[{ required: true, message: "请输入 Compose 内容" }]}
            extra="直接填写 compose.yaml 原文。"
          >
            <Input.TextArea rows={12} spellCheck={false} />
          </Form.Item>
        </Form>
      </Modal>

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

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : "操作失败";
}

function isContainerStopped(state?: string) {
  switch (state) {
    case "created":
    case "dead":
    case "exited":
    case "removing":
      return true;
    default:
      return false;
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

function dockerSettingsTextToList(values: string[]) {
  return values.map((item) => item.trim()).filter(Boolean);
}
