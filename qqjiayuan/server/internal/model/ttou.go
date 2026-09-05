package model

import "time"

// TtouApply T台秀上榜申请（我要上榜）
type TtouApply struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Slogan    string    `gorm:"type:varchar(100)" json:"slogan"` // 上榜宣言
	Status    int       `gorm:"default:0" json:"status"`         // 0待选 1已当选
	CreatedAt time.Time `json:"created_at"`
}

// TtouWorship T台秀膜拜记录（我要膜拜）
type TtouWorship struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TargetID  uint      `gorm:"index" json:"target_id"` // 被膜拜的秀主
	UserID    uint      `gorm:"index" json:"user_id"`   // 膜拜者
	CreatedAt time.Time `json:"created_at"`
}
