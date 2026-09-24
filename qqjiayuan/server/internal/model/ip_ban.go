package model

import "time"

// IPBan IP 封禁（管理端「在线查看」对在线用户/游客的 IP 一键封禁；存在记录即视为封禁）
type IPBan struct {
	ID        uint      `gorm:"primaryKey;comment:主键ID" json:"id"`
	IP        string    `gorm:"type:varchar(45);uniqueIndex:uk_ip_ban;comment:IP地址" json:"ip"`
	Reason    string    `gorm:"type:varchar(255);comment:原因" json:"reason"`
	AdminID   uint      `gorm:"comment:管理员ID" json:"admin_id"`
	AdminName string    `gorm:"type:varchar(50);comment:管理员名称" json:"admin_name"`
	CreatedAt time.Time `gorm:"comment:创建时间" json:"created_at"`
}

func (IPBan) TableName() string { return "ip_bans" }
