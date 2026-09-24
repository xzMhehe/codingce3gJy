package model

import (
	"time"

	"gorm.io/gorm"
)

// 用户表：家园号码即 ID（自增起始 10000），类似当年 QQ 号码
type User struct {
	ID             uint       `gorm:"primaryKey;comment:主键ID" json:"id"`
	Username       string     `gorm:"type:varchar(20);uniqueIndex;comment:家园号码（同 ID）" json:"username"` // 家园号码（同 ID）
	Nickname       string     `gorm:"type:varchar(20);uniqueIndex;comment:昵称" json:"nickname"`
	Password       string     `gorm:"type:varchar(100);comment:Password" json:"-"`
	Gender         int        `gorm:"default:1;comment:1小哥哥 2小姐姐" json:"gender"` // 1小哥哥 2小姐姐
	Age            int        `gorm:"default:0;comment:Age" json:"age"`
	BirthYear      int        `gorm:"default:0;comment:生日年" json:"birth_year"`
	BirthMonth     int        `gorm:"default:0;comment:生日月" json:"birth_month"`
	BirthDay       int        `gorm:"default:0;comment:生日天数" json:"birth_day"`
	BirthType      int        `gorm:"default:1;comment:生日类型 0阴历 1阳历（诺哈 wap_user.birth）" json:"birth_type"`                      // 生日类型 0阴历 1阳历（诺哈 wap_user.birth）
	Solar          string     `gorm:"type:varchar(20);comment:阳历生日（诺哈 wap_user.solar）" json:"solar"`                            // 阳历生日（诺哈 wap_user.solar）
	Lunar          string     `gorm:"type:varchar(20);comment:阴历生日（诺哈 wap_user.lunar）" json:"lunar"`                            // 阴历生日（诺哈 wap_user.lunar）
	Hours          int        `gorm:"default:0;comment:累计在线分钟数（诺哈 wap_user.hours，在线时长等级依据）" json:"hours"`                       // 累计在线分钟数（诺哈 wap_user.hours，在线时长等级依据）
	FriendPolicy   int        `gorm:"default:0;comment:加好友策略 0允许 1需要验证 2拒绝（诺哈 wap_user.friend）" json:"friend_policy"`           // 加好友策略 0允许 1需要验证 2拒绝（诺哈 wap_user.friend）
	Config         string     `gorm:"type:varchar(50);comment:个性设置 CSV：每页帖子数,页面字数,书城字数,每页字数（诺哈 wap_user.config）" json:"config"` // 个性设置 CSV：每页帖子数,页面字数,书城字数,每页字数（诺哈 wap_user.config）
	AddIP          string     `gorm:"type:varchar(45);comment:注册IP（诺哈 wap_user.addip）" json:"-"`                                // 注册IP（诺哈 wap_user.addip）
	LastIP         string     `gorm:"type:varchar(45);comment:最后登录IP（诺哈 wap_user.endip）" json:"-"`                              // 最后登录IP（诺哈 wap_user.endip）
	PayPass        string     `gorm:"type:varchar(100);comment:支付密码（bcrypt，独立于登录密码，诺哈 wap_user_money.pass）" json:"-"`           // 支付密码（bcrypt，独立于登录密码，诺哈 wap_user_money.pass）
	Paid           int        `gorm:"default:0;comment:累计消费（诺哈 wap_user.paid）" json:"paid"`                                     // 累计消费（诺哈 wap_user.paid）
	Introduction   string     `gorm:"type:varchar(200);comment:个人简介" json:"introduction"`                                       // 个人简介
	Signature      string     `gorm:"type:varchar(100);comment:Signature" json:"signature"`
	City           string     `gorm:"type:varchar(30);comment:城市设置" json:"city"`                                       // 城市设置
	Color          string     `gorm:"type:varchar(200);comment:昵称颜色，情怀功能（可存逗号分隔的逐字颜色序列），普通用户默认蓝色" json:"color"`        // 昵称颜色，情怀功能（可存逗号分隔的逐字颜色序列），普通用户默认蓝色
	NameStart      *time.Time `gorm:"comment:个性昵称开通时间（复刻 3GQQ name.html/name_buy.asp）" json:"name_start"`              // 个性昵称开通时间（复刻 3GQQ name.html/name_buy.asp）
	NameEnd        *time.Time `gorm:"comment:个性昵称有效期" json:"name_end"`                                                 // 个性昵称有效期
	Avatar         string     `gorm:"type:varchar(100);comment:头像图片文件名（static/picture 下）" json:"avatar"`               // 头像图片文件名（static/picture 下）
	AvatarBase64   string     `gorm:"type:longtext;comment:自定义头像 base64（data URI，优先于 avatar 展示）" json:"avatar_base64"` // 自定义头像 base64（data URI，优先于 avatar 展示）
	Coins          int        `gorm:"default:0;comment:G币（主货币，发帖回帖/打工/签到获得）" json:"coins"`                             // G币（主货币，发帖回帖/打工/签到获得）
	YuanBao        int        `gorm:"column:yuanbao;default:0;comment:元宝（活动/连签奖励）" json:"yuanbao"`                     // 元宝（活动/连签奖励）
	JinZuan        int        `gorm:"column:jinzuan;default:0;comment:金钻（稀有货币，活动/后台发放）" json:"jinzuan"`                // 金钻（稀有货币，活动/后台发放）
	YouQuan        int        `gorm:"column:youquan;default:0;comment:友友券（活动/连签奖励）" json:"youquan"`                    // 友友券（活动/连签奖励）
	Exp            int        `gorm:"default:0;comment:经验" json:"exp"`
	Level          int        `gorm:"default:1;comment:等级" json:"level"`
	Noble          int        `gorm:"default:0;comment:贵族身份 0无 1一级 2二级" json:"noble"`  // 贵族身份 0无 1一级 2二级
	PartnerID      uint       `gorm:"default:0;comment:婚恋：伴侣（城堡）" json:"partner_id"`   // 婚恋：伴侣（城堡）
	BabyName       string     `gorm:"type:varchar(20);comment:婚恋：宝宝" json:"baby_name"` // 婚恋：宝宝
	Achieve        int        `gorm:"default:0;comment:社区成就点" json:"achieve"`          // 社区成就点
	NobleExp       int        `gorm:"default:0;comment:超Q/蓝钻成长值" json:"noble_exp"`     // 超Q/蓝钻成长值
	BlueLv         int        `gorm:"default:0;comment:蓝钻等级" json:"blue_lv"`           // 蓝钻等级
	BlueExp        int        `gorm:"default:0;comment:蓝钻成长值" json:"blue_exp"`         // 蓝钻成长值
	BlueStart      *time.Time `gorm:"comment:蓝钻开始" json:"blue_start"`
	BlueEnd        *time.Time `gorm:"comment:蓝钻结束" json:"blue_end"`
	BluePtime      *time.Time `gorm:"comment:蓝钻上次每日成长时间" json:"blue_ptime"`                 // 蓝钻上次每日成长时间
	BlueSpeed      int        `gorm:"default:0;comment:蓝钻成长速度（点/天，0=默认）" json:"blue_speed"` // 蓝钻成长速度（点/天，0=默认）
	QqLv           int        `gorm:"default:0;comment:超Q等级" json:"qq_lv"`                  // 超Q等级
	QqExp          int        `gorm:"default:0;comment:超Q成长值" json:"qq_exp"`                // 超Q成长值
	QqStart        *time.Time `gorm:"comment:QQ开始" json:"qq_start"`
	QqEnd          *time.Time `gorm:"comment:QQ结束" json:"qq_end"`
	QqPtime        *time.Time `gorm:"comment:超Q上次每日成长时间" json:"qq_ptime"`                 // 超Q上次每日成长时间
	QqSpeed        int        `gorm:"default:0;comment:超Q成长速度（点/天，0=默认）" json:"qq_speed"` // 超Q成长速度（点/天，0=默认）
	GardenPots     int        `gorm:"default:4;comment:魔法花园花盆数" json:"garden_pots"`       // 魔法花园花盆数
	Status         int        `gorm:"default:1;comment:1正常 0封禁" json:"status"`            // 1正常 0封禁
	LastActiveAt   *time.Time `gorm:"comment:最后活跃时间" json:"last_active_at"`
	LastBoardID    uint       `gorm:"default:0;comment:最后停留版块（版块在线统计，参考诺哈 wap_online.bbsid）" json:"last_board_id"` // 最后停留版块（版块在线统计，参考诺哈 wap_online.bbsid）
	LastLoginAt    *time.Time `gorm:"comment:最后Login时间" json:"last_login_at"`
	ActiveDays     float64    `gorm:"default:0;comment:家园活跃天数" json:"active_days"`                 // 家园活跃天数
	LastActiveDate string     `gorm:"type:varchar(10);comment:最后活跃日期(去重)" json:"last_active_date"` // 最后活跃日期(去重)
	InvitedBy      uint       `gorm:"default:0;comment:邀请人（0=自然注册）" json:"invited_by"`             // 邀请人（0=自然注册）
	CreatedAt      time.Time  `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"comment:更新时间" json:"updated_at"`
	Roles          []Role     `gorm:"many2many:user_roles;" json:"roles,omitempty"`
	Badges         []Badge    `gorm:"many2many:user_badges;joinForeignKey:UserID;joinReferences:BadgeID;" json:"badges,omitempty"` // 会员勋章（含排序/过期）
	PrivID         uint       `gorm:"default:0;comment:特权（蓝钻/超Q等级图标）" json:"priv_id"`                                              // 特权（蓝钻/超Q等级图标）
	Priv           *Resource  `gorm:"foreignKey:PrivID" json:"priv,omitempty"`
	LevelIcon      int        `gorm:"-" json:"level_icon"`    // 等级图标 v{N}.gif
	LevelTitle     string     `gorm:"-" json:"level_title"`   // 等级称号（如 册封骑士）
	AchieveLevel   int        `gorm:"-" json:"achieve_level"` // 成就等级（每100点升1级）
}

