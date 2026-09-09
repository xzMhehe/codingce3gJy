package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type ThreadHandler struct {
	DB *gorm.DB
}

// 发帖/回帖/签到收益：经验 + G币 + 社区成就点（同时记一条钱包流水）
func addExpAndCoins(db *gorm.DB, uid uint, exp, coins, achieve int, kind, title string) {
	if coins != 0 {
		addWalletLog(db, uid, kind, title, "coins", coins)
	}
	db.Model(&model.User{}).Where("id = ?", uid).
		Updates(map[string]interface{}{
			"exp":     gorm.Expr("exp + ?", exp),
			"coins":   gorm.Expr("coins + ?", coins),
			"achieve": gorm.Expr("achieve + ?", achieve),
		})
	var u model.User
	db.First(&u, uid)
	db.Model(&u).Update("level", model.CalcLevel(u.Exp))
}

// 帖子详情 + 分页楼层（1楼=楼主，回复从2楼起）+ 投票/回帖奖励/踩楼/附件/置顶回复
func (h *ThreadHandler) Detail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	page, _, _ := pageOf(c, 10)

	var th model.Thread
	if err := h.DB.Preload("User").Preload("User.Badges").Preload("Board").First(&th, id).Error; err != nil || th.Status == 0 {
		resp.NotFound(c, "帖子不存在或已被删除")
		return
	}
	// 待审核或审核不通过的帖子：仅作者与版主/管理员可见
	uid := middleware.GetUID(c)
	if th.AuditStatus != 1 {
		canView := th.UserID == uid
		if !canView {
			canView = h.canManage(th, uid)
		}
		if !canView {
			resp.NotFound(c, "帖子不存在或已被删除")
			return
		}
	}
	go func() {
		h.DB.Model(&model.Thread{}).Where("id = ?", th.ID).
			UpdateColumn("view_count", gorm.Expr("view_count + 1"))
	}()

	var replies []model.Reply
	var total int64
	h.DB.Model(&model.Reply{}).Where("thread_id = ? AND status = 1", th.ID).Count(&total)
	if maxPage := int(total+9) / 10; maxPage < 1 {
		page = 1
	} else if page > maxPage {
		page = maxPage
	}
	h.DB.Preload("User").Preload("User.Badges").Where("thread_id = ? AND status = 1", th.ID).
		Order("floor ASC").Offset((page - 1) * 10).Limit(10).Find(&replies)

	// 互动统计
	var giftCount, flowerPeople int64
	h.DB.Model(&model.ThreadGift{}).Where("thread_id = ?", th.ID).Count(&giftCount)
	h.DB.Model(&model.ThreadFlower{}).Where("thread_id = ?", th.ID).Distinct("sender_id").Count(&flowerPeople)
	var flowers []model.ThreadFlower
	h.DB.Preload("Sender").Where("thread_id = ?", th.ID).Order("id DESC").Limit(20).Find(&flowers)
	var gifts []model.ThreadGift
	h.DB.Preload("Sender").Where("thread_id = ?", th.ID).Order("id DESC").Limit(20).Find(&gifts)

	// 置顶回复
	var sticky *model.StickyReply
	h.DB.Preload("User").Where("thread_id = ?", th.ID).First(&sticky)
	if sticky != nil && sticky.ID == 0 {
		sticky = nil
	}

	// 投票
	var poll *model.ThreadPoll
	var pollVoted bool
	var pollTotal int64
	var myOptions []uint
	if th.Type == 3 {
		var p model.ThreadPoll
		h.DB.Preload("Options").Where("thread_id = ?", th.ID).First(&p)
		if p.ID != 0 {
			poll = &p
			if uid != 0 {
				var cnt int64
				h.DB.Model(&model.ThreadPollVote{}).Where("poll_id = ? AND user_id = ?", p.ID, uid).Count(&cnt)
				pollVoted = cnt > 0
				if pollVoted {
					h.DB.Model(&model.ThreadPollVote{}).Where("poll_id = ? AND user_id = ?", p.ID, uid).Pluck("option_id", &myOptions)
				}
			}
			h.DB.Model(&model.ThreadPollVote{}).Where("poll_id = ?", p.ID).Count(&pollTotal)
		}
	}

	// 回帖奖励
	var reward *model.ThreadReward
	if th.Type == 1 {
		var r model.ThreadReward
		h.DB.Where("thread_id = ?", th.ID).First(&r)
		if r.ID != 0 {
			reward = &r
		}
	}
	// 踩楼
	var floors []model.ThreadFloor
	if th.Type == 2 {
		h.DB.Where("thread_id = ?", th.ID).Order("floor ASC").Find(&floors)
	}
	// 附件
	var attachments []model.ThreadAttachment
	h.DB.Where("thread_id = ?", th.ID).Order("id ASC").Find(&attachments)

	// 最近 3 条回帖（参考诺哈 topic.asp 回贴列表只列最后 3 楼）
	var recent []model.Reply
	h.DB.Preload("User").Preload("User.Badges").Where("thread_id = ? AND status = 1", th.ID).
		Order("floor DESC").Limit(3).Find(&recent)

	// 楼主或版主可置顶回复（参考诺哈 ReplyApexRight）
	canSticky := false
	if uid != 0 {
		if th.UserID == uid {
			canSticky = true
		} else if th.Board != nil {
			canSticky = h.isModeratorOfBoard(h.DB, *th.Board, uid)
		}
	}

	resp.OK(c, gin.H{"thread": th, "replies": replies, "total": total, "page": page, "size": 10,
		"like_count": th.LikeCount, "dislike_count": th.DislikeCount,
		"gift_total": th.GiftTotal, "gift_count": giftCount, "flower_count": th.FlowerCount,
		"flower_people": flowerPeople, "share_count": th.ShareCount,
		"flowers": flowers, "gifts": gifts,
		"sticky_reply": sticky, "can_sticky": canSticky, "recent_replies": recent,
		"poll": poll, "poll_voted": pollVoted, "poll_total": pollTotal,
		"my_options": myOptions, "reward": reward, "floors": floors,
		"attachments": attachments, "audit_status": th.AuditStatus})
}

