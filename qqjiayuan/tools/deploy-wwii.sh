#!/usr/bin/env bash
#
# deploy-wwii.sh — 二战征途独立服一键部署器（Mac / Linux 用，Windows 用 deploy-wwii.bat）
#
# 部署 WWIIbushu.tar.gz → 远程 /opt/WWIIbushu（二战 1 区，端口 8081）。
# 全程不碰家园服 → 改二战内容不用重启家园，玩家无感知。
# --zone2：克隆并启动 2 区（/opt/wwii-2，端口 8082，独立库 qq_ezzt2，bootstrap=fresh）。
#
# 密码：优先环境变量 SSH_PASS；否则交互输入（直接回车用 ssh-run.js 默认值）。
#   SSH_PASS='密码' ./tools/deploy-wwii.sh -h 124.222.208.92 --no-backup
#
# 用法（仓库根目录 qqjiayuan 下执行）：
#   ./tools/deploy-wwii.sh -p '密码' -h 124.222.208.92 --no-backup
#   ./tools/deploy-wwii.sh -p '密码' -h 124.222.208.92 --zone2 --no-backup
#
set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SSH_RUN="$SCRIPT_DIR/ssh-run.js"
PROJ_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

SSH_HOST="${SSH_HOST:-39.105.151.141}"
SSH_USER="${SSH_USER:-root}"
PKG="${PKG:-$PROJ_DIR/WWIIbushu.tar.gz}"
REMOTE_DIR="/opt/WWIIbushu"
REMOTE_TGZ="/opt/WWIIbushu.tar.gz"
CONFIG_SEED="/opt/config.yaml.wwii"
WWII_PORT=8081
ZONE2_DIR="/opt/wwii-2"
ZONE2_PORT=8082

NO_RESTART=0
NO_BACKUP=0
DRY_RUN=0
ZONE2=0

step()  { printf '\n============================================================\n  %s\n============================================================\n' "$*"; }
info() { printf '  [wwii] %s\n' "$*"; }
die()  { printf '\n  [FATAL] %s\n\n' "$*"; exit 1; }

# 参数解析
while [ $# -gt 0 ]; do
  case "$1" in
    -f|--file)       PKG="$2"; shift 2 ;;
    -h|--host)       SSH_HOST="$2"; shift 2 ;;
    -u|--user)       SSH_USER="$2"; shift 2 ;;
    -p|--pass)       SSH_PASS="$2"; shift 2 ;;
    --no-restart)    NO_RESTART=1; shift ;;
    --no-backup)     NO_BACKUP=1; shift ;;
    --dry-run)       DRY_RUN=1; shift ;;
    --zone2)         ZONE2=1; shift ;;
    --help|-?)       echo "用法: $0 [-f 包] [-h 主机] [-u 用户] [-p 密码] [--no-restart] [--no-backup] [--dry-run] [--zone2]"; exit 0 ;;
    *) die "未知参数: $1 （--help 查看用法）" ;;
  esac
done

export SSH_PASS SSH_HOST SSH_USER

[ -f "$SSH_RUN" ] || die "缺少 $SSH_RUN"
command -v node >/dev/null 2>&1 || die "找不到 node，请安装 Node.js"

echo "==> 目标  $SSH_USER@$SSH_HOST:$REMOTE_DIR"
[ "$DRY_RUN" = "1" ] && info "WARN: DRY-RUN，只打印计划，不执行任何操作"

step "0/7 预检"
[ -z "${SSH_PASS:-}" ] && { printf 'SSH 密码 (%s@%s): ' "$SSH_USER" "$SSH_HOST"; read -r -s SSH_PASS; echo; }
export SSH_PASS

[ -f "$PKG" ] || die "找不到包: $PKG （先跑 pack-linxu.sh / pack-linxu.bat）"
LOCAL_SIZE=$(stat -f%z "$PKG" 2>/dev/null || stat -c%s "$PKG" 2>/dev/null)
LOCAL_MD5=$(md5 -q "$PKG" 2>/dev/null || md5sum "$PKG" | awk '{print $1}')
[ -n "$LOCAL_MD5" ] || die "算 md5 失败"
info "包 $PKG 约 $((LOCAL_SIZE/1048576)) MB  md5=$LOCAL_MD5"

rsh_read() { node "$SSH_RUN" exec "$1" 2>/dev/null | tr -d '\r'; }

[ "$(rsh_read "echo ping")" = "ping" ] || die "SSH 连不上 $SSH_USER@$SSH_HOST"
info "SSH 通"

DISK_AVAIL=$(rsh_read "df -P /opt | tail -1 | awk '{print \$4}'")
[ -n "$DISK_AVAIL" ] || die "读不到 /opt 剩余空间"
AVAIL_MB=$((DISK_AVAIL/1024)); NEED_MB=$((LOCAL_SIZE/1048576+120))
[ "$AVAIL_MB" -ge "$NEED_MB" ] || die "/opt 剩余 ${AVAIL_MB}MB，需要约 ${NEED_MB}MB"
info "/opt 剩余 ${AVAIL_MB}MB"

