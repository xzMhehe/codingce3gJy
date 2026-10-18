#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
清理一区军官「悬空穿戴缓存」（装备实体已迁移+删除，仅军官 equipment JSON 残留引用）。

安全设计
--------
    · 只处理明确的 8 个军官 id（前面只读探查已确认其缓存全部为悬空非珠宝装备）
    · 防御校验：若该军官缓存里**任一装备 id 仍存在于一区 ezfy_equipment 表**
      （说明装备还在、不能动），则跳过该军官并告警 —— 绝不误删还在的装备
    · 若缓存含采集珠宝名字（说明珠宝还穿在身上）→ 跳过
    · 单事务执行，出错整体回滚；执行前把原 equipment 备份成 JSON
    · 默认 dry-run，必须 --execute 才写库

用法
----
    python cleanup_officer_cache.py --host ... --user root --password '...'
    python cleanup_officer_cache.py --host ... --user root --password '...' --execute
"""

import argparse
import json
import os
import sys
import time

import pymysql

# 采集珠宝名单：若缓存里还有珠宝，说明装备还穿在身上（保留的采集珠宝），绝不清理
TREASURE_NAMES = {
    "黄金手镯", "玛瑙项坠", "红宝石戒指", "黑曜石戒指", "琥珀项链",
    "铂金戒指", "翡翠项链", "祖母绿", "蓝宝石戒指",
}

# 待清理军官（前面只读探查已确认：其 equipment 缓存全部为悬空非珠宝装备引用）
TARGETS = [170, 171, 172, 323, 554, 580, 720, 4913]


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
    ap = argparse.ArgumentParser(description="清理一区军官悬空穿戴缓存")
    ap.add_argument("--db", default="qq_ezzt", help="一区源库，默认 qq_ezzt")
    ap.add_argument("--host", default="127.0.0.1")
    ap.add_argument("--port", type=int, default=3306)
    ap.add_argument("--user", default="root")
    ap.add_argument("--password", default="1234567890")
    ap.add_argument("--execute", action="store_true", help="真正清理（不加只预览）")
    ap.add_argument("--no-backup", action="store_true", help="不导出备份 JSON")
    args = ap.parse_args()

    print("=" * 68)
    print("一区军官悬空穿戴缓存清理  库=%s  目标军官=%s" % (args.db, TARGETS))
    print("  模式 = %s" % ("执行（会写库）" if args.execute else "预览（只读，不写）"))
    print("=" * 68)

    conn = connect(args)
    try:
        with conn.cursor() as cur:
            # ---------- 读取目标军官 + 防御校验 ----------
            ph = ",".join(["%s"] * len(TARGETS))
            offs = qall(cur, "SELECT id, name, equipment FROM `%s`.ezfy_officer WHERE id IN (%s)" % (args.db, ph), TARGETS)
            by_id = {o["id"]: o for o in offs}
            # 一区装备表现存 id 集合（防御校验用）
            eq_ids = {r["id"] for r in qall(cur, "SELECT id FROM `%s`.ezfy_equipment" % args.db)}

            plan = []   # 允许清理的军官
            skip = []   # 需跳过的军官（缓存里还有装备在表里 / 含珠宝）
            for oid in TARGETS:
                o = by_id.get(oid)
                if o is None:
                    skip.append((oid, "军官不存在"))
                    continue
                items = parse_equip(o["equipment"])
                if not items:
                    skip.append((oid, "缓存为空(无需处理)"))
                    continue
                ids = [x.get("id") for x in items if x.get("id")]
                still_in_eq = [i for i in ids if i in eq_ids]
                has_treasure = any((x.get("name") or "") in TREASURE_NAMES for x in items)
                if still_in_eq or has_treasure:
                    skip.append((oid, "缓存装备仍存在(珠宝/未删): %s" % (still_in_eq or "含珠宝")))
                    continue
                plan.append((oid, o["name"], items, len(ids)))

            print("\n[1/2] 校验结果")
            for oid, name, items, n in plan:
                print("  可清理: 军官 id=%d %s  悬空缓存 %d 件（装备均已从装备表删除）" % (oid, name, n))
            for oid, why in skip:
                print("  跳过:   军官 id=%d  %s" % (oid, why))

            if not plan:
                print("\n[结束] 没有可清理的军官，无需写库。")
                return 0

            if not args.execute:
                print("\n[预览结束] 没有写任何数据。确认无误后加 --execute 执行。")
                return 0

            # ---------- 备份 + 执行 ----------
            if not args.no_backup:
                bdir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "backups")
                os.makedirs(bdir, exist_ok=True)
                bpath = os.path.join(bdir, "ezfy_officer_cache_clean_%s.json" % time.strftime("%Y%m%d_%H%M%S"))
                with open(bpath, "w", encoding="utf-8") as f:
                    json.dump(
                        [{"officer_id": oid, "name": name, "equipment": items} for oid, name, items, n in plan],
                        f, ensure_ascii=False, default=str, indent=1,
                    )
                print("\n[2/2] 备份已导出：%s" % bpath)

            print("开始清理（单事务，出错整体回滚）…")
            try:
                conn.begin()
                for oid, name, items, n in plan:
                    cur.execute("UPDATE `%s`.ezfy_officer SET equipment='' WHERE id=%%s" % args.db, (oid,))
                conn.commit()
            except Exception as e:
                conn.rollback()
                print("\n[失败] 清理出错，已整体回滚：%s" % e)
                return 4

            # ---------- 复核 ----------
            after = qall(cur, "SELECT id, equipment FROM `%s`.ezfy_officer WHERE id IN (%s)" % (args.db, ph), TARGETS)
            print("\n完成 ✅")
            for r in after:
                print("  军官 id=%d equipment=%r" % (r["id"], r["equipment"]))
            return 0
    finally:
        conn.close()


if __name__ == "__main__":
    sys.exit(main())