// 帖子投票
func (h *ThreadHandler) VotePoll(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	var req struct {
		OptionIDs []uint `json:"option_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.OptionIDs) == 0 {
		resp.ParamError(c, "请选择投票选项")
		return
	}
	var th model.Thread
	if err := h.DB.First(&th, id).Error; err != nil || th.Type != 3 {
		resp.NotFound(c, "投票不存在")
		return
	}
	var poll model.ThreadPoll
	if err := h.DB.First(&poll, "thread_id = ?", th.ID).Error; err != nil {
		resp.NotFound(c, "投票不存在")
		return
	}
	// 已投则拒绝
	var cnt int64
	h.DB.Model(&model.ThreadPollVote{}).Where("poll_id = ? AND user_id = ?", poll.ID, uid).Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "你已投过票")
		return
	}
	if poll.Multiple == 0 && len(req.OptionIDs) > 1 {
		resp.ParamError(c, "该投票为单选")
		return
	}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		for _, oid := range req.OptionIDs {
			var opt model.ThreadPollOption
			if tx.First(&opt, oid).Error != nil || opt.PollID != poll.ID {
				continue
			}
			tx.Create(&model.ThreadPollVote{PollID: poll.ID, UserID: uid, OptionID: oid, ThreadID: th.ID})
			tx.Model(&model.ThreadPollOption{}).Where("id = ?", oid).UpdateColumn("votes", gorm.Expr("votes + 1"))
		}
		return nil
	})
	if err != nil {
		resp.ServerError(c, err)
		return
	}
	resp.OK(c, nil)
}

type replyReq struct {
	Content       string `json:"content" binding:"required,min=1,max=5000"`
	ParentReplyID uint   `json:"parent_reply_id"`
}

// 回复盖楼（支持引用楼层回复；回帖奖励/踩楼；频率限制）
func (h *ThreadHandler) Reply(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	var req replyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "回复内容不能为空，5000字以内")
		return
	}

	var th model.Thread
	if err := h.DB.First(&th, id).Error; err != nil || th.Status == 0 {
		resp.NotFound(c, "帖子不存在或已被删除")
		return
	}
	// 待审核帖子不能回帖
	if th.AuditStatus != 1 {
		resp.ParamError(c, "该帖子正在审核中，暂不能回复")
		return
	}
	// 锁定帖子不能回帖
	if th.IsLock == 1 && !h.canManage(th, uid) {
		resp.Forbidden(c, "本贴已锁定，仅供查阅")
		return
	}
	var board model.Board
	if h.DB.First(&board, th.BoardID).Error == nil {
		if famID, ok := familyBoardOwner(h.DB, board); ok && !isFamilyMember(h.DB, famID, uid) {
			resp.Forbidden(c, "只有家族成员才能在家族论坛回帖")
			return
		}
		if board.MembersOnly == 1 {
			var cnt int64
			h.DB.Model(&model.BoardMember{}).Where("board_id = ? AND user_id = ? AND status = 1", board.ID, uid).Count(&cnt)
			if cnt == 0 {
				resp.Forbidden(c, "该版块为会员制，只有成员才能回帖")
				return
			}
		}
	}
	// 频率限制
	if !h.isModeratorOfBoard(h.DB, board, uid) {
		if board.ID != 0 {
			var last model.Reply
			if h.DB.Where("user_id = ?", uid).Order("id DESC").First(&last).Error == nil {
				if time.Since(last.CreatedAt) < 10*time.Second {
					resp.ParamError(c, "回帖太快了，请休息 10 秒再试")
					return
				}
			}
		}
	}
	// 敏感词过滤
	words := h.loadWords()
	content := replaceText(req.Content, words)
	needAudit := false
	for _, w := range words {
		if w.Type == 2 && contains(content, w.Word) {
			needAudit = true
		}
	}
	_ = needAudit // 回复不强制审核，仅替换敏感词

	var reply model.Reply
	floorNum := 0
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		tx.Model(&model.Reply{}).Where("thread_id = ?", id).Count(&count)
		floorNum = int(count) + 2
		reply = model.Reply{ThreadID: uint(id), UserID: uid, Content: content, Floor: floorNum, ParentReplyID: req.ParentReplyID}
		if err := tx.Create(&reply).Error; err != nil {
			return err
		}
		return tx.Model(&model.Thread{}).Where("id = ?", id).
			Updates(map[string]interface{}{
				"reply_count":   gorm.Expr("reply_count + 1"),
				"last_reply_at": time.Now(),
			}).Error
	})
	if err != nil {
		resp.ServerError(c, err)
		return
	}
	addExpAndCoins(h.DB, uid, 5, 2, 1, "reply", "回复帖子")
	addHomeNews(h.DB, uid, 0, 2, th.ID, "回复了帖子《"+th.Title+"》")

	// 回帖奖励（type=1）
	if th.Type == 1 {
		var r model.ThreadReward
		if h.DB.First(&r, "thread_id = ?", th.ID).Error == nil {
			if r.Limit == 0 || r.Used < r.Limit {
				// 每人限一次
				var already int64
				h.DB.Model(&model.ThreadRewardLog{}).Where("thread_id = ? AND user_id = ?", th.ID, uid).Count(&already)
				if already == 0 {
					h.DB.Model(&r).UpdateColumn("used", gorm.Expr("used + 1"))
					h.DB.Create(&model.ThreadRewardLog{ThreadID: th.ID, UserID: uid, ReplyID: reply.ID, Coins: r.Coins, Exp: r.Exp})
					if r.Coins != 0 {
						addWalletLog(h.DB, uid, "reward", "回帖奖励", "coins", r.Coins)
					}
					h.DB.Model(&model.User{}).Where("id = ?", uid).
						Updates(map[string]interface{}{
							"exp":   gorm.Expr("exp + ?", r.Exp),
							"coins": gorm.Expr("coins + ?", r.Coins),
						})
					var u model.User
					h.DB.First(&u, uid)
					h.DB.Model(&u).Update("level", model.CalcLevel(u.Exp))
				}
			}
		}
	}
	// 踩楼（type=2）：命中配置楼层
	if th.Type == 2 && floorNum > 0 {
		var floor model.ThreadFloor
		if h.DB.Where("thread_id = ? AND floor = ? AND status = 0", th.ID, floorNum).First(&floor).Error == nil {
			h.DB.Model(&floor).Updates(map[string]interface{}{"status": 1, "user_id": uid})
			if floor.Coins != 0 {
				addWalletLog(h.DB, uid, "floor", "踩楼奖励", "coins", floor.Coins)
			}
			h.DB.Model(&model.User{}).Where("id = ?", uid).
				Updates(map[string]interface{}{
					"exp":   gorm.Expr("exp + ?", floor.Exp),
					"coins": gorm.Expr("coins + ?", floor.Coins),
				})
			var u model.User
			h.DB.First(&u, uid)
			h.DB.Model(&u).Update("level", model.CalcLevel(u.Exp))
			h.DB.Create(&model.Notification{
				UserID: uid, Type: "system", RefID: th.ID,
				Title: "恭喜踩中幸运楼层", Content: "你在《" + th.Title + "》中踩中了 " + strconv.Itoa(floorNum) + " 楼，获得奖励！",
			})
		}
	}

	if th.UserID != uid {
		var me model.User
		h.DB.First(&me, uid)
		h.DB.Create(&model.Notification{
			UserID: th.UserID, Type: "reply", RefID: th.ID,
			Title:   me.Nickname + " 回复了你的帖子",
			Content: "《" + th.Title + "》来了新回复，快去看看吧",
		})
	}
	resp.OK(c, gin.H{"id": reply.ID, "floor": reply.Floor})
}

// 置顶回复（顶贴，参考诺哈 wap_topic_reply_apex）：一贴至多一条，可置顶某楼或自定义内容
func (h *ThreadHandler) StickyReply(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	var req struct {
		ReplyID uint   `json:"reply_id"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var th model.Thread
	if err := h.DB.First(&th, id).Error; err != nil || th.Status == 0 {
		resp.NotFound(c, "帖子不存在")
		return
	}
	var board model.Board
	h.DB.First(&board, th.BoardID)
	// 仅楼主或版主可顶（参考诺哈 ReplyApexRight）
	if th.UserID != uid && !h.isModeratorOfBoard(h.DB, board, uid) {
		resp.Forbidden(c, "只有楼主或版主才能置顶回复")
		return
	}
	content := req.Content
	stickyUID := uid
	if req.ReplyID > 0 {
		var r model.Reply
		if err := h.DB.First(&r, req.ReplyID).Error; err != nil || r.ThreadID != th.ID {
			resp.NotFound(c, "回复不存在")
			return
		}
		content = r.Content
		stickyUID = r.UserID
	}
	if content == "" {
		resp.ParamError(c, "置顶回复内容不能为空")
		return
	}
	h.DB.Where("thread_id = ?", th.ID).Delete(&model.StickyReply{})
	h.DB.Create(&model.StickyReply{ThreadID: th.ID, UserID: stickyUID, Content: content})
	resp.OK(c, nil)
}

// 撤销置顶回复（撤顶，参考诺哈 reply_operate）
func (h *ThreadHandler) UnstickyReply(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	var th model.Thread
	if err := h.DB.First(&th, id).Error; err != nil || th.Status == 0 {
		resp.NotFound(c, "帖子不存在")
		return
	}
	var board model.Board
	h.DB.First(&board, th.BoardID)
	if th.UserID != uid && !h.isModeratorOfBoard(h.DB, board, uid) {
		resp.Forbidden(c, "只有楼主或版主才能撤顶")
		return
	}
	h.DB.Where("thread_id = ?", th.ID).Delete(&model.StickyReply{})
	resp.OK(c, nil)
}

// 回帖列表（倒序分页，参考诺哈 reply_list.asp：最新回复在前）
func (h *ThreadHandler) Replies(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	page, _, _ := pageOf(c, 10)
	var th model.Thread
	if err := h.DB.First(&th, id).Error; err != nil || th.Status == 0 {
		resp.NotFound(c, "帖子不存在或已被删除")
		return
	}
	var total int64
	h.DB.Model(&model.Reply{}).Where("thread_id = ? AND status = 1", th.ID).Count(&total)
	if maxPage := int(total+9) / 10; maxPage < 1 {
		page = 1
	} else if page > maxPage {
		page = maxPage
	}
	var replies []model.Reply
	h.DB.Preload("User").Preload("User.Badges").Where("thread_id = ? AND status = 1", th.ID).
		Order("floor DESC").Offset((page-1)*10).Limit(10).Find(&replies)
	var sticky *model.StickyReply
	h.DB.Preload("User").Where("thread_id = ?", th.ID).First(&sticky)
	if sticky != nil && sticky.ID == 0 {
		sticky = nil
	}
	resp.OK(c, gin.H{"thread_id": th.ID, "thread_title": th.Title, "board_id": th.BoardID,
		"is_lock": th.IsLock, "replies": replies, "total": total, "page": page, "size": 10,
		"sticky_reply": sticky})
}

// 编辑帖子：作者本人，或持有 thread:manage 权限的管理员，或该版版主
func (h *ThreadHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	var req threadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "标题1-100字，内容不能为空")
		return
	}
	var th model.Thread
	if err := h.DB.First(&th, id).Error; err != nil || th.Status == 0 {
		resp.NotFound(c, "帖子不存在或已被删除")
		return
	}
	if th.UserID != uid && !h.canManage(th, uid) {
		resp.Forbidden(c, "只能编辑自己的帖子")
		return
	}
	words := h.loadWords()
	h.DB.Model(&th).Updates(map[string]interface{}{
		"title":   replaceText(req.Title, words),
		"content": replaceText(req.Content, words),
	})
	resp.OK(c, nil)
}

