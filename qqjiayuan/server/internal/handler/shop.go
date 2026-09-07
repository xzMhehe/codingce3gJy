package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type ShopHandler struct{ DB *gorm.DB }

// ============ 分类 ============

// Categories 商店分类
func (h *ShopHandler) Categories(c *gin.Context) {
	var cats []model.ShopCategory
	h.DB.Order("sort ASC, id ASC").Find(&cats)
	resp.OK(c, cats)
}

// ============ 店铺 ============

// MyShop 我的店铺
func (h *ShopHandler) MyShop(c *gin.Context) {
	uid := middleware.GetUID(c)
	var shop model.Shop
	h.DB.Where("user_id = ?", uid).First(&shop)
	var goods []model.ShopGoods
	h.DB.Where("user_id = ?", uid).Order("created_at DESC").Limit(50).Find(&goods)
	var myOrders []model.ShopOrder
	h.DB.Where("buyer_id = ?", uid).Order("created_at DESC").Limit(30).Find(&myOrders)
	var sellOrders []model.ShopOrder
	h.DB.Where("seller_id = ?", uid).Order("created_at DESC").Limit(30).Find(&sellOrders)
	resp.OK(c, gin.H{
		"shop":  shop,
		"goods": goods,
		"buy_orders": myOrders, "sell_orders": sellOrders,
	})
}

