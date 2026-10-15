#!/usr/bin/env bash
set -Eeuo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
VERSION=${1:-}
[[ $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]] || {
  printf '用法：bash scripts/package-release.sh vX.Y.Z [amd64|arm64 ...]\n' >&2
  exit 1
}
shift
if (($# == 0)); then set -- amd64 arm64; fi
for arch in "$@"; do
  [[ $arch == amd64 || $arch == arm64 ]] || { printf '不支持的架构：%s\n' "$arch" >&2; exit 1; }
done

cd "$ROOT"
bun run --cwd frontend build
mkdir -p "$ROOT/dist"
stage=$(mktemp -d)
trap 'rm -rf -- "$stage"' EXIT
printf '%s\n' "$VERSION" > "$stage/VERSION"
cp -R frontend/dist "$stage/web"
cp install.sh "$stage/install.sh"
cp README.md "$stage/README.md"
assets=()
for arch in "$@"; do
  printf '构建 linux/%s %s\n' "$arch" "$VERSION"
  (cd backend; CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build \
    -buildvcs=false -trimpath -ldflags='-s -w' -o "$stage/camopanel" ./cmd/server)
  asset="camopanel_linux_${arch}.tar.gz"
  tar -czf "$ROOT/dist/$asset" -C "$stage" camopanel web VERSION install.sh README.md
  assets+=("$asset")
done
cp install.sh "$ROOT/dist/install.sh"
(cd "$ROOT/dist"; sha256sum "${assets[@]}" install.sh > checksums.txt)
printf '已生成发布文件：%s/dist/\n' "$ROOT"
