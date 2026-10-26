package model

import (
	"time"

	"gorm.io/gorm"
)

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
	// ★ 2026-10-14 效果数值（百分点，**该等级的累计加成**；0 = 未配置 → 走原公式兜底）。
	//   管理端可直接维护数字（如「治愈伤兵 Lv.10 改成 50」），游戏逻辑/展示读它优先于文案。
	EffectValue int `gorm:"default:0;comment:效果数值(百分点,该等级累计加成;0=未配置走公式)" json:"effect_value"`
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
	OfficerId int `gorm:"comment:军官ID" json:"officer_id"`
	// ★ 2026-10-05 宝物掉落（管理端下拉编辑器生成）：JSON [{"name":"宝物名","count":数量,"pct":概率%},...]，
	//   老文本值（如「珠宝(平原)」）保留兼容，解析失败不掉落。
	Treasure string `gorm:"type:varchar(500);comment:宝物掉落JSON" json:"treasure"`
	// ★ 2026-10-05 商城道具掉落：JSON [[cfg_id,数量],...]，留空=不掉（默认）。
	//   打赢该类型野地（含海野/寇城）后按配置掉落商城道具到背包。
	DropItems string `gorm:"type:varchar(500);comment:商城道具掉落JSON" json:"drop_items"`
	Des       string `gorm:"type:varchar(500);comment:描述" json:"des"`
}

func (EzfyCfgWildland) TableName() string { return "ezfy_cfg_wildland" }

