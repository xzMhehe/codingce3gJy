package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type BookHandler struct{ DB *gorm.DB }

// 书城分类
var bookCategories = []string{"武侠", "言情", "都市", "灵异"}

// 分类列表
func (h *BookHandler) Categories(c *gin.Context) {
	resp.OK(c, bookCategories)
}

// 书城聚合：强推 / 最新连载 / 新书上架 / 分类书库首页
func (h *BookHandler) Index(c *gin.Context) {
	var recommend, latest, newest []model.Book
	h.DB.Where("recommend = 1").Order("id ASC").Limit(10).Find(&recommend)
	h.DB.Where("status = '连载'").Order("id DESC").Limit(10).Find(&latest)
	h.DB.Where("new_book = 1").Order("id DESC").Limit(10).Find(&newest)
	all := []model.Book{}
	h.DB.Order("id ASC").Limit(60).Find(&all)
	resp.OK(c, gin.H{"recommend": recommend, "latest": latest, "newest": newest, "all": all, "categories": bookCategories})
}

// 按分类列表
func (h *BookHandler) List(c *gin.Context) {
	cat := c.Query("category")
	q := h.DB.Model(&model.Book{})
	if cat != "" {
		q = q.Where("category = ?", cat)
	}
	q = q.Order("id ASC")
	if kw := c.Query("wd"); kw != "" {
		q = q.Where("title LIKE ? OR author LIKE ?", "%"+kw+"%", "%"+kw+"%")
	}
	var books []model.Book
	q.Limit(100).Find(&books)
	resp.OK(c, books)
}

// 书籍详情
func (h *BookHandler) Detail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var b model.Book
	if err := h.DB.First(&b, id).Error; err != nil {
		resp.NotFound(c, "书未找到")
		return
	}
	h.DB.Model(&b).Update("views", gorm.Expr("views + ?", 1))
	resp.OK(c, b)
}

// ---- 以下为书城联动扩展（章节/书评/书架，对齐诺哈 wap/book） ----

// Chapters 章节目录
func (h *BookHandler) Chapters(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var chs []model.BookChapter
	h.DB.Where("book_id = ?", id).Order("sort ASC, id ASC").Find(&chs)
	resp.OK(c, chs)
}

// Chapter 章节内容（阅读）
func (h *BookHandler) Chapter(c *gin.Context) {
	var ch model.BookChapter
	if err := h.DB.First(&ch, c.Param("id")).Error; err != nil {
		resp.NotFound(c, "章节不存在")
		return
	}
	var b model.Book
	h.DB.First(&b, ch.BookID)
	// 上一章/下一章
	var prev, next model.BookChapter
	h.DB.Where("book_id = ? AND sort < ?", ch.BookID, ch.Sort).Order("sort DESC").First(&prev)
	h.DB.Where("book_id = ? AND sort > ?", ch.BookID, ch.Sort).Order("sort ASC").First(&next)
	resp.OK(c, gin.H{"book": gin.H{"id": b.ID, "title": b.Title},
		"chapter": ch,
		"prev_id": prev.ID, "next_id": next.ID,
		"prev_title": prev.Title, "next_title": next.Title})
}

// Comments 书评列表
func (h *BookHandler) Comments(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var list []model.BookComment
	h.DB.Where("book_id = ? AND status = 1", id).Order("id DESC").Limit(50).Find(&list)
	uids := map[uint]bool{}
	for _, m := range list {
		uids[m.UserID] = true
	}
	nick := h.nickMap(uids)
	out := make([]gin.H, 0, len(list))
	for _, m := range list {
		out = append(out, gin.H{"id": m.ID, "user_id": m.UserID, "nick": nick[m.UserID],
			"score": m.Score, "content": m.Content, "time_txt": m.CreatedAt.Format("2006-01-02 15:04")})
	}
	resp.OK(c, out)
}

// CommentAdd 发表书评（1-5星）
func (h *BookHandler) CommentAdd(c *gin.Context) {
	uid := middleware.GetUID(c)
	bid, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Score   int    `json:"score"`
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写评论内容")
		return
	}
	if req.Score < 1 || req.Score > 5 {
		req.Score = 5
	}
	var b model.Book
	if err := h.DB.First(&b, bid).Error; err != nil {
		resp.NotFound(c, "书未找到")
		return
	}
	m := model.BookComment{BookID: uint(bid), UserID: uid, Score: req.Score, Content: req.Content}
	h.DB.Create(&m)
	resp.OK(c, gin.H{"msg": "书评发表成功", "id": m.ID})
}

// ShelfAdd 加入书架
func (h *BookHandler) ShelfAdd(c *gin.Context) {
	uid := middleware.GetUID(c)
	bid, _ := strconv.Atoi(c.Param("id"))
	var b model.Book
	if err := h.DB.First(&b, bid).Error; err != nil {
		resp.NotFound(c, "书未找到")
		return
	}
	var n int64
	h.DB.Model(&model.BookShelf{}).Where("user_id = ? AND book_id = ?", uid, bid).Count(&n)
	if n > 0 {
		resp.OK(c, gin.H{"msg": "已在书架中"})
		return
	}
	h.DB.Create(&model.BookShelf{UserID: uid, BookID: uint(bid)})
	resp.OK(c, gin.H{"msg": "已加入书架"})
}

// ShelfDel 移出书架
func (h *BookHandler) ShelfDel(c *gin.Context) {
	uid := middleware.GetUID(c)
	bid, _ := strconv.Atoi(c.Param("id"))
	h.DB.Where("user_id = ? AND book_id = ?", uid, bid).Delete(&model.BookShelf{})
	resp.OK(c, gin.H{"msg": "已移出书架"})
}

// ShelfList 我的书架
func (h *BookHandler) ShelfList(c *gin.Context) {
	uid := middleware.GetUID(c)
	var shelves []model.BookShelf
	h.DB.Where("user_id = ?", uid).Order("id DESC").Find(&shelves)
	ids := make([]uint, 0, len(shelves))
	for _, s := range shelves {
		ids = append(ids, s.BookID)
	}
	books := []model.Book{}
	if len(ids) > 0 {
		h.DB.Where("id IN ?", ids).Find(&books)
	}
	resp.OK(c, books)
}

// nickMap 批量取昵称
func (h *BookHandler) nickMap(uidSet map[uint]bool) map[uint]string {
	out := map[uint]string{}
	if len(uidSet) == 0 {
		return out
	}
	ids := make([]uint, 0, len(uidSet))
	for k := range uidSet {
		ids = append(ids, k)
	}
	var us []model.User
	h.DB.Select("id, nickname").Where("id IN ?", ids).Find(&us)
	for _, u := range us {
		out[u.ID] = u.Nickname
	}
	return out
}
