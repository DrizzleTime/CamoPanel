import {
  DatabaseOutlined,
  DeleteOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  SyncOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import { useEffect, useMemo, useState } from "react";
import { useShellHeader } from "../components/ShellHeaderContext";
import { apiRequest } from "../lib/api";
import type {
  DatabaseAccountItem,
  DatabaseEngine,
  DatabaseInstance,
  DatabaseOverview,
  TemplateParam,
  TemplateSpec,
} from "../lib/types";

const DATABASE_ENGINES: Array<{ key: DatabaseEngine; label: string }> = [
  { key: "mysql", label: "MySQL" },
  { key: "postgres", label: "PostgreSQL" },
  { key: "redis", label: "Redis" },
];

const ENGINE_ACTION_LABELS: Record<DatabaseEngine, string> = {
  mysql: "MySQL",
  postgres: "PostgreSQL",
  redis: "Redis",
};

export function DatabasesPage() {
  const [activeEngine, setActiveEngine] = useState<DatabaseEngine>("mysql");
  const [templates, setTemplates] = useState<TemplateSpec[]>([]);
  const [instances, setInstances] = useState<DatabaseInstance[]>([]);
  const [selectedInstanceId, setSelectedInstanceId] = useState<string>();
  const [overview, setOverview] = useState<DatabaseOverview | null>(null);
  const [loadingInstances, setLoadingInstances] = useState(true);
  const [loadingOverview, setLoadingOverview] = useState(false);
  const [deployOpen, setDeployOpen] = useState(false);
  const [databaseModalOpen, setDatabaseModalOpen] = useState(false);
  const [accountModalOpen, setAccountModalOpen] = useState(false);
  const [grantModalOpen, setGrantModalOpen] = useState(false);
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [passwordTarget, setPasswordTarget] = useState<DatabaseAccountItem | null>(null);
  const [deployForm] = Form.useForm();
  const [databaseForm] = Form.useForm();
  const [accountForm] = Form.useForm();
  const [grantForm] = Form.useForm();
  const [passwordForm] = Form.useForm();

  const activeTemplate = useMemo(
    () => templates.find((item) => item.id === activeEngine) ?? null,
    [activeEngine, templates],
  );

  const headerContent = useMemo(
    () => (
      <div className="shell-header-tabs">
        {DATABASE_ENGINES.map((item) => (
          <Button
            key={item.key}
            size="small"
            type={activeEngine === item.key ? "primary" : "text"}
            className="shell-header-tab"
            onClick={() => setActiveEngine(item.key)}
          >
            {item.label}
          </Button>
        ))}
      </div>
    ),
    [activeEngine],
  );

  useShellHeader(headerContent);

  const loadTemplates = async () => {
    const response = await apiRequest<{ items: TemplateSpec[] }>("/api/templates");
    setTemplates(response.items.filter((item) => DATABASE_ENGINES.some((engine) => engine.key === item.id)));
  };

  const loadInstances = async (engine: DatabaseEngine, preferredId?: string) => {
    setLoadingInstances(true);
    try {
      const response = await apiRequest<{ items: DatabaseInstance[] }>(`/api/databases?engine=${engine}`);
      setInstances(response.items);
      const nextSelectedId =
        (preferredId && response.items.some((item) => item.id === preferredId) ? preferredId : undefined) ??
        (response.items[0]?.id || undefined);
      setSelectedInstanceId(nextSelectedId);
      if (!nextSelectedId) {
        setOverview(null);
      }
      return nextSelectedId;
    } finally {
      setLoadingInstances(false);
    }
  };

  const loadOverview = async (instanceId: string) => {
    setLoadingOverview(true);
    try {
      const response = await apiRequest<DatabaseOverview>(`/api/databases/${instanceId}/overview`);
      setOverview(response);
    } finally {
      setLoadingOverview(false);
    }
  };

  useEffect(() => {
    void loadTemplates();
  }, []);

  useEffect(() => {
    void loadInstances(activeEngine);
  }, [activeEngine]);

  useEffect(() => {
    if (!selectedInstanceId) {
      setOverview(null);
      return;
    }
    void loadOverview(selectedInstanceId);
  }, [selectedInstanceId]);

  const refreshCurrent = async () => {
    const currentId = selectedInstanceId;
    const nextId = await loadInstances(activeEngine, currentId);
    if (nextId) {
      await loadOverview(nextId);
    }
  };

  const runInstanceAction = async (action: string) => {
    if (!overview) return;
    await apiRequest(`/api/projects/${overview.instance.id}/actions`, {
      method: "POST",
      body: JSON.stringify({ action }),
    });
    message.success(`${actionLabel(action)}已执行`);
    if (action === "delete") {
      await loadInstances(activeEngine);
      return;
    }
    await refreshCurrent();
  };

  const createInstance = async (values: Record<string, unknown>) => {
    if (!activeTemplate) return;
    setSubmitting(true);
    try {
      const projectName = String(values.projectName || "").trim();
      const { project } = await apiRequest<{ project: DatabaseInstance } | { project: { id: string } }>(
        "/api/projects",
        {
          method: "POST",
          body: JSON.stringify({
            name: projectName,
            template_id: activeTemplate.id,
            parameters: Object.fromEntries(
              Object.entries(values).filter(([key]) => key !== "projectName"),
            ),
          }),
        },
      );
      message.success(`${ENGINE_ACTION_LABELS[activeEngine]} 实例创建完成`);
      setDeployOpen(false);
      deployForm.resetFields();
      await loadInstances(activeEngine, project.id);
    } finally {
      setSubmitting(false);
    }
  };

  const createDatabase = async (values: { name: string }) => {
    if (!overview) return;
    setSubmitting(true);
    try {
      await apiRequest(`/api/databases/${overview.instance.id}/databases`, {
        method: "POST",
        body: JSON.stringify(values),
      });
      message.success("数据库已创建");
      setDatabaseModalOpen(false);
      databaseForm.resetFields();
      await loadOverview(overview.instance.id);
    } finally {
      setSubmitting(false);
    }
  };

  const createAccount = async (values: { name: string; password: string; database_name?: string }) => {
    if (!overview) return;
    setSubmitting(true);
    try {
      await apiRequest(`/api/databases/${overview.instance.id}/accounts`, {
        method: "POST",
        body: JSON.stringify(values),
      });
      message.success("账号已创建");
      setAccountModalOpen(false);
      accountForm.resetFields();
      await loadOverview(overview.instance.id);
    } finally {
      setSubmitting(false);
    }
  };

  const grantAccount = async (values: { account_name: string; database_name: string }) => {
    if (!overview) return;
    setSubmitting(true);
    try {
      await apiRequest(`/api/databases/${overview.instance.id}/grants`, {
        method: "POST",
        body: JSON.stringify(values),
      });
      message.success("授权已执行");
      setGrantModalOpen(false);
      grantForm.resetFields();
    } finally {
      setSubmitting(false);
    }
  };

  const updateAccountPassword = async (values: { password: string }) => {
    if (!overview || !passwordTarget) return;
    setSubmitting(true);
    try {
      await apiRequest(`/api/databases/${overview.instance.id}/accounts/${passwordTarget.name}/password`, {
        method: "POST",
        body: JSON.stringify(values),
      });
      message.success("密码已更新");
      setPasswordModalOpen(false);
      setPasswordTarget(null);
      passwordForm.resetFields();
    } finally {
      setSubmitting(false);
    }
  };

  const updateRedisConfig = async (key: string, value: boolean | number | string) => {
    if (!overview) return;
    await apiRequest(`/api/databases/${overview.instance.id}/redis/config`, {
      method: "POST",
      body: JSON.stringify({ key, value }),
    });
    message.success("Redis 配置已更新并重建实例");
    await refreshCurrent();
  };

  const databaseOptions = overview?.databases?.map((item) => ({
    label: item.name,
    value: item.name,
  })) ?? [];

  const accountOptions = overview?.accounts?.map((item) => ({
    label: item.name,
    value: item.name,
  })) ?? [];

  return (
    <div className="page-grid database-page">
      <div>
        <Typography.Title className="page-title">数据库管理</Typography.Title>
        <Typography.Paragraph className="page-subtitle">
          统一管理 {ENGINE_ACTION_LABELS[activeEngine]} 实例里的数据库、账号、授权和 Redis
          基础配置。实例部署和数据库内部管理放在同一页完成。
        </Typography.Paragraph>
      </div>

      <div className="database-layout">
        <div className="database-sidebar">
          <Card className="glass-card" variant="borderless">
            <Space direction="vertical" size="middle" style={{ width: "100%" }}>
              <Alert
                showIcon
                type="info"
                message={`先创建 ${ENGINE_ACTION_LABELS[activeEngine]} 实例，再进入内部数据库管理。`}
              />
              <Button type="primary" block icon={<DatabaseOutlined />} onClick={() => setDeployOpen(true)}>
                创建 {ENGINE_ACTION_LABELS[activeEngine]} 实例
              </Button>
            </Space>
          </Card>

          <Card
            className="glass-card"
            title={`实例列表 (${instances.length})`}
            extra={
              <Button type="text" icon={<ReloadOutlined />} onClick={() => void loadInstances(activeEngine)}>
                刷新
              </Button>
            }
          >
            <div className="database-instance-list">
              {loadingInstances ? (
                <Typography.Text type="secondary">正在读取实例...</Typography.Text>
              ) : instances.length ? (
                instances.map((item) => (
                  <button
                    type="button"
                    key={item.id}
                    className={`database-instance-card ${selectedInstanceId === item.id ? "active" : ""}`}
                    onClick={() => setSelectedInstanceId(item.id)}
                  >
                    <div className="database-instance-card-top">
                      <Typography.Text strong>{item.name}</Typography.Text>
                      <Tag color={statusColor(item.runtime.status)}>{item.runtime.status}</Tag>
                    </div>
                    <Typography.Text type="secondary">
                      {item.connection.host}:{item.connection.port}
                    </Typography.Text>
                    {item.last_error ? (
                      <Typography.Text type="danger" className="database-instance-error">
                        {item.last_error}
                      </Typography.Text>
                    ) : null}
                  </button>
                ))
              ) : (
                <Empty description={`还没有 ${ENGINE_ACTION_LABELS[activeEngine]} 实例`} />
              )}
            </div>
          </Card>
        </div>

        <div className="database-main">
          {overview ? (
            <>
              <Card
                className="glass-card"
                loading={loadingOverview}
                title={overview.instance.name}
                extra={
                  <Space wrap>
                    <Button icon={<ReloadOutlined />} onClick={() => void refreshCurrent()}>
                      刷新
                    </Button>
                    <Button icon={<PlayCircleOutlined />} onClick={() => void runInstanceAction("start")}>
                      启动
                    </Button>
                    <Button icon={<PauseCircleOutlined />} onClick={() => void runInstanceAction("stop")}>
                      停止
                    </Button>
                    <Button icon={<SyncOutlined />} onClick={() => void runInstanceAction("restart")}>
                      重启
                    </Button>
                    <Button danger icon={<DeleteOutlined />} onClick={() => void runInstanceAction("delete")}>
                      删除实例
                    </Button>
                  </Space>
                }
              >
                <div className="database-overview-grid">
                  <Descriptions bordered size="small" column={1} className="database-descriptions">
                    <Descriptions.Item label="引擎">{ENGINE_ACTION_LABELS[overview.instance.engine]}</Descriptions.Item>
                    <Descriptions.Item label="主机">
                      {overview.instance.connection.host}:{overview.instance.connection.port}
                    </Descriptions.Item>
                    {overview.instance.connection.admin_username ? (
                      <Descriptions.Item label="管理账号">
                        {overview.instance.connection.admin_username}
                      </Descriptions.Item>
                    ) : null}
                    {overview.instance.connection.app_username ? (
                      <Descriptions.Item label="业务账号">
                        {overview.instance.connection.app_username}
                      </Descriptions.Item>
                    ) : null}
                    {overview.instance.connection.default_database ? (
                      <Descriptions.Item label="默认库">
                        {overview.instance.connection.default_database}
                      </Descriptions.Item>
                    ) : null}
                    <Descriptions.Item label="密码">
                      {overview.instance.connection.password_managed ? "已托管" : "未托管"}
                    </Descriptions.Item>
                  </Descriptions>

                  <Card className="glass-card database-summary-card" variant="borderless">
                    <Space direction="vertical" size="middle" style={{ width: "100%" }}>
                      <div className="database-summary-row">
                        <Tag color={statusColor(overview.instance.runtime.status)}>
                          {overview.instance.runtime.status}
                        </Tag>
                        <Typography.Text type="secondary">
                          {overview.instance.runtime.containers.length} 个关联容器
                        </Typography.Text>
                      </div>
                      {overview.notice ? (
                        <Alert showIcon type="warning" message={overview.notice} />
                      ) : (
                        <Alert
                          showIcon
                          type="success"
                          message="实例运行中，可以直接管理数据库内部对象。"
                        />
                      )}
                      {overview.summary ? (
                        <div className="database-summary-kpis">
                          {Object.entries(overview.summary).map(([key, value]) => (
                            <div key={key} className="database-summary-kpi">
                              <Typography.Text type="secondary">{summaryLabel(key)}</Typography.Text>
                              <Typography.Text strong>{value}</Typography.Text>
                            </div>
                          ))}
                        </div>
                      ) : null}
                    </Space>
                  </Card>
                </div>
              </Card>

              {overview.instance.engine === "redis" ? (
                <>
                  <Card className="glass-card" title="逻辑库使用情况" loading={loadingOverview}>
                    <Table
                      rowKey="name"
                      pagination={false}
                      dataSource={overview.redis_keyspaces ?? []}
                      locale={{ emptyText: <Empty description="当前没有逻辑库使用数据" /> }}
                      columns={[
                        { title: "逻辑库", dataIndex: "name" },
                        { title: "Key 数量", dataIndex: "keys" },
                      ]}
                    />
                  </Card>

                  <Card className="glass-card" title="Redis 配置" loading={loadingOverview}>
                    <div className="database-redis-config-grid">
                      <RedisConfigCard
                        label="AOF 持久化"
                        value={(overview.redis_config ?? []).find((item) => item.key === "appendonly")?.value ?? "-"}
                        actionLabel="切换"
                        onSubmit={(nextValue) => updateRedisConfig("appendonly", nextValue === "yes" ? "no" : "yes")}
                      />
                      <RedisConfigCard
                        label="逻辑库数量"
                        value={(overview.redis_config ?? []).find((item) => item.key === "databases")?.value ?? "-"}
                        actionLabel="更新"
                        onSubmit={(value) => {
                          const next = window.prompt("输入新的逻辑库数量", value);
                          if (!next) return Promise.resolve();
                          return updateRedisConfig("databases", Number(next));
                        }}
                      />
                      <RedisConfigCard
                        label="访问密码"
                        value="已托管"
                        actionLabel="修改"
                        onSubmit={() => {
                          const next = window.prompt("输入新的 Redis 密码");
                          if (!next) return Promise.resolve();
                          return updateRedisConfig("password", next);
                        }}
                      />
                    </div>
                  </Card>
                </>
              ) : (
                <>
                  <Card
                    className="glass-card"
                    title="数据库列表"
                    extra={
                      <Button type="primary" onClick={() => setDatabaseModalOpen(true)}>
                        创建数据库
                      </Button>
                    }
                    loading={loadingOverview}
                  >
                    <Table
                      rowKey="name"
                      pagination={false}
                      dataSource={overview.databases ?? []}
                      locale={{ emptyText: <Empty description="当前还没有业务数据库" /> }}
                      columns={[{ title: "数据库名", dataIndex: "name" }]}
                    />
                  </Card>

                  <Card
                    className="glass-card"
                    title="账号管理"
                    extra={
                      <Space wrap>
                        <Button onClick={() => setGrantModalOpen(true)}>授权账号</Button>
                        <Button type="primary" icon={<SafetyCertificateOutlined />} onClick={() => setAccountModalOpen(true)}>
                          创建账号
                        </Button>
                      </Space>
                    }
                    loading={loadingOverview}
                  >
                    <Table
                      rowKey={(record) => `${record.name}-${record.host ?? ""}`}
                      pagination={false}
                      dataSource={overview.accounts ?? []}
                      locale={{ emptyText: <Empty description="当前还没有账号" /> }}
                      columns={[
                        { title: "账号名", dataIndex: "name" },
                        {
                          title: "来源",
                          dataIndex: "host",
                          render: (value?: string) => value || "-",
                        },
                        {
                          title: "属性",
                          render: (_, record) =>
                            record.superuser ? <Tag color="gold">superuser</Tag> : <Tag>普通账号</Tag>,
                        },
                        {
                          title: "动作",
                          render: (_, record) => (
                            <Button
                              size="small"
                              onClick={() => {
                                setPasswordTarget(record);
                                passwordForm.resetFields();
                                setPasswordModalOpen(true);
                              }}
                            >
                              修改密码
                            </Button>
                          ),
                        },
                      ]}
                    />
                  </Card>
                </>
              )}
            </>
          ) : (
            <Card className="glass-card" variant="borderless">
              <Empty
                description={`当前没有可管理的 ${ENGINE_ACTION_LABELS[activeEngine]} 实例`}
                image={Empty.PRESENTED_IMAGE_SIMPLE}
              >
                <Button type="primary" onClick={() => setDeployOpen(true)}>
                  创建实例
                </Button>
              </Empty>
            </Card>
          )}
        </div>
      </div>

      <Modal
        open={deployOpen}
        title={`创建 ${ENGINE_ACTION_LABELS[activeEngine]} 实例`}
        okText="立即创建"
        cancelText="取消"
        confirmLoading={submitting}
        onCancel={() => {
          setDeployOpen(false);
          deployForm.resetFields();
        }}
        onOk={() => void deployForm.submit()}
        destroyOnClose
      >
        <Form
          form={deployForm}
          layout="vertical"
          onFinish={createInstance}
          initialValues={{
            projectName: `${activeEngine}-demo`,
            ...Object.fromEntries(
              (activeTemplate?.params ?? [])
                .filter((item) => item.default !== undefined)
                .map((item) => [item.name, item.default]),
            ),
          }}
        >
          <Form.Item
            label="实例名"
            name="projectName"
            rules={[{ required: true, message: "请输入实例名" }]}
          >
            <Input placeholder={`${activeEngine}-demo`} />
          </Form.Item>
          {(activeTemplate?.params ?? []).map((param) => (
            <TemplateField key={param.name} param={param} />
          ))}
        </Form>
      </Modal>

      <Modal
        open={databaseModalOpen}
        title="创建数据库"
        okText="创建"
        cancelText="取消"
        confirmLoading={submitting}
        onCancel={() => {
          setDatabaseModalOpen(false);
          databaseForm.resetFields();
        }}
        onOk={() => void databaseForm.submit()}
        destroyOnClose
      >
        <Form form={databaseForm} layout="vertical" onFinish={createDatabase}>
          <Form.Item label="数据库名" name="name" rules={[{ required: true, message: "请输入数据库名" }]}>
            <Input placeholder="app_data" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={accountModalOpen}
        title="创建账号"
        okText="创建"
        cancelText="取消"
        confirmLoading={submitting}
        onCancel={() => {
          setAccountModalOpen(false);
          accountForm.resetFields();
        }}
        onOk={() => void accountForm.submit()}
        destroyOnClose
      >
        <Form form={accountForm} layout="vertical" onFinish={createAccount}>
          <Form.Item label="账号名" name="name" rules={[{ required: true, message: "请输入账号名" }]}>
            <Input placeholder="app_user" />
          </Form.Item>
          <Form.Item label="密码" name="password" rules={[{ required: true, message: "请输入密码" }]}>
            <Input.Password placeholder="请输入密码" />
          </Form.Item>
          <Form.Item label="默认授权数据库" name="database_name">
            <Input placeholder={databaseOptions[0]?.value ?? "可选"} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={grantModalOpen}
        title="授权账号"
        okText="授权"
        cancelText="取消"
        confirmLoading={submitting}
        onCancel={() => {
          setGrantModalOpen(false);
          grantForm.resetFields();
        }}
        onOk={() => void grantForm.submit()}
        destroyOnClose
      >
        <Form form={grantForm} layout="vertical" onFinish={grantAccount}>
          <Form.Item label="账号名" name="account_name" rules={[{ required: true, message: "请选择账号" }]}>
            <Input placeholder={accountOptions[0]?.value ?? "输入账号名"} />
          </Form.Item>
          <Form.Item label="数据库名" name="database_name" rules={[{ required: true, message: "请输入数据库名" }]}>
            <Input placeholder={databaseOptions[0]?.value ?? "输入数据库名"} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={passwordModalOpen}
        title={passwordTarget ? `修改 ${passwordTarget.name} 密码` : "修改密码"}
        okText="更新"
        cancelText="取消"
        confirmLoading={submitting}
        onCancel={() => {
          setPasswordModalOpen(false);
          setPasswordTarget(null);
          passwordForm.resetFields();
        }}
        onOk={() => void passwordForm.submit()}
        destroyOnClose
      >
        <Form form={passwordForm} layout="vertical" onFinish={updateAccountPassword}>
          <Form.Item label="新密码" name="password" rules={[{ required: true, message: "请输入新密码" }]}>
            <Input.Password placeholder="请输入新密码" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
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

function RedisConfigCard({
  label,
  value,
  actionLabel,
  onSubmit,
}: {
  label: string;
  value: string;
  actionLabel: string;
  onSubmit: (value: string) => Promise<void>;
}) {
  return (
    <Card className="glass-card" variant="borderless">
      <Space direction="vertical" size="middle" style={{ width: "100%" }}>
        <Typography.Text type="secondary">{label}</Typography.Text>
        <Typography.Text strong>{value}</Typography.Text>
        <Button onClick={() => void onSubmit(value)}>{actionLabel}</Button>
      </Space>
    </Card>
  );
}

function actionLabel(action: string) {
  switch (action) {
    case "start":
      return "启动";
    case "stop":
      return "停止";
    case "restart":
      return "重启";
    case "delete":
      return "删除";
    default:
      return action;
  }
}

function statusColor(status: string) {
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

function summaryLabel(key: string) {
  switch (key) {
    case "redis_version":
      return "Redis 版本";
    case "uptime_in_days":
      return "运行天数";
    case "used_memory_human":
      return "内存占用";
    case "connected_clients":
      return "连接数";
    default:
      return key;
  }
}
