import {
  CloudServerOutlined,
  DeleteOutlined,
  EyeOutlined,
  FolderOpenOutlined,
  LinkOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
} from "@ant-design/icons";
import { Alert, Button, Card, Empty, Popconfirm, Space, Table, Tag, Typography } from "antd";
import type { ReactNode } from "react";
import type { Project, TemplateSpec } from "../../../shared/types";
import type {
  Certificate,
  OpenRestyStatus,
  Website,
} from "../types";

type WebsiteSectionsProps = {
  activeSection: "sites" | "certificates" | "environments";
  status: OpenRestyStatus | null;
  websites: Website[];
  certificates: Certificate[];
  phpProjects: Project[];
  phpProjectMap: Map<string, Project>;
  phpTemplate: TemplateSpec | null;
  phpUsageMap: Map<string, number>;
  loading: boolean;
  previewLoading: boolean;
  environmentActionKey: string;
  onRefresh: () => Promise<void>;
  onOpenCreateModal: () => void;
  onOpenCertificateModal: () => void;
  onOpenEnvironmentModal: () => void;
  onOpenEditModal: (website: Website) => void;
  onPreviewConfig: (website: Website) => Promise<void>;
  onDeleteWebsite: (website: Website) => Promise<void>;
  onDeleteCertificate: (certificate: Certificate) => Promise<void>;
  onRunEnvironmentAction: (project: Project, action: "start" | "stop" | "restart" | "delete") => Promise<void>;
  parseDomains: (raw?: string) => string[];
  formatWebsiteType: (value: Website["type"]) => ReactNode;
  formatRewriteLabel: (website: Website) => ReactNode;
  formatCertificateStatus: (certificate: Certificate) => ReactNode;
  formatProjectStatus: (status: string) => ReactNode;
  projectConfigText: (project: Project, key: string) => string;
  projectConfigNumber: (project: Project, key: string) => number;
  formatDateTime: (value?: string) => string;
};

