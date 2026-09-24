package model

// 通用键值配置
type Setting struct {
	Key   string `gorm:"type:varchar(50);primaryKey;comment:键名" json:"key"`
	Value string `gorm:"type:varchar(255);comment:值" json:"value"`
}

func (Setting) TableName() string { return "settings" }