HAS_DIR=$(rsh_read "test -d '$REMOTE_DIR' && echo yes || echo no")
if [ "$HAS_DIR" = "yes" ]; then
  info "远程已有 $REMOTE_DIR（二战 ${WWII_PORT} 监听数: $(rsh_read "ss -lntp 2>/dev/null | grep -c ':$WWII_PORT ' || true")）"
else
  info "WARN: 远程无 $REMOTE_DIR —— 全新部署（没有可备份的旧配置）"
fi

[ "$DRY_RUN" = "1" ] && { echo "==> DRY-RUN 到此为止"; exit 0; }

step "1/7 备份线上配置"
BAK_CONFIG=""
TS="$RANDOM$RANDOM"
if [ "$NO_RESTART" = "0" ] && [ "$HAS_DIR" = "yes" ]; then
  rsh() { node "$SSH_RUN" exec "$1" || return $?; }
  rsh "cp -a $REMOTE_DIR/server/config.yaml $REMOTE_DIR/server/config.yaml.bak.$TS" || die "备份 config.yaml 失败"
  BAK_CONFIG="$REMOTE_DIR/server/config.yaml.bak.$TS"
  info "已备份: $BAK_CONFIG"
else
  info "跳过备份（全新 / --no-backup）"
fi

step "2/7 上传"
[ "$DRY_RUN" = "1" ] && { echo "      dry-run: scp $PKG -> $SSH_USER@$SSH_HOST:$REMOTE_TGZ"; exit 0; }
node "$SSH_RUN" put "$PKG" "$REMOTE_TGZ" || die "上传失败"
REMOTE_MD5=$(rsh_read "md5sum '$REMOTE_TGZ' | awk '{print \$1}'")
REMOTE_SIZE=$(rsh_read "wc -c < '$REMOTE_TGZ'")
[ "$REMOTE_MD5" = "$LOCAL_MD5" ] || die "md5 不一致 local=$LOCAL_MD5 remote=$REMOTE_MD5"
[ "$REMOTE_SIZE" = "$LOCAL_SIZE" ] || die "字节数不一致 local=$LOCAL_SIZE remote=$REMOTE_SIZE"
info "md5 + 字节数校验一致"

if [ "$NO_RESTART" = "1" ]; then
  echo "==> 完成（--no-restart：包已就位，服务未动）"
  exit 0
fi

step "3/7 停服"
node "$SSH_RUN" exec "chmod +x $REMOTE_DIR/*.sh 2>/dev/null; $REMOTE_DIR/stop.sh 2>/dev/null; sleep 1; ss -lntp 2>/dev/null | grep ':$WWII_PORT ' || echo '$WWII_PORT free'" \
  || info "WARN: stop.sh 返回非 0（可能从未启动过），继续"

step "4/7 解压（排除 server/config.yaml）"
node "$SSH_RUN" exec "tar -xzf '$REMOTE_TGZ' --exclude='WWIIbushu/server/config.yaml' -C /opt && echo 'extract ok'" \
  || die "解压失败"
node "$SSH_RUN" exec "chmod +x $REMOTE_DIR/*.sh $REMOTE_DIR/server/server && echo 'chmod ok'" || die "chmod 失败"
info "解压完成 + 执行位已设置"

step "5/7 恢复线上配置"
if [ -n "$BAK_CONFIG" ]; then
  node "$SSH_RUN" exec "cp -a '$BAK_CONFIG' $REMOTE_DIR/server/config.yaml && echo restored" || die "恢复备份配置失败"
  info "已从备份恢复配置"
elif [ "$(rsh_read "test -f '$CONFIG_SEED' && echo yes || echo no")" = "yes" ]; then
  node "$SSH_RUN" exec "cp -a '$CONFIG_SEED' $REMOTE_DIR/server/config.yaml" || die "恢复种子配置失败"
  info "已从 $CONFIG_SEED 恢复配置"
else
  # 无备份/种子 → 模板就位。上传 wwii-patch-cfg.sh，从服务器上家园配置
  # /opt/Linuxbushu/server/config.yaml（退路 /opt/config.yaml）回填 MySQL host/密码，
  # 并写入 home_url（家园服公网地址 http://<主机IP>:<家园端口>）供二战「家园」按钮整页跳转。
  node "$SSH_RUN" put "$SCRIPT_DIR/wwii-patch-cfg.sh" "$REMOTE_DIR/wwii-patch-cfg.sh" || die "上传补丁脚本失败"
  HOME_PORT=$(rsh_read "sed -n 's/^[[:space:]]*port:[[:space:]]*//p' /opt/Linuxbushu/server/config.yaml 2>/dev/null | head -n1 | tr -d '\"' | tr -d '\r'")
  [ -n "$HOME_PORT" ] || HOME_PORT=8080
  node "$SSH_RUN" exec "bash $REMOTE_DIR/wwii-patch-cfg.sh $REMOTE_DIR/server/config.yaml /opt/Linuxbushu/server/config.yaml http://$SSH_HOST:$HOME_PORT" || die "回填 WWII 配置失败"
  info "已从家园配置回填 MySQL host/密码 + home_url（http://$SSH_HOST:$HOME_PORT）"