export function WebsiteSections({
  activeSection,
  status,
  websites,
  certificates,
  phpProjects,
  phpProjectMap,
  phpTemplate,
  phpUsageMap,
  loading,
  previewLoading,
  environmentActionKey,
  onRefresh,
  onOpenCreateModal,
  onOpenCertificateModal,
  onOpenEnvironmentModal,
  onOpenEditModal,
  onPreviewConfig,
  onDeleteWebsite,
  onDeleteCertificate,
  onRunEnvironmentAction,
  parseDomains,
  formatWebsiteType,
  formatRewriteLabel,
  formatCertificateStatus,
  formatProjectStatus,
  projectConfigText,
  projectConfigNumber,
  formatDateTime,
}: WebsiteSectionsProps) {
  const certificateReady = Boolean(status?.ready && status?.certificate_ready);

  if (activeSection === "sites") {
    return (
      <Card
        className="glass-card"
        title="站点列表"
        extra={
          <Space wrap>
            <Button icon={<ReloadOutlined />} onClick={() => void onRefresh()}>
              刷新
            </Button>
            <Button type="primary" onClick={onOpenCreateModal} disabled={!status?.ready}>
              创建站点
            </Button>
          </Space>
        }
      >
        <Space direction="vertical" size="middle" style={{ width: "100%" }}>
          {status?.exists === false ? (
            <Alert
              showIcon
              type="info"
              message="固定 OpenResty 容器还没安装，请先到应用商店安装 OpenResty 模板。"
            />
          ) : null}

          {status?.exists && !status.ready ? (
            <Alert showIcon type="warning" message={status.message || "OpenResty 当前不可用"} />
          ) : null}

          <Table
            rowKey="id"
            loading={loading}
            dataSource={websites}
            locale={{ emptyText: <Empty description="还没有站点" /> }}
            columns={[
              { title: "站点名", dataIndex: "name" },
              {
                title: "类型",
                dataIndex: "type",
                render: (value: Website["type"]) => formatWebsiteType(value),
              },
              {
                title: "域名",
                render: (_, record) => {
                  const allDomains = [record.domain, ...parseDomains(record.domains_json)];
                  return allDomains.join(", ");
                },
              },
              {
                title: "目标",
                render: (_, record) => {
                  if (record.type === "proxy") {
                    return (
                      <Space size="small">
                        <LinkOutlined />
                        {record.proxy_pass}
                      </Space>
                    );
                  }

                  const phpProject = record.php_project_id ? phpProjectMap.get(record.php_project_id) : undefined;
                  return (
                    <Space direction="vertical" size={2}>
                      <Space size="small">
                        <FolderOpenOutlined />
                        {record.root_path}
                      </Space>
                      {record.type === "php" ? (
                        <Typography.Text type="secondary">
                          PHP 环境：{phpProject?.name || "已删除"}
                          {record.php_port ? ` / 127.0.0.1:${record.php_port}` : ""}
                        </Typography.Text>
                      ) : null}
                    </Space>
                  );
                },
              },
              {
                title: "伪静态",
                render: (_, record) => formatRewriteLabel(record),
              },
              {
                title: "状态",
                dataIndex: "status",
                render: (value: string) => <Tag>{value}</Tag>,
              },
              {
                title: "操作",
                width: 260,
                render: (_, record) => (
                  <Space wrap>
                    <Button size="small" onClick={() => onOpenEditModal(record)}>
                      配置
                    </Button>
                    <Button
                      size="small"
                      icon={<EyeOutlined />}
                      loading={previewLoading}
                      onClick={() => void onPreviewConfig(record)}
                    >
                      预览
                    </Button>
                    <Popconfirm
                      title={`删除站点 ${record.name}`}
                      description="会删除站点配置并 reload OpenResty，站点目录内容不会自动删除。"
                      okText="删除"
                      cancelText="取消"
                      okButtonProps={{ danger: true }}
                      onConfirm={() => void onDeleteWebsite(record)}
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
        </Space>
      </Card>
    );
  }

  if (activeSection === "certificates") {
    return (
      <>
        <Card className="glass-card">
          <Space direction="vertical" size="middle" style={{ width: "100%" }}>
            <div className="page-inline-bar">
              <div>
                <Typography.Title level={4} style={{ marginTop: 0 }}>
                  证书管理
                </Typography.Title>
                <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                  当前版本支持 Let&apos;s Encrypt 单域名证书申请、列表查看和删除。
                </Typography.Paragraph>
              </div>
              <Space>
                <Tag icon={<SafetyCertificateOutlined />} color={certificateReady ? "blue" : "orange"}>
                  {certificateReady ? "HTTP-01" : "需补证书挂载"}
                </Tag>
                <Button type="primary" onClick={onOpenCertificateModal} disabled={!certificateReady}>
                  申请证书
                </Button>
              </Space>
            </div>

            <Alert
              showIcon
              type={certificateReady ? "info" : "warning"}
              message={
                certificateReady
                  ? "第一版仅支持 Let's Encrypt + HTTP-01 + 单域名证书。"
                  : (status?.message || "OpenResty 当前不可用")
              }
              description={
                certificateReady
                  ? "如果站点主域名和证书域名一致，签发成功后会自动启用 HTTPS。多域名站点暂不支持自动启用 HTTPS。"
                  : "请确认 OpenResty 已运行，并已按最新模板挂载证书目录后再申请证书。"
              }
            />
          </Space>
        </Card>

        <Card className="glass-card" title="证书列表">
          <Table
            rowKey="id"
            loading={loading}
            dataSource={certificates}
            locale={{ emptyText: <Empty description="还没有证书" /> }}
            columns={[
              { title: "域名", dataIndex: "domain" },
              { title: "签发方", dataIndex: "provider", render: (value: string) => value || "-" },
              {
                title: "绑定站点",
                render: (_, record) => record.website_name || <Tag>未绑定</Tag>,
              },
              {
                title: "到期时间",
                dataIndex: "expires_at",
                render: (value: string) => formatDateTime(value),
              },
              {
                title: "状态",
                render: (_, record) => formatCertificateStatus(record),
              },
              {
                title: "错误",
                dataIndex: "last_error",
                render: (value: string) => value || "-",
              },
              {
                title: "操作",
                width: 120,
                render: (_, record) => (
                  <Popconfirm
                    title={`删除证书 ${record.domain}`}
                    description="会删除证书文件，并把同域名网站切回 HTTP。"
                    okText="删除"
                    cancelText="取消"
                    okButtonProps={{ danger: true }}
                    onConfirm={() => void onDeleteCertificate(record)}
                  >
                    <Button danger size="small" icon={<DeleteOutlined />}>
                      删除
                    </Button>
                  </Popconfirm>
                ),
              },
            ]}
          />
        </Card>
      </>
    );
  }

  return (
    <>
      <Card className="glass-card">
        <Space direction="vertical" size="middle" style={{ width: "100%" }}>
          <div className="page-inline-bar">
            <div>
              <Typography.Title level={4} style={{ marginTop: 0 }}>
                PHP 环境
              </Typography.Title>
              <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                PHP 环境直接复用项目系统管理。创建后，网站页可直接绑定对应 FPM 实例。
              </Typography.Paragraph>
            </div>
            <Space>
              <Tag icon={<CloudServerOutlined />} color="green">
                {phpProjects.length} 个环境
              </Tag>
              <Button onClick={() => void onRefresh()}>刷新</Button>
              <Button type="primary" onClick={onOpenEnvironmentModal} disabled={!phpTemplate}>
                新建环境
              </Button>
            </Space>
          </div>

          {!phpTemplate ? (
            <Alert
              showIcon
              type="warning"
              message="未找到 php-fpm 模板"
              description="请确认模板目录已包含 php-fpm，或重新加载服务后再试。"
            />
          ) : (
            <Alert
              showIcon
              type="info"
              message="当前版本支持 PHP 8.1 / 8.2 / 8.3。"
              description="每个环境都会把 OpenResty 站点目录挂载进容器，站点绑定后即可直接解析 PHP。"
            />
          )}
        </Space>
      </Card>

      <Card className="glass-card" title="环境列表">
        <Table
          rowKey="id"
          loading={loading}
          dataSource={phpProjects}
          locale={{ emptyText: <Empty description="还没有 PHP 环境" /> }}
          columns={[
            { title: "环境名", dataIndex: "name" },
            {
              title: "PHP 版本",
              render: (_, record) => <Tag color="blue">{projectConfigText(record, "php_version") || "-"}</Tag>,
            },
            {
              title: "FPM 端口",
              render: (_, record) => {
                const port = projectConfigNumber(record, "port");
                return port ? `127.0.0.1:${port}` : "-";
              },
            },
            {
              title: "绑定站点",
              render: (_, record) => {
                const usage = phpUsageMap.get(record.id) ?? 0;
                return usage > 0 ? <Tag color="gold">{usage} 个站点</Tag> : <Tag>未使用</Tag>;
              },
            },
            {
              title: "状态",
              render: (_, record) => formatProjectStatus(record.runtime.status || record.status),
            },
            {
              title: "操作",
              width: 300,
              render: (_, record) => {
                const usage = phpUsageMap.get(record.id) ?? 0;
                return (
                  <Space wrap>
                    <Button
                      size="small"
                      loading={environmentActionKey === `${record.id}:start`}
                      onClick={() => void onRunEnvironmentAction(record, "start")}
                    >
                      启动
                    </Button>
                    <Button
                      size="small"
                      loading={environmentActionKey === `${record.id}:stop`}
                      onClick={() => void onRunEnvironmentAction(record, "stop")}
                    >
                      停止
                    </Button>
                    <Button
                      size="small"
                      loading={environmentActionKey === `${record.id}:restart`}
                      onClick={() => void onRunEnvironmentAction(record, "restart")}
                    >
                      重启
                    </Button>
                    <Popconfirm
                      title={`删除环境 ${record.name}`}
                      description={
                        usage > 0
                          ? `当前有 ${usage} 个站点绑定此环境，删除后这些站点会失效。`
                          : "会删除对应 PHP-FPM 项目实例。"
                      }
                      okText="删除"
                      cancelText="取消"
                      okButtonProps={{ danger: true }}
                      onConfirm={() => void onRunEnvironmentAction(record, "delete")}
                    >
                      <Button danger size="small" loading={environmentActionKey === `${record.id}:delete`}>
                        删除
                      </Button>
                    </Popconfirm>
                  </Space>
                );
              },
            },
          ]}
        />
      </Card>
    </>
  );
}
