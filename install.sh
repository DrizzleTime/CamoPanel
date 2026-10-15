#!/usr/bin/env bash
# Install the native CamoPanel release. Docker is used by managed apps and Nginx.
set -Eeuo pipefail

REPOSITORY=DrizzleTime/CamoPanel
SYSTEMD_DIR=/etc/systemd/system
INSTALL_DIR=/opt/camopanel
VERSION=latest
PORT=8080
USERNAME='admin'
PASSWORD=${CAMO_ADMIN_PASSWORD:-}
NONINTERACTIVE=false
WORK_DIR=

info() { printf '\n[CamoPanel] %s\n' "$*"; }
die() { printf '\n[CamoPanel] 错误：%s\n' "$*" >&2; exit 1; }
cleanup() { if [[ -n $WORK_DIR ]]; then rm -rf -- "$WORK_DIR"; fi; }

usage() {
  cat <<'EOF'
CamoPanel 二进制安装脚本（Linux / systemd / amd64、arm64）

用法：sudo bash install.sh [选项]
  --yes                  使用默认值，无交互安装
  --version <vX.Y.Z>      安装指定 GitHub Release，默认 latest
  --install-dir <目录>   安装及数据目录，默认 /opt/camopanel
  --port <端口>          面板端口，默认 8080
  --username <账号>      管理员账号，默认 admin
  -h, --help             查看帮助

密码默认随机生成；可通过 CAMO_ADMIN_PASSWORD 环境变量指定（8–72 字节）。
脚本会自动检查并安装 Docker Engine、Docker Compose，注册开机自启服务。
已有 Docker 不会重装。已有 CamoPanel 安装不会被覆盖。
EOF
}

parse_args() {
  while (($#)); do
    case "$1" in
      --yes) NONINTERACTIVE=true; shift ;;
      --version|--install-dir|--port|--username)
        (($# >= 2)) && [[ -n $2 && $2 != --* ]] || die "$1 缺少参数"
        case "$1" in
          --version) VERSION=$2 ;;
          --install-dir) INSTALL_DIR=$2 ;;
          --port) PORT=$2 ;;
          --username) USERNAME=$2 ;;
        esac
        shift 2 ;;
      -h|--help) usage; exit 0 ;;
      *) die "未知选项：$1（使用 --help 查看帮助）" ;;
    esac
  done
}

require_root() { [[ $EUID -eq 0 ]] || die "请使用 sudo bash install.sh 或切换到 root 后运行。"; }

check_system() {
  [[ $(uname -s) == Linux ]] || die "仅支持 Linux。"
  case "$(uname -m)" in
    x86_64|amd64) ARCH=amd64 ;;
    aarch64|arm64) ARCH=arm64 ;;
    *) die "不支持的 CPU 架构：$(uname -m)" ;;
  esac
  command -v systemctl >/dev/null && [[ -d /run/systemd/system ]] || die "需要正在运行的 systemd。"
}

