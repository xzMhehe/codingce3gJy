package model

import "time"

// 精武堂（复刻 3GQQ 精武堂 wap 版，全系统）
// 玩法拆解：玩家属性+能量分配、装备八槽换装、商店(药品/武器/防具/配饰/材料)、
// 技能书店(被动/主动)、修炼练功、比武、每日任务+每日礼包、每周签到、
// 装备锻造(图纸+材料)、头衔(10档)、帮派(创建/加入)、排行。

// JwtPlayer 玩家档案（一个社区用户对应一条，懒创建）
type JwtPlayer struct {
	ID      uint `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID  uint `gorm:"uniqueIndex;comment:用户ID" json:"user_id"`
	Sex     int  `gorm:"default:0;comment:0保密 1男 2女" json:"sex"` // 0保密 1男 2女
	Level   int  `gorm:"default:1;comment:等级" json:"level"`
	Exp     int  `gorm:"default:0;comment:当前等级累计经验" json:"exp"`         // 当前等级累计经验
	Title   int  `gorm:"default:0;comment:已激活头衔档位 0=无名小卒" json:"title"` // 已激活头衔档位 0=无名小卒
	GangID  uint `gorm:"default:0;comment:帮派ID" json:"gang_id"`
	Starter int  `gorm:"default:0;comment:是否已发放新手元宝" json:"starter"` // 是否已发放新手元宝
	// 当前气血/气力（比武受创后低于上限，可用药品恢复）
	CurHp int `gorm:"default:0;comment:当前气血/气力（比武受创后低于上限，可用药品恢复）" json:"cur_hp"`
	CurMp int `gorm:"default:0;comment:当前魔法" json:"cur_mp"`
	// 能量分配存量（分配点数）
	EHp  int `gorm:"default:0;comment:能量分配存量（分配点数）" json:"e_hp"`
	EMp  int `gorm:"default:0;comment:E魔法" json:"e_mp"`
	ESpd int `gorm:"default:0;comment:ESpd" json:"e_spd"`
	EAtk int `gorm:"default:0;comment:E攻击" json:"e_atk"`
	EDef int `gorm:"default:0;comment:E防御" json:"e_def"`
	// 可用能量点（每升 1 级 +1，初始 5）
	Energy int `gorm:"default:5;comment:可用能量点（每升 1 级 +1，初始 5）" json:"energy"`
	// 装备（存背包物品 id）
	WeaponID   uint `gorm:"default:0;comment:装备（存背包物品 id）" json:"weapon_id"`
	HelmetID   uint `gorm:"default:0;comment:HelmetID" json:"helmet_id"`
	ArmorID    uint `gorm:"default:0;comment:ArmorID" json:"armor_id"`
	ShoesID    uint `gorm:"default:0;comment:ShoesID" json:"shoes_id"`
	NecklaceID uint `gorm:"default:0;comment:NecklaceID" json:"necklace_id"`
	BraceletID uint `gorm:"default:0;comment:BraceletID" json:"bracelet_id"`
	RingID     uint `gorm:"default:0;comment:RingID" json:"ring_id"`
	MedalID    uint `gorm:"default:0;comment:MedalID" json:"medal_id"`
	// 修炼练功
	Practicing  int        `gorm:"default:0;comment:0未修炼 1修炼中" json:"practicing"` // 0未修炼 1修炼中
	PracticeAt  *time.Time `gorm:"comment:修炼时间" json:"practice_at"`
	PracticeEnd *time.Time `gorm:"comment:修炼结束" json:"practice_end"`
	TrainCnt    int        `gorm:"default:0;comment:今日修炼次数（每日任务）" json:"train_cnt"` // 今日修炼次数（每日任务）
	// 修炼技能点（修炼获得，用于技能深造/展示）
	SkillPoint int `gorm:"default:0;comment:修炼技能点（修炼获得，用于技能深造/展示）" json:"skill_point"`
	// 今日任务计数
	TaskDate    string `gorm:"type:varchar(10);default:'';comment:今日任务计数" json:"task_date"`
	ArenaCnt    int    `gorm:"default:0;comment:竞技场数量" json:"arena_cnt"`
	ArenaOpps   string `gorm:"type:varchar(1000);default:'';comment:今日已比武过的对手user_id（逗号分隔）" json:"arena_opps"` // 今日已比武过的对手user_id（逗号分隔）
	PillCnt     int    `gorm:"default:0;comment:Pill数量" json:"pill_cnt"`
	SkillCnt    int    `gorm:"default:0;comment:技能数量" json:"skill_cnt"`
	ChatCnt     int    `gorm:"default:0;comment:聊天数量" json:"chat_cnt"`
	RewardClaim int    `gorm:"default:0;comment:今日礼包是否已领" json:"reward_claim"` // 今日礼包是否已领
	// 每周签到（逗号分隔已签星期 1-7）
	SignWeek string `gorm:"type:varchar(20);default:'';comment:每周签到（逗号分隔已签星期 1-7）" json:"sign_week"`
	SignDate string `gorm:"type:varchar(10);default:'';comment:签到日期" json:"sign_date"`
	// 比武连胜/荣誉（可选，排行用）
	Honor int `gorm:"default:0;comment:比武连胜/荣誉（可选，排行用）" json:"honor"`
	// 昵称冗余（比武列表/排行免连表）
	Nick string `gorm:"type:varchar(30);default:'';comment:昵称冗余（比武列表/排行免连表）" json:"nick"`
}

// JwtItem 道具（商店物品 + 锻造产物 + 材料 + 药品），统一一张表
// Cat: medicine/weapon/helmet/armor/shoes/necklace/bracelet/ring/medal/material/other
// Src: shop=商店可购 forge=锻造产物 material=比武/商店材料
type JwtItem struct {
	ID       uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name     string `gorm:"type:varchar(30);comment:名称" json:"name"`
	Cat      string `gorm:"type:varchar(20);comment:分类" json:"cat"`
	Src      string `gorm:"type:varchar(10);default:'shop';comment:Src" json:"src"`
	Price    int    `gorm:"default:0;comment:价格（价格）" json:"price"`                                  // 价格
	Currency string `gorm:"type:varchar(10);default:'coins';comment:coins yuanbao" json:"currency"` // coins yuanbao
	Level    int    `gorm:"default:1;comment:需要等级" json:"level"`                                    // 需要等级
	// 战斗加成
	Atk   int `gorm:"default:0;comment:战斗加成" json:"atk"`
	Def   int `gorm:"default:0;comment:防御" json:"def"`
	Hp    int `gorm:"default:0;comment:生命" json:"hp"`
	Mp    int `gorm:"default:0;comment:魔法" json:"mp"`
	Spd   int `gorm:"default:0;comment:Spd" json:"spd"`
	Hit   int `gorm:"default:0;comment:命中" json:"hit"`
	Crit  int `gorm:"default:0;comment:暴击" json:"crit"`
	Dodge int `gorm:"default:0;comment:闪避" json:"dodge"`
	// 药品效果
	RecoverHp int    `gorm:"default:0;comment:药品效果" json:"recover_hp"`
	RecoverMp int    `gorm:"default:0;comment:恢复魔法" json:"recover_mp"`
	Desc      string `gorm:"type:varchar(100);comment:描述" json:"desc"`
	Status    int    `gorm:"default:1;comment:状态" json:"status"`
	// 锻造材料需求（锻造产物用）: "图纸x1,布料x1,精铁x3"
	Mats string `gorm:"type:varchar(100);comment:锻造材料需求（锻造产物用）: ”图纸x1,布料x1,精铁x3”" json:"mats"`
	Fee  int    `gorm:"default:0;comment:锻造费(yuanbao)" json:"fee"` // 锻造费(yuanbao)
}

// JwtBag 背包
type JwtBag struct {
	ID     uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID uint   `gorm:"index;comment:用户ID" json:"user_id"`
	ItemID uint   `gorm:"index;comment:道具ID" json:"item_id"`
	Name   string `gorm:"type:varchar(30);comment:名称" json:"name"`
	Cat    string `gorm:"type:varchar(20);comment:分类" json:"cat"`
	Amount int    `gorm:"default:0;comment:数量" json:"amount"`
}

// JwtSkill 技能（书店）
type JwtSkill struct {
	ID        uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name      string `gorm:"type:varchar(30);comment:名称" json:"name"`
	Act       int    `gorm:"default:1;comment:1主动 0被动" json:"act"`  // 1主动 0被动
	Level     int    `gorm:"default:1;comment:学习等级要求" json:"level"` // 学习等级要求
	Price     int    `gorm:"default:0;comment:价格" json:"price"`
	Currency  string `gorm:"type:varchar(10);default:'coins';comment:货币" json:"currency"`
	WeaponReq string `gorm:"type:varchar(20);default:'无限制';comment:WeaponReq" json:"weapon_req"`
	Coef      int    `gorm:"default:100;comment:伤害系数 (倍*100)" json:"coef"` // 伤害系数 (倍*100)
	Hit       int    `gorm:"default:90;comment:命中" json:"hit"`
	Crit      int    `gorm:"default:5;comment:暴击" json:"crit"`
	CritMul   int    `gorm:"default:150;comment:暴击倍数 %" json:"crit_mul"` // 暴击倍数 %
	DodgeAdd  int    `gorm:"default:0;comment:闪避添加" json:"dodge_add"`
	// 被动效果
	PAtk   int    `gorm:"default:0;comment:被动效果" json:"p_atk"`
	PDef   int    `gorm:"default:0;comment:P防御" json:"p_def"`
	PHp    int    `gorm:"default:0;comment:P生命" json:"p_hp"`
	PMp    int    `gorm:"default:0;comment:P魔法" json:"p_mp"`
	Desc   string `gorm:"type:varchar(100);comment:描述" json:"desc"`
	Status int    `gorm:"default:1;comment:状态" json:"status"`
}

// JwtLearnedSkill 已学技能
type JwtLearnedSkill struct {
	ID       uint `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID   uint `gorm:"index:uk_userskill,unique;comment:用户ID" json:"user_id"`
	SkillID  uint `gorm:"index:uk_userskill,unique;comment:技能ID" json:"skill_id"`
	Practice int  `gorm:"default:0;comment:熟练度(修炼/比武增加，复刻原站 熟练度:0/100)" json:"practice"` // 熟练度(修炼/比武增加，复刻原站 熟练度:0/100)
	Level    int  `gorm:"default:1;comment:已学技能当前等级（熟练度满100可领悟升级）" json:"level"`         // 已学技能当前等级（熟练度满100可领悟升级）
	Equip    int  `gorm:"default:0;comment:1已装备（比武/修炼使用该技能）" json:"equip"`               // 1已装备（比武/修炼使用该技能）
}

