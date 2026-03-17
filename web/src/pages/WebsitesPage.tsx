import {
  CloudServerOutlined,
  DeleteOutlined,
  EyeOutlined,
  FolderOpenOutlined,
  LinkOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Radio,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import { useEffect, useMemo, useState } from "react";
import { useShellHeader } from "../components/ShellHeaderContext";
import { apiRequest } from "../lib/api";
import type { Certificate, OpenRestyStatus, Project, TemplateSpec, Website } from "../lib/types";

type WebsiteSection = "sites" | "certificates" | "environments";
type WebsiteFormValues = {
  name: string;
  type: "static" | "php" | "proxy";
  domain: string;
  domains: string;
  root_path?: string;
  index_files?: string;
  proxy_pass?: string;
  php_project_id?: string;
  rewrite_mode: "off" | "preset" | "custom";
  rewrite_preset?: string;
  rewrite_rules?: string;
};
type CertificateFormValues = {
  domain: string;
  email: string;
};
type EnvironmentFormValues = {
  name: string;
  php_version: "8.1" | "8.2" | "8.3";
  port: number;
};

const WEBSITE_SECTIONS: Array<{ key: WebsiteSection; label: string }> = [
  { key: "sites", label: "网站" },
  { key: "certificates", label: "证书" },
  { key: "environments", label: "环境" },
];

const REWRITE_PRESET_OPTIONS = [
  { value: "spa", label: "SPA History" },
  { value: "front_controller", label: "Front Controller" },
];

export function WebsitesPage() {
  const [activeSection, setActiveSection] = useState<WebsiteSection>("sites");
  const [status, setStatus] = useState<OpenRestyStatus | null>(null);
  const [websites, setWebsites] = useState<Website[]>([]);
  const [certificates, setCertificates] = useState<Certificate[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [templates, setTemplates] = useState<TemplateSpec[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [certificateModalOpen, setCertificateModalOpen] = useState(false);
  const [environmentModalOpen, setEnvironmentModalOpen] = useState(false);
  const [previewOpen, setPreviewOpen] = useState(false);
  const [previewContent, setPreviewContent] = useState("");
  const [previewTitle, setPreviewTitle] = useState("配置预览");
  const [editingWebsite, setEditingWebsite] = useState<Website | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [certificateSubmitting, setCertificateSubmitting] = useState(false);
  const [environmentSubmitting, setEnvironmentSubmitting] = useState(false);
  const [environmentActionKey, setEnvironmentActionKey] = useState("");
  const [previewLoading, setPreviewLoading] = useState(false);
  const [form] = Form.useForm<WebsiteFormValues>();
  const [certificateForm] = Form.useForm<CertificateFormValues>();
  const [environmentForm] = Form.useForm<EnvironmentFormValues>();
  const websiteType = Form.useWatch("type", form) ?? "static";
  const rewriteMode = Form.useWatch("rewrite_mode", form) ?? "off";
  const certificateReady = Boolean(status?.ready && status?.certificate_ready);
  const phpProjects = useMemo(
    () => projects.filter((item) => item.template_id === "php-fpm"),
    [projects],
  );
  const phpProjectMap = useMemo(
    () => new Map(phpProjects.map((item) => [item.id, item])),
    [phpProjects],
  );
  const phpTemplate = useMemo(
    () => templates.find((item) => item.id === "php-fpm") ?? null,
    [templates],
  );
  const phpUsageMap = useMemo(() => {
    const usage = new Map<string, number>();
    websites.forEach((website) => {
      if (!website.php_project_id) {
        return;
      }
      usage.set(website.php_project_id, (usage.get(website.php_project_id) ?? 0) + 1);
    });
    return usage;
  }, [websites]);

  const headerContent = useMemo(
    () => (
      <div className="shell-header-tabs">
        {WEBSITE_SECTIONS.map((item) => (
          <Button
            key={item.key}
            size="small"
            type={activeSection === item.key ? "primary" : "text"}
            className="shell-header-tab"
            onClick={() => setActiveSection(item.key)}
          >
            {item.label}
          </Button>
        ))}
      </div>
    ),
    [activeSection],
  );

  useShellHeader(headerContent);

  const loadData = async () => {
    setLoading(true);
    try {
      const [statusResponse, websiteResponse, certificateResponse, projectResponse, templateResponse] =
        await Promise.all([
        apiRequest<OpenRestyStatus>("/api/openresty/status"),
        apiRequest<{ items: Website[] }>("/api/websites"),
        apiRequest<{ items: Certificate[] }>("/api/certificates"),
        apiRequest<{ items: Project[] }>("/api/projects"),
        apiRequest<{ items: TemplateSpec[] }>("/api/templates"),
      ]);
      setStatus(statusResponse);
      setWebsites(websiteResponse.items);
      setCertificates(certificateResponse.items);
      setProjects(projectResponse.items);
      setTemplates(templateResponse.items);
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadData();
  }, []);

  const openCreateModal = () => {
    setEditingWebsite(null);
    form.resetFields();
    form.setFieldsValue({
      type: "static",
      domains: "",
      rewrite_mode: "off",
      index_files: "index.html index.htm",
    });
    setModalOpen(true);
  };

  const openCertificateModal = () => {
    certificateForm.resetFields();
    certificateForm.setFieldsValue({ email: "", domain: "" });
    setCertificateModalOpen(true);
  };

  const openEditModal = (website: Website) => {
    setEditingWebsite(website);
    form.resetFields();
    form.setFieldsValue({
      name: website.name,
      type: (website.site_mode ?? website.type) as "static" | "php" | "proxy",
      domain: website.domain,
      domains: parseDomainsText(website.domains_json),
      root_path: website.root_path,
      index_files:
        website.index_files ||
        ((website.site_mode ?? website.type) === "php" ? "index.php index.html index.htm" : "index.html index.htm"),
      proxy_pass: website.proxy_pass,
      php_project_id: website.php_project_id,
      rewrite_mode: (website.rewrite_mode ?? "off") as "off" | "preset" | "custom",
      rewrite_preset: website.rewrite_preset,
      rewrite_rules: website.rewrite_rules,
    });
    setModalOpen(true);
  };

  const submitWebsite = async (values: WebsiteFormValues) => {
    const payload = {
      name: values.name,
      type: values.type,
      domain: values.domain,
      domains: splitDomainInput(values.domains),
      root_path: values.type === "proxy" ? "" : values.root_path ?? "",
      index_files: values.type === "proxy" ? "" : values.index_files ?? "",
      proxy_pass: values.type === "proxy" ? values.proxy_pass ?? "" : "",
      php_project_id: values.type === "php" ? values.php_project_id ?? "" : "",
      rewrite_mode: values.type === "proxy" ? "off" : values.rewrite_mode,
      rewrite_preset: values.type !== "proxy" && values.rewrite_mode === "preset" ? values.rewrite_preset ?? "" : "",
      rewrite_rules: values.type !== "proxy" && values.rewrite_mode === "custom" ? values.rewrite_rules ?? "" : "",
    };

    setSubmitting(true);
    try {
      if (editingWebsite) {
        await apiRequest<{ website: Website }>(`/api/websites/${editingWebsite.id}`, {
          method: "PUT",
          body: JSON.stringify(payload),
        });
        message.success("站点配置已更新");
      } else {
        await apiRequest<{ website: Website }>("/api/websites", {
          method: "POST",
          body: JSON.stringify(payload),
        });
        message.success("站点创建完成");
      }
      setModalOpen(false);
      setEditingWebsite(null);
      form.resetFields();
      await loadData();
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setSubmitting(false);
    }
  };

  const previewWebsiteConfig = async (website: Website) => {
    setPreviewLoading(true);
    try {
      const response = await apiRequest<{ config: string }>(`/api/websites/${website.id}/config-preview`);
      setPreviewTitle(`${website.name} 配置预览`);
      setPreviewContent(response.config);
      setPreviewOpen(true);
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setPreviewLoading(false);
    }
  };

  const deleteWebsite = async (website: Website) => {
    setSubmitting(true);
    try {
      await apiRequest(`/api/websites/${website.id}`, { method: "DELETE" });
      message.success("站点已删除");
      await loadData();
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setSubmitting(false);
    }
  };

  const submitCertificate = async (values: CertificateFormValues) => {
    setCertificateSubmitting(true);
    try {
      await apiRequest<{ certificate: Certificate }>("/api/certificates", {
        method: "POST",
        body: JSON.stringify({
          domain: values.domain,
          email: values.email,
        }),
      });
      message.success("证书申请完成");
      setCertificateModalOpen(false);
      certificateForm.resetFields();
      await loadData();
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setCertificateSubmitting(false);
    }
  };

  const deleteCertificate = async (certificate: Certificate) => {
    setCertificateSubmitting(true);
    try {
      await apiRequest(`/api/certificates/${certificate.id}`, { method: "DELETE" });
      message.success("证书已删除");
      await loadData();
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setCertificateSubmitting(false);
    }
  };

  const openEnvironmentModal = () => {
    const defaultVersion = getTemplateDefaultText(phpTemplate, "php_version", "8.3");
    const defaultPort = getTemplateDefaultNumber(phpTemplate, "port", 9000);
    environmentForm.resetFields();
    environmentForm.setFieldsValue({
      name: `php-${defaultVersion.replace(".", "")}`,
      php_version: normalizePHPVersion(defaultVersion),
      port: nextAvailablePort(phpProjects, defaultPort),
    });
    setEnvironmentModalOpen(true);
  };

  const submitEnvironment = async (values: EnvironmentFormValues) => {
    if (!phpTemplate) {
      message.error("缺少 php-fpm 模板，无法创建环境");
      return;
    }
    setEnvironmentSubmitting(true);
    try {
      await apiRequest<{ project: Project }>("/api/projects", {
        method: "POST",
        body: JSON.stringify({
          name: values.name,
          template_id: phpTemplate.id,
          parameters: {
            php_version: values.php_version,
            port: values.port,
          },
        }),
      });
      message.success("PHP 环境已创建");
      setEnvironmentModalOpen(false);
      environmentForm.resetFields();
      await loadData();
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setEnvironmentSubmitting(false);
    }
  };

  const runEnvironmentAction = async (project: Project, action: "start" | "stop" | "restart" | "delete") => {
    setEnvironmentActionKey(`${project.id}:${action}`);
    try {
      await apiRequest(`/api/projects/${project.id}/actions`, {
        method: "POST",
        body: JSON.stringify({ action }),
      });
      message.success(`PHP 环境已${actionLabel(action)}`);
      await loadData();
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setEnvironmentActionKey("");
    }
  };

  const handleWebsiteTypeChange = (nextType: WebsiteFormValues["type"]) => {
    if (nextType === "php") {
      if (!form.getFieldValue("php_project_id") && phpProjects[0]) {
        form.setFieldValue("php_project_id", phpProjects[0].id);
      }
      if (!form.getFieldValue("index_files") || form.getFieldValue("index_files") === "index.html index.htm") {
        form.setFieldValue("index_files", "index.php index.html index.htm");
      }
      if (form.getFieldValue("rewrite_mode") === "off" || !form.getFieldValue("rewrite_mode")) {
        form.setFieldValue("rewrite_mode", "preset");
        form.setFieldValue("rewrite_preset", "front_controller");
      }
      return;
    }
    if (nextType === "static" && !form.getFieldValue("index_files")) {
      form.setFieldValue("index_files", "index.html index.htm");
    }
  };

  return (
    <div className="page-grid">
      {activeSection === "sites" ? (
        <Card
          className="glass-card"
          title="站点列表"
          extra={
            <Space wrap>
              <Button icon={<ReloadOutlined />} onClick={() => void loadData()}>
                刷新
              </Button>
              <Button type="primary" onClick={openCreateModal} disabled={!status?.ready}>
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
                      <Button size="small" onClick={() => openEditModal(record)}>
                        配置
                      </Button>
                      <Button
                        size="small"
                        icon={<EyeOutlined />}
                        loading={previewLoading}
                        onClick={() => void previewWebsiteConfig(record)}
                      >
                        预览
                      </Button>
                      <Popconfirm
                        title={`删除站点 ${record.name}`}
                        description="会删除站点配置并 reload OpenResty，站点目录内容不会自动删除。"
                        okText="删除"
                        cancelText="取消"
                        okButtonProps={{ danger: true }}
                        onConfirm={() => void deleteWebsite(record)}
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
      ) : null}

      {activeSection === "certificates" ? (
        <>
          <Card className="glass-card">
            <Space direction="vertical" size="middle" style={{ width: "100%" }}>
              <div className="page-inline-bar">
                <div>
                  <Typography.Title level={4} style={{ marginTop: 0 }}>
                    证书管理
                  </Typography.Title>
                  <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                    当前版本支持 Let's Encrypt 单域名证书申请、列表查看和删除。
                  </Typography.Paragraph>
                </div>
                <Space>
                  <Tag icon={<SafetyCertificateOutlined />} color={certificateReady ? "blue" : "orange"}>
                    {certificateReady ? "HTTP-01" : "需补证书挂载"}
                  </Tag>
                  <Button type="primary" onClick={openCertificateModal} disabled={!certificateReady}>
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
                      onConfirm={() => void deleteCertificate(record)}
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
      ) : null}

      {activeSection === "environments" ? (
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
                  <Button onClick={() => void loadData()}>刷新</Button>
                  <Button type="primary" onClick={openEnvironmentModal} disabled={!phpTemplate}>
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
                          onClick={() => void runEnvironmentAction(record, "start")}
                        >
                          启动
                        </Button>
                        <Button
                          size="small"
                          loading={environmentActionKey === `${record.id}:stop`}
                          onClick={() => void runEnvironmentAction(record, "stop")}
                        >
                          停止
                        </Button>
                        <Button
                          size="small"
                          loading={environmentActionKey === `${record.id}:restart`}
                          onClick={() => void runEnvironmentAction(record, "restart")}
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
                          onConfirm={() => void runEnvironmentAction(record, "delete")}
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
      ) : null}

      <Modal
        open={modalOpen}
        title={editingWebsite ? `配置站点 ${editingWebsite.name}` : "创建站点"}
        width={760}
        okText={editingWebsite ? "保存配置" : "立即创建"}
        cancelText="取消"
        onCancel={() => {
          setModalOpen(false);
          setEditingWebsite(null);
          form.resetFields();
        }}
        onOk={() => void form.submit()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            type: "static",
            domains: "",
            rewrite_mode: "off",
            index_files: "index.html index.htm",
          }}
          onFinish={submitWebsite}
        >
          {!status?.ready ? (
            <Alert
              showIcon
              type="warning"
              message={status?.message || "OpenResty 当前不可用"}
              style={{ marginBottom: 16 }}
            />
          ) : null}

          <Form.Item
            label="站点名"
            name="name"
            rules={[{ required: true, message: "请输入站点名" }]}
            extra="只允许小写字母、数字、下划线和中划线。当前版本不支持修改站点名。"
          >
            <Input placeholder="my-site" disabled={!!editingWebsite} />
          </Form.Item>

          <Form.Item
            label="站点模式"
            name="type"
            rules={[{ required: true, message: "请选择站点模式" }]}
          >
            <Radio.Group
              onChange={(event) => handleWebsiteTypeChange(event.target.value as WebsiteFormValues["type"])}
              options={[
                { label: "静态站点", value: "static" },
                { label: "PHP 站点", value: "php" },
                { label: "整站反向代理", value: "proxy" },
              ]}
            />
          </Form.Item>

          <Form.Item
            label="主域名"
            name="domain"
            rules={[{ required: true, message: "请输入主域名" }]}
          >
            <Input placeholder="example.com" />
          </Form.Item>

          <Form.Item
            label="附加域名"
            name="domains"
            extra="多个域名用逗号、空格或换行分隔。"
          >
            <Input.TextArea rows={3} placeholder="www.example.com, static.example.com" />
          </Form.Item>

          {websiteType === "static" || websiteType === "php" ? (
            <>
              {websiteType === "php" && phpProjects.length === 0 ? (
                <Alert
                  showIcon
                  type="warning"
                  message="还没有可用的 PHP 环境"
                  description="请先在上方“环境”页创建 php-fpm 实例，再回来绑定站点。"
                  style={{ marginBottom: 16 }}
                />
              ) : null}

              <Form.Item
                label="站点目录"
                name="root_path"
                extra="支持相对路径或绝对路径，但必须位于 OpenResty 站点挂载目录下。留空时默认使用站点名目录。"
              >
                <Input placeholder="test 或 /root/CamoPanel/server/data/openresty/www/test" />
              </Form.Item>

              <Form.Item
                label="首页文件"
                name="index_files"
                extra="使用空格分隔多个首页文件。"
              >
                <Input placeholder={websiteType === "php" ? "index.php index.html index.htm" : "index.html index.htm"} />
              </Form.Item>

              {websiteType === "php" ? (
                <Form.Item
                  label="PHP 环境"
                  name="php_project_id"
                  rules={[{ required: true, message: "请选择 PHP 环境" }]}
                  extra="站点会把 .php 请求转发到选中的 php-fpm 实例。"
                >
                  <Select
                    options={phpProjects.map((item) => ({
                      value: item.id,
                      label: `${item.name} / PHP ${projectConfigText(item, "php_version") || "-"} / 127.0.0.1:${projectConfigNumber(item, "port") || "-"}`,
                    }))}
                    placeholder="选择 PHP 环境"
                  />
                </Form.Item>
              ) : null}

              <Form.Item label="伪静态" name="rewrite_mode">
                <Radio.Group
                  options={[
                    { label: "关闭", value: "off" },
                    { label: "预设", value: "preset" },
                    { label: "自定义", value: "custom" },
                  ]}
                />
              </Form.Item>

              {rewriteMode === "preset" ? (
                <Form.Item
                  label="伪静态预设"
                  name="rewrite_preset"
                  rules={[{ required: true, message: "请选择伪静态预设" }]}
                >
                  <Select options={REWRITE_PRESET_OPTIONS} placeholder="选择伪静态预设" />
                </Form.Item>
              ) : null}

              {rewriteMode === "custom" ? (
                <Form.Item
                  label="自定义伪静态规则"
                  name="rewrite_rules"
                  rules={[{ required: true, message: "请输入自定义规则" }]}
                  extra="直接填写 location / 内部规则片段。"
                >
                  <Input.TextArea
                    rows={8}
                    placeholder={
                      websiteType === "php"
                        ? `try_files $uri $uri/ /index.php?$query_string;`
                        : `try_files $uri $uri/ /index.html;`
                    }
                  />
                </Form.Item>
              ) : null}
            </>
          ) : (
            <Form.Item
              label="代理地址"
              name="proxy_pass"
              rules={[{ required: true, message: "请输入代理地址" }]}
              extra="示例：http://127.0.0.1:3000"
            >
              <Input placeholder="http://127.0.0.1:3000" />
            </Form.Item>
          )}
        </Form>
      </Modal>

      <Modal
        open={previewOpen}
        title={previewTitle}
        footer={[
          <Button key="close" onClick={() => setPreviewOpen(false)}>
            关闭
          </Button>,
        ]}
        width={860}
        onCancel={() => setPreviewOpen(false)}
      >
        <pre className="mono-box">{previewContent}</pre>
      </Modal>

      <Modal
        open={environmentModalOpen}
        title="新建 PHP 环境"
        okText="创建环境"
        cancelText="取消"
        onCancel={() => {
          setEnvironmentModalOpen(false);
          environmentForm.resetFields();
        }}
        onOk={() => void environmentForm.submit()}
        confirmLoading={environmentSubmitting}
        destroyOnClose
      >
        <Form
          form={environmentForm}
          layout="vertical"
          initialValues={{ name: "", php_version: "8.3", port: 9000 }}
          onFinish={submitEnvironment}
        >
          <Form.Item
            label="环境名"
            name="name"
            rules={[{ required: true, message: "请输入环境名" }]}
            extra="只允许小写字母、数字、下划线和中划线。"
          >
            <Input placeholder="php83-blog" />
          </Form.Item>

          <Form.Item
            label="PHP 版本"
            name="php_version"
            rules={[{ required: true, message: "请选择 PHP 版本" }]}
          >
            <Select
              options={[
                { value: "8.1", label: "PHP 8.1" },
                { value: "8.2", label: "PHP 8.2" },
                { value: "8.3", label: "PHP 8.3" },
              ]}
            />
          </Form.Item>

          <Form.Item
            label="FPM 端口"
            name="port"
            rules={[{ required: true, message: "请输入 FPM 端口" }]}
            extra="会绑定到宿主机 127.0.0.1，仅供 OpenResty 转发。"
          >
            <InputNumber min={1} max={65535} style={{ width: "100%" }} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={certificateModalOpen}
        title="申请证书"
        okText="立即申请"
        cancelText="取消"
        onCancel={() => {
          setCertificateModalOpen(false);
          certificateForm.resetFields();
        }}
        onOk={() => void certificateForm.submit()}
        confirmLoading={certificateSubmitting}
        destroyOnClose
      >
        <Form
          form={certificateForm}
          layout="vertical"
          initialValues={{ domain: "", email: "" }}
          onFinish={submitCertificate}
        >
          {!certificateReady ? (
            <Alert
              showIcon
              type="warning"
              message={status?.message || "OpenResty 当前不可用"}
              style={{ marginBottom: 16 }}
            />
          ) : null}

          <Form.Item
            label="域名"
            name="domain"
            rules={[{ required: true, message: "请输入域名" }]}
            extra="第一版只支持单域名证书。"
          >
            <Input placeholder="example.com" />
          </Form.Item>

          <Form.Item
            label="邮箱"
            name="email"
            rules={[{ required: true, message: "请输入邮箱" }, { type: "email", message: "邮箱格式不正确" }]}
            extra="Let's Encrypt 注册和证书通知会使用这个邮箱。"
          >
            <Input placeholder="admin@example.com" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

function parseDomains(raw?: string) {
  if (!raw) {
    return [];
  }
  try {
    const items = JSON.parse(raw) as string[];
    return Array.isArray(items) ? items : [];
  } catch {
    return [];
  }
}

function parseDomainsText(raw?: string) {
  return parseDomains(raw).join("\n");
}

function splitDomainInput(raw: string) {
  return raw
    .split(/[\s,]+/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function formatRewriteLabel(website: Website) {
  if (website.type === "proxy") {
    return <Tag>未启用</Tag>;
  }
  switch (website.rewrite_mode ?? "off") {
    case "preset":
      return <Tag color="gold">{website.rewrite_preset === "spa" ? "SPA" : "Front Controller"}</Tag>;
    case "custom":
      return <Tag color="purple">自定义</Tag>;
    default:
      return <Tag>关闭</Tag>;
  }
}

function formatWebsiteType(value: Website["type"]) {
  switch (value) {
    case "php":
      return <Tag color="green">PHP 站点</Tag>;
    case "proxy":
      return <Tag color="cyan">反向代理</Tag>;
    default:
      return <Tag color="blue">静态站点</Tag>;
  }
}

function formatProjectStatus(status: string) {
  switch (status) {
    case "running":
      return <Tag color="green">运行中</Tag>;
    case "stopped":
      return <Tag color="default">已停止</Tag>;
    case "error":
      return <Tag color="red">异常</Tag>;
    case "docker_unavailable":
      return <Tag color="orange">Docker 不可用</Tag>;
    default:
      return <Tag>{status || "-"}</Tag>;
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
    case "delete":
      return "删除";
    default:
      return action;
  }
}

function projectConfigText(project: Project, key: string) {
  const value = project.config?.[key];
  if (typeof value === "string") {
    return value.trim();
  }
  if (value == null) {
    return "";
  }
  return String(value).trim();
}

function projectConfigNumber(project: Project, key: string) {
  const value = project.config?.[key];
  if (typeof value === "number") {
    return value;
  }
  if (typeof value === "string") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
}

function getTemplateDefaultText(template: TemplateSpec | null, key: string, fallback: string) {
  const value = template?.params.find((item) => item.name === key)?.default;
  if (typeof value === "string" && value.trim()) {
    return value.trim();
  }
  if (typeof value === "number") {
    return String(value);
  }
  return fallback;
}

function getTemplateDefaultNumber(template: TemplateSpec | null, key: string, fallback: number) {
  const value = template?.params.find((item) => item.name === key)?.default;
  if (typeof value === "number") {
    return value;
  }
  if (typeof value === "string") {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }
  return fallback;
}

function normalizePHPVersion(value: string): EnvironmentFormValues["php_version"] {
  if (value === "8.1" || value === "8.2" || value === "8.3") {
    return value;
  }
  return "8.3";
}

function nextAvailablePort(projects: Project[], start: number) {
  const used = new Set(projects.map((item) => projectConfigNumber(item, "port")).filter(Boolean));
  let port = start;
  while (used.has(port)) {
    port += 1;
  }
  return port;
}

function getErrorMessage(error: unknown) {
  if (error instanceof Error && error.message.trim()) {
    return error.message;
  }
  return "操作失败";
}

function formatCertificateStatus(certificate: Certificate) {
  switch (certificate.status) {
    case "issued":
      return <Tag color={certificate.last_error ? "gold" : "green"}>{certificate.last_error ? "已签发/需处理" : "已签发"}</Tag>;
    case "error":
      return <Tag color="red">失败</Tag>;
    case "applying":
      return <Tag color="blue">申请中</Tag>;
    default:
      return <Tag>{certificate.status}</Tag>;
  }
}

function formatDateTime(value?: string) {
  if (!value) {
    return "-";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  if (date.getUTCFullYear() <= 1) {
    return "-";
  }
  return date.toLocaleString("zh-CN", { hour12: false });
}
