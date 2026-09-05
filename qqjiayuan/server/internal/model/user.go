package model

import (
	"time"

	"gorm.io/gorm"
)

// 用户表：家园号码即 ID（自增起始 10000），类似当年 QQ 号码
type User struct {
	ID           uint    `gorm:"primaryKey" json:"id"`
	Username     string  `gorm:"type:varchar(20);uniqueIndex" json:"username"` // 家园号码（同 ID）
	Nickname     string  `gorm:"type:varchar(20);uniqueIndex" json:"nickname"`
	Password     string  `gorm:"type:varchar(100)" json:"-"`
	Gender       int     `gorm:"default:1" json:"gender"` // 1小哥哥 2小姐姐
	Age          int     `gorm:"default:0" json:"age"`
	BirthYear    int     `gorm:"default:0" json:"birth_year"`
	BirthMonth   int     `gorm:"default:0" json:"birth_month"`
	BirthDay     int     `gorm:"default:0" json:"birth_day"`
	Introduction string  `gorm:"type:varchar(200)" json:"introduction"` // 个人简介
	Signature    string    `gorm:"type:varchar(100)" json:"signature"`
	City         string    `gorm:"type:varchar(30)" json:"city"`    // 城市设置
	Color        string    `gorm:"type:varchar(10)"  json:"color"`  // 昵称颜色，情怀功能
	Avatar       string  `gorm:"type:varchar(100)" json:"avatar"` // 头像图片文件名（static/picture 下）
	Coins        int     `gorm:"default:0" json:"coins"` // G币（主货币，发帖回帖/打工/签到获得）
	YuanBao      int     `gorm:"column:yuanbao;default:0" json:"yuanbao"`      // 元宝（活动/连签奖励）
	JinZuan      int     `gorm:"column:jinzuan;default:0" json:"jinzuan"`      // 金钻（稀有货币，活动/后台发放）
	YouQuan      int     `gorm:"column:youquan;default:0" json:"youquan"`      // 友友券（活动/连签奖励）
	Exp          int     `gorm:"default:0" json:"exp"`
	Level        int     `gorm:"default:1" json:"level"`
	Noble        int     `gorm:"default:0" json:"noble"`         // 贵族身份 0无 1一级 2二级
	PartnerID    uint    `gorm:"default:0" json:"partner_id"`    // 婚恋：伴侣（城堡）
	BabyName     string  `gorm:"type:varchar(20)" json:"baby_name"` // 婚恋：宝宝
	Achieve      int     `gorm:"default:0" json:"achieve"`       // 社区成就点
	NobleExp     int     `gorm:"default:0" json:"noble_exp"`   // 超Q/蓝钻成长值
	BlueLv       int        `gorm:"default:0" json:"blue_lv"`     // 蓝钻等级
	BlueExp      int        `gorm:"default:0" json:"blue_exp"`    // 蓝钻成长值
	BlueStart    *time.Time `json:"blue_start"`
	BlueEnd      *time.Time `json:"blue_end"`
	QqLv         int        `gorm:"default:0" json:"qq_lv"`       // 超Q等级
	QqExp        int        `gorm:"default:0" json:"qq_exp"`      // 超Q成长值
	QqStart      *time.Time `json:"qq_start"`
	QqEnd        *time.Time `json:"qq_end"`
	GardenPots   int     `gorm:"default:4" json:"garden_pots"` // 魔法花园花盆数
	Status       int     `gorm:"default:1" json:"status"` // 1正常 0封禁
	LastActiveAt  *time.Time `json:"last_active_at"`
	LastLoginAt   *time.Time `json:"last_login_at"`
	ActiveDays    float64    `gorm:"default:0" json:"active_days"`  // 家园活跃天数
	LastActiveDate string    `gorm:"type:varchar(10)" json:"last_active_date"` // 最后活跃日期(去重)
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Roles        []Role  `gorm:"many2many:user_roles;" json:"roles,omitempty"`
	Badges       []Badge `gorm:"many2many:user_badges;" json:"badges,omitempty"` // 马甲/勋章图标
	PrivID       uint    `gorm:"default:0" json:"priv_id"`                       // 特权（蓝钻/超Q等级图标）
	Priv         *Resource `gorm:"foreignKey:PrivID" json:"priv,omitempty"`
	LevelIcon    int     `gorm:"-" json:"level_icon"`                            // 等级图标 v{N}.gif
	LevelTitle   string  `gorm:"-" json:"level_title"`                           // 等级称号（如 册封骑士）
	AchieveLevel int     `gorm:"-" json:"achieve_level"`                         // 成就等级（每100点升1级）
}

// 勋章/马甲：Icon 为 static/picture 下的图片文件名，昵称前的一串小图标
type Badge struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	Name   string `gorm:"type:varchar(30)" json:"name"`
	Icon   string `gorm:"type:varchar(50)" json:"icon"`
	Remark string `gorm:"type:varchar(100)" json:"remark"`
	Status int    `gorm:"default:1" json:"status"`
}

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
func (Badge) TableName() string      { return "badges" }

// 角色
type Role struct {
	ID          uint         `gorm:"primaryKey" json:"id"`
	Name        string       `gorm:"type:varchar(30);uniqueIndex" json:"name"`
	Code        string       `gorm:"type:varchar(30);uniqueIndex" json:"code"`
	Remark      string       `gorm:"type:varchar(100)" json:"remark"`
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

// 权限点
type Permission struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	Name   string `gorm:"type:varchar(30)" json:"name"`
	Code   string `gorm:"type:varchar(50);uniqueIndex" json:"code"`
	Remark string `gorm:"type:varchar(100)" json:"remark"`
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
