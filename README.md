# CamoPanel

AI 原生服务器运维面板的起步框架。目前支持单台本地 Docker 引擎，前后端分离开发，构建后由 Go 服务托管页面。推荐通过一键脚本安装二进制，由 systemd 管理宿主机上的面板服务；Docker 用于应用商店和 Nginx 网关。

## 一键安装（推荐）

需要 Linux、systemd 和 root 权限，支持 amd64 / arm64。服务器无需安装 Go、Bun 或 Node.js。

```bash
curl -fSL https://github.com/DrizzleTime/CamoPanel/releases/latest/download/install.sh -o camopanel-install.sh
sudo bash camopanel-install.sh
```

脚本会检查 Docker Engine 和 Compose，缺失时自动安装；下载并校验二进制安装包，配置管理员账号，注册开机自启服务。默认安装到 `/opt/camopanel`、监听 `8080`，完成后显示访问地址与随机密码。已有 Docker 安装会保留；已有 CamoPanel 配置和数据不会被覆盖。

```bash
# 无交互安装，可指定端口、账号、目录或版本
sudo bash camopanel-install.sh --yes --port 18080 --username admin --install-dir /opt/camopanel

sudo systemctl status camopanel
sudo journalctl -u camopanel -f
```

配置文件为 `/opt/camopanel/.env`，数据目录为 `/opt/camopanel/data`。修改配置后执行 `sudo systemctl restart camopanel`。请在云安全组和服务器防火墙中放行面板端口，网站服务按需放行 `80/443`。

**发布前提**：上述下载地址需要先发布包含安装包的 GitHub Release。仓库已提供标签触发的发布工作流，推送 `v0.1.0` 等版本标签即可构建 Linux 双架构安装包。完整说明见 [安装与发布](docs/installation.md)。

## 已实现

- **网站功能管理**：内置 Nginx 网关托管；支持静态站点（带在线 HTML 编辑）与反向代理（支持 WebSocket 升级协议与代理头）。
- **域名绑定与别名**：支持单主域名及多个别名域名一同绑定到同一虚拟主机。
- **SSL 证书一键申请**：支持 Let's Encrypt 官方 ACME HTTP-01 自动化一键申请、本地开发免验证自签名证书与自定义 PEM 证书上传，支持 HTTP 强制 301 重定向到 HTTPS。
- **Docker 应用商店**：基于标准 Docker Compose 模板一键安装流行开源应用（WordPress、MySQL、Redis、PostgreSQL、Nextcloud、Gitea、Uptime Kuma、Vaultwarden 等）。
- **严格受控安装规范**：所有应用强制约束在独立沙箱目录 `data/apps/<应用名>/`（包含 data/、config/、logs/），Compose 挂载路径严格限制在应用专属目录下，严禁逃逸至宿主机系统路径，保障数据安全与彻底干净卸载。
- **一键网站反代联动**：应用商店已安装应用可一键直达创建网站反向代理，无缝打通应用与外网域名。
- **内置 Nginx 网关**：自动管理 Nginx 虚拟主机配置文件与生命周期，支持配置重载与重启。
- **Docker 核心资源管理**：中文响应式工作台、容器/镜像/网络/数据卷管理、生命周期控制及实时日志查看。
- **主机状态**：工作台显示 CPU、内存、交换区、负载、磁盘、网卡和占用最高的进程。读取本机 `/proc`，读取失败时返回错误。
- **Docker 引擎**：通过 Unix socket 连接本地 Docker Engine。

## 本地开发

需要 Go 1.26+、Bun 1.4.3+、Node.js 22.12+（推荐 Node 24）及本地 Docker 引擎。依赖和脚本由根目录的 Bun workspace 统一管理；Vite 和 TypeScript 使用 Node 运行，以兼容 esbuild 子进程。

```bash
cp .env.example .env
# 编辑 .env，填写 CAMO_ADMIN_USERNAME 和 CAMO_ADMIN_PASSWORD
bun install
cd backend
go mod download
cd ..
bun run dev
```

根目录 `bun.lock` 统一锁定前端依赖，应纳入版本管理。可使用 `bun install --frozen-lockfile` 严格按照锁文件安装。

打开 **http://127.0.0.1:5173/login**，使用 `.env` 中配置的账号密码登录。后端连接本地 Docker 引擎，未登录不会加载工作台或资源数据。文件页直接操作真实服务器文件系统。

后端默认监听 `127.0.0.1:8080`。健康检查为 `GET /api/health`。两个服务也可以分开启动：

```bash
bun run dev:api
# 在另一个终端
bun run dev:web
```

如果修改后端端口，前端代理也需同步设置，例如 `API_PROXY_TARGET=http://127.0.0.1:8081 bun run dev:web`。

## 连接真实 Docker

在根目录 `.env` 修改以下配置，然后重启后端：

```dotenv
DOCKER_HOST=unix:///var/run/docker.sock
```

运行后端的用户须能访问该 socket。Rootless Docker 可使用 `unix:///run/user/1000/docker.sock`；Docker Desktop 使用实际的 Unix socket 路径。当前仅支持 Unix socket，不支持远程 TCP/TLS 和多主机。

Docker 连接失败会显示错误。

创建容器仅使用本地已有的带标签镜像，不自动拉取镜像。端口映射可选，当前只支持单个 TCP 映射，宿主机绑定 `127.0.0.1`。创建后手动启动。未提供环境变量、挂载、命令、Compose、容器终端和暂停/恢复配置。

