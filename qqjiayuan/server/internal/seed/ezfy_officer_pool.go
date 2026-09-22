// 军官池（普通军官 1000 名） / 装备套装种子 + 存量军官属性点迁移
//
// ★ 2026-09-22 用户要求：
//  1. 军官分两类，都放在军官池 ezfy_cfg_general 里由管理端维护：
//     kind=1 普通军官（军校招募/刷新**从池子抽**，不再纯随机生成）
//     kind=2 名将（只由管理端发放）
//  2. 每升 1 级给 1 点自由属性点；加点/升级只写玩家自己的军官实例（ezfy_officer）。
//  3. 装备要有「大池子」+ 套装，套装在商城用黄金/钻石买。
package seed

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// ============ 一、军官池：普通军官 ============

// ezfyPoolFirstNames / ezfyPoolLastNames 普通军官姓名库（40×40 = 1600 种组合，取前 1000）
var ezfyPoolFirstNames = []string{
	"Pater", "Jeremy", "David", "Michael", "John", "Robert", "James", "William", "Charles", "Henry",
	"George", "Edward", "Frank", "Albert", "Arthur", "Walter", "Harold", "Ralph", "Roy", "Earl",
	"Bernard", "Clifford", "Norman", "Stanley", "Leonard", "Herbert", "Frederick", "Raymond", "Ernest", "Douglas",
	"Andrew", "Patrick", "Victor", "Oliver", "Samuel", "Theodore", "Vincent", "Howard", "Gordon", "Lester",
}

var ezfyPoolLastNames = []string{
	"Robinson", "Lee", "Smith", "Brown", "Wilson", "Taylor", "Clark", "Hall", "Young", "Wright",
	"King", "Scott", "Green", "Baker", "Adams", "Nelson", "Carter", "Mitchell", "Perez", "Roberts",
	"Turner", "Phillips", "Campbell", "Parker", "Evans", "Edwards", "Collins", "Stewart", "Morris", "Murphy",
	"Cook", "Rogers", "Morgan", "Peterson", "Cooper", "Reed", "Bailey", "Bell", "Kelly", "Ward",
}

// ezfyPoolOfficerCount 军官池里预置的普通军官数量（用户举例「比如有 1000 个军官」）
const ezfyPoolOfficerCount = 1000

// ezfyPoolStarCap 各星级的属性上限（与「一键生成军官」同口径：不超过同星级名将）
var ezfyPoolStarCap = map[int][3]int{
	1: {60, 60, 60},
	2: {90, 90, 90},
	3: {130, 130, 130},
	4: {190, 190, 190},
	5: {260, 260, 260},
}

// ezfyPoolStarDist 星级分布：5星3% 4星7% 3星20% 2星30% 1星40%（与军校抽取概率一致）
func ezfyPoolStarRoll(r *rand.Rand) int {
	v := r.Intn(100)
	switch {
	case v < 3:
		return 5
	case v < 10:
		return 4
	case v < 30:
		return 3
	case v < 60:
		return 2
	default:
		return 1
	}
}

// buildEzfyPoolOfficers 生成 1000 名普通军官（固定随机种子 → 每次生成结果一致，便于对账）
//
// ID 从 1001 开始，避开名将的 1~31，方便以后人工增删时互不干扰。
func buildEzfyPoolOfficers() []model.EzfyCfgGeneral {
	r := rand.New(rand.NewSource(20260922))
	out := make([]model.EzfyCfgGeneral, 0, ezfyPoolOfficerCount)
	id := 1001
	for fi := 0; fi < len(ezfyPoolFirstNames) && len(out) < ezfyPoolOfficerCount; fi++ {
		for li := 0; li < len(ezfyPoolLastNames) && len(out) < ezfyPoolOfficerCount; li++ {
			name := ezfyPoolFirstNames[fi] + "·" + ezfyPoolLastNames[li]
			star := ezfyPoolStarRoll(r)
			cap3 := ezfyPoolStarCap[star]
			// 每名军官有一个随机「倾向」（0军事 1后勤 2学识 3均衡），让属性分布有差异
			focus := r.Intn(4)
			val := func(idx int) int {
				base := cap3[idx]
				if cap3[idx] < 20 {
					base = 20
				}
				// 主属性 55%~100%，其它属性 25%~60%
				if focus == 3 || focus == idx {
					return base/2 + r.Intn(base/2+1)
				}
				return base/4 + r.Intn(base/3+1)
			}
			mil, log, lea := val(0), val(1), val(2)
			if mil < 5 {
				mil = 5
			}
			if log < 5 {
				log = 5
			}
			if lea < 5 {
				lea = 5
			}
			out = append(out, model.EzfyCfgGeneral{
				ID: id, Name: name, Level: 150,
				Military: mil, Logistics: log, Learning: lea,
				Star: star, Kind: 1, Weight: 100, Recruit: 1,
				Source: "军校招募", Skill: "",
				Des: "普通军官，可在军校招募获得",
			})
			id++
		}
	}
	return out
}

// seedEzfyOfficerPool 军官池普通军官（幂等：池子里已经**一个普通军官都没有**时才灌）
//
// ★ 不能用 batchKeep（按主键 DoNothing）—— 那样管理端删掉的军官会在下次启动又冒出来。
// 用「池子为空」作为闸门：管理端删几个不会被补回，删光了才会重新灌。
func seedEzfyOfficerPool(db *gorm.DB) {
	var cnt int64
	if err := db.Model(&model.EzfyCfgGeneral{}).Where("kind = ?", 1).Count(&cnt).Error; err != nil {
		return
	}
	if cnt > 0 {
		return
	}
	list := buildEzfyPoolOfficers()
	if err := db.CreateInBatches(list, 200).Error; err != nil {
		// 主键冲突（比如管理端手工加过 1001 号）时逐条补，尽量多灌一些
		for i := range list {
			_ = db.Clauses().Create(&list[i]).Error
		}
	}
}

// ============ 二、装备套装 ============

// ezfySetSlots9 9 件套的部位（与原版「头/肩/胸/腰/手/足 + 饰品/挂件/勋章」一致）
var ezfySetSlots9 = []string{"头盔", "护肩", "胸甲", "腰带", "手套", "战靴", "饰品", "挂件", "勋章"}

