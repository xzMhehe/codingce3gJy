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

type BoardHandler struct {
	DB *gorm.DB
}

// 板块树：分区 -> (分类 -> 子板块)。返回带分类分组的树。
func (h *BoardHandler) Tree(c *gin.Context) {
	var boards []model.Board
	h.DB.Where("status = 1").Order("sort ASC, id ASC").Find(&boards)
	var categories []model.BoardCategory
	h.DB.Order("sort ASC, id ASC").Find(&categories)

	type childNode struct {
		model.Board
		Moderator *model.User `json:"moderator,omitempty"`
	}
	type catNode struct {
		model.BoardCategory
		Boards []childNode `json:"boards"`
	}
	type node struct {
		model.Board
		Children   []childNode `json:"children"`
		Categories []catNode   `json:"categories"`
	}

	childrenByParent := map[uint][]childNode{}
	for _, b := range boards {
		if b.ParentID != 0 {
			var mod *model.User
			if b.ModeratorID != 0 {
				h.DB.First(&mod, b.ModeratorID)
			}
			childrenByParent[b.ParentID] = append(childrenByParent[b.ParentID], childNode{Board: b, Moderator: mod})
		}
	}
	// 版主用户预取（避免 N+1）
	modIDs := map[uint]model.User{}
	var modUsers []model.User
	var modSet []uint
	for _, b := range boards {
		if b.ModeratorID != 0 {
			modSet = append(modSet, b.ModeratorID)
		}
	}
	if len(modSet) > 0 {
		h.DB.Where("id IN ?", modSet).Find(&modUsers)
		for _, u := range modUsers {
			modIDs[u.ID] = u
		}
	}

	catByParent := map[uint][]model.BoardCategory{}
	for _, ct := range categories {
		catByParent[ct.ParentID] = append(catByParent[ct.ParentID], ct)
	}

	nodes := []node{}
	for _, b := range boards {
		if b.ParentID != 0 {
			continue
		}
		// 有分类的分区：按分类归组
		hasCat := len(catByParent[b.ID]) > 0
		n := node{Board: b}
		if hasCat {
			// 已分类的子板块从 children 中剔除，归入分类
			var cats []catNode
			for _, ct := range catByParent[b.ID] {
				var cb []childNode
				for _, ch := range childrenByParent[b.ID] {
					if ch.CategoryID == ct.ID {
						cb = append(cb, ch)
					}
				}
				cats = append(cats, catNode{BoardCategory: ct, Boards: cb})
			}
			// 未分类的子板块仍放 children
			var rest []childNode
			for _, ch := range childrenByParent[b.ID] {
				if ch.CategoryID == 0 {
					rest = append(rest, ch)
				}
			}
			n.Children = rest
			n.Categories = cats
		} else {
			n.Children = childrenByParent[b.ID]
		}
		nodes = append(nodes, n)
	}
	// 修正版主引用
	for i := range nodes {
		for j := range nodes[i].Children {
			if nodes[i].Children[j].ModeratorID != 0 {
				if u, ok := modIDs[nodes[i].Children[j].ModeratorID]; ok {
					uu := u
					nodes[i].Children[j].Moderator = &uu
				}
			}
		}
		for k := range nodes[i].Categories {
			for j := range nodes[i].Categories[k].Boards {
				if nodes[i].Categories[k].Boards[j].ModeratorID != 0 {
					if u, ok := modIDs[nodes[i].Categories[k].Boards[j].ModeratorID]; ok {
						uu := u
						nodes[i].Categories[k].Boards[j].Moderator = &uu
					}
				}
			}
		}
	}
	resp.OK(c, nodes)
}

func (h *BoardHandler) Info(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var board model.Board
	if err := h.DB.First(&board, id).Error; err != nil {
		resp.NotFound(c, "板块不存在")
		return
	}
	var mod *model.User
	if board.ModeratorID != 0 {
		h.DB.First(&mod, board.ModeratorID)
	}
	resp.OK(c, gin.H{"board": board, "moderator": mod})
}

