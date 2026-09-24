package model

import "time"

// 好友分组（对齐诺哈 wap_friend_group）
type FriendGroup struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Name      string    `gorm:"type:varchar(20);comment:名称" json:"name"`
	Sort      int       `gorm:"default:0;comment:排序（诺哈 wap_friend_group.sort）" json:"sort"`        // 排序（诺哈 wap_friend_group.sort）
	Amount    int       `gorm:"default:0;comment:组内好友数（诺哈 wap_friend_group.amount）" json:"amount"` // 组内好友数（诺哈 wap_friend_group.amount）
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (FriendGroup) TableName() string { return "friend_groups" }

// 分组内的好友
type FriendGroupItem struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	GroupID   uint      `gorm:"index;comment:分组ID" json:"group_id"`
	UserID    uint      `gorm:"comment:归属者" json:"user_id"`    // 归属者
	FriendID  uint      `gorm:"comment:好友id" json:"friend_id"` // 好友id
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (FriendGroupItem) TableName() string { return "friend_group_items" }
