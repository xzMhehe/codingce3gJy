package model

import "time"

// 道具商城商品（后台可管理）
type Good struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Name     string `gorm:"type:varchar(40)" json:"name"`
	Desc     string `gorm:"type:varchar(200)" json:"desc"`
	Icon     string `gorm:"type:varchar(50)" json:"icon"`
	Category string `gorm:"type:varchar(20)" json:"category"` // 鲜花/道具/装扮/特权
	Price    int    `gorm:"default:0" json:"price"`
	Status   int    `gorm:"default:1" json:"status"` // 1上架 0下架
	Sort     int    `gorm:"default:0" json:"sort"`
}

func (Good) TableName() string { return "goods" }

// 我的道具（背包）：购买后入库，可叠加数量
type UserGood struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex:uk_ug" json:"user_id"`
	GoodID    uint      `gorm:"uniqueIndex:uk_ug" json:"good_id"`
	Count     int       `gorm:"default:0" json:"count"`
	UpdatedAt time.Time `json:"updated_at"`
	Good      *Good     `gorm:"foreignKey:GoodID" json:"good,omitempty"`
}

func (UserGood) TableName() string { return "user_goods" }

var GoodPresets = []Good{
	{Name: "玫瑰花", Desc: "娇艳欲滴的玫瑰，可在帖子下方送给好友", Icon: "flower_rose.gif", Category: "鲜花", Price: 20, Sort: 1},
	{Name: "向日葵", Desc: "阳光灿烂的向日葵，送花示爱暖人心", Icon: "flower_sun.gif", Category: "鲜花", Price: 15, Sort: 2},
	{Name: "郁金香", Desc: "高贵典雅的郁金香，送花祝福好运", Icon: "flower_tulip.gif", Category: "鲜花", Price: 35, Sort: 3},
	{Name: "月光花", Desc: "月光下的神秘花朵，稀有珍品", Icon: "flower_moon.gif", Category: "鲜花", Price: 60, Sort: 4},
	{Name: "改名卡", Desc: "可修改一次昵称（仓库中使用）", Icon: "card_rename.gif", Category: "道具", Price: 500, Sort: 5},
	{Name: "家园皮肤·蓝", Desc: "家园首页蓝色皮肤", Icon: "skin_blue.gif", Category: "装扮", Price: 1000, Sort: 6},
	{Name: "聊天气泡·金", Desc: "聊天室金色气泡", Icon: "bubble_gold.gif", Category: "装扮", Price: 800, Sort: 7},
	{Name: "经验加速卡", Desc: "发帖回帖经验+50%（使用后当日生效）", Icon: "card_exp.gif", Category: "特权", Price: 1200, Sort: 8},
	{Name: "幸运星", Desc: "今日星运更亮", Icon: "star_luck.gif", Category: "道具", Price: 300, Sort: 9},
	{Name: "头像框·玫瑰", Desc: "主页头像玫瑰框", Icon: "avatar_rose.gif", Category: "装扮", Price: 600, Sort: 10},
}