// 版块帖子列表（置顶/头条优先，然后按最后回复时间）
func (h *BoardHandler) Threads(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	page, _, _ := pageOf(c, 10)

	var board model.Board
	if err := h.DB.First(&board, id).Error; err != nil {
		resp.NotFound(c, "板块不存在")
		return
	}
	// 分区页：聚合所有子板块的帖子
	boardIDs := []uint{board.ID}
	if board.ParentID == 0 {
		var subs []model.Board
		h.DB.Where("parent_id = ?", board.ID).Find(&subs)
		for _, s := range subs {
			boardIDs = append(boardIDs, s.ID)
		}
	}

	// 登录用户心跳：记录所在版块（参考诺哈 wap_online.bbsid，用于版块在线统计）
	if uid := middleware.GetUID(c); uid != 0 {
		h.DB.Model(&model.User{}).Where("id = ?", uid).Updates(map[string]interface{}{
			"last_board_id": board.ID, "last_active_at": time.Now(),
		})
	}
	// 版块在线人数：10 分钟内活跃且停留在本版块（分区则统计所有子板块）
	tenMinAgo := time.Now().Add(-10 * time.Minute)
	var boardOnline int64
	h.DB.Model(&model.User{}).Where("last_board_id IN ? AND last_active_at > ?", boardIDs, tenMinAgo).Count(&boardOnline)

	// 版块头条（参考诺哈 ForumTopicHead：最新一条 head 帖）
	var head *model.Thread
	var ht model.Thread
	if err := h.DB.Preload("User").Where("board_id IN ? AND is_head = 1 AND status = 1 AND audit_status = 1", boardIDs).
		Order("id DESC").First(&ht).Error; err == nil {
		head = &ht
	}

	q := h.DB.Model(&model.Thread{}).Where("board_id IN ? AND status = 1 AND audit_status = 1", boardIDs)
	if f := c.Query("filter"); f == "fine" {
		q = q.Where("is_fine = 1")
	} else if f == "new" {
		// 新贴：最近 24h
	}
	if c.Query("notice") == "1" {
		q = q.Where("is_notice = 1")
	}
	order := "is_top DESC, is_head DESC, IFNULL(last_reply_at, created_at) DESC"
	if c.Query("sort") == "new" {
		order = "is_top DESC, is_head DESC, created_at DESC"
	}
	var total int64
	q.Count(&total)
	if maxPage := int(total+9) / 10; maxPage < 1 {
		page = 1
	} else if page > maxPage {
		page = maxPage
	}

	var threads []model.Thread
	q.Preload("User").Preload("User.Badges").Preload("Board").
		Order(order).
		Offset((page - 1) * 10).Limit(10).Find(&threads)

	// 版主信息
	var mod *model.User
	if board.ModeratorID != 0 {
		h.DB.First(&mod, board.ModeratorID)
	}
	// 当前登录用户是否为该板块成员
	isMember := false
	if uid := middleware.GetUID(c); uid != 0 && board.MembersOnly == 1 {
		var cnt int64
		h.DB.Model(&model.BoardMember{}).Where("board_id = ? AND user_id = ? AND status = 1", id, uid).Count(&cnt)
		isMember = cnt > 0
	}
	resp.OK(c, gin.H{"board": board, "moderator": mod, "is_member": isMember,
		"board_online": boardOnline, "head": head,
		"total": total, "page": page, "size": 10, "list": threads})
}

type threadReq struct {
	Title   string `json:"title" binding:"required,min=1,max=100"`
	Content string `json:"content" binding:"required,min=1,max=20000"`
}

