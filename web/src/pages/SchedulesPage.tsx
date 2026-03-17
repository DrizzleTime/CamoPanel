import {
  ClockCircleOutlined,
  CodeOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  PlusOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import { Alert, Button, Card, Space, Table, Tag, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useMemo, useState, type ReactNode } from "react";
import { useShellHeader } from "../components/ShellHeaderContext";

type ScheduleTab = "tasks" | "scripts";

type ScheduleTask = {
  id: string;
  name: string;
  command: string;
  cron: string;
  status: "running" | "paused";
  lastRun: string;
  updatedAt: string;
};

type ScriptItem = {
  id: string;
  name: string;
  language: "py" | "js" | "bash";
  path: string;
  updatedAt: string;
  note: string;
};

const TASKS: ScheduleTask[] = [
  {
    id: "task-1",
    name: "每日签到",
    command: "task daily-checkin",
    cron: "0 8 * * *",
    status: "running",
    lastRun: "2026-03-17 08:00",
    updatedAt: "2026-03-16 21:40",
  },
  {
    id: "task-2",
    name: "同步订阅数据",
    command: "task sync-subscriptions --force",
    cron: "*/30 * * * *",
    status: "running",
    lastRun: "2026-03-17 12:30",
    updatedAt: "2026-03-17 09:12",
  },
  {
    id: "task-3",
    name: "清理过期缓存",
    command: "bash /opt/scripts/cleanup-cache.sh",
    cron: "15 3 * * 1",
    status: "paused",
    lastRun: "2026-03-10 03:15",
    updatedAt: "2026-03-15 18:05",
  },
];

const SCRIPTS: ScriptItem[] = [
  {
    id: "script-1",
    name: "京豆签到",
    language: "js",
    path: "/ql/scripts/jd_sign.js",
    updatedAt: "2026-03-17 10:18",
    note: "Node 风格脚本，适合直接由 cron 调用。",
  },
  {
    id: "script-2",
    name: "账单汇总",
    language: "py",
    path: "/ql/scripts/billing_report.py",
    updatedAt: "2026-03-16 23:41",
    note: "Python 脚本，通常用于数据整理或 API 调用。",
  },
  {
    id: "script-3",
    name: "日志打包",
    language: "bash",
    path: "/ql/scripts/archive_logs.sh",
    updatedAt: "2026-03-14 07:30",
    note: "Shell 脚本，适合宿主机巡检与文件处理。",
  },
];

const TAB_LABELS: Record<ScheduleTab, string> = {
  tasks: "任务",
  scripts: "脚本",
};

const LANGUAGE_LABELS: Record<ScriptItem["language"], string> = {
  py: "Python",
  js: "JavaScript",
  bash: "Bash",
};

export function SchedulesPage() {
  const [activeTab, setActiveTab] = useState<ScheduleTab>("tasks");

  const headerContent = useMemo(
    () => (
      <div className="shell-header-tabs">
        {(["tasks", "scripts"] as ScheduleTab[]).map((item) => (
          <Button
            key={item}
            size="small"
            type={activeTab === item ? "primary" : "text"}
            className="shell-header-tab"
            onClick={() => setActiveTab(item)}
          >
            {TAB_LABELS[item]}
          </Button>
        ))}
      </div>
    ),
    [activeTab],
  );

  useShellHeader(headerContent);

  const taskColumns: ColumnsType<ScheduleTask> = [
    {
      title: "名称",
      dataIndex: "name",
      key: "name",
      render: (value: string, record: ScheduleTask) => (
        <div className="schedule-primary-cell">
          <Typography.Text strong>{value}</Typography.Text>
          <Typography.Text type="secondary">{record.command}</Typography.Text>
        </div>
      ),
    },
    {
      title: "Cron",
      dataIndex: "cron",
      key: "cron",
      render: (value: string) => <Typography.Text code>{value}</Typography.Text>,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      render: (value: ScheduleTask["status"]) =>
        value === "running" ? <Tag color="green">运行中</Tag> : <Tag color="default">已暂停</Tag>,
    },
    {
      title: "最近运行",
      dataIndex: "lastRun",
      key: "lastRun",
    },
    {
      title: "更新时间",
      dataIndex: "updatedAt",
      key: "updatedAt",
    },
    {
      title: "操作",
      key: "actions",
      render: (_: unknown, record: ScheduleTask) => (
        <Space size="small">
          <Button
            size="small"
            icon={record.status === "running" ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
            disabled
          >
            {record.status === "running" ? "暂停" : "启用"}
          </Button>
          <Button size="small" disabled>
            编辑
          </Button>
        </Space>
      ),
    },
  ];

  const scriptColumns: ColumnsType<ScriptItem> = [
    {
      title: "名称",
      dataIndex: "name",
      key: "name",
      render: (value: string, record: ScriptItem) => (
        <div className="schedule-primary-cell">
          <Typography.Text strong>{value}</Typography.Text>
          <Typography.Text type="secondary">{record.note}</Typography.Text>
        </div>
      ),
    },
    {
      title: "语言",
      dataIndex: "language",
      key: "language",
      render: (value: ScriptItem["language"]) => <Tag color={languageColor(value)}>{LANGUAGE_LABELS[value]}</Tag>,
    },
    {
      title: "路径",
      dataIndex: "path",
      key: "path",
      render: (value: string) => <Typography.Text code>{value}</Typography.Text>,
    },
    {
      title: "更新时间",
      dataIndex: "updatedAt",
      key: "updatedAt",
    },
    {
      title: "操作",
      key: "actions",
      render: () => (
        <Space size="small">
          <Button size="small" disabled>
            编辑
          </Button>
          <Button size="small" disabled>
            运行
          </Button>
        </Space>
      ),
    },
  ];

  const showingTasks = activeTab === "tasks";

  return (
    <div className="page-grid schedule-page">
      <Alert
        showIcon
        type="info"
        message="当前页面只完成 UI 骨架，表格、按钮和状态都是静态 mock 数据，暂未接入执行器。"
      />

      <Card className="glass-card schedule-hero-card" variant="borderless">
        <div className="schedule-hero">
          <div className="schedule-hero-copy">
            <Typography.Title level={4} style={{ margin: 0 }}>
              {showingTasks ? "Cron 任务面板" : "脚本资产面板"}
            </Typography.Title>
            <Typography.Paragraph style={{ margin: 0 }}>
              {showingTasks
                ? "按青龙面板的核心结构保留任务名称、命令、cron、运行状态和最近执行时间，后续可直接接入真实任务中心。"
                : "按脚本资产的思路先区分 Python、JavaScript 和 Bash，后续可以接入编辑器、存储目录和运行日志。"}
            </Typography.Paragraph>
          </div>

          <div className="schedule-kpi-grid">
            <ScheduleStatCard
              icon={<ClockCircleOutlined />}
              label={showingTasks ? "任务总数" : "脚本总数"}
              value={String(showingTasks ? TASKS.length : SCRIPTS.length)}
              helper={showingTasks ? "当前展示 cron 任务" : "当前展示脚本资产"}
            />
            <ScheduleStatCard
              icon={showingTasks ? <PlayCircleOutlined /> : <CodeOutlined />}
              label={showingTasks ? "启用任务" : "支持语言"}
              value={showingTasks ? String(TASKS.filter((item) => item.status === "running").length) : "3"}
              helper={showingTasks ? "暂停任务后续可直接在此恢复" : "Python / JavaScript / Bash"}
            />
            <ScheduleStatCard
              icon={showingTasks ? <ReloadOutlined /> : <PlusOutlined />}
              label={showingTasks ? "最近执行" : "目录规划"}
              value={showingTasks ? "30 分钟内" : "/ql/scripts"}
              helper={showingTasks ? "保留最近运行时间字段" : "后续建议统一脚本存放路径"}
            />
          </div>
        </div>
      </Card>

      <div className="schedule-layout">
        <Card
          className="glass-card"
          title={showingTasks ? "任务列表" : "脚本列表"}
          extra={
            <Space>
              <Button size="small" icon={<ReloadOutlined />} disabled>
                刷新
              </Button>
              <Button type="primary" size="small" icon={<PlusOutlined />} disabled>
                {showingTasks ? "新建任务" : "新建脚本"}
              </Button>
            </Space>
          }
        >
          {showingTasks ? (
            <Table<ScheduleTask>
              rowKey="id"
              pagination={false}
              columns={taskColumns}
              dataSource={TASKS}
              scroll={{ x: 980 }}
            />
          ) : (
            <Table<ScriptItem>
              rowKey="id"
              pagination={false}
              columns={scriptColumns}
              dataSource={SCRIPTS}
              scroll={{ x: 980 }}
            />
          )}
        </Card>

        <Card
          className="glass-card"
          title={showingTasks ? "Cron 示例" : "脚本说明"}
          variant="borderless"
        >
          {showingTasks ? (
            <div className="schedule-help-list">
              <div className="schedule-help-item">
                <Typography.Text strong>每天早上 8 点</Typography.Text>
                <Typography.Text code>0 8 * * *</Typography.Text>
              </div>
              <div className="schedule-help-item">
                <Typography.Text strong>每 30 分钟执行一次</Typography.Text>
                <Typography.Text code>*/30 * * * *</Typography.Text>
              </div>
              <div className="schedule-help-item">
                <Typography.Text strong>每周一凌晨 3:15</Typography.Text>
                <Typography.Text code>15 3 * * 1</Typography.Text>
              </div>
            </div>
          ) : (
            <div className="schedule-help-list">
              <div className="schedule-help-item">
                <Typography.Text strong>Python</Typography.Text>
                <Typography.Text type="secondary">适合 API 调用、数据处理和报表生成。</Typography.Text>
              </div>
              <div className="schedule-help-item">
                <Typography.Text strong>JavaScript</Typography.Text>
                <Typography.Text type="secondary">适合 Node 环境下的签到、通知和脚本迁移。</Typography.Text>
              </div>
              <div className="schedule-help-item">
                <Typography.Text strong>Bash</Typography.Text>
                <Typography.Text type="secondary">适合文件整理、系统巡检和宿主机命令编排。</Typography.Text>
              </div>
            </div>
          )}
        </Card>
      </div>
    </div>
  );
}

function ScheduleStatCard(props: {
  icon: ReactNode;
  label: string;
  value: string;
  helper: string;
}) {
  return (
    <div className="schedule-stat-card">
      <div className="schedule-stat-icon">{props.icon}</div>
      <Typography.Text type="secondary">{props.label}</Typography.Text>
      <Typography.Title level={4} className="schedule-stat-value">
        {props.value}
      </Typography.Title>
      <Typography.Text type="secondary">{props.helper}</Typography.Text>
    </div>
  );
}

function languageColor(value: ScriptItem["language"]) {
  if (value === "py") return "blue";
  if (value === "js") return "gold";
  return "green";
}