// ezfyEquipSetSeed 套装模板（第一批：军/后/学 向的成套装备）
//
// 数值参考 `stzb-fk/参考材料/开发文档/部分名将及装备属性与获取方式（最新）.xlsx` 的
// 「装备（部分）」页：整套属性 ÷ 件数 = 每件属性。
type ezfyEquipSetSeed struct {
	ID      int
	Name    string
	Parts   int // 触发套装效果所需件数
	Slots   []string
	Level   int // 穿戴等级需求
	Tier    int // 品质 1~4
	PieceMi int // 每件：军事
	PieceLo int // 每件：后勤
	PieceLe int // 每件：学识
	SetMi   int // 套装加成（触发后）
	SetLo   int
	SetLe   int
	Gold    int64 // 商城黄金价（每件），0=不卖黄金
	Diamond int64 // 商城钻石价（每件），0=不卖钻石
	Effect  string
	Des     string
}

var ezfyEquipSetSeeds = []ezfyEquipSetSeed{
	{ID: 1, Name: "新兵套装(Recruit)", Parts: 9, Slots: ezfySetSlots9, Level: 20, Tier: 1,
		PieceMi: 1, PieceLo: 2, SetMi: 10, SetLo: 14, Gold: 50000,
		Effect: "9件：攻击+33，防御+22", Des: "新号起步套装，用黄金购买"},
	{ID: 2, Name: "战士套装(Warrior)", Parts: 9, Slots: ezfySetSlots9, Level: 40, Tier: 1,
		PieceMi: 3, PieceLo: 4, SetMi: 30, SetLo: 37, Gold: 300000,
		Effect: "9件：攻击+75，防御+50", Des: "完成上士军衔任务可换，也可直接购买"},
	{ID: 3, Name: "海军上将套装(Admiral)", Parts: 9, Slots: ezfySetSlots9, Level: 80, Tier: 2,
		PieceMi: 35, PieceLe: 9, SetMi: 120, SetLe: 40, Gold: 2000000,
		Effect: "9件：海军突击+2，海军伏击+2，反击+2", Des: "海军指挥官专用套装"},
	{ID: 4, Name: "传说英雄套装(Legendary Heroism)", Parts: 9, Slots: ezfySetSlots9, Level: 60, Tier: 2,
		PieceMi: 24, PieceLe: 6, SetMi: 90, SetLe: 24, Gold: 5000000,
		Effect: "9件：攻击+973，防御+824", Des: "刷第五师团、黄金箱子零件兑换"},
	{ID: 5, Name: "传说无畏套装(Legendary Dreadnaught)", Parts: 9, Slots: ezfySetSlots9, Level: 80, Tier: 3,
		PieceMi: 35, PieceLe: 9, SetMi: 130, SetLe: 40, Gold: 8000000,
		Effect: "9件：攻击+646，防御+430", Des: "刷第五师团、黄金箱子零件兑换"},
	{ID: 6, Name: "传说征服套装(Legendary Conquer)", Parts: 9, Slots: ezfySetSlots9, Level: 100, Tier: 3,
		PieceMi: 48, PieceLe: 12, SetMi: 180, SetLe: 55, Gold: 15000000,
		Effect: "9件：攻击+867，防御+578", Des: "刷第五师团、黄金箱子零件兑换"},
	{ID: 7, Name: "名门征服套装(Renowned Conquer)", Parts: 9, Slots: ezfySetSlots9, Level: 100, Tier: 2,
		PieceMi: 27, PieceLe: 8, SetMi: 100, SetLe: 30, Gold: 10000000,
		Effect: "9件：攻击+341，防御+228", Des: "刷野兑换"},
	{ID: 8, Name: "混沌套装一(头/肩/胸)", Parts: 3, Slots: []string{"头盔", "护肩", "胸甲"}, Level: 140, Tier: 4,
		PieceMi: 21, PieceLo: 25, PieceLe: 65, SetMi: 60, SetLo: 70, SetLe: 180, Diamond: 3000,
		Effect: "3件：战术防御+3级", Des: "混沌三件套，钻石购买"},
	{ID: 9, Name: "混沌套装二(腰/手/足)", Parts: 3, Slots: []string{"腰带", "手套", "战靴"}, Level: 140, Tier: 4,
		PieceMi: 21, PieceLo: 25, PieceLe: 65, SetMi: 60, SetLo: 70, SetLe: 180, Diamond: 3000,
		Effect: "3件：英雄突击+3", Des: "混沌三件套，钻石购买"},
	{ID: 10, Name: "混沌套装三(饰品/挂件/勋章)", Parts: 3, Slots: []string{"饰品", "挂件", "勋章"}, Level: 140, Tier: 4,
		PieceMi: 21, PieceLo: 25, PieceLe: 65, SetMi: 60, SetLo: 70, SetLe: 180, Diamond: 3000,
		Effect: "3件：军队生命+30%", Des: "混沌三件套，钻石购买"},
	{ID: 11, Name: "精英守护者套装(Elite Guardian)", Parts: 9, Slots: ezfySetSlots9, Level: 130, Tier: 3,
		PieceMi: 49, PieceLe: 21, SetMi: 190, SetLe: 75, Diamond: 300,
		Effect: "9件：反击+3", Des: "钻石购买"},
	{ID: 12, Name: "传说守护者套装(Legendary Guardian)", Parts: 9, Slots: ezfySetSlots9, Level: 130, Tier: 3,
		PieceMi: 59, PieceLe: 23, SetMi: 230, SetLe: 90, Diamond: 450,
		Effect: "9件：反击+3", Des: "钻石购买"},
	{ID: 13, Name: "暴君之怒套装(King Fury)", Parts: 9, Slots: ezfySetSlots9, Level: 140, Tier: 4,
		PieceMi: 72, PieceLe: 28, SetMi: 280, SetLe: 105, Diamond: 700,
		Effect: "9件：英雄突击+3，反击+3", Des: "军团战奖励 / 钻石购买"},
	{ID: 14, Name: "审判者套装(Judicator)", Parts: 9, Slots: ezfySetSlots9, Level: 140, Tier: 4,
		PieceMi: 78, PieceLe: 28, SetMi: 300, SetLe: 110, Diamond: 800,
		Effect: "9件：三绝+3级", Des: "军团战奖励 / 钻石购买"},
	{ID: 15, Name: "亡魂套装(Revenant)", Parts: 9, Slots: ezfySetSlots9, Level: 140, Tier: 4,
		PieceMi: 78, PieceLe: 34, SetMi: 310, SetLe: 130, Diamond: 1200,
		Effect: "9件：弧形防御+2，反击+3", Des: "刷野活动 / 钻石购买"},
	{ID: 16, Name: "遗失传说套装(The Lost Legend)", Parts: 9, Slots: ezfySetSlots9, Level: 150, Tier: 4,
		PieceMi: 90, PieceLo: 47, PieceLe: 47, SetMi: 360, SetLo: 190, SetLe: 190, Diamond: 3600,
		Effect: "9件：+95%基础军事，攻击+2984，防御+2980", Des: "顶级套装，钻石购买"},
	{ID: 17, Name: "隐秘宝藏套装(The Hidden Treasure)", Parts: 9, Slots: ezfySetSlots9, Level: 150, Tier: 4,
		PieceMi: 99, PieceLo: 51, PieceLe: 51, SetMi: 400, SetLo: 210, SetLe: 210, Diamond: 5000,
		Effect: "9件：+100%基础军事，攻击+2984，防御+2890", Des: "顶级套装，钻石购买"},
}

