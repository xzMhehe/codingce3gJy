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
// 简约列表：按时间倒序，只展示最近 5 次（总量 visit_count 见信息页，全量累加）。
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
//
// ★ 2026-10-14 修复「访问记录一直是 5」：原来每次新记录后都会把「第 6 新及更旧」的记录
//   物理删除，而信息页顶部的 visit_count 是按现存行数 Count 统计的 → 总量永远卡在 5，
//   涨不上去。现在表内全量保留（访问列表接口 Visits 仍只取最近 5 条展示），
//   visit_count 即为真实的历史访问总量（累加）。
func (h *EzfyHandler) recordVisit(targetUid, visitorUid uint) {
	if targetUid == 0 || visitorUid == 0 || targetUid == visitorUid {
		return
	}
	// 异步落库：调用方是 /player/:id（本身就有性能要求），不让写日志拖慢响应。
	go func() {
		if err := h.DB.Create(&model.EzfyVisitLog{TargetUid: targetUid, VisitorUid: visitorUid}).Error; err != nil {
			return
		}
	}()
}