#!/usr/bin/env bash
#
# 家园社区 — Linux 停止脚本
#   用法: ./stop.sh
#
set -u

BASE="$(cd "$(dirname "$0")" && pwd)"
PIDFILE="$BASE/logs/server.pid"

if [ ! -f "$PIDFILE" ]; then
  echo "没找到 $PIDFILE ,服务可能没在跑。"
  echo "若确认有残留进程: ps -ef | grep '[s]erver' 或 ss -lntp | grep 8080"
  exit 0
fi

PID="$(cat "$PIDFILE" 2>/dev/null || true)"
if [ -z "$PID" ]; then
  echo "pid 文件是空的,已清理。"
  rm -f "$PIDFILE"
  exit 0
fi

if ! kill -0 "$PID" 2>/dev/null; then
  echo "进程 $PID 已不存在,清理 pid 文件。"
  rm -f "$PIDFILE"
  exit 0
fi

kill "$PID" 2>/dev/null || true
for _ in 1 2 3 4 5 6 7 8 9 10; do
  kill -0 "$PID" 2>/dev/null || break
  sleep 0.5
done

if kill -0 "$PID" 2>/dev/null; then
  echo "优雅退出超时,强制结束 pid $PID"
  kill -9 "$PID" 2>/dev/null || true
  sleep 0.5
fi

rm -f "$PIDFILE"
echo "已停止 (pid $PID)"
