#!/usr/bin/env bash
#
# pack-linxu.sh — 一键打包 Linux(amd64) 部署产物
#
# 产物:Linuxbushu/{server,web,admin-web} + 同级 Linuxbushu.tar.gz
#   server/server       最新代码,GOOS=linux GOARCH=amd64 CGO_ENABLED=0 静态编译
#   server/dbinit       建库+建表+种子工具,同上
#   server/config.yaml  已存在则保留(不覆盖手动改动),不存在则生成模板
#   web/dist            全新构建(含 static 素材)
#   admin-web/dist      全新构建
#
# 文档与辅助脚本(部署流程.txt、start.sh、stop.sh、init-db.sh、qqjiayuan.service)
# 不会被脚本改动或删除;脚本只会把其中的 .sh/.service/.txt 统一成 LF 换行,
# 避免在 Windows 上编辑后带 CRLF 传到 Linux 报 "bad interpreter"。
#
# 用法:
#   ./pack-linxu.sh                        # 部署目录默认 /opt/qqjiayuan
#   ./pack-linxu.sh /srv/qqjiayuan         # 指定服务器部署目录(只影响新生成的 config.yaml)
#   DEPLOY_DIR=/srv/qqjiayuan ./pack-linxu.sh
#
set -euo pipefail
cd "$(dirname "$0")"