// ============ 二·B、军官装备系列（11 部位 / 6 项战斗属性 / 11 件套） ============
//
// ★ 2026-09-22 用户指定参考 `装备距离伤害表.xlsx` 的「装备属性」页。
//
// 原版军官装备的规则：
//   - **11 个部位**：肩部 / 头部 / 挂件 / 胸部 / 腰部 / 勋章 / 左手 / 饰品 / 名将勋章 / 足部 / 名将史册
//   - 每件提供 **6 项战斗属性（百分比）**：伤害 / 防御 / 生命 / 移动距离 / 暴击几率 / 暴击伤害
//   - 同一系列凑齐 11 件 = 一套「套装」，套装属性就是 11 件之和（表里的「套装」行就是求和）
//   - 表里数值是「装备+20」时的值，这里直接按该值配（强化等级只做展示，不参与计算）

// ezfyOfficerEquipSlots 11 个部位（顺序与表格一致）
var ezfyOfficerEquipSlots = []string{"肩部", "头部", "挂件", "胸部", "腰部", "勋章", "左手", "饰品", "名将勋章", "足部", "名将史册"}

// ezfyOfficerEquipPiece 一件军官装备
type ezfyOfficerEquipPiece struct {
	Slot    string // 部位（留空则用 ezfyOfficerEquipSlots[i]）
	Sub     string // 表里的具体名字，如「围巾」「折扇」「AWM」
	Dmg     int
	Def     int
	Hp      int
	Move    int
	Crit    int
	CritDmg int
	// ★ 三维属性（军事/后勤/学识）—— 进军官有效属性。
	//   只给六项战斗属性不给三维的话，玩家会看到「穿了一整套但军官的军/后/学一点没变」。
	Mi int // 军事
	Lo int // 后勤
	Le int // 学识
}

// ezfyOfficerSeries 一个系列
type ezfyOfficerSeries struct {
	ID      int
	Series  string // 系列名（用于展示）
	SetName string // 套装名（系列 + 括号里的雅号）
	Level   int    // 穿戴等级需求
	Tier    int
	Diamond int64 // 每件钻石价
	Gold    int64 // 每件黄金价
	// ★ 每件的三维属性（军事/后勤/学识）；11 件套 = 该值 × 11。
	//   量级参考 `部分名将及装备属性与获取方式（最新）.xlsx`：整套 700~900 军 / 300~460 后 / 300~460 学。
	Mi, Lo, Le int
	Pieces     []ezfyOfficerEquipPiece
}

var ezfyOfficerSeriesSeeds = []ezfyOfficerSeries{
	{ID: 21, Series: "革命者", SetName: "革命者[迷雾幽灵]", Level: 120, Tier: 4, Diamond: 1500, Mi: 60, Lo: 30, Le: 30,
		Pieces: []ezfyOfficerEquipPiece{
			{Sub: "围巾", Dmg: 125, Def: 120},
			{Sub: "头盔", Def: 130, Hp: 135},
			{Sub: "折扇", Dmg: 115, Crit: 125, CritDmg: 135},
			{Sub: "卫衣", Hp: 135},
			{Sub: "腰部", Def: 120, Move: 115},
			{Sub: "勋章", Dmg: 130},
			{Sub: "AWM", Dmg: 155, CritDmg: 130},
			{Sub: "指环", Def: 110, Hp: 125, Move: 110},
			{Sub: "八一", Def: 155},
			{Sub: "草鞋", Hp: 125, Move: 135},
			{Sub: "史册", Def: 135},
		}},
	{ID: 22, Series: "渡鸦之魂", SetName: "渡鸦之魂[无尽怒火]", Level: 120, Tier: 4, Diamond: 1500, Mi: 60, Lo: 30, Le: 30,
		Pieces: []ezfyOfficerEquipPiece{
			{Sub: "肩章", Def: 125, Hp: 120},
			{Sub: "帽子", Def: 130},
			{Sub: "挂件", Dmg: 125, CritDmg: 115},
			{Sub: "夹克", Hp: 130},
			{Sub: "工装", Def: 135, Hp: 125},
			{Sub: "勋章", Dmg: 130},
			{Sub: "手枪", Dmg: 120, Crit: 120, CritDmg: 135},
			{Sub: "饰品", Dmg: 115, Crit: 125, CritDmg: 130},
			{Sub: "徽章", Dmg: 115, Move: 120, Crit: 120},
			{Sub: "鞋子", Def: 115, Move: 120},
			{Sub: "名册", Crit: 115, CritDmg: 120},
		}},
	{ID: 23, Series: "黑色幽灵", SetName: "黑色幽灵[其人之道]", Level: 110, Tier: 4, Diamond: 1200, Mi: 50, Lo: 25, Le: 25,
		Pieces: []ezfyOfficerEquipPiece{
			{Sub: "盾牌", Def: 110, Hp: 115},
			{Sub: "帽子", Def: 125},
			{Sub: "沙漏", Dmg: 110, Def: 110},
			{Sub: "钢笔", Hp: 125},
			{Sub: "夹克", Dmg: 125},
			{Sub: "徽章", Dmg: 135, Crit: 125},
			{Sub: "手套", Dmg: 115, Def: 115, Hp: 115, Move: 115, Crit: 115},
			{Sub: "西裤", Def: 125},
			{Sub: "勋章", Move: 110},
			{Sub: "足靴", Def: 110, Move: 120},
			{Sub: "名册", Crit: 125},
		}},
	{ID: 24, Series: "巨匠", SetName: "巨匠[匠人之心]", Level: 110, Tier: 3, Diamond: 800, Mi: 40, Lo: 20, Le: 20,
		Pieces: []ezfyOfficerEquipPiece{
			{Sub: "对讲机", Def: 120},
			{Sub: "头盔", Crit: 125},
			{Sub: "FMJ05A面具", Def: 110, Hp: 110, Move: 110},
			{Sub: "防化服", Def: 130, Hp: 130},
			{Sub: "腰带", CritDmg: 155},
			{Sub: "农业勋章", Dmg: 110, Crit: 110, CritDmg: 110},
			{Sub: "喷雾器", Def: 135},
			{Sub: "扳手", Move: 125},
			{Sub: "名将勋章", Dmg: 135},
			{Sub: "雨靴", Hp: 130},
			{Sub: "史册", CritDmg: 135},
		}},
	{ID: 25, Series: "青天白日", SetName: "青天白日[审判]", Level: 130, Tier: 4, Diamond: 2000, Mi: 70, Lo: 35, Le: 35,
		Pieces: []ezfyOfficerEquipPiece{
			{Sub: "肩部", Dmg: 120},
			{Sub: "头部", Def: 120},
			{Sub: "挂件", Hp: 130},
			{Sub: "胸部", Hp: 115, CritDmg: 130},
			{Sub: "腰部", Crit: 130, CritDmg: 130},
			{Sub: "勋章", Hp: 115, Crit: 125},
			{Sub: "手部", Dmg: 155, Crit: 125, CritDmg: 125},
			{Sub: "饰品", Dmg: 120, Def: 120},
			{Sub: "名将勋章", Def: 135, Hp: 135},
			{Sub: "足部", Def: 115, Hp: 115, Move: 140},
			{Sub: "名将史册", Crit: 115, CritDmg: 130},
		}},
	{ID: 26, Series: "赤色锤镰", SetName: "赤色锤镰[裁决]", Level: 130, Tier: 4, Diamond: 2000, Mi: 70, Lo: 35, Le: 35,
		Pieces: []ezfyOfficerEquipPiece{
			{Sub: "肩部", Dmg: 120},
			{Sub: "头部", Def: 120},
			{Sub: "挂件", Hp: 130},
			{Sub: "胸部", Hp: 115, CritDmg: 130},
			{Sub: "腰部", Crit: 130, CritDmg: 130},
			{Sub: "勋章", Hp: 115, Crit: 125},
			{Sub: "手部", Dmg: 155, Crit: 125, CritDmg: 125},
			{Sub: "饰品", Dmg: 120, Def: 120},
			{Sub: "名将勋章", Def: 135, Hp: 135},
			{Sub: "足部", Def: 115, Hp: 115, Move: 140},
			{Sub: "名将史册", Crit: 115, CritDmg: 130},
		}},
}

