import {
  FolderOpenOutlined,
  LinkOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Empty,
  Form,
  Input,
  Modal,
  Radio,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import { useEffect, useState } from "react";
import { apiRequest } from "../lib/api";
import type { OpenRestyStatus, Website } from "../lib/types";

export function WebsitesPage() {
  const [status, setStatus] = useState<OpenRestyStatus | null>(null);
  const [websites, setWebsites] = useState<Website[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm();
  const websiteType = Form.useWatch("type", form) ?? "static";

  const loadData = async () => {
    setLoading(true);
    try {
      const [statusResponse, websiteResponse] = await Promise.all([
        apiRequest<OpenRestyStatus>("/api/openresty/status"),
        apiRequest<{ items: Website[] }>("/api/websites"),
      ]);
      setStatus(statusResponse);
      setWebsites(websiteResponse.items);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadData();
  }, []);

  const createWebsite = async (values: Record<string, string>) => {
    setSubmitting(true);
    try {
      await apiRequest<{ approval: { id: string } }>("/api/websites", {
        method: "POST",
        body: JSON.stringify({
          name: values.name,
          type: values.type,
          domain: values.domain,
          proxy_pass: values.proxy_pass ?? "",
        }),
      });
      message.success("已生成 OpenResty 站点创建审批单");
      setModalOpen(false);
      form.resetFields();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="page-grid">
      <div>
        <Typography.Title className="page-title">OpenResty</Typography.Title>
        <Typography.Paragraph className="page-subtitle">
          操作固定 OpenResty 容器里的站点入口，支持静态站点和反向代理，所有写操作仍然走审批。
        </Typography.Paragraph>
        <Typography.Paragraph type="secondary" style={{ marginTop: -8 }}>
          如果固定 OpenResty 容器还没部署，可以先到应用商店部署 OpenResty 模板。
        </Typography.Paragraph>
      </div>

      <Card className="glass-card">
        <Space direction="vertical" size="middle" style={{ width: "100%" }}>
          <div
            style={{ display: "flex", justifyContent: "space-between", gap: 16, alignItems: "start" }}
          >
            <div>
              <Typography.Title level={4} style={{ marginTop: 0 }}>
                OpenResty 容器
              </Typography.Title>
              <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                创建站点前会先检查容器状态，然后在容器内执行配置校验和 reload。
              </Typography.Paragraph>
            </div>
            <Space>
              <Button icon={<ReloadOutlined />} onClick={() => void loadData()}>
                刷新
              </Button>
              <Button
                type="primary"
                onClick={() => {
                  form.resetFields();
                  form.setFieldsValue({ type: "static" });
                  setModalOpen(true);
                }}
                disabled={!status?.ready}
              >
                创建站点
              </Button>
            </Space>
          </div>

          <Alert
            showIcon
            type={status?.ready ? "success" : status?.exists ? "warning" : "error"}
            message={status?.message || "正在读取 OpenResty 状态"}
            description={
              status ? (
                <Space direction="vertical" size="small">
                  <Typography.Text type="secondary">
                    容器名：{status.container_name || "-"}
                  </Typography.Text>
                  <Typography.Text type="secondary">
                    容器状态：{status.container_status || "-"}
                  </Typography.Text>
                  <Typography.Text type="secondary">
                    配置挂载目录：{status.host_config_dir || "-"}
                  </Typography.Text>
                  <Typography.Text type="secondary">
                    站点挂载目录：{status.host_site_dir || "-"}
                  </Typography.Text>
                </Space>
              ) : null
            }
          />
        </Space>
      </Card>

      <Card className="glass-card">
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
              render: (value: Website["type"]) => (
                <Tag color={value === "static" ? "blue" : "green"}>
                  {value === "static" ? "静态站点" : "反向代理"}
                </Tag>
              ),
            },
            { title: "域名", dataIndex: "domain" },
            {
              title: "目标",
              render: (_, record) =>
                record.type === "static" ? (
                  <Space size="small">
                    <FolderOpenOutlined />
                    {record.root_path}
                  </Space>
                ) : (
                  <Space size="small">
                    <LinkOutlined />
                    {record.proxy_pass}
                  </Space>
                ),
            },
            {
              title: "状态",
              dataIndex: "status",
              render: (value: string) => <Tag>{value}</Tag>,
            },
            { title: "配置文件", dataIndex: "config_path" },
          ]}
        />
      </Card>

      <Modal
        open={modalOpen}
        title="创建站点"
        okText="生成审批单"
        cancelText="取消"
        onCancel={() => {
          setModalOpen(false);
          form.resetFields();
        }}
        onOk={() => void form.submit()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form form={form} layout="vertical" initialValues={{ type: "static" }} onFinish={createWebsite}>
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
            extra="只允许小写字母、数字、下划线和中划线。"
          >
            <Input placeholder="my-site" />
          </Form.Item>

          <Form.Item
            label="站点类型"
            name="type"
            rules={[{ required: true, message: "请选择站点类型" }]}
          >
            <Radio.Group
              options={[
                { label: "静态站点", value: "static" },
                { label: "反向代理", value: "proxy" },
              ]}
            />
          </Form.Item>

          <Form.Item
            label="域名"
            name="domain"
            rules={[{ required: true, message: "请输入域名" }]}
          >
            <Input placeholder="example.com" />
          </Form.Item>

          {websiteType === "proxy" ? (
            <Form.Item
              label="代理地址"
              name="proxy_pass"
              rules={[{ required: true, message: "请输入代理地址" }]}
              extra="示例：http://127.0.0.1:3000"
            >
              <Input placeholder="http://127.0.0.1:3000" />
            </Form.Item>
          ) : (
            <Alert
              showIcon
              type="info"
              message="静态站点根目录会自动创建在 OpenResty 数据挂载目录下。"
            />
          )}
        </Form>
      </Modal>
    </div>
  );
}
