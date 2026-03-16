import {
  AppstoreOutlined,
  CloudServerOutlined,
  ContainerOutlined,
  DownOutlined,
  FolderOpenOutlined,
  GlobalOutlined,
  LogoutOutlined,
  MessageOutlined,
  UserOutlined,
} from "@ant-design/icons";
import { useState, type ReactNode } from "react";
import { Avatar, Button, Dropdown, Layout, Menu, Typography, theme } from "antd";
import { Outlet, useLocation, useNavigate } from "react-router-dom";
import { ShellHeaderContext, ShellPageMeta } from "./ShellHeaderContext";
import { useAuthStore } from "../store/auth";

const items = [
  {
    key: "/app/dashboard",
    icon: <CloudServerOutlined />,
    label: "总览",
    headerTitle: "控制台",
    headerDescription: "先看宿主机健康，再看项目、站点和待处理异常。",
  },
  {
    key: "/app/store",
    icon: <AppstoreOutlined />,
    label: "应用商店",
    headerTitle: "应用商店",
    headerDescription: "从模板安装应用，并统一查看部署结果。",
  },
  {
    key: "/app/websites",
    icon: <GlobalOutlined />,
    label: "网站管理",
    headerTitle: "网站管理",
    headerDescription: "管理固定 OpenResty 容器里的站点入口。",
  },
  {
    key: "/app/files",
    icon: <FolderOpenOutlined />,
    label: "文件管理",
    headerTitle: "文件管理",
    headerDescription: "直接浏览和修改宿主机文件系统。",
  },
  {
    key: "/app/containers",
    icon: <ContainerOutlined />,
    label: "容器管理",
    headerTitle: "容器管理",
    headerDescription: "查看容器、镜像、编排、网络和 Docker 状态。",
  },
  {
    key: "/app/copilot",
    icon: <MessageOutlined />,
    label: "Copilot",
    headerTitle: "Camo Copilot",
    headerDescription: "只读分析，不直接执行写操作。",
  },
];

export function ShellLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, logout } = useAuthStore();
  const { token } = theme.useToken();
  const [headerContent, setHeaderContent] = useState<ReactNode | null>(null);
  const currentItem = items.find((item) => location.pathname.startsWith(item.key));
  const userMenu = {
    items: [
      {
        key: "logout",
        icon: <LogoutOutlined />,
        label: "退出",
      },
    ],
    onClick: ({ key }: { key: string }) => {
      if (key === "logout") {
        void logout();
      }
    },
  };

  return (
    <ShellHeaderContext.Provider value={setHeaderContent}>
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
            <div className="shell-header-main">
              {headerContent ?? (
                <ShellPageMeta
                  title={currentItem?.headerTitle ?? "控制台"}
                  description={currentItem?.headerDescription}
                />
              )}
            </div>
            <Dropdown menu={userMenu} trigger={["click"]} placement="bottomRight">
              <Button type="text" className="shell-user-trigger">
                <Avatar
                  size={28}
                  icon={<UserOutlined />}
                  style={{ background: token.colorBgContainer, color: token.colorText }}
                />
                <Typography.Text>{user?.username}</Typography.Text>
                <DownOutlined style={{ fontSize: 12, color: token.colorTextTertiary }} />
              </Button>
            </Dropdown>
          </Layout.Header>
          <Layout.Content className="shell-content">
            <Outlet />
          </Layout.Content>
        </Layout>
      </Layout>
    </ShellHeaderContext.Provider>
  );
}
