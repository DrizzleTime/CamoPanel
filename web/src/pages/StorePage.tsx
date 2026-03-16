import {
  AppstoreOutlined,
  DatabaseOutlined,
  GlobalOutlined,
  SearchOutlined,
} from "@ant-design/icons";
import {
  Button,
  Card,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  Space,
  Switch,
  Tag,
  Typography,
  message,
} from "antd";
import { useEffect, useMemo, useState } from "react";
import { useShellHeader } from "../components/ShellHeaderContext";
import { apiRequest } from "../lib/api";
import type { Project, TemplateParam, TemplateSpec } from "../lib/types";

type StoreStatusFilter = "all" | "deployed" | "available";
type StoreCategoryFilter = "all" | "site" | "database" | "general";

const MANAGED_OPENRESTY_TEMPLATE_ID = "openresty";
const MANAGED_OPENRESTY_PROJECT_NAME = "openresty";

const STATUS_OPTIONS: Array<{ key: StoreStatusFilter; label: string }> = [
  { key: "all", label: "全部" },
  { key: "deployed", label: "已安装" },
  { key: "available", label: "未安装" },
];

const CATEGORY_OPTIONS: Array<{ key: StoreCategoryFilter; label: string }> = [
  { key: "all", label: "全部" },
  { key: "site", label: "建站" },
  { key: "database", label: "数据库" },
  { key: "general", label: "通用" },
];

const TEMPLATE_CATEGORY_MAP: Record<string, Exclude<StoreCategoryFilter, "all">> = {
  openresty: "site",
  postgres: "database",
  wordpress: "site",
};

