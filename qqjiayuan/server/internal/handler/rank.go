package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/pkg/resp"
)

type RankHandler struct{ DB *gorm.DB }

// 排行榜：金币/经验/等级/签到
func (h *RankHandler) Top(c *gin.Context) {
	typ := c.DefaultQuery("type", "coins")
	type row struct {
		ID       uint   `json:"id"`
		Nickname string `json:"nickname"`
		Color    string `json:"color"`
		Value    int64  `json:"value"`
	}
	var rows []row
	switch typ {
	case "exp":
		h.DB.Raw(`SELECT id, nickname, color, exp AS value FROM users WHERE exp > 0 ORDER BY exp DESC LIMIT 10`).Scan(&rows)
	case "level":
		h.DB.Raw(`SELECT id, nickname, color, level AS value FROM users ORDER BY level DESC, exp DESC LIMIT 10`).Scan(&rows)
	case "sign":
		h.DB.Raw(`SELECT s.user_id AS id, u.nickname, u.color, COUNT(*) AS value FROM sign_ins s JOIN users u ON u.id = s.user_id GROUP BY s.user_id, u.nickname, u.color ORDER BY value DESC LIMIT 10`).Scan(&rows)
	default:
		h.DB.Raw(`SELECT id, nickname, color, coins AS value FROM users WHERE coins > 0 ORDER BY coins DESC LIMIT 10`).Scan(&rows)
	}
	resp.OK(c, rows)
}