// ezfyOfficerEquipLooseSeeds 散件（不属于任何系列，用黄金买，新手过渡用）
var ezfyOfficerEquipLooseSeeds = []struct {
	ID      int
	Name    string
	Slot    string
	Dmg     int
	Def     int
	Hp      int
	Move    int
	Crit    int
	CritDmg int
	Gold    int64
	Level   int
}{
	{ID: 3001, Name: "和平使者", Slot: "肩部", Dmg: 106, Gold: 2000000, Level: 60},
	{ID: 3002, Name: "军帽", Slot: "头部", Dmg: 106, Gold: 2000000, Level: 60},
	{ID: 3003, Name: "智能机器人头盔", Slot: "头部", Dmg: 107, Gold: 3000000, Level: 80},
	{ID: 3004, Name: "怀表", Slot: "挂件", Def: 107, Gold: 2000000, Level: 60},
	{ID: 3005, Name: "马甲", Slot: "胸部", Def: 110, Gold: 3000000, Level: 80},
	{ID: 3006, Name: "功守道", Slot: "腰部", Def: 108, Gold: 2500000, Level: 70},
	{ID: 3007, Name: "鬼才设计师", Slot: "腰部", Def: 106, Gold: 2000000, Level: 60},
	{ID: 3008, Name: "战无不胜勋章", Slot: "勋章", Dmg: 110, Def: 110, Gold: 5000000, Level: 100},
	{ID: 3009, Name: "开山斧", Slot: "左手", Dmg: 106, Gold: 2000000, Level: 60},
	{ID: 3010, Name: "项链", Slot: "饰品", Crit: 106, Gold: 2000000, Level: 60},
	{ID: 3011, Name: "五星勋章", Slot: "名将勋章", Hp: 115, Gold: 4000000, Level: 90},
	{ID: 3012, Name: "忍者足具", Slot: "足部", Def: 108, Move: 115, Gold: 3000000, Level: 80},
	{ID: 3013, Name: "杜工部集", Slot: "名将史册", Dmg: 115, Def: 115, Gold: 6000000, Level: 100},
}

// seedEzfyEquipSetFamily 按「ID 段」幂等灌一套装备配置
//
// ★ 用 ID 段做闸门而不是「整表为空」：老库已经有第一批套装了，
// 但第二批（英雄系列 21~26 / 散件 3001+）还得能补进去；
// 同时又不会因为管理端删掉某一件就在下次启动把它变回来。
func seedEzfyEquipSetFamily(db *gorm.DB, minID, maxID int, build func() ([]model.EzfyCfgEquipSet, []model.EzfyCfgEquipment)) {
	var cnt int64
	if err := db.Model(&model.EzfyCfgEquipSet{}).Where("id BETWEEN ? AND ?", minID, maxID).
		Count(&cnt).Error; err != nil {
		return
	}
	if cnt > 0 {
		return
	}
	sets, pieces := build()
	if len(sets) > 0 {
		if err := db.Create(&sets).Error; err != nil {
			return
		}
	}
	if len(pieces) > 0 {
		_ = db.Clauses().CreateInBatches(pieces, 200).Error
	}
}

