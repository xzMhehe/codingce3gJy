package model

import "time"

// 二战风云（ezfy）—— 复刻 stzb-fk「二战风云」，全部数据表使用 ezfy_ 前缀

// ============ 配置表（seed 幂等写入） ============

type EzfyCfgBuilding struct {
	ID          int    `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"type:varchar(50)" json:"name"`
	Type        int    `gorm:"default:0" json:"type"` // 1资源 2军事 3城防 4市政
	MaxLevel    int    `gorm:"default:1" json:"max_level"`
	UniqueFlag  int    `gorm:"default:0" json:"unique_flag"` // 市政厅=1
	CanDelete   int    `json:"can_delete"`
	PreBuilding string `gorm:"type:varchar(255)" json:"pre_building"`
	Des         string `gorm:"type:varchar(500)" json:"des"`
}

func (EzfyCfgBuilding) TableName() string { return "ezfy_cfg_building" }

type EzfyCfgBuildingLevel struct {
	ID         int    `gorm:"primaryKey" json:"id"`
	BuildingId int    `json:"building_id"`
	Level      int    `json:"level"`
	Pop        int    `json:"pop"`
	Food       int64  `json:"food"`
	Steel      int64  `json:"steel"`
	Oil        int64  `json:"oil"`
	Rare       int64  `json:"rare"`
	Gold       int64  `json:"gold"`
	BuildTime  int    `json:"build_time"` // 秒
	Capacity   int64  `json:"capacity"`
	Effect     string `gorm:"type:varchar(500)" json:"effect"`
}

func (EzfyCfgBuildingLevel) TableName() string { return "ezfy_cfg_building_level" }

type EzfyCfgTroop struct {
	ID          int    `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"type:varchar(50)" json:"name"`
	NameAxis    string `gorm:"type:varchar(50)" json:"name_axis"` // 轴心国名称
	NameAlly    string `gorm:"type:varchar(50)" json:"name_ally"` // 同盟国名称
	Type        int    `gorm:"default:2" json:"type"`             // 1海军 2陆军 3空军 4城防
	Health      int    `json:"health"`
	AtkSea      int    `json:"atk_sea"`
	AtkGround   int    `json:"atk_ground"`
	AtkAir      int    `json:"atk_air"`
	AtkDef      int    `json:"atk_def"`
	Defence     int    `json:"defence"`
	Speed       int    `json:"speed"`
	AttackRange int    `json:"attack_range"`
	Carry       int    `json:"carry"`
	Pop         int    `json:"pop"`
	FoodKeep    int    `json:"food_keep"` // 维护耗粮/小时/个
	OilKeep     int    `json:"oil_keep"`
	Food        int64  `json:"food"`
	Steel       int64  `json:"steel"`
	Oil         int64  `json:"oil"`
	Rare        int64  `json:"rare"`
	TrainTime   int    `json:"train_time"` // 秒/个
	Require     string `gorm:"type:varchar(500)" json:"require"`
	Icon        string `gorm:"type:varchar(50)" json:"icon"`
	RepairRate  int    `json:"repair_rate"` // 战损修复率%
}

func (EzfyCfgTroop) TableName() string { return "ezfy_cfg_troop" }

type EzfyCfgTech struct {
	ID           int    `gorm:"primaryKey" json:"id"`
	Name         string `gorm:"type:varchar(50)" json:"name"`
	Type         int    `gorm:"default:3" json:"type"` // 1生产 2军事 3辅助
	MaxLevel     int    `gorm:"default:10" json:"max_level"`
	PreBuilding  int    `json:"pre_building"` // 需要科研中心等级
	PreTech      int    `json:"pre_tech"`
	PreTechLevel int    `json:"pre_tech_level"`
	Effect       string `gorm:"type:varchar(255)" json:"effect"`
	Des          string `gorm:"type:varchar(500)" json:"des"`
}

func (EzfyCfgTech) TableName() string { return "ezfy_cfg_tech" }

type EzfyCfgTechLevel struct {
	ID           int    `gorm:"primaryKey" json:"id"`
	TechId       int    `json:"tech_id"`
	Level        int    `json:"level"`
	Food         int64  `json:"food"`
	Steel        int64  `json:"steel"`
	Oil          int64  `json:"oil"`
	Rare         int64  `json:"rare"`
	Gold         int64  `json:"gold"`
	ResearchTime int    `json:"research_time"` // 秒
	Effect       string `gorm:"type:varchar(255)" json:"effect"`
}

func (EzfyCfgTechLevel) TableName() string { return "ezfy_cfg_tech_level" }

