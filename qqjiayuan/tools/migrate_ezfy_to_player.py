#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
二战一区 → 二区 **特殊迁移**：多个一区小号资产 → 二区指定大号
（源账号与目标账号不同：源 = 一区小号 user_id，目标 = 二区大号 user_id）

用法
----
    # 预览（只读）
    python migrate_ezfy_to_player.py 35806435 --to-user 35806436 \
        --host ... --user root --password '...'

    # 执行
    python migrate_ezfy_to_player.py 35806435 --to-user 35806436 \
        --host ... --user root --password '...' --execute

参数
----
    game_id           源账号（一区小号，= 二战 game_id）
    --to-user         目标账号（二区大号 user_id）
    --from-db/--to-db 默认 qq_ezzt / qq_ezzt2
    --host/--port/--user/--password  MySQL 连接
    --execute         真正写入；不加只预览
    --force           忽略「该源已迁移过」检查（会重复累加，慎用）
    --no-backup       不导出备份 JSON

迁移范围（与 migrate_ezfy_assets.py 同口径）
-------------------------------------------
    · 钻石   ezfy_profile.diamond 累加到目标（不清空目标原有）
    · 装备   非采集珠宝全量复制到目标，city_id 一律 0、officer_id / original_user_id 归零
    · 道具   ezfy_item 按 (目标 user_id, cfg_id) 累加（ON DUPLICATE KEY UPDATE）
    · 发电卡 剩余天数合并到目标同类型未领完的卡（否则新建）
    · 流水   目标 ezfy_diamond_logs 写 reason='一区资产迁移 源<源账号>' 兼作防重标记

安全设计
--------
    · 默认 dry-run，必须显式 --execute 才写库
    · 全部写入在一个事务里，中途报错整体回滚
    · 同一源账号只迁一次（防重标记）；不同源各自独立事务、可顺序累加
    · 目标账号必须在二区存在（ezfy_account + ezfy_profile）
    · 配置 id 会先在二区 ezfy_cfg_item / ezfy_cfg_equipment 里校验，缺失的跳过并告警

