package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type GoodHandler struct{ DB *gorm.DB }

// 公开：上架商品列表
func (h *GoodHandler) List(c *gin.Context) {
	var list []model.Good
	h.DB.Where("status = 1").Order("sort ASC, id ASC").Find(&list)
	resp.OK(c, list)
}

// 购买道具（扣金币）
func (h *GoodHandler) Buy(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var g model.Good
	if err := h.DB.First(&g, id).Error; err != nil {
		resp.NotFound(c, "商品不存在")
		return
	}
	if g.Status == 0 {
		resp.ParamError(c, "商品已下架")
		return
	}
	var u model.User
	h.DB.First(&u, uid)
	if u.Coins < g.Price {
		resp.ParamError(c, "金币不足，需要 "+strconv.Itoa(g.Price)+" 金币")
		return
	}
	h.DB.Model(&u).Update("coins", gorm.Expr("coins - ?", g.Price))
	resp.OK(c, gin.H{"name": g.Name, "coins": u.Coins - g.Price})
}

// 后台：商品列表
func (h *GoodHandler) AdminList(c *gin.Context) {
	var list []model.Good
	h.DB.Order("sort ASC, id ASC").Find(&list)
	resp.OK(c, list)
}

type goodReq struct {
	Name     string `json:"name" binding:"required"`
	Desc     string `json:"desc"`
	Icon     string `json:"icon"`
	Category string `json:"category"`
	Price    int    `json:"price"`
	Status   int    `json:"status"`
	Sort     int    `json:"sort"`
}

func (h *GoodHandler) AdminCreate(c *gin.Context) {
	var req goodReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "商品名必填")
		return
	}
	if req.Status == 0 {
		req.Status = 1
	}
	h.DB.Create(&model.Good{Name: req.Name, Desc: req.Desc, Icon: req.Icon, Category: req.Category, Price: req.Price, Status: req.Status, Sort: req.Sort})
	resp.OK(c, nil)
}

func (h *GoodHandler) AdminUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req goodReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数有误")
		return
	}
	h.DB.Model(&model.Good{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name": req.Name, "desc": req.Desc, "icon": req.Icon, "category": req.Category,
		"price": req.Price, "status": req.Status, "sort": req.Sort,
	})
	resp.OK(c, nil)
}

func (h *GoodHandler) AdminDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.Good{}, id)
	resp.OK(c, nil)
}