// EzfyMapTile 地图格子覆盖（管理端维护）
//
// ★ 管理端要能维护**所有**野地（不只是玩家已占领的），能改土地类型，
//
//	也能把某格设成 寇城 / 活动寇城。
//
//	地图本身是「坐标哈希推导」出来的（地形、野地等级、寇城、活动目标全都不落库），
//	所以这里做一张**覆盖表**：命中就覆盖哈希结果，没命中就照旧走哈希。
type EzfyMapTile struct {
	ID uint `gorm:"primaryKey;comment:主键ID" json:"id"`
	X  int  `gorm:"uniqueIndex:uk_map_tile;comment:X坐标" json:"x"`
	Y  int  `gorm:"uniqueIndex:uk_map_tile;comment:Y坐标" json:"y"`
	// Terrain: 0 = 不覆盖（按哈希），1-11 = 强制成该地形（1平原…8海洋, 9沿海平原, 10渔场, 11矿礁）
	Terrain int `gorm:"comment:地形" json:"terrain"`
	// MarkKind: 0=无 1=寇城 2=活动寇城 3=活动野地 4=特殊城市
	MarkKind int `gorm:"comment:MarkKind: 0=无 1=寇城 2=活动寇城 3=活动野地 4=特殊城市" json:"mark_kind"`
	// MarkLevel: 活动目标等级 1~3（仅 2/3/4 有意义）
	MarkLevel int       `gorm:"comment:MarkLevel: 活动目标等级 1~3（仅 2/3/4 有意义）" json:"mark_level"`
	Des       string    `gorm:"type:varchar(200);comment:描述" json:"des"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyMapTile) TableName() string { return "ezfy_map_tile" }

// EzfyActWild 活动野地配置（地图管理「活动野地」tab）
//
// ★ 2026-09-29 活动野地配置不友好，优化成「按坐标列表管理」。
//
//	活动野地原有坐标由哈希 + mark 覆盖标记决定，这里给**每个活动野地坐标**挂一份可配置数据：
//	- Enabled：启用开关。开 = 该格按活动野地玩法（守军/奖励/等级用本条配置，缺省回退默认）；
//	  关 = 该格**区别于普通野地**，按普通野地/无活动处理。
//	- 守军/奖励 可配，留 0/空 = 用代码默认。
type EzfyActWild struct {
	ID uint `gorm:"primaryKey;comment:主键ID" json:"id"`
	X  int  `gorm:"uniqueIndex:uk_actwild;comment:X坐标" json:"x"`
	Y  int  `gorm:"uniqueIndex:uk_actwild;comment:Y坐标" json:"y"`
	// Enabled 启用开关：1=启用活动（是活动野地） 0=关闭（该格按普通野地处理）
	Enabled int `gorm:"default:0;comment:启用开关 1启用 0关闭" json:"enabled"`
	Level   int `gorm:"default:0;comment:活动等级1-3，0=用默认" json:"level"`
	// Troops 守军配置 JSON [[兵种id,最小,最大],...]（兼容旧格式 [兵种id,数量]），空=用默认守军
	Troops string `gorm:"type:varchar(1000);comment:守军JSON [[兵种id,min,max],...] 兼容 [id,count]" json:"troops"`
	// Res 每次胜利资源奖励(粮/钢/油/稀矿各加)，0=用默认
	Res int64 `gorm:"default:0;comment:资源奖励，0=用默认" json:"res"`
	// Gold 黄金(元宝)奖励，0=用默认
	Gold int64 `gorm:"default:0;comment:黄金奖励，0=用默认" json:"gold"`
	// Prestige 声望奖励，0=用默认
	Prestige int `gorm:"default:0;comment:声望奖励，0=用默认" json:"prestige"`
	// Jewel 必掉宝物名，空=用默认地形宝石
	Jewel string `gorm:"type:varchar(100);comment:必掉宝物，空=默认" json:"jewel"`
	// ★ 2026-09-29 必掉宝物改为多行配置：JSON [[cfg_id,数量],...]，空=用默认地形珠宝
	Treasures string `gorm:"type:varchar(500);comment:必掉宝物JSON [[cfg_id,count],...]" json:"treasures"`
	// OfficerId 该坐标活动野地的守将军官（军官池 ID，0=不设守将）。普通军官/名将都可选。
	OfficerId int `gorm:"default:0;comment:守将军官ID，0=不设" json:"officer_id"`
	// CaptureRate 守将被俘虏概率%（0=不俘虏；>0 按该百分比，100=必俘虏）
	CaptureRate int `gorm:"default:0;comment:守将被俘虏概率%，0=不俘虏" json:"capture_rate"`
	// ★ 2026-09-30 同一玩家可抓守将次数上限（默认 1）。玩家已抓到上限后，
	//   再次胜利抓取概率强制 0（详见 ezfy_activity_target.go 的结算判定）。
	MaxCapture int `gorm:"default:1;comment:同一玩家可抓该守将次数，默认1" json:"max_capture"`
	// ★ 2026-10-25 兜底被俘次数：至少**打赢**该守将多少次之后才可能被俘虏。
	//   0 = 第一次打赢就按 CaptureRate 概率判（原行为）；N>0 = 前 N-1 次打赢必定俘不到。
	//   ⚠️ **不下发到用户端**（玩家看不到这个门槛，战报里也不提示）。
	MinCaptureWins int `gorm:"default:0;comment:兜底被俘次数:至少打赢多少次才可能被俘(0=直接按概率)" json:"min_capture_wins"`
	// ★ 2026-10-24 守军强度微调（0 = 用军官池默认）：
	//   - DefLevel：守将战斗等级（影响守将技能 / 组合技等级）；0 = 跟随军官池 Level（通常 1）
	DefLevel int `gorm:"default:0;comment:守将等级,0=用军官池默认" json:"def_level"`
	// CaptiveLevel 被俘虏默认等级（0 = 用军官池 Level，即默认 1 级；>0 按该等级发放俘虏）
	CaptiveLevel int `gorm:"default:0;comment:被俘虏默认等级,0=用军官池默认(1级)" json:"captive_level"`
	// ★ 2026-10-24 守军行为（**按兵种**配置）：JSON map {"兵种ID":方向}，
	//   方向 1 前进（默认）/ 0 停止 / -1 后退；空/{} = 全部兵种默认前进（主动迎击）。
	DefMoveDir string `gorm:"type:varchar(255);default:'';comment:守军行为JSON map{兵种ID:1前进/0停止/-1后退}" json:"def_move_dir"`
	Des        string `gorm:"type:varchar(200);comment:描述" json:"des"`
}

func (EzfyActWild) TableName() string { return "ezfy_act_wild" }

// EzfyActLevel 活动目标「等级配置」（地图管理 →「活动目标配置」tab）
//
// ★ 2026-10-22 用户需求：活动野地 / 活动寇城 / 特殊城市 三类活动目标，
// 位置可以先不维护，但**每个等级的兵力、资源、宝物、掉落道具**必须能在管理端维护。
//
// 运行时按 (ActType, Level) 取配置：
//   - Troops 空 → 走代码内置的陆/海守军默认配比；
//   - Res/Gold/Prestige 为 0 → 走代码内置默认奖励；
//   - Treasures 空 → 不额外掉落宝物；
//   - DropItems 空 → 不额外掉落商城道具。
//
// 坐标级「活动野地」配置（EzfyActWild）仍然优先于本表，用于个别坐标特调。
type EzfyActLevel struct {
	ID      uint `gorm:"primaryKey;comment:主键ID" json:"id"`
	ActType int  `gorm:"uniqueIndex:uk_act_level;comment:活动类型 1活动野地 2活动寇城 3特殊城市" json:"act_type"`
	Level   int  `gorm:"uniqueIndex:uk_act_level;comment:活动等级1-3" json:"level"`
	// Troops 守军配置 JSON [[兵种id,最小,最大],...]（兼容旧格式 [兵种id,数量]），空=用默认守军
	Troops string `gorm:"type:varchar(1000);comment:守军JSON [[兵种id,min,max],...]" json:"troops"`
	// Res 每次胜利资源奖励（粮/钢/油/稀矿各加），0=用默认
	Res int64 `gorm:"default:0;comment:资源奖励，0=用默认" json:"res"`
	// Gold 黄金（元宝）奖励，0=用默认
	Gold int64 `gorm:"default:0;comment:黄金奖励，0=用默认" json:"gold"`
	// Prestige 声望奖励，0=用默认
	Prestige int `gorm:"default:0;comment:声望奖励，0=用默认" json:"prestige"`
	// Treasures 必掉宝物 JSON [[装备cfg_id,数量],...]
	Treasures string `gorm:"type:varchar(500);comment:必掉宝物JSON [[cfg_id,count],...]" json:"treasures"`
	// DropItems 商城道具掉落 JSON [[道具cfg_id,数量,概率%],...]
	DropItems string `gorm:"type:varchar(1000);comment:商城道具掉落JSON [[cfg_id,count,pct],...]" json:"drop_items"`
	// ★ 2026-10-24 与活动野地同款：守将 / 守军微调 / 被俘（坐标级 aw 启用时优先，否则回退本表）。
	//   - OfficerId：守将军官（军官池 ID，0 = 不设守将）
	//   - CaptureRate：守将被俘虏概率%（0 = 不俘虏；>0 按该百分比，100 = 必俘虏）
	//   - MaxCapture：同一玩家可抓该守将次数（默认 1；抓满后概率归 0；0 = 不限）
	//   - MinCaptureWins：兜底被俘次数（至少打赢多少次才可能被俘；0 = 第一次就按概率）
	//   - DefLevel：守将战斗等级（影响守将技能 / 组合技等级）；0 = 用军官池默认
	//   - CaptiveLevel：被俘虏默认等级（0 = 用军官池 Level，通常 1 级）
	//   - DefMoveDir：守军行为（按兵种）JSON map {"兵种ID":1前进/0停止/-1后退}；空/{} = 全部默认前进
	OfficerId   int `gorm:"default:0;comment:守将军官ID，0=不设" json:"officer_id"`
	CaptureRate int `gorm:"default:0;comment:守将被俘虏概率%，0=不俘虏" json:"capture_rate"`
	MaxCapture  int `gorm:"default:1;comment:同一玩家可抓该守将次数，默认1" json:"max_capture"`
	// ★ 2026-10-25 兜底被俘次数：至少**打赢**该守将多少次之后才可能被俘虏
	//   （0 = 第一次打赢就按概率；⚠️ 不下发到用户端）
	MinCaptureWins int    `gorm:"default:0;comment:兜底被俘次数:至少打赢多少次才可能被俘(0=直接按概率)" json:"min_capture_wins"`
	DefLevel       int    `gorm:"default:0;comment:守将等级,0=用军官池默认" json:"def_level"`
	CaptiveLevel   int    `gorm:"default:0;comment:被俘虏默认等级,0=用军官池默认(1级)" json:"captive_level"`
	DefMoveDir     string `gorm:"type:varchar(255);default:'';comment:守军行为JSON map{兵种ID:1前进/0停止/-1后退}" json:"def_move_dir"`
	Des            string `gorm:"type:varchar(500);comment:说明" json:"des"`
	UpdatedAt      time.Time
}

func (EzfyActLevel) TableName() string { return "ezfy_act_level" }

// EzfyActCaptureTry 活动目标守将「兜底被俘次数」进度：玩家对某守将的累计**胜场数**。
//
// ★ 2026-10-25：ezfy_act_wild / ezfy_act_level 新增 min_capture_wins 后，
// 打赢次数按 (玩家, 守将) 累计 —— 与 MaxCapture（同一玩家可抓该守将次数）同维度：
// 坐标级配置（ezfy_act_wild）与等级配置（ezfy_act_level）**共用同一份进度**，
// 同一个守将在哪个坐标打都算次数。
//
// ⚠️ 只在管理端/服务端使用，**任何用户端接口都不下发**。
type EzfyActCaptureTry struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId    uint      `gorm:"uniqueIndex:uk_act_try_ug,priority:1;comment:玩家ID" json:"user_id"`
	GeneralId int       `gorm:"uniqueIndex:uk_act_try_ug,priority:2;comment:守将军官池ID" json:"general_id"`
	Wins      int64     `gorm:"default:0;comment:已打赢该守将次数" json:"wins"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (EzfyActCaptureTry) TableName() string { return "ezfy_act_capture_try" }

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
	// ★ 2026-09-28 军衔晋升所需宝物（JSON 数组 [{name,count}]，管理端军衔配置可维护）
	Treasures string `gorm:"type:text;comment:晋升所需宝物JSON" json:"treasures"`
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

// EzfyMail 游戏内邮件（收件箱 / 系统邮件 / 发件箱）
//
// ★ 2026-10-13 邮箱改造：玩家之间的邮件 + 系统邮件都在这一张表。
//
//	一封邮件只有一行，发件人与收件人各自按自己视角读：
//	- 收件人视角：receiver_id = 我 且 sender_deleted = 0（sender_id=0 即系统邮件）
//	- 发件人视角：sender_id = 我 且 receiver_deleted = 0
//	删除只置自己视角的标记，不影响对方。
type EzfyMail struct {
	ID         uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	SenderId   uint   `gorm:"index:idx_ezfy_mail_send;comment:发件人(0=系统)" json:"sender_id"` // 0 = 系统邮件
	ReceiverId uint   `gorm:"index:idx_ezfy_mail_recv;comment:收件人" json:"receiver_id"`     // 0 = 收件人为全体（系统群发）
	SenderName string `gorm:"type:varchar(30);comment:发件人昵称快照" json:"sender_name"`         // 系统邮件为「系统」
	Subject    string `gorm:"type:varchar(30);comment:主题(≤20字)" json:"subject"`
	Content    string `gorm:"type:varchar(1000);comment:正文(≤500字)" json:"content"`
	IsRead     int    `gorm:"default:0;comment:0未读 1已读" json:"is_read"`
	SenderDel  int    `gorm:"default:0;comment:发件人视角已删" json:"sender_del"`
	RecvDel    int    `gorm:"default:0;comment:收件人视角已删" json:"recv_del"`
	// ★★ 2026-10-18 邮件附件（风云榜奖励 / 系统发放走这里）：
	//   HasAttach 0=纯文本邮件 1=带附件（列表徽标与查询用，省一次子查询）
	//   Claimed   0=未领取 1=已领取（领取后附件入背包/宝箱/钻石，邮件正文保留）
	//   ExpireAt  附件过期时间(ms)，0=永不过期；过期后不可领取（玩家仍能看到「已过期」）
	HasAttach int   `gorm:"default:0;comment:0无附件 1有附件" json:"has_attach"`
	Claimed   int   `gorm:"default:0;comment:0未领取 1已领取" json:"claimed"`
	ExpireAt  int64 `gorm:"default:0;comment:附件过期时间(ms) 0=不过期" json:"expire_at"`
	// ★ 2026-10-18 批次号：一次「一键发放」写入的所有邮件共用同一个 BatchNo。
	//   用途：批量插入后按它回查邮件 id（比依赖自增连号可靠），以及管理端按批次对账。
	BatchNo   string    `gorm:"type:varchar(40);index:idx_ezfy_mail_batch;comment:发放批次号" json:"batch_no"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyMail) TableName() string { return "ezfy_mail" }

// EzfyMailAttach 邮件附件（★ 2026-10-18 风云榜奖励 / 系统发放）
//
// 一封邮件可带多件附件，Kind 区分三类：
//
//	1 道具（CfgID = ezfy_cfg_item.id，进背包 ezfy_item）
//	2 宝箱（CfgID = ezfy_cfg_chest.id，进待开宝箱 ezfy_user_chest）
//	3 钻石（CfgID = 0，直接加 ezfy_profile.diamond）
//
// ★ Name 是发放时的**名称快照**：配置被改名/删除后，邮件里仍能显示玩家当时看到的名字。
// ★ UserID 冗余存收件人：删号清理（ezfyPurgeByUid）按 user_id 一把清，不必先查邮件 id。
type EzfyMailAttach struct {
	ID     uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	MailID uint   `gorm:"index:idx_ezfy_mail_attach_mail;comment:邮件ID" json:"mail_id"`
	UserID uint   `gorm:"index:idx_ezfy_mail_attach_user;comment:收件人(删号清理用)" json:"user_id"`
	Kind   int    `gorm:"comment:1道具 2宝箱 3钻石" json:"kind"`
	CfgID  int    `gorm:"comment:道具/宝箱配置ID(钻石=0)" json:"cfg_id"`
	Num    int64  `gorm:"comment:数量" json:"num"`
	Name   string `gorm:"type:varchar(50);comment:名称快照" json:"name"`
}

func (EzfyMailAttach) TableName() string { return "ezfy_mail_attach" }

// EzfyPvpReward 风云榜奖励发放记录（★ 2026-10-18）
//
// 作用只有一个：**同一个人同一周只发一次**。(week,user_id) 唯一索引，
// 重复点「发放奖励」时只补发还没发过的人，绝不重复发。
type EzfyPvpReward struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	Week      string    `gorm:"type:varchar(20);uniqueIndex:uk_ezfy_pvp_reward;comment:周(如 20261012-20261018)" json:"week"`
	UserID    uint      `gorm:"uniqueIndex:uk_ezfy_pvp_reward;comment:玩家ID" json:"user_id"`
	RankNo    int       `gorm:"comment:当周名次" json:"rank_no"`
	Tier      string    `gorm:"type:varchar(10);comment:档位 rank1/rank2/rank3/join" json:"tier"`
	MailID    uint      `gorm:"comment:发出的邮件ID" json:"mail_id"`
	Detail    string    `gorm:"type:varchar(255);comment:奖励明细摘要(如 钻石x500、宝箱[名]x2、道具[名]x5)" json:"detail"`
	CreatedAt time.Time `gorm:"comment:发放时间" json:"created_at"`
}

func (EzfyPvpReward) TableName() string { return "ezfy_pvp_reward" }

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
	ItemType    int    `gorm:"comment:1资源包 2黄金包 3建筑加速 4训练加速 5科技加速 6建筑图纸 7增产 8免战 27粮食包 28钢铁包 29石油包 30稀矿包" json:"item_type"` // 1资源包 2黄金包 3建筑加速 4训练加速 5科技加速 6建筑图纸 7增产 8免战 27粮食包 28钢铁包 29石油包 30稀矿包
	Param1      int64  `gorm:"comment:参数1" json:"param1"`
	PriceGold   int64  `gorm:"comment:黄金价格" json:"price_gold"`
	Icon        string `gorm:"type:varchar(50);comment:图标" json:"icon"`
	Description string `gorm:"type:varchar(500);comment:描述" json:"description"`
	// ★ 商城库存（管理端「数据管理 → 道具配置」可改）；-1 = 无限，0 = 售罄。
	//   ★ 2026-10-22 用户规则「商城道具库存都改成无限」→ 新道具默认 -1。
	Stock int `gorm:"default:-1;comment:库存(-1=无限)" json:"stock"`
	// ★ 钻石售价：> 0 表示这是「钻石道具」，只能用钻石购买（黄金价 price_gold 忽略）。
	//   管理端「数据管理 → 道具配置」可维护。
	PriceDiamond int64 `gorm:"comment:钻石价格" json:"price_diamond"`
	// ★ 商城分类（管理端可填；留空时按 item_type 自动归类）。
	//   「钻石道具 / 黄金道具」只用于锁定支付货币，不再霸占展示分类。
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
	// ★ 2026-10-22 用户规则「军事区、资源区建造队列默认每个城市最多 5 个，
	//   二战系统配置可调整上限」：这里是**基础上限**，「征募告示」的临时层数另加。
	BuildQueueMax int `gorm:"default:5;comment:每城建造队列基础上限" json:"build_queue_max"`
	// ★ 「首页公告默认只能展示一条，管理端可以配置」→ 首页外露公告条数（默认 1）
	NoticeHomeCount int `gorm:"default:1;comment:公告家园数量" json:"notice_home_count"`
	// ★ 「出征集结令上限后台管理系统可维护，最大默认 99」→ 单次出征最多用几个集结令（默认 99）
	GatherMaxPerOrder int `gorm:"default:99;comment:集结上限每订单" json:"gather_max_per_order"`
	// ★ 2026-09-28 「挂单出售按 1:100 卡控单价，比例系统可灵活配置」→ 挂单出售单价上限（黄金/单位，默认 100）
	SellPriceMax int `gorm:"default:100;comment:挂单出售单价上限" json:"sell_price_max"`
	// ★ 2026-09-28 「军官军事累加上限/加速出征，数值可配置」→ 每点军事累加的出征上限（默认 2000）
	OfficerCapPerMilitary int `gorm:"default:2000;comment:军官军事每点出征上限" json:"officer_cap_per_military"`
	// ★ 2026-09-28 军官军事加成出征速度（每点 %，默认 0.1）
	OfficerSpeedPerMilitary float64 `gorm:"default:0.1;comment:军官军事每点速度加成" json:"officer_speed_per_military"`

	// ============ 战斗 / 经济数值（管理端「建筑上限配置」页可维护）============
	//
	// ★ 用户反馈「征服民心每次 -5 现在太多」→ 做成可配置（线上现值 5）。
	//   征服(3)：单次最多扣掉目标多少民心（原来写死 20，且按幸存兵力/2000 动态计算）。
	//   掠夺(2)：每次固定扣目标多少民心（原来写死 5，线上现值 3）。
	ConquerFeelingsMax int `gorm:"default:5;comment:Conquer民心上限" json:"conquer_feelings_max"`
	LootFeelings       int `gorm:"default:3;comment:掠夺民心" json:"loot_feelings"`
	// ★ 2026-10-24 民心平滑回归（秒级配置，入库持久化）：
	//   上升：被掠夺/征服扣掉的民心按秒**平滑涨回**上限（默认 3600 秒回归到合理位置）；
	//   下降：税率上调/民怨上升导致民心高于上限时，按秒**平滑降回**上限（默认 3600 秒）；
	//   恢复上限受「民心 + 民怨 + 税率 = 100」约束（民怨没清掉民心只能回到上限）；
	//   0 无意义 → 读取端回落默认 3600。
	FeelingsRecoverSeconds int `gorm:"default:3600;comment:民心平滑回归上升秒数" json:"feelings_recover_seconds"`
	FeelingsDropSeconds    int `gorm:"default:3600;comment:民心平滑回归下降秒数" json:"feelings_drop_seconds"`
	// ★ 2026-10-07 赎城功能：被占城市原主人赎回需消耗的钻石（管理端「建筑上限配置」可维护，默认 500）。
	//   0 无意义（= 禁止赎城）→ 读取端回落默认 500。
	RansomCost int `gorm:"default:500;comment:赎城金额钻石" json:"ransom_cost"`
	// ★ 用户反馈「军官是消耗黄金的，黄金现在消耗 0」→ 军官工资：每名军官每小时消耗
	//   「等级 × 该值」黄金，在 calcResource 里随资源懒结算一起扣。
	//   ★ 2026-09-26 「现值 2000 改成 20」：默认值 100 → **20**。
	//   起因：名将发放等级修好之后（原来被发成 1 级），150 级名将的工资按 2000/级
	//   高达 30 万黄金/小时，远超玩家产出（人口 × 税率 ≈ 4 万/小时）→ 会直接破产。
	//   ⚠️ 字段名必须让 GORM 推出 officer_salary_per_level（与 seed 里补的列名一致），
	//   否则 AutoMigrate 会另外建一列 officer_salary_per_lv，两个列各存各的。
	OfficerSalaryPerLevel int `gorm:"default:20;comment:军官Salary每等级" json:"officer_salary_per_level"`
	// ★ 用户反馈「恢复伤兵需要黄金」→ 恢复 1 个伤兵消耗
	//   ceil(该兵种总造价 / 该值) 黄金，最低 1 黄金。默认 50。
	WoundHealDivisor int `gorm:"default:50;comment:伤兵治疗Divisor" json:"wound_heal_divisor"`
	// ★ 2026-10-22 用户规则：商城单次购买上限统一 999（集结令原 15）。
	//   读不到或 <= 0 时回落默认值（0 无意义 = 等于禁止购买）。
	MallBuyMax int `gorm:"default:999;comment:Mall购买上限" json:"mall_buy_max"`
	// ★ 2026-10-11 用户规则：「玩家背包道具上限卡控，默认 5000，超过就不能再买，二战系统配置可配置」：
	//   同一种道具在玩家背包里的**持有上限**（ezfy_item.count 不得超过它）。
	//   Buy 时按「已有 + 本次购买 <= 上限」卡控；<= 0 无意义 → 读取端回落默认 5000。
	//   已超上限的存量不做回溯处理（「超过的暂时先这样」）。
	BagItemMax int `gorm:"default:5000;comment:背包单件道具上限" json:"bag_item_max"`
	// ★ 2026-10-11 用户规则：「宝物库存上限 5000，超过 5000 采集的时候停止给提示，打地掉落宝物超上限也提示」：
	//   玩家**未穿戴**的采集宝物（9 种珠宝）持有上限。采集结算 / 打地掉落宝物时卡控；
	//   <= 0 无意义 → 读取端回落默认 5000。已超上限的存量不回溯。
	TreasureMax int `gorm:"default:5000;comment:宝物库存上限" json:"treasure_max"`
	// ★ 2026-10-26 用户规则：「采集珠宝播报在二战系统配置加个开关，默认关，
	//   采集不播报也不入播报的库」→ 采集（驻守结算/一键收获）掉宝时的世界频道播报开关。
	//   1 = 播报（写 ezfy_chat 系统频道）；0 = 不播报、也**不写播报库**（默认 0 关）。
	//   ⚠️ 0 是合法值（= 关），读取端不能走 ezfyLimitOr 兜底（那会把 0 改回默认）。
	LootBroadcastOn int `gorm:"default:0;comment:采集宝物播报开关(1开0关)" json:"loot_broadcast_on"`

	// ★ 2026-10-11 用户规则「打野地掉落宝物太频繁了，即使我配置了 1%，
	//   如果玩家 2 小时内掉落了宝物那么就不会再掉」：
	//   打野地（野地/寇城战斗掉落）掉宝的**冷却小时数**，默认 2。
	//   ★ 0 是**合法值**（= 关闭冷却）→ 读取端不能用「<=0 回落默认」那套，见 ezfyWildTreasureCooldownH。
	WildTreasureCooldownH int `gorm:"default:2;comment:打野地掉宝冷却小时(0=关闭)" json:"wild_treasure_cooldown_h"`

	// ★ 2026-10-13 用户规则「野地商城道具掉落也加个限制，有人用脚本刷野地，暂时卡控 1 小时内
	//   掉落过了就不能再次掉落，二战系统配置可配置间隔时间」：
	//   打野地（野地/寇城战斗掉落）掉**商城道具**的冷却小时数，默认 1。
	//   ★ 0 是**合法值**（= 关闭冷却）→ 读取端不能用「<=0 回落默认」那套，见 ezfyWildItemCooldownH。
	WildItemCooldownH int `gorm:"default:1;comment:打野地掉道具冷却小时(0=关闭)" json:"wild_item_cooldown_h"`

	// ★ 2026-10-24 活动目标掉落冷却（秒级，默认 7200 = 2 小时）：
	//   必掉宝物 / 掉落道具 各自独立，按**玩家**维度：打活动目标（活动野地/活动寇城/特殊城市）
	//   胜利掉到该类后，冷却期内（距上次掉落不足该秒数）本次不再掉该类。
	//   ★ 0 是**合法值**（= 关闭冷却）→ 读取端不能用「<=0 回落默认」那套，见 ezfyActTreasureCooldownSec。
	ActTreasureCooldownSec int `gorm:"default:7200;comment:活动目标掉宝冷却秒(0=关闭)" json:"act_treasure_cooldown_sec"`
	//   ★ 0 是**合法值**（= 关闭冷却）→ 读取端不能用「<=0 回落默认」那套，见 ezfyActItemCooldownSec。
	ActItemCooldownSec int `gorm:"default:7200;comment:活动目标掉道具冷却秒(0=关闭)" json:"act_item_cooldown_sec"`

	// ★ 2026-10-12 用户规则「迁城道具加个冷却时间，针对迁过城的城市冷却 24 小时，二战系统配置可配置迁城冷却」：
	//   迁城成功后该城要等多少小时才能再迁（默认 24），冷却记在 `ezfy_city.move_cool_until`。
	//   ★ 0 是**合法值**（= 关闭迁城冷却）→ 读取端不能用 ezfyLimitOr（它把 0 当「未配置」回落默认），
	//   见 ezfyMoveCoolH()。
	MoveCoolH int `gorm:"default:24;comment:迁城冷却小时(0=关闭)" json:"move_cool_h"`

	// ★ 2026-09-28 用户规则：「安抚 消费黄金 5w 降低民怨 2 增加民心 1 15分钟可安抚一次」
	//   四个值都可由管理端调整；<= 0 时回落默认（5 万 / 2 / 1 / 15 分钟）。
	PlacateGold        int64 `gorm:"default:50000;comment:安抚花费黄金" json:"placate_gold"`
	PlacateGrievance   int   `gorm:"default:2;comment:安抚降低民怨" json:"placate_grievance"`
	PlacateFeelings    int   `gorm:"default:1;comment:安抚提升民心" json:"placate_feelings"`
	PlacateCooldownMin int   `gorm:"default:15;comment:安抚冷却分钟" json:"placate_cooldown_min"`

	// ★ 「采集 12 小时才有宝物 → 可配置」：
	//   常驻采集结算一期的小时数（默认 1），由 ezfyDispatchPeriod() 读取。
	DispatchPeriodH int `gorm:"default:1;comment:采集周期小时" json:"dispatch_period_h"`

	// ★ 出征速度加成（百分比口径，0 = 无加成）：实际行军时间 = 原时间 × 100/(100+加成)。
	//   ★ 2026-09-24 「节假日我好让玩家队伍走快点」。默认 100。
	MarchSpeedBonus float64 `gorm:"default:100;comment:出征速度加成" json:"march_speed_bonus"`

	// ============ 回合战斗公式（管理端「二战系统配置」页可维护）============
	//
	// ★ 2026-10-13 用户需求「回合战斗公式能维护起来，放到二战系统配置」：
	//   战斗引擎 ezfy_battle.go 顶部的 5 个硬编码机制常量全部配置化，管理端可调、保存即热生效。
	//   默认值 = 移植前的硬编码值（40/6000/10000/50/50），不改默认战斗行为；
	//   读取端用 ezfyLimitOr 兜底（0 = 未配置/配置错 → 回落默认值）。
	BattleMaxRounds  int `gorm:"default:40;comment:战斗最大回合数" json:"battle_max_rounds"`      // 战斗最多多少回合，达到上限按平局（默认 40）
	BattleStartDist  int `gorm:"default:6000;comment:战场初始距离" json:"battle_start_dist"`     // 战场初始距离（攻方 0，守方 6000，相距 6000，默认 6000）
	BattleRetreatMax int `gorm:"default:10000;comment:战斗最大后退距离" json:"battle_retreat_max"` // 双方各自最多后退多少（默认 10000）
	CritBaseBonusPct int `gorm:"default:50;comment:暴击基础伤害加成%" json:"crit_base_bonus_pct"`  // 装备无暴击伤害(crit_dmg=0)时的兜底暴击倍率%（默认 50）
	GarrisonBreakPct int `gorm:"default:50;comment:盟军驻军溃败撤退阈值%" json:"garrison_break_pct"` // 盟军驻军战：守方兵力跌破初始的此比例% 即判败撤退（默认 50）
	// ★ 2026-10-13 伤害公式系数：伤害 = (攻×攻系数+攻加值)/(防×(100+防加成%)×防系数+防加值)×数量, 再乘攻加成; 最小伤害=数量×最小伤害系数
	BattleRoundMs int `gorm:"default:30000;comment:战斗每回合时长(毫秒)" json:"battle_round_ms"` // 指挥室实时战斗每回合毫秒数（默认 30000 = 30 秒）
	AtkCoeff      int `gorm:"default:5;comment:伤害公式攻击系数" json:"atk_coeff"`              // 伤害分子 攻×系数（默认 5）
	AtkBase       int `gorm:"default:1000;comment:伤害公式攻击加值" json:"atk_base"`            // 伤害分子 +加值（默认 1000）
	DefCoeff      int `gorm:"default:3;comment:伤害公式防御系数" json:"def_coeff"`              // 伤害分母 防×系数（默认 3）
	DefBase       int `gorm:"default:10;comment:伤害公式防御加值" json:"def_base"`              // 伤害分母 +加值（默认 10）
	MinDmgCoeff   int `gorm:"default:5;comment:最小伤害系数(数量×系数)" json:"min_dmg_coeff"`     // 最小伤害 = 数量×系数（默认 5）

	// ★ 2026-10-23 用户需求「战斗公式按套来，二战系统配置加个战斗公式维护，为当前服选择用哪套」：
	//   当前服务器生效的「战斗公式套」ID（见 EzfyCfgBattleFormula），0 = 使用本行下面的内置公式字段。
	//   读取端见 ezfyBattleFormulaActive()：选了套就整套覆盖，未选/套已删除则回落本行内置值。
	BattleFormulaID int `gorm:"default:0;comment:当前生效战斗公式套ID(0=内置)" json:"battle_formula_id"`

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
	// ★ 2026-09-26 「二战加个产量加成倍率，默认 1，可以调整 >= 0 的任意数量」：
	//   城市每小时资源产量（粮/钢/油/稀矿/金）整体乘这个倍率。
	//   ⚠️ **0 是合法值**（= 产量归零），不是「未配置」——
	//   所以读取端**不能**用「<= 0 就回落默认」那套（见 ezfyResProdMult）。
	//   ⚠️ 老库补列时只能回填 NULL，别用 `WHERE col <= 0`（那样每次启动都会把玩家设的 0 改回 1）。
	ResProdMult float64 `gorm:"default:1;comment:城市资源产量倍率" json:"res_prod_mult"`
	// ★ 2026-10-05 「产量加成倍率」拆成两个：资源(粮/钢/油/稀矿)与黄金分开。
	//   本字段只乘「粮/钢/油/稀矿」产量；黄金产量走 GoldProdMult。
	//   ⚠️ 0 合法（黄金产量归零）；NULL → seed 回填 1。
	GoldProdMult float64 `gorm:"default:1;comment:黄金产量倍率" json:"gold_prod_mult"`
	// ⚠️ 2026-10-21 **已废弃**（采集改为新公式后不再参与计算）：
	//   旧公式「野地等级^幂次 × 800 × 后勤加成% × 陆海系数 × 本倍率」已被
	//   `F = 卡车数量 × 整点小时 × 野地加成% × 军官后勤 ÷ 除数` 取代。
	//   列保留只为不动历史数据；管理端不再提供该项，dispatchGatherYield 也不再读取。
	GatherResMult float64 `gorm:"default:10;comment:已废弃(2026-10-21)：采集资源倍率" json:"gather_res_mult"`
	// ⚠️ 2026-10-21 **已废弃**：旧公式把后勤折算成 +N% 并封顶 +100%，新公式直接乘后勤属性。
	//   列保留只为不动历史数据；管理端不再提供该项。
	OfficerGatherMult float64 `gorm:"default:1;comment:已废弃(2026-10-21)：采集后勤加成倍率" json:"officer_gather_mult"`
	// ★ 2026-09-28 「市长加成整体可调」：
	//   作用点 mayorBonusPct：市长产量加成% 整体 × 该倍率（默认 1 = 现状）。
	//   0 合法（关闭市长加成）；NULL 无意义 → 回落 1。
	MayorGainMult float64 `gorm:"default:1;comment:市长加成倍率" json:"mayor_gain_mult"`
	// ⚠️ 2026-10-21 **已废弃**：旧公式的野地等级幂次。
	//   新公式里野地等级通过「野地加成%」参与计算（见 GatherWildBonusPct）。
	//   列保留只为不动历史数据；管理端不再提供该项。
	GatherLevelPow float64 `gorm:"default:1.3;comment:已废弃(2026-10-21)：采集等级成长幂次" json:"gather_level_pow"`
	// ⚠️ 2026-10-09 **已废弃**（「掉落 都走手动配置的」）：下面四个字段对应的全局掉落概率
	//   不再被任何代码读写，管理端也已移除。**列保留**只为不动历史数据 ——
	//   现在野地/寇城掉落走「地图管理 → 野地类型」的 treasure/drop_items（`wildlandConfigLoot`），
	//   活动野地走 `EzfyActWild.Treasures`。
	DropT2 int `gorm:"default:18;comment:已废弃(2026-10-09)：中级宝物掉落roll阈值" json:"drop_t2"`
	DropT3 int `gorm:"default:4;comment:已废弃(2026-10-09)：高级宝物掉落roll阈值" json:"drop_t3"`
	DropT4 int `gorm:"default:1;comment:已废弃(2026-10-09)：特殊宝物掉落roll阈值" json:"drop_t4"`
	// ⚠️ 2026-10-09 同上**已废弃**（活动野地掉宝总概率%）。
	DropActPct int `gorm:"default:85;comment:已废弃(2026-10-09)：活动野地战斗掉宝概率" json:"drop_act_pct"`
	// ⚠️ 2026-10-21 **已废弃**：旧公式的海野采集系数。
	//   新公式按野地实际产出资源（粮/钢/油/稀）走「同名资源野地总等级」加成；
	//   海野改由新字段 GatherSeaBonusMult（倍率，默认 3）控制，本字段不再读取。
	//   列保留只为不动历史数据；管理端不再提供该项。
	GatherSeaMult float64 `gorm:"default:1.5;comment:已废弃(2026-10-21)：海野采集系数" json:"gather_sea_mult"`
	// ★★ 2026-10-21 采集新公式（与线上原版一致）：
	//   F = M × N × R × A ÷ 2
	//     M = 部队总兵力（出征部队**所有兵种数量之和**；原说明以运输卡车举例，
	//         不同兵种的差异只体现在负重，产量贡献一律按数量计）
	//     N = 整点小时数（16:59 按 16 小时算，向下取整）
	//     R = 野地加成%（城市名下**同资源**野地总等级 × 每级加成%，默认每级 3%）
	//     A = 带队军官后勤属性
	//   例：180000 兵力 × 16 小时 × 42% × 324 后勤 ÷ 2 = 195955200。
	//
	//   下面几个参数就是管理端「二战系统配置」可调的公式项：
	//   GatherWildBonusPct 每 1 级同资源野地提供的加成%（默认 3）
	//   GatherFormulaDivisor 公式除数（默认 2）
	//   GatherSeaBonusMult 海野采集倍率（默认 3）：海野产出 = 公式结果 × 倍率
	// ⚠️ GatherTruckTroopID 已废弃（M 口径改为全部兵种总数，兵种 ID 限定不再参与计算），列保留。
	GatherTruckTroopID   int     `gorm:"default:3;comment:已废弃(2026-10-21)：采集卡车兵种ID" json:"gather_truck_troop_id"`
	GatherWildBonusPct   float64 `gorm:"default:3;comment:每级野地加成%" json:"gather_wild_bonus_pct"`
	GatherFormulaDivisor float64 `gorm:"default:2;comment:采集公式除数" json:"gather_formula_divisor"`
	// ★ 2026-10-21 海野采集倍率（默认 3）：**海野**（海底森林/岛屿/渔场/矿礁，ezfyIsSeaWildTerrain）
	//   的产出 = 公式结果 × 该倍率 —— 同参数下海野比陆地野地高 3 倍；陆地野地不乘。
	//   0 / 负 / NULL → 回落 3（见 ezfyGatherSeaBonusMult）；旧废弃字段 GatherSeaMult 不参与计算。
	GatherSeaBonusMult float64 `gorm:"default:3;comment:海野采集倍率" json:"gather_sea_bonus_mult"`
	// ★ 2026-09-28 军校刷新周期可在二战系统配置切换按天/按小时（默认按小时）。
	RecruitCycleMode int `gorm:"default:2;comment:军校刷新周期(1=按天 2=按小时)" json:"recruit_cycle_mode"`
	// ★ 2026-10-08 「新玩家落地洲」做成可配置：新玩家首次建城/落地默认落到哪个洲
	//   （大洲 ID：1欧洲 2亚洲 3非洲 4北美洲 5南美洲 6大洋洲 7南极洲，默认欧洲）。
	//   0 / 越界读取端回落默认欧洲，见 ezfyDefaultContinent()。
	DefaultContinent int `gorm:"default:1;comment:新玩家落地洲(1欧洲 2亚洲 3非洲 4北美洲 5南美洲 6大洋洲 7南极洲)" json:"default_continent"`

	// ★ 2026-10-22 用户规则「驻军 [一键采集]/[一键收获]/[一键召回] 默认本城市，
	//   二战系统配置可以改成全部城市」：
	//   1 = 当前城市（默认，与军情→驻军列表口径一致）；2 = 全部城市。
	//   ⚠️ 0 是「老库未配置」→ 读取端按当前城市处理，管理端展示也回落 1。
	GarrisonBatchScope int `gorm:"default:1;comment:驻军批量操作范围(1=当前城市 2=全部城市)" json:"garrison_batch_scope"`

	// ★ 2026-09-27 「资源产量也做成累加」：**每项资源的唯一硬上限**，默认 21 亿 = 2100000000。
	//   产量与一切获取方式都无条件累加到该值为止，不再看仓储/库存上限。
	//
	//   规则（用户确认口径）：
	//     ① 资源**产量**（calcResource 自动产出）**同样无条件累加**，只在此处收敛；
	//     ② 其它一切获取方式（战斗掠夺/野地战利品/采集返航/运输/派遣/资源包/签到福利/任务奖励/交易行/军团商城…）
	//        也一律累加，同样只在这个「资源最大值」处停下来；
	//     ③ 已经超过该值的老数据**不会被拉低**（读端用 GREATEST 保住较大值）。
	//   ⚠️ 存库够用即可（21 亿在 int32 内），seed 补列走 bigint。
	//   0 无意义 → 回落默认 21 亿。
	ResMaxFood  int64 `gorm:"default:2100000000;comment:粮食最大值" json:"res_max_food"`
	ResMaxSteel int64 `gorm:"default:2100000000;comment:钢铁最大值" json:"res_max_steel"`
	ResMaxOil   int64 `gorm:"default:2100000000;comment:石油最大值" json:"res_max_oil"`
	ResMaxRare  int64 `gorm:"default:2100000000;comment:稀矿最大值" json:"res_max_rare"`
	ResMaxGold  int64 `gorm:"default:2100000000;comment:黄金最大值" json:"res_max_gold"`

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

	// ★ 2026-10-22 人口上限只由民居容量（EzfyCity.PopMax）决定：
	//   自然增长到 pop_max 封顶；人口未满时召集同时增加总人口和空闲人口。
	//   原「民居容量限制」、「允许长期超过民居上限」、「召集人口上限」三个配置已删除。
	// ★ 2026-10-23 总人口永远不允许超过民居上限：超限数据在下一次懒结算时直接夹平。
	//   人口已满时召集/移民法案仍可继续：总人口不增加，召集量转为额外空闲人口
	//   （EzfyCity.PopExtra，允许空闲暂时超过总人口，按 pop_extra_decay_seconds 回落，默认 300 秒）。
	//   原「超限人口回落速度」配置也已删除。
	//
	// ★ 2026-09-26 「花费 10万粮食 召集 10万人口也要能配置，现在是写死的」。
	// ★ 2026-10-22 用户规则：召集获得人口不再固定，改为「当前民居上限 × N%」，默认 10%。
	//   召集消耗粮食仍为固定值。两者 0 无意义 → 回落默认（seed 走 addLimitCol）。
	ConveneFoodCost int `gorm:"comment:召集消耗粮食" json:"convene_food_cost"`
	ConvenePopPct   int `gorm:"comment:召集人口占当前民居上限的百分比（默认10，可超过100）" json:"convene_pop_pct"`
	// ★ 2026-10-23 用户规则「召集间隔放到二战系统配置，方便灵活调整和测试」：
	//   普通状态 / 人口达到民居上限后，两次召集的最短间隔（秒）。
	//   0/未配置无意义 → 回落默认 3 秒 / 300 秒（seed 走 addLimitCol 只回填 NULL/<=0）。
	ConveneCooldownSeconds    int `gorm:"default:3;comment:普通召集冷却秒数（默认3）" json:"convene_cooldown_seconds"`
	ConveneCapCooldownSeconds int `gorm:"default:300;comment:人口满时召集冷却秒数（默认300）" json:"convene_cap_cooldown_seconds"`

	// ★ 2026-10-13 用户规则「用完的人口不会立即恢复，是一点一点恢复到当前人口数量的」：
	// ★ 2026-10-22 用户规则：不再配置「每小时恢复量」，改为「多少秒内空闲人口恢复满」，默认 1200 秒。
	//   作用点见 EzfyCity.PopRecover / PopRecoverEndAt。
	//   0 / 未配置无意义 → 回落默认 1200（seed 走 addLimitCol / EnsureEzfyLimitColumns）。
	PopRecoverSeconds int `gorm:"comment:空闲人口恢复满所需秒数（默认1200）" json:"pop_recover_seconds"`

	// ★ 2026-10-23 用户规则「临时空闲按 5 分钟线性回落，这个也要能在二战系统配置里面配置」：
	//   人口达到民居上限后，召集/移民法案产生的额外空闲人口（EzfyCity.PopExtra）
	//   在配置的秒数内线性回落。默认 300 秒（5 分钟）；0 / 未配置无意义 → 回落默认。
	PopExtraDecaySeconds int `gorm:"default:300;comment:临时空闲人口回落秒数（默认300）" json:"pop_extra_decay_seconds"`

	// ============ 军官升星（2026-09-22 ，2026-09-23 按简化）============
	//
	// ★ 简化后的规则：升星按固定概率（officer_star_chance），失败也消耗 1 枚星级徽章，
	//   每升 1 星三维各 +officer_star_attr_gain，星级上限 officer_star_max。
	//   原先的「概率开关/每高 1 星递减/成功率下限/失败保留徽章」四个配置已按去掉。
	//
	// ⚠️ OfficerStarUpOn 是**开关**（0 有意义），不能带 gorm:"default:x" 标签，seed 走 addSwitchCol。
	OfficerStarUpOn int `gorm:"comment:升星功能：1 开 / 0 关（关了不能用升星卡）" json:"officer_star_up_on"` // 升星功能：1 开 / 0 关（关了不能用升星卡）

	// 下面是**数值**（0 无意义 → 回落默认值）
	OfficerStarChance   int `gorm:"default:20;comment:升星成功率%（固定值，默认 20）" json:"officer_star_chance"`      // 升星成功率%（固定值，默认 20）
	OfficerStarAttrGain int `gorm:"default:10;comment:每升 1 星三维各 +N（默认 10）" json:"officer_star_attr_gain"` // 每升 1 星三维各 +N（默认 10）
	OfficerStarMax      int `gorm:"default:5;comment:星级上限（默认 5）" json:"officer_star_max"`                 // 星级上限（默认 5）

	// ★ 2026-09-30 「使用招生简章出五星军官的概率」：
	//   招生简章刷新军校候选时，以 N% 概率把候选中的 1 名置为 5 星。
	//   默认 1 = 1%（≈维持现状）；2 = 2 倍率；100 = 100% 必刷出 5 星军官。
	//   （名将不参与：名将只由管理端发放，「名将开启才可刷到」暂不实现。）
	RecruitFiveStarRate int `gorm:"default:1;comment:招生简章出五星军官概率（1~100，默认1；100=必出5星）" json:"recruit_five_star_rate"`

	// ★ 2026-10-02 用户规则「侦查成功率按侦查机数量，梯度要陡、按 10/1千/1万/5万/10万 拉开」：
	//   成功率 = (本字段作为封顶) × 1/(1+e^-k·(log10(n)-x0))，n = 携带侦察机数。
	//   每架成功率不再线性累乘（那样 10 架就封顶、梯度太浅），改按数量级平滑上升：
	//     n=10≈11%  n=100≈26%  n=1千≈48%  n=1万≈69%  n=5万≈80%  n=10万≈84%
	//   本字段 = 封顶百分比（默认 95），0 无意义 → 回落 95，见 ezfyReconSucceed。
	ReconSuccessPct float64 `gorm:"default:95;comment:侦查成功率封顶(%)" json:"recon_success_pct"`

	// ★ 训练一键加速黄金倍率（百分比口径）：实际费用 = 剩余秒数 × 10 × 倍率/100。
	//   默认 0.1 = 0.1% = 几乎免费（线上现值）；100 = 原价；50 = 半价。
	//   0 无意义 → 回落 0.1。
	SpeedTrainRate float64 `gorm:"default:0.1;comment:速度训练比率" json:"speed_train_rate"`

	// ★ 2026-10-17 训练一键加速开关：1 开（默认）= 训练页展示并使用「训练一键加速 / 所有城市训练一键加速」；
	//   0 关 = 前端不展示按钮、后端接口也直接拒绝。
	//   ⚠️ 开关（0 有意义），不能带 gorm:"default:x" 标签，seed 走 addSwitchCol（只回填 NULL）。
	SpeedTrainOn int `gorm:"comment:训练一键加速开关（关 = 按钮不展示也无法使用）" json:"speed_train_on"`

	// ★ 2026-10-21 造兵时间市长军事加成开关：1 开（默认）= 训练时间按
	//   floor(面板 train_time × 250/(250+市长军事) × 10/(10+军训艺术等级)) 计算；
	//   0 关 = 市长军事按 0 处理，仍吃军训艺术等级。
	//   ⚠️ 开关（0 有意义），不能带 gorm:"default:x" 标签，seed 走 addSwitchCol（只回填 NULL）。
	TrainMayorMilitaryOn int `gorm:"comment:造兵时间市长军事加成开关(1开0关)" json:"train_mayor_military_on"`
	// ★ 2026-10-21 装备/套装军事是否计入市长军事：1 算（默认，取自身+装备+套装）/ 0 不算（只取军官自身）。
	//   仅在上面的 TrainMayorMilitaryOn 开启时有效。
	TrainMayorEquipOn int `gorm:"comment:造兵市长军事计入装备套装(1算0不算)" json:"train_mayor_equip_on"`

	// ★ 伤兵恢复黄金折扣率（百分比口径）：恢复费用 = 兵种总造价 / wound_heal_divisor × 折扣率/100。
	//   默认 100 = 100% = 原价；调低 = 恢复便宜。0 无意义 → 回落 100。
	WoundHealRate float64 `gorm:"default:100;comment:伤兵治疗比率" json:"wound_heal_rate"`

	// ★ 2026-10-14 伤兵营「钻石一键恢复全部伤兵」单价（默认 100 钻，0 无意义 → 回落 100）
	HealAllDiamond int64 `gorm:"default:100;comment:伤兵营钻石一键恢复全部伤兵单价" json:"heal_all_diamond"`

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

	// ★ 2026-09-30 「玩家可向系统出售资源获得黄金，比例可配置」：
	//   玩家把资源**直接卖给系统**（不走挂单），每 100 单位 → N 黄金，实得再扣手续费。
	//   默认：粮食 100:10、钢铁 100:10、石油 100:20、稀矿 100:25。
	//   0 无意义 → 回落各自默认值（见 ezfySysSellRatio）。
	SysSellFood  int `gorm:"default:10;comment:粮食回收比例(每100单位黄金)" json:"sys_sell_food"`
	SysSellSteel int `gorm:"default:10;comment:钢铁回收比例(每100单位黄金)" json:"sys_sell_steel"`
	SysSellOil   int `gorm:"default:20;comment:石油回收比例(每100单位黄金)" json:"sys_sell_oil"`
	SysSellRare  int `gorm:"default:25;comment:稀矿回收比例(每100单位黄金)" json:"sys_sell_rare"`

	// ★ 2026-10-22 用户反馈「商城大资源包 5 万买、交易所卖 6.5 万，扣 10% 仍赚」：
	//   系统回收手续费从写死 10% 改成可配置，默认 30%。大资源包整套卖出：
	//   65,000 × (1 − 30%) = 45,500 < 50,000，低买高卖套利直接变亏。
	//   ⚠️ 0 / 越界（>90）无意义 → 读取端回落默认 30（见 ezfySysSellFeePct）。
	SysSellFee int `gorm:"default:30;comment:系统回收手续费%" json:"sys_sell_fee"`
	// ★ 2026-10-22 每日每种资源向系统出售的数量上限（默认 100 万；0 = 不限）。
	//   在手续费之外再加一层数量卡控，防止脚本无限刷/一次性倾泻资源。
	//   按玩家、按自然日统计（计数落在 ezfy_profile.sys_sell_*）。
	SysSellDailyLimit int64 `gorm:"default:1000000;comment:系统回收每日每种资源上限(0=不限)" json:"sys_sell_daily_limit"`

	// ============ 战力榜权重（★ 2026-10-02 兵力榜 → 战力榜；★ 2026-10-25 兵种分改人口口径）============
	//
	// 战力 = 科技战力 + 建筑战力 + 兵种战力（三项直接相加，明细三列之和 == 总战力）。
	// 完整口径见 handler/ezfy/ezfy_power.go（唯一实现）。
	//   · 科技每级 +power_tech_per_level（默认 120）
	//   · 每项已研究科技 +power_tech_per_tech（默认 100）
	//   · 建筑每级 +power_build_per_level（默认 80）
	//   · 兵种分 = 5 万 × 人口占用/(人口占用+80 亿) —— **纯「数量 × 占用人口」口径**，
	//     侦察机(1 人口)堆量打不过航母(55 人口)，且恒不超过 5 万（20 亿兵也不爆表）。
	//   · 科技/建筑原始分各自 ^power_compress_pow（默认 0.5：只缩大数、严格单调）。
	//   ⚠️ power_troop_type / power_troop_pow 自 2026-10-25 起**不再参与战力榜**
	//     （加任何额外加成都会翻转名次：兵多的人反而排后面 = 用户抱怨的那类不合理）；
	//     字段保留只为兼容旧库/旧管理端表单，改动它们不影响战力。
	PowerTechPerLevel  int     `gorm:"default:120;comment:战力科技每级" json:"power_tech_per_level"`
	PowerTechPerTech   int     `gorm:"default:100;comment:战力每项科技" json:"power_tech_per_tech"`
	PowerBuildPerLevel int     `gorm:"default:80;comment:战力建筑每级" json:"power_build_per_level"`
	PowerTroopType     int     `gorm:"default:300;comment:战力每兵种类型(2026-10-25 起不再参与战力)" json:"power_troop_type"`
	PowerTroopPow      float64 `gorm:"default:0.8;comment:战力兵种数量幂次(2026-10-25 起不再参与战力)" json:"power_troop_pow"`
	PowerCompressPow   float64 `gorm:"default:0.5;comment:战力总量压缩幂次(科技/建筑^此值, 默认0.5)" json:"power_compress_pow"`

	// ★ 2026-10-14 风云榜统计开关（1 = 开：玩家对战结算写周榜；0 = 关：完全不做统计）。
	//   管理端「排行榜 → 风云榜」页可直接开关；关闭后新战斗不再累计，
	//   历史周榜数据保留（重新开启即继续统计，不清历史）。
	PvpRankOn int `gorm:"default:1;comment:风云榜统计开关(1开0关)" json:"pvp_rank_on"`
	// ★ 2026-10-18 风云榜锁分：锁定某周后该周战绩**不再累计积分**（相当于终止当周竞争），
	//   存锁定的周串（如 20261012-20261018），空 = 未锁定；管理端「风云榜」页可锁/解锁。
	PvpRankLockWeek string `gorm:"type:varchar(17);default:'';comment:风云榜锁定周(锁分后该周不再累计)" json:"pvp_rank_lock_week"`

	// ★ 2026-10-17 人机（AI 玩家）风控开关
	//   BotAtkOn：0 = 只发育、不打扰玩家（**0 是合法值**，读取时不能用 ezfyLimitOr 兜底）
	//   BotAtkDaily：每个人机每日出征次数上限（默认 3）
	BotAtkOn    int `gorm:"default:1;comment:人机允许攻击玩家(1开0关=只发育)" json:"bot_atk_on"`
	BotAtkDaily int `gorm:"default:3;comment:每个人机每日出征上限" json:"bot_atk_daily"`
	// ★ 2026-10-17 人机免耗粮：1 = 人机（city.is_bot=1）的城内军队不扣维护粮（真人照常扣）。
	//   人机不打劫不打野时粮食全靠产出，军队耗粮经常把它饿死 → 运营可开这个让 AI 专注发育。
	BotFoodFree int `gorm:"default:0;comment:人机军队免耗粮(1=不扣,真人照常)" json:"bot_food_free"`
	// ★ 2026-10-17 人机自动挑目标的最低声望门槛（默认 500000；0 = 不限；只影响自动选目标，管理端手动指定不受限）
	BotMinPrestige int `gorm:"default:500000;comment:人机自动攻击目标最低声望(0=不限)" json:"bot_min_prestige"`
	// ★ 2026-10-26 人机自动挑目标时「玩家城市最小兵力」（默认 10000；**0 合法 = 不限**）：
	//   只打城内兵力 ≥ 该值的玩家城 —— 避免人机去啃空城/几千兵的小号（打空城既没收益也不像真人）。
	//   管理端「人机管理」顶栏可配；只影响自动选目标，管理端手动指定目标不受限。
	BotTargetMinTroops int64 `gorm:"default:10000;comment:人机攻击玩家城最小兵力(0=不限)" json:"bot_target_min_troops"`
	// ★ 2026-10-18 人机征兵线（默认 500000）：城内部队总数低于此值就执行征兵行动，
	//   管理端「人机管理」可配置（0 或过小无意义 → 回落默认 50 万）。
	BotTrainThreshold int64 `gorm:"default:500000;comment:人机征兵线(低于此值执行征兵)" json:"bot_train_threshold"`
	// ★ 2026-10-18 人机出征时间段（管理端可配）：逗号分隔多个时段，如 "20:00-22:00,08:00-12:00"，
	//   空 = 全天不限；支持跨天时段（22:00-02:00）。只影响人机主动出击，发育不受限。
	BotAtkWindows string `gorm:"type:varchar(120);default:'';comment:人机出征时间段(空=全天)" json:"bot_atk_windows"`
}

func (EzfyCfgLimit) TableName() string { return "ezfy_cfg_limit" }

// EzfyCfgBattleFormula 战斗公式套（二战系统配置 → 战斗公式维护）
//
// ★ 2026-10-23 用户需求「战斗公式搞灵活一点，按套来，然后为当前服选择用哪套」：
//
//	一套公式 = 回合战斗引擎的全部可调参数。管理端可维护多套（如「原版」「快节奏」「测试」），
//	当前服在 ezfy_cfg_limit.battle_formula_id 里选一套生效；选 0 = 使用内置公式。
//	套里的字段 0 都无意义（= 未配置/配置错，读取端回落移植前的硬编码默认值）。
type EzfyCfgBattleFormula struct {
	ID     int    `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name   string `gorm:"type:varchar(50);uniqueIndex:uk_ezfy_battle_formula_name;comment:公式套名称" json:"name"`
	Remark string `gorm:"type:varchar(500);comment:说明" json:"remark"`
	// ★ 2026-10-24 伤害算法：0 = 原版（攻×攻系数+加值）/（防×(1+防%)×防系数+加值）；
	//   1 = 家园版（图片公式：数量×基础伤害²×(1+攻%) / [基础防御×(1+防%)]）。
	//   家园版不使用 CritBaseBonusPct/AtkCoeff/AtkBase/DefCoeff/DefBase/MinDmgCoeff 六项。
	DamageFormula int `gorm:"default:0;comment:伤害算法(0=原版 1=家园版)" json:"damage_formula"`
	// ★ 2026-10-24 战斗加成明细开关：1 = 行动日志展示「(军官·名+% 科技·+%…)」逐项来源；
	//   0 = 关闭，只显示加成总百分比（明细原始数据仍在战场快照 JSON 里，可事后对账）。
	ShowBattleBonusDetail int `gorm:"default:1;comment:展示战斗加成明细(1=展示 0=关闭)" json:"show_battle_bonus_detail"`
	// 战斗机制
	BattleMaxRounds  int `gorm:"default:40;comment:战斗最大回合数" json:"battle_max_rounds"`
	BattleStartDist  int `gorm:"default:6000;comment:战场初始距离" json:"battle_start_dist"`
	BattleRetreatMax int `gorm:"default:10000;comment:战斗最大后退距离" json:"battle_retreat_max"`
	CritBaseBonusPct int `gorm:"default:50;comment:暴击基础伤害加成%" json:"crit_base_bonus_pct"`
	GarrisonBreakPct int `gorm:"default:50;comment:盟军驻军溃败撤退阈值%" json:"garrison_break_pct"`
	BattleRoundMs    int `gorm:"default:30000;comment:战斗每回合时长(毫秒)" json:"battle_round_ms"`
	// 伤害公式系数：伤害 = (攻×攻系数+攻加值)/(防×(100+防加成%)×防系数+防加值)×数量, 再乘攻加成
	AtkCoeff    int `gorm:"default:5;comment:伤害公式攻击系数" json:"atk_coeff"`
	AtkBase     int `gorm:"default:1000;comment:伤害公式攻击加值" json:"atk_base"`
	DefCoeff    int `gorm:"default:3;comment:伤害公式防御系数" json:"def_coeff"`
	DefBase     int `gorm:"default:10;comment:伤害公式防御加值" json:"def_base"`
	MinDmgCoeff int `gorm:"default:5;comment:最小伤害系数(数量×系数)" json:"min_dmg_coeff"`
}

func (EzfyCfgBattleFormula) TableName() string { return "ezfy_cfg_battle_formula" }

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

	// ★ 2026-09-28 军衔不再自动跟随声望：声望达标只是前提，还需提交对应宝物（见 handler.ezfyRankTreasures）。
	//   Rank > 0 = 实际已晋升军衔等级；Rank = 0 = 老玩家未晋升过（展示时回落声望推导，保证旧账不缩水）。
	Rank int `gorm:"default:0;comment:已晋升军衔等级(0=跟随声望旧规则)" json:"rank"`

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

	// ★ 2026-10-02 免战保护令冷却：使用后一段时间内不能再次使用（时间戳, 0 = 无冷却）。
	//   冷却时长 = handler/ezfy/ezfy_geo.go 的 ezfyPeaceCoolH（2026-10-16 起 8 小时）。
	PeaceCoolUntil int64 `gorm:"default:0;comment:免战保护令冷却截止时间戳(ms, 0=无冷却)" json:"peace_cool_until"`

	// ★ 2026-10-10 首页布局：0=跟随默认(新布局) 1=新布局 2=老布局（统帅页单选切换）
	//   玩家 10000（首个管理员）由 seed 幂等置为 2 保留现在的首页老布局。
	HomeLayout int `gorm:"default:0;comment:首页布局(0/1新布局 2老布局)" json:"home_layout"`

	// ★ 2026-10-22 向系统出售资源的每日限额计数（按玩家、按自然日重置）。
	//   目的：卡控「商城买大资源包 → 交易所卖给系统」的套利/脚本刷金。
	//   日期不同时视为新的一天（读取/出售时惰性清零），不改玩家其它数据。
	SysSellDay   string `gorm:"type:varchar(8);default:'';comment:系统回收统计日期(YYYYMMDD)" json:"-"`
	SysSellFood  int64  `gorm:"default:0;comment:今日向系统出售粮食数量" json:"-"`
	SysSellSteel int64  `gorm:"default:0;comment:今日向系统出售钢铁数量" json:"-"`
	SysSellOil   int64  `gorm:"default:0;comment:今日向系统出售石油数量" json:"-"`
	SysSellRare  int64  `gorm:"default:0;comment:今日向系统出售稀矿数量" json:"-"`

	// ★ 2026-10-11 用户规则「打野地掉落宝物太频繁了，即使我配置了 1%」→
	//   记录该玩家**上一次打野地掉到宝物**的时间戳(ms)，0 = 从未掉过。
	//   距上次不足 ezfy_cfg_limit.wild_treasure_cooldown_h 小时（默认 2）则本次不掉宝物。
	WildTreasureAt int64 `gorm:"default:0;comment:打野地掉宝冷却(上次掉宝时间戳ms,0=从未)" json:"wild_treasure_at"`

	// ★ 2026-10-13 用户规则「野地商城道具掉落也加个限制，1 小时内掉落过了就不能再次掉落」→
	//   记录该玩家**上一次打野地掉到商城道具**的时间戳(ms)，0 = 从未掉过。
	//   距上次不足 ezfy_cfg_limit.wild_item_cooldown_h 小时（默认 1）则本次不掉道具。
	//   （与宝物冷却 wild_treasure_at 各自独立：两条掉落互不影响。）
	WildItemAt int64 `gorm:"default:0;comment:打野地掉道具冷却(上次掉道具时间戳ms,0=从未)" json:"wild_item_at"`

	// ★ 2026-10-24 活动目标「必掉宝物」冷却（与野地掉宝同款按**玩家**维度）：
	//   记录该玩家**上一次打活动目标掉到宝物**的时间戳(ms)，0 = 从未掉过。
	//   距上次不足 ezfy_cfg_limit.act_treasure_cooldown_sec 秒（默认 7200 = 2 小时）则本次不掉宝物。
	ActTreasureAt int64 `gorm:"default:0;comment:活动目标掉宝冷却(上次掉宝时间戳ms,0=从未)" json:"act_treasure_at"`

	// ★ 2026-10-24 活动目标「掉落道具」冷却（与野地掉道具同款按**玩家**维度）：
	//   记录该玩家**上一次打活动目标掉到商城道具**的时间戳(ms)，0 = 从未掉过。
	//   距上次不足 ezfy_cfg_limit.act_item_cooldown_sec 秒（默认 7200 = 2 小时）则本次不掉道具。
	//   （与必掉宝物冷却各自独立，两条掉落互不影响。）
	ActItemAt int64 `gorm:"default:0;comment:活动目标掉道具冷却(上次掉道具时间戳ms,0=从未)" json:"act_item_at"`

	// ★★ 2026-10-17 人机标识（二战侧权威源）：与家园库 users.is_bot 保持一致，
	//   由 ezfyBotProvision / 管理端「转人机/转真人」唯一写入点维护。
	IsBot     int       `gorm:"default:0;index:idx_profile_is_bot;comment:是否人机(1=AI玩家)" json:"is_bot"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyProfile) TableName() string { return "ezfy_profile" }

// EzfyDiamondLog 玩家钻石流水（2026-09-28 新增：管理端「数据管理 → 钻石流水」查看的数据来源，
// 所有钻石变动点（商城购买/宝箱/装备/交易所/管理端发放/为爱发电卡）都会写入一条）。
type EzfyDiamondLog struct {
	ID     uint  `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId uint  `gorm:"index;comment:玩家ID" json:"user_id"`
	Change int64 `gorm:"comment:变动数量(+入/-出)" json:"change"`
	// ★ 2026-10-11 「购买宝箱」这类流水的开箱结果在扣款之后才产生 → 先记流水、开箱后回填这里，
	//   管理端「数据管理→钻石流水」加一列展示（历史行留空）。
	Extra     string    `gorm:"type:text;comment:附加信息(宝箱开出的道具，完整不截断)" json:"extra"`
	Balance   int64     `gorm:"comment:变动后余额" json:"balance"`
	Reason    string    `gorm:"type:varchar(100);comment:变动原因" json:"reason"`
	CreatedAt time.Time `gorm:"comment:发生时间" json:"created_at"`
}

