package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type SiteArticleHandler struct{ DB *gorm.DB }

// Categories 文章分类
func (h *SiteArticleHandler) Categories(c *gin.Context) {
	var cats []model.SiteArticleCategory
	h.DB.Order("sort ASC, id ASC").Find(&cats)
	resp.OK(c, cats)
}

type siteArticleRow struct {
	model.SiteArticle
	Author  string `json:"author"`
	Color   string `json:"color"`
	CatName string `json:"cat_name"`
}

// List 文章列表（可按分类）
func (h *SiteArticleHandler) List(c *gin.Context) {
	page, size := pageParams(c, 10)
	q := h.DB.Model(&model.SiteArticle{}).Where("status = 1")
	if catID, err := strconv.Atoi(c.Query("cat_id")); err == nil && catID > 0 {
		q = q.Where("cat_id = ?", catID)
	}
	var total int64
	q.Count(&total)
	var list []model.SiteArticle
	q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&list)
	out := make([]siteArticleRow, 0, len(list))
	for _, a := range list {
		row := siteArticleRow{SiteArticle: a}
		var u model.User
		h.DB.Select("nickname,color").First(&u, a.UserID)
		row.Author = u.Nickname
		row.Color = u.Color
		if a.CatID > 0 {
			var cat model.SiteArticleCategory
			h.DB.Select("name").First(&cat, a.CatID)
			row.CatName = cat.Name
		}
		out = append(out, row)
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// Detail 文章正文（浏览+1）
func (h *SiteArticleHandler) Detail(c *gin.Context) {
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	var a model.SiteArticle
	if err := h.DB.Where("id = ? AND status = 1", id).First(&a).Error; err != nil {
		resp.NotFound(c, "文章不存在")
		return
	}
	h.DB.Model(&a).UpdateColumn("click", gorm.Expr("click + 1"))
	a.Click++
	var u model.User
	h.DB.Select("nickname,color").First(&u, a.UserID)
	var cat model.SiteArticleCategory
	if a.CatID > 0 {
		h.DB.Select("name").First(&cat, a.CatID)
	}
	resp.OK(c, gin.H{
		"id": a.ID, "title": a.Title, "content": a.Content, "click": a.Click,
		"comment": a.Comment, "writer": a.Writer, "source": a.Source,
		"created_at": a.CreatedAt, "author": u.Nickname, "author_id": a.UserID,
		"color": u.Color, "cat_id": a.CatID, "cat_name": cat.Name,
	})
}

// Comments 文章评论列表
func (h *SiteArticleHandler) Comments(c *gin.Context) {
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	var list []model.SiteArticleComment
	h.DB.Where("article_id = ? AND status = 1", id).Order("created_at ASC").Limit(100).Find(&list)
	type row struct {
		model.SiteArticleComment
		Nickname string `json:"nickname"`
		Color    string `json:"color"`
	}
	out := make([]row, 0, len(list))
	for _, cm := range list {
		r := row{SiteArticleComment: cm}
		var u model.User
		h.DB.Select("nickname,color").First(&u, cm.UserID)
		r.Nickname = u.Nickname
		r.Color = u.Color
		out = append(out, r)
	}
	resp.OK(c, out)
}

// CommentAdd 发表评论（评论+1）
func (h *SiteArticleHandler) CommentAdd(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	var req struct {
		Content string `json:"content" binding:"required,max=300"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "评论内容不能为空且不超过 300 字")
		return
	}
	var a model.SiteArticle
	if err := h.DB.Where("id = ? AND status = 1", id).First(&a).Error; err != nil {
		resp.NotFound(c, "文章不存在")
		return
	}
	h.DB.Create(&model.SiteArticleComment{ArticleID: a.ID, UserID: uid, Content: req.Content, Status: 1})
	h.DB.Model(&a).UpdateColumn("comment", gorm.Expr("comment + 1"))
	resp.OK(c, nil)
}

// Add 投稿文章（登录用户；奖励 2 经验 + 5 G币，对齐诺哈发稿奖励）
func (h *SiteArticleHandler) Add(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Title  string `json:"title" binding:"required,max=60"`
		CatID  uint   `json:"cat_id"`
		Writer string `json:"writer"`
		Source string `json:"source"`
		Content string `json:"content" binding:"required,min=10"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "标题不能为空，正文至少 10 个字")
		return
	}
	if !verifyTime("article_"+strconv.FormatUint(uint64(uid), 10), 15) {
		resp.ParamError(c, "发布太频繁了，歇 15 秒再来")
		return
	}
	if req.CatID > 0 {
		var cnt int64
		h.DB.Model(&model.SiteArticleCategory{}).Where("id = ?", req.CatID).Count(&cnt)
		if cnt == 0 {
			req.CatID = 0
		}
	}
	a := model.SiteArticle{
		UserID: uid, Title: req.Title, CatID: req.CatID,
		Writer: req.Writer, Source: req.Source, Content: req.Content, Status: 1,
	}
	if err := h.DB.Create(&a).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	h.DB.Model(&model.User{}).Where("id = ?", uid).Updates(map[string]interface{}{
		"exp":   gorm.Expr("exp + 2"),
		"coins": gorm.Expr("coins + 5"),
	})
	addWalletLog(h.DB, uid, "post", "发表文章", "coins", 5)
	addHomeNews(h.DB, uid, 0, 105, a.ID, "发表了文章《"+a.Title+"》")
	resp.OK(c, gin.H{"id": a.ID})
}

// Del 删除文章（作者或管理员）
func (h *SiteArticleHandler) Del(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	var a model.SiteArticle
	if err := h.DB.First(&a, id).Error; err != nil {
		resp.NotFound(c, "文章不存在")
		return
	}
	if a.UserID != uid && !h.isAdmin(uid) {
		resp.Forbidden(c, "只能删除自己的文章")
		return
	}
	h.DB.Model(&a).Update("status", 0)
	resp.OK(c, nil)
}

func (h *SiteArticleHandler) isAdmin(uid uint) bool {
	for _, code := range middleware.UserPermissionCodes(h.DB, uid) {
		if code == "admin:access" {
			return true
		}
	}
	return false
}
