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
//
// ★ 2026-10-25 用户需求「系统邮件可以删除，发错了 / 发错道具了要能一起删除」：
//   列表补 `has_attach / claimed / expire_at / batch_no / attaches / batch_total`，
//   管理端才能看出「哪封带了道具、有没有被领走、这一批一共发了多少封」，
//   删除入口见 `AdminEzfyMailSystemDelete`（邮件 + 附件一起删）。
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
		// 用 home()（单库模式回落 DB）：与其它管理端一致，避免 HomeDB 未配置时 panic
		a.home().Select("id, nickname").Where("id IN ?", ids).Find(&us)
		for _, u := range us {
			names[u.ID] = u.Nickname
		}
	}
	// ★ 2026-10-25 附件与批次：各一次 SQL，避免逐行 N+1
	mailIDs := make([]uint, 0, len(rows))
	batchNos := make([]string, 0, len(rows))
	for _, m := range rows {
		mailIDs = append(mailIDs, m.ID)
		if m.BatchNo != "" {
			batchNos = append(batchNos, m.BatchNo)
		}
	}
	attachOf := ezfyMailAttachMap(a.DB, mailIDs)
	batchTotal := map[string]int64{}
	if len(batchNos) > 0 {
		var bcs []struct {
			BatchNo string
			N       int64
		}
		a.DB.Model(&model.EzfyMail{}).
			Select("batch_no, COUNT(*) AS n").
			Where("sender_id = 0 AND batch_no IN ?", batchNos).
			Group("batch_no").Scan(&bcs)
		for _, b := range bcs {
			batchTotal[b.BatchNo] = b.N
		}
	}
	out := make([]gin.H, 0, len(rows))
	for _, m := range rows {
		as := attachOf[m.ID]
		views := make([]gin.H, 0, len(as))
		for _, x := range as {
			views = append(views, ezfyMailAttachView(x))
		}
		out = append(out, gin.H{"id": m.ID, "receiver_id": m.ReceiverId,
			"receiver_name": names[m.ReceiverId], "subject": m.Subject,
			"content": m.Content, "created_at": m.CreatedAt,
			"has_attach": m.HasAttach, "claimed": m.Claimed, "expire_at": m.ExpireAt,
			"batch_no": m.BatchNo, "batch_total": batchTotal[m.BatchNo],
			"attaches": views})
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
	// ★ 2026-10-25 群发也要写批次号：一次发送的所有邮件共用同一 batch_no，
	//   发错时管理端才能「删本批」一把删掉（否则几百封只能逐条删，跨页还删不干净）。
	batchNo := ezfyNewBatchNo()
	for _, rid := range receivers {
		mails = append(mails, model.EzfyMail{
			SenderId: 0, ReceiverId: rid, SenderName: "系统",
			Subject: strings.TrimSpace(req.Subject), Content: content,
			BatchNo: batchNo, CreatedAt: now,
		})
	}
	a.DB.Create(&mails)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("系统邮件已发送给 %d 名玩家", len(receivers)), "sent": len(receivers)})
}

