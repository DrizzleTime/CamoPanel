import {
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  GlobalOutlined,
  RocketOutlined,
  WarningOutlined,
} from "@ant-design/icons";
import { Alert, Button, Card, Empty, Popover, Progress, Skeleton, Space, Tag, Typography, theme } from "antd";
import type { ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { bytesToSize } from "../../shared/lib/format";
import { useDashboardStream } from "./hooks/useDashboardStream";
import type { HostMetricsPoint, TopProcess } from "./types";

type ChartSeries = {
  label: string;
  color: string;
  values: number[];
  formatter: (value: number) => string;
};

export function DashboardPage() {
  const { token } = theme.useToken();
  const navigate = useNavigate();
  const { data, loading, streamState, error } = useDashboardStream();

  const summary = data?.metrics.summary ?? null;
  const history = data?.metrics.history ?? [];
  const projects = data?.projects ?? [];
  const websites = data?.websites ?? [];

  const runningProjects = projects.filter((item) => item.runtime.status === "running").length;
  const stoppedProjects = projects.filter((item) => item.runtime.status === "stopped").length;
  const degradedProjects = projects.filter((item) => item.runtime.status === "degraded").length;
  const attentionProjects = projects
    .filter((item) => item.runtime.status !== "running" || item.last_error)
    .slice(0, 5);
  const recentWebsites = websites.slice(0, 5);

  const cpuPercent = clamp(summary?.cpu_percent ?? 0);
  const memoryPercent = ratioToPercent(summary?.memory_used ?? 0, summary?.memory_total ?? 0);
  const diskPercent = ratioToPercent(summary?.disk_used ?? 0, summary?.disk_total ?? 0);
  const loadPercent = ratioToPercent(summary?.load_1 ?? 0, Math.max(summary?.cpu_cores ?? 1, 1));
  const historyWindow = formatTimeWindow((data?.metrics.sample_interval_seconds ?? 0) * history.length);
  const surfaceStyle = {
    border: `1px solid ${token.colorBorderSecondary}`,
    boxShadow: token.boxShadowTertiary,
  };
  const streamTag = streamState === "live" ? "实时推送中" : streamState === "retrying" ? "重连中" : "连接中";
  const streamTagColor = streamState === "live" ? "green" : "gold";
  const softTagStyle = {
    marginInlineEnd: 0,
    paddingInline: 12,
    paddingBlock: 6,
    borderRadius: 999,
    border: `1px solid ${token.colorBorderSecondary}`,
    background: token.colorFillAlter,
    color: token.colorTextSecondary,
  };

  const cpuMemorySeries: ChartSeries[] = [
    {
      label: "CPU",
      color: token.colorText,
      values: history.map((item) => item.cpu_percent),
      formatter: formatPercent,
    },
    {
      label: "内存",
      color: token.colorTextSecondary,
      values: history.map((item) => ratioToPercent(item.memory_used, item.memory_total)),
      formatter: formatPercent,
    },
  ];

  const networkSeries: ChartSeries[] = [
    {
      label: "上行",
      color: token.colorText,
      values: history.map((item) => item.network_tx_rate),
      formatter: formatSpeed,
    },
    {
      label: "下行",
      color: token.colorTextSecondary,
      values: history.map((item) => item.network_rx_rate),
      formatter: formatSpeed,
    },
  ];

  const diskSeries: ChartSeries[] = [
    {
      label: "读取",
      color: token.colorText,
      values: history.map((item) => item.disk_read_rate),
      formatter: formatSpeed,
    },
    {
      label: "写入",
      color: token.colorTextSecondary,
      values: history.map((item) => item.disk_write_rate),
      formatter: formatSpeed,
    },
  ];

  return (
    <div className="page-grid">
      {error ? <Alert showIcon type="error" message={error} /> : null}

      {loading && !data ? (
        <Card className="glass-card" variant="borderless" style={surfaceStyle}>
          <Skeleton active paragraph={{ rows: 12 }} />
        </Card>
      ) : null}

      {data ? (
        <>
          <div className="dashboard-kpi-grid">
            <KpiCard
              icon={<RocketOutlined />}
              label="项目"
              value={String(projects.length)}
              helper={`${runningProjects} 个运行中`}
            />
            <KpiCard
              icon={<GlobalOutlined />}
              label="网站"
              value={String(websites.length)}
              helper="固定 OpenResty 容器"
            />
            <KpiCard
              icon={<WarningOutlined />}
              label="异常项目"
              value={String(attentionProjects.length)}
              helper={`${degradedProjects} 个退化 / ${stoppedProjects} 个停止`}
            />
            <KpiCard
              icon={<CheckCircleOutlined />}
              label="1 分钟负载"
              value={(summary?.load_1 ?? 0).toFixed(2)}
              helper={`${summary?.cpu_cores ?? 0} 核 CPU`}
            />
          </div>

          <div className="dashboard-layout">
            <div className="dashboard-main">
              <Card className="glass-card" title="资源状态" variant="borderless" style={surfaceStyle}>
                <div className="dashboard-ring-grid">
                  <StatusRing
                    label="CPU"
                    percent={cpuPercent}
                    detail={`${summary?.cpu_cores ?? 0} 核 / 负载 ${summary?.load_5.toFixed(2) ?? "0.00"}`}
                    tone={toneFromPercent(cpuPercent)}
                    tooltipContent={
                      <StatusTooltip
                        rows={[
                          ["使用率", `${cpuPercent.toFixed(1)}%`],
                          ["核心数", `${summary?.cpu_cores ?? 0} 核`],
                          ["1 分钟负载", (summary?.load_1 ?? 0).toFixed(2)],
                          ["5 分钟负载", (summary?.load_5 ?? 0).toFixed(2)],
                        ]}
                        processes={summary?.top_cpu}
                        processLabel="Top 5 CPU 进程"
                        processValueKey="cpu"
                      />
                    }
                  />
                  <StatusRing
                    label="内存"
                    percent={memoryPercent}
                    detail={`${bytesToSize(summary?.memory_used ?? 0)} / ${bytesToSize(summary?.memory_total ?? 0)}`}
                    tone={toneFromPercent(memoryPercent)}
                    tooltipContent={
                      <StatusTooltip
                        rows={[
                          ["使用率", `${memoryPercent.toFixed(1)}%`],
                          ["已用", bytesToSize(summary?.memory_used ?? 0)],
                          ["总量", bytesToSize(summary?.memory_total ?? 0)],
                          ["可用", bytesToSize((summary?.memory_total ?? 0) - (summary?.memory_used ?? 0))],
                        ]}
                        processes={summary?.top_memory}
                        processLabel="Top 5 内存进程"
                        processValueKey="memory"
                      />
                    }
                  />
                  <StatusRing
                    label="磁盘"
                    percent={diskPercent}
                    detail={`${bytesToSize(summary?.disk_used ?? 0)} / ${bytesToSize(summary?.disk_total ?? 0)}`}
                    tone={toneFromPercent(diskPercent)}
                    tooltipContent={
                      <StatusTooltip
                        rows={[
                          ["使用率", `${diskPercent.toFixed(1)}%`],
                          ["已用", bytesToSize(summary?.disk_used ?? 0)],
                          ["总量", bytesToSize(summary?.disk_total ?? 0)],
                          ["可用", bytesToSize((summary?.disk_total ?? 0) - (summary?.disk_used ?? 0))],
                        ]}
                      />
                    }
                  />
                  <StatusRing
                    label="负载"
                    percent={loadPercent}
                    detail={`${(summary?.load_1 ?? 0).toFixed(2)} / ${summary?.cpu_cores ?? 0}`}
                    tone={toneFromPercent(loadPercent)}
                    tooltipContent={
                      <StatusTooltip
                        rows={[
                          ["1 分钟负载", (summary?.load_1 ?? 0).toFixed(2)],
                          ["5 分钟负载", (summary?.load_5 ?? 0).toFixed(2)],
                          ["CPU 核心数", `${summary?.cpu_cores ?? 0}`],
                          ["负载 / 核心", `${loadPercent.toFixed(1)}%`],
                        ]}
                      />
                    }
                  />
                </div>
              </Card>

              <div className="dashboard-chart-grid">
                <MetricChartCard
                  className="dashboard-chart-wide"
                  title="资源趋势"
                  description={historyWindow}
                  points={history}
                  axisFormatter={formatPercent}
                  series={cpuMemorySeries}
                />
                <MetricChartCard
                  title="网络吞吐"
                  description={historyWindow}
                  points={history}
                  axisFormatter={formatSpeed}
                  series={networkSeries}
                />
                <MetricChartCard
                  title="磁盘 IO"
                  description={historyWindow}
                  points={history}
                  axisFormatter={formatSpeed}
                  series={diskSeries}
                />
              </div>
            </div>

            <div className="dashboard-side">
              <Card className="glass-card" title="宿主机信息" variant="borderless" style={surfaceStyle}>
                {summary ? (
                  <div className="dashboard-info-grid">
                    <InfoPair label="主机名" value={summary.hostname} />
                    <InfoPair label="系统" value={`${summary.os} / ${summary.platform}`} />
                    <InfoPair label="内核" value={summary.kernel} />
                    <InfoPair label="架构" value={summary.architecture} />
                  </div>
                ) : (
                  <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无宿主机数据" />
                )}
              </Card>

              <Card className="glass-card" title="待处理" variant="borderless" style={surfaceStyle}>
                <div className="dashboard-alert-list">
                  <StatusItem
                    icon={<WarningOutlined />}
                    label="异常项目"
                    value={`${attentionProjects.length} 个`}
                    tone={attentionProjects.length > 0 ? "danger" : "good"}
                  />
                  <StatusItem
                    icon={<GlobalOutlined />}
                    label="网站数量"
                    value={`${websites.length} 个`}
                    tone={websites.length > 0 ? "good" : "warn"}
                  />
                  <StatusItem
                    icon={<ExclamationCircleOutlined />}
                    label="停止项目"
                    value={`${stoppedProjects} 个`}
                    tone={stoppedProjects > 0 ? "warn" : "good"}
                  />
                  <StatusItem
                    icon={<ExclamationCircleOutlined />}
                    label="退化项目"
                    value={`${degradedProjects} 个`}
                    tone={degradedProjects > 0 ? "danger" : "good"}
                  />
                </div>

                <div className="dashboard-action-row">
                  <Button type="primary" onClick={() => navigate("/app/store")}>
                    新建应用
                  </Button>
                  <Button onClick={() => navigate("/app/containers")}>容器管理</Button>
                  <Button onClick={() => navigate("/app/websites")}>网站管理</Button>
                </div>
              </Card>

              <Card className="glass-card" title="需要关注的项目" variant="borderless" style={surfaceStyle}>
                {attentionProjects.length ? (
                  <div style={{ display: "grid", gap: 4 }}>
                    {attentionProjects.map((item, index) => (
                      <div
                        key={item.id}
                        style={{
                          display: "flex",
                          alignItems: "flex-start",
                          justifyContent: "space-between",
                          gap: 12,
                          padding: "12px 0",
                          borderBottom:
                            index === attentionProjects.length - 1
                              ? "none"
                              : `1px solid ${token.colorBorderSecondary}`,
                        }}
                      >
                        <div>
                          <Typography.Text strong>{item.name}</Typography.Text>
                          <Typography.Paragraph
                            style={{
                              margin: "4px 0 0",
                              color: token.colorTextSecondary,
                              fontSize: 13,
                            }}
                          >
                            {item.last_error || `${item.template_id} / ${item.runtime.status}`}
                          </Typography.Paragraph>
                        </div>
                        <Tag color={projectStatusColor(item.runtime.status)}>{item.runtime.status}</Tag>
                      </div>
                    ))}
                  </div>
                ) : (
                  <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="当前没有异常项目" />
                )}
              </Card>

              <Card className="glass-card" title="最近网站" variant="borderless" style={surfaceStyle}>
                {recentWebsites.length ? (
                  <div style={{ display: "grid", gap: 4 }}>
                    {recentWebsites.map((item, index) => (
                      <div
                        key={item.id}
                        style={{
                          display: "flex",
                          alignItems: "flex-start",
                          justifyContent: "space-between",
                          gap: 12,
                          padding: "12px 0",
                          borderBottom:
                            index === recentWebsites.length - 1
                              ? "none"
                              : `1px solid ${token.colorBorderSecondary}`,
                        }}
                      >
                        <div>
                          <Typography.Text strong>{item.domain}</Typography.Text>
                          <Typography.Paragraph
                            style={{
                              margin: "4px 0 0",
                              color: token.colorTextSecondary,
                              fontSize: 13,
                            }}
                          >
                            {item.name} / {item.type === "static" ? "静态站点" : "反向代理"}
                          </Typography.Paragraph>
                        </div>
                        <Tag>{item.status}</Tag>
                      </div>
                    ))}
                  </div>
                ) : (
                  <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无站点记录" />
                )}
              </Card>
            </div>
          </div>
        </>
      ) : null}
    </div>
  );
}

function KpiCard({
  icon,
  label,
  value,
  helper,
}: {
  icon: ReactNode;
  label: string;
  value: string;
  helper: string;
}) {
  const { token } = theme.useToken();

  return (
    <Card
      variant="borderless"
      style={{
        border: `1px solid ${token.colorBorderSecondary}`,
        boxShadow: token.boxShadowTertiary,
      }}
      styles={{ body: { padding: 20 } }}
    >
      <Space align="start" size={14}>
        <div
          style={{
            display: "grid",
            placeItems: "center",
            width: 44,
            height: 44,
            borderRadius: 14,
            background: token.colorFillAlter,
            color: token.colorText,
            fontSize: 18,
            flex: "none",
          }}
        >
          {icon}
        </div>
        <div>
          <Typography.Text type="secondary">{label}</Typography.Text>
          <Typography.Title level={2} style={{ margin: "8px 0 0", lineHeight: 1 }}>
            {value}
          </Typography.Title>
          <Typography.Text type="secondary" style={{ display: "block", marginTop: 10 }}>
            {helper}
          </Typography.Text>
        </div>
      </Space>
    </Card>
  );
}

function StatusTooltip({
  rows,
  processes,
  processLabel,
  processValueKey,
}: {
  rows: [string, string][];
  processes?: TopProcess[];
  processLabel?: string;
  processValueKey?: "cpu" | "memory";
}) {
  return (
    <div style={{ minWidth: 220 }}>
      {rows.map(([key, val]) => (
        <div
          key={key}
          style={{
            display: "flex",
            justifyContent: "space-between",
            gap: 24,
            padding: "4px 0",
          }}
        >
          <Typography.Text type="secondary">{key}</Typography.Text>
          <Typography.Text strong>{val}</Typography.Text>
        </div>
      ))}
      {processes && processes.length > 0 && (
        <>
          <div
            style={{
              borderTop: "1px solid var(--ant-color-border-secondary, #f0f0f0)",
              margin: "8px 0 6px",
            }}
          />
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {processLabel ?? "Top 进程"}
          </Typography.Text>
          <div style={{ marginTop: 4 }}>
            {processes.map((p) => (
              <div
                key={p.pid}
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  gap: 16,
                  padding: "2px 0",
                  fontSize: 12,
                }}
              >
                <Typography.Text
                  ellipsis
                  style={{ maxWidth: 140, fontSize: 12 }}
                  title={`${p.name} (PID ${p.pid})`}
                >
                  {p.name}
                </Typography.Text>
                <Typography.Text strong style={{ fontSize: 12, whiteSpace: "nowrap" }}>
                  {processValueKey === "memory"
                    ? bytesToSize(p.memory_bytes)
                    : `${p.cpu.toFixed(1)}%`}
                </Typography.Text>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
}

function StatusRing({
  label,
  percent,
  detail,
  tone,
  tooltipContent,
}: {
  label: string;
  percent: number;
  detail: string;
  tone: "good" | "warn" | "danger";
  tooltipContent?: ReactNode;
}) {
  const { token } = theme.useToken();

  const ring = (
    <div
      style={{
        display: "grid",
        justifyItems: "center",
        gap: 14,
        padding: "18px 12px",
        borderRadius: token.borderRadiusLG,
        border: `1px solid ${token.colorBorderSecondary}`,
        background: token.colorFillAlter,
        cursor: tooltipContent ? "pointer" : undefined,
        transition: "border-color 0.2s",
      }}
    >
      <Progress
        type="circle"
        percent={Number(percent.toFixed(1))}
        size={132}
        strokeColor={toneColor(token, tone)}
        trailColor={token.colorBorderSecondary}
        format={() => (
          <div style={{ textAlign: "center" }}>
            <Typography.Title level={4} style={{ margin: 0, lineHeight: 1.05 }}>
              {percent.toFixed(1)}%
            </Typography.Title>
            <Typography.Text type="secondary" style={{ display: "block", marginTop: 6, fontSize: 12 }}>
              {label}
            </Typography.Text>
          </div>
        )}
      />
      <Typography.Text style={{ textAlign: "center", color: token.colorTextSecondary }}>
        {detail}
      </Typography.Text>
    </div>
  );

  if (!tooltipContent) return ring;

  return (
    <Popover content={tooltipContent} title={label} trigger="hover" placement="bottom">
      {ring}
    </Popover>
  );
}

function MetricChartCard({
  title,
  description,
  points,
  series,
  axisFormatter,
  className,
}: {
  title: string;
  description: string;
  points: HostMetricsPoint[];
  series: ChartSeries[];
  axisFormatter: (value: number) => string;
  className?: string;
}) {
  const { token } = theme.useToken();

  if (!points.length) {
    return (
      <Card
        className={`glass-card ${className ?? ""}`.trim()}
        title={title}
        variant="borderless"
        style={{
          border: `1px solid ${token.colorBorderSecondary}`,
          boxShadow: token.boxShadowTertiary,
        }}
      >
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="还没有采样数据" />
      </Card>
    );
  }

  const values = series.flatMap((item) => item.values);
  const rawMin = Math.min(...values);
  const rawMax = Math.max(...values);
  const padding = rawMax === rawMin ? Math.max(rawMax, 1) * 0.35 : (rawMax - rawMin) * 0.12;
  const minValue = Math.max(0, rawMin - padding);
  const maxValue = rawMax + padding;
  const startLabel = formatClock(points[0].timestamp);
  const middleLabel = formatClock(points[Math.floor((points.length - 1) / 2)].timestamp);
  const endLabel = formatClock(points[points.length - 1].timestamp);

  return (
    <Card
      className={`glass-card dashboard-chart-card ${className ?? ""}`.trim()}
      title={title}
      extra={<Typography.Text type="secondary">{description}</Typography.Text>}
      variant="borderless"
      style={{
        border: `1px solid ${token.colorBorderSecondary}`,
        boxShadow: token.boxShadowTertiary,
      }}
      styles={{ body: { display: "grid", gap: 16 } }}
    >
      <div
        style={{
          display: "flex",
          flexWrap: "wrap",
          alignItems: "flex-start",
          justifyContent: "space-between",
          gap: 16,
        }}
      >
        <div style={{ display: "flex", flexWrap: "wrap", gap: 12 }}>
          {series.map((item) => (
            <div
              key={item.label}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 10,
                padding: "10px 12px",
                borderRadius: token.borderRadius,
                background: token.colorFillAlter,
              }}
            >
              <span
                style={{
                  width: 10,
                  height: 10,
                  borderRadius: 999,
                  flex: "none",
                  backgroundColor: item.color,
                }}
              />
              <div>
                <Typography.Text type="secondary" style={{ display: "block", fontSize: 12 }}>
                  {item.label}
                </Typography.Text>
                <Typography.Text strong>
                  {item.formatter(item.values[item.values.length - 1] ?? 0)}
                </Typography.Text>
              </div>
            </div>
          ))}
        </div>
        <div
          style={{
            display: "grid",
            gap: 8,
            justifyItems: "end",
            color: token.colorTextSecondary,
            fontSize: 12,
          }}
        >
          <span>{axisFormatter(maxValue)}</span>
          <span>{axisFormatter(minValue)}</span>
        </div>
      </div>

      <div
        style={{
          height: 220,
          borderRadius: token.borderRadiusLG,
          overflow: "hidden",
          background: token.colorFillAlter,
          border: `1px solid ${token.colorBorderSecondary}`,
        }}
      >
        <svg style={{ width: "100%", height: "100%" }} viewBox="0 0 800 220" preserveAspectRatio="none">
          {[0, 1, 2, 3].map((lineIndex) => {
            const y = 22 + (176 / 3) * lineIndex;
            return (
              <line
                key={lineIndex}
                x1="0"
                x2="800"
                y1={y}
                y2={y}
                style={{ stroke: token.colorBorderSecondary, strokeDasharray: "5 6" }}
              />
            );
          })}

          {series.map((item) => {
            const path = buildLinePath(item.values, minValue, maxValue, 800, 220);
            const lastPoint = pointAt(item.values, item.values.length - 1, minValue, maxValue, 800, 220);

            return (
              <g key={item.label}>
                <path d={path} stroke={item.color} strokeWidth="3.5" fill="none" strokeLinecap="round" />
                <circle cx={lastPoint.x} cy={lastPoint.y} r="5.5" fill={item.color} />
                <circle cx={lastPoint.x} cy={lastPoint.y} r="10" fill={item.color} opacity="0.16" />
              </g>
            );
          })}
        </svg>
      </div>

      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          color: token.colorTextSecondary,
          fontSize: 12,
        }}
      >
        <span>{startLabel}</span>
        <span>{middleLabel}</span>
        <span>{endLabel}</span>
      </div>
    </Card>
  );
}

function InfoPair({ label, value }: { label: string; value: string }) {
  const { token } = theme.useToken();

  return (
    <div
      style={{
        padding: "14px 16px",
        borderRadius: token.borderRadius,
        border: `1px solid ${token.colorBorderSecondary}`,
        background: token.colorFillAlter,
      }}
    >
      <Typography.Text type="secondary" style={{ display: "block", fontSize: 12 }}>
        {label}
      </Typography.Text>
      <Typography.Text strong style={{ display: "block", marginTop: 6 }}>
        {value}
      </Typography.Text>
    </div>
  );
}

function StatusItem({
  icon,
  label,
  value,
  tone,
}: {
  icon: ReactNode;
  label: string;
  value: string;
  tone: "good" | "warn" | "danger";
}) {
  const { token } = theme.useToken();

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "12px 14px",
        borderRadius: token.borderRadius,
        border: `1px solid ${token.colorBorderSecondary}`,
        background: token.colorFillAlter,
      }}
    >
      <div
        style={{
          display: "grid",
          placeItems: "center",
          width: 40,
          height: 40,
          borderRadius: 14,
          fontSize: 16,
          flex: "none",
          background: toneBackground(token, tone),
          color: toneColor(token, tone),
        }}
      >
        {icon}
      </div>
      <div>
        <Typography.Text type="secondary" style={{ display: "block", fontSize: 12 }}>
          {label}
        </Typography.Text>
        <Typography.Text strong style={{ display: "block", marginTop: 4 }}>
          {value}
        </Typography.Text>
      </div>
    </div>
  );
}

