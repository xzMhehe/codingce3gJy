package model

import "time"

// Board 板块：parent_id=0 为分区（如公共论坛），category_id 用于分区下的分类分组，否则为子板块
// 参考诺哈三代 wap_bbs 的三级结构：分区(page) → 分类(category) → 版块(board)
type Board struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ParentID    uint      `gorm:"index;default:0" json:"parent_id"`       // 0=分区，非0=所属分区
	CategoryID  uint      `gorm:"index;default:0" json:"category_id"`     // 所属分类（0=未分类）
	Name        string    `gorm:"type:varchar(30)" json:"name"`
	Description string    `gorm:"type:varchar(500)" json:"description"` // 简介
	Notice      string    `gorm:"type:text" json:"notice"`             // 版块公告
	Tags        string    `gorm:"type:varchar(100)" json:"tags"`       // 标签
	ModeratorID uint      `gorm:"default:0" json:"moderator_id"`       // 版主用户ID（0=无版主）
	MembersOnly int       `gorm:"default:0" json:"members_only"`       // 1=会员制版块，仅成员可发帖
	CityCode    string    `gorm:"type:varchar(10);default:''" json:"city_code"` // 区号（同城城市，参考诺哈 wap_bbs.ccode）
	CreatorID   uint      `gorm:"default:0" json:"creator_id"`         // 创建人（同城城市，参考诺哈 wap_bbs.uid）
	Click       int       `gorm:"default:0" json:"click"`              // 人气（同城城市，参考诺哈 wap_bbs.click）
	Sort        int       `gorm:"default:0" json:"sort"`
	Status      int       `gorm:"default:1" json:"status"` // 1显示 0隐藏
	ThreadCount int       `gorm:"default:0" json:"thread_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BoardCategory 分区下的分类分组（参考诺哈 wap_bbs_category）
type BoardCategory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ParentID  uint      `gorm:"index;default:0" json:"parent_id"` // 所属分区ID
	Name      string    `gorm:"type:varchar(30)" json:"name"`
	Sort      int       `gorm:"default:0" json:"sort"`
	CreatedAt time.Time `json:"created_at"`
}

func (BoardCategory) TableName() string { return "board_categories" }

// BoardMember 会员制版块成员
type BoardMember struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	BoardID   uint      `gorm:"index;uniqueIndex:uk_bm" json:"board_id"`
	UserID    uint      `gorm:"index;uniqueIndex:uk_bm" json:"user_id"`
	Status    int       `gorm:"default:1" json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (BoardMember) TableName() string { return "board_members" }

// Thread 帖子
// 参考诺哈三代 wap_topic：head头条 / apex置顶 / lock锁定 / fine精华 / recom推荐 / notice公告 / dtype类型 / 审核状态
type Thread struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	BoardID      uint       `gorm:"index" json:"board_id"`
	UserID       uint       `gorm:"index" json:"user_id"`
	Title        string     `gorm:"type:varchar(100)" json:"title"`
	Content      string     `gorm:"type:text" json:"content"`
	IsTop        int        `gorm:"default:0" json:"is_top"`        // 置顶
	IsFine       int        `gorm:"default:0" json:"is_fine"`       // 精华
	IsHead       int        `gorm:"default:0" json:"is_head"`       // 头条
	IsLock       int        `gorm:"default:0" json:"is_lock"`       // 锁定
	IsRecom      int        `gorm:"default:0" json:"is_recom"`      // 推荐
	IsNotice     int        `gorm:"default:0" json:"is_notice"`     // 公告帖
	IsActive     int        `gorm:"default:0" json:"is_active"`     // 活动帖（参考诺哈 wap_topic.active，活动专区收录）
	Type         int        `gorm:"default:0" json:"type"`          // 0普通 1回帖奖励 2踩楼 3投票
	AuditStatus  int        `gorm:"not null" json:"audit_status"`   // 1已发布 0待审核 2审核不通过
	ViewCount    int        `gorm:"default:0" json:"view_count"`
	ReplyCount   int        `gorm:"default:0" json:"reply_count"`
	LikeCount    int        `gorm:"default:0" json:"like_count"`
	DislikeCount int        `gorm:"default:0" json:"dislike_count"`
	ShareCount   int        `gorm:"default:0" json:"share_count"`
	GiftTotal    int        `gorm:"default:0" json:"gift_total"`
	FlowerCount  int        `gorm:"default:0" json:"flower_count"`
	Status       int        `gorm:"default:1" json:"status"` // 1正常 0删除
	LastReplyAt  *time.Time `json:"last_reply_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	User         *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Board        *Board     `gorm:"foreignKey:BoardID" json:"board,omitempty"`
}

// Reply 回复（盖楼），Floor 为楼层数，1楼是楼主；ParentReplyID 支持引用/楼层回复
type Reply struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ThreadID      uint      `gorm:"index" json:"thread_id"`
	UserID        uint      `gorm:"index" json:"user_id"`
	Content       string    `gorm:"type:text" json:"content"`
	Floor         int       `json:"floor"`
	ParentReplyID uint      `gorm:"default:0" json:"parent_reply_id"` // 被引用的回复ID（0=直接回复楼主）
	LikeCount     int       `gorm:"default:0" json:"like_count"`
	Status        int       `gorm:"default:1" json:"status"` // 1正常 0删除
	CreatedAt     time.Time `json:"created_at"`
	User          *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Thread        *Thread   `gorm:"foreignKey:ThreadID" json:"thread,omitempty"`
}

// StickyReply 置顶回复（参考诺哈 wap_topic_reply_apex，一贴至多一条）
type StickyReply struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ThreadID  uint      `gorm:"uniqueIndex" json:"thread_id"`
	UserID    uint      `json:"user_id"`
	Content   string    `gorm:"type:text" json:"content"`
	CreatedAt time.Time `json:"created_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (StickyReply) TableName() string { return "sticky_replies" }

