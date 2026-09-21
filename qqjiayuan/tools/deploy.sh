#!/usr/bin/env bash
#
# deploy.sh —— 一键推送并部署「家园社区」到线上 Linux 服务器
#
# 用法:
#   ./tools/deploy.sh                     # 默认用 qqjiayuan/Linuxbushu.tar.gz 全量部署
#   ./tools/deploy.sh -f /path/pkg.tar.gz # 指定部署包
#   ./tools/deploy.sh --no-restart        # 只上传 + 校验, 不停服不解压(纯推包)
#   ./tools/deploy.sh --dry-run           # 只打印计划 + 预检, 不做任何改动
#   ./tools/deploy.sh --no-backup         # 跳过备份(省 55MB 空间, 不推荐)
#
# 密码: 优先读环境变量 SSH_PASS; 没设置就交互输入, 直接回车用 ssh-run.js 里的默认值
#   SSH_PASS='xxx' ./tools/deploy.sh
#
# 部署步骤(与线上既有约定一致):
#   0. 预检   本地包存在 / 本地 md5 / 远端连通 / 磁盘余量 / 当前服务状态
#   1. 备份   打包当前 /opt/Linuxbushu -> /opt/Linuxbushu.bak.<MMDD-HHMM>.tar.gz
#   2. 上传   scp 到 /opt/Linuxbushu.tar.gz, md5 双向校验
#             ★ 不一致立即中止, 绝不继续动线上
#   3. 停服   chmod +x /opt/Linuxbushu/*.sh && ./stop.sh
#             ★ 必须先停再解压, 否则覆盖运行中的二进制会报 "Text file busy"
#   4. 解压   tar -xzf ... --exclude='Linuxbushu/server/config.yaml'
#             ★ 包里的 server/config.yaml 是模板(占位密码 + 旧 web_dir), 必须排除
#   5. 恢复   把备份的真实 config.yaml 放回 server/, 并 chown root:root
#   6. 启动   chmod +x server/* && ./start.sh
#   7. 验证   8080 在监听 + curl 返回 200 + /api 返回 JSON(不是 index.html 回落)
#
# 回滚(在服务器上手动执行, 时间戳取 /opt 下最近一个 Linuxbushu.bak.*.tar.gz):
#   chmod +x /opt/Linuxbushu/*.sh; /opt/Linuxbushu/stop.sh
#   rm -rf /opt/Linuxbushu
#   tar -xzf /opt/Linuxbushu.bak.<时间戳>.tar.gz -C /opt
#   chmod +x /opt/Linuxbushu/server/*; /opt/Linuxbushu/start.sh
#

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SSH_RUN="$SCRIPT_DIR/ssh-run.js"

SSH_HOST="${SSH_HOST:-39.105.151.141}"
SSH_USER="${SSH_USER:-root}"
SSH_PASS="${SSH_PASS:-}"

PKG="$SCRIPT_DIR/../Linuxbushu.tar.gz"
REMOTE_DIR="/opt/Linuxbushu"
REMOTE_TGZ="/opt/Linuxbushu.tar.gz"
CONFIG_SEED="/opt/config.yaml"

NO_RESTART=0
NO_BACKUP=0
DRY_RUN=0

# ---------- 输出 ----------
C_RESET=$'\033[0m'; C_CYAN=$'\033[36m'; C_GREEN=$'\033[32m'
C_YELLOW=$'\033[33m'; C_RED=$'\033[31m'; C_GRAY=$'\033[90m'

log()  { printf '%s==>%s %s\n' "$C_CYAN" "$C_RESET" "$*"; }
ok()   { printf '%s  OK %s%s\n' "$C_GREEN" "$C_RESET" "$*"; }
warn() { printf '%s  !! %s%s\n' "$C_YELLOW" "$C_RESET" "$*"; }
step() { printf '%s  -- %s%s\n' "$C_GRAY" "$C_RESET" "$*"; }
die()  { printf '%s  XX %s%s\n' "$C_RED" "$C_RESET" "$*" >&2; exit 1; }