type EzfyCfgWildland struct {
	ID         int    `gorm:"primaryKey" json:"id"`
	Type       int    `json:"type"` // 1陆地野地 2海野 3寇城
	Level      int    `json:"level"`
	Troops     string `gorm:"type:varchar(1000)" json:"troops"` // [[兵种id,最小,最大],...]
	ResMin     int64  `json:"res_min"`
	ResMax     int64  `json:"res_max"`
	OfficerMin int    `json:"officer_min"` // 旧字段，已被 OfficerId 取代（保留避免迁移麻烦）
	OfficerMax int    `json:"officer_max"` // 旧字段，同上
	// ★ 守军军官：**最多 1 个**，且只能从「军官池」（ezfy_cfg_general）里选。
	//   0 = 该野地没有守将（打下来也俘不到军官）。
	OfficerId int    `json:"officer_id"`
	Treasure  string `gorm:"type:varchar(100)" json:"treasure"`
	Des       string `gorm:"type:varchar(500)" json:"des"`
}

func (EzfyCfgWildland) TableName() string { return "ezfy_cfg_wildland" }

// EzfyMapTile 地图格子覆盖（管理端维护）
//
// ★ 用户要求：管理端要能维护**所有**野地（不只是玩家已占领的），能改土地类型，
//   也能把某格设成 寇城 / 活动寇城。
//
//   地图本身是「坐标哈希推导」出来的（地形、野地等级、寇城、活动目标全都不落库），
//   所以这里做一张**覆盖表**：命中就覆盖哈希结果，没命中就照旧走哈希。
type EzfyMapTile struct {
	ID uint `gorm:"primaryKey" json:"id"`
	X  int  `gorm:"uniqueIndex:uk_map_tile" json:"x"`
	Y  int  `gorm:"uniqueIndex:uk_map_tile" json:"y"`
	// Terrain: 0 = 不覆盖（按哈希），1-9 = 强制成该地形（1平原…8海洋, 9沿海平原）
	Terrain int `json:"terrain"`
	// MarkKind: 0=无 1=寇城 2=活动寇城 3=活动野地 4=特殊城市
	MarkKind int `json:"mark_kind"`
	// MarkLevel: 活动目标等级 1~3（仅 2/3/4 有意义）
	MarkLevel int       `json:"mark_level"`
	Des       string    `gorm:"type:varchar(200)" json:"des"`
	UpdatedAt time.Time `json:"updated_at"`
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
	ID           int    `gorm:"primaryKey" json:"id"` // 1..20，同时也是军衔等级
	Name         string `gorm:"type:varchar(20)" json:"name"`
	Post         string `gorm:"type:varchar(20)" json:"post"` // 职位
	NeedPrestige int    `json:"need_prestige"`                // 需要的声望
	CityMax      int    `json:"city_max"`                     // 可建城数
	Des          string `gorm:"type:varchar(200)" json:"des"`
}

func (EzfyCfgRank) TableName() string { return "ezfy_cfg_rank" }

// EzfyFriend 游戏内好友（★ 与家园好友完全分开，不共用 Friendship 表）
//
// 双向各存一行：A 加 B 成功时写 (A,B) 与 (B,A)，删除时两行一起删。
type EzfyFriend struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserId    uint      `gorm:"uniqueIndex:uk_ezfy_friend" json:"user_id"`
	FriendId  uint      `gorm:"uniqueIndex:uk_ezfy_friend" json:"friend_id"`
	Remark    string    `gorm:"type:varchar(50)" json:"remark"`
	CreatedAt time.Time `json:"created_at"`
}

func (EzfyFriend) TableName() string { return "ezfy_friend" }

// EzfyFriendApply 游戏内好友申请
type EzfyFriendApply struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserId    uint      `gorm:"index:idx_ezfy_apply" json:"user_id"`   // 申请人
	TargetId  uint      `gorm:"index:idx_ezfy_apply" json:"target_id"` // 被申请人
	Remark    string    `gorm:"type:varchar(100)" json:"remark"`
	Status    int       `gorm:"default:0" json:"status"` // 0待处理 1已同意 2已拒绝
	CreatedAt time.Time `json:"created_at"`
}

func (EzfyFriendApply) TableName() string { return "ezfy_friend_apply" }

// EzfyCfgResource 资源显示名配置（管理端可改名，全站展示跟随）
//
// ★ 这是「预留」能力：后期把「稀矿」改成别的叫法，只要改这张表，
// 游戏端 / 管理端的资源名就全部跟着变，不用改代码。
// Key 是程序内部标识（gold/food/steel/oil/rare），Name 是展示名，Short 是单字简称（金/粮/钢/油/稀）。
type EzfyCfgResource struct {
	ID    int    `gorm:"primaryKey" json:"id"`
	Key   string `gorm:"type:varchar(20);uniqueIndex" json:"key"`
	Name  string `gorm:"type:varchar(30)" json:"name"`
	Short string `gorm:"type:varchar(10)" json:"short"`
	Sort  int    `json:"sort"`
}