configure() {
  if [[ $NONINTERACTIVE == false && -t 0 ]]; then
    info "安装 CamoPanel：面板直接运行于宿主机，Docker 用于应用商店和 Nginx 网关。"
    local answer
    read -r -p "安装目录 [$INSTALL_DIR]：" answer
    INSTALL_DIR=${answer:-$INSTALL_DIR}
    read -r -p "面板端口 [$PORT]：" answer
    PORT=${answer:-$PORT}
    read -r -p "管理员账号 [$USERNAME]：" answer
    USERNAME=${answer:-$USERNAME}
    if [[ -z $PASSWORD ]]; then
      read -r -s -p '管理员密码 [留空随机生成]：' PASSWORD
      printf '\n'
    fi
  fi
  # Restrict paths so they are also safe as literal systemd unit values.
  [[ $INSTALL_DIR =~ ^/[a-zA-Z0-9_./-]+$ && $INSTALL_DIR != / ]] || die "安装目录必须为绝对路径，且只含字母、数字、/、_、-、.。"
  INSTALL_DIR=$(realpath -m -- "$INSTALL_DIR")
  [[ $INSTALL_DIR != / ]] || die "不能安装到根目录。"
  [[ $PORT =~ ^[0-9]{1,5}$ ]] || die "面板端口必须为 1–65535。"
  PORT=$((10#$PORT))
  ((PORT >= 1 && PORT <= 65535 && PORT != 80 && PORT != 443)) || die "端口必须为 1–65535，且不能占用 Nginx 网关的 80/443。"
  [[ $USERNAME =~ ^[a-zA-Z0-9_][a-zA-Z0-9_.-]{0,63}$ ]] || die "管理员账号需为 1–64 位字母、数字、_、-、.。"
  [[ $VERSION == latest || $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]] || die "版本号格式应为 vX.Y.Z。"
  if [[ -z $PASSWORD ]]; then
    PASSWORD=$(od -An -N24 -tx1 /dev/urandom | tr -d ' \n')
  fi
  local password_bytes
  password_bytes=$(printf %s "$PASSWORD" | wc -c)
  ((password_bytes >= 8 && password_bytes <= 72)) && [[ $PASSWORD != *$'\n'* && $PASSWORD != *$'\r'* ]] || die "密码必须为 8–72 字节且不含换行。"
  [[ ! -e $INSTALL_DIR/.env && ! -e $INSTALL_DIR/camopanel && ! -e $INSTALL_DIR/web && ! -L $INSTALL_DIR ]] || die "目录中已有 CamoPanel 文件，已停止以保留配置和数据：$INSTALL_DIR"
  [[ ! -e $SYSTEMD_DIR/camopanel.service ]] || die "已存在 camopanel.service，请先检查现有服务。"
  [[ -z $(ss -H -ltn "sport = :$PORT") ]] || die "面板端口 $PORT 已被占用，请用 --port 指定其他端口。"
}

install_packages() {
  if command -v apt-get >/dev/null; then
    apt-get update
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends "$@"
  elif command -v dnf >/dev/null; then
    dnf install -y "$@"
  elif command -v yum >/dev/null; then
    yum install -y "$@"
  else
    die "无法自动安装依赖，请先安装 curl、ca-certificates、tar、gzip、coreutils、iproute2 以及 Docker Compose。"
  fi
}

ensure_tools() {
  local tool missing=false
  for tool in curl tar gzip sha256sum realpath od tr wc ss; do
    if ! command -v "$tool" >/dev/null; then missing=true; fi
  done
  if [[ $missing == true || ! -s /etc/ssl/certs/ca-certificates.crt && ! -s /etc/pki/tls/certs/ca-bundle.crt ]]; then
    info '安装下载、校验和网络检查工具'
    if command -v apt-get >/dev/null; then
      install_packages curl ca-certificates tar gzip coreutils iproute2
    else
      install_packages curl ca-certificates tar gzip coreutils iproute
    fi
  fi
}

download() {
  curl --fail --show-error --location --retry 3 --connect-timeout 15 --max-time 600 \
    --proto '=https' --proto-redir '=https' "$1" --output "$2"
}

fetch_release() {
  local base asset expected
  asset="camopanel_linux_${ARCH}.tar.gz"
  if [[ $VERSION == latest ]]; then
    base="https://github.com/$REPOSITORY/releases/latest/download"
  else
    base="https://github.com/$REPOSITORY/releases/download/$VERSION"
  fi
  info "下载 $VERSION / linux/$ARCH 安装包"
  download "$base/$asset" "$WORK_DIR/$asset" || die "下载安装包失败。请确认 GitHub Release 已发布且包含 $asset，并检查网络连接。"
  download "$base/checksums.txt" "$WORK_DIR/checksums.txt" || die "下载 SHA256 校验文件失败。"
  expected=$(awk -v asset="$asset" '$2 == asset { print $1 }' "$WORK_DIR/checksums.txt")
  [[ $expected =~ ^[a-fA-F0-9]{64}$ ]] || die "校验文件中没有唯一有效的 $asset 校验值。"
  printf '%s  %s\n' "$expected" "$WORK_DIR/$asset" | sha256sum --check --status || die "安装包 SHA256 校验失败，请重试。"
  mkdir "$WORK_DIR/package"
  tar -xzf "$WORK_DIR/$asset" --no-same-owner --no-same-permissions -C "$WORK_DIR/package"
  [[ -x $WORK_DIR/package/camopanel && -f $WORK_DIR/package/web/index.html && -s $WORK_DIR/package/VERSION ]] || die "安装包不完整。"
  RELEASE_VERSION=$(cat "$WORK_DIR/package/VERSION")
  [[ $RELEASE_VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]] || die "安装包版本号无效。"
  [[ $VERSION == latest || $VERSION == "$RELEASE_VERSION" ]] || die "安装包版本与请求的版本不一致。"
}

configure_docker_repository() {
  local ID VERSION_CODENAME
  [[ -r /etc/os-release ]] || die "无法识别系统，请手动安装 Docker Engine 和 Compose 后重试。"
  # /etc/os-release is maintained by the operating system.
  # shellcheck source=/dev/null
  . /etc/os-release
  case "$ID" in
    debian|ubuntu)
      [[ -n ${VERSION_CODENAME:-} ]] || die "无法识别发行版代号，请手动安装 Docker。"
      install -d -m 0755 /etc/apt/keyrings
      download "https://download.docker.com/linux/$ID/gpg" "$WORK_DIR/docker.asc"
      install -m 0644 "$WORK_DIR/docker.asc" /etc/apt/keyrings/camopanel-docker.asc
      cat > /etc/apt/sources.list.d/camopanel-docker.sources <<EOF
Types: deb
URIs: https://download.docker.com/linux/$ID
Suites: $VERSION_CODENAME
Components: stable
Architectures: $(dpkg --print-architecture)
Signed-By: /etc/apt/keyrings/camopanel-docker.asc
EOF
      ;;
    centos|rhel|fedora)
      download "https://download.docker.com/linux/$ID/docker-ce.repo" "$WORK_DIR/docker-ce.repo"
      install -m 0644 "$WORK_DIR/docker-ce.repo" /etc/yum.repos.d/docker-ce.repo
      ;;
    *) die "暂不支持自动安装 $ID 的 Docker，请先按 Docker 官方文档安装 Engine 和 Compose，再重新运行脚本。" ;;
  esac
}

