package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type GoodHandler struct{ DB *gorm.DB }

// 分类列表（按现有商品聚合）
func goodCategories(list []model.Good) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, g := range list {
		if g.Category != "" && !seen[g.Category] {
			seen[g.Category] = true
			out = append(out, g.Category)
		}
	}
	return out
}

// 公开：商城列表（?cat=分类 筛选；登录时带G币余额）
func (h *GoodHandler) List(c *gin.Context) {
	q := h.DB.Where("status = 1")
	if cat := c.Query("cat"); cat != "" && cat != "全部" {
		q = q.Where("category = ?", cat)
	}
	var list []model.Good
	q.Order("sort ASC, id ASC").Find(&list)
	uid := middleware.GetUID(c)
	coins := -1
	if uid > 0 {
		var u model.User
		if h.DB.First(&u, uid).Error == nil {
			coins = u.Coins
		}
	}
	resp.OK(c, gin.H{"list": list, "categories": goodCategories(list), "coins": coins})
}

// 购买道具（扣G币，数量可叠加进仓库）
func (h *GoodHandler) Buy(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Num int `json:"num"` // 购买数量，默认1
	}
	_ = c.ShouldBindJSON(&req)
	if req.Num < 1 {
		req.Num = 1
	}
	if req.Num > 999 {
		req.Num = 999
	}
	var g model.Good
	if err := h.DB.First(&g, id).Error; err != nil {
		resp.NotFound(c, "商品不存在")
		return
	}
	if g.Status == 0 {
		resp.ParamError(c, "商品已下架")
		return
	}
	cost := g.Price * req.Num
	var u model.User
	if err := h.DB.First(&u, uid).Error; err != nil {
		resp.Unauthorized(c, "请先登录")
		return
	}
	if u.Coins < cost {
		resp.ParamError(c, "G币不足，需要 "+strconv.Itoa(cost)+" G币")
		return
	}
	h.DB.Model(&u).Update("coins", gorm.Expr("coins - ?", cost))
	addWalletLog(h.DB, uid, "buy", "购买「"+g.Name+"」×"+strconv.Itoa(req.Num), "coins", -cost)
	// 入库（背包叠加）
	var ug model.UserGood
	if err := h.DB.Where("user_id = ? AND good_id = ?", uid, g.ID).First(&ug).Error; err != nil {
		h.DB.Create(&model.UserGood{UserID: uid, GoodID: g.ID, Count: req.Num})
	} else {
		h.DB.Model(&ug).Update("count", gorm.Expr("count + ?", req.Num))
	}
	resp.OK(c, gin.H{"name": g.Name, "num": req.Num, "coins": u.Coins - cost, "bag_count": req.Num + (func() int {
		if ug.ID == 0 {
			return 0
		}
		return ug.Count
	})()})
}

// 我的仓库（背包列表）
func (h *GoodHandler) Bag(c *gin.Context) {
	uid := middleware.GetUID(c)
	var list []model.UserGood
	h.DB.Preload("Good").Where("user_id = ? AND count > 0", uid).Order("updated_at DESC").Find(&list)
	out := []gin.H{}
	for _, ug := range list {
		if ug.Good == nil {
			continue
		}
		out = append(out, gin.H{
			"id": ug.ID, "good_id": ug.GoodID, "count": ug.Count,
			"name": ug.Good.Name, "icon": ug.Good.Icon, "desc": ug.Good.Desc,
			"category": ug.Good.Category, "price": ug.Good.Price,
		})
	}
	resp.OK(c, gin.H{"list": out})
}

// 使用道具：改名卡真改昵称；鲜花提示去送花；其余提示待开放（不消耗）
func (h *GoodHandler) BagUse(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Nickname string `json:"nickname"` // 改名卡目标昵称
	}
	_ = c.ShouldBindJSON(&req)
	var ug model.UserGood
	if err := h.DB.Preload("Good").Where("id = ? AND user_id = ?", id, uid).First(&ug).Error; err != nil || ug.Good == nil {
		resp.NotFound(c, "仓库中没有该道具")
		return
	}
	if ug.Count < 1 {
		resp.ParamError(c, "该道具数量不足")
		return
	}
	name := ug.Good.Name
	switch {
	case name == "改名卡":
		nick := strings.TrimSpace(req.Nickname)
		if len([]rune(nick)) < 2 || len([]rune(nick)) > 12 {
			resp.ParamError(c, "新昵称需2~12个字符")
			return
		}
		h.DB.Model(&model.User{}).Where("id = ?", uid).Update("nickname", nick)
		h.DB.Model(&ug).Update("count", gorm.Expr("count - 1"))
		resp.OK(c, gin.H{"used": true, "name": name, "msg": "昵称已修改为「" + nick + "」"})
	case ug.Good.Category == "鲜花":
		resp.OK(c, gin.H{"used": false, "name": name, "msg": "「" + name + "」请到帖子下方【送花】使用"})
	default:
		resp.OK(c, gin.H{"used": false, "name": name, "msg": "「" + name + "」的使用功能即将开放，敬请期待"})
	}
}

// 后台：商品列表（分页）
func (h *GoodHandler) AdminList(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	q := h.DB.Model(&model.Good{})
	var total int64
	q.Count(&total)
	var list []model.Good
	q.Order("sort ASC, id ASC").Offset(offset).Limit(size).Find(&list)
	resp.OK(c, gin.H{"list": list, "total": total, "page": page, "size": size, "categories": goodCategories(list)})
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
