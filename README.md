# CamoPanel

CamoPanel 是一个面向 Linux 单机的容器管理面板。  
当前仓库实现的是 MVP：应用商店部署、统一审批链路、基础项目管理、OpenResty 容器管理、宿主机概览和只读 AI Copilot。

## 当前实现

- `server`
  - Go 控制面，负责鉴权、应用模板扫描、OpenResty 容器管理、审批、Docker 执行、主机信息和 Copilot API。
- `web`
  - React 19 + Bun SPA，负责登录、应用商店、OpenResty、项目管理、审批中心和 Copilot UI。
- `templates`
  - 本地应用模板仓库。MVP 内置 `openresty`、`postgres` 和 `wordpress`。

## 运行前提

- Linux
- Docker Engine + `docker compose`
- 一个固定的 OpenResty 容器（可通过应用商店 `openresty` 模板部署，仅创建站点时必需）
- Go 1.25+
- Bun 1.3+

## 本地开发

### 推荐命令

```bash
make dev
```

这会并行启动后端和前端开发服务。  
如果只想单独调某一侧：

```bash
make dev-server
make dev-web
```

### 1. 启动后端

```bash
cd server
env GOPROXY=https://goproxy.cn,direct go run ./cmd/camopanel
```

默认配置：

- HTTP 地址：`:8080`
- 数据目录：`./data`
- 模板目录：优先自动探测 `./templates`，否则回退到 `../templates`
- OpenResty 容器名：`camopanel-openresty`
- OpenResty 数据目录：`${CAMO_DATA_DIR}/openresty`
- OpenResty 约定挂载：
  - `${CAMO_DATA_DIR}/openresty/conf.d` -> `/etc/nginx/conf.d`
  - `${CAMO_DATA_DIR}/openresty/conf.d` -> `/etc/openresty/conf.d`（兼容旧路径）
  - `${CAMO_DATA_DIR}/openresty/www` -> `/var/www/openresty`
- 初始管理员：`admin / admin123`

创建站点前，面板会检查这个固定容器是否存在、是否运行中、挂载目录是否符合约定。  
如果不满足条件，站点创建会直接失败，不会尝试自动修复容器。

最小示例：

```bash
mkdir -p ./data/openresty/conf.d ./data/openresty/www

docker run -d \
  --name camopanel-openresty \
  -p 80:80 \
  -v "$(pwd)/data/openresty/conf.d:/etc/nginx/conf.d" \
  -v "$(pwd)/data/openresty/conf.d:/etc/openresty/conf.d" \
  -v "$(pwd)/data/openresty/www:/var/www/openresty" \
  openresty/openresty:alpine
```

### 2. 启动前端

```bash
cd web
bun install
bun run dev
```

前端开发服务器会把 `/api` 代理到 `http://localhost:8080`。

## 关键环境变量

- `CAMO_HTTP_ADDR`
- `CAMO_DATA_DIR`
- `CAMO_TEMPLATES_DIR`
- `CAMO_SESSION_SECRET`
- `CAMO_ADMIN_USERNAME`
- `CAMO_ADMIN_PASSWORD`
- `CAMO_OPENRESTY_CONTAINER`
- `CAMO_AI_BASE_URL`
- `CAMO_AI_MODEL`
- `CAMO_AI_API_KEY`

## 构建

```bash
make build
```

这会执行：

1. 前端构建
2. 把 `web/dist` 拷贝到 `server/internal/webui/dist`
3. 后端编译

## 发布

仓库内置 GitHub Actions 工作流 `.github/workflows/build-release.yml`。

- `workflow_dispatch` 可手动触发
- 推送 `v*` tag 时会构建 Linux `amd64`、`arm64` 两个平台
- 每个平台都会产出一个 `.tar.gz` 发布包和对应的 `sha256` 文件

发布包内容：

- `camopanel`
- `templates/`
- `deploy/camopanel.service`
- `deploy/install.sh`
- `deploy/camopanel.env.example`

## 部署

解压发布包后可直接执行：

```bash
tar -xzf camopanel_linux_amd64.tar.gz
cd camopanel_linux_amd64
sudo ./deploy/install.sh
```

也可以让脚本直接处理本地包或远程包：

```bash
sudo ./deploy/install.sh --package ./camopanel_linux_amd64.tar.gz
sudo ./deploy/install.sh --package https://example.com/camopanel_linux_amd64.tar.gz
```

默认安装位置：

- 二进制：`/usr/local/bin/camopanel`
- 数据目录：`/var/lib/camopanel`
- 环境文件：`/etc/camopanel/camopanel.env`
- service：`/etc/systemd/system/camopanel.service`

## systemd

`deploy/camopanel.service` 提供了一个最小可用的 `systemd` 单元文件。  
默认读取 `/etc/camopanel/camopanel.env` 作为环境变量文件，运行目录是 `/var/lib/camopanel`。

## 测试

```bash
make test-server
```

当前包含：

- 模板校验与渲染单测
- 审批流转与 AI 提案转审批单测
- Docker 集成测试骨架（`integration` build tag，需要本机可用 Docker）