// JwtGang 帮派
type JwtGang struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name      string    `gorm:"type:varchar(30);uniqueIndex;comment:名称" json:"name"`
	Level     int       `gorm:"default:1;comment:等级" json:"level"`
	Exp       int       `gorm:"default:0;comment:经验" json:"exp"`
	MasterID  uint      `gorm:"index;comment:师傅ID" json:"master_id"`
	Master    string    `gorm:"type:varchar(30);comment:师傅" json:"master"`
	Notice    string    `gorm:"type:varchar(100);comment:公告" json:"notice"`
	Members   int       `gorm:"default:1;comment:成员" json:"members"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

// JwtGangMember 帮派成员
type JwtGangMember struct {
	ID       uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	GangID   uint      `gorm:"index;comment:帮派ID" json:"gang_id"`
	UserID   uint      `gorm:"uniqueIndex;comment:用户ID" json:"user_id"`
	IsMaster int       `gorm:"default:0;comment:是否师傅" json:"is_master"`
	JoinedAt time.Time `gorm:"comment:Joined时间" json:"joined_at"`
}

// JwtGangApply 帮派申请（加入申请，等帮主审批）
type JwtGangApply struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	GangID    uint      `gorm:"index;comment:帮派ID" json:"gang_id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Msg       string    `gorm:"type:varchar(200);comment:消息" json:"msg"`
	Status    int       `gorm:"default:0;comment:0待审批 1同意 2拒绝" json:"status"` // 0待审批 1同意 2拒绝
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

