#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
二战一区 → 二区 资产迁移补充：**清空一区已迁移资产**（配合 migrate_ezfy_assets.py 的复制步骤，
组成完整的「移动转移」语义）。

注意
----
    本脚本只清理**已经迁到二区**的那部分资产。执行前请先预览并与迁移记录核对：
    钻石当前值应等于迁移时快照值；装备数量应等于「迁移时 eqs 数量」；道具/发电卡同理。
    若玩家在迁移后又在一区产生新资产，此处会一并清掉 —— 属预期（账号资产整体转移到二区）。

清理范围（一区 qq_ezzt）
------------------------
    · 钻石   UPDATE ezfy_profile SET diamond = 0
    · 装备   DELETE ezfy_equipment WHERE name NOT IN (采集珠宝名单)   ← 采集珠宝保留
    · 道具   DELETE ezfy_item
    · 发电卡 DELETE ezfy_love_card

用法
----
    # 预览（只读）
    python cleanup_ezfy_move.py 35806480 --host ... --user root --password '...'

    # 执行
    python cleanup_ezfy_move.py 35806480 --host ... --user root --password '...' --execute

依赖：pip install pymysql
"""

import argparse
import json
import os
import sys
import time

import pymysql

# 采集宝物名单（与 migrate_ezfy_assets.py / 后端 ezfyCollectibleTreasureNames 一致）：保留不删
TREASURE_NAMES = {
    "黄金手镯", "玛瑙项坠", "红宝石戒指", "黑曜石戒指", "琥珀项链",
    "铂金戒指", "翡翠项链", "祖母绿", "蓝宝石戒指",
}


def connect(args):
    return pymysql.connect(
        host=args.host, port=args.port, user=args.user, password=args.password,
        charset="utf8mb4", autocommit=False, cursorclass=pymysql.cursors.DictCursor,
    )


def q1(cur, sql, params=None):
    cur.execute(sql, params or ())
    return cur.fetchone()


def qall(cur, sql, params=None):
    cur.execute(sql, params or ())
    return cur.fetchall()


def table_exists(cur, schema, table):
    return bool(q1(cur,
                   "SELECT 1 AS ok FROM information_schema.tables "
                   "WHERE table_schema=%s AND table_name=%s LIMIT 1",
                   (schema, table)))


def main():
    ap = argparse.ArgumentParser(description="一区已迁移资产清理（移动转移的清除半程）")
    ap.add_argument("game_id", type=int, help="家园号（= 二战 game_id）")
    ap.add_argument("--db", default="qq_ezzt", help="一区源库，默认 qq_ezzt")
    ap.add_argument("--host", default="127.0.0.1")
    ap.add_argument("--port", type=int, default=3306)
    ap.add_argument("--user", default="root")
    ap.add_argument("--password", default="1234567890")
    ap.add_argument("--execute", action="store_true", help="真正清理（不加只预览）")
    ap.add_argument("--no-backup", action="store_true", help="不导出清理前备份 JSON")
    args = ap.parse_args()

    uid = args.game_id
    print("=" * 68)
    print("一区资产清理（移动转移-清除）  game_id=%d  库=%s" % (uid, args.db))
    print("  模式 = %s" % ("执行（会清一区）" if args.execute else "预览（只读，不写）"))
    print("=" * 68)

    conn = connect(args)
    try:
        with conn.cursor() as cur:
            # ---------- 1. 读一区当前快照 ----------
            prof = q1(cur, "SELECT diamond FROM `%s`.ezfy_profile WHERE user_id=%%s" % args.db, (uid,))
            if not prof:
                print("\n[中止] 一区没有该玩家档案，无需清理。")
                return 2
            diamond = int(prof["diamond"] or 0)

            eq_all = (qall(cur, "SELECT * FROM `%s`.ezfy_equipment WHERE user_id=%%s" % args.db, (uid,))
                      if table_exists(cur, args.db, "ezfy_equipment") else [])
            eq_del = [e for e in eq_all if (e.get("name") or "") not in TREASURE_NAMES]
            treasure_cnt = len(eq_all) - len(eq_del)

            items = (qall(cur, "SELECT cfg_id, count FROM `%s`.ezfy_item WHERE user_id=%%s ORDER BY cfg_id" % args.db, (uid,))
                     if table_exists(cur, args.db, "ezfy_item") else [])
            cards = (qall(cur, "SELECT * FROM `%s`.ezfy_love_card WHERE user_id=%%s ORDER BY id" % args.db, (uid,))
                     if table_exists(cur, args.db, "ezfy_love_card") else [])

            print("\n[1/3] 一区当前快照")
            print("  钻石=%d  装备=%d 件（其中采集珠宝 %d 件**保留**，其余 %d 件将删除）  道具=%d 种  发电卡=%d 张"
                  % (diamond, len(eq_all), treasure_cnt, len(eq_del), len(items), len(cards)))

            if diamond == 0 and not eq_del and not items and not cards:
                print("\n[跳过] 一区可迁移资产已经为空，无需清理。")
                return 0

            # ---------- 2. 清理计划 ----------
            print("\n[2/3] 清理计划")
            print("  钻石 : 清零（%d → 0）" % diamond)
            print("  装备 : 删除 %d 件（不含采集珠宝 %d 件）" % (len(eq_del), treasure_cnt))
            print("  道具 : 删除 %d 种（共 %d 个）" % (len(items), sum(int(i["count"] or 0) for i in items)))
            print("  发电卡: 删除 %d 张" % len(cards))

            if not args.execute:
                print("\n[预览结束] 没有写任何数据。确认无误后加 --execute 执行。")
                return 0

            # ---------- 3. 备份 + 执行 ----------
            if not args.no_backup:
                bdir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "backups")
                os.makedirs(bdir, exist_ok=True)
                bpath = os.path.join(bdir, "ezfy_clean1_%d_%s.json" % (uid, time.strftime("%Y%m%d_%H%M%S")))
                with open(bpath, "w", encoding="utf-8") as f:
                    json.dump({
                        "game_id": uid,
                        "db": args.db,
                        "taken_at": time.strftime("%Y-%m-%d %H:%M:%S"),
                        "diamond": diamond,
                        "equipment": eq_del,   # 备份将要删除的装备（珠宝不备份）
                        "items": items,
                        "love_cards": cards,
                    }, f, ensure_ascii=False, default=str, indent=1)
                print("\n[3/3] 清理前备份已导出：%s" % bpath)

            print("开始清理（单事务，出错整体回滚）…")
            try:
                conn.begin()
                if diamond:
                    cur.execute("UPDATE `%s`.ezfy_profile SET diamond = 0 WHERE user_id=%%s" % args.db, (uid,))
                if eq_del:
                    ph = ",".join(["%s"] * len(eq_del))
                    cur.execute(
                        "DELETE FROM `%s`.ezfy_equipment WHERE id IN (%s)" % (args.db, ph),
                        [e["id"] for e in eq_del],
                    )
                if items and table_exists(cur, args.db, "ezfy_item"):
                    cur.execute("DELETE FROM `%s`.ezfy_item WHERE user_id=%%s" % args.db, (uid,))
                if cards and table_exists(cur, args.db, "ezfy_love_card"):
                    cur.execute("DELETE FROM `%s`.ezfy_love_card WHERE user_id=%%s" % args.db, (uid,))
                conn.commit()
            except Exception as e:
                conn.rollback()
                print("\n[失败] 清理出错，已整体回滚：%s" % e)
                return 4

            # ---------- 4. 复核 ----------
            prof_after = q1(cur, "SELECT diamond FROM `%s`.ezfy_profile WHERE user_id=%%s" % args.db, (uid,))
            eq_after = qall(cur, "SELECT * FROM `%s`.ezfy_equipment WHERE user_id=%%s" % args.db, (uid,)) \
                if table_exists(cur, args.db, "ezfy_equipment") else []
            it_after = q1(cur, "SELECT IFNULL(SUM(count),0) AS s FROM `%s`.ezfy_item WHERE user_id=%%s" % args.db, (uid,)) \
                if table_exists(cur, args.db, "ezfy_item") else {"s": 0}
            card_after = qall(cur, "SELECT id FROM `%s`.ezfy_love_card WHERE user_id=%%s" % args.db, (uid,)) \
                if table_exists(cur, args.db, "ezfy_love_card") else []
            treasure_after = [e for e in eq_after if (e.get("name") or "") in TREASURE_NAMES]
            print("\n完成 ✅  一区剩余：钻石=%d  装备=%d 件（其中采集珠宝 %d 件保留）  道具合计=%d 个  发电卡=%d 张"
                  % (prof_after["diamond"], len(eq_after), len(treasure_after),
                     int(it_after["s"] or 0), len(card_after)))
            print("\n提示：一区已迁移资产已清除，二区资产不受影响。")
            return 0
    finally:
        conn.close()


if __name__ == "__main__":
    sys.exit(main())
