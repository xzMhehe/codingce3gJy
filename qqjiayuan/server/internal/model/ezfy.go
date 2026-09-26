package model

import "time"

// 二战风云（ezfy）—— 复刻 stzb-fk「二战风云」，全部数据表使用 ezfy_ 前缀

// ============ 配置表（seed 幂等写入） ============

type EzfyCfgBuilding struct {
	ID          int    `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name        string `gorm:"type:varchar(50);comment:名称" json:"name"`
	Type        int    `gorm:"default:0;comment:1资源 2军事 3城防 4市政" json:"type"` // 1资源 2军事 3城防 4市政
	MaxLevel    int    `gorm:"default:1;comment:上限等级" json:"max_level"`
	UniqueFlag  int    `gorm:"default:0;comment:市政厅=1" json:"unique_flag"` // 市政厅=1
	CanDelete   int    `gorm:"comment:可否删除" json:"can_delete"`
	PreBuilding string `gorm:"type:varchar(255);comment:前缀建筑" json:"pre_building"`
	Des         string `gorm:"type:varchar(500);comment:描述" json:"des"`
}

func (EzfyCfgBuilding) TableName() string { return "ezfy_cfg_building" }

type EzfyCfgBuildingLevel struct {
	ID         int    `gorm:"primaryKey;comment:主键ID" json:"id"`
	BuildingId int    `gorm:"comment:建筑ID" json:"building_id"`
	Level      int    `gorm:"comment:等级" json:"level"`
	Pop        int    `gorm:"comment:人口" json:"pop"`
	Food       int64  `gorm:"comment:粮食" json:"food"`
	Steel      int64  `gorm:"comment:钢铁" json:"steel"`
	Oil        int64  `gorm:"comment:石油" json:"oil"`
	Rare       int64  `gorm:"comment:稀矿" json:"rare"`
	Gold       int64  `gorm:"comment:黄金" json:"gold"`
	BuildTime  int    `gorm:"comment:建造时间（秒）" json:"build_time"` // 秒
	Capacity   int64  `gorm:"comment:容量" json:"capacity"`
	Effect     string `gorm:"type:varchar(500);comment:效果" json:"effect"`
}

func (EzfyCfgBuildingLevel) TableName() string { return "ezfy_cfg_building_level" }

type EzfyCfgTroop struct {
	ID          int    `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name        string `gorm:"type:varchar(50);comment:名称" json:"name"`
	NameAxis    string `gorm:"type:varchar(50);comment:轴心国名称" json:"name_axis"` // 轴心国名称
	NameAlly    string `gorm:"type:varchar(50);comment:同盟国名称" json:"name_ally"` // 同盟国名称
	Type        int    `gorm:"default:2;comment:1海军 2陆军 3空军 4城防" json:"type"`   // 1海军 2陆军 3空军 4城防
	Health      int    `gorm:"comment:耐久" json:"health"`
	AtkSea      int    `gorm:"comment:攻击海域" json:"atk_sea"`
	AtkGround   int    `gorm:"comment:攻击陆地" json:"atk_ground"`
	AtkAir      int    `gorm:"comment:攻击空中" json:"atk_air"`
	AtkDef      int    `gorm:"comment:攻击防御" json:"atk_def"`
	Defence     int    `gorm:"comment:防御" json:"defence"`
	Speed       int    `gorm:"comment:速度" json:"speed"`
	AttackRange int    `gorm:"comment:攻击范围" json:"attack_range"`
	Carry       int    `gorm:"comment:携带" json:"carry"`
	Pop         int    `gorm:"comment:人口" json:"pop"`
	FoodKeep    int    `gorm:"comment:维护耗粮/小时/个" json:"food_keep"` // 维护耗粮/小时/个
	OilKeep     int    `gorm:"comment:石油保留" json:"oil_keep"`
	Food        int64  `gorm:"comment:粮食" json:"food"`
	Steel       int64  `gorm:"comment:钢铁" json:"steel"`
	Oil         int64  `gorm:"comment:石油" json:"oil"`
	Rare        int64  `gorm:"comment:稀矿" json:"rare"`
	TrainTime   int    `gorm:"comment:秒/个" json:"train_time"` // 秒/个
	Require     string `gorm:"type:varchar(500);comment:需要" json:"require"`
	Icon        string `gorm:"type:varchar(50);comment:图标" json:"icon"`
	RepairRate  int    `gorm:"comment:战损修复率%" json:"repair_rate"` // 战损修复率%
}

func (EzfyCfgTroop) TableName() string { return "ezfy_cfg_troop" }

type EzfyCfgTech struct {
	ID           int    `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name         string `gorm:"type:varchar(50);comment:名称" json:"name"`
	Type         int    `gorm:"default:3;comment:1生产 2军事 3辅助" json:"type"` // 1生产 2军事 3辅助
	MaxLevel     int    `gorm:"default:10;comment:上限等级" json:"max_level"`
	PreBuilding  int    `gorm:"comment:需要科研中心等级" json:"pre_building"` // 需要科研中心等级
	PreTech      int    `gorm:"comment:前缀科技" json:"pre_tech"`
	PreTechLevel int    `gorm:"comment:前缀科技等级" json:"pre_tech_level"`
	Effect       string `gorm:"type:varchar(255);comment:效果" json:"effect"`
	Des          string `gorm:"type:varchar(500);comment:描述" json:"des"`
}

func (EzfyCfgTech) TableName() string { return "ezfy_cfg_tech" }

type EzfyCfgTechLevel struct {
	ID           int    `gorm:"primaryKey;comment:主键ID" json:"id"`
	TechId       int    `gorm:"comment:科技ID" json:"tech_id"`
	Level        int    `gorm:"comment:等级" json:"level"`
	Food         int64  `gorm:"comment:粮食" json:"food"`
	Steel        int64  `gorm:"comment:钢铁" json:"steel"`
	Oil          int64  `gorm:"comment:石油" json:"oil"`
	Rare         int64  `gorm:"comment:稀矿" json:"rare"`
	Gold         int64  `gorm:"comment:黄金" json:"gold"`
	ResearchTime int    `gorm:"comment:研究时间（秒）" json:"research_time"` // 秒
	Effect       string `gorm:"type:varchar(255);comment:效果" json:"effect"`
}

func (EzfyCfgTechLevel) TableName() string { return "ezfy_cfg_tech_level" }

type EzfyCfgWildland struct {
	ID         int    `gorm:"primaryKey;comment:主键ID" json:"id"`
	Type       int    `gorm:"comment:1陆地野地 2海野 3寇城" json:"type"` // 1陆地野地 2海野 3寇城
	Level      int    `gorm:"comment:等级" json:"level"`
	Troops     string `gorm:"type:varchar(1000);comment:[[兵种id,最小,最大],...]" json:"troops"` // [[兵种id,最小,最大],...]
	ResMin     int64  `gorm:"comment:资源下限" json:"res_min"`
	ResMax     int64  `gorm:"comment:资源上限" json:"res_max"`
	OfficerMin int    `gorm:"comment:旧字段，已被 OfficerId 取代（保留避免迁移麻烦）" json:"officer_min"` // 旧字段，已被 OfficerId 取代（保留避免迁移麻烦）
	OfficerMax int    `gorm:"comment:旧字段，同上" json:"officer_max"`                        // 旧字段，同上
	// ★ 守军军官：**最多 1 个**，且只能从「军官池」（ezfy_cfg_general）里选。
	//   0 = 该野地没有守将（打下来也俘不到军官）。
	OfficerId int    `gorm:"comment:军官ID" json:"officer_id"`
	Treasure  string `gorm:"type:varchar(100);comment:宝物" json:"treasure"`
	Des       string `gorm:"type:varchar(500);comment:描述" json:"des"`
}

func (EzfyCfgWildland) TableName() string { return "ezfy_cfg_wildland" }

// EzfyMapTile 地图格子覆盖（管理端维护）
//
// ★ 用户要求：管理端要能维护**所有**野地（不只是玩家已占领的），能改土地类型，
//
//	也能把某格设成 寇城 / 活动寇城。
//
//	地图本身是「坐标哈希推导」出来的（地形、野地等级、寇城、活动目标全都不落库），
//	所以这里做一张**覆盖表**：命中就覆盖哈希结果，没命中就照旧走哈希。
type EzfyMapTile struct {
	ID uint `gorm:"primaryKey;comment:主键ID" json:"id"`
	X  int  `gorm:"uniqueIndex:uk_map_tile;comment:X坐标" json:"x"`
	Y  int  `gorm:"uniqueIndex:uk_map_tile;comment:Y坐标" json:"y"`
	// Terrain: 0 = 不覆盖（按哈希），1-9 = 强制成该地形（1平原…8海洋, 9沿海平原）
	Terrain int `gorm:"comment:地形" json:"terrain"`
	// MarkKind: 0=无 1=寇城 2=活动寇城 3=活动野地 4=特殊城市
	MarkKind int `gorm:"comment:MarkKind: 0=无 1=寇城 2=活动寇城 3=活动野地 4=特殊城市" json:"mark_kind"`
	// MarkLevel: 活动目标等级 1~3（仅 2/3/4 有意义）
	MarkLevel int       `gorm:"comment:MarkLevel: 活动目标等级 1~3（仅 2/3/4 有意义）" json:"mark_level"`
	Des       string    `gorm:"type:varchar(200);comment:描述" json:"des"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyMapTile) TableName() string { return "ezfy_map_tile" }

// 地图格子标记类型
const (
	EzfyMarkNone    = 0 // 无（按哈希）
	EzfyMarkKou     = 1 // 普通寇城
	EzfyMarkActKou  = 2 // 活动寇城
	EzfyMarkActWild = 3 // 活动野地
	EzfyMarkActCity = 4 // 特殊城市
)

// EzfyCfgRank 军衔配置（复刻原版 rankIndex.html：军衔等级/职位要求/可建城数）
//
// 原来硬编码在 handler 里，现在落表，管理端可维护。
// ★ 可建城数就是「玩家能拥有的城市数量上限」。
type EzfyCfgRank struct {
	ID           int    `gorm:"primaryKey;comment:1..20，同时也是军衔等级" json:"id"` // 1..20，同时也是军衔等级
	Name         string `gorm:"type:varchar(20);comment:名称" json:"name"`
	Post         string `gorm:"type:varchar(20);comment:帖子（职位）" json:"post"` // 职位
	NeedPrestige int    `gorm:"comment:需要的声望" json:"need_prestige"`          // 需要的声望
	CityMax      int    `gorm:"comment:可建城数" json:"city_max"`                // 可建城数
	Des          string `gorm:"type:varchar(200);comment:描述" json:"des"`
}

func (EzfyCfgRank) TableName() string { return "ezfy_cfg_rank" }

