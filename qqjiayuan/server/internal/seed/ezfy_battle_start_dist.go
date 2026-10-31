package seed

import "gorm.io/gorm"

// ezfyBattleStartDistAutoFix 一次性把「战场初始距离」从旧默认 6000 改成 0（= 自动）。
//
// ★★ 2026-10-31 用户规则：默认距离不再是固定 6000，而是
// `max(攻方最大单兵种射程, 守方最大单兵种射程) + 余量`
// （余量 = ezfy_cfg_limit.battle_start_dist_margin，默认 2000，二战系统配置可调）。
//
// 只改**仍等于旧默认 6000** 的行：管理员后来自己填过别的固定值的，保持不动。
// 用 settings 打标只跑一次 —— 之后管理员把距离填回 6000（当固定值用）也不会被再次清零。
//
// 两张表都有这一列：
//   - ezfy_cfg_limit（二战系统配置）
//   - ezfy_cfg_battle_formula（战斗公式套，优先级更高）
func ezfyBattleStartDistAutoFix(db *gorm.DB) {
	ezfyOneTimeLimitFix(db, "ezfy_battle_start_dist_auto_v1",
		"UPDATE ezfy_cfg_limit SET battle_start_dist = 0 WHERE battle_start_dist = 6000")
	if db.Migrator().HasTable("ezfy_cfg_battle_formula") {
		ezfyOneTimeLimitFix(db, "ezfy_battle_start_dist_auto_formula_v1",
			"UPDATE ezfy_cfg_battle_formula SET battle_start_dist = 0 WHERE battle_start_dist = 6000")
	}
}
