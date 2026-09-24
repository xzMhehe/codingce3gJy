package model

import "time"

// WelfareFund 福利院·慈善基金池（单行表，pool 为当前池内 G币）
type WelfareFund struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	Pool      int       `gorm:"default:0;comment:Pool" json:"pool"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (WelfareFund) TableName() string { return "welfare_funds" }

// WelfareClaim 福利院·每日领取记录
type WelfareClaim struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Amount    int       `gorm:"comment:数量" json:"amount"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (WelfareClaim) TableName() string { return "welfare_claims" }

// WelfareDonate 福利院·捐献记录（捐 100 慈善币得 1 财富值）
type WelfareDonate struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Amount    int       `gorm:"comment:数量" json:"amount"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (WelfareDonate) TableName() string { return "welfare_donations" }
