package seed

import (
	"fmt"
	"log"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"qqjiayuan/server/internal/model"
)

// seedEzfy 二战风云：配置表幂等种子（数据源自 stzb-fk inithebing.sql 转换，见 ezfy_cfg_gen.go）
//
// 两种写入策略，按「该表是否允许管理端改」区分：
//
//   - batch（upsert，UpdateAll）：纯代码维护的表（科技/野地/道具/任务）。
//     每次启动用代码里的值对齐库，避免「代码对、库里错」的配置漂移。
//   - batchKeep（insert-only，DoNothing）：**管理端可编辑**的表。
//     只补缺失行，已存在的行原样保留 —— 否则管理端在「建筑总配置 / 兵种配置 /
//     名将列表 / 技能列表 / 装备列表」里改的参数，会在下次重启时被种子悄悄改回去。
//     代价：改了 ezfy_cfg_gen.go 里的默认值不会自动落到老库；
//     需要强制对齐时，删掉该行重启即可。
func seedEzfy(db *gorm.DB) {
	batch := func(items interface{}, table string) {
		if err := db.Clauses(clause.OnConflict{UpdateAll: true}).
			CreateInBatches(items, 200).Error; err != nil {
			log.Printf("ezfy 种子失败 %s: %v", table, err)
		}
	}
	// 只补缺，不覆盖：管理端可编辑的配置表
	batchKeep := func(items interface{}, table string) {
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).
			CreateInBatches(items, 200).Error; err != nil {
			log.Printf("ezfy 种子失败 %s: %v", table, err)
		}
	}

	// —— 管理端可编辑：只补缺 ——
	batchKeep(ezfyEzfyCfgBuilding, "ezfy_cfg_building")
	batchKeep(ezfyEzfyCfgBuildingLevel, "ezfy_cfg_building_level")
	batchKeep(ezfyEzfyCfgTroop, "ezfy_cfg_troop")
	batchKeep(ezfyEzfyCfgGeneral, "ezfy_cfg_general")
	batchKeep(ezfyEzfyCfgSkill, "ezfy_cfg_skill")
	batchKeep(ezfyEzfyCfgEquipment, "ezfy_cfg_equipment")

	// 阵营兵种名: inithebing.sql 里是用 UPDATE 补的(不在 INSERT 列里),
	// 早期生成脚本只解析了 INSERT 导致全丢, 这里对「还没填过」的行补一次。
	// 加 name_axis='' 条件是为了不覆盖管理端改过的兵种名。
	for _, t := range ezfyEzfyCfgTroop {
		if t.NameAxis == "" && t.NameAlly == "" {
			continue
		}
		if err := db.Model(&model.EzfyCfgTroop{}).
			Where("id = ? AND (name_axis = '' OR name_axis IS NULL)", t.ID).
			Updates(map[string]interface{}{"name_axis": t.NameAxis, "name_ally": t.NameAlly}).Error; err != nil {
			log.Printf("ezfy 阵营兵种名写入失败 id=%d: %v", t.ID, err)
		}
	}

	// —— 纯代码维护：每次对齐 ——
	batch(ezfyEzfyCfgTech, "ezfy_cfg_tech")
	batch(ezfyEzfyCfgTechLevel, "ezfy_cfg_tech_level")
	// ★ 野地类型管理端可编辑（守军 + 守军军官 officer_id），必须只补缺不覆盖，
	//   否则每次重启会把管理端/守将补缺的 officer_id 冲回 0（2026-09-24 野地军官需求）。
	batchKeep(ezfyEzfyCfgWildland, "ezfy_cfg_wildland")
	batch(ezfyEzfyCfgItem, "ezfy_cfg_item")
	// ★ 2026-09-26 「道具配置按现在线上跑的初始化」：
	//   stock 列自带 DB 默认值 100，而 GORM 对「带 default 标签的字段」会跳过 Go 零值，
	//   于是线上「0 = 已售罄」的道具（大资源包 / 增产令）在库里会落成 100
	//   （非 0 库存不受影响，上面的 batch 正常写入）。
	//   这里只对「快照库存为 0」的条目补一次显式写，让售罄状态也能原样初始化。
	// ★ 2026-09-27 快照里商城道具（ID 1~12）已无售罄项，这段循环暂不生效，保留备用。
	for i := range ezfyEzfyCfgItem {
		if ezfyEzfyCfgItem[i].Stock != 0 {
			continue
		}
		db.Model(&model.EzfyCfgItem{}).Where("id = ?", ezfyEzfyCfgItem[i].ID).
			Update("stock", 0)
	}
	// ★ 任务类型/任务：改「只补缺不覆盖」—— 管理端在「数据管理」里调的奖励(数值)
	//   不能被下次启动的种子悄悄改回去（「后台能灵活配置奖励」）。
	batchKeep(ezfyEzfyCfgTaskType, "ezfy_cfg_task_type")
	batchKeep(ezfyEzfyCfgTask, "ezfy_cfg_task")

	seedEzfyNotices(db)
	seedEzfyOfficerItems(db)
	seedEzfyMoveItems(db)
	seedEzfyLoveCardItems(db)
	seedEzfyActivities(db)
	seedEzfyResources(db)
	seedEzfyRanks(db)

	// —— 军官池（普通军官 1000 名）/ 装备套装 / 存量军官属性点迁移 ——
	// ★ 必须在 batchKeep(ezfy_cfg_general) 之后：名将已入库，再补普通军官不会互相覆盖。
	normalizeEzfyNewCols(db)
	seedEzfyOfficerPool(db)
	seedEzfyEliteFiveStars(db) // ★ 2026-09-30 五星精英（后勤/学识 各50名）
	seedEzfyWildOfficers(db)
	seedEzfyExchangeTpls(db)
	seedEzfyEquipSets(db)
	backfillOfficerEquipSetBonus(db)
	nerfEquipSetPct(db)
	repairEquipSnapshots(db)
	backfillLooseEquipPrice(db)
	seedEzfyChests(db)
	backfillEzfyChestPool(db)
	seedEzfySchemes(db)
	migrateOfficerAttrPoints(db)
	fixEzfySignIndex(db)

	EnsureEzfyIndexes(db)
}

