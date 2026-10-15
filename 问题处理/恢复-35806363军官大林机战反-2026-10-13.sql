-- 恢复 35806363（种田玩家）军官「大林①机战反」(ezfy_officer id=5774)
--
-- 原因：2026-10-13 07:31~07:48 玩家用该军官带队反复掠夺寇城10级(463,381)，
--       其中 12 次战败，每次忠诚度 -9，忠诚归零后于 07:48:09 触发「将领离职」（系统报告 id=378945）。
--       玩家并不知晓「战败扣忠诚、归零离职」这一规则，属运营关怀恢复。
-- 处理：① 取消逻辑删除(deleted_at → NULL)；② 忠诚度回满 100；③ status 1(出征中) → 0(在职)
--       （出征早已结束，保持出征中会让玩家误以为部队仍在野外）；④ is_captive 归 0；update_time 刷新。
-- 说明：离职报告 ezfy_report id=378945 保留（2026-10-13 起代码已把「当天离职播报」置顶展示，玩家可见）。

SET @oid := 5774;

-- ① 诊断：恢复前快照
SELECT id, city_id, general_id, `name`, star, level, loyalty, status, is_captive, source, deleted_at
  FROM ezfy_officer WHERE id = @oid;

-- ② 恢复
UPDATE ezfy_officer
   SET deleted_at = NULL, loyalty = 100, status = 0, is_captive = 0, update_time = NOW()
 WHERE id = @oid;

-- ③ 校验：恢复后状态（deleted_at 应为 NULL、loyalty 100、status 0）
SELECT id, city_id, general_id, `name`, star, level, loyalty, status, is_captive, source, deleted_at
  FROM ezfy_officer WHERE id = @oid;