ensure_docker() {
  # The panel and Compose must use the same local Engine, regardless of shell context.
  export DOCKER_HOST=unix:///var/run/docker.sock
  unset DOCKER_CONTEXT DOCKER_TLS_VERIFY DOCKER_CERT_PATH
  if ! command -v docker >/dev/null; then
    info '未检测到 Docker，通过 Docker 官方软件源安装 Engine 和 Compose'
    configure_docker_repository
    install_packages docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  else
    info '检测到已有 Docker，保留现有安装'
  fi
  if ! docker compose version >/dev/null 2>&1; then
    info '安装缺失的 Docker Compose 插件'
    if command -v apt-get >/dev/null; then
      apt-get update
      if apt-cache show docker-compose-plugin >/dev/null 2>&1; then
        DEBIAN_FRONTEND=noninteractive apt-get install -y docker-compose-plugin
      else
        DEBIAN_FRONTEND=noninteractive apt-get install -y docker-compose-v2
      fi
    else
      install_packages docker-compose-plugin
    fi
  fi
  docker compose version >/dev/null || die "Docker Compose 不可用，请安装 docker compose 插件后重试。"
  if ! docker info >/dev/null 2>&1; then
    info '启动 Docker 服务'
    systemctl enable --now docker
  fi
  local attempt
  for ((attempt=1; attempt<=15; attempt++)); do
    if docker info >/dev/null 2>&1; then
      info "Docker Engine 和 Compose 已就绪"
      return
    fi
    sleep 2
  done
  die "无法连接 /var/run/docker.sock，请用 journalctl -u docker 检查 Docker 服务。"
}

