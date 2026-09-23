-- ============================================================================
-- 二战风云 · 线上「负数兵力」数据修复脚本
-- 日期：2026-09-23
-- ----------------------------------------------------------------------------
-- 现象：玩家总兵力 = -8843547888967622000（int64 正向溢出翻负）
-- 根因：训练 / 伤兵恢复累加没有任何上限，单兵种 count 加到超过 int64 上限就翻负
-- 处理：把「负数」与「超过上限」的兵力统一刷成上限值（默认 10 亿 = 1000000000）
--       资源/人口出现负数则归 0（正常不该有，属防溢出兜底）
--
-- ⚠️ 执行前务必备份（三张表都很小，秒级完成）：
--   mysqldump -uroot -p qq_jiayuan ezfy_city ezfy_city_troop ezfy_train_queue ezfy_wounded \
--     > backup_ezfy_$(date +%Y%m%d).sql
--
-- ⚠️ 上限值以管理端「系统配置 → 单城兵力上限」为准。若你改过该值，
--    请把下面所有 1000000000 替换成实际配置值。
-- ============================================================================

-- ---- 0. 先看一眼「哪些数据有问题」（不改数据，仅查看）--------------------
SELECT 'ezfy_city_troop 负数/超限' AS 问题, COUNT(*) AS 条数 FROM ezfy_city_troop WHERE count < 0 OR count > 1000000000
UNION ALL SELECT 'ezfy_train_queue 负数/超限', COUNT(*) FROM ezfy_train_queue WHERE count < 0 OR count > 1000000000
UNION ALL SELECT 'ezfy_wounded 负数/超限', COUNT(*) FROM ezfy_wounded WHERE count < 0 OR count > 1000000000
UNION ALL SELECT 'ezfy_city 资源/人口负数', COUNT(*) FROM ezfy_city
  WHERE gold < 0 OR food < 0 OR steel < 0 OR oil < 0 OR rare < 0 OR pop < 0 OR pop_max < 0;

-- ---- 1. 城内部队：负数 / 超上限 → 上限值 ---------------------------------
UPDATE ezfy_city_troop
   SET count = 1000000000
 WHERE count < 0 OR count > 1000000000;

-- ---- 2. 训练队列：同上 ---------------------------------------------------
UPDATE ezfy_train_queue
   SET count = 1000000000
 WHERE count < 0 OR count > 1000000000;

-- ---- 3. 伤兵 / 逃兵：负数 → 0，超上限 → 上限 -----------------------------
UPDATE ezfy_wounded SET count = 0          WHERE count < 0;
UPDATE ezfy_wounded SET count = 1000000000 WHERE count > 1000000000;

-- ---- 4. 城市资源 / 人口：负数 → 0（防溢出兜底）---------------------------
UPDATE ezfy_city SET gold  = 0 WHERE gold  < 0;
UPDATE ezfy_city SET food  = 0 WHERE food  < 0;
UPDATE ezfy_city SET steel = 0 WHERE steel < 0;
UPDATE ezfy_city SET oil   = 0 WHERE oil   < 0;
UPDATE ezfy_city SET rare  = 0 WHERE rare  < 0;
UPDATE ezfy_city SET pop     = 0 WHERE pop     < 0;
UPDATE ezfy_city SET pop_max = 0 WHERE pop_max < 0;

-- ---- 5. 复核：以下每个数字都应为 0 ---------------------------------------
SELECT '剩余异常-城内部队' AS 检查项, COUNT(*) AS 应为0 FROM ezfy_city_troop WHERE count < 0 OR count > 1000000000
UNION ALL SELECT '剩余异常-训练队列', COUNT(*) FROM ezfy_train_queue WHERE count < 0 OR count > 1000000000
UNION ALL SELECT '剩余异常-伤兵', COUNT(*) FROM ezfy_wounded WHERE count < 0 OR count > 1000000000
UNION ALL SELECT '剩余异常-城市资源', COUNT(*) FROM ezfy_city
  WHERE gold < 0 OR food < 0 OR steel < 0 OR oil < 0 OR rare < 0 OR pop < 0 OR pop_max < 0;

-- ---- 6. 看看出事那个玩家（号码 35806363）现在还剩多少兵 -------------------
--   用户号码 → user_id 需要先查（ezfy_profile 或 users 表按号码找）
-- SELECT c.user_id, c.name, t.troop_id, t.count
--   FROM ezfy_city c JOIN ezfy_city_troop t ON t.city_id = c.id
--  WHERE c.user_id = (SELECT id FROM users WHERE number = '35806363');
