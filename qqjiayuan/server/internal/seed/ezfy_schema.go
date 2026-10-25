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
	&model.EzfyActWild{}, &model.EzfyActLevel{}, &model.EzfyActCaptureTry{}, &model.EzfyActivity{}, &model.EzfyBattle{},
	&model.EzfyCfgBuilding{}, &model.EzfyCfgBuildingLevel{}, &model.EzfyCfgChest{},
	&model.EzfyCfgChestItem{}, &model.EzfyCfgEquipSet{}, &model.EzfyCfgEquipment{},
	&model.EzfyCfgGeneral{}, &model.EzfyCfgGeneralSkill{}, &model.EzfyCfgItem{}, &model.EzfyCfgLimit{},
	&model.EzfyCfgBattleFormula{},
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
	&model.EzfyUserTech{}, &model.EzfyVisitLog{}, &model.EzfyWar{}, &model.EzfyWildland{},
	&model.EzfyWordFilter{}, &model.EzfyWounded{},
}

// ezfySchemaCatalog 一次拉全库「表 / 列」信息（跨 WAN 只发 2 条查询）。
//
// ★ 2026-10-10 性能（用户「部署启动怎么这么久」）：原实现用 HasTable/HasColumn
//
//	逐表逐字段探测，60 张表 × 每表二三十字段 ≈ 近两千次串行跨 WAN 查询，
//	每次 100~300ms → 启动链 10~40s+。改成一次 SELECT 拿全库列，内存比对。
func ezfySchemaCatalog(db *gorm.DB) (tables map[string]bool, cols map[string]map[string]bool) {
	tables = map[string]bool{}
	cols = map[string]map[string]bool{}
	if rows, err := db.Raw("SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE()").Rows(); err == nil {
		for rows.Next() {
			var t string
			if rows.Scan(&t) == nil {
				tables[t] = true
			}
		}
		rows.Close()
	}
	if rows, err := db.Raw("SELECT TABLE_NAME, COLUMN_NAME FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE()").Rows(); err == nil {
		for rows.Next() {
			var t, c string
			if rows.Scan(&t, &c) == nil {
				if cols[t] == nil {
					cols[t] = map[string]bool{}
				}
				cols[t][c] = true
			}
		}
		rows.Close()
	}
	return
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
//
//	所以对线上大表也安全（MySQL 8.0 的 ADD COLUMN 默认 INSTANT）。
func EnsureEzfySchema(db *gorm.DB) {
	tables, cols := ezfySchemaCatalog(db)
	for _, m := range ezfySchemaModels {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(m); err != nil {
			log.Printf("【严重】ezfy 表结构解析失败（补列被跳过）: %v", err)
			continue
		}
		if !tables[stmt.Schema.Table] {
			continue // 表还没建（全新库走 seed.Run）→ 交给 AutoMigrate
		}
		for _, f := range stmt.Schema.Fields {
			if f.DBName == "" || f.IgnoreMigration {
				continue
			}
			if cols[stmt.Schema.Table][f.DBName] {
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

// homeSchemaModels 家园库（非二战）全部需要在 skip 分支补列的表。
//
// ⚠️ 与 ezfySchemaModels 同款坑（2026-10-16 注册报 `Unknown column 'session_at'`）：
//   给 model 加字段后，列只在 `seed.Run` 的 AutoMigrate 里建；多机共享库走
//   `seed.skip: true` 会跳过整段 seed.Run → 库里永远缺这一列 → 注册/写表
//   `INSERT` 报 `ERROR 1054 Unknown column`。之前只补了 ezfy 侧，家园 users 的
//   session_at（2026-10-15 token 服务端作废）漏了 → 新号注册全字段 INSERT 直接报错。
//   新增家园表 model 时**记得往这里加一笔**。
var homeSchemaModels = []interface{}{
	&model.User{}, &model.UserBadge{}, &model.NumHistory{},
	&model.Friendship{}, &model.FriendGroup{}, &model.FriendGroupItem{}, &model.FriendApply{}, &model.FriendBlack{},
	&model.PrivateMessage{}, &model.ChatMessage{}, &model.Notification{},
	&model.Home{}, &model.HomeNews{}, &model.HomeFavorite{}, &model.UserContact{},
	&model.Invite{}, &model.JwtPlayer{}, &model.JwtItem{}, &model.JwtBag{},
}

// EnsureHomeSchema 幂等补齐家园库**缺失列**（main.go 的 skip 分支必须调用，见 2026-10-16
// 注册 `Unknown column 'session_at'` 事故注释）。逻辑与 EnsureEzfySchema 完全一致：
// 只 ADD COLUMN，不删列不改类型——对线上大表安全（MySQL 8.0 ADD COLUMN 默认 INSTANT）。
func EnsureHomeSchema(db *gorm.DB) {
	tables, cols := ezfySchemaCatalog(db)
	for _, m := range homeSchemaModels {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(m); err != nil {
			log.Printf("【严重】家园表结构解析失败（补列被跳过）: %v", err)
			continue
		}
		if !tables[stmt.Schema.Table] {
			continue // 表还没建（全新库走 seed.Run）→ 交给 AutoMigrate
		}
		for _, f := range stmt.Schema.Fields {
			if f.DBName == "" || f.IgnoreMigration {
				continue
			}
			if cols[stmt.Schema.Table][f.DBName] {
				continue
			}
			if err := db.Migrator().AddColumn(m, f.Name); err != nil {
				log.Printf("【严重】%s.%s 补列失败（该表的写入会报 Unknown column）: %v",
					stmt.Schema.Table, f.DBName, err)
				continue
			}
			log.Printf("家园补列: %s.%s", stmt.Schema.Table, f.DBName)
		}
	}
}
