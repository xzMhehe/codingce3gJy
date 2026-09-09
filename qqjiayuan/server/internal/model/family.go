package model

import "time"

// 家族
type Family struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"type:varchar(30)" json:"name"`
	Slogan       string    `gorm:"type:varchar(60)" json:"slogan"`
	Description  string    `gorm:"type:varchar(200)" json:"description"`
	Announcement string    `gorm:"type:varchar(500)" json:"announcement"`
	Category     string    `gorm:"type:varchar(20)" json:"category"` // 家族类别
	OwnerID      uint      `gorm:"index" json:"owner_id"`      // 族长
	TreeLevel    int       `gorm:"default:1" json:"tree_level"` // 守护树等级
	TreeExp      int       `gorm:"default:0" json:"tree_exp"`   // 守护树成长值
	BattleScore  int       `gorm:"default:0" json:"battle_score"` // 家族乐斗积分
	IsFeature    int       `gorm:"default:0" json:"is_feature"`   // 特色家族 1是 0否
	Status       int       `gorm:"default:1" json:"status"`       // 1正常 2待审核 0解散
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Owner        *User     `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Members      int64     `gorm:"-" json:"members"` // 成员数（运行时统计）
	Role         string    `gorm:"-" json:"role"`    // 当前用户在该家族中的角色（运行时填充）
}

func (Family) TableName() string { return "families" }

// 家族成员
type FamilyMember struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	FamilyID  uint      `gorm:"index" json:"family_id"`
	UserID    uint      `gorm:"uniqueIndex:uk_fam_user" json:"user_id"`
	Role      string    `gorm:"type:varchar(10);default:member" json:"role"` // owner/admin/member
	Exp       int       `gorm:"default:0" json:"exp"`                        // 家族贡献
	CreatedAt time.Time `json:"created_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (FamilyMember) TableName() string { return "family_members" }

// 家族每日互动（type=sign 签到 / type=tree 守护树，各自每日一次）
type FamilySignIn struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	FamilyID  uint      `gorm:"index" json:"family_id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	SignDate  string    `gorm:"type:varchar(10)" json:"sign_date"`
	Type      string    `gorm:"type:varchar(10);default:sign" json:"type"` // sign/tree
	CreatedAt time.Time `json:"created_at"`
}

func (FamilySignIn) TableName() string { return "family_sign_ins" }

// 收藏家族（对齐诺哈 wap_bbs_favor type=4）
type FamilyFavorite struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:uk_ff" json:"user_id"`
	FamilyID  uint      `gorm:"uniqueIndex:uk_ff" json:"family_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (FamilyFavorite) TableName() string { return "family_favorites" }

// 家族区动态（加入/签到/守护/乐斗等）
type FamilyActivity struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	FamilyID  uint      `gorm:"index" json:"family_id"`
	UserID    uint      `json:"user_id"`
	Content   string    `gorm:"type:varchar(160)" json:"content"`
	CreatedAt time.Time `json:"created_at"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (FamilyActivity) TableName() string { return "family_activities" }
