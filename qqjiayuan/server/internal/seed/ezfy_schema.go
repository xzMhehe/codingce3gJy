package seed

import (
	"log"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// ezfySchemaModels 二战风云全部数据表（与 seed.Run 里 AutoMigrate 的 ezfy 部分同源）。
//
// ⚠️ 新增 Ezfy* model 时**记得往这里加一笔**，否则它的新列在 skip 分支上补不出来。
var ezfySchemaModels = []interface{}{
	&model.EzfyActWild{}, &model.EzfyActivity{}, &model.EzfyBattle{},
	&model.EzfyCfgBuilding{}, &model.EzfyCfgBuildingLevel{}, &model.EzfyCfgChest{},
	&model.EzfyCfgChestItem{}, &model.EzfyCfgEquipSet{}, &model.EzfyCfgEquipment{},
	&model.EzfyCfgGeneral{}, &model.EzfyCfgItem{}, &model.EzfyCfgLimit{},
	&model.EzfyCfgRank{}, &model.EzfyCfgResource{}, &model.EzfyCfgScheme{},
	&model.EzfyCfgSkill{}, &model.EzfyCfgTask{}, &model.EzfyCfgTaskType{},
	&model.EzfyCfgTech{}, &model.EzfyCfgTechLevel{}, &model.EzfyCfgTroop{},
	&model.EzfyCfgWildland{}, &model.EzfyChat{}, &model.EzfyCity{},
	&model.EzfyCityBuilding{}, &model.EzfyCityEffect{}, &model.EzfyCityTarget{},
	&model.EzfyCityTech{}, &model.EzfyCityTroop{}, &model.EzfyCorps{},
	&model.EzfyCorpsApply{}, &model.EzfyCorpsChat{}, &model.EzfyCorpsMall{},
	&model.EzfyCorpsMallLog{}, &model.EzfyCorpsMember{}, &model.EzfyCorpsRelation{},
	&model.EzfyCorpsWar{}, &model.EzfyDiamondLog{}, &model.EzfyEquipment{},
	&model.EzfyExchange{}, &model.EzfyExchangeTemplate{}, &model.EzfyFriend{},
	&model.EzfyFriendApply{}, &model.EzfyGift{}, &model.EzfyItem{},
	&model.EzfyItemUseLog{}, &model.EzfyLoveCard{}, &model.EzfyMapArea{},
	&model.EzfyMapStar{}, &model.EzfyMapTile{}, &model.EzfyNotice{},
	&model.EzfyOccupy{}, &model.EzfyOfficer{}, &model.EzfyOrder{},
	&model.EzfyPreset{}, &model.EzfyProfile{}, &model.EzfyRansom{},
	&model.EzfyRecruit{}, &model.EzfyReport{}, &model.EzfySign{},
	&model.EzfyTask{}, &model.EzfyTrainQueue{}, &model.EzfyTreasureSign{},
	&model.EzfyUserTech{}, &model.EzfyWar{}, &model.EzfyWildland{},
	&model.EzfyWordFilter{}, &model.EzfyWounded{},
}

// EnsureEzfySchema 幂等补齐 ezfy 全部表的**缺失列**（main.go 的 skip 分支必须调用）。
//
// ★★ 为什么需要（2026-10-09 第 5 次踩同一个坑）：
//
//	给 model 加字段后，列只在 `seed.Run` 的 `AutoMigrate` 里建；多机共享库走
//	`seed.skip: true` 会跳过整段 seed.Run → 库里永远缺这一列 → `INSERT` 报
//	`ERROR 1054 Unknown column` → 而写入的 error 往往被忽略 → **静默失败**。
//
//	已经踩过的：`ezfy_officer.source`（军官全写不进）、`ezfy_battle.atk_lock/def_lock`
//	（指挥室全失效）、`ezfy_order.auto_battle`（出征订单静默丢失、兵油照扣）、
//	`ezfy_report.type_name`（**战报全丢**）。
//	每次都手写一个 `EnsureXxxColumns` 太容易漏 —— 这里对 ezfy 全部 model 做一次通用补齐。
//
// ★ 安全性：**只 ADD COLUMN，不删列、不改已有列的类型**（不走 AutoMigrate），
//   所以对线上大表也安全（MySQL 8.0 的 ADD COLUMN 默认 INSTANT）。
func EnsureEzfySchema(db *gorm.DB) {
	for _, m := range ezfySchemaModels {
		if !db.Migrator().HasTable(m) {
			continue // 表还没建（全新库走 seed.Run）→ 交给 AutoMigrate
		}
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(m); err != nil {
			log.Printf("【严重】ezfy 表结构解析失败（补列被跳过）: %v", err)
			continue
		}
		for _, f := range stmt.Schema.Fields {
			if f.DBName == "" || f.IgnoreMigration {
				continue
			}
			if db.Migrator().HasColumn(m, f.Name) {
				continue
			}
			if err := db.Migrator().AddColumn(m, f.Name); err != nil {
				log.Printf("【严重】%s.%s 补列失败（该表的写入会报 Unknown column）: %v",
					stmt.Schema.Table, f.DBName, err)
				continue
			}
			log.Printf("ezfy 补列: %s.%s", stmt.Schema.Table, f.DBName)
		}
	}
}
