package model

import "time"

// 魔法花园：花园实体（每个用户一座，懒创建）
// 对齐诺哈三代 ASP 版 wap_garden 表
type Garden struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	UserID     uint   `gorm:"uniqueIndex" json:"user_id"`
	Name       string `gorm:"type:varchar(30)" json:"name"`        // 花园名
	Level      int    `gorm:"default:1" json:"level"`              // 花园等级 1 起
	Point      int    `gorm:"default:0" json:"point"`              // 当前等级经验
	Lands      int    `gorm:"default:2" json:"lands"`              // 花圃数量
	Notice     string `gorm:"type:varchar(100)" json:"notice"`     // 公告
	Common     int    `gorm:"default:0" json:"common"`             // 已点亮普通图谱
	Festival   int    `gorm:"default:0" json:"festival"`           // 已点亮独特图谱
	Scarce     int    `gorm:"default:0" json:"scarce"`             // 已点亮珍稀图谱
	BasketCnt  int    `gorm:"default:0" json:"basket_cnt"`         // 花篮累计
	BottleCnt  int    `gorm:"default:0" json:"bottle_cnt"`         // 花瓶累计
	Config     int    `gorm:"default:0" json:"config"`             // 采摘权限 0全部 1仅好友 2禁止
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (Garden) TableName() string { return "gardens" }

// 花种（商店/背包/播种）
// 对齐 wap_garden_seed：seed/ling/buds 为分钟
type GardenSeed struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	Name   string `gorm:"type:varchar(30)" json:"name"`
	DType  int    `gorm:"default:0" json:"dtype"`   // 0普通可购买 1特殊(魔法屋合成)
	Level  int    `gorm:"default:1" json:"level"`   // 购买所需花园等级
	Price  int    `gorm:"default:0" json:"price"`   // 金币价格(特殊种子为0)
	Seed   int    `gorm:"default:1" json:"seed"`    // 种子期 分钟
	Ling   int    `gorm:"default:1" json:"ling"`    // 花苗期 分钟
	Buds   int    `gorm:"default:1" json:"buds"`    // 花蕾期 分钟
	Less   int    `gorm:"default:2" json:"less"`    // 最低产量
	More   int    `gorm:"default:4" json:"more"`    // 最高产量
	Remark string `gorm:"type:varchar(60)" json:"remark"` // 鲜花花语
	Status int    `gorm:"default:1" json:"status"`
}

func (GardenSeed) TableName() string { return "garden_seeds" }

// 花之图谱（开花时随机变出一朵）
// 对齐 wap_garden_map：dtype 0普通 1独特 2珍稀
type GardenMap struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	SeedID uint   `gorm:"index" json:"seed_id"` // 所属种子
	Name   string `gorm:"type:varchar(30)" json:"name"`
	DType  int    `gorm:"default:0" json:"dtype"`
	Img    string `gorm:"type:varchar(50)" json:"img"` // 图鉴图片 m_s_X.gif
}

func (GardenMap) TableName() string { return "garden_maps" }

