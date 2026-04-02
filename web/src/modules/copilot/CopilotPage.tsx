import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  SendOutlined,
  StarOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Empty,
  Form,
  Input,
  Modal,
  Popconfirm,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import { useEffect, useMemo, useState } from "react";
import { useShellHeader } from "../../widgets/shell/ShellHeaderContext";
import {
  deleteCopilotModel,
  deleteCopilotProvider,
  getCopilotConfigStatus,
  listCopilotProviders,
  saveCopilotModel,
  saveCopilotProvider,
  setCopilotDefaultModel,
} from "./api";
import { useCopilotChat } from "./hooks/useCopilotChat";
import type { CopilotConfigStatus, CopilotModel, CopilotProvider } from "./types";

type CopilotSection = "chat" | "services";

type ProviderFormValues = {
  name: string;
  base_url: string;
  api_key: string;
  enabled: boolean;
};

type ModelFormValues = {
  name: string;
  enabled: boolean;
  is_default: boolean;
};

export function CopilotPage() {
  const [activeSection, setActiveSection] = useState<CopilotSection>("chat");
  const [configStatus, setConfigStatus] = useState<CopilotConfigStatus | null>(null);
  const [configLoading, setConfigLoading] = useState(true);
  const [providers, setProviders] = useState<CopilotProvider[]>([]);
  const [providersLoading, setProvidersLoading] = useState(true);
  const [providerModalOpen, setProviderModalOpen] = useState(false);
  const [editingProvider, setEditingProvider] = useState<CopilotProvider | null>(null);
  const [modelModalOpen, setModelModalOpen] = useState(false);
  const [editingModel, setEditingModel] = useState<CopilotModel | null>(null);
  const [activeProvider, setActiveProvider] = useState<CopilotProvider | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [providerForm] = Form.useForm<ProviderFormValues>();
  const [modelForm] = Form.useForm<ModelFormValues>();
  const { sessionId, messages, input, setInput, sending, sessionError, send } = useCopilotChat();

  const loadConfigStatus = async () => {
    setConfigLoading(true);
    try {
      const response = await getCopilotConfigStatus();
      setConfigStatus(response);
    } catch (error) {
      setConfigStatus(null);
      message.error(getErrorMessage(error));
    } finally {
      setConfigLoading(false);
    }
  };

  const loadProviders = async () => {
    setProvidersLoading(true);
    try {
      const response = await listCopilotProviders();
      setProviders(response.items);
    } catch (error) {
      setProviders([]);
      message.error(getErrorMessage(error));
    } finally {
      setProvidersLoading(false);
    }
  };

  const refreshCopilotData = async () => {
    await Promise.all([loadConfigStatus(), loadProviders()]);
  };

  useEffect(() => {
    void refreshCopilotData();
  }, []);

  const headerContent = useMemo(
    () => (
      <div className="shell-header-tabs">
        {[
          { key: "chat", label: "对话" },
          { key: "services", label: `模型服务 (${providers.length})` },
        ].map((item) => (
          <Button
            key={item.key}
            size="small"
            type={activeSection === item.key ? "primary" : "text"}
            className="shell-header-tab"
            onClick={() => setActiveSection(item.key as CopilotSection)}
          >
            {item.label}
          </Button>
        ))}
      </div>
    ),
    [activeSection, providers.length],
  );

  useShellHeader(headerContent);

  const openCreateProvider = () => {
    setEditingProvider(null);
    providerForm.resetFields();
    providerForm.setFieldsValue({
      enabled: true,
    });
    setProviderModalOpen(true);
  };

  const openEditProvider = (provider: CopilotProvider) => {
    setEditingProvider(provider);
    providerForm.setFieldsValue({
      name: provider.name,
      base_url: provider.base_url,
      api_key: "",
      enabled: provider.enabled,
    });
    setProviderModalOpen(true);
  };

  const openCreateModel = (provider: CopilotProvider) => {
    setActiveProvider(provider);
    setEditingModel(null);
    modelForm.resetFields();
    modelForm.setFieldsValue({
      enabled: true,
      is_default: provider.models.length === 0,
    });
    setModelModalOpen(true);
  };

  const openEditModel = (provider: CopilotProvider, aiModel: CopilotModel) => {
    setActiveProvider(provider);
    setEditingModel(aiModel);
    modelForm.setFieldsValue({
      name: aiModel.name,
      enabled: aiModel.enabled,
      is_default: aiModel.is_default,
    });
    setModelModalOpen(true);
  };

  const saveProvider = async () => {
    try {
      const values = await providerForm.validateFields();
      setSubmitting(true);
      await saveCopilotProvider(values, editingProvider?.id);
      message.success(editingProvider ? "模型服务已更新" : "模型服务已创建");
      setProviderModalOpen(false);
      providerForm.resetFields();
      await refreshCopilotData();
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setSubmitting(false);
    }
  };

  const deleteProvider = async (provider: CopilotProvider) => {
    try {
      await deleteCopilotProvider(provider.id);
      message.success("模型服务已删除");
      await refreshCopilotData();
    } catch (error) {
      message.error(getErrorMessage(error));
    }
  };

  const saveModel = async () => {
    if (!activeProvider) return;

    try {
      const values = await modelForm.validateFields();
      setSubmitting(true);
      await saveCopilotModel(values, activeProvider.id, editingModel?.id);
      message.success(editingModel ? "模型已更新" : "模型已创建");
      setModelModalOpen(false);
      modelForm.resetFields();
      await refreshCopilotData();
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setSubmitting(false);
    }
  };

  const deleteModel = async (aiModel: CopilotModel) => {
    try {
      await deleteCopilotModel(aiModel.id);
      message.success("模型已删除");
      await refreshCopilotData();
    } catch (error) {
      message.error(getErrorMessage(error));
    }
  };

  const setDefaultModel = async (aiModel: CopilotModel) => {
    try {
      await setCopilotDefaultModel(aiModel);
      message.success("默认模型已切换");
      await refreshCopilotData();
    } catch (error) {
      message.error(getErrorMessage(error));
    }
  };

  const renderConfigStatus = () => {
    if (configLoading) {
      return <Alert showIcon type="info" message="正在读取 Copilot 配置..." />;
    }

    if (!configStatus?.configured) {
      return <Alert showIcon type="warning" message="当前没有可用模型，请先在模型服务中配置默认模型。" />;
    }

    const sourceText = configStatus.source === "database" ? "数据库配置" : "环境变量";
    return (
      <Alert
        showIcon
        type={configStatus.source === "env" ? "warning" : "success"}
        message={`当前使用 ${sourceText}: ${configStatus.provider_name} / ${configStatus.model_name}`}
        description={configStatus.base_url}
      />
    );
  };

  return (
    <div className="page-grid">
      {activeSection === "chat" ? (
        <>
          <Alert showIcon type="info" message="Copilot 目前只提供只读分析，不直接执行写操作。" />
          {renderConfigStatus()}
          {sessionError ? <Alert type="error" message={sessionError} /> : null}

          <Card className="glass-card">
            <div className="copilot-stream">
              {messages.map((item) => (
                <div key={item.id} className={`chat-bubble ${item.role}`}>
                  {item.content || "正在思考..."}
                </div>
              ))}
            </div>
          </Card>

          <Card className="glass-card">
            <Space.Compact style={{ width: "100%" }}>
              <Input.TextArea
                value={input}
                onChange={(event) => setInput(event.target.value)}
                placeholder="例如：帮我部署一个 WordPress，或者这个项目为什么起不来？"
                autoSize={{ minRows: 2, maxRows: 5 }}
              />
              <Button
                type="primary"
                icon={<SendOutlined />}
                loading={sending}
                disabled={!configStatus?.configured || !sessionId}
                onClick={() => void send()}
              >
                发送
              </Button>
            </Space.Compact>
          </Card>
        </>
      ) : (
        <>
          <div className="page-inline-bar">
            <Typography.Text type="secondary">
              先维护服务商，再为每个服务商挂多个模型。聊天页只使用默认模型。
            </Typography.Text>
            <Space>
              <Button icon={<ReloadOutlined />} onClick={() => void refreshCopilotData()}>
                刷新
              </Button>
              <Button type="primary" icon={<PlusOutlined />} onClick={openCreateProvider}>
                新增服务商
              </Button>
            </Space>
          </div>

          {renderConfigStatus()}
          {configStatus?.source === "env" ? (
            <Alert
              showIcon
              type="warning"
              message="当前仍在使用环境变量兜底配置。创建并设置数据库默认模型后，Copilot 将优先使用数据库配置。"
            />
          ) : null}

          {providersLoading ? (
            <Card className="glass-card" loading />
          ) : providers.length ? (
            <Space direction="vertical" size="large" style={{ width: "100%" }}>
              {providers.map((provider) => (
                <Card
                  key={provider.id}
                  className="glass-card"
                  title={
                    <Space wrap>
                      <Typography.Text strong>{provider.name}</Typography.Text>
                      <Tag color={provider.enabled ? "green" : "default"}>
                        {provider.enabled ? "已启用" : "已停用"}
                      </Tag>
                      <Tag color="blue">{providerTypeLabel(provider.type)}</Tag>
                    </Space>
                  }
                  extra={
                    <Space>
                      <Button size="small" icon={<PlusOutlined />} onClick={() => openCreateModel(provider)}>
                        新增模型
                      </Button>
                      <Button size="small" icon={<EditOutlined />} onClick={() => openEditProvider(provider)}>
                        编辑
                      </Button>
                      <Popconfirm
                        title={`确认删除服务商 ${provider.name}？`}
                        description="删除后会一并删除它下面的所有模型。"
                        okText="删除"
                        cancelText="取消"
                        onConfirm={() => void deleteProvider(provider)}
                      >
                        <Button danger size="small" icon={<DeleteOutlined />}>
                          删除
                        </Button>
                      </Popconfirm>
                    </Space>
                  }
                >
                  <Space direction="vertical" size="middle" style={{ width: "100%" }}>
                    <Space wrap size={[8, 8]}>
                      <Tag>{provider.base_url}</Tag>
                      <Tag>{provider.has_api_key ? provider.api_key_masked : "未保存 API Key"}</Tag>
                    </Space>

                    {provider.models.length ? (
                      <Table<CopilotModel>
                        rowKey="id"
                        pagination={false}
                        dataSource={provider.models}
                        columns={[
                          { title: "模型", dataIndex: "name" },
                          {
                            title: "状态",
                            render: (_, record) => (
                              <Tag color={record.enabled ? "green" : "default"}>
                                {record.enabled ? "已启用" : "已停用"}
                              </Tag>
                            ),
                          },
                          {
                            title: "默认",
                            render: (_, record) =>
                              record.is_default ? <Tag color="gold">默认</Tag> : <Typography.Text type="secondary">否</Typography.Text>,
                          },
                          {
                            title: "更新时间",
                            dataIndex: "updated_at",
                            render: (value: string) => formatDateTime(value),
                          },
                          {
                            title: "动作",
                            render: (_, record) => (
                              <Space wrap>
                                {!record.is_default ? (
                                  <Button
                                    size="small"
                                    icon={<StarOutlined />}
                                    onClick={() => void setDefaultModel(record)}
                                  >
                                    设为默认
                                  </Button>
                                ) : null}
                                <Button size="small" icon={<EditOutlined />} onClick={() => openEditModel(provider, record)}>
                                  编辑
                                </Button>
                                <Popconfirm
                                  title={`确认删除模型 ${record.name}？`}
                                  okText="删除"
                                  cancelText="取消"
                                  onConfirm={() => void deleteModel(record)}
                                >
                                  <Button danger size="small" icon={<DeleteOutlined />}>
                                    删除
                                  </Button>
                                </Popconfirm>
                              </Space>
                            ),
                          },
                        ]}
                      />
                    ) : (
                      <Empty description="这个服务商下还没有模型" />
                    )}
                  </Space>
                </Card>
              ))}
            </Space>
          ) : (
            <Card className="glass-card">
              <Empty description="还没有模型服务">
                <Button type="primary" icon={<PlusOutlined />} onClick={openCreateProvider}>
                  添加第一个服务商
                </Button>
              </Empty>
            </Card>
          )}
        </>
      )}

      <Modal
        open={providerModalOpen}
        title={editingProvider ? "编辑服务商" : "新增服务商"}
        onCancel={() => setProviderModalOpen(false)}
        onOk={() => void saveProvider()}
        confirmLoading={submitting}
        destroyOnHidden
      >
        <Form form={providerForm} layout="vertical" initialValues={{ enabled: true }}>
          <Form.Item label="类型">
            <Input value="OpenAI Compatible" disabled />
          </Form.Item>
          <Form.Item name="name" label="服务商名称" rules={[{ required: true, message: "请输入服务商名称" }]}>
            <Input placeholder="例如 OpenAI / DeepSeek / SiliconFlow" />
          </Form.Item>
          <Form.Item name="base_url" label="Base URL" rules={[{ required: true, message: "请输入 Base URL" }]}>
            <Input placeholder="例如 https://api.openai.com/v1" />
          </Form.Item>
          <Form.Item
            name="api_key"
            label="API Key"
            rules={
              editingProvider
                ? []
                : [{ required: true, message: "请输入 API Key" }]
            }
            extra={editingProvider ? "留空表示保持现有 API Key 不变" : undefined}
          >
            <Input.Password placeholder="sk-..." />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={modelModalOpen}
        title={editingModel ? "编辑模型" : "新增模型"}
        onCancel={() => setModelModalOpen(false)}
        onOk={() => void saveModel()}
        confirmLoading={submitting}
        destroyOnHidden
      >
        <Form form={modelForm} layout="vertical" initialValues={{ enabled: true, is_default: false }}>
          <Form.Item label="所属服务商">
            <Input value={activeProvider?.name || ""} disabled />
          </Form.Item>
          <Form.Item name="name" label="模型名称" rules={[{ required: true, message: "请输入模型名称" }]}>
            <Input placeholder="例如 gpt-5 / gpt-4.1 / deepseek-chat" />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="is_default" label="设为默认模型" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : "操作失败";
}

function providerTypeLabel(value: string) {
  return value === "openai" ? "OpenAI Compatible" : value;
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