// OpenShop 开通/改名店铺
func (h *ShopHandler) OpenShop(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Name string `json:"name" binding:"required,max=30"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "店铺名称不能为空且不超过 30 字")
		return
	}
	var shop model.Shop
	if err := h.DB.Where("user_id = ?", uid).First(&shop).Error; err != nil {
		shop = model.Shop{UserID: uid, Name: req.Name, Status: 1}
		h.DB.Create(&shop)
		resp.OK(c, gin.H{"id": shop.ID, "name": shop.Name, "opened": true})
		return
	}
	h.DB.Model(&shop).Update("name", req.Name)
	resp.OK(c, gin.H{"id": shop.ID, "name": shop.Name, "opened": true})
}

// ============ 商品 ============

type marketRow struct {
	model.ShopGoods
	Seller   string `json:"seller"`
	CatName  string `json:"cat_name"`
	HasShop  bool   `json:"has_shop"`
}

// GoodsList 市场列表（上架中）
func (h *ShopHandler) GoodsList(c *gin.Context) {
	page, size := pageParams(c, 10)
	q := h.DB.Model(&model.ShopGoods{}).Where("status = 1")
	if catID, err := strconv.Atoi(c.Query("cat_id")); err == nil && catID > 0 {
		q = q.Where("cat_id = ?", catID)
	}
	if kw := c.Query("kw"); kw != "" {
		q = q.Where("name LIKE ?", "%"+kw+"%")
	}
	var total int64
	q.Count(&total)
	var list []model.ShopGoods
	q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&list)
	out := make([]marketRow, 0, len(list))
	for _, g := range list {
		row := marketRow{ShopGoods: g}
		var u model.User
		h.DB.Select("nickname").First(&u, g.UserID)
		row.Seller = u.Nickname
		var shop model.Shop
		h.DB.Select("id,name").Where("user_id = ?", g.UserID).First(&shop)
		row.HasShop = shop.ID > 0
		if g.CatID > 0 {
			var cat model.ShopCategory
			h.DB.Select("name").First(&cat, g.CatID)
			row.CatName = cat.Name
		}
		out = append(out, row)
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// GoodsDetail 商品详情（浏览+1，含成交记录 TOP5 与评论）
func (h *ShopHandler) GoodsDetail(c *gin.Context) {
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	var g model.ShopGoods
	if err := h.DB.Where("id = ? AND status = 1", id).First(&g).Error; err != nil {
		resp.NotFound(c, "商品不存在或已下架")
		return
	}
	h.DB.Model(&g).UpdateColumn("clicks", gorm.Expr("clicks + 1"))
	g.Clicks++
	var u model.User
	h.DB.Select("id,nickname,color").First(&u, g.UserID)
	var shop model.Shop
	h.DB.Where("user_id = ?", g.UserID).First(&shop)
	var cat model.ShopCategory
	if g.CatID > 0 {
		h.DB.Select("name").First(&cat, g.CatID)
	}
	// 成交记录（已完成订单）
	var deals []model.ShopOrder
	h.DB.Where("goods_id = ? AND status = 4", g.ID).Order("created_at DESC").Limit(5).Find(&deals)
	type dealRow struct {
		model.ShopOrder
		Buyer string `json:"buyer"`
	}
	dealsOut := make([]dealRow, 0, len(deals))
	for _, d := range deals {
		var bu model.User
		h.DB.Select("nickname").First(&bu, d.BuyerID)
		dealsOut = append(dealsOut, dealRow{ShopOrder: d, Buyer: bu.Nickname})
	}
	// 评论
	var comments []model.ShopComment
	h.DB.Where("goods_id = ? AND status = 1", g.ID).Order("created_at DESC").Limit(20).Find(&comments)
	type commentRow struct {
		model.ShopComment
		User string `json:"user"`
	}
	commentsOut := make([]commentRow, 0, len(comments))
	for _, cm := range comments {
		var cu model.User
		h.DB.Select("nickname").First(&cu, cm.UserID)
		commentsOut = append(commentsOut, commentRow{ShopComment: cm, User: cu.Nickname})
	}
	resp.OK(c, gin.H{
		"goods": g, "seller": gin.H{"id": u.ID, "nickname": u.Nickname, "color": u.Color},
		"shop": shop, "cat_name": cat.Name,
		"deals": dealsOut, "comments": commentsOut,
	})
}

// GoodsAdd 发布商品（需开通店铺）
func (h *ShopHandler) GoodsAdd(c *gin.Context) {
	uid := middleware.GetUID(c)
	var shop model.Shop
	if err := h.DB.Where("user_id = ?", uid).First(&shop).Error; err != nil {
		resp.Forbidden(c, "请先开通店铺再发布商品")
		return
	}
	var req struct {
		Name    string `json:"name" binding:"required,max=60"`
		CatID   uint   `json:"cat_id"`
		Price   int    `json:"price" binding:"required,min=1"`
		Amount  int    `json:"amount" binding:"required,min=1"`
		Intro   string `json:"intro"`
		Image   string `json:"image"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写完整商品信息")
		return
	}
	if !verifyTime("goods_"+strconv.FormatUint(uint64(uid), 10), 5) {
		resp.ParamError(c, "发布太频繁了，歇 5 秒再来")
		return
	}
	g := model.ShopGoods{
		UserID: uid, Name: req.Name, CatID: req.CatID, BidMT: 1, Money: 0,
		Price: req.Price, Amount: req.Amount, Intro: req.Intro, Image: req.Image, Status: 1,
	}
	h.DB.Create(&g)
	resp.OK(c, gin.H{"id": g.ID})
}

// GoodsUpdate 修改商品（价格/库存/上下架）——店主
func (h *ShopHandler) GoodsUpdate(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	var g model.ShopGoods
	if err := h.DB.Where("id = ? AND user_id = ?", id, uid).First(&g).Error; err != nil {
		resp.Forbidden(c, "只能修改自己的商品")
		return
	}
	var req struct {
		Price  *int `json:"price"`
		Amount *int `json:"amount"`
		Status *int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数不对")
		return
	}
	updates := map[string]interface{}{}
	if req.Price != nil && *req.Price >= 1 {
		updates["price"] = *req.Price
	}
	if req.Amount != nil && *req.Amount >= 0 {
		updates["amount"] = *req.Amount
	}
	if req.Status != nil && (*req.Status == 0 || *req.Status == 1) {
		updates["status"] = *req.Status
	}
	h.DB.Model(&g).Updates(updates)
	resp.OK(c, nil)
}

// GoodsDelete 删除商品——店主
func (h *ShopHandler) GoodsDelete(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	h.DB.Where("id = ? AND user_id = ?", id, uid).Delete(&model.ShopGoods{})
	resp.OK(c, nil)
}

