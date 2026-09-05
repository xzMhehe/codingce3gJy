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

// 聊天室消息（family_id=0 为公共聊天，>0 为家族聊室，轮询拉取）
type ChatMessage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	FamilyID  uint      `gorm:"index;default:0" json:"family_id"`
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

// 举报（帖子/回复），后台可处理：ignore忽略 / delete删内容 / ban封人
type Report struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ReporterID  uint      `gorm:"index" json:"reporter_id"`
	TargetType  string    `gorm:"type:varchar(10);index" json:"target_type"` // thread / reply
	TargetID    uint      `gorm:"index" json:"target_id"`
	Reason      string    `gorm:"type:varchar(200)" json:"reason"`
	Status      int       `gorm:"default:0" json:"status"` // 0待处理 1已忽略 2已删除内容 3已封禁发布者
	Result      string    `gorm:"type:varchar(200)" json:"result"`
	HandlerID   uint      `json:"handler_id"`
	HandledAt   *time.Time `json:"handled_at"`
	CreatedAt   time.Time `json:"created_at"`
	Reporter    *User     `gorm:"foreignKey:ReporterID" json:"reporter,omitempty"`
}

func (Report) TableName() string { return "reports" }
