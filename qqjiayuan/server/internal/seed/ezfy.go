package seed

import (
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
	// ★ 2026-09-26 用户要求「道具配置按现在线上跑的初始化」：
	//   stock 列自带 DB 默认值 100，而 GORM 对「带 default 标签的字段」会跳过 Go 零值，
	//   于是线上「0 = 已售罄」的道具（1~10、12）在库里会落成 100（非 0 库存不受影响，batch 正常写入）。
	//   这里只对「快照库存为 0」的条目显式写一次，让售罄状态也能原样初始化。
	for i := range ezfyEzfyCfgItem {
		if ezfyEzfyCfgItem[i].Stock != 0 {
			continue
		}
		db.Model(&model.EzfyCfgItem{}).Where("id = ?", ezfyEzfyCfgItem[i].ID).
			Update("stock", 0)
	}
	// ★ 任务类型/任务：改「只补缺不覆盖」—— 管理端在「数据管理」里调的奖励(数值)
	//   不能被下次启动的种子悄悄改回去（用户要求「后台能灵活配置奖励」）。
	batchKeep(ezfyEzfyCfgTaskType, "ezfy_cfg_task_type")
	batchKeep(ezfyEzfyCfgTask, "ezfy_cfg_task")

	seedEzfyNotices(db)
	seedEzfyOfficerItems(db)
	seedEzfyMoveItems(db)
	seedEzfyActivities(db)
	seedEzfyResources(db)
	seedEzfyRanks(db)

	// —— 军官池（普通军官 1000 名）/ 装备套装 / 存量军官属性点迁移 ——
	// ★ 必须在 batchKeep(ezfy_cfg_general) 之后：名将已入库，再补普通军官不会互相覆盖。
	normalizeEzfyNewCols(db)
	seedEzfyOfficerPool(db)
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
}

// seedEzfyWildOfficers 野地/寇城守将补缺（2026-09-24 用户要求）
//
// 野地军官必须来自「军官池」ezfy_cfg_general（每块野地最多 1 名，配在野地类型的
// officer_id 上）。老库 ezfy_cfg_wildland 的 officer_id 全为 0 —— 玩家反馈
// 「野地军官太少了，好像没见过」。这里做一次性补缺：只要还没有**任何**野地类型
// 配过守将，就按「类型→等级」顺序从军官池依次分配一名守将。
// 任意一行已有军官即视为「管理员已配置过」，不再动。
func seedEzfyWildOfficers(db *gorm.DB) {
	var had int64
	db.Model(&model.EzfyCfgWildland{}).Where("officer_id > 0").Count(&had)
	if had > 0 {
		return
	}
	// 优先用普通军官池(kind=1)；池子被删空则回退到全部名将
	var gens []model.EzfyCfgGeneral
	db.Where("kind = 1").Order("id").Find(&gens)
	if len(gens) == 0 {
		db.Order("id").Find(&gens)
	}
	if len(gens) == 0 {
		return
	}
	var rows []model.EzfyCfgWildland
	db.Where("officer_min > 0").Order("type, level, id").Find(&rows)
	for i, r := range rows {
		g := gens[i%len(gens)]
		if err := db.Model(&model.EzfyCfgWildland{}).Where("id = ? AND officer_id = 0", r.ID).
			Update("officer_id", g.ID).Error; err != nil {
			log.Printf("野地守将补缺失败 id=%d: %v", r.ID, err)
		}
	}
}

