-- 修复 35806363（小哥哥）军官「获取方式：管理端发放 → 抢玩家获取」
--
-- ⚠️ 重要前提（上一版 SQL 失效的根因）：
--   玩家面板上的「家园号」= users.username，**不是**内部 user_id！
--   之前 `SET @uid := 35806363` 用 user_id = 35806363 匹配，0 行命中 → 数据没被改。
--   本版改为自动反查：从 users 表按 username(家园号) 或 id 反查真实 user_id，再统一使用。
--
-- 依据：玩家实际是 PvP 抢自己小号得来 —— 攻方战报 content 含
--       「敌方军官 <名> 忠诚归零, 弃城归降, 已收入我方战俘营(随身装备随俘转移)」。
--       管理端 GetWay 判定原来只认「俘虏敌将:<名>」反查战报 + IsCaptive 兜底，
--       收编后 IsCaptive=0 落入 else → 错误标成「管理端发放」。
-- 处理：① 诊断——反查该玩家真实 user_id，列出名下军官 + 战报中所有 PvP 归降证据；
--       ② 修复——按战报证据把真正抢来的军官 source 修正为 4（抢玩家获取），幂等。
-- 说明：同一名将如有多个实例（系统发放 + 抢来的同名），EXISTS 证据会把同名全改为 4。
--       请先备份后整体执行。代码侧已同步修复管理端显示（ezfy_admin.go：优先按 source 标注）。

-- 0) 自动反查真实 user_id（家园号 35806363 = users.username）
SET @uid := (SELECT id FROM users WHERE username = '35806363' OR id = 35806363 LIMIT 1);
-- 校验反查结果（id 就是真实 user_id，应是非零数字）
SELECT @uid AS real_user_id;

-- ① 诊断 1：该玩家名下军官全览（看 source 现状，0=未标注 1=系统发放 2=军校招募 3=野地俘虏 4=抢玩家获取）
SELECT o.id, c.id AS city_id, c.`name` AS city, o.general_id, o.`name`, o.star, o.source, o.is_captive, o.position, o.update_time
  FROM ezfy_officer o JOIN ezfy_city c ON c.id = o.city_id
 WHERE c.user_id = @uid AND o.deleted_at IS NULL
 ORDER BY o.source ASC, o.id ASC;

-- ① 诊断 2：该玩家战报里的 PvP 归降证据（攻方视角 user_id=本人，一次抢多个将会有多行）
SELECT id, created_at, content
  FROM ezfy_report
 WHERE user_id = @uid AND content LIKE '%忠诚归零, 弃城归降, 已收入我方战俘营%'
 ORDER BY created_at DESC;

-- ② 修复：按战报证据将真正 PvP 抢来的军官 source 修正为 4（幂等：跳过已是 4 的）
UPDATE ezfy_officer o
  JOIN ezfy_city c ON c.id = o.city_id
 SET o.source = 4, o.update_time = NOW()
 WHERE c.user_id = @uid AND o.deleted_at IS NULL AND o.source <> 4
   AND EXISTS (
         SELECT 1 FROM ezfy_report r
          WHERE r.user_id = @uid
            AND r.content LIKE CONCAT('%敌方军官 ', o.`name`, ' 忠诚归零%')
       );

-- ③ 校验：修复后 source=4 的军官应与诊断 2 战报里的将军名匹配（如漏改，把名字发回给开发补一条 UPDATE）
SELECT o.id, c.id AS city_id, c.`name` AS city, o.general_id, o.`name`, o.star, o.source, o.is_captive
  FROM ezfy_officer o JOIN ezfy_city c ON c.id = o.city_id
 WHERE c.user_id = @uid AND o.deleted_at IS NULL
 ORDER BY o.source ASC, o.id ASC;