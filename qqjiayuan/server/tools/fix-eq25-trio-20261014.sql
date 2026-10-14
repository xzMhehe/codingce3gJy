-- ============================================================
-- 修复：青天白日[审判]（套装 25）单件三维 88/44/44 → 100/50/50
-- 2026-10-14 玩家反馈：与同档位「赤色锤镰[裁决]」（套装 26）单件加成不一致
--
-- 目标口径（用户确认）：11 件之和 = 军事 1100 / 后勤 550 / 学识 550
--   即单件 100/50/50，与系列 26 完全一致（种子 ezfyOfficerSeriesSeeds 里 25/26 都是 Mi:100 Lo:50 Le:50）
--
-- 套装行 ezfy_cfg_equip_set 25/26 本就是 96/49/49，两套一致 → 不动。
-- 幂等：可重复执行。
-- 回滚：见 备份-青天白日三维-2026-10-14.sql
-- 用法：mysql -h<RDS> -uroot -p qq_ezzt < fix-eq25-trio-20261014.sql
-- ============================================================

-- ① 配置池件行（11 件）
UPDATE ezfy_cfg_equipment SET military=100, logistics=50, learning=50
 WHERE id BETWEEN 2501 AND 2511;

-- ② 玩家背包装备快照（买到时抄的那份）
UPDATE ezfy_equipment SET military=100, logistics=50, learning=50
 WHERE cfg_id BETWEEN 2501 AND 2511;

-- ③ 军官身上装备 JSON（穿戴时抄的那份）：只重建「穿着套装 25」的军官，
--    口径与 seed.repairEquipSnapshots ② 一致（从背包行重建整份 JSON）
UPDATE ezfy_officer o
  JOIN (
    SELECT officer_id,
           JSON_ARRAYAGG(JSON_OBJECT(
             'crit', e.crit, 'crit_dmg', e.crit_dmg, 'def', e.def, 'dmg', e.dmg,
             'enhance', e.enhance, 'hp', e.hp, 'id', e.id,
             'learning', e.learning, 'logistics', e.logistics, 'military', e.military,
             'move', e.move, 'name', COALESCE(e.name,''), 'series', COALESCE(e.series,''),
             'set_id', e.set_id, 'slot', COALESCE(e.slot,''), 'tier', e.tier,
             'type', COALESCE(e.type,''))) AS j
      FROM ezfy_equipment e
     WHERE e.officer_id > 0
     GROUP BY e.officer_id
  ) t ON t.officer_id = o.id
   SET o.equipment = t.j
 WHERE o.equipment <> '' AND o.equipment LIKE '%"set_id":25,%';

-- ④ 校验：两套应完全一致（25/26 每件 100/50/50，之和 1100/550/550）
SELECT set_id, COUNT(*) n, MIN(military) mn_mi, MAX(military) mx_mi,
       SUM(military) sum_mi, SUM(logistics) sum_lo, SUM(learning) sum_le
  FROM ezfy_cfg_equipment WHERE id BETWEEN 2501 AND 2611 GROUP BY set_id ORDER BY set_id;

-- ⑤ 校验：军官 JSON 里不应再有 88/44/44
SELECT COUNT(*) officer_json_still_88 FROM ezfy_officer
 WHERE equipment LIKE '%"military":88%' AND equipment LIKE '%"set_id":25,%';
