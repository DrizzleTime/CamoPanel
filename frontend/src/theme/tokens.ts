export interface AccentOption {
  id: string;
  name: string;
  hex: string;
  hover: string;
  light: string;
  text: string;
}

export const ACCENT_COLORS: AccentOption[] = [
  {
    id: "coral",
    name: "暖珊瑚",
    hex: "#d85c49",
    hover: "#c44f3d",
    light: "#fef2f0",
    text: "#b93826",
  },
  {
    id: "blue",
    name: "经典蓝",
    hex: "#2563eb",
    hover: "#1d4ed8",
    light: "#eff6ff",
    text: "#1d4ed8",
  },
  {
    id: "green",
    name: "翡翠绿",
    hex: "#10b981",
    hover: "#059669",
    light: "#ecfdf5",
    text: "#047857",
  },
  {
    id: "amber",
    name: "琥珀橙",
    hex: "#f59e0b",
    hover: "#d97706",
    light: "#fffbeb",
    text: "#b45309",
  },
  {
    id: "cyan",
    name: "水鸭青",
    hex: "#06b6d4",
    hover: "#0891b2",
    light: "#ecfeff",
    text: "#0e7490",
  },
  {
    id: "purple",
    name: "紫罗兰",
    hex: "#8b5cf6",
    hover: "#7c3aed",
    light: "#f5f3ff",
    text: "#6d28d9",
  },
  {
    id: "slate",
    name: "深灰黑",
    hex: "#4b5563",
    hover: "#374151",
    light: "#f1f5f9",
    text: "#1e293b",
  },
];

export function applyThemeAccent(colorId: string) {
  const found = ACCENT_COLORS.find((c) => c.id === colorId) || ACCENT_COLORS[0];
  localStorage.setItem("camo-accent", found.id);
  document.documentElement.style.setProperty("--accent", found.hex);
  document.documentElement.style.setProperty("--accent-hover", found.hover);
  document.documentElement.style.setProperty("--accent-light", found.light);
  document.documentElement.style.setProperty("--accent-text", found.text);
}

export function applyThemeBg(bg: "warm" | "white") {
  localStorage.setItem("camo-bg", bg);
  document.body.setAttribute("data-theme-bg", bg);
}

export type Density = "comfort" | "compact";

export function readTypography() {
  const size = Number(localStorage.getItem("camo-font-size") || 14);
  const height = Number(localStorage.getItem("camo-line-height") || 1.6);
  return {
    fontSize: Number.isFinite(size) ? Math.min(18, Math.max(12, size)) : 14,
    lineHeight: Number.isFinite(height)
      ? Math.min(1.8, Math.max(1.3, height))
      : 1.6,
    density: (localStorage.getItem("camo-density") === "compact"
      ? "compact"
      : "comfort") as Density,
  };
}

export function applyTypography(
  fontSize: number,
  lineHeight: number,
  density: Density,
) {
  localStorage.setItem("camo-font-size", String(fontSize));
  localStorage.setItem("camo-line-height", String(lineHeight));
  localStorage.setItem("camo-density", density);
  document.documentElement.style.setProperty("--font-size", `${fontSize}px`);
  document.documentElement.style.setProperty(
    "--line-height",
    String(lineHeight),
  );
  document.documentElement.style.setProperty(
    "--cell-padding",
    density === "compact" ? "11px 20px" : "16px 20px",
  );
  document.documentElement.style.setProperty(
    "--card-padding",
    density === "compact" ? "16px 20px" : "20px",
  );
  document.documentElement.dataset.density = density;
  document.documentElement.style.setProperty(
    "--control-height",
    density === "compact" ? "36px" : "40px",
  );
  document.documentElement.style.setProperty(
    "--control-height-sm",
    density === "compact" ? "30px" : "34px",
  );
  document.documentElement.style.setProperty(
    "--control-padding",
    density === "compact" ? "7px 12px" : "9px 12px",
  );
  document.documentElement.style.setProperty(
    "--table-row-height",
    density === "compact" ? "42px" : "52px",
  );
}