// 魔法花园地块（花圃）
// 对齐 wap_garden_land：status 0空 1生长中 2成熟
type GardenPlot struct {
	ID      uint       `gorm:"primaryKey" json:"id"`
	UserID  uint       `gorm:"index" json:"user_id"`
	Plot    int        `gorm:"index" json:"plot"` // 第几块地
	SeedID  uint       `gorm:"default:0" json:"seed_id"`
	Name    string     `gorm:"type:varchar(30)" json:"name"` // 当前花朵名(开花后)
	Drys    int        `gorm:"default:0" json:"drys"`        // 0需浇水 1已浇水
	Weed    int        `gorm:"default:0" json:"weed"`        // 0需锄草 1已锄草
	Pest    int        `gorm:"default:0" json:"pest"`        // 0需捉虫 1已捉虫
	Yield   int        `gorm:"default:0" json:"yield"`       // 成熟产量
	Amount  int        `gorm:"default:0" json:"amount"`      // 当前可摘数量
	Status  int        `gorm:"default:0" json:"status"`      // 0空 1生长中 2成熟
	SeedAt  *time.Time `json:"seed_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (GardenPlot) TableName() string { return "garden_plots" }

// 花园背包（种子库存）
// 对齐 wap_garden_bag
type GardenBag struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	UserID uint   `gorm:"index" json:"user_id"`
	SeedID uint   `gorm:"index" json:"seed_id"`
	Name   string `gorm:"type:varchar(30)" json:"name"`
	Amount int    `gorm:"default:0" json:"amount"`
}

func (GardenBag) TableName() string { return "garden_bags" }

// 花朵库存（花篮）——兼容保留原表名 user_flowers
type UserFlower struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	UserID uint   `gorm:"uniqueIndex:uk_uf" json:"user_id"`
	Flower string `gorm:"uniqueIndex:uk_uf;type:varchar(30)" json:"flower"`
	Count  int    `gorm:"default:0" json:"count"`
}

func (UserFlower) TableName() string { return "user_flowers" }

// 花瓶（收到的花）——对齐 wap_garden_bottle
type GardenBottle struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	UserID uint   `gorm:"uniqueIndex:uk_gb" json:"user_id"`
	Flower string `gorm:"uniqueIndex:uk_gb;type:varchar(30)" json:"flower"`
	Count  int    `gorm:"default:0" json:"count"`
}

func (GardenBottle) TableName() string { return "garden_bottles" }

// 送花记录 —— 对齐 wap_garden_gift
type GardenGift struct {
	ID      uint      `gorm:"primaryKey" json:"id"`
	FromUID uint      `gorm:"index" json:"from_uid"`
	ToUID   uint      `gorm:"index" json:"to_uid"`
	Flower  string    `gorm:"type:varchar(30)" json:"flower"`
	Amount  int       `gorm:"default:0" json:"amount"`
	Remark  string    `gorm:"type:varchar(100)" json:"remark"`
	CreatedAt time.Time `json:"created_at"`
}

func (GardenGift) TableName() string { return "garden_gifts" }

// 合成配方 —— 对齐 wap_garden_mix：消耗花篮花朵 → 获得特殊种子
type GardenMix struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	SeedID  uint   `gorm:"index" json:"seed_id"` // 产物种子
	Flower  string `gorm:"type:varchar(30)" json:"flower"` // 材料花
	Need    int    `gorm:"default:1" json:"need"` // 需要数量
}

func (GardenMix) TableName() string { return "garden_mixes" }

// 花园消息 —— 对齐 wap_garden_message（uid=操作者 fid=花园主）
type GardenMsg struct {
	ID      uint      `gorm:"primaryKey" json:"id"`
	UID     uint      `gorm:"column:uid;index" json:"uid"`
	FID     uint      `gorm:"column:fid;index" json:"fid"`
	Remark  string    `gorm:"type:varchar(100)" json:"remark"`
	CreatedAt time.Time `json:"created_at"`
}

func (GardenMsg) TableName() string { return "garden_msgs" }

// 图鉴点亮记录 —— 对齐 wap_garden_map_log
type GardenMapLog struct {
	ID     uint `gorm:"primaryKey" json:"id"`
	UserID uint `gorm:"uniqueIndex:uk_gml" json:"user_id"`
	MapID  uint `gorm:"uniqueIndex:uk_gml" json:"map_id"`
}

func (GardenMapLog) TableName() string { return "garden_map_logs" }

// 采摘记录 —— 对齐 wap_garden_land_log（每人每花圃限摘一次）
type GardenLandLog struct {
	ID     uint `gorm:"primaryKey" json:"id"`
	LandID uint `gorm:"index" json:"land_id"`
	UserID uint `gorm:"index" json:"user_id"`
}

func (GardenLandLog) TableName() string { return "garden_land_logs" }

// 花园活动（后台可管理）
type GardenActivity struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"type:varchar(60)" json:"title"`
	Desc      string    `gorm:"type:varchar(200)" json:"desc"`
	Status    int       `gorm:"default:1" json:"status"` // 1显示 0隐藏
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (GardenActivity) TableName() string { return "garden_activities" }

// 精灵花册（魔法花园精灵收集，后台可管理）
type GardenElf struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Name     string `gorm:"type:varchar(30)" json:"name"`
	Desc     string `gorm:"type:varchar(200)" json:"desc"`
	Img      string `gorm:"type:varchar(50)" json:"img"`    // 精灵图片 elf_X.gif
	NeedMap  int    `gorm:"default:1" json:"need_map"`      // 解锁所需点亮图谱数
	Sort     int    `gorm:"default:0" json:"sort"`
	Status   int    `gorm:"default:1" json:"status"`        // 1启用 0停用
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (GardenElf) TableName() string { return "garden_elves" }

// 精灵点亮记录
type GardenElfLog struct {
	ID     uint `gorm:"primaryKey" json:"id"`
	UserID uint `gorm:"uniqueIndex:uk_gel" json:"user_id"`
	ElfID  uint `gorm:"uniqueIndex:uk_gel" json:"elf_id"`
}

func (GardenElfLog) TableName() string { return "garden_elf_logs" }

// 七日签到（对齐参考站 check：周一~周日，7天一轮，奖励按累计天数）
type GardenSign struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index:uk_gs" json:"user_id"`
	SignDate  string    `gorm:"type:varchar(10);index:uk_gs" json:"sign_date"` // YYYY-MM-DD
	WeekDay   int       `gorm:"default:0" json:"week_day"`                     // 1-7 周一~周日
	DayNo     int       `gorm:"default:0" json:"day_no"`                       // 本轮累计签到第几天
	CreatedAt time.Time `json:"created_at"`
}

func (GardenSign) TableName() string { return "garden_signs" }
