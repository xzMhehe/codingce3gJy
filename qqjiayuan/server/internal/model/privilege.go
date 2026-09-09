package model

import "time"

// 贵宾等级配置（复刻诺哈三代 wap_vip_config）：升级经验 + 等级图标
// id 即等级（1-8 级），point 为升到本级所需成长值，图标为 static/picture 下文件名
type NobleLevel struct {
	ID       uint   `gorm:"primaryKey" json:"id"`                // 等级
	Point    int    `gorm:"default:0" json:"point"`              // 升级经验（诺哈 vip_edit 的 point）
	IconBlue string `gorm:"type:varchar(50)" json:"icon_blue"`   // 蓝钻等级图标
	IconQQ   string `gorm:"type:varchar(50)" json:"icon_qq"`     // 超Q等级图标
}

func (NobleLevel) TableName() string { return "noble_levels" }

// 默认等级配置（对齐诺哈 wap_vip_config 初始数据）
var NobleLevelPresets = []NobleLevel{
	{ID: 1, Point: 0, IconBlue: "noble_2_1.gif", IconQQ: "noble_1_1.gif"},
	{ID: 2, Point: 600, IconBlue: "noble_2_2.gif", IconQQ: "noble_1_2.gif"},
	{ID: 3, Point: 1800, IconBlue: "noble_2_3.gif", IconQQ: "noble_1_3.gif"},
	{ID: 4, Point: 3600, IconBlue: "noble_2_4.gif", IconQQ: "noble_1_4.gif"},
	{ID: 5, Point: 6000, IconBlue: "noble_2_5.gif", IconQQ: "noble_1_5.gif"},
	{ID: 6, Point: 10800, IconBlue: "noble_2_6.gif", IconQQ: "noble_1_6.gif"},
	{ID: 7, Point: 32400, IconBlue: "noble_2_7.gif", IconQQ: "noble_1_7.gif"},
	{ID: 8, Point: 46800, IconBlue: "noble_2_8.gif", IconQQ: "noble_1_8.gif"},
}

// NobleLvOf 按成长值算贵宾等级：0 成长值未开通为 0 级，其余取满足门槛的最高级
func NobleLvOf(levels []NobleLevel, exp int) int {
	if exp <= 0 {
		return 0
	}
	lv := 1
	for _, l := range levels {
		if exp >= l.Point && int(l.ID) > lv {
			lv = int(l.ID)
		}
	}
	return lv
}

// NobleIconOf 取某等级图标，typ=blue 蓝钻 / qq 超Q
func NobleIconOf(levels []NobleLevel, lv int, typ string) string {
	for _, l := range levels {
		if int(l.ID) == lv {
			if typ == "blue" {
				return l.IconBlue
			}
			return l.IconQQ
		}
	}
	if len(levels) > 0 {
		if typ == "blue" {
			return levels[0].IconBlue
		}
		return levels[0].IconQQ
	}
	return ""
}

// 特权开通方案（后台可管理）：type=blue 蓝钻 / qq 超Q
// 复刻诺哈三代 wap_vip_shop：标题/周期/币种/价格/成长速度/赠送经验/限购/库存/销量/出售时间/结束时间/状态
type NoblePlan struct {
	ID     uint       `gorm:"primaryKey" json:"id"`
	Type   string     `gorm:"type:varchar(10)" json:"type"` // blue/qq
	Name   string     `gorm:"type:varchar(40)" json:"name"`
	Cost   int        `gorm:"default:0" json:"cost"` // 订购价格（G币/月）
	Gain   int        `gorm:"default:0" json:"gain"` // 成长赠送（开通时一次性）
	Speed  int        `gorm:"default:0" json:"speed"` // 成长速度（点/天）
	Days   int        `gorm:"default:0" json:"days"`  // 周期天数（30=包月，诺哈 cycle 销售几月）
	Money  int        `gorm:"default:1" json:"money"` // 币种 1=G币（诺哈 money）
	Limit  int        `gorm:"default:0" json:"limit"` // 每号限购（0=不限）
	Stock  int        `gorm:"default:0" json:"stock"` // 库存（0=不限）
	Sales  int        `gorm:"default:0" json:"sales"` // 销售数量（累计）
	Stime  *time.Time `json:"stime"`                  // 出售时间
	Etime  *time.Time `json:"etime"`                  // 结束时间
	Status int        `gorm:"default:1" json:"status"` // 销售状态 1上架 0下架
	Sort   int        `gorm:"default:0" json:"sort"`
}

func (NoblePlan) TableName() string { return "noble_plans" }

// 默认方案（对齐诺哈：成长速度点/天、成长赠送、限购、库存）
var NoblePlanPresets = []NoblePlan{
	{Type: "blue", Name: "包月蓝钻等级加速", Cost: 500, Gain: 100, Speed: 10, Days: 30, Money: 1, Status: 1, Sort: 1},
	{Type: "qq", Name: "包月超Q等级加速", Cost: 1000, Gain: 200, Speed: 15, Days: 30, Money: 1, Status: 1, Sort: 2},
	{Type: "blue", Name: "年费蓝钻等级飞速", Cost: 5000, Gain: 1200, Speed: 12, Days: 365, Money: 1, Status: 1, Sort: 3},
	{Type: "qq", Name: "年费超Q等级飞速", Cost: 8000, Gain: 2400, Speed: 18, Days: 365, Money: 1, Status: 1, Sort: 4},
}

// OnSale 方案是否在售：上架且在出售~结束时间窗口内
func (p NoblePlan) OnSale(now time.Time) bool {
	if p.Status != 1 {
		return false
	}
	if p.Stime != nil && p.Stime.After(now) {
		return false
	}
	if p.Etime != nil && p.Etime.Before(now) {
		return false
	}
	return true
}
