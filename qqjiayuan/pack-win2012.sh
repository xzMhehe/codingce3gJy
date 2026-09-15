#!/usr/bin/env bash
#
# pack-win2012.sh — 一键打包 Windows Server 2012 R2 部署产物
#
# 产物:WindowsServer2012bushu/{server,web,admin-web} + 同级 zip
#   server/server.exe   最新代码,官方 Go(>=1.26) 交叉编译
#   server/dbinit.exe   建库+建表+种子工具,同上
#   server/config.yaml  已存在则保留(不覆盖手动改动),不存在则生成模板
#   web/dist            全新构建(含 static 素材)
#   admin-web/dist      全新构建
# 文档(部署流程.txt 等)不会被脚本改动或删除。
#
# 用法: ./pack-win2012.sh
#
set -euo pipefail
cd "$(dirname "$0")"

# 服务器上的部署目录(正斜杠写法)。覆盖方式:./pack-win2012.sh "D:/qqjiayuan" 或 DEPLOY_DIR=D:/qqjiayuan ./pack-win2012.sh
# 只影响新生成的 config.yaml 模板;已存在的 config.yaml 一律保留不覆盖
DEPLOY_DIR="${DEPLOY_DIR:-C:/Users/Public/Code}"
[ $# -ge 1 ] && [ -n "$1" ] && DEPLOY_DIR="$1"

ROOT="$PWD"
PKG="$ROOT/WindowsServer2012bushu"
ZIP="$ROOT/WindowsServer2012bushu.zip"
NPMREG="${NPM_REGISTRY:-https://registry.npmmirror.com}"
ORIG_PATH="$PATH"

log()  { printf '\n============================================================\n  %s\n============================================================\n' "$*"; }
info() { printf '  [pack] %s\n' "$*"; }
die()  { printf '\n  [FATAL] %s\n\n' "$*"; exit 1; }

node_major() { "$1" -v 2>/dev/null | sed 's/^v//' | cut -d. -f1; }

find_node_new() {
  local best="" bestv=0 p v
  for p in "$HOME"/.nvm/versions/node/v*/bin/node \
           /opt/homebrew/opt/node*/bin/node \
           /usr/local/opt/node*/bin/node; do
    [ -x "$p" ] || continue
    v=$(node_major "$p") || continue
    [ -n "$v" ] || continue
    if [ "$v" -ge 14 ] && [ "$v" -gt "$bestv" ]; then best="$p"; bestv=$v; fi
  done
  printf '%s' "$best"
}

find_go() {
  local best="" bestv=-1 g v minor
  for g in "$(command -v go 2>/dev/null || true)" \
           /usr/local/go/bin/go /opt/homebrew/bin/go \
           "$HOME"/go-sdk/go*/bin/go \
           "$HOME"/.local/go-toolchain/go/bin/go; do
    [ -n "$g" ] && [ -x "$g" ] || continue
    v=$("$g" version 2>/dev/null | awk '{print $3}' | sed 's/^go//' | cut -d. -f1,2) || continue
    case "$v" in 1.*) ;; *) continue ;; esac
    minor="${v#1.}"
    case "$minor" in (*[!0-9]*|"") continue ;; esac
    if [ "$minor" -ge 26 ] && [ "$minor" -gt "$bestv" ]; then best="$g"; bestv=$minor; fi
  done
  printf '%s' "$best"
}

log "STEP 0/5 检查工具链"

CUR_NODE="$(command -v node 2>/dev/null || true)"
CUR_MAJOR=0
[ -n "$CUR_NODE" ] && CUR_MAJOR=$(node_major "$CUR_NODE") || true

NODE_BIN=""
if [ "$CUR_MAJOR" -ge 14 ] 2>/dev/null; then
  NODE_BIN="$CUR_NODE"
else
  NODE_BIN="$(find_node_new)"
fi
[ -n "$NODE_BIN" ] || die "找不到 Node >= 14(admin-web 的 html-webpack-plugin 需要 node: 前缀支持)。请安装 Node 16+ 或 nvm install 16"
NODE_DIR="$(dirname "$NODE_BIN")"
export PATH="$NODE_DIR:$ORIG_PATH"
info "node 使用 $(node -v) ($NODE_BIN)"
[ "$CUR_MAJOR" -ge 17 ] 2>/dev/null && export NODE_OPTIONS="--openssl-legacy-provider" || true

GO_BIN="$(find_go)"
[ -n "$GO_BIN" ] || die "找不到 Go >= 1.26(go.mod 要求)。已在这些位置查找:PATH、/usr/local/go、homebrew、~/go-sdk、~/.local/go-toolchain"
info "go 使用 $("$GO_BIN" version | awk '{print $2}') ($GO_BIN)"

command -v zip >/dev/null 2>&1 || info "警告: 未找到 zip 命令,最后将跳过打 zip 包"

