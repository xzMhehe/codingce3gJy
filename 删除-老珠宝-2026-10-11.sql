-- ============================================================================
-- 删除「老珠宝」（非采集宝物的旧珠宝）—— 2026-10-11
--
-- 背景：`ezfy_cfg_equipment` 里的珠宝分两批：
--   · 19~26 红宝石/蓝宝石/绿宝石/翡翠/紫水晶/猫眼石/玛瑙/珍珠   ← **本脚本要删的「老珠宝」**
--     （Type=珠宝、价格 0、库存 -1，不在任何宝箱奖池/军团商城，纯历史残留；
--       采集不会掉、军衔晋升/赏赐/出售 都不认它们）
--   · 27~35 黄金手镯/玛瑙项坠/红宝石戒指/黑曜石戒指/琥珀项链/铂金戒指/翡翠项链/祖母绿/蓝宝石戒指
--     ← **9 种采集宝物，绝对不能删！**（宝物页/军衔晋升/赏赐/出售/宝物上限都用它）
--
-- 本脚本删三处（都按**名字**精确匹配，绝不按 ID 区间误伤）：
--   ① ezfy_equipment      玩家身上的老珠宝实例（含被军官穿着的）
--   ② ezfy_officer        军官 equipment JSON 里残留的老珠宝条目（按 JSON 重建数组）
--   ③ ezfy_cfg_equipment  老珠宝的 8 行配置
--
-- 幂等：删完再跑一遍就是 0 行受影响。建议先备份：mysqldump 该库。
-- 执行库：二战库（线上 qq_ezzt；单库部署则为 qq_jiayuan）
-- ============================================================================

-- USE qq_ezzt;

-- ---------- 0. 执行前核对（应约：实例 2548 行 / 配置 8 行） ----------
SELECT '① 待删实例' AS step, name, COUNT(*) AS rows_cnt
FROM ezfy_equipment
WHERE type = '珠宝'
  AND name IN ('红宝石','蓝宝石','绿宝石','翡翠','紫水晶','猫眼石','玛瑙','珍珠')
GROUP BY name
ORDER BY rows_cnt DESC;

SELECT '③ 待删配置' AS step, id, name
FROM ezfy_cfg_equipment
WHERE type = '珠宝'
  AND name IN ('红宝石','蓝宝石','绿宝石','翡翠','紫水晶','猫眼石','玛瑙','珍珠')
ORDER BY id;

-- ---------- 1. 删玩家身上的老珠宝实例（含已穿戴） ----------
DELETE FROM ezfy_equipment
WHERE type = '珠宝'
  AND name IN ('红宝石','蓝宝石','绿宝石','翡翠','紫水晶','猫眼石','玛瑙','珍珠');

-- ---------- 2. 清军官 equipment JSON 里残留的老珠宝条目 ----------
-- 只动「JSON 合法 + 确实含老珠宝」的军官；按 JSON 数组重建，精确剔除老珠宝项。
-- （军官下次上线/服务重启时 repairEquipSnapshots 还会按装备行重建一次，顺序会被归一到标准口径。）
UPDATE ezfy_officer o
SET o.equipment = (
  SELECT COALESCE(JSON_ARRAYAGG(j), JSON_ARRAY())
  FROM JSON_TABLE(o.equipment, '$[*]' COLUMNS (j JSON PATH '$')) t
  WHERE JSON_UNQUOTE(JSON_EXTRACT(j, '$.name'))
        NOT IN ('红宝石','蓝宝石','绿宝石','翡翠','紫水晶','猫眼石','玛瑙','珍珠')
)
WHERE JSON_VALID(o.equipment)
  AND (JSON_CONTAINS(o.equipment, '{"name":"红宝石"}')
    OR JSON_CONTAINS(o.equipment, '{"name":"蓝宝石"}')
    OR JSON_CONTAINS(o.equipment, '{"name":"绿宝石"}')
    OR JSON_CONTAINS(o.equipment, '{"name":"翡翠"}')
    OR JSON_CONTAINS(o.equipment, '{"name":"紫水晶"}')
    OR JSON_CONTAINS(o.equipment, '{"name":"猫眼石"}')
    OR JSON_CONTAINS(o.equipment, '{"name":"玛瑙"}')
    OR JSON_CONTAINS(o.equipment, '{"name":"珍珠"}'));

-- ---------- 3. 删老珠宝配置 ----------
DELETE FROM ezfy_cfg_equipment
WHERE type = '珠宝'
  AND name IN ('红宝石','蓝宝石','绿宝石','翡翠','紫水晶','猫眼石','玛瑙','珍珠');

-- ---------- 4. 执行后校验（三项都应为 0） ----------
SELECT '剩余老珠宝实例' AS check_name, COUNT(*) AS remain
FROM ezfy_equipment
WHERE type = '珠宝'
  AND name IN ('红宝石','蓝宝石','绿宝石','翡翠','紫水晶','猫眼石','玛瑙','珍珠')
UNION ALL
SELECT '剩余老珠宝配置', COUNT(*)
FROM ezfy_cfg_equipment
WHERE type = '珠宝'
  AND name IN ('红宝石','蓝宝石','绿宝石','翡翠','紫水晶','猫眼石','玛瑙','珍珠')
UNION ALL
SELECT '9种采集宝物实例(应保持不变)', COUNT(*)
FROM ezfy_equipment
WHERE name IN ('黄金手镯','玛瑙项坠','红宝石戒指','黑曜石戒指','琥珀项链','铂金戒指','翡翠项链','祖母绿','蓝宝石戒指')
UNION ALL
SELECT '9种采集宝物配置(应保持不变)', COUNT(*)
FROM ezfy_cfg_equipment
WHERE name IN ('黄金手镯','玛瑙项坠','红宝石戒指','黑曜石戒指','琥珀项链','铂金戒指','翡翠项链','祖母绿','蓝宝石戒指');