// EnsureEzfyIndexes 幂等补建高频查询所需的索引。
//
// ★ 为什么单独导出：多机部署时只有一台跑全量 seed，另一台走 `seed.skip: true` 路径
// （main.go 只调 EnsureEzfyLimitColumns）—— 把索引也挂到那条路径上，
// 保证任何一台启动都会把缺的索引补齐（ensureEzfyIndex 本身幂等，先到先建）。
//
// ★ 2026-10-03 起：首页 /view 30s 轮询的命令数/未读战报计数、装备检索，
//   都按 user_id + status/is_read 过滤。只靠单列 user_id 会在引擎层窄化后再筛，
//   补复合索引直接命中，省掉每一段的扫描。
// ★ 2026-10-05 起（用户反馈「这些接口还是 2s+」）：跨 WAN 每次往返 13~37ms，
//   慢接口的每一个**全表扫**都会被放大。以下三张表原先**只有主键**（或单列），
//   而懒结算/展示每次都按这些条件过滤：
//     · ezfy_occupy     —— cityOf() 查 `city_id + status=1`、wildfull 查 `atk_city_id + status=1`
//     · ezfy_wildland   —— 地图/详情按 `x,y` 精确定位、地图按 x 区间裁剪
//     · ezfy_order      —— processIncoming 按 `target_id IN (...) + status=0` 找来袭订单
//     · ezfy_city_tech  —— 多城研究结算按 `city_id IN (...) + status=1`
//     · ezfy_officer    —— 市长加成按 `city_id + position` 取 1 行
func EnsureEzfyIndexes(db *gorm.DB) {
	ensureEzfyIndex(db, "ezfy_order", "idx_order_user_status", "user_id,status,order_type", false)
	ensureEzfyIndex(db, "ezfy_report", "idx_report_user_read", "user_id,is_read", false)
	// ★ 2026-10-08 管理端「战报查询」提速：按玩家列列表要 ORDER BY id DESC、
	//   新增时间起止检索按 created_at 过滤 —— 各补一个索引（存量表走幂等 ALTER）。
	//   名字与线上已建的 idx_report_user_id_desc / idx_report_created 对齐，ensureEzfyIndex 幂等跳过、不重复建。
	ensureEzfyIndex(db, "ezfy_report", "idx_report_user_id_desc", "user_id,id", false)
	ensureEzfyIndex(db, "ezfy_report", "idx_report_created", "created_at", false)
	// ★ 2026-10-08 管理端按「类型名」过滤走持久化列 type_name（同列回填见 cmd/backfilltype）。
	//   用复合索引 (type_name,id)：`WHERE type_name=? ORDER BY id DESC` 可直接定位顶部 N 条。
	ensureEzfyIndex(db, "ezfy_report", "idx_report_type_id", "type_name,id", false)
	ensureEzfyIndex(db, "ezfy_equipment", "idx_equip_user_cfg_off", "user_id,cfg_id,officer_id", false)

	// ★ 2026-10-05 二战风云慢接口补索引（见函数注释）
	ensureEzfyIndex(db, "ezfy_occupy", "idx_occupy_city_status", "city_id,status", false)
	ensureEzfyIndex(db, "ezfy_occupy", "idx_occupy_atkcity_status", "atk_city_id,status", false)
	ensureEzfyIndex(db, "ezfy_wildland", "idx_wildland_xy", "x,y", false)
	ensureEzfyIndex(db, "ezfy_order", "idx_order_target_status", "target_id,status", false)
	ensureEzfyIndex(db, "ezfy_city_tech", "idx_city_tech_status", "city_id,status", false)
	ensureEzfyIndex(db, "ezfy_officer", "idx_officer_city_position", "city_id,position", false)

	// ★ 2026-10-03 家园论坛索引：版块帖子列表(board_id+状态)、我的帖子/回复(user_id+状态)是高频查询。
	//   Thread/Reply 只有单列外键索引，status 过滤会扫整块；补状态复合索引直接命中。
	//   GORM 默认表名 thread→threads、reply→replies；表名若逢差异只会少建、不会报错（helper 仅 log）。
	ensureEzfyIndex(db, "threads", "idx_thread_board_status", "board_id,status,audit_status", false)
	ensureEzfyIndex(db, "threads", "idx_thread_user_status", "user_id,status", false)
	ensureEzfyIndex(db, "replies", "idx_reply_thread_status", "thread_id,status", false)
	ensureEzfyIndex(db, "replies", "idx_reply_user_status", "user_id,status", false)

	ezfyBackfillReportCityId(db)
}