// EzfyFriend 游戏内好友（★ 与家园好友完全分开，不共用 Friendship 表）
//
// 双向各存一行：A 加 B 成功时写 (A,B) 与 (B,A)，删除时两行一起删。
type EzfyFriend struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId    uint      `gorm:"uniqueIndex:uk_ezfy_friend;comment:用户ID" json:"user_id"`
	FriendId  uint      `gorm:"uniqueIndex:uk_ezfy_friend;comment:好友ID" json:"friend_id"`
	Remark    string    `gorm:"type:varchar(50);comment:备注" json:"remark"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyFriend) TableName() string { return "ezfy_friend" }

// EzfyFriendApply 游戏内好友申请
type EzfyFriendApply struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId    uint      `gorm:"index:idx_ezfy_apply;comment:申请人" json:"user_id"`    // 申请人
	TargetId  uint      `gorm:"index:idx_ezfy_apply;comment:被申请人" json:"target_id"` // 被申请人
	Remark    string    `gorm:"type:varchar(100);comment:备注" json:"remark"`
	Status    int       `gorm:"default:0;comment:0待处理 1已同意 2已拒绝" json:"status"` // 0待处理 1已同意 2已拒绝
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyFriendApply) TableName() string { return "ezfy_friend_apply" }

// EzfyCfgResource 资源显示名配置（管理端可改名，全站展示跟随）
//
// ★ 这是「预留」能力：后期把「稀矿」改成别的叫法，只要改这张表，
// 游戏端 / 管理端的资源名就全部跟着变，不用改代码。
// Key 是程序内部标识（gold/food/steel/oil/rare），Name 是展示名，Short 是单字简称（金/粮/钢/油/稀）。
type EzfyCfgResource struct {
	ID    int    `gorm:"primaryKey;comment:主键ID" json:"id"`
	Key   string `gorm:"type:varchar(20);uniqueIndex;comment:键名" json:"key"`
	Name  string `gorm:"type:varchar(30);comment:名称" json:"name"`
	Short string `gorm:"type:varchar(10);comment:Short" json:"short"`
	Sort  int    `gorm:"comment:排序值" json:"sort"`
}

func (EzfyCfgResource) TableName() string { return "ezfy_cfg_resource" }

type EzfyCfgItem struct {
	ID          int    `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name        string `gorm:"type:varchar(50);comment:名称" json:"name"`
	ItemType    int    `gorm:"comment:1资源包 2黄金包 3建筑加速 4训练加速 5科技加速 6建筑图纸 7增产 8免战" json:"item_type"` // 1资源包 2黄金包 3建筑加速 4训练加速 5科技加速 6建筑图纸 7增产 8免战
	Param1      int64  `gorm:"comment:参数1" json:"param1"`
	PriceGold   int64  `gorm:"comment:黄金价格" json:"price_gold"`
	Icon        string `gorm:"type:varchar(50);comment:图标" json:"icon"`
	Description string `gorm:"type:varchar(500);comment:描述" json:"description"`
	// ★ 商城库存（管理端「数据管理 → 道具配置」可改），默认 100；0 = 售罄
	Stock int `gorm:"default:100;comment:库存" json:"stock"`
	// ★ 钻石售价：> 0 表示这是「钻石道具」，只能用钻石购买（黄金价 price_gold 忽略）。
	//   管理端「数据管理 → 道具配置」可维护。
	PriceDiamond int64 `gorm:"comment:钻石价格" json:"price_diamond"`
	// ★ 商城分类（管理端可填；留空时按 item_type / price_diamond 自动归类）
	Category string `gorm:"type:varchar(20);comment:分类" json:"category"`
}

func (EzfyCfgItem) TableName() string { return "ezfy_cfg_item" }

// EzfyCfgLimit 二战风云「建筑数量上限」全局配置（单行，id = 1）
//
// 用户规则：军事区与资源区数量上限**分开**，各 36；管理端可维护，默认 36。
type EzfyCfgLimit struct {
	ID          int `gorm:"primaryKey;comment:主键ID" json:"id"`
	MilitaryMax int `gorm:"default:36;comment:军事区建筑数量上限（type 2/3/4）" json:"military_max"` // 军事区建筑数量上限（type 2/3/4）
	ResourceMax int `gorm:"default:36;comment:资源区建筑数量上限（type 1）" json:"resource_max"`     // 资源区建筑数量上限（type 1）
	HouseMax    int `gorm:"default:33;comment:民居数量上限" json:"house_max"`                   // 民居数量上限
	FactoryMax  int `gorm:"default:20;comment:军工厂数量上限（0 = 不限）" json:"factory_max"`        // 军工厂数量上限（0 = 不限）
	// ★ 用户要求「首页公告默认只能展示一条，管理端可以配置」→ 首页外露公告条数（默认 1）
	NoticeHomeCount int `gorm:"default:1;comment:公告家园数量" json:"notice_home_count"`
	// ★ 用户要求「出征集结令上限后台管理系统可维护，最大默认 99」→ 单次出征最多用几个集结令（默认 99）
	GatherMaxPerOrder int `gorm:"default:99;comment:集结上限每订单" json:"gather_max_per_order"`

	// ============ 战斗 / 经济数值（管理端「建筑上限配置」页可维护）============
	//
	// ★ 用户反馈「征服民心每次 -5 现在太多」→ 做成可配置（线上现值 5）。
	//   征服(3)：单次最多扣掉目标多少民心（原来写死 20，且按幸存兵力/2000 动态计算）。
	//   掠夺(2)：每次固定扣目标多少民心（原来写死 5，线上现值 3）。
	ConquerFeelingsMax int `gorm:"default:5;comment:Conquer民心上限" json:"conquer_feelings_max"`
	LootFeelings       int `gorm:"default:3;comment:掠夺民心" json:"loot_feelings"`
	// ★ 用户反馈「军官是消耗黄金的，黄金现在消耗 0」→ 军官工资：每名军官每小时消耗
	//   「等级 × 该值」黄金，在 calcResource 里随资源懒结算一起扣。默认 100 黄金/级/小时。
	//   ⚠️ 字段名必须让 GORM 推出 officer_salary_per_level（与 seed 里补的列名一致），
	//   否则 AutoMigrate 会另外建一列 officer_salary_per_lv，两个列各存各的。
	OfficerSalaryPerLevel int `gorm:"default:100;comment:军官Salary每等级" json:"officer_salary_per_level"`
	// ★ 用户反馈「恢复伤兵需要黄金」→ 恢复 1 个伤兵消耗
	//   ceil(该兵种总造价 / 该值) 黄金，最低 1 黄金。默认 50。
	WoundHealDivisor int `gorm:"default:50;comment:伤兵治疗Divisor" json:"wound_heal_divisor"`
	// ★ 用户要求「商城购买卡控改成可配置的」→
	//   商城单次购买数量上限（下限恒为 1）。线上现值 99。
	//   读不到或 <= 0 时回落默认值（0 无意义 = 等于禁止购买）。
	MallBuyMax int `gorm:"default:99;comment:Mall购买上限" json:"mall_buy_max"`

	// ★ 用户要求「采集 12 小时才有宝物 → 可配置」：
	//   常驻采集结算一期的小时数（默认 1），由 ezfyDispatchPeriod() 读取。
	DispatchPeriodH int `gorm:"default:1;comment:采集周期小时" json:"dispatch_period_h"`

	// ★ 出征速度加成（百分比口径，0 = 无加成）：实际行军时间 = 原时间 × 100/(100+加成)。
	//   ★ 2026-09-24 用户要求「节假日我好让玩家队伍走快点」。默认 100。
	MarchSpeedBonus float64 `gorm:"default:100;comment:出征速度加成" json:"march_speed_bonus"`

	// ============ 系统配置（管理端「系统配置」页可维护）============
	//
	//   ★ 野地兵力倍数：野地/海野/寇城的守军兵力 = 配置值 × 该倍数，默认 10。
	//   预览(野地详情)与战斗结算(parseWildlandTroops)共用，避免「看到的」和「打到的」不一致。
	//   允许小数（0.5 = 兵力减半，2 = 翻倍）。0 无意义 → 回落 10。
	WildTroopMult float64 `gorm:"default:10;comment:野地部队倍数" json:"wild_troop_mult"`
	// ★ 2026-09-25 用户反馈「野地打完获得的资源太少」→ 加「野地获取资源倍率」。
	//   作用点：**野地/海野/寇城战斗胜利后的战利品**（ezfy_order.go 的 `rnd` 那一处），
	//   默认 10 = 10 倍（线上现值）；2 = 翻倍；0.5 = 减半。允许小数。
	//   注意：只作用于「打赢的战利品」，不含驻守采集（采集另有自己的产出公式）。
	WildResMult float64 `gorm:"default:10;comment:野地战利品资源倍率" json:"wild_res_mult"`
	// ★ 2026-09-25 用户要求「采集资源倍率也加到系统管理里」→ 常驻采集产出资源 × 该倍数。
	//   作用点：dispatchGatherYield 的产出（等级 × 800 × 后勤加成 × 陆海系数）。
	//   默认 10 = 10 倍（线上现值）；0.5 = 减半。允许小数。0 无意义 → 回落 10。
	GatherResMult float64 `gorm:"default:10;comment:采集资源倍率" json:"gather_res_mult"`

	// ★ 2026-09-25 用户要求「各项资源有最大的配置放到二战系统配置里面，默认 100 亿」：
	//   每项资源的**硬上限**（入库累加的收敛点），默认 100 亿 = 10000000000。
	//
	//   规则（用户确认的原版口径）：
	//     ① 资源**产量**（calcResource 自动产出）超过「仓储上限」就不再增加（原有逻辑，不动）；
	//     ② 其它一切获取方式（战斗掠夺/野地战利品/采集返航/运输/派遣/资源包/签到福利/交易行/军团商城…）
	//        **一律无条件累加**，只在这个「资源最大值」处停下来；
	//     ③ 已经超过该值的老数据**不会被拉低**（读端用 GREATEST 保住较大值）。
	//   ⚠️ 必须 BIGINT：100 亿超出 int32，seed 补列走的是 bigint。
	//   0 无意义 → 回落默认 100 亿。
	ResMaxFood int64 `gorm:"default:10000000000;comment:粮食最大值" json:"res_max_food"`
	ResMaxSteel int64 `gorm:"default:10000000000;comment:钢铁最大值" json:"res_max_steel"`
	ResMaxOil   int64 `gorm:"default:10000000000;comment:石油最大值" json:"res_max_oil"`
	ResMaxRare  int64 `gorm:"default:10000000000;comment:稀矿最大值" json:"res_max_rare"`
	ResMaxGold  int64 `gorm:"default:10000000000;comment:黄金最大值" json:"res_max_gold"`

	// ★ 下面三个是「开关」：1 = 开（按原规则消耗），0 = 关（不消耗）。
	//   ⚠️ 语义陷阱（踩过）：
	//     ① 开关字段**不能**带 `gorm:"default:x"` 标签 —— GORM 建 INSERT/ON DUPLICATE 时
	//        会跳过零值字段，导致「关」(0) 永远写不进库。
	//     ② seed 里**不能**走 addLimitCol()（那个 `WHERE col <= 0` 会在每次启动把 0 回填成默认值）。
	//        开关的补列走单独的 addSwitchCol()，只回填 NULL。
	//     ③ 读取端**不能**用 ezfyLimitOr(v, def)（它把 0 当「没配」回落默认值）——
	//        这里 0 是有意义的值，直接 `!= 0` 判断。
	RecruitCostOn int `gorm:"comment:征兵消耗资源（关 = 不消耗资源、也无需空闲人口）" json:"recruit_cost_on"` // 征兵消耗资源（关 = 不消耗资源、也无需空闲人口）
	FoodUpkeepOn  int `gorm:"comment:军队耗粮（关 = 城内军队每小时不扣粮）" json:"food_upkeep_on"`       // 军队耗粮（关 = 城内军队每小时不扣粮）
	MarchOilOn    int `gorm:"comment:出征油耗（关 = 出征不消耗石油）" json:"march_oil_on"`            // 出征油耗（关 = 出征不消耗石油）
	// ★ 宣战功能（关 = 玩家之间不需要宣战，直接就能掠夺/征服别人城市）
	WarRequireOn int `gorm:"comment:战争需要开启" json:"war_require_on"`
	// ★ 出征兵力上限（关 = 出征不限兵力，随便带多少；司令部等级那套上限失效）
	MarchCapOn int `gorm:"comment:出征上限开启" json:"march_cap_on"`

	// ★ 2026-09-26 用户要求「召集人口那里加『民居容量限制』『召集人口灵活配置』两个开关」：
	//   ① HousePopLimitOn 民居容量限制：1 开（默认）= 民居容量决定人口上限 pop_max，
	//      人口自然增长到 pop_max 封顶；0 关 = 民居不再限制人口，人口可无限增长。
	//   ② ConveneFlexibleOn 召集人口灵活配置：1 开（默认）= 召集人口不受民居上限限制、
	//      可突破 pop_max（原有行为）；0 关 = 召集同样受民居容量上限约束。
	//   ⚠️ 两个都是开关（0 有意义），不能带 gorm:"default:x" 标签，seed 走 addSwitchCol。
	HousePopLimitOn   int `gorm:"comment:民居容量限制开关（关 = 民居不限制人口上限）" json:"house_pop_limit_on"`
	ConveneFlexibleOn int `gorm:"comment:召集人口灵活配置（关 = 召集同样受民居上限约束）" json:"convene_flexible_on"`

	// ★ 2026-09-26 用户要求「花费 10万粮食 召集 10万人口也要能配置，现在是写死的」：
	//   召集消耗粮食 + 召集获得人口，默认各 10 万。0 无意义 → 回落默认（seed 走 addLimitCol）。
	ConveneFoodCost int `gorm:"comment:召集消耗粮食" json:"convene_food_cost"`
	ConvenePopGain  int `gorm:"comment:召集获得人口" json:"convene_pop_gain"`

	// ============ 军官升星（2026-09-22 用户要求，2026-09-23 按用户要求简化）============
	//
	// ★ 简化后的规则：升星按固定概率（officer_star_chance），失败也消耗 1 枚星级徽章，
	//   每升 1 星三维各 +officer_star_attr_gain，星级上限 officer_star_max。
	//   原先的「概率开关/每高 1 星递减/成功率下限/失败保留徽章」四个配置已按用户要求去掉。
	//
	// ⚠️ OfficerStarUpOn 是**开关**（0 有意义），不能带 gorm:"default:x" 标签，seed 走 addSwitchCol。
	OfficerStarUpOn int `gorm:"comment:升星功能：1 开 / 0 关（关了不能用升星卡）" json:"officer_star_up_on"` // 升星功能：1 开 / 0 关（关了不能用升星卡）

	// 下面是**数值**（0 无意义 → 回落默认值）
	OfficerStarChance   int `gorm:"default:20;comment:升星成功率%（固定值，默认 20）" json:"officer_star_chance"`      // 升星成功率%（固定值，默认 20）
	OfficerStarAttrGain int `gorm:"default:10;comment:每升 1 星三维各 +N（默认 10）" json:"officer_star_attr_gain"` // 每升 1 星三维各 +N（默认 10）
	OfficerStarMax      int `gorm:"default:5;comment:星级上限（默认 5）" json:"officer_star_max"`                 // 星级上限（默认 5）

	// ★ 训练一键加速黄金倍率（百分比口径）：实际费用 = 剩余秒数 × 10 × 倍率/100。
	//   默认 0.1 = 0.1% = 几乎免费（线上现值）；100 = 原价；50 = 半价。
	//   0 无意义 → 回落 0.1。
	SpeedTrainRate float64 `gorm:"default:0.1;comment:速度训练比率" json:"speed_train_rate"`

	// ★ 伤兵恢复黄金折扣率（百分比口径）：恢复费用 = 兵种总造价 / wound_heal_divisor × 折扣率/100。
	//   默认 100 = 100% = 原价；调低 = 恢复便宜。0 无意义 → 回落 100。
	WoundHealRate float64 `gorm:"default:100;comment:伤兵治疗比率" json:"wound_heal_rate"`

	// ============ 数值安全卡控（2026-09-23 线上「负数兵力」事故）============
	//
	// ★ 事故现象：玩家「总兵力」显示 -8843547888967622000 —— int64 正向溢出翻负。
	//   根因：训练 / 伤兵恢复累加**没有任何上限**，单兵种 count 加到超过 int64 上限就翻负。
	//
	// troop_max：单城兵力上限（口径 = 城内现有部队 + 训练队列里还没出厂的新兵）。
	//   训练与伤兵恢复前先校验，超出直接拒绝并提示「超过限额」；
	//   addTroop 落库前再夹取一次作为最后保险，保证任何路径都写不进负数/溢出值。
	//   默认 50 亿 —— 远小于 int64 上限，正常玩法摸不到，纯防溢出与数值膨胀。
	TroopMax int64 `gorm:"default:5000000000;comment:部队上限" json:"troop_max"`
	// wound_expire_days：伤兵在营存活天数，超过则自动消失（默认 3 天）。
	//   口径按「最后一次入营时间」(ezfy_wounded.updated_at) 算，持续有新伤兵入营会顺延。
	WoundExpireDays int `gorm:"default:3;comment:伤兵过期天数" json:"wound_expire_days"`
}

