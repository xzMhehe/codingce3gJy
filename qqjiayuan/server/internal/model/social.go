package model

import "time"

// 每日签到
type SignIn struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index:idx_user_date,unique;comment:用户ID" json:"user_id"`
	SignDate  string    `gorm:"type:varchar(10);index:idx_user_date,unique;comment:2006-01-02" json:"sign_date"` // 2006-01-02
	Consec    int       `gorm:"comment:连续签到天数" json:"consec"`                                                    // 连续签到天数
	Reward    int       `gorm:"comment:金币奖励" json:"reward"`                                                      // 金币奖励
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

// 好友关系（对齐诺哈 wap_friend）：单向一条记录，双向好友即存在「我→TA」「TA→我」两行。
// status: 1为已确认好友（对方接受后另一侧也为1）；0为待对方处理的申请（对侧不存在）。
// 通过后，双方各插入一条 status=1 的记录，实现各自的备注/分组/亲密度。
type Friendship struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index:idx_uf;comment:uid 归属者（我）" json:"user_id"`           // uid 归属者（我）
	FriendID  uint      `gorm:"index:idx_uf;comment:oid 好友（TA）" json:"friend_id"`         // oid 好友（TA）
	Status    int       `gorm:"default:1;comment:1好友 0待对方处理（对侧建立后转1）" json:"status"`      // 1好友 0待对方处理（对侧建立后转1）
	Remark    string    `gorm:"type:varchar(30);comment:name 备注（空=显示对方昵称）" json:"remark"` // name 备注（空=显示对方昵称）
	GroupID   uint      `gorm:"default:0;comment:group 分组（0=未分组）" json:"group_id"`        // group 分组（0=未分组）
	Degree    int       `gorm:"default:0;comment:亲密度" json:"degree"`                      // 亲密度
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (Friendship) TableName() string { return "friendships" }

// 好友申请（对齐诺哈 wap_friend_apply）：uid=接收方（被加的人），friend_id=申请方，remark=验证信息
type FriendApply struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index:idx_ua,unique;comment:接收方" json:"user_id"`   // 接收方
	FriendID  uint      `gorm:"index:idx_ua,unique;comment:申请方" json:"friend_id"` // 申请方
	Remark    string    `gorm:"type:varchar(100);comment:备注" json:"remark"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (FriendApply) TableName() string { return "friend_applies" }

// 黑名单（对齐诺哈 wap_friend_black）：加入后双方不再互为好友，且对方消息被屏蔽
type FriendBlack struct {
	ID       uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID   uint      `gorm:"index:idx_ub,unique;index;comment:uid 归属者（我）" json:"user_id"`        // uid 归属者（我）
	FriendID uint      `gorm:"index:idx_ub,unique;comment:oid 被拉黑者（TA）" json:"friend_id"`          // oid 被拉黑者（TA）
	Name     string    `gorm:"type:varchar(30);comment:备注名（诺哈 wap_friend_black.name）" json:"name"` // 备注名（诺哈 wap_friend_black.name）
	AddTime  time.Time `gorm:"comment:添加时间" json:"add_time"`
	EndTime  time.Time `gorm:"comment:1年有效（诺哈 wap_friend_black.endtime）" json:"end_time"` // 1年有效（诺哈 wap_friend_black.endtime）
}

func (FriendBlack) TableName() string { return "friend_blacks" }

// 私信
type PrivateMessage struct {
	ID         uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	SenderID   uint      `gorm:"index;comment:发送者ID" json:"sender_id"`
	ReceiverID uint      `gorm:"index;comment:接收者ID" json:"receiver_id"`
	Content    string    `gorm:"type:varchar(500);comment:内容" json:"content"`
	IsRead     int       `gorm:"default:0;comment:是否已读" json:"is_read"`
	CreatedAt  time.Time `gorm:"comment:创建时间" json:"created_at"`
	Sender     *User     `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
}

// 聊天室消息（family_id=0 且 board_id=0 为公共聊天；>0 家族聊室；board_id>0 为同城老乡聊天室，轮询拉取）
type ChatMessage struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	FamilyID  uint      `gorm:"index;default:0;comment:家族ID" json:"family_id"`
	BoardID   uint      `gorm:"index;default:0;comment:同城城市板块ID（老乡聊天室）" json:"board_id"` // 同城城市板块ID（老乡聊天室）
	Content   string    `gorm:"type:varchar(500);comment:内容" json:"content"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// 系统通知：reply回复/好友friend/系统system
type Notification struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Type      string    `gorm:"type:varchar(20);comment:类型" json:"type"`
	Title     string    `gorm:"type:varchar(100);comment:标题" json:"title"`
	Content   string    `gorm:"type:varchar(500);comment:内容" json:"content"`
	RefID     uint      `gorm:"comment:关联ID" json:"ref_id"`
	IsRead    int       `gorm:"default:0;comment:是否已读" json:"is_read"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

// 举报（帖子/回复），后台可处理：ignore忽略 / delete删内容 / ban封人
type Report struct {
	ID         uint       `gorm:"primaryKey;comment:主键ID" json:"id"`
	ReporterID uint       `gorm:"index;comment:ReporterID" json:"reporter_id"`
	TargetType string     `gorm:"type:varchar(10);index;comment:thread / reply" json:"target_type"` // thread / reply
	TargetID   uint       `gorm:"index;comment:目标ID" json:"target_id"`
	Reason     string     `gorm:"type:varchar(200);comment:原因" json:"reason"`
	Status     int        `gorm:"default:0;comment:0待处理 1已忽略 2已删除内容 3已封禁发布者" json:"status"` // 0待处理 1已忽略 2已删除内容 3已封禁发布者
	Result     string     `gorm:"type:varchar(200);comment:结果" json:"result"`
	HandlerID  uint       `gorm:"comment:HandlerID" json:"handler_id"`
	HandledAt  *time.Time `gorm:"comment:已处理时间" json:"handled_at"`
	CreatedAt  time.Time  `gorm:"comment:创建时间" json:"created_at"`
	Reporter   *User      `gorm:"foreignKey:ReporterID" json:"reporter,omitempty"`
}

func (Report) TableName() string { return "reports" }
