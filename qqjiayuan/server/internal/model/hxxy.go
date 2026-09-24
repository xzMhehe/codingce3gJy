package model

import "time"

// 幻想西游（复刻 PHP WAP 版"幻想西游"，全系统）
// 数据表统一前缀 hxxy_；游戏用户 hxxy_players 通过 user_id 关联家园 users 表；
// 游戏内独立货币：银两 money + 金豆 beans（与家园 G 币不通）。
// 装备部位(分类)：1法宝 2坐骑 3武器 4护甲 5头盔 6靴子 7项链 8手镯，9+为宝石等。
// 门派：1将军府 2龙宫 3月宫(限女) 4方寸山 5普陀山(限男)。

// HxxyPlayer 游戏玩家档案（一个社区用户对应一条，建角创建）
type HxxyPlayer struct {
	ID     uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID uint   `gorm:"uniqueIndex;comment:关联 users.id" json:"user_id"` // 关联 users.id
	Name   string `gorm:"type:varchar(30);uniqueIndex;comment:名称" json:"name"`
	Sex    int    `gorm:"default:1;comment:1男 2女" json:"sex"`                   // 1男 2女
	Sect   int    `gorm:"default:1;comment:1将军府 2龙宫 3月宫 4方寸山 5普陀山" json:"sect"` // 1将军府 2龙宫 3月宫 4方寸山 5普陀山
	Level  int    `gorm:"default:1;comment:等级" json:"level"`
	Exp    int    `gorm:"default:0;comment:当前等级累计经验" json:"exp"` // 当前等级累计经验
	// 修炼（开关：开启后战斗经验存入修炼经验，用于四线修炼升级；关闭直接获得等级经验）
	XiulianExp    int `gorm:"default:0;comment:修炼（开关：开启后战斗经验存入修炼经验，用于四线修炼升级；关闭直接获得等级经验）" json:"xiulian_exp"`
	XiulianSwitch int `gorm:"default:0;comment:XiulianSwitch" json:"xiulian_switch"`
	// 人物修炼四线等级（1血 2攻 3魔 4防，复刻原版 xl_ini 修炼等级）
	XlLv1 int `gorm:"default:0;comment:人物修炼四线等级（1血 2攻 3魔 4防，复刻原版 xl_ini 修炼等级）" json:"xl_lv1"`
	XlLv2 int `gorm:"default:0;comment:修炼Lv2" json:"xl_lv2"`
	XlLv3 int `gorm:"default:0;comment:修炼Lv3" json:"xl_lv3"`
	XlLv4 int `gorm:"default:0;comment:修炼Lv4" json:"xl_lv4"`
	// 西游声望（修炼升级消耗）
	Sw int64 `gorm:"default:0;comment:西游声望（修炼升级消耗）" json:"sw"`
	HP int   `gorm:"default:0;comment:当前气血" json:"hp"` // 当前气血
	MP int   `gorm:"default:0;comment:当前法力" json:"mp"` // 当前法力
	// 货币（游戏内独立）
	Money int64 `gorm:"default:0;comment:银两（银两）" json:"money"` // 银两
	Bank  int64 `gorm:"default:0;comment:银行存款" json:"bank"`    // 银行存款
	Beans int   `gorm:"default:0;comment:金豆" json:"beans"`     // 金豆
	// VIP 练级祝福（分钟）
	Vip    int   `gorm:"default:0;comment:VIP 练级祝福（分钟）" json:"vip"`
	VipExp int64 `gorm:"default:0;comment:VIP经验" json:"vip_exp"`
	// VIP 会员等级（0~20，充值等级，决定每日兑换万能果/金豆数量）
	VipLv int `gorm:"default:0;comment:VIP 会员等级（0~20，充值等级，决定每日兑换万能果/金豆数量）" json:"vip_lv"`
	// 头衔（佩戴）
	TitleID uint `gorm:"default:0;comment:头衔（佩戴）" json:"title_id"`
	// 容量
	BagCap int `gorm:"default:50;comment:容量" json:"bag_cap"`
	WhCap  int `gorm:"default:50;comment:Wh上限" json:"wh_cap"`
	// 恶名（PK）
	Emz int `gorm:"default:0;comment:恶名（PK）" json:"emz"`
	// 位置：map_x=区域(dtx) map_y=节点(dty)
	MapX int `gorm:"default:0;comment:位置：map_x=区域(dtx) map_y=节点(dty)" json:"map_x"`
	MapY int `gorm:"default:0;comment:地图Y坐标" json:"map_y"`
	// 已穿装备（按分类1-8 存 hxxy_bag.id，0=未穿）
	EqSlot1 uint `gorm:"default:0;comment:装备槽位1（法宝）" json:"eq_slot1"` // 法宝
	EqSlot2 uint `gorm:"default:0;comment:装备槽位2（坐骑）" json:"eq_slot2"` // 坐骑
	EqSlot3 uint `gorm:"default:0;comment:装备槽位3（武器）" json:"eq_slot3"` // 武器
	EqSlot4 uint `gorm:"default:0;comment:装备槽位4（护甲）" json:"eq_slot4"` // 护甲
	EqSlot5 uint `gorm:"default:0;comment:装备槽位5（头盔）" json:"eq_slot5"` // 头盔
	EqSlot6 uint `gorm:"default:0;comment:装备槽位6（靴子）" json:"eq_slot6"` // 靴子
	EqSlot7 uint `gorm:"default:0;comment:装备槽位7（项链）" json:"eq_slot7"` // 项链
	EqSlot8 uint `gorm:"default:0;comment:装备槽位8（手镯）" json:"eq_slot8"` // 手镯
	// 每日数据（date + 计数）
	DayDate    string `gorm:"type:varchar(10);default:'';comment:每日数据（date + 计数）" json:"day_date"`
	DaySignin  int    `gorm:"default:0;comment:天数Signin" json:"day_signin"`
	DayBattle  int    `gorm:"default:0;comment:天数战斗" json:"day_battle"`
	DayDungeon int    `gorm:"default:0;comment:天数副本" json:"day_dungeon"`
	DayArena   int    `gorm:"default:0;comment:每日比武次数（上限5）" json:"day_arena"` // 每日比武次数（上限5）
	DayHunt    int    `gorm:"default:0;comment:今日狩猎数（活跃度）" json:"day_hunt"`   // 今日狩猎数（活跃度）
	// 通天塔（当前层，战败归零）
	TowerFloor int `gorm:"default:0;comment:通天塔（当前层，战败归零）" json:"tower_floor"`
	TowerBest  int `gorm:"default:0;comment:历史最高层" json:"tower_best"` // 历史最高层
	// 管理封禁（unix 时间戳，0=未封禁）
	BanUntil  int64     `gorm:"default:0;comment:封号截止" json:"ban_until"`  // 封号截止
	MuteUntil int64     `gorm:"default:0;comment:禁言截止" json:"mute_until"` // 禁言截止
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (HxxyPlayer) TableName() string { return "hxxy_players" }

// HxxyMapNode 地图节点（dtx 区域 + dty 节点编号，方向值为 "dtx_dty"）
type HxxyMapNode struct {
	ID        uint   `gorm:"primaryKey;comment:= node_id" json:"id"` // = node_id
	Dtx       int    `gorm:"comment:地图X" json:"dtx"`
	Dty       int    `gorm:"comment:地图Y" json:"dty"`
	Name      string `gorm:"type:varchar(50);comment:名称" json:"name"`
	Desc      string `gorm:"type:varchar(255);comment:描述" json:"desc"`
	Up        string `gorm:"type:varchar(20);default:'';comment:上" json:"up"`
	Down      string `gorm:"type:varchar(20);default:'';comment:下" json:"down"`
	Left      string `gorm:"type:varchar(20);default:'';comment:左" json:"left"`
	Right     string `gorm:"type:varchar(20);default:'';comment:右" json:"right"`
	UpJump    string `gorm:"type:varchar(20);default:'';comment:上Jump" json:"up_jump"`
	DownJump  string `gorm:"type:varchar(20);default:'';comment:下Jump" json:"down_jump"`
	LeftJump  string `gorm:"type:varchar(20);default:'';comment:剩余Jump" json:"left_jump"`
	RightJump string `gorm:"type:varchar(20);default:'';comment:右Jump" json:"right_jump"`
}

func (HxxyMapNode) TableName() string { return "hxxy_map_nodes" }

// HxxyMapGrid 地图网格布局（复刻原版 xdt/*.php + MapViewer 查看地图）
// Grid 为二维数组 JSON：0=空位 "|"墙 "—"路 {dtxy,mz,is_jump}=节点
type HxxyMapGrid struct {
	ID   uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	Dtx  int    `gorm:"uniqueIndex;comment:地图编号" json:"dtx"` // 地图编号
	Grid string `gorm:"type:mediumtext;comment:Grid" json:"grid"`
}

func (HxxyMapGrid) TableName() string { return "hxxy_map_grids" }

// HxxyNpc NPC（含战斗属性；出现在刷怪表的为战斗 NPC，其余为功能 NPC）
type HxxyNpc struct {
	ID    uint   `gorm:"primaryKey;comment:= npc_id" json:"id"` // = npc_id
	Name  string `gorm:"type:varchar(50);comment:名称" json:"name"`
	Level int    `gorm:"default:1;comment:等级" json:"level"`
	HP    int    `gorm:"default:0;comment:生命" json:"hp"`
	MaxHP int    `gorm:"default:0;comment:上限生命" json:"max_hp"`
	MP    int    `gorm:"default:0;comment:魔法" json:"mp"`
	MaxMP int    `gorm:"default:0;comment:上限魔法" json:"max_mp"`
	Atk   int    `gorm:"default:0;comment:攻击" json:"atk"`
	Mg    int    `gorm:"default:0;comment:魔宫" json:"mg"`
	Def   int    `gorm:"default:0;comment:防御" json:"def"`
	Mf    int    `gorm:"default:0;comment:魔防" json:"mf"`
	Bg    int    `gorm:"default:0;comment:背景（冰攻）" json:"bg"`                   // 冰攻
	Hg    int    `gorm:"default:0;comment:皇宫（火攻）" json:"hg"`                   // 火攻
	Lg    int    `gorm:"default:0;comment:龙宫（雷攻）" json:"lg"`                   // 雷攻
	Bf    int    `gorm:"default:0;comment:宝坊（冰防）" json:"bf"`                   // 冰防
	Hf    int    `gorm:"default:0;comment:花坊（火防）" json:"hf"`                   // 火防
	Lf    int    `gorm:"default:0;comment:练坊（雷防）" json:"lf"`                   // 雷防
	Take  string `gorm:"type:varchar(255);default:'';comment:被打语" json:"take"` // 被打语
	Kind  int    `gorm:"default:0;comment:0功能 1战斗" json:"kind"`                // 0功能 1战斗
	// 掉落（JSON 数组：[{"type":"item|equip","id":1,"rate":30}]，rate 为万分比）
	Drops string `gorm:"type:varchar(1000);default:'';comment:Drops" json:"drops"`
	// 战斗奖励
	ExpReward   int `gorm:"default:0;comment:战斗奖励" json:"exp_reward"`
	MoneyReward int `gorm:"default:0;comment:银两奖励" json:"money_reward"`
}

func (HxxyNpc) TableName() string { return "hxxy_npcs" }

// HxxySpawn 刷怪点（dtx 区域 dty 节点，difficulty 普通/困难）
type HxxySpawn struct {
	ID         uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	Dtx        int    `gorm:"comment:地图X" json:"dtx"`
	Dty        int    `gorm:"comment:0=区域随机池" json:"dty"` // 0=区域随机池
	NpcID      uint   `gorm:"comment:NPCID" json:"npc_id"`
	Name       string `gorm:"type:varchar(50);comment:名称" json:"name"`
	Difficulty string `gorm:"type:varchar(10);default:'普通';comment:Difficulty" json:"difficulty"`
}

func (HxxySpawn) TableName() string { return "hxxy_spawns" }

// HxxyMapNpc 功能NPC放置（地图节点上的功能人物：传送/商店/对话，复刻原版 mapnpc+fznpc）
// Teles 为 JSON 数组 [{"name":"龙宫","dtx":2,"dty":1}]；Shop 为前端商店/服务页标识
type HxxyMapNpc struct {
	ID       uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	Dtx      int    `gorm:"comment:地图X" json:"dtx"`
	Dty      int    `gorm:"comment:地图Y" json:"dty"`
	NpcID    uint   `gorm:"comment:关联 hxxy_npcs（0=无战斗数据）" json:"npc_id"` // 关联 hxxy_npcs（0=无战斗数据）
	Name     string `gorm:"type:varchar(50);comment:名称" json:"name"`
	Img      string `gorm:"type:varchar(50);default:'';comment:pic/npc/ 下图片文件名" json:"img"` // pic/npc/ 下图片文件名
	Dialogue string `gorm:"type:varchar(255);default:'';comment:Dialogue" json:"dialogue"`
	Shop     string `gorm:"type:varchar(20);default:'';comment:medicine/weapon/armor/jewel/grocery/pet/bank/warehouse/rest" json:"shop"` // medicine/weapon/armor/jewel/grocery/pet/bank/warehouse/rest
	Teles    string `gorm:"type:varchar(1000);default:'';comment:Teles" json:"teles"`
}

func (HxxyMapNpc) TableName() string { return "hxxy_map_npcs" }

// HxxyItem 物品（wpxx）
// 分类：1卷轴秘籍 2宝石 4礼包特殊 5药品食物 6任务剧情 8宝箱
type HxxyItem struct {
	ID        uint   `gorm:"primaryKey;comment:= item_id" json:"id"` // = item_id
	Name      string `gorm:"type:varchar(50);comment:名称" json:"name"`
	Desc      string `gorm:"type:varchar(255);comment:描述" json:"desc"`
	Category  int    `gorm:"default:0;comment:分类" json:"category"`
	BeanPrice int    `gorm:"default:0;comment:金豆价格" json:"bean_price"` // 金豆价
	Price     int    `gorm:"default:0;comment:银两价" json:"price"`       // 银两价
	Level     int    `gorm:"default:1;comment:等级" json:"level"`
	Weight    int    `gorm:"default:1;comment:权重" json:"weight"`
	Bind      int    `gorm:"default:1;comment:绑定" json:"bind"`
	// 使用效果（JSON，复刻原版效果）：{"hp":100,"mp":50,"exp":100,"money":100,"beans":1,
	// "xiulian":1,"skill":1,"pet":1,"daily":3,"title":147,"vip":30,"box":1,"quest":1}
	Effect string `gorm:"type:varchar(500);default:'';comment:效果" json:"effect"`
}

func (HxxyItem) TableName() string { return "hxxy_items" }

// HxxyEquip 装备（zbxx）
// 分类(部位)：1法宝 2坐骑 3武器 4护甲 5头盔 6靴子 7项链 8手镯
// 门派：0通用 1-5门派 6全门派 7无门派限制
type HxxyEquip struct {
	ID        uint   `gorm:"primaryKey;comment:= equip_id" json:"id"` // = equip_id
	Name      string `gorm:"type:varchar(50);comment:名称" json:"name"`
	Desc      string `gorm:"type:varchar(255);comment:描述" json:"desc"`
	HP        int    `gorm:"default:0;comment:生命" json:"hp"`
	Atk       int    `gorm:"default:0;comment:攻击" json:"atk"`
	Mg        int    `gorm:"default:0;comment:魔宫" json:"mg"`
	Def       int    `gorm:"default:0;comment:防御" json:"def"`
	Bg        int    `gorm:"default:0;comment:背景" json:"bg"`
	Hg        int    `gorm:"default:0;comment:皇宫" json:"hg"`
	Lg        int    `gorm:"default:0;comment:龙宫" json:"lg"`
	Bf        int    `gorm:"default:0;comment:宝坊" json:"bf"`
	Hf        int    `gorm:"default:0;comment:花坊" json:"hf"`
	Lf        int    `gorm:"default:0;comment:练坊" json:"lf"`
	Level     int    `gorm:"default:1;comment:等级" json:"level"`
	Weight    int    `gorm:"default:1;comment:权重" json:"weight"`
	Bind      int    `gorm:"default:1;comment:绑定" json:"bind"`
	BeanPrice int    `gorm:"default:0;comment:金豆价格" json:"bean_price"`
	Price     int    `gorm:"default:0;comment:价格" json:"price"`
	Slot      int    `gorm:"default:0;comment:佩戴（原版 pd，0=全部位可穿标记，实际部位看 category）" json:"slot"` // 佩戴（原版 pd，0=全部位可穿标记，实际部位看 category）
	Sect      int    `gorm:"default:0;comment:门派（门派）" json:"sect"`                             // 门派
	Category  int    `gorm:"default:3;comment:部位分类" json:"category"`                           // 部位分类
}

func (HxxyEquip) TableName() string { return "hxxy_equips" }

// HxxyBag 背包/仓库（玩家物品与装备实例；kind=item/equip）
type HxxyBag struct {
	ID       uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	PlayerID uint   `gorm:"index;comment:玩家ID" json:"player_id"`
	Kind     string `gorm:"type:varchar(10);comment:item / equip" json:"kind"`    // item / equip
	RefID    uint   `gorm:"comment:hxxy_items.id / hxxy_equips.id" json:"ref_id"` // hxxy_items.id / hxxy_equips.id
	Count    int    `gorm:"default:1;comment:数量" json:"count"`
	Bind     int    `gorm:"default:0;comment:绑定" json:"bind"`
	Store    int    `gorm:"default:0;comment:0背包 1仓库 2挂售中" json:"store"` // 0背包 1仓库 2挂售中
	// 装备实例附加（JSON）：{"star":0,"holes":0,"gems":[]}，宝石为分类2物品id数组
	Extra string `gorm:"type:varchar(500);default:'';comment:Extra" json:"extra"`
	// 药品每日已用次数（每日限用）
	Used      int       `gorm:"default:0;comment:药品每日已用次数（每日限用）" json:"used"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxyBag) TableName() string { return "hxxy_bag" }

// HxxySkill 技能（jnxx：门派技能+普攻+捕捉/查看）
// 分类：1普攻 2门派攻击 3捕捉 4查看 等按原版
type HxxySkill struct {
	ID         uint   `gorm:"primaryKey;comment:= skill_id" json:"id"` // = skill_id
	Category   int    `gorm:"comment:分类" json:"category"`
	Name       string `gorm:"type:varchar(50);comment:名称" json:"name"`
	Desc       string `gorm:"type:varchar(255);comment:描述" json:"desc"`
	Multiplier int    `gorm:"default:1;comment:倍率 shxs" json:"multiplier"` // 倍率 shxs
	MpCost     int    `gorm:"default:0;comment:魔法消耗（耗蓝）" json:"mp_cost"`   // 耗蓝
	LearnLevel int    `gorm:"default:1;comment:Learn等级" json:"learn_level"`
	Sect       int    `gorm:"default:0;comment:0通用 1-5门派" json:"sect"` // 0通用 1-5门派
}

func (HxxySkill) TableName() string { return "hxxy_skills" }

// HxxyPlayerSkill 玩家已学技能
type HxxyPlayerSkill struct {
	ID       uint `gorm:"primaryKey;comment:主键ID" json:"id"`
	PlayerID uint `gorm:"index;comment:玩家ID" json:"player_id"`
	SkillID  uint `gorm:"comment:技能ID" json:"skill_id"`
	Level    int  `gorm:"default:1;comment:等级" json:"level"`
}

func (HxxyPlayerSkill) TableName() string { return "hxxy_player_skills" }

// HxxyPetSpecies 宠物种族（cwxx 基础属性）
type HxxyPetSpecies struct {
	ID      uint   `gorm:"primaryKey;comment:= species_id" json:"id"` // = species_id
	Name    string `gorm:"type:varchar(50);comment:名称" json:"name"`
	Level   int    `gorm:"default:1;comment:等级" json:"level"`
	HP      int    `gorm:"default:0;comment:生命" json:"hp"`
	MaxHP   int    `gorm:"default:0;comment:上限生命" json:"max_hp"`
	MP      int    `gorm:"default:0;comment:魔法" json:"mp"`
	MaxMP   int    `gorm:"default:0;comment:上限魔法" json:"max_mp"`
	Atk     int    `gorm:"default:0;comment:攻击" json:"atk"`
	Mg      int    `gorm:"default:0;comment:魔宫" json:"mg"`
	Def     int    `gorm:"default:0;comment:防御" json:"def"`
	Mf      int    `gorm:"default:0;comment:魔防" json:"mf"`
	Bg      int    `gorm:"default:0;comment:背景" json:"bg"`
	Hg      int    `gorm:"default:0;comment:皇宫" json:"hg"`
	Lg      int    `gorm:"default:0;comment:龙宫" json:"lg"`
	Bf      int    `gorm:"default:0;comment:宝坊" json:"bf"`
	Hf      int    `gorm:"default:0;comment:花坊" json:"hf"`
	Lf      int    `gorm:"default:0;comment:练坊" json:"lf"`
	Star    int    `gorm:"default:1;comment:星级（星级）" json:"star"`    // 星级
	Quality int    `gorm:"default:1;comment:品质（品质）" json:"quality"` // 品质
}

func (HxxyPetSpecies) TableName() string { return "hxxy_pet_species" }

// HxxyPet 玩家宠物实例
type HxxyPet struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	PlayerID  uint      `gorm:"index;comment:玩家ID" json:"player_id"`
	SpeciesID uint      `gorm:"comment:SpeciesID" json:"species_id"`
	Name      string    `gorm:"type:varchar(50);comment:名称" json:"name"`
	Level     int       `gorm:"default:1;comment:等级" json:"level"`
	Exp       int       `gorm:"default:0;comment:经验" json:"exp"`
	Star      int       `gorm:"default:1;comment:星级" json:"star"`
	Mutate    int       `gorm:"default:0;comment:变异" json:"mutate"` // 变异
	Quality   int       `gorm:"default:1;comment:品质" json:"quality"`
	Fighting  int       `gorm:"default:0;comment:1参战" json:"fighting"` // 1参战
	CurHP     int       `gorm:"default:0;comment:当前生命" json:"cur_hp"`
	CurMP     int       `gorm:"default:0;comment:当前魔法" json:"cur_mp"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxyPet) TableName() string { return "hxxy_pets" }

// HxxyBattle 战斗（进行中存敌方/我方快照，结束后归档）
type HxxyBattle struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	PlayerID  uint      `gorm:"index;comment:玩家ID" json:"player_id"`
	Type      string    `gorm:"type:varchar(10);comment:npc/boss/dungeon/pk" json:"type"` // npc/boss/dungeon/pk
	EnemyID   uint      `gorm:"comment:npc_id / boss_id" json:"enemy_id"`                 // npc_id / boss_id
	EnemyName string    `gorm:"type:varchar(50);default:'';comment:敌方名称" json:"enemy_name"`
	Round     int       `gorm:"default:0;comment:回合" json:"round"`
	Enemy     string    `gorm:"type:text;comment:敌方快照 JSON" json:"enemy"`         // 敌方快照 JSON
	Self      string    `gorm:"type:text;comment:我方快照 JSON" json:"self"`          // 我方快照 JSON
	Log       string    `gorm:"type:text;comment:战报行 JSON 数组" json:"log"`         // 战报行 JSON 数组
	Status    int       `gorm:"default:1;comment:1进行中 2胜利 3失败 4逃跑" json:"status"` // 1进行中 2胜利 3失败 4逃跑
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (HxxyBattle) TableName() string { return "hxxy_battles" }

// HxxyBattleLog 战斗历史（奖励/掉落流水）
type HxxyBattleLog struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	PlayerID  uint      `gorm:"index;comment:玩家ID" json:"player_id"`
	Type      string    `gorm:"type:varchar(10);comment:类型" json:"type"`
	EnemyName string    `gorm:"type:varchar(50);comment:敌方名称" json:"enemy_name"`
	Result    int       `gorm:"comment:2胜 3败 4逃" json:"result"` // 2胜 3败 4逃
	Round     int       `gorm:"comment:回合" json:"round"`
	Exp       int       `gorm:"default:0;comment:经验" json:"exp"`
	Money     int       `gorm:"default:0;comment:银两" json:"money"`
	Loot      string    `gorm:"type:varchar(500);default:'';comment:掉落 JSON" json:"loot"` // 掉落 JSON
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxyBattleLog) TableName() string { return "hxxy_battle_logs" }

