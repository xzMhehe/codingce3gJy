#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
清理一区军官穿戴缓存（移动转移后专用版：按玩家查军官）。

背景
----
    一区资产「移动转移」到二区后，一区非采集珠宝装备已删除 → 军官 equipment
    JSON 缓存里对这些装备的引用变成**悬空引用**，军官面板会显示不存在的装备。
    本脚本按玩家（家园号）查其全部军官，只清理缓存里**非采集珠宝**的悬空引用；
    采集珠宝装备保留在一区且仍穿戴 → 缓存里的珠宝条目**原样保留**。

安全设计
--------
    · 防御校验：若缓存里某非珠宝装备 id **仍存在于一区 ezfy_equipment 表**
      （说明装备还在、不能动）→ 跳过该军官并告警 —— 绝不误清还在的装备
    · 单事务执行，出错整体回滚；执行前把各军官原 equipment 备份成 JSON
    · 默认 dry-run，必须 --execute 才写库

用法
----
    python cleanup_officer_cache_users.py --players 35806418,35806363,35806436 \
        --host ... --user root --password '...'
    python cleanup_officer_cache_users.py --players ... --host ... --execute
"""

import argparse
import json
import os
import sys
import time

import pymysql

# 采集珠宝名单：缓存里含珠宝条目时**保留**（装备仍在表里、仍穿戴）
TREASURE_NAMES = {
    "黄金手镯", "玛瑙项坠", "红宝石戒指", "黑曜石戒指", "琥珀项链",
    "铂金戒指", "翡翠项链", "祖母绿", "蓝宝石戒指",
}


def connect(args):
    return pymysql.connect(
        host=args.host, port=args.port, user=args.user, password=args.password,
        charset="utf8mb4", autocommit=False, cursorclass=pymysql.cursors.DictCursor,
    )


def qall(cur, sql, params=None):
    cur.execute(sql, params or ())
    return cur.fetchall()


def q1(cur, sql, params=None):
    cur.execute(sql, params or ())
    return cur.fetchone()


def parse_equip(raw):
    if not raw:
        return []
    try:
        v = json.loads(raw)
    except Exception:
        return []
    return v if isinstance(v, list) else [v]


def main():
    ap = argparse.ArgumentParser(description="清理一区军官穿戴缓存（按玩家）")
    ap.add_argument("--players", required=True, help="玩家家园号，逗号分隔，如 35806418,35806363")
    ap.add_argument("--db", default="qq_ezzt", help="一区源库，默认 qq_ezzt")
    ap.add_argument("--host", default="127.0.0.1")
    ap.add_argument("--port", type=int, default=3306)
    ap.add_argument("--user", default="root")
    ap.add_argument("--password", default="1234567890")
    ap.add_argument("--execute", action="store_true", help="真正清理（不加只预览）")
    ap.add_argument("--no-backup", action="store_true", help="不导出备份 JSON")
    args = ap.parse_args()

    players = [int(x.strip()) for x in args.players.split(",") if x.strip()]
    if not players:
        print("未提供有效玩家列表")
        return 2

    print("=" * 68)
    print("一区军官穿戴缓存清理  库=%s  玩家=%s" % (args.db, players))
    print("  模式 = %s" % ("执行（会写库）" if args.execute else "预览（只读，不写）"))
    print("=" * 68)

    conn = connect(args)
    try:
        with conn.cursor() as cur:
            # ---------- 1. 按玩家查军官（JOIN 城市归属） ----------
            ph = ",".join(["%s"] * len(players))
            offs = qall(cur,
                        "SELECT o.id, o.name, o.equipment FROM `%s`.ezfy_officer o "
                        "JOIN `%s`.ezfy_city c ON c.id = o.city_id "
                        "WHERE c.user_id IN (%s) AND o.deleted_at IS NULL "
                        "ORDER BY o.id" % (args.db, args.db, ph), players)

            # 一区装备表现存 id 集合（防御校验用；当前只剩采集珠宝）
            remain_ids = {r["id"] for r in qall(
                cur, "SELECT id FROM `%s`.ezfy_equipment WHERE user_id IN (%s)" % (args.db, ph), players)}

            plan = []   # (officer_id, name, 保留珠宝条目, 清理的悬空条目数)
            skip = []   # (officer_id, 原因)
            for o in offs:
                oid = o["id"]
                items = parse_equip(o["equipment"])
                if not items:
                    skip.append((oid, "缓存为空(无需处理)"))
                    continue
                keep = []       # 珠宝条目（装备仍在表里）→ 保留
                dangling = []   # 悬空引用（非珠宝且装备已删）→ 清理
                still = []      # 非珠宝但装备仍存在 → 防御跳过
                for x in items:
                    nm = x.get("name") or ""
                    eid = x.get("id")
                    if nm in TREASURE_NAMES:
                        keep.append(x)
                        continue
                    if eid in remain_ids:
                        still.append(eid)
                    else:
                        dangling.append(x)
                if still:
                    skip.append((oid, "缓存非珠宝装备仍存在于装备表，跳过: %s" % still))
                    continue
                if not dangling:
                    skip.append((oid, "无悬空引用(无需处理)"))
                    continue
                plan.append((oid, o["name"], keep, dangling))

            print("\n[1/2] 校验结果（共 %d 名军官）" % len(offs))
            for oid, name, keep, dangling in plan:
                print("  可清理: 军官 id=%d %s  悬空引用 %d 件 → 移除；珠宝条目 %d 件保留"
                      % (oid, name, len(dangling), len(keep)))
            for oid, why in skip:
                print("  跳过:   军官 id=%d  %s" % (oid, why))

            if not plan:
                print("\n[结束] 没有可清理的军官，无需写库。")
                return 0

            if not args.execute:
                print("\n[预览结束] 没有写任何数据。确认无误后加 --execute 执行。")
                return 0

            # ---------- 2. 备份 + 执行 ----------
            if not args.no_backup:
                bdir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "backups")
                os.makedirs(bdir, exist_ok=True)
                bpath = os.path.join(bdir, "ezfy_officer_cache_users_clean_%s.json" % time.strftime("%Y%m%d_%H%M%S"))
                with open(bpath, "w", encoding="utf-8") as f:
                    json.dump(
                        [{"officer_id": oid, "name": name,
                          "original": keep + dangling, "keep": keep, "removed": dangling}
                         for oid, name, keep, dangling in plan],
                        f, ensure_ascii=False, default=str, indent=1,
                    )
                print("\n[2/2] 备份已导出：%s" % bpath)

            print("开始清理（单事务，出错整体回滚）…")
            try:
                conn.begin()
                for oid, name, keep, dangling in plan:
                    new_raw = json.dumps(keep, ensure_ascii=False) if keep else ""
                    cur.execute("UPDATE `%s`.ezfy_officer SET equipment=%%s WHERE id=%%s" % args.db,
                                (new_raw, oid))
                conn.commit()
            except Exception as e:
                conn.rollback()
                print("\n[失败] 清理出错，已整体回滚：%s" % e)
                return 4

            # ---------- 3. 复核 ----------
            ph2 = ",".join(["%s"] * len(players))
            after = qall(cur,
                         "SELECT o.id, o.equipment FROM `%s`.ezfy_officer o "
                         "JOIN `%s`.ezfy_city c ON c.id = o.city_id "
                         "WHERE c.user_id IN (%s) AND o.deleted_at IS NULL" % (args.db, args.db, ph2), players)
            print("\n完成 ✅")
            for r in after:
                print("  军官 id=%d equipment=%r" % (r["id"], r["equipment"]))
            return 0
    finally:
        conn.close()


if __name__ == "__main__":
    sys.exit(main())
