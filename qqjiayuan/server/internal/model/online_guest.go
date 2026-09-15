package model

import "time"

// OnlineGuest 全站在线游客（诺哈 online.html：游客按 IP 展示在在线用户列表，30 分钟滑动窗口）
type OnlineGuest struct {
	IP           string    `gorm:"type:varchar(45);primaryKey" json:"ip"`
	LastActiveAt time.Time `json:"last_active_at"`
}

func (OnlineGuest) TableName() string { return "online_guests" }
