-- ============================================================================
-- 二战风云 · 科技数据「跟玩家走」迁移
-- 2026-09-23 用户要求：科技跟玩家走（所有城市共享），不是跟城市走。
--
-- 背景：
--   第九轮重构后游戏内科技统一读写「科技城」（= 玩家主城 MIN(ezfy_city.id)），
--   只有主城行的科技生效。但历史上（按城独立时期）和管理端旧「设置城池科技」
--   写出的非主城行仍留在库里 —— 这些是无效数据，且可能让管理端看到多个城市
--   各行一套科技。
--
-- 本脚本：
--   0. 同 (city_id, tech_id) 重复行先去重（等级取最大、研究中状态保留）
--   1. 把每个玩家非主城行的科技归并到主城行（等级/状态/结束时间取大，不丢进度）
--   2. 删除所有非主城行
--   3. 校验
--
-- 幂等：执行后非主城行归零，重复执行无副作用。
-- ⚠️ 执行前建议备份：
--   mysqldump -h39.105.151.141 -uroot -p qq_jiayuan ezfy_city_tech ezfy_city > backup_tech_20260923.sql
-- ============================================================================

-- ---- 0. 同 (city_id, tech_id) 去重 ----------------------------------------
UPDATE ezfy_city_tech t
  JOIN (SELECT city_id, tech_id, MIN(id) AS keep_id, MAX(level) AS mlv,
               MAX(status) AS mst, MAX(end_time) AS mend
          FROM ezfy_city_tech
         GROUP BY city_id, tech_id
        HAVING COUNT(*) > 1) d
    ON d.city_id = t.city_id AND d.tech_id = t.tech_id AND t.id = d.keep_id
   SET t.level = d.mlv, t.status = d.mst, t.end_time = d.mend;

DELETE t FROM ezfy_city_tech t
  JOIN (SELECT city_id, tech_id, MIN(id) AS keep_id
          FROM ezfy_city_tech
         GROUP BY city_id, tech_id
        HAVING COUNT(*) > 1) d
    ON d.city_id = t.city_id AND d.tech_id = t.tech_id AND t.id > d.keep_id;

-- ---- 1. 每个玩家的主城（科技城） -------------------------------------------
DROP TEMPORARY TABLE IF EXISTS tmp_main_city;
CREATE TEMPORARY TABLE tmp_main_city (
  user_id BIGINT PRIMARY KEY,
  main_id BIGINT NOT NULL
);
INSERT INTO tmp_main_city (user_id, main_id)
SELECT user_id, MIN(id) FROM ezfy_city GROUP BY user_id;

-- ---- 1.5 非主城行聚合结果物化（避免 MySQL「同一临时表不能在同语句打开两次」）---
DROP TEMPORARY TABLE IF EXISTS tmp_tech_agg;
CREATE TEMPORARY TABLE tmp_tech_agg (
  user_id BIGINT NOT NULL,
  tech_id BIGINT NOT NULL,
  mlv BIGINT NOT NULL,
  mst BIGINT NOT NULL,
  mend BIGINT NOT NULL,
  PRIMARY KEY (user_id, tech_id)
);
INSERT INTO tmp_tech_agg (user_id, tech_id, mlv, mst, mend)
SELECT m.user_id, x.tech_id,
       MAX(x.level) AS mlv, MAX(x.status) AS mst, MAX(x.end_time) AS mend
  FROM ezfy_city_tech x
  JOIN ezfy_city c ON c.id = x.city_id
  JOIN tmp_main_city m ON m.user_id = c.user_id AND m.main_id <> x.city_id
 GROUP BY m.user_id, x.tech_id;

-- ---- 2. 主城已有该科技 → 并入非主城行的高等级 / 研究中状态 ----------------
UPDATE ezfy_city_tech t
  JOIN tmp_main_city m ON t.city_id = m.main_id
  JOIN tmp_tech_agg agg ON agg.user_id = m.user_id AND agg.tech_id = t.tech_id
   SET t.level = GREATEST(t.level, agg.mlv),
       t.status = GREATEST(t.status, agg.mst),
       t.end_time = GREATEST(t.end_time, agg.mend);

-- ---- 3. 主城没有该科技 → 把非主城行搬一条到主城 ----------------------------
INSERT INTO ezfy_city_tech (city_id, tech_id, level, status, end_time, updated_at)
SELECT m.main_id, agg.tech_id, agg.mlv, agg.mst, agg.mend, NOW()
  FROM tmp_tech_agg agg
  JOIN tmp_main_city m ON m.user_id = agg.user_id
 WHERE NOT EXISTS (SELECT 1 FROM ezfy_city_tech t2
                    WHERE t2.city_id = m.main_id AND t2.tech_id = agg.tech_id);

-- ---- 4. 删除所有非主城行 ---------------------------------------------------
DELETE t FROM ezfy_city_tech t
  JOIN ezfy_city c ON c.id = t.city_id
  JOIN tmp_main_city m ON m.user_id = c.user_id AND m.main_id <> c.id;

-- ---- 5. 校验 -----------------------------------------------------------------
-- 以下应全部为 0 / 空
SELECT '非主城残留行' AS 检查项, COUNT(*) AS 应为0 FROM ezfy_city_tech t
  JOIN ezfy_city c ON c.id = t.city_id
  JOIN (SELECT user_id, MIN(id) AS main_id FROM ezfy_city GROUP BY user_id) m
    ON m.user_id = c.user_id AND m.main_id <> c.id;

SELECT '同城同科技重复行' AS 检查项, COUNT(*) AS 应为0 FROM (
  SELECT city_id, tech_id FROM ezfy_city_tech GROUP BY city_id, tech_id HAVING COUNT(*) > 1
) d;

SELECT c.user_id, c.id AS city_id, c.name, COUNT(t.id) AS tech_rows
  FROM ezfy_city c JOIN ezfy_city_tech t ON t.city_id = c.id
 GROUP BY c.user_id, c.id, c.name
 ORDER BY tech_rows DESC
 LIMIT 20;

-- ---- 6. 无主城玩家的科技残留（极少见：主城被删的玩家）-----------------------
SELECT '无城市玩家的科技行' AS 检查项, COUNT(*) AS 应留意 FROM ezfy_city_tech t
 WHERE NOT EXISTS (SELECT 1 FROM ezfy_city c WHERE c.id = t.city_id);