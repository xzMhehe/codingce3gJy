package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type GoodHandler struct{ DB *gorm.DB }

// 分类列表（后台商品管理用）
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

// 在售商品查询条件（复刻诺哈：status 上架 且 未过结束时间）
func goodOnSale(db *gorm.DB) *gorm.DB {
	return db.Where("status = 1 AND (end_time IS NULL OR end_time > ?)", time.Now())
}

// 公开：商店列表（复刻诺哈 shop_list.asp：平铺编号列表，按ID倒序，每页10条）
func (h *GoodHandler) List(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	var total int64
	goodOnSale(h.DB.Model(&model.Good{})).Count(&total)
	var list []model.Good
	goodOnSale(h.DB).Order("id DESC").Offset(offset).Limit(size).Find(&list)
	uid := middleware.GetUID(c)
	coins, youquan := -1, -1
	if uid > 0 {
		var u model.User
		if h.DB.First(&u, uid).Error == nil {
			coins, youquan = u.Coins, u.YouQuan
		}
	}
	resp.OK(c, gin.H{
		"list": list, "total": total, "page": page, "size": size,
		"coins": coins, "youquan": youquan,
	})
}

// 公开：商品详情（复刻诺哈 shop.asp）
func (h *GoodHandler) Detail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var g model.Good
	if err := goodOnSale(h.DB).First(&g, id).Error; err != nil {
		resp.NotFound(c, "商品不存在或已下架")
		return
	}
	resp.OK(c, g)
}

// 购买道具（复刻诺哈 shop_buy_ok.asp：支付密码确认，扣G币入仓库，扣库存加销量）
func (h *GoodHandler) Buy(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Num     int    `json:"num"`
		PayPass string `json:"pay_pass"`
	}
	_ = c.ShouldBindJSON(&req)
	var g model.Good
	if err := h.DB.First(&g, id).Error; err != nil {
		resp.NotFound(c, "商品不存在")
		return
	}
	if g.Status == 0 {
		resp.ParamError(c, "商品已下架")
		return
	}
	if g.EndTime != nil && g.EndTime.Before(time.Now()) {
		resp.ParamError(c, "商品已过销售时间！")
		return
	}
	if msg := verifyPayPass(h.DB, uid, req.PayPass); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	if req.Num < 1 {
		resp.ParamError(c, "购买数量错误！")
		return
	}
	if req.Num > g.Stock {
		resp.ParamError(c, "购买数量超过库存数量！")
		return
	}
	cost := g.Price * req.Num
	var u model.User
	if err := h.DB.First(&u, uid).Error; err != nil {
		resp.Unauthorized(c, "请先登录")
		return
	}
	if u.Coins < cost {
		resp.ParamError(c, "您的G币不足！")
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
	// 扣库存加销量（诺哈 shop_buy_ok）
	h.DB.Model(&model.Good{}).Where("id = ?", g.ID).Updates(map[string]interface{}{
		"stock": gorm.Expr("stock - ?", req.Num), "sales": gorm.Expr("sales + ?", req.Num),
	})
	h.DB.First(&u, uid)
	resp.OK(c, gin.H{"name": g.Name, "num": req.Num, "coins": u.Coins, "youquan": u.YouQuan, "bag_count": req.Num + (func() int {
		if ug.ID == 0 {
			return 0
		}
		return ug.Count
	})()})
}

// 赠送预览（复刻诺哈 shop_send.asp 确认页数据：校验商品/号码/数量/库存，不扣款不验密码）
func (h *GoodHandler) SendPreview(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	to := strings.TrimSpace(c.Query("to"))
	num, _ := strconv.Atoi(c.Query("num"))
	var g model.Good
	if err := goodOnSale(h.DB).First(&g, id).Error; err != nil {
		resp.NotFound(c, "商品不存在或已下架")
		return
	}
	if to == "" {
		resp.ParamError(c, "请填写赠送号码！")
		return
	}
	if num < 1 {
		resp.ParamError(c, "赠送数量错误！")
		return
	}
	if num > g.Stock {
		resp.ParamError(c, "赠送数量超过库存数量！")
		return
	}
	var toU model.User
	if err := h.DB.Where("username = ?", to).First(&toU).Error; err != nil {
		resp.ParamError(c, "会员号码不正确！")
		return
	}
	resp.OK(c, gin.H{
		"name": g.Name, "nickname": toU.Nickname, "username": toU.Username,
		"num": num, "total": g.Price * num, "currency_name": "G币",
	})
}

