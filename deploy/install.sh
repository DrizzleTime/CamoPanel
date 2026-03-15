#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_PACKAGE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

PACKAGE_SOURCE=""
BIN_DIR="/usr/local/bin"
DATA_DIR="/var/lib/camopanel"
ENV_FILE=""
SERVICE_FILE=""
RUN_USER="camopanel"
RUN_GROUP="camopanel"
NO_START=0

TMP_PATHS=()

log() {
  printf '[camopanel] %s\n' "$*"
}

fail() {
  printf '[camopanel] %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'EOF'
用法:
  ./deploy/install.sh [--package PATH_OR_URL] [--bin-dir DIR] [--data-dir DIR]
                      [--env-file FILE] [--service-file FILE]
                      [--user USER] [--group GROUP] [--no-start]

说明:
  - 不传 --package 时，默认使用脚本上一级目录作为发布包根目录
  - --package 支持本地目录、本地 .tar.gz/.tgz 文件，或 http(s) URL
  - 正式部署建议使用 root 运行
EOF
}

cleanup() {
  local path
  for path in "${TMP_PATHS[@]}"; do
    if [ -e "$path" ]; then
      rm -rf "$path"
    fi
  done
}

trap cleanup EXIT

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "缺少命令: $1"
}

abs_path() {
  local target=$1

  if [ -d "$target" ]; then
    (cd "$target" && pwd)
    return
  fi

  (cd "$(dirname "$target")" && printf '%s/%s\n' "$(pwd)" "$(basename "$target")")
}

escape_sed() {
  printf '%s' "$1" | sed 's/[&|\\]/\\&/g'
}

random_string() {
  local length=$1

  if command -v openssl >/dev/null 2>&1; then
    openssl rand -base64 64 | tr -dc 'A-Za-z0-9' | cut -c1-"$length"
    return
  fi

  if command -v python3 >/dev/null 2>&1; then
    python3 - "$length" <<'PY'
import secrets
import string
import sys

alphabet = string.ascii_letters + string.digits
length = int(sys.argv[1])
print("".join(secrets.choice(alphabet) for _ in range(length)), end="")
PY
    return
  fi

  fail "缺少随机数生成工具: 需要 openssl 或 python3"
}

