package model

import "time"

// 好友分组
type FriendGroup struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Name      string    `gorm:"type:varchar(20)" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (FriendGroup) TableName() string { return "friend_groups" }

// 分组内的好友
type FriendGroupItem struct {
	ID       uint      `gorm:"primaryKey" json:"id"`
	GroupID  uint      `gorm:"index" json:"group_id"`
	UserID   uint      `json:"user_id"`   // 归属者
	FriendID uint      `json:"friend_id"` // 好友id
	CreatedAt time.Time `json:"created_at"`
}

func (FriendGroupItem) TableName() string { return "friend_group_items" }