依赖：pip install pymysql
"""

import argparse
import json
import os
import sys
import time

try:
    import pymysql
except ImportError:
    print("缺少依赖：pip install pymysql")
    sys.exit(1)

# 为爱发电卡 1 天的毫秒数（与后端 ezfyLoveCardDayMS 一致）
LOVE_DAY_MS = 24 * 3600 * 1000

# 采集宝物名单（与后端 ezfyCollectibleTreasureNames 同一套）：**不迁移**
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


def cfg_ids(cur, schema, table, ids):
    ids = [i for i in set(ids) if i is not None]
    if not ids:
        return set()
    ph = ",".join(["%s"] * len(ids))
    rows = qall(cur, "SELECT id FROM `%s`.`%s` WHERE id IN (%s)" % (schema, table, ph), ids)
    return {r["id"] for r in rows}


def table_exists(cur, schema, table):
    return bool(q1(
        cur,
        "SELECT 1 AS ok FROM information_schema.tables "
        "WHERE table_schema=%s AND table_name=%s LIMIT 1",
        (schema, table),
    ))


def main():
    ap = argparse.ArgumentParser(description="一区小号 → 二区大号 特殊资产迁移")
    ap.add_argument("game_id", type=int, help="源账号（一区小号 game_id）")
    ap.add_argument("--to-user", required=True, type=int, help="目标账号（二区大号 user_id）")
    ap.add_argument("--from-db", default="qq_ezzt")
    ap.add_argument("--to-db", default="qq_ezzt2")
    ap.add_argument("--host", default="127.0.0.1")
    ap.add_argument("--port", type=int, default=3306)
    ap.add_argument("--user", default="root")
    ap.add_argument("--password", default="1234567890")
    ap.add_argument("--execute", action="store_true")
    ap.add_argument("--force", action="store_true")
    ap.add_argument("--no-backup", action="store_true")
    args = ap.parse_args()

    uid = args.game_id
    dst_uid = args.to_user
    SRC, DST = args.from_db, args.to_db
    if uid == dst_uid:
        print("源与目标是同一个账号（%d），请用 migrate_ezfy_assets.py" % uid)
        return 2

    print("=" * 68)
    print("二战特殊迁移  源(一区小号)=%d → 目标(二区大号)=%d" % (uid, dst_uid))
    print("  源库(一区) = %s  目标库(二区) = %s" % (SRC, DST))
    print("  模式 = %s" % ("执行（会写库）" if args.execute else "预览（只读，不写）"))
    print("=" * 68)

    conn = connect(args)
    try:
        with conn.cursor() as cur:
            # ---------- 1. 校验二区目标账号 ----------
            acct = q1(cur, "SELECT game_id, nickname FROM `%s`.ezfy_account WHERE game_id=%%s" % DST, (dst_uid,))
            if not acct:
                print("\n[中止] 二区(%s) 没有目标账号（ezfy_account.game_id=%d），请先在二区建档。" % (DST, dst_uid))
                return 2
            prof2 = q1(cur, "SELECT user_id, diamond FROM `%s`.ezfy_profile WHERE user_id=%%s" % DST, (dst_uid,))
            if not prof2:
                print("\n[中止] 二区有账号但 ezfy_profile 缺失（档案不完整），请先让玩家登录一次。")
                return 2
            print("\n[1/7] 二区目标账号 ✅  %d / %s  当前钻石=%d"
                  % (acct["game_id"], acct["nickname"], prof2["diamond"]))

            # ---------- 2. 防重（每个源独立标记） ----------
            dup = q1(
                cur,
                "SELECT id, created_at FROM `%s`.ezfy_diamond_logs "
                "WHERE user_id=%%s AND reason=%%s LIMIT 1" % DST,
                (dst_uid, "一区资产迁移 源%d" % uid),
            )
            if dup and not args.force:
                print("\n[中止] 该源（%d）**已经迁移过**（流水 id=%s，时间=%s）" % (uid, dup["id"], dup["created_at"]))
                print("       再跑会重复累加。确实要重迁请加 --force。")
                return 3

            # ---------- 3. 读一区源账号 ----------
            prof1 = q1(cur, "SELECT user_id, diamond FROM `%s`.ezfy_profile WHERE user_id=%%s" % SRC, (uid,))
            if not prof1:
                print("\n[中止] 一区(%s) 没有源账号档案（user_id=%d），请先确认账号存在。" % (SRC, uid))
                return 2
            diamond1 = int(prof1["diamond"] or 0)

            eq_all = (qall(cur, "SELECT * FROM `%s`.ezfy_equipment WHERE user_id=%%s" % SRC, (uid,))
                      if table_exists(cur, SRC, "ezfy_equipment") else [])
            eqs = [e for e in eq_all if (e.get("name") or "") not in TREASURE_NAMES]
            treasure_cnt = len(eq_all) - len(eqs)
            items = (qall(cur, "SELECT cfg_id, count FROM `%s`.ezfy_item WHERE user_id=%%s ORDER BY cfg_id" % SRC, (uid,))
                     if table_exists(cur, SRC, "ezfy_item") else [])
            cards = (qall(cur, "SELECT * FROM `%s`.ezfy_love_card WHERE user_id=%%s ORDER BY id" % SRC, (uid,))
                     if table_exists(cur, SRC, "ezfy_love_card") else [])
            love_plan = {}
            for c in cards:
                remain = int(c["total_days"] or 0) - int(c["claimed_days"] or 0)
                if remain <= 0:
                    continue
                p = love_plan.setdefault(c["cfg_id"], {
                    "remain": 0,
                    "name": c["name"] or ("为爱发电卡" if c["cfg_id"] == 26 else "为爱发电高级卡"),
                    "daily": int(c["daily_diamond"] or (150 if c["cfg_id"] == 26 else 200)),
                })
                p["remain"] += remain
            love_days = sum(p["remain"] for p in love_plan.values())
            print("\n[2/7] 一区源账号数据 ✅  钻石=%d  装备=%d 件（另有采集宝物 %d 件**不迁移**）  道具=%d 种  发电卡=%d 种/剩 %d 天"
                  % (diamond1, len(eqs), treasure_cnt, len(items), len(love_plan), love_days))

            if diamond1 == 0 and not eqs and not items and not love_plan:
                print("\n[跳过] 源账号这几样都是空的，没有可迁移的内容。")
                return 0

            # ---------- 4. 校验配置 id ----------
            eq_ok = cfg_ids(cur, DST, "ezfy_cfg_equipment", [e["cfg_id"] for e in eqs])
            eq_use = [e for e in eqs if e["cfg_id"] in eq_ok]
            it_ok = cfg_ids(cur, DST, "ezfy_cfg_item", [i["cfg_id"] for i in items])
            it_use = [i for i in items if i["cfg_id"] in it_ok]
            if len(eq_use) != len(eqs):
                print("  [告警] %d 件装备的 cfg_id 在二区配置表里不存在，将跳过"
                      % (len(eqs) - len(eq_use)))
            if len(it_use) != len(items):
                print("  [告警] %d 种道具的 cfg_id 在二区配置表里不存在，将跳过"
                      % (len(items) - len(it_use)))

            # ---------- 5. 计划 ----------
            new_diamond = int(prof2["diamond"] or 0) + diamond1
            print("\n[3/7] 迁移计划")
            print("  钻石 : 目标 %d + %d = %d" % (prof2["diamond"], diamond1, new_diamond))
            print("  装备 : %d 件 → 二区（user_id=%d，city_id=0，officer_id/original_user_id 归零）"
                  % (len(eq_use), dst_uid))
            print("  道具 : %d 种 → 按 (user_id=%d,cfg_id) 累加" % (len(it_use), dst_uid))
            for cid, p in sorted(love_plan.items()):
                print("  发电卡: cfg_id=%d %s 剩 %d 天 → 并入二区目标同类型卡" % (cid, p["name"], p["remain"]))

            if not args.execute:
                print("\n[预览结束] 没有写任何数据。确认无误后加 --execute 执行。")
                return 0

            # ---------- 6. 备份 + 执行 ----------
            if not args.no_backup:
                bdir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "backups")
                os.makedirs(bdir, exist_ok=True)
                bpath = os.path.join(bdir, "ezfy_migrate_to_%d_src_%d_%s.json" % (dst_uid, uid, time.strftime("%Y%m%d_%H%M%S")))
                with open(bpath, "w", encoding="utf-8") as f:
                    json.dump({
                        "src_user": uid, "dst_user": dst_uid,
                        "from_db": SRC, "to_db": DST,
                        "taken_at": time.strftime("%Y-%m-%d %H:%M:%S"),
                        "src_diamond": diamond1,
                        "dst_diamond_before": int(prof2["diamond"] or 0),
                        "equipment": eq_all,
                        "items": items,
                        "love_cards": cards,
                    }, f, ensure_ascii=False, default=str, indent=1)
                print("\n[4/7] 备份已导出：%s" % bpath)

            print("\n[5/7] 开始写入（单事务，出错整体回滚）…")
            try:
                conn.begin()

                # 6.1 钻石累加
                cur.execute(
                    "UPDATE `%s`.ezfy_profile SET diamond = diamond + %%s WHERE user_id=%%s" % DST,
                    (diamond1, dst_uid),
                )

                # 6.2 装备复制（user_id → 目标；city_id/officer_id/original_user_id 归零）
                if eq_use:
                    cols = [
                        "user_id", "city_id", "cfg_id", "name", "type", "tier",
                        "military", "logistics", "learning", "level",
                        "officer_id", "created_at", "slot", "set_id", "series",
                        "enhance", "dmg", "def", "hp", "move", "crit", "crit_dmg",
                        "original_user_id",
                    ]
                    sql = (
                        "INSERT INTO `%s`.ezfy_equipment (%s) VALUES (%s)"
                        % (DST, ",".join("`%s`" % c for c in cols), ",".join(["%s"] * len(cols)))
                    )
                    rows = []
                    for e in eq_use:
                        rows.append(tuple(
                            dst_uid if c == "user_id" else
                            0 if c in ("city_id", "officer_id", "original_user_id") else
                            e.get(c)
                            for c in cols
                        ))
                    for i in range(0, len(rows), 200):
                        cur.executemany(sql, rows[i:i + 200])

                # 6.3 道具累加（目标 user_id）
                if it_use and table_exists(cur, DST, "ezfy_item"):
                    cur.executemany(
                        "INSERT INTO `%s`.ezfy_item (user_id, cfg_id, count) VALUES (%%s,%%s,%%s) "
                        "ON DUPLICATE KEY UPDATE count = count + VALUES(count)" % DST,
                        [(dst_uid, i["cfg_id"], int(i["count"] or 0)) for i in it_use],
                    )
                elif it_use:
                    print("  [告警] 二区没有 ezfy_item 表，跳过 %d 种道具" % len(it_use))

                # 6.4 发电卡：并入目标同类型未领完的卡，否则新建
                if love_plan and not table_exists(cur, DST, "ezfy_love_card"):
                    print("  [告警] 二区没有 ezfy_love_card 表，跳过 %d 种发电卡" % len(love_plan))
                elif love_plan:
                    now_ms = int(time.time() * 1000)
                    for cid, p in love_plan.items():
                        cur_card = q1(
                            cur,
                            "SELECT id FROM `%s`.ezfy_love_card "
                            "WHERE user_id=%%s AND cfg_id=%%s AND claimed_days < total_days "
                            "ORDER BY created_at ASC, id ASC LIMIT 1" % DST,
                            (dst_uid, cid),
                        )
                        if cur_card:
                            cur.execute(
                                "UPDATE `%s`.ezfy_love_card "
                                "SET total_days = total_days + %%s, end_time = end_time + %%s "
                                "WHERE id = %%s" % DST,
                                (p["remain"], p["remain"] * LOVE_DAY_MS, cur_card["id"]),
                            )
                        else:
                            cur.execute(
                                "INSERT INTO `%s`.ezfy_love_card "
                                "(user_id,cfg_id,name,daily_diamond,start_time,end_time,total_days,"
                                " claimed_days,last_claim_time,created_at) "
                                "VALUES (%%s,%%s,%%s,%%s,%%s,%%s,%%s,0,0,NOW())" % DST,
                                (dst_uid, cid, p["name"], p["daily"], now_ms,
                                 now_ms + p["remain"] * LOVE_DAY_MS, p["remain"]),
                            )

                # 6.5 钻石流水（兼作「该源已迁移」标记）
                cur.execute(
                    "INSERT INTO `%s`.ezfy_diamond_logs (user_id, `change`, extra, balance, reason, created_at) "
                    "VALUES (%%s,%%s,%%s,%%s,%%s,NOW())" % DST,
                    (dst_uid, diamond1, "", new_diamond, "一区资产迁移 源%d" % uid),
                )

                conn.commit()
            except Exception as e:
                conn.rollback()
                print("\n[失败] 写入出错，已整体回滚：%s" % e)
                return 4

            # ---------- 7. 复核 ----------
            prof_after = q1(cur, "SELECT diamond FROM `%s`.ezfy_profile WHERE user_id=%%s" % DST, (dst_uid,))
            eq_after = q1(cur, "SELECT COUNT(*) AS n FROM `%s`.ezfy_equipment WHERE user_id=%%s" % DST, (dst_uid,))
            it_after = q1(cur, "SELECT COUNT(*) AS n, IFNULL(SUM(count),0) AS s FROM `%s`.ezfy_item WHERE user_id=%%s" % DST, (dst_uid,))
            love_after = qall(
                cur,
                "SELECT name, total_days, claimed_days FROM `%s`.ezfy_love_card "
                "WHERE user_id=%%s AND claimed_days < total_days ORDER BY cfg_id" % DST,
                (dst_uid,),
            )
            love_txt = "、".join("%s剩%d天" % (r["name"], int(r["total_days"]) - int(r["claimed_days"]))
                                for r in love_after) or "无"
            print("\n[6/7] 完成 ✅  二区目标当前：钻石=%d  装备=%d 件  道具=%d 种/%d 个  发电卡=%s"
                  % (prof_after["diamond"], eq_after["n"], it_after["n"], it_after["s"], love_txt))
            print("\n提示：目标玩家在二区**重新登录/刷新**后即可看到新资产。")
            return 0
    finally:
        conn.close()


if __name__ == "__main__":
    sys.exit(main())