fi

step "6/7 启动"
node "$SSH_RUN" exec "cd $REMOTE_DIR && ./start.sh" || {
  info "启动失败，日志尾部："
  node "$SSH_RUN" exec "tail -n 25 $REMOTE_DIR/logs/server.log" 2>/dev/null || true
  die "启动失败"
}

step "7/7 验证"
sleep 6
LISTEN=$(rsh_read "ss -lntp 2>/dev/null | grep ':$WWII_PORT ' >/dev/null && echo LISTEN_OK || echo LISTEN_FAIL")
[ "$LISTEN" = "LISTEN_OK" ] || { info "二战 ${WWII_PORT} 未监听，日志尾部："; node "$SSH_RUN" exec "tail -n 25 $REMOTE_DIR/logs/server.log" 2>/dev/null || true; die "二战 ${WWII_PORT} 未监听"; }
info "二战 ${WWII_PORT} 已监听"
ZONE_JSON=$(rsh_read "curl -s --max-time 8 http://127.0.0.1:$WWII_PORT/api/ezfy/zone | head -c 80")
case "$ZONE_JSON" in *code*) info "zone 接口返回 JSON: $ZONE_JSON" ;; *) info "WARN: zone 接口暂无 JSON（可能还在灌种子），稍后自查" ;; esac

if [ "$ZONE2" = "1" ]; then
  echo "==> --zone2: 克隆 2 区（$ZONE2_DIR，端口 $ZONE2_PORT，独立库 qq_ezzt2，fresh）"
  if [ "$(rsh_read "test -d '$ZONE2_DIR' && echo yes || echo no")" = "yes" ]; then
    info "$ZONE2_DIR 已存在 —— 同步新二进制/web/脚本（保留其 config.yaml），重启 2 区"
    node "$SSH_RUN" exec "cp -f $REMOTE_DIR/server/server $ZONE2_DIR/server/server && cp -R $REMOTE_DIR/web/dist/. $ZONE2_DIR/web/dist/ && cp -f $REMOTE_DIR/start.sh $REMOTE_DIR/stop.sh $ZONE2_DIR/ && chmod +x $ZONE2_DIR/*.sh $ZONE2_DIR/server/server && $ZONE2_DIR/stop.sh 2>/dev/null; sleep 1; $ZONE2_DIR/start.sh" || {
      info "2 区同步/重启失败，日志尾部："
      node "$SSH_RUN" exec "tail -n 15 $ZONE2_DIR/logs/server.log 2>/dev/null" || true
      exit 1
    }
    sleep 6
    if [ "$(rsh_read "ss -lntp 2>/dev/null | grep ':$ZONE2_PORT ' >/dev/null && echo ZONE2_OK || echo ZONE2_FAIL")" = "ZONE2_OK" ]; then
      info "2 区 $ZONE2_PORT 已监听（已更新）"
    else
      info "WARN: 2 区 $ZONE2_PORT 还没监听 —— 看 $ZONE2_DIR/logs/server.log"
    fi
  else
    node "$SSH_RUN" exec "cd $REMOTE_DIR && bash ./zone2.sh && chmod +x $ZONE2_DIR/*.sh $ZONE2_DIR/server/server && $ZONE2_DIR/start.sh" || {
      info "2 区克隆/启动失败，日志尾部："
      node "$SSH_RUN" exec "tail -n 15 $ZONE2_DIR/logs/server.log 2>/dev/null" || true
      exit 1
    }
    sleep 6
    if [ "$(rsh_read "ss -lntp 2>/dev/null | grep ':$ZONE2_PORT ' >/dev/null && echo ZONE2_OK || echo ZONE2_FAIL")" = "ZONE2_OK" ]; then
      info "2 区 $ZONE2_PORT 已监听"
    else
      info "WARN: 2 区 $ZONE2_PORT 还没监听 —— 看 $ZONE2_DIR/logs/server.log"
    fi
  fi
fi

echo
echo "  二战已部署:  http://$SSH_HOST:$WWII_PORT  （1 区）"
[ "$ZONE2" = "1" ] && echo "  2 区:         http://$SSH_HOST:$ZONE2_PORT"
echo "  下一步: 家园管理端『游戏大厅』给二战加区服条目 host=http://$SSH_HOST:$WWII_PORT"
