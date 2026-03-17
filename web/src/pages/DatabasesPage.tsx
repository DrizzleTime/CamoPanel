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
  Checkbox,
  Descriptions,
  Empty,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useShellHeader } from "../components/ShellHeaderContext";
import { apiRequest } from "../lib/api";
import type {
  DatabaseAccountItem,
  DatabaseEngine,
  DatabaseInstance,
  DatabaseOverview,
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
  const navigate = useNavigate();
  const [activeEngine, setActiveEngine] = useState<DatabaseEngine>("mysql");
  const [instances, setInstances] = useState<DatabaseInstance[]>([]);
  const [selectedInstanceId, setSelectedInstanceId] = useState<string>();
  const [overview, setOverview] = useState<DatabaseOverview | null>(null);
  const [loadingInstances, setLoadingInstances] = useState(true);
  const [loadingOverview, setLoadingOverview] = useState(false);
  const [quickCreateOpen, setQuickCreateOpen] = useState(false);
  const [databaseModalOpen, setDatabaseModalOpen] = useState(false);
  const [accountModalOpen, setAccountModalOpen] = useState(false);
  const [grantModalOpen, setGrantModalOpen] = useState(false);
  const [passwordModalOpen, setPasswordModalOpen] = useState(false);
  const [deleteDatabaseTarget, setDeleteDatabaseTarget] = useState<string | null>(null);
  const [deleteAccountTarget, setDeleteAccountTarget] = useState<DatabaseAccountItem | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [passwordTarget, setPasswordTarget] = useState<DatabaseAccountItem | null>(null);
  const [quickCreateForm] = Form.useForm();
  const [databaseForm] = Form.useForm();
  const [accountForm] = Form.useForm();
  const [grantForm] = Form.useForm();
  const [passwordForm] = Form.useForm();
  const [deleteDatabaseForm] = Form.useForm();
  const deleteAccountEnabled = Form.useWatch("deleteAccount", deleteDatabaseForm) ?? true;

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
    try {
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
    } catch (error) {
      message.error(getErrorMessage(error));
    }
  };

  const createQuickWorkspace = async (values: { name: string; password: string }) => {
    if (!overview) return;

    let databaseCreated = false;
    setSubmitting(true);
    try {
      await apiRequest(`/api/databases/${overview.instance.id}/databases`, {
        method: "POST",
        body: JSON.stringify({ name: values.name }),
      });
      databaseCreated = true;
      await apiRequest(`/api/databases/${overview.instance.id}/accounts`, {
        method: "POST",
        body: JSON.stringify({
          name: values.name,
          password: values.password,
          database_name: values.name,
        }),
      });
      message.success("业务库已创建，可直接使用同名账号连接");
      setQuickCreateOpen(false);
      quickCreateForm.resetFields();
      await loadOverview(overview.instance.id);
    } catch (error) {
      if (databaseCreated) {
        message.warning("数据库已创建，但账号创建或授权失败，请检查账号列表后重试。");
      } else {
        message.error(getErrorMessage(error));
      }
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
    } catch (error) {
      message.error(getErrorMessage(error));
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
    } catch (error) {
      message.error(getErrorMessage(error));
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
      await loadOverview(overview.instance.id);
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setSubmitting(false);
    }
  };

  const updateAccountPassword = async (values: { password: string }) => {
    if (!overview || !passwordTarget) return;
    setSubmitting(true);
    try {
      await apiRequest(`/api/databases/${overview.instance.id}/accounts/${encodeURIComponent(passwordTarget.name)}/password`, {
        method: "POST",
        body: JSON.stringify(values),
      });
      message.success("密码已更新");
      setPasswordModalOpen(false);
      setPasswordTarget(null);
      passwordForm.resetFields();
      await loadOverview(overview.instance.id);
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setSubmitting(false);
    }
  };

  const deleteDatabase = async (values: { deleteAccount: boolean; accountName?: string }) => {
    if (!overview || !deleteDatabaseTarget) return;

    let databaseDeleted = false;
    const accountName = (values.accountName || "").trim();

    setSubmitting(true);
    try {
      await apiRequest(
        `/api/databases/${overview.instance.id}/databases/${encodeURIComponent(deleteDatabaseTarget)}`,
        { method: "DELETE" },
      );
      databaseDeleted = true;

      if (values.deleteAccount && accountName) {
        await apiRequest(
          `/api/databases/${overview.instance.id}/accounts/${encodeURIComponent(accountName)}`,
          { method: "DELETE" },
        );
      }

      message.success(values.deleteAccount && accountName ? "数据库和账号已删除" : "数据库已删除");
      setDeleteDatabaseTarget(null);
      deleteDatabaseForm.resetFields();
      await loadOverview(overview.instance.id);
    } catch (error) {
      if (databaseDeleted) {
        message.warning("数据库已删除，但账号删除失败，请检查账号列表。");
        setDeleteDatabaseTarget(null);
        deleteDatabaseForm.resetFields();
        await loadOverview(overview.instance.id);
      } else {
        message.error(getErrorMessage(error));
      }
    } finally {
      setSubmitting(false);
    }
  };

  const deleteAccount = async () => {
    if (!overview || !deleteAccountTarget) return;
    setSubmitting(true);
    try {
      await apiRequest(
        `/api/databases/${overview.instance.id}/accounts/${encodeURIComponent(deleteAccountTarget.name)}`,
        { method: "DELETE" },
      );
      message.success("账号已删除");
      setDeleteAccountTarget(null);
      await loadOverview(overview.instance.id);
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setSubmitting(false);
    }
  };

  const updateRedisConfig = async (key: string, value: boolean | number | string) => {
    if (!overview) return;
    try {
      await apiRequest(`/api/databases/${overview.instance.id}/redis/config`, {
        method: "POST",
        body: JSON.stringify({ key, value }),
      });
      message.success("Redis 配置已更新并重建实例");
      await refreshCurrent();
    } catch (error) {
      message.error(getErrorMessage(error));
    }
  };

  const databaseOptions = overview?.databases?.map((item) => ({
    label: item.name,
    value: item.name,
  })) ?? [];

  const accountOptions = overview?.accounts?.map((item) => ({
    label: item.host ? `${item.name} (${item.host})` : item.name,
    value: item.name,
  })) ?? [];

  const canManageCurrent = overview?.instance.runtime.status === "running";

  return (
    <div className="page-grid database-page">
      <div className="page-inline-bar database-toolbar">
        <Space wrap>
          <Select
            value={selectedInstanceId}
            placeholder={`选择 ${ENGINE_ACTION_LABELS[activeEngine]} 实例`}
            loading={loadingInstances}
            style={{ minWidth: 280 }}
            options={instances.map((item) => ({
              label: `${item.name} (${item.connection.host}:${item.connection.port})`,
              value: item.id,
            }))}
            onChange={(value) => setSelectedInstanceId(value)}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void loadInstances(activeEngine, selectedInstanceId)}>
            刷新实例
          </Button>
        </Space>
        <Button type="primary" onClick={() => navigate("/app/store")}>
          去应用商店安装
        </Button>
      </div>

      <div className="database-main">
        {overview ? (
          <>
            <Card className="glass-card" loading={loadingOverview}>
              <Space direction="vertical" size="middle" style={{ width: "100%" }}>
                <div className="database-status-header">
                  <div>
                    <Typography.Title level={4} style={{ margin: 0 }}>
                      {overview.instance.name}
                    </Typography.Title>
                    <Typography.Paragraph type="secondary" style={{ margin: "6px 0 0" }}>
                      {ENGINE_ACTION_LABELS[overview.instance.engine]} 实例状态和连接信息集中展示，常用操作直接在当前页完成。
                    </Typography.Paragraph>
                  </div>
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
                </div>

                <Alert
                  showIcon
                  type={overview.notice ? "warning" : "success"}
                  message={overview.notice || "实例运行中，可以直接管理数据库内部对象。"}
                  description={
                    <Space direction="vertical" size="small">
                      <Typography.Text type="secondary">
                        连接地址：{overview.instance.connection.host}:{overview.instance.connection.port}
                      </Typography.Text>
                      {overview.instance.connection.admin_username ? (
                        <Typography.Text type="secondary">
                          管理账号：{overview.instance.connection.admin_username}
                        </Typography.Text>
                      ) : null}
                      {overview.instance.connection.app_username ? (
                        <Typography.Text type="secondary">
                          默认业务账号：{overview.instance.connection.app_username}
                        </Typography.Text>
                      ) : null}
                      {overview.instance.connection.default_database ? (
                        <Typography.Text type="secondary">
                          默认库：{overview.instance.connection.default_database}
                        </Typography.Text>
                      ) : null}
                    </Space>
                  }
                />

                <div className="database-overview-grid">
                  <Descriptions bordered size="small" column={1} className="database-descriptions">
                    <Descriptions.Item label="引擎">{ENGINE_ACTION_LABELS[overview.instance.engine]}</Descriptions.Item>
                    <Descriptions.Item label="主机">
                      {overview.instance.connection.host}:{overview.instance.connection.port}
                    </Descriptions.Item>
                    <Descriptions.Item label="实例状态">
                      <Tag color={statusColor(overview.instance.runtime.status)}>
                        {overview.instance.runtime.status}
                      </Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="关联容器">
                      {overview.instance.runtime.containers.length}
                    </Descriptions.Item>
                    <Descriptions.Item label="密码状态">
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
                          当前引擎：{ENGINE_ACTION_LABELS[overview.instance.engine]}
                        </Typography.Text>
                      </div>
                      {overview.summary ? (
                        <div className="database-summary-kpis">
                          {Object.entries(overview.summary).map(([key, value]) => (
                            <div key={key} className="database-summary-kpi">
                              <Typography.Text type="secondary">{summaryLabel(key)}</Typography.Text>
                              <Typography.Text strong>{value}</Typography.Text>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <Typography.Text type="secondary">
                          关系型数据库优先走“快速创建”，会自动完成数据库、账号和授权。
                        </Typography.Text>
                      )}
                    </Space>
                  </Card>
                </div>
              </Space>
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
                <Card className="glass-card">
                  <div className="page-inline-bar database-actions">
                    <Space wrap>
                      <Button
                        type="primary"
                        disabled={!canManageCurrent}
                        onClick={() => {
                          quickCreateForm.resetFields();
                          setQuickCreateOpen(true);
                        }}
                      >
                        快速创建
                      </Button>
                      <Button
                        disabled={!canManageCurrent}
                        onClick={() => {
                          databaseForm.resetFields();
                          setDatabaseModalOpen(true);
                        }}
                      >
                        仅创建数据库
                      </Button>
                      <Button
                        disabled={!canManageCurrent}
                        onClick={() => {
                          accountForm.resetFields();
                          setAccountModalOpen(true);
                        }}
                      >
                        创建账号
                      </Button>
                      <Button
                        disabled={!canManageCurrent}
                        onClick={() => {
                          grantForm.resetFields();
                          setGrantModalOpen(true);
                        }}
                      >
                        手动授权
                      </Button>
                    </Space>
                    <Typography.Text type="secondary">
                      快速创建会自动生成同名数据库和账号，并完成授权。
                    </Typography.Text>
                  </div>
                </Card>

                <Card className="glass-card" title="数据库列表" loading={loadingOverview}>
                  <Table
                    rowKey="name"
                    pagination={false}
                    dataSource={overview.databases ?? []}
                    locale={{ emptyText: <Empty description="当前还没有业务数据库" /> }}
                    columns={[
                      { title: "数据库名", dataIndex: "name" },
                      {
                        title: "操作",
                        width: 180,
                        render: (_, record: { name: string }) => (
                          <Button
                            danger
                            size="small"
                            disabled={!canDeleteDatabase(overview.instance.engine, record.name)}
                            onClick={() => {
                              deleteDatabaseForm.setFieldsValue({
                                deleteAccount: true,
                                accountName: record.name,
                              });
                              setDeleteDatabaseTarget(record.name);
                            }}
                          >
                            删除数据库
                          </Button>
                        ),
                      },
                    ]}
                  />
                </Card>

                <Card className="glass-card" title="账号管理" loading={loadingOverview}>
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
                        title: "操作",
                        width: 240,
                        render: (_, record) => {
                          const canManageAccount = canManageDatabaseAccount(overview.instance, record);

                          return (
                            <Space wrap>
                              <Button
                                size="small"
                                disabled={!canManageAccount}
                                onClick={() => {
                                  setPasswordTarget(record);
                                  passwordForm.resetFields();
                                  setPasswordModalOpen(true);
                                }}
                              >
                                改密
                              </Button>
                              <Button
                                danger
                                size="small"
                                disabled={!canManageAccount}
                                onClick={() => setDeleteAccountTarget(record)}
                              >
                                删除
                              </Button>
                            </Space>
                          );
                        },
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
              description={
                loadingInstances
                  ? `正在读取 ${ENGINE_ACTION_LABELS[activeEngine]} 实例`
                  : `当前没有可管理的 ${ENGINE_ACTION_LABELS[activeEngine]} 实例，请先前往应用商店安装`
              }
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            >
              {!loadingInstances ? (
                <Button type="primary" onClick={() => navigate("/app/store")}>
                  去应用商店安装
                </Button>
              ) : null}
            </Empty>
          </Card>
        )}
      </div>

      <Modal
        open={quickCreateOpen}
        title="快速创建业务库"
        okText="立即创建"
        cancelText="取消"
        confirmLoading={submitting}
        onCancel={() => {
          setQuickCreateOpen(false);
          quickCreateForm.resetFields();
        }}
        onOk={() => void quickCreateForm.submit()}
        destroyOnClose
      >
        <Form form={quickCreateForm} layout="vertical" onFinish={createQuickWorkspace}>
          <Alert
            showIcon
            type="info"
            message="会自动创建同名数据库和同名账号，并完成授权。"
            style={{ marginBottom: 16 }}
          />
          <Form.Item
            label="业务名"
            name="name"
            rules={[{ required: true, message: "请输入业务名" }]}
            extra="会同时用作数据库名和账号名。"
          >
            <Input placeholder="app_prod" />
          </Form.Item>
          <Form.Item label="密码" name="password" rules={[{ required: true, message: "请输入密码" }]}>
            <Input.Password placeholder="请输入密码" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={databaseModalOpen}
        title="仅创建数据库"
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
            <Select allowClear showSearch options={databaseOptions} placeholder="可选，创建后直接完成授权" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={grantModalOpen}
        title="手动授权"
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
            <Select showSearch options={accountOptions} placeholder="选择账号" />
          </Form.Item>
          <Form.Item label="数据库名" name="database_name" rules={[{ required: true, message: "请选择数据库" }]}>
            <Select showSearch options={databaseOptions} placeholder="选择数据库" />
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

      <Modal
        open={!!deleteDatabaseTarget}
        title={deleteDatabaseTarget ? `删除数据库 ${deleteDatabaseTarget}` : "删除数据库"}
        okText="确认删除"
        okButtonProps={{ danger: true }}
        cancelText="取消"
        confirmLoading={submitting}
        onCancel={() => {
          setDeleteDatabaseTarget(null);
          deleteDatabaseForm.resetFields();
        }}
        onOk={() => void deleteDatabaseForm.submit()}
        destroyOnClose
      >
        <Form form={deleteDatabaseForm} layout="vertical" onFinish={deleteDatabase}>
          <Alert
            showIcon
            type="warning"
            message="删除数据库后，库内数据会一并丢失。"
            style={{ marginBottom: 16 }}
          />
          <Form.Item name="deleteAccount" valuePropName="checked">
            <Checkbox>同步删除账号</Checkbox>
          </Form.Item>
          <Form.Item label="账号名" name="accountName">
            <Input disabled={!deleteAccountEnabled} placeholder="留空表示只删除数据库" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={!!deleteAccountTarget}
        title={deleteAccountTarget ? `删除账号 ${deleteAccountTarget.name}` : "删除账号"}
        okText="确认删除"
        okButtonProps={{ danger: true }}
        cancelText="取消"
        confirmLoading={submitting}
        onCancel={() => setDeleteAccountTarget(null)}
        onOk={() => void deleteAccount()}
        destroyOnClose
      >
        <Alert
          showIcon
          type="warning"
          message="删除账号后，将无法再使用该账号连接数据库。"
        />
      </Modal>
    </div>
  );
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

function canDeleteDatabase(engine: DatabaseEngine, name: string) {
  const normalized = name.trim().toLowerCase();
  if (engine === "mysql") {
    return !["information_schema", "mysql", "performance_schema", "sys"].includes(normalized);
  }
  if (engine === "postgres") {
    return !["postgres", "template0", "template1"].includes(normalized);
  }
  return false;
}

function canManageDatabaseAccount(instance: DatabaseInstance, account: DatabaseAccountItem) {
  if (instance.connection.admin_username && account.name === instance.connection.admin_username) {
    return false;
  }
  if (instance.engine === "mysql" && account.host && account.host !== "%") {
    return false;
  }
  return true;
}

function getErrorMessage(error: unknown) {
  if (error instanceof Error && error.message.trim()) {
    return error.message;
  }
  return "操作失败";
}
