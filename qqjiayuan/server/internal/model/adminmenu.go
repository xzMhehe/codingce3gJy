package model

// AdminMenu 管理端菜单覆盖配置（菜单维护）
// 以 key 对应 admin-web/src/menu.js 菜单树的节点 key（分组/页面均可覆盖），
// 无记录的节点沿用 menu.js 的默认配置；支持改名/图标/排序/隐藏。
type AdminMenu struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	Key    string `gorm:"type:varchar(50);uniqueIndex" json:"key"` // 菜单节点 key
	Name   string `gorm:"type:varchar(50)" json:"name"`            // 显示名称
	Icon   string `gorm:"type:varchar(60)" json:"icon"`            // 图标 class
	Perm   string `gorm:"type:varchar(50)" json:"perm"`            // 权限码（module:*）
	Sort   int    `gorm:"default:0" json:"sort"`                   // 排序值（同级升序，小的在前）
	Hidden int    `gorm:"default:0" json:"hidden"`                 // 0 显示 1 隐藏
}

func (AdminMenu) TableName() string { return "admin_menus" }
