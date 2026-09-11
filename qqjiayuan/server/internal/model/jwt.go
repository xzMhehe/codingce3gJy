package model

import "time"

// 精武堂（复刻 3GQQ 精武堂 wap 版，全系统）
// 玩法拆解：玩家属性+能量分配、装备八槽换装、商店(药品/武器/防具/配饰/材料)、
// 技能书店(被动/主动)、修炼练功、比武、每日任务+每日礼包、每周签到、
// 装备锻造(图纸+材料)、头衔(10档)、帮派(创建/加入)、排行。

// JwtPlayer 玩家档案（一个社区用户对应一条，懒创建）
type JwtPlayer struct {
	ID        uint `gorm:"primaryKey" json:"id"`
	UserID    uint `gorm:"uniqueIndex" json:"user_id"`
	Sex       int  `gorm:"default:0" json:"sex"` // 0保密 1男 2女
	Level     int  `gorm:"default:1" json:"level"`
	Exp       int  `gorm:"default:0" json:"exp"` // 当前等级累计经验
	Title     int  `gorm:"default:0" json:"title"` // 已激活头衔档位 0=无名小卒
	GangID    uint `gorm:"default:0" json:"gang_id"`
	Starter   int  `gorm:"default:0" json:"starter"` // 是否已发放新手元宝
	// 当前气血/气力（比武受创后低于上限，可用药品恢复）
	CurHp int `gorm:"default:0" json:"cur_hp"`
	CurMp int `gorm:"default:0" json:"cur_mp"`
	// 能量分配存量（分配点数）
	EHp  int `gorm:"default:0" json:"e_hp"`
	EMp  int `gorm:"default:0" json:"e_mp"`
	ESpd int `gorm:"default:0" json:"e_spd"`
	EAtk int `gorm:"default:0" json:"e_atk"`
	EDef int `gorm:"default:0" json:"e_def"`
	// 可用能量点（每升 1 级 +1，初始 5）
	Energy int `gorm:"default:5" json:"energy"`
	// 装备（存背包物品 id）
	WeaponID  uint `gorm:"default:0" json:"weapon_id"`
	HelmetID  uint `gorm:"default:0" json:"helmet_id"`
	ArmorID   uint `gorm:"default:0" json:"armor_id"`
	ShoesID   uint `gorm:"default:0" json:"shoes_id"`
	NecklaceID uint `gorm:"default:0" json:"necklace_id"`
	BraceletID uint `gorm:"default:0" json:"bracelet_id"`
	RingID    uint `gorm:"default:0" json:"ring_id"`
	MedalID   uint `gorm:"default:0" json:"medal_id"`
	// 修炼练功
	Practicing int       `gorm:"default:0" json:"practicing"` // 0未修炼 1修炼中
	PracticeAt *time.Time `json:"practice_at"`
	PracticeEnd *time.Time `json:"practice_end"`
	TrainCnt   int       `gorm:"default:0" json:"train_cnt"` // 今日修炼次数（每日任务）
	// 今日任务计数
	TaskDate    string `gorm:"type:varchar(10);default:''" json:"task_date"`
	ArenaCnt    int    `gorm:"default:0" json:"arena_cnt"`
	ArenaOpps   string `gorm:"type:varchar(1000);default:''" json:"arena_opps"` // 今日已比武过的对手user_id（逗号分隔）
	PillCnt     int    `gorm:"default:0" json:"pill_cnt"`
	SkillCnt    int    `gorm:"default:0" json:"skill_cnt"`
	ChatCnt     int    `gorm:"default:0" json:"chat_cnt"`
	RewardClaim int    `gorm:"default:0" json:"reward_claim"` // 今日礼包是否已领
	// 每周签到（逗号分隔已签星期 1-7）
	SignWeek string `gorm:"type:varchar(20);default:''" json:"sign_week"`
	SignDate string `gorm:"type:varchar(10);default:''" json:"sign_date"`
	// 比武连胜/荣誉（可选，排行用）
	Honor int `gorm:"default:0" json:"honor"`
	// 昵称冗余（比武列表/排行免连表）
	Nick string `gorm:"type:varchar(30);default:''" json:"nick"`
}