func (EzfyDiamondLog) TableName() string { return "ezfy_diamond_logs" }

// EzfyItemUseLog 玩家道具使用流水（2026-10-02 新增：管理端「数据管理 → 道具使用」查看的数据来源，
// 玩家任何消耗道具的行为（使用/出征集结令/建筑图纸/军校招募/改名等）都会写入一条，
// 用于核实「道具莫名消失」类反馈——每条消耗都有据可查）。
type EzfyItemUseLog struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId    uint      `gorm:"index:idx_item_use_user;comment:玩家ID" json:"user_id"`
	CfgId     int       `gorm:"comment:道具配置ID" json:"cfg_id"`
	ItemName  string    `gorm:"type:varchar(50);comment:道具名" json:"item_name"`
	ItemType  int       `gorm:"comment:道具类型" json:"item_type"`
	Count     int       `gorm:"comment:消耗数量" json:"count"`
	Reason    string    `gorm:"type:varchar(100);comment:消耗原因" json:"reason"`
	CreatedAt time.Time `gorm:"comment:发生时间" json:"created_at"`
}

func (EzfyItemUseLog) TableName() string { return "ezfy_item_use_logs" }

// EzfyConveneLog 玩家召集人口流水（2026-10-11 新增：管理端「活动监测」的召集次数数据来源，
// Convene 召集成功时写入一行，用于统计最近时间段内玩家的召集人口次数）。
type EzfyConveneLog struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId    uint      `gorm:"index:idx_convene_user_time;comment:玩家ID" json:"user_id"`
	CityId    int64     `gorm:"comment:城市ID" json:"city_id"`
	PopGain   int64     `gorm:"comment:本次实际加入人口" json:"pop_gain"`
	CreatedAt time.Time `gorm:"index:idx_convene_user_time;comment:召集时间" json:"created_at"`
}

