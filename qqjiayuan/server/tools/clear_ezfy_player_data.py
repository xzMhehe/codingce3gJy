#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
二战风云（ezfy）玩家数据清档脚本

用途
    删档测试前清空所有二战风云玩家数据，玩家从家园重新跳转进来时按「新玩家」处理
    （没有城市 → 自动走 createMainCity → findFreePos，默认仍落欧洲）。

清什么
    只清「玩家产生的数据」：档案/城市/建筑/部队/科技/订单/战报/野地/军团/背包/
    军官/装备/交易行/好友/聊天/地图动态格子 …（见 CLEAR_TABLES）。
    配置表、世界地图覆盖表、活动、敏感词、交易行模板一律不动（见 KEEP_TABLES）。
    ----------
        家园登录账号（users）、游戏大厅（games / my_games）不受影响，玩家还能正常登录进游戏。

安全设计
    - 默认【干跑】：只统计各表会删多少行，一行都不删。
    - 真正执行要加 --execute；非本机库还要求手动输入库名二次确认，防手滑删线上。
    - 执行前请先备份：mysqldump -u root -p qq_jiayuan > backup_$(date +%F).sql

用法
    # 1) 干跑（先看会清哪些表、各多少行）
    python3 clear_ezfy_player_data.py

    # 2) 切测试库真正执行（会要求输入库名确认）
    python3 clear_ezfy_player_data.py --execute

    # 3) 脚本化 / 免交互（确认无误再用）
    python3 clear_ezfy_player_data.py --execute --force

切换测试库 / 线上库
    只改下面 ACTIVE_DB 一行即可（"test" / "prod"），也可命令行 --db prod 临时覆盖。
