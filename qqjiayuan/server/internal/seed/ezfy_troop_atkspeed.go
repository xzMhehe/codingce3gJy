package seed

import (
	"log"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// ezfyTroopAtkSpeedFix 给兵种配置补「基础攻速」atk_speed 并回填。
//
// ★★ 2026-10-31 用户反馈「编辑兵种配置没有攻速编辑」：
// 原来兵种只有 speed（速度），战斗引擎直接把它当出手顺序（攻速）用，
// 管理端也就没有独立的攻速字段。现在模型新增 atk_speed：
//   - 0 = 跟随 speed（老库/老快照兼容，行为不变）；
//   - 管理端可单独调攻速，不影响移动/行军（移动/行军仍读 speed）。
//
// AutoMigrate 会给老库补出 atk_speed=0 的列 → 这里一次性回填成 speed，
// 让管理端打开看到的就是原来的出手速度；用 settings 打标只跑一次，
// 之后管理端怎么调都不会被重启冲掉。
func ezfyTroopAtkSpeedFix(db *gorm.DB) {
	if !db.Migrator().HasTable(&model.EzfyCfgTroop{}) || !db.Migrator().HasTable("settings") {
		return
	}
	const key = "ezfy_troop_atkspeed_20261031"
	var marked int64
	db.Raw("SELECT COUNT(*) FROM settings WHERE `key` = ?", key).Scan(&marked)
	if marked > 0 {
		return
	}
	res := db.Exec("UPDATE ezfy_cfg_troop SET atk_speed = speed WHERE atk_speed IS NULL OR atk_speed = 0")
	if res.Error != nil {
		log.Printf("ezfy 兵种攻速：atk_speed 回填失败（下次启动重试）: %v", res.Error)
		return
	}
	db.Exec("INSERT INTO settings (`key`, `value`) VALUES (?, ?)", key, "1")
	log.Printf("ezfy 兵种攻速：atk_speed 已按 speed 回填 %d 行", res.RowsAffected)
}