func (EzfyConveneLog) TableName() string { return "ezfy_convene_log" }

// EzfyVisitLog 统帅页访问记录
//
// ★ 2026-10-12 功能「访问记录」：别人打开我的 `games/ezfy?cur=playerinfo` 就记一笔。
// 信息页(cur=info)「占领野地」下方展示历史访问总量 visit_count，点进城查看访问列表。
// TargetUid 是被查看方，VisitorUid 是访问方；自己看自己不记。
type EzfyVisitLog struct {
	ID         uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	TargetUid  uint      `gorm:"index:idx_ezfy_visit_target;comment:被查看统帅ID" json:"target_uid"`
	VisitorUid uint      `gorm:"comment:访问者ID" json:"visitor_uid"`
	CreatedAt  time.Time `gorm:"comment:访问时间" json:"created_at"`
}

func (EzfyVisitLog) TableName() string { return "ezfy_visit_log" }

// EzfyUserChest 玩家持有的未开宝箱（管理端/活动发放，玩家在「我的宝箱」里自己开）
//
// ★ 2026-10-14 用户需求「宝箱也能发放给玩家」：购买的宝箱买了即开（不进背包），
//
//	发放的宝箱进这张表（user_id + chest_id 唯一，数量累加），玩家在自己背包/宝箱页开箱。
//	开箱走 ezfy_cfg_chest 的奖池，不扣钱、不扣全库库存（发放是运营行为，库存是商城口径）。
type EzfyUserChest struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:uk_ezfy_user_chest;comment:玩家ID" json:"user_id"`
	ChestId   int       `gorm:"uniqueIndex:uk_ezfy_user_chest;comment:宝箱配置ID(ezfy_cfg_chest.id)" json:"chest_id"`
	Count     int       `gorm:"default:1;comment:未开数量" json:"count"`
	CreatedAt time.Time `gorm:"comment:获取时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyUserChest) TableName() string { return "ezfy_user_chest" }

// EzfyUserChestPity 玩家宝箱累计抽取次数（★ 2026-10-14）
//
// 奖池奖品配置 MinDraws 保底次数 = **解锁门槛**：该奖品在玩家抽取此宝箱的累计次数
// 达到 MinDraws 之前，不进奖池（开不到）；达到后才按正常权重参与抽奖（不是必出）。
// 默认 1 = 首次抽取即解锁（多数奖品）；传说类按品质配 50~500。
// 每个玩家每个宝箱一条累计次数（total_draw），抽一次 +1。
type EzfyUserChestPity struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:uk_ezfy_chest_pity;comment:玩家ID" json:"user_id"`
	ChestId   int       `gorm:"uniqueIndex:uk_ezfy_chest_pity;comment:宝箱配置ID(ezfy_cfg_chest.id)" json:"chest_id"`
	TotalDraw int       `gorm:"default:0;comment:该宝箱累计抽取次数" json:"total_draw"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyUserChestPity) TableName() string { return "ezfy_user_chest_pity" }

// EzfyChestLog 玩家开箱流水（2026-10-16 新增：管理端「数据管理 → 宝箱开启」查看的数据来源）。
//
// 无论是商城购买即开（ChestOpen）还是发放/背包待开宝箱开箱（ChestBagOpen），
// 每次开箱成功都会写一条：本次开了什么（开出的奖品聚合文本，同名合并成「名×数量」）。
// 用于核实「开了宝箱东西没到账」「开箱结果不对」类反馈——管理员按玩家ID/昵称/宝箱名
// 搜索即可看到每次开箱开出了什么。
type EzfyChestLog struct {
	ID        uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint   `gorm:"index:idx_ezfy_chest_log_user;comment:玩家ID" json:"user_id"`
	ChestId   int    `gorm:"comment:宝箱配置ID(ezfy_cfg_chest.id)" json:"chest_id"`
	ChestName string `gorm:"type:varchar(50);comment:宝箱名快照(开箱时)" json:"chest_name"`
	Count     int    `gorm:"comment:本次开的宝箱数量" json:"count"`
	// 开箱结果聚合文本，如「获得：黄金×5000、经验×100」（同名合并，与开箱播报同口径）
	// ★ 2026-10-21 改为 text：30/100 连开箱结果会超过 255 字符，旧字段会截断最后几条，核查时看不到全量。
	Result    string    `gorm:"type:text;comment:开出的东西(聚合文本，完整不截断)" json:"result"`
	CreatedAt time.Time `gorm:"comment:开箱时间" json:"created_at"`
}

func (EzfyChestLog) TableName() string { return "ezfy_chest_logs" }

// EzfyPvpWeek 风云榜（每周玩家对战战绩，★ 2026-10-14 创建并加积分列）
//
// 按**自然周**（周一 00:00 ~ 周日 23:59，Week 形如 "20261012-20261018"）统计
// **玩家 vs 玩家** 战斗（攻打玩家城 target_type=3，排除野地/寇城/活动目标）：
// 灭敌(击杀敌军数) / 战损(己方阵亡数) / 参战场次 / 胜场 / 积分(每周累计)，每玩家每周一条。
// 榜单按**积分**排名。
type EzfyPvpWeek struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	Week      string    `gorm:"type:varchar(17);uniqueIndex:uk_pvp_week_user;comment:自然周 20060102-20060102" json:"week"`
	UserID    uint      `gorm:"uniqueIndex:uk_pvp_week_user;comment:玩家ID" json:"user_id"`
	Nickname  string    `gorm:"type:varchar(20);comment:昵称快照(写榜时)" json:"nickname"`
	Corps     string    `gorm:"type:varchar(20);comment:军团名快照(写榜时)" json:"corps"`
	Battles   int64     `gorm:"default:0;comment:参战场次" json:"battles"`
	Wins      int64     `gorm:"default:0;comment:胜场" json:"wins"`
	Kills     int64     `gorm:"default:0;comment:灭敌(击杀敌军数)" json:"kills"`
	Losses    int64     `gorm:"default:0;comment:战损(己方阵亡数)" json:"losses"`
	Points    int64     `gorm:"default:0;comment:风云积分(周累计，榜单排名依据)" json:"points"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyPvpWeek) TableName() string { return "ezfy_pvp_week" }

