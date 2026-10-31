package ezfy

import (
	"log"
	"sort"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 军团日志（★ 2026-10-30 用户需求）
//
// 用户端「军团」页新增「军团日志」tab：记录**进团 / 出团 / 任命（含撤职）**，
// 与「军团宣战」并入「军团外交」是同一轮改动。
// 口径：只增不删（军团解散时随军团一起清理，见 ezfy_admin_purge.go），按时间倒序分页。

const (
	ezfyCorpsLogJoin    = 1 // 进团
	ezfyCorpsLogLeave   = 2 // 出团
	ezfyCorpsLogAppoint = 3 // 任命
	ezfyCorpsLogRevoke  = 4 // 撤职
)

// ezfyCorpsLogKindName 日志类型中文名（用户端直接展示）
func ezfyCorpsLogKindName(kind int) string {
	switch kind {
	case ezfyCorpsLogJoin:
		return "进团"
	case ezfyCorpsLogLeave:
		return "出团"
	case ezfyCorpsLogAppoint:
		return "任命"
	case ezfyCorpsLogRevoke:
		return "撤职"
	}
	return "其他"
}

// ezfyCorpsTitleWeight 军团成员排序权重（★「按职位排序」唯一实现）：
//
//	0 军团长 > 1 副团长 > 2 参谋长 > 3 普通成员。
//
// ★ 职位存的是中文（`ezfy_corps_member.title`，空/「成员」= 普通成员），
// 所以按权重排序必须在 Go 里做，不能靠 SQL 字符串排序。
func ezfyCorpsTitleWeight(m model.EzfyCorpsMember) int {
	if m.IsLeader == 1 {
		return 0
	}
	switch m.Title {
	case ezfyCorpsTitleVice:
		return 1
	case ezfyCorpsTitleChief:
		return 2
	}
	return 3
}

// ezfyCorpsMembersSorted 成员按「职位 → 军团积分高→低 → 加入早→晚」排序（原地排）。
func ezfyCorpsMembersSorted(members []model.EzfyCorpsMember) {
	sort.SliceStable(members, func(i, j int) bool {
		wi, wj := ezfyCorpsTitleWeight(members[i]), ezfyCorpsTitleWeight(members[j])
		if wi != wj {
			return wi < wj
		}
		if members[i].Points != members[j].Points {
			return members[i].Points > members[j].Points
		}
		return members[i].ID < members[j].ID
	})
}

// corpsLogAdd 记一条军团日志。
//
// ★ 写失败**不影响主流程**（进团/任命照常成功），但必须留日志便于排查 ——
// 与 ezfyActCaptureLogAdd 同口径（表没建出来时静默失败会让「军团日志」永远空白）。
func (h *EzfyHandler) corpsLogAdd(corpsId, userId, actorId uint, kind int, title, note string) {
	if corpsId == 0 || userId == 0 {
		return
	}
	if err := h.DB.Create(&model.EzfyCorpsLog{
		CorpsId: corpsId, UserId: userId, ActorId: actorId,
		Kind: kind, Title: title, Note: note,
	}).Error; err != nil {
		log.Printf("【严重】军团日志写入失败 corps=%d user=%d kind=%d: %v", corpsId, userId, kind, err)
	}
}

// CorpsLogs GET /corps/logs?page=&size= —— 本军团日志（时间倒序，分页）
func (h *EzfyHandler) CorpsLogs(c *gin.Context) {
	uid := middleware.GetUID(c)
	var mb model.EzfyCorpsMember
	if err := h.DB.Where("user_id = ?", uid).First(&mb).Error; err != nil {
		resp.OK(c, gin.H{"logs": []gin.H{}, "in_corps": false, "total": 0, "page": 1, "size": 20})
		return
	}
	page, offset, size := pageOf(c, 20)
	var total int64
	h.DB.Model(&model.EzfyCorpsLog{}).Where("corps_id = ?", mb.CorpsId).Count(&total)
	var rows []model.EzfyCorpsLog
	h.DB.Where("corps_id = ?", mb.CorpsId).Order("id DESC").
		Offset(offset).Limit(size).Find(&rows)

	// ★ 昵称批量：当事人 + 操作人合成一个 IN（一页最多 20 条 → 最多 40 个 id，2 条 SQL）
	idSet := map[uint]bool{}
	ids := make([]uint, 0, len(rows)*2)
	for _, l := range rows {
		for _, id := range []uint{l.UserId, l.ActorId} {
			if id > 0 && !idSet[id] {
				idSet[id] = true
				ids = append(ids, id)
			}
		}
	}
	nameOf := make(map[uint]string, len(ids))
	if len(ids) > 0 {
		var profs []model.EzfyProfile
		h.DB.Select("user_id", "nickname").Where("user_id IN ?", ids).Find(&profs)
		for _, p := range profs {
			if p.Nickname != "" {
				nameOf[p.UserID] = p.Nickname
			}
		}
		var users []model.User
		h.home().Select("id", "nickname").Where("id IN ?", ids).Find(&users)
		for _, u := range users {
			if nameOf[u.ID] == "" {
				nameOf[u.ID] = u.Nickname
			}
		}
	}

	out := make([]gin.H, 0, len(rows))
	for _, l := range rows {
		out = append(out, gin.H{
			"id": l.ID, "kind": l.Kind, "kind_name": ezfyCorpsLogKindName(l.Kind),
			"user_id": l.UserId, "user_name": nameOf[l.UserId],
			"actor_id": l.ActorId, "actor_name": nameOf[l.ActorId],
			"title": l.Title, "note": l.Note,
			"ts": l.CreatedAt.UnixMilli(), // 前端 fmtTime 格式化
		})
	}
	resp.OK(c, gin.H{"logs": out, "in_corps": true, "total": total, "page": page, "size": size})
}
