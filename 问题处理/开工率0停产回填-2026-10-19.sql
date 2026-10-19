-- ============================================================
-- 2026-10-19 调整生产(开工率)语义变更：0 = 停产，且资源建筑不占用人口
-- ------------------------------------------------------------
-- 背景：
--   之前 0 表示「从未调整过」，代码按 100% 处理；本次改为
--   「开工率 0 = 该资源停产 + 对应资源建筑不再占用人口」。
--   老数据里 rate_* = 0 其实是「没调过」，必须回填成 100，
--   否则新代码一上线老城会全部停产。
--
-- ⚠️ 执行时机：在**启动新版本 server 之前**先跑本 SQL。
--
-- 执行对象：二战所有数据库（1区 qq_ezzt、2区 qq_ezzt2，
--   若 qq_jiayuan 里也有 ezfy_city 表则一并执行）。
-- ============================================================

UPDATE ezfy_city SET rate_food  = 100 WHERE rate_food  = 0;
UPDATE ezfy_city SET rate_steel = 100 WHERE rate_steel = 0;
UPDATE ezfy_city SET rate_oil   = 100 WHERE rate_oil   = 0;
UPDATE ezfy_city SET rate_rare  = 100 WHERE rate_rare  = 0;

-- 校验：回填后应没有任何 0 行（除非玩家上线后自己改成 0）
SELECT COUNT(*) AS still_zero
FROM ezfy_city
WHERE rate_food = 0 OR rate_steel = 0 OR rate_oil = 0 OR rate_rare = 0;