is_url() {
  case "$1" in
    http://*|https://*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

download_file() {
  local url=$1
  local output=$2

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$output"
    return
  fi

  if command -v wget >/dev/null 2>&1; then
    wget -qO "$output" "$url"
    return
  fi

  fail "下载发布包需要 curl 或 wget"
}

resolve_archive_root() {
  local extract_dir=$1
  local entries=()

  shopt -s nullglob dotglob
  entries=("$extract_dir"/*)
  shopt -u nullglob dotglob

  if [ "${#entries[@]}" -eq 1 ] && [ -d "${entries[0]}" ]; then
    printf '%s\n' "${entries[0]}"
    return
  fi

  printf '%s\n' "$extract_dir"
}

prepare_package_dir() {
  local source=$1
  local tmp_dir=""
  local archive_path=""
  local extract_dir=""

  if [ -z "$source" ]; then
    printf '%s\n' "$DEFAULT_PACKAGE_DIR"
    return
  fi

  if is_url "$source"; then
    need_cmd mktemp
    tmp_dir=$(mktemp -d)
    TMP_PATHS+=("$tmp_dir")
    archive_path="$tmp_dir/package.tar.gz"
    log "下载发布包: $source"
    download_file "$source" "$archive_path"
    source="$archive_path"
  elif [ -d "$source" ]; then
    abs_path "$source"
    return
  else
    source=$(abs_path "$source")
  fi

  [ -f "$source" ] || fail "发布包不存在: $source"

  case "$source" in
    *.tar.gz|*.tgz)
      ;;
    *)
      fail "仅支持 .tar.gz 或 .tgz 发布包: $source"
      ;;
  esac

  need_cmd tar
  need_cmd mktemp
  extract_dir=$(mktemp -d)
  TMP_PATHS+=("$extract_dir")
  tar -xzf "$source" -C "$extract_dir"
  resolve_archive_root "$extract_dir"
}

ensure_runtime_account() {
  local current_user
  local current_group
  local nologin_bin

  current_user=$(id -un)
  current_group=$(id -gn)

  if [ "$EUID" -ne 0 ]; then
    [ "$RUN_USER" = "$current_user" ] || fail "非 root 模式下 --user 必须等于当前用户: $current_user"
    [ "$RUN_GROUP" = "$current_group" ] || fail "非 root 模式下 --group 必须等于当前用户组: $current_group"
    [ "$NO_START" -eq 1 ] || fail "非 root 模式下请使用 --no-start"
    return
  fi

  if ! getent group "$RUN_GROUP" >/dev/null 2>&1; then
    groupadd --system "$RUN_GROUP"
  fi

  if id -u "$RUN_USER" >/dev/null 2>&1; then
    return
  fi

  nologin_bin=$(command -v nologin || true)
  if [ -z "$nologin_bin" ]; then
    nologin_bin="/usr/sbin/nologin"
  fi

  useradd --system --gid "$RUN_GROUP" --home-dir "$DATA_DIR" --shell "$nologin_bin" "$RUN_USER"
}

render_service_file() {
  local source=$1
  local target=$2
  local bin_path=$3

  sed \
    -e "s|^User=.*$|User=$(escape_sed "$RUN_USER")|" \
    -e "s|^Group=.*$|Group=$(escape_sed "$RUN_GROUP")|" \
    -e "s|^WorkingDirectory=.*$|WorkingDirectory=$(escape_sed "$DATA_DIR")|" \
    -e "s|^EnvironmentFile=.*$|EnvironmentFile=-$(escape_sed "$ENV_FILE")|" \
    -e "s|^ExecStart=.*$|ExecStart=$(escape_sed "$bin_path")|" \
    "$source" > "$target"
  chmod 644 "$target"
}

render_env_example() {
  local source=$1
  local target=$2

  sed \
    -e "s|^CAMO_DATA_DIR=.*$|CAMO_DATA_DIR=$(escape_sed "$DATA_DIR")|" \
    -e "s|^CAMO_TEMPLATES_DIR=.*$|CAMO_TEMPLATES_DIR=$(escape_sed "$DATA_DIR/templates")|" \
    "$source" > "$target"
  chmod 644 "$target"
}

render_env_file() {
  local source=$1
  local target=$2
  local session_secret=$3
  local admin_password=$4

  sed \
    -e "s|^CAMO_DATA_DIR=.*$|CAMO_DATA_DIR=$(escape_sed "$DATA_DIR")|" \
    -e "s|^CAMO_TEMPLATES_DIR=.*$|CAMO_TEMPLATES_DIR=$(escape_sed "$DATA_DIR/templates")|" \
    -e "s|^CAMO_SESSION_SECRET=.*$|CAMO_SESSION_SECRET=$(escape_sed "$session_secret")|" \
    -e "s|^CAMO_ADMIN_PASSWORD=.*$|CAMO_ADMIN_PASSWORD=$(escape_sed "$admin_password")|" \
    "$source" > "$target"
  chmod 640 "$target"
}

while [ $# -gt 0 ]; do
  case "$1" in
    --package)
      [ $# -ge 2 ] || fail "--package 缺少参数"
      PACKAGE_SOURCE=$2
      shift 2
      ;;
    --bin-dir)
      [ $# -ge 2 ] || fail "--bin-dir 缺少参数"
      BIN_DIR=$2
      shift 2
      ;;
    --data-dir)
      [ $# -ge 2 ] || fail "--data-dir 缺少参数"
      DATA_DIR=$2
      shift 2
      ;;
    --env-file)
      [ $# -ge 2 ] || fail "--env-file 缺少参数"
      ENV_FILE=$2
      shift 2
      ;;
    --service-file)
      [ $# -ge 2 ] || fail "--service-file 缺少参数"
      SERVICE_FILE=$2
      shift 2
      ;;
    --user)
      [ $# -ge 2 ] || fail "--user 缺少参数"
      RUN_USER=$2
      shift 2
      ;;
    --group)
      [ $# -ge 2 ] || fail "--group 缺少参数"
      RUN_GROUP=$2
      shift 2
      ;;
    --no-start)
      NO_START=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "未知参数: $1"
      ;;
  esac
done

if [ -z "$ENV_FILE" ]; then
  ENV_FILE="/etc/camopanel/camopanel.env"
fi

if [ -z "$SERVICE_FILE" ]; then
  SERVICE_FILE="/etc/systemd/system/camopanel.service"
fi

PACKAGE_DIR=$(prepare_package_dir "$PACKAGE_SOURCE")
PACKAGE_DIR=$(abs_path "$PACKAGE_DIR")

BINARY_SOURCE="$PACKAGE_DIR/camopanel"
TEMPLATES_SOURCE="$PACKAGE_DIR/templates"
SERVICE_SOURCE="$PACKAGE_DIR/deploy/camopanel.service"
ENV_EXAMPLE_SOURCE="$PACKAGE_DIR/deploy/camopanel.env.example"
BIN_PATH="$BIN_DIR/camopanel"
TEMPLATES_TARGET="$DATA_DIR/templates"
SERVICE_UNIT=$(basename "$SERVICE_FILE")

[ -f "$BINARY_SOURCE" ] || fail "未找到二进制: $BINARY_SOURCE"
[ -d "$TEMPLATES_SOURCE" ] || fail "未找到模板目录: $TEMPLATES_SOURCE"
[ -f "$SERVICE_SOURCE" ] || fail "未找到 service 文件: $SERVICE_SOURCE"
[ -f "$ENV_EXAMPLE_SOURCE" ] || fail "未找到环境变量模板: $ENV_EXAMPLE_SOURCE"

ensure_runtime_account

mkdir -p "$BIN_DIR"
mkdir -p "$DATA_DIR"
mkdir -p "$TEMPLATES_TARGET"
mkdir -p "$DATA_DIR/openresty/conf.d"
mkdir -p "$DATA_DIR/openresty/www"
mkdir -p "$(dirname "$ENV_FILE")"
mkdir -p "$(dirname "$SERVICE_FILE")"

install -m 755 "$BINARY_SOURCE" "$BIN_PATH"
cp -R "$TEMPLATES_SOURCE"/. "$TEMPLATES_TARGET"/
render_service_file "$SERVICE_SOURCE" "$SERVICE_FILE" "$BIN_PATH"
render_env_example "$ENV_EXAMPLE_SOURCE" "${ENV_FILE}.example"

created_env=0
generated_admin_password=""

if [ ! -f "$ENV_FILE" ]; then
  created_env=1
  generated_admin_password=$(random_string 20)
  render_env_file "$ENV_EXAMPLE_SOURCE" "$ENV_FILE" "$(random_string 48)" "$generated_admin_password"
fi

if [ "$EUID" -eq 0 ]; then
  chown -R "$RUN_USER:$RUN_GROUP" "$DATA_DIR"
fi

if [ "$NO_START" -eq 0 ]; then
  need_cmd systemctl
  systemctl daemon-reload
  systemctl enable "$SERVICE_UNIT"

  if systemctl is-active --quiet "$SERVICE_UNIT"; then
    systemctl restart "$SERVICE_UNIT"
  else
    systemctl start "$SERVICE_UNIT"
  fi
fi

log "安装完成"
printf '  二进制: %s\n' "$BIN_PATH"
printf '  数据目录: %s\n' "$DATA_DIR"
printf '  模板目录: %s\n' "$TEMPLATES_TARGET"
printf '  环境文件: %s\n' "$ENV_FILE"
printf '  服务文件: %s\n' "$SERVICE_FILE"

if [ "$created_env" -eq 1 ]; then
  printf '  初始管理员: admin / %s\n' "$generated_admin_password"
fi

if [ "$NO_START" -eq 1 ]; then
  log "已跳过服务启动"
else
  log "服务已启动: $SERVICE_UNIT"
fi