// ============ 订单（诺哈 wap_shop_order：下单→付款→发货→收货） ============

// OrderCreate 下单（生成待付款订单）
func (h *ShopHandler) OrderCreate(c *gin.Context) {
	uid := middleware.GetUID(c)
	gid, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	var req struct {
		Amount int `json:"amount" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写购买数量")
		return
	}
	var g model.ShopGoods
	if err := h.DB.Where("id = ? AND status = 1", gid).First(&g).Error; err != nil {
		resp.NotFound(c, "商品不存在或已下架")
		return
	}
	if g.UserID == uid {
		resp.ParamError(c, "不能买自己店铺的商品")
		return
	}
	if req.Amount > g.Amount {
		resp.ParamError(c, "库存不足，当前仅剩 "+strconv.Itoa(g.Amount)+" 件")
		return
	}
	if !verifyTime("order_"+strconv.FormatUint(uint64(uid), 10), 3) {
		resp.ParamError(c, "操作太频繁了，歇 3 秒再来")
		return
	}
	order := model.ShopOrder{
		GoodsID: g.ID, SellerID: g.UserID, BuyerID: uid,
		GoodsName: g.Name, Money: g.Money, Price: g.Price, Amount: req.Amount, Status: 1,
	}
	h.DB.Create(&order)
	resp.OK(c, gin.H{"id": order.ID, "total": g.Price * req.Amount, "status": 1})
}

// OrderPay 付款（扣买家金币→待发货，流水记 buy）
func (h *ShopHandler) OrderPay(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	var o model.ShopOrder
	if err := h.DB.Where("id = ? AND buyer_id = ? AND status = 1", id, uid).First(&o).Error; err != nil {
		resp.ParamError(c, "订单不存在或不能付款")
		return
	}
	total := o.Price * o.Amount
	var buyer model.User
	h.DB.First(&buyer, uid)
	if buyer.Coins < total {
		resp.ParamError(c, "G币不足，需要 "+strconv.Itoa(total)+" G币")
		return
	}
	h.DB.Model(&buyer).Update("coins", gorm.Expr("coins - ?", total))
	addWalletLog(h.DB, uid, "buy", "商店购买《"+o.GoodsName+"》", "coins", -total)
	h.DB.Model(&o).Updates(map[string]interface{}{"status": 2})
	// 通知卖家
	h.DB.Create(&model.Notification{
		UserID: o.SellerID, Type: "system", Title: "新订单待发货",
		Content: "有买家付款购买了《" + o.GoodsName + "》x" + strconv.Itoa(o.Amount) + "，快去发货吧。",
	})
	resp.OK(c, gin.H{"id": o.ID, "status": 2})
}

// OrderShip 发货（卖家）
func (h *ShopHandler) OrderShip(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	var o model.ShopOrder
	if err := h.DB.Where("id = ? AND seller_id = ? AND status = 2", id, uid).First(&o).Error; err != nil {
		resp.ParamError(c, "订单不存在或不能发货")
		return
	}
	h.DB.Model(&o).Update("status", 3)
	h.DB.Create(&model.Notification{
		UserID: o.BuyerID, Type: "system", Title: "商品已发货",
		Content: "你购买的《" + o.GoodsName + "》已发货，请留意查收。",
	})
	resp.OK(c, gin.H{"id": o.ID, "status": 3})
}

// OrderReceive 收货（买家；成交后扣库存+加销量，货款转卖家）
func (h *ShopHandler) OrderReceive(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	var o model.ShopOrder
	if err := h.DB.Where("id = ? AND buyer_id = ? AND status = 3", id, uid).First(&o).Error; err != nil {
		resp.ParamError(c, "订单不存在或不能收货")
		return
	}
	total := o.Price * o.Amount
	h.DB.Model(&model.User{}).Where("id = ?", o.SellerID).Update("coins", gorm.Expr("coins + ?", total))
	addWalletLog(h.DB, o.SellerID, "sell", "商店卖出《"+o.GoodsName+"》", "coins", total)
	h.DB.Model(&o).Update("status", 4)
	h.DB.Model(&model.ShopGoods{}).Where("id = ?", o.GoodsID).
		Updates(map[string]interface{}{"amount": gorm.Expr("amount - ?", o.Amount), "sales": gorm.Expr("sales + ?", o.Amount)})
	resp.OK(c, gin.H{"id": o.ID, "status": 4})
}

// OrderCancel 取消订单（待付款可直接取消；已付款需退款）
func (h *ShopHandler) OrderCancel(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	var o model.ShopOrder
	if err := h.DB.Where("id = ? AND buyer_id = ? AND status IN (1,2)", id, uid).First(&o).Error; err != nil {
		resp.ParamError(c, "订单不存在或不能取消")
		return
	}
	if o.Status == 2 {
		total := o.Price * o.Amount
		h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins + ?", total))
		addWalletLog(h.DB, uid, "refund", "取消订单退款《"+o.GoodsName+"》", "coins", total)
	}
	h.DB.Model(&o).Update("status", 5)
	resp.OK(c, gin.H{"id": o.ID, "status": 5})
}

// ============ 评价 ============

// CommentAdd 商品评价（订单完成后可评：1好评 2中评 3差评）
func (h *ShopHandler) CommentAdd(c *gin.Context) {
	uid := middleware.GetUID(c)
	gid, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	var req struct {
		DType   int    `json:"d_type" binding:"required,min=1,max=3"`
		OrderID uint   `json:"order_id"`
		Content string `json:"content" binding:"required,max=300"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写评价内容")
		return
	}
	var o model.ShopOrder
	if err := h.DB.Where("id = ? AND buyer_id = ? AND goods_id = ? AND status = 4", req.OrderID, uid, gid).First(&o).Error; err != nil {
		resp.ParamError(c, "只有购买并收货后才能评价")
		return
	}
	var cnt int64
	h.DB.Model(&model.ShopComment{}).Where("order_id = ? AND user_id = ?", o.ID, uid).Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "该订单已评价过了")
		return
	}
	h.DB.Create(&model.ShopComment{GoodsID: uint(gid), OrderID: o.ID, UserID: uid, DType: req.DType, Content: req.Content, Status: 1})
	h.DB.Model(&model.ShopGoods{}).Where("id = ?", gid).UpdateColumn("comment", gorm.Expr("comment + 1"))
	resp.OK(c, nil)
}

