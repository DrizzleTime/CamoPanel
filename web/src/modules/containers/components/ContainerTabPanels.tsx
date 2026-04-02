import {
  DeleteOutlined,
  EyeOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  PlusOutlined,
  SyncOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Empty,
  Input,
  Popconfirm,
  Space,
  Table,
  Tag,
  Typography,
} from "antd";
import type { ReactNode } from "react";
import type { Project } from "../../../shared/types";
import { bytesToSize } from "../../../shared/lib/format";
import type {
  DockerContainer,
  DockerImage,
  DockerNetwork,
  DockerSettings,
  DockerSystemInfo,
} from "../types";

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

type ContainerTabPanelsProps = {
  activeTab: string;
  errors: ErrorState;
  loading: LoadingState;
  containers: DockerContainer[];
  images: DockerImage[];
  projects: Project[];
  networks: DockerNetwork[];
  systemInfo: DockerSystemInfo | null;
  dockerSettings: DockerSettings | null;
  containerActionKey: string | null;
  imageActionKey: string | null;
  dockerSettingsSubmitting: boolean;
  dockerRestarting: boolean;
  onOpenProjectDetails: (project: Project) => void;
  onOpenContainerLogs: (container: DockerContainer) => void;
  onRunContainerAction: (container: DockerContainer, action: string) => Promise<void>;
  onRunProjectAction: (project: Project, action: string) => Promise<void>;
  onRemoveImage: (image: DockerImage) => Promise<void>;
  onPruneImages: () => Promise<void>;
  onOpenCustomCompose: () => void;
  onSaveDockerSettings: () => Promise<void>;
  onRestartDocker: () => Promise<void>;
  onDockerSettingsChange: (value: string) => void;
  renderError: (messageText?: string) => ReactNode;
  renderTagList: (values: string[], emptyText: string) => ReactNode;
  renderImageTags: (tags: string[]) => ReactNode;
  containerStateColor: (state?: string) => string;
  projectStatusColor: (status?: string) => string;
  isContainerStopped: (state?: string) => boolean;
  actionLabel: (action: string) => string;
  formatDateTime: (value: string) => string;
  shortId: (value: string) => string;
};

export function ContainerTabPanels({
  activeTab,
  errors,
  loading,
  containers,
  images,
  projects,
  networks,
  systemInfo,
  dockerSettings,
  containerActionKey,
  imageActionKey,
  dockerSettingsSubmitting,
  dockerRestarting,
  onOpenProjectDetails,
  onOpenContainerLogs,
  onRunContainerAction,
  onRunProjectAction,
  onRemoveImage,
  onPruneImages,
  onOpenCustomCompose,
  onSaveDockerSettings,
  onRestartDocker,
  onDockerSettingsChange,
  renderError,
  renderTagList,
  renderImageTags,
  containerStateColor,
  projectStatusColor,
  isContainerStopped,
  actionLabel,
  formatDateTime,
  shortId,
}: ContainerTabPanelsProps) {
  const panels = {
    containers: (
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
                      onClick={() => void onRunContainerAction(record, "start")}
                    />
                    <Button
                      size="small"
                      icon={<PauseCircleOutlined />}
                      disabled={isContainerStopped(record.state) || containerActionKey !== null}
                      loading={containerActionKey === `${record.id}:stop`}
                      onClick={() => void onRunContainerAction(record, "stop")}
                    />
                    <Button
                      size="small"
                      icon={<SyncOutlined />}
                      disabled={containerActionKey !== null}
                      loading={containerActionKey === `${record.id}:restart`}
                      onClick={() => void onRunContainerAction(record, "restart")}
                    />
                    <Button size="small" icon={<EyeOutlined />} onClick={() => void onOpenContainerLogs(record)}>
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
                      onConfirm={() => void onRunContainerAction(record, "delete")}
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
    images: (
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
            onConfirm={() => void onPruneImages()}
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
                    onConfirm={() => void onRemoveImage(record)}
                  >
                    <Button danger size="small" icon={<DeleteOutlined />} disabled={imageActionKey !== null}>
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
    projects: (
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
          <Button type="primary" icon={<PlusOutlined />} onClick={onOpenCustomCompose}>
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
              onClick: () => void onOpenProjectDetails(record),
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
                    <Button size="small" icon={<PlayCircleOutlined />} onClick={() => void onRunProjectAction(record, "start")} />
                    <Button size="small" icon={<PauseCircleOutlined />} onClick={() => void onRunProjectAction(record, "stop")} />
                    <Button size="small" icon={<SyncOutlined />} onClick={() => void onRunProjectAction(record, "restart")} />
                    <Button size="small" onClick={() => void onRunProjectAction(record, "redeploy")}>
                      重新部署
                    </Button>
                    <Button danger size="small" icon={<DeleteOutlined />} onClick={() => void onRunProjectAction(record, "delete")} />
                  </Space>
                ),
              },
            ]}
          />
        </Card>
      </div>
    ),
    networks: (
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
    system: (
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
                onClick={() => void onSaveDockerSettings()}
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
                onConfirm={() => void onRestartDocker()}
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
                  {dockerSettings.control_enabled ? <Tag color="green">已启用</Tag> : <Tag color="red">未启用</Tag>}
                </Descriptions.Item>
              </Descriptions>
              <Input.TextArea
                rows={6}
                value={dockerSettings.registry_mirrors.join("\n")}
                onChange={(event) => onDockerSettingsChange(event.target.value)}
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
  };

  return panels[activeTab as keyof typeof panels] ?? null;
}
