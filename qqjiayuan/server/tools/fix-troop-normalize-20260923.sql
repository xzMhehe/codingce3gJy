-- ============================================================================
-- 二战风云 · 单城兵力「合计归一化」补丁
-- 日期：2026-09-23
-- ----------------------------------------------------------------------------
-- 背景：fix-negative-troop-20260923.sql 把「负数 / 超上限」的**逐行**刷成了 10 亿。
--       但代码里 checkTroopCap 的上限口径是**单城合计**
--       （cityTroopTotal = 城内所有兵种 + 训练队列 status=0 合计 ≤ troop_max）。
--       逐行刷 10 亿 → 单城合计可达 160 亿，这些玩家会卡在「超过限额」无法再训练。
--
-- 本脚本：把「单城合计 > troop_max」的城，按比例缩放到 troop_max。
--         口径与 cityTroopTotal / checkTroopCap 完全一致（含城防 type=4）。
--         比例缩放保留兵种结构；向下取整保证缩放后合计 ≤ troop_max。
--
-- ⚠️ 执行前务必备份：
--   mysqldump -uroot -p qq_jiayuan ezfy_city_troop ezfy_train_queue > backup_xxx.sql
--
-- ⚠️ 上限以管理端「系统配置 → 单城兵力上限」为准。线上实测 troop_max = 1000000000。
-- ============================================================================

SET @cap = 1000000000;

-- ---- 0. 算出「合计超限」的城（口径同 cityTroopTotal）----------------------
DROP TEMPORARY TABLE IF EXISTS tmp_city_total;
CREATE TEMPORARY TABLE tmp_city_total (
  city_id BIGINT PRIMARY KEY,
  total   BIGINT NOT NULL
);

INSERT INTO tmp_city_total (city_id, total)
SELECT city_id, SUM(s) FROM (
    SELECT city_id, SUM(count) AS s FROM ezfy_city_troop GROUP BY city_id
    UNION ALL
    SELECT city_id, SUM(count) AS s FROM ezfy_train_queue WHERE status = 0 GROUP BY city_id
) u
GROUP BY city_id
HAVING SUM(s) > @cap;

-- ---- 1. 预览：哪些城会被缩、缩多少 ---------------------------------------
SELECT c.user_id, c.id AS city_id, c.name,
       t.total AS before_total,
       ROUND(@cap / t.total, 4) AS scale_factor,
       @cap AS after_total
  FROM tmp_city_total t JOIN ezfy_city c ON c.id = t.city_id
 ORDER BY t.total DESC;

-- ---- 2. 城内部队：按比例缩（CAST 成 DECIMAL 防大数相乘溢出）-------------
UPDATE ezfy_city_troop ct
  JOIN tmp_city_total x ON x.city_id = ct.city_id
   SET ct.count = FLOOR(CAST(ct.count AS DECIMAL(30,0)) * @cap / x.total);

-- ---- 3. 训练中队列：同样缩（只动 status=0，已完成的 status=2 历史不动）--
UPDATE ezfy_train_queue q
  JOIN tmp_city_total x ON x.city_id = q.city_id
   SET q.count = FLOOR(CAST(q.count AS DECIMAL(30,0)) * @cap / x.total)
 WHERE q.status = 0;

-- ---- 4. 复核：单城合计超限的城应为 0 ------------------------------------
SELECT '剩余单城合计超限' AS 检查项, COUNT(*) AS 应为0 FROM (
  SELECT city_id FROM (
    SELECT city_id, SUM(s) AS total FROM (
      SELECT city_id, SUM(count) AS s FROM ezfy_city_troop GROUP BY city_id
      UNION ALL
      SELECT city_id, SUM(count) AS s FROM ezfy_train_queue WHERE status = 0 GROUP BY city_id
    ) u GROUP BY city_id
  ) v WHERE total > @cap
) w;

-- ---- 5. 抽查：改前超限的城，改后合计 ------------------------------------
SELECT c.user_id, c.id AS city_id, c.name,
       SUM(ct.count) AS in_city,
       (SELECT COALESCE(SUM(q.count),0) FROM ezfy_train_queue q
         WHERE q.city_id = c.id AND q.status = 0) AS training,
       SUM(ct.count) + (SELECT COALESCE(SUM(q.count),0) FROM ezfy_train_queue q
         WHERE q.city_id = c.id AND q.status = 0) AS total_after
  FROM ezfy_city_troop ct JOIN ezfy_city c ON c.id = ct.city_id
 WHERE c.id IN (89, 90, 13, 52, 16, 5, 18, 15, 2, 163)
 GROUP BY c.user_id, c.id, c.name
 ORDER BY total_after DESC;