usage() {
  # 打印文件头部的注释块(第 2 行起, 到第一个非注释行为止) —— 别用硬编码行号, 改注释就会截断
  awk 'NR>1 && /^#/ { sub(/^# ?/, ""); print; next } NR>1 { exit }' "$0"
  exit 0
}

# ---------- 参数 ----------
while [ $# -gt 0 ]; do
  case "$1" in
    -f|--file)    PKG="${2:-}"; shift 2 ;;
    -h|--host)    SSH_HOST="${2:-}"; shift 2 ;;
    -u|--user)    SSH_USER="${2:-}"; shift 2 ;;
    --no-restart) NO_RESTART=1; shift ;;
    --no-backup)  NO_BACKUP=1; shift ;;
    --dry-run)    DRY_RUN=1; shift ;;
    --help)       usage ;;
    *) die "未知参数: $1  (用 --help 看用法)" ;;
  esac
done

# ---------- 远端执行 ----------
# rsh      —— 有副作用的命令, DRY_RUN 时只打印
# rsh_read —— 只读命令, DRY_RUN 时也真执行(预检需要真实结果)
rsh() {
  if [ "$DRY_RUN" = 1 ]; then
    printf '%s     [dry-run] ssh %s@%s: %s%s\n' "$C_GRAY" "$SSH_USER" "$SSH_HOST" "$1" "$C_RESET"
    return 0
  fi
  node "$SSH_RUN" exec "$1"
}

rsh_read() {
  node "$SSH_RUN" exec "$1" 2>/dev/null | tr -d '\r'
}

# ---------- 本地 md5(兼容 macOS / Linux) ----------
local_md5() {
  if command -v md5 >/dev/null 2>&1; then
    md5 -q "$1"
  else
    md5sum "$1" | awk '{print $1}'
  fi
}

local_size() { wc -c < "$1" | tr -d ' '; }

remote_has_dir() {
  [ "$(rsh_read "test -d '$REMOTE_DIR' && echo yes || echo no")" = "yes" ]
}

# =========================================================
log "部署目标  $SSH_USER@$SSH_HOST:$REMOTE_DIR"
[ "$DRY_RUN" = 1 ] && warn "DRY-RUN 模式: 只做预检, 不会改任何东西"

# ---------- 0. 预检 ----------
log "0/7 预检"

[ -n "$SSH_PASS" ] || {
  printf 'SSH 密码 (%s@%s, 直接回车用内置默认值): ' "$SSH_USER" "$SSH_HOST"
  read -r -s SSH_PASS; echo
}
export SSH_PASS

[ -f "$PKG" ] || die "部署包不存在: $PKG"
PKG="$(cd "$(dirname "$PKG")" && pwd)/$(basename "$PKG")"
LOCAL_MD5="$(local_md5 "$PKG")"
LOCAL_SIZE="$(local_size "$PKG")"
ok "部署包 $(basename "$PKG")  $(awk -v s="$LOCAL_SIZE" 'BEGIN{printf "%.1f", s/1048576}') MB  md5=$LOCAL_MD5"

rsh_read "echo ping" >/dev/null || die "SSH 连不上 $SSH_USER@$SSH_HOST (密码错? 端口不通?)"
ok "SSH 连通"

DISK_AVAIL="$(rsh_read "df -P /opt | tail -1 | awk '{print \$4}'")"
[ -n "$DISK_AVAIL" ] || die "读不到 /opt 磁盘余量"
AVAIL_MB=$(( ${DISK_AVAIL:-0} / 1024 ))
NEED_MB=$(( LOCAL_SIZE / 1048576 + 120 ))   # 包 + 备份 + 解压余量
if [ "$AVAIL_MB" -lt "$NEED_MB" ]; then
  die "/opt 剩余 ${AVAIL_MB}MB, 不够(需约 ${NEED_MB}MB)"
fi
ok "/opt 剩余 ${AVAIL_MB}MB (需约 ${NEED_MB}MB)"

if remote_has_dir; then
  OLD_PID="$(rsh_read "cat $REMOTE_DIR/logs/server.pid 2>/dev/null")"
  LISTENING="$(rsh_read "ss -lntp 2>/dev/null | grep -c ':8080 ' || true")"
  ok "线上已有部署, pid=${OLD_PID:-无}, 8080 监听数=${LISTENING:-0}"
