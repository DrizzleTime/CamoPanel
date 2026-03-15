import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { ConfigProvider, theme } from "antd";
import App from "./App";
import "./styles.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <ConfigProvider
      theme={{
        algorithm: theme.defaultAlgorithm,
        token: {
          colorPrimary: "#141414",
          colorInfo: "#141414",
          colorBgLayout: "#f5f5f5",
          colorBgContainer: "#ffffff",
          colorBgElevated: "#ffffff",
          colorFillAlter: "#fafafa",
          colorFillSecondary: "#f5f5f5",
          colorBorder: "#d9d9d9",
          colorBorderSecondary: "#ececec",
          colorText: "#141414",
          colorTextSecondary: "#595959",
          colorTextTertiary: "#8c8c8c",
          borderRadius: 16,
          controlHeight: 40,
          boxShadowSecondary: "0 16px 40px rgba(0, 0, 0, 0.06)",
          boxShadowTertiary: "0 12px 30px rgba(0, 0, 0, 0.04)",
          fontFamily:
            '"Space Grotesk", "IBM Plex Sans SC", "Segoe UI", sans-serif',
        },
        components: {
          Layout: {
            bodyBg: "#f5f5f5",
            headerBg: "#ffffff",
            headerColor: "#141414",
            headerHeight: 64,
            headerPadding: "0 28px",
            siderBg: "#111111",
            lightSiderBg: "#111111",
            triggerBg: "#111111",
            triggerColor: "#fafafa",
            lightTriggerBg: "#111111",
            lightTriggerColor: "#fafafa",
          },
          Menu: {
            itemHeight: 44,
            itemBorderRadius: 14,
            subMenuItemBorderRadius: 14,
            itemMarginInline: 8,
            itemMarginBlock: 4,
            activeBarWidth: 0,
            activeBarHeight: 0,
            darkPopupBg: "#111111",
            darkItemBg: "#111111",
            darkSubMenuItemBg: "#111111",
            darkItemColor: "rgba(255, 255, 255, 0.72)",
            darkItemHoverColor: "#ffffff",
            darkItemHoverBg: "rgba(255, 255, 255, 0.08)",
            darkItemSelectedBg: "#fafafa",
            darkItemSelectedColor: "#111111",
            darkDangerItemSelectedBg: "#fafafa",
            darkDangerItemActiveBg: "rgba(255, 255, 255, 0.08)",
          },
          Card: {
            headerBg: "transparent",
            headerHeight: 56,
            headerPadding: 20,
            bodyPadding: 20,
            bodyPaddingSM: 16,
          },
          Table: {
            headerBg: "#fafafa",
            rowHoverBg: "#fafafa",
            borderColor: "#ececec",
            headerBorderRadius: 16,
            cellPaddingBlock: 14,
          },
          Tag: {
            defaultBg: "#f5f5f5",
            defaultColor: "#141414",
          },
        },
      }}
    >
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </ConfigProvider>
  </React.StrictMode>,
);