DEPLOY_DIR="${DEPLOY_DIR:-/opt/qqjiayuan}"
[ $# -ge 1 ] && [ -n "$1" ] && DEPLOY_DIR="$1"

ROOT="$PWD"
PKG="$ROOT/Linuxbushu"
TARBALL="$ROOT/Linuxbushu.tar.gz"
ZIP="$ROOT/Linuxbushu.zip"
NPMREG="${NPM_REGISTRY:-https://registry.npmmirror.com}"
ORIG_PATH="$PATH"

log()  { printf '\n============================================================\n  %s\n============================================================\n' "$*"; }
info() { printf '  [pack] %s\n' "$*"; }
die()  { printf '\n  [FATAL] %s\n\n' "$*"; exit 1; }

node_major() { "$1" -v 2>/dev/null | sed 's/^v//' | cut -d. -f1; }

# ★ git bash / MSYS 上的路径陷阱(踩过,别删):
#   同一个路径有两种写法,两类程序各认一种,混用必然出错:
#     · 原生 Windows 程序(go.exe / node.exe)认 D:/xxx
#         —— 传 /d/xxx 给它,go 会当成"当前盘符下的相对路径",-o 把产物写到
#            D:\d\xxx 去,表现为 "go build 退出码 0 却找不到产物"
#     · MSYS 自带工具(tar / cp / find / rm ...)只认 /d/xxx
#         —— 传 D:/xxx 给 tar,它会当成远程主机语法 host:path,报
#            "Cannot connect to D:"
#   所以脚本内部一律用 POSIX 路径,只在调用原生程序时用 to_native 转一次。
to_native() {
  if command -v cygpath >/dev/null 2>&1; then
    cygpath -m "$1" 2>/dev/null || printf '%s' "$1"
  else
    printf '%s' "$1"   # 真正的 Linux 上原样返回
  fi
}

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
           /d/Go/bin/go /c/Go/bin/go \
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

# 构建前清理 dist。vue-cli 构建时本来就会清空输出目录,这里只是兜底;
# 某些受限环境(沙箱/CI)会拦截批量删除,删不掉不致命,交给构建自己清。
preclean() {
  [ -e "$1" ] || return 0
  rm -rf "$1" 2>/dev/null || info "预清理 ${1#"$ROOT"/} 被跳过(交给构建自行清空)"
  return 0
}

# 把包内文本文件统一成 LF(Windows 上编辑过会带 CRLF,传 Linux 会 bad interpreter)
normalize_lf() {
  local f n
  for f in "$PKG"/*.sh "$PKG"/*.service "$PKG"/*.txt; do
    [ -f "$f" ] || continue
    # 用 tr 数 CR 字节,不用 grep(某些 MSYS grep 对 \r 会误报)
    n=$(tr -cd '\r' < "$f" | wc -c | tr -d ' ')
    [ "${n:-0}" -gt 0 ] || continue
    sed -i 's/\r$//' "$f"
    info "CRLF → LF  ${f#"$ROOT"/}  (去掉 $n 个 CR)"
  done
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
[ -n "$GO_BIN" ] || die "找不到 Go >= 1.26(go.mod 要求)。已在这些位置查找:PATH、/usr/local/go、homebrew、D:/Go、C:/Go、~/go-sdk、~/.local/go-toolchain"
info "go 使用 $("$GO_BIN" version | awk '{print $3}') ($GO_BIN)"

command -v tar >/dev/null 2>&1 || info "警告: 未找到 tar 命令,最后将跳过打 tar.gz 包"
command -v zip >/dev/null 2>&1 || info "提示: 未找到 zip 命令,只出 tar.gz(Linux 上用 tar.gz 就够)"

log "STEP 1/5 构建用户端 web"

if [ ! -d "$ROOT/web/node_modules" ]; then
  info "node_modules 缺失,先 npm install(首次较慢)"
  ( cd "$ROOT/web" && npm install --registry="$NPMREG" )
fi
preclean "$ROOT/web/dist"
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
preclean "$ROOT/admin-web/dist"
( cd "$ROOT/admin-web" && npm run build )
[ -f "$ROOT/admin-web/dist/index.html" ] || die "admin-web/dist/index.html 未生成"
info "admin-web/dist 构建完成"

log "STEP 3/5 交叉编译后端 server / dbinit (linux/amd64)"

export PATH="$(dirname "$GO_BIN"):$PATH"
export GOOS=linux GOARCH=amd64 CGO_ENABLED=0
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
mkdir -p "$PKG/server"
# -o 的路径要转成原生形式(见 to_native 注释),否则 git bash 下产物会写飞
OUT_SERVER="$(to_native "$PKG/server/server")"
OUT_DBINIT="$(to_native "$PKG/server/dbinit")"
OUT_MIGRATE="$(to_native "$PKG/server/ezfymigrate")"
( cd "$ROOT/server" && "$GO_BIN" build -trimpath -ldflags "-s -w" -o "$OUT_SERVER" . )
( cd "$ROOT/server" && "$GO_BIN" build -trimpath -ldflags "-s -w" -o "$OUT_DBINIT" ./cmd/dbinit )
# ★ 二战风云城池批量迁移工具（运营用，不参与游戏运行）
#   用途：把所有玩家城池批量迁到指定大洲（内测时统一迁到欧洲，玩家离得近才打得起来）
( cd "$ROOT/server" && "$GO_BIN" build -trimpath -ldflags "-s -w" -o "$OUT_MIGRATE" ./cmd/ezfymigrate )
unset GOOS GOARCH CGO_ENABLED
[ -s "$PKG/server/server" ] && [ -s "$PKG/server/dbinit" ] && [ -s "$PKG/server/ezfymigrate" ] || die "后端编译产物缺失"

# 交叉编译出来的文件在 Windows 上没有可执行位,显式补上(tar 会记录)
chmod 755 "$PKG/server/server" "$PKG/server/dbinit" "$PKG/server/ezfymigrate" 2>/dev/null || true

# 确认真的是 Linux ELF,而不是误编成了 Windows exe
for b in "$PKG/server/server" "$PKG/server/dbinit" "$PKG/server/ezfymigrate"; do
  MAGIC="$(head -c 4 "$b" | od -An -tx1 | tr -d ' \n')"
  [ "$MAGIC" = "7f454c46" ] || die "$(basename "$b") 不是 Linux ELF(魔数 $MAGIC),检查 GOOS/GOARCH"
done
info "server / dbinit / ezfymigrate 编译完成(linux/amd64 ELF 校验通过)"

log "STEP 4/5 组装部署包"

preclean "$PKG/web"
preclean "$PKG/admin-web"
mkdir -p "$PKG/web/dist" "$PKG/admin-web/dist"
# 用 src/. 的形式拷贝:即使上面预清理没删成功、目标目录已存在,
# cp 也不会把它套成 dist/dist(受限环境删不掉目录时会踩这个坑)
cp -R "$ROOT/web/dist/." "$PKG/web/dist/"
cp -R "$ROOT/admin-web/dist/." "$PKG/admin-web/dist/"

if [ ! -f "$PKG/server/config.yaml" ]; then
  SECRET="$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')"
  cat > "$PKG/server/config.yaml" <<EOF
# 家园社区 Linux 部署配置
# 本包假定整个文件夹放在 ${DEPLOY_DIR%/} ,否则同步修改下面两个 web_dir(必须是绝对路径)
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

normalize_lf
[ -f "$PKG/部署流程.txt" ] || info "提示: $PKG/部署流程.txt 不存在,部署文档未打包"

log "STEP 5/5 打 tar.gz / zip 包与校验"

rm -f "$TARBALL" "$ZIP"
if command -v tar >/dev/null 2>&1; then
  ( cd "$ROOT" && tar -czf "$TARBALL" Linuxbushu )
  info "tar.gz: $TARBALL ($(du -h "$TARBALL" | cut -f1))"
  info "上传命令: scp \"$TARBALL\" root@你的服务器:/opt/"
  # 可执行位检查:在 Linux/macOS 上打包,chmod 生效,tar 会记 0755;
  # 在 Windows(git bash / MSYS)上文件系统没有 unix 权限位,无扩展名文件
  # chmod 不生效,tar 只能记 0644 —— 属正常现象,服务器解压后补一次 chmod 即可
  # (部署流程.txt 第一节已写明,start.sh / init-db.sh 内部也会自己补)。
  if tar -tvzf "$TARBALL" 2>/dev/null | grep -q '^-rwxr-xr-x.*Linuxbushu/server/server$'; then
    info "可执行位: 包内 server / dbinit 已带 0755,解压后可直接运行"
  else
    info "可执行位: 包内二进制为 0644(Windows 文件系统没有 unix 权限位,正常)"
    info "  → 服务器解压后先跑: chmod +x server/server server/dbinit *.sh"
    info "  → 或直接 ./init-db.sh 和 ./start.sh,脚本内部会自己补 chmod"
  fi
else
  info "跳过 tar.gz(无 tar 命令)"
fi
if command -v zip >/dev/null 2>&1; then
  ( cd "$ROOT" && zip -rq "$ZIP" Linuxbushu )
  info "zip: $ZIP ($(du -h "$ZIP" | cut -f1))"
fi

FAIL=0
for f in "$PKG/server/server" "$PKG/server/dbinit" "$PKG/server/ezfymigrate" "$PKG/server/config.yaml" \
         "$PKG/web/dist/index.html" "$PKG/web/dist/static" \
         "$PKG/admin-web/dist/index.html" "$PKG/start.sh" "$PKG/部署流程.txt"; do  if [ -e "$f" ]; then info "OK  ${f#"$ROOT"/}"; else info "缺失 $f"; FAIL=1; fi
done
[ "$FAIL" -eq 0 ] || die "包内产物不完整,请检查上方输出"

info "文件数: $(find "$PKG" -type f | wc -l | tr -d ' ')  体积: $(du -sh "$PKG" | cut -f1)"

log "打包完成 → $PKG"
printf '  服务器上最小步骤:\n'
printf '    tar -xzf Linuxbushu.tar.gz -C /opt && cd /opt/Linuxbushu\n'
printf '    mv /opt/Linuxbushu %s        # 非默认路径就跳过这步\n' "${DEPLOY_DIR%/}"
printf '    chmod +x server/server server/dbinit server/ezfymigrate *.sh\n'
printf '    vi server/config.yaml                       # 填 MySQL 密码\n'
printf '    ./init-db.sh && ./start.sh\n\n'