// seedEzfyEquipSets 第一批套装（军/后/学 向，9 件）+ 第二批英雄系列（11 件）+ 散件
func seedEzfyEquipSets(db *gorm.DB) {
	// 第一批：套装 ID 1~17
	seedEzfyEquipSetFamily(db, 1, 20, func() ([]model.EzfyCfgEquipSet, []model.EzfyCfgEquipment) {
		sets := make([]model.EzfyCfgEquipSet, 0, len(ezfyEquipSetSeeds))
		pieces := []model.EzfyCfgEquipment{}
		for _, s := range ezfyEquipSetSeeds {
			sets = append(sets, model.EzfyCfgEquipSet{
				ID: s.ID, Name: s.Name, Parts: s.Parts,
				Military: s.SetMi, Logistics: s.SetLo, Learning: s.SetLe,
				Effect: s.Effect, Des: s.Des,
			})
			for i, slot := range s.Slots {
				pieces = append(pieces, model.EzfyCfgEquipment{
					// 套装件 ID 段：套装 1 → 101~109，套装 17 → 1701~1709
					ID:   s.ID*100 + i + 1,
					Name: s.Name + "·" + slot,
					Type: "套装", Slot: slot, SetId: s.ID, Tier: s.Tier,
					Military: s.PieceMi, Logistics: s.PieceLo, Learning: s.PieceLe,
					Level: s.Level, PriceGold: s.Gold, PriceDiamond: s.Diamond,
					Stock: -1, Effect: s.Effect,
					Des: s.Name + " 套装件（" + slot + "）",
				})
			}
		}
		return sets, pieces
	})

	// 第二批：军官装备系列 21~26（11 件套，六项战斗属性 + 三维属性）
	seedEzfyEquipSetFamily(db, 21, 40, func() ([]model.EzfyCfgEquipSet, []model.EzfyCfgEquipment) {
		sets := []model.EzfyCfgEquipSet{}
		pieces := []model.EzfyCfgEquipment{}
		for _, s := range ezfyOfficerSeriesSeeds {
			// ★ 套装是**额外**加成（穿齐才生效）：
			//   - 六项战斗属性：各件之和 ÷ 4（+25%）
			//   - 三维属性：各件之和 ÷ 4（+25%）
			//   所以「穿齐整套」的实际总加成 ≈ 各件之和 × 1.25。
			sum := struct{ dmg, def, hp, mv, cr, cd, mi, lo, le int }{}
			for _, p := range s.Pieces {
				sum.dmg += p.Dmg
				sum.def += p.Def
				sum.hp += p.Hp
				sum.mv += p.Move
				sum.cr += p.Crit
				sum.cd += p.CritDmg
				// 三维：每件用系列统一值（件上没单独写就用系列的）
				mi, lo, le := p.Mi, p.Lo, p.Le
				if mi == 0 && lo == 0 && le == 0 {
					mi, lo, le = s.Mi, s.Lo, s.Le
				}
				sum.mi += mi
				sum.lo += lo
				sum.le += le
			}
			sets = append(sets, model.EzfyCfgEquipSet{
				ID: s.ID, Name: s.SetName, Parts: len(s.Pieces), Series: s.Series,
				Dmg: sum.dmg / 4, Def: sum.def / 4, Hp: sum.hp / 4,
				Move: sum.mv / 4, Crit: sum.cr / 4, CritDmg: sum.cd / 4,
				Military: sum.mi / 4, Logistics: sum.lo / 4, Learning: sum.le / 4,
				Effect: fmt.Sprintf("穿齐%d件，**额外**再获得：军事+%d 后勤+%d 学识+%d；伤害+%d%% 防御+%d%% 生命+%d%% 移动距离+%d%% 暴击几率+%d%% 暴击伤害+%d%%（各件本身属性另计）",
					len(s.Pieces), sum.mi/4, sum.lo/4, sum.le/4,
					sum.dmg/4, sum.def/4, sum.hp/4, sum.mv/4, sum.cr/4, sum.cd/4),
				Des: s.Series + "系列军官装备（11 部位各 1 件）",
			})
			for i, p := range s.Pieces {
				slot := p.Slot
				if slot == "" {
					if i < len(ezfyOfficerEquipSlots) {
						slot = ezfyOfficerEquipSlots[i]
					}
				}
				mi, lo, le := p.Mi, p.Lo, p.Le
				if mi == 0 && lo == 0 && le == 0 {
					mi, lo, le = s.Mi, s.Lo, s.Le
				}
				pieces = append(pieces, model.EzfyCfgEquipment{
					ID: s.ID*100 + i + 1, Name: s.Series + "[" + p.Sub + "]",
					Type: "军官装备", Series: s.Series, Slot: slot, SetId: s.ID, Tier: s.Tier,
					Level: s.Level, PriceDiamond: s.Diamond, Stock: -1, EnhanceMax: 20,
					Dmg: p.Dmg, Def: p.Def, Hp: p.Hp, Move: p.Move, Crit: p.Crit, CritDmg: p.CritDmg,
					Military: mi, Logistics: lo, Learning: le,
					Effect: "装备+20", Des: s.SetName + " 的" + slot + "部件",
				})
			}
		}
		return sets, pieces
	})

	// 散件：ID 3001+（不属于任何系列；也给一点三维，别让玩家觉得「散件啥都不加」）
	seedEzfyEquipSetFamily(db, 3001, 4000, func() ([]model.EzfyCfgEquipSet, []model.EzfyCfgEquipment) {
		pieces := []model.EzfyCfgEquipment{}
		for _, l := range ezfyOfficerEquipLooseSeeds {
			pieces = append(pieces, model.EzfyCfgEquipment{
				ID: l.ID, Name: l.Name, Type: "军官装备", Slot: l.Slot, SetId: 0, Tier: 2,
				Level: l.Level, PriceGold: l.Gold, Stock: -1, EnhanceMax: 20,
				Dmg: l.Dmg, Def: l.Def, Hp: l.Hp, Move: l.Move, Crit: l.Crit, CritDmg: l.CritDmg,
				Military: 20, Logistics: 10, Learning: 10,
				Effect: "装备+20", Des: "散件军官装备（不属于套装）",
			})
		}
		return nil, pieces
	})
}

// ============ 二·C、宝箱（钻石买，开箱按权重出套装件） ============

// ezfyChestSeed 宝箱模板（奖池由 buildEzfyChestItems 按规则生成）
type ezfyChestSeed struct {
	ID           int
	Name         string
	PriceDiamond int64
	PriceGold    int64
	OpenMax      int
	Des          string
	Effect       string
}

var ezfyChestSeeds = []ezfyChestSeed{
	{ID: 1, Name: "新兵装备宝箱", PriceGold: 500000, OpenMax: 10,
		Des:    "用黄金购买，开出散件军官装备（不属于套装）",
		Effect: "奖池：13 种散件军官装备"},
	{ID: 2, Name: "军官装备宝箱", PriceDiamond: 500, OpenMax: 10,
		Des:    "用钻石购买，随机开出六大系列的一件军官装备（11 部位）",
		Effect: "奖池：六大系列 66 件 + 散件；越稀有的系列权重越低"},
	{ID: 3, Name: "顶级套装宝箱", PriceDiamond: 3000, OpenMax: 5,
		Des:    "用钻石购买，高概率开出顶级系列装备",
		Effect: "奖池：青天白日/赤色锤镰/革命者/渡鸦之魂 为主"},
}

