package seed

import (
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

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
//
// ★ 2026-10-28 v1 → v2：v1 直接数「当前拥有的军官实例数」，把 PvP 抢来的/发放的/招募的
//
//	都算成「抓住过」，且时间全写成回填那一刻（同一秒）—— 用户反馈「抓取错了吧，我的意思
//	是抓住的记录呀」。v2 改成**战报优先**，并清理 v1 的旧补记重算，所以标记位换新 key。
const ezfyActCaptureLogBackfillKey = "ezfy_act_capture_log_backfill_v2"

// EnsureEzfyActCaptureLogBackfill 一次性把**存量抓取**补进流水表（★ 2026-10-28，v2）。
//
// ★★ 为什么必须补：流水表是本次新加的，「可抓次数」改成数流水条数之后，
//
//	**已经抓过守将的老玩家**在表里一条流水都没有 → 判定当成「没抓过」→ 能再抓一次
//	（甚至抓过又被抢走的能反复抓）；同时管理端「抓取记录」在已经开活动的区（如 2 区）
//	会**一片空白**，运营看不到任何数据。
//
// ★★ 口径（v2，用户纠正后）—— **战报优先，军官兜底**：
//
//	① 战报正文里有 `俘虏敌将:XXX(5星, 忠诚30)` = 真抓到过。时间、坐标全用战报的
//	   （标题尾部 `(x,y)`），kind=2「战报补记」。战报只留几天，但能覆盖**抓过又被抢走**的。
//	② 战报覆盖不到的（更早、战报已被清理）才按军官实例兜底，kind=3「军官补记」，
//	   时间用军官的 update_time（近似抓取时间）。
//	   ★ 只认 `source IN (0,3)`（历史未标注 / 野地俘虏）——
//	   **PvP 抢来的（4）、管理端发放（1）、军校招募（2）都不算「抓过」**：
//	   抢来的守将不是自己抓的，不该消耗「可抓次数」。
//	③ 兜底**只补差额**（已有流水 >= 拥有实例数的不动）→ 天然幂等。
//
// 局限：战报已删 + 军官又被抢走的，没有任何数据可追溯（属历史数据不可恢复项）。
func EnsureEzfyActCaptureLogBackfill(db *gorm.DB) {
	if !db.Migrator().HasTable(&model.EzfyActCaptureLog{}) {
		return // 表都没建（老库）→ 交给 EnsureEzfyActCaptureLogTable 先建
	}
	var done int64
	db.Model(&model.Setting{}).Where("`key` = ?", ezfyActCaptureLogBackfillKey).Count(&done)
	if done > 0 {
		return
	}
	// ① 旧补记一律重算（kind=1 是线上真打出来的，绝不动）
	if res := db.Where("kind IN ?", []int{model.EzfyActCaptureKindReport, model.EzfyActCaptureKindOfficer}).
		Delete(&model.EzfyActCaptureLog{}); res.Error != nil {
		log.Printf("[seed] 抓取流水：清理旧补记失败（本次不重算，下次启动重试）: %v", res.Error)
		return
	} else if res.RowsAffected > 0 {
		log.Printf("[seed] 抓取流水：清理旧补记 %d 条，按「战报优先」重算", res.RowsAffected)
	}

	// ② 军官池「原始名 → id」（战报里记的是军官池名，玩家改名不影响）
	var gens []model.EzfyCfgGeneral
	db.Select("id, name").Where("name <> ''").Find(&gens)
	byName := make(map[string]int, len(gens))
	for _, g := range gens {
		byName[g.Name] = g.ID
	}
	// 名字长的优先匹配（避免「巴顿」吃掉「巴顿·史密斯」这类前缀重合）
	names := make([]string, 0, len(byName))
	for n := range byName {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool { return len(names[i]) > len(names[j]) })

	// ③ 战报补记
	type repRow struct {
		UserId    uint
		Title     string
		Content   string
		CreatedAt time.Time
	}
	// ★★ 战报**只在这里读一次**（一次性历史还原），运行时 / 管理端列表都**不碰战报**
	//
	//	（用户 2026-10-28 提醒：「别用战报关联查询，战报只保留 3 天的我会定期删除」）。
	//	所以：① 不做 JOIN，独立查一次；② 限定最近 7 天（战报本来只留 3 天，留点余量），
	//	避免 longtext 全表扫；③ 读到的结果**落库成流水**，之后战报删光也不影响任何功能 ——
	//	抓取记录的长期唯一数据源就是 ezfy_act_capture_log 自己。
	var reps []repRow
	if err := db.Model(&model.EzfyReport{}).
		Select("user_id, title, content, created_at").
		Where("content LIKE ?", "%俘虏敌将:%").
		Where("created_at >= ?", time.Now().AddDate(0, 0, -7)).
		Order("created_at ASC").Scan(&reps).Error; err != nil {
		log.Printf("[seed] 抓取流水：读战报失败（下次启动重试）: %v", err)
		return
	}
	batch := make([]model.EzfyActCaptureLog, 0, len(reps))
	for _, r := range reps {
		i := strings.Index(r.Content, "俘虏敌将:")
		if i < 0 {
			continue
		}
		rest := r.Content[i+len("俘虏敌将:"):]
		gid := 0
		for _, n := range names {
			if strings.HasPrefix(rest, n) {
				gid = byName[n]
				break
			}
		}
		if gid <= 0 {
			continue // 军官池里已删掉这个守将 → 无从对应
		}
		x, y := actCaptureTitleXY(r.Title)
		batch = append(batch, model.EzfyActCaptureLog{
			UserId: r.UserId, GeneralId: gid, X: x, Y: y,
			Kind: model.EzfyActCaptureKindReport, CreatedAt: r.CreatedAt,
		})
	}
	if len(batch) > 0 {
		if err := db.CreateInBatches(batch, 500).Error; err != nil {
			log.Printf("[seed] 抓取流水：战报补记写入失败（下次启动重试）: %v", err)
			return
		}
		log.Printf("[seed] 抓取流水：从战报补记 %d 条真实抓取记录", len(batch))
	}

	// ④ 军官兜底：战报已删的更早抓取（只认 历史未标注 / 野地俘虏）
	type offRow struct {
		UserId     uint
		GeneralId  int
		CityId     int64
		UpdateTime time.Time
	}
	var offs []offRow
	if err := db.Raw(`SELECT c.user_id AS user_id, o.general_id AS general_id,
	o.city_id AS city_id, o.update_time AS update_time
FROM ezfy_officer o JOIN ezfy_city c ON c.id = o.city_id
WHERE o.general_id > 0 AND o.source IN (?, ?) AND o.deleted_at IS NULL
ORDER BY o.update_time ASC`,
		model.EzfyOfficerSourceUnknown, model.EzfyOfficerSourceWildland).Scan(&offs).Error; err != nil {
		log.Printf("[seed] 抓取流水：统计存量军官失败（下次启动重试）: %v", err)
		return
	}
	// 已有流水条数（含刚写入的战报补记）
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
	type ugKey struct{ u, g int64 }
	groups := make(map[ugKey][]offRow, len(offs))
	order := make([]ugKey, 0, len(offs))
	for _, o := range offs {
		k := ugKey{int64(o.UserId), int64(o.GeneralId)}
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], o)
	}
	batch2 := make([]model.EzfyActCaptureLog, 0, len(offs))
	for _, k := range order {
		rows := groups[k]
		need := int64(len(rows)) - have[[2]int64{k.u, k.g}]
		if need <= 0 {
			continue
		}
		// 战报覆盖的是「最近」的，剩下没记录的应是更早的 → 取最早的那几条
		for i := 0; i < len(rows) && int64(i) < need; i++ {
			batch2 = append(batch2, model.EzfyActCaptureLog{
				UserId: rows[i].UserId, GeneralId: rows[i].GeneralId,
				X: 0, Y: 0, CityId: rows[i].CityId,
				Kind: model.EzfyActCaptureKindOfficer, CreatedAt: rows[i].UpdateTime,
			})
		}
	}
	if len(batch2) > 0 {
		if err := db.CreateInBatches(batch2, 500).Error; err != nil {
			log.Printf("[seed] 抓取流水：军官补记写入失败（下次启动重试）: %v", err)
			return
		}
		log.Printf("[seed] 抓取流水：按军官实例补记 %d 条（战报已删的更早抓取）", len(batch2))
	}
	if err := db.Exec("INSERT IGNORE INTO settings (`key`, `value`) VALUES (?, ?)",
		ezfyActCaptureLogBackfillKey, "1").Error; err != nil {
		log.Printf("[seed] 写抓取流水回填标记失败（下次启动会重复补，但差额补是幂等的）: %v", err)
	}
}

// actCaptureTitleXY 从战报标题尾部取坐标，如「活动野地3级战斗报告: 活动野地3级(108,280)」→ 108,280
func actCaptureTitleXY(title string) (int, int) {
	i := strings.LastIndex(title, "(")
	if i < 0 || !strings.HasSuffix(title, ")") {
		return 0, 0
	}
	parts := strings.Split(strings.TrimSuffix(title[i+1:], ")"), ",")
	if len(parts) != 2 {
		return 0, 0
	}
	x, e1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	y, e2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if e1 != nil || e2 != nil {
		return 0, 0
	}
	return x, y
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
