import { Button, Card, Popconfirm, Space, Table, Tag, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { apiRequest } from "../lib/api";
import type { Approval } from "../lib/types";

export function ApprovalsPage() {
  const [approvals, setApprovals] = useState<Approval[]>([]);
  const [loading, setLoading] = useState(true);

  const loadApprovals = async () => {
    setLoading(true);
    try {
      const response = await apiRequest<{ items: Approval[] }>("/api/approvals");
      setApprovals(response.items);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadApprovals();
  }, []);

  const approve = async (id: string) => {
    await apiRequest(`/api/approvals/${id}/approve`, { method: "POST" });
    message.success("审批已执行");
    void loadApprovals();
  };

  const reject = async (id: string) => {
    await apiRequest(`/api/approvals/${id}/reject`, {
      method: "POST",
      body: JSON.stringify({ reason: "手动拒绝" }),
    });
    message.success("审批已拒绝");
    void loadApprovals();
  };

  return (
    <div className="page-grid">
      <div>
        <Typography.Title className="page-title">审批中心</Typography.Title>
        <Typography.Paragraph className="page-subtitle">
          所有写操作都统一走这里，包括 UI 操作和 AI 生成的执行计划。
        </Typography.Paragraph>
      </div>

      <Card className="glass-card">
        <Table
          rowKey="id"
          loading={loading}
          dataSource={approvals}
          columns={[
            { title: "摘要", dataIndex: "summary" },
            { title: "来源", dataIndex: "source" },
            { title: "动作", dataIndex: "action" },
            {
              title: "状态",
              dataIndex: "status",
              render: (value: string) => {
                const color = value === "pending" ? "gold" : value === "approved" ? "green" : value === "failed" ? "red" : "default";
                return <Tag color={color}>{value}</Tag>;
              },
            },
            { title: "错误", dataIndex: "error_message", ellipsis: true },
            {
              title: "处理",
              render: (_, record) =>
                record.status === "pending" ? (
                  <Space>
                    <Button type="primary" size="small" onClick={() => void approve(record.id)}>
                      批准
                    </Button>
                    <Popconfirm title="确认拒绝这个审批单？" onConfirm={() => void reject(record.id)}>
                      <Button size="small">拒绝</Button>
                    </Popconfirm>
                  </Space>
                ) : (
                  <Typography.Text type="secondary">已处理</Typography.Text>
                ),
            },
          ]}
        />
      </Card>
    </div>
  );
}

