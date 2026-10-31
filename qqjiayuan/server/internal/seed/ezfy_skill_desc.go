package seed

import (
	"log"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// ezfySkillDescPolishKey settings 打标 key（只跑一次）
const ezfySkillDescPolishKey = "ezfy_skill_desc_polish_20261031"

// ezfySkillDescPolish 把 12 条军官技能的「效果 / 描述」一次性对齐到 2026-10-31 润色版。
//
// ★★ 用户要求：「技能 tab 军官技能描述 润色优化一版本，要求简约、官方、通俗易懂」。
//
//	技能表是 batchKeep（insert-only）—— 管理端可编辑，所以改 ezfy_officer_cfg.go 里的
//	默认值**不会**落到已部署的库。这里用 settings 打标**只对齐一次**：
//	之后管理员在后台再改文案不会被顶回去（与 alignEzfyNavalBomber 同一套路）。
//
// 覆盖列：effect（效果短句）/ des（一句话说明）。新库由 batchKeep 直接插入新值。
func ezfySkillDescPolish(db *gorm.DB) {
	if !db.Migrator().HasTable(&model.EzfyCfgSkill{}) || !db.Migrator().HasTable("settings") {
		return
	}
	var marked int64
	db.Raw("SELECT COUNT(*) FROM settings WHERE `key` = ?", ezfySkillDescPolishKey).Scan(&marked)
	if marked > 0 {
		return
	}
	for _, s := range ezfyEzfyCfgSkill {
		if err := db.Model(&model.EzfyCfgSkill{}).Where("id = ?", s.ID).
			Updates(map[string]interface{}{"effect": s.Effect, "des": s.Des}).Error; err != nil {
			log.Printf("ezfy 技能文案对齐失败 id=%d: %v", s.ID, err)
		}
	}
	db.Exec("INSERT INTO settings (`key`, `value`) VALUES (?, ?)", ezfySkillDescPolishKey, "1")
}
