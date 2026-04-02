import {
  AppstoreOutlined,
  DatabaseOutlined,
  GlobalOutlined,
} from "@ant-design/icons";
import { Button, Card, Empty, Space, Tag, Typography } from "antd";
import type { Project, TemplateSpec } from "../../../shared/types";

type StoreCategoryFilter = "all" | "site" | "database" | "general";

const TEMPLATE_CATEGORY_MAP: Record<string, Exclude<StoreCategoryFilter, "all">> = {
  mysql: "database",
  openresty: "site",
  postgres: "database",
  redis: "database",
  wordpress: "site",
};

type StoreCatalogProps = {
  loading: boolean;
  templates: TemplateSpec[];
  projects: Project[];
  filteredTemplates: TemplateSpec[];
  projectCountByTemplate: Map<string, number>;
  projectsByTemplate: Map<string, Project[]>;
  deployedTemplateCount: number;
  onResetFilters: () => void;
  onOpenManage: (template: TemplateSpec) => void;
  onOpenInstall: (template: TemplateSpec) => void;
};

export function StoreCatalog({
  loading,
  templates,
  projects,
  filteredTemplates,
  projectCountByTemplate,
  projectsByTemplate,
  deployedTemplateCount,
  onResetFilters,
  onOpenManage,
  onOpenInstall,
}: StoreCatalogProps) {
  const totalTemplates = templates.length;
  const pendingTemplateCount = Math.max(totalTemplates - deployedTemplateCount, 0);

  return (
    <>
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
            const templateProjects = projectsByTemplate.get(item.id) ?? [];
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
                      <Typography.Text type="secondary">模板 ID：{item.id}</Typography.Text>
                    </div>
                    <Space wrap>
                      {isDeployed ? <Button onClick={() => onOpenManage(item)}>管理实例</Button> : null}
                      <Button type="primary" onClick={() => onOpenInstall(item)}>
                        {templateProjects.length > 0 ? "新安装" : "安装"}
                      </Button>
                    </Space>
                  </div>
                </div>
              </Card>
            );
          })}
        </div>
      ) : (
        <Card className="glass-card store-empty-card" variant="borderless">
          <Empty description="没有匹配的应用模板">
            <Button onClick={onResetFilters}>重置筛选</Button>
          </Empty>
        </Card>
      )}
    </>
  );
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

export function getTemplateCategory(template: TemplateSpec): Exclude<StoreCategoryFilter, "all"> {
  return TEMPLATE_CATEGORY_MAP[template.id] ?? "general";
}

export function getCategoryLabel(category: Exclude<StoreCategoryFilter, "all">) {
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