// HxxyQuest 任务（通用引擎：hunt打怪 collect收集 talk对话）
type HxxyQuest struct {
	ID          uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name        string `gorm:"type:varchar(50);comment:名称" json:"name"`
	Desc        string `gorm:"type:varchar(255);comment:描述" json:"desc"`
	Type        string `gorm:"type:varchar(10);comment:hunt/collect/talk" json:"type"`       // hunt/collect/talk
	Category    int    `gorm:"default:1;comment:1主线 2支线 3日常" json:"category"`                // 1主线 2支线 3日常
	TargetID    uint   `gorm:"default:0;comment:npc_id / item_id / npc_id" json:"target_id"` // npc_id / item_id / npc_id
	Count       int    `gorm:"default:1;comment:数量" json:"count"`
	MinLevel    int    `gorm:"default:1;comment:下限等级" json:"min_level"`
	FromNpc     uint   `gorm:"default:0;comment:接取 NPC" json:"from_npc"` // 接取 NPC
	ExpReward   int    `gorm:"default:0;comment:经验奖励" json:"exp_reward"`
	MoneyReward int64  `gorm:"default:0;comment:银两奖励" json:"money_reward"`
	BeanReward  int    `gorm:"default:0;comment:金豆奖励" json:"bean_reward"`
	ItemReward  uint   `gorm:"default:0;comment:奖励物品 id" json:"item_reward"` // 奖励物品 id
	ItemEquip   uint   `gorm:"default:0;comment:0物品 1装备" json:"item_equip"`  // 0物品 1装备
	NextQuest   uint   `gorm:"default:0;comment:后续任务" json:"next_quest"`     // 后续任务
}

