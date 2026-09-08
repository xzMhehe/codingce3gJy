package model

import "time"

// 帖子赞/踩（一人一票，value: 1赞 -1踩）
type ThreadVote struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ThreadID  uint      `gorm:"uniqueIndex:uk_tv" json:"thread_id"`
	UserID    uint      `gorm:"uniqueIndex:uk_tv" json:"user_id"`
	Value     int       `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

func (ThreadVote) TableName() string { return "thread_votes" }

// 回复点赞（一人一票，仅赞）
type ReplyVote struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ReplyID   uint      `gorm:"uniqueIndex:uk_rv" json:"reply_id"`
	UserID    uint      `gorm:"uniqueIndex:uk_rv" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (ReplyVote) TableName() string { return "reply_votes" }

// 帖子打赏记录（金币从打赏人转给楼主）
type ThreadGift struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ThreadID  uint      `gorm:"index" json:"thread_id"`
	SenderID  uint      `gorm:"index" json:"sender_id"`
	Coins     int       `json:"coins"`
	CreatedAt time.Time `json:"created_at"`
	Sender    *User     `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
}

func (ThreadGift) TableName() string { return "thread_gifts" }

// 帖子送花记录（花从送花人花篮扣除）
type ThreadFlower struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ThreadID  uint      `gorm:"index" json:"thread_id"`
	SenderID  uint      `gorm:"index" json:"sender_id"`
	Flower    string    `gorm:"type:varchar(20)" json:"flower"`
	Count     int       `json:"count"`
	CreatedAt time.Time `json:"created_at"`
	Sender    *User     `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
}

func (ThreadFlower) TableName() string { return "thread_flowers" }

// 公告/广播
type Announcement struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Type      string    `gorm:"type:varchar(10)" json:"type"` // notice公告 broadcast广播 activity活动
	Title     string    `gorm:"type:varchar(100)" json:"title"`
	Content   string    `gorm:"type:text" json:"content"`
	Status    int       `gorm:"default:1" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}