"""

import argparse
import sys

try:
    import pymysql
except ImportError:
    sys.exit("缺少依赖 pymysql，请先安装：pip3 install pymysql")


# ============================ 数据库配置（替换这里即可切库）============================
# 要跑哪个库，只改这一行："test" = 测试库  /  "prod" = 线上库
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


# ============================ 要清空的「玩家数据表」============================
# TRUNCATE：清空数据并把自增 ID 归零（删档测试要的就是「跟新服一样」）
CLEAR_TABLES = [
    "ezfy_profile",         # 玩家档案（等级 / 声望 / 资源 / 钻石）
    "ezfy_city",            # 城市
    "ezfy_city_building",   # 城内建筑
    "ezfy_city_troop",      # 城内部队
    "ezfy_city_tech",       # 已研究科技
    "ezfy_train_queue",     # 造兵队列
    "ezfy_map_area",        # 地图动态格子（野地占领 / 寇城废墟复活计时 / 资源田）
    "ezfy_order",           # 出征订单
    "ezfy_battle",          # 实时战斗战场
    "ezfy_report",          # 战报
    "ezfy_wildland",        # 已占野地
    "ezfy_occupy",          # 被占领的玩家城
    "ezfy_wounded",         # 伤兵 / 逃兵
    "ezfy_war",             # 宣战记录
    "ezfy_corps",           # 军团
    "ezfy_corps_member",    # 军团成员
    "ezfy_corps_chat",      # 军团聊天
    "ezfy_corps_relation",  # 军团外交
    "ezfy_corps_war",       # 军团宣战
    "ezfy_corps_mall",      # 军团商城
    "ezfy_corps_mall_log",  # 军团商城记录
    "ezfy_item",            # 背包道具
    "ezfy_sign",            # 签到
    "ezfy_gift",            # 礼包 / 领取记录
    "ezfy_city_effect",     # 城市增益
    "ezfy_city_target",     # 城市目标
    "ezfy_task",            # 任务进度
    "ezfy_chat",            # 世界 / 私聊
    "ezfy_exchange",        # 交易行挂单
    "ezfy_officer",         # 军官
    "ezfy_equipment",       # 装备
    "ezfy_recruit",         # 军校招募记录
    "ezfy_map_star",        # 地图收藏
    "ezfy_friend",          # 游戏内好友
    "ezfy_friend_apply",    # 游戏内好友申请
]

# 只删一部分的表：(表名, 删除条件)
# ezfy_notice 既有全服公告(user_id=0，保留)，也有发给个人的通知(user_id<>0，清掉)
PARTIAL_CLEAR = [
    ("ezfy_notice", "user_id <> 0"),
]

# 保留不动（配置 / 世界地图覆盖 / 敏感词 / 活动 / 交易行模板）—— 只打印提示，不做任何删除
KEEP_TABLES = [
    "ezfy_cfg_building", "ezfy_cfg_building_level", "ezfy_cfg_troop",
    "ezfy_cfg_tech", "ezfy_cfg_tech_level", "ezfy_cfg_wildland",
    "ezfy_cfg_item", "ezfy_cfg_task_type", "ezfy_cfg_task",
    "ezfy_cfg_general", "ezfy_cfg_skill", "ezfy_cfg_equipment",
    "ezfy_cfg_equip_set", "ezfy_cfg_chest", "ezfy_cfg_chest_item",
    "ezfy_cfg_scheme", "ezfy_cfg_resource", "ezfy_cfg_rank", "ezfy_cfg_limit",
    "ezfy_word_filter", "ezfy_exchange_tpl", "ezfy_activity", "ezfy_map_tile",
]


def connect(db):
    return pymysql.connect(
        host=db["host"], port=db["port"], user=db["user"], password=db["password"],
        database=db["dbname"], charset=db.get("charset", "utf8mb4"),
        autocommit=True, cursorclass=pymysql.cursors.Cursor,
    )


def existing_tables(cur):
    cur.execute("SHOW TABLES")
    return {row[0] for row in cur.fetchall()}


def count_rows(cur, table, where=None):
    sql = "SELECT COUNT(*) FROM `%s`" % table
    if where:
        sql += " WHERE " + where
    cur.execute(sql)
    return cur.fetchone()[0]


def truncate(cur, table):
    cur.execute("TRUNCATE TABLE `%s`" % table)


def build_plan(cur, tables):
    """返回 [(表名, 动作说明, 影响行数)]，只包含库里真实存在的表。"""
    plan = []
    for t in CLEAR_TABLES:
        if t in tables:
            plan.append((t, "TRUNCATE 整表清空", count_rows(cur, t)))
    for t, where in PARTIAL_CLEAR:
        if t in tables:
            plan.append((t, "DELETE WHERE " + where, count_rows(cur, t, where)))
    return plan


def main():
    parser = argparse.ArgumentParser(description="二战风云玩家数据清档脚本")
    parser.add_argument("--execute", action="store_true", help="真正删除数据（不加则只干跑统计）")
    parser.add_argument("--force", action="store_true", help="跳过二次确认（需配合 --execute）")
    parser.add_argument("--db", choices=["test", "prod"], help="临时覆盖脚本顶部的 ACTIVE_DB")
    args = parser.parse_args()

    key = args.db or ACTIVE_DB
    if key not in DATABASES:
        sys.exit("未知的库标识：%s（只支持 test / prod）" % key)
    db = DATABASES[key]

    print("=" * 68)
    print("二战风云玩家数据清档脚本")
    print("目标库：%s@%s:%s/%s  （配置键：%s）" % (
        db["user"], db["host"], db["port"], db["dbname"], key))
    print("模式  ：%s" % ("【真正执行·会删数据】" if args.execute else "【干跑·只看不删】"))
    print("=" * 68)

    try:
        conn = connect(db)
    except Exception as e:
        sys.exit("连接数据库失败：%s" % e)

    try:
        with conn.cursor() as cur:
            tables = existing_tables(cur)
            plan = build_plan(cur, tables)
            missing = [t for t in CLEAR_TABLES if t not in tables]
            missing += [t for t, _ in PARTIAL_CLEAR if t not in tables]

            if not plan:
                print("库里没有找到任何 ezfy 玩家数据表，无需清理。")
                return

            total = sum(n for _, _, n in plan)
            print("\n将清理以下 %d 张表，共 %d 行：" % (len(plan), total))
            print("-" * 68)
            print("%-26s %-24s %s" % ("表名", "动作", "行数"))
            print("-" * 68)
            for t, action, n in plan:
                print("%-26s %-24s %d" % (t, action, n))
            print("-" * 68)
            print("合计：%d 行\n" % total)

            if missing:
                print("提示：以下表在库中不存在，已跳过 —— %s\n" % ", ".join(missing))

            print("以下配置 / 世界表保留不动：")
            print("  %s" % ", ".join(t for t in KEEP_TABLES if t in tables))
            print("（另外：users 登录账号、games / my_games 游戏大厅均不受影响）\n")

            if not args.execute:
                print("当前是【干跑】模式，未删除任何数据。")
                print("确认无误后加 --execute 真正执行：python3 %s --execute" % sys.argv[0])
                return

            # ---- 二次确认：非本机库要求手动输入库名 ----
            is_local = db["host"] in ("127.0.0.1", "localhost", "::1")
            if not args.force:
                if not is_local:
                    print("⚠️  注意：目标不是本机数据库（%s），请再次确认这是你要清的库！" % db["host"])
                tip = ("请输入库名 %s 以确认清空（其它输入将取消）：" % db["dbname"])
                try:
                    answer = input(tip).strip()
                except EOFError:
                    answer = ""
                if answer != db["dbname"]:
                    print("已取消，未做任何修改。")
                    return

            # ---- 真正执行 ----
            print("\n开始清空 …")
            cur.execute("SET FOREIGN_KEY_CHECKS = 0")
            for t, action, _ in plan:
                try:
                    if action.startswith("TRUNCATE"):
                        truncate(cur, t)
                    else:
                        where = action[len("DELETE WHERE "):]
                        cur.execute("DELETE FROM `%s` WHERE %s" % (t, where))
                    print("  ✓ %-26s %s" % (t, action))
                except Exception as e:
                    print("  ✗ %-26s 失败：%s" % (t, e))
            cur.execute("SET FOREIGN_KEY_CHECKS = 1")
            print("\n清理完成，共处理 %d 张表。玩家再进游戏即按新玩家重新建城。" % len(plan))
    finally:
        conn.close()


if __name__ == "__main__":
    main()