func (HxxyQuest) TableName() string { return "hxxy_quests" }

// HxxyActivityLog 活动领取记录（act: login7七日登录礼/daily50 daily100每日活跃宝箱）
type HxxyActivityLog struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	PlayerID  uint      `gorm:"index;comment:玩家ID" json:"player_id"`
	Act       string    `gorm:"type:varchar(20);comment:活动" json:"act"`
	Day       string    `gorm:"type:varchar(10);comment:天数" json:"day"`
	Tier      int       `gorm:"default:0;comment:品质档位" json:"tier"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxyActivityLog) TableName() string { return "hxxy_activity_logs" }

// HxxyPlayerQuest 玩家任务进度
type HxxyPlayerQuest struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	PlayerID  uint      `gorm:"index;comment:玩家ID" json:"player_id"`
	QuestID   uint      `gorm:"comment:任务ID" json:"quest_id"`
	Status    int       `gorm:"default:1;comment:1进行中 2可提交 3已完成" json:"status"` // 1进行中 2可提交 3已完成
	Progress  int       `gorm:"default:0;comment:Progress" json:"progress"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (HxxyPlayerQuest) TableName() string { return "hxxy_player_quests" }

// HxxyDungeon 副本定义（大雁塔/兵马俑/碑林/小雁塔/冰风谷）
type HxxyDungeon struct {
	ID       uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name     string `gorm:"type:varchar(50);comment:名称" json:"name"`
	Desc     string `gorm:"type:varchar(255);comment:描述" json:"desc"`
	Floors   int    `gorm:"default:10;comment:总层数" json:"floors"`                // 总层数
	NpcBase  uint   `gorm:"default:0;comment:怪物起始 npc_id（逐层递进）" json:"npc_base"` // 怪物起始 npc_id（逐层递进）
	Daily    int    `gorm:"default:2;comment:每日次数" json:"daily"`                 // 每日次数
	MinLevel int    `gorm:"default:10;comment:下限等级" json:"min_level"`
}

