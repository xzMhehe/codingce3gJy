#!/usr/bin/env bash
#
# 家园社区 — Linux 数据库初始化(建库 + 建表 + 种子数据,全部幂等)
#   用法: ./init-db.sh
#   等价于: cd server && ./dbinit
#
set -u

BASE="$(cd "$(dirname "$0")" && pwd)"
SERVER_DIR="$BASE/server"

if [ ! -f "$SERVER_DIR/dbinit" ]; then
  echo "找不到 $SERVER_DIR/dbinit ,请确认包已完整解压。" >&2
  exit 1
fi

chmod +x "$SERVER_DIR/dbinit" 2>/dev/null || true
cd "$SERVER_DIR" || exit 1

exec ./dbinit "$@"
