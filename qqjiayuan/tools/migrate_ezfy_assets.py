#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
二战一区 → 二区 资产迁移：钻石余额 + 全部装备 + 背包道具

用法
----
    # 1) 预览（默认只读，绝不写库）
    python migrate_ezfy_assets.py 35806360

    # 2) 真正执行
    python migrate_ezfy_assets.py 35806360 --execute

    # 3) 线上库（RDS）示例
    python migrate_ezfy_assets.py 35806360 \
        --host rm-xxx.mysql.rds.aliyuncs.com --user root --password 'xxx' --execute

参数
----
    game_id          家园号（= 二战 game_id）
    --from-db        源库（一区），默认 qq_ezzt
    --to-db          目标库（二区），默认 qq_ezzt2
    --host/--port/--user/--password   MySQL 连接，默认 127.0.0.1:3306 root/1234567890
    --execute        真正写入；不加只预览
    --force          忽略「已迁移过」检查强制再迁（会重复累加钻石/道具，慎用）
    --no-backup      不导出备份 JSON

做了什么
--------
    1. 校验二区有没有这个账号（ezfy_account.game_id）+ 有没有城市（拿主城 city_id）
    2. 一区读取：ezfy_profile.diamond、ezfy_equipment 全部行、ezfy_item 全部行
    3. 二区写入（单个事务）：
         · 钻石   ezfy_profile.diamond += 一区余额（累加，不清空二区原有）
         · 装备   逐行复制到 ezfy_equipment，**city_id 写 0**、
                 officer_id / original_user_id 归零（二区没有对应军官，穿戴状态不保留）
         · 道具   ezfy_item 按 (user_id,cfg_id) 累加（ON DUPLICATE KEY UPDATE）
         · 流水   写一条 ezfy_diamond_logs（reason='一区资产迁移'）兼作「已迁移」标记
    4. 执行前把一区原始数据导出成 JSON 备份（默认写到脚本同目录 backups/）

★★ 2026-10-18 装备与城市解耦（重要）
--------------------------------------
    装备只认 `user_id`（账号）+ `officer_id`（穿戴者，0=背包），**和城市没有任何关系**。
    一区的 `ezfy_equipment.city_id` 是历史「出生城」，已全量清零并废弃（恒 0）；
    所以迁移时**必须写 0**，不能再映射到二区主城 —— 否则一旦那座城被摧毁，
    旧逻辑 `DELETE FROM ezfy_equipment WHERE city_id = ?` 又会把刚迁过去的装备删掉。

安全设计
--------
    · 默认 dry-run，必须显式 --execute 才写库
    · 全部写入在一个事务里，中途报错整体回滚
    · 已迁移过会拒绝（除非 --force），避免重复加钻石
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

# 采集宝物名单（与后端 ezfyCollectibleTreasureNames 同一套，cfg 27~35）。
# ★★ 2026-10-18 用户规则：**采集宝物不迁移** —— 宝物是逐日采集攒出来的，
#     二区从零开始采集；迁过去反而直接顶满 treasure_max(5000) 上限。
#     按名字过滤（与后端判定一致），不按 id 段。
TREASURE_NAMES = {
    "黄金手镯", "玛瑙项坠", "红宝石戒指", "黑曜石戒指", "琥珀项链",
    "铂金戒指", "翡翠项链", "祖母绿", "蓝宝石戒指",
}


# ---------------------------------------------------------------- 连接
def connect(args):
    return pymysql.connect(
        host=args.host,
        port=args.port,
        user=args.user,
        password=args.password,
        charset="utf8mb4",
        autocommit=False,
        cursorclass=pymysql.cursors.DictCursor,
    )


def q1(cur, sql, params=None):
    cur.execute(sql, params or ())
    return cur.fetchone()


def qall(cur, sql, params=None):
    cur.execute(sql, params or ())
    return cur.fetchall()


def cfg_ids(cur, schema, table, ids):
    """返回这批 id 里在目标库里**存在**的集合。"""
    ids = [i for i in set(ids) if i is not None]
    if not ids:
        return set()
    ph = ",".join(["%s"] * len(ids))
    rows = qall(cur, "SELECT id FROM `%s`.`%s` WHERE id IN (%s)" % (schema, table, ph), ids)
    return {r["id"] for r in rows}


def table_exists(cur, schema, table):
    """目标库里有没有这张表（老库可能缺后加的表，缺了就跳过而不是整个崩）。"""
    return bool(q1(
        cur,
        "SELECT 1 AS ok FROM information_schema.tables "
        "WHERE table_schema=%s AND table_name=%s LIMIT 1",
        (schema, table),
    ))


