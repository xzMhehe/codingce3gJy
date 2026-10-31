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

// ezfySkillWolfPackFix 把「狼群围猎(8)」的文案改成**海军打海军**口径。
//
// ★★ 2026-10-31 用户反馈两轮：
//
//	① 原描述「海军集群突击，攻击力提升且不限攻击目标」读起来像「全军攻击力都提升」→ 已改过一次；
//	② 随后明确「目标也是对海军兵种」→ 最终口径 = **攻击方海军 + 目标海军**，与四指阵列（空军打空军）同构，
//	   实现是 `addVsType(海军, 海军, 15%/级)`（见 officerTypeBonus / generalTypeBonus）。
//
// 只改文案，数值口径见 handler 侧。
func ezfySkillWolfPackFix(db *gorm.DB) {
	ezfySkillTextSync(db, "ezfy_skill_wolfpack_vs_20261031", []int{8},
		"ezfy 军官技能：狼群围猎 效果/描述已改为「海军对海攻击」")
}

// ezfySkillArcBarrierMaxFix 把「弧形壁垒(2)」文案改成**满级每回合发动**版。
//
// ★★ 2026-10-31 用户口径：弧形壁垒未满级每 3 回合发动 1 回合，满级后每回合发动
// （普通军官 5 级 / 名将 6 级，实现见 ezfyTimedIntervalFor）。
// 上一版文案已打标，必须用新 key 才落得到已部署的库。
func ezfySkillArcBarrierMaxFix(db *gorm.DB) {
	ezfySkillTextSync(db, "ezfy_skill_arc_max_20261031", []int{2},
		"ezfy 军官技能：弧形壁垒 描述已改为「满级每回合发动」")
}

// ezfySkillGoldenInsightFix 把「黄金洞察(11)」的文案一次性改成**侦察/情报等级**版。
//
// ★★ 2026-10-31 用户需求：「黄金眼效果不对，它应该就是提高侦察等级的 —— 带队侦查时叠加侦察等级，
// 当市长时提高情报等级、预警更清晰」→ 效果从「战利品」改成「等级」，文案同步（见 ezfyReconSkillLevel）。
func ezfySkillGoldenInsightFix(db *gorm.DB) {
	ezfySkillTextSync(db, "ezfy_skill_golden_insight_20261031", []int{11},
		"ezfy 军官技能：黄金洞察 效果已改为侦察/情报等级")
}
