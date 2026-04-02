import { Button, Form, Tag, message } from "antd";
import { Suspense, lazy, useMemo, useState } from "react";
import type { Project, TemplateSpec } from "../../shared/types";
import { useShellHeader } from "../../widgets/shell/ShellHeaderContext";
import {
  createCertificate,
  createPhpEnvironment,
  deleteCertificate,
  deleteWebsite,
  previewWebsiteConfig,
  runWebsiteEnvironmentAction,
  saveWebsite,
} from "./api";
import { WebsiteSections } from "./components/WebsiteSections";
import { useWebsitesData } from "./hooks/useWebsitesData";
import type {
  Certificate,
  CertificateFormValues,
  EnvironmentFormValues,
  OpenRestyStatus,
  Website,
  WebsiteFormValues,
} from "./types";

type WebsiteSection = "sites" | "certificates" | "environments";

const WEBSITE_SECTIONS: Array<{ key: WebsiteSection; label: string }> = [
  { key: "sites", label: "网站" },
  { key: "certificates", label: "证书" },
  { key: "environments", label: "环境" },
];

const REWRITE_PRESET_OPTIONS = [
  { value: "spa", label: "SPA History" },
  { value: "front_controller", label: "Front Controller" },
];

const LazyWebsiteModals = lazy(async () => {
  const module = await import("./components/WebsiteModals");
  return { default: module.WebsiteModals };
});

export function WebsitesPage() {
  const [activeSection, setActiveSection] = useState<WebsiteSection>("sites");
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
  const { status, websites, certificates, projects, templates, loading, refresh } = useWebsitesData();
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
      await saveWebsite(payload, editingWebsite?.id);
      message.success(editingWebsite ? "站点配置已更新" : "站点创建完成");
      setModalOpen(false);
      setEditingWebsite(null);
      form.resetFields();
      await refresh();
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setSubmitting(false);
    }
  };

  const handlePreviewWebsiteConfig = async (website: Website) => {
    setPreviewLoading(true);
    try {
      const response = await previewWebsiteConfig(website.id);
      setPreviewTitle(`${website.name} 配置预览`);
      setPreviewContent(response.config);
      setPreviewOpen(true);
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setPreviewLoading(false);
    }
  };

  const handleDeleteWebsite = async (website: Website) => {
    setSubmitting(true);
    try {
      await deleteWebsite(website.id);
      message.success("站点已删除");
      await refresh();
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setSubmitting(false);
    }
  };

  const submitCertificate = async (values: CertificateFormValues) => {
    setCertificateSubmitting(true);
    try {
      await createCertificate({
        domain: values.domain,
        email: values.email,
      });
      message.success("证书申请完成");
      setCertificateModalOpen(false);
      certificateForm.resetFields();
      await refresh();
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setCertificateSubmitting(false);
    }
  };

  const handleDeleteCertificate = async (certificate: Certificate) => {
    setCertificateSubmitting(true);
    try {
      await deleteCertificate(certificate.id);
      message.success("证书已删除");
      await refresh();
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
      await createPhpEnvironment({
        name: values.name,
        template_id: phpTemplate.id,
        parameters: {
          php_version: values.php_version,
          port: values.port,
        },
      });
      message.success("PHP 环境已创建");
      setEnvironmentModalOpen(false);
      environmentForm.resetFields();
      await refresh();
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setEnvironmentSubmitting(false);
    }
  };

  const runEnvironmentAction = async (project: Project, action: "start" | "stop" | "restart" | "delete") => {
    setEnvironmentActionKey(`${project.id}:${action}`);
    try {
      await runWebsiteEnvironmentAction(project.id, action);
      message.success(`PHP 环境已${actionLabel(action)}`);
      await refresh();
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

  const showWebsiteModals =
    modalOpen || previewOpen || environmentModalOpen || certificateModalOpen;

  return (
    <div className="page-grid">
      <WebsiteSections
        activeSection={activeSection}
        status={status}
        websites={websites}
        certificates={certificates}
        phpProjects={phpProjects}
        phpProjectMap={phpProjectMap}
        phpTemplate={phpTemplate}
        phpUsageMap={phpUsageMap}
        loading={loading}
        previewLoading={previewLoading}
        environmentActionKey={environmentActionKey}
        onRefresh={refresh}
        onOpenCreateModal={openCreateModal}
        onOpenCertificateModal={openCertificateModal}
        onOpenEnvironmentModal={openEnvironmentModal}
        onOpenEditModal={openEditModal}
        onPreviewConfig={handlePreviewWebsiteConfig}
        onDeleteWebsite={handleDeleteWebsite}
        onDeleteCertificate={handleDeleteCertificate}
        onRunEnvironmentAction={runEnvironmentAction}
        parseDomains={parseDomains}
        formatWebsiteType={formatWebsiteType}
        formatRewriteLabel={formatRewriteLabel}
        formatCertificateStatus={formatCertificateStatus}
        formatProjectStatus={formatProjectStatus}
        projectConfigText={projectConfigText}
        projectConfigNumber={projectConfigNumber}
        formatDateTime={formatDateTime}
      />

      {showWebsiteModals ? (
        <Suspense fallback={null}>
          <LazyWebsiteModals
            status={status}
            editingWebsite={editingWebsite}
            modalOpen={modalOpen}
            previewOpen={previewOpen}
            previewTitle={previewTitle}
            previewContent={previewContent}
            environmentModalOpen={environmentModalOpen}
            certificateModalOpen={certificateModalOpen}
            submitting={submitting}
            environmentSubmitting={environmentSubmitting}
            certificateSubmitting={certificateSubmitting}
            websiteType={websiteType}
            rewriteMode={rewriteMode}
            phpProjects={phpProjects}
            phpTemplate={phpTemplate}
            form={form}
            environmentForm={environmentForm}
            certificateForm={certificateForm}
            certificateReady={certificateReady}
            onCloseWebsiteModal={() => {
              setModalOpen(false);
              setEditingWebsite(null);
              form.resetFields();
            }}
            onClosePreview={() => setPreviewOpen(false)}
            onCloseEnvironmentModal={() => {
              setEnvironmentModalOpen(false);
              environmentForm.resetFields();
            }}
            onCloseCertificateModal={() => {
              setCertificateModalOpen(false);
              certificateForm.resetFields();
            }}
            onSubmitWebsite={submitWebsite}
            onSubmitEnvironment={submitEnvironment}
            onSubmitCertificate={submitCertificate}
            onHandleWebsiteTypeChange={handleWebsiteTypeChange}
            projectConfigText={projectConfigText}
            projectConfigNumber={projectConfigNumber}
            rewritePresetOptions={REWRITE_PRESET_OPTIONS}
          />
        </Suspense>
      ) : null}
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