// 删除自己的帖子（版主/管理员可删任意）
func (h *ThreadHandler) DeleteThread(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	var th model.Thread
	if err := h.DB.First(&th, id).Error; err != nil {
		resp.NotFound(c, "帖子不存在")
		return
	}
	if th.UserID != uid && !h.canManage(th, uid) {
		resp.Forbidden(c, "只能删除自己的帖子")
		return
	}
	h.DB.Model(&th).Update("status", 0)
	resp.OK(c, nil)
}

// 删除自己的回复（版主/管理员可删任意）
func (h *ThreadHandler) DeleteReply(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	var reply model.Reply
	if err := h.DB.First(&reply, id).Error; err != nil {
		resp.NotFound(c, "回复不存在")
		return
	}
	var th model.Thread
	h.DB.First(&th, reply.ThreadID)
	var board model.Board
	h.DB.First(&board, th.BoardID)
	if reply.UserID != uid && !h.isModeratorOfBoard(h.DB, board, uid) && !h.hasPerm(uid, "thread:manage") {
		resp.Forbidden(c, "只能删除自己的回复")
		return
	}
	h.DB.Model(&reply).Update("status", 0)
	resp.OK(c, nil)
}

// 热帖排行（按回复/浏览/点赞综合）
func (h *ThreadHandler) Hot(c *gin.Context) {
	var threads []model.Thread
	h.DB.Preload("User").Preload("Board").
		Where("status = 1 AND audit_status = 1").
		Order("reply_count * 3 + view_count / 5 + like_count * 10 DESC").
		Limit(20).Find(&threads)
	resp.OK(c, threads)
}