export function StorePage() {
  const [templates, setTemplates] = useState<TemplateSpec[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTemplate, setActiveTemplate] = useState<TemplateSpec | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [statusFilter, setStatusFilter] = useState<StoreStatusFilter>("all");
  const [categoryFilter, setCategoryFilter] = useState<StoreCategoryFilter>("all");
  const [searchValue, setSearchValue] = useState("");
  const [form] = Form.useForm();

  const loadStoreData = async () => {
    setLoading(true);
    try {
      const [templateResponse, projectResponse] = await Promise.all([
        apiRequest<{ items: TemplateSpec[] }>("/api/templates"),
        apiRequest<{ items: Project[] }>("/api/projects"),
      ]);
      setTemplates(templateResponse.items);
      setProjects(projectResponse.items);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadStoreData();
  }, []);

  const projectCountByTemplate = useMemo(() => {
    const counts = new Map<string, number>();
    for (const project of projects) {
      counts.set(project.template_id, (counts.get(project.template_id) ?? 0) + 1);
    }
    return counts;
  }, [projects]);

  const deployedTemplateCount = useMemo(() => {
    return Array.from(projectCountByTemplate.values()).filter((count) => count > 0).length;
  }, [projectCountByTemplate]);

  const initialValues = useMemo(() => {
    if (!activeTemplate) return {};
    const values = Object.fromEntries(
      activeTemplate.params
        .filter((param) => param.default !== undefined)
        .map((param) => [param.name, param.default]),
    );
    if (isManagedOpenRestyTemplate(activeTemplate)) {
      values.projectName = MANAGED_OPENRESTY_PROJECT_NAME;
    }
    return values;
  }, [activeTemplate]);

  useEffect(() => {
    form.resetFields();
    form.setFieldsValue(initialValues);
  }, [form, initialValues]);

  const filteredTemplates = useMemo(() => {
    const keyword = searchValue.trim().toLowerCase();

    return templates.filter((template) => {
      const category = getTemplateCategory(template);
      const categoryLabel = getCategoryLabel(category).toLowerCase();
      const projectCount = projectCountByTemplate.get(template.id) ?? 0;
      const matchesKeyword =
        keyword.length === 0 ||
        [template.name, template.description, template.id, categoryLabel].some((value) =>
          value.toLowerCase().includes(keyword),
        );
      const matchesStatus =
        statusFilter === "all" ||
        (statusFilter === "deployed" ? projectCount > 0 : projectCount === 0);
      const matchesCategory = categoryFilter === "all" || category === categoryFilter;

      return matchesKeyword && matchesStatus && matchesCategory;
    });
  }, [categoryFilter, projectCountByTemplate, searchValue, statusFilter, templates]);

  const headerContent = useMemo(
    () => (
      <div className="shell-header-store">
        <Input
          allowClear
          value={searchValue}
          prefix={<SearchOutlined />}
          placeholder="搜索应用名、描述或分类"
          className="shell-header-search"
          onChange={(event) => setSearchValue(event.target.value)}
        />
      </div>
    ),
    [filteredTemplates.length, searchValue],
  );

  useShellHeader(headerContent);

  const deploy = async (values: Record<string, unknown>) => {
    if (!activeTemplate) return;

    setSubmitting(true);
    try {
      const { projectName, ...parameters } = values;
      await apiRequest<{ project: Project }>("/api/projects", {
        method: "POST",
        body: JSON.stringify({
          name: projectName,
          template_id: activeTemplate.id,
          parameters,
        }),
      });
      message.success("安装完成");
      setActiveTemplate(null);
      form.resetFields();
      void loadStoreData();
    } finally {
      setSubmitting(false);
    }
  };

  const totalTemplates = templates.length;
  const pendingTemplateCount = Math.max(totalTemplates - deployedTemplateCount, 0);
  const projectNameLocked = isManagedOpenRestyTemplate(activeTemplate);

  return (
    <div className="page-grid store-page">
      <Card className="glass-card store-hero-card" variant="borderless">
        <div className="store-hero-content">
          <div className="store-overview-bar">
            <Space wrap size={[8, 8]}>
              <Tag>模板驱动安装</Tag>
              <Tag>直接执行</Tag>
              <Tag>{totalTemplates} 个可用模板</Tag>
            </Space>
            <Typography.Text type="secondary">
              已部署 {deployedTemplateCount} 个模板，当前共有 {projects.length} 个项目实例。
            </Typography.Text>
          </div>

          <div className="store-stats-grid">
            <StoreStatCard label="模板总数" value={String(totalTemplates)} helper="当前可直接安装" />
            <StoreStatCard
              label="已安装模板"
              value={String(deployedTemplateCount)}
              helper="至少已有一个项目实例"
            />
            <StoreStatCard label="现有项目" value={String(projects.length)} helper="由模板派生的项目总数" />
            <StoreStatCard label="待安装模板" value={String(pendingTemplateCount)} helper="还没有项目实例" />
          </div>
        </div>
      </Card>

      <Card className="glass-card store-toolbar-card" variant="borderless">
        <div className="store-toolbar">
          <div className="store-filter-stack">
            <FilterChipGroup
              label="状态"
              options={STATUS_OPTIONS}
              activeKey={statusFilter}
              onChange={(value) => setStatusFilter(value as StoreStatusFilter)}
            />
            <FilterChipGroup
              label="分类"
              options={CATEGORY_OPTIONS}
              activeKey={categoryFilter}
              onChange={(value) => setCategoryFilter(value as StoreCategoryFilter)}
            />
          </div>
          <Typography.Text type="secondary" className="store-toolbar-meta">
            用状态和分类快速筛选结果。
          </Typography.Text>
        </div>
      </Card>

      {loading ? (
        <div className="store-app-grid">
          {Array.from({ length: 6 }).map((_, index) => (
            <Card key={index} className="glass-card store-app-card" loading />
          ))}
        </div>
      ) : filteredTemplates.length ? (
        <div className="store-app-grid">
          {filteredTemplates.map((item) => {
            const category = getTemplateCategory(item);
            const categoryLabel = getCategoryLabel(category);
            const projectCount = projectCountByTemplate.get(item.id) ?? 0;
            const isDeployed = projectCount > 0;

            return (
              <Card className="glass-card store-app-card" key={item.id} variant="borderless">
                <div className="store-app-card-body">
                  <div className="store-app-header">
                    <div className="store-app-icon">{renderTemplateIcon(item)}</div>
                    <div className="store-app-summary">
                      <div className="store-app-title-row">
                        <Typography.Title level={4} className="store-app-title">
                          {item.name}
                        </Typography.Title>
                        <Tag>{item.version}</Tag>
                      </div>
                      <Typography.Paragraph className="store-app-description">
                        {item.description}
                      </Typography.Paragraph>
                    </div>
                  </div>

                  <div className="store-app-meta">
                    <Tag color="blue">{categoryLabel}</Tag>
                    <Tag>{item.params.length} 个配置项</Tag>
                    {isDeployed ? <Tag color="green">已安装</Tag> : <Tag>未安装</Tag>}
                  </div>

                  <Typography.Text className="store-app-hint" type="secondary">
                    {item.health_hints[0] || "安装后可在项目页查看容器状态和运行日志。"}
                  </Typography.Text>

                  <div className="store-app-footer">
                    <div className="store-app-footer-note">
                      <Typography.Text strong>
                        {isDeployed ? `已创建 ${projectCount} 个项目` : "尚未创建项目"}
                      </Typography.Text>
                      <Typography.Text type="secondary">
                        模板 ID：{item.id}
                      </Typography.Text>
                    </div>
                    <Button type="primary" onClick={() => setActiveTemplate(item)}>
                      安装
                    </Button>
                  </div>
                </div>
              </Card>
            );
          })}
        </div>
      ) : (
        <Card className="glass-card store-empty-card" variant="borderless">
          <Empty description="没有匹配的应用模板">
            <Button
              onClick={() => {
                setSearchValue("");
                setStatusFilter("all");
                setCategoryFilter("all");
              }}
            >
              重置筛选
            </Button>
          </Empty>
        </Card>
      )}

      <Drawer
        open={!!activeTemplate}
        title={activeTemplate ? `安装 ${activeTemplate.name}` : "安装应用"}
        width={520}
        onClose={() => setActiveTemplate(null)}
        destroyOnHidden
        extra={
          <Space>
            <Button onClick={() => setActiveTemplate(null)}>取消</Button>
            <Button type="primary" loading={submitting} onClick={() => void form.submit()}>
              立即安装
            </Button>
          </Space>
        }
      >
        {activeTemplate ? (
          <Form form={form} layout="vertical" onFinish={deploy} initialValues={initialValues}>
            <Form.Item
              label="项目名"
              name="projectName"
              rules={[{ required: true, message: "请输入项目名" }]}
              extra={projectNameLocked ? "固定 OpenResty 项目，安装后会直接供网站管理页复用。" : undefined}
            >
              <Input
                placeholder={projectNameLocked ? MANAGED_OPENRESTY_PROJECT_NAME : `${activeTemplate.id}-demo`}
                disabled={projectNameLocked}
              />
            </Form.Item>
            {activeTemplate.params.map((param) => (
              <TemplateField key={param.name} param={param} />
            ))}
          </Form>
        ) : null}
      </Drawer>
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

function StoreStatCard({
  label,
  value,
  helper,
}: {
  label: string;
  value: string;
  helper: string;
}) {
  return (
    <div className="store-stat-card">
      <Typography.Text type="secondary">{label}</Typography.Text>
      <Typography.Title level={3} className="store-stat-value">
        {value}
      </Typography.Title>
      <Typography.Text type="secondary">{helper}</Typography.Text>
    </div>
  );
}

function FilterChipGroup({
  label,
  options,
  activeKey,
  onChange,
}: {
  label: string;
  options: Array<{ key: string; label: string }>;
  activeKey: string;
  onChange: (key: string) => void;
}) {
  return (
    <div className="store-filter-group">
      <Typography.Text className="store-filter-label">{label}</Typography.Text>
      <div className="store-filter-list">
        {options.map((option) => (
          <Button
            key={option.key}
            type={activeKey === option.key ? "primary" : "default"}
            onClick={() => onChange(option.key)}
          >
            {option.label}
          </Button>
        ))}
      </div>
    </div>
  );
}

function getTemplateCategory(template: TemplateSpec): Exclude<StoreCategoryFilter, "all"> {
  return TEMPLATE_CATEGORY_MAP[template.id] ?? "general";
}

function getCategoryLabel(category: Exclude<StoreCategoryFilter, "all">) {
  switch (category) {
    case "site":
      return "建站";
    case "database":
      return "数据库";
    default:
      return "通用";
  }
}

function renderTemplateIcon(template: TemplateSpec) {
  switch (getTemplateCategory(template)) {
    case "site":
      return <GlobalOutlined />;
    case "database":
      return <DatabaseOutlined />;
    default:
      return <AppstoreOutlined />;
  }
}

function isManagedOpenRestyTemplate(template: Pick<TemplateSpec, "id"> | null | undefined) {
  return template?.id === MANAGED_OPENRESTY_TEMPLATE_ID;
}