// buildEzfyChestItems 生成宝箱奖池
func buildEzfyChestItems() []model.EzfyCfgChestItem {
	out := []model.EzfyCfgChestItem{}
	add := func(chestID, kind, ref, weight int, quality string) {
		out = append(out, model.EzfyCfgChestItem{
			ID: chestID*1000 + len(out)%1000 + 1, ChestId: chestID,
			Kind: kind, RefId: ref, Count: 1, Weight: weight, Quality: quality,
			Des: "",
		})
	}
	// 宝箱 1：散件（黄金）
	for _, l := range ezfyOfficerEquipLooseSeeds {
		add(1, 1, l.ID, 100, "普通")
	}
	// 宝箱 2：六大系列（钻石 500）
	for _, s := range ezfyOfficerSeriesSeeds {
		w, q := 30, "史诗"
		switch s.ID {
		case 24: // 巨匠（最低档）
			w, q = 100, "稀有"
		case 23: // 黑色幽灵
			w, q = 60, "稀有"
		case 25, 26: // 青天白日 / 赤色锤镰（最高档）
			w, q = 8, "传说"
		}
		for i := 0; i < 11; i++ {
			add(2, 1, s.ID*100+i+1, w, q)
		}
	}
	// 宝箱 3：顶级系列（钻石 3000）
	for _, s := range ezfyOfficerSeriesSeeds {
		w, q := 60, "史诗"
		switch s.ID {
		case 25, 26:
			w, q = 100, "传说"
		case 24:
			w, q = 20, "稀有"
		}
		for i := 0; i < 11; i++ {
			add(3, 1, s.ID*100+i+1, w, q)
		}
	}
	// 三个宝箱都塞一点道具当安慰奖
	for _, c := range []int{1, 2, 3} {
		add(c, 2, 14, 40, "普通") // 经验书
		add(c, 2, 16, 20, "稀有") // 重修书
		add(c, 2, 23, 10, "史诗") // 军官升星卡
	}
	return out
}

// seedEzfyChests 宝箱 + 奖池（按 ID 段幂等，同装备套装那套闸门逻辑）
func seedEzfyChests(db *gorm.DB) {
	var cnt int64
	if err := db.Model(&model.EzfyCfgChest{}).Count(&cnt).Error; err != nil {
		return
	}
	if cnt > 0 {
		return
	}
	chests := []model.EzfyCfgChest{}
	for _, c := range ezfyChestSeeds {
		chests = append(chests, model.EzfyCfgChest{
			ID: c.ID, Name: c.Name, PriceDiamond: c.PriceDiamond, PriceGold: c.PriceGold,
			Stock: -1, OpenMax: c.OpenMax, Enabled: 1, SortNo: c.ID,
			Des: c.Des, Effect: c.Effect,
		})
	}
	if err := db.Create(&chests).Error; err != nil {
		return
	}
	items := buildEzfyChestItems()
	_ = db.CreateInBatches(items, 200).Error
}

// backfillOfficerEquipSetBonus 给已经灌过的军官装备系列补属性（幂等，只补「还是 0」的）
//
// ★ 用户明确：「套装是额外的战斗加成」+「套装装备要有属性加成」→
//   - 每件装备要有 **六项战斗属性**（表格里的）+ **三维属性**（军/后/学，进军官有效属性）
//   - 套装行要有**额外**加成 = 各件之和 ÷ 4（六项 + 三维都算）
//
// 三个 backfill 都只动「全 0」的行，所以：
//   - 管理端改过的值不会被冲掉
//   - 跑过一次之后条件不再命中，天然幂等
func backfillOfficerEquipSetBonus(db *gorm.DB) {
	// ① 散件的三维（ID 3001+）
	db.Exec("UPDATE ezfy_cfg_equipment SET military = 20, logistics = 10, learning = 10 " +
		"WHERE id BETWEEN 3001 AND 4000 AND military = 0 AND logistics = 0 AND learning = 0")

	// ② 各系列的「件」：补三维（每件用系列统一值）
	for _, s := range ezfyOfficerSeriesSeeds {
		db.Exec("UPDATE ezfy_cfg_equipment SET military = ?, logistics = ?, learning = ? "+
			"WHERE set_id = ? AND military = 0 AND logistics = 0 AND learning = 0",
			s.Mi, s.Lo, s.Le, s.ID)
	}

	// ③ 各系列的「套装行」：补额外加成（六项 + 三维，都按各件之和 ÷ 4）
	//
	// ★ 逐字段判断「还是 0 就补」：不能整行一起判断 ——
	//   六项上一轮已经补过、三维还是 0，用整行守卫会被 `Dmg != 0` 直接 continue 掉（踩过）。
	//   逐字段判断同时也保证「管理端手工配过的字段不会被覆盖」。
	for _, s := range ezfyOfficerSeriesSeeds {
		var st model.EzfyCfgEquipSet
		if err := db.First(&st, s.ID).Error; err != nil {
			continue
		}
		var pieces []model.EzfyCfgEquipment
		db.Where("set_id = ?", st.ID).Find(&pieces)
		if len(pieces) == 0 {
			continue
		}
		var d, df, hp, mv, cr, cd, mi, lo, le int
		for _, p := range pieces {
			d += p.Dmg
			df += p.Def
			hp += p.Hp
			mv += p.Move
			cr += p.Crit
			cd += p.CritDmg
			mi += p.Military
			lo += p.Logistics
			le += p.Learning
		}
		updates := map[string]interface{}{}
		fill := func(cur int, col string, val int) {
			if cur == 0 && val != 0 {
				updates[col] = val
			}
		}
		fill(st.Dmg, "dmg", d/4)
		fill(st.Def, "def", df/4)
		fill(st.Hp, "hp", hp/4)
		fill(st.Move, "move", mv/4)
		fill(st.Crit, "crit", cr/4)
		fill(st.CritDmg, "crit_dmg", cd/4)
		fill(st.Military, "military", mi/4)
		fill(st.Logistics, "logistics", lo/4)
		fill(st.Learning, "learning", le/4)
		if len(updates) == 0 {
			continue
		}
		updates["effect"] = fmt.Sprintf(
			"穿齐%d件，**额外**再获得：军事+%d 后勤+%d 学识+%d；伤害+%d%% 防御+%d%% 生命+%d%% 移动距离+%d%% 暴击几率+%d%% 暴击伤害+%d%%（各件本身属性另计）",
			st.Parts, mi/4, lo/4, le/4, d/4, df/4, hp/4, mv/4, cr/4, cd/4)
		db.Model(&model.EzfyCfgEquipSet{}).Where("id = ?", st.ID).Updates(updates)
	}

}

