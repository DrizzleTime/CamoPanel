import { Card, Descriptions, Drawer, Empty, Space, Tag, Typography } from "antd";
import type { Project } from "../../../shared/types";
import type { DockerContainer } from "../types";

type ContainerDetailsDrawersProps = {
  activeContainer: DockerContainer | null;
  activeProject: Project | null;
  containerLogs: string;
  containerLogsLoading: boolean;
  projectLogs: string;
  projectLogsLoading: boolean;
  onCloseContainer: () => void;
  onCloseProject: () => void;
  containerStateColor: (state?: string) => string;
  projectStatusColor: (status?: string) => string;
};

export function ContainerDetailsDrawers({
  activeContainer,
  activeProject,
  containerLogs,
  containerLogsLoading,
  projectLogs,
  projectLogsLoading,
  onCloseContainer,
  onCloseProject,
  containerStateColor,
  projectStatusColor,
}: ContainerDetailsDrawersProps) {
  return (
    <>
      <Drawer
        open={!!activeContainer}
        onClose={onCloseContainer}
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
        onClose={onCloseProject}
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
    </>
  );
}
