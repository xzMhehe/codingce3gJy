package model

import "time"

// 板块：parent_id=0 为分区（如公共论坛），否则为子板块
type Board struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ParentID    uint      `gorm:"index;default:0" json:"parent_id"`
	Name        string    `gorm:"type:varchar(30)" json:"name"`
	Description string    `gorm:"type:varchar(200)" json:"description"`
	Sort        int       `gorm:"default:0" json:"sort"`
	Status      int       `gorm:"default:1" json:"status"` // 1显示 0隐藏
	ThreadCount int       `gorm:"default:0" json:"thread_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// 帖子
 type Thread struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	BoardID     uint       `gorm:"index" json:"board_id"`
	UserID      uint       `gorm:"index" json:"user_id"`
	Title       string     `gorm:"type:varchar(100)" json:"title"`
	Content     string     `gorm:"type:text" json:"content"`
	IsTop       int        `gorm:"default:0" json:"is_top"`
	IsFine      int        `gorm:"default:0" json:"is_fine"`
	ViewCount   int        `gorm:"default:0" json:"view_count"`
	ReplyCount  int        `gorm:"default:0" json:"reply_count"`
	LikeCount   int        `gorm:"default:0" json:"like_count"`
	DislikeCount int       `gorm:"default:0" json:"dislike_count"`
	ShareCount  int        `gorm:"default:0" json:"share_count"`
	GiftTotal   int        `gorm:"default:0" json:"gift_total"`
	FlowerCount int        `gorm:"default:0" json:"flower_count"`
	Status      int        `gorm:"default:1" json:"status"` // 1正常 0删除
	LastReplyAt *time.Time `json:"last_reply_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	User        *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Board       *Board     `gorm:"foreignKey:BoardID" json:"board,omitempty"`
}

// 回复（盖楼），Floor 为楼层数，1楼是楼主
type Reply struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ThreadID  uint      `gorm:"index" json:"thread_id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Content   string    `gorm:"type:text" json:"content"`
	Floor     int       `json:"floor"`
	LikeCount int       `gorm:"default:0" json:"like_count"`
	Status    int       `gorm:"default:1" json:"status"` // 1正常 0删除
	CreatedAt time.Time `json:"created_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Thread    *Thread   `gorm:"foreignKey:ThreadID" json:"thread,omitempty"`
}

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