// ============ 二·E、计谋（消耗信号弹） ============
//
// ★ 2026-09-22 用户要求：「信号弹也是道具，可以黄金、钻石购买，加上，用于计谋消耗。」
//
// 12 条计谋来自原版 acade/scheme.html（前端原来写死在 Ezfy.vue 的 schemes 数组里）。
// 现在挪到 ezfy_cfg_scheme 由管理端维护 —— 改消耗数量、上下架都不用改代码。
//
// Kind：0 = 纯说明（原版就是「需要进入相应界面才可以使用」，这里只做消耗 + 战报记录）；
//
//	1 = 先发制人（使双方立即进入可战争状态）
var ezfySchemeSeeds = []model.EzfyCfgScheme{
	{ID: 1, Name: "恫疑虚喝", Des: "恫疑虚喝", Bullet: 4, SortNo: 1},
	{ID: 2, Name: "隐真示假", Des: "隐真示假", Bullet: 4, SortNo: 2},
	{ID: 3, Name: "十面埋伏", Des: "十面埋伏", Bullet: 7, SortNo: 3},
	{ID: 4, Name: "欲擒故纵", Des: "欲擒故纵", Bullet: 7, SortNo: 4},
	{ID: 5, Name: "偷梁换柱", Des: "偷梁换柱", Bullet: 6, SortNo: 5},
	{ID: 6, Name: "反客为主", Des: "反客为主", Bullet: 6, SortNo: 6},
	{ID: 7, Name: "先发制人",
		Des:    "使我军与敌军城市直接进入可战争状态，可战争时间为军官学识×1分钟，最多持续6小时，中计城市6小时内不再中计",
		Bullet: 12, Kind: 1, WarMinutes: 60, WarMaxMinutes: 360, SortNo: 7},
	{ID: 8, Name: "未雨绸缪", Des: "未雨绸缪", Bullet: 12, SortNo: 8},
	{ID: 9, Name: "虚实相乱", Des: "虚实相乱", Bullet: 4, SortNo: 9},
	{ID: 10, Name: "调虎离山", Des: "调虎离山", Bullet: 4, SortNo: 10},
	{ID: 11, Name: "各个击破", Des: "各个击破", Bullet: 8, SortNo: 11},
	{ID: 12, Name: "舍车保帅", Des: "舍车保帅", Bullet: 8, SortNo: 12},
}

// seedEzfySchemes 计谋配置（按 ID 段幂等，同装备套装那套闸门逻辑）
func seedEzfySchemes(db *gorm.DB) {
	var cnt int64
	if err := db.Model(&model.EzfyCfgScheme{}).Count(&cnt).Error; err != nil {
		return
	}
	if cnt > 0 {
		return
	}
	rows := make([]model.EzfyCfgScheme, 0, len(ezfySchemeSeeds))
	for _, s := range ezfySchemeSeeds {
		s.Enabled = 1
		if s.WarMinutes <= 0 {
			s.WarMinutes = 60
		}
		if s.WarMaxMinutes <= 0 {
			s.WarMaxMinutes = 360
		}
		rows = append(rows, s)
	}
	_ = db.CreateInBatches(rows, 50).Error
}

// ============ 二·D、装备快照自愈 ============
//
// ★ 为什么需要它：装备属性是**穿戴时的快照** ——
//
//	`ezfy_equipment`（买到时从装备池抄一份）→ 军官 `equipment` JSON（穿戴时再从背包抄一份）。
//	管理端后来在装备池里补了属性（比如给军官装备补三维），已经买到/已经穿上的不会跟着变，
//	玩家看到的就是「穿了一整套，属性一点没加」（这次踩到的就是这个）。
//
// 两步都只在「快照是 0 / 两边不一致」时才写，稳态下是 no-op（幂等）。
func repairEquipSnapshots(db *gorm.DB) {
	// ① 玩家背包里的装备：从装备池补齐「还是 0」的字段（不覆盖非 0 值 = 不动管理端单独改过的）
	var owned []model.EzfyEquipment
	db.Where("cfg_id > 0").Find(&owned)
	for _, e := range owned {
		var cfg model.EzfyCfgEquipment
		if err := db.First(&cfg, e.CfgId).Error; err != nil {
			continue
		}
		up := map[string]interface{}{}
		fillI := func(cur, val int, col string) {
			if cur == 0 && val != 0 {
				up[col] = val
			}
		}
		fillI(e.Military, cfg.Military, "military")
		fillI(e.Logistics, cfg.Logistics, "logistics")
		fillI(e.Learning, cfg.Learning, "learning")
		fillI(e.Dmg, cfg.Dmg, "dmg")
		fillI(e.Def, cfg.Def, "def")
		fillI(e.Hp, cfg.Hp, "hp")
		fillI(e.Move, cfg.Move, "move")
		fillI(e.Crit, cfg.Crit, "crit")
		fillI(e.CritDmg, cfg.CritDmg, "crit_dmg")
		fillI(e.Enhance, cfg.Enhance, "enhance")
		if e.Slot == "" && cfg.EquipSlot() != "" {
			up["slot"] = cfg.EquipSlot()
		}
		if e.SetId == 0 && cfg.SetId != 0 {
			up["set_id"] = cfg.SetId
		}
		if e.Series == "" && cfg.Series != "" {
			up["series"] = cfg.Series
		}
		if len(up) > 0 {
			db.Model(&model.EzfyEquipment{}).Where("id = ?", e.ID).Updates(up)
		}
	}

	// ② 军官身上的装备 JSON：从背包行重建，保证「穿在身上的」和「背包里的」永远一致
	var offs []model.EzfyOfficer
	db.Where("equipment <> ''").Find(&offs)
	for _, o := range offs {
		var items []model.EzfyEquipment
		db.Where("officer_id = ?", o.ID).Order("id").Find(&items)
		if len(items) == 0 {
			continue // 一件都没有就别动（可能是历史脏数据，宁可留着让人排查）
		}
		list := []map[string]interface{}{}
		for _, e := range items {
			list = append(list, map[string]interface{}{
				"id": e.ID, "name": e.Name, "type": e.Type, "slot": e.EquipSlot(), "set_id": e.SetId,
				"military": e.Military, "logistics": e.Logistics, "learning": e.Learning,
				"series": e.Series, "enhance": e.Enhance,
				"dmg": e.Dmg, "def": e.Def, "hp": e.Hp,
				"move": e.Move, "crit": e.Crit, "crit_dmg": e.CritDmg,
			})
		}
		b, err := json.Marshal(list)
		if err != nil || string(b) == o.Equipment {
			continue
		}
		db.Model(&model.EzfyOfficer{}).Where("id = ?", o.ID).Update("equipment", string(b))
	}
}