func (EzfyCfgResource) TableName() string { return "ezfy_cfg_resource" }

type EzfyCfgItem struct {
	ID          int    `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"type:varchar(50)" json:"name"`
	ItemType    int    `json:"item_type"` // 1资源包 2黄金包 3建筑加速 4训练加速 5科技加速 6建筑图纸 7增产 8免战
	Param1      int64  `json:"param1"`
	PriceGold   int64  `json:"price_gold"`
	Icon        string `gorm:"type:varchar(50)" json:"icon"`
	Description string `gorm:"type:varchar(500)" json:"description"`
	// ★ 商城库存（管理端「数据管理 → 道具配置」可改），默认 100；0 = 售罄
	Stock int `gorm:"default:100" json:"stock"`
	// ★ 钻石售价：> 0 表示这是「钻石道具」，只能用钻石购买（黄金价 price_gold 忽略）。
	//   管理端「数据管理 → 道具配置」可维护。
	PriceDiamond int64 `json:"price_diamond"`
	// ★ 商城分类（管理端可填；留空时按 item_type / price_diamond 自动归类）
	Category string `gorm:"type:varchar(20)" json:"category"`
}

func (EzfyCfgItem) TableName() string { return "ezfy_cfg_item" }

// EzfyCfgLimit 二战风云「建筑数量上限」全局配置（单行，id = 1）
//
// 用户规则：军事区与资源区数量上限**分开**，各 33；管理端可维护，默认 33。
type EzfyCfgLimit struct {
	ID          int `gorm:"primaryKey" json:"id"`
	MilitaryMax int `gorm:"default:33" json:"military_max"` // 军事区建筑数量上限（type 2/3/4）
	ResourceMax int `gorm:"default:33" json:"resource_max"` // 资源区建筑数量上限（type 1）
	HouseMax    int `gorm:"default:10" json:"house_max"`    // 民居数量上限
	FactoryMax  int `gorm:"default:0" json:"factory_max"`   // 军工厂数量上限（0 = 不限）
}

func (EzfyCfgLimit) TableName() string { return "ezfy_cfg_limit" }

