package model

import "time"

// FlaDonation 福利院·捐款慈善基金（每日捐款价高者上榜，榜首可受全社区膜拜）
type FlaDonation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Amount    int       `json:"amount"`
	Word      string    `gorm:"type:varchar(30)" json:"word"` // 捐赠宣言
	Worships  int       `gorm:"default:0" json:"worships"`    // 膜拜数
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (FlaDonation) TableName() string { return "fla_donations" }

// FlaWorship 福利院膜拜记录（每人每日限一次）
type FlaWorship struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	DonID     uint      `gorm:"index" json:"don_id"`  // 被膜拜的当日榜首捐款记录
	UserID    uint      `gorm:"index" json:"user_id"` // 膜拜者
	CreatedAt time.Time `json:"created_at"`
}

func (FlaWorship) TableName() string { return "fla_worships" }
