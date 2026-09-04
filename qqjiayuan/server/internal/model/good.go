package model

// 道具商城商品（后台可管理）
type Good struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Name     string `gorm:"type:varchar(40)" json:"name"`
	Desc     string `gorm:"type:varchar(200)" json:"desc"`
	Icon     string `gorm:"type:varchar(50)" json:"icon"`
	Category string `gorm:"type:varchar(20)" json:"category"` // 装扮/道具/特权/其他
	Price    int    `gorm:"default:0" json:"price"`
	Status   int    `gorm:"default:1" json:"status"` // 1上架 0下架
	Sort     int    `gorm:"default:0" json:"sort"`
}

func (Good) TableName() string { return "goods" }

var GoodPresets = []Good{
	{Name: "改名卡", Desc: "可修改一次昵称", Category: "道具", Price: 500, Sort: 1},
	{Name: "家园皮肤·蓝", Desc: "家园首页皮肤", Category: "装扮", Price: 1000, Sort: 2},
	{Name: "聊天气泡·金", Desc: "聊天室金色气泡", Category: "装扮", Price: 800, Sort: 3},
	{Name: "经验加速卡", Desc: "发帖回帖经验+50%", Category: "特权", Price: 1200, Sort: 4},
	{Name: "幸运星", Desc: "每日星运更亮", Category: "道具", Price: 300, Sort: 5},
	{Name: "头像框·玫瑰", Desc: "主页头像玫瑰框", Category: "装扮", Price: 600, Sort: 6},
}
