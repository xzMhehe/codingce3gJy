package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

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