// EzfyPvpDetail 风云榜「我的战绩」场次明细（★ 2026-10-16 玩家端可看每场加了多分）。
//
// 每场玩家对战结算时，攻守双方**各写一条**（points = 本场实际入账积分，封顶/门槛/同对手上限之后的值，0 也记——玩家看得明明白白）。
// 独立于战报表：战报会定期清理，明细表不受影响，历史照查。
type EzfyPvpDetail struct {
	ID      uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	Week    string `gorm:"type:varchar(17);index:uk_pvp_detail;comment:自然周 20060102-20060102" json:"week"`
	UserID  uint   `gorm:"index:uk_pvp_detail;comment:玩家ID" json:"user_id"`
	FoeID   uint   `gorm:"comment:对手玩家ID" json:"foe_id"`
	FoeNick string `gorm:"type:varchar(20);comment:对手昵称快照(结算时)" json:"foe_nick"`
	// ★ 2026-10-17 加 ts 索引：统帅称号「风云榜每日第一」按天聚合积分（ts 区间 + GROUP BY user_id），
	//   没有索引就是每天一次全表扫。
	Ts        int64     `gorm:"index:idx_pvp_detail_ts;comment:战斗时间(毫秒)" json:"ts"`
	Kills     int64     `gorm:"default:0;comment:本场灭敌(原始数量，展示口径)" json:"kills"`
	Losses    int64     `gorm:"default:0;comment:本场战损(原始数量，展示口径)" json:"losses"`
	Points    int64     `gorm:"default:0;comment:本场入账积分(0 也记)" json:"points"`
	IsWin     bool      `gorm:"comment:本场胜" json:"is_win"`
	IsDraw    bool      `gorm:"comment:本场平局" json:"is_draw"`
	// ★ 2026-10-26 管理端「积分明细 → 查看本场战报」：本场**我方视角**战报的 id。
	//   结算时先写明细、后写战报，所以战报落库后回填（见 processArrive 末尾 attachPvpReport）。
	//   0 = 没关联上（老数据 / 战报写入失败），管理端按「战报已清理」展示。
	ReportID  uint      `gorm:"default:0;index:idx_pvp_detail_report;comment:本场战报id(ezfy_report.id)" json:"report_id"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyPvpDetail) TableName() string { return "ezfy_pvp_detail" }

// EzfyPvpFoe 风云榜「同对手对战积分」记录（★ 2026-10-14 防小号互刷）
//
// 同一自然周内，同一玩家与**同一对手**对战的积分累计封顶（ezfyPvpFoeCap）：
// 已达上限后，再与该对手交战的积分不再计入榜单，但场次/灭敌/战损照常累计。
// 表内 points 为「与该对手本场累计已计积分」，榜单积分以 ezfy_pvp_week.points 为准。
type EzfyPvpFoe struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	Week      string    `gorm:"type:varchar(17);uniqueIndex:uk_pvp_foe_week;comment:自然周 20060102-20060102" json:"week"`
	UserID    uint      `gorm:"uniqueIndex:uk_pvp_foe_week;comment:玩家ID" json:"user_id"`
	FoeID     uint      `gorm:"uniqueIndex:uk_pvp_foe_week;comment:对手玩家ID" json:"foe_id"`
	Points    int64     `gorm:"default:0;comment:与该对手对战的已计积分(周内封顶)" json:"points"`
	Battles   int64     `gorm:"default:0;comment:与该对手交战次数" json:"battles"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyPvpFoe) TableName() string { return "ezfy_pvp_foe" }

