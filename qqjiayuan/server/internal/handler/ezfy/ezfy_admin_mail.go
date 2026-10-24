package ezfy

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 —— 管理端「系统邮件」
//
// ★ 2026-10-13 邮箱改造：管理端可发系统邮件给全体或指定玩家，玩家在「邮箱 → 系统邮件」收到。
//   GET  /admin/ezfy-mails/system?target_id=   最近系统邮件列表
//   POST /admin/ezfy-mails/system {target_id(0=全体), subject, content}  发送系统邮件

// AdminEzfyMailSystemList GET /admin/ezfy-mails/system?target_id=&page=&size=
//
// ★ 2026-10-24 用户需求「系统邮件 做下分页」：原来固定 Limit(50) 不分页，记录一多就看不到历史。
//   改为标准分页（page/size，返回 total），前端翻页查看。
func (a *EzfyAdmin) AdminEzfyMailSystemList(c *gin.Context) {
	page, offset, size := pageOf(c, 20)
	var rows []model.EzfyMail
	q := a.DB.Model(&model.EzfyMail{}).Where("sender_id = 0")
	if v := c.Query("target_id"); v != "" && v != "0" {
		q = q.Where("receiver_id = ?", v)
	}
	var total int64
	q.Count(&total)
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	// 批量取收件人昵称（家园库）
	names := map[uint]string{}
	var ids []uint
	for _, m := range rows {
		if m.ReceiverId > 0 {
			ids = append(ids, m.ReceiverId)
		}
	}
	if len(ids) > 0 {
		var us []model.User
		a.HomeDB.Select("id, nickname").Where("id IN ?", ids).Find(&us)
		for _, u := range us {
			names[u.ID] = u.Nickname
		}
	}
	out := make([]gin.H, 0, len(rows))
	for _, m := range rows {
		out = append(out, gin.H{"id": m.ID, "receiver_id": m.ReceiverId,
			"receiver_name": names[m.ReceiverId], "subject": m.Subject,
			"content": m.Content, "created_at": m.CreatedAt})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyMailSystemSend POST /admin/ezfy-mails/system {target_id(0=全体), subject, content}
func (a *EzfyAdmin) AdminEzfyMailSystemSend(c *gin.Context) {
	var req struct {
		TargetID uint   `json:"target_id"` // 0 = 全体
		Subject  string `json:"subject"`
		Content  string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		resp.ParamError(c, "请填写邮件内容")
		return
	}
	if r := []rune(content); len(r) > 500 {
		resp.ParamError(c, "邮件内容最多 500 字")
		return
	}
	if r := []rune(req.Subject); len(r) > 20 {
		resp.ParamError(c, "邮件主题最多 20 字")
		return
	}
	if filtered, blocked := ezfyFilterChat(content); blocked {
		resp.Forbidden(c, "内容包含敏感词，请修改后再发")
		return
	} else {
		content = filtered
	}

	var receivers []uint
	if req.TargetID > 0 {
		var n int64
		a.DB.Model(&model.EzfyProfile{}).Where("user_id = ?", req.TargetID).Count(&n)
		if n == 0 {
			resp.NotFound(c, "目标玩家不存在")
			return
		}
		receivers = append(receivers, req.TargetID)
	} else {
		var profs []model.EzfyProfile
		a.DB.Select("user_id").Find(&profs)
		for _, p := range profs {
			if p.UserID > 0 {
				receivers = append(receivers, p.UserID)
			}
		}
		if len(receivers) == 0 {
			resp.ParamError(c, "当前没有玩家，无法群发系统邮件")
			return
		}
	}
	mails := make([]model.EzfyMail, 0, len(receivers))
	now := time.Now()
	for _, rid := range receivers {
		mails = append(mails, model.EzfyMail{
			SenderId: 0, ReceiverId: rid, SenderName: "系统",
			Subject: strings.TrimSpace(req.Subject), Content: content,
			CreatedAt: now,
		})
	}
	a.DB.Create(&mails)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("系统邮件已发送给 %d 名玩家", len(receivers)), "sent": len(receivers)})
}
