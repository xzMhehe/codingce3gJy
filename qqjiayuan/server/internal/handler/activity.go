package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// ActivityHandler 活动专区（参考诺哈三代 topic_active.asp：活动帖=帖子 active 标记）
// 活动与精华/置顶/公告/推荐并列，是帖子的一种属性，由版主/管理员在帖子管理里设为活动帖。
type ActivityHandler struct{ DB *gorm.DB }

type activityRow struct {
	model.Thread
	Nickname  string `json:"nickname"`
	Color     string `json:"color"`
	Avatar    string `json:"avatar"`
	BoardName string `json:"board_name"`
	IsOwner   bool   `json:"is_owner"`
}

// List 活动专区：列出全部活动帖（active=1 AND status=1），10条/页，按 id 倒序
// 布局对齐诺哈：序号.标题 (头像 发帖人:回复/阅读)，分页 下页.上页 (第x/y页/共z条记录)
func (h *ActivityHandler) List(c *gin.Context) {
	page, size := pageParams(c, 10)
	q := h.DB.Model(&model.Thread{}).Where("is_active = 1 AND status = 1")
	var total int64
	q.Count(&total)
	var list []model.Thread
	q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&list)

	out := make([]activityRow, 0, len(list))
	for _, th := range list {
		row := activityRow{Thread: th}
		var u model.User
		h.DB.Select("nickname,color,avatar").First(&u, th.UserID)
		row.Nickname = u.Nickname
		row.Color = u.Color
		row.Avatar = u.Avatar
		if th.BoardID > 0 {
			var b model.Board
			h.DB.Select("name").First(&b, th.BoardID)
			row.BoardName = b.Name
		}
		out = append(out, row)
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}