-- ============================================================================
-- 修复：采集队「召回后残留 status=3」变成野地幽灵守军
-- 日期：2026-10-25
-- 现象（玩家反馈）：
--   「之前不小心全部召回的卡车，实际没有回来。现在野地有卡车，大号显示没卡车，
--     我用小号征服，就能发现有卡车。」
-- 根因：
--   1. finishReturn 返航归队时**只删 order_type=6** 的订单，采集队(7) 残留 status=3；
--   2. 各处野地守军口径写成 order_type IN (6,7) AND status IN (1,3)，
--      把「已经回城」的采集队继续当成野地守军 → 野地永远挂着一支
--      「打不死、召不回、驻军列表也看不到」的幽灵部队。
-- 代码侧已修（ezfyWildDefenderOrderCond + finishReturn 删 6/7）。
-- 本脚本只清**存量**僵尸行；不跑也不影响玩法（守军口径已排除 status=3 的采集队）。
--
-- 执行顺序：1 区(qq_ezzt) 跑一遍，2 区(qq_ezzt2) 把库名换掉再跑一遍。
-- ⚠️ 部署新 server 之后再执行（旧 server 还会继续产生新的僵尸行）。
--
-- ============================ 执行记录 ============================
-- 2026-10-25 11:15 已执行（用户授权，直连 RDS）：
--   qq_ezzt  628 条 / 23 玩家 / 533 野地坐标 → 已删，备份表 ezfy_order_ghost_bak_20261025
--   qq_ezzt2 353 条 / 12 玩家 / 272 野地坐标 → 已删，备份表 ezfy_order_ghost_bak_20261025
--   两库 order_type=7 AND status=3 均归 0；剩余 7/1 活跃采集 1区 1049 条、2区 321 条（正常）。
--   守军翻倍影响面：1区 182 个野地、2区 141 个野地（僵尸单与活跃采集队同坐标）。
-- ⚠️ 代码尚未部署到线上 —— 旧 server 仍在跑，会继续产生新的僵尸单，部署后需再跑一次本脚本。
-- ==================================================================
-- ============================================================================

-- ---------- 【0】部署前先看规模 ----------
SELECT COUNT(*) AS ghost_rows
  FROM qq_ezzt.ezfy_order
 WHERE order_type = 7 AND status = 3;

-- 分玩家看（谁被坑得最多）
SELECT user_id, COUNT(*) AS n
  FROM qq_ezzt.ezfy_order
 WHERE order_type = 7 AND status = 3
 GROUP BY user_id
 ORDER BY n DESC;

-- 异常态：result 为空说明「返航归队时没写回兵力」——这些**不要自动删**，要人工核
SELECT id, user_id, city_id, target_x, target_y, troops, result, carry,
       FROM_UNIXTIME(start_time/1000) AS start_at,
       FROM_UNIXTIME(return_time/1000) AS return_at
  FROM qq_ezzt.ezfy_order
 WHERE order_type = 7 AND status = 3
   AND (result IS NULL OR result = '')
 ORDER BY id DESC;

-- ---------- 【1】备份 ----------
CREATE TABLE qq_ezzt.ezfy_order_ghost_bak_20261025 AS
SELECT * FROM qq_ezzt.ezfy_order
 WHERE order_type = 7 AND status = 3;

-- ---------- 【2】删除僵尸行 ----------
-- 条件与代码同口径：
--   order_type=7（驻守采集）且 status=3（已返航归队）
--   且 result 非空（兵力确实已入城）且 carry 为空（负重已清）
-- 现场核对：135/135 全部满足（本地 qq_ezzt 快照，14 个玩家 / 110 个野地坐标）
DELETE FROM qq_ezzt.ezfy_order
 WHERE order_type = 7
   AND status = 3
   AND result IS NOT NULL AND result <> ''
   AND (carry IS NULL OR carry = '');

-- ---------- 【3】验证 ----------
-- 应只剩「result 为空」的异常行（若有，人工处理）
SELECT COUNT(*) AS remain
  FROM qq_ezzt.ezfy_order
 WHERE order_type = 7 AND status = 3;
