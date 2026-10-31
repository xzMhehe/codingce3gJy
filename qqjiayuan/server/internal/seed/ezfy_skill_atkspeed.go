package seed

import (
	"log"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// 军官技能「文案一次性对齐」通用实现（settings 打标，只跑一次）
//
// 为什么必须单独迁移：技能表是 batchKeep（insert-only），而且上一版润色已经用
// `ezfy_skill_desc_polish_20261031` 打过标 → 只改 ezfy_officer_cfg.go 里的默认值
// **落不到已部署的库**。所以每次「同一天内的二次/三次文案调整」都要用**新 key** 再对齐一次。
// 之后管理员在后台改文案不会被顶回去；新库由 batchKeep 直接插入新文案。
func ezfySkillTextSync(db *gorm.DB, key string, ids []int, doneMsg string) {
	if !db.Migrator().HasTable(&model.EzfyCfgSkill{}) || !db.Migrator().HasTable("settings") {
		return
	}
	var marked int64
	db.Raw("SELECT COUNT(*) FROM settings WHERE `key` = ?", key).Scan(&marked)
	if marked > 0 {
		return
	}
	want := make(map[int]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	for _, s := range ezfyEzfyCfgSkill {
		if !want[s.ID] {
			continue
		}
		if err := db.Model(&model.EzfyCfgSkill{}).Where("id = ?", s.ID).
			Updates(map[string]interface{}{"effect": s.Effect, "des": s.Des}).Error; err != nil {
			log.Printf("ezfy 技能文案对齐失败 id=%d: %v（下次启动重试）", s.ID, err)
			return
		}
	}
	db.Exec("INSERT INTO settings (`key`, `value`) VALUES (?, ?)", key, "1")
	log.Printf("%s", doneMsg)
}

// ezfySkillAtkSpeedFix 把「坦克冲锋(5)/闪电疾攻(7)/越岛登陆(9)」的文案一次性改成**攻速**版。
//
// ★★ 2026-10-31 用户需求：
//
//	闪电疾攻 空军速度 → 攻速、越岛登陆 海军速度 → 攻速、坦克冲锋 坦克速度 → 攻速
//	（攻速只决定战斗内出手先后，不再加移动/行军速度，见 ezfyTypeBonus.AtkSpeed）。
func ezfySkillAtkSpeedFix(db *gorm.DB) {
	ezfySkillTextSync(db, "ezfy_skill_atkspeed_20261031", []int{5, 7, 9},
		"ezfy 军官技能：坦克冲锋/闪电疾攻/越岛登陆 效果已改为攻速")
}

// ezfySkillGoldenInsightFix 把「黄金洞察(11)」的文案一次性改成**侦察/情报等级**版。
//
// ★★ 2026-10-31 用户需求：「黄金眼效果不对，它应该就是提高侦察等级的 —— 带队侦查时叠加侦察等级，
// 当市长时提高情报等级、预警更清晰」→ 效果从「战利品」改成「等级」，文案同步（见 ezfyReconSkillLevel）。
func ezfySkillGoldenInsightFix(db *gorm.DB) {
	ezfySkillTextSync(db, "ezfy_skill_golden_insight_20261031", []int{11},
		"ezfy 军官技能：黄金洞察 效果已改为侦察/情报等级")
}
