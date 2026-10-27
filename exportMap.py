#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
exportMap.py —— 把二战风云「线上一区」的世界地图导出成 Excel（一格一色，玩家城市紫色写「城」）。

为什么不能只 select 一张表：
  线上地图的**地形不落库**，是后端按坐标哈希推导出来的（见 server/internal/handler/ezfy/ezfy_geo.go）：
    ezfyTerrain(x,y)   = 先按七大洲几何判海/陆 → 陆地再用散列取 1~7
    ezfyTerrainEx(x,y) = ① ezfy_map_tile 里配了地形就以覆盖为准
                         ② 平原且 8 邻域有海洋 → 沿海平原(9)
                         ③ 海洋格 + 该格有野地 → 渔场(10)/矿礁(11)/海底森林(8)
  所以本脚本把 Go 的那套哈希**逐位复刻**（含 int64 溢出回绕、算术右移），
  再叠加线上 ezfy_map_tile 的人工覆盖，才等于玩家在游戏里真正看到的地图。

数据来源（只读，不改任何线上数据）：
  · ezfy_map_tile  —— 管理端「地图格子覆盖」（terrain > 0 才生效）
  · ezfy_city      —— 城市坐标 / 名字 / 等级 / 是否机器人城

用法：
  python exportMap.py                          # 默认导出一区 → 线上一区地图.xlsx
  python exportMap.py --out /tmp/map.xlsx      # 指定输出
  python exportMap.py --db qq_ezzt2 --zone 2   # 二区
  python exportMap.py --dump 234,96 231,97     # 只打印若干坐标的地形（对拍用，不写 Excel）
  python exportMap.py --no-excel               # 只跑统计不写文件

依赖：pymysql、XlsxWriter
  /Users/mxz/.workbuddy-ai/binaries/python/envs/default/bin/pip install pymysql XlsxWriter
