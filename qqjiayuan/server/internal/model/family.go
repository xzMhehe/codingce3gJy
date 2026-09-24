package model

import "time"

// 家族
type Family struct {
	ID           uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name         string    `gorm:"type:varchar(30);comment:名称" json:"name"`
	Slogan       string    `gorm:"type:varchar(60);comment:标语" json:"slogan"`
	Description  string    `gorm:"type:varchar(200);comment:描述" json:"description"`
	Announcement string    `gorm:"type:varchar(500);comment:Announcement" json:"announcement"`
	Category     string    `gorm:"type:varchar(20);comment:家族类别" json:"category"`    // 家族类别
	OwnerID      uint      `gorm:"index;comment:拥有者ID（族长）" json:"owner_id"`          // 族长
	TreeLevel    int       `gorm:"default:1;comment:守护树等级" json:"tree_level"`        // 守护树等级
	TreeExp      int       `gorm:"default:0;comment:守护树成长值" json:"tree_exp"`         // 守护树成长值
	BattleScore  int       `gorm:"default:0;comment:家族乐斗积分" json:"battle_score"`     // 家族乐斗积分
	WarPoints    int       `gorm:"default:0;comment:族斗荣誉点" json:"war_points"`        // 族斗荣誉点
	IsFeature    int       `gorm:"default:0;comment:是否特色家族 1是 0否" json:"is_feature"` // 特色家族 1是 0否
	Status       int       `gorm:"default:1;comment:1正常 2待审核 0解散" json:"status"`     // 1正常 2待审核 0解散
	CreatedAt    time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt    time.Time `gorm:"comment:更新时间" json:"updated_at"`
	Owner        *User     `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Members      int64     `gorm:"-" json:"members"` // 成员数（运行时统计）
	Role         string    `gorm:"-" json:"role"`    // 当前用户在该家族中的角色（运行时填充）
}

func (Family) TableName() string { return "families" }

// 家族成员
type FamilyMember struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	FamilyID  uint      `gorm:"index;comment:家族ID" json:"family_id"`
	UserID    uint      `gorm:"uniqueIndex:uk_fam_user;comment:用户ID" json:"user_id"`
	Role      string    `gorm:"type:varchar(10);default:member;comment:owner/admin/member" json:"role"` // owner/admin/member
	Exp       int       `gorm:"default:0;comment:家族贡献" json:"exp"`                                      // 家族贡献
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (FamilyMember) TableName() string { return "family_members" }

// 家族每日互动（type=sign 签到 / type=tree 守护树，各自每日一次）
type FamilySignIn struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	FamilyID  uint      `gorm:"index;comment:家族ID" json:"family_id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	SignDate  string    `gorm:"type:varchar(10);comment:签到日期" json:"sign_date"`
	Type      string    `gorm:"type:varchar(10);default:sign;comment:sign/tree" json:"type"` // sign/tree
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (FamilySignIn) TableName() string { return "family_sign_ins" }

// 收藏家族（对齐诺哈 wap_bbs_favor type=4）
type FamilyFavorite struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:uk_ff;comment:用户ID" json:"user_id"`
	FamilyID  uint      `gorm:"uniqueIndex:uk_ff;comment:家族ID" json:"family_id"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (FamilyFavorite) TableName() string { return "family_favorites" }

// 家族访客（每人每日一条，统计今日访客数，对齐诺哈「访客：今天N人」）
type FamilyVisit struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	FamilyID  uint      `gorm:"uniqueIndex:uk_fv;comment:家族ID" json:"family_id"`
	UserID    uint      `gorm:"uniqueIndex:uk_fv;comment:用户ID" json:"user_id"`
	Day       string    `gorm:"type:varchar(10);uniqueIndex:uk_fv;comment:天数" json:"day"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (FamilyVisit) TableName() string { return "family_visits" }

// 家族区动态（加入/签到/守护/乐斗等）
type FamilyActivity struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	FamilyID  uint      `gorm:"index;comment:家族ID" json:"family_id"`
	UserID    uint      `gorm:"comment:用户ID" json:"user_id"`
	Content   string    `gorm:"type:varchar(160);comment:内容" json:"content"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (FamilyActivity) TableName() string { return "family_activities" }

// 家族乐斗个人数据（战斗力/功勋值/体力值，乐斗与族斗共用功勋值）
type FamilyLdUser struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"uniqueIndex;comment:用户ID" json:"user_id"`
	Fight     int       `gorm:"default:10;comment:战斗力" json:"fight"`                       // 战斗力
	Merit     int       `gorm:"default:0;comment:功勋值" json:"merit"`                        // 功勋值
	Stamina   int       `gorm:"default:10;comment:体力值" json:"stamina"`                     // 体力值
	LdDate    string    `gorm:"type:varchar(10);default:'';comment:今日乐斗日期" json:"ld_date"` // 今日乐斗日期
	LdCount   int       `gorm:"default:0;comment:今日已斗次数" json:"ld_count"`                  // 今日已斗次数
	WinCount  int       `gorm:"default:0;comment:胜利数量" json:"win_count"`
	LoseCount int       `gorm:"default:0;comment:失败数量" json:"lose_count"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (FamilyLdUser) TableName() string { return "family_ld_users" }

// 乐斗动态（家族乐斗页展示）
type FamilyLdLog struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	FamilyID  uint      `gorm:"index;comment:家族ID" json:"family_id"`
	UserID    uint      `gorm:"comment:用户ID" json:"user_id"`
	Content   string    `gorm:"type:varchar(160);comment:内容" json:"content"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (FamilyLdLog) TableName() string { return "family_ld_logs" }

// 族斗今日战局（每家族每日一条，保留历史战况）
type FamilyWarBattle struct {
	ID         uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	WarDate    string    `gorm:"type:varchar(10);uniqueIndex:uk_war;comment:战争日期" json:"war_date"`
	FamilyID   uint      `gorm:"uniqueIndex:uk_war;comment:家族ID" json:"family_id"`
	EnemyID    uint      `gorm:"comment:敌方ID" json:"enemy_id"`
	EnemyName  string    `gorm:"type:varchar(30);comment:敌方名称" json:"enemy_name"`
	MyScore    int       `gorm:"default:0;comment:我的积分" json:"my_score"`
	EnemyScore int       `gorm:"default:0;comment:敌方积分" json:"enemy_score"`
	CreatedAt  time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (FamilyWarBattle) TableName() string { return "family_war_battles" }

// 族斗生命力（每人每日 5 点）
type FamilyWarLife struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"uniqueIndex;comment:用户ID" json:"user_id"`
	Life      int       `gorm:"default:5;comment:Life" json:"life"`
	WarDate   string    `gorm:"type:varchar(10);comment:战争日期" json:"war_date"`
	Bought    int       `gorm:"default:0;comment:今日购买次数" json:"bought"` // 今日购买次数
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (FamilyWarLife) TableName() string { return "family_war_lives" }

// 族斗喊话（私聊/对话，每次发言扣 1000 G币）
type FamilyWarChat struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	FamilyID  uint      `gorm:"index;comment:家族ID" json:"family_id"`
	UserID    uint      `gorm:"comment:用户ID" json:"user_id"`
	Type      string    `gorm:"type:varchar(10);default:chat;comment:chat 家族对话 / private 私聊" json:"type"` // chat 家族对话 / private 私聊
	Content   string    `gorm:"type:varchar(120);comment:内容" json:"content"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (FamilyWarChat) TableName() string { return "family_war_chats" }