// ezfyExchangeTpls 交易行挂单模板（2026-09-24 用户要求：管理端「维护模版」tab 维护）
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
// ★ 2026-09-24 用户要求「军衔需要声望太少，统一在原来基础上 ×10」：
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
		{ID: 6, Name: "军士长", Post: "连长", NeedPrestige: 15000, CityMax: 6},
		{ID: 7, Name: "准尉", Post: "营长", NeedPrestige: 22000, CityMax: 7},
		{ID: 8, Name: "少尉", Post: "营长", NeedPrestige: 36000, CityMax: 8},
		{ID: 9, Name: "中尉", Post: "营长", NeedPrestige: 45000, CityMax: 9},
		{ID: 10, Name: "上尉", Post: "团长", NeedPrestige: 56000, CityMax: 10},
		{ID: 11, Name: "大尉", Post: "团长", NeedPrestige: 76000, CityMax: 11},
		{ID: 12, Name: "少校", Post: "旅长", NeedPrestige: 92000, CityMax: 12},
		{ID: 13, Name: "中校", Post: "旅长", NeedPrestige: 100000, CityMax: 13},
		{ID: 14, Name: "上校", Post: "旅长", NeedPrestige: 120000, CityMax: 14},
		{ID: 15, Name: "大校", Post: "师长", NeedPrestige: 145000, CityMax: 15},
		{ID: 16, Name: "少将", Post: "师长", NeedPrestige: 175000, CityMax: 16},
		{ID: 17, Name: "中将", Post: "军长", NeedPrestige: 210000, CityMax: 17},
		{ID: 18, Name: "上将", Post: "军长", NeedPrestige: 250000, CityMax: 18},
		{ID: 19, Name: "大将", Post: "军长", NeedPrestige: 300000, CityMax: 19},
		{ID: 20, Name: "五星上将", Post: "司令", NeedPrestige: 4000000, CityMax: 20},
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
	// ★ 2026-09-26 用户要求「初始化数据按线上现值对齐」：下面价格/库存一律取线上库快照值。
	rows := []model.EzfyCfgItem{
		{ID: 13, Name: "招生简章", ItemType: 9, Param1: 1, PriceGold: 0, PriceDiamond: 20, Stock: -1,
			Description: "立即刷新军校候选名将, 不占用每日刷新次数"},
		{ID: 14, Name: "荣誉史记", ItemType: 10, Param1: 27068000, PriceGold: 0, PriceDiamond: 100, Stock: -1,
			Category:    "军官道具",
			Description: "在军官管理页面使用, 每本增加 27,068,000 经验"},
		{ID: 15, Name: "军官技能书", ItemType: 11, Param1: 1, PriceGold: 0, PriceDiamond: 100, Stock: -1,
			Category:    "军官道具",
			Description: "在军官技能管理页面使用, 消耗技能书学习技能"},
		// ★ 2026-09-26 用户明确：洗点**只动属性**，技能/等级/经验都保留
		{ID: 16, Name: "军官洗点卡", ItemType: 12, Param1: 0, PriceGold: 0, PriceDiamond: 2, Stock: 97,
			Category:    "军官道具",
			Description: "洗点: 军官属性重置为军官池初始属性, 已分配的点退回待分配点(等级/经验/技能保留)"},
		{ID: 17, Name: "改名卡", ItemType: 13, Param1: 1, PriceGold: 500, Stock: -1,
			Description: "在统帅页修改玩家昵称(首次改名免费, 之后每次消耗1张)"},
		{ID: 18, Name: "阵营转换道具", ItemType: 14, Param1: 1, PriceGold: 0, PriceDiamond: 500, Stock: -1,
			Description: "在统帅页转换阵营(首次转换免费, 之后每次消耗1个)"},
		// ★ 用户规则：集结令走**钻石**渠道，先默认 0 钻石（等于免费发放，方便先放开玩）；
		//   库存 -1 = 无限，玩家可任意购买（见 Buy 里的 stock < 0 分支）。
		//   Param1 = 每个集结令提升的出征上限（10 万）。
		// ★ 用户要求：说明里**不要**再写「单次最多使用10个」——
		//   单次上限由管理端 `ezfy_cfg_limit.gather_max_per_order` 维护（线上现值 99），
		//   写死 10 会和管理端配置对不上，玩家会以为只能买 10 个。
		{ID: 19, Name: "集结令", ItemType: 15, Param1: 100000,
			PriceGold: 0, PriceDiamond: 20, Stock: -1, Category: "钻石道具",
			Description: "出征时使用: 每使用1个本次出征兵力上限+10万"},
		// ★ 2026-09-23 用户要求「军官升星卡」改名「星级徽章」，固定 20% 概率升 1 星、最高 5 星，
		//   失败消耗徽章、不降星级与属性（星级上限/失败保留开关仍走管理端「系统配置」页）。
		{ID: 23, Name: "星级徽章", ItemType: 19, Param1: 1, PriceGold: 0, PriceDiamond: 50, Stock: 100,
			Category:    "军官道具",
			Description: "对军官使用, 每枚有20%概率升1星, 最高五星; 失败消耗徽章, 不降低星级和属性"},
		// ★ 2026-09-23 用户要求「玩家自己的军官也能改名」：消耗「军官改名卡」，
		//   在军官管理页面使用，成功改名消耗 1 张，不改动军官池里的原军官。
		{ID: 25, Name: "军官改名卡", ItemType: 21, Param1: 1, PriceGold: 10000, PriceDiamond: 0, Stock: 100,
			Category:    "军官道具",
			Description: "在军官管理页面使用, 成功改名消耗1张, 不影响军官池的原军官"},
		// ★ 2026-09-22 用户要求「信号弹也是道具，可以黄金、钻石购买，加上，用于计谋消耗」。
		//   ★ Category 必须显式写「计谋道具」：ezfyItemCategory 里「PriceDiamond>0 → 钻石道具」
		//   那一步在 ItemType 判断**之前**，不写的话它会被归到「钻石道具」里。
		//   Category 不是「黄金道具/钻石道具」→ 不锁货币 → 前端两种价格都列出来让玩家选。
		{ID: 24, Name: "信号弹", ItemType: 20, Param1: 1, PriceGold: 0, PriceDiamond: 20, Stock: 100,
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

	// ★ 2026-09-23 用户要求：军官道具改为钻石定价（荣誉史记/军官技能书/军官洗点卡/星级徽章）。
	//   循环里的价格保护「不动价格」是为后台调价留的余地，但这次是明确重新定价，
	//   所以对这 4 个道具额外强制对齐价格（黄金清零 + 钻石价）。
	priceFix := map[int]int64{
		14: 100, // 荣誉史记   100 钻石
		15: 100, // 军官技能书 100 钻石
		16: 2,   // 军官洗点卡 2 钻石（线上现值）
		23: 50,  // 星级徽章   50 钻石
	}
	for id, diamond := range priceFix {
		db.Model(&model.EzfyCfgItem{}).Where("id = ?", id).
			Updates(map[string]interface{}{"price_gold": 0, "price_diamond": diamond})
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
// ★ 价格（钻石单渠道，管理端随时可改；2026-09-26 按线上现值对齐）：
//
//	迁城计划       200 钻石
//	高级迁城计划   400 钻石
//	沿海迁城计划   250 钻石
//
// 库存按线上现值：迁城计划 / 高级 / 沿海 均为 100（-1 = 无限）。
func seedEzfyMoveItems(db *gorm.DB) {
	rows := []model.EzfyCfgItem{
		{ID: 20, Name: "迁城计划", ItemType: 16, Param1: 1,
			PriceGold: 0, PriceDiamond: 200, Stock: 100, Category: "迁城道具",
			Description: "在市政厅→城市迁移使用: 选择一个大洲, 城市随机迁移到该洲内未被占领的平原"},
		{ID: 21, Name: "高级迁城计划", ItemType: 17, Param1: 1,
			PriceGold: 0, PriceDiamond: 400, Stock: 100, Category: "迁城道具",
			Description: "在市政厅→城市迁移使用: 指定坐标迁移城市, 目标必须是未被占领的平原"},
		{ID: 22, Name: "沿海迁城计划", ItemType: 18, Param1: 1,
			PriceGold: 0, PriceDiamond: 250, Stock: 100, Category: "迁城道具",
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

// seedEzfyNotices 游戏内置公告（幂等：标题存在即跳过）
//
// ★ 2026-09-26 用户要求「初始化数据按线上现值对齐」：
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
