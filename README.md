<div align="center">

<br>

# CamoPanel

**Lightweight Linux Server Management Panel**

<br>

A modern, self-hosted container management panel designed for single-server Linux environments.
<br>
Deploy applications, manage websites, databases, and containers — all from one clean interface.

<br>

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
[![React](https://img.shields.io/badge/React-19-61DAFB?style=flat&logo=react&logoColor=white)](https://react.dev)
[![Bun](https://img.shields.io/badge/Bun-1.3+-F9F1E1?style=flat&logo=bun&logoColor=black)](https://bun.sh)
[![Platform](https://img.shields.io/badge/Platform-Linux-FCC624?style=flat&logo=linux&logoColor=black)](https://kernel.org)

<br>

[中文文档](README_zh.md)

<br>

</div>

## Features

<table>
<tr>
<td width="50%">

**App Store**
<br>Deploy OpenResty, PostgreSQL, MySQL, Redis, WordPress, PHP-FPM from built-in templates with one click.

</td>
<td width="50%">

**Website Management**
<br>Create and manage OpenResty-powered sites with automatic reverse proxy configuration.

</td>
</tr>
<tr>
<td>

**Database Management**
<br>Unified interface for MySQL, PostgreSQL, and Redis instances.

</td>
<td>

**Container Control**
<br>Full Docker container, image, and network lifecycle management.

</td>
</tr>
<tr>
<td>

**File Browser**
<br>Browse and manage host filesystem directly from the panel.

</td>
<td>

**System Dashboard**
<br>Real-time host resource monitoring at a glance.

</td>
</tr>
<tr>
<td>

**AI Copilot**
<br>Built-in read-only AI assistant for server troubleshooting.

</td>
<td>

**Single Binary**
<br>Ships as one Go binary with embedded frontend. No external runtime required.

</td>
</tr>
</table>

## Tech Stack

| Layer | Technology |
|:------|:-----------|
| Backend | Go, SQLite, Docker Engine API |
| Frontend | React 19, Ant Design, Bun |
| Reverse Proxy | OpenResty (managed) |
| Deployment | systemd, GitHub Actions CI/CD |

<br>

## Quick Start

### Prerequisites

- Linux (amd64 or arm64)
- Docker Engine with `docker compose`

### Install

```bash
curl -fsSL https://raw.githubusercontent.com/<owner>/CamoPanel/main/deploy/install.sh | sudo bash
```

Or download a [release](../../releases) archive and run manually:

```bash
tar -xzf camopanel_linux_amd64.tar.gz
cd camopanel_linux_amd64
sudo ./deploy/install.sh
```

Once installed, open `http://<server-ip>:8080` in your browser.

> Default credentials: `admin` / `admin123`

<br>

## Configuration

All configuration is managed via environment variables in `/etc/camopanel/camopanel.env`.

<details>
<summary><b>View all variables</b></summary>
<br>

| Variable | Description | Default |
|:---------|:------------|:--------|
| `CAMO_HTTP_ADDR` | HTTP listen address | `:8080` |
| `CAMO_DATA_DIR` | Data storage directory | `./data` |
| `CAMO_TEMPLATES_DIR` | App template directory | Auto-detected |
| `CAMO_SESSION_SECRET` | Session encryption key | — |
| `CAMO_ADMIN_USERNAME` | Initial admin username | `admin` |
| `CAMO_ADMIN_PASSWORD` | Initial admin password | `admin123` |
| `CAMO_BRIDGE_NETWORK` | Docker bridge network name | `camopanel` |
| `CAMO_OPENRESTY_CONTAINER` | OpenResty container name | `camopanel-openresty` |
| `CAMO_HOST_CONTROL_HELPER` | Host control helper path | `/usr/local/bin/camopanel-hostctl` |
| `CAMO_AI_BASE_URL` | AI service base URL | — |
| `CAMO_AI_MODEL` | AI model identifier | — |
| `CAMO_AI_API_KEY` | AI service API key | — |

</details>

<br>

## Development

### Requirements

| Tool | Version |
|:-----|:--------|
| Go | 1.25+ |
| Bun | 1.3+ |
| Docker Engine | Latest |

### Dev Server

```bash
make dev              # start backend + frontend in parallel
make dev-server       # backend only  (localhost:8080)
make dev-web          # frontend only (proxies /api → localhost:8080)
```

### Build & Test

```bash
make build            # single binary with embedded frontend → server/camopanel
make test-server      # run all backend tests
```

<br>

## Deployment

### Installation Paths

| Item | Path |
|:-----|:-----|
| Binary | `/opt/camopanel/camopanel` |
| Data | `/opt/camopanel/data` |
| Environment | `/etc/camopanel/camopanel.env` |
| systemd Unit | `/etc/systemd/system/camopanel.service` |

### Release Builds

Pushing a `v*` tag triggers GitHub Actions to produce release archives for `linux/amd64` and `linux/arm64`.

<br>

## Project Structure

```
CamoPanel/
├── server/                 # Go backend
│   ├── cmd/                #   Entry point
│   └── internal/
│       ├── bootstrap/      #   App initialization
│       ├── modules/        #   Business domains
│       │                   #     auth · projects · runtime · websites
│       │                   #     databases · files · system · copilot
│       └── platform/       #   Infrastructure adapters
│                           #     Docker · OpenResty · SQLite · filesystem
├── web/                    # React SPA
│   └── src/
│       ├── app/            #   Router & entry
│       ├── modules/        #   Feature modules
│       │                   #     dashboard · store · websites · databases
│       │                   #     containers · files · copilot
│       ├── widgets/        #   Shell layout & shared UI
│       └── shared/         #   HTTP client & shared types
├── templates/              # Docker Compose app templates
└── deploy/                 # systemd unit · install script · env example
```
