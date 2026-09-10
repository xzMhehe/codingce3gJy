package model

import "time"

// CityManager 同城城市管理（参考诺哈 wap_manage：bid=城市板块，uid=管理用户，name=职务名称）
type CityManager struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	BoardID   uint      `gorm:"index" json:"board_id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Title     string    `gorm:"type:varchar(30)" json:"title"`
	Sort      int       `gorm:"default:0" json:"sort"`
	CreatedAt time.Time `json:"created_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (CityManager) TableName() string { return "city_managers" }