func (HxxyDungeon) TableName() string { return "hxxy_dungeons" }

// HxxyDungeonRun 玩家副本进度（复刻原版 fb_ini：激活→杀5守护BOSS→完成，每日每难度一次）
type HxxyDungeonRun struct {
	ID         uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	PlayerID   uint   `gorm:"index;comment:玩家ID" json:"player_id"`
	DungeonID  uint   `gorm:"comment:副本ID" json:"dungeon_id"`
	Floor      int    `gorm:"default:0;comment:已击杀守护BOSS数（原版怪物1~5）" json:"floor"`                           // 已击杀守护BOSS数（原版怪物1~5）
	Guards     string `gorm:"type:varchar(255);default:'';comment:激活时随机出的守护怪 npc_id 列表 JSON" json:"guards"` // 激活时随机出的守护怪 npc_id 列表 JSON
	Done       int    `gorm:"default:0;comment:1=当日已完成" json:"done"`                                        // 1=当日已完成
	DayDate    string `gorm:"type:varchar(10);default:'';comment:天数日期" json:"day_date"`
	CountToday int    `gorm:"default:0;comment:数量Today" json:"count_today"`
}

func (HxxyDungeonRun) TableName() string { return "hxxy_dungeon_runs" }

// HxxyBoss 世界 BOSS（bosses 数据 + 刷新）
type HxxyBoss struct {
	ID        uint       `gorm:"primaryKey;comment:= boss_id" json:"id"` // = boss_id
	Name      string     `gorm:"type:varchar(50);comment:名称" json:"name"`
	Level     int        `gorm:"default:1;comment:等级" json:"level"`
	HP        int        `gorm:"default:0;comment:生命" json:"hp"`
	MaxHP     int        `gorm:"default:0;comment:上限生命" json:"max_hp"`
	MP        int        `gorm:"default:0;comment:魔法" json:"mp"`
	MaxMP     int        `gorm:"default:0;comment:上限魔法" json:"max_mp"`
	Atk       int        `gorm:"default:0;comment:攻击" json:"atk"`
	Mg        int        `gorm:"default:0;comment:魔宫" json:"mg"`
	Def       int        `gorm:"default:0;comment:防御" json:"def"`
	Mf        int        `gorm:"default:0;comment:魔防" json:"mf"`
	Bg        int        `gorm:"default:0;comment:背景" json:"bg"`
	Hg        int        `gorm:"default:0;comment:皇宫" json:"hg"`
	Lg        int        `gorm:"default:0;comment:龙宫" json:"lg"`
	Bf        int        `gorm:"default:0;comment:宝坊" json:"bf"`
	Hf        int        `gorm:"default:0;comment:花坊" json:"hf"`
	Lf        int        `gorm:"default:0;comment:练坊" json:"lf"`
	Take      string     `gorm:"type:varchar(255);default:'';comment:领取" json:"take"`
	RespawnAt *time.Time `gorm:"comment:击杀后刷新时间" json:"respawn_at"` // 击杀后刷新时间
}