// JwtItem 道具（商店物品 + 锻造产物 + 材料 + 药品），统一一张表
// Cat: medicine/weapon/helmet/armor/shoes/necklace/bracelet/ring/medal/material/other
// Src: shop=商店可购 forge=锻造产物 material=比武/商店材料
type JwtItem struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Name     string `gorm:"type:varchar(30)" json:"name"`
	Cat      string `gorm:"type:varchar(20)" json:"cat"`
	Src      string `gorm:"type:varchar(10);default:'shop'" json:"src"`
	Price    int    `gorm:"default:0" json:"price"`      // 价格
	Currency string `gorm:"type:varchar(10);default:'coins'" json:"currency"` // coins yuanbao
	Level    int    `gorm:"default:1" json:"level"`      // 需要等级
	// 战斗加成
	Atk  int `gorm:"default:0" json:"atk"`
	Def  int `gorm:"default:0" json:"def"`
	Hp   int `gorm:"default:0" json:"hp"`
	Mp   int `gorm:"default:0" json:"mp"`
	Spd  int `gorm:"default:0" json:"spd"`
	Hit  int `gorm:"default:0" json:"hit"`
	Crit int `gorm:"default:0" json:"crit"`
	Dodge int `gorm:"default:0" json:"dodge"`
	// 药品效果
	RecoverHp int `gorm:"default:0" json:"recover_hp"`
	RecoverMp int `gorm:"default:0" json:"recover_mp"`
	Desc      string `gorm:"type:varchar(100)" json:"desc"`
	Status    int    `gorm:"default:1" json:"status"`
	// 锻造材料需求（锻造产物用）: "图纸x1,布料x1,精铁x3"
	Mats string `gorm:"type:varchar(100)" json:"mats"`
	Fee  int    `gorm:"default:0" json:"fee"` // 锻造费(yuanbao)
}

// JwtBag 背包
type JwtBag struct {
	ID     uint `gorm:"primaryKey" json:"id"`
	UserID uint `gorm:"index" json:"user_id"`
	ItemID uint `gorm:"index" json:"item_id"`
	Name   string `gorm:"type:varchar(30)" json:"name"`
	Cat    string `gorm:"type:varchar(20)" json:"cat"`
	Amount int    `gorm:"default:0" json:"amount"`
}

// JwtSkill 技能（书店）
type JwtSkill struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Name      string `gorm:"type:varchar(30)" json:"name"`
	Act       int    `gorm:"default:1" json:"act"` // 1主动 0被动
	Level     int    `gorm:"default:1" json:"level"` // 学习等级要求
	Price     int    `gorm:"default:0" json:"price"`
	Currency  string `gorm:"type:varchar(10);default:'coins'" json:"currency"`
	WeaponReq string `gorm:"type:varchar(20);default:'无限制'" json:"weapon_req"`
	Coef      int    `gorm:"default:100" json:"coef"` // 伤害系数 (倍*100)
	Hit       int    `gorm:"default:90" json:"hit"`
	Crit      int    `gorm:"default:5" json:"crit"`
	CritMul   int    `gorm:"default:150" json:"crit_mul"` // 暴击倍数 %
	DodgeAdd  int    `gorm:"default:0" json:"dodge_add"`
	// 被动效果
	PAtk int `gorm:"default:0" json:"p_atk"`
	PDef int `gorm:"default:0" json:"p_def"`
	PHp  int `gorm:"default:0" json:"p_hp"`
	PMp  int `gorm:"default:0" json:"p_mp"`
	Desc string `gorm:"type:varchar(100)" json:"desc"`
	Status int `gorm:"default:1" json:"status"`
}

// JwtLearnedSkill 已学技能
type JwtLearnedSkill struct {
	ID       uint `gorm:"primaryKey" json:"id"`
	UserID   uint `gorm:"index:uk_userskill,unique" json:"user_id"`
	SkillID  uint `gorm:"index:uk_userskill,unique" json:"skill_id"`
	Practice int  `gorm:"default:0" json:"practice"` // 熟练度(修炼/比武增加，复刻原站 熟练度:0/100)
	Level    int  `gorm:"default:1" json:"level"`    // 已学技能当前等级（熟练度满100可领悟升级）
	Equip    int  `gorm:"default:0" json:"equip"`    // 1已装备（比武/修炼使用该技能）
}