// EzfyWordFilter 二战风云聊天敏感词（独立于社区「黑名单榜」的 word_filters）
//
// 用户规则：二战的聊天敏感词走自己的单独维护页面。
// Type: 1 = 替换（用 Replace 覆盖），2 = 拦截（直接拒绝发言）。
type EzfyWordFilter struct {
	ID      uint      `gorm:"primaryKey" json:"id"`
	Word    string    `gorm:"type:varchar(50);uniqueIndex:uk_ezfy_word" json:"word"`
	Replace string    `gorm:"type:varchar(50)" json:"replace"`
	Type    int       `gorm:"default:1" json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

func (EzfyWordFilter) TableName() string { return "ezfy_word_filter" }

type EzfyCfgTaskType struct {
	ID        int    `gorm:"primaryKey" json:"id"`
	Name      string `gorm:"type:varchar(50)" json:"name"`
	Code      string `gorm:"type:varchar(30)" json:"code"`
	ResetType int    `json:"reset_type"` // 0一次性 1每日
	SortNo    int    `json:"sort_no"`
	Status    int    `gorm:"default:1" json:"status"`
}

func (EzfyCfgTaskType) TableName() string { return "ezfy_cfg_task_type" }

type EzfyCfgTask struct {
	ID             int    `gorm:"primaryKey" json:"id"`
	Name           string `gorm:"type:varchar(100)" json:"name"`
	TaskType       string `gorm:"type:varchar(30)" json:"task_type"`
	Target         int    `json:"target"`
	RewardGold     int64  `json:"reward_gold"`
	RewardFood     int64  `json:"reward_food"`
	RewardSteel    int64  `json:"reward_steel"`
	RewardOil      int64  `json:"reward_oil"`
	RewardRare     int64  `json:"reward_rare"`
	RewardPrestige int    `json:"reward_prestige"`
	SortNo         int    `json:"sort_no"`
	TypeId         int    `json:"type_id"`
	Status         int    `gorm:"default:1" json:"status"`
}

func (EzfyCfgTask) TableName() string { return "ezfy_cfg_task" }

// ============ 玩家档案 ============

// EzfyProfile 玩家游戏档案（替代 Java 版挂在 user 表上的声望/阵营字段，保持 ezfy_ 前缀约束）
type EzfyProfile struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	UserID   uint   `gorm:"uniqueIndex" json:"user_id"`
	Nickname string `gorm:"type:varchar(20)" json:"nickname"`
	Prestige int    `gorm:"default:0" json:"prestige"` // 军功声望
	Camp     int    `gorm:"default:1" json:"camp"`     // 1同盟国 2轴心国

	// ★ 游戏ID：与家园ID 解耦，**首次 = 家园ID，之后永不随家园ID变化**
	//   游戏内所有业务交互都以它为准（为「游戏单独运行」预留）。
	GameUID int64 `gorm:"index" json:"game_uid"`

	// 当前操作的城市（分城切换用；0/无效时回落到 id 最小的主城）
	CurrentCityId int64 `json:"current_city_id"`

	// 首次免费次数是否已用掉（0=还能免费一次，1=已用过，之后要消耗道具）
	RenameUsed int `json:"rename_used"` // 改昵称
	CampUsed   int `json:"camp_used"`   // 改阵营

	// 军校每日免费刷新次数覆盖（0 = 跟随全局默认，管理端可单独调整）
	RecruitFreeLimit int `json:"recruit_free_limit"`

	// ★ 钻石：二战风云专属币种，**只能由管理端充值**，玩家端只读余额；
	//   用于购买「钻石道具」（ezfy_cfg_item.price_diamond > 0）。
	Diamond int64 `gorm:"default:0" json:"diamond"`

	UpdatedAt time.Time `json:"updated_at"`
}

func (EzfyProfile) TableName() string { return "ezfy_profile" }

// ============ 运行时表 ============

type EzfyCity struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	UserID    uint   `gorm:"index:idx_user" json:"user_id"`
	Name      string `gorm:"type:varchar(50)" json:"name"`
	X         int    `gorm:"index:idx_xy" json:"x"`
	Y         int    `gorm:"index:idx_xy" json:"y"`
	CityLevel int    `gorm:"default:1" json:"city_level"` // 市政厅等级
	Feelings  int    `json:"feelings"`                    // 民心
	Grievance int    `gorm:"default:0" json:"grievance"`  // 民怨
	TaxRate   int    `json:"tax_rate"`                    // 税率%
	Pop       int64  `json:"pop"`
	PopMax    int64  `json:"pop_max"`
	Gold      int64  `json:"gold"`
	Food      int64  `json:"food"`
	Steel     int64  `json:"steel"`
	Oil       int64  `json:"oil"`
	Rare      int64  `json:"rare"`
	GoldCap   int64  `json:"gold_cap"`
	FoodCap   int64  `json:"food_cap"`
	SteelCap  int64  `json:"steel_cap"`
	OilCap    int64  `json:"oil_cap"`
	RareCap   int64  `json:"rare_cap"`
	LastTime  int64  `json:"last_time"` // 上次资源结算时间戳(ms)

	// 仓库保护配比(4 项资源的保护额度占比, 合计 ≤ 100; 默认各 25)
	// 每项保护额度 = 仓库等级对应保护总量 × 该项占比 / 100
	WareFood  int `json:"ware_food"`
	WareSteel int `json:"ware_steel"`
	WareOil   int `json:"ware_oil"`
	WareRare  int `json:"ware_rare"`

	// 调整生产·开工率(0~100, 复刻原版 city/sourceSet.html)
	// 实际产量 = 基础产量 × 开工率 / 100, 降低开工率可减少军队耗粮以外的资源消耗压力
	RateFood  int `json:"rate_food"`
	RateSteel int `json:"rate_steel"`
	RateOil   int `json:"rate_oil"`
	RateRare  int `json:"rate_rare"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (EzfyCity) TableName() string { return "ezfy_city" }

type EzfyCityBuilding struct {
	ID         uint  `gorm:"primaryKey" json:"id"`
	CityId     int64 `gorm:"index:idx_city_building" json:"city_id"`
	BuildingId int   `json:"building_id"`
	Level      int   `json:"level"`                   // 0=未建造
	Status     int   `gorm:"default:0" json:"status"` // 0空闲 1建造中 2升级中
	StartTime  int64 `json:"start_time"`              // 0=一键满级连锁模式
	EndTime    int64 `json:"end_time"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (EzfyCityBuilding) TableName() string { return "ezfy_city_building" }

type EzfyCityTroop struct {
	ID      uint  `gorm:"primaryKey" json:"id"`
	CityId  int64 `gorm:"uniqueIndex:uk_city_troop" json:"city_id"`
	TroopId int   `gorm:"uniqueIndex:uk_city_troop" json:"troop_id"`
	Count   int64 `json:"count"`

	UpdatedAt time.Time `json:"updated_at"`
}

func (EzfyCityTroop) TableName() string { return "ezfy_city_troop" }

type EzfyCityTech struct {
	ID      uint  `gorm:"primaryKey" json:"id"`
	CityId  int64 `gorm:"uniqueIndex:uk_city_tech" json:"city_id"`
	TechId  int   `gorm:"uniqueIndex:uk_city_tech" json:"tech_id"`
	Level   int   `json:"level"`
	Status  int   `gorm:"default:0" json:"status"` // 0空闲 1研究中
	EndTime int64 `json:"end_time"`

	UpdatedAt time.Time `json:"updated_at"`
}

func (EzfyCityTech) TableName() string { return "ezfy_city_tech" }

type EzfyTrainQueue struct {
	ID        uint  `gorm:"primaryKey" json:"id"`
	CityId    int64 `gorm:"index:idx_city" json:"city_id"`
	TroopId   int   `json:"troop_id"`
	Count     int64 `json:"count"`
	Status    int   `gorm:"default:0" json:"status"` // 0训练中 1待领取 2已领取
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time"`

	UpdatedAt time.Time `json:"updated_at"`
}

func (EzfyTrainQueue) TableName() string { return "ezfy_train_queue" }

type EzfyMapArea struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	X         int    `gorm:"uniqueIndex:uk_xy" json:"x"`
	Y         int    `gorm:"uniqueIndex:uk_xy" json:"y"`
	AreaType  int    `gorm:"default:0" json:"area_type"` // 0空地 1野地(被占) 2寇城 3玩家城 4资源田
	Terrain   int    `gorm:"default:1" json:"terrain"`   // 1平原..8海洋
	Level     int    `json:"level"`                      // 野地等级
	OwnerId   int64  `json:"owner_id"`                   // 占领城市ID
	Troops    string `gorm:"type:varchar(2000)" json:"troops"`
	Resources string `gorm:"type:varchar(500)" json:"resources"`
	Officer   string `gorm:"type:varchar(500)" json:"officer"`
	Hp        int64  `json:"hp"`
	StartTime int64  `json:"start_time"` // 寇城复活时间

	UpdatedAt time.Time `json:"updated_at"`
}

