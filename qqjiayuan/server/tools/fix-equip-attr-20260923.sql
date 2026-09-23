-- ============================================================
-- 军官装备「三维属性（军事/后勤/学识）」同比压降补丁
-- 2026-09-23 用户要求：百分比加成已降，三维属性加成本太高，按品质同比减少
--
-- 新值（与种子字面量 ezfyOfficerSeriesSeeds 一致）：
--   系列 25/26 青天白日/赤色锤镰 (T4-130)  70/35/35 → 35/18/18
--   系列 21/22 革命者/渡鸦之魂 (T4-120)    60/30/30 → 25/13/13
--   系列 23    黑色幽灵       (T4-110)     50/25/25 → 18/ 9/ 9
--   系列 24    巨匠           (T3)         40/20/20 → 10/ 5/ 5
--   散件 3001~3013           (T2)         20/10/10 → 10/ 5/ 5
--   套装行三维 = 各件之和 ÷ 4（与 Go 端 seed 取整口径一致，用 DIV）
--
-- 幂等：件行三维只在「仍是压降前旧值（管理端没改过）」时命中；
--       背包/军官 JSON 无条件对齐（重启时 repairEquipSnapshots 也是这个口径）。
-- 用法：mysql -h39.105.151.141 -P3306 -uroot -p qq_jiayuan < fix-equip-attr-20260923.sql
-- ============================================================

-- ① 件行三维：旧值 → 新值（仅命中压降前旧值）
UPDATE ezfy_cfg_equipment SET military=25, logistics=13, learning=13
WHERE id BETWEEN 2101 AND 2299 AND military=60 AND logistics=30 AND learning=30;

UPDATE ezfy_cfg_equipment SET military=18, logistics=9, learning=9
WHERE id BETWEEN 2301 AND 2311 AND military=50 AND logistics=25 AND learning=25;

UPDATE ezfy_cfg_equipment SET military=10, logistics=5, learning=5
WHERE id BETWEEN 2401 AND 2411 AND military=40 AND logistics=20 AND learning=20;

UPDATE ezfy_cfg_equipment SET military=35, logistics=18, learning=18
WHERE id BETWEEN 2501 AND 2699 AND military=70 AND logistics=35 AND learning=35;

-- ② 散件三维：20/10/10 → 10/5/5
UPDATE ezfy_cfg_equipment SET military=10, logistics=5, learning=5
WHERE id BETWEEN 3001 AND 3013 AND military=20 AND logistics=10 AND learning=10;

-- ③ 套装行三维 = 各件之和 ÷ 4（DIV 取整，与 Go 一致）
UPDATE ezfy_cfg_equip_set s
JOIN (SELECT set_id, SUM(military) m, SUM(logistics) l, SUM(learning) e
        FROM ezfy_cfg_equipment
       WHERE set_id BETWEEN 21 AND 26
       GROUP BY set_id) t ON t.set_id = s.id
   SET s.military = t.m DIV 4, s.logistics = t.l DIV 4, s.learning = t.e DIV 4
 WHERE s.id BETWEEN 21 AND 26;

-- ④ 套装行 effect 文案里的三维数字同步（保留「；」之后的百分比部分）
UPDATE ezfy_cfg_equip_set
   SET effect = CONCAT('穿齐', parts, '件，额外再获得：军事+', military, ' 后勤+', logistics,
                       ' 学识+', learning, '；', SUBSTRING(effect, LOCATE('；', effect)+1))
 WHERE id BETWEEN 21 AND 26 AND effect LIKE '穿齐%件，额外再获得：军事+%' AND effect LIKE '%；%';

-- ⑤ 玩家背包装备三维：对齐配置池
UPDATE ezfy_equipment e
  JOIN ezfy_cfg_equipment c ON e.cfg_id = c.id
   SET e.military = c.military, e.logistics = c.logistics, e.learning = c.learning
 WHERE (c.id BETWEEN 2101 AND 2699 OR c.id BETWEEN 3001 AND 3013);

-- ⑥ 军官身上的装备 JSON：从背包重建（同 repairEquipSnapshots 第②步口径）
UPDATE ezfy_officer o
  JOIN (
    SELECT officer_id, JSON_ARRAYAGG(JSON_OBJECT(
             'id', e.id, 'name', e.name, 'type', e.type,
             'slot', COALESCE(NULLIF(e.slot,''), e.type), 'set_id', e.set_id,
             'military', e.military, 'logistics', e.logistics, 'learning', e.learning,
             'series', COALESCE(e.series,''), 'enhance', e.enhance,
             'dmg', e.dmg, 'def', e.def, 'hp', e.hp, 'move', e.move,
             'crit', e.crit, 'crit_dmg', e.crit_dmg)) AS j
      FROM ezfy_equipment e WHERE e.officer_id > 0 GROUP BY e.officer_id
  ) t ON t.officer_id = o.id
   SET o.equipment = t.j
 WHERE o.equipment <> '';

-- ⑦ 校验：以下应无残留旧值
SELECT COUNT(*) AS old3d_cfg_remaining FROM ezfy_cfg_equipment
 WHERE (id BETWEEN 2101 AND 2699 OR id BETWEEN 3001 AND 3013)
   AND ((military=60 AND logistics=30 AND learning=30)
     OR (military=50 AND logistics=25 AND learning=25)
     OR (military=40 AND logistics=20 AND learning=20)
     OR (military=70 AND logistics=35 AND learning=35)
     OR (military=20 AND logistics=10 AND learning=10));

SELECT id, name, military, logistics, learning
  FROM ezfy_cfg_equip_set WHERE id BETWEEN 21 AND 26 ORDER BY id;