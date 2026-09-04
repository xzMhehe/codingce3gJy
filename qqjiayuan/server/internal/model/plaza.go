package model

// 广场板块开关（后台可控制显示/隐藏）
type PlazaSection struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Key     string `gorm:"type:varchar(40);uniqueIndex" json:"key"`
	Name    string `gorm:"type:varchar(30)" json:"name"`
	Enabled int    `gorm:"default:1" json:"enabled"` // 1显示 0隐藏
	Sort    int    `gorm:"default:0" json:"sort"`
}

func (PlazaSection) TableName() string { return "plaza_sections" }

// 广场板块清单（顺序即页面顺序；默认全部显示）
var PlazaSectionPresets = []struct {
	Key, Name string
}{
	{"welcome", "欢迎·在线"},
	{"greeting", "问候"},
	{"tongcheng", "同城推荐"},
	{"tv", "家园TV"},
	{"tt", "T台秀"},
	{"joy", "欢乐坊"},
	{"playground", "游乐场"},
	{"newthread", "最新发帖"},
	{"newreply", "最新回帖"},
	{"channels", "频道(公共/同城/家族)"},
	{"chat", "聊天大厅"},
	{"service", "社区服务"},
	{"dynamics", "用户动态"},
	{"search", "搜搜"},
}