func (EzfyCfgLimit) TableName() string { return "ezfy_cfg_limit" }

// EzfyWordFilter 二战风云聊天敏感词（独立于社区「黑名单榜」的 word_filters）
//
// 用户规则：二战的聊天敏感词走自己的单独维护页面。
// Type: 1 = 替换（用 Replace 覆盖），2 = 拦截（直接拒绝发言）。
type EzfyWordFilter struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	Word      string    `gorm:"type:varchar(50);uniqueIndex:uk_ezfy_word;comment:词语" json:"word"`
	Replace   string    `gorm:"type:varchar(50);comment:替换" json:"replace"`
	Type      int       `gorm:"default:1;comment:类型" json:"type"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyWordFilter) TableName() string { return "ezfy_word_filter" }

type EzfyCfgTaskType struct {
	ID        int    `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name      string `gorm:"type:varchar(50);comment:名称" json:"name"`
	Code      string `gorm:"type:varchar(30);comment:编码" json:"code"`
	ResetType int    `gorm:"comment:0一次性 1每日 2每周" json:"reset_type"` // 0一次性 1每日 2每周
	SortNo    int    `gorm:"comment:排序编号" json:"sort_no"`
	Status    int    `gorm:"default:1;comment:状态" json:"status"`
}

func (EzfyCfgTaskType) TableName() string { return "ezfy_cfg_task_type" }

