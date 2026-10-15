# 二进制安装与发布

CamoPanel 推荐直接在宿主机运行 Go 二进制，由 systemd 管理。发布包包含编译后的前端资源，服务器无需 Go、Bun 或 Node.js。应用商店和 Nginx 网关使用宿主机 Docker。

## 安装

要求 Linux、正在运行的 systemd、root 权限，以及 amd64 / arm64 CPU。自动安装 Docker 支持 Debian、Ubuntu、CentOS Stream、RHEL、Fedora；已有可用 Docker Engine 和 Compose 的其他 Linux 系统也可以安装。下载需能访问 GitHub；安装缺失依赖时需要访问发行版软件源和 Docker 官方软件源。

```bash
curl -fSL https://github.com/DrizzleTime/CamoPanel/releases/latest/download/install.sh -o camopanel-install.sh
sudo bash camopanel-install.sh
```

交互模式依次设置安装目录、端口、账号、密码；密码留空则随机生成。无交互时使用默认配置，也可显式传参：

```bash
sudo bash camopanel-install.sh --yes \
  --version v0.1.0 \
  --install-dir /opt/camopanel \
  --port 18080 \
  --username admin
```

默认版本是最新正式 Release，指定版本也可以安装预发布包。自定义密码通过 `CAMO_ADMIN_PASSWORD` 环境变量传入，8–72 字节、不含换行；使用 sudo 时需显式保留此变量，例如 `sudo --preserve-env=CAMO_ADMIN_PASSWORD bash camopanel-install.sh --yes`。不要将密码写入脚本参数。

| 项目 | 默认值 |
| --- | --- |
| 面板二进制 | `/opt/camopanel/camopanel` |
| 前端资源 | `/opt/camopanel/web/` |
| 配置文件 | `/opt/camopanel/.env`，权限 `0600` |
| 持久化数据 | `/opt/camopanel/data/` |
| 已安装版本 | `/opt/camopanel/VERSION` |
| systemd 服务 | `/etc/systemd/system/camopanel.service` |
| 面板监听地址 | `0.0.0.0:8080` |
| Docker socket | `/var/run/docker.sock` |

安装脚本先下载 Release 并校验 SHA256，再配置 Docker 和安装面板。缺失 Docker Engine 时，使用 Docker 官方 apt / RPM 软件源安装 Engine、CLI、Buildx 和 Compose；已安装 Engine 时仅补充缺失的 Compose 插件。它会启动尚未运行的 Docker 服务，并验证引擎连接。对应流程参考 Docker 官方的 [Ubuntu](https://docs.docker.com/engine/install/ubuntu/)、[Debian](https://docs.docker.com/engine/install/debian/)、[CentOS](https://docs.docker.com/engine/install/centos/)、[RHEL](https://docs.docker.com/engine/install/rhel/)、[Fedora](https://docs.docker.com/engine/install/fedora/) 与 [Compose](https://docs.docker.com/compose/install/linux/) 安装文档。

脚本使用标准 rootful Docker socket；已有 Rootless Docker 或自定义守护进程的服务器应先手动配置合适的 Docker 环境。面板安装为 root 服务，直接管理宿主机文件、进程与 Docker。安装完成后输出访问地址和管理员凭据；脚本不会自动修改防火墙。请放行面板端口，网站按需放行 `80/443`。远程访问如需 HTTPS，可在 `.env` 中同时设置 `CAMO_TLS_CERT` 和 `CAMO_TLS_KEY`，再重启服务。

## 管理与排错

```bash
sudo systemctl status camopanel
sudo systemctl restart camopanel
sudo systemctl stop camopanel
sudo journalctl -u camopanel -f
```

管理员凭据及其他环境配置保存在安装目录的 `.env` 中，由面板自身读取，修改后重启服务生效。数据始终位于安装目录的 `data/`。脚本拒绝覆盖已有 CamoPanel 文件或服务，重复运行不会重置账号或删除数据；当前脚本用于首次安装，不执行升级。

若提示安装包下载失败，请确认仓库已有正式 Release，且该版本包含对应架构的安装包和 `checksums.txt`。若 SHA256 不匹配，脚本会停止，不安装面板，也不改动 Docker。

若提示面板启动检查失败，安装文件及配置会保留。查看日志后修复 `.env` 或依赖问题，再执行 `systemctl restart camopanel`；不必重复执行安装脚本。首次启动会尝试拉取 Nginx 镜像，网关相关错误也可通过服务日志和 `docker logs camo-nginx` 排查。

## 构建并发布安装包

开发机或 CI 需要 Go、Bun、Node.js；这些工具不需要安装在目标服务器上。

```bash
bun install --frozen-lockfile
bun run test
bun run release v0.1.0
# 也可只构建指定架构
bun run release v0.1.0 amd64
```

打包脚本先构建前端，再以 `CGO_ENABLED=0` 交叉编译 Go 服务，生成：

```text
dist/
  camopanel_linux_amd64.tar.gz
  camopanel_linux_arm64.tar.gz
  checksums.txt
  install.sh
```

每个压缩包包含 `camopanel`、`web/`、`VERSION`、`install.sh` 和 `README.md`。`checksums.txt` 校验本次构建的压缩包和独立安装脚本。根目录的 `Dockerfile`、Compose 部署配置已移除；应用商店内的 Compose 模板仍用于部署被管理的应用。

提交代码后推送版本标签：

```bash
git tag v0.1.0
git push origin v0.1.0
```

`.github/workflows/release.yml` 会运行检查、构建两个架构，并通过 GitHub CLI 发布安装包与校验文件。`vX.Y.Z-rc.1` 等带后缀标签发布为预发布版本，不作为默认安装版本。工作流需要仓库允许 Actions 创建 Release（`contents: write`）。首次 Release 完成后，README 中的一键下载地址才可使用。
