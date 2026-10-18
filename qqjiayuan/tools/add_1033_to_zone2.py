# -*- coding: utf-8 -*-
import argparse
import json
import os
import sys
import time

import pymysql


def connect(args):
    return pymysql.connect(
        host=args.host, port=args.port, user=args.user, password=args.password,
        charset="utf8mb4", autocommit=False, cursorclass=pymysql.cursors.DictCursor,
    )


def q1(cur, sql, params=None):
    cur.execute(sql, params or ())
    return cur.fetchone()


def main():
    ap = argparse.ArgumentParser(description="补迁 15:47 后一区新增训练加速1033 → 二区（移动语义）")
    ap.add_argument("game_id", type=int)
    ap.add_argument("--from-db", default="qq_ezzt")
    ap.add_argument("--to-db", default="qq_ezzt2")
    ap.add_argument("--host", default="127.0.0.1")
    ap.add_argument("--port", type=int, default=3306)
    ap.add_argument("--user", default="root")
    ap.add_argument("--password", default="1234567890")
    ap.add_argument("--execute", action="store_true")
    args = ap.parse_args()
    uid = args.game_id
    CFG = 1033  # 训练加速80%

    conn = connect(args)
    try:
        with conn.cursor() as cur:
            src = q1(cur, "SELECT cfg_id, count FROM `%s`.ezfy_item WHERE user_id=%%s AND cfg_id=%%s" % args.from_db, (uid, CFG))
            dst = q1(cur, "SELECT cfg_id, count FROM `%s`.ezfy_item WHERE user_id=%%s AND cfg_id=%%s" % args.to_db, (uid, CFG))
            src_n = int(src["count"]) if src else 0
            dst_n = int(dst["count"]) if dst else 0
            print("一区(%s) 1033=%d  二区(%s) 1033=%d  补迁量=%d → 二区变 %d"
                  % (args.from_db, src_n, args.to_db, dst_n, src_n, dst_n + src_n))
            if src_n == 0:
                print("[跳过] 一区没有 1033，无需补迁。")
                return 0
            if not args.execute:
                print("[预览结束] 加 --execute 执行。")
                return 0

            if not os.path.exists(os.path.join(os.path.dirname(os.path.abspath(__file__)), "backups")):
                os.makedirs(os.path.join(os.path.dirname(os.path.abspath(__file__)), "backups"))
            bpath = os.path.join(os.path.dirname(os.path.abspath(__file__)), "backups",
                                 "ezfy_add_1033_%d_%s.json" % (uid, time.strftime("%Y%m%d_%H%M%S")))
            with open(bpath, "w", encoding="utf-8") as f:
                json.dump({"game_id": uid, "cfg_id": CFG, "src_count": src_n,
                           "dst_before": dst_n, "taken_at": time.strftime("%Y-%m-%d %H:%M:%S")},
                          f, ensure_ascii=False, indent=1)
            print("备份:", bpath)

            try:
                conn.begin()
                # 二区累加
                cur.execute(
                    "INSERT INTO `%s`.ezfy_item (user_id, cfg_id, count) VALUES (%%s,%%s,%%s) "
                    "ON DUPLICATE KEY UPDATE count = count + VALUES(count)" % args.to_db,
                    (uid, CFG, src_n),
                )
                # 一区清零（移动）
                cur.execute("UPDATE `%s`.ezfy_item SET count = 0 WHERE user_id=%%s AND cfg_id=%%s" % args.from_db,
                            (uid, CFG))
                conn.commit()
            except Exception as e:
                conn.rollback()
                print("[失败] %s" % e)
                return 4

            dst2 = q1(cur, "SELECT count FROM `%s`.ezfy_item WHERE user_id=%%s AND cfg_id=%%s" % args.to_db, (uid, CFG))
            src2 = q1(cur, "SELECT count FROM `%s`.ezfy_item WHERE user_id=%%s AND cfg_id=%%s" % args.from_db, (uid, CFG))
            print("完成 ✅  二区 1033=%d  一区 1033=%d" % (
                int(dst2["count"]) if dst2 else 0, int(src2["count"]) if src2 else 0))
            return 0
    finally:
        conn.close()


if __name__ == "__main__":
    sys.exit(main())
