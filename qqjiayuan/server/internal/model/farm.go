package model

import "time"

// 开心农场：农场实体（每个用户一个，懒创建）
// 对齐诺哈三代 ASP 版 wap_farm 表
type Farm struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex" json:"user_id"`
	Name      string    `gorm:"type:varchar(30)" json:"name"`  // 农场名
	Level     int       `gorm:"default:1" json:"level"`        // 农场等级 1 起
	Point     int       `gorm:"default:0" json:"point"`        // 当前等级经验
	Steal     int       `gorm:"default:0" json:"steal"`        // 预留
	Mucks     int       `gorm:"default:3" json:"mucks"`        // 今日剩余施肥次数(每日恢复为3)
	CSteal    int       `gorm:"default:0" json:"csteal"`       // 摘取权限 0所有人 1仅好友 4禁止
	LastRefresh string  `gorm:"type:varchar(10)" json:"last_refresh"` // 最近刷新日期(施肥恢复用)
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Farm) TableName() string { return "farms" }

// 作物种子（商店/种植）
// 对齐 wap_farm_seed：aging/again 为分钟
type FarmSeed struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	Name   string `gorm:"type:varchar(30)" json:"name"`
	Cycle  int    `gorm:"default:1" json:"cycle"`   // 共几季
	Aging  int    `gorm:"default:15" json:"aging"`  // 首季成熟 分钟
	Again  int    `gorm:"default:0" json:"again"`   // 再次成熟 分钟(多季)
	Yield  int    `gorm:"default:8" json:"yield"`   // 每季产量
	Price  int    `gorm:"default:10" json:"price"`  // 果实单价(种子价=price*5*cycle)
	Point  int    `gorm:"default:10" json:"point"`  // 每季收获经验
	Level  int    `gorm:"default:1" json:"level"`   // 种植所需农场等级
	Status int    `gorm:"default:1" json:"status"`
}

func (FarmSeed) TableName() string { return "farm_seeds" }

// 化肥（缩短成熟时间）
// 对齐 wap_farm_muck：speed 为减少的分钟数
type FarmMuck struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	Name   string `gorm:"type:varchar(30)" json:"name"`
	Speed  int    `gorm:"default:10" json:"speed"` // 减少 分钟
	Price  int    `gorm:"default:100" json:"price"`
	Status int    `gorm:"default:1" json:"status"`
}

func (FarmMuck) TableName() string { return "farm_mucks" }

// 陷阱（防偷菜）
// 对齐 wap_farm_trap：rate 为触发几率%
type FarmTrap struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	Name   string `gorm:"type:varchar(30)" json:"name"`
	Rate   int    `gorm:"default:30" json:"rate"` // 触发几率 %
	Price  int    `gorm:"default:150" json:"price"`
	Status int    `gorm:"default:1" json:"status"`
}

func (FarmTrap) TableName() string { return "farm_traps" }

// 农场地块
// 对齐 wap_farm_land：drys/weed/pest 0未触发 1需处理 2自然良好 3已处理
type FarmLand struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	UserID    uint   `gorm:"index" json:"user_id"`
	Sort      int    `gorm:"default:0" json:"sort"`          // 第几块地
	Type      int    `gorm:"default:0" json:"type"`          // 0空地 1种植中
	Plow      int    `gorm:"default:0" json:"plow"`          // 0未翻地 1已翻
	SeedID    uint   `gorm:"default:0" json:"seed_id"`
	Name      string `gorm:"type:varchar(30)" json:"name"`   // 当前作物名
	Drys      int    `gorm:"default:0" json:"drys"`
	Weed      int    `gorm:"default:0" json:"weed"`
	Pest      int    `gorm:"default:0" json:"pest"`
	Trap      int    `gorm:"default:0" json:"trap"`          // 陷阱几率 %
	Yield     int    `gorm:"default:0" json:"yield"`         // 本季剩余果实数
	Cycle     int    `gorm:"default:0" json:"cycle"`         // 作物总季数
	Period    int    `gorm:"default:0" json:"period"`        // 当前第几季
	PlantedAt *time.Time `json:"planted_at"`                  // 本季种植时间(空地为NULL)
	MatureAt  *time.Time `json:"mature_at"`                   // 成熟时间(空地为NULL)
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (FarmLand) TableName() string { return "farm_lands" }

// 农场背包：dtype 1种子 2化肥 3陷阱 11果实
// 对齐 wap_farm_bag
type FarmBag struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	UserID uint   `gorm:"index" json:"user_id"`
	Oid    uint   `gorm:"index" json:"oid"` // 对应种子/化肥/陷阱 id
	Name   string `gorm:"type:varchar(30)" json:"name"`
	DType  int    `gorm:"column:dtype;default:1" json:"dtype"`
	Amount int    `gorm:"default:0" json:"amount"`
}

func (FarmBag) TableName() string { return "farm_bags" }

// 农场消息（系统通知/偷菜留言）
// 对齐 wap_farm_message
type FarmMsg struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"` // 接收人
	FID       uint      `gorm:"column:fid;default:0" json:"fid"` // 发送人 0=系统
	Content   string    `gorm:"type:varchar(200)" json:"content"`
	Status    int       `gorm:"default:0" json:"status"` // 0未读 1已读
	CreatedAt time.Time `json:"created_at"`
}

func (FarmMsg) TableName() string { return "farm_msgs" }

// 奴隶记录（偷菜触发陷阱）
// 对齐 wap_farm_slave：uid=农场主 fid=奴隶，2天后自动解除
type FarmSlave struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	OwnerUID  uint      `gorm:"index" json:"owner_uid"` // 农场主
	FID       uint      `gorm:"column:fid;index" json:"fid"` // 奴隶(用户)
	Name      string    `gorm:"type:varchar(20)" json:"name"`
	Punish    int       `gorm:"default:0" json:"punish"`
	Appease   int       `gorm:"default:0" json:"appease"`
	CreatedAt time.Time `json:"created_at"`
}

func (FarmSlave) TableName() string { return "farm_slaves" }

// 偷菜记录（一块地每人只能偷一次）
// 对齐 wap_farm_steal
type FarmSteal struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	OwnerUID  uint      `gorm:"index" json:"owner_uid"`
	FID       uint      `gorm:"column:fid;index" json:"fid"` // 小偷
	LandID    uint      `gorm:"index" json:"land_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (FarmSteal) TableName() string { return "farm_steals" }
