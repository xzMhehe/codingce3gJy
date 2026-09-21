package model

import "time"

// IPBan IP 封禁（管理端「在线查看」对在线用户/游客的 IP 一键封禁；存在记录即视为封禁）
type IPBan struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	IP        string    `gorm:"type:varchar(45);uniqueIndex:uk_ip_ban" json:"ip"`
	Reason    string    `gorm:"type:varchar(255)" json:"reason"`
	AdminID   uint      `json:"admin_id"`
	AdminName string    `gorm:"type:varchar(50)" json:"admin_name"`
	CreatedAt time.Time `json:"created_at"`
}

func (IPBan) TableName() string { return "ip_bans" }