// ezfyBackfillReportCityId 战报 city_id 兜底回填（幂等）。
//
// ★ 2026-10-07：city_id 是后加字段。AutoMigrate 给存量表补列时，若字段没写 default，
//   建出来的是可空列 → 6000+ 历史行全是 NULL → GORM 扫进 int64 直接报
//   `converting NULL to int64 is unsupported`，战报/军情接口整体 500。
//   model 已补 `default:0`；这里再兜一次，确保历史行落成 0（口径 =「待解析」，
//   军情按城过滤时走 ezfyCityReportCond 的坐标反查分支）。
//   同款样板见 seed.go 的 `UPDATE ezfy_profile SET current_city_id = 0 WHERE ... IS NULL`。
func ezfyBackfillReportCityId(db *gorm.DB) {
	if err := db.Exec("UPDATE ezfy_report SET city_id = 0 WHERE city_id IS NULL").Error; err != nil {
		log.Printf("ezfy 战报 city_id 回填失败: %v", err)
	}
}

// ensureEzfyIndex 幂等补建普通索引。GORM AutoMigrate 对存量表只补列/主键，
// 不保证补普通（尤其复合）索引，所以这里显式判存在再建。
// unique=true 建 UNIQUE INDEX，否则普通 KEY。表/索引/列名都是内部常量，安全拼接。
func ensureEzfyIndex(db *gorm.DB, table, index, cols string, unique bool) {
	var n int64
	if err := db.Raw(`SELECT COUNT(*) FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND INDEX_NAME = ?`,
		table, index).Scan(&n).Error; err != nil {
		log.Printf("ezfy 索引检查失败 %s.%s: %v", table, index, err)
		return
	}
	if n > 0 {
		return
	}
	kind := "INDEX"
	if unique {
		kind = "UNIQUE INDEX"
	}
	if err := db.Exec(fmt.Sprintf("ALTER TABLE `%s` ADD %s `%s` (%s)", table, kind, index, cols)).Error; err != nil {
		log.Printf("ezfy 建索引失败 %s.%s(%s): %v", table, index, cols, err)
	}
}

// fixEzfySignIndex 修复 ezfy_sign 建错的唯一索引（2026-09-26 线上「无限签到刷资源」事故）
//
// 旧标签只把 `uniqueIndex:uk_user_date` 挂在 SignDate 上，AutoMigrate 建出来的是
// `UNIQUE KEY uk_user_date (sign_date)` —— 全服每天只放行一条签到记录。
// GORM 的 AutoMigrate 见到同名索引就直接跳过、不会改成复合索引，所以要显式修：
// 按 information_schema 判断该索引有没有包含 user_id，没有就 DROP 再按
// (user_id, sign_date) 重建。幂等：修好后再启动直接 return。
func fixEzfySignIndex(db *gorm.DB) {
	var hasUser int64
	if err := db.Raw(`SELECT COUNT(*) FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'ezfy_sign'
		  AND INDEX_NAME = 'uk_user_date' AND COLUMN_NAME = 'user_id'`).Scan(&hasUser).Error; err != nil {
		log.Printf("ezfy 签到索引检查失败: %v", err)
		return
	}
	if hasUser > 0 {
		return
	}
	if err := db.Exec("ALTER TABLE ezfy_sign DROP INDEX uk_user_date").Error; err != nil {
		log.Printf("ezfy 签到索引删除失败(可能本来就没有): %v", err)
	}
	if err := db.Exec("ALTER TABLE ezfy_sign ADD UNIQUE KEY uk_user_date (user_id, sign_date)").Error; err != nil {
		log.Printf("ezfy 签到索引重建失败: %v", err)
	}
}

// seedEzfyWildOfficers 野地/寇城守将补缺（2026-09-24 ）
//
// 野地军官必须来自「军官池」ezfy_cfg_general（每块野地最多 1 名，配在野地类型的
// officer_id 上）。老库 ezfy_cfg_wildland 的 officer_id 全为 0 —— 玩家反馈
// 「野地军官太少了，好像没见过」。这里做一次性补缺：只要还没有**任何**野地类型
// 配过守将，就按「类型→等级」顺序从军官池依次分配一名守将。
// 任意一行已有军官即视为「管理员已配置过」，不再动。
// ezfyWildOfficerStarOf 按野地等级决定守将星级
//
// ★ 2026-09-29 用户规则：高级野地守将星级高、低级野地守将星级低，用最新军官池里
//   对应星级的**普通军官**（kind=1）。分档（按各自身形互补，覆盖 1~5 星）：
//
//	等级 0-1 → 1 星 · 2-3 → 2 星 · 4-6 → 3 星 · 7-8 → 4 星 · 9-10 → 5 星
func ezfyWildOfficerStarOf(level int) int {
	switch {
	case level <= 1:
		return 1
	case level <= 3:
		return 2
	case level <= 6:
		return 3
	case level <= 8:
		return 4
	default:
		return 5
	}
}