function buildLinePath(values: number[], minValue: number, maxValue: number, width: number, height: number) {
  if (values.length === 0) {
    return "";
  }

  return values
    .map((value, index) => {
      const point = pointAt(values, index, minValue, maxValue, width, height);
      return `${index === 0 ? "M" : "L"} ${point.x} ${point.y}`;
    })
    .join(" ");
}

function pointAt(values: number[], index: number, minValue: number, maxValue: number, width: number, height: number) {
  const horizontalPadding = 14;
  const verticalPadding = 22;
  const plotWidth = width - horizontalPadding * 2;
  const plotHeight = height - verticalPadding * 2;
  const denominator = Math.max(values.length - 1, 1);
  const range = Math.max(maxValue - minValue, 1);

  const x = horizontalPadding + (plotWidth * index) / denominator;
  const y = verticalPadding + plotHeight - ((values[index] - minValue) / range) * plotHeight;
  return { x, y };
}

function ratioToPercent(value: number, total: number) {
  if (!total) return 0;
  return clamp((value / total) * 100);
}

function clamp(value: number) {
  if (Number.isNaN(value)) return 0;
  return Math.min(100, Math.max(0, value));
}

function toneFromPercent(value: number) {
  if (value >= 85) return "danger";
  if (value >= 65) return "warn";
  return "good";
}

