package model

// 特权开通方案（后台可管理）：type=blue 蓝钻 / qq 超Q
// 复刻诺哈三代 wap_vip_shop：订购价格/成长速度/成长赠送/每号限购/库存/销量
type NoblePlan struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Type  string `gorm:"type:varchar(10)" json:"type"` // blue/qq
	Name  string `gorm:"type:varchar(40)" json:"name"`
	Cost  int    `gorm:"default:0" json:"cost"` // 订购价格（G币/月）
	Gain  int    `gorm:"default:0" json:"gain"` // 成长赠送（开通时一次性）
	Speed int    `gorm:"default:0" json:"speed"` // 成长速度（点/天）
	Days  int    `gorm:"default:0" json:"days"`  // 周期天数（30=包月）
	Limit int    `gorm:"default:0" json:"limit"` // 每号限购（0=不限）
	Stock int    `gorm:"default:0" json:"stock"` // 库存（0=不限）
	Sales int    `gorm:"default:0" json:"sales"` // 销售数量（累计）
	Sort  int    `gorm:"default:0" json:"sort"`
}

func (NoblePlan) TableName() string { return "noble_plans" }

// 默认方案（对齐诺哈：成长速度点/天、成长赠送、限购、库存）
var NoblePlanPresets = []NoblePlan{
	{Type: "blue", Name: "包月蓝钻等级加速", Cost: 500, Gain: 100, Speed: 10, Days: 30, Limit: 0, Stock: 0, Sort: 1},
	{Type: "qq", Name: "包月超Q等级加速", Cost: 1000, Gain: 200, Speed: 15, Days: 30, Limit: 0, Stock: 0, Sort: 2},
	{Type: "blue", Name: "年费蓝钻等级飞速", Cost: 5000, Gain: 1200, Speed: 12, Days: 365, Limit: 0, Stock: 0, Sort: 3},
	{Type: "qq", Name: "年费超Q等级飞速", Cost: 8000, Gain: 2400, Speed: 18, Days: 365, Limit: 0, Stock: 0, Sort: 4},
}