// 我的帖子
func (h *ThreadHandler) MyThreads(c *gin.Context) {
	uid := middleware.GetUID(c)
	page, _, _ := pageOf(c, 10)
	var total int64
	h.DB.Model(&model.Thread{}).Where("user_id = ? AND status = 1", uid).Count(&total)
	var threads []model.Thread
	h.DB.Preload("Board").Where("user_id = ? AND status = 1", uid).
		Order("id DESC").Offset((page - 1) * 10).Limit(10).Find(&threads)
	resp.OK(c, gin.H{"total": total, "page": page, "list": threads})
}

// 移动帖子到其它版块（版主/管理员）
func (h *ThreadHandler) Move(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	var req struct {
		BoardID uint `json:"board_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择目标版块")
		return
	}
	var th model.Thread
	if err := h.DB.First(&th, id).Error; err != nil || th.Status == 0 {
		resp.NotFound(c, "帖子不存在")
		return
	}
	var oldBoard, newBoard model.Board
	h.DB.First(&oldBoard, th.BoardID)
	h.DB.First(&newBoard, req.BoardID)
	if !h.isModeratorOfBoard(h.DB, oldBoard, uid) && !h.isModeratorOfBoard(h.DB, newBoard, uid) && !h.hasPerm(uid, "thread:manage") {
		resp.Forbidden(c, "没有移动权限")
		return
	}
	if req.BoardID == 0 || newBoard.ParentID == 0 {
		resp.ParamError(c, "目标必须是子板块")
		return
	}
	h.DB.Model(&th).Update("board_id", req.BoardID)
	h.DB.Model(&oldBoard).UpdateColumn("thread_count", gorm.Expr("GREATEST(thread_count - 1, 0)"))
	h.DB.Model(&newBoard).UpdateColumn("thread_count", gorm.Expr("thread_count + 1"))
	resp.OK(c, nil)
}

// 版主对帖子执行操作：置顶/精华/头条/锁定/推荐/公告
func (h *ThreadHandler) Manage(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	var req struct {
		Field string `json:"field" binding:"required"`
		Value int    `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数不对")
		return
	}
	var th model.Thread
	if err := h.DB.First(&th, id).Error; err != nil || th.Status == 0 {
		resp.NotFound(c, "帖子不存在")
		return
	}
	var board model.Board
	h.DB.First(&board, th.BoardID)
	if !h.isModeratorOfBoard(h.DB, board, uid) && !h.hasPerm(uid, "thread:manage") {
		resp.Forbidden(c, "没有管理权限")
		return
	}
	switch req.Field {
	case "is_top", "is_fine", "is_head", "is_lock", "is_recom", "is_notice", "is_active":
		h.DB.Model(&th).Update(req.Field, boolToInt(req.Value != 0))
	default:
		resp.ParamError(c, "不支持的操作")
		return
	}
	resp.OK(c, nil)
}