# ---------------------------------------------------------------- 主流程
def main():
    ap = argparse.ArgumentParser(description="二战一区 → 二区 资产迁移")
    ap.add_argument("game_id", type=int, help="家园号（= 二战 game_id）")
    ap.add_argument("--from-db", default="qq_ezzt", help="源库（一区），默认 qq_ezzt")
    ap.add_argument("--to-db", default="qq_ezzt2", help="目标库（二区），默认 qq_ezzt2")
    ap.add_argument("--host", default="127.0.0.1")
    ap.add_argument("--port", type=int, default=3306)
    ap.add_argument("--user", default="root")
    ap.add_argument("--password", default="1234567890")
    ap.add_argument("--execute", action="store_true", help="真正写入（不加只预览）")
    ap.add_argument("--force", action="store_true", help="忽略已迁移检查（会重复累加，慎用）")
    ap.add_argument("--no-backup", action="store_true", help="不导出备份 JSON")
    args = ap.parse_args()

    uid = args.game_id
    SRC, DST = args.from_db, args.to_db
    print("=" * 68)
    print("二战资产迁移  game_id=%d" % uid)
    print("  源库(一区) = %s" % SRC)
    print("  目标库(二区) = %s" % DST)
    print("  模式 = %s" % ("执行（会写库）" if args.execute else "预览（只读，不写）"))
    print("=" * 68)

    conn = connect(args)
    try:
        with conn.cursor() as cur:
            # ---------- 1. 校验二区账号 ----------
            acct = q1(cur, "SELECT game_id, nickname FROM `%s`.ezfy_account WHERE game_id=%%s" % DST, (uid,))
            if not acct:
                print("\n[中止] 二区(%s) 没有该账号（ezfy_account.game_id=%d）" % (DST, uid))
                print("       请先让玩家在二区登录一次建档，再执行迁移。")
                return 2
            print("\n[1/6] 二区账号 ✅  %d / %s" % (acct["game_id"], acct["nickname"]))

            prof2 = q1(cur, "SELECT user_id, diamond FROM `%s`.ezfy_profile WHERE user_id=%%s" % DST, (uid,))
            if not prof2:
                print("[中止] 二区有账号但没有 ezfy_profile（档案不完整），请先让玩家登录一次。")
                return 2
            city2 = q1(cur, "SELECT id, name FROM `%s`.ezfy_city WHERE user_id=%%s ORDER BY id LIMIT 1" % DST, (uid,))
            # ★ 2026-10-18 装备/道具/钻石/发电卡**都不挂城市**（装备 city_id 恒 0），
            #   所以二区还没建城也能迁 —— 玩家进游戏时后端会自动建主城。
            if city2:
                print("        二区档案 ✅  当前钻石=%d  主城 id=%d(%s)" % (prof2["diamond"], city2["id"], city2["name"]))
            else:
                print("        二区档案 ✅  当前钻石=%d（还没有城市：资产不挂城市，照常迁移；"
                      "玩家进游戏后自动建主城）" % prof2["diamond"])

            # ---------- 2. 防重复 ----------
            dup = q1(
                cur,
                "SELECT id, created_at FROM `%s`.ezfy_diamond_logs "
                "WHERE user_id=%%s AND reason=%%s LIMIT 1" % DST,
                (uid, "一区资产迁移"),
            )
            if dup and not args.force:
                print("\n[中止] 该玩家**已经迁移过**（流水 id=%s，时间=%s）" % (dup["id"], dup["created_at"]))
                print("       再跑一次会重复加钻石/道具。确实要重迁请加 --force（自己承担重复风险）。")
                return 3

            # ---------- 3. 读一区 ----------
            prof1 = q1(cur, "SELECT user_id, diamond FROM `%s`.ezfy_profile WHERE user_id=%%s" % SRC, (uid,))
            if not prof1:
                print("\n[中止] 一区(%s) 没有该玩家档案（没在一区玩过）。" % SRC)
                return 2
            diamond1 = int(prof1["diamond"] or 0)

            # ★ 装备/道具/发电卡三张表老库可能缺（尤其二区新开服）→ 缺表就按空处理，不整段崩
            eq_all = (qall(cur, "SELECT * FROM `%s`.ezfy_equipment WHERE user_id=%%s" % SRC, (uid,))
                      if table_exists(cur, SRC, "ezfy_equipment") else [])
            # ★★ 2026-10-18 用户规则：**采集宝物不迁移**（二区从零开始采集）。
            #   采集宝物 = 装备表里那 9 种珠宝（cfg 27~35，与后端 ezfyCollectibleTreasureNames 同一名单）。
            #   按**名字**过滤而不是按 id 段 —— 与后端口径一致，id 段变了也不受影响。
            eqs = [e for e in eq_all if (e.get("name") or "") not in TREASURE_NAMES]
            treasure_cnt = len(eq_all) - len(eqs)
            items = (qall(cur, "SELECT cfg_id, count FROM `%s`.ezfy_item WHERE user_id=%%s ORDER BY cfg_id" % SRC, (uid,))
                     if table_exists(cur, SRC, "ezfy_item") else [])
            # ★ 为爱发电卡：独立表 ezfy_love_card（不在道具体系）。迁移口径 = **剩余未领天数**累加：
            #   已领的天数对应钻石玩家早就拿到了，不能再给；只把 (total_days - claimed_days) 搬过去。
            #   同 cfg_id 有多张历史卡 → 剩余天数合并后按类型处理一次。
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
            print("\n[2/6] 一区数据 ✅  钻石=%d  装备=%d 件（另有采集宝物 %d 件**不迁移**）  道具=%d 种  为爱发电卡=%d 种/剩 %d 天"
                  % (diamond1, len(eqs), treasure_cnt, len(items), len(love_plan), love_days))

            if diamond1 == 0 and not eqs and not items and not love_plan:
                print("\n[跳过] 一区这几样都是空的，没有可迁移的内容。")
                return 0

            # ---------- 4. 校验配置 id（两区配置表应一致，防插进不存在的 cfg） ----------
            eq_ok = cfg_ids(cur, DST, "ezfy_cfg_equipment", [e["cfg_id"] for e in eqs])
            eq_skip = [e for e in eqs if e["cfg_id"] not in eq_ok]
            it_ok = cfg_ids(cur, DST, "ezfy_cfg_item", [i["cfg_id"] for i in items])
            it_skip = [i for i in items if i["cfg_id"] not in it_ok]
            eq_use = [e for e in eqs if e["cfg_id"] in eq_ok]
            it_use = [i for i in items if i["cfg_id"] in it_ok]

            if eq_skip:
                print("  [告警] %d 件装备的 cfg_id 在二区配置表里不存在，将跳过：%s"
                      % (len(eq_skip), sorted({e["cfg_id"] for e in eq_skip})[:12]))
            if it_skip:
                print("  [告警] %d 种道具的 cfg_id 在二区配置表里不存在，将跳过：%s"
                      % (len(it_skip), [i["cfg_id"] for i in it_skip][:12]))

            # ---------- 5. 计划 ----------
            new_diamond = int(prof2["diamond"] or 0) + diamond1
            eq2_now = q1(cur, "SELECT COUNT(*) AS n FROM `%s`.ezfy_equipment WHERE user_id=%%s" % DST, (uid,)) \
                if table_exists(cur, DST, "ezfy_equipment") else {"n": 0}
            print("\n[3/6] 迁移计划")
            print("  钻石 : %d + %d = %d" % (prof2["diamond"], diamond1, new_diamond))
            # ★ city_id 一律写 0：装备是玩家背包资产，不挂城市（详见文件头 2026-10-18 说明）
            print("  装备 : %d 件 → 二区（city_id=0，officer_id/original_user_id 归零，穿戴状态不保留）"
                  % len(eq_use))
            if len(eq_use) > 4000:
                print("        ⚠️ 装备较多（%d 件）：宝物库存上限 treasure_max 默认 5000，"
                      "迁完若未穿戴宝物超过上限，**采集会暂停**（已有资产不受影响，消耗后自动恢复）" % len(eq_use))
            if int(eq2_now["n"] or 0) > 0:
                print("        ⚠️ 二区该玩家**已有 %d 件装备**：本次是全量复制、不去重，"
                      "迁完会变成 %d 件（若之前迁过又加了 --force，请先确认要不要先清二区装备）"
                      % (int(eq2_now["n"]), int(eq2_now["n"]) + len(eq_use)))
            print("  道具 : %d 种（同名道具在二区已有则数量累加）" % len(it_use))
            for cid, p in sorted(love_plan.items()):
                print("  发电卡: cfg_id=%d %s 剩 %d 天（%d 钻石/天，二区已有同类型未领完则续天数）"
                      % (cid, p["name"], p["remain"], p["daily"]))

            if not args.execute:
                print("\n[预览结束] 没有写任何数据。确认无误后加 --execute 执行。")
                return 0

            # ---------- 6. 备份 + 执行 ----------
            if not args.no_backup:
                bdir = os.path.join(os.path.dirname(os.path.abspath(__file__)), "backups")
                os.makedirs(bdir, exist_ok=True)
                bpath = os.path.join(bdir, "ezfy_migrate_%d_%s.json" % (uid, time.strftime("%Y%m%d_%H%M%S")))
                with open(bpath, "w", encoding="utf-8") as f:
                    json.dump(
                        {
                            "game_id": uid,
                            "from_db": SRC,
                            "to_db": DST,
                            "taken_at": time.strftime("%Y-%m-%d %H:%M:%S"),
                            "src_diamond": diamond1,
                            "dst_diamond_before": int(prof2["diamond"] or 0),
                            "equipment": eq_all,  # ★ 备份一区全量原始数据（含未迁移的宝物），便于以后对账
                            "items": items,
                            "love_cards": cards,
                        },
                        f,
                        ensure_ascii=False,
                        default=str,
                        indent=1,
                    )
                print("\n[4/6] 备份已导出：%s" % bpath)

            print("\n[5/6] 开始写入（单事务，出错整体回滚）…")
            try:
                conn.begin()

                # 6.1 钻石累加
                cur.execute(
                    "UPDATE `%s`.ezfy_profile SET diamond = diamond + %%s WHERE user_id=%%s" % DST,
                    (diamond1, uid),
                )

                # 6.2 装备：★ city_id 一律 0（装备不挂城市，与后端 buildEquipment 同口径）；
                #     officer_id / original_user_id 归零（二区没有对应军官，穿戴状态不保留）
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
                            uid if c == "user_id" else
                            0 if c in ("city_id", "officer_id", "original_user_id") else
                            e.get(c)
                            for c in cols
                        ))
                    for i in range(0, len(rows), 200):
                        cur.executemany(sql, rows[i:i + 200])

                # 6.3 道具：按 (user_id,cfg_id) 累加
                if it_use and table_exists(cur, DST, "ezfy_item"):
                    cur.executemany(
                        "INSERT INTO `%s`.ezfy_item (user_id, cfg_id, count) VALUES (%%s,%%s,%%s) "
                        "ON DUPLICATE KEY UPDATE count = count + VALUES(count)" % DST,
                        [(uid, i["cfg_id"], int(i["count"] or 0)) for i in it_use],
                    )
                elif it_use:
                    print("  [告警] 二区没有 ezfy_item 表，跳过 %d 种道具" % len(it_use))

                # 6.4 为爱发电卡：二区同类型还有未领天数的卡 → 续天数；否则新建（从今天起 remain 天）
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
                            (uid, cid),
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
                                (uid, cid, p["name"], p["daily"], now_ms,
                                 now_ms + p["remain"] * LOVE_DAY_MS, p["remain"]),
                            )

                # 6.5 钻石流水（同时作为「已迁移」标记）
                cur.execute(
                    "INSERT INTO `%s`.ezfy_diamond_logs (user_id, `change`, extra, balance, reason, created_at) "
                    "VALUES (%%s,%%s,%%s,%%s,%%s,NOW())" % DST,
                    (uid, diamond1, "", new_diamond, "一区资产迁移"),
                )

                conn.commit()
            except Exception as e:
                conn.rollback()
                print("\n[失败] 写入出错，已整体回滚：%s" % e)
                return 4

            # ---------- 7. 复核 ----------
            prof_after = q1(cur, "SELECT diamond FROM `%s`.ezfy_profile WHERE user_id=%%s" % DST, (uid,))
            eq_after = q1(cur, "SELECT COUNT(*) AS n FROM `%s`.ezfy_equipment WHERE user_id=%%s" % DST, (uid,))
            it_after = q1(cur, "SELECT COUNT(*) AS n, IFNULL(SUM(count),0) AS s FROM `%s`.ezfy_item WHERE user_id=%%s" % DST, (uid,))
            love_after = qall(
                cur,
                "SELECT name, total_days, claimed_days FROM `%s`.ezfy_love_card "
                "WHERE user_id=%%s AND claimed_days < total_days ORDER BY cfg_id" % DST,
                (uid,),
            )
            love_txt = "、".join("%s剩%d天" % (r["name"], int(r["total_days"]) - int(r["claimed_days"]))
                                for r in love_after) or "无"
            print("\n[6/6] 完成 ✅  二区当前：钻石=%d  装备=%d 件  道具=%d 种/%d 个  发电卡=%s"
                  % (prof_after["diamond"], eq_after["n"], it_after["n"], it_after["s"], love_txt))
            print("\n提示：玩家在二区**重新登录/刷新**后即可看到新资产。")
            return 0
    finally:
        conn.close()


if __name__ == "__main__":
    sys.exit(main())
