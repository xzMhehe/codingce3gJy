#!/usr/bin/env bash
#
# 家园社区 — Linux 后台启动脚本
#   用法: ./start.sh
#   日志: ../logs/server.log    pid: ../logs/server.pid
#   停止: ./stop.sh
#
# 生产环境推荐用 systemd(见 部署流程.txt 第八节),此时不要再跑本脚本。
#
set -u

BASE="$(cd "$(dirname "$0")" && pwd)"
SERVER_DIR="$BASE/server"
LOG_DIR="$BASE/logs"
PIDFILE="$LOG_DIR/server.pid"
LOGFILE="$LOG_DIR/server.log"

if [ ! -x "$SERVER_DIR/server" ]; then
  chmod +x "$SERVER_DIR/server" "$SERVER_DIR/dbinit" 2>/dev/null || true
fi
if [ ! -f "$SERVER_DIR/server" ]; then
  echo "找不到 $SERVER_DIR/server ,请确认包已完整解压。" >&2
  exit 1
fi
if [ ! -f "$SERVER_DIR/config.yaml" ]; then
  echo "找不到 $SERVER_DIR/config.yaml ,请先配置好再启动。" >&2
  exit 1
fi

mkdir -p "$LOG_DIR"

if [ -f "$PIDFILE" ]; then
  OLD="$(cat "$PIDFILE" 2>/dev/null || true)"
  if [ -n "$OLD" ] && kill -0 "$OLD" 2>/dev/null; then
    echo "服务已在运行 (pid $OLD)。如需重启请先 ./stop.sh"
    exit 0
  fi
  rm -f "$PIDFILE"
fi

cd "$SERVER_DIR" || exit 1
nohup ./server >> "$LOGFILE" 2>&1 &
NEWPID=$!
echo "$NEWPID" > "$PIDFILE"

sleep 1
if kill -0 "$NEWPID" 2>/dev/null; then
  echo "已启动 (pid $NEWPID)"
  echo "  日志: $LOGFILE"
  echo "  地址: http://127.0.0.1:8080"
  echo "  停止: $BASE/stop.sh"
else
  echo "启动失败,看日志: $LOGFILE" >&2
  tail -n 20 "$LOGFILE" >&2 2>/dev/null || true
  rm -f "$PIDFILE"
  exit 1
fi
