package handler

import (
	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// numMap 批量取家园号码（管理端列表展示用；靓号转换后随 username 动态变化）
func numMap(db *gorm.DB, uidSet map[uint]bool) map[uint]string {
	m := map[uint]string{}
	if len(uidSet) == 0 {
		return m
	}
	ids := make([]uint, 0, len(uidSet))
	for k := range uidSet {
		ids = append(ids, k)
	}
	var us []model.User
	db.Select("id,username").Where("id IN ?", ids).Find(&us)
	for _, u := range us {
		m[u.ID] = u.Username
	}
	return m
}
