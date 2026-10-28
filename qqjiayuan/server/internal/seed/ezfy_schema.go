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
	&model.EzfyActWild{}, &model.EzfyActLevel{}, &model.EzfyActCaptureTry{}, &model.EzfyActCaptureLog{}, &model.EzfyActivity{}, &model.EzfyBattle{},
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

// EnsureEzfyActCaptureLogTable 幂等补建 ezfy_act_capture_log 表（★ 2026-10-28 新表）。
//
// ★★ 为什么必须单独建表：`EnsureEzfySchema` **只补列、不建表**（表不存在直接 continue，
// 交给 `seed.Run`/`RunEzfy` 的 AutoMigrate）。而多机共享库走 `seed.skip: true` 时
// 单库分支根本不跑 AutoMigrate → 新表永远建不出来 → 抓取流水写不进去 →
// 「可抓次数」永远读到 0 → **玩家可以无限抓同一个守将**（比缺列更隐蔽）。
// 与 `EnsureEzfyRansomTable` / `EnsureEzfyVisitTable` 同款兜底，HasTable 幂等。
func EnsureEzfyActCaptureLogTable(db *gorm.DB) {
	if db.Migrator().HasTable(&model.EzfyActCaptureLog{}) {
		return
	}
	if err := db.AutoMigrate(&model.EzfyActCaptureLog{}); err != nil {
		log.Printf("[seed] 补建 ezfy_act_capture_log 表失败（活动守将可抓次数将失效）: %v", err)
	}
}

// ezfyActCaptureLogBackfillKey 抓取流水「历史补记」的一次性标记位（settings.key）。
const ezfyActCaptureLogBackfillKey = "ezfy_act_capture_log_backfill_v1"

// EnsureEzfyActCaptureLogBackfill 一次性把**存量抓取**补进流水表（★ 2026-10-28）。
//
// ★★ 为什么必须补：流水表是本次新加的，「可抓次数」改成数流水条数之后，
//
//	**已经抓过守将的老玩家**在表里一条流水都没有 → 判定当成「没抓过」→ 能再抓一次
//	（甚至抓过又被抢走的能反复抓）；同时管理端「抓取记录」在已经开活动的区（如 2 区）
//	会**一片空白**，运营看不到任何数据。
//
// 口径（与运行时兜底 ezfyActCaptureUsed 完全一致）：按「当前拥有的该守将实例数」补记，
//
//	`source <> recruit`（军校招募的同池军官不算抓过）；一条流水都没有才补差额，
//	所以**天然幂等** —— 标记位丢了重跑也不会重复补。
//
// 局限：只能补「现在还留在手里的」。抓过之后又被别人抢走/策反的历史，没有任何数据可追溯
// （那时没记流水），无法补 —— 这部分玩家仍可能多抓一次，属历史数据不可恢复项。
func EnsureEzfyActCaptureLogBackfill(db *gorm.DB) {
	if !db.Migrator().HasTable(&model.EzfyActCaptureLog{}) {
		return // 表都没建（老库）→ 交给 EnsureEzfyActCaptureLogTable 先建
	}
	var done int64
	db.Model(&model.Setting{}).Where("`key` = ?", ezfyActCaptureLogBackfillKey).Count(&done)
	if done > 0 {
		return
	}
	// ① 每个 (玩家, 守将) 当前拥有几个实例（= 历史上抓到过几个，旧口径的判定依据）
	type ownRow struct {
		UserId    uint
		GeneralId int
		CityId    int64
		Had       int64
	}
	var owns []ownRow
	if err := db.Raw(`SELECT c.user_id AS user_id, o.general_id AS general_id,
	MIN(o.city_id) AS city_id, COUNT(*) AS had
FROM ezfy_officer o JOIN ezfy_city c ON c.id = o.city_id
WHERE o.general_id > 0 AND o.source <> ? AND o.deleted_at IS NULL
GROUP BY c.user_id, o.general_id`, model.EzfyOfficerSourceRecruit).Scan(&owns).Error; err != nil {
		log.Printf("[seed] 活动守将抓取流水回填：统计存量失败（下次启动会重试）: %v", err)
		return
	}
	// ② 已有流水条数（刚建表时全空；部分玩家已被运行时兜底补过）
	type cntRow struct {
		UserId    uint
		GeneralId int
		N         int64
	}
	var cnts []cntRow
	db.Raw("SELECT user_id, general_id, COUNT(*) AS n FROM ezfy_act_capture_log GROUP BY user_id, general_id").Scan(&cnts)
	have := make(map[[2]int64]int64, len(cnts))
	for _, c := range cnts {
		have[[2]int64{int64(c.UserId), int64(c.GeneralId)}] = c.N
	}
	// ③ 只补差额（已有流水 >= 拥有数的不动）
	batch := make([]model.EzfyActCaptureLog, 0, len(owns))
	for _, o := range owns {
		need := o.Had - have[[2]int64{int64(o.UserId), int64(o.GeneralId)}]
		for i := int64(0); i < need; i++ {
			batch = append(batch, model.EzfyActCaptureLog{
				UserId: o.UserId, GeneralId: o.GeneralId,
				X: 0, Y: 0, CityId: o.CityId,
				Kind: model.EzfyActCaptureKindMigrate,
			})
		}
	}
	if len(batch) > 0 {
		if err := db.CreateInBatches(batch, 500).Error; err != nil {
			log.Printf("[seed] 活动守将抓取流水回填失败（下次启动会重试）: %v", err)
			return
		}
		log.Printf("[seed] 活动守将抓取流水：已补记 %d 条历史抓取（玩家 %d 组）", len(batch), len(owns))
	}
	if err := db.Exec("INSERT IGNORE INTO settings (`key`, `value`) VALUES (?, ?)",
		ezfyActCaptureLogBackfillKey, "1").Error; err != nil {
		log.Printf("[seed] 写抓取流水回填标记失败（下次启动会重复补，但差额补是幂等的）: %v", err)
	}
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
