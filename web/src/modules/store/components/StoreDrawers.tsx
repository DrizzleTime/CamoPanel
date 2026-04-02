import { DeleteOutlined, ReloadOutlined } from "@ant-design/icons";
import {
  Button,
  Card,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  Popconfirm,
  Space,
  Switch,
  Tag,
  Typography,
} from "antd";
import type { FormInstance } from "antd/es/form";
import type { Project, TemplateParam, TemplateSpec } from "../../../shared/types";

type StoreDrawersProps = {
  activeTemplate: TemplateSpec | null;
  managedTemplate: TemplateSpec | null;
  projectsByTemplate: Map<string, Project[]>;
  initialValues: Record<string, unknown>;
  projectNameLocked: boolean;
  submitting: boolean;
  projectActionKey: string | null;
  form: FormInstance;
  onCloseInstall: () => void;
  onCloseManage: () => void;
  onSubmitInstall: () => void;
  onOpenInstallFromManage: (template: TemplateSpec) => void;
  onDeploy: (values: Record<string, unknown>) => Promise<void>;
  onRunProjectAction: (project: Project, action: "delete" | "redeploy") => Promise<void>;
  projectActionLabel: (action: "delete" | "redeploy") => string;
  projectStatusColor: (status?: string) => string;
  projectStatusLabel: (status?: string) => string;
  projectRuntimeHint: (project: Project) => string;
  formatDateTime: (value: string) => string;
  managedOpenrestyProjectName: string;
};

export function StoreDrawers({
  activeTemplate,
  managedTemplate,
  projectsByTemplate,
  initialValues,
  projectNameLocked,
  submitting,
  projectActionKey,
  form,
  onCloseInstall,
  onCloseManage,
  onSubmitInstall,
  onOpenInstallFromManage,
  onDeploy,
  onRunProjectAction,
  projectActionLabel,
  projectStatusColor,
  projectStatusLabel,
  projectRuntimeHint,
  formatDateTime,
  managedOpenrestyProjectName,
}: StoreDrawersProps) {
  return (
    <>
      <Drawer
        open={!!activeTemplate}
        title={activeTemplate ? `安装 ${activeTemplate.name}` : "安装应用"}
        size={520}
        onClose={onCloseInstall}
        destroyOnHidden
        extra={
          <Space>
            <Button onClick={onCloseInstall}>取消</Button>
            <Button type="primary" loading={submitting} onClick={onSubmitInstall}>
              立即安装
            </Button>
          </Space>
        }
      >
        {activeTemplate ? (
          <Form form={form} layout="vertical" onFinish={onDeploy} initialValues={initialValues}>
            <Form.Item
              label="项目名"
              name="projectName"
              rules={[{ required: true, message: "请输入项目名" }]}
              extra={projectNameLocked ? "固定 OpenResty 项目，安装后会直接供网站管理页复用。" : undefined}
            >
              <Input
                placeholder={projectNameLocked ? managedOpenrestyProjectName : `${activeTemplate.id}-app`}
                disabled={projectNameLocked}
              />
            </Form.Item>
            {activeTemplate.params.map((param) => (
              <TemplateField key={param.name} param={param} />
            ))}
          </Form>
        ) : null}
      </Drawer>

      <Drawer
        open={!!managedTemplate}
        title={managedTemplate ? `${managedTemplate.name} 实例管理` : "实例管理"}
        size={520}
        onClose={onCloseManage}
        destroyOnHidden
      >
        {managedTemplate ? (
          <Space direction="vertical" size="middle" style={{ width: "100%" }}>
            <Typography.Text type="secondary">
              卸载会删除当前项目实例，但不会清理卷数据。重装会重新部署当前配置，并保留已有数据。
            </Typography.Text>
            {(projectsByTemplate.get(managedTemplate.id) ?? []).length > 0 ? (
              (projectsByTemplate.get(managedTemplate.id) ?? []).map((project) => (
                <Card key={project.id} className="store-instance-card" variant="borderless">
                  <div className="store-instance-card-body">
                    <div className="store-instance-header">
                      <div className="store-instance-summary">
                        <Typography.Title level={5} className="store-instance-title">
                          {project.name}
                        </Typography.Title>
                        <Space wrap size={[8, 8]}>
                          <Tag color={projectStatusColor(project.status)}>{projectStatusLabel(project.status)}</Tag>
                          <Tag>{project.runtime.containers.length} 个容器</Tag>
                          <Tag>{formatDateTime(project.created_at)}</Tag>
                        </Space>
                      </div>
                    </div>

                    <Typography.Text type="secondary">
                      {project.last_error || projectRuntimeHint(project)}
                    </Typography.Text>

                    <div className="store-instance-footer">
                      <Typography.Text type="secondary">项目 ID：{project.id}</Typography.Text>
                      <Space wrap>
                        <Button
                          icon={<ReloadOutlined />}
                          loading={projectActionKey === `${project.id}:redeploy`}
                          onClick={() => void onRunProjectAction(project, "redeploy")}
                        >
                          {projectActionLabel("redeploy")}
                        </Button>
                        <Popconfirm
                          title={`卸载 ${project.name}`}
                          description="会删除项目实例和关联容器，但不会自动删除卷数据。"
                          okText="卸载"
                          cancelText="取消"
                          okButtonProps={{ danger: true }}
                          onConfirm={() => void onRunProjectAction(project, "delete")}
                        >
                          <Button
                            danger
                            icon={<DeleteOutlined />}
                            loading={projectActionKey === `${project.id}:delete`}
                          >
                            {projectActionLabel("delete")}
                          </Button>
                        </Popconfirm>
                      </Space>
                    </div>
                  </div>
                </Card>
              ))
            ) : (
              <Empty description="当前模板还没有项目实例" image={Empty.PRESENTED_IMAGE_SIMPLE}>
                <Button type="primary" onClick={() => onOpenInstallFromManage(managedTemplate)}>
                  去安装
                </Button>
              </Empty>
            )}
          </Space>
        ) : null}
      </Drawer>
    </>
  );
}

function TemplateField({ param }: { param: TemplateParam }) {
  const rules = param.required ? [{ required: true, message: `请输入${param.label}` }] : [];

  switch (param.type) {
    case "number":
      return (
        <Form.Item label={param.label} name={param.name} rules={rules} extra={param.description}>
          <InputNumber style={{ width: "100%" }} />
        </Form.Item>
      );
    case "boolean":
      return (
        <Form.Item
          label={param.label}
          name={param.name}
          valuePropName="checked"
          extra={param.description}
        >
          <Switch />
        </Form.Item>
      );
    case "secret":
      return (
        <Form.Item label={param.label} name={param.name} rules={rules} extra={param.description}>
          <Input.Password placeholder={param.placeholder} />
        </Form.Item>
      );
    default:
      return (
        <Form.Item label={param.label} name={param.name} rules={rules} extra={param.description}>
          <Input placeholder={param.placeholder} />
        </Form.Item>
      );
  }
}
