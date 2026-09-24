package model

import "time"

// 开心农场：农场实体（每个用户一个，懒创建）
// 对齐诺哈三代 ASP 版 wap_farm 表
type Farm struct {
	ID          uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID      uint      `gorm:"uniqueIndex;comment:用户ID" json:"user_id"`
	Name        string    `gorm:"type:varchar(30);comment:农场名" json:"name"`                   // 农场名
	Level       int       `gorm:"default:1;comment:农场等级 1 起" json:"level"`                    // 农场等级 1 起
	Point       int       `gorm:"default:0;comment:当前等级经验" json:"point"`                      // 当前等级经验
	Steal       int       `gorm:"default:0;comment:偷取（预留）" json:"steal"`                      // 预留
	Mucks       int       `gorm:"default:3;comment:今日剩余施肥次数(每日恢复为3)" json:"mucks"`            // 今日剩余施肥次数(每日恢复为3)
	CSteal      int       `gorm:"default:0;comment:摘取权限 0所有人 1仅好友 4禁止" json:"csteal"`         // 摘取权限 0所有人 1仅好友 4禁止
	LastRefresh string    `gorm:"type:varchar(10);comment:最近刷新日期(施肥恢复用)" json:"last_refresh"` // 最近刷新日期(施肥恢复用)
	CreatedAt   time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt   time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (Farm) TableName() string { return "farms" }

// 作物种子（商店/种植）
// 对齐 wap_farm_seed：aging/again 为分钟
type FarmSeed struct {
	ID     uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name   string `gorm:"type:varchar(30);comment:名称" json:"name"`
	Cycle  int    `gorm:"default:1;comment:共几季" json:"cycle"`                      // 共几季
	Aging  int    `gorm:"default:15;comment:首季成熟 分钟" json:"aging"`                 // 首季成熟 分钟
	Again  int    `gorm:"default:0;comment:再次成熟 分钟(多季)" json:"again"`              // 再次成熟 分钟(多季)
	Yield  int    `gorm:"default:8;comment:每季产量" json:"yield"`                     // 每季产量
	Price  int    `gorm:"default:10;comment:果实单价(种子价=price*5*cycle)" json:"price"` // 果实单价(种子价=price*5*cycle)
	Point  int    `gorm:"default:10;comment:每季收获经验" json:"point"`                  // 每季收获经验
	Level  int    `gorm:"default:1;comment:种植所需农场等级" json:"level"`                 // 种植所需农场等级
	Status int    `gorm:"default:1;comment:状态" json:"status"`
}

func (FarmSeed) TableName() string { return "farm_seeds" }

// 化肥（缩短成熟时间）
// 对齐 wap_farm_muck：speed 为减少的分钟数
type FarmMuck struct {
	ID     uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name   string `gorm:"type:varchar(30);comment:名称" json:"name"`
	Speed  int    `gorm:"default:10;comment:减少 分钟" json:"speed"` // 减少 分钟
	Price  int    `gorm:"default:100;comment:价格" json:"price"`
	Status int    `gorm:"default:1;comment:状态" json:"status"`
}

func (FarmMuck) TableName() string { return "farm_mucks" }

// 陷阱（防偷菜）
// 对齐 wap_farm_trap：rate 为触发几率%
type FarmTrap struct {
	ID     uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name   string `gorm:"type:varchar(30);comment:名称" json:"name"`
	Rate   int    `gorm:"default:30;comment:触发几率 %" json:"rate"` // 触发几率 %
	Price  int    `gorm:"default:150;comment:价格" json:"price"`
	Status int    `gorm:"default:1;comment:状态" json:"status"`
}

func (FarmTrap) TableName() string { return "farm_traps" }

// 农场地块
// 对齐 wap_farm_land：drys/weed/pest 0未触发 1需处理 2自然良好 3已处理
type FarmLand struct {
	ID        uint       `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint       `gorm:"index;comment:用户ID" json:"user_id"`
	Sort      int        `gorm:"default:0;comment:第几块地" json:"sort"`     // 第几块地
	Type      int        `gorm:"default:0;comment:0空地 1种植中" json:"type"` // 0空地 1种植中
	Plow      int        `gorm:"default:0;comment:0未翻地 1已翻" json:"plow"` // 0未翻地 1已翻
	SeedID    uint       `gorm:"default:0;comment:种子ID" json:"seed_id"`
	Name      string     `gorm:"type:varchar(30);comment:当前作物名" json:"name"` // 当前作物名
	Drys      int        `gorm:"default:0;comment:Drys" json:"drys"`
	Weed      int        `gorm:"default:0;comment:杂草" json:"weed"`
	Pest      int        `gorm:"default:0;comment:害虫" json:"pest"`
	Trap      int        `gorm:"default:0;comment:陷阱几率 %" json:"trap"`      // 陷阱几率 %
	Yield     int        `gorm:"default:0;comment:本季剩余果实数" json:"yield"`    // 本季剩余果实数
	Cycle     int        `gorm:"default:0;comment:作物总季数" json:"cycle"`      // 作物总季数
	Period    int        `gorm:"default:0;comment:当前第几季" json:"period"`     // 当前第几季
	PlantedAt *time.Time `gorm:"comment:本季种植时间(空地为NULL)" json:"planted_at"` // 本季种植时间(空地为NULL)
	MatureAt  *time.Time `gorm:"comment:成熟时间(空地为NULL)" json:"mature_at"`    // 成熟时间(空地为NULL)
	CreatedAt time.Time  `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time  `gorm:"comment:更新时间" json:"updated_at"`
}

func (FarmLand) TableName() string { return "farm_lands" }

// 农场背包：dtype 1种子 2化肥 3陷阱 11果实
// 对齐 wap_farm_bag
type FarmBag struct {
	ID     uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID uint   `gorm:"index;comment:用户ID" json:"user_id"`
	Oid    uint   `gorm:"index;comment:对应种子/化肥/陷阱 id" json:"oid"` // 对应种子/化肥/陷阱 id
	Name   string `gorm:"type:varchar(30);comment:名称" json:"name"`
	DType  int    `gorm:"column:dtype;default:1;comment:D类型" json:"dtype"`
	Amount int    `gorm:"default:0;comment:数量" json:"amount"`
}

func (FarmBag) TableName() string { return "farm_bags" }

// 农场消息（系统通知/偷菜留言）
// 对齐 wap_farm_message
type FarmMsg struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:接收人" json:"user_id"`                 // 接收人
	FID       uint      `gorm:"column:fid;default:0;comment:发送人 0=系统" json:"fid"` // 发送人 0=系统
	Content   string    `gorm:"type:varchar(200);comment:内容" json:"content"`
	Status    int       `gorm:"default:0;comment:0未读 1已读" json:"status"` // 0未读 1已读
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (FarmMsg) TableName() string { return "farm_msgs" }

// 奴隶记录（偷菜触发陷阱）
// 对齐 wap_farm_slave：uid=农场主 fid=奴隶，2天后自动解除
type FarmSlave struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	OwnerUID  uint      `gorm:"index;comment:农场主" json:"owner_uid"`         // 农场主
	FID       uint      `gorm:"column:fid;index;comment:奴隶(用户)" json:"fid"` // 奴隶(用户)
	Name      string    `gorm:"type:varchar(20);comment:名称" json:"name"`
	Punish    int       `gorm:"default:0;comment:Punish" json:"punish"`
	Appease   int       `gorm:"default:0;comment:Appease" json:"appease"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (FarmSlave) TableName() string { return "farm_slaves" }

// 偷菜记录（一块地每人只能偷一次）
// 对齐 wap_farm_steal
type FarmSteal struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	OwnerUID  uint      `gorm:"index;comment:拥有者用户ID" json:"owner_uid"`
	FID       uint      `gorm:"column:fid;index;comment:小偷" json:"fid"` // 小偷
	LandID    uint      `gorm:"index;comment:土地ID" json:"land_id"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (FarmSteal) TableName() string { return "farm_steals" }