// JwtGang 帮派
type JwtGang struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Name     string `gorm:"type:varchar(30);uniqueIndex" json:"name"`
	Level    int    `gorm:"default:1" json:"level"`
	Exp      int    `gorm:"default:0" json:"exp"`
	MasterID uint   `gorm:"index" json:"master_id"`
	Master   string `gorm:"type:varchar(30)" json:"master"`
	Notice   string `gorm:"type:varchar(100)" json:"notice"`
	Members  int    `gorm:"default:1" json:"members"`
	CreatedAt time.Time `json:"created_at"`
}

// JwtGangMember 帮派成员
type JwtGangMember struct {
	ID      uint `gorm:"primaryKey" json:"id"`
	GangID  uint `gorm:"index" json:"gang_id"`
	UserID  uint `gorm:"uniqueIndex" json:"user_id"`
	IsMaster int `gorm:"default:0" json:"is_master"`
	JoinedAt time.Time `json:"joined_at"`
}

// JwtGangApply 帮派申请（加入申请，等帮主审批）
type JwtGangApply struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	GangID  uint   `gorm:"index" json:"gang_id"`
	UserID  uint   `gorm:"index" json:"user_id"`
	Msg     string `gorm:"type:varchar(200)" json:"msg"`
	Status  int    `gorm:"default:0" json:"status"` // 0待审批 1同意 2拒绝
	CreatedAt time.Time `json:"created_at"`
}

// 精武堂战斗属性（由玩家+装备+被动技能合成，仅运行期计算，不入库）
type JwtCombat struct {
	MaxHp   int `json:"max_hp"`
	MaxMp   int `json:"max_mp"`
	Speed   int `json:"speed"`
	Atk     int `json:"atk"`
	Def     int `json:"def"`
	Hit     int `json:"hit"`
	Crit    int `json:"crit"`
	CritMul int `json:"crit_mul"`
	Dodge   int `json:"dodge"`
}

// JwtChat 精武堂聊天（复刻原站 chat_list.aspx：type=0公共/1个人/2世界）
type JwtChat struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Nick      string    `gorm:"type:varchar(30)" json:"nick"`
	Content   string    `gorm:"type:varchar(200)" json:"content"`
	Type      int       `gorm:"default:0;index" json:"type"` // 0公共 1个人 2世界
	CreatedAt time.Time `json:"created_at"`
}

// JwtLog 精武堂动态（对应原站首页「动态」，记录比武等事件）
type JwtLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"` // 归属于谁的动态（这条动态展示在谁的动态列表）
	Nick      string    `gorm:"type:varchar(30)" json:"nick"`
	Msg       string    `gorm:"type:varchar(200)" json:"msg"`
	CreatedAt time.Time `json:"created_at"`
}

// JwtArenaRecord 比武记录（复刻 比武记录.xhtml：持久化整场战报，供详情查看）
type JwtArenaRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"` // 归属于谁（在谁的记录列表展示）
	MyUID     uint      `json:"my_uid"`
	MyNick    string    `gorm:"type:varchar(30)" json:"my_nick"`
	OppUID    uint      `json:"opp_uid"`
	OppNick   string    `gorm:"type:varchar(30)" json:"opp_nick"`
	Result    string    `gorm:"type:varchar(10)" json:"result"` // 胜利/失败/平手
	Exp       int       `json:"exp"`
	Coin      int       `json:"coin"`
	MyLevel   int       `json:"my_level"`
	MyMaxHp   int       `json:"my_max_hp"`
	MyCurHp   int       `json:"my_cur_hp"`
	OppLevel  int       `json:"opp_level"`
	OppMaxHp  int       `json:"opp_max_hp"`
	OppCurHp  int       `json:"opp_cur_hp"`
	LogsText  string    `gorm:"type:text" json:"logs_text"` // 战报文本行（\n 分隔）
	CreatedAt time.Time `json:"created_at"`
}