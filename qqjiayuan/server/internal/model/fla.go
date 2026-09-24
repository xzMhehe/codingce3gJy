package model

import "time"

// FlaDonation 福利院·捐款慈善基金（每日捐款价高者上榜，榜首可受全社区膜拜）
type FlaDonation struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Amount    int       `gorm:"comment:数量" json:"amount"`
	Word      string    `gorm:"type:varchar(30);comment:捐赠宣言" json:"word"` // 捐赠宣言
	Worships  int       `gorm:"default:0;comment:膜拜数" json:"worships"`     // 膜拜数
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (FlaDonation) TableName() string { return "fla_donations" }

// FlaWorship 福利院膜拜记录（每人每日限一次）
type FlaWorship struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	DonID     uint      `gorm:"index;comment:被膜拜的当日榜首捐款记录" json:"don_id"` // 被膜拜的当日榜首捐款记录
	UserID    uint      `gorm:"index;comment:膜拜者" json:"user_id"`         // 膜拜者
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (FlaWorship) TableName() string { return "fla_worships" }