// EzfyPvpExclude 榜单「不参与排名」名单（★ 2026-10-14 风云榜；★ 2026-10-16 拆成三份独立名单）
//
// 管理端「排行榜」的三个 tab（风云榜 / 战力榜 / 军衔声望榜）**各自维护一份**名单：
// Board 区分榜单（pvp / power / prestige），同一玩家可以「风云榜不参与、战力榜照常参与」。
// upsert 语义（board+user_id 唯一），删除某条 = 该榜恢复参与排名（不影响另两榜）。
//
// ★ 老数据（拆分前）没有 board 概念、三个榜共用一份 → 升级时统一落到 pvp，
//
//	再由 seed.EnsureEzfyPvpExcludeTable 一次性复制到 power/prestige，
//	保持「拆分前标一个 = 三个榜都不参与」的既有效果（不让玩家突然出现在另外两个榜上）。
type EzfyPvpExclude struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:uk_excl_board_user,priority:2;comment:玩家ID" json:"user_id"`
	Board     string    `gorm:"type:varchar(16);not null;default:'pvp';uniqueIndex:uk_excl_board_user,priority:1;comment:榜单类型 pvp风云榜/power战力榜/prestige军衔声望榜" json:"board"`
	Nickname  string    `gorm:"type:varchar(20);comment:昵称快照(便于后台识别)" json:"nickname"`
	Reason    string    `gorm:"type:varchar(100);comment:排除原因(后台备注)" json:"reason"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyPvpExclude) TableName() string { return "ezfy_pvp_exclude" }

// EzfyTitle 统帅称号定义（★ 2026-10-17）
//
// 管理端「二战征途 → 统帅称号」维护：称号名称 / 类型 / 有效期(天) / 创建日期。
// 玩家端统帅页（`games/ezfy?cur=playerinfo` 与他人统帅页）在**昵称下方、阵营上方**
// 展示生效中的称号徽章，样式按 Type 区分颜色（榜单=紫金，最醒目）。
//
// ★ 自动发放：风云榜**每日积分第一** → 自动获得「天下无双」（有效期 1 天）。
//
//	结算记录见 EzfyTitleDaily（每天一条，day 唯一，多实例安全），玩家持有见 EzfyTitleGrant。
type EzfyTitle struct {
	ID        uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name      string `gorm:"type:varchar(20);not null;uniqueIndex:uk_ezfy_title_name;comment:称号名称" json:"name"`
	Type      int    `gorm:"default:1;comment:称号类型(1战功 2榜单 3活动 4荣誉)" json:"type"`
	ValidDays int    `gorm:"default:1;comment:有效期(天,0=永久)" json:"valid_days"`
	Remark    string `gorm:"type:varchar(100);comment:备注/获得条件说明" json:"remark"`
	Status    int    `gorm:"default:1;comment:状态(1启用 0停用)" json:"status"`

	CreatedAt time.Time `gorm:"comment:称号创建日期" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyTitle) TableName() string { return "ezfy_title" }

// EzfyTitleGrant 玩家持有的称号（★ 2026-10-17）
//
// (title_id, user_id) 唯一：同一玩家同一称号只有一行，重复授予 = 续期（到期时间取较晚的，
// 永久(0) 不被缩短）。管理端「撤销」= 物理删除该行。
type EzfyTitleGrant struct {
	ID       uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	TitleId  uint   `gorm:"uniqueIndex:uk_title_grant_user;comment:称号ID(ezfy_title.id)" json:"title_id"`
	UserID   uint   `gorm:"uniqueIndex:uk_title_grant_user;index:idx_title_grant_user;comment:玩家ID" json:"user_id"`
	Nickname string `gorm:"type:varchar(20);comment:昵称快照(授予时)" json:"nickname"`
	// Source 来源：admin=管理端手动授予 / pvp_daily=风云榜每日第一自动发放
	Source    string    `gorm:"type:varchar(20);default:'admin';comment:来源 admin/pvp_daily" json:"source"`
	ExpireAt  int64     `gorm:"default:0;comment:到期时间戳(ms,0=永久)" json:"expire_at"`
	CreatedAt time.Time `gorm:"comment:获得时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyTitleGrant) TableName() string { return "ezfy_title_grant" }

// EzfyTitleDaily 风云榜「每日第一」称号结算记录（★ 2026-10-17）
//
// 每天一条（Day 唯一），用于**幂等**：多实例同时跑定时器时，靠 INSERT IGNORE 抢锁，
// 只有抢到的那台会真的发称号（重复发放虽无副作用，但会多打库）。
// UserID = 0 表示当天无有效冠军（无人上榜 / 全部被排除 / 称号被停用），也记一条避免反复重算。
type EzfyTitleDaily struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	Day       string    `gorm:"type:varchar(10);uniqueIndex:uk_title_daily_day;comment:结算日期 yyyymmdd" json:"day"`
	UserID    uint      `gorm:"index;comment:当日冠军玩家ID(0=当天无冠军)" json:"user_id"`
	Nickname  string    `gorm:"type:varchar(20);comment:昵称快照" json:"nickname"`
	Points    int64     `gorm:"default:0;comment:当日风云积分" json:"points"`
	TitleId   uint      `gorm:"default:0;comment:授予的称号ID(0=未授予)" json:"title_id"`
	CreatedAt time.Time `gorm:"comment:结算时间" json:"created_at"`
}

func (EzfyTitleDaily) TableName() string { return "ezfy_title_daily" }

// EzfyAccount 二战征途独立登录账号
//
// ★ 2026-10-12 二战账号体系与家园分离：二战登录不再复用家园 token。
//
//	GameID = 家园号(users.id)：二战全部业务表本就按这个 uid 存储 → 零迁移。
//	首次登录用「家园号 + 家园密码」校验，通过后把家园密码 hash 拷进本表，
//	此后二战自助管密码、不再依赖家园账号（为二战/家园拆开部署做准备）。
type EzfyAccount struct {
	GameID   uint   `gorm:"primaryKey;comment:游戏ID(=家园号)" json:"game_id"`
	Password string `gorm:"type:varchar(100);comment:二战登录密码hash(bcrypt)" json:"-"`
	Nickname string `gorm:"type:varchar(100);comment:玩家昵称" json:"nickname"`
	MSync    uint   `gorm:"default:1;comment:是否已从家园同步(1=已同步)" json:"msync"`

	// ★ 2026-10-12 管理端「玩家信息管理」要展示最后登陆时间 → 每次二战登录成功落库（ms, 0 = 从未记过）
	LastLoginAt int64 `gorm:"default:0;comment:最后登陆时间戳(ms,0=从未)" json:"last_login_at"`

	// ★ 2026-10-12 管理端「踢下线」：被强制下线的时间戳(ms，0 = 从未被踢)。
	//   二战 token 是无状态的（JWT），作废判据 = 「token 的 iat <= kick_at → 这个 token 已失效」，
	//   见 middleware.EzfyJWTAuth（带 60s 进程内缓存，踢出接口会立即刷新缓存）。
	//   玩家重新登录会签发新 token（iat > kick_at）→ 自然恢复。
	KickAt int64 `gorm:"default:0;comment:强制下线时间戳(ms,0=从未)" json:"kick_at"`

	// ★★ 2026-10-15 单点登录（只能一个 IP 在线，防脚本多 IP 同时刷号）：
	//   每次签发二战 token 都覆盖为「签发时刻」（秒级 ×1000，与 JWT 的 iat 精度对齐）。
	//   中间件判据 = 「token 的 iat < session_at → 该 token 是旧登录签发的 → 掉线」，
	//   新登录（session_at）会把所有更早的 token 全部顶下线 —— 其他地方登录，前面必掉线。
	SessionAt int64 `gorm:"default:0;comment:最近签发token时间戳(ms,0=从未)" json:"session_at"`

	// ★ 2026-10-15 最后登陆 IP 留痕（配合 session_at 排查「不同城市/IP 刷号」），默认空串。
	LastLoginIP string `gorm:"type:varchar(64);default:'';comment:最后登陆IP" json:"last_login_ip"`

	// ★ 2026-10-17 人机标识：登录路径性能敏感（每次登录都要判「人机不许登」），直接读本表列。
	IsBot     int       `gorm:"default:0;comment:是否人机(1=AI玩家)" json:"is_bot"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyAccount) TableName() string { return "ezfy_account" }

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
	// ★ 2026-10-24 民心/民怨最近一次结算(或直接变更)的时间戳(ms)：民心恢复/回落按它累进。
	//   与资源 last_time 分开 —— last_time 被写合并（约 60s）频繁推进，共用会让在线玩家
	//   每轮变化量都被 int() 舍成 0 且时钟已推进 → 民心永远不动（掉和涨都停住）。
	FeelSettleAt int64 `gorm:"comment:民心/民怨最近结算时间戳(ms)" json:"feel_settle_at"`
	TaxRate      int   `gorm:"comment:税率%" json:"tax_rate"` // 税率%
	// ★ 2026-09-28 「安抚 15 分钟可安抚一次」→ 记录上次安抚时间(ms)，0 = 从未安抚。
	PlacateTime int64 `gorm:"comment:上次安抚时间戳(ms)" json:"placate_time"`
	Pop         int64 `gorm:"comment:人口" json:"pop"`
	PopMax      int64 `gorm:"comment:人口上限" json:"pop_max"`
	// ★ 2026-10-13 用户规则「空闲人口不该一训练完就立刻恢复，要一点一点恢复到当前人口」：
	//
	//	★ 2026-10-18「用完就恢复」：**训练开始（消耗人口）当下**就把占用人口转成这里的
	//	「待恢复人口」，不等出厂；之后按 `ezfy_cfg_limit.pop_recover_seconds`
	//	（默认 1200 秒内恢复满）线性减回 0。
	//
	//	空闲人口 = 人口 − 建筑占用 − 训练中占用 − 待恢复人口 + 额外空闲人口（下限 0）。
	//	★ 人口未满民居上限时，召集同时增加总人口与空闲人口；人口已满时，
	//	召集/移民法案（ItemType 31）先扣减这笔待恢复人口，剩余计入 PopExtra
	//	（额外空闲人口，允许空闲暂时超过总人口，并按 pop_extra_decay_seconds 回落，默认 300 秒）。
	PopRecover int64 `gorm:"comment:待恢复人口(训练已消耗、还没回到空闲)" json:"pop_recover"`
	// ★ 2026-10-13 待恢复人口上次结算时间戳(ms)，0 = 还没结算过（首次结算只记时间、不恢复）。
	PopRecoverAt int64 `gorm:"comment:待恢复人口上次结算时间戳(ms)" json:"pop_recover_at"`
	// ★ 2026-10-22 本轮待恢复人口的恢复截止时间戳(ms)，0 = 当前没有恢复任务。
	//   有新训练消耗人口时重新起算（默认 1200 秒后恢复满）。
	PopRecoverEndAt int64 `gorm:"comment:待恢复人口恢复截止时间戳(ms)" json:"pop_recover_end_at"`
	// ★★ 2026-10-23 用户规则：人口达到民居上限后，召集仍可继续，但**不再增加总人口**，
	//   而是把这些人口记成「额外空闲人口」；因此空闲人口可以暂时超过总人口。
	//   超过正常空闲（人口 − 建筑占用 − 待恢复）的部分会在配置的
	//   pop_extra_decay_seconds（默认 300 秒）内线性回落。
	//   不占人口位置、不计入总人口，只参与空闲人口/征兵可用人口的计算。
	PopExtra      int64 `gorm:"comment:额外空闲人口(可超过总人口, 到时回落)" json:"pop_extra"`
	PopExtraAt    int64 `gorm:"comment:额外空闲人口上次结算时间戳(ms)" json:"pop_extra_at"`
	PopExtraEndAt int64 `gorm:"comment:额外空闲人口回落截止时间戳(ms)" json:"pop_extra_end_at"`
	Gold          int64 `gorm:"comment:黄金" json:"gold"`
	Food          int64 `gorm:"comment:粮食" json:"food"`
	Steel         int64 `gorm:"comment:钢铁" json:"steel"`
	Oil           int64 `gorm:"comment:石油" json:"oil"`
	Rare          int64 `gorm:"comment:稀矿" json:"rare"`
	GoldCap       int64 `gorm:"comment:黄金上限" json:"gold_cap"`
	FoodCap       int64 `gorm:"comment:粮食上限" json:"food_cap"`
	SteelCap      int64 `gorm:"comment:钢铁上限" json:"steel_cap"`
	OilCap        int64 `gorm:"comment:石油上限" json:"oil_cap"`
	RareCap       int64 `gorm:"comment:稀矿上限" json:"rare_cap"`
	LastTime      int64 `gorm:"comment:上次资源结算时间戳(ms)" json:"last_time"` // 上次资源结算时间戳(ms)
	// ★ 2026-10-12 用户规则「迁城道具加个冷却，针对**迁过城的城市**冷却 24 小时」：
	//   冷却记在**城市**上 —— 本城迁城时间 + 配置的冷却时长的截止时间戳(ms)，0 = 无冷却。
	//   别的城不受影响；时长由管理端「二战系统配置 → 迁城冷却(小时)」维护（0 = 关闭）。
	MoveCoolUntil int64 `gorm:"default:0;comment:迁城冷却截止时间戳(ms,0=无冷却)" json:"move_cool_until"`
	// ★ 2026-10-24 城墙耐久：-1 = 未初始化（首次读取时按围墙等级补满），0 = 已摧毁，
	//   >0 = 当前耐久。上限 = 围墙等级² × 20000（见 ezfyWallDurabilityMax）。
	//   只有「征服(3)」会打城墙耐久；「掠夺(2)」完全忽略城墙。
	WallHp int64 `gorm:"default:-1;comment:城墙当前耐久(-1=未初始化)" json:"wall_hp"`

	// 仓库保护配比(4 项资源的保护额度占比, 合计 ≤ 100; 默认各 25)
	// 每项保护额度 = 仓库等级对应保护总量 × 该项占比 / 100
	WareFood  int `gorm:"comment:仓库粮食" json:"ware_food"`
	WareSteel int `gorm:"comment:仓库钢铁" json:"ware_steel"`
	WareOil   int `gorm:"comment:仓库石油" json:"ware_oil"`
	WareRare  int `gorm:"comment:仓库稀矿" json:"ware_rare"`

	// 调整生产·开工率(0~100, 复刻原版 city/sourceSet.html)
	// 实际产量 = 基础产量 × 开工率 / 100。
	// ★ 2026-10-19 语义变更：0 = 停产（不生产，且该资源的资源建筑不再占用人口）。
	//   老数据 0 表示「从未调整」——上线前必须回填为 100（见上线 SQL），否则老城全停产。
	RateFood  int `gorm:"comment:比率粮食(0=停产)" json:"rate_food"`
	RateSteel int `gorm:"comment:比率钢铁(0=停产)" json:"rate_steel"`
	RateOil   int `gorm:"comment:比率石油(0=停产)" json:"rate_oil"`
	RateRare  int `gorm:"comment:比率稀矿(0=停产)" json:"rate_rare"`

	// ★ 2026-10-17 人机标识（profile 的冗余副本）：地图渲染 / 目标筛选 / 城市统计大量**直接查 city 表**、
	//   不 join profile —— 线上 1核1G+跨 WAN，冗余一列比每次 join 便宜。写入点统一从 profile 取。
	IsBot     int       `gorm:"default:0;index:idx_city_is_bot;comment:是否人机(1=AI玩家)" json:"is_bot"`
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

