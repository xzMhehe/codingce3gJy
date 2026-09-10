package model

import "time"

// 抢车位（对齐诺哈三代 ASP 版 wap/game/car 玩法与 wap_car* 表）
// 等级 = 经验/100；停车收入 = 停车小时数*每小时盈利，20%入国库(税)，贡献 = 总收入/10

// ParkUser 玩家数据（对齐 wap_car）
type ParkUser struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex" json:"user_id"`
	Cars      int       `gorm:"default:0" json:"cars"`  // 拥有车辆数(最多10)
	Love      int       `gorm:"default:0" json:"love"`  // 爱心
	Point     int       `gorm:"default:0" json:"point"` // 经验(等级=point/100)
	Contri    int       `gorm:"default:0" json:"contri"`// 贡献
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ParkUser) TableName() string { return "park_users" }

// CarShop 车市车辆（对齐 wap_car_shop）
// dtype 1普通车 2高级车 3酷族车 4贵族车 5试驾车；status 0上架 1下架
type CarShop struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	Name   string `gorm:"type:varchar(30)" json:"name"`
	Icon   string `gorm:"type:varchar(10)" json:"icon"`   // 图片扩展名(复刻存gif/jpg，本站前端按颜色渲染)
	Price  int    `gorm:"default:0" json:"price"`         // 价格(G币)
	Money  int    `gorm:"default:0" json:"money"`         // 每小时盈利(G币/小时)
	DType  int    `gorm:"column:dtype;default:1" json:"dtype"`
	Status int    `gorm:"default:0" json:"status"`
}

func (CarShop) TableName() string { return "car_shops" }

// CarGarage 车库（对齐 wap_car_garage）
// stop_id=0 表示流动中，new_time 为变流动/上次收车时间
type CarGarage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	CarID     uint      `gorm:"default:0" json:"car_id"`
	StopID    uint      `gorm:"default:0" json:"stop_id"`
	NewTime   time.Time `json:"new_time"`
	CreatedAt time.Time `json:"created_at"`
}

func (CarGarage) TableName() string { return "car_garages" }

// CarStop 车位（对齐 wap_car_stop）
// 每个玩家默认3个；car_id>0 且 owner>0 表示被占；owner=0 为空位
type CarStop struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	UserID    uint       `gorm:"index" json:"user_id"`
	Sort      int        `gorm:"default:0" json:"sort"` // 第几号车位
	GarID     uint       `gorm:"default:0" json:"gar_id"`
	CarID     uint       `gorm:"default:0" json:"car_id"`
	Owner     uint       `gorm:"default:0" json:"owner"` // 停车人 0=空位
	StopTime  *time.Time `json:"stop_time"`              // 停车时间(空位为NULL)
	CreatedAt time.Time  `json:"created_at"`
}

func (CarStop) TableName() string { return "car_stops" }

// CarLog 加入记录（对齐 wap_car_log：最新加入）
type CarLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Name      string    `gorm:"type:varchar(30)" json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func (CarLog) TableName() string { return "car_logs" }

// CarMsg 游戏消息（停车/贴条/赠送通知，对齐 wap_farm_message 做法）
type CarMsg struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"` // 接收人
	FID       uint      `gorm:"column:fid;default:0" json:"fid"` // 发送人 0=系统
	Content   string    `gorm:"type:varchar(200)" json:"content"`
	Status    int       `gorm:"default:0" json:"status"` // 0未读 1已读
	CreatedAt time.Time `json:"created_at"`
}

func (CarMsg) TableName() string { return "car_msgs" }
