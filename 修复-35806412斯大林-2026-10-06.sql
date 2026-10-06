-- 修复 35806412（9527）「斯大林没了装备还显示已穿戴」
-- 依据：战报 96927（04:33 荣誉史记升到350级，获得193点属性点）
--       战报 96906（04:29 李靖忠诚归零弃城归降，收入战俘营）
--       战报 97000（04:45 李靖忠诚归零离职，物理删行不清装备）
--       战报 97001（玩家截图：李靖350级带兵反打桐丘战败）
-- 处理：按 general_id=1（斯大林池）重建军官到 35806412 城30(id=30, 273,37)，
--       按「战俘营俘虏」口径补回（is_captive=1、loyalty=30、source=4，与丢官名单被俘修复一致），
--       穿回 36862/36863。李靖 04:29 本就是被 35806412（胜利方）收进战俘营的。
-- 请在执行前先备份；可整体复制到 MySQL 一次性执行

SET @user := 35806412;
SET @city := 30;

-- 1) 幂等保护：若之前已重建过（同城同名斯大林）先删旧，避免重复
DELETE FROM ezfy_officer WHERE city_id = @city AND general_id = 1 AND name = '斯大林' AND deleted_at IS NULL;

-- 2) 重建军官（350级，基础 300/150/200，193自由点待分配，source=4 抢玩家获取，战俘营 is_captive=1）
INSERT INTO ezfy_officer
  (city_id, general_id, name, star, level, exp, military, logistics, learning, loyalty,
   skill, equipment, position, status, is_captive, update_time,
   base_military, base_logistics, base_learning, free_points, duty_exp_at, star_points, source, deleted_at)
VALUES
  (@city, 1, '斯大林', 5, 350, 0, 300, 150, 200, 30,
   '["尖兵突击","弧形防御","绝地反击"]', '[{"crit": 0, "crit_dmg": 0, "def": 58, "dmg": 0, "enhance": 0, "hp": 58, "id": 36862, "learning": 18, "logistics": 18, "military": 35, "move": 40, "name": "赤色锤镰[足部]", "series": "赤色锤镰", "set_id": 26, "slot": "足部", "tier": 4, "type": "军官装备"}, {"crit": 0, "crit_dmg": 0, "def": 60, "dmg": 60, "enhance": 0, "hp": 0, "id": 36863, "learning": 18, "logistics": 18, "military": 35, "move": 0, "name": "赤色锤镰[饰品]", "series": "赤色锤镰", "set_id": 26, "slot": "饰品", "tier": 4, "type": "军官装备"}]', 0, 0, 1, NOW(),
   300, 150, 200, 193, NULL, 0, 4, NULL);

-- 3) 穿回装备（36862 赤色锤镰[足部]、36863 赤色锤镰[饰品]）
SET @nid = LAST_INSERT_ID();
UPDATE ezfy_equipment SET officer_id = @nid WHERE id IN (36862, 36863) AND user_id = @user;

-- 4) 校验
SELECT id, city_id, general_id, name, star, level, military, logistics, learning,
       loyalty, free_points, source, equipment
  FROM ezfy_officer WHERE id = @nid;
SELECT id, user_id, officer_id, name, cfg_id FROM ezfy_equipment WHERE id IN (36862, 36863);

