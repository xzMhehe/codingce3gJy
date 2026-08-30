package model

import "time"

// 每日签到
type SignIn struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index:idx_user_date,unique" json:"user_id"`
	SignDate  string    `gorm:"type:varchar(10);index:idx_user_date,unique" json:"sign_date"` // 2006-01-02
	Consec    int       `json:"consec"`                                                        // 连续签到天数
	Reward    int       `json:"reward"`                                                        // 金币奖励
	CreatedAt time.Time `json:"created_at"`
}

// 好友关系：申请方向 user -> friend，status: 0待处理 1已同意 2已拒绝
type Friendship struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"` // 发起方
	FriendID  uint      `gorm:"index" json:"friend_id"` // 接收方
	Status    int       `gorm:"default:0" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// 私信
type PrivateMessage struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SenderID   uint      `gorm:"index" json:"sender_id"`
	ReceiverID uint      `gorm:"index" json:"receiver_id"`
	Content    string    `gorm:"type:varchar(500)" json:"content"`
	IsRead     int       `gorm:"default:0" json:"is_read"`
	CreatedAt  time.Time `json:"created_at"`
	Sender     *User     `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
}

// 聊天室消息（公共聊天，轮询拉取）
type ChatMessage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Content   string    `gorm:"type:varchar(500)" json:"content"`
	CreatedAt time.Time `json:"created_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// 系统通知：reply回复/好友friend/系统system
type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Type      string    `gorm:"type:varchar(20)" json:"type"`
	Title     string    `gorm:"type:varchar(100)" json:"title"`
	Content   string    `gorm:"type:varchar(500)" json:"content"`
	RefID     uint      `json:"ref_id"`
	IsRead    int       `gorm:"default:0" json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}