func (HxxyBoss) TableName() string { return "hxxy_bosses" }

// HxxyTitle 头衔（激活档位，属性加成）
type HxxyTitle struct {
	ID   uint   `gorm:"primaryKey;comment:= title_id" json:"id"` // = title_id
	Name string `gorm:"type:varchar(50);comment:名称" json:"name"`
	Desc string `gorm:"type:varchar(255);comment:描述" json:"desc"`
	HP   int    `gorm:"default:0;comment:生命" json:"hp"`
	Atk  int    `gorm:"default:0;comment:攻击" json:"atk"`
	Def  int    `gorm:"default:0;comment:防御" json:"def"`
	Mg   int    `gorm:"default:0;comment:魔宫" json:"mg"`
}

func (HxxyTitle) TableName() string { return "hxxy_titles" }

// HxxyPlayerTitle 玩家已激活头衔
type HxxyPlayerTitle struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	PlayerID  uint      `gorm:"index;comment:玩家ID" json:"player_id"`
	TitleID   uint      `gorm:"comment:标题ID" json:"title_id"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxyPlayerTitle) TableName() string { return "hxxy_player_titles" }

// HxxyGang 国家（复刻原版 all_bp：国家=原版"帮派"，xy172 主页 / xy595 升级 / xy186 国家商城）
type HxxyGang struct {
	ID          uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name        string    `gorm:"type:varchar(30);uniqueIndex;comment:名称" json:"name"`
	Level       int       `gorm:"default:1;comment:等级" json:"level"`
	FounderID   uint      `gorm:"comment:首任君主" json:"founder_id"` // 首任君主
	FounderName string    `gorm:"type:varchar(30);comment:Founder名称" json:"founder_name"`
	LeaderID    uint      `gorm:"comment:现任君主" json:"leader_id"`                     // 现任君主
	MemberMax   int       `gorm:"default:20;comment:国家人数上限（升级提升）" json:"member_max"` // 国家人数上限（升级提升）
	ExpMax      int64     `gorm:"default:5000;comment:国家经验上限（升级提升）" json:"exp_max"`  // 国家经验上限（升级提升）
	Money       int64     `gorm:"default:0;comment:国家资金（国库）" json:"money"`           // 国家资金（国库）
	Exp         int64     `gorm:"default:0;comment:国家经验" json:"exp"`                 // 国家经验
	Sw          int64     `gorm:"default:0;comment:国家声望" json:"sw"`                  // 国家声望
	CreatedAt   time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxyGang) TableName() string { return "hxxy_gangs" }

// HxxyGangMember 国家成员（职务复刻原版：0成员 1君主 2辅助 3军机 4财政 5工部 6外交 7军团长）
type HxxyGangMember struct {
	ID                uint `gorm:"primaryKey;comment:主键ID" json:"id"`
	GangID            uint `gorm:"index;comment:帮派ID" json:"gang_id"`
	PlayerID          uint `gorm:"uniqueIndex;comment:玩家ID" json:"player_id"`
	Role              int  `gorm:"default:0;comment:角色" json:"role"`
	Contribution      int  `gorm:"default:0;comment:可用贡献" json:"contribution"`       // 可用贡献
	TotalContribution int  `gorm:"default:0;comment:历史贡献" json:"total_contribution"` // 历史贡献
}

func (HxxyGangMember) TableName() string { return "hxxy_gang_members" }

// HxxyMarriage 结婚
type HxxyMarriage struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	PlayerA   uint      `gorm:"index;comment:玩家A" json:"player_a"`
	PlayerB   uint      `gorm:"index;comment:玩家B" json:"player_b"`
	Status    int       `gorm:"default:1;comment:1求婚中 2已婚 3已离" json:"status"` // 1求婚中 2已婚 3已离
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxyMarriage) TableName() string { return "hxxy_marriages" }

// HxxyHouse 住宅（家具 JSON：[{"id":1,"name":"屏风","bonus":...}]）
type HxxyHouse struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	PlayerID  uint      `gorm:"uniqueIndex;comment:玩家ID" json:"player_id"`
	Furniture string    `gorm:"type:varchar(1000);default:'[]';comment:Furniture" json:"furniture"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (HxxyHouse) TableName() string { return "hxxy_houses" }