// ThreadPoll 帖子投票（type=3）
type ThreadPoll struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ThreadID  uint      `gorm:"index" json:"thread_id"`
	Question  string    `gorm:"type:varchar(200)" json:"question"`
	Multiple  int       `gorm:"default:0" json:"multiple"` // 1多选
	CreatedAt time.Time `json:"created_at"`
	Options   []ThreadPollOption `gorm:"foreignKey:PollID" json:"options,omitempty"`
}

func (ThreadPoll) TableName() string { return "thread_polls" }

// ThreadPollOption 投票选项
type ThreadPollOption struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	PollID uint   `gorm:"index" json:"poll_id"`
	Name   string `gorm:"type:varchar(100)" json:"name"`
	Votes  int    `gorm:"default:0" json:"votes"`
}

func (ThreadPollOption) TableName() string { return "thread_poll_options" }

// ThreadPollVote 投票记录（一人一次，多选时一条一选项）
type ThreadPollVote struct {
	ID       uint `gorm:"primaryKey" json:"id"`
	PollID   uint `gorm:"index;uniqueIndex:uk_pv" json:"poll_id"`
	UserID   uint `gorm:"uniqueIndex:uk_pv" json:"user_id"`
	OptionID uint `gorm:"index" json:"option_id"`
	ThreadID uint `gorm:"index" json:"thread_id"`
}

func (ThreadPollVote) TableName() string { return "thread_poll_votes" }

// ThreadReward 回帖奖励配置（type=1，参考诺哈 wap_topic_sign）
type ThreadReward struct {
	ID       uint      `gorm:"primaryKey" json:"id"`
	ThreadID uint      `gorm:"uniqueIndex" json:"thread_id"`
	Coins    int       `gorm:"default:0" json:"coins"`   // 每次回帖奖励金币
	Exp      int       `gorm:"default:0" json:"exp"`     // 每次回帖奖励经验
	Limit    int       `gorm:"default:0" json:"limit"`   // 最大奖励次数（0=不限）
	Used     int       `gorm:"default:0" json:"used"`    // 已发放次数
	CreatedAt time.Time `json:"created_at"`
}

func (ThreadReward) TableName() string { return "thread_rewards" }

// ThreadRewardLog 回帖奖励发放日志
type ThreadRewardLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ThreadID  uint      `gorm:"index" json:"thread_id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	ReplyID   uint      `json:"reply_id"`
	Coins     int       `json:"coins"`
	Exp       int       `json:"exp"`
	CreatedAt time.Time `json:"created_at"`
}

func (ThreadRewardLog) TableName() string { return "thread_reward_logs" }

// ThreadFloor 踩楼奖励配置（type=2，参考诺哈 wap_topic_floor）
type ThreadFloor struct {
	ID       uint `gorm:"primaryKey" json:"id"`
	ThreadID uint `gorm:"index" json:"thread_id"`
	Floor    int  `gorm:"default:0" json:"floor"`   // 踩中楼层（0=系统随机）
	Coins    int  `gorm:"default:0" json:"coins"`   // 奖励金币
	Exp      int  `gorm:"default:0" json:"exp"`     // 奖励经验
	Status   int  `gorm:"default:0" json:"status"`  // 0待踩中 1已踩中
	UserID   uint `gorm:"default:0" json:"user_id"` // 踩中用户
}

func (ThreadFloor) TableName() string { return "thread_floors" }

// ThreadAttachment 附件（图片/文件）
type ThreadAttachment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ThreadID  uint      `gorm:"index" json:"thread_id"`
	UserID    uint      `json:"user_id"`
	Type      string    `gorm:"type:varchar(10)" json:"type"` // image / file
	Name      string    `gorm:"type:varchar(255)" json:"name"`
	Path      string    `gorm:"type:varchar(500)" json:"path"`
	Size      int64     `gorm:"default:0" json:"size"`
	Price     int       `gorm:"default:0" json:"price"` // 付费下载价格（金币）
	Downloads int       `gorm:"default:0" json:"downloads"`
	CreatedAt time.Time `json:"created_at"`
}

func (ThreadAttachment) TableName() string { return "thread_attachments" }

// WordFilter 敏感词
type WordFilter struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Word      string    `gorm:"type:varchar(50);index" json:"word"`
	Replace   string    `gorm:"type:varchar(50);default:*" json:"replace"`
	Type      int       `gorm:"default:1" json:"type"` // 1替换 2审核
	CreatedAt time.Time `json:"created_at"`
}

func (WordFilter) TableName() string { return "word_filters" }