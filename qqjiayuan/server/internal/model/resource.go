package model

// 静态资源目录：图标/头像/游戏logo/特权图标等，管理端「文件管理」维护
// File 为 static 相对路径（如 picture/v1.gif）；管理端上传的图片存 Data(base64 data URI)，File 形如 db/xxx.gif，经 /api/res/db/xxx.gif 提供
type Resource struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	File     string `gorm:"type:varchar(120);uniqueIndex" json:"file"` // 相对 static 的路径或 db/ 上传件名
	Category string `gorm:"type:varchar(10);index" json:"category"`    // badge/avatar/game/priv/other
	Name     string `gorm:"type:varchar(30)" json:"name"`              // 资源名称（特权名称等）
	Level    int    `gorm:"default:0" json:"level"`                    // 特权等级 1-8
	Status   int    `gorm:"default:1" json:"status"`                   // 1启用 0停用（停用后不出现在选择器）
	Data     string `gorm:"type:longtext" json:"-"`                    // base64 data URI（库存图片，管理端上传）
}

func (Resource) TableName() string { return "resources" }

// 预置素材清单（均来自演示站 static 目录）
var (
	BadgeIconPresets = []string{
		"706.jpg", "3.gif", "501.gif", "704.gif", "803.gif",
		"804.gif", "15.gif", "103.gif", "903.gif", "45.gif",
		// 复刻诺哈勋章图标（按诺哈 wap_medal_shop 的职务/贡献勋章补全）
		"201.gif", "202.gif", "701.gif", "702.gif", "703.gif", "705.gif",
		"801.gif", "802.gif", "902.gif", "904.gif", "906.gif",
		"1002.gif", "1004.gif", "1202.gif", "1204.gif", "1206.gif",
		"1301.gif", "1303.gif", "1305.gif", "1307.gif", "1402.gif",
		// 线上勋章中心勋章图标（复刻自 3gqq.cn/bbs/medal）
		"200.gif", "201.gif", "140.gif", "141.gif", "1689.gif", "163.gif", "164.gif", "165.gif",
		"166.gif", "172.gif", "173.gif", "176.gif", "177.gif", "178.gif", "179.gif", "180.gif",
		"181.gif", "182.gif", "184.gif", "185.gif", "186.gif", "187.gif", "188.gif", "189.gif",
		"190.gif", "191.gif", "192.gif", "193.gif", "206.gif", "207.gif", "208.gif", "209.gif",
		"210.gif", "211.gif", "212.gif", "214.gif", "205.gif", "194.gif", "390.gif", "2018.gif",
		"99996.gif", "99997.gif", "888886.gif", "379.gif", "380.gif", "398.gif", "1683.gif",
		"151.gif", "3767.gif", "2011.gif", "2012.gif", "2016.gif", "229.gif", "230.gif", "235.gif",
		"237.gif", "240.gif", "364.gif", "375.gif", "376.gif", "88810.gif", "88811.gif", "88812.gif",
		"88889.gif", "101.gif", "213.gif", "135.gif", "137.gif", "138.gif", "132.gif", "143.gif",
		"170.gif", "174.gif", "171.gif", "167.gif", "168.gif", "144.gif", "133.gif", "150.gif",
		"183.gif", "134.png", "139.gif", "142.gif", "145.gif", "175.gif", "2002.gif", "2003.gif",
		"2000.gif", "261.gif", "262.gif", "263.gif", "20260611.png", "1989.gif", "1990.gif",
		"1991.gif", "1992.gif", "1993.gif", "1994.gif", "1995.gif", "366.gif", "367.gif", "368.gif",
		"370.gif", "371.gif", "372.gif", "373.gif", "374.gif", "1996.gif", "2021.jpg",
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