func (EzfyMapArea) TableName() string { return "ezfy_map_area" }

// EzfyActivity 节日活动(复刻 `参考材料/开发文档/福利.txt` 的活动设计, 原版 Java 未实现)
// Type: 1资源增产 2造兵打折 3建造加速 4研究加速 5声望加成
// Param 为百分比; Status: 0未开启 1进行中; 生效还要求 start_time <= now < end_time
type EzfyActivity struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Name      string `gorm:"type:varchar(50)" json:"name"`
	Type      int    `json:"type"`
	Param     int    `json:"param"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	Status    int    `gorm:"default:0" json:"status"`
	Des       string `gorm:"type:varchar(500)" json:"des"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (EzfyActivity) TableName() string { return "ezfy_activity" }

// EzfyMapStar 地图坐标收藏(复刻原版地图页的「收藏列表」)
type EzfyMapStar struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	UserID uint   `gorm:"index:idx_star_user" json:"user_id"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Name   string `gorm:"type:varchar(50)" json:"name"`

	CreatedAt time.Time `json:"created_at"`
}

func (EzfyMapStar) TableName() string { return "ezfy_map_star" }

type EzfyOrder struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	UserID     uint   `gorm:"index:idx_user" json:"user_id"`
	CityId     int64  `json:"city_id"`
	OrderType  int    `json:"order_type"`  // 1侦查 2掠夺 3征服 4采集 5运输 6增援 7派遣
	TargetType int    `json:"target_type"` // 1野地 2寇城 3玩家城
	TargetX    int    `json:"target_x"`
	TargetY    int    `json:"target_y"`
	TargetId   int64  `json:"target_id"`
	Troops     string `gorm:"type:varchar(2000)" json:"troops"` // [{"troopId":1,"count":100}]
	Officer    string `gorm:"type:varchar(255)" json:"officer"`
	StartTime  int64  `json:"start_time"`
	ArriveTime int64  `json:"arrive_time"`
	ReturnTime int64  `json:"return_time"`
	Status     int    `gorm:"index:idx_status;default:0" json:"status"` // 0行进 1驻守中 2返回 3完成 4阵亡
	Result     string `gorm:"type:varchar(3000)" json:"result"`         // 返回部队JSON/采集标记
	// ★ 采集到的资源先记在部队身上（待带回），只有「返航到达」才入城；
	//   容量上限 = 部队各兵种 carry 之和。JSON: {"food":..,"steel":..,"oil":..,"rare":..,"gold":..}
	Carry     string `gorm:"type:varchar(500)" json:"carry"`
	Resources string `gorm:"type:varchar(500)" json:"resources"`
	OilUsed   int64  `json:"oil_used"`
	WaitMin   int    `json:"wait_min"` // 宿营分钟数(0~1440), 到达后停留该时长再返航

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (EzfyOrder) TableName() string { return "ezfy_order" }

type EzfyReport struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	UserID     uint   `gorm:"index:idx_user" json:"user_id"`
	OrderId    int64  `json:"order_id"`
	ReportType int    `json:"report_type"` // 1侦察 2掠夺 3征服 4战斗 5采集/派遣 6系统
	Title      string `gorm:"type:varchar(255)" json:"title"`
	Content    string `gorm:"type:longtext" json:"content"`
	Detail     string `gorm:"type:longtext" json:"detail"`
	IsRead     int    `gorm:"default:0" json:"is_read"`

	CreatedAt time.Time `json:"created_at"`
}

func (EzfyReport) TableName() string { return "ezfy_report" }

type EzfyWildland struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	CityId    int64  `gorm:"index:idx_city" json:"city_id"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	WildType  int    `json:"wild_type"` // 1陆地野地 2海野 3特殊野地
	Level     int    `json:"level"`
	Gain      string `gorm:"type:varchar(500)" json:"gain"`
	Status    int    `gorm:"default:0" json:"status"` // 0空闲 1采集中
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (EzfyWildland) TableName() string { return "ezfy_wildland" }

