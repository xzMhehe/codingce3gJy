package handler

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/pkg/authutil"
	"qqjiayuan/server/pkg/resp"
)

type RankHandler struct {
	DB     *gorm.DB
	Secret string
}

// 综合排行榜（复刻参考站 /bbs/gov/top.html）
// 榜型：月发 topic / 月回 reply / 超Q noble / G币 coins / 元宝 yuanbao / 论坛 exp / 家园 home / 等级 level
func (h *RankHandler) Top(c *gin.Context) {
	typ := c.DefaultQuery("type", "noble")
	uid := middleware.GetUID(c)
	// 排行页为公开路由：已登录用户从 token 手动解析 uid 以计算「我的排名」
	if uid == 0 && h.Secret != "" {
		if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			if claims, err := authutil.ParseToken(strings.TrimPrefix(auth, "Bearer "), h.Secret); err == nil {
				uid = claims.UserID
			}
		}
	}
	type row struct {
		ID       uint   `json:"id"`
		Nickname string `json:"nickname"`
		Color    string `json:"color"`
		Value    int64  `json:"value"`
		Unit     string `json:"unit"`
	}
	var rows []row
	var myRank int64
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	switch typ {
	case "exp":
		h.DB.Raw(`SELECT id, nickname, color, exp AS value, '经验值' AS unit FROM users WHERE exp > 0 ORDER BY exp DESC LIMIT 15`).Scan(&rows)
		var myVal int64
		h.DB.Raw(`SELECT exp FROM users WHERE id = ?`, uid).Scan(&myVal)
		rankOf(h.DB, &myRank, myVal, "exp")
	case "level":
		h.DB.Raw(`SELECT id, nickname, color, level AS value, '级' AS unit FROM users ORDER BY level DESC, exp DESC LIMIT 15`).Scan(&rows)
		var m struct {
			Level int64
			Exp   int64
		}
		h.DB.Raw(`SELECT level, exp FROM users WHERE id = ?`, uid).Scan(&m)
		if m.Level > 0 {
			h.DB.Raw(`SELECT COUNT(*) + 1 FROM users WHERE level > ? OR (level = ? AND exp > ?)`, m.Level, m.Level, m.Exp).Scan(&myRank)
		}
	case "yuanbao":
		h.DB.Raw(`SELECT id, nickname, color, yuanbao AS value, '' AS unit FROM users WHERE yuanbao > 0 ORDER BY yuanbao DESC LIMIT 15`).Scan(&rows)
		var myVal int64
		h.DB.Raw(`SELECT yuanbao FROM users WHERE id = ?`, uid).Scan(&myVal)
		rankOf(h.DB, &myRank, myVal, "yuanbao")
	case "noble":
		h.DB.Raw(`SELECT id, nickname, color, noble_exp AS value, CONCAT('贵族', noble, '级') AS unit FROM users WHERE noble_exp > 0 ORDER BY noble_exp DESC LIMIT 15`).Scan(&rows)
		var myVal int64
		h.DB.Raw(`SELECT noble_exp FROM users WHERE id = ?`, uid).Scan(&myVal)
		rankOf(h.DB, &myRank, myVal, "noble_exp")
	case "home":
		h.DB.Raw(`SELECT id, nickname, color, FLOOR(active_days) AS value, '天' AS unit FROM users WHERE active_days > 0 ORDER BY active_days DESC LIMIT 15`).Scan(&rows)
		var myVal int64
		h.DB.Raw(`SELECT FLOOR(active_days) FROM users WHERE id = ?`, uid).Scan(&myVal)
		rankOf(h.DB, &myRank, myVal, "FLOOR(active_days)")
	case "topic":
		h.DB.Raw(`SELECT u.id, u.nickname, u.color, COUNT(t.id) AS value, '帖' AS unit FROM threads t JOIN users u ON u.id = t.user_id WHERE t.created_at >= ? AND t.status = 1 GROUP BY u.id, u.nickname, u.color ORDER BY value DESC LIMIT 15`, monthStart).Scan(&rows)
		var myVal int64
		h.DB.Raw(`SELECT COUNT(*) FROM threads WHERE user_id = ? AND created_at >= ? AND status = 1`, uid, monthStart).Scan(&myVal)
		if myVal > 0 {
			h.DB.Raw(`SELECT COUNT(*) + 1 FROM (SELECT user_id, COUNT(*) AS c FROM threads WHERE created_at >= ? AND status = 1 GROUP BY user_id) t WHERE t.c > ?`, monthStart, myVal).Scan(&myRank)
		}
	case "reply":
		h.DB.Raw(`SELECT u.id, u.nickname, u.color, COUNT(r.id) AS value, '回' AS unit FROM replies r JOIN users u ON u.id = r.user_id WHERE r.created_at >= ? AND r.status = 1 GROUP BY u.id, u.nickname, u.color ORDER BY value DESC LIMIT 15`, monthStart).Scan(&rows)
		var myVal int64
		h.DB.Raw(`SELECT COUNT(*) FROM replies WHERE user_id = ? AND created_at >= ? AND status = 1`, uid, monthStart).Scan(&myVal)
		if myVal > 0 {
			h.DB.Raw(`SELECT COUNT(*) + 1 FROM (SELECT user_id, COUNT(*) AS c FROM replies WHERE created_at >= ? AND status = 1 GROUP BY user_id) t WHERE t.c > ?`, monthStart, myVal).Scan(&myRank)
		}
	default: // coins
		h.DB.Raw(`SELECT id, nickname, color, coins AS value, '' AS unit FROM users WHERE coins > 0 ORDER BY coins DESC LIMIT 15`).Scan(&rows)
		var myVal int64
		h.DB.Raw(`SELECT coins FROM users WHERE id = ?`, uid).Scan(&myVal)
		rankOf(h.DB, &myRank, myVal, "coins")
	}
	if myRank == 0 {
		myRank = 0 // 未上榜/我的值为0，显示 0 名（参考站原样）
	}
	resp.OK(c, gin.H{"list": rows, "my_rank": myRank})
}

// rankOf 计算我的排名：值>我的 人数 +1；我的值为0时排 0
func rankOf(db *gorm.DB, out *int64, myVal int64, field string) {
	if myVal <= 0 {
		*out = 0
		return
	}
	db.Raw("SELECT COUNT(*) + 1 FROM users WHERE "+field+" > ?", myVal).Scan(out)
}