// ============ 三、存量军官：属性点迁移 ============

// migrateOfficerAttrPoints 给存量军官补「原始属性 base_*」与「可用属性点 free_points」
//
// ★ 规则（2026-09-22 用户确认）：
//   - base_* = 军官的**原始属性**：能从军官池查到就用池子里的值（重修书洗点回退到这个值）；
//     查不到（历史随机生成的普通军官，general_id=0）就用当前属性。
//   - free_points = 等级 − 1（每级 1 点；老军官之前没有点数概念，等于都还没点，一次性补发）。
//
// 幂等：只要库里已经有任何一名军官 base_* 非 0，就认为迁移跑过了，直接跳过。
func migrateOfficerAttrPoints(db *gorm.DB) {
	var done int64
	if err := db.Model(&model.EzfyOfficer{}).
		Where("base_military > 0 OR base_logistics > 0 OR base_learning > 0").
		Count(&done).Error; err != nil {
		return
	}
	if done > 0 {
		return
	}
	var total int64
	if err := db.Model(&model.EzfyOfficer{}).Count(&total).Error; err != nil || total == 0 {
		return
	}
	// ★ 必须用 COALESCE：AutoMigrate 新加的列在老行上是 NULL，
	//   而 `col = 0` 对 NULL 不成立（会一条都更新不到，看起来像「迁移没跑」）。
	// ① 先全部按「当前属性」落 base（兜底）
	db.Exec("UPDATE ezfy_officer SET base_military = military, base_logistics = logistics, " +
		"base_learning = learning " +
		"WHERE COALESCE(base_military,0) = 0 AND COALESCE(base_logistics,0) = 0 " +
		"AND COALESCE(base_learning,0) = 0")
	// ② 能对上军官池的（名将 / 以后新招的池子军官），base 改成池子里的原始属性
	db.Exec("UPDATE ezfy_officer o JOIN ezfy_cfg_general g ON g.id = o.general_id " +
		"SET o.base_military = g.military, o.base_logistics = g.logistics, o.base_learning = g.learning " +
		"WHERE o.general_id > 0")
	// ③ 补发可用属性点：每级 1 点
	db.Exec("UPDATE ezfy_officer SET free_points = GREATEST(level - 1, 0) WHERE COALESCE(free_points,0) = 0")
	// ④ 顺手把老行上的 NULL 归零，避免 JSON 里出现 null
	db.Exec("UPDATE ezfy_officer SET base_military = COALESCE(base_military,0), " +
		"base_logistics = COALESCE(base_logistics,0), base_learning = COALESCE(base_learning,0), " +
		"free_points = COALESCE(free_points,0)")
}

// normalizeEzfyNewCols 把 AutoMigrate 新加列在老行上留下的 NULL 归零/归空串
//
// ★ 为什么必须做：`gorm:"default:0"` 只会影响**新加列时的建表语句**，
// 对已经存在的列（生产库升级）不会回填；而 Go 的 int/string 读到 NULL
// 虽然不报错，但下发给前端会变成 JSON 的 null，前端 `row.stock > 0` 之类的判断会走偏。
// 幂等：条件里全是 IS NULL，跑过一次之后就不会再命中。
func normalizeEzfyNewCols(db *gorm.DB) {
	// ★ 命名统一：这批装备是**军官装备**（原来叫「英雄装备」），种子灌过一遍之后
	//   老库里还是旧名字，这里做一次幂等改名（改过之后条件不再命中）。
	db.Exec("UPDATE ezfy_cfg_equipment SET type = '军官装备' WHERE type = '英雄装备'")
	db.Exec("UPDATE ezfy_equipment SET type = '军官装备' WHERE type = '英雄装备'")
	// ★ 用户规则「军官最多 5 星」：早期默认值灌成了 10，这里改回 5。
	//   条件写 `= 10`（而不是 `> 5`）是为了**只修这一次**——
	//   管理端后来自己调过的值（比如 8）不会在每次启动被打回去。
	db.Exec("UPDATE ezfy_cfg_limit SET officer_star_max = 5 WHERE id = 1 AND officer_star_max = 10")
	// 超过 5 星的存量军官夹回 5 星（幂等：夹过之后没有行再匹配）
	db.Exec("UPDATE ezfy_officer SET star = 5 WHERE star > 5")
	db.Exec("UPDATE ezfy_cfg_equipment SET slot = COALESCE(slot,''), set_id = COALESCE(set_id,0), " +
		"price_gold = COALESCE(price_gold,0), price_diamond = COALESCE(price_diamond,0), " +
		"stock = COALESCE(stock,-1), effect = COALESCE(effect,''), " +
		"series = COALESCE(series,''), enhance = COALESCE(enhance,0), enhance_max = COALESCE(enhance_max,20), " +
		"dmg = COALESCE(dmg,0), def = COALESCE(def,0), hp = COALESCE(hp,0), " +
		"move = COALESCE(move,0), crit = COALESCE(crit,0), crit_dmg = COALESCE(crit_dmg,0)")
	db.Exec("UPDATE ezfy_equipment SET slot = COALESCE(slot,''), set_id = COALESCE(set_id,0), " +
		"series = COALESCE(series,''), enhance = COALESCE(enhance,0), " +
		"dmg = COALESCE(dmg,0), def = COALESCE(def,0), hp = COALESCE(hp,0), " +
		"move = COALESCE(move,0), crit = COALESCE(crit,0), crit_dmg = COALESCE(crit_dmg,0)")
	db.Exec("UPDATE ezfy_cfg_general SET kind = COALESCE(kind,2), weight = COALESCE(weight,100)")
	db.Exec("UPDATE ezfy_cfg_equip_set SET military = COALESCE(military,0), logistics = COALESCE(logistics,0), " +
		"learning = COALESCE(learning,0), effect = COALESCE(effect,''), des = COALESCE(des,''), " +
		"series = COALESCE(series,''), dmg = COALESCE(dmg,0), def = COALESCE(def,0), hp = COALESCE(hp,0), " +
		"move = COALESCE(move,0), crit = COALESCE(crit,0), crit_dmg = COALESCE(crit_dmg,0)")
}
