import {
  LockOutlined,
  UserOutlined,
  SafetyCertificateOutlined,
  RocketOutlined,
  RobotOutlined,
} from "@ant-design/icons";
import { Alert, Button, Form, Input, Typography, message, Row, Col, Flex, theme } from "antd";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { ApiError } from "../../shared/api/client";
import { useAuthStore } from "./store";

const { Title, Text, Paragraph } = Typography;
const { useToken } = theme;

export function LoginPage() {
  const navigate = useNavigate();
  const login = useAuthStore((state) => state.login);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string>();
  const { token } = useToken();

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
    <Row style={{ minHeight: "100vh", backgroundColor: token.colorBgContainer }}>
      {/* 左侧品牌与介绍 */}
      <Col
        xs={0}
        sm={0}
        md={12}
        lg={14}
        style={{
          background: "linear-gradient(135deg, #0f172a 0%, #1e1b4b 100%)",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          padding: "40px",
          position: "relative",
          overflow: "hidden",
        }}
      >
        <Flex vertical gap="large" style={{ maxWidth: 500, position: "relative", zIndex: 10 }}>
          <Flex align="center" gap="middle" style={{ marginBottom: 16 }}>
            <SafetyCertificateOutlined style={{ fontSize: 36, color: token.colorPrimary }} />
            <Text style={{ fontSize: 28, fontWeight: "bold", color: "#fff", letterSpacing: 0.5 }}>
              CamoPanel
            </Text>
          </Flex>
          
          <Title level={1} style={{ color: "#fff", margin: 0, fontSize: 42, lineHeight: 1.2 }}>
            现代化服务器面板
          </Title>
          
          <Paragraph style={{ color: "rgba(255, 255, 255, 0.7)", fontSize: 16, lineHeight: 1.6, marginBottom: 32 }}>
            专注于容器化部署与运维，提供干净、安全、高效的宿主机管理体验。应用环境隔离，告别系统污染。
          </Paragraph>

          <Flex vertical gap="large" style={{ width: "100%" }}>
            <Flex align="flex-start" gap="middle">
              <Flex
                align="center"
                justify="center"
                style={{
                  width: 48,
                  height: 48,
                  background: "rgba(255, 255, 255, 0.1)",
                  borderRadius: token.borderRadiusLG,
                  fontSize: 24,
                  color: "#fff",
                  flexShrink: 0,
                }}
              >
                <RocketOutlined />
              </Flex>
              <Flex vertical gap={4} style={{ marginTop: 2 }}>
                <Text style={{ color: "#fff", fontSize: 16, fontWeight: 500 }}>极速容器部署</Text>
                <Text style={{ color: "rgba(255, 255, 255, 0.6)", fontSize: 14 }}>
                  一键拉起 Docker 容器与各类环境应用
                </Text>
              </Flex>
            </Flex>

            <Flex align="flex-start" gap="middle">
              <Flex
                align="center"
                justify="center"
                style={{
                  width: 48,
                  height: 48,
                  background: "rgba(255, 255, 255, 0.1)",
                  borderRadius: token.borderRadiusLG,
                  fontSize: 24,
                  color: "#fff",
                  flexShrink: 0,
                }}
              >
                <SafetyCertificateOutlined />
              </Flex>
              <Flex vertical gap={4} style={{ marginTop: 2 }}>
                <Text style={{ color: "#fff", fontSize: 16, fontWeight: 500 }}>纯净环境隔离</Text>
                <Text style={{ color: "rgba(255, 255, 255, 0.6)", fontSize: 14 }}>
                  所有服务容器化运行，确保宿主机极致干净
                </Text>
              </Flex>
            </Flex>

            <Flex align="flex-start" gap="middle">
              <Flex
                align="center"
                justify="center"
                style={{
                  width: 48,
                  height: 48,
                  background: "rgba(255, 255, 255, 0.1)",
                  borderRadius: token.borderRadiusLG,
                  fontSize: 24,
                  color: "#fff",
                  flexShrink: 0,
                }}
              >
                <RobotOutlined />
              </Flex>
              <Flex vertical gap={4} style={{ marginTop: 2 }}>
                <Text style={{ color: "#fff", fontSize: 16, fontWeight: 500 }}>AI 智能运维</Text>
                <Text style={{ color: "rgba(255, 255, 255, 0.6)", fontSize: 14 }}>
                  深度分析系统日志，提供自动化排错与诊断建议
                </Text>
              </Flex>
            </Flex>
          </Flex>
        </Flex>

        {/* 背景装饰图形 */}
        <div
          style={{
            position: "absolute",
            top: "-10%",
            right: "-10%",
            width: 500,
            height: 500,
            background: "radial-gradient(circle, rgba(99,102,241,0.15) 0%, rgba(99,102,241,0) 70%)",
            borderRadius: "50%",
            zIndex: 1,
          }}
        />
        <div
          style={{
            position: "absolute",
            bottom: "-10%",
            left: "-10%",
            width: 600,
            height: 600,
            background: "radial-gradient(circle, rgba(236,72,153,0.1) 0%, rgba(236,72,153,0) 70%)",
            borderRadius: "50%",
            zIndex: 1,
          }}
        />
      </Col>

      {/* 右侧登录表单 */}
      <Col
        xs={24}
        sm={24}
        md={12}
        lg={10}
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          padding: "40px",
        }}
      >
        <Flex vertical gap="large" style={{ width: "100%", maxWidth: 380 }}>
          <Flex vertical gap={8} align="center" style={{ width: "100%", marginBottom: 24 }}>
            <Title level={2} style={{ margin: 0 }}>欢迎回来</Title>
            <Text type="secondary">登录 CamoPanel 管理控制台</Text>
          </Flex>

          {error ? <Alert type="error" message={error} showIcon /> : null}

          <Form
            layout="vertical"
            onFinish={onFinish}
            initialValues={{ username: "admin" }}
            size="large"
            style={{ width: "100%" }}
          >
            <Form.Item
              label="用户名"
              name="username"
              rules={[{ required: true, message: "请输入用户名" }]}
            >
              <Input prefix={<UserOutlined style={{ color: token.colorTextTertiary }} />} placeholder="admin" />
            </Form.Item>
            
            <Form.Item
              label="密码"
              name="password"
              rules={[{ required: true, message: "请输入密码" }]}
            >
              <Input.Password prefix={<LockOutlined style={{ color: token.colorTextTertiary }} />} placeholder="请输入密码" />
            </Form.Item>

            <Form.Item style={{ marginTop: 32, marginBottom: 0 }}>
              <Button type="primary" htmlType="submit" block loading={submitting} style={{ height: 48, fontSize: 16 }}>
                登录面板
              </Button>
            </Form.Item>
          </Form>
          
          <Flex align="center" justify="center" style={{ width: "100%", marginTop: 24 }}>
            <Text type="secondary">CamoPanel &copy; {new Date().getFullYear()}</Text>
          </Flex>
        </Flex>
      </Col>
    </Row>
  );
}