// CommentReply 卖家回复评价
func (h *ShopHandler) CommentReply(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, ok := positiveParam(c, "id")
	if !ok {
		return
	}
	var req struct {
		Content string `json:"content" binding:"required,max=300"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "回复内容不能为空")
		return
	}
	var cm model.ShopComment
	if err := h.DB.First(&cm, id).Error; err != nil {
		resp.NotFound(c, "评价不存在")
		return
	}
	var g model.ShopGoods
	if err := h.DB.Where("id = ? AND user_id = ?", cm.GoodsID, uid).First(&g).Error; err != nil {
		resp.Forbidden(c, "只有店主才能回复")
		return
	}
	h.DB.Model(&cm).Update("reply", req.Content)
	resp.OK(c, nil)
}

// MyOrders 我的订单（买家=my 卖家=sell）
func (h *ShopHandler) MyOrders(c *gin.Context) {
	uid := middleware.GetUID(c)
	role := c.DefaultQuery("role", "buy")
	q := h.DB.Model(&model.ShopOrder{})
	if role == "sell" {
		q = q.Where("seller_id = ?", uid)
	} else {
		q = q.Where("buyer_id = ?", uid)
	}
	var list []model.ShopOrder
	q.Order("created_at DESC").Limit(50).Find(&list)
	type row struct {
		model.ShopOrder
		OtherNick string `json:"other_nick"`
	}
	out := make([]row, 0, len(list))
	for _, o := range list {
		r := row{ShopOrder: o}
		target := o.SellerID
		if role == "sell" {
			target = o.BuyerID
		}
		var u model.User
		h.DB.Select("nickname").First(&u, target)
		r.OtherNick = u.Nickname
		out = append(out, r)
	}
	resp.OK(c, out)
}