type EzfyCfgTask struct {
	ID             int    `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name           string `gorm:"type:varchar(100);comment:名称" json:"name"`
	TaskType       string `gorm:"type:varchar(30);comment:任务类型" json:"task_type"`
	Target         int    `gorm:"comment:目标" json:"target"`
	RewardGold     int64  `gorm:"comment:奖励黄金" json:"reward_gold"`
	RewardFood     int64  `gorm:"comment:奖励粮食" json:"reward_food"`
	RewardSteel    int64  `gorm:"comment:奖励钢铁" json:"reward_steel"`
	RewardOil      int64  `gorm:"comment:奖励石油" json:"reward_oil"`
	RewardRare     int64  `gorm:"comment:奖励稀矿" json:"reward_rare"`
	RewardPrestige int    `gorm:"comment:奖励威望" json:"reward_prestige"`
	SortNo         int    `gorm:"comment:排序编号" json:"sort_no"`
	TypeId         int    `gorm:"comment:类型ID" json:"type_id"`
	Status         int    `gorm:"default:1;comment:状态" json:"status"`
}

func (EzfyCfgTask) TableName() string { return "ezfy_cfg_task" }

// ============ 玩家档案 ============

// EzfyProfile 玩家游戏档案（替代 Java 版挂在 user 表上的声望/阵营字段，保持 ezfy_ 前缀约束）
type EzfyProfile struct {
	ID       uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID   uint   `gorm:"uniqueIndex;comment:用户ID" json:"user_id"`
	Nickname string `gorm:"type:varchar(20);comment:昵称" json:"nickname"`
	Prestige int    `gorm:"default:0;comment:军功声望" json:"prestige"`  // 军功声望
	Camp     int    `gorm:"default:1;comment:1同盟国 2轴心国" json:"camp"` // 1同盟国 2轴心国

	// ★ 游戏ID：与家园ID 解耦，**首次 = 家园ID，之后永不随家园ID变化**
	//   游戏内所有业务交互都以它为准（为「游戏单独运行」预留）。
	GameUID int64 `gorm:"index;comment:游戏用户ID" json:"game_uid"`

	// 当前操作的城市（分城切换用；0/无效时回落到 id 最小的主城）
	CurrentCityId int64 `gorm:"comment:当前操作的城市（分城切换用；0/无效时回落到 id 最小的主城）" json:"current_city_id"`

	// 首次免费次数是否已用掉（0=还能免费一次，1=已用过，之后要消耗道具）
	RenameUsed int `gorm:"comment:改昵称" json:"rename_used"` // 改昵称
	CampUsed   int `gorm:"comment:改阵营" json:"camp_used"`   // 改阵营

	// 军校每日免费刷新次数覆盖（0 = 跟随全局默认，管理端可单独调整）
	RecruitFreeLimit int `gorm:"comment:军校每日免费刷新次数覆盖（0 = 跟随全局默认，管理端可单独调整）" json:"recruit_free_limit"`

	// ★ 钻石：二战风云专属币种，**只能由管理端充值**，玩家端只读余额；
	//   用于购买「钻石道具」（ezfy_cfg_item.price_diamond > 0）。
	Diamond int64 `gorm:"default:0;comment:钻石" json:"diamond"`

	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyProfile) TableName() string { return "ezfy_profile" }

// ============ 运行时表 ============

type EzfyCity struct {
	ID        uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint   `gorm:"index:idx_user;comment:用户ID" json:"user_id"`
	Name      string `gorm:"type:varchar(50);comment:名称" json:"name"`
	X         int    `gorm:"index:idx_xy;comment:X坐标" json:"x"`
	Y         int    `gorm:"index:idx_xy;comment:Y坐标" json:"y"`
	CityLevel int    `gorm:"default:1;comment:市政厅等级" json:"city_level"` // 市政厅等级
	Feelings  int    `gorm:"comment:民心（民心）" json:"feelings"`            // 民心
	Grievance int    `gorm:"default:0;comment:民怨" json:"grievance"`     // 民怨
	TaxRate   int    `gorm:"comment:税率%" json:"tax_rate"`               // 税率%
	Pop       int64  `gorm:"comment:人口" json:"pop"`
	PopMax    int64  `gorm:"comment:人口上限" json:"pop_max"`
	Gold      int64  `gorm:"comment:黄金" json:"gold"`
	Food      int64  `gorm:"comment:粮食" json:"food"`
	Steel     int64  `gorm:"comment:钢铁" json:"steel"`
	Oil       int64  `gorm:"comment:石油" json:"oil"`
	Rare      int64  `gorm:"comment:稀矿" json:"rare"`
	GoldCap   int64  `gorm:"comment:黄金上限" json:"gold_cap"`
	FoodCap   int64  `gorm:"comment:粮食上限" json:"food_cap"`
	SteelCap  int64  `gorm:"comment:钢铁上限" json:"steel_cap"`
	OilCap    int64  `gorm:"comment:石油上限" json:"oil_cap"`
	RareCap   int64  `gorm:"comment:稀矿上限" json:"rare_cap"`
	LastTime  int64  `gorm:"comment:上次资源结算时间戳(ms)" json:"last_time"` // 上次资源结算时间戳(ms)

	// 仓库保护配比(4 项资源的保护额度占比, 合计 ≤ 100; 默认各 25)
	// 每项保护额度 = 仓库等级对应保护总量 × 该项占比 / 100
	WareFood  int `gorm:"comment:仓库粮食" json:"ware_food"`
	WareSteel int `gorm:"comment:仓库钢铁" json:"ware_steel"`
	WareOil   int `gorm:"comment:仓库石油" json:"ware_oil"`
	WareRare  int `gorm:"comment:仓库稀矿" json:"ware_rare"`

	// 调整生产·开工率(0~100, 复刻原版 city/sourceSet.html)
	// 实际产量 = 基础产量 × 开工率 / 100, 降低开工率可减少军队耗粮以外的资源消耗压力
	RateFood  int `gorm:"comment:比率粮食" json:"rate_food"`
	RateSteel int `gorm:"comment:比率钢铁" json:"rate_steel"`
	RateOil   int `gorm:"comment:比率石油" json:"rate_oil"`
	RateRare  int `gorm:"comment:比率稀矿" json:"rate_rare"`

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyCity) TableName() string { return "ezfy_city" }

type EzfyCityBuilding struct {
	ID         uint  `gorm:"primaryKey;comment:主键ID" json:"id"`
	CityId     int64 `gorm:"index:idx_city_building;comment:城市ID" json:"city_id"`
	BuildingId int   `gorm:"comment:建筑ID" json:"building_id"`
	Level      int   `gorm:"comment:0=未建造" json:"level"`                    // 0=未建造
	Status     int   `gorm:"default:0;comment:0空闲 1建造中 2升级中" json:"status"` // 0空闲 1建造中 2升级中
	StartTime  int64 `gorm:"comment:0=一键满级连锁模式" json:"start_time"`          // 0=一键满级连锁模式
	EndTime    int64 `gorm:"comment:结束时间" json:"end_time"`
	// ★ 2026-09-25 用户纠正「一键9级 = 一键升级到 9 级，而不是升级满」：
	//   一键升级的目标等级。StartTime=0 的连锁模式每 10 秒升 1 级，
	//   升到 TargetLevel 就停（0 = 老数据/未指定 → 升到该建筑的上限，保持旧行为）。
	TargetLevel int `gorm:"default:0;comment:一键升级目标等级（0=升到上限）" json:"target_level"`

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyCityBuilding) TableName() string { return "ezfy_city_building" }

type EzfyCityTroop struct {
	ID      uint  `gorm:"primaryKey;comment:主键ID" json:"id"`
	CityId  int64 `gorm:"uniqueIndex:uk_city_troop;comment:城市ID" json:"city_id"`
	TroopId int   `gorm:"uniqueIndex:uk_city_troop;comment:部队ID" json:"troop_id"`
	Count   int64 `gorm:"comment:数量" json:"count"`

	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyCityTroop) TableName() string { return "ezfy_city_troop" }

type EzfyCityTech struct {
	ID      uint  `gorm:"primaryKey;comment:主键ID" json:"id"`
	CityId  int64 `gorm:"uniqueIndex:uk_city_tech;comment:城市ID" json:"city_id"`
	TechId  int   `gorm:"uniqueIndex:uk_city_tech;comment:科技ID" json:"tech_id"`
	Level   int   `gorm:"comment:等级" json:"level"`
	Status  int   `gorm:"default:0;comment:0空闲 1研究中" json:"status"` // 0空闲 1研究中
	EndTime int64 `gorm:"comment:结束时间" json:"end_time"`

	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyCityTech) TableName() string { return "ezfy_city_tech" }

type EzfyTrainQueue struct {
	ID        uint  `gorm:"primaryKey;comment:主键ID" json:"id"`
	CityId    int64 `gorm:"index:idx_city;comment:城市ID" json:"city_id"`
	TroopId   int   `gorm:"comment:部队ID" json:"troop_id"`
	Count     int64 `gorm:"comment:数量" json:"count"`
	Status    int   `gorm:"default:0;comment:0训练中 1待领取 2已领取" json:"status"` // 0训练中 1待领取 2已领取
	StartTime int64 `gorm:"comment:开始时间" json:"start_time"`
	EndTime   int64 `gorm:"comment:结束时间" json:"end_time"`
	// ★ 免费征兵标记：1 = 建这条队列时「征兵消耗资源」开关是关的（没扣任何资源）。
	//   取消训练时据此**不退还**资源 —— 否则玩家可以趁开关关着白嫖排队，
	//   等管理员把开关打开后再取消，凭空换出从没付过的资源。
	//   列名故意用 free_train（不叫 free，避开保留字风险）；无 gorm default 标签
	//   （0 是有意义的值：正常扣费建的队列，GORM 不会跳过零值写入）。
	FreeTrain int `gorm:"comment:免费训练" json:"free"`

	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyTrainQueue) TableName() string { return "ezfy_train_queue" }

type EzfyMapArea struct {
	ID        uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	X         int    `gorm:"uniqueIndex:uk_xy;comment:X坐标" json:"x"`
	Y         int    `gorm:"uniqueIndex:uk_xy;comment:Y坐标" json:"y"`
	AreaType  int    `gorm:"default:0;comment:0空地 1野地(被占) 2寇城 3玩家城 4资源田" json:"area_type"` // 0空地 1野地(被占) 2寇城 3玩家城 4资源田
	Terrain   int    `gorm:"default:1;comment:1平原..8海洋" json:"terrain"`                    // 1平原..8海洋
	Level     int    `gorm:"comment:野地等级" json:"level"`                                    // 野地等级
	OwnerId   int64  `gorm:"comment:占领城市ID" json:"owner_id"`                               // 占领城市ID
	Troops    string `gorm:"type:varchar(2000);comment:部队" json:"troops"`
	Resources string `gorm:"type:varchar(500);comment:资源" json:"resources"`
	Officer   string `gorm:"type:varchar(500);comment:军官" json:"officer"`
	Hp        int64  `gorm:"comment:生命" json:"hp"`
	StartTime int64  `gorm:"comment:寇城复活时间" json:"start_time"` // 寇城复活时间

	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyMapArea) TableName() string { return "ezfy_map_area" }

// EzfyActivity 节日活动(复刻 `参考材料/开发文档/福利.txt` 的活动设计, 原版 Java 未实现)
// Type: 1资源增产 2造兵打折 3建造加速 4研究加速 5声望加成
// Param 为百分比; Status: 0未开启 1进行中; 生效还要求 start_time <= now < end_time
type EzfyActivity struct {
	ID        uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name      string `gorm:"type:varchar(50);comment:名称" json:"name"`
	Type      int    `gorm:"comment:类型" json:"type"`
	Param     int    `gorm:"comment:参数" json:"param"`
	StartTime int64  `gorm:"comment:开始时间" json:"start_time"`
	EndTime   int64  `gorm:"comment:结束时间" json:"end_time"`
	Status    int    `gorm:"default:0;comment:状态" json:"status"`
	Des       string `gorm:"type:varchar(500);comment:描述" json:"des"`

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyActivity) TableName() string { return "ezfy_activity" }

// EzfyMapStar 地图坐标收藏(复刻原版地图页的「收藏列表」)
type EzfyMapStar struct {
	ID     uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID uint   `gorm:"index:idx_star_user;comment:用户ID" json:"user_id"`
	X      int    `gorm:"comment:X坐标" json:"x"`
	Y      int    `gorm:"comment:Y坐标" json:"y"`
	Name   string `gorm:"type:varchar(50);comment:名称" json:"name"`

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyMapStar) TableName() string { return "ezfy_map_star" }

type EzfyOrder struct {
	ID         uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID     uint   `gorm:"index:idx_user;comment:用户ID" json:"user_id"`
	CityId     int64  `gorm:"comment:城市ID" json:"city_id"`
	OrderType  int    `gorm:"comment:1侦查 2掠夺 3征服 4采集 5运输 6增援 7派遣" json:"order_type"` // 1侦查 2掠夺 3征服 4采集 5运输 6增援 7派遣
	TargetType int    `gorm:"comment:1野地 2寇城 3玩家城" json:"target_type"`               // 1野地 2寇城 3玩家城
	TargetX    int    `gorm:"comment:目标X坐标" json:"target_x"`
	TargetY    int    `gorm:"comment:目标Y坐标" json:"target_y"`
	TargetId   int64  `gorm:"comment:目标ID" json:"target_id"`
	Troops     string `gorm:"type:varchar(2000);comment:[{”troopId”:1,”count”:100}]" json:"troops"` // [{"troopId":1,"count":100}]
	Officer    string `gorm:"type:varchar(255);comment:军官" json:"officer"`
	StartTime  int64  `gorm:"comment:开始时间" json:"start_time"`
	ArriveTime int64  `gorm:"comment:Arrive时间" json:"arrive_time"`
	ReturnTime int64  `gorm:"comment:Return时间" json:"return_time"`
	Status     int    `gorm:"index:idx_status;default:0;comment:0行进 1驻守中 2返回 3完成 4阵亡" json:"status"` // 0行进 1驻守中 2返回 3完成 4阵亡
	Result     string `gorm:"type:varchar(3000);comment:返回部队JSON/采集标记" json:"result"`                // 返回部队JSON/采集标记
	// ★ 采集到的资源先记在部队身上（待带回），只有「返航到达」才入城；
	//   容量上限 = 部队各兵种 carry 之和。JSON: {"food":..,"steel":..,"oil":..,"rare":..,"gold":..}
	Carry     string `gorm:"type:varchar(500);comment:携带" json:"carry"`
	Resources string `gorm:"type:varchar(500);comment:资源" json:"resources"`
	OilUsed   int64  `gorm:"comment:石油已用" json:"oil_used"`
	WaitMin   int    `gorm:"comment:宿营分钟数(0~1440), 到达后停留该时长再返航" json:"wait_min"` // 宿营分钟数(0~1440), 到达后停留该时长再返航

	// ★ 指挥室（实时战斗）打完的结果（ezfyBattleResult 的 JSON）。
	//   非空 = 这场仗已经由玩家在指挥室里打完 → processArrive 跳过模拟、直接拿它结算，
	//   战报/掠夺/经验/征服等战后逻辑全部复用，不重复实现一套。
	BattleResult string `gorm:"type:mediumtext;comment:战斗结果" json:"battle_result"`

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyOrder) TableName() string { return "ezfy_order" }

// EzfyBattle 战场（实时指挥室）
//
// ★ 2026-09-22 用户要求「实现指挥功能」（入口：军情 → 军队动态 → [指挥]）：
// 出征部队到达目标后**不立即结算**，而是开一场战场；每回合 30 秒，
// 前 25 秒玩家可下达前进/暂停/后退，后 5 秒锁定并由服务器结算一回合，最多 40 回合。
// 战场结束时把结果写回 ezfy_order.battle_result，再走原有的战后结算逻辑。
//
// 推进方式是**懒结算**：State 里存整场快照，每次请求按「已经过去多少个回合时长」补算，
// 所以不需要常驻定时器；玩家离线时战斗也会自然推进（默认前进，不会卡死）。
type EzfyBattle struct {
	ID         uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	OrderId    int64  `gorm:"index:idx_battle_order;comment:订单ID" json:"order_id"`
	UserID     uint   `gorm:"index:idx_battle_user;comment:用户ID" json:"user_id"`
	CityId     int64  `gorm:"comment:城市ID" json:"city_id"`
	TargetType int    `gorm:"comment:1野地 2寇城 3玩家城" json:"target_type"` // 1野地 2寇城 3玩家城
	TargetId   int64  `gorm:"comment:目标ID" json:"target_id"`
	TargetX    int    `gorm:"comment:目标X坐标" json:"target_x"`
	TargetY    int    `gorm:"comment:目标Y坐标" json:"target_y"`
	TargetName string `gorm:"type:varchar(120);default:'';comment:目标名称" json:"target_name"`
	// Status 1 进行中 / 2 已结束（结果已回写订单，等 processArrive 结算）
	Status int `gorm:"index:idx_battle_status;default:1;comment:状态" json:"status"`
	Round  int `gorm:"default:0;comment:回合" json:"round"`
	// Win 0 未分胜负 / 1 攻方胜 / 2 攻方负
	Win int `gorm:"default:0;comment:Win 0 未分胜负 / 1 攻方胜 / 2 攻方负" json:"win"`
	// AtkCmd 攻方**逐兵种**指令表，JSON: {"1":"advance","3":"hold"}（troopId → advance|hold|retreat）。
	// ★ 用户要求「指挥不是指挥全部，自己带的兵种都能指挥，就是单独指挥」。
	//   没给的兵种回落司令部「兵种战斗配置」；键 0 = 旧格式遗留的「全军统一指令」。
	AtkCmd string `gorm:"type:varchar(500);default:'';comment:攻击指令" json:"atk_cmd"`
	// DefUserID 守方玩家 uid（仅 target_type=3 攻击玩家城时有值，0 = 野地/寇城无玩家守方）。
	// ★ 2026-09-23 用户要求「敌人打自己，自己也能指挥」—— 防守方据此找回并进入战场。
	DefUserID uint `gorm:"index:idx_battle_def;comment:守方用户ID" json:"def_user_id"`
	// DefCmd 守方**逐兵种**指令表，JSON 同 AtkCmd（玩家守城时指挥守军用；AI 为空）。
	DefCmd string `gorm:"type:varchar(500);default:'';comment:防御指令" json:"def_cmd"`
	// State 战场快照（ezfyBattleSnapshot 的 JSON，含双方兵力/位置/加成/日志）
	State string `gorm:"type:mediumtext;comment:状态" json:"-"`
	// RoundStart 本回合开始时间(ms)：过了回合时长就推进一回合
	RoundStart int64 `gorm:"comment:RoundStart 本回合开始时间(ms)：过了回合时长就推进一回合" json:"round_start"`

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyBattle) TableName() string { return "ezfy_battle" }

type EzfyReport struct {
	ID         uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID     uint   `gorm:"index:idx_user;comment:用户ID" json:"user_id"`
	OrderId    int64  `gorm:"comment:订单ID" json:"order_id"`
	ReportType int    `gorm:"comment:1侦察 2掠夺 3征服 4战斗 5采集/派遣 6系统" json:"report_type"` // 1侦察 2掠夺 3征服 4战斗 5采集/派遣 6系统
	Title      string `gorm:"type:varchar(255);comment:标题" json:"title"`
	Content    string `gorm:"type:longtext;comment:内容" json:"content"`
	Detail     string `gorm:"type:longtext;comment:Detail" json:"detail"`
	IsRead     int    `gorm:"default:0;comment:是否已读" json:"is_read"`

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyReport) TableName() string { return "ezfy_report" }

type EzfyWildland struct {
	ID        uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	CityId    int64  `gorm:"index:idx_city;comment:城市ID" json:"city_id"`
	X         int    `gorm:"comment:X坐标" json:"x"`
	Y         int    `gorm:"comment:Y坐标" json:"y"`
	WildType  int    `gorm:"comment:1陆地野地 2海野 3特殊野地" json:"wild_type"` // 1陆地野地 2海野 3特殊野地
	Level     int    `gorm:"comment:等级" json:"level"`
	Gain      string `gorm:"type:varchar(500);comment:获得" json:"gain"`
	Status    int    `gorm:"default:0;comment:0空闲 1采集中" json:"status"` // 0空闲 1采集中
	StartTime int64  `gorm:"comment:开始时间" json:"start_time"`
	EndTime   int64  `gorm:"comment:结束时间" json:"end_time"`

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyWildland) TableName() string { return "ezfy_wildland" }

// EzfyOccupy 占领的玩家城市（城市归入攻击方附属）
type EzfyOccupy struct {
	ID        uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	CityId    int64  `gorm:"comment:城市ID" json:"city_id"`
	CityName  string `gorm:"type:varchar(50);comment:城市名称" json:"city_name"`
	AtkUserId uint   `gorm:"comment:攻方用户ID" json:"atk_user_id"`
	AtkCityId int64  `gorm:"comment:攻击城市ID" json:"atk_city_id"`
	DefUserId uint   `gorm:"comment:守方用户ID" json:"def_user_id"`
	X         int    `gorm:"comment:X坐标" json:"x"`
	Y         int    `gorm:"comment:Y坐标" json:"y"`
	Status    int    `gorm:"default:1;comment:1占领中 2已摧毁(归还)" json:"status"` // 1占领中 2已摧毁(归还)

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyOccupy) TableName() string { return "ezfy_occupy" }

type EzfyWounded struct {
	ID      uint  `gorm:"primaryKey;comment:主键ID" json:"id"`
	CityId  int64 `gorm:"index:idx_city;comment:城市ID" json:"city_id"`
	TroopId int   `gorm:"comment:部队ID" json:"troop_id"`
	Type    int   `gorm:"comment:0伤兵 1逃兵" json:"type"` // 0伤兵 1逃兵
	Count   int64 `gorm:"comment:数量" json:"count"`

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyWounded) TableName() string { return "ezfy_wounded" }

// EzfyWar 宣战记录：宣战后延迟生效，生效后一段时间内可互相掠夺/征服
type EzfyWar struct {
	ID          uint  `gorm:"primaryKey;comment:主键ID" json:"id"`
	AtkUserId   uint  `gorm:"index:idx_pair;comment:攻方用户ID" json:"atk_user_id"`
	DefUserId   uint  `gorm:"index:idx_pair;comment:守方用户ID" json:"def_user_id"`
	Status      int   `gorm:"comment:1宣战待生效 2交战中" json:"status"` // 1宣战待生效 2交战中
	DeclareTime int64 `gorm:"comment:Declare时间" json:"declare_time"`
	EffectTime  int64 `gorm:"comment:效果时间" json:"effect_time"`
	ExpireTime  int64 `gorm:"comment:过期时间" json:"expire_time"`

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyWar) TableName() string { return "ezfy_war" }

type EzfyCorps struct {
	ID           uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name         string `gorm:"type:varchar(20);uniqueIndex:uk_name;comment:名称" json:"name"`
	LeaderUserId uint   `gorm:"comment:首领用户ID" json:"leader_user_id"`
	Notice       string `gorm:"type:varchar(200);comment:公告" json:"notice"`
	MemberCount  int    `gorm:"default:1;comment:Member数量" json:"member_count"`
	// ★ 2026-09-25 用户要求「军团积分」：军团战绩总积分（成员在军团交战期获胜累加，军团商城可查看）
	Points int64 `gorm:"default:0;comment:军团总积分" json:"points"`

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyCorps) TableName() string { return "ezfy_corps" }

type EzfyCorpsMember struct {
	ID       uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	CorpsId  uint   `gorm:"index:idx_corps;comment:军团ID" json:"corps_id"`
	UserId   uint   `gorm:"uniqueIndex:uk_user;comment:用户ID" json:"user_id"`
	IsLeader int    `gorm:"default:0;comment:是否首领" json:"is_leader"`
	Title    string `gorm:"type:varchar(20);comment:标题" json:"title"`
	// ★ 2026-09-25 用户要求「军团商城货币 = 成员个人军团积分」：成员个人战绩积分（商城消费用它扣）
	Points    int64     `gorm:"default:0;comment:个人军团积分" json:"points"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyCorpsMember) TableName() string { return "ezfy_corps_member" }