"""

import argparse
import datetime
import os
import sys
import time

import pymysql
import xlsxwriter

# ---------------------------------------------------------------- 线上库连接
DB_HOST = os.environ.get("EZFY_DB_HOST", "rm-uf6vobze9179joev7ho.mysql.cn-shanghai.rds.aliyuncs.com")
DB_PORT = int(os.environ.get("EZFY_DB_PORT", "3306"))
DB_USER = os.environ.get("EZFY_DB_USER", "root")
DB_PASS = os.environ.get("EZFY_DB_PASS", "Mzd980625@@#")
DB_NAME = os.environ.get("EZFY_DB_NAME", "qq_ezzt")  # 一区；二区是 qq_ezzt2

# ---------------------------------------------------------------- 世界常量（与 Go 一致）
WORLD = 500          # ezfyWorldSize
MIN_C, MAX_C = 1, 499  # 游戏有效坐标 1~499（前端 isMapOOB 判定 <1 或 >499 是空地）
LAND_SCALE = 150.0 / float(WORLD)

T_OCEAN = 0
T_PLAIN, T_GRASS, T_FOREST, T_BASIN, T_HILL, T_SWAMP, T_ISLAND = 1, 2, 3, 4, 5, 6, 7
T_SEA, T_COASTAL = 8, 9
T_FISHING, T_OREEF = 10, 11

TERRAIN_NAMES = {
    0: "大海", 1: "平原", 2: "草原", 3: "森林", 4: "盆地", 5: "丘陵", 6: "沼泽",
    7: "岛屿", 8: "海洋", 9: "沿海平原", 10: "渔场", 11: "矿礁",
}

# 配色：海洋/森林等按常识取色；城市按要求用紫色。
# 括号里是 (底色, 字色)；只填色不写字，城市格写「城」。
COLORS = {
    1:  ("#E4D59A", None),   # 平原      —— 麦黄
    2:  ("#A8D08D", None),   # 草原      —— 草绿
    3:  ("#2E7D32", None),   # 森林      —— 深绿
    4:  ("#C2A878", None),   # 盆地      —— 土褐
    5:  ("#9C8B5E", None),   # 丘陵      —— 橄榄褐
    6:  ("#5F7A61", None),   # 沼泽      —— 墨绿
    7:  ("#EFDFAE", None),   # 岛屿      —— 沙黄
    8:  ("#3C7FC0", None),   # 海洋      —— 海蓝
    9:  ("#8FD3C1", None),   # 沿海平原  —— 海岸青
    10: ("#35A7C7", None),   # 渔场      —— 亮青蓝
    11: ("#6C7F9B", None),   # 矿礁      —— 灰蓝
}
CITY_BG = "#8E44AD"          # 玩家城市 —— 紫
BOT_CITY_BG = "#C39BD3"      # 机器人城市 —— 浅紫（图例里单独说明）

# 格子边长（像素）。Excel 里列宽是「字符数」、行高是「磅」，两者单位不同，
# 想画出正方形必须换算：像素宽 = round(列宽×7) + 5，像素高 = 行高×4/3（96 DPI 下）。
# ⚠️ xlsxwriter 的 set_column(width) 自己还会再加 5px padding 后再写盘
#    （写盘值 = (width×7+5)/7），所以传参要按 (像素-10)/7 反算，实测 22px → 1.7143。
CELL_PX = 22
COL_WIDTH = (CELL_PX - 10) / 7.0    # 1.7143 → 写盘 2.4258 → 渲染 22px
ROW_HEIGHT = CELL_PX * 0.75         # 16.5 磅 → 渲染 22px

# ---------------------------------------------------------------- 大陆几何（复刻 WorldGeo.java）
LANDS = [
    [[0, 3, 3, 40, 46], [0, 3, 3, 13, 10], [0, 30, 5, 44, 16], [0, 36, 12, 46, 30]],
    [[0, 33, 46, 44, 60], [1, 28, 74, 12, 14], [1, 38, 98, 24, 24], [0, 33, 60, 45, 72]],
    [[0, 50, 4, 86, 34], [0, 55, 2, 65, 9], [0, 46, 8, 50, 19]],
    [[0, 53, 32, 81, 84], [0, 62, 76, 79, 92], [0, 48, 76, 54, 80],
     [0, 26, 34, 34, 46], [1, 30, 60, 16, 12]],
    [[0, 78, 2, 150, 36], [0, 86, 36, 116, 58], [0, 112, 50, 128, 68],
     [0, 122, 34, 148, 60], [0, 140, 18, 150, 52], [0, 102, 20, 116, 30]],
    [[0, 116, 96, 141, 124], [0, 142, 124, 148, 127], [0, 118, 70, 138, 82],
     [0, 140, 78, 148, 88]],
    [[0, 12, 133, 138, 148]],
]
HOLES = [
    [[0, 13, 27, 25, 45], [0, 23, 21, 38, 32]],
    [],
    [[0, 36, 24, 72, 30], [0, 48, 30, 68, 38], [0, 80, 24, 84, 30]],
    [[0, 60, 40, 70, 52], [0, 72, 34, 78, 42]],
    [[0, 86, 22, 100, 34], [0, 92, 36, 108, 46]],
    [],
    [],
]
CONTINENT_IDS = [4, 5, 1, 3, 2, 6, 7]

I64 = 1 << 64
I64_MAX = (1 << 63) - 1


def i64(v):
    """按 Go int（64 位）语义回绕。Python 整数是无限精度，这里必须手动截断。"""
    v &= I64 - 1
    return v - I64 if v > I64_MAX else v


def in_shape(x, y, s):
    fx = x * LAND_SCALE
    fy = y * LAND_SCALE
    if s[0] == 0:
        return s[1] <= fx <= s[3] and s[2] <= fy <= s[4]
    dx = (fx - s[1]) / float(s[3])
    dy = (fy - s[2]) / float(s[4])
    return dx * dx + dy * dy <= 1.0


def continent_of(x, y):
    """0 = 大海；1..7 = 各洲（复刻 ezfyContinentOf）"""
    for i in range(len(CONTINENT_IDS)):
        for s in LANDS[i]:
            if in_shape(x, y, s):
                for h in HOLES[i]:
                    if in_shape(x, y, h):
                        return T_OCEAN
                return CONTINENT_IDS[i]
    return T_OCEAN


def land_terrain(x, y):
    """陆地地形 1~7（复刻 ezfyLandTerrain，注意两次 int64 溢出与算术右移）"""
    h = abs(x * 374761393 + y * 668265263)
    h = i64((h ^ (h >> 13)) * 1274126177)
    h = abs(h ^ (h >> 16))
    return h % 7 + 1


def base_terrain(x, y):
    """ezfyTerrain：先判大陆，海洋成片，陆地上再取 1~7"""
    if continent_of(x, y) == T_OCEAN:
        return T_SEA
    return land_terrain(x, y)


def wildland_level(x, y):
    """ezfyWildlandLevel：坐标哈希的野地等级（0 = 这格没野地）"""
    h = abs((x * 83492791) ^ (y * 6291469))
    return h % 11


def sea_wild_terrain(x, y):
    """ezfySeaWildTerrain：海洋野地细分为 渔场/矿礁/海底森林；纯海洋仍是 8"""
    if wildland_level(x, y) == 0:
        return T_SEA
    h = abs((x * 7919) ^ (y * 104729))
    m = h % 10
    if m <= 3:
        return T_FISHING
    if m <= 6:
        return T_OREEF
    return T_SEA


# ---------------------------------------------------------------- 读库
def fetch(conn, sql):
    with conn.cursor(pymysql.cursors.DictCursor) as cur:
        cur.execute(sql)
        return cur.fetchall()


def load_tiles(conn):
    """ezfy_map_tile：人工覆盖地形（terrain > 0 才生效）"""
    rows = fetch(conn, "SELECT x, y, terrain, mark_kind, mark_level, des FROM ezfy_map_tile")
    tiles = {}
    for r in rows:
        t = r["terrain"] or 0
        if t > 0:
            tiles[(int(r["x"]), int(r["y"]))] = t
    return tiles, len(rows)


def load_cities(conn):
    rows = fetch(conn, """
        SELECT id, user_id, name, x, y, city_level, is_bot
        FROM ezfy_city
        ORDER BY id
    """)
    cities = []
    for r in rows:
        x, y = int(r["x"]), int(r["y"])
        if not (MIN_C <= x <= MAX_C and MIN_C <= y <= MAX_C):
            continue
        cities.append({
            "id": int(r["id"]),
            "user_id": int(r["user_id"] or 0),
            "name": (r["name"] or "").strip(),
            "x": x, "y": y,
            "level": int(r["city_level"] or 0),
            "bot": int(r["is_bot"] or 0) == 1,
        })
    return cities


# ---------------------------------------------------------------- 生成地形网格
def build_grid(tiles):
    """返回 grid[y][x] = 地形 id（1..499）。先预计算 base 与邻域，避免 25 万格反复递归。"""
    span = MAX_C + 2  # 0..500，含外圈一格用于邻居判断
    base = [[0] * span for _ in range(span)]
    for y in range(span):
        row = base[y]
        for x in range(span):
            row[x] = base_terrain(x, y)

    # 沿海平原：基础地形是平原(1) 且 8 邻域存在海洋(8)
    coastal = [[False] * span for _ in range(span)]
    for y in range(span):
        for x in range(span):
            if base[y][x] != T_PLAIN:
                continue
            hit = False
            for dy in (-1, 0, 1):
                yy = y + dy
                if yy < 0 or yy >= span:
                    continue
                for dx in (-1, 0, 1):
                    if dx == 0 and dy == 0:
                        continue
                    xx = x + dx
                    if 0 <= xx < span and base[yy][xx] == T_SEA:
                        hit = True
                        break
                if hit:
                    break
            coastal[y][x] = hit

    grid = {}
    for y in range(MIN_C, MAX_C + 1):
        for x in range(MIN_C, MAX_C + 1):
            t = tiles.get((x, y), 0)
            if t > 0:                      # ① 人工覆盖优先
                grid[(x, y)] = t
            elif coastal[y][x]:            # ② 沿海平原
                grid[(x, y)] = T_COASTAL
            elif base[y][x] == T_SEA:      # ③ 海洋野地细分
                grid[(x, y)] = sea_wild_terrain(x, y)
            else:
                grid[(x, y)] = base[y][x]
    return grid


# ---------------------------------------------------------------- 写 Excel
def write_excel(path, grid, cities, meta):
    t0 = time.time()
    wb = xlsxwriter.Workbook(path, {"constant_memory": False, "default_date_format": "yyyy-mm-dd"})

    # ---- 样式（复用同一个 format 对象，xlsxwriter 会按样式去重，文件不会膨胀）
    fmts = {}
    for tid, (bg, fg) in COLORS.items():
        fmts[tid] = wb.add_format({
            "bg_color": bg, "pattern": 1, "border": 0,
            "font_size": 8, "font_color": fg or "#000000",
        })
    city_fmt = wb.add_format({
        "bg_color": CITY_BG, "pattern": 1, "font_size": 9, "bold": True,
        "font_color": "#FFFFFF", "align": "center", "valign": "vcenter", "border": 0,
    })
    bot_fmt = wb.add_format({
        "bg_color": BOT_CITY_BG, "pattern": 1, "font_size": 9, "bold": True,
        "font_color": "#3B1E45", "align": "center", "valign": "vcenter", "border": 0,
    })
    head_fmt = wb.add_format({
        "bg_color": "#F2F2F3", "font_size": 7, "font_color": "#666666",
        "align": "center", "border": 0,
    })

    ws = wb.add_worksheet("地图")
    ws.hide_gridlines(2)
    ws.freeze_panes(1, 1)
    ws.set_zoom(50)          # 打开时先缩到 50%，一眼能看到较大范围
    # ★ 正方形格：列宽与行高都按 CELL_PX 换算（见文件头常量说明）
    ws.set_column(0, 0, COL_WIDTH)
    ws.set_column(1, MAX_C, COL_WIDTH)
    ws.set_row(0, ROW_HEIGHT)

    ws.write_string(0, 0, "y\\x", head_fmt)
    for x in range(MIN_C, MAX_C + 1):
        ws.write_number(0, x, x, head_fmt)

    city_at = {}
    for c in cities:
        city_at[(c["x"], c["y"])] = c

    for y in range(MIN_C, MAX_C + 1):
        # 行序：y 小的在上。这套大陆几何（ezfyLands，150×150 屏幕坐标）是按「y 向下 = 向南」
        # 设计的：欧洲左上、亚洲右上、南美右下、南极最底 → 正着排就是标准世界地图（北在上）。
        row = y
        ws.set_row(row, ROW_HEIGHT)
        ws.write_number(row, 0, y, head_fmt)
        for x in range(MIN_C, MAX_C + 1):
            c = city_at.get((x, y))
            if c is not None:
                ws.write_string(row, x, "城", bot_fmt if c["bot"] else city_fmt)
            else:
                ws.write_blank(row, x, None, fmts[grid[(x, y)]])

    # ---- 图例 / 统计
    lg = wb.add_worksheet("图例与统计")
    lg.set_column(0, 0, 12)
    lg.set_column(1, 1, 10)
    lg.set_column(2, 2, 12)
    lg.set_column(3, 3, 40)
    title_fmt = wb.add_format({"bold": True, "font_size": 14})
    sub_fmt = wb.add_format({"font_size": 10, "font_color": "#666666"})
    th_fmt = wb.add_format({"bold": True, "bg_color": "#EDEDED", "border": 1, "font_size": 10})
    td_fmt = wb.add_format({"border": 1, "font_size": 10})

    lg.write(0, 0, meta["title"], title_fmt)
    lg.write(1, 0, meta["subtitle"], sub_fmt)
    lg.write(2, 0, "地图范围：x 1~499，y 1~499（共 249001 格）；一格一色、格子宽高相等（正方形），"
                   "城市格写「城」。", sub_fmt)
    lg.write(3, 0, "行序自上而下 y=1 → y=499（与大陆几何一致，北方在上）；"
                   "首行是 x 坐标、首列是 y 坐标，已冻结。", sub_fmt)
    lg.write(4, 0, meta["overlay_note"], sub_fmt)

    row = 5
    lg.write(row, 0, "颜色", th_fmt)
    lg.write(row, 1, "地形", th_fmt)
    lg.write(row, 2, "格数", th_fmt)
    lg.write(row, 3, "说明", th_fmt)
    row += 1

    stat = meta["stat"]
    note = {
        1: "平原：可建城，采集只有粮食",
        2: "草原：采集粮食",
        3: "森林：采集粮食",
        4: "盆地：采集稀矿",
        5: "丘陵：采集钢铁",
        6: "沼泽：采集石油",
        7: "岛屿：海野玩法，采集钢铁",
        8: "海洋：无野地的纯海面",
        9: "沿海平原：平原靠海派生地形，海城只能建这里",
        10: "渔场：海洋野地细分，采集粮食",
        11: "矿礁：海洋野地细分，采集稀矿",
    }
    for tid in range(1, 12):
        cell = wb.add_format({"bg_color": COLORS[tid][0], "border": 1})
        lg.write_blank(row, 0, None, cell)
        lg.write(row, 1, TERRAIN_NAMES[tid], td_fmt)
        lg.write_number(row, 2, stat.get(tid, 0), td_fmt)
        lg.write(row, 3, note[tid], td_fmt)
        row += 1

    cell = wb.add_format({"bg_color": CITY_BG, "border": 1, "font_color": "#FFFFFF", "bold": True})
    lg.write(row, 0, "城", cell)
    lg.write(row, 1, "玩家城市", td_fmt)
    lg.write_number(row, 2, meta["city_player"], td_fmt)
    lg.write(row, 3, "紫色格，格内写「城」", td_fmt)
    row += 1
    cell = wb.add_format({"bg_color": BOT_CITY_BG, "border": 1, "font_color": "#3B1E45", "bold": True})
    lg.write(row, 0, "城", cell)
    lg.write(row, 1, "机器人城市", td_fmt)
    lg.write_number(row, 2, meta["city_bot"], td_fmt)
    lg.write(row, 3, "浅紫格（is_bot=1），同样写「城」", td_fmt)

    row += 2
    lg.write(row, 0, "数据来源", th_fmt)
    lg.write(row, 1, "", th_fmt)
    lg.write(row, 2, "", th_fmt)
    lg.write(row, 3, "", th_fmt)
    row += 1
    for line in meta["sources"]:
        lg.write(row, 0, "", td_fmt)
        lg.merge_range(row, 0, row, 3, line, td_fmt)
        row += 1

    # ---- 城市列表
    cl = wb.add_worksheet("城市列表")
    cl.freeze_panes(1, 0)
    widths = [6, 10, 10, 26, 8, 8, 10, 12]
    for i, w in enumerate(widths):
        cl.set_column(i, i, w)
    heads = ["#", "城市ID", "玩家ID", "城市名", "X", "Y", "市政厅等级", "类型"]
    for i, h in enumerate(heads):
        cl.write(0, i, h, th_fmt)
    cl.set_row(0, 18)
    for i, c in enumerate(cities, 1):
        cl.write_number(i, 0, i, td_fmt)
        cl.write_number(i, 1, c["id"], td_fmt)
        cl.write_number(i, 2, c["user_id"], td_fmt)
        cl.write(i, 3, c["name"], td_fmt)
        cl.write_number(i, 4, c["x"], td_fmt)
        cl.write_number(i, 5, c["y"], td_fmt)
        cl.write_number(i, 6, c["level"], td_fmt)
        cl.write(i, 7, "机器人" if c["bot"] else "玩家", td_fmt)
    cl.autofilter(0, 0, len(cities), len(heads) - 1)

    wb.close()
    return time.time() - t0


def render_png(grid, cities, path, scale=3):
    """额外输出一张 PNG 预览：一格 = scale×scale 像素，北在上，城市画紫块。"""
    from PIL import Image, ImageDraw

    def rgb(h):
        return tuple(int(h[i:i + 2], 16) for i in (1, 3, 5))

    n = MAX_C - MIN_C + 1
    img = Image.new("RGB", (n * scale, n * scale), "#FFFFFF")
    px = img.load()
    for (x, y), t in grid.items():
        c = rgb(COLORS[t][0])
        row = (y - MIN_C) * scale
        col = (x - MIN_C) * scale
        for dy in range(scale):
            for dx in range(scale):
                px[col + dx, row + dy] = c

    draw = ImageDraw.Draw(img)
    for c in cities:
        row = (c["y"] - MIN_C) * scale
        col = (c["x"] - MIN_C) * scale
        color = rgb(BOT_CITY_BG) if c["bot"] else rgb(CITY_BG)
        draw.rectangle([col - 1, row - 1, col + scale, row + scale],
                       fill=color, outline=(255, 255, 255))
    img.save(path)
    return img.size


def check_cell_px(path):
    """写盘后回读 cols/ht，换算成像素，确认格子是正方形（避免 set_column 换算偏差）。"""
    import re
    import zipfile
    with zipfile.ZipFile(path) as z:
        xml = z.read("xl/worksheets/sheet1.xml").decode("utf-8")
    w = float(re.search(r'<col [^>]*width="([\d.]+)"', xml).group(1))
    ht = float(re.search(r'<row r="2"[^>]*?\sht="([\d.]+)"', xml).group(1))
    return round(w * 7) + 5, round(ht * 4 / 3)


def main():
    ap = argparse.ArgumentParser(description="导出二战风云世界地图到 Excel")
    ap.add_argument("--host", default=DB_HOST)
    ap.add_argument("--port", type=int, default=DB_PORT)
    ap.add_argument("--user", default=DB_USER)
    ap.add_argument("--password", default=DB_PASS)
    ap.add_argument("--db", default=DB_NAME, help="一区 qq_ezzt / 二区 qq_ezzt2")
    ap.add_argument("--zone", default="1", help="仅用于标题展示")
    ap.add_argument("--out", default=None, help="输出 xlsx 路径")
    ap.add_argument("--dump", nargs="*", default=None, help="只打印坐标地形，如 --dump 234,96 231,97")
    ap.add_argument("--no-excel", action="store_true", help="只跑统计，不写文件")
    ap.add_argument("--raw", action="store_true",
                    help="忽略管理端「地图格子覆盖」，只看坐标哈希的自然地形（线上 2026-09-28 "
                         "曾把 49465 格海面批量覆盖成沿海平原，加这个参数才能看到原始海陆）")
    ap.add_argument("--png", nargs="?", const="", default=None,
                    help="同时输出 PNG 预览图（不传路径则与 xlsx 同名）")
    args = ap.parse_args()

    out = args.out or os.path.join(
        os.getcwd(),
        f"线上一区地图{'-原始地形' if args.raw else ''}.xlsx")

    print(f"[1/4] 连接 {args.host} / {args.db} ...")
    conn = pymysql.connect(host=args.host, port=args.port, user=args.user,
                           password=args.password, database=args.db,
                           charset="utf8mb4", connect_timeout=15, read_timeout=120)
    try:
        print("[2/4] 读取地图格子覆盖 / 城市 ...")
        tiles, tile_rows = load_tiles(conn)
        cities = load_cities(conn)
    finally:
        conn.close()
    print(f"      覆盖行 {tile_rows} 条（生效 {len(tiles)} 条），城市 {len(cities)} 座")

    print("[3/4] 按坐标哈希推算地形（25 万格，约 10~30 秒）...")
    t0 = time.time()
    grid = build_grid({} if args.raw else tiles)
    print(f"      完成，用时 {time.time() - t0:.1f}s")

    stat = {}
    for t in grid.values():
        stat[t] = stat.get(t, 0) + 1

    if args.dump is not None:
        for item in args.dump:
            x, y = [int(v) for v in item.split(",")]
            t = grid.get((x, y))
            src = "覆盖" if (x, y) in tiles else "哈希"
            print(f"      ({x},{y}) terrain={t} {TERRAIN_NAMES.get(t, '?')}  [{src}]")
        if args.no_excel:
            return

    player = sum(1 for c in cities if not c["bot"])
    bot = len(cities) - player
    print("      地形分布：" + "，".join(
        f"{TERRAIN_NAMES[t]}{stat.get(t, 0)}" for t in range(1, 12)))

    if args.no_excel:
        return

    meta = {
        "title": f"二战风云 线上一区 世界地图（{args.db}）"
                 + ("　—　原始地形（不含人工覆盖）" if args.raw else ""),
        "subtitle": f"导出时间 {datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S')}　"
                    f"城市 {len(cities)} 座（玩家 {player} / 机器人 {bot}）　"
                    + ("已忽略管理端地形覆盖" if args.raw
                       else f"管理端地形覆盖 {len(tiles)} 格"),
        "stat": stat,
        "city_player": player,
        "city_bot": bot,
        "overlay_note": ("地形不是库里的字段，是后端按坐标哈希推导；本表**已忽略**管理端「地图格子覆盖」，"
                         "是自然地形（--raw）。"
                         if args.raw else
                         "地形不是库里的字段，是后端按坐标哈希推导；本表已叠加管理端「地图格子覆盖」，"
                         "与游戏内一致。"),
        "sources": [
            f"库：{args.host} / {args.db}",
            f"ezfy_city：城市坐标（{len(cities)} 座，x/y 均在 1~499）",
            (f"ezfy_map_tile：管理端「地图格子覆盖」共 {len(tiles)} 格，本表**已忽略**（--raw）"
             if args.raw else
             f"ezfy_map_tile：管理端「地图格子覆盖」，生效 {len(tiles)} 格（terrain>0 覆盖哈希）"),
            "其余地形由 ezfy_geo.go 的坐标哈希推导，本脚本逐位复刻（含 int64 溢出与算术右移）",
            "注意：2026-09-28 有一次批量操作「海底森林改沿海平原」写入了 49465 格覆盖，"
            "所以带覆盖的图上大片海面显示为沿海平原；要看不含人工干预的自然海陆，请加 --raw 再导一次。",
        ],
    }

    print(f"[4/4] 写 Excel → {out}")
    cost = write_excel(out, grid, cities, meta)
    size = os.path.getsize(out) / 1024 / 1024
    wpx, hpx = check_cell_px(out)
    ok = "正方形 ✓" if wpx == hpx else "不是正方形 ✗"
    print(f"      完成，用时 {cost:.1f}s，文件 {size:.1f} MB；格子 {wpx}×{hpx}px（{ok}）")
    print(f"      城市 {len(cities)} 座（玩家 {player} / 机器人 {bot}），地形格 {len(grid)}")

    if args.png is not None:
        png = args.png or os.path.splitext(out)[0] + ".png"
        t0 = time.time()
        w, h = render_png(grid, cities, png)
        print(f"      预览图 → {png}（{w}×{h}px，{time.time() - t0:.1f}s）")


if __name__ == "__main__":
    sys.exit(main())
