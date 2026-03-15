import {
  AppstoreOutlined,
  CheckCircleOutlined,
  CloudServerOutlined,
  ContainerOutlined,
  FolderOpenOutlined,
  GlobalOutlined,
  LogoutOutlined,
  MessageOutlined,
  UserOutlined,
} from "@ant-design/icons";
import { Avatar, Button, Layout, Menu, Space, Typography, theme } from "antd";
import { Outlet, useLocation, useNavigate } from "react-router-dom";
import { useAuthStore } from "../store/auth";

const items = [
  { key: "/app/dashboard", icon: <CloudServerOutlined />, label: "总览" },
  { key: "/app/store", icon: <AppstoreOutlined />, label: "应用商店" },
  { key: "/app/websites", icon: <GlobalOutlined />, label: "网站管理" },
  { key: "/app/files", icon: <FolderOpenOutlined />, label: "文件管理" },
  { key: "/app/containers", icon: <ContainerOutlined />, label: "容器管理" },
  { key: "/app/approvals", icon: <CheckCircleOutlined />, label: "审批" },
  { key: "/app/copilot", icon: <MessageOutlined />, label: "Copilot" },
];

export function ShellLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, logout } = useAuthStore();
  const { token } = theme.useToken();
  const currentItem = items.find((item) => location.pathname.startsWith(item.key));

  return (
    <Layout className="shell-layout" style={{ background: token.colorBgLayout }}>
      <Layout.Sider
        width={248}
        breakpoint="lg"
        collapsedWidth="0"
        className="shell-sider"
        style={{ borderRight: `1px solid rgba(255, 255, 255, 0.08)` }}
      >
        <div
          className="shell-brand"
          style={{ borderBottom: `1px solid rgba(255, 255, 255, 0.08)` }}
        >
          <Typography.Title level={3} style={{ margin: 0, color: "#fafafa" }}>
            CamoPanel
          </Typography.Title>
          <Typography.Paragraph
            style={{ margin: "8px 0 0", color: "rgba(255, 255, 255, 0.56)" }}
          >
            一款 AI 原生的服务器面板
          </Typography.Paragraph>
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={items}
          onClick={({ key }) => navigate(key)}
          style={{ background: "transparent", borderInlineEnd: 0, padding: "12px 8px 0" }}
        />
      </Layout.Sider>
      <Layout className="shell-main" style={{ background: token.colorBgLayout }}>
        <Layout.Header
          className="shell-header"
          style={{
            background: token.colorBgContainer,
            borderBottom: `1px solid ${token.colorBorderSecondary}`,
          }}
        >
          <Space size={12} style={{ minWidth: 0 }}>
            <Typography.Text type="secondary">控制台</Typography.Text>
          </Space>
          <Space size="middle">
            <Button type="text">
              <Avatar
                size={28}
                icon={<UserOutlined />}
                style={{ background: token.colorBgContainer, color: token.colorText }}
              />
              <Typography.Text>{user?.username}</Typography.Text>
            </Button>
            <Button type="text" icon={<LogoutOutlined />} onClick={() => logout()}>
              退出
            </Button>
          </Space>
        </Layout.Header>
        <Layout.Content className="shell-content">
          <Outlet />
        </Layout.Content>
      </Layout>
    </Layout>
  );
}
