-- ============================================================
-- 军官套装装备百分比压降 —— 玩家存量装备快照修正（本地执行）
-- 2026-09-23 用户要求：套装百分比加成全部压到 100% 以下（10%~100% 按品质）
--
-- 背景：
--   1. 配置表（ezfy_cfg_equipment / ezfy_cfg_equip_set）的压降由服务器启动时的
--      种子迁移 nerfEquipSetPct 完成 —— 需先重启 server（重启后配置即已压降）。
--   2. 玩家「买到/穿到身上」的装备是从配置表抄的快照（ezfy_equipment），
--      重启不会自动改，需要执行本脚本把快照同步成压降后的配置值。
--   3. 军官身上的装备 JSON（ezfy_officer.equipment）不用本脚本改：
--      执行完本脚本后再重启一次 server，启动时的 repairEquipSnapshots
--      会用背包行自动重建军官装备 JSON。
--
-- 执行顺序：重启 server（配置压降）→ 执行本脚本 → 再重启 server（JSON 重建）
-- 幂等：只动六项里还有 >100% 旧值的行，压完后条件不再命中，可重复执行。
-- 用法：mysql -h127.0.0.1 -P3306 -uroot -p1234567890 qq_jiayuan < fix-equip-pct-20260923.sql
-- ============================================================

-- ① 玩家背包装备快照：六项百分比按配置行同步（套装件 2101~2611 / 散件 3001~3013）
UPDATE ezfy_equipment e
JOIN ezfy_cfg_equipment c ON e.cfg_id = c.id
SET e.dmg = c.dmg,
    e.def = c.def,
    e.hp = c.hp,
    e.move = c.move,
    e.crit = c.crit,
    e.crit_dmg = c.crit_dmg
WHERE (c.id BETWEEN 2101 AND 2699 OR c.id BETWEEN 3001 AND 3013)
  AND (e.dmg > 100 OR e.def > 100 OR e.hp > 100
       OR e.move > 100 OR e.crit > 100 OR e.crit_dmg > 100);

-- ② 校验：以下查询应全部为空 / 最大百分比应 < 100
SELECT COUNT(*) AS still_over100_owned
FROM ezfy_equipment e
JOIN ezfy_cfg_equipment c ON e.cfg_id = c.id
WHERE (c.id BETWEEN 2101 AND 2699 OR c.id BETWEEN 3001 AND 3013)
  AND (e.dmg > 100 OR e.def > 100 OR e.hp > 100
       OR e.move > 100 OR e.crit > 100 OR e.crit_dmg > 100);

SELECT MAX(GREATEST(COALESCE(dmg,0), COALESCE(def,0), COALESCE(hp,0),
                    COALESCE(move,0), COALESCE(crit,0), COALESCE(crit_dmg,0))) AS max_pct_cfg
FROM ezfy_cfg_equipment
WHERE id BETWEEN 2101 AND 2699 OR id BETWEEN 3001 AND 3013;

SELECT id, name, dmg, def, hp, move, crit, crit_dmg
FROM ezfy_cfg_equip_set
WHERE id BETWEEN 21 AND 26
ORDER BY id;