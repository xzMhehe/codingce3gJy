package seed

import (
	"log"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// seedEzfyActLevels 活动目标等级配置（活动野地 / 活动寇城 / 特殊城市）
//
// ★ 2026-10-22 用户需求：三类活动目标的「等级」配置要能在
// 二战管理 → 地图管理 → 活动目标配置 里维护。
//
// 这里只在表**完全为空**时灌入一份与代码默认值一致的初始数据：
//   - 老库升级后会看到 3 类 × 3 级共 9 行，直接改数值即可生效；
//   - 管理端删过/改过之后，重启不会覆盖（只有清空整表才会重新初始化）。
//
// 守军 Troops / 宝物 Treasures / 道具 DropItems 默认留空：
//   - 守军空 = 沿用代码内置的陆/海配比（活动野地海里出海军）；
//   - 宝物/道具空 = 不额外掉落，由运营按需配置。
func seedEzfyActLevels(db *gorm.DB) {
	var n int64
	if err := db.Model(&model.EzfyActLevel{}).Count(&n).Error; err != nil {
		log.Printf("ezfy 活动目标等级配置读取失败: %v", err)
		return
	}
	if n > 0 {
		return // 已有配置（可能是管理端维护过的）→ 绝不覆盖
	}
	rows := []model.EzfyActLevel{
		// 活动野地：10万/20万/30万守军；资源 10万/25万/60万，黄金 5千/1万/2万，声望 200/400/600
		{ActType: 1, Level: 1, Res: 100000, Gold: 5000, Prestige: 200,
			Des: "活动野地1级：10万守军；胜利获得大量资源+黄金+必定掉落宝物+大量声望"},
		{ActType: 1, Level: 2, Res: 250000, Gold: 10000, Prestige: 400,
			Des: "活动野地2级：20万守军；奖励高于1级"},
		{ActType: 1, Level: 3, Res: 600000, Gold: 20000, Prestige: 600,
			Des: "活动野地3级：30万守军；奖励高于2级"},
		// 活动寇城：20万/40万/60万守军；资源 20万/50万/120万，黄金 1万/2万/3万，声望 600/1200/1800
		{ActType: 2, Level: 1, Res: 200000, Gold: 10000, Prestige: 600,
			Des: "活动寇城1级：20万守军；胜利获得巨大资源+黄金+宝物+声望"},
		{ActType: 2, Level: 2, Res: 500000, Gold: 20000, Prestige: 1200,
			Des: "活动寇城2级：40万守军；奖励高于1级"},
		{ActType: 2, Level: 3, Res: 1200000, Gold: 30000, Prestige: 1800,
			Des: "活动寇城3级：60万守军；奖励高于2级"},
		// 特殊城市：100万/200万/500万守军；资源 50万/125万/300万，黄金 2万/4万/6万，声望 1000/2000/3000
		{ActType: 3, Level: 1, Res: 500000, Gold: 20000, Prestige: 1000,
			Des: "特殊城市1级：100万守军；全服最强活动目标，胜利必定获得高级/特殊宝物"},
		{ActType: 3, Level: 2, Res: 1250000, Gold: 40000, Prestige: 2000,
			Des: "特殊城市2级：200万守军；奖励高于1级"},
		{ActType: 3, Level: 3, Res: 3000000, Gold: 60000, Prestige: 3000,
			Des: "特殊城市3级：500万守军；全服最强活动目标，奖励最丰厚"},
	}
	if err := db.Create(&rows).Error; err != nil {
		log.Printf("ezfy 活动目标等级配置种子失败: %v", err)
	}
}

// EnsureEzfyActLevelTable 单库 skip 模式（多机共享库）下幂等补建「活动目标等级配置」表。
//
// ★ 与 EnsureEzfyRansomTable / EnsureEzfyUserChestTable 同款原因：
//
//	skip 分支不跑 seed.Run 的 AutoMigrate，新表不会自动建；
//	不补的话管理端保存会报 Error 1146（表不存在），运行时也读不到配置。
func EnsureEzfyActLevelTable(db *gorm.DB) {
	if !db.Migrator().HasTable(&model.EzfyActLevel{}) {
		if err := db.AutoMigrate(&model.EzfyActLevel{}); err != nil {
			log.Printf("ezfy_act_level 建表失败: %v", err)
			return
		}
		log.Printf("ezfy 补表: ezfy_act_level")
	}
	seedEzfyActLevels(db)
}