// EzfyOccupy 占领的玩家城市（城市归入攻击方附属）
type EzfyOccupy struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	CityId    int64  `json:"city_id"`
	CityName  string `gorm:"type:varchar(50)" json:"city_name"`
	AtkUserId uint   `json:"atk_user_id"`
	AtkCityId int64  `json:"atk_city_id"`
	DefUserId uint   `json:"def_user_id"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Status    int    `gorm:"default:1" json:"status"` // 1占领中 2已摧毁(归还)

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (EzfyOccupy) TableName() string { return "ezfy_occupy" }

type EzfyWounded struct {
	ID      uint  `gorm:"primaryKey" json:"id"`
	CityId  int64 `gorm:"index:idx_city" json:"city_id"`
	TroopId int   `json:"troop_id"`
	Type    int   `json:"type"` // 0伤兵 1逃兵
	Count   int64 `json:"count"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (EzfyWounded) TableName() string { return "ezfy_wounded" }

// EzfyWar 宣战记录：宣战后延迟生效，生效后一段时间内可互相掠夺/征服
type EzfyWar struct {
	ID          uint  `gorm:"primaryKey" json:"id"`
	AtkUserId   uint  `gorm:"index:idx_pair" json:"atk_user_id"`
	DefUserId   uint  `gorm:"index:idx_pair" json:"def_user_id"`
	Status      int   `json:"status"` // 1宣战待生效 2交战中
	DeclareTime int64 `json:"declare_time"`
	EffectTime  int64 `json:"effect_time"`
	ExpireTime  int64 `json:"expire_time"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (EzfyWar) TableName() string { return "ezfy_war" }

type EzfyCorps struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Name         string `gorm:"type:varchar(20);uniqueIndex:uk_name" json:"name"`
	LeaderUserId uint   `json:"leader_user_id"`
	Notice       string `gorm:"type:varchar(200)" json:"notice"`
	MemberCount  int    `gorm:"default:1" json:"member_count"`

	CreatedAt time.Time `json:"created_at"`
}

func (EzfyCorps) TableName() string { return "ezfy_corps" }

type EzfyCorpsMember struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CorpsId   uint      `gorm:"index:idx_corps" json:"corps_id"`
	UserId    uint      `gorm:"uniqueIndex:uk_user" json:"user_id"`
	IsLeader  int       `gorm:"default:0" json:"is_leader"`
	Title     string    `gorm:"type:varchar(20)" json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

func (EzfyCorpsMember) TableName() string { return "ezfy_corps_member" }

type EzfyCorpsChat struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CorpsId   uint      `gorm:"index:idx_corps" json:"corps_id"`
	UserId    uint      `json:"user_id"`
	UserName  string    `gorm:"type:varchar(20)" json:"user_name"`
	Content   string    `gorm:"type:varchar(200)" json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func (EzfyCorpsChat) TableName() string { return "ezfy_corps_chat" }

// EzfyItem 玩家背包
type EzfyItem struct {
	ID     uint `gorm:"primaryKey" json:"id"`
	UserId uint `gorm:"uniqueIndex:uk_user_item" json:"user_id"`
	CfgId  int  `gorm:"uniqueIndex:uk_user_item" json:"cfg_id"`
	Count  int  `json:"count"`
}

func (EzfyItem) TableName() string { return "ezfy_item" }

type EzfySign struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserId    uint      `json:"user_id"`
	SignDate  string    `gorm:"type:varchar(10);uniqueIndex:uk_user_date" json:"sign_date"`
	SignCount int       `json:"sign_count"` // 连续签到天数
	CreatedAt time.Time `json:"created_at"`
}

func (EzfySign) TableName() string { return "ezfy_sign" }

// EzfyGift 礼包领取记录
type EzfyGift struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserId    uint      `json:"user_id"`
	GiftType  string    `gorm:"type:varchar(20)" json:"gift_type"`
	CreatedAt time.Time `json:"created_at"`
}

func (EzfyGift) TableName() string { return "ezfy_gift" }

// EzfyCityEffect 城市效果：1增产 2免战
type EzfyCityEffect struct {
	ID         uint  `gorm:"primaryKey" json:"id"`
	CityId     int64 `gorm:"uniqueIndex:uk_city_effect" json:"city_id"`
	EffectType int   `gorm:"uniqueIndex:uk_city_effect" json:"effect_type"`
	Param1     int   `json:"param1"`
	UntilTime  int64 `json:"until_time"`
}

func (EzfyCityEffect) TableName() string { return "ezfy_city_effect" }

// EzfyCityTarget 司令部兵种战斗配置：优先攻击目标/前进停止
type EzfyCityTarget struct {
	ID             uint  `gorm:"primaryKey" json:"id"`
	CityId         int64 `gorm:"uniqueIndex:uk_city_troop" json:"city_id"`
	TroopId        int   `gorm:"uniqueIndex:uk_city_troop" json:"troop_id"`
	AtkTargetTroop int   `json:"atk_target_troop"` // 0=最近目标
	AtkMove        int   `json:"atk_move"`
	DefTargetTroop int   `json:"def_target_troop"`
	DefMove        int   `json:"def_move"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (EzfyCityTarget) TableName() string { return "ezfy_city_target" }

// EzfyTask 玩家任务进度
type EzfyTask struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserId     uint       `gorm:"uniqueIndex:uk_user_task" json:"user_id"`
	CfgId      int        `gorm:"uniqueIndex:uk_user_task" json:"cfg_id"`
	Current    int        `json:"current"`
	Status     int        `json:"status"` // 0进行中 1可领取 2已领取
	TaskDate   string     `gorm:"type:varchar(10)" json:"task_date"`
	CreatedAt  time.Time  `json:"created_at"`
	FinishTime *time.Time `json:"finish_time"`
}

