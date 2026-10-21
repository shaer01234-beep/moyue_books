#!/usr/bin/env bash
# =============================================================================
# 将本项目打包为飞牛 fnOS 应用 (.fpk)
#
# 用法:
#   ./fnos/build.sh                # 构建 arm64 应用目录（默认，manifest 为 arm）
#   ./fnos/build.sh amd64          # 构建 x86_64 应用目录
#   ./fnos/build.sh armv7          # 构建 32 位 armv7 应用目录（老款 ARM 设备）
#   ./fnos/build.sh --pack         # 额外生成 .fpk（有 fnpack 用官方工具，否则用内置打包器）
#   ./fnos/build.sh --skip-web     # 跳过前端构建（沿用已有 frontend/dist）
#
# 产物目录: fnos/moyue/
#   app/bin/moyue        Go 单二进制（前端已内嵌，含 HTTP + Unix Socket 网关）
#   app/ui               桌面入口配置（.url + 图标）
#   cmd/ config/ wizard/ 飞牛生命周期钩子
#   manifest             应用清单（platform 会按架构自动同步）
#
# 说明：
#   - 前端产物通过 //go:embed 内嵌进二进制，故包内不再单独附带 web/dist
#   - 网关前缀固定 /app/moyue（见 app/ui/config 与 cmd/main），打包时注入 VITE_BASE
#   - SQLite 使用纯 Go 驱动（glebarez/sqlite），无需 CGO，可静态交叉编译
# =============================================================================
set -euo pipefail

ROOT="$(cd -- "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_DIR="$ROOT/fnos/moyue"
PACK_DIR="$APP_DIR/app"

# 网关前缀（需与 app/ui/config 的 gatewayPrefix 一致）
GATEWAY_PREFIX="/app/moyue"

ARCH_IN="arm"
DO_PACK=0
SKIP_WEB=0
for a in "$@"; do
  case "$a" in
    amd64|x86_64|x86) ARCH_IN="amd64" ;;
    arm64|aarch64|arm) ARCH_IN="arm64" ;;
    armv7|armhf) ARCH_IN="armv7" ;;
    --pack) DO_PACK=1 ;;
    --skip-web) SKIP_WEB=1 ;;
    -h|--help) sed -n '2,25p' "${BASH_SOURCE[0]}"; exit 0 ;;
    *) echo "未知参数: $a（支持 amd64 / arm64 / armv7 / --pack / --skip-web）" >&2; exit 1 ;;
  esac
done

GOARM=""
case "$ARCH_IN" in
  amd64|x86) GOARCH=amd64; PLATFORM=x86 ;;
  arm64|aarch64|arm) GOARCH=arm64; PLATFORM=arm ;;
  armv7|armhf) GOARCH=arm; GOARM=7; PLATFORM=arm ;;
  *) echo "未知架构: ${ARCH_IN}（支持 amd64 / arm64 / armv7）" >&2; exit 1 ;;
esac

CYAN='\033[1;36m'; GREEN='\033[1;32m'; YELLOW='\033[1;33m'; OFF='\033[0m'
info(){ echo -e "${CYAN}[fnos]${OFF} $*"; }
ok(){ echo -e "${GREEN}[ok]  ${OFF} $*"; }
warn(){ echo -e "${YELLOW}[warn]${OFF} $*"; }

# ---------------------------------------------------------------------------
# 0) 同步 fnOS 包版本（manifest.version）到前端 APP_VERSION（界面展示用）
#    必须在前端构建之前执行，保证打进前端的版本号与 fpk 一致
# ---------------------------------------------------------------------------
MANIFEST_VERSION="$(grep '^version' "$APP_DIR/manifest" | awk -F'=' '{print $2}' | tr -d ' \r')"
VERSION_JS="$ROOT/frontend/src/utils/version.js"
if [ -n "$MANIFEST_VERSION" ] && [ -f "$VERSION_JS" ]; then
  sed -i.bak -E "s|^export const APP_VERSION = .*|export const APP_VERSION = '${MANIFEST_VERSION}'|" \
    "$VERSION_JS" && rm -f "$VERSION_JS.bak"
  ok "同步前端 APP_VERSION = ${MANIFEST_VERSION}"
else
  warn "未找到版本文件或未读取到 manifest version，跳过 APP_VERSION 同步"
fi

# ---------------------------------------------------------------------------
# 1) 构建前端（注入网关前缀）
# ---------------------------------------------------------------------------
if [ "$SKIP_WEB" = 0 ]; then
  if [ ! -f "$ROOT/frontend/node_modules/.bin/vite" ]; then
    info "安装前端依赖 npm install ..."
    (cd "$ROOT/frontend" && npm install --silent --no-audit --no-fund)
  fi
  info "构建前端（网关前缀 ${GATEWAY_PREFIX}）..."
  # VITE_BASE：index.html 资源引用前缀（兼容无尾斜杠访问）
  # VITE_API_BASE：前端请求 /app/moyue/api/...，由网关反代到应用
  (cd "$ROOT/frontend" && VITE_BASE="${GATEWAY_PREFIX}/" VITE_API_BASE="${GATEWAY_PREFIX}/api" npm run build) \
    > /tmp/fnos_web_build.log 2>&1 \
    || { echo "前端构建失败，详见 /tmp/fnos_web_build.log" >&2; tail -20 /tmp/fnos_web_build.log; exit 1; }
  ok "前端打包完成"