// AdminEzfyMailSystemDelete POST /admin/ezfy-mails/system/delete {ids:[] | batch_no}
//
// ★ 2026-10-25 用户需求（原话）：
//
//	「系统邮件 可以删除，有时候可能发错了、或者发错道具了要能一起删除」
//
// 所以删除是**邮件 + 附件一起物理删除**，两条路：
//   - ids     按勾选删（前端多选，单条删除也走它）
//   - batch_no 按批次删：一次「发放道具」几百上千封共用同一个批次号，
//     逐条勾选不现实，发错时按批次一把删掉才是可用功能。
//
// 硬规则：
//  1. **只删 `sender_id = 0`（系统邮件）**——玩家之间互发的邮件绝不能从这里删掉。
//  2. 事务内删 `ezfy_mail_attach` + `ezfy_mail`，避免出现「邮件没了、附件行还在」的孤儿数据。
//  3. 已领取（claimed=1）的附件**收不回来**（道具已进玩家背包/宝箱/钻石），
//     接口如实统计并在 msg 里说清，让管理员知道要另行处理。
func (a *EzfyAdmin) AdminEzfyMailSystemDelete(c *gin.Context) {
	var in struct {
		IDs     []uint `json:"ids"`
		BatchNo string `json:"batch_no"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	batchNo := strings.TrimSpace(in.BatchNo)

	var ids []uint
	if batchNo != "" {
		// 批次删除：先按批次号把该批全部邮件 id 捞出来（同一批次同一次发放，条数可控）
		if err := a.DB.Model(&model.EzfyMail{}).
			Where("sender_id = 0 AND batch_no = ?", batchNo).
			Pluck("id", &ids).Error; err != nil {
			resp.ServerError(c, err)
			return
		}
	} else {
		// 勾选删除：去重 + 上限保护（防误传巨大数组打满 SQL）
		seen := map[uint]bool{}
		for _, id := range in.IDs {
			if id == 0 || seen[id] {
				continue
			}
			seen[id] = true
			ids = append(ids, id)
		}
		if len(ids) > 2000 {
			resp.ParamError(c, "一次最多删除 2000 封，请分批操作")
			return
		}
	}
	if len(ids) == 0 {
		resp.NotFound(c, "没有找到可删除的系统邮件")
		return
	}

	// 再过滤一次：只留系统邮件（batch 分支已带条件，这里对 ids 分支是必须的）
	var mails []model.EzfyMail
	if err := a.DB.Select("id, receiver_id, has_attach, claimed").
		Where("id IN ? AND sender_id = 0", ids).Find(&mails).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	if len(mails) == 0 {
		resp.NotFound(c, "没有找到可删除的系统邮件")
		return
	}
	realIDs := make([]uint, 0, len(mails))
	claimedMail := 0     // 已领取的邮件数（附件收不回）
	receiverSet := map[uint]bool{}
	for _, m := range mails {
		realIDs = append(realIDs, m.ID)
		receiverSet[m.ReceiverId] = true
		if m.HasAttach == 1 && m.Claimed == 1 {
			claimedMail++
		}
	}

	// 统计附件（一行一件，Num 是数量）：已领取的那封视为已发出，未领取的随邮件撤回
	var as []model.EzfyMailAttach
	if err := a.DB.Where("mail_id IN ?", realIDs).Find(&as).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	claimedSet := map[uint]bool{}
	for _, m := range mails {
		if m.Claimed == 1 {
			claimedSet[m.ID] = true
		}
	}
	revoked, kept := 0, 0
	for _, x := range as {
		if claimedSet[x.MailID] {
			kept++
		} else {
			revoked++
		}
	}

	// 事务：先删附件再删邮件（分块，避免超长 IN）
	tx := a.DB.Begin()
	if tx.Error != nil {
		resp.ServerError(c, tx.Error)
		return
	}
	for i := 0; i < len(realIDs); i += 500 {
		j := i + 500
		if j > len(realIDs) {
			j = len(realIDs)
		}
		chunk := realIDs[i:j]
		if err := tx.Where("mail_id IN ?", chunk).Delete(&model.EzfyMailAttach{}).Error; err != nil {
			tx.Rollback()
			resp.ServerError(c, fmt.Errorf("删除邮件附件失败：%w", err))
			return
		}
		if err := tx.Where("id IN ? AND sender_id = 0", chunk).Delete(&model.EzfyMail{}).Error; err != nil {
			tx.Rollback()
			resp.ServerError(c, fmt.Errorf("删除系统邮件失败：%w", err))
			return
		}
	}
	if err := tx.Commit().Error; err != nil {
		resp.ServerError(c, err)
		return
	}

	msg := fmt.Sprintf("已删除 %d 封系统邮件（涉及 %d 名玩家）", len(realIDs), len(receiverSet))
	if revoked > 0 {
		msg += fmt.Sprintf("，撤回未领取附件 %d 件", revoked)
	}
	if kept > 0 {
		msg += fmt.Sprintf("；另有 %d 件附件已被领取（%d 封），已到玩家手中无法收回", kept, claimedMail)
	}
	resp.OK(c, gin.H{
		"msg": msg, "mails": len(realIDs), "players": len(receiverSet),
		"attach_revoked": revoked, "attach_kept": kept,
	})
}