// EzfyUserTech 科技等级（用户级）：等级全城共用、无主城概念，所有城市操作平等。
// 研究队列仍存 ezfy_city_tech（status=1，按发起城市），等级记录统一存本表。
type EzfyUserTech struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId    uint      `gorm:"uniqueIndex:uk_user_tech;comment:用户ID" json:"user_id"`
	TechId    int       `gorm:"uniqueIndex:uk_user_tech;comment:科技ID" json:"tech_id"`
	Level     int       `gorm:"comment:等级" json:"level"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyUserTech) TableName() string { return "ezfy_user_tech" }

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
	// ★ 2026-10-21 军工厂队列归属：这条训练队列占用哪一座军工厂的队列位。
	//   每个军工厂的队列上限 = 自身等级；[全部工厂] 平分时只会选还有空位的厂。
	//   老数据 / 管理端补单可能为 0，读取时按「老队列」均匀回填到可用厂（不写库），
	//   所以已有队列的 EndTime 不会被重算。
	FactoryId int64 `gorm:"default:0;comment:军工厂建筑ID(0=老数据/不占厂)" json:"factory_id"`

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

// EzfyPreset 预设编队(司令部保存的出征模板: 军官+集结令+兵力, 不含目标/随军资源/宿营)
type EzfyPreset struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index:idx_preset_user;comment:用户ID" json:"user_id"`
	Name      string    `gorm:"type:varchar(50);comment:预设名称" json:"name"`
	Officer   string    `gorm:"type:varchar(255);comment:军官名(空=未指定)" json:"officer"`
	Gather    int       `gorm:"comment:集结令个数" json:"gather"`
	Troops    string    `gorm:"type:varchar(2000);comment:[{\"troopId\":1,\"count\":100}]" json:"troops"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyPreset) TableName() string { return "ezfy_preset" }

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
	// ★ 2026-10-24 带队军官 ID（默认 0）：命令与军官按 **id** 关联，解决同城同名军官
	//   （历史遗留重名）互相误伤 —— 出征校验/自愈/改名/结算都只认 id。
	//   存量在途订单由 seed.EnsureEzfyOrderOfficerIdBackfill 一次性回填；
	//   仍为 0 的只是「军官已改名/删除」的孤儿订单（名字也关联不到，运行时不误伤任何人）。
	OfficerId  uint   `gorm:"default:0;index;comment:带队军官ID" json:"officer_id"`
	StartTime  int64  `gorm:"comment:开始时间" json:"start_time"`
	ArriveTime int64  `gorm:"comment:Arrive时间" json:"arrive_time"`
	ReturnTime int64  `gorm:"comment:Return时间" json:"return_time"`
	Status     int    `gorm:"index:idx_status;default:0;comment:0行进 1驻守中 2返回 3完成 4阵亡" json:"status"` // 0行进 1驻守中 2返回 3完成 4阵亡
	Result     string `gorm:"type:varchar(3000);comment:返回部队JSON/采集标记" json:"result"`                // 返回部队JSON/采集标记
	// ★ 采集到的资源先记在部队身上（待带回），只有「返航到达」才入城；
	//   容量上限 = 部队各兵种 carry 之和。JSON: {"food":..,"steel":..,"oil":..,"rare":..,"gold":..}
	Carry     string `gorm:"type:varchar(500);comment:携带" json:"carry"`
	Resources string `gorm:"type:varchar(500);comment:资源" json:"resources"`
	// ★ 2026-09-28 采集起始时间(ms, 0=未采集/空闲驻守): 用于「累计采集时长」展示。
	//   每次开始采集(StartCollect/一键采集)时写入, 全周期累计算到现在(按小时/分钟累计)。
	CollectStart int64 `gorm:"comment:采集起始时间(ms)" json:"collect_start"`
	OilUsed      int64 `gorm:"comment:石油已用" json:"oil_used"`
	WaitMin      int   `gorm:"comment:宿营分钟数(0~1440), 到达后停留该时长再返航" json:"wait_min"` // 宿营分钟数(0~1440), 到达后停留该时长再返航
	// ★ 2026-10-07 自动战斗（出征页可配）：1=到达即自动打完（无需指挥）；0=到达后进指挥室等玩家指挥。
	//   打野地 / AI 寇城（target_type != 3）默认 1；打玩家城市**强制 0**（真人守方，必须留指挥机会，
	//   前端该项只读）。例外：打玩家城时若**攻守双方都不在线**，仍直接自动结算（见 processArrive）。
	// ★★ 2026-10-08 去掉 `default:1` 标签：
	//   GORM 对带 default 标签的 int 字段在 INSERT 时会**跳过 0 值**，让数据库默认 1 落库 →
	//   出征页选了「否」(0) 也会被存成 1 → 到达被自动打完、进不了指挥室（用户反馈的 bug）。
	//   去掉后 GORM 会显式写入玩家所选值，0/1 都原样落库。
	AutoBattle int `gorm:"comment:自动战斗 1是 0否" json:"auto_battle"`

	// ★ 指挥室（实时战斗）打完的结果（ezfyBattleResult 的 JSON）。
	//   非空 = 这场仗已经由玩家在指挥室里打完 → processArrive 跳过模拟、直接拿它结算，
	//   战报/掠夺/经验/征服等战后逻辑全部复用，不重复实现一套。
	BattleResult string `gorm:"type:mediumtext;comment:战斗结果" json:"battle_result"`

	// ★ 2026-09-30 计谋「神兵天降/战略转移」使用标记（位标记，每支部队各计谋限一次）：
	//   bit0 = 神兵天降已用（去程减时）
	//   bit1 = 战略转移已用（回程减时）
	SchemeUsed int `gorm:"default:0;comment:计谋使用位标记 bit0神兵天降 bit1战略转移" json:"scheme_used"`

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyOrder) TableName() string { return "ezfy_order" }

// EzfyBattle 战场（实时指挥室）
//
// ★ 2026-09-22 「实现指挥功能」（入口：军情 → 军队动态 → [指挥]）：
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
	// ★ 「指挥不是指挥全部，自己带的兵种都能指挥，就是单独指挥」。
	//   没给的兵种回落司令部「兵种战斗配置」；键 0 = 旧格式遗留的「全军统一指令」。
	AtkCmd string `gorm:"type:varchar(500);default:'';comment:攻击指令" json:"atk_cmd"`
	// DefUserID 守方玩家 uid（target_type=3 玩家城；target_type=1/2 有玩家驻军的附属野地；
	// 0 = 无真人守方，如中立野地/寇城/AI）。
	// ★ 2026-09-23 「敌人打自己，自己也能指挥」—— 防守方据此找回并进入战场。
	DefUserID uint `gorm:"index:idx_battle_def;comment:守方用户ID" json:"def_user_id"`
	// DefCmd 守方**逐兵种**指令表，JSON 同 AtkCmd（玩家守城时指挥守军用；AI 为空）。
	DefCmd string `gorm:"type:varchar(500);default:'';comment:防御指令" json:"def_cmd"`
	// State 战场快照（ezfyBattleSnapshot 的 JSON，含双方兵力/位置/加成/日志）
	State string `gorm:"type:mediumtext;comment:状态" json:"-"`
	// RoundStart 本回合开始时间(ms)：过了回合时长就推进一回合
	RoundStart int64 `gorm:"comment:RoundStart 本回合开始时间(ms)：过了回合时长就推进一回合" json:"round_start"`
	// ★ 2026-10-06 保存配置 = 本回合锁定：AtkLock/DefLock 1=已锁定（禁改指令，双方都锁或到点即结算）。
	//   锁定窗口内可取消(置0)重新配置；下回合开始时清零。
	AtkLock int `gorm:"default:0;comment:攻方本回合是否已锁定配置 0/1" json:"atk_lock"`
	DefLock int `gorm:"default:0;comment:守方本回合是否已锁定配置 0/1" json:"def_lock"`

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyBattle) TableName() string { return "ezfy_battle" }

type EzfyReport struct {
	ID     uint `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID uint `gorm:"index:idx_user;comment:用户ID" json:"user_id"`
	// ★ 2026-10-07 补 default:0：此前无 default → AutoMigrate 建的是可空列，
	//   存量 6000+ 行会变 NULL，GORM 扫进 int64 直接报 converting NULL to int64。
	CityId     int64  `gorm:"index:idx_user_city;default:0;comment:所属城市ID(军情按当前城过滤, 0=旧数据待解析)" json:"city_id"`
	OrderId    int64  `gorm:"comment:订单ID" json:"order_id"`
	ReportType int    `gorm:"comment:1侦察 2掠夺 3征服 4战斗 5采集/派遣 6系统" json:"report_type"` // 1侦察 2掠夺 3征服 4战斗 5采集/派遣 6系统
	Title      string `gorm:"type:varchar(255);comment:标题" json:"title"`
	// ★ 2026-10-08 持久化「真实战报类型名」，供管理端战报查询做 SQL 过滤（不等价于粗粒度
	//   report_type：采集/运输/增援/派遣共用一个编号）。写战报时由 EzfyReportTypeName 落库，
	//   存量行用 cmd/backfilltype 一次性回填，避免管理端捞整表进内存过滤。
	//   二级索引 (type_name,id) 由 seed.ensureEzfyIndex 幂等创建（AutoMigrate 对存量表不补二级索引）。
	TypeName string `gorm:"type:varchar(20);comment:战报类型名" json:"type_name"`
	Content  string `gorm:"type:longtext;comment:内容" json:"content"`
	Detail   string `gorm:"type:longtext;comment:Detail" json:"detail"`
	IsRead   int    `gorm:"default:0;comment:是否已读" json:"is_read"`

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

// EzfyRansom 赎城请求：被占城市原主人发起赎回，占领方同意后返还城市、金额归占领方。
//
// ★ 2026-10-07 新增。发起即扣钻石（押金），拒绝/撤销退回，同意后金额转给占领方。
//
//	Status: 0 待处理 / 1 已同意(城市已返还) / 2 已拒绝(押金已退) / 3 已撤销(押金已退)
type EzfyRansom struct {
	ID        uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	CityId    int64  `gorm:"comment:被赎城市ID" json:"city_id"`
	OccupyId  uint   `gorm:"comment:关联占领记录ID(ezfy_occupy.id)" json:"occupy_id"`
	DefUserId uint   `gorm:"comment:发起人(被占城市原主人)" json:"def_user_id"`
	AtkUserId uint   `gorm:"comment:占领方(同意人)" json:"atk_user_id"`
	CityName  string `gorm:"type:varchar(50);comment:城市名称" json:"city_name"`
	X         int    `gorm:"comment:X坐标" json:"x"`
	Y         int    `gorm:"comment:Y坐标" json:"y"`
	Cost      int64  `gorm:"comment:发起时快照的赎城金额(钻石)" json:"cost"`
	Status    int    `gorm:"default:0;comment:0待处理 1已同意返还 2已拒绝 3已撤销" json:"status"`

	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyRansom) TableName() string { return "ezfy_ransom" }

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
	// ★ 2026-09-25 「军团积分」：军团战绩总积分（成员在军团交战期获胜累加，军团商城可查看）
	Points int64 `gorm:"default:0;comment:军团总积分" json:"points"`
	// ★ 2026-09-30 「入团需审核」：0=无需审核直接入团(默认)，1=需军团长审核。
	NeedReview int `gorm:"default:0;comment:入团是否需审核(0=直接入团 1=需军团长审核)" json:"need_review"`

	// ★ 2026-10-17 人机军团标识（军团管理列表要能区分）
	IsBot     int       `gorm:"default:0;comment:是否人机军团" json:"is_bot"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyCorps) TableName() string { return "ezfy_corps" }

// EzfyCorpsApply 入团申请（军团长开启审核后，玩家申请入团走此表待审）
//
// ★ 2026-09-30 「进军团需要审核」：open 军团直接入团（不走本表），
//
//	开启审核的军团，玩家申请先落这里，军团长 [通过]/[拒绝] 后入团或驳回。
//	Status：0 待审 / 1 通过(已入团) / 2 拒绝。
type EzfyCorpsApply struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	CorpsId   uint      `gorm:"index:idx_apply_corps;comment:军团ID" json:"corps_id"`
	UserId    uint      `gorm:"index:idx_apply_user;comment:申请用户ID" json:"user_id"`
	Status    int       `gorm:"default:0;comment:0待审 1通过 2拒绝" json:"status"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyCorpsApply) TableName() string { return "ezfy_corps_apply" }

type EzfyCorpsMember struct {
	ID       uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	CorpsId  uint   `gorm:"index:idx_corps;comment:军团ID" json:"corps_id"`
	UserId   uint   `gorm:"uniqueIndex:uk_user;comment:用户ID" json:"user_id"`
	IsLeader int    `gorm:"default:0;comment:是否首领" json:"is_leader"`
	Title    string `gorm:"type:varchar(20);comment:标题" json:"title"`
	// ★ 2026-09-25 「军团商城货币 = 成员个人军团积分」：成员个人战绩积分（商城消费用它扣）
	Points    int64     `gorm:"default:0;comment:个人军团积分" json:"points"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyCorpsMember) TableName() string { return "ezfy_corps_member" }

// EzfyCorpsRelation 军团外交关系（军团长标记；友好/敌对均双向各写一条）
//
// ★ 2026-09-25 「军团之间可标记友好/敌对，双向记录，友好敌对都可以宣战」。
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
// ★ 2026-09-25 生效期间双方**军团成员之间**可互相掠夺/征服，无需个人宣战；
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
// ★ 2026-09-25 商品分两类 —— 资源包（food/steel/oil/rare/gold）与道具
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

// ★ 2026-09-26 线上「无限签到刷资源」事故根因：
//
//	`uniqueIndex:uk_user_date` 早期只写在 SignDate 一个字段上 → AutoMigrate 建出的是
//	`UNIQUE KEY uk_user_date (sign_date)` 单列唯一索引，等于**全服每天只允许一条签到记录**。
//	除了当天第一个玩家，其余人的 INSERT 全部撞重复键失败；而旧 Sign 处理器又忽略了
//	Create 的 error、照样发奖励 → 有人反复请求就能一直领资源，自己的签到记录还从没落库
//	（前端因此永远显示「未签到」）。
//
//	现在把索引标签同时挂到 UserId + SignDate 上，建出 (user_id, sign_date) 复合唯一索引；
//	老库上那个建错的同名索引由 seed 的 fixEzfySignIndex 显式 DROP 再重建。
type EzfySign struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId    uint      `gorm:"uniqueIndex:uk_user_date,priority:1;comment:用户ID" json:"user_id"`
	SignDate  string    `gorm:"type:varchar(10);uniqueIndex:uk_user_date,priority:2;comment:签到日期" json:"sign_date"`
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

// EzfyTreasureSign 宝物签到记录：独立于每日签到的宝物日签到。
//
//	7 天一轮，逢第 5/6/7 天多给宝物（懒人不采集也能攒晋升宝物）。
type EzfyTreasureSign struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId    uint      `gorm:"uniqueIndex:uk_trs_user_date,priority:1;comment:用户ID" json:"user_id"`
	SignDate  string    `gorm:"type:varchar(10);uniqueIndex:uk_trs_user_date,priority:2;comment:签到日期" json:"sign_date"`
	SignCount int       `gorm:"comment:连续宝物签到天数" json:"sign_count"` // 连续宝物签到天数
	NReward   string    `gorm:"type:varchar(255);comment:本次签到获得的宝物名(逗号分隔)" json:"n_reward"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyTreasureSign) TableName() string { return "ezfy_treasure_sign" }

// EzfyCityEffect 城市效果：1增产 2免战 3恫疑虚喝 4建造队列+Param1层
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