fi

# ---------------------------------------------------------------------------
# 2) 同步前端产物到 backend/html/dist（由 //go:embed 内嵌进二进制）
# ---------------------------------------------------------------------------
info "同步前端产物到 backend/html/dist ..."
rm -rf "$ROOT/backend/html/dist"
mkdir -p "$ROOT/backend/html"
cp -R "$ROOT/frontend/dist" "$ROOT/backend/html/dist"
ok "前端已就位（将内嵌进二进制）"

# ---------------------------------------------------------------------------
# 3) 交叉编译 Go 服务（Linux 静态二进制）
# ---------------------------------------------------------------------------
info "编译 Go 服务 (linux/${GOARCH}${GOARM:+ (GOARM=$GOARM)}) ..."
mkdir -p "$PACK_DIR/bin"
# 注意：赋值前缀必须是字面量，不能由变量展开得到，因此 armv7 单独分支
if [ -n "$GOARM" ]; then
  (cd "$ROOT/backend" && CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" GOARM="$GOARM" \
    go build -trimpath -ldflags "-s -w" -o "$PACK_DIR/bin/moyue" .)
else
  (cd "$ROOT/backend" && CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" \
    go build -trimpath -ldflags "-s -w" -o "$PACK_DIR/bin/moyue" .)
fi
ok "二进制: fnos/moyue/app/bin/moyue"

# ---------------------------------------------------------------------------
# 4) 同步 manifest 平台字段
# ---------------------------------------------------------------------------
sed -i.bak -e "s|^platform .*|platform              = $PLATFORM|" "$APP_DIR/manifest" && rm -f "$APP_DIR/manifest.bak"
ok "manifest platform=$PLATFORM"

# ---------------------------------------------------------------------------
# 5) 打包 .fpk：优先官方 fnpack，未安装则用内置打包器（fpk = tar.gz）
#    包结构：app.tgz + cmd/ + config/ + wizard/ + ui/ + manifest(含 checksum) + ICON*
# ---------------------------------------------------------------------------
manifest_val() { grep "^$1" "$APP_DIR/manifest" | awk -F'=' '{print $2}' | tr -d ' \r'; }

pack_fpk() {
  local appname version platform work pkg sum name
  appname="$(manifest_val appname)"
  version="$(manifest_val version)"
  platform="$(manifest_val platform)"
  if [ -z "$appname" ] || [ -z "$version" ]; then
    warn "manifest 缺少 appname/version，无法打包"
    return 1
  fi

  work="$(mktemp -d)"
  pkg="$work/package"
  mkdir -p "$pkg"

  # app.tgz：payload 直接放在包根（安装后即 TRIM_APPDEST 内容）
  # 用显式条目打包，避免产生 "./" 前缀
  (cd "$APP_DIR/app" && tar -czf "$pkg/app.tgz" $(ls -A))
  cp -a "$APP_DIR/cmd" "$APP_DIR/config" "$pkg/"
  [ -d "$APP_DIR/wizard" ] && cp -a "$APP_DIR/wizard" "$pkg/"
  # 入口目录：官方要求 app/ui/config（已随 app.tgz 安装），同时在包根放一份兼容旧实现
  [ -d "$APP_DIR/app/ui" ] && cp -a "$APP_DIR/app/ui" "$pkg/"
  [ -d "$APP_DIR/ui" ] && cp -a "$APP_DIR/ui" "$pkg/"
  cp "$APP_DIR/ICON.PNG" "$APP_DIR/ICON_256.PNG" "$pkg/"
  cp "$APP_DIR/manifest" "$pkg/manifest"

  # app.tgz 的 md5 写入 manifest（与官方 build-fpk.sh 一致）
  if command -v md5sum >/dev/null 2>&1; then
    sum="$(md5sum "$pkg/app.tgz" | cut -d' ' -f1)"
  else
    sum="$(md5 -q "$pkg/app.tgz")"
  fi
  if grep -q "^checksum" "$pkg/manifest"; then
    sed -i.bak "s|^checksum.*|checksum        = ${sum}|" "$pkg/manifest"
    rm -f "$pkg/manifest.bak"
  else
    printf 'checksum        = %s\n' "$sum" >> "$pkg/manifest"
  fi

  name="${appname}_${version}_${platform:-x86}.fpk"
  (cd "$pkg" && tar -czf "$APP_DIR/$name" *)
  rm -rf "$work"
  ok "已生成 fnos/moyue/${name}"
}

if [ "$DO_PACK" = 1 ]; then
  if command -v fnpack >/dev/null 2>&1; then
    info "使用官方 fnpack 打包 ..."
    (cd "$APP_DIR" && fnpack build) && ok "fnpack 打包完成"
  else
    warn "未检测到 fnpack，改用内置打包器生成 .fpk"
    pack_fpk
  fi
fi

echo
info "============================================================"
info "  fnOS 应用目录就绪: fnos/moyue"
info "  生成 .fpk: ./fnos/build.sh ${ARCH_IN} --pack （或用官方 fnpack build -d fnos/moyue）"
info "  然后在飞牛 fnOS 应用中心手动安装生成的 .fpk"
info "  默认端口: 18080（网关入口 /app/moyue，也可直接用 IP:18080 访问）"
info "============================================================"