log "STEP 1/5 构建用户端 web"

if [ ! -d "$ROOT/web/node_modules" ]; then
  info "node_modules 缺失,先 npm install(首次较慢)"
  ( cd "$ROOT/web" && npm install --registry="$NPMREG" )
fi
rm -rf "$ROOT/web/dist"
if ! ( cd "$ROOT/web" && npm run build ); then
  if [ "$CUR_MAJOR" -ge 1 ] && [ "$CUR_MAJOR" -lt 14 ] 2>/dev/null; then
    info "新版 node 构建失败,回退用原 node ($(node_major "$CUR_NODE")) 重试 web"
    export PATH="$ORIG_PATH"
    ( cd "$ROOT/web" && npm run build )
  else
    die "web 构建失败"
  fi
fi
[ -f "$ROOT/web/dist/index.html" ] || die "web/dist/index.html 未生成"
info "web/dist 构建完成"

log "STEP 2/5 构建管理端 admin-web"

if [ ! -d "$ROOT/admin-web/node_modules" ]; then
  info "node_modules 缺失,先 npm install(首次较慢)"
  ( cd "$ROOT/admin-web" && npm install --registry="$NPMREG" )
fi
rm -rf "$ROOT/admin-web/dist"
( cd "$ROOT/admin-web" && npm run build )
[ -f "$ROOT/admin-web/dist/index.html" ] || die "admin-web/dist/index.html 未生成"
info "admin-web/dist 构建完成"

log "STEP 3/5 交叉编译后端 server.exe / dbinit.exe"

export PATH="$(dirname "$GO_BIN"):$PATH"
export GOOS=windows GOARCH=amd64 CGO_ENABLED=0
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
mkdir -p "$PKG/server"
( cd "$ROOT/server" && "$GO_BIN" build -trimpath -ldflags "-s -w" -o "$PKG/server/server.exe" . )
( cd "$ROOT/server" && "$GO_BIN" build -trimpath -ldflags "-s -w" -o "$PKG/server/dbinit.exe" ./cmd/dbinit )
unset GOOS GOARCH CGO_ENABLED
[ -s "$PKG/server/server.exe" ] && [ -s "$PKG/server/dbinit.exe" ] || die "后端编译产物缺失"
rm -f "$PKG/server/"*-2012r2.exe
info "server.exe / dbinit.exe 编译完成"

log "STEP 4/5 组装部署包"

rm -rf "$PKG/web" "$PKG/admin-web"
mkdir -p "$PKG/web" "$PKG/admin-web"
cp -R "$ROOT/web/dist" "$PKG/web/dist"
cp -R "$ROOT/admin-web/dist" "$PKG/admin-web/dist"

if [ ! -f "$PKG/server/config.yaml" ]; then
  SECRET="$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')"
  cat > "$PKG/server/config.yaml" <<EOF
# 家园社区 Windows Server 2012 R2 部署配置
# 本包假定整个文件夹放在 ${DEPLOY_DIR%/}\ ,否则同步修改下面两个 web_dir(正斜杠)
server:
  port: 8080
  web_dir: "${DEPLOY_DIR%/}/web/dist"
  admin_web_dir: "${DEPLOY_DIR%/}/admin-web/dist"

mysql:
  host: 127.0.0.1
  port: 3306
  user: root
  password: "改成服务器MySQL的root密码"
  dbname: qq_jiayuan

jwt:
  secret: "$SECRET"
  expire_hours: 168
EOF
  info "已生成 server/config.yaml 模板(部署目录 ${DEPLOY_DIR%/} ,记得填 MySQL 密码)"
else
  info "保留已有 server/config.yaml(不覆盖手动改动)"
fi

[ -f "$PKG/部署流程.txt" ] || info "提示: $PKG/部署流程.txt 不存在,部署文档未打包"

log "STEP 5/5 打 zip 包与校验"

rm -f "$ZIP"
if command -v zip >/dev/null 2>&1; then
  ( cd "$ROOT" && zip -rq "$ZIP" WindowsServer2012bushu )
  info "zip: $ZIP ($(du -h "$ZIP" | cut -f1))"
fi

FAIL=0
for f in "$PKG/server/server.exe" "$PKG/server/dbinit.exe" "$PKG/server/config.yaml" \
         "$PKG/web/dist/index.html" "$PKG/web/dist/static" "$PKG/admin-web/dist/index.html"; do
  if [ -e "$f" ]; then info "OK  ${f#"$ROOT"/}"; else info "缺失 $f"; FAIL=1; fi
done
[ "$FAIL" -eq 0 ] || die "包内产物不完整,请检查上方输出"

info "文件数: $(find "$PKG" -type f | wc -l | tr -d ' ')  体积: $(du -sh "$PKG" | cut -f1)"

log "打包完成 → $PKG"
