#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
二战风云（ezfy）误删背包道具恢复脚本

背景（2026-10-10 严重 bug）
    ezfyMigrateTreasureBag 本意是把历史「宝物签到」误发到道具表(ezfy_item)的残留转回装备表，
    但它用 sync.Once 包裹、每次进程重启后首个请求都会重跑，而 ezfy_item 的 cfg_id 区间(27~35)
    已被真实道具（为爱发电高级卡 27、建筑/训练/科技加速 28~35）占用，与装备(珠宝)配置 ID 重叠，
    导致玩家手里的真实加速道具每次重启都被当成「误发宝物」删掉并换成了珠宝装备。

    本脚本依据钻石商城购买流水，为受影响玩家补齐缺失的加速道具：
        应补数量 = Σ 该玩家该道具的钻石购买件数 − 当前背包剩余数
    购买件数由 ezfy_diamond_logs(reason='商城购买: 道具名' 且 change<0)的扣钻额 ÷ PriceDiamond 还原。

范围
    只处理真实加速道具 cfg_id 28~35（为爱发电卡是单独模块，不在此列，不处理）。

安全设计（与 clear_ezfy_player_data.py 一致）
    - 默认【干跑】：只统计每个玩家每种道具应补多少，一行不改。
    - 真正执行加 --execute；连接非本机库时要求手动输入库名二次确认。
    - 执行前请先备份：mysqldump -u root -p qq_jiayuan > backup_$(date +%F).sql

用法
    # 1) 干跑（先看会补哪些玩家哪些道具、各补几个）
    python3 restore_ezfy_missing_items.py

    # 2) 切线上库真正执行（会要求输入库名确认）
    python3 restore_ezfy_missing_items.py --execute

    # 3) 脚本化 / 免交互（确认无误再用）
    python3 restore_ezfy_missing_items.py --execute --force
