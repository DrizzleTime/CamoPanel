import { test as base, expect } from "@playwright/test";
import type { FileEntry } from "../src/lib/api";

const entry = (name: string, dir = true): FileEntry => ({
  name,
  path: `/${name}`,
  dir,
  size: dir ? 0 : 2048,
  modified: "2026-10-15T06:20:00Z",
  permissions: dir ? "0755" : "0644",
  mode: dir ? "drwxr-xr-x" : "-rw-r--r--",
  owner: "root",
  group: "root",
  type: dir ? "directory" : "file",
  symlink: false,
});

export const test = base.extend<{ mockedApi: void }>({
  mockedApi: [
    async ({ page }, use) => {
      const errors: string[] = [];
      page.on("pageerror", (error) => errors.push(error.message));
      let entries = [
        ...[
          "bin",
          "boot",
          "dev",
          "etc",
          "home",
          "lib",
          "media",
          "mnt",
          "opt",
          "proc",
          "root",
          "run",
          "srv",
          "tmp",
          "usr",
          "var",
        ].map((name) => entry(name)),
        ...["README.md", "access.log", "nginx.conf", ".hidden"].map((name) =>
          entry(name, false),
        ),
      ];
      let rules = [
        { port: 22, policy: "allow" },
        { port: 80, policy: "allow" },
        { port: 443, policy: "allow" },
      ];
      await page.route("**/api/**", async (route) => {
        const url = new URL(route.request().url());
        const path = url.pathname.replace("/api/v1", "");
        const method = route.request().method();
        let body: unknown = [];
        if (path === "/auth/status")
          body = {
            configured: true,
            source: "environment",
            totp_enabled: false,
          };
        else if (path === "/auth/session")
          body = { actor: "admin", role: "admin" };
        else if (path === "/docker/overview")
          body = {
            mode: "demo",
            generated_at: "",
            engine: {
              Name: "preview",
              ServerVersion: "27.0",
              OperatingSystem: "Linux",
              NCPU: 4,
              MemTotal: 8589934592,
            },
            containers: [],
            images: [],
            networks: [],
            volumes: [],
          };
        else if (path === "/host/overview" || path === "/nginx/status")
          body = {};
        else if (path === "/host/processes") body = { processes: [] };
        else if (path === "/files/roots")
          body = {
            roots: [
              { name: "server", label: "服务器根目录", path: "/" },
              { name: "sites", label: "网站目录", path: "/srv/sites" },
              { name: "apps", label: "应用数据", path: "/srv/apps" },
              { name: "backups", label: "备份目录", path: "/srv/backups" },
            ],
            server_root: "/",
            home: "/root",
            hostname: "camo-server",
            read_only: false,
            upload_limit: 268435456,
            text_limit: 4194304,
          };
        else if (path === "/files" && method === "GET")
          body = {
            entries: url.searchParams.get("path") === "/" ? entries : [],
            path: url.searchParams.get("path") || "/",
            parent: "/",
            root: "server",
          };
        else if (path === "/files/create") {
          const request = route.request().postDataJSON();
          entries = [
            ...entries,
            entry(request.path.slice(1), request.directory),
          ];
          body = { message: "已创建" };
        } else if (path === "/files/rename") {
          const request = route.request().postDataJSON();
          entries = entries.map((item) =>
            item.path === request.from
              ? { ...item, path: request.to, name: request.to.split("/").pop() }
              : item,
          );
          body = { message: "已重命名" };
        } else if (path === "/files/delete") {
          const request = route.request().postDataJSON();
          entries = entries.filter(
            (item) => !request.paths.includes(item.path),
          );
          body = {
            succeeded: request.paths.length,
            results: request.paths.map((path: string) => ({ path })),
          };
        } else if (path === "/files/content")
          body = { content: "# Deployment notes\n\nReady to deploy.\n" };
        else if (path === "/files" && method === "PUT")
          body = { message: "已保存" };
        else if (path === "/firewall/rules") {
          const request = route.request().postDataJSON();
          rules = [...rules, { port: request.port, policy: request.policy }];
          body = { available: true, backend: "ufw", panel_port: 8080, rules };
        } else if (path === "/firewall")
          body = { available: true, backend: "ufw", panel_port: 8080, rules };
        await route.fulfill({ json: body });
      });
      await use();
      expect(errors, "页面应无未捕获的 JavaScript 错误").toEqual([]);
    },
    { auto: true },
  ],
});
export { expect };
