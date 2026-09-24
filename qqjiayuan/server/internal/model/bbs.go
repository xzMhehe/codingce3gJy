package model

import "time"

// Board 板块：parent_id=0 为分区（如公共论坛），category_id 用于分区下的分类分组，否则为子板块
// 参考诺哈三代 wap_bbs 的三级结构：分区(page) → 分类(category) → 版块(board)
type Board struct {
	ID          uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	ParentID    uint      `gorm:"index;default:0;comment:0=分区，非0=所属分区" json:"parent_id"`  // 0=分区，非0=所属分区
	CategoryID  uint      `gorm:"index;default:0;comment:所属分类（0=未分类）" json:"category_id"` // 所属分类（0=未分类）
	Name        string    `gorm:"type:varchar(30);comment:名称" json:"name"`
	Description string    `gorm:"type:varchar(500);comment:描述（简介）" json:"description"`                              // 简介
	Notice      string    `gorm:"type:text;comment:版块公告" json:"notice"`                                             // 版块公告
	Tags        string    `gorm:"type:varchar(100);comment:标签（标签）" json:"tags"`                                     // 标签
	ModeratorID uint      `gorm:"default:0;comment:版主用户ID（0=无版主）" json:"moderator_id"`                              // 版主用户ID（0=无版主）
	MembersOnly int       `gorm:"default:0;comment:1=会员制版块，仅成员可发帖" json:"members_only"`                             // 1=会员制版块，仅成员可发帖
	CityCode    string    `gorm:"type:varchar(10);default:'';comment:区号（同城城市，参考诺哈 wap_bbs.ccode）" json:"city_code"` // 区号（同城城市，参考诺哈 wap_bbs.ccode）
	CreatorID   uint      `gorm:"default:0;comment:创建人（同城城市，参考诺哈 wap_bbs.uid）" json:"creator_id"`                   // 创建人（同城城市，参考诺哈 wap_bbs.uid）
	Click       int       `gorm:"default:0;comment:人气（同城城市，参考诺哈 wap_bbs.click）" json:"click"`                       // 人气（同城城市，参考诺哈 wap_bbs.click）
	Sort        int       `gorm:"default:0;comment:排序值" json:"sort"`
	Status      int       `gorm:"default:1;comment:1显示 0隐藏" json:"status"` // 1显示 0隐藏
	ThreadCount int       `gorm:"default:0;comment:帖子数量" json:"thread_count"`
	CreatedAt   time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt   time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

// BoardCategory 分区下的分类分组（参考诺哈 wap_bbs_category）
type BoardCategory struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	ParentID  uint      `gorm:"index;default:0;comment:所属分区ID" json:"parent_id"` // 所属分区ID
	Name      string    `gorm:"type:varchar(30);comment:名称" json:"name"`
	Sort      int       `gorm:"default:0;comment:排序值" json:"sort"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (BoardCategory) TableName() string { return "board_categories" }

// BoardMember 会员制版块成员
type BoardMember struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	BoardID   uint      `gorm:"index;uniqueIndex:uk_bm;comment:版块ID" json:"board_id"`
	UserID    uint      `gorm:"index;uniqueIndex:uk_bm;comment:用户ID" json:"user_id"`
	Status    int       `gorm:"default:1;comment:状态" json:"status"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (BoardMember) TableName() string { return "board_members" }

// Thread 帖子
// 参考诺哈三代 wap_topic：head头条 / apex置顶 / lock锁定 / fine精华 / recom推荐 / notice公告 / dtype类型 / 审核状态
type Thread struct {
	ID           uint       `gorm:"primaryKey;comment:主键ID" json:"id"`
	BoardID      uint       `gorm:"index;comment:版块ID" json:"board_id"`
	UserID       uint       `gorm:"index;comment:用户ID" json:"user_id"`
	Title        string     `gorm:"type:varchar(100);comment:标题" json:"title"`
	Content      string     `gorm:"type:text;comment:内容" json:"content"`
	IsTop        int        `gorm:"default:0;comment:是否置顶" json:"is_top"`                                 // 置顶
	IsFine       int        `gorm:"default:0;comment:是否精华" json:"is_fine"`                                // 精华
	IsHead       int        `gorm:"default:0;comment:是否头条" json:"is_head"`                                // 头条
	IsLock       int        `gorm:"default:0;comment:是否锁定" json:"is_lock"`                                // 锁定
	IsRecom      int        `gorm:"default:0;comment:是否推荐" json:"is_recom"`                               // 推荐
	IsNotice     int        `gorm:"default:0;comment:是否公告帖" json:"is_notice"`                             // 公告帖
	IsActive     int        `gorm:"default:0;comment:活动帖（参考诺哈 wap_topic.active，活动专区收录）" json:"is_active"` // 活动帖（参考诺哈 wap_topic.active，活动专区收录）
	Type         int        `gorm:"default:0;comment:0普通 1回帖奖励 2踩楼 3投票" json:"type"`                      // 0普通 1回帖奖励 2踩楼 3投票
	AuditStatus  int        `gorm:"not null;comment:1已发布 0待审核 2审核不通过" json:"audit_status"`                // 1已发布 0待审核 2审核不通过
	ViewCount    int        `gorm:"default:0;comment:浏览数量" json:"view_count"`
	ReplyCount   int        `gorm:"default:0;comment:回复数量" json:"reply_count"`
	LikeCount    int        `gorm:"default:0;comment:点赞数量" json:"like_count"`
	DislikeCount int        `gorm:"default:0;comment:踩数量" json:"dislike_count"`
	ShareCount   int        `gorm:"default:0;comment:分享数量" json:"share_count"`
	GiftTotal    int        `gorm:"default:0;comment:礼物总计" json:"gift_total"`
	FlowerCount  int        `gorm:"default:0;comment:鲜花数量" json:"flower_count"`
	Status       int        `gorm:"default:1;comment:1正常 0删除" json:"status"` // 1正常 0删除
	LastReplyAt  *time.Time `gorm:"comment:最后回复时间" json:"last_reply_at"`
	CreatedAt    time.Time  `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"comment:更新时间" json:"updated_at"`
	User         *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Board        *Board     `gorm:"foreignKey:BoardID" json:"board,omitempty"`
}

// Reply 回复（盖楼），Floor 为楼层数，1楼是楼主；ParentReplyID 支持引用/楼层回复
type Reply struct {
	ID            uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	ThreadID      uint      `gorm:"index;comment:帖子ID" json:"thread_id"`
	UserID        uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Content       string    `gorm:"type:text;comment:内容" json:"content"`
	Floor         int       `gorm:"comment:楼层" json:"floor"`
	ParentReplyID uint      `gorm:"default:0;comment:被引用的回复ID（0=直接回复楼主）" json:"parent_reply_id"` // 被引用的回复ID（0=直接回复楼主）
	LikeCount     int       `gorm:"default:0;comment:点赞数量" json:"like_count"`
	Status        int       `gorm:"default:1;comment:1正常 0删除" json:"status"` // 1正常 0删除
	CreatedAt     time.Time `gorm:"comment:创建时间" json:"created_at"`
	User          *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Thread        *Thread   `gorm:"foreignKey:ThreadID" json:"thread,omitempty"`
}

// StickyReply 置顶回复（参考诺哈 wap_topic_reply_apex，一贴至多一条）
type StickyReply struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	ThreadID  uint      `gorm:"uniqueIndex;comment:帖子ID" json:"thread_id"`
	UserID    uint      `gorm:"comment:用户ID" json:"user_id"`
	Content   string    `gorm:"type:text;comment:内容" json:"content"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (StickyReply) TableName() string { return "sticky_replies" }

// ThreadPoll 帖子投票（type=3）
type ThreadPoll struct {
	ID        uint               `gorm:"primaryKey;comment:主键ID" json:"id"`
	ThreadID  uint               `gorm:"index;comment:帖子ID" json:"thread_id"`
	Question  string             `gorm:"type:varchar(200);comment:问题" json:"question"`
	Multiple  int                `gorm:"default:0;comment:1多选" json:"multiple"` // 1多选
	CreatedAt time.Time          `gorm:"comment:创建时间" json:"created_at"`
	Options   []ThreadPollOption `gorm:"foreignKey:PollID" json:"options,omitempty"`
}

func (ThreadPoll) TableName() string { return "thread_polls" }

// ThreadPollOption 投票选项
type ThreadPollOption struct {
	ID     uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	PollID uint   `gorm:"index;comment:投票ID" json:"poll_id"`
	Name   string `gorm:"type:varchar(100);comment:名称" json:"name"`
	Votes  int    `gorm:"default:0;comment:票数" json:"votes"`
}

func (ThreadPollOption) TableName() string { return "thread_poll_options" }

// ThreadPollVote 投票记录（一人一次，多选时一条一选项）
type ThreadPollVote struct {
	ID       uint `gorm:"primaryKey;comment:主键ID" json:"id"`
	PollID   uint `gorm:"index;uniqueIndex:uk_pv;comment:投票ID" json:"poll_id"`
	UserID   uint `gorm:"uniqueIndex:uk_pv;comment:用户ID" json:"user_id"`
	OptionID uint `gorm:"index;comment:选项ID" json:"option_id"`
	ThreadID uint `gorm:"index;comment:帖子ID" json:"thread_id"`
}

func (ThreadPollVote) TableName() string { return "thread_poll_votes" }

// ThreadReward 回帖奖励配置（type=1，参考诺哈 wap_topic_sign）
type ThreadReward struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	ThreadID  uint      `gorm:"uniqueIndex;comment:帖子ID" json:"thread_id"`
	Coins     int       `gorm:"default:0;comment:每次回帖奖励金币" json:"coins"`     // 每次回帖奖励金币
	Exp       int       `gorm:"default:0;comment:每次回帖奖励经验" json:"exp"`       // 每次回帖奖励经验
	Limit     int       `gorm:"default:0;comment:最大奖励次数（0=不限）" json:"limit"` // 最大奖励次数（0=不限）
	Used      int       `gorm:"default:0;comment:已发放次数" json:"used"`         // 已发放次数
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (ThreadReward) TableName() string { return "thread_rewards" }

// ThreadRewardLog 回帖奖励发放日志
type ThreadRewardLog struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	ThreadID  uint      `gorm:"index;comment:帖子ID" json:"thread_id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	ReplyID   uint      `gorm:"comment:回复ID" json:"reply_id"`
	Coins     int       `gorm:"comment:金币" json:"coins"`
	Exp       int       `gorm:"comment:经验" json:"exp"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (ThreadRewardLog) TableName() string { return "thread_reward_logs" }

// ThreadFloor 踩楼奖励配置（type=2，参考诺哈 wap_topic_floor）
type ThreadFloor struct {
	ID       uint `gorm:"primaryKey;comment:主键ID" json:"id"`
	ThreadID uint `gorm:"index;comment:帖子ID" json:"thread_id"`
	Floor    int  `gorm:"default:0;comment:踩中楼层（0=系统随机）" json:"floor"` // 踩中楼层（0=系统随机）
	Coins    int  `gorm:"default:0;comment:奖励金币" json:"coins"`         // 奖励金币
	Exp      int  `gorm:"default:0;comment:奖励经验" json:"exp"`           // 奖励经验
	Status   int  `gorm:"default:0;comment:0待踩中 1已踩中" json:"status"`   // 0待踩中 1已踩中
	UserID   uint `gorm:"default:0;comment:踩中用户" json:"user_id"`       // 踩中用户
}

func (ThreadFloor) TableName() string { return "thread_floors" }

// ThreadAttachment 附件（图片/文件）
type ThreadAttachment struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	ThreadID  uint      `gorm:"index;comment:帖子ID" json:"thread_id"`
	UserID    uint      `gorm:"comment:用户ID" json:"user_id"`
	Type      string    `gorm:"type:varchar(10);comment:image / file" json:"type"` // image / file
	Name      string    `gorm:"type:varchar(255);comment:名称" json:"name"`
	Path      string    `gorm:"type:varchar(500);comment:路径" json:"path"`
	Size      int64     `gorm:"default:0;comment:大小" json:"size"`
	Price     int       `gorm:"default:0;comment:付费下载价格（金币）" json:"price"` // 付费下载价格（金币）
	Downloads int       `gorm:"default:0;comment:下载数" json:"downloads"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (ThreadAttachment) TableName() string { return "thread_attachments" }

// WordFilter 敏感词
type WordFilter struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	Word      string    `gorm:"type:varchar(50);index;comment:词语" json:"word"`
	Replace   string    `gorm:"type:varchar(50);default:*;comment:替换" json:"replace"`
	Type      int       `gorm:"default:1;comment:1替换 2审核" json:"type"` // 1替换 2审核
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (WordFilter) TableName() string { return "word_filters" }