// 精武堂战斗属性（由玩家+装备+被动技能合成，仅运行期计算，不入库）
type JwtCombat struct {
	MaxHp   int `gorm:"comment:上限生命" json:"max_hp"`
	MaxMp   int `gorm:"comment:上限魔法" json:"max_mp"`
	Speed   int `gorm:"comment:速度" json:"speed"`
	Atk     int `gorm:"comment:攻击" json:"atk"`
	Def     int `gorm:"comment:防御" json:"def"`
	Hit     int `gorm:"comment:命中" json:"hit"`
	Crit    int `gorm:"comment:暴击" json:"crit"`
	CritMul int `gorm:"comment:暴击Mul" json:"crit_mul"`
	Dodge   int `gorm:"comment:闪避" json:"dodge"`
}

// JwtChat 精武堂聊天（复刻原站 chat_list.aspx：type=0公共/1个人/2世界）
type JwtChat struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Nick      string    `gorm:"type:varchar(30);comment:昵称" json:"nick"`
	Content   string    `gorm:"type:varchar(200);comment:内容" json:"content"`
	Type      int       `gorm:"default:0;index;comment:0公共 1个人 2世界" json:"type"` // 0公共 1个人 2世界
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

// JwtLog 精武堂动态（对应原站首页「动态」，记录比武等事件）
type JwtLog struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:归属于谁的动态（这条动态展示在谁的动态列表）" json:"user_id"` // 归属于谁的动态（这条动态展示在谁的动态列表）
	Nick      string    `gorm:"type:varchar(30);comment:昵称" json:"nick"`
	Msg       string    `gorm:"type:varchar(200);comment:消息" json:"msg"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

// JwtArenaRecord 比武记录（复刻 比武记录.xhtml：持久化整场战报，供详情查看）
type JwtArenaRecord struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:归属于谁（在谁的记录列表展示）" json:"user_id"` // 归属于谁（在谁的记录列表展示）
	MyUID     uint      `gorm:"comment:我的用户ID" json:"my_uid"`
	MyNick    string    `gorm:"type:varchar(30);comment:我的昵称" json:"my_nick"`
	OppUID    uint      `gorm:"comment:对方用户ID" json:"opp_uid"`
	OppNick   string    `gorm:"type:varchar(30);comment:对方昵称" json:"opp_nick"`
	Result    string    `gorm:"type:varchar(10);comment:胜利/失败/平手" json:"result"` // 胜利/失败/平手
	Exp       int       `gorm:"comment:经验" json:"exp"`
	Coin      int       `gorm:"comment:Coin" json:"coin"`
	MyLevel   int       `gorm:"comment:我的等级" json:"my_level"`
	MyMaxHp   int       `gorm:"comment:我的上限生命" json:"my_max_hp"`
	MyCurHp   int       `gorm:"comment:我的当前生命" json:"my_cur_hp"`
	OppLevel  int       `gorm:"comment:对方等级" json:"opp_level"`
	OppMaxHp  int       `gorm:"comment:对方上限生命" json:"opp_max_hp"`
	OppCurHp  int       `gorm:"comment:对方当前生命" json:"opp_cur_hp"`
	LogsText  string    `gorm:"type:text;comment:战报文本行（\n 分隔）" json:"logs_text"` // 战报文本行（\n 分隔）
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}
