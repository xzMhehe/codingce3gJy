package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type BoardHandler struct {
	DB *gorm.DB
}

// 板块树：分区 -> 子板块
func (h *BoardHandler) Tree(c *gin.Context) {
	var boards []model.Board
	h.DB.Where("status = 1").Order("sort ASC, id ASC").Find(&boards)
	type node struct {
		model.Board
		Children []model.Board `json:"children"`
	}
	nodes := []node{}
	byParent := map[uint][]model.Board{}
	for _, b := range boards {
		if b.ParentID != 0 {
			byParent[b.ParentID] = append(byParent[b.ParentID], b)
		}
	}
	for _, b := range boards {
		if b.ParentID == 0 {
			nodes = append(nodes, node{Board: b, Children: byParent[b.ID]})
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
	resp.OK(c, board)
}

// 板块帖子列表（置顶优先，然后按最后回复时间）
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

	q := h.DB.Model(&model.Thread{}).Where("board_id IN ? AND status = 1", boardIDs)
	if c.Query("filter") == "fine" {
		q = q.Where("is_fine = 1")
	}
	order := "is_top DESC, IFNULL(last_reply_at, created_at) DESC"
	if c.Query("sort") == "new" {
		order = "is_top DESC, created_at DESC"
	}
	var total int64
	q.Count(&total)
	// 页码夹紧到有效范围
	if maxPage := int(total+9) / 10; maxPage < 1 {
		page = 1
	} else if page > maxPage {
		page = maxPage
	}

	var threads []model.Thread
	q.Preload("User").Preload("User.Badges").Preload("Board").
		Order(order).
		Offset((page - 1) * 10).Limit(10).Find(&threads)
	resp.OK(c, gin.H{"board": board, "total": total, "page": page, "size": 10, "list": threads})
}

type threadReq struct {
	Title   string `json:"title" binding:"required,min=1,max=100"`
	Content string `json:"content" binding:"required,min=1,max=10000"`
}

func (h *BoardHandler) CreateThread(c *gin.Context) {
	boardID, _ := strconv.Atoi(c.Param("id"))
	uid := middleware.GetUID(c)
	var req threadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "标题1-100字，内容不能为空")
		return
	}
	var board model.Board
	if err := h.DB.First(&board, boardID).Error; err != nil || board.ParentID == 0 {
		resp.ParamError(c, "请选择具体的子板块发帖")
		return
	}
	// 家族专属论坛板块（家族·xxx）：仅家族成员可发帖
	if famID, ok := familyBoardOwner(h.DB, board); ok && !isFamilyMember(h.DB, famID, uid) {
		resp.Forbidden(c, "只有家族成员才能在家族论坛发帖")
		return
	}
	th := model.Thread{BoardID: uint(boardID), UserID: uid, Title: req.Title, Content: req.Content}
	if err := h.DB.Create(&th).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	h.DB.Model(&board).UpdateColumn("thread_count", gorm.Expr("thread_count + 1"))
	addExpAndCoins(h.DB, uid, 10, 5, 3, "post", "发布帖子")
	addHomeNews(h.DB, uid, 0, 1, th.ID, "发表了帖子《"+th.Title+"》")
	resp.OK(c, gin.H{"id": th.ID})
}
