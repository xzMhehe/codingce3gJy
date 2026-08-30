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
	Status    int       `gorm:"default:1" json:"status"` // 1正常 0删除
	CreatedAt time.Time `json:"created_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Thread    *Thread   `gorm:"foreignKey:ThreadID" json:"thread,omitempty"`
}

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
