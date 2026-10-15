# API v1

开发时经 Vite 同源代理访问，生产时由 Gin 同源提供。浏览器使用账号密码登录后的 HttpOnly Cookie 访问受保护接口；API 客户端也可使用 `Authorization: Bearer <token>`。登录凭据由 `CAMO_ADMIN_USERNAME` 与 `CAMO_ADMIN_PASSWORD` 配置。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/health` | 进程健康，不检查 Docker 连通性，无需令牌 |
| GET | `/api/v1/docker/overview` | 模式、引擎、容器、镜像、网络、数据卷和同步时间 |
| GET | `/api/v1/host/overview` | 主机 CPU、内存、交换区、负载、磁盘、网卡和近一小时采样。`mode` 固定为 `host` |
| GET | `/api/v1/host/processes` | 占用最高的进程。`limit` 默认 10，范围 1–50；`sort` 为 `cpu` 或 `mem` |
| POST | `/api/v1/docker/containers` | 创建容器，返回 201 和 `{ "id": "..." }` |
| POST | `/api/v1/docker/containers/:id/start` | 启动 |
| POST | `/api/v1/docker/containers/:id/stop` | 停止，给予 10 秒退出时间 |
| POST | `/api/v1/docker/containers/:id/restart` | 重启，给予 10 秒退出时间 |
| POST | `/api/v1/docker/containers/:id/remove` | 非强制删除，保留数据卷 |
| GET | `/api/v1/docker/containers/:id/logs` | 最近 200 行日志，响应 `{ "logs": "..." }` |
| GET | `/api/v1/store/apps` | 应用商店列表与模板，包含分类与安装状态 |
| GET | `/api/v1/store/apps/:id` | 应用详情与标准 compose 模板 |
| GET | `/api/v1/store/installed` | 已安装应用列表，附带端口、受控目录与运行状态 |
| GET | `/api/v1/store/installed/:id` | 已安装应用详情 |
| POST | `/api/v1/store/installed` | 安装应用，自动在 `apps/<app_id>` 下创建受控数据、配置与日志目录 |
| POST | `/api/v1/store/installed/:id/action` | 控制应用生命周期：`start`、`stop`、`restart`、`uninstall` |
| GET | `/api/v1/store/installed/:id/compose` | 获取已安装应用的 docker-compose.yml |
| PUT | `/api/v1/store/installed/:id/compose` | 更新并校验应用的 docker-compose.yml 路径约束 |
| GET | `/api/v1/store/installed/:id/logs` | 查看已安装应用容器日志 |
| GET | `/api/v1/sites` | 网站列表（静态站点与反向代理，包含绑定域名与 SSL 状态） |
| POST | `/api/v1/sites` | 创建网站，绑定主域名与别名，自动配置 Nginx |
| GET | `/api/v1/sites/:id` | 网站详情 |
| PUT | `/api/v1/sites/:id` | 修改网站配置 |
| DELETE | `/api/v1/sites/:id` | 删除网站并移除 Nginx 虚拟主机配置 |
| POST | `/api/v1/sites/:id/toggle` | 启用 / 停用网站 |
| POST | `/api/v1/sites/:id/ssl/apply` | SSL 一键申请（支持 Let's Encrypt ACME、自签名测试证书和自定义证书） |
| GET | `/api/v1/sites/:id/ssl` | 获取网站 SSL 证书详情与剩余天数 |
| GET | `/api/v1/sites/:id/config` | 预览当前生成的 Nginx 完整配置文件 |
| GET | `/api/v1/sites/:id/files/:filename` | 读取静态站点网页文件（如 `index.html`） |
| PUT | `/api/v1/sites/:id/files/:filename` | 在线保存静态网页内容 |
| GET | `/api/v1/nginx/status` | 内置 Nginx 网关状态（容器运行状态、端口、配置文件路径、活动站点数） |
| POST | `/api/v1/nginx/action` | 内置 Nginx 运维操作：`start`、`stop`、`restart`、`reload` |

应用商店安装路径约束规范：
- 所有应用安装目录强制约束在 `<CAMO_DATA_DIR>/apps/<app-name>/` 下。
- 标准受控子目录：`data/`（持久化数据）、`config/`（配置文件）、`logs/`（日志文件）。
- Compose 模板中的宿主机挂载仅允许相对路径 `./data`、`./config`、`./logs` 或受控命名卷，严格禁止逃逸至宿主机系统路径（如 `/etc`、`/var/run`、`/root`）。

网站配置规范：
- 内置 Nginx 托管：支持静态站点 (`root /var/www/sites/<site-id>`) 与反向代理 (`proxy_pass <target>` 带 WebSocket 升级头)。
- 域名绑定：支持单主域名及多个别名域名一同绑定到同一虚拟主机。
- SSL 一键申请：Let's Encrypt ACME HTTP-01 验证、离线开发自签名证书及用户自定义 PEM 证书上传。

创建请求：

```json
{
  "name": "my-web",
  "image": "nginx:alpine",
  "host_port": 8088,
  "container_port": 80
}
```

端口必须同时填写，范围 1–65535；不映射时均为 0 或省略。名称 2–63 字符，以字母或数字开头，其余字符允许字母、数字、下划线、点和短横线。镜像须已存在于本地。

概览的集合字段使用 Docker 字段名称（例如 `Id`、`Names`、`RepoTags`），顶层为 `mode`、`engine`、`containers`、`images`、`networks`、`volumes` 和 `generated_at`。`mode` 固定为 `docker`，同步时间为 UTC RFC 3339。

错误格式为 `{ "error": { "message": "..." } }`。常见状态码：400 参数错误、401 令牌错误、403 请求来源错误、404 资源不存在、409 操作冲突、503 Docker 不可用。

日志可以单次读取，也可以用 `GET /api/v1/docker/containers/:id/logs/stream` 做有上限的 SSE。响应大小限制 8 MiB，请求体限制 1 MiB。强制删除是单独的已确认动作，默认不删除数据卷。终端只记录会话起止和账号，不执行任意命令。

配置 `CAMO_TLS_CERT` 与 `CAMO_TLS_KEY` 后，面板以 HTTPS 提供服务；否则保持 HTTP。设置里会提示：公网暴露面板需要 HTTPS。未配置环境管理员、存量管理员或 API 令牌时，服务拒绝启动。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/auth/status` | 公开的登录配置状态与 TOTP 开关，不返回账号或密码 |
| POST | `/api/v1/auth/bootstrap` | 存量账号初始化接口；使用环境账号时返回 409 |
| POST | `/api/v1/auth/login` | 登录，写入 HttpOnly `camo_session`。启用 TOTP 时密码之外还要验证码；Bearer 令牌不要求 TOTP |
| POST | `/api/v1/auth/logout` | 退出并清除会话 |
| POST | `/api/v1/auth/password` | 存量账号修改密码；环境账号需修改 `.env` 并重启，接口返回 409 |
| POST | `/api/v1/auth/viewer` | 存量只读账号配置；环境账号模式下返回 409 |
| GET | `/api/v1/auth/session` | 当前会话的账号和角色 |
| POST | `/api/v1/auth/totp/begin` | 生成 TOTP 密钥 |
| POST | `/api/v1/auth/totp/enable` | 用验证码启用 TOTP |
| GET | `/api/v1/settings/exposure` | 是否正在使用 HTTPS，以及公网暴露提示 |
| GET | `/api/v1/certificates` | 证书库。列表含域名、到期时间和已绑定网站 |
| POST | `/api/v1/certificates/self-signed` | 自签证书 |
| POST | `/api/v1/certificates/upload` | 上传 PEM，私钥留在证书库 |
| POST | `/api/v1/certificates/issued` | 登记已签发证书 |
| POST | `/api/v1/certificates/:id/bind` | 把同一张证书绑到多个网站，不复制私钥 |
| GET | `/api/v1/certificates/dns` | DNS 配置公开字段，只报告 `token_set`，不回显密钥 |
| PUT | `/api/v1/certificates/dns` | 保存一个 DNS 服务商的 API 地址、zone 和令牌 |
| POST | `/api/v1/certificates/dns/manual` | 记录手动 TXT |
| POST | `/api/v1/certificates/dns/issue` | 走 DNS-01 |
| GET | `/api/v1/files/roots` | 返回服务器根、快捷目录、主机名、只读权限与大小限制 |
| GET | `/api/v1/files` | `root=server` 使用真实服务器绝对路径；应用根使用相对路径。返回时间、权限、所有者与链接信息 |
| GET | `/api/v1/files/content` | 读取最多 4 MB 的 UTF-8 文本；应用根保留路径和私钥限制 |
| PUT | `/api/v1/files` | 保存文本 |
| POST | `/api/v1/files/upload` | multipart 二进制上传（最大 256 MB），字段 `root`、`path`、`file`、`overwrite` |
| POST | `/api/v1/files/create` | 排他创建文件或目录，字段 `root`、`path`、`directory` |
| POST | `/api/v1/files/transfer` | 复制/移动多项，字段 `root`、`paths`、`dest_root`、`dest`、`move`；返回逐项结果 |
| POST | `/api/v1/files/delete` | 批量删除，字段 `root`、`paths`；返回逐项结果 |
| POST | `/api/v1/files/permissions` | 修改单项权限，字段 `root`、`path`、`mode`（0000–0777） |
| POST | `/api/v1/files/mkdir` | 建目录 |
| POST | `/api/v1/files/rename` | 重命名 |
| DELETE | `/api/v1/files` | 删除 |
| POST | `/api/v1/files/zip` | ZIP 压缩，支持 `paths` 多选和 `dest` 输出路径 |
| POST | `/api/v1/files/unzip` | 解压 |
| GET | `/api/v1/files/download-link` | 生成 2 分钟内有效的一次性下载链接，支持令牌认证和浏览器原生流式下载 |
| GET | `/api/v1/files/download` | 二进制流式下载，支持 Range 与原文件名 |
| GET | `/api/v1/databases` | MySQL、PostgreSQL、Redis 是否可用。未安装时 `store_path` 指向应用商店 |
| POST | `/api/v1/databases/:kind/databases` | 创建数据库 |
| POST | `/api/v1/databases/:kind/users` | 创建用户。审计不记录密码 |
| POST | `/api/v1/databases/:kind/export` | 导出 SQL |
| POST | `/api/v1/databases/:kind/import` | 导入 SQL |
| POST | `/api/v1/databases/:kind/exec` | 执行 SQL。审计只记语句 |
| GET | `/api/v1/databases/redis/keys` | 按前缀扫描，默认最多 50 个 |
| DELETE | `/api/v1/databases/redis/keys` | 删除一个键 |
| GET | `/api/v1/backups` | 本地备份 |
| POST | `/api/v1/backups/sites/:id` | 备份网站 |
| POST | `/api/v1/backups/:id/restore` | 恢复。`confirm` 必须为 true，且不会写出数据根目录 |
| POST | `/api/v1/backups/:id/publish` | 登记到 S3 兼容存储。密钥不写入响应 |
| GET | `/api/v1/schedules` | 计划任务，重启后仍在 |
| POST | `/api/v1/schedules` | 创建。五段 cron，类型含网站/应用/数据库/目录备份、证书续签、站点日志轮转、URL 拉取和脚本 |
| POST | `/api/v1/schedules/:id/run` | 立即运行 |
| POST | `/api/v1/schedules/:id/enabled` | 启用或停用 |
| POST | `/api/v1/schedules/scripts` | 把脚本写入面板脚本目录 |
| POST | `/api/v1/schedules/reload` | 重新读取任务。错过的一次性任务只标记，不补跑 |
| GET | `/api/v1/docker/containers/:id` | 容器详情 |
| GET | `/api/v1/docker/containers/:id/stats` | 该容器的 CPU 和内存 |
| GET | `/api/v1/docker/containers/:id/logs/stream` | 有上限的日志流 |
| POST | `/api/v1/docker/containers/:id/pause` | 暂停 |
| POST | `/api/v1/docker/containers/:id/unpause` | 恢复 |
| POST | `/api/v1/docker/containers/:id/force-remove` | 确认后强制删除，默认 `volumes_deleted` 为 false |
| POST | `/api/v1/docker/images/pull` | 拉取镜像并返回进度。创建容器时只有显式要求才拉取 |
| DELETE | `/api/v1/docker/images/:id` | 删除镜像 |
| POST | `/api/v1/docker/images/prune` | 清理悬空镜像 |
| GET | `/api/v1/firewall` | 防火墙状态。没有 ufw、firewalld 或 iptables 时说明无法管理，不安装 |
| POST | `/api/v1/firewall/rules` | 添加规则。关掉面板端口的规则在确认前会被拒绝 |
| GET | `/api/v1/alerts` | 告警规则 |
| POST | `/api/v1/alerts` | 磁盘、证书到期、容器退出或网站宕机。通知 SMTP 或通用 webhook |
| POST | `/api/v1/alerts/evaluate` | 用一份样本评估并发送 |
| POST | `/api/v1/terminals/host` | 记录主机终端会话起止 |
| POST | `/api/v1/terminals/containers/:id` | 记录容器终端会话起止 |
| GET | `/api/v1/ai/settings` | 模型供应商。只报告密钥是否已设置 |
| PUT | `/api/v1/ai/settings` | 保存接口地址、模型、密钥和附加请求头。默认地址 `https://api.openai.com/v1` |
| POST | `/api/v1/ai/chat` | 通过 OpenAI Responses API 对话。写入工具要在对话里确认后才调用现有接口 |

创建容器仍接受原来的单端口字段。也可以传 `ports`、`env`、`command`、`restart_policy` 和 `binds`。数据目录之外的挂载需要 `confirm_host_mount`。网站类型为 `static`、`proxy` 或 `runtime`；运行环境为 `php`、`node`、`java`、`go`、`python`。自定义 Nginx 片段不能离开 server 块，也不能引用任意宿主机文件。

每次会改数据的 `/api/v1` 请求都会追加审计：时间、账号、方法、路径和结果。审计不保存密码、私钥或令牌。只读账号可以看概览、网站、容器和日志；创建、删除、安装、证书、Compose、上传、防火墙和终端返回 403。

数据库控制台尚未接入真实数据库：查询返回 `installed: false`，SQL、导入导出及 Redis 键操作返回 HTTP 501。防火墙规则管理尚未接入系统后端，状态返回 `available: false`，写入请求返回错误。
