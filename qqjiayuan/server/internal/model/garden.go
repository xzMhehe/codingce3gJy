package model

import "time"

// 魔法花园地块
type GardenPlot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Plot      int       `gorm:"index" json:"plot"`   // 第几块地
	Crop      string     `gorm:"type:varchar(20)" json:"crop"` // 作物名
	SeedAt    *time.Time `json:"seed_at"`
	Status    int        `gorm:"default:0" json:"status"` // 0空 1种植中
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (GardenPlot) TableName() string { return "garden_plots" }

// 花朵库存（花篮）
type UserFlower struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	UserID uint   `gorm:"uniqueIndex:uk_uf" json:"user_id"`
	Flower string `gorm:"uniqueIndex:uk_uf;type:varchar(20)" json:"flower"`
	Count  int    `gorm:"default:0" json:"count"`
}

func (UserFlower) TableName() string { return "user_flowers" }

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