// EzfyCorpsRelation 军团外交关系（军团长标记；友好/敌对均双向各写一条）
//
// ★ 2026-09-25 用户要求「军团之间可标记友好/敌对，双向记录，友好敌对都可以宣战」。
type EzfyCorpsRelation struct {
	ID            uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	CorpsId       uint      `gorm:"uniqueIndex:uk_pair;comment:军团ID" json:"corps_id"`
	TargetCorpsId uint      `gorm:"uniqueIndex:uk_pair;comment:目标军团ID" json:"target_corps_id"`
	Type          int       `gorm:"comment:1友好 2敌对" json:"type"` // 1友好 2敌对
	CreatedAt     time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt     time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyCorpsRelation) TableName() string { return "ezfy_corps_relation" }

// EzfyCorpsWar 军团宣战记录：宣战后 12 小时生效、48 小时整场结束（生效窗口 = 第 12~48 小时，共 36 小时）
//
// ★ 2026-09-25 用户要求：生效期间双方**军团成员之间**可互相掠夺/征服，无需个人宣战；
// 掠夺胜 +10、征服胜 +20 军团战绩（记到获胜方所属军团 AtkPoint/DefPoint 与攻击者个人积分）。
type EzfyCorpsWar struct {
	ID           uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	AtkCorpsId   uint   `gorm:"index:idx_pair;comment:攻方军团ID" json:"atk_corps_id"`
	DefCorpsId   uint   `gorm:"index:idx_pair;comment:守方军团ID" json:"def_corps_id"`
	AtkCorpsName string `gorm:"type:varchar(20);comment:攻方军团名快照" json:"atk_corps_name"`
	DefCorpsName string `gorm:"type:varchar(20);comment:守方军团名快照" json:"def_corps_name"`
	AtkUserId    uint   `gorm:"comment:发起宣战的军团长用户ID" json:"atk_user_id"`
	Status       int    `gorm:"comment:1待生效 2交战中 3已结束" json:"status"` // 1待生效 2交战中 3已结束
	AtkPoint     int64  `gorm:"default:0;comment:攻方本次战绩" json:"atk_point"`
	DefPoint     int64  `gorm:"default:0;comment:守方本次战绩" json:"def_point"`
	DeclareTime  int64  `gorm:"comment:宣战时间" json:"declare_time"`
	EffectTime   int64  `gorm:"comment:生效时间" json:"effect_time"`
	ExpireTime   int64  `gorm:"comment:整场结束时间" json:"expire_time"`
	EndTime      int64  `gorm:"comment:实际结束时间(0=未结束)" json:"end_time"`

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyCorpsWar) TableName() string { return "ezfy_corps_war" }