// HxxyFriend 游戏内好友
type HxxyFriend struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	PlayerID  uint      `gorm:"uniqueIndex:uk_hxxyfriend;comment:玩家ID" json:"player_id"`
	FriendID  uint      `gorm:"uniqueIndex:uk_hxxyfriend;comment:好友ID" json:"friend_id"`
	Status    int       `gorm:"default:1;comment:1好友 2黑名单（复刻原版 hyfl 单向关系）" json:"status"` // 1好友 2黑名单（复刻原版 hyfl 单向关系）
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxyFriend) TableName() string { return "hxxy_friends" }

// HxxyChat 世界聊天
type HxxyChat struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	PlayerID  uint      `gorm:"index;comment:玩家ID" json:"player_id"`
	Name      string    `gorm:"type:varchar(30);comment:名称" json:"name"`
	Content   string    `gorm:"type:varchar(255);comment:内容" json:"content"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxyChat) TableName() string { return "hxxy_chats" }

// HxxyMsg 玩家消息（复刻原版首页消息区：sys=系统动态 pv=私聊，展示一次后标已读）
type HxxyMsg struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	PlayerID  uint      `gorm:"index;comment:收件玩家" json:"player_id"`      // 收件玩家
	FromID    uint      `gorm:"comment:私聊发送者玩家ID（系统消息为0）" json:"from_id"` // 私聊发送者玩家ID（系统消息为0）
	FromName  string    `gorm:"type:varchar(30);comment:来源名称" json:"from_name"`
	Kind      string    `gorm:"type:varchar(4);comment:sys / pv" json:"kind"` // sys / pv
	Content   string    `gorm:"type:varchar(255);comment:内容" json:"content"`
	IsRead    int       `gorm:"default:0;index;comment:是否已读" json:"is_read"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxyMsg) TableName() string { return "hxxy_msgs" }

