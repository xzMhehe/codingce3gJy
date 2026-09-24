package model

import "time"

// 帖子赞/踩（一人一票，value: 1赞 -1踩）
type ThreadVote struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	ThreadID  uint      `gorm:"uniqueIndex:uk_tv;comment:帖子ID" json:"thread_id"`
	UserID    uint      `gorm:"uniqueIndex:uk_tv;comment:用户ID" json:"user_id"`
	Value     int       `gorm:"comment:值" json:"value"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (ThreadVote) TableName() string { return "thread_votes" }

// 回复点赞（一人一票，仅赞）
type ReplyVote struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	ReplyID   uint      `gorm:"uniqueIndex:uk_rv;comment:回复ID" json:"reply_id"`
	UserID    uint      `gorm:"uniqueIndex:uk_rv;comment:用户ID" json:"user_id"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (ReplyVote) TableName() string { return "reply_votes" }

// 帖子打赏记录（金币从打赏人转给楼主）
type ThreadGift struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	ThreadID  uint      `gorm:"index;comment:帖子ID" json:"thread_id"`
	SenderID  uint      `gorm:"index;comment:发送者ID" json:"sender_id"`
	Coins     int       `gorm:"comment:金币" json:"coins"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	Sender    *User     `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
}

func (ThreadGift) TableName() string { return "thread_gifts" }

// 帖子送花记录（花从送花人花篮扣除）
type ThreadFlower struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	ThreadID  uint      `gorm:"index;comment:帖子ID" json:"thread_id"`
	SenderID  uint      `gorm:"index;comment:发送者ID" json:"sender_id"`
	Flower    string    `gorm:"type:varchar(20);comment:鲜花" json:"flower"`
	Count     int       `gorm:"comment:数量" json:"count"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	Sender    *User     `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
}

func (ThreadFlower) TableName() string { return "thread_flowers" }

// 公告/广播
type Announcement struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	Type      string    `gorm:"type:varchar(10);comment:notice公告 broadcast广播 activity活动" json:"type"` // notice公告 broadcast广播 activity活动
	Title     string    `gorm:"type:varchar(100);comment:标题" json:"title"`
	Content   string    `gorm:"type:text;comment:内容" json:"content"`
	Status    int       `gorm:"default:1;comment:状态" json:"status"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}
