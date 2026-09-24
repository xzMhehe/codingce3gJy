package model

import "time"

// ============ 用户附属信息（对齐诺哈三代 wap_user_address / wap_user_docu / wap_user_protec / wap_user_log） ============

// UserAddress 通信地址：故乡 + 现居（诺哈 wap_user_address，一人一条）
type UserAddress struct {
	ID     uint `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID uint `gorm:"uniqueIndex;comment:用户ID" json:"user_id"`
	// 故乡 h=home
	HomeNation string `gorm:"type:varchar(20);comment:故乡 h=home" json:"home_nation"`
	HomeProv   string `gorm:"type:varchar(20);comment:家园省份" json:"home_prov"`
	HomeCity   string `gorm:"type:varchar(20);comment:家园城市" json:"home_city"`
	HomeDist   string `gorm:"type:varchar(20);comment:家园地区" json:"home_dist"`
	HomeAddr   string `gorm:"type:varchar(100);comment:家园地址" json:"home_addr"`
	HomeZip    string `gorm:"type:varchar(10);comment:家园邮编" json:"home_zip"`
	// 现居 l=live
	LiveNation string    `gorm:"type:varchar(20);comment:现居 l=live" json:"live_nation"`
	LiveProv   string    `gorm:"type:varchar(20);comment:在线省份" json:"live_prov"`
	LiveCity   string    `gorm:"type:varchar(20);comment:在线城市" json:"live_city"`
	LiveDist   string    `gorm:"type:varchar(20);comment:在线地区" json:"live_dist"`
	LiveAddr   string    `gorm:"type:varchar(100);comment:在线地址" json:"live_addr"`
	LiveZip    string    `gorm:"type:varchar(10);comment:在线邮编" json:"live_zip"`
	UpdatedAt  time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (UserAddress) TableName() string { return "user_addresses" }

// UserDocument 实名证件（诺哈 wap_user_docu：type 1=身份证，设置需登录密码确认）
type UserDocument struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"uniqueIndex;comment:用户ID" json:"user_id"`
	Type      int       `gorm:"default:1;comment:1身份证" json:"type"` // 1身份证
	RealName  string    `gorm:"type:varchar(30);comment:Real名称" json:"real_name"`
	Number    string    `gorm:"type:varchar(30);comment:Number" json:"number"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (UserDocument) TableName() string { return "user_documents" }

// UserProtection 密保问题（诺哈 wap_user_protec：issue 问题编号，answer MD5）
type UserProtection struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"uniqueIndex;comment:用户ID" json:"user_id"`
	Issue     int       `gorm:"default:0;comment:问题编号（见 ProtectionQuestions）" json:"issue"` // 问题编号（见 ProtectionQuestions）
	Answer    string    `gorm:"type:varchar(64);comment:MD5(答案)" json:"-"`                  // MD5(答案)
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"comment:更新时间" json:"updated_at"`
}

func (UserProtection) TableName() string { return "user_protections" }

// ProtectionQuestions 密保问题库（对齐诺哈密保问题选项）
var ProtectionQuestions = []string{
	"您母亲的姓名是？",
	"您父亲的姓名是？",
	"您的小学名称是？",
	"您的出生地是？",
	"您最爱的电影是？",
	"您第一只宠物的名字是？",
	"您最喜欢的运动是？",
	"您母亲的生日是？",
}

// UserLog 登录/操作日志（诺哈 wap_user_log：name=动作名 如"登陆成功/登陆失败"，intro=详情，addip）
type UserLog struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	UserID    uint      `gorm:"index;comment:用户ID" json:"user_id"`
	Action    string    `gorm:"type:varchar(30);comment:登陆成功/登陆失败/修改密码/..." json:"action"` // 登陆成功/登陆失败/修改密码/...
	Intro     string    `gorm:"type:varchar(200);comment:简介" json:"intro"`
	IP        string    `gorm:"type:varchar(45);comment:IP地址" json:"ip"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (UserLog) TableName() string { return "user_logs" }