删除容器采用非强制删除，不删除挂载的数据卷。镜像合计显示的是逻辑大小，各镜像可能共享镜像层，不能视为实际磁盘占用。

数据库控制台尚未接入真实数据库，相关操作返回 HTTP 501；防火墙规则管理尚未接入系统后端，界面显示不可用。应用商店安装与生命周期操作需要 Docker Compose 插件，执行失败会返回错误。

## 配置与访问

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `CAMO_ADDR` | `127.0.0.1:8080` | 后端监听地址 |
| `DOCKER_HOST` | `unix:///var/run/docker.sock` | Docker Unix socket |
| `CAMO_ADMIN_USERNAME` | `admin` | 登录账号；配置后只有此用户名可作为管理员登录 |
| `CAMO_ADMIN_PASSWORD` | 空 | 登录密码，8–72 字节；账号由环境变量管理，修改后需重启后端 |
| `CAMO_API_TOKEN` | 空 | 可选 API 访问令牌，用于脚本和外部客户端 |
| `CAMO_WEB_DIR` | 自动查找 `frontend/dist` | 构建页面目录 |
| `CAMO_DATA_DIR` | `./data` | 网站、SSL 证书与应用商店持久化根目录 |
| `CAMO_FILES_ROOT` | `/` | 服务器文件管理的访问根；可改为指定的服务器目录 |
| `CAMO_NGINX_PORT` | `80` | 内置 Nginx 网关 HTTP 监听端口 |
| `CAMO_NGINX_SSL_PORT` | `443` | 内置 Nginx 网关 HTTPS 监听端口 |
| `CAMO_ACME_EMAIL` | `admin@example.com` | Let's Encrypt 证书申请默认通知邮箱 |
| `CAMO_ACME_DIRECTORY` | Let's Encrypt | ACME 目录地址；留空时证书续签走 Let's Encrypt |

后端自动读取根目录 `.env`；进程环境变量优先。填写管理员账号密码后重启后端，浏览器使用独立登录页和 HttpOnly Cookie 会话访问面板。API 客户端可使用可选的 `CAMO_API_TOKEN`。未配置账号、API 令牌或存量管理员时，服务拒绝启动。

环境变量中的管理员凭据只在后端生成内存密码哈希，不写入 `auth.json`，不发送到浏览器。现有 `auth.json` 可继续保存两步验证元数据；配置环境账号后，旧账号密码不再用于登录。会话有效期为 24 小时，退出或重启服务后失效。

## 构建与运行

```bash
bun run test
bun run build
./bin/camopanel
```

构建后的 Go 服务同时托管页面和 API，打开 **http://127.0.0.1:8080**。前端样式采用 StyleX 编译提取，Vite 配置依据 [StyleX 官方安装文档](https://stylexjs.com/docs/learn/installation/vite/)。后端直接使用 [Docker Engine HTTP API](https://docs.docker.com/reference/api/engine/)，读取守护进程版本后访问对应版本的接口。

## 目录与扩展

```text
backend/
  cmd/server/          启动、配置与优雅退出
  internal/config/     环境配置
  internal/httpapi/    Gin 路由、校验与鉴权
  internal/docker/     Service 接口与 Docker 引擎实现
frontend/
  src/components/     表单、日志弹窗、运维摘要等组件
  src/lib/api.ts       类型与 API 客户端
  src/styles.ts        StyleX 样式与响应式布局
  src/App.tsx          工作台与资源页面
docs/api.md            API 合约
```

后续 AI 层可以通过 `docker.Service` 接入只读诊断、工具调用和任务计划；写操作建议增加用户确认、持久化审计和权限检查后再开放给模型。当前无需数据库、模型供应商或 API Key。

## 本次验证

已使用 Bun 完成根目录 workspace 的锁文件生成和离线 `--frozen-lockfile` 安装校验，`bun run test` 与 `bun run build` 均已通过。构建包含 TypeScript 检查、Vite / StyleX 前端编译和 Go 后端编译。后端测试覆盖容器生命周期、参数校验、鉴权、请求来源、Docker 请求格式、日志解码和静态页面路由。

当前环境下强制使用 Bun 运行 Vite 会使 esbuild 子进程失败，因此 Vite 和 TypeScript 使用 Node 执行，Bun 负责依赖管理及脚本调度。真实 Docker 联调尚未完成。安装脚本测试通过模拟下载、Docker 和 systemd 检查成功、失败与配置保留流程，不会修改开发机服务或安装 Docker。


## 服务器文件管理

文件页直接操作后端所在服务器的文件系统，与 Docker 连接状态无关。默认从 `/` 浏览，访问和修改受运行后端的系统账号权限约束。`CAMO_FILES_ROOT` 可指定可访问的服务器目录；网站、应用和备份目录仍提供独立快捷入口。脚本安装的面板服务以 root 运行，以支持宿主机文件与系统管理。

支持绝对路径导航、目录树、搜索与排序、多选、复制/剪切/粘贴、重命名、权限修改、批量删除、ZIP 压缩和解压、文本编辑及多文件上传。上传保留二进制内容并显示进度，单文件最大 256 MB；UTF-8 文本编辑最大 4 MB。ZIP 解压最多 10000 项、总大小 1 GB，不覆盖现有文件。只读账号可浏览和下载，写入操作由后端拒绝。