// seedEzfyWildOfficers 一次性按「野地等级→星级」从普通军官池(kind=1)分配守将
//
// 野地军官必须来自「军官池」ezfy_cfg_general（每块野地最多 1 名，配在野地类型的
// officer_id 上）。老库 ezfy_cfg_wildland 的 officer_id 全为 0 —— 玩家反馈
// 「野地军官太少了，好像没见过」。这里做一次性补缺：只要还没有**任何**野地类型
// 配过守将，就按「等级→星级」从普通军官池取对应星级的军官当守将。
// 任意一行已有军官即视为「管理员已配置过」，不再动。
func seedEzfyWildOfficers(db *gorm.DB) {
	var had int64
	db.Model(&model.EzfyCfgWildland{}).Where("officer_id > 0").Count(&had)
	if had > 0 {
		return
	}
	// 只从普通军官池(kind=1)挑守将，按星级分桶（5星3% / 4星7% / 3星20% / 2星30% / 1星40%）
	stars := map[int][]model.EzfyCfgGeneral{}
	var gens []model.EzfyCfgGeneral
	db.Where("kind = 1").Order("id").Find(&gens)
	for _, g := range gens {
		s := g.Star
		if s < 1 {
			s = 1
		}
		if s > 5 {
			s = 5
		}
		stars[s] = append(stars[s], g)
	}
	if len(gens) == 0 {
		return
	}
	// 每个星级的轮转游标，让同一星级里也能换着用不同军官
	cursor := map[int]int{}
	var rows []model.EzfyCfgWildland
	db.Where("officer_min > 0").Order("type, level, id").Find(&rows)
	for i, r := range rows {
		dst := ezfyWildOfficerStarOf(r.Level)
		pool := stars[dst]
		if len(pool) == 0 {
			// 该星级没有普通军官 → 兜底用任意普通军官
			pool = gens
		}
		idx := cursor[dst] % len(pool)
		cursor[dst] = idx + 1
		g := pool[idx]
		if err := db.Model(&model.EzfyCfgWildland{}).Where("id = ? AND officer_id = 0", r.ID).
			Update("officer_id", g.ID).Error; err != nil {
			log.Printf("野地守将补缺失败 id=%d: %v", r.ID, err)
		}
		_ = i
	}
}

// ezfyExchangeTpls 交易行挂单模板（2026-09-24 管理端「维护模版」tab 维护）
var ezfyExchangeTpls = []model.EzfyExchangeTemplate{
	{ID: 1, Name: "粮食 · 小包", EsType: 1, EsCount: 10000, TotalPrice: 100, Currency: 1, SortNo: 1},
	{ID: 2, Name: "粮食 · 中包", EsType: 1, EsCount: 100000, TotalPrice: 900, Currency: 1, SortNo: 2},
	{ID: 3, Name: "粮食 · 大包", EsType: 1, EsCount: 1000000, TotalPrice: 8000, Currency: 1, SortNo: 3},
	{ID: 4, Name: "钢铁 · 小包", EsType: 2, EsCount: 10000, TotalPrice: 100, Currency: 1, SortNo: 4},
	{ID: 5, Name: "钢铁 · 中包", EsType: 2, EsCount: 100000, TotalPrice: 900, Currency: 1, SortNo: 5},
	{ID: 6, Name: "钢铁 · 大包", EsType: 2, EsCount: 1000000, TotalPrice: 8000, Currency: 1, SortNo: 6},
	{ID: 7, Name: "石油 · 小包", EsType: 3, EsCount: 10000, TotalPrice: 100, Currency: 1, SortNo: 7},
	{ID: 8, Name: "石油 · 中包", EsType: 3, EsCount: 100000, TotalPrice: 900, Currency: 1, SortNo: 8},
	{ID: 9, Name: "石油 · 大包", EsType: 3, EsCount: 1000000, TotalPrice: 8000, Currency: 1, SortNo: 9},
	{ID: 10, Name: "稀矿 · 小包", EsType: 4, EsCount: 10000, TotalPrice: 100, Currency: 1, SortNo: 10},
	{ID: 11, Name: "稀矿 · 中包", EsType: 4, EsCount: 100000, TotalPrice: 900, Currency: 1, SortNo: 11},
	{ID: 12, Name: "稀矿 · 大包", EsType: 4, EsCount: 1000000, TotalPrice: 8000, Currency: 1, SortNo: 12},
}

// seedEzfyExchangeTpls 交易行挂单模板种子（只补缺不覆盖，管理端增删改能活过重启）
func seedEzfyExchangeTpls(db *gorm.DB) {
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(ezfyExchangeTpls, 50).Error; err != nil {
		log.Printf("ezfy 交易行模板种子失败: %v", err)
	}
}