else
  warn "线上没有 $REMOTE_DIR —— 这是全新部署(没有旧配置可保留)"
fi

if [ "$NO_RESTART" = 1 ]; then
  warn "--no-restart: 只上传 + 校验, 到此为止"
fi

[ "$DRY_RUN" = 1 ] && { log "DRY-RUN 结束, 下面是完整计划:"; }

# ---------- 1. 备份 ----------
TS="$(date +%m%d-%H%M)"
BAK_TGZ="/opt/Linuxbushu.bak.${TS}.tar.gz"
if [ "$NO_RESTART" = 0 ] && remote_has_dir; then
  log "1/7 备份线上现有部署"
  if [ "$NO_BACKUP" = 1 ]; then
    warn "--no-backup: 跳过整包备份, 只留 server/config.yaml"
    rsh "cp -a $REMOTE_DIR/server/config.yaml $REMOTE_DIR/server/config.yaml.bak.$TS" \
      || die "备份 config.yaml 失败"
  else
    # ★ 必须排除 logs/: 服务正在写 server.log, 不排除 tar 会报
    #   "file changed as we read it" 并以 1 退出(只是 warning, 但会被当成失败)
    rsh "tar -czf '$BAK_TGZ' --exclude='Linuxbushu/logs' -C /opt Linuxbushu; rc=\$?; \
         if [ \$rc -gt 1 ]; then exit \$rc; fi; [ -s '$BAK_TGZ' ] && ls -la '$BAK_TGZ'" \
      || die "备份失败, 已中止(线上未改动)"
    rsh "cp -a $REMOTE_DIR/server/config.yaml $REMOTE_DIR/server/config.yaml.bak.$TS" \
      || die "备份 config.yaml 失败"
    ok "备份: $BAK_TGZ (不含 logs/)"
  fi
  BAK_CONFIG="$REMOTE_DIR/server/config.yaml.bak.$TS"
else
  log "1/7 备份  (跳过)"
  BAK_CONFIG=""
fi

# ---------- 2. 上传 + 校验 ----------
log "2/7 上传部署包"
if [ "$DRY_RUN" = 1 ]; then
  step "[dry-run] scp $PKG -> $SSH_USER@$SSH_HOST:$REMOTE_TGZ"
else
  node "$SSH_RUN" put "$PKG" "$REMOTE_TGZ" || die "上传失败"
  REMOTE_MD5="$(rsh_read "md5sum '$REMOTE_TGZ' | awk '{print \$1}'")"
  REMOTE_SIZE="$(rsh_read "stat -c %s '$REMOTE_TGZ'")"
  [ "$REMOTE_MD5" = "$LOCAL_MD5" ] || die "md5 不一致! 本地=$LOCAL_MD5 远端=$REMOTE_MD5 (线上未改动)"
  [ "$REMOTE_SIZE" = "$LOCAL_SIZE" ] || die "字节数不一致! 本地=$LOCAL_SIZE 远端=$REMOTE_SIZE (线上未改动)"
  ok "md5 + 字节数双向校验一致 ($REMOTE_MD5)"
fi

if [ "$NO_RESTART" = 1 ]; then
  log "完成  --no-restart 模式: 包已就位, 线上服务未受影响"
  printf '    要完成部署, 在本地项目目录再跑一次(去掉 --no-restart):\n'
  printf '      %sSSH_PASS=... %s/tools/deploy.sh%s\n' "$C_GRAY" "$(cd "$SCRIPT_DIR/.." && pwd)" "$C_RESET"
  exit 0
fi

# ---------- 3. 停服 ----------
log "3/7 停服"
# ★ /opt/Linuxbushu/*.sh 首次部署后常常没有执行位, 不 chmod 会 Permission denied
#   却以为停成功了 —— 结果覆盖运行中的二进制报 Text file busy
rsh "chmod +x $REMOTE_DIR/*.sh 2>/dev/null; $REMOTE_DIR/stop.sh; sleep 1; ss -lntp 2>/dev/null | grep ':8080 ' || echo '8080 已释放'" \
  || warn "stop.sh 返回非 0(可能本来就没在跑), 继续"