// EzfyLoveCard 为爱发电卡 激活/领取记录
// 卡由管理端发放即生效，有效期 30 天，每天可领固定钻石（普通 150 / 高级 200）。
// 领取按「完整天数累加、封顶总天数」：漏领的天数会在之后领取时一次性补齐，
// 总领取天数不超过 TotalDays(30)。
type EzfyLoveCard struct {
	ID            uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId        uint      `gorm:"index:idx_love_user;comment:用户ID" json:"user_id"`
	CfgId         int       `gorm:"comment:卡类型(ezfy_cfg_item.id)" json:"cfg_id"`
	Name          string    `gorm:"type:varchar(50);comment:卡片名称快照" json:"name"`
	DailyDiamond  int64     `gorm:"default:150;comment:每日钻石数量" json:"daily_diamond"`
	StartTime     int64     `gorm:"comment:生效开始时间戳(ms)" json:"start_time"`
	EndTime       int64     `gorm:"comment:失效时间戳(ms, Start+30天)" json:"end_time"`
	TotalDays     int       `gorm:"default:30;comment:总天数" json:"total_days"`
	ClaimedDays   int       `gorm:"default:0;comment:已领取天数" json:"claimed_days"`
	LastClaimTime int64     `gorm:"comment:上次领取时间戳(ms)" json:"last_claim_time"`
	CreatedAt     time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (EzfyLoveCard) TableName() string { return "ezfy_love_card" }

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
//
// ★ 2026-10-07 CityId = **挂单所在城市**：卖资源时资源是从「当前操作城市」扣的，
//
//	成交收款 / 下架退款都必须回到**同一座城**（用户反馈 bug：切城后挂单，成交的黄金却跑进了主城）。
//	老数据该列为 NULL/0 → 回落卖家主城（id 最小），保持兼容。
type EzfyExchange struct {
	ID         uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	SellerId   uint   `gorm:"index:idx_seller;comment:卖家ID" json:"seller_id"`
	CityId     int64  `gorm:"comment:挂单所在城市ID" json:"city_id"`
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
// ★ 2026-09-24 资源包模板原来硬编码在前端，选中还有「触发全选」的 bug；
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

// EzfyStarCapPct 各星级的属性上限系数（百分比），用于推导「普通军官不得超过同星级名将」
//
// ★ 2026-09-26 用户定的属性规则：
//   - 普通军官的任何一项属性都**不得超过同星级名将**（有的名将军事高、有的后勤高、
//     有的学识高 → **逐项**比较，不是比总和）；
//   - 且**星级递减**：一星 ≤ 二星 ≤ 三星 ≤ 四星 ≤ 五星。
//
// 实现 = 「名将逐项最大值 × 本表系数%」，天然同时满足这两条：
// 系数单调递增 → 上限随星级递增；五星 = 100% → 五星上限恰好等于名将。
//
// 代入线上名将 max（军事 300 / 后勤 336 / 学识 300）得到：
//
//	1星 60 /  67 /  60      2星 90 / 100 /  90
//	3星 135/ 151 / 135      4星 210 / 235 / 210      5星 300 / 336 / 300
//
// （与原来的硬编码 60/90/130/190/260 基本一致，5 星由 260 放宽到名将水平。）
var EzfyStarCapPct = map[int]int{
	1: 20,
	2: 30,
	3: 45,
	4: 70,
	5: 100,
}

// EzfyStarCap 某星级下三维的上限 [军事,后勤,学识] = 名将逐项 max × 系数%
//
// gMil/gLog/gLea 传「名将(kind=2) 的逐项最大值」。取整后下限 1，避免出现 0 上限。
func EzfyStarCap(star, gMil, gLog, gLea int) [3]int {
	if star < 1 {
		star = 1
	}
	if star > 5 {
		star = 5
	}
	pct := EzfyStarCapPct[star]
	if pct <= 0 {
		pct = 100
	}
	one := func(v int) int {
		r := v * pct / 100
		if r < 1 {
			r = 1
		}
		return r
	}
	return [3]int{one(gMil), one(gLog), one(gLea)}
}

// EzfyCfgGeneral 军官池（源自 inithebing.sql cfg_general 31 条名将 + 管理端可新增普通军官）
//
// ★ 2026-09-22 军官分两类，**都在这张池子里**，由管理端统一维护：
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
	// ★★ 2026-10-24 名将组合技能改为**可学多个**（用户要求「能学多个，现在只能学一个不对」）：
	//   组合技池 ID 列表，JSON 数组字符串（如 "[1,5,9]"；空 = 没有组合技）。
	//   只有 kind=2 名将可配，普通军官池一律空；效果按**军官等级**成长，见 ezfy_combo.go。
	ComboSkillIds string `gorm:"type:varchar(255);default:'';comment:名将组合技能池ID列表(JSON数组)" json:"combo_skill_ids"`
	// 旧字段（单技能时代）只作**回退**：combo_skill_ids 为空时按它读，管理端保存后清 0。
	ComboSkillId int `gorm:"default:0;comment:旧字段(单技能时代)仅回退用" json:"-"`
}

func (EzfyCfgGeneral) TableName() string { return "ezfy_cfg_general" }

// EzfyCfgGeneralSkill 名将组合技能**池**（★ 2026-10-23）。
//
// 「组合技能」是**系统专属**的一类战法：普通军官（kind=1）学不到，
// 只能由管理端把池子里的技能搭配给名将（ezfy_cfg_general.combo_skill_ids，
// JSON 数组，★ 2026-10-24 一个名将可配多个）。
//
// 设计口径（参考率土之滨战法，但只取「每回合按概率发动」这一层，不引入拆解/连携）：
//   - 通用：一套效果对所有兵种生效，不区分陆/海/空（兵种专属还是走通用技能那套）；
//   - 按等级成长：组合技等级由**军官等级**分档（默认每 35 级 +1，350 级满级 = 10 级），
//     效果值 = BaseValue + (等级-1)×GrowPerLevel；
//   - 概率发动：每回合开始按 ProcPct 掷一次，命中后持续 Rounds 回合（1=只本回合）；
//   - 数值克制：单技能满级不超过 30%，避免「一个组合技顶一套装备」。
type EzfyCfgGeneralSkill struct {
	ID   int    `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name string `gorm:"type:varchar(50);comment:技能名(管理端可改)" json:"name"`
	Des  string `gorm:"type:varchar(255);comment:说明(管理端可改)" json:"des"`
	// Kind 效果类型（1~9，见 ezfyComboKindName）：
	//   1 攻击加成%  2 防御加成%  3 射程加成%  4 速度加成%
	//   5 受到伤害减免%  6 破防%（无视目标 X% 防御）  7 回合治疗%（恢复已损兵力 X%）
	//   8 连击（发动后攻击追加一次）  9 反击回合 +N
	Kind int `gorm:"default:1;comment:效果类型1~9" json:"kind"`
	// BaseValue 1 级效果值（百分点 / 回合数 / 百分比，含义随 Kind）
	BaseValue int `gorm:"default:10;comment:1级效果值" json:"base_value"`
	// GrowPerLevel 每高 1 级增加多少（组合技等级 +1，与星级无关）
	GrowPerLevel int `gorm:"default:1;comment:每级增幅" json:"grow_per_level"`
	// MaxLevel 组合技等级上限（默认 10）
	MaxLevel int `gorm:"default:10;comment:组合技等级上限" json:"max_level"`
	// LevelPerStep 每多少**军官等级**升 1 级组合技（默认 35：35/70/…/350 → 1~10 级）
	//
	// ★★ 2026-10-23 用户纠正：名将满级 350 级、且**所有名将都是 5 星**，
	//   用星级当等级没有成长空间 → 改成按**军官等级**分档；
	//   同日二次调整：上限 5 → 10（步长 70 → 35），成长更细、满级数值不变。
	LevelPerStep int `gorm:"default:35;comment:每多少军官等级+1级" json:"level_per_step"`
	// ProcPct 每回合发动概率%（0~100）
	ProcPct int `gorm:"default:35;comment:发动概率%" json:"proc_pct"`
	// Rounds 发动后持续回合（1=只本回合；>1 = 含本回合在内的连续 N 回合）
	Rounds int `gorm:"default:1;comment:持续回合" json:"rounds"`
	// Quality 品质 1普通 2精良 3稀有 4传说（只影响展示色）
	Quality int `gorm:"default:2;comment:品质1~4" json:"quality"`
	Enabled int `gorm:"default:1;comment:1=启用 0=下架" json:"enabled"`
	SortNo  int `gorm:"default:0;comment:排序" json:"sort_no"`
}

func (EzfyCfgGeneralSkill) TableName() string { return "ezfy_cfg_general_skill" }

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
	// Shelf 商城上下架开关（★ 2026-10-14）：1=上架（商城可见）0=下架（商城隐藏）。
	//   ★ 与价格解耦：下架**不清价**（价格留着，重新上架即恢复原价）。
	//   ★ 默认 1：新增装备默认上架；存量老行补列后也是 1（保持既有可见性不变）。
	//   ★ 玩家端上架判定从「价格>0」改为「shelf=1 且 价格>0」（见 ezfy_officer.go::equipShopList）。
	Shelf int `gorm:"default:1;comment:Shelf 商城上下架 1=上架 0=下架" json:"shelf"`
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
// ★ 2026-09-22 军官穿的装备有套装，玩家用黄金或钻石在商城购买。
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

// ============ 宝箱（2026-09-22 ）============
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
	// MinDraws 保底抽取次数（★ 2026-10-14）：玩家抽该宝箱累计满 MinDraws 次仍没出此奖品，
	// 则下一次必出此奖品（出后清零重新累计）。1 = 不启用保底（默认，普通奖品）；传说类按品质 50~500。
	MinDraws int    `gorm:"default:1;comment:MinDraws 保底抽取次数(1=不启用,>=2生效,出后清零)" json:"min_draws"`
	Des      string `gorm:"type:varchar(200);default:'';comment:描述" json:"des"`
}

func (EzfyCfgChestItem) TableName() string { return "ezfy_cfg_chest_item" }

// EzfyCfgScheme 计谋配置（2026-09-22 ）
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
	//   2 = 神兵天降（队伍去程/行进中剩余时间减 80%）
	//   3 = 战略转移（队伍回程减 360 分钟）
	//   4 = 恫疑虚喝（自己所有城市生效1小时，被侦查时展示随机兵种1亿假兵）
	//   5 = 隐真示假（自己所有城市生效1小时，被侦查时展示随机兵种1000内假兵力）
	//   6 = 秘密策反（按目标城市坐标，降低敌方城守忠诚）
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
// ★ 2026-09-22 **加点与升级只影响玩家自己的军官实例，绝不回写军官池**。
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
	// ★ 2026-09-29 市长/城守在任期间被动经验结算基准时间（NULL = 未初始化/不领在职经验）
	DutyExpAt *time.Time `gorm:"index;comment:任职被动经验结算基准时间" json:"duty_exp_at"`
	// ★ 2026-09-22 新增：原始属性 + 可用属性点
	//
	// 带 default:0 是为了让 AutoMigrate 建出 `NOT NULL DEFAULT 0` 的列
	// （不带 default 时建出来是 NULL，老行会读到 NULL → JSON 里出现 null）。
	BaseMilitary  int `gorm:"default:0;comment:基础军事" json:"base_military"`
	BaseLogistics int `gorm:"default:0;comment:基础后勤" json:"base_logistics"`
	BaseLearning  int `gorm:"default:0;comment:基础学识" json:"base_learning"`
	FreePoints    int `gorm:"default:0;comment:免费点数" json:"free_points"`
	// ★ 2026-09-30 升星累计加的点数（每次升星成功 军事+后勤+学识 之和累加）。
	//   升星加的是当前属性、不动 base_*，洗点时差额会自动退回 ——
	//   这里持久记录「其中多少点是升星来的」，玩家洗点前一眼能看懂。
	StarPoints int `gorm:"default:0;comment:升星累计加点" json:"star_points"`
	// ★★ 2026-10-25 升星加点的**逐项**明细（军事/后勤/学识各多少），恒等于 StarPoints 的拆分。
	//
	//	注意口径（反复确认过）：**洗点会把三维全部洗回军官池属性**，
	//	「玩家加的点 + 升星加的点」一起退回成待分配点 —— 所以洗点时这四项一并归零。
	//	保留它们是为了「升星加了多少」有据可查（不参与洗点计算）。
	StarMilitary  int `gorm:"default:0;comment:升星累计加军事" json:"star_military"`
	StarLogistics int `gorm:"default:0;comment:升星累计加后勤" json:"star_logistics"`
	StarLearning  int `gorm:"default:0;comment:升星累计加学识" json:"star_learning"`
	// ★ 2026-10-05 获取途径：取值见下列 EzfyOfficerSource* 枚举（每种来源独立值，便于后续扩展）。
	//   管理端军官列表据此标注来源（抢玩家获取、野地俘虏……）。
	Source int `gorm:"default:0;comment:获取途径 0未标注 1系统发放 2军校招募 3野地俘虏 4抢玩家获取" json:"source"`
	// ★★ 2026-10-06 用户规则：军官一律**逻辑删除**，不要物理删除。
	//
	//	被俘 / 释放 / 流放 后原行仍留在库里（deleted_at 非空），等级与属性可追溯、可恢复 ——
	//	否则 PvP 抢将「先删防守方、再建俘虏」一旦中间失败，原军官的等级就永远找不回来了
	//	（2026-10-06 事故：11 名军官被物理删掉，只能靠 10-05 的全库备份还原）。
	//	GORM 语义：`Delete` 自动变成 `UPDATE ... SET deleted_at=NOW()`，
	//	所有 `Find/First/Count` 自动带 `deleted_at IS NULL`；要看已删行用 `.Unscoped()`。
	DeletedAt gorm.DeletedAt `gorm:"index;comment:逻辑删除时间(NULL=有效)" json:"-"`
}

// 军官获取途径（ezfy_officer.source）—— 每种来源独立枚举值，新增来源往后追加即可（勿改旧值）。
const (
	EzfyOfficerSourceUnknown  = 0 // 未标注（历史数据 / 其它途径）
	EzfyOfficerSourceGrant    = 1 // 系统发放（管理端发放 / 一键生成）
	EzfyOfficerSourceRecruit  = 2 // 军校招募
	EzfyOfficerSourceWildland = 3 // 野地俘虏（野地 / 寇城 / 活动野地守将）
	EzfyOfficerSourcePvp      = 4 // 抢玩家获取（PvP 打赢后敌方守将叛逃收编）
)

// EzfyOfficerSourceName 返回获取途径的中文标注（未标注 / 未知返回空串）
func EzfyOfficerSourceName(src int) string {
	switch src {
	case EzfyOfficerSourceGrant:
		return "系统发放"
	case EzfyOfficerSourceRecruit:
		return "军校招募"
	case EzfyOfficerSourceWildland:
		return "野地俘虏"
	case EzfyOfficerSourcePvp:
		return "抢玩家获取"
	}
	return ""
}

func (EzfyOfficer) TableName() string { return "ezfy_officer" }

// EzfyEquipment 玩家装备背包（野地掉宝 / 商城购买入库，可穿戴到军官）
//
// ★★ 装备只认两个外键：`user_id`（属于哪个账号）+ `officer_id`（穿在哪名军官身上，0=在背包）。
//
//	**和城市没有任何关系** —— 弃城 / 占城摧毁 / 派遣 / 俘虏转移都不应影响装备的归属。
type EzfyEquipment struct {
	ID     uint `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId uint `gorm:"index:idx_user;comment:用户ID" json:"user_id"`
	// CityId 已废弃（2026-10-18 起恒 0）：历史「获得时的出生城」，穿戴/俘虏/派遣后不会更新。
	// 曾两次被误当删除依据（弃城丢 305 件、占城摧毁丢 3 件）→ 代码里禁止按它删/迁装备。
	CityId    int64     `gorm:"comment:已废弃（历史出生城，恒0）；装备归属只看 user_id，禁止按它删除/迁移" json:"city_id"`
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
	// ★ 2026-09-29 被俘军官的随身装备：记录**原归属玩家**，
	//   收编后归当前玩家、释放时按此返还给原玩家（>0 才参与返还；0=普通装备）。
	OriginalUserId uint `gorm:"default:0;comment:被俘装备的原归属玩家" json:"original_user_id"`
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

// ★ 2026-09-28 军校免费刷新次数从「每天 5 次」改为「每 1 小时 5 次」。
//
//	RecruitDate 存的是**小时周期 key**（time.Now().Format("2006010215")，10 位 YmdH，装得进 varchar(10)），
//	整点窗口变化即视为新周期 → (user_id, recruit_date) 唯一索引天然按小时切分，无需扩列迁移。
type EzfyRecruit struct {
	ID           uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserId       uint      `gorm:"uniqueIndex:uk_role_date;comment:用户ID" json:"user_id"`
	RecruitDate  string    `gorm:"type:varchar(10);uniqueIndex:uk_role_date;comment:招募周期(2006010215整点窗)" json:"recruit_date"`
	RefreshCount int       `gorm:"comment:Refresh数量" json:"refresh_count"`
	Candidates   string    `gorm:"type:text;comment:JSON: 随机军官候选" json:"candidates"` // JSON: 随机军官候选
	CreatedAt    time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt    time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (EzfyRecruit) TableName() string { return "ezfy_recruit" }
