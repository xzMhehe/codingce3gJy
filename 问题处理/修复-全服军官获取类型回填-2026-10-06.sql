-- 全服军官「获取方式」数据回填（取代单玩家版《修复-35806363小哥哥军官获取类型》）
--
-- 背景：ezfy_officer.source 列是 2026-10-05 才通过 AutoMigrate 加的；之前创建的历史军官 source=0
--       （未标注），管理端判定兜底曾统一显示「管理端发放」——实际很多是：
--         野地任务/野地打猎俘获的名将（应显示「活动野地俘虏」）、
--         打自己小号 PvP 抢来的名将（应显示「抢玩家获取」）。
-- 本 SQL 按「该军官所属城市的玩家本人历史战报」反查证据，把 source=0 的老数据分类回填，
-- JOIN userId = c.user_id，不同玩家同名将不会串。
-- 幂等：只处理 source=0 的行，可整体复制执行；True 一步到位（① 诊断 → ② 更新 → ③ 校验）。

-- ① 诊断段（只读）：先看 source=0 老数据分布，确认影响面
SELECT
  COUNT(*)                               AS source_0_total,
  SUM(is_captive = 1)                    AS in_captive,
  SUM(general_id > 0)                    AS has_pool_id,
  SUM(general_id = 0)                    AS recruit_no_pool_id
FROM ezfy_officer
WHERE source = 0 AND deleted_at IS NULL;

-- ①.1 分类命中预估（SELECT 不更新）：PvP 打小号 / 野地战报 各多少
SELECT
  (SELECT COUNT(*) FROM ezfy_officer o JOIN ezfy_city c ON c.id = o.city_id
    WHERE o.source = 0 AND o.deleted_at IS NULL
      AND EXISTS (SELECT 1 FROM ezfy_report r WHERE r.user_id = c.user_id
                   AND r.content LIKE CONCAT('%敌方军官 ', o.name, ' 忠诚归零, 弃城归降%'))) AS pvp_gained,
  (SELECT COUNT(*) FROM ezfy_officer o JOIN ezfy_city c ON c.id = o.city_id
    WHERE o.source = 0 AND o.deleted_at IS NULL
      AND EXISTS (SELECT 1 FROM ezfy_report r WHERE r.user_id = c.user_id
                   AND r.content LIKE CONCAT('%俘虏敌将:', o.name, '%'))) AS wild_gained,
  (SELECT COUNT(*) FROM ezfy_officer WHERE source = 0 AND deleted_at IS NULL AND is_captive = 1) AS captive_rest,
  (SELECT COUNT(*) FROM ezfy_officer WHERE source = 0 AND deleted_at IS NULL AND general_id = 0) AS recruit_general0;

-- ② 更新段（按序执行；每步 ROW_COUNT 即该类命中数）————————————
-- ②-1 PvP 抢玩家/打小号 → 4（抢玩家获取）
UPDATE ezfy_officer o
  JOIN ezfy_city c ON c.id = o.city_id
 SET o.source = 4, o.update_time = NOW()
 WHERE o.source = 0 AND o.deleted_at IS NULL
   AND EXISTS (SELECT 1 FROM ezfy_report r
                WHERE r.user_id = c.user_id
                  AND r.content LIKE CONCAT('%敌方军官 ', o.name, ' 忠诚归零, 弃城归降%'));

-- ②-2 野地战报（名将野地任务 / 野地打猎俘获）→ 3（活动野地俘虏）
UPDATE ezfy_officer o
  JOIN ezfy_city c ON c.id = o.city_id
 SET o.source = 3, o.update_time = NOW()
 WHERE o.source = 0 AND o.deleted_at IS NULL
   AND EXISTS (SELECT 1 FROM ezfy_report r
                WHERE r.user_id = c.user_id
                  AND r.content LIKE CONCAT('%俘虏敌将:', o.name, '%'));

-- ②-3 战俘营里剩余的（is_captive=1，PvP 已被 ②-1 挑走）→ 3（野地俘虏）
UPDATE ezfy_officer
 SET source = 3, update_time = NOW()
 WHERE source = 0 AND deleted_at IS NULL AND is_captive = 1;

-- ②-4 无军官池 id 的老招募军官（历史招募路径不落池子 id）→ 2（军校招募）
UPDATE ezfy_officer
 SET source = 2, update_time = NOW()
 WHERE source = 0 AND deleted_at IS NULL AND general_id = 0;

-- ②-5 剩余（有池子 id 且无任何战报证据）→ 1（系统/管理端发放）
UPDATE ezfy_officer
 SET source = 1, update_time = NOW()
 WHERE source = 0 AND deleted_at IS NULL;

-- ③ 校验：在职军官不应再存在 source=0（未标注）
SELECT source, COUNT(*) AS n
  FROM ezfy_officer
 WHERE deleted_at IS NULL
 GROUP BY source ORDER BY source;

-- 刀笔：野地任务名将战报「俘虏敌将:」若因战报过期被清理，会落进 ②-5（显示系统发放），
--       实际仍显偏；如需进一步区分可联系开发按活动野地发放记录补 source=3。