// HxxySignin 签到（连签递增奖励）
type HxxySignin struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	PlayerID  uint      `gorm:"index;comment:玩家ID" json:"player_id"`
	Day       string    `gorm:"type:varchar(10);comment:天数" json:"day"`
	Streak    int       `gorm:"default:1;comment:Streak" json:"streak"`
	Reward    string    `gorm:"type:varchar(200);default:'';comment:奖励" json:"reward"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxySignin) TableName() string { return "hxxy_signins" }

// HxxySigninClaim 月度累计签到阶梯奖励领取记录（每月清零重领）
type HxxySigninClaim struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	PlayerID  uint      `gorm:"index;comment:玩家ID" json:"player_id"`
	Month     string    `gorm:"type:varchar(7);comment:2026-09" json:"month"` // 2026-09
	Tier      int       `gorm:"comment:2/5/10/15/25" json:"tier"`             // 2/5/10/15/25
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxySigninClaim) TableName() string { return "hxxy_signin_claims" }

// HxxyStall 摆摊挂售
type HxxyStall struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	SellerID  uint      `gorm:"index;comment:卖家ID" json:"seller_id"`
	BagID     uint      `gorm:"comment:挂售的背包行（交易时校验/转移）" json:"bag_id"` // 挂售的背包行（交易时校验/转移）
	Kind      string    `gorm:"type:varchar(10);comment:种类" json:"kind"`
	RefID     uint      `gorm:"comment:关联ID" json:"ref_id"`
	Name      string    `gorm:"type:varchar(50);comment:名称" json:"name"`
	Count     int       `gorm:"default:1;comment:数量" json:"count"`
	Price     int64     `gorm:"comment:银两价" json:"price"`                    // 银两价
	Status    int       `gorm:"default:1;comment:1在售 2已售 3下架" json:"status"` // 1在售 2已售 3下架
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxyStall) TableName() string { return "hxxy_stalls" }

// HxxyAuction 全区拍卖场（复刻原版 xy489/all_pm：一口价直购，买方付1%手续费最低1两，10天过期下架）
type HxxyAuction struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	SellerID  uint      `gorm:"index;comment:卖家ID" json:"seller_id"`
	BagID     uint      `gorm:"comment:拍卖的背包行" json:"bag_id"` // 拍卖的背包行
	Kind      string    `gorm:"type:varchar(10);comment:种类" json:"kind"`
	RefID     uint      `gorm:"comment:关联ID" json:"ref_id"`
	Name      string    `gorm:"type:varchar(50);comment:名称" json:"name"`
	Category  int       `gorm:"default:0;comment:物品分类（书卷1/宝石2/材料4/丹药5/商城6/宝箱8/装备100）" json:"category"` // 物品分类（书卷1/宝石2/材料4/丹药5/商城6/宝箱8/装备100）
	Count     int       `gorm:"default:1;comment:数量" json:"count"`
	Price     int64     `gorm:"comment:价格" json:"price"`
	Status    int       `gorm:"default:1;comment:1在售 2已售 3下架" json:"status"` // 1在售 2已售 3下架
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxyAuction) TableName() string { return "hxxy_auctions" }

// HxxyWalletLog 货币流水（银两/金豆）
type HxxyWalletLog struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	PlayerID  uint      `gorm:"index;comment:玩家ID" json:"player_id"`
	Currency  string    `gorm:"type:varchar(10);comment:money/beans/bank" json:"currency"` // money/beans/bank
	Amount    int64     `gorm:"comment:正=获得 负=消耗" json:"amount"`                           // 正=获得 负=消耗
	Balance   int64     `gorm:"comment:变动后余额" json:"balance"`                              // 变动后余额
	Reason    string    `gorm:"type:varchar(100);comment:原因" json:"reason"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxyWalletLog) TableName() string { return "hxxy_wallet_logs" }