// seedEzfyRanks 军衔配置（复刻原版 rankIndex.html 的「军衔等级/职位要求/可建城数」）
//
// 原版：1:列兵/士兵/1 ... 20:五星上将/司令/20 —— 可建城数 = 军衔等级。
// ★ 只补缺不覆盖，管理端调过的值不会被启动打回。
// ★ 2026-09-24 「军衔需要声望太少，统一在原来基础上 ×10」：
//
//	内置默认门槛已 ×10；老库已存在的行做一次 ×10 迁移
//	（以「全表最大门槛 ≤ 40000（旧内置上限）」为标记，乘过后最大 400000 不再触发）。
func seedEzfyRanks(db *gorm.DB) {
	rows := []model.EzfyCfgRank{
		{ID: 1, Name: "列兵", Post: "士兵", NeedPrestige: 0, CityMax: 1},
		{ID: 2, Name: "上等兵", Post: "班长", NeedPrestige: 1000, CityMax: 2},
		{ID: 3, Name: "下士", Post: "排长", NeedPrestige: 3000, CityMax: 3},
		{ID: 4, Name: "中士", Post: "排长", NeedPrestige: 6000, CityMax: 4},
		{ID: 5, Name: "上士", Post: "连长", NeedPrestige: 10000, CityMax: 5},
		// ★ 2026-09-27 军衔门槛按**线上现版标准**对齐（1-5 级不变，6-20 级取线上 ezfy_cfg_rank 现值）
		{ID: 6, Name: "军士长", Post: "连长", NeedPrestige: 30000, CityMax: 6},
		{ID: 7, Name: "准尉", Post: "营长", NeedPrestige: 50000, CityMax: 7},
		{ID: 8, Name: "少尉", Post: "营长", NeedPrestige: 80000, CityMax: 8},
		{ID: 9, Name: "中尉", Post: "营长", NeedPrestige: 120000, CityMax: 9},
		{ID: 10, Name: "上尉", Post: "团长", NeedPrestige: 200000, CityMax: 10},
		{ID: 11, Name: "大尉", Post: "团长", NeedPrestige: 300000, CityMax: 11},
		{ID: 12, Name: "少校", Post: "旅长", NeedPrestige: 500000, CityMax: 12},
		{ID: 13, Name: "中校", Post: "旅长", NeedPrestige: 1000000, CityMax: 13},
		{ID: 14, Name: "上校", Post: "旅长", NeedPrestige: 2000000, CityMax: 14},
		{ID: 15, Name: "大校", Post: "师长", NeedPrestige: 4000000, CityMax: 15},
		{ID: 16, Name: "少将", Post: "师长", NeedPrestige: 8000000, CityMax: 16},
		{ID: 17, Name: "中将", Post: "军长", NeedPrestige: 16000000, CityMax: 17},
		{ID: 18, Name: "上将", Post: "军长", NeedPrestige: 32000000, CityMax: 18},
		{ID: 19, Name: "大将", Post: "军长", NeedPrestige: 64000000, CityMax: 19},
		{ID: 20, Name: "五星上将", Post: "司令", NeedPrestige: 100000000, CityMax: 20},
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(rows, 50).Error; err != nil {
		log.Printf("ezfy 军衔种子失败: %v", err)
	}
	// ★ 一次性 ×10 迁移：老库旧规模（最大门槛 ≤ 40000）→ 全表 ×10。
	//   乘过的库最大门槛是 400000 > 40000，后续启动不会再触发。
	var maxP int
	db.Model(&model.EzfyCfgRank{}).Select("COALESCE(MAX(need_prestige), 0)").Scan(&maxP)
	if maxP > 0 && maxP <= 40000 {
		if err := db.Exec("UPDATE ezfy_cfg_rank SET need_prestige = need_prestige * 10").Error; err != nil {
			log.Printf("ezfy 军衔声望 ×10 迁移失败: %v", err)
		} else {
			log.Printf("ezfy 军衔声望门槛已统一 ×10（原最大门槛 %d）", maxP)
		}
	}
}

// seedEzfyResources 资源显示名配置（管理端可改名，全站跟随）
//
// ★ 「只补缺不覆盖」：管理端改过的名字不能被启动时打回。
func seedEzfyResources(db *gorm.DB) {
	rows := []model.EzfyCfgResource{
		{ID: 1, Key: "gold", Name: "黄金", Short: "金", Sort: 1},
		{ID: 2, Key: "food", Name: "粮食", Short: "粮", Sort: 2},
		{ID: 3, Key: "steel", Name: "钢铁", Short: "钢", Sort: 3},
		{ID: 4, Key: "oil", Name: "石油", Short: "油", Sort: 4},
		{ID: 5, Key: "rare", Name: "稀矿", Short: "稀", Sort: 5},
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(rows, 50).Error; err != nil {
		log.Printf("ezfy 资源名种子失败: %v", err)
	}
}

// seedEzfyActivities 节日活动模板（E1，设计依据 `参考材料/开发文档/福利.txt`）
//
// 原版 Java 只做了静态说明页，这里做成可配置、真实生效的活动。
// 默认全部「未开启」(status=0)，避免上线即改数值；管理端「数据管理 → 节日活动」
// 把 status 改成 1 并填好起止时间即可生效。时间戳单位毫秒。
//
//	type: 1资源增产 2造兵打折 3建造加速 4研究加速 5声望加成
func seedEzfyActivities(db *gorm.DB) {
	var count int64
	if err := db.Table("ezfy_activity").Count(&count).Error; err != nil || count > 0 {
		return
	}
	rows := []model.EzfyActivity{
		{ID: 1, Name: "丰收节", Type: 1, Param: 20, Status: 0,
			Des: "活动期间全城资源产量 +20%，野地产出同样生效"},
		{ID: 2, Name: "军工动员", Type: 2, Param: 25, Status: 0,
			Des: "活动期间训练部队的资源消耗 -25%"},
		{ID: 3, Name: "建设狂潮", Type: 3, Param: 30, Status: 0,
			Des: "活动期间建造/升级建筑耗时 -30%"},
		{ID: 4, Name: "科技峰会", Type: 4, Param: 20, Status: 0,
			Des: "活动期间科技研究耗时 -20%"},
		{ID: 5, Name: "荣耀之战", Type: 5, Param: 10, Status: 0,
			Des: "活动期间战斗获得的军功声望 +10%"},
	}
	if err := db.Create(&rows).Error; err != nil {
		log.Printf("ezfy 节日活动种子失败: %v", err)
	}
}

// seedEzfyOfficerItems 军官类道具（复刻设计文档《QQ家园二战风云.txt》道具 #7/#8/#9）
//
// 原工程这批道具标为「未实现」，这里补齐；按 ID 幂等 upsert，老库也能补上。
//
//	13 招生简章   ItemType 9  立即刷新军校候选名将(不占每日 5 次)
//	14 荣誉史记   ItemType 10 指定军官获得经验(每本 27,068,000 经验)
//	15 军官技能书 ItemType 11 指定军官消耗技能书学习 1 个技能
//	16 军官洗点卡 ItemType 12 重置军官属性为军官池初始属性 + 退回待分配点(等级/经验/技能保留)
//	17 改名卡     ItemType 13 统帅页改昵称(首次免费, 之后每次消耗 1 张)
//	18 阵营转换道具 ItemType 14 统帅页改阵营(首次免费, 之后每次消耗 1 个)
//	19 集结令     ItemType 15 出征时提高本次出征兵力上限(每个 +10 万，单次上限由管理端配置)
//	23 星级徽章   ItemType 19 对军官使用, 20% 概率升 1 星(最高五星, 失败不降星级属性)
//	24 信号弹     ItemType 20 计谋消耗品(黄金/钻石双渠道，发动计谋时消耗)
//	25 军官改名卡 ItemType 21 在军官管理页面改名玩家自己的军官(消耗 1 张, 不影响军官池)
func seedEzfyOfficerItems(db *gorm.DB) {
	// ★★ 2026-09-21 用户反馈「道具商城上架军官技能书」的根因：
	//   这几条种子**没有显式写 Stock**，而 GORM 创建时会把 Go 的零值 `0` 一起写进去
	//   （结构体字段的 `gorm:"default:100"` 标签只在「完全省略该列」时才生效），
	//   结果 14 经验书 / 15 军官技能书 / 16 重修书 / 18 阵营转换道具 全部库存 = 0，
	//   玩家买的时候被 `Buy` 里的库存校验拦成「已售罄」—— 看着就是「没上架」。
	//   军官类道具定位是**常驻消耗品**（跟迁城道具一样），统一给 -1 = 无限库存。
	// ★ 2026-09-26 「初始化数据按线上现值对齐」：下面价格/库存一律取线上库快照值。
	// ★ 2026-09-27 按线上现值更新种子：招生简章 20→2 钻石、改名卡 250000 黄金→10 钻石、
	//   阵营转换道具 500→10 钻石、集结令 20→1 钻石、信号弹 20→2 钻石；
	//   库存：军官洗点卡 96→-1、信号弹 100→0、军官改名卡 79→-1。
	rows := []model.EzfyCfgItem{
		{ID: 13, Name: "招生简章", ItemType: 9, Param1: 1, PriceGold: 0, PriceDiamond: 2, Stock: -1,
			Description: "立即刷新军校候选名将, 不占用每日刷新次数"},
		{ID: 14, Name: "荣誉史记", ItemType: 10, Param1: 20000000, PriceGold: 0, PriceDiamond: 80, Stock: -1,
			Category:    "军官道具",
			Description: "在军官管理页面使用, 每本增加 20000000 经验"},
		{ID: 15, Name: "军官技能书", ItemType: 11, Param1: 1, PriceGold: 0, PriceDiamond: 100, Stock: -1,
			Category:    "军官道具",
			Description: "在军官技能管理页面使用, 消耗技能书学习技能"},
		// ★ 2026-09-26 用户明确：洗点**只动属性**，技能/等级/经验都保留
		{ID: 16, Name: "军官洗点卡", ItemType: 12, Param1: 0, PriceGold: 80000, PriceDiamond: 0, Stock: -1,
			Category:    "军官道具",
			Description: "洗点: 军官属性重置为军官池初始属性, 已分配的点退回待分配点(等级/经验/技能保留)"},
		{ID: 17, Name: "改名卡", ItemType: 13, Param1: 1, PriceGold: 0, PriceDiamond: 10, Stock: -1,
			Description: "在统帅页修改玩家昵称(首次改名免费, 之后每次消耗1张)"},
		{ID: 18, Name: "阵营转换道具", ItemType: 14, Param1: 1, PriceGold: 0, PriceDiamond: 10, Stock: -1,
			Description: "在统帅页转换阵营(首次转换免费, 之后每次消耗1个)"},
		// ★ 用户规则：集结令走**钻石**渠道，先默认 0 钻石（等于免费发放，方便先放开玩）；
		//   库存 -1 = 无限，玩家可任意购买（见 Buy 里的 stock < 0 分支）。
		//   Param1 = 每个集结令提升的出征上限（10 万）。
		// ★ 说明里**不要**再写「单次最多使用10个」——
		//   单次上限由管理端 `ezfy_cfg_limit.gather_max_per_order` 维护（线上现值 99），
		//   写死 10 会和管理端配置对不上，玩家会以为只能买 10 个。
		{ID: 19, Name: "集结令", ItemType: 15, Param1: 100000,
			PriceGold: 0, PriceDiamond: 1, Stock: -1, Category: "钻石道具",
			Description: "出征时使用: 每使用1个本次出征兵力上限+10万"},
		// ★ 2026-09-23 「军官升星卡」改名「星级徽章」，固定 20% 概率升 1 星、最高 5 星，
		//   失败消耗徽章、不降星级与属性（星级上限/失败保留开关仍走管理端「系统配置」页）。
		{ID: 23, Name: "星级徽章", ItemType: 19, Param1: 1, PriceGold: 0, PriceDiamond: 50, Stock: 100,
			Category:    "军官道具",
			Description: "对军官使用, 每枚有20%概率升1星, 最高五星; 失败消耗徽章, 不降低星级和属性"},
		// ★ 2026-09-23 「玩家自己的军官也能改名」：消耗「军官改名卡」，
		//   在军官管理页面使用，成功改名消耗 1 张，不改动军官池里的原军官。
		{ID: 25, Name: "军官改名卡", ItemType: 21, Param1: 1, PriceGold: 10000, PriceDiamond: 0, Stock: -1,
			Category:    "军官道具",
			Description: "在军官管理页面使用, 成功改名消耗1张, 不影响军官池的原军官"},
		// ★ 2026-09-22 「信号弹也是道具，可以黄金、钻石购买，加上，用于计谋消耗」。
		//   ★ Category 必须显式写「计谋道具」：ezfyItemCategory 里「PriceDiamond>0 → 钻石道具」
		//   那一步在 ItemType 判断**之前**，不写的话它会被归到「钻石道具」里。
		//   Category 不是「黄金道具/钻石道具」→ 不锁货币 → 前端两种价格都列出来让玩家选。
		{ID: 24, Name: "信号弹", ItemType: 20, Param1: 1, PriceGold: 0, PriceDiamond: 2, Stock: 0,
			Category:    "计谋道具",
			Description: "计谋消耗品: 发动计谋时消耗, 每条计谋需要的数量不同"},
	}
	for _, it := range rows {
		var count int64
		db.Model(&model.EzfyCfgItem{}).Where("id = ?", it.ID).Count(&count)
		if count > 0 {
			// 已存在则只同步名称/说明/分类, 不动价格与库存(避免覆盖后台调价)
			db.Model(&model.EzfyCfgItem{}).Where("id = ?", it.ID).
				Updates(map[string]interface{}{"name": it.Name, "item_type": it.ItemType,
					"param1": it.Param1, "description": it.Description, "category": it.Category})
			// ★ 库存修复（只补 bug、不覆盖运营）：
			//   老库里这几条军官道具因为种子漏写 Stock 而落成 0（= 售罄，玩家买不了）。
			//   仅当库存**恰为 0** 时才补成 -1（无限）；-1 与 >0 一律不动，
			//   所以管理员手工设过的库存不会被冲掉。
			if it.Stock == -1 {
				db.Model(&model.EzfyCfgItem{}).
					Where("id = ? AND stock = 0", it.ID).
					Update("stock", -1)
			}
			continue
		}
		db.Create(&it)
	}

	// ★ 2026-09-23 军官道具改为钻石定价（荣誉史记/军官技能书/星级徽章）。
	//   循环里的价格保护「不动价格」是为后台调价留的余地，但这次是明确重新定价，
	//   所以对这 4 个道具额外强制对齐价格。
	// ★ 2026-09-26 按线上现值：军官洗点卡改回**黄金**渠道 80000（原为 2 钻石）→ 值成 {黄金, 钻石}。
	priceFix := map[int][2]int64{
		14: {0, 80},    // 荣誉史记   80 钻石（2026-09-30 按线上现值）
		15: {0, 100},   // 军官技能书 100 钻石
		16: {80000, 0}, // 军官洗点卡 80000 黄金
		23: {0, 50},    // 星级徽章   50 钻石
	}
	for id, p := range priceFix {
		db.Model(&model.EzfyCfgItem{}).Where("id = ?", id).
			Updates(map[string]interface{}{"price_gold": p[0], "price_diamond": p[1]})
	}
}

// seedEzfyMoveItems 迁城类道具（第十二轮新增）
//
// 用户规则：「迁城计划 是道具 可以用黄金 和 钻石 购买 单独的 但是功能是一样的」
//
//	         「用 迁城计划、高级迁城计划、沿海迁城计划 …… 可以灵活批量迁移城池」
//
//		20 迁城计划     ItemType 16 选洲迁城（落该洲随机空平原）
//		21 高级迁城计划 ItemType 17 指定坐标迁城（平原）
//		22 沿海迁城计划 ItemType 18 选洲 / 指定坐标迁城（沿海平原，海城专用）
//
// ★ 价格（钻石单渠道，管理端随时可改；2026-09-27 按线上现值对齐）：
//
//	迁城计划       10 钻石
//	高级迁城计划   30 钻石
//	沿海迁城计划   30 钻石
//
// 库存按线上现值：迁城计划 / 高级迁城计划 100，沿海迁城计划 99（-1 = 无限）。
func seedEzfyMoveItems(db *gorm.DB) {
	rows := []model.EzfyCfgItem{
		{ID: 20, Name: "迁城计划", ItemType: 16, Param1: 1,
			PriceGold: 0, PriceDiamond: 10, Stock: 100, Category: "迁城道具",
			Description: "在市政厅→城市迁移使用: 选择一个大洲, 城市随机迁移到该洲内未被占领的平原"},
		{ID: 21, Name: "高级迁城计划", ItemType: 17, Param1: 1,
			PriceGold: 0, PriceDiamond: 30, Stock: 100, Category: "迁城道具",
			Description: "在市政厅→城市迁移使用: 指定坐标迁移城市, 目标必须是未被占领的平原"},
		{ID: 22, Name: "沿海迁城计划", ItemType: 18, Param1: 1,
			PriceGold: 0, PriceDiamond: 30, Stock: 99, Category: "迁城道具",
			Description: "在市政厅→城市迁移使用: 选择大洲或指定坐标, 城市迁移到沿海平原(海城专用)"},
	}
	for _, it := range rows {
		var count int64
		db.Model(&model.EzfyCfgItem{}).Where("id = ?", it.ID).Count(&count)
		if count > 0 {
			// 已存在则只同步名称/类型/说明/分类, 不动价格与库存(避免覆盖后台调价)
			db.Model(&model.EzfyCfgItem{}).Where("id = ?", it.ID).
				Updates(map[string]interface{}{"name": it.Name, "item_type": it.ItemType,
					"param1": it.Param1, "description": it.Description, "category": it.Category})
			continue
		}
		db.Create(&it)
	}
}

// seedEzfyLoveCardItems 清理「为爱发电卡」在道具配置表中的残留
//
// ★ 2026-10-10 用户需求：为爱发电卡不是道具, 有专门模块维护, 要从道具体系彻底移除。
//   此前曾把两张卡(cfg 26/27)写入 ezfy_cfg_item, 现改为跨版本删除这两行,
//   后续发卡/下拉/查看均由专属模块(loveCardCatalog + ezfy_love_card)承担。
//   卡片规则：为爱发电卡每日150钻石 / 为爱发电高级卡每日200钻石, 各30天。
func seedEzfyLoveCardItems(db *gorm.DB) {
	var removed int
	res := db.Where("id IN ?", []int{26, 27}).Delete(&model.EzfyCfgItem{})
	if res != nil && res.RowsAffected > 0 {
		removed = int(res.RowsAffected)
	}
	if removed > 0 {
		log.Printf("ezfy 为爱发电卡已从道具配置表移除 %d 行（改由专属模块维护）", removed)
	}
}

// seedEzfyNotices 游戏内置公告（幂等：标题存在即跳过）
//
// ★ 2026-09-26 「初始化数据按线上现值对齐」：
//   - 新增线上第一条「《二战征途》开服公告」（置顶）；
//   - 「新手提示」文案改为线上版（先用**黄金**召集人口）。
func seedEzfyNotices(db *gorm.DB) {
	rows := []model.EzfyNotice{
		{UserId: 0, IsTop: 1, Title: "《二战征途》开服公告",
			Content: "各位司令官，欢迎来到《二战征途》！建造城池、发展资源、训练部队，出征野地掠夺资源。攻占寇城可以获得丰厚战利品。掠夺/征服其他玩家城池需先宣战，宣战24小时后生效。祝各位武运昌隆！\n\n目前处于测试阶段，好的玩法、建议送资源包！\n\n官方QQ群：431442049\n\n为爱发电中，钻石用于共筹服务器运行以及代码开发。"},
		{UserId: 0, IsTop: 0, Title: "新手提示",
			Content: "进入游戏自动获得主城(市政厅/民居/农田各1级)。先用黄金召集人口，再建资源建筑。造兵需要军工厂，研究科技需要科研中心。市政厅等级决定可占领野地数量上限。"},
	}
	for _, n := range rows {
		var count int64
		db.Model(&model.EzfyNotice{}).Where("title = ? AND user_id = 0", n.Title).Count(&count)
		if count == 0 {
			db.Create(&n)
		}
	}
	// 老库里的「新手提示」还是旧文案（粮食召集人口）→ 只在「原文就是旧默认」时才改，
	// 管理端改过的文案不会被冲掉。
	db.Model(&model.EzfyNotice{}).
		Where("title = ? AND user_id = 0 AND content LIKE ?", "新手提示", "%先用粮食召集人口%").
		Update("content", rows[1].Content)
}
