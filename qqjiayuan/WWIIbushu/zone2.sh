#!/usr/bin/env bash
#
# 开2区.sh — 从当前二战部署目录复制出 2 区(独立库 + fresh 建表)
#
# 用法(在已部署好的 1 区二战目录里执行):
#   ./开2区.sh                                      # 默认: 目录 ../wwii-2, 端口=当前+1, 库=当前库名+2,
#                                                    #       区名=【2区】诺曼底登陆
#   ./开2区.sh /srv/wwii-2                          # 指定目录
#   ./开2区.sh /srv/wwii-2 8082 qq_ezzt2 诺曼底登陆  # 全指定(第4参=区名后缀)
#
# 脚本只做三件事:
#   1) 复制整个部署目录(不含 logs)
#   2) 改 server/config.yaml: port=新端口, zone_id/zone_name/zone_sub=新区,
#      ezfy_mysql.dbname=新库, ezfy_mysql.bootstrap=fresh
#      ★ fresh 是关键: 空库只建表+灌种子,绝不把 1 区/家园残留数据搬进 2 区
#   3) 打印启动前检查清单
#
# ★★ 2026-10-17 二战独立部署: 2 区二战风云用**独立库**(如 qq_ezzt2),
#    与 1 区库 qq_ezzt 不共用;家园库 qq_jiayuan 全区共用(账号/管理端权限仍在家园)。
#    ★ 区名: zone_name=【2区】, zone_sub=诺曼底登陆 → 完整标题「二战征途-【2区】诺曼底登陆」
#      (config.ZoneTitle() = game_name + "-【" + zone_name + "】" + zone_sub)
#
set -u

BASE="$(cd "$(dirname "$0")" && pwd)"
CFG="$BASE/server/config.yaml"
[ -f "$CFG" ] || { echo "找不到 $CFG ,请在已部署的二战目录里运行本脚本。" >&2; exit 1; }

# 读指定块里的键值(server 块取 port/zone_id, ezfy_mysql 块取 dbname)
block_val() { # $1=块名 $2=键名
  awk -v b="$1" -v k="$2" '
    $0 ~ "^" b ":" { inb=1; next }
    inb && /^[a-zA-Z_][a-zA-Z0-9_]*:/ { inb=0 }
    inb && $0 ~ "^[ \t]*" k ":" {
      sub(/^[ \t]*[^:]+:[ \t]*/, ""); gsub(/["\r]/, ""); print; exit
    }' "$CFG"
}

CUR_PORT="$(block_val server port)";  [ -n "$CUR_PORT" ] || CUR_PORT=8081
CUR_ZID="$(block_val server zone_id)"; [ -n "$CUR_ZID" ] || CUR_ZID=1
CUR_DB="$(block_val ezfy_mysql dbname)"; [ -n "$CUR_DB" ] || CUR_DB=qq_ezzt

TARGET="${1:-$(dirname "$BASE")/wwii-2}"
NEW_PORT="${2:-$((CUR_PORT+1))}"
NEW_DB="${3:-${CUR_DB}2}"
NEW_SUB="${4:-诺曼底登陆}"
NEW_ZID=$((CUR_ZID+1))

if [ -e "$TARGET" ] && [ -n "$(ls -A "$TARGET" 2>/dev/null)" ]; then
  echo "目标目录 $TARGET 已存在且有内容,为避免覆盖请先清空再运行。" >&2
  exit 1
fi

echo "== 复制 1 区部署目录 → $TARGET =="
mkdir -p "$(dirname "$TARGET")"
cp -a "$BASE/." "$TARGET/"
rm -rf "$TARGET/logs"

echo "== 改写 $TARGET/server/config.yaml =="
sed -i -E "
  /^server:/,/^[a-zA-Z_][a-zA-Z0-9_]*:/ {
    s/^(  port:).*/\1 $NEW_PORT/
    s/^(  zone_id:).*/\1 $NEW_ZID/
    s/^(  zone_name:).*/\1 \"${NEW_ZID}区\"/
  }
  /^ezfy_mysql:/,/^[a-zA-Z_][a-zA-Z0-9_]*:/ {
    s/^(  dbname:).*/\1 $NEW_DB/
    s/^(  bootstrap:).*/\1 \"fresh\"/
  }" "$TARGET/server/config.yaml"
# zone_sub 老配置可能没有该行,先补一行再替换(区名后缀,默认 诺曼底登陆)
if ! grep -qE '^[[:space:]]*zone_sub:' "$TARGET/server/config.yaml"; then
  sed -i -E '/^[[:space:]]*zone_name:/a\  zone_sub: ""' "$TARGET/server/config.yaml"
fi
sed -i -E "s/^(  zone_sub:).*/\1 \"$NEW_SUB\"/" "$TARGET/server/config.yaml"

echo "2 区目录已生成: $TARGET"
echo "  端口: $NEW_PORT    库: $NEW_DB    区名: 二战征途-【${NEW_ZID}区】${NEW_SUB}    zone_id: $NEW_ZID"
echo "  ezfy_mysql.bootstrap = fresh(已写入,只建表灌种子,不搬 1 区数据)"
echo ""
echo "启动前检查:"
echo "  1) vi $TARGET/server/config.yaml 确认 MySQL 密码/主机可连、jwt.secret 与家园包一致"
echo "  2) $TARGET/start.sh  启动 2 区(二战库 $NEW_DB 首次启动自动创建+建表+灌种子)"
echo "  3) 家园服管理端『游戏大厅』给二战加 2 区服条目:"
echo "     name=二战征途-【${NEW_ZID}区】${NEW_SUB}, host=http://服务器IP:$NEW_PORT, path=/games/ezfy, status=1"
echo "  4) 玩家从家园进二战默认落 1 区;管理端二战菜单顶栏『管理区』可切 2 区"
echo "  5) ★ 2 区玩家在二战首页刷新只显示本区(二战服 zone 接口只下发本区),不会闪出 1 区信息"