// HxxyTeam 组队（上限4人：1队长+3队员，复刻原版 xy111.php）
type HxxyTeam struct {
	ID         uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	LeaderID   uint      `gorm:"uniqueIndex;comment:一人只能当一队队长" json:"leader_id"` // 一人只能当一队队长
	LeaderName string    `gorm:"type:varchar(30);comment:首领名称" json:"leader_name"`
	CreatedAt  time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxyTeam) TableName() string { return "hxxy_teams" }

// HxxyTeamMember 队伍成员
type HxxyTeamMember struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	TeamID    uint      `gorm:"index;comment:队伍ID" json:"team_id"`
	PlayerID  uint      `gorm:"uniqueIndex;comment:一人只能在一个队伍" json:"player_id"` // 一人只能在一个队伍
	Name      string    `gorm:"type:varchar(30);comment:名称" json:"name"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxyTeamMember) TableName() string { return "hxxy_team_members" }

// HxxyTeamInvite 组队邀请（0待处理 1已同意 2已拒绝）
type HxxyTeamInvite struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	TeamID    uint      `gorm:"index;comment:队伍ID" json:"team_id"`
	FromID    uint      `gorm:"comment:来源ID" json:"from_id"`
	FromName  string    `gorm:"type:varchar(30);comment:来源名称" json:"from_name"`
	ToID      uint      `gorm:"index;comment:接收者ID" json:"to_id"`
	ToName    string    `gorm:"type:varchar(30);comment:至名称" json:"to_name"`
	Status    int       `gorm:"default:0;comment:状态" json:"status"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxyTeamInvite) TableName() string { return "hxxy_team_invites" }

// HxxyGangInvite 帮派(国家)邀请（复刻原版 yq2.php：邀请直接显示在首页，0待处理 1已同意 2已拒绝）
type HxxyGangInvite struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	GangID    uint      `gorm:"index;comment:帮派ID" json:"gang_id"`
	GangName  string    `gorm:"type:varchar(30);comment:帮派名称" json:"gang_name"`
	FromID    uint      `gorm:"comment:来源ID" json:"from_id"`
	FromName  string    `gorm:"type:varchar(30);comment:来源名称" json:"from_name"`
	ToID      uint      `gorm:"index;comment:接收者ID" json:"to_id"`
	ToName    string    `gorm:"type:varchar(30);comment:至名称" json:"to_name"`
	Status    int       `gorm:"default:0;comment:状态" json:"status"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxyGangInvite) TableName() string { return "hxxy_gang_invites" }

// HxxyHouseInvite 住宅参观邀请（复刻原版 yq3.php：邀请直接显示在首页，0待处理 1已同意 2已拒绝）
type HxxyHouseInvite struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	FromID    uint      `gorm:"comment:来源ID" json:"from_id"`
	FromName  string    `gorm:"type:varchar(30);comment:来源名称" json:"from_name"`
	ToID      uint      `gorm:"index;comment:接收者ID" json:"to_id"`
	ToName    string    `gorm:"type:varchar(30);comment:至名称" json:"to_name"`
	Status    int       `gorm:"default:0;comment:状态" json:"status"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (HxxyHouseInvite) TableName() string { return "hxxy_house_invites" }

// HxxyGzWar 国战战局（每日一条，复刻原版 gz.php：按星期轮换战场国家，整点后30分钟开战）
type HxxyGzWar struct {
	ID             uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	WarDate        string `gorm:"type:varchar(10);uniqueIndex;comment:2026-09-12" json:"war_date"` // 2026-09-12
	ZcID           int    `gorm:"default:0;comment:战场国家：1傲来 2宝象 3乌鸡 4女儿 5车迟 7祭赛 6休整" json:"zc_id"` // 战场国家：1傲来 2宝象 3乌鸡 4女儿 5车迟 7祭赛 6休整
	DefGangID      uint   `gorm:"default:0;comment:防守方（报名帮派）" json:"def_gang_id"`                  // 防守方（报名帮派）
	DefGangName    string `gorm:"type:varchar(30);default:'';comment:防御帮派名称" json:"def_gang_name"`
	HolderGangID   uint   `gorm:"default:0;comment:权杖当前占据帮派" json:"holder_gang_id"` // 权杖当前占据帮派
	HolderGangName string `gorm:"type:varchar(30);default:'';comment:Holder帮派名称" json:"holder_gang_name"`
	HoldAt         int64  `gorm:"default:0;comment:权杖占据开始时间戳（守满300秒+10分）" json:"hold_at"` // 权杖占据开始时间戳（守满300秒+10分）
	NeijianAt      int64  `gorm:"default:0;comment:上次内奸生成时间戳" json:"neijian_at"`          // 上次内奸生成时间戳
	NeijianName    string `gorm:"type:varchar(50);default:'';comment:Neijian名称" json:"neijian_name"`
}

func (HxxyGzWar) TableName() string { return "hxxy_gz_wars" }

// HxxyGzScore 国家（帮派）积分榜
type HxxyGzScore struct {
	ID       uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	GangID   uint   `gorm:"uniqueIndex;comment:帮派ID" json:"gang_id"`
	GangName string `gorm:"type:varchar(30);comment:帮派名称" json:"gang_name"`
	Total    int    `gorm:"default:0;comment:总计" json:"total"`
}

func (HxxyGzScore) TableName() string { return "hxxy_gz_scores" }

// HxxyGzPlayer 国战个人积分榜
type HxxyGzPlayer struct {
	ID       uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	PlayerID uint   `gorm:"uniqueIndex;comment:玩家ID" json:"player_id"`
	Name     string `gorm:"type:varchar(30);comment:名称" json:"name"`
	GangName string `gorm:"type:varchar(30);default:'';comment:帮派名称" json:"gang_name"`
	Total    int    `gorm:"default:0;comment:总计" json:"total"`
}

func (HxxyGzPlayer) TableName() string { return "hxxy_gz_players" }
