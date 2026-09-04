package model

// 特权开通方案（后台可管理）：type=blue 蓝钻 / qq 超Q
type NoblePlan struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Type  string `gorm:"type:varchar(10)" json:"type"` // blue/qq
	Name  string `gorm:"type:varchar(40)" json:"name"`
	Cost  int    `gorm:"default:0" json:"cost"`
	Gain  int    `gorm:"default:0" json:"gain"` // 成长值
	Days  int    `gorm:"default:0" json:"days"`
	Sort  int    `gorm:"default:0" json:"sort"`
}

func (NoblePlan) TableName() string { return "noble_plans" }

// 默认方案
var NoblePlanPresets = []NoblePlan{
	{Type: "blue", Name: "包月蓝钻等级加速", Cost: 500, Gain: 100, Days: 30, Sort: 1},
	{Type: "qq", Name: "包月超Q等级加速", Cost: 1000, Gain: 200, Days: 30, Sort: 2},
	{Type: "blue", Name: "年费蓝钻等级飞速", Cost: 5000, Gain: 1200, Days: 365, Sort: 3},
	{Type: "qq", Name: "年费超Q等级飞速", Cost: 8000, Gain: 2400, Days: 365, Sort: 4},
}