// NumHistory 用户曾用家园号（靓号转换记录，曾用过的都展示）
type NumHistory struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Num       string    `gorm:"type:varchar(10);comment:转换前的家园号码" json:"num"` // 转换前的家园号码
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (NumHistory) TableName() string { return "num_histories" }

// 勋章商店（复刻诺哈 wap_medal_shop）：Icon 为 static/picture 下的图片文件名
// sort 排序 / price 价格 / period 有效期限(天,0=永久) / status 状态
type Badge struct {
	ID     uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name   string `gorm:"type:varchar(30);comment:名称" json:"name"`
	Icon   string `gorm:"type:varchar(50);comment:图标" json:"icon"`
	Remark string `gorm:"type:varchar(100);comment:备注" json:"remark"`
	Price  int    `gorm:"default:0;comment:价格" json:"price"`
	Period int    `gorm:"default:0;comment:周期" json:"period"`
	Sort   int    `gorm:"default:0;comment:排序值" json:"sort"`
	Status int    `gorm:"default:1;comment:状态" json:"status"`
}

func (Badge) TableName() string { return "badges" }

// 会员勋章（复刻诺哈 wap_medal）：授予用户的勋章记录，含排序与过期时间
type UserBadge struct {
	ID        uint       `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint       `gorm:"uniqueIndex:uk_ub;index;comment:用户ID" json:"user_id"`
	BadgeID   uint       `gorm:"uniqueIndex:uk_ub;comment:勋章ID" json:"badge_id"`
	Sort      int        `gorm:"default:0;comment:排序值" json:"sort"`
	GrantedAt time.Time  `gorm:"comment:授予时间" json:"granted_at"`
	ExpireAt  *time.Time `gorm:"comment:过期时间" json:"expire_at"`
}

func (UserBadge) TableName() string { return "user_badges" }

// 等级图标可用的 v{N}.gif（来自演示站素材）
var levelIcons = []int{1, 6, 9, 10, 11, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 36}

// 等级称号阶梯（达到阈值即获得）
var levelTitles = []struct {
	Level int
	Title string
}{
	{40, "采邑领主"}, {36, "一代宗师"}, {32, "社区支柱"}, {28, "家园贵族"}, {25, "金牌骑士"},
	{22, "册封骑士"}, {20, "银牌友友"}, {18, "社区元老"}, {15, "家园红人"},
	{12, "论坛达人"}, {10, "社区明星"}, {8, "活跃分子"}, {5, "正式居民"},
	{3, "见习友友"}, {1, "新人报到"},
}

// AfterFind 每次查询自动带上等级图标、称号与特权图标
func (u *User) AfterFind(tx *gorm.DB) error {
	u.LevelIcon = 1
	for _, n := range levelIcons {
		if u.Level >= n {
			u.LevelIcon = n
		}
	}
	for _, lt := range levelTitles {
		if u.Level >= lt.Level {
			u.LevelTitle = lt.Title
			break
		}
	}
	u.AchieveLevel = u.Achieve / 100
	if u.PrivID > 0 {
		var r Resource
		if err := tx.First(&r, u.PrivID).Error; err == nil && r.Status == 1 {
			u.Priv = &r
		}
	}
	return nil
}

func (User) TableName() string       { return "users" }
func (Role) TableName() string       { return "roles" }
func (Permission) TableName() string { return "permissions" }

// 角色
type Role struct {
	ID          uint         `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name        string       `gorm:"type:varchar(30);uniqueIndex;comment:名称" json:"name"`
	Code        string       `gorm:"type:varchar(30);uniqueIndex;comment:编码" json:"code"`
	Remark      string       `gorm:"type:varchar(100);comment:备注" json:"remark"`
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

// 权限点
type Permission struct {
	ID     uint   `gorm:"primaryKey;comment:主键ID" json:"id"`
	Name   string `gorm:"type:varchar(30);comment:名称" json:"name"`
	Code   string `gorm:"type:varchar(50);uniqueIndex;comment:编码" json:"code"`
	Remark string `gorm:"type:varchar(100);comment:备注" json:"remark"`
}

// 由经验值计算等级：每 100*(n) 经验升 n 级，简单直观
func CalcLevel(exp int) int {
	level, need := 1, 100
	for exp >= need {
		exp -= need
		level++
		need = level * 100
	}
	return level
}
