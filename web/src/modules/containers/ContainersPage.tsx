import {
  DeleteOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  SyncOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  Modal,
  Space,
  Tag,
  Typography,
  message,
} from "antd";
import { Suspense, lazy, useEffect, useMemo, useState } from "react";
import type { Project } from "../../shared/types";
import { bytesToSize } from "../../shared/lib/format";
import { useShellHeader } from "../../widgets/shell/ShellHeaderContext";
import { ContainerTabPanels } from "./components/ContainerTabPanels";
import {
  createCustomComposeProject,
  deleteDockerImage,
  getContainerProjectLogs,
  getDockerContainerLogs,
  getDockerSettings,
  getDockerSystemInfo,
  listContainerProjects,
  listDockerContainers,
  listDockerImages,
  listDockerNetworks,
  pruneDockerImages,
  restartDockerService,
  runContainerProjectAction,
  runDockerContainerAction,
  updateDockerSettings as saveDockerSettingsRequest,
} from "./api";
import type {
  CustomComposeValues,
  DockerContainer,
  DockerImage,
  DockerNetwork,
  DockerSettings,
  DockerSystemInfo,
} from "./types";

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

const CONTAINER_TAB_LABELS = {
  containers: "容器",
  images: "镜像",
  projects: "编排",
  networks: "网络",
  system: "系统设置",
} as const;

const LazyContainerDetailsDrawers = lazy(async () => {
  const module = await import("./components/ContainerDrawers");
  return { default: module.ContainerDetailsDrawers };
});

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
      const response = await listDockerContainers();
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
      const response = await listDockerImages();
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
      const response = await listContainerProjects();
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
      const response = await listDockerNetworks();
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
      const response = await getDockerSystemInfo();
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
      const response = await getDockerSettings();
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
      await runContainerProjectAction(project.id, action);
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
      await deleteDockerImage(image.id);
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
      const result = await pruneDockerImages();
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
      await createCustomComposeProject(values);
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
      const response = await saveDockerSettingsRequest({ registry_mirrors: mirrors });
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
      await restartDockerService();
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
      await runDockerContainerAction(container.id, action);
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
      const response = await getDockerContainerLogs(container.id);
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
      const response = await getContainerProjectLogs(project.id);
      setProjectLogs(response.logs || "暂无日志");
    } catch {
      setProjectLogs("当前无法读取日志，通常是 Docker 未安装或 daemon 未启动。");
    } finally {
      setProjectLogsLoading(false);
    }
  };

  const showDetailsDrawers = activeContainer !== null || activeProject !== null;

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

      <ContainerTabPanels
        activeTab={activeTab}
        errors={errors}
        loading={loading}
        containers={containers}
        images={images}
        projects={projects}
        networks={networks}
        systemInfo={systemInfo}
        dockerSettings={dockerSettings}
        containerActionKey={containerActionKey}
        imageActionKey={imageActionKey}
        dockerSettingsSubmitting={dockerSettingsSubmitting}
        dockerRestarting={dockerRestarting}
        onOpenProjectDetails={openProjectDetails}
        onOpenContainerLogs={openContainerLogs}
        onRunContainerAction={runContainerAction}
        onRunProjectAction={runProjectAction}
        onRemoveImage={removeImage}
        onPruneImages={pruneImages}
        onOpenCustomCompose={() => setCustomComposeOpen(true)}
        onSaveDockerSettings={saveDockerSettings}
        onRestartDocker={restartDocker}
        onDockerSettingsChange={(value) =>
          setDockerSettings((current) =>
            current
              ? {
                  ...current,
                  registry_mirrors: dockerSettingsTextToList(value.split("\n")),
                }
              : current,
          )
        }
        renderError={renderError}
        renderTagList={renderTagList}
        renderImageTags={renderImageTags}
        containerStateColor={containerStateColor}
        projectStatusColor={projectStatusColor}
        isContainerStopped={isContainerStopped}
        actionLabel={actionLabel}
        formatDateTime={formatDateTime}
        shortId={shortId}
      />

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

      {showDetailsDrawers ? (
        <Suspense fallback={null}>
          <LazyContainerDetailsDrawers
            activeContainer={activeContainer}
            activeProject={activeProject}
            containerLogs={containerLogs}
            containerLogsLoading={containerLogsLoading}
            projectLogs={projectLogs}
            projectLogsLoading={projectLogsLoading}
            onCloseContainer={() => setActiveContainer(null)}
            onCloseProject={() => setActiveProject(null)}
            containerStateColor={containerStateColor}
            projectStatusColor={projectStatusColor}
          />
        </Suspense>
      ) : null}
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