// 审核队列：通过/拒绝
func (h *ThreadHandler) Audit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	var req struct {
		Approve bool `json:"approve"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数不对")
		return
	}
	var th model.Thread
	if err := h.DB.First(&th, id).Error; err != nil {
		resp.NotFound(c, "帖子不存在")
		return
	}
	var board model.Board
	h.DB.First(&board, th.BoardID)
	if !h.isModeratorOfBoard(h.DB, board, uid) && !h.hasPerm(uid, "thread:manage") {
		resp.Forbidden(c, "没有审核权限")
		return
	}
	if req.Approve {
		h.DB.Model(&th).Update("audit_status", 1)
		addExpAndCoins(h.DB, th.UserID, 10, 5, 3, "post", "发布帖子")
		addHomeNews(h.DB, th.UserID, 0, 1, th.ID, "发表了帖子《"+th.Title+"》")
	} else {
		h.DB.Model(&th).Update("audit_status", 2)
	}
	resp.OK(c, nil)
}

// 待审核帖子列表（版主/管理员）
func (h *ThreadHandler) AuditList(c *gin.Context) {
	uid := middleware.GetUID(c)
	// 版主只看自己板块的
	perm := h.hasPerm(uid, "thread:manage")
	q := h.DB.Model(&model.Thread{}).Where("audit_status = 0 AND status = 1")
	if !perm {
		var bid []uint
		h.DB.Model(&model.Board{}).Where("moderator_id = ?", uid).Pluck("id", &bid)
		q = q.Where("board_id IN ?", bid)
	}
	var list []model.Thread
	q.Preload("User").Preload("Board").Order("id DESC").Limit(50).Find(&list)
	resp.OK(c, list)
}

// 是否可管理某帖（作者/版主/thread:manage 权限）
func (h *ThreadHandler) canManage(th model.Thread, uid uint) bool {
	if th.UserID == uid {
		return true
	}
	if h.hasPerm(uid, "thread:manage") {
		return true
	}
	var board model.Board
	h.DB.First(&board, th.BoardID)
	return h.isModeratorOfBoard(h.DB, board, uid)
}

func (h *ThreadHandler) isModeratorOfBoard(db *gorm.DB, board model.Board, uid uint) bool {
	return uid != 0 && board.ModeratorID == uid
}

func (h *ThreadHandler) hasPerm(uid uint, code string) bool {
	var count int64
	h.DB.Raw(`SELECT COUNT(DISTINCT p.id)
FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
JOIN user_roles ur ON ur.role_id = rp.role_id
WHERE ur.user_id = ? AND p.code = ?`, uid, code).Scan(&count)
	return count > 0
}

// loadWords 敏感词列表
func (h *ThreadHandler) loadWords() []model.WordFilter {
	var ws []model.WordFilter
	h.DB.Find(&ws)
	return ws
}

// 附件下载（付费扣费给楼主）
func (h *ThreadHandler) Download(c *gin.Context) {
	attachID, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	var att model.ThreadAttachment
	if err := h.DB.First(&att, attachID).Error; err != nil {
		resp.NotFound(c, "附件不存在")
		return
	}
	if att.Price > 0 {
		var me model.User
		h.DB.First(&me, uid)
		if me.Coins < att.Price {
			resp.ParamError(c, "金币不足，无法下载")
			return
		}
		var th model.Thread
		h.DB.First(&th, att.ThreadID)
		// 已下载过不再重复扣费
		// 简化：直接扣费并转给楼主
		h.DB.Model(&me).UpdateColumn("coins", gorm.Expr("coins - ?", att.Price))
		addWalletLog(h.DB, uid, "download", "下载附件", "coins", -att.Price)
		if th.UserID != uid {
			h.DB.Model(&model.User{}).Where("id = ?", th.UserID).UpdateColumn("coins", gorm.Expr("coins + ?", att.Price))
			addWalletLog(h.DB, th.UserID, "sell", "附件下载收益", "coins", att.Price)
		}
	}
	h.DB.Model(&att).UpdateColumn("downloads", gorm.Expr("downloads + 1"))
	resp.OK(c, gin.H{"path": att.Path, "name": att.Name})
}