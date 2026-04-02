import {
  ClockCircleOutlined,
  CloudOutlined,
  DesktopOutlined,
  EditOutlined,
  ReloadOutlined,
  SwapOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Divider,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Spin,
  Tag,
  Typography,
  message,
} from "antd";
import { useEffect, useState } from "react";
import { bytesToSize } from "../../shared/lib/format";
import {
  createSwap,
  getSystemConfig,
  removeSwap,
  updateDNS,
  updateHostname,
  updateTimezone,
} from "./api";
import type { SystemConfig } from "./types";

const COMMON_TIMEZONES = [
  "Asia/Shanghai",
  "Asia/Tokyo",
  "Asia/Singapore",
  "Asia/Hong_Kong",
  "Asia/Seoul",
  "Asia/Kolkata",
  "America/New_York",
  "America/Chicago",
  "America/Denver",
  "America/Los_Angeles",
  "Europe/London",
  "Europe/Paris",
  "Europe/Berlin",
  "Europe/Moscow",
  "Australia/Sydney",
  "Pacific/Auckland",
  "UTC",
];

export function SystemConfigPage() {
  const [config, setConfig] = useState<SystemConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();

  const [hostnameEditing, setHostnameEditing] = useState(false);
  const [hostnameValue, setHostnameValue] = useState("");
  const [hostnameSubmitting, setHostnameSubmitting] = useState(false);

  const [dnsEditing, setDnsEditing] = useState(false);
  const [dnsValue, setDnsValue] = useState("");
  const [dnsSubmitting, setDnsSubmitting] = useState(false);

  const [timezoneEditing, setTimezoneEditing] = useState(false);
  const [timezoneValue, setTimezoneValue] = useState("");
  const [timezoneSubmitting, setTimezoneSubmitting] = useState(false);

  const [swapModalOpen, setSwapModalOpen] = useState(false);
  const [swapSize, setSwapSize] = useState(1024);
  const [swapSubmitting, setSwapSubmitting] = useState(false);
  const [swapRemoving, setSwapRemoving] = useState(false);

  const loadConfig = async () => {
    setLoading(true);
    try {
      const data = await getSystemConfig();
      setConfig(data);
      setError(undefined);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载配置失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadConfig();
  }, []);

  const handleHostnameSave = async () => {
    if (!hostnameValue.trim()) return;
    setHostnameSubmitting(true);
    try {
      await updateHostname(hostnameValue.trim());
      message.success("主机名已更新");
      setHostnameEditing(false);
      await loadConfig();
    } catch (err) {
      message.error(err instanceof Error ? err.message : "更新失败");
    } finally {
      setHostnameSubmitting(false);
    }
  };

  const handleDNSSave = async () => {
    const servers = dnsValue
      .split("\n")
      .map((s) => s.trim())
      .filter(Boolean);
    if (!servers.length) return;
    setDnsSubmitting(true);
    try {
      await updateDNS(servers);
      message.success("DNS 已更新");
      setDnsEditing(false);
      await loadConfig();
    } catch (err) {
      message.error(err instanceof Error ? err.message : "更新失败");
    } finally {
      setDnsSubmitting(false);
    }
  };

  const handleTimezoneSave = async () => {
    if (!timezoneValue) return;
    setTimezoneSubmitting(true);
    try {
      await updateTimezone(timezoneValue);
      message.success("时区已更新");
      setTimezoneEditing(false);
      await loadConfig();
    } catch (err) {
      message.error(err instanceof Error ? err.message : "更新失败");
    } finally {
      setTimezoneSubmitting(false);
    }
  };

  const handleSwapCreate = async () => {
    setSwapSubmitting(true);
    try {
      await createSwap(swapSize);
      message.success("Swap 已创建");
      setSwapModalOpen(false);
      await loadConfig();
    } catch (err) {
      message.error(err instanceof Error ? err.message : "创建 Swap 失败");
    } finally {
      setSwapSubmitting(false);
    }
  };

  const handleSwapRemove = async () => {
    setSwapRemoving(true);
    try {
      await removeSwap();
      message.success("Swap 已移除");
      await loadConfig();
    } catch (err) {
      message.error(err instanceof Error ? err.message : "移除 Swap 失败");
    } finally {
      setSwapRemoving(false);
    }
  };

  if (loading) {
    return (
      <div className="page-grid">
        <Spin size="large" style={{ margin: "20vh auto", display: "block" }} />
      </div>
    );
  }

  if (error) {
    return (
      <div className="page-grid">
        <Alert showIcon type="error" message={error} />
      </div>
    );
  }

  return (
    <div className="page-grid">
      <div className="page-inline-bar">
        <Typography.Text type="secondary">查看和修改宿主机的基础配置。</Typography.Text>
        <Button icon={<ReloadOutlined />} onClick={() => void loadConfig()}>
          刷新
        </Button>
      </div>

      <div className="sysconfig-grid">
        <Card className="glass-card" style={{ borderRadius: 18 }}>
          <div className="sysconfig-card-header">
            <div className="sysconfig-card-icon">
              <DesktopOutlined />
            </div>
            <div>
              <Typography.Title level={5} style={{ margin: 0 }}>
                主机名
              </Typography.Title>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                设置服务器的主机名标识
              </Typography.Text>
            </div>
          </div>
          <Divider style={{ margin: "16px 0" }} />
          {hostnameEditing ? (
            <Space direction="vertical" style={{ width: "100%" }}>
              <Input
                value={hostnameValue}
                onChange={(e) => setHostnameValue(e.target.value)}
                placeholder="输入新主机名"
                onPressEnter={() => void handleHostnameSave()}
              />
              <Space>
                <Button
                  type="primary"
                  loading={hostnameSubmitting}
                  onClick={() => void handleHostnameSave()}
                >
                  保存
                </Button>
                <Button onClick={() => setHostnameEditing(false)}>取消</Button>
              </Space>
            </Space>
          ) : (
            <div className="sysconfig-value-row">
              <Typography.Text strong style={{ fontSize: 16 }}>
                {config.hostname || "未设置"}
              </Typography.Text>
              <Button
                type="text"
                icon={<EditOutlined />}
                onClick={() => {
                  setHostnameValue(config.hostname);
                  setHostnameEditing(true);
                }}
              >
                修改
              </Button>
            </div>
          )}
        </Card>

        <Card className="glass-card" style={{ borderRadius: 18 }}>
          <div className="sysconfig-card-header">
            <div className="sysconfig-card-icon">
              <CloudOutlined />
            </div>
            <div>
              <Typography.Title level={5} style={{ margin: 0 }}>
                DNS 服务器
              </Typography.Title>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                配置域名解析服务器地址
              </Typography.Text>
            </div>
          </div>
          <Divider style={{ margin: "16px 0" }} />
          {dnsEditing ? (
            <Space direction="vertical" style={{ width: "100%" }}>
              <Input.TextArea
                value={dnsValue}
                onChange={(e) => setDnsValue(e.target.value)}
                placeholder={"8.8.8.8\n8.8.4.4\n114.114.114.114"}
                rows={4}
                spellCheck={false}
              />
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                每行一个 DNS 地址
              </Typography.Text>
              <Space>
                <Button
                  type="primary"
                  loading={dnsSubmitting}
                  onClick={() => void handleDNSSave()}
                >
                  保存
                </Button>
                <Button onClick={() => setDnsEditing(false)}>取消</Button>
              </Space>
            </Space>
          ) : (
            <div className="sysconfig-value-row">
              <Space wrap size={[6, 6]}>
                {config.dns.length > 0 ? (
                  config.dns.map((server) => (
                    <Tag key={server} color="blue">
                      {server}
                    </Tag>
                  ))
                ) : (
                  <Typography.Text type="secondary">未配置</Typography.Text>
                )}
              </Space>
              <Button
                type="text"
                icon={<EditOutlined />}
                onClick={() => {
                  setDnsValue(config.dns.join("\n"));
                  setDnsEditing(true);
                }}
              >
                修改
              </Button>
            </div>
          )}
        </Card>

        <Card className="glass-card" style={{ borderRadius: 18 }}>
          <div className="sysconfig-card-header">
            <div className="sysconfig-card-icon">
              <ClockCircleOutlined />
            </div>
            <div>
              <Typography.Title level={5} style={{ margin: 0 }}>
                时区
              </Typography.Title>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                设置系统时区以确保时间显示正确
              </Typography.Text>
            </div>
          </div>
          <Divider style={{ margin: "16px 0" }} />
          {timezoneEditing ? (
            <Space direction="vertical" style={{ width: "100%" }}>
              <Select
                value={timezoneValue}
                onChange={setTimezoneValue}
                style={{ width: "100%" }}
                showSearch
                placeholder="选择时区"
                options={COMMON_TIMEZONES.map((tz) => ({ value: tz, label: tz }))}
              />
              <Space>
                <Button
                  type="primary"
                  loading={timezoneSubmitting}
                  onClick={() => void handleTimezoneSave()}
                >
                  保存
                </Button>
                <Button onClick={() => setTimezoneEditing(false)}>取消</Button>
              </Space>
            </Space>
          ) : (
            <div className="sysconfig-value-row">
              <Tag color="purple">{config.timezone || "未设置"}</Tag>
              <Button
                type="text"
                icon={<EditOutlined />}
                onClick={() => {
                  setTimezoneValue(config.timezone);
                  setTimezoneEditing(true);
                }}
              >
                修改
              </Button>
            </div>
          )}
        </Card>

        <Card className="glass-card" style={{ borderRadius: 18 }}>
          <div className="sysconfig-card-header">
            <div className="sysconfig-card-icon">
              <SwapOutlined />
            </div>
            <div>
              <Typography.Title level={5} style={{ margin: 0 }}>
                Swap 交换分区
              </Typography.Title>
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                管理虚拟内存交换空间
              </Typography.Text>
            </div>
          </div>
          <Divider style={{ margin: "16px 0" }} />
          {config.swap.total > 0 ? (
            <Space direction="vertical" style={{ width: "100%" }}>
              <Descriptions column={1} size="small">
                <Descriptions.Item label="总大小">
                  {bytesToSize(config.swap.total)}
                </Descriptions.Item>
                <Descriptions.Item label="已使用">
                  {bytesToSize(config.swap.used)}
                </Descriptions.Item>
                {config.swap.file ? (
                  <Descriptions.Item label="文件">{config.swap.file}</Descriptions.Item>
                ) : null}
              </Descriptions>
              <Space>
                <Button
                  onClick={() => setSwapModalOpen(true)}
                >
                  调整大小
                </Button>
                <Button danger loading={swapRemoving} onClick={() => void handleSwapRemove()}>
                  移除 Swap
                </Button>
              </Space>
            </Space>
          ) : (
            <div className="sysconfig-value-row">
              <Typography.Text type="secondary">未启用 Swap</Typography.Text>
              <Button type="primary" onClick={() => setSwapModalOpen(true)}>
                创建 Swap
              </Button>
            </div>
          )}
        </Card>
      </div>

      <Modal
        open={swapModalOpen}
        title="创建 / 调整 Swap"
        okText="确认"
        cancelText="取消"
        onCancel={() => setSwapModalOpen(false)}
        onOk={() => void handleSwapCreate()}
        confirmLoading={swapSubmitting}
      >
        <Space direction="vertical" style={{ width: "100%", marginTop: 12 }}>
          <Typography.Text>设置 Swap 大小（MB）：</Typography.Text>
          <InputNumber
            min={64}
            max={65536}
            step={256}
            value={swapSize}
            onChange={(v) => setSwapSize(v ?? 1024)}
            style={{ width: "100%" }}
            addonAfter="MB"
          />
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            建议设为内存的 1-2 倍，常见值：512、1024、2048、4096
          </Typography.Text>
        </Space>
      </Modal>
    </div>
  );
}