// EzfyCorpsMall 军团商城商品：货币 = 成员个人军团积分
//
// ★ 2026-09-25 用户要求：商品分两类 —— 资源包（food/steel/oil/rare/gold）与道具
// （复用现有游戏道具配置 EzfyCfgItem，「道具池」= 现有道具表，发放走现有 addItem）。
type EzfyCorpsMall struct {
	ID         uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	Kind       int    `gorm:"comment:1资源包 2道具" json:"kind"` // 1资源包 2道具
	Name       string `gorm:"type:varchar(40);comment:名称" json:"name"`
	Food       int64  `gorm:"default:0;comment:粮食" json:"food"`
	Steel      int64  `gorm:"default:0;comment:钢铁" json:"steel"`
	Oil        int64  `gorm:"default:0;comment:石油" json:"oil"`
	Rare       int64  `gorm:"default:0;comment:稀矿" json:"rare"`
	Gold       int64  `gorm:"default:0;comment:黄金" json:"gold"`
	ItemId     int    `gorm:"default:0;comment:道具配置ID" json:"item_id"`
	ItemCount  int    `gorm:"default:0;comment:道具数量" json:"item_count"`
	Price      int64  `gorm:"default:0;comment:个人军团积分单价" json:"price"`
	LimitCount int    `gorm:"default:0;comment:每人限购(0=不限)" json:"limit_count"`
	Stock      int    `gorm:"default:-1;comment:总库存(-1=不限)" json:"stock"`
	Sold       int    `gorm:"default:0;comment:已售数量" json:"sold"`
	Sort       int    `gorm:"default:0;comment:排序" json:"sort"`
	Enabled    int    `gorm:"default:1;comment:1上架 0下架" json:"enabled"`

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyCorpsMall) TableName() string { return "ezfy_corps_mall" }

// EzfyCorpsMallLog 军团商城购买记录（每人限购统计用它按 user+mall 聚合）
type EzfyCorpsMallLog struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	CorpsId   uint      `gorm:"comment:军团ID" json:"corps_id"`
	UserId    uint      `gorm:"index:idx_user_mall;comment:用户ID" json:"user_id"`
	MallId    uint      `gorm:"index:idx_user_mall;comment:商品ID" json:"mall_id"`
	Kind      int       `gorm:"comment:1资源包 2道具" json:"kind"`
	Count     int       `gorm:"comment:购买数量" json:"count"`
	Cost      int64     `gorm:"comment:消耗个人军团积分" json:"cost"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyCorpsMallLog) TableName() string { return "ezfy_corps_mall_log" }

type EzfyCorpsChat struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	CorpsId   uint      `gorm:"index:idx_corps;comment:军团ID" json:"corps_id"`
	UserId    uint      `gorm:"comment:用户ID" json:"user_id"`
	UserName  string    `gorm:"type:varchar(20);comment:用户名称" json:"user_name"`
	Content   string    `gorm:"type:varchar(200);comment:内容" json:"content"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyCorpsChat) TableName() string { return "ezfy_corps_chat" }

// EzfyItem 玩家背包
type EzfyItem struct {
	ID     uint `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId uint `gorm:"uniqueIndex:uk_user_item;comment:用户ID" json:"user_id"`
	CfgId  int  `gorm:"uniqueIndex:uk_user_item;comment:配置ID" json:"cfg_id"`
	Count  int  `gorm:"comment:数量" json:"count"`
}

func (EzfyItem) TableName() string { return "ezfy_item" }

type EzfySign struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId    uint      `gorm:"comment:用户ID" json:"user_id"`
	SignDate  string    `gorm:"type:varchar(10);uniqueIndex:uk_user_date;comment:签到日期" json:"sign_date"`
	SignCount int       `gorm:"comment:连续签到天数" json:"sign_count"` // 连续签到天数
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfySign) TableName() string { return "ezfy_sign" }

// EzfyGift 礼包领取记录
type EzfyGift struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId    uint      `gorm:"comment:用户ID" json:"user_id"`
	GiftType  string    `gorm:"type:varchar(20);comment:礼物类型" json:"gift_type"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyGift) TableName() string { return "ezfy_gift" }

// EzfyCityEffect 城市效果：1增产 2免战
type EzfyCityEffect struct {
	ID         uint  `gorm:"primaryKey;comment:主键ID" json:"id"`
	CityId     int64 `gorm:"uniqueIndex:uk_city_effect;comment:城市ID" json:"city_id"`
	EffectType int   `gorm:"uniqueIndex:uk_city_effect;comment:效果类型" json:"effect_type"`
	Param1     int   `gorm:"comment:参数1" json:"param1"`
	UntilTime  int64 `gorm:"comment:截止时间" json:"until_time"`
}

func (EzfyCityEffect) TableName() string { return "ezfy_city_effect" }

// EzfyCityTarget 司令部兵种战斗配置：优先攻击目标/前进停止
type EzfyCityTarget struct {
	ID             uint  `gorm:"primaryKey;comment:主键ID" json:"id"`
	CityId         int64 `gorm:"uniqueIndex:uk_city_troop;comment:城市ID" json:"city_id"`
	TroopId        int   `gorm:"uniqueIndex:uk_city_troop;comment:部队ID" json:"troop_id"`
	AtkTargetTroop int   `gorm:"comment:0=最近目标" json:"atk_target_troop"` // 0=最近目标
	AtkMove        int   `gorm:"comment:攻击移动" json:"atk_move"`
	DefTargetTroop int   `gorm:"comment:防御目标部队" json:"def_target_troop"`
	DefMove        int   `gorm:"comment:防御移动" json:"def_move"`

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyCityTarget) TableName() string { return "ezfy_city_target" }

// EzfyTask 玩家任务进度
type EzfyTask struct {
	ID         uint       `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId     uint       `gorm:"uniqueIndex:uk_user_task;comment:用户ID" json:"user_id"`
	CfgId      int        `gorm:"uniqueIndex:uk_user_task;comment:配置ID" json:"cfg_id"`
	Current    int        `gorm:"comment:当前" json:"current"`
	Status     int        `gorm:"comment:0进行中 1可领取 2已领取" json:"status"` // 0进行中 1可领取 2已领取
	TaskDate   string     `gorm:"type:varchar(10);comment:任务日期" json:"task_date"`
	CreatedAt  time.Time  `gorm:"comment:创建时间" json:"created_at"`
	FinishTime *time.Time `gorm:"comment:Finish时间" json:"finish_time"`
}

func (EzfyTask) TableName() string { return "ezfy_task" }

// EzfyNotice 公告/通知（0=全员公告）
type EzfyNotice struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId    uint      `gorm:"index:idx_user;comment:用户ID" json:"user_id"`
	Title     string    `gorm:"type:varchar(100);comment:标题" json:"title"`
	Content   string    `gorm:"type:varchar(2000);comment:内容" json:"content"`
	IsTop     int       `gorm:"default:0;comment:是否置顶" json:"is_top"`
	IsRead    int       `gorm:"default:0;comment:是否已读" json:"is_read"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyNotice) TableName() string { return "ezfy_notice" }

// EzfyChat 游戏聊天（复刻原版 chatB?type=：1公共 2军团 4系统）
type EzfyChat struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId    uint      `gorm:"comment:用户ID" json:"user_id"`
	UserName  string    `gorm:"type:varchar(20);comment:用户名称" json:"user_name"`
	Content   string    `gorm:"type:varchar(200);comment:内容" json:"content"`
	Channel   int       `gorm:"default:1;index:idx_channel;comment:1公共 2军团 4系统" json:"channel"` // 1公共 2军团 4系统
	TalkType  int       `gorm:"default:1;comment:0系统(只读) 1玩家" json:"talk_type"`                 // 0系统(只读) 1玩家
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyChat) TableName() string { return "ezfy_chat" }

