import { LockOutlined, UserOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Form, Input, Typography, message } from "antd";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { ApiError } from "../lib/api";
import { useAuthStore } from "../store/auth";

export function LoginPage() {
  const navigate = useNavigate();
  const login = useAuthStore((state) => state.login);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string>();

  const onFinish = async (values: { username: string; password: string }) => {
    setSubmitting(true);
    setError(undefined);
    try {
      await login(values.username, values.password);
      message.success("登录成功");
      navigate("/app/dashboard", { replace: true });
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "登录失败");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="login-wrap">
      <Card className="glass-card login-panel" variant="borderless">
        <Typography.Text className="login-kicker">CamoPanel MVP</Typography.Text>
        <Typography.Title level={2} style={{ marginTop: 8 }}>
          管理容器，不污染宿主机
        </Typography.Title>
        <Typography.Paragraph type="secondary">
          首版只保留一名管理员账号，但审批、应用商店、AI 和执行链路已经按未来扩展方式搭好。
        </Typography.Paragraph>
        {error ? <Alert style={{ marginBottom: 16 }} type="error" message={error} /> : null}
        <Form layout="vertical" onFinish={onFinish} initialValues={{ username: "admin", password: "admin123" }}>
          <Form.Item label="用户名" name="username" rules={[{ required: true, message: "请输入用户名" }]}>
            <Input prefix={<UserOutlined />} placeholder="admin" size="large" />
          </Form.Item>
          <Form.Item label="密码" name="password" rules={[{ required: true, message: "请输入密码" }]}>
            <Input.Password prefix={<LockOutlined />} placeholder="admin123" size="large" />
          </Form.Item>
          <Button type="primary" htmlType="submit" block size="large" loading={submitting}>
            登录面板
          </Button>
        </Form>
      </Card>
    </div>
  );
}