// 发帖（支持类型：0普通 1回帖奖励 2踩楼 3投票，及附件/审核/敏感词/频率限制）
func (h *BoardHandler) CreateThread(c *gin.Context) {
	boardID, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	var req struct {
		Title    string `json:"title" binding:"required,min=1,max=100"`
		Content  string `json:"content" binding:"required,min=1,max=20000"`
		Type     int    `json:"type"`
		PollQ    string `json:"poll_question"`
		PollOpts []string `json:"poll_options"`
		PollMulti bool  `json:"poll_multiple"`
		RewardCoins int  `json:"reward_coins"`
		RewardExp   int  `json:"reward_exp"`
		RewardLimit int  `json:"reward_limit"`
		Floors   []model.ThreadFloor `json:"floors"`
		Attachments []struct {
			Name string `json:"name"`
			Path string `json:"path"`
			Size int64  `json:"size"`
			Price int   `json:"price"`
		} `json:"attachments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "标题1-100字，内容不能为空")
		return
	}
	var board model.Board
	if err := h.DB.First(&board, uint(boardID)).Error; err != nil || board.ParentID == 0 {
		resp.ParamError(c, "请选择具体的子板块发帖")
		return
	}
	// 家族专属论坛板块（家族·xxx）：仅家族成员可发帖
	if famID, ok := familyBoardOwner(h.DB, board); ok && !isFamilyMember(h.DB, famID, uid) {
		resp.Forbidden(c, "只有家族成员才能在家族论坛发帖")
		return
	}
	// 会员制版块：仅成员可发帖
	if board.MembersOnly == 1 {
		var cnt int64
		h.DB.Model(&model.BoardMember{}).Where("board_id = ? AND user_id = ? AND status = 1", boardID, uid).Count(&cnt)
		if cnt == 0 {
			resp.Forbidden(c, "该版块为会员制，只有成员才能发帖")
			return
		}
	}
	// 版主可发帖到任意板块
	if !h.isModerator(board, uid) {
		if msg := h.checkPostRate(uid, "post"); msg != "" {
			resp.ParamError(c, msg)
			return
		}
	}

	// 敏感词过滤 / 审核
	replace, needAudit := h.filterText(req.Title + "\n" + req.Content)
	title := replaceText(req.Title, h.loadWords())
	content := replaceText(req.Content, h.loadWords())
	_ = replace

	th := model.Thread{
		BoardID: uint(boardID), UserID: uid, Title: title, Content: content, Type: req.Type,
		AuditStatus: 1,
	}
	if needAudit {
		th.AuditStatus = 0
	}
	if err := h.DB.Create(&th).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	// 投票
	if req.Type == 3 && req.PollQ != "" && len(req.PollOpts) > 0 {
		poll := model.ThreadPoll{ThreadID: th.ID, Question: req.PollQ, Multiple: boolToInt(req.PollMulti)}
		for _, o := range req.PollOpts {
			if o != "" {
				poll.Options = append(poll.Options, model.ThreadPollOption{Name: o})
			}
		}
		h.DB.Create(&poll)
	}
	// 回帖奖励
	if req.Type == 1 {
		h.DB.Create(&model.ThreadReward{ThreadID: th.ID, Coins: req.RewardCoins, Exp: req.RewardExp, Limit: req.RewardLimit})
	}
	// 踩楼
	if req.Type == 2 && len(req.Floors) > 0 {
		for _, f := range req.Floors {
			if f.Floor > 0 {
				h.DB.Create(&model.ThreadFloor{ThreadID: th.ID, Floor: f.Floor, Coins: f.Coins, Exp: f.Exp})
			}
		}
	}
	// 附件
	if len(req.Attachments) > 0 {
		for _, a := range req.Attachments {
			if a.Path != "" {
				h.DB.Create(&model.ThreadAttachment{ThreadID: th.ID, UserID: uid, Name: a.Name, Path: a.Path, Size: a.Size, Price: a.Price})
			}
		}
	}

	h.DB.Model(&board).UpdateColumn("thread_count", gorm.Expr("thread_count + 1"))
	if th.AuditStatus == 1 {
		addExpAndCoins(h.DB, uid, 10, 5, 3, "post", "发布帖子")
		addHomeNews(h.DB, uid, 0, 1, th.ID, "发表了帖子《"+th.Title+"》")
	}
	resp.OK(c, gin.H{"id": th.ID, "audit": th.AuditStatus})
}

// loadWords 敏感词列表
func (h *BoardHandler) loadWords() []model.WordFilter {
	var ws []model.WordFilter
	h.DB.Find(&ws)
	return ws
}

// 敏感词过滤：返回（是否需审核）
func (h *BoardHandler) filterText(txt string) (string, bool) {
	needAudit := false
	for _, w := range h.loadWords() {
		if w.Type == 2 && contains(txt, w.Word) {
			needAudit = true
		}
	}
	return "", needAudit
}

// 判断用户是否该版块版主
func (h *BoardHandler) isModerator(board model.Board, uid uint) bool {
	return uid != 0 && board.ModeratorID == uid
}

// 发帖/回帖频率限制（秒）
func (h *BoardHandler) checkPostRate(uid uint, kind string) string {
	if kind == "post" {
		var last model.Thread
		if h.DB.Where("user_id = ?", uid).Order("id DESC").First(&last).Error == nil {
			if time.Since(last.CreatedAt) < 30*time.Second {
				return "发帖太快了，请休息 30 秒再试"
			}
		}
	}
	return ""
}

func contains(s, sub string) bool {
	return len(sub) > 0 && (len(s) >= len(sub)) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

func replaceText(s string, words []model.WordFilter) string {
	for _, w := range words {
		if w.Type != 1 {
			continue
		}
		r := w.Replace
		if r == "" {
			r = "***"
		}
		s = replaceAllStr(s, w.Word, r)
	}
	return s
}

func replaceAllStr(s, old, new string) string {
	if old == "" {
		return s
	}
	out := ""
	for {
		i := indexStr(s, old)
		if i < 0 {
			out += s
			break
		}
		out += s[:i] + new
		s = s[i+len(old):]
	}
	return out
}

func indexStr(s, sub string) int {
	if sub == "" {
		return -1
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}