# ---------- 4. 解压 ----------
log "4/7 解压 (排除模板 config.yaml)"
# ★ 解压会把 tar 里存的权限位盖回去 —— macOS 打包出来的 *.sh 常常是 644,
#   于是第 3 步刚 chmod 出来的执行位又被冲掉, 后面 ./start.sh 直接 Permission denied。
#   所以解压完必须再补一次 chmod(实测踩过)。
rsh "tar -xzf '$REMOTE_TGZ' --exclude='Linuxbushu/server/config.yaml' -C /opt && echo 'extract ok'" \
  || die "解压失败 —— 线上现在没在跑, 用备份回滚: tar -xzf $BAK_TGZ -C /opt"
rsh "chmod +x $REMOTE_DIR/*.sh && echo 'chmod sh ok'" || die "chmod *.sh 失败"
ok "解压完成 + 补回 *.sh 执行位"

# ---------- 5. 恢复配置 ----------
log "5/7 恢复真实配置"
if [ -n "$BAK_CONFIG" ]; then
  rsh "cp -a '$BAK_CONFIG' $REMOTE_DIR/server/config.yaml && echo 'restored from backup'" \
    || die "恢复 config.yaml 失败"
  ok "已用备份的真实配置覆盖模板"
elif [ "$(rsh_read "test -f '$CONFIG_SEED' && echo yes || echo no")" = "yes" ]; then
  rsh "cp -a '$CONFIG_SEED' $REMOTE_DIR/server/config.yaml" || die "恢复 config.yaml 失败"
  ok "已用 $CONFIG_SEED 覆盖模板"
else
  warn "找不到真实配置! 包里的是模板(占位密码 + 旧 web_dir), 服务会起不来"
  warn "请先准备 $REMOTE_DIR/server/config.yaml, 再执行 ./start.sh"
fi
rsh "chown -R root:root $REMOTE_DIR" || warn "chown 失败(macOS 打包的 uid 501), 不影响启动"

# ---------- 6. 启动 ----------
log "6/7 启动"
rsh "chmod +x $REMOTE_DIR/*.sh $REMOTE_DIR/server/server $REMOTE_DIR/server/dbinit $REMOTE_DIR/server/ezfymigrate 2>/dev/null; cd $REMOTE_DIR && ./start.sh" \
  || die "启动脚本失败, 看日志: $REMOTE_DIR/logs/server.log"

# ---------- 7. 验证 ----------
log "7/7 验证"
sleep 6
RESULT="$(rsh_read "
  ss -lntp 2>/dev/null | grep ':8080 ' >/dev/null && echo LISTEN_OK || echo LISTEN_FAIL
  curl -s -o /dev/null -w 'HTTP=%{http_code}' --max-time 8 http://127.0.0.1:8080/ || echo 'HTTP=000'
  echo
  curl -s --max-time 8 http://127.0.0.1:8080/api/games/ezfy/view | head -c 60 || true
  echo
")"

echo "$RESULT" | grep -q LISTEN_OK  || { rsh_read "tail -n 25 $REMOTE_DIR/logs/server.log"; die "8080 没在监听, 日志见上"; }
ok "8080 已监听"
echo "$RESULT" | grep -q 'HTTP=200' || { rsh_read "tail -n 25 $REMOTE_DIR/logs/server.log"; die "首页不是 200"; }
ok "首页 HTTP 200"

# 版本指纹: 未知路径会回落 index.html 返 200, 所以必须看响应体是不是 JSON
if echo "$RESULT" | grep -q '"code"'; then
  ok "接口返回 JSON —— 新版本已生效"
else
  warn "接口返回的不是 JSON(可能是 index.html 回落) —— 确认下版本"
  echo "$RESULT" | tail -2
fi

rsh_read "tail -n 8 $REMOTE_DIR/logs/server.log"

printf '\n%s部署完成%s  http://%s:8080\n' "$C_GREEN" "$C_RESET" "$SSH_HOST"
printf '  备份: %s\n' "${BAK_TGZ:-无}"
printf '  回滚: tar -xzf %s -C /opt\n' "${BAK_TGZ:-<备份包>}"