function toneColor(token: ReturnType<typeof theme.useToken>["token"], tone: "good" | "warn" | "danger") {
  switch (tone) {
    case "danger":
      return token.colorError;
    case "warn":
      return token.colorWarning;
    default:
      return token.colorSuccess;
  }
}

function toneBackground(token: ReturnType<typeof theme.useToken>["token"], tone: "good" | "warn" | "danger") {
  switch (tone) {
    case "danger":
      return token.colorErrorBg;
    case "warn":
      return token.colorWarningBg;
    default:
      return token.colorSuccessBg;
  }
}

function projectStatusColor(status: string) {
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

function formatPercent(value: number) {
  return `${value.toFixed(1)}%`;
}

function formatSpeed(value: number) {
  return `${bytesToSize(value)}/s`;
}

function formatDateTime(value: string) {
  return new Date(value).toLocaleString("zh-CN", {
    hour12: false,
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function formatClock(value: string) {
  return new Date(value).toLocaleTimeString("zh-CN", {
    hour12: false,
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatTimeWindow(seconds: number) {
  if (!seconds) {
    return "采样初始化中";
  }
  if (seconds < 60) {
    return `最近 ${seconds} 秒`;
  }
  const minutes = Math.max(1, Math.round(seconds / 60));
  return `最近 ${minutes} 分钟`;
}