// 赠送道具（复刻诺哈 shop_send_ok.asp：扣款、道具入对方仓库、扣库存加销量、站内信）
func (h *GoodHandler) Send(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		To      string `json:"to"`
		Num     int    `json:"num"`
		PayPass string `json:"pay_pass"`
	}
	_ = c.ShouldBindJSON(&req)
	var g model.Good
	if err := h.DB.First(&g, id).Error; err != nil {
		resp.NotFound(c, "商品不存在")
		return
	}
	if g.Status == 0 {
		resp.ParamError(c, "商品已下架")
		return
	}
	if g.EndTime != nil && g.EndTime.Before(time.Now()) {
		resp.ParamError(c, "商品已过销售时间！")
		return
	}
	if msg := verifyPayPass(h.DB, uid, req.PayPass); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	if strings.TrimSpace(req.To) == "" {
		resp.ParamError(c, "请填写赠送号码！")
		return
	}
	if req.Num < 1 {
		resp.ParamError(c, "赠送数量错误！")
		return
	}
	if req.Num > g.Stock {
		resp.ParamError(c, "赠送数量超过库存数量！")
		return
	}
	var toU model.User
	if err := h.DB.Where("username = ?", strings.TrimSpace(req.To)).First(&toU).Error; err != nil {
		resp.ParamError(c, "会员号码不正确！")
		return
	}
	if toU.ID == uid {
		resp.ParamError(c, "不能赠送给自己！")
		return
	}
	cost := g.Price * req.Num
	var u model.User
	if err := h.DB.First(&u, uid).Error; err != nil {
		resp.Unauthorized(c, "请先登录")
		return
	}
	if u.Coins < cost {
		resp.ParamError(c, "您的G币不足！")
		return
	}
	// 扣赠送方G币
	h.DB.Model(&u).Update("coins", gorm.Expr("coins - ?", cost))
	addWalletLog(h.DB, uid, "buy", "赠送「"+g.Name+"」×"+strconv.Itoa(req.Num)+" 给"+toU.Nickname+"("+toU.Username+")", "coins", -cost)
	// 道具入对方仓库
	var ug model.UserGood
	if err := h.DB.Where("user_id = ? AND good_id = ?", toU.ID, g.ID).First(&ug).Error; err != nil {
		h.DB.Create(&model.UserGood{UserID: toU.ID, GoodID: g.ID, Count: req.Num})
	} else {
		h.DB.Model(&ug).Update("count", gorm.Expr("count + ?", req.Num))
	}
	// 扣库存加销量
	h.DB.Model(&model.Good{}).Where("id = ?", g.ID).Updates(map[string]interface{}{
		"stock": gorm.Expr("stock - ?", req.Num), "sales": gorm.Expr("sales + ?", req.Num),
	})
	// 站内信（诺哈 MessageSend）
	h.DB.Create(&model.PrivateMessage{SenderID: uid, ReceiverID: toU.ID,
		Content: "恭喜！赠送了您" + strconv.Itoa(req.Num) + "个「" + g.Name + "」。"})
	resp.OK(c, gin.H{"name": g.Name, "num": req.Num, "coins": u.Coins - cost})
}

// Bag 我的仓库（复刻诺哈 my_bag.asp：平铺编号列表，ID倒序，每页10条）
func (h *GoodHandler) Bag(c *gin.Context) {
	uid := middleware.GetUID(c)
	page, offset, size := pageOf(c, 10)
	q := h.DB.Model(&model.UserGood{}).Where("user_id = ? AND count > 0", uid)
	var total int64
	q.Count(&total)
	var list []model.UserGood
	q.Preload("Good").Order("id DESC").Offset(offset).Limit(size).Find(&list)
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
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
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
	Name         string     `json:"name" binding:"required"`
	Desc         string     `json:"desc"`
	Icon         string     `json:"icon"`
	Category     string     `json:"category"`
	Price        int        `json:"price"`
	YouQuanPrice int        `json:"youquan_price"`
	Stock        int        `json:"stock"`
	Sales        int        `json:"sales"`
	Status       int        `json:"status"`
	Sort         int        `json:"sort"`
	EndTime      *time.Time `json:"end_time"`
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
	h.DB.Create(&model.Good{Name: req.Name, Desc: req.Desc, Icon: req.Icon, Category: req.Category,
		Price: req.Price, YouQuanPrice: req.YouQuanPrice, Stock: req.Stock, Sales: req.Sales,
		Status: req.Status, Sort: req.Sort, AddTime: time.Now(), EndTime: req.EndTime})
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
		"price": req.Price, "youquan_price": req.YouQuanPrice, "stock": req.Stock, "sales": req.Sales,
		"status": req.Status, "sort": req.Sort, "end_time": req.EndTime,
	})
	resp.OK(c, nil)
}

func (h *GoodHandler) AdminDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.Good{}, id)
	resp.OK(c, nil)
}