func (EzfyTask) TableName() string { return "ezfy_task" }

// EzfyNotice 公告/通知（0=全员公告）
type EzfyNotice struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserId    uint      `gorm:"index:idx_user" json:"user_id"`
	Title     string    `gorm:"type:varchar(100)" json:"title"`
	Content   string    `gorm:"type:varchar(2000)" json:"content"`
	IsTop     int       `gorm:"default:0" json:"is_top"`
	IsRead    int       `gorm:"default:0" json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

func (EzfyNotice) TableName() string { return "ezfy_notice" }

// EzfyChat 游戏聊天（复刻原版 chatB?type=：1公共 2军团 4系统）
type EzfyChat struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserId    uint      `json:"user_id"`
	UserName  string    `gorm:"type:varchar(20)" json:"user_name"`
	Content   string    `gorm:"type:varchar(200)" json:"content"`
	Channel   int       `gorm:"default:1;index:idx_channel" json:"channel"` // 1公共 2军团 4系统
	TalkType  int       `gorm:"default:1" json:"talk_type"`                 // 0系统(只读) 1玩家
	CreatedAt time.Time `json:"created_at"`
}

func (EzfyChat) TableName() string { return "ezfy_chat" }

// EzfyExchange 交易所挂单：玩家卖资源换黄金
type EzfyExchange struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SellerId   uint      `gorm:"index:idx_seller" json:"seller_id"`
	SellerName string    `gorm:"type:varchar(20)" json:"seller_name"`
	EsType     int       `json:"es_type"` // 1粮 2钢 3油 4稀矿
	EsCount    int64     `json:"es_count"`
	TotalPrice int64     `json:"total_price"`                              // 总价(黄金)
	Status     int       `gorm:"index:idx_status;default:0" json:"status"` // 0在售 1成交 2下架
	BuyerId    uint      `json:"buyer_id"`
	IsSystem   int       `gorm:"default:0" json:"is_system"` // 1系统挂单
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (EzfyExchange) TableName() string { return "ezfy_exchange" }

// ============ 军官/学院系统（复刻 stzb-fk：军校/参谋部/技能/装备/俘虏/任命） ============

// EzfyCfgGeneral 名将配置（源自 inithebing.sql cfg_general 31 条）
type EzfyCfgGeneral struct {
	ID           int    `gorm:"primaryKey" json:"id"`
	Name         string `gorm:"type:varchar(100)" json:"name"`
	Level        int    `json:"level"` // 名将等级(决定招募费用=等级×500黄金)
	Military     int    `json:"military"`
	Logistics    int    `json:"logistics"`
	Learning     int    `json:"learning"`
	Star         int    `gorm:"default:5" json:"star"`
	Source       string `gorm:"type:varchar(255)" json:"source"`
	GetCondition string `gorm:"type:varchar(255)" json:"get_condition"`
	Skill        string `gorm:"type:varchar(500)" json:"skill"`
	Des          string `gorm:"type:varchar(500)" json:"des"`
	Recruit      int    `gorm:"default:1" json:"recruit"` // 1=可入军校候选 0=停用
}

func (EzfyCfgGeneral) TableName() string { return "ezfy_cfg_general" }

// EzfyCfgSkill 技能配置（15 个，每名军官最多学 3 个）
type EzfyCfgSkill struct {
	ID     int    `gorm:"primaryKey" json:"id"`
	Name   string `gorm:"type:varchar(50)" json:"name"`
	Effect string `gorm:"type:varchar(100)" json:"effect"`
	Type   int    `gorm:"default:1" json:"type"` // 1攻击类 2防御类 3辅助类
	Des    string `gorm:"type:text" json:"des"`  // 完整技能说明(参考 skill.html, 较长)
}

func (EzfyCfgSkill) TableName() string { return "ezfy_cfg_skill" }

// EzfyCfgEquipment 装备配置（26 件：18 装备 + 8 地形珠宝）
type EzfyCfgEquipment struct {
	ID        int    `gorm:"primaryKey" json:"id"`
	Name      string `gorm:"type:varchar(50)" json:"name"`
	Type      string `gorm:"type:varchar(20);default:武器" json:"type"` // 武器/防具/饰品/珠宝
	Tier      int    `gorm:"default:1" json:"tier"`                   // 1初级 2中级 3高级 4特殊
	Military  int    `json:"military"`
	Logistics int    `json:"logistics"`
	Learning  int    `json:"learning"`
	Level     int    `gorm:"default:1" json:"level"` // 穿戴等级需求
	Des       string `gorm:"type:varchar(200)" json:"des"`
}

func (EzfyCfgEquipment) TableName() string { return "ezfy_cfg_equipment" }

// EzfyOfficer 武将实例（军官）
type EzfyOfficer struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CityId     int64     `gorm:"index:idx_city" json:"city_id"`
	GeneralId  int       `json:"general_id"`
	Name       string    `gorm:"type:varchar(100)" json:"name"`
	Star       int       `gorm:"default:1" json:"star"`
	Level      int       `gorm:"default:1" json:"level"`
	Exp        int64     `json:"exp"`
	Military   int       `json:"military"`
	Logistics  int       `json:"logistics"`
	Learning   int       `json:"learning"`
	Loyalty    int       `gorm:"default:100" json:"loyalty"`         // 0-100，归零离职
	Skill      string    `gorm:"type:varchar(500)" json:"skill"`     // 技能名 JSON 数组
	Equipment  string    `gorm:"type:varchar(500)" json:"equipment"` // 已穿戴装备 JSON 数组
	Position   int       `gorm:"default:0" json:"position"`          // 0无 1市长 2城守
	Status     int       `gorm:"default:0" json:"status"`            // 0在职 1出征中 2被俘
	IsCaptive  int       `gorm:"default:0" json:"is_captive"`
	UpdateTime time.Time `json:"update_time"`
}

func (EzfyOfficer) TableName() string { return "ezfy_officer" }

// EzfyEquipment 玩家装备背包（野地掉宝入库，可穿戴到军官）
type EzfyEquipment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserId    uint      `gorm:"index:idx_user" json:"user_id"`
	CityId    int64     `json:"city_id"`
	CfgId     int       `json:"cfg_id"`
	Name      string    `gorm:"type:varchar(50)" json:"name"`
	Type      string    `gorm:"type:varchar(20)" json:"type"`
	Tier      int       `json:"tier"`
	Military  int       `json:"military"`
	Logistics int       `json:"logistics"`
	Learning  int       `json:"learning"`
	Level     int       `json:"level"`
	OfficerId int64     `gorm:"index:idx_officer" json:"officer_id"` // 0=未穿戴
	CreatedAt time.Time `json:"created_at"`
}

func (EzfyEquipment) TableName() string { return "ezfy_equipment" }

// EzfyRecruit 军校每日候选名将 / 刷新次数（每日 0 点重置，限刷 5 次）
type EzfyRecruit struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserId       uint      `gorm:"uniqueIndex:uk_role_date" json:"user_id"`
	RecruitDate  string    `gorm:"type:varchar(10);uniqueIndex:uk_role_date" json:"recruit_date"`
	RefreshCount int       `json:"refresh_count"`
	Candidates   string    `gorm:"type:text" json:"candidates"` // JSON: 随机军官候选
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (EzfyRecruit) TableName() string { return "ezfy_recruit" }
