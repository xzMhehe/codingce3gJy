package model

import "time"

// TtouApply T台秀上榜申请（我要上榜）
type TtouApply struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Slogan    string    `gorm:"type:varchar(100);comment:上榜宣言" json:"slogan"` // 上榜宣言
	Status    int       `gorm:"default:0;comment:0待选 1已当选" json:"status"`     // 0待选 1已当选
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

// TtouWorship T台秀膜拜记录（我要膜拜）
type TtouWorship struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	TargetID  uint      `gorm:"index;comment:被膜拜的秀主" json:"target_id"` // 被膜拜的秀主
	UserID    uint      `gorm:"index;comment:膜拜者" json:"user_id"`      // 膜拜者
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}
