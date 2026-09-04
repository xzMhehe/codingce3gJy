package model

// 静态资源目录：图标/头像/游戏logo/特权图标等，管理端「资源管理」维护
type Resource struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	File     string `gorm:"type:varchar(120);uniqueIndex" json:"file"` // 相对 static 的路径，如 picture/v1.gif
	Category string `gorm:"type:varchar(10);index" json:"category"`    // badge/avatar/game/priv/other
	Name     string `gorm:"type:varchar(30)" json:"name"`              // 资源名称（特权名称等）
	Level    int    `gorm:"default:0" json:"level"`                    // 特权等级 1-8
	Status   int    `gorm:"default:1" json:"status"`                   // 1启用 0停用（停用后不出现在选择器）
}

func (Resource) TableName() string { return "resources" }

// 预置素材清单（均来自演示站 static 目录）
var (
	BadgeIconPresets = []string{
		"706.jpg", "3.gif", "501.gif", "704.gif", "803.gif",
		"804.gif", "15.gif", "103.gif", "903.gif", "45.gif",
	}
	AvatarPresets = []string{
		"131851611.jpg", "1985acg.jpg", "104039478.jpg",
		"125703412.png", "125751186.gif", "1031047330.png",
	}
	GameLogoPresets = []string{
		"logo.jpg", "mofahuayuan.gif", "hunli2.jpg", "kaixinnongchang.gif",
		"kuangqiangchewei.gif", "jwt.png", "cwlogo.gif", "shuiguoleyuan.gif",
		"quanminliema.gif", "jiayuangushi.gif", "dahuachuiniu.gif",
	}
)

// 特权等级种子：超Q 1-8 级 + 蓝钻 1-8 级（图标来自真实  资源）
var PrivSeed = []struct {
	File  string
	Name  string
	Level int
}{
	{"picture/sq1.1.gif", "超Q 1级", 1},
	{"picture/sq1.2.gif", "超Q 2级", 2},
	{"picture/sq1.3.gif", "超Q 3级", 3},
	{"picture/sq1.4.gif", "超Q 4级", 4},
	{"picture/sq1.5.gif", "超Q 5级", 5},
	{"picture/sq1.6.gif", "超Q 6级", 6},
	{"picture/sq1.7.gif", "超Q 7级", 7},
	{"picture/sq1.8.gif", "超Q 8级", 8},
	{"picture/lz2.1.gif", "蓝钻 1级", 9},
	{"picture/lz2.2.gif", "蓝钻 2级", 10},
	{"picture/lz2.3.gif", "蓝钻 3级", 11},
	{"picture/lz2.4.gif", "蓝钻 4级", 12},
	{"picture/lz2.5.gif", "蓝钻 5级", 13},
	{"picture/lz2.6.gif", "蓝钻 6级", 14},
	{"picture/lz2.7.gif", "蓝钻 7级", 15},
	{"picture/lz2.8.gif", "蓝钻 8级", 16},
}
