import {
  SearchOutlined,
} from "@ant-design/icons";
import {
  Form,
  Input,
  Select,
  message,
} from "antd";
import { Suspense, lazy, useEffect, useMemo, useState } from "react";
import type { Project, TemplateSpec } from "../../shared/types";
import { useShellHeader } from "../../widgets/shell/ShellHeaderContext";
import { deployTemplateProject, loadStoreBundle, runStoreProjectAction } from "./api";
import { StoreCatalog, getCategoryLabel, getTemplateCategory } from "./components/StoreCatalog";

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

const LazyStoreDrawers = lazy(async () => {
  const module = await import("./components/StoreDrawers");
  return { default: module.StoreDrawers };
});

export function StorePage() {
  const [templates, setTemplates] = useState<TemplateSpec[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTemplate, setActiveTemplate] = useState<TemplateSpec | null>(null);
  const [managedTemplate, setManagedTemplate] = useState<TemplateSpec | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [projectActionKey, setProjectActionKey] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<StoreStatusFilter>("all");
  const [categoryFilter, setCategoryFilter] = useState<StoreCategoryFilter>("all");
  const [searchValue, setSearchValue] = useState("");
  const [form] = Form.useForm();

  const loadStoreData = async () => {
    setLoading(true);
    try {
      const data = await loadStoreBundle();
      setTemplates(data.templates);
      setProjects(data.projects);
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

  const projectsByTemplate = useMemo(() => {
    const grouped = new Map<string, Project[]>();
    for (const project of projects) {
      const items = grouped.get(project.template_id);
      if (items) {
        items.push(project);
        continue;
      }
      grouped.set(project.template_id, [project]);
    }
    return grouped;
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
        <Select
          value={statusFilter}
          options={STATUS_OPTIONS.map((option) => ({
            value: option.key,
            label: `状态：${option.label}`,
          }))}
          className="shell-header-filter"
          onChange={(value) => setStatusFilter(value)}
        />
        <Select
          value={categoryFilter}
          options={CATEGORY_OPTIONS.map((option) => ({
            value: option.key,
            label: `分类：${option.label}`,
          }))}
          className="shell-header-filter"
          onChange={(value) => setCategoryFilter(value)}
        />
      </div>
    ),
    [categoryFilter, searchValue, statusFilter],
  );

  useShellHeader(headerContent);

  const deploy = async (values: Record<string, unknown>) => {
    if (!activeTemplate) return;

    setSubmitting(true);
    try {
      const { projectName, ...parameters } = values;
      await deployTemplateProject({
        name: projectName,
        template_id: activeTemplate.id,
        parameters,
      });
      message.success("安装完成");
      setActiveTemplate(null);
      form.resetFields();
      await loadStoreData();
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setSubmitting(false);
    }
  };

  const runProjectAction = async (project: Project, action: "delete" | "redeploy") => {
    setProjectActionKey(`${project.id}:${action}`);
    try {
      await runStoreProjectAction(project.id, action);
      message.success(`${projectActionLabel(action)}完成`);
      await loadStoreData();
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setProjectActionKey(null);
    }
  };

  const projectNameLocked = isManagedOpenRestyTemplate(activeTemplate);
  const showStoreDrawers = activeTemplate !== null || managedTemplate !== null;

  return (
    <div className="page-grid store-page">
      <StoreCatalog
        loading={loading}
        templates={templates}
        projects={projects}
        filteredTemplates={filteredTemplates}
        projectCountByTemplate={projectCountByTemplate}
        projectsByTemplate={projectsByTemplate}
        deployedTemplateCount={deployedTemplateCount}
        onResetFilters={() => {
          setSearchValue("");
          setStatusFilter("all");
          setCategoryFilter("all");
        }}
        onOpenManage={setManagedTemplate}
        onOpenInstall={setActiveTemplate}
      />

      {showStoreDrawers ? (
        <Suspense fallback={null}>
          <LazyStoreDrawers
            activeTemplate={activeTemplate}
            managedTemplate={managedTemplate}
            projectsByTemplate={projectsByTemplate}
            initialValues={initialValues}
            projectNameLocked={projectNameLocked}
            submitting={submitting}
            projectActionKey={projectActionKey}
            form={form}
            onCloseInstall={() => setActiveTemplate(null)}
            onCloseManage={() => setManagedTemplate(null)}
            onSubmitInstall={() => void form.submit()}
            onOpenInstallFromManage={(template) => {
              setManagedTemplate(null);
              setActiveTemplate(template);
            }}
            onDeploy={deploy}
            onRunProjectAction={runProjectAction}
            projectActionLabel={projectActionLabel}
            projectStatusColor={projectStatusColor}
            projectStatusLabel={projectStatusLabel}
            projectRuntimeHint={projectRuntimeHint}
            formatDateTime={formatDateTime}
            managedOpenrestyProjectName={MANAGED_OPENRESTY_PROJECT_NAME}
          />
        </Suspense>
      ) : null}
    </div>
  );
}

function projectActionLabel(action: "delete" | "redeploy") {
  switch (action) {
    case "delete":
      return "卸载";
    case "redeploy":
      return "重装";
    default:
      return action;
  }
}

function projectStatusColor(status?: string) {
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

function projectStatusLabel(status?: string) {
  switch (status) {
    case "running":
      return "运行中";
    case "stopped":
      return "已停止";
    case "degraded":
      return "异常";
    case "not_found":
      return "未发现容器";
    case "docker_unavailable":
      return "Docker 不可用";
    default:
      return status || "未知";
  }
}

function projectRuntimeHint(project: Project) {
  if (project.runtime.containers.length === 0) {
    return "当前没有发现关联容器，可尝试重装恢复。";
  }
  if (project.status === "degraded") {
    return "部分容器未正常运行，建议先重装确认镜像和编排状态。";
  }
  if (project.status === "stopped") {
    return "实例当前处于停止状态，重装会按原配置重新拉起。";
  }
  return "使用当前安装参数重新部署或直接卸载该实例。";
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

function getErrorMessage(error: unknown) {
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return "操作失败";
}

function isManagedOpenRestyTemplate(template: Pick<TemplateSpec, "id"> | null | undefined) {
  return template?.id === MANAGED_OPENRESTY_TEMPLATE_ID;
}
