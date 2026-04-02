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
  Descriptions,
  Empty,
  Space,
  Table,
  Tag,
  Typography,
} from "antd";
import type { DatabaseAccountItem, DatabaseEngine, DatabaseInstance, DatabaseOverview } from "../types";

type DatabaseContentProps = {
  overview: DatabaseOverview | null;
  loadingOverview: boolean;
  loadingInstances: boolean;
  activeEngine: DatabaseEngine;
  canManageCurrent: boolean;
  onRefreshCurrent: () => Promise<void>;
  onRunInstanceAction: (action: string) => Promise<void>;
  onOpenQuickCreate: () => void;
  onOpenDatabaseModal: () => void;
  onOpenAccountModal: () => void;
  onOpenGrantModal: () => void;
  onRequestDeleteDatabase: (name: string) => void;
  onRequestDeleteAccount: (account: DatabaseAccountItem) => void;
  onRequestPasswordReset: (account: DatabaseAccountItem) => void;
  onUpdateRedisConfig: (key: string, value: boolean | number | string) => Promise<void>;
  onGoInstall: () => void;
  statusColor: (status: string) => string;
  summaryLabel: (key: string) => string;
  canDeleteDatabase: (engine: DatabaseEngine, name: string) => boolean;
  canManageDatabaseAccount: (instance: DatabaseInstance, account: DatabaseAccountItem) => boolean;
  engineActionLabels: Record<DatabaseEngine, string>;
};

export function DatabaseContent({
  overview,
  loadingOverview,
  loadingInstances,
  activeEngine,
  canManageCurrent,
  onRefreshCurrent,
  onRunInstanceAction,
  onOpenQuickCreate,
  onOpenDatabaseModal,
  onOpenAccountModal,
  onOpenGrantModal,
  onRequestDeleteDatabase,
  onRequestDeleteAccount,
  onRequestPasswordReset,
  onUpdateRedisConfig,
  onGoInstall,
  statusColor,
  summaryLabel,
  canDeleteDatabase,
  canManageDatabaseAccount,
  engineActionLabels,
}: DatabaseContentProps) {
  if (!overview) {
    return (
      <Card className="glass-card" variant="borderless">
        <Empty
          description={
            loadingInstances
              ? `正在读取 ${engineActionLabels[activeEngine]} 实例`
              : `当前没有可管理的 ${engineActionLabels[activeEngine]} 实例，请先前往应用商店安装`
          }
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        >
          {!loadingInstances ? (
            <Button type="primary" onClick={onGoInstall}>
              去应用商店安装
            </Button>
          ) : null}
        </Empty>
      </Card>
    );
  }

  return (
    <>
      <Card className="glass-card" loading={loadingOverview}>
        <Space direction="vertical" size="middle" style={{ width: "100%" }}>
          <div className="database-status-header">
            <div>
              <Typography.Title level={4} style={{ margin: 0 }}>
                {overview.instance.name}
              </Typography.Title>
              <Typography.Paragraph type="secondary" style={{ margin: "6px 0 0" }}>
                {engineActionLabels[overview.instance.engine]} 实例状态和连接信息集中展示，常用操作直接在当前页完成。
              </Typography.Paragraph>
            </div>
            <Space wrap>
              <Button icon={<ReloadOutlined />} onClick={() => void onRefreshCurrent()}>
                刷新
              </Button>
              <Button icon={<PlayCircleOutlined />} onClick={() => void onRunInstanceAction("start")}>
                启动
              </Button>
              <Button icon={<PauseCircleOutlined />} onClick={() => void onRunInstanceAction("stop")}>
                停止
              </Button>
              <Button icon={<SyncOutlined />} onClick={() => void onRunInstanceAction("restart")}>
                重启
              </Button>
              <Button danger icon={<DeleteOutlined />} onClick={() => void onRunInstanceAction("delete")}>
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
              <Descriptions.Item label="引擎">{engineActionLabels[overview.instance.engine]}</Descriptions.Item>
              <Descriptions.Item label="主机">
                {overview.instance.connection.host}:{overview.instance.connection.port}
              </Descriptions.Item>
              <Descriptions.Item label="实例状态">
                <Tag color={statusColor(overview.instance.runtime.status)}>{overview.instance.runtime.status}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="关联容器">{overview.instance.runtime.containers.length}</Descriptions.Item>
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
                    当前引擎：{engineActionLabels[overview.instance.engine]}
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
                onSubmit={(nextValue) => onUpdateRedisConfig("appendonly", nextValue === "yes" ? "no" : "yes")}
              />
              <RedisConfigCard
                label="逻辑库数量"
                value={(overview.redis_config ?? []).find((item) => item.key === "databases")?.value ?? "-"}
                actionLabel="更新"
                onSubmit={(value) => {
                  const next = window.prompt("输入新的逻辑库数量", value);
                  if (!next) return Promise.resolve();
                  return onUpdateRedisConfig("databases", Number(next));
                }}
              />
              <RedisConfigCard
                label="访问密码"
                value="已托管"
                actionLabel="修改"
                onSubmit={() => {
                  const next = window.prompt("输入新的 Redis 密码");
                  if (!next) return Promise.resolve();
                  return onUpdateRedisConfig("password", next);
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
                <Button type="primary" disabled={!canManageCurrent} onClick={onOpenQuickCreate}>
                  快速创建
                </Button>
                <Button disabled={!canManageCurrent} onClick={onOpenDatabaseModal}>
                  仅创建数据库
                </Button>
                <Button disabled={!canManageCurrent} onClick={onOpenAccountModal}>
                  创建账号
                </Button>
                <Button disabled={!canManageCurrent} onClick={onOpenGrantModal}>
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
                      onClick={() => onRequestDeleteDatabase(record.name)}
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
                          onClick={() => onRequestPasswordReset(record)}
                        >
                          改密
                        </Button>
                        <Button
                          danger
                          size="small"
                          disabled={!canManageAccount}
                          onClick={() => onRequestDeleteAccount(record)}
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
