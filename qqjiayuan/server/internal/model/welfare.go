package model

import "time"

// WelfareFund 福利院·慈善基金池（单行表，pool 为当前池内 G币）
type WelfareFund struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Pool      int       `gorm:"default:0" json:"pool"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (WelfareFund) TableName() string { return "welfare_funds" }

// WelfareClaim 福利院·每日领取记录
type WelfareClaim struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Amount    int       `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (WelfareClaim) TableName() string { return "welfare_claims" }

// WelfareDonate 福利院·捐献记录（捐 100 慈善币得 1 财富值）
type WelfareDonate struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Amount    int       `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (WelfareDonate) TableName() string { return "welfare_donations" }
