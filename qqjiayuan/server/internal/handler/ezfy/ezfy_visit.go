// 二战风云 统帅页访问记录
//
// ★ 2026-10-12 功能「访问记录」：别人打开自己的 `games/ezfy?cur=playerinfo` 记一笔访问。
//   信息页(cur=info)「占领野地」下方展示历史访问总量 visit_count，点进来看访问列表。
package ezfy

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// Visits GET /games/ezfy/visits —— 当前玩家的访问列表（被谁看过我的统帅页）
//
// 简约列表：按时间倒序，只保留最近 5 次。
// 点击访问列表里的昵称可再跳进对方的统帅页（复用 openPlayer → playerinfo）。
func (h *EzfyHandler) Visits(c *gin.Context) {
	uid := middleware.GetUID(c)
	var logs []model.EzfyVisitLog
	h.DB.Where("target_uid = ?", uid).Order("id DESC").Limit(5).Find(&logs)
	list := make([]gin.H, 0, len(logs))
	if len(logs) == 0 {
		resp.OK(c, gin.H{"list": list})
		return
	}
	// 一次性取访问者昵称（内存拼 map，杜绝 N+1）
	visitorIds := make([]uint, 0, len(logs))
	for _, l := range logs {
		visitorIds = append(visitorIds, l.VisitorUid)
	}
	nameOf := make(map[uint]string, len(visitorIds)) // 昵称：先按二战档案，再按家园昵称兜底
	var profs []model.EzfyProfile
	h.DB.Where("user_id IN ?", visitorIds).Find(&profs)
	profileNick := make(map[uint]string, len(profs))
	for _, pr := range profs {
		if pr.Nickname != "" {
			profileNick[pr.UserID] = pr.Nickname
		}
	}
	var users []model.User
	h.home().Where("id IN ?", visitorIds).Find(&users)
	for _, uu := range users {
		nameOf[uu.ID] = uu.Nickname
	}
	for _, l := range logs {
		if pn := profileNick[l.VisitorUid]; pn != "" {
			nameOf[l.VisitorUid] = pn
		}
		nick := nameOf[l.VisitorUid]
		if nick == "" {
			nick = "#" + strconv.FormatUint(uint64(l.VisitorUid), 10)
		}
		list = append(list, gin.H{
			"visitor_uid": l.VisitorUid,
			"nickname":    nick,
			"created_at":  l.CreatedAt,
		})
	}
	resp.OK(c, gin.H{"list": list})
}

// recordVisit 记一笔「别人查看了 targetUid 的统帅页」；自己看自己不记，失败不影响主流程。
// 只保留最近 5 次：超出部分物理删除（访问是瞬时行为，无需留存旧记录）。
func (h *EzfyHandler) recordVisit(targetUid, visitorUid uint) {
	if targetUid == 0 || visitorUid == 0 || targetUid == visitorUid {
		return
	}
	// 异步落库：调用方是 /player/:id（本身就有性能要求），不让写日志拖慢响应。
	go func() {
		rec := model.EzfyVisitLog{TargetUid: targetUid, VisitorUid: visitorUid}
		if err := h.DB.Create(&rec).Error; err != nil {
			return
		}
		// 删掉「第 6 新及更旧」的记录，保持该玩家最多 5 条
		var oldID uint
		if err := h.DB.Model(&model.EzfyVisitLog{}).
			Where("target_uid = ?", targetUid).
			Order("id DESC").Offset(5).Limit(1).Pluck("id", &oldID).Error; err == nil && oldID > 0 {
			h.DB.Delete(&model.EzfyVisitLog{}, "target_uid = ? AND id <= ?", targetUid, oldID)
		}
	}()
}