install_panel() {
  info "安装 $RELEASE_VERSION 到 $INSTALL_DIR"
  install -d -m 0755 "$INSTALL_DIR" "$INSTALL_DIR/data"
  install -m 0755 "$WORK_DIR/package/camopanel" "$INSTALL_DIR/camopanel"
  cp -R "$WORK_DIR/package/web" "$INSTALL_DIR/web"
  install -m 0644 "$WORK_DIR/package/VERSION" "$INSTALL_DIR/VERSION"
  # This file is parsed by CamoPanel, never sourced as shell code.
  (umask 077; cat > "$INSTALL_DIR/.env" <<EOF
CAMO_ADDR=0.0.0.0:$PORT
DOCKER_HOST=unix:///var/run/docker.sock
CAMO_ADMIN_USERNAME=$USERNAME
CAMO_ADMIN_PASSWORD='$PASSWORD'
CAMO_WEB_DIR=$INSTALL_DIR/web
CAMO_DATA_DIR=$INSTALL_DIR/data
CAMO_FILES_ROOT=/
CAMO_NGINX_PORT=80
CAMO_NGINX_SSL_PORT=443
EOF
  )
  cat > "$SYSTEMD_DIR/camopanel.service" <<EOF
[Unit]
Description=CamoPanel server management panel
Wants=network-online.target docker.service
After=network-online.target docker.service
StartLimitIntervalSec=60
StartLimitBurst=5

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/camopanel
Environment=PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
Restart=on-failure
RestartSec=5
TimeoutStopSec=15
UMask=0022

[Install]
WantedBy=multi-user.target
EOF
  chmod 0644 "$SYSTEMD_DIR/camopanel.service"
  systemctl daemon-reload
  systemctl enable --now camopanel
  info '等待面板启动（首次启动可能需要拉取 Nginx 镜像）'
  local attempt
  for ((attempt=1; attempt<=60; attempt++)); do
    if systemctl is-active --quiet camopanel && \
      curl --fail --silent --noproxy '*' --connect-timeout 1 --max-time 2 "http://127.0.0.1:$PORT/api/health" >/dev/null; then
      info '安装完成'
      printf '  版本：%s\n  面板：http://<服务器IP>:%s/login\n  账号：%s\n  密码：%s\n  配置：%s/.env（仅 root 可读写）\n  数据：%s/data\n' \
        "$RELEASE_VERSION" "$PORT" "$USERNAME" "$PASSWORD" "$INSTALL_DIR" "$INSTALL_DIR"
      printf '\n  管理服务：systemctl {start|stop|restart|status} camopanel\n  查看日志：journalctl -u camopanel -f\n  请在服务器防火墙和云安全组放行面板端口 %s；网站按需放行 80/443。\n' "$PORT"
      return
    fi
    sleep 2
  done
  die "面板未通过启动检查。文件已保留于 $INSTALL_DIR；请运行 journalctl -u camopanel -n 100 --no-pager 排查，再执行 systemctl restart camopanel。"
}

main() {
  parse_args "$@"
  require_root
  check_system
  trap cleanup EXIT
  trap 'printf "\n[CamoPanel] 安装中断（第 %s 行），请检查上方错误。\n" "$LINENO" >&2' ERR
  ensure_tools
  configure
  WORK_DIR=$(mktemp -d)
  # Verify the release before changing Docker or writing panel files.
  fetch_release
  ensure_docker
  install_panel
}

if [[ ${BASH_SOURCE[0]} == "$0" ]]; then main "$@"; fi
