import {
  DeleteOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  SyncOutlined,
} from "@ant-design/icons";
import {
  Button,
  Form,
  Input,
  Select,
  Space,
  Tag,
  message,
} from "antd";
import { Suspense, lazy, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useShellHeader } from "../../widgets/shell/ShellHeaderContext";
import {
  createDatabaseAccount,
  createDatabaseGrant,
  createDatabaseSchema,
  deleteDatabaseAccount,
  deleteDatabaseSchema,
  runDatabaseInstanceAction,
  updateDatabasePassword,
  updateRedisConfig as saveRedisConfigValue,
} from "./api";
import { DatabaseContent } from "./components/DatabaseContent";
import { useDatabaseInstances } from "./hooks/useDatabaseInstances";
import { useDatabaseOverview } from "./hooks/useDatabaseOverview";
import type { DatabaseAccountItem, DatabaseEngine, DatabaseInstance } from "./types";

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

const LazyDatabaseModals = lazy(async () => {
  const module = await import("./components/DatabaseModals");
  return { default: module.DatabaseModals };
});

export function DatabasesPage() {
  const navigate = useNavigate();
  const [activeEngine, setActiveEngine] = useState<DatabaseEngine>("mysql");
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
  const { instances, selectedInstanceId, setSelectedInstanceId, loadingInstances, refresh: refreshInstances } =
    useDatabaseInstances(activeEngine);
  const { overview, loadingOverview, refresh: refreshOverview } = useDatabaseOverview(selectedInstanceId);

  const refreshCurrent = async () => {
    const currentId = selectedInstanceId;
    const nextId = await refreshInstances(currentId);
    if (nextId) {
      await refreshOverview(nextId);
    }
  };

  const runInstanceAction = async (action: string) => {
    if (!overview) return;
    try {
      await runDatabaseInstanceAction(overview.instance.id, action);
      message.success(`${actionLabel(action)}已执行`);
      if (action === "delete") {
        await refreshInstances();
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
      await createDatabaseSchema(overview.instance.id, values.name);
      databaseCreated = true;
      await createDatabaseAccount(overview.instance.id, {
        name: values.name,
        password: values.password,
        database_name: values.name,
      });
      message.success("业务库已创建，可直接使用同名账号连接");
      setQuickCreateOpen(false);
      quickCreateForm.resetFields();
      await refreshOverview(overview.instance.id);
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
      await createDatabaseSchema(overview.instance.id, values.name);
      message.success("数据库已创建");
      setDatabaseModalOpen(false);
      databaseForm.resetFields();
      await refreshOverview(overview.instance.id);
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
      await createDatabaseAccount(overview.instance.id, values);
      message.success("账号已创建");
      setAccountModalOpen(false);
      accountForm.resetFields();
      await refreshOverview(overview.instance.id);
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
      await createDatabaseGrant(overview.instance.id, {
        name: values.account_name,
        database_name: values.database_name,
      });
      message.success("授权已执行");
      setGrantModalOpen(false);
      grantForm.resetFields();
      await refreshOverview(overview.instance.id);
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
      await updateDatabasePassword(overview.instance.id, passwordTarget.name, values.password);
      message.success("密码已更新");
      setPasswordModalOpen(false);
      setPasswordTarget(null);
      passwordForm.resetFields();
      await refreshOverview(overview.instance.id);
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
      await deleteDatabaseSchema(overview.instance.id, deleteDatabaseTarget);
      databaseDeleted = true;

      if (values.deleteAccount && accountName) {
        await deleteDatabaseAccount(overview.instance.id, accountName);
      }

      message.success(values.deleteAccount && accountName ? "数据库和账号已删除" : "数据库已删除");
      setDeleteDatabaseTarget(null);
      deleteDatabaseForm.resetFields();
      await refreshOverview(overview.instance.id);
    } catch (error) {
      if (databaseDeleted) {
        message.warning("数据库已删除，但账号删除失败，请检查账号列表。");
        setDeleteDatabaseTarget(null);
        deleteDatabaseForm.resetFields();
        await refreshOverview(overview.instance.id);
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
      await deleteDatabaseAccount(overview.instance.id, deleteAccountTarget.name);
      message.success("账号已删除");
      setDeleteAccountTarget(null);
      await refreshOverview(overview.instance.id);
    } catch (error) {
      message.error(getErrorMessage(error));
    } finally {
      setSubmitting(false);
    }
  };

  const updateRedisConfig = async (key: string, value: boolean | number | string) => {
    if (!overview) return;
    try {
      await saveRedisConfigValue(overview.instance.id, { key, value });
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
  const showDatabaseModals =
    quickCreateOpen ||
    databaseModalOpen ||
    accountModalOpen ||
    grantModalOpen ||
    passwordModalOpen ||
    deleteDatabaseTarget !== null ||
    deleteAccountTarget !== null;

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
          <Button icon={<ReloadOutlined />} onClick={() => void refreshInstances(selectedInstanceId)}>
            刷新实例
          </Button>
        </Space>
        <Button type="primary" onClick={() => navigate("/app/store")}>
          去应用商店安装
        </Button>
      </div>

      <div className="database-main">
        <DatabaseContent
          overview={overview}
          loadingOverview={loadingOverview}
          loadingInstances={loadingInstances}
          activeEngine={activeEngine}
          canManageCurrent={canManageCurrent}
          onRefreshCurrent={refreshCurrent}
          onRunInstanceAction={runInstanceAction}
          onOpenQuickCreate={() => {
            quickCreateForm.resetFields();
            setQuickCreateOpen(true);
          }}
          onOpenDatabaseModal={() => {
            databaseForm.resetFields();
            setDatabaseModalOpen(true);
          }}
          onOpenAccountModal={() => {
            accountForm.resetFields();
            setAccountModalOpen(true);
          }}
          onOpenGrantModal={() => {
            grantForm.resetFields();
            setGrantModalOpen(true);
          }}
          onRequestDeleteDatabase={(name) => {
            deleteDatabaseForm.setFieldsValue({
              deleteAccount: true,
              accountName: name,
            });
            setDeleteDatabaseTarget(name);
          }}
          onRequestDeleteAccount={setDeleteAccountTarget}
          onRequestPasswordReset={(account) => {
            setPasswordTarget(account);
            passwordForm.resetFields();
            setPasswordModalOpen(true);
          }}
          onUpdateRedisConfig={updateRedisConfig}
          onGoInstall={() => navigate("/app/store")}
          statusColor={statusColor}
          summaryLabel={summaryLabel}
          canDeleteDatabase={canDeleteDatabase}
          canManageDatabaseAccount={canManageDatabaseAccount}
          engineActionLabels={ENGINE_ACTION_LABELS}
        />
      </div>

      {showDatabaseModals ? (
        <Suspense fallback={null}>
          <LazyDatabaseModals
            quickCreateOpen={quickCreateOpen}
            databaseModalOpen={databaseModalOpen}
            accountModalOpen={accountModalOpen}
            grantModalOpen={grantModalOpen}
            passwordModalOpen={passwordModalOpen}
            deleteDatabaseTarget={deleteDatabaseTarget}
            deleteAccountTarget={deleteAccountTarget}
            passwordTarget={passwordTarget}
            submitting={submitting}
            deleteAccountEnabled={deleteAccountEnabled}
            quickCreateForm={quickCreateForm}
            databaseForm={databaseForm}
            accountForm={accountForm}
            grantForm={grantForm}
            passwordForm={passwordForm}
            deleteDatabaseForm={deleteDatabaseForm}
            databaseOptions={databaseOptions}
            accountOptions={accountOptions}
            onCloseQuickCreate={() => {
              setQuickCreateOpen(false);
              quickCreateForm.resetFields();
            }}
            onCloseDatabaseModal={() => {
              setDatabaseModalOpen(false);
              databaseForm.resetFields();
            }}
            onCloseAccountModal={() => {
              setAccountModalOpen(false);
              accountForm.resetFields();
            }}
            onCloseGrantModal={() => {
              setGrantModalOpen(false);
              grantForm.resetFields();
            }}
            onClosePasswordModal={() => {
              setPasswordModalOpen(false);
              setPasswordTarget(null);
              passwordForm.resetFields();
            }}
            onCloseDeleteDatabase={() => {
              setDeleteDatabaseTarget(null);
              deleteDatabaseForm.resetFields();
            }}
            onCloseDeleteAccount={() => setDeleteAccountTarget(null)}
            onCreateQuickWorkspace={createQuickWorkspace}
            onCreateDatabase={createDatabase}
            onCreateAccount={createAccount}
            onGrantAccount={grantAccount}
            onUpdateAccountPassword={updateAccountPassword}
            onDeleteDatabase={deleteDatabase}
            onDeleteAccount={deleteAccount}
          />
        </Suspense>
      ) : null}
    </div>
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