"""

import argparse
import sys

try:
    import pymysql
except ImportError:
    sys.exit("缺少依赖 pymysql，请先安装：pip3 install pymysql")


# ============================ 数据库配置（与清档脚本保持一致）============================
ACTIVE_DB = "test"

DATABASES = {
    # 测试库：先在测试库跑通，确认没问题再切线上
    "test": {
        "host": "127.0.0.1",
        "port": 3306,
        "user": "root",
        "password": "1234567890",  # ← 改成测试库 MySQL 密码
        "dbname": "qq_jiayuan",
        "charset": "utf8mb4",
    },
    # 线上库：测试通过后，把下面几项换成线上真实连接信息
    "prod": {
        "host": "39.105.151.141",
        "port": 3306,
        "user": "root",
        "password": "Mzd980625@@..",
        "dbname": "qq_jiayuan",
        "charset": "utf8mb4",
    },
}

# 加速道具范围（cfg_id）——对应 ezfy_cfg_item 中建筑/训练/科技加速；为爱发电卡不在此列
ITEM_IDS = list(range(28, 36))


def connect(db):
    return pymysql.connect(
        host=db["host"], port=db["port"], user=db["user"], password=db["password"],
        database=db["dbname"], charset=db.get("charset", "utf8mb4"),
        autocommit=True, cursorclass=pymysql.cursors.Cursor,
    )


def load_item_cfgs(cur):
    """cfg_id -> {name, price_diamond}，只取加速道具范围。价格用于从流水还原购买件数。"""
    cur.execute(
        "SELECT id, name, price_diamond FROM ezfy_cfg_item WHERE id BETWEEN %s AND %s" %
        (ITEM_IDS[0], ITEM_IDS[-1])
    )
    out = {}
    for cid, name, price in cur.fetchall():
        out[cid] = {"name": name, "price": price}
    return out


def load_purchases(cur, cfgs):
    """返回 {user_id: {cfg_id: 购买件数}}，依据钻石商城购买流水。"""
    # 每条 reason 是「商城购买: 道具名」（钻石路径在 h.logDiamond(uid,-cost,"商城购买: "+cfg.Name)）。
    # 一次购买可能多件，流水只记总额 -cost；件数 = cost / PriceDiamond。
    placeholders = ",".join(["%s"] * len(cfgs))
    sql = (
        "SELECT user_id, change, reason FROM ezfy_diamond_logs "
        "WHERE reason LIKE '商城购买: %%' AND change < 0"
    )
    cur.execute(sql)
    result = {}
    for uid, change, reason in cur.fetchall():
        name = reason.replace("商城购买: ", "").strip()
        for cid, cfg in cfgs.items():
            if cfg["name"] == name and cfg["price"] > 0:
                cnt = int((-change) // cfg["price"])  # 向下取整，避免浮点误差
                if cnt <= 0:
                    continue
                result.setdefault(uid, {})
                result[uid][cid] = result[uid].get(cid, 0) + cnt
                break
    return result


def load_held(cur):
    """返回 {user_id: {cfg_id: 现存背包数}}。"""
    cur.execute(
        "SELECT user_id, cfg_id, count FROM ezfy_item WHERE cfg_id BETWEEN %s AND %s" %
        (ITEM_IDS[0], ITEM_IDS[-1])
    )
    result = {}
    for uid, cid, cnt in cur.fetchall():
        result.setdefault(uid, {})
        result[uid][cid] = cnt
    return result


def main():
    parser = argparse.ArgumentParser(description="二战风云误删背包道具恢复脚本")
    parser.add_argument("--execute", action="store_true", help="真正补发道具（不加则只干跑统计）")
    parser.add_argument("--force", action="store_true", help="跳过二次确认（需配合 --execute）")
    parser.add_argument("--db", choices=["test", "prod"], help="临时覆盖脚本顶部的 ACTIVE_DB")
    args = parser.parse_args()

    key = args.db or ACTIVE_DB
    if key not in DATABASES:
        sys.exit("未知的库标识：%s（只支持 test / prod）" % key)
    db = DATABASES[key]

    print("=" * 68)
    print("二战风云误删背包道具恢复脚本")
    print("目标库：%s@%s:%s/%s  （配置键：%s）" % (
        db["user"], db["host"], db["port"], db["dbname"], key))
    print("模式  ：%s" % ("【真正执行·会补发道具】" if args.execute else "【干跑·只看不改】"))
    print("=" * 68)

    try:
        conn = connect(db)
    except Exception as e:
        sys.exit("连接数据库失败：%s" % e)

    try:
        with conn.cursor() as cur:
            cfgs = load_item_cfgs(cur)
            if not cfgs:
                sys.exit("未在 ezfy_cfg_item 找到加速道具配置（cfg 28~35），疑似库不正确。")

            purchases = load_purchases(cur, cfgs)
            held = load_held(cur)

            # 合并所有涉及玩家
            users = set(purchases) | set(held)
            # {user_id: {cfg_id: 应补数}}，>0 才补
            to_fix = {}
            for uid in sorted(users):
                buy = purchases.get(uid, {})
                own = held.get(uid, {})
                for cid in ITEM_IDS:
                    got = buy.get(cid, 0)
                    have = own.get(cid, 0)
                    if got - have > 0:
                        to_fix.setdefault(uid, {})[cid] = got - have

            total_user = len(to_fix)
            total_piece = sum(v for m in to_fix.values() for v in m.values())
            print("\n命中 %d 个玩家，共需补发 %d 件加速道具。\n" % (total_user, total_piece))
            print("明细（玩家ID → 道具cfg_id → 购买件数/现存/应补）：")
            print("-" * 68)
            for uid in sorted(to_fix):
                row = []
                for cid, need in to_fix[uid].items():
                    name = cfgs[cid]["name"]
                    got = purchases[uid].get(cid, 0)
                    have = held.get(uid, {}).get(cid, 0)
                    row.append("%s(cfg%d) 买%d剩%d补%d" % (name, cid, got, have, need))
                print("玩家 %d：%s" % (uid, "；".join(row)))
            if not to_fix:
                print("无需恢复：没有任何玩家存在缺失。")
                return
            print("-" * 68)

            if not args.execute:
                print("\n当前是【干跑】模式，未做任何修改。")
                print("确认无误后加 --execute 真正补发：python3 %s --execute" % sys.argv[0])
                return

            # ---- 二次确认 ----
            is_local = db["host"] in ("127.0.0.1", "localhost", "::1")
            if not args.force:
                if not is_local:
                    print("\n⚠️  注意：目标不是本机数据库（%s），请再次确认这是要补发的库！" % db["host"])
                tip = ("请输入库名 %s 以确认补发（其它输入将取消）：" % db["dbname"])
                try:
                    answer = input(tip).strip()
                except EOFError:
                    answer = ""
                if answer != db["dbname"]:
                    print("已取消，未做任何修改。")
                    return

            # ---- 真正补发：向 ezfy_item 加回缺失数量 ----
            print("\n开始补发 …")
            success = 0
            for uid in sorted(to_fix):
                for cid, need in to_fix[uid].items():
                    # 若已有该 cfg 行则累加，否则插入
                    cur.execute(
                        "SELECT id FROM ezfy_item WHERE user_id=%s AND cfg_id=%s LIMIT 1", (uid, cid))
                    row = cur.fetchone()
                    try:
                        if row:
                            cur.execute(
                                "UPDATE ezfy_item SET count=count+%s WHERE id=%s", (need, row[0]))
                        else:
                            cur.execute(
                                "INSERT INTO ezfy_item (user_id, cfg_id, count, created_at) "
                                "VALUES (%s, %s, %s, NOW())", (uid, cid, need))
                        success += 1
                    except Exception as e:
                        print("  ✗ 玩家%s 道具%s 补发失败：%s" % (uid, cid, e))
            print("\n补发完成，共处理 %d 条（道具行）。" % success)
            print("建议：补发后再确认一次玩家背包道具与钻石流水一致。")
    finally:
        conn.close()


if __name__ == "__main__":
    main()