// EzfyExchange 交易所挂单：玩家卖资源换黄金；系统挂单可定价黄金或钻石
type EzfyExchange struct {
	ID         uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	SellerId   uint   `gorm:"index:idx_seller;comment:卖家ID" json:"seller_id"`
	SellerName string `gorm:"type:varchar(20);comment:卖家名称" json:"seller_name"`
	EsType     int    `gorm:"comment:1粮 2钢 3油 4稀矿" json:"es_type"` // 1粮 2钢 3油 4稀矿
	EsCount    int64  `gorm:"comment:Es数量" json:"es_count"`
	TotalPrice int64  `gorm:"comment:总价(货币见 Currency)" json:"total_price"`                  // 总价(货币见 Currency)
	Status     int    `gorm:"index:idx_status;default:0;comment:0在售 1成交 2下架" json:"status"` // 0在售 1成交 2下架
	BuyerId    uint   `gorm:"comment:买家ID" json:"buyer_id"`
	IsSystem   int    `gorm:"default:0;comment:是否1系统挂单" json:"is_system"` // 1系统挂单
	// ★ 计价货币：1 黄金 2 钻石。
	//   用户规则：**玩家挂单只能用黄金**；系统挂单（管理端新增）可以用黄金或钻石定价。
	Currency  int       `gorm:"default:1;comment:货币" json:"currency"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyExchange) TableName() string { return "ezfy_exchange" }

// EzfyExchangeTemplate 交易行挂单模板（管理端「维护模版」tab 维护）
//
// ★ 2026-09-24 用户要求：资源包模板原来硬编码在前端，选中还有「触发全选」的 bug；
//
//	改成数据库里的模板表，管理端可增删改；新增系统挂单时选模板只是快速填充，
//	资源数量(EsCount)上架前**仍可二次修改**，挂单数量(Repeat)支持一次挂多单。
type EzfyExchangeTemplate struct {
	ID         uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name       string    `gorm:"type:varchar(50);comment:模板名" json:"name"`
	EsType     int       `gorm:"comment:1粮 2钢 3油 4稀矿" json:"es_type"`
	EsCount    int64     `gorm:"comment:资源数量" json:"es_count"`
	TotalPrice int64     `gorm:"comment:总价" json:"total_price"`
	Currency   int       `gorm:"default:1;comment:1黄金 2钻石" json:"currency"`
	SortNo     int       `gorm:"default:0;comment:排序" json:"sort_no"`
	CreatedAt  time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt  time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyExchangeTemplate) TableName() string { return "ezfy_exchange_tpl" }

// ============ 军官/学院系统（复刻 stzb-fk：军校/参谋部/技能/装备/俘虏/任命） ============

// EzfyCfgGeneral 军官池（源自 inithebing.sql cfg_general 31 条名将 + 管理端可新增普通军官）
//
// ★ 2026-09-22 用户要求：军官分两类，**都在这张池子里**，由管理端统一维护：
//   - kind=1 普通军官：军校招募/刷新时**从池子里按权重抽**（不再纯随机生成）
//   - kind=2 名将    ：只由管理端发放，不进招募池
//
// Level 的语义 = 该军官的**等级上限**（名将就是 110~150，普通军官一般也填 150）。
// Military/Logistics/Learning = 该军官的**原始属性**（玩家实例的 base_*，升级加点只改实例）。
type EzfyCfgGeneral struct {
	ID           int    `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name         string `gorm:"type:varchar(100);comment:名称" json:"name"`
	Level        int    `gorm:"comment:等级上限(名将决定招募费用=等级×1000黄金)" json:"level"` // 等级上限(名将决定招募费用=等级×1000黄金)
	Military     int    `gorm:"comment:军事" json:"military"`
	Logistics    int    `gorm:"comment:后勤" json:"logistics"`
	Learning     int    `gorm:"comment:学识" json:"learning"`
	Star         int    `gorm:"default:5;comment:星级" json:"star"`
	Source       string `gorm:"type:varchar(255);comment:来源" json:"source"`
	GetCondition string `gorm:"type:varchar(255);comment:GetCondition" json:"get_condition"`
	Skill        string `gorm:"type:varchar(500);comment:技能" json:"skill"`
	Des          string `gorm:"type:varchar(500);comment:描述" json:"des"`
	// Recruit 1=可招募 0=停用
	// ★ 2026-09-26 默认值改为 0（线上名将全部 recruit=0）：名将只由管理端发放，
	//   「可招募」只对 kind=1 的军官池生效（kind=2 无功能影响）。
	Recruit int `gorm:"default:0;comment:1=可招募 0=停用" json:"recruit"`
	// ★ 2026-09-22 新增：1=普通军官（军校池） 2=名将（管理端发放）
	Kind int `gorm:"default:2;comment:种类" json:"kind"`
	// ★ 普通军官在军校刷新时的抽取权重（越大越容易刷到），名将不用
	Weight int `gorm:"default:100;comment:权重" json:"weight"`
}

func (EzfyCfgGeneral) TableName() string { return "ezfy_cfg_general" }

// EzfyCfgSkill 技能配置（15 个，每名军官最多学 3 个）
type EzfyCfgSkill struct {
	ID     int    `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name   string `gorm:"type:varchar(50);comment:名称" json:"name"`
	Effect string `gorm:"type:varchar(100);comment:效果" json:"effect"`
	Type   int    `gorm:"default:1;comment:1攻击类 2防御类 3辅助类" json:"type"`           // 1攻击类 2防御类 3辅助类
	Des    string `gorm:"type:text;comment:完整技能说明(参考 skill.html, 较长)" json:"des"` // 完整技能说明(参考 skill.html, 较长)
}

func (EzfyCfgSkill) TableName() string { return "ezfy_cfg_skill" }

// EzfyCfgEquipment 装备池（管理端维护：可新增装备/套装件、定价、上架商城）
type EzfyCfgEquipment struct {
	ID        int    `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name      string `gorm:"type:varchar(50);comment:名称" json:"name"`
	Type      string `gorm:"type:varchar(20);default:武器;comment:武器/防具/饰品/珠宝/套装" json:"type"` // 武器/防具/饰品/珠宝/套装
	Tier      int    `gorm:"default:1;comment:1初级 2中级 3高级 4特殊" json:"tier"`                  // 1初级 2中级 3高级 4特殊
	Military  int    `gorm:"comment:军事" json:"military"`
	Logistics int    `gorm:"comment:后勤" json:"logistics"`
	Learning  int    `gorm:"comment:学识" json:"learning"`
	Level     int    `gorm:"default:1;comment:穿戴等级需求" json:"level"` // 穿戴等级需求
	Des       string `gorm:"type:varchar(200);comment:描述" json:"des"`
	// ★ 2026-09-22 新增（装备池「大池子」+ 套装 + 商城上架）
	// Slot 穿戴部位（头/肩/胸/腰/手/足/饰品/挂件/勋章/左槽/右槽/武器/防具/珠宝…）
	// 留空则回落到 Type（老数据兼容）。同部位唯一靠它判定。
	Slot string `gorm:"type:varchar(20);default:'';comment:部位" json:"slot"`
	// SetId 所属套装（ezfy_cfg_equip_set.id），0=非套装
	SetId int `gorm:"default:0;comment:SetId 所属套装（ezfy_cfg_equip_set.id），0=非套装" json:"set_id"`
	// PriceGold / PriceDiamond 商城售价，0 = 该渠道不卖
	PriceGold    int64 `gorm:"default:0;comment:PriceGold / PriceDiamond 商城售价，0 = 该渠道不卖" json:"price_gold"`
	PriceDiamond int64 `gorm:"default:0;comment:钻石价格" json:"price_diamond"`
	// Stock 商城库存，-1 = 无上限；0 = 已售罄
	Stock int `gorm:"default:-1;comment:Stock 商城库存，-1 = 无上限；0 = 已售罄" json:"stock"`
	// Effect 额外效果说明（如「攻速+40%」），仅展示 + 战斗加成文本
	Effect string `gorm:"type:varchar(200);default:'';comment:Effect 额外效果说明（如「攻速+40%」），仅展示 + 战斗加成文本" json:"effect"`
	// ===== ★ 2026-09-22 第二批：参照 装备距离伤害表.xlsx =====
	//
	// 原版军官装备是 **11 个部位 + 6 项战斗属性（全部是百分比加成）**，
	// 集齐同一系列 11 件就是一套「套装」（套装属性 = 各件之和，另外可再配额外加成）。
	//
	// Series 系列名（革命者 / 渡鸦之魂 / 黑色幽灵 / 巨匠 / 青天白日 / 赤色锤镰 / 空=普通散件）
	Series string `gorm:"type:varchar(30);default:'';comment:系列" json:"series"`
	// Enhance / EnhanceMax 强化等级与上限（表里写「装备+20」，数值就是 +20 时的值）
	Enhance    int `gorm:"default:0;comment:强化等级" json:"enhance"`
	EnhanceMax int `gorm:"default:20;comment:强化上限" json:"enhance_max"`
	// 六项战斗属性，单位「百分点」：125 = 伤害+125%
	Dmg     int `gorm:"default:0;comment:伤害加成%" json:"dmg"`        // 伤害加成%
	Def     int `gorm:"default:0;comment:防御加成%" json:"def"`        // 防御加成%
	Hp      int `gorm:"default:0;comment:生命加成%" json:"hp"`         // 生命加成%
	Move    int `gorm:"default:0;comment:移动距离加成%" json:"move"`     // 移动距离加成%
	Crit    int `gorm:"default:0;comment:暴击几率加成%" json:"crit"`     // 暴击几率加成%
	CritDmg int `gorm:"default:0;comment:暴击伤害加成%" json:"crit_dmg"` // 暴击伤害加成%
}

func (EzfyCfgEquipment) TableName() string { return "ezfy_cfg_equipment" }

// EquipSlot 实际穿戴部位（Slot 为空时回落到 Type，兼容老数据）
func (e EzfyCfgEquipment) EquipSlot() string {
	if e.Slot != "" {
		return e.Slot
	}
	return e.Type
}

// EzfyCfgEquipSet 军官装备套装（穿戴同套 N 件触发套装加成）
//
// ★ 2026-09-22 用户要求：军官穿的装备有套装，玩家用黄金或钻石在商城购买。
type EzfyCfgEquipSet struct {
	ID        int    `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name      string `gorm:"type:varchar(100);comment:名称" json:"name"`
	Parts     int    `gorm:"default:3;comment:触发套装效果所需件数" json:"parts"` // 触发套装效果所需件数
	Military  int    `gorm:"default:0;comment:军事" json:"military"`
	Logistics int    `gorm:"default:0;comment:后勤" json:"logistics"`
	Learning  int    `gorm:"default:0;comment:学识" json:"learning"`
	Effect    string `gorm:"type:varchar(300);default:'';comment:额外效果说明(展示)" json:"effect"` // 额外效果说明(展示)
	Des       string `gorm:"type:varchar(300);default:'';comment:描述" json:"des"`
	// ★ 2026-09-22：系列套装（11 件）的六项战斗属性加成。
	// 注意：装备距离伤害表里的「套装属性」= 11 件之和，**各件本身已经加了**，
	// 这里填的是**额外**加成（默认 0，管理端可加）。
	Series  string `gorm:"type:varchar(30);default:'';comment:系列" json:"series"`
	Dmg     int    `gorm:"default:0;comment:伤害" json:"dmg"`
	Def     int    `gorm:"default:0;comment:防御" json:"def"`
	Hp      int    `gorm:"default:0;comment:生命" json:"hp"`
	Move    int    `gorm:"default:0;comment:移动" json:"move"`
	Crit    int    `gorm:"default:0;comment:暴击" json:"crit"`
	CritDmg int    `gorm:"default:0;comment:暴击伤害" json:"crit_dmg"`
}

func (EzfyCfgEquipSet) TableName() string { return "ezfy_cfg_equip_set" }

// ============ 宝箱（2026-09-22 用户要求）============
//
// 「有的套装是开宝箱概率得到的，看看怎么引入宝箱，宝箱一般用钻石买。」
//
// 玩法：宝箱用钻石（或黄金）买 → 开箱按奖池权重随机出一件奖品 →
// 装备直接进玩家背包（可就地穿到军官身上），道具进道具背包。

// EzfyCfgChest 宝箱配置（管理端维护）
type EzfyCfgChest struct {
	ID           int    `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name         string `gorm:"type:varchar(100);comment:名称" json:"name"`
	PriceDiamond int64  `gorm:"default:0;comment:钻石价，0=不卖钻石" json:"price_diamond"` // 钻石价，0=不卖钻石
	PriceGold    int64  `gorm:"default:0;comment:黄金价，0=不卖黄金" json:"price_gold"`    // 黄金价，0=不卖黄金
	// Stock 库存：-1 = 无上限，0 = 已售罄
	Stock int `gorm:"default:-1;comment:Stock 库存：-1 = 无上限，0 = 已售罄" json:"stock"`
	// OpenMax 单次最多开几个（防误点把钻石全花了）
	OpenMax int `gorm:"default:10;comment:OpenMax 单次最多开几个（防误点把钻石全花了）" json:"open_max"`
	// Enabled 是否上架：1 上架 / 0 下架（管理端用 map 写，0 能写进去）
	Enabled int    `gorm:"default:1;comment:已启用" json:"enabled"`
	SortNo  int    `gorm:"default:0;comment:排序编号" json:"sort_no"`
	Des     string `gorm:"type:varchar(300);default:'';comment:描述" json:"des"`
	// Effect 奖池说明（展示用，如「必出军官装备一件」）
	Effect string `gorm:"type:varchar(300);default:'';comment:Effect 奖池说明（展示用，如「必出军官装备一件」）" json:"effect"`
}

func (EzfyCfgChest) TableName() string { return "ezfy_cfg_chest" }

// EzfyCfgChestItem 宝箱奖池（一条 = 一个奖品 + 权重）
type EzfyCfgChestItem struct {
	ID      int `gorm:"primaryKey;comment:主键ID" json:"id"`
	ChestId int `gorm:"index:idx_chest;comment:ChestID" json:"chest_id"`
	// Kind 奖品类型：1=装备(ezfy_cfg_equipment) 2=道具(ezfy_cfg_item) 3=资源
	Kind  int `gorm:"default:1;comment:种类" json:"kind"`
	RefId int `gorm:"default:0;comment:装备/道具的 cfg_id；资源时填 0" json:"ref_id"` // 装备/道具的 cfg_id；资源时填 0
	Count int `gorm:"default:1;comment:数量" json:"count"`
	// Weight 权重（越大越容易抽到）；全部为 0 时按等概率
	Weight int `gorm:"default:100;comment:Weight 权重（越大越容易抽到）；全部为 0 时按等概率" json:"weight"`
	// Quality 展示用品质标签（普通/稀有/史诗/传说），纯展示
	Quality string `gorm:"type:varchar(20);default:'';comment:Quality 展示用品质标签（普通/稀有/史诗/传说），纯展示" json:"quality"`
	Des     string `gorm:"type:varchar(200);default:'';comment:描述" json:"des"`
}

func (EzfyCfgChestItem) TableName() string { return "ezfy_cfg_chest_item" }

// EzfyCfgScheme 计谋配置（2026-09-22 用户要求）
//
// 「信号弹也是道具，可以黄金、钻石购买，加上，用于计谋消耗。」
//
// 原版 acade/scheme.html 有 12 条计谋，每条消耗不同数量的信号弹。
// 这里做成管理端可维护的配置表，不再写死在前端。
type EzfyCfgScheme struct {
	ID   int    `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name string `gorm:"type:varchar(50);comment:名称" json:"name"`
	Des  string `gorm:"type:varchar(500);default:'';comment:描述" json:"des"`
	// Bullet 发动一次消耗几个信号弹
	Bullet int `gorm:"default:1;comment:Bullet 发动一次消耗几个信号弹" json:"bullet"`
	// Kind 计谋类型：
	//   0 = 纯说明（原版就是「需要进入相应界面才可以使用」，这里只做消耗 + 战报记录）
	//   1 = 先发制人（使双方立即进入可战争状态 N 分钟）
	Kind int `gorm:"default:0;comment:种类" json:"kind"`
	// WarMinutes Kind=1 时的可战争时长（分钟）；实际还会被军官学识夹一次
	WarMinutes int `gorm:"default:60;comment:WarMinutes Kind=1 时的可战争时长（分钟）；实际还会被军官学识夹一次" json:"war_minutes"`
	// WarMaxMinutes 可战争时长上限（分钟，原版 6 小时 = 360）
	WarMaxMinutes int `gorm:"default:360;comment:WarMaxMinutes 可战争时长上限（分钟，原版 6 小时 = 360）" json:"war_max_minutes"`
	Enabled       int `gorm:"default:1;comment:已启用" json:"enabled"`
	SortNo        int `gorm:"default:0;comment:排序编号" json:"sort_no"`
}

func (EzfyCfgScheme) TableName() string { return "ezfy_cfg_scheme" }

// EzfyOfficer 玩家拥有的军官实例（从军官池 ezfy_cfg_general 复制而来）
//
// ★ 2026-09-22 用户要求：**加点与升级只影响玩家自己的军官实例，绝不回写军官池**。
//   - Base* = 从池子复制来的**原始属性**（重修书洗点后回退到这个值）
//   - Military/Logistics/Learning = **当前属性** = Base* + 玩家加点 + 装备/套装加成另算
//   - FreePoints = 还没分配的属性点，每升 1 级 +1
type EzfyOfficer struct {
	ID        uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	CityId    int64  `gorm:"index:idx_city;comment:城市ID" json:"city_id"`
	GeneralId int    `gorm:"comment:GeneralID" json:"general_id"`
	Name      string `gorm:"type:varchar(100);comment:名称" json:"name"`
	Star      int    `gorm:"default:1;comment:星级" json:"star"`
	Level     int    `gorm:"default:1;comment:等级" json:"level"`
	Exp       int64  `gorm:"comment:经验" json:"exp"`
	Military  int    `gorm:"comment:军事" json:"military"`
	Logistics int    `gorm:"comment:后勤" json:"logistics"`
	Learning  int    `gorm:"comment:学识" json:"learning"`
	Loyalty   int    `gorm:"default:100;comment:0-100，归零离职" json:"loyalty"`      // 0-100，归零离职
	Skill     string `gorm:"type:varchar(500);comment:技能名 JSON 数组" json:"skill"` // 技能名 JSON 数组
	// ★ Equipment 存的是已穿戴装备的 JSON 数组，**一条 ~200 字符**，
	//   11 件套（军官装备）就要 ~2200 字符 —— 用 varchar 很容易被 MySQL
	//   **静默截断/报 1406**（实测 varchar(500) 只存进 4 件，玩家看到「穿了但没效果」）。
	//   直接上 text，别再算边界了。
	Equipment  string    `gorm:"type:text;comment:Equipment" json:"equipment"`
	Position   int       `gorm:"default:0;comment:0无 1市长 2城守" json:"position"` // 0无 1市长 2城守
	Status     int       `gorm:"default:0;comment:0在职 1出征中 2被俘" json:"status"` // 0在职 1出征中 2被俘
	IsCaptive  int       `gorm:"default:0;comment:是否Captive" json:"is_captive"`
	UpdateTime time.Time `gorm:"comment:更新时间" json:"update_time"`
	// ★ 2026-09-22 新增：原始属性 + 可用属性点
	//
	// 带 default:0 是为了让 AutoMigrate 建出 `NOT NULL DEFAULT 0` 的列
	// （不带 default 时建出来是 NULL，老行会读到 NULL → JSON 里出现 null）。
	BaseMilitary  int `gorm:"default:0;comment:基础军事" json:"base_military"`
	BaseLogistics int `gorm:"default:0;comment:基础后勤" json:"base_logistics"`
	BaseLearning  int `gorm:"default:0;comment:基础学识" json:"base_learning"`
	FreePoints    int `gorm:"default:0;comment:免费点数" json:"free_points"`
}

func (EzfyOfficer) TableName() string { return "ezfy_officer" }

// EzfyEquipment 玩家装备背包（野地掉宝 / 商城购买入库，可穿戴到军官）
type EzfyEquipment struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId    uint      `gorm:"index:idx_user;comment:用户ID" json:"user_id"`
	CityId    int64     `gorm:"comment:城市ID" json:"city_id"`
	CfgId     int       `gorm:"comment:配置ID" json:"cfg_id"`
	Name      string    `gorm:"type:varchar(50);comment:名称" json:"name"`
	Type      string    `gorm:"type:varchar(20);comment:类型" json:"type"`
	Tier      int       `gorm:"comment:品质档位" json:"tier"`
	Military  int       `gorm:"comment:军事" json:"military"`
	Logistics int       `gorm:"comment:后勤" json:"logistics"`
	Learning  int       `gorm:"comment:学识" json:"learning"`
	Level     int       `gorm:"comment:等级" json:"level"`
	OfficerId int64     `gorm:"index:idx_officer;comment:0=未穿戴" json:"officer_id"` // 0=未穿戴
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	// ★ 2026-09-22 新增：穿戴部位 + 所属套装（从装备池复制，供套装效果判定）
	Slot  string `gorm:"type:varchar(20);default:'';comment:部位" json:"slot"`
	SetId int    `gorm:"default:0;comment:套装ID" json:"set_id"`
	// ★ 六项战斗属性（从装备池复制，直接进战斗计算）
	Series  string `gorm:"type:varchar(30);default:'';comment:系列" json:"series"`
	Enhance int    `gorm:"default:0;comment:强化等级" json:"enhance"`
	Dmg     int    `gorm:"default:0;comment:伤害" json:"dmg"`
	Def     int    `gorm:"default:0;comment:防御" json:"def"`
	Hp      int    `gorm:"default:0;comment:生命" json:"hp"`
	Move    int    `gorm:"default:0;comment:移动" json:"move"`
	Crit    int    `gorm:"default:0;comment:暴击" json:"crit"`
	CritDmg int    `gorm:"default:0;comment:暴击伤害" json:"crit_dmg"`
}

func (EzfyEquipment) TableName() string { return "ezfy_equipment" }

// EquipSlot 实际穿戴部位（Slot 为空时回落到 Type，兼容老数据）
func (e EzfyEquipment) EquipSlot() string {
	if e.Slot != "" {
		return e.Slot
	}
	return e.Type
}

// EzfySlotCanon 装备部位别名归一（2026-09-23 用户反馈「同部位能穿多件」）
//
// ★ 根本原因：不同的套装对同一个身体部位用了**不同的字符串**——「头盔」和「头部」
//
//	都指头、「胸甲」和「胸部」都指胸、「手套/左手/手部」都指手…… 只做**精确字符串**
//	判重时，玩家能同时穿「传说英雄[头盔]」和「赤色锤镰[头部]」两件头装 → 同部位穿了两件。
//	这里把所有同名部位的书写统一成一个规范词，判重和落库都走它，才能真正做到「同部位唯一」。
//	（函数放 model 包是为了 handler 与 seed 两处共用同一份归一逻辑，避免各写各的跑偏）
func EzfySlotCanon(s string) string {
	switch s {
	case "头盔":
		return "头部"
	case "护肩":
		return "肩部"
	case "胸甲":
		return "胸部"
	case "手套", "左手":
		return "手部"
	case "战靴":
		return "足部"
	case "腰带":
		return "腰部"
	}
	return s
}

// EzfyRecruit 军校每日候选名将 / 刷新次数（每日 0 点重置，限刷 5 次）
type EzfyRecruit struct {
	ID           uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId       uint      `gorm:"uniqueIndex:uk_role_date;comment:用户ID" json:"user_id"`
	RecruitDate  string    `gorm:"type:varchar(10);uniqueIndex:uk_role_date;comment:招募日期" json:"recruit_date"`
	RefreshCount int       `gorm:"comment:Refresh数量" json:"refresh_count"`
	Candidates   string    `gorm:"type:text;comment:JSON: 随机军官候选" json:"candidates"` // JSON: 随机军官候选
	CreatedAt    time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt    time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyRecruit) TableName() string { return "ezfy_recruit" }
