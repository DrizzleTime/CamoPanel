import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": "http://localhost:8080",
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.indexOf("node_modules") === -1) {
            return undefined;
          }

          if (
            id.indexOf("/react/") !== -1 ||
            id.indexOf("/react-dom/") !== -1 ||
            id.indexOf("/scheduler/") !== -1
          ) {
            return "react";
          }

          if (id.indexOf("/react-router/") !== -1 || id.indexOf("/react-router-dom/") !== -1) {
            return "router";
          }

          if (id.indexOf("/zustand/") !== -1) {
            return "state";
          }

          if (id.indexOf("/@ant-design/icons/") !== -1) {
            return "antd-icons";
          }

          return undefined;
        },
      },
    },
  },
});
