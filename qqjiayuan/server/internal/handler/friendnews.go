package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/pkg/resp"
)

// 好友新鲜事（对齐诺哈 friend_news.asp）：好友最近发表/回复的帖子动态流
type FriendNewsHandler struct{ DB *gorm.DB }

// 我的好友新鲜事：合并好友的发帖与回帖，按时间倒序
func (h *FriendNewsHandler) List(c *gin.Context) {
	uid := middleware.GetUID(c)
	limit := 20

	// 好友列表（双向：我→TA 或 TA→我 有记录即算）
	var friendIDs []uint
	h.DB.Raw(`
SELECT DISTINCT friend_id FROM friendships WHERE user_id = ? AND status = 1
UNION
SELECT DISTINCT user_id FROM friendships WHERE friend_id = ? AND status = 1`, uid, uid).Scan(&friendIDs)

	if len(friendIDs) == 0 {
		resp.OK(c, gin.H{"list": []gin.H{}, "total": 0})
		return
	}

	// 好友最近发帖
	type newsItem struct {
		ID         uint   `json:"id"`
		Type       string `json:"type"` // thread 发帖 / reply 回帖
		UserID     uint   `json:"user_id"`
		Nickname   string `json:"nickname"`
		Color      string `json:"color"`
		Level      int    `json:"level"`
		ThreadID   uint   `json:"thread_id"`
		Title      string `json:"title"`
		BoardName  string `json:"board_name"`
		CreatedAt  string `json:"created_at"`
	}
	var items []newsItem
	h.DB.Raw(`
(SELECT t.id, 'thread' AS type, u.id AS user_id, u.nickname, u.color, u.level,
        t.id AS thread_id, t.title, IFNULL(b.name,'') AS board_name, t.created_at
 FROM threads t
 JOIN users u ON u.id = t.user_id
 LEFT JOIN boards b ON b.id = t.board_id
 WHERE t.user_id IN ? AND t.status = 1 AND t.audit_status = 1)
UNION ALL
(SELECT r.id, 'reply' AS type, u.id AS user_id, u.nickname, u.color, u.level,
        t.id AS thread_id, t.title, IFNULL(b.name,'') AS board_name, r.created_at
 FROM replies r
 JOIN users u ON u.id = r.user_id
 JOIN threads t ON t.id = r.thread_id AND t.status = 1
 LEFT JOIN boards b ON b.id = t.board_id
 WHERE r.user_id IN ? AND r.status = 1)
ORDER BY created_at DESC LIMIT ?`, friendIDs, friendIDs, limit).Scan(&items)

	resp.OK(c, gin.H{"list": items, "total": len(items)})
}