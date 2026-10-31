package seed

import (
	"log"

	"gorm.io/gorm"
)

// ezfySkillRenameKey settings 打标 key（只跑一次）
const ezfySkillRenameKey = "ezfy_skill_rename_20261031"

// ezfySkillRenameMap 军官技能改名：**前两个字保留，后两个字迭代**（2026-10-31 用户需求）。
//
//	尖兵突击→尖兵猛攻  弧形防御→弧形壁垒  绝地反击→绝地逆袭  火炮控制→火炮压制
//	坦克突袭→坦克冲锋  四指编队→四指阵列  闪电袭击→闪电疾攻  狼群战术→狼群围猎
//	越岛战术→越岛登陆  弹幕支援→弹幕覆盖  黄金眼→黄金洞察（3 字，保留前两字）
//	机械改造→机械改装
//
// ⚠️ 技能名在代码里是**判定用的字面量**（`case "坦克冲锋"`、`officerHasSkill(o,"绝地逆袭")`），
// 同时也会落库（`ezfy_officer.skill` JSON 数组 / `ezfy_cfg_general.skill` / `ezfy_cfg_skill.name`），
// 所以改名必须「代码 + 存量数据」一起动，否则老玩家已学的技能会**静默失效**（加成归零、不报错）。
var ezfySkillRenameMap = [][2]string{
	{"尖兵突击", "尖兵猛攻"},
	{"弧形防御", "弧形壁垒"},
	{"绝地反击", "绝地逆袭"},
	{"火炮控制", "火炮压制"},
	{"坦克突袭", "坦克冲锋"},
	{"四指编队", "四指阵列"},
	{"闪电袭击", "闪电疾攻"},
	{"狼群战术", "狼群围猎"},
	{"越岛战术", "越岛登陆"},
	{"弹幕支援", "弹幕覆盖"},
	{"黄金眼", "黄金洞察"},
	// ★ 同日中间名「黄金瞳」也映射到最终名：开发/测试机已按中间名替换过的库能自愈；
	//   线上库只有「黄金眼」会命中，另一条匹配 0 行，无副作用。
	{"黄金瞳", "黄金洞察"},
	{"机械改造", "机械改装"},
}

// ezfySkillRename 把已部署库里的旧技能名一次性替换成新名（settings 打标只跑一次）。
//
// 覆盖：
//   - ezfy_cfg_skill.name     技能配置表（batchKeep = insert-only，改种子文案落不到老库）
//   - ezfy_officer.skill      玩家/人机军官已学技能（JSON 数组字符串，直接子串替换）
//   - ezfy_cfg_general.skill  名将 / 野地守将配置的技能（JSON 数组字符串）
//   - ezfy_cfg_equip_set.effect 套装效果文案（如「9件：弧形壁垒+2，反击+3」，纯展示）
//
// 历史战报/战场快照里的旧名**不动**（那是已经打完的文字记录，改了反而对不上）。
// 新库不需要它（种子直接插新名），打标也照写，避免重复执行。
func ezfySkillRename(db *gorm.DB) {
	if !db.Migrator().HasTable("ezfy_cfg_skill") || !db.Migrator().HasTable("settings") {
		return
	}
	var marked int64
	db.Raw("SELECT COUNT(*) FROM settings WHERE `key` = ?", ezfySkillRenameKey).Scan(&marked)
	if marked > 0 {
		return
	}
	for _, kv := range ezfySkillRenameMap {
		from, to := kv[0], kv[1]
		// ① 技能配置表：按名字精确改（主键 id 不动，玩家已学技能只存名字）
		if err := db.Exec("UPDATE ezfy_cfg_skill SET name = ? WHERE name = ?", to, from).Error; err != nil {
			log.Printf("ezfy 技能改名失败 cfg_skill %s→%s: %v（下次启动重试）", from, to, err)
			return
		}
		// ② 军官已学技能（JSON 数组字符串；LIKE 先收窄，避免全表 REPLACE）
		if db.Migrator().HasTable("ezfy_officer") {
			if err := db.Exec("UPDATE ezfy_officer SET skill = REPLACE(skill, ?, ?) WHERE skill LIKE ?",
				from, to, "%"+from+"%").Error; err != nil {
				log.Printf("ezfy 技能改名失败 officer %s→%s: %v（下次启动重试）", from, to, err)
				return
			}
		}
		// ③ 名将 / 野地守将技能
		if db.Migrator().HasTable("ezfy_cfg_general") {
			if err := db.Exec("UPDATE ezfy_cfg_general SET skill = REPLACE(skill, ?, ?) WHERE skill LIKE ?",
				from, to, "%"+from+"%").Error; err != nil {
				log.Printf("ezfy 技能改名失败 cfg_general %s→%s: %v（下次启动重试）", from, to, err)
				return
			}
		}
		// ④ 套装效果文案（纯展示，可能提到技能名）
		if db.Migrator().HasTable("ezfy_cfg_equip_set") {
			if err := db.Exec("UPDATE ezfy_cfg_equip_set SET effect = REPLACE(effect, ?, ?) WHERE effect LIKE ?",
				from, to, "%"+from+"%").Error; err != nil {
				log.Printf("ezfy 技能改名失败 equip_set %s→%s: %v（下次启动重试）", from, to, err)
				return
			}
		}
	}
	db.Exec("INSERT INTO settings (`key`, `value`) VALUES (?, ?)", ezfySkillRenameKey, "1")
	log.Printf("ezfy 军官技能改名：12 条已替换（cfg_skill / officer.skill / cfg_general.skill / 套装文案）")
}
