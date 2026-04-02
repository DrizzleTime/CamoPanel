<div align="center">

<br>

# CamoPanel

**轻量级 Linux 服务器管理面板**

<br>

一款现代化的自托管容器管理面板，专为单机 Linux 环境设计。
<br>
应用部署、站点管理、数据库运维、容器控制 — 一个界面全部搞定。

<br>

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
[![React](https://img.shields.io/badge/React-19-61DAFB?style=flat&logo=react&logoColor=white)](https://react.dev)
[![Bun](https://img.shields.io/badge/Bun-1.3+-F9F1E1?style=flat&logo=bun&logoColor=black)](https://bun.sh)
[![Platform](https://img.shields.io/badge/Platform-Linux-FCC624?style=flat&logo=linux&logoColor=black)](https://kernel.org)

<br>

[English](README.md)

<br>

</div>

## 功能特性

<table>
<tr>
<td width="50%">

**应用商店**
<br>内置 OpenResty、PostgreSQL、MySQL、Redis、WordPress、PHP-FPM 等模板，一键部署。

</td>
<td width="50%">

**站点管理**
<br>创建和管理 OpenResty 站点，自动配置反向代理。

</td>
</tr>
<tr>
<td>

**数据库管理**
<br>统一管理 MySQL、PostgreSQL、Redis 实例。

</td>
<td>

**容器控制**
<br>完整的 Docker 容器、镜像、网络生命周期管理。

</td>
</tr>
<tr>
<td>

**文件浏览器**
<br>在面板中直接浏览和管理宿主机文件系统。

</td>
<td>

**系统仪表盘**
<br>实时展示宿主机资源使用状态。

</td>
</tr>
<tr>
<td>

**AI 助手**
<br>内置只读 AI Copilot，辅助服务器问题排查。

</td>
<td>

**单文件部署**
<br>编译为单个 Go 二进制文件，前端内嵌，无需额外运行时。

</td>
</tr>
</table>

## 技术栈

| 层级 | 技术 |
|:-----|:-----|
| 后端 | Go、SQLite、Docker Engine API |
| 前端 | React 19、Ant Design、Bun |
| 反向代理 | OpenResty（托管式） |
| 部署 | systemd、GitHub Actions CI/CD |

<br>

## 快速开始

### 环境要求

- Linux（amd64 或 arm64）
- Docker Engine 及 `docker compose`

### 安装

```bash
curl -fsSL https://raw.githubusercontent.com/<owner>/CamoPanel/main/deploy/install.sh | sudo bash
```

或下载 [Release](../../releases) 压缩包后手动安装：

```bash
tar -xzf camopanel_linux_amd64.tar.gz
cd camopanel_linux_amd64
sudo ./deploy/install.sh
```

安装完成后，在浏览器中打开 `http://<服务器IP>:8080`。

> 默认账号：`admin` / `admin123`

<br>

## 配置说明

所有配置通过环境变量完成，环境文件位于 `/etc/camopanel/camopanel.env`。

<details>
<summary><b>查看全部变量</b></summary>
<br>

| 变量 | 说明 | 默认值 |
|:-----|:-----|:-------|
| `CAMO_HTTP_ADDR` | HTTP 监听地址 | `:8080` |
| `CAMO_DATA_DIR` | 数据存储目录 | `./data` |
| `CAMO_TEMPLATES_DIR` | 应用模板目录 | 自动探测 |
| `CAMO_SESSION_SECRET` | 会话加密密钥 | — |
| `CAMO_ADMIN_USERNAME` | 初始管理员用户名 | `admin` |
| `CAMO_ADMIN_PASSWORD` | 初始管理员密码 | `admin123` |
| `CAMO_BRIDGE_NETWORK` | Docker 桥接网络名称 | `camopanel` |
| `CAMO_OPENRESTY_CONTAINER` | OpenResty 容器名 | `camopanel-openresty` |
| `CAMO_HOST_CONTROL_HELPER` | 宿主机控制 helper 路径 | `/usr/local/bin/camopanel-hostctl` |
| `CAMO_AI_BASE_URL` | AI 服务地址 | — |
| `CAMO_AI_MODEL` | AI 模型标识 | — |
| `CAMO_AI_API_KEY` | AI 服务密钥 | — |

</details>

<br>

## 本地开发

### 开发环境要求

| 工具 | 版本 |
|:-----|:-----|
| Go | 1.25+ |
| Bun | 1.3+ |
| Docker Engine | 最新版 |

### 开发服务

```bash
make dev              # 同时启动后端和前端
make dev-server       # 仅后端（localhost:8080）
make dev-web          # 仅前端（/api 代理到 localhost:8080）
```

### 构建与测试

```bash
make build            # 产出单个二进制文件（前端内嵌）→ server/camopanel
make test-server      # 运行全部后端测试
```

<br>

## 部署

### 安装路径

| 内容 | 路径 |
|:-----|:-----|
| 二进制文件 | `/opt/camopanel/camopanel` |
| 数据目录 | `/opt/camopanel/data` |
| 环境文件 | `/etc/camopanel/camopanel.env` |
| systemd 单元 | `/etc/systemd/system/camopanel.service` |

### 发布构建

推送 `v*` 标签后，GitHub Actions 自动构建 `linux/amd64` 和 `linux/arm64` 两个平台的发布包。

<br>

## 项目结构

```
CamoPanel/
├── server/                 # Go 后端
│   ├── cmd/                #   入口
│   └── internal/
│       ├── bootstrap/      #   应用初始化
│       ├── modules/        #   业务域
│       │                   #     auth · projects · runtime · websites
│       │                   #     databases · files · system · copilot
│       └── platform/       #   基础设施适配
│                           #     Docker · OpenResty · SQLite · 文件系统
├── web/                    # React SPA
│   └── src/
│       ├── app/            #   路由入口
│       ├── modules/        #   功能模块
│       │                   #     dashboard · store · websites · databases
│       │                   #     containers · files · copilot
│       ├── widgets/        #   壳层布局与共享 UI
│       └── shared/         #   请求器与共享类型
├── templates/              # Docker Compose 应用模板
└── deploy/                 # systemd 单元 · 安装脚本 · 环境变量示例
```
