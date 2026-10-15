import React, { lazy, Suspense } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import "./reset.css";
import {
  applyThemeAccent,
  applyThemeBg,
  applyTypography,
  readTypography,
} from "./theme/tokens";

// 初始化保存的主题偏好
const savedAccent = localStorage.getItem("camo-accent") || "coral";
applyThemeAccent(savedAccent);

const savedBg = (localStorage.getItem("camo-bg") as "warm" | "white") || "warm";
applyThemeBg(savedBg);
const typography = readTypography();
applyTypography(typography.fontSize, typography.lineHeight, typography.density);

const Root =
  import.meta.env.DEV && window.location.pathname === "/__ui"
    ? lazy(() => import("./components/ui/ComponentPreview"))
    : App;

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <Suspense fallback={<div role="status">正在加载…</div>}>
      <Root />
    </Suspense>
  </React.StrictMode>,
);
