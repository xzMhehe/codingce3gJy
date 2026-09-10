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

// StoreHandler 店铺商城（对齐诺哈 wap/shop：店铺街/店铺/商品/交易/评价/卖家中心）
type StoreHandler struct{ DB *gorm.DB }

func (h *StoreHandler) nickMap(uidSet map[uint]bool) map[uint]string {
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

// Shops 店铺街（对齐 wap/shop/index.asp：店铺列表+搜索）
func (h *StoreHandler) Shops(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	size := 10
	wd := strings.TrimSpace(c.Query("wd"))
	q := h.DB.Model(&model.Shop{}).Where("status = 1")
	if wd != "" {
		q = q.Where("name LIKE ?", "%"+wd+"%")
	}
	var total int64
	q.Count(&total)
	var shops []model.Shop
	q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&shops)

	type shopRow struct {
		ID        uint   `json:"id"`
		UserID    uint   `json:"user_id"`
		Name      string `json:"name"`
		Nick      string `json:"nick"`
		GoodsN    int64  `json:"goods_n"`
		CreatedAt string `json:"created_at"`
	}
	uids := map[uint]bool{}
	for _, s := range shops {
		uids[s.UserID] = true
	}
	nick := h.nickMap(uids)
	out := make([]shopRow, 0, len(shops))
	for _, s := range shops {
		var gn int64
		h.DB.Model(&model.ShopGoods{}).Where("user_id = ? AND status = 1", s.UserID).Count(&gn)
		out = append(out, shopRow{s.ID, s.UserID, s.Name, nick[s.UserID], gn, s.CreatedAt.Format("2006-01-02")})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// MyShop 我的店铺
func (h *StoreHandler) MyShop(c *gin.Context) {
	uid := middleware.GetUID(c)
	var s model.Shop
	if err := h.DB.Where("user_id = ?", uid).First(&s).Error; err != nil {
		resp.OK(c, gin.H{"exists": false})
		return
	}
	resp.OK(c, gin.H{"exists": true, "shop": s})
}

// ShopCreate 开店（一人一店，对齐 wap/shop/seller 开店流程）
func (h *StoreHandler) ShopCreate(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写店铺名称")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 30 {
		resp.ParamError(c, "店铺名称需1-30字")
		return
	}
	var n int64
	h.DB.Model(&model.Shop{}).Where("user_id = ?", uid).Count(&n)
	if n > 0 {
		resp.ParamError(c, "您已经拥有店铺了！")
		return
	}
	s := model.Shop{UserID: uid, Name: name}
	h.DB.Create(&s)
	resp.OK(c, gin.H{"msg": "店铺「" + name + "」开设成功！", "shop": s})
}

// ShopDetail 店铺详情（店主+上架商品，对齐 wap/shop/shop.asp）
func (h *StoreHandler) ShopDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var s model.Shop
	if err := h.DB.First(&s, id).Error; err != nil {
		resp.NotFound(c, "店铺不存在")
		return
	}
	nick := h.nickMap(map[uint]bool{s.UserID: true})
	var goods []model.ShopGoods
	h.DB.Where("user_id = ? AND status = 1", s.UserID).Order("id DESC").Find(&goods)
	resp.OK(c, gin.H{"id": s.ID, "user_id": s.UserID, "name": s.Name, "nick": nick[s.UserID],
		"created_at": s.CreatedAt.Format("2006-01-02"), "goods": goods})
}

// GoodsDetail 商品详情（对齐 wap/shop/image_list 详情 + 点击数）
func (h *StoreHandler) GoodsDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var g model.ShopGoods
	if err := h.DB.First(&g, id).Error; err != nil {
		resp.NotFound(c, "商品不存在")
		return
	}
	h.DB.Model(&g).Update("clicks", gorm.Expr("clicks + ?", 1))
	nick := h.nickMap(map[uint]bool{g.UserID: true})
	var shop model.Shop
	h.DB.Where("user_id = ?", g.UserID).First(&shop)
	resp.OK(c, gin.H{"goods": g, "seller_nick": nick[g.UserID], "shop_name": shop.Name, "shop_id": shop.ID})
}

// GoodsAdd 上架商品（对齐 wap/shop/seller 上货）
func (h *StoreHandler) GoodsAdd(c *gin.Context) {
	uid := middleware.GetUID(c)
	var shop model.Shop
	if err := h.DB.Where("user_id = ?", uid).First(&shop).Error; err != nil {
		resp.ParamError(c, "请先开设店铺！")
		return
	}
	var req struct {
		Name   string `json:"name" binding:"required"`
		Price  int    `json:"price"`
		Amount int    `json:"amount"`
		Intro  string `json:"intro"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写商品名称")
		return
	}
	if req.Price < 0 || req.Amount < 1 {
		resp.ParamError(c, "价格或数量错误！")
		return
	}
	g := model.ShopGoods{UserID: uid, Name: strings.TrimSpace(req.Name), Price: req.Price,
		Amount: req.Amount, Intro: strings.TrimSpace(req.Intro), Status: 1}
	h.DB.Create(&g)
	resp.OK(c, gin.H{"msg": "商品「" + g.Name + "」上架成功", "goods": g})
}

// GoodsEdit 编辑商品（卖家本人）
func (h *StoreHandler) GoodsEdit(c *gin.Context) {
	uid := middleware.GetUID(c)
	var g model.ShopGoods
	if err := h.DB.Where("id = ? AND user_id = ?", c.Param("id"), uid).First(&g).Error; err != nil {
		resp.ParamError(c, "商品不存在")
		return
	}
	var req struct {
		Name   *string `json:"name"`
		Price  *int    `json:"price"`
		Amount *int    `json:"amount"`
		Intro  *string `json:"intro"`
		Status *int    `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	up := map[string]interface{}{}
	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		up["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Price != nil && *req.Price >= 0 {
		up["price"] = *req.Price
	}
	if req.Amount != nil && *req.Amount >= 0 {
		up["amount"] = *req.Amount
	}
	if req.Intro != nil {
		up["intro"] = *req.Intro
	}
	if req.Status != nil {
		up["status"] = *req.Status
	}
	if len(up) > 0 {
		h.DB.Model(&g).Updates(up)
	}
	resp.OK(c, gin.H{"msg": "已保存"})
}

// GoodsDel 删除商品（卖家本人）
func (h *StoreHandler) GoodsDel(c *gin.Context) {
	uid := middleware.GetUID(c)
	var g model.ShopGoods
	if err := h.DB.Where("id = ? AND user_id = ?", c.Param("id"), uid).First(&g).Error; err != nil {
		resp.ParamError(c, "商品不存在")
		return
	}
	h.DB.Delete(&g)
	resp.OK(c, gin.H{"msg": "已删除"})
}

// MyGoods 卖家中心：我的商品
func (h *StoreHandler) MyGoods(c *gin.Context) {
	uid := middleware.GetUID(c)
	var goods []model.ShopGoods
	h.DB.Where("user_id = ?", uid).Order("id DESC").Find(&goods)
	resp.OK(c, goods)
}

// OrderBuy 购买下单（担保交易：下单即扣款冻结，收货后卖家到账，对齐 wap/shop/buy.asp）
func (h *StoreHandler) OrderBuy(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		GoodsID uint   `json:"goods_id" binding:"required"`
		Amount  int    `json:"amount"`
		PayPass string `json:"pay_pass"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.Amount < 1 {
		req.Amount = 1
	}
	if msg := verifyPayPass(h.DB, uid, req.PayPass); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	var g model.ShopGoods
	if err := h.DB.First(&g, req.GoodsID).Error; err != nil || g.Status != 1 {
		resp.ParamError(c, "商品不存在或已下架！")
		return
	}
	if g.UserID == uid {
		resp.ParamError(c, "不能购买自己店铺的商品！")
		return
	}
	if req.Amount > g.Amount {
		resp.ParamError(c, "购买数量超过库存数量！")
		return
	}
	cost := g.Price * req.Amount
	var u model.User
	h.DB.First(&u, uid)
	if u.Coins < cost {
		resp.ParamError(c, "您的G币不足！")
		return
	}
	// 事务：扣买家、扣库存加销量、建订单（状态2待发货）
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).Where("id = ?", uid).
			Update("coins", gorm.Expr("coins - ?", cost)).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.ShopGoods{}).Where("id = ?", g.ID).Updates(map[string]interface{}{
			"amount": gorm.Expr("amount - ?", req.Amount), "sales": gorm.Expr("sales + ?", req.Amount),
		}).Error; err != nil {
			return err
		}
		o := model.ShopOrder{GoodsID: g.ID, SellerID: g.UserID, BuyerID: uid,
			GoodsName: g.Name, Price: g.Price, Amount: req.Amount, Status: 2}
		return tx.Create(&o).Error
	})
	if err != nil {
		resp.ParamError(c, "下单失败，请重试")
		return
	}
	addWalletLog(h.DB, uid, "shop", "购买「"+g.Name+"」×"+strconv.Itoa(req.Amount), "coins", -cost)
	h.DB.First(&u, uid)
	resp.OK(c, gin.H{"msg": "下单成功，等待卖家发货！", "coins": u.Coins})
}

// MyOrders 我的订单（type=buy 买入 / sell 卖出）
func (h *StoreHandler) MyOrders(c *gin.Context) {
	uid := middleware.GetUID(c)
	typ := c.DefaultQuery("type", "buy")
	col := "buyer_id"
	if typ == "sell" {
		col = "seller_id"
	}
	var orders []model.ShopOrder
	h.DB.Where(col+" = ?", uid).Order("id DESC").Limit(50).Find(&orders)
	uids := map[uint]bool{}
	for _, o := range orders {
		uids[o.BuyerID] = true
		uids[o.SellerID] = true
	}
	nick := h.nickMap(uids)
	type orderRow struct {
		model.ShopOrder
		BuyerNick  string `json:"buyer_nick"`
		SellerNick string `json:"seller_nick"`
		Commented  bool   `json:"commented"`
	}
	out := make([]orderRow, 0, len(orders))
	for _, o := range orders {
		var n int64
		h.DB.Model(&model.ShopComment{}).Where("order_id = ?", o.ID).Count(&n)
		out = append(out, orderRow{o, nick[o.BuyerID], nick[o.SellerID], n > 0})
	}
	resp.OK(c, out)
}

// OrderDeliver 发货（卖家，2→3）
func (h *StoreHandler) OrderDeliver(c *gin.Context) {
	uid := middleware.GetUID(c)
	var o model.ShopOrder
	if err := h.DB.Where("id = ? AND seller_id = ?", c.Param("id"), uid).First(&o).Error; err != nil {
		resp.ParamError(c, "订单不存在")
		return
	}
	if o.Status != 2 {
		resp.ParamError(c, "当前订单状态不可发货")
		return
	}
	h.DB.Model(&o).Update("status", 3)
	resp.OK(c, gin.H{"msg": "已发货，等待买家确认收货"})
}

// OrderReceive 确认收货（买家，3→4，卖家到账）
func (h *StoreHandler) OrderReceive(c *gin.Context) {
	uid := middleware.GetUID(c)
	var o model.ShopOrder
	if err := h.DB.Where("id = ? AND buyer_id = ?", c.Param("id"), uid).First(&o).Error; err != nil {
		resp.ParamError(c, "订单不存在")
		return
	}
	if o.Status != 3 {
		resp.ParamError(c, "当前订单状态不可收货")
		return
	}
	total := o.Price * o.Amount
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&o).Update("status", 4).Error; err != nil {
			return err
		}
		return tx.Model(&model.User{}).Where("id = ?", o.SellerID).
			Update("coins", gorm.Expr("coins + ?", total)).Error
	})
	if err != nil {
		resp.ParamError(c, "操作失败，请重试")
		return
	}
	addWalletLog(h.DB, o.SellerID, "shop", "售出「"+o.GoodsName+"」×"+strconv.Itoa(o.Amount), "coins", total)
	resp.OK(c, gin.H{"msg": "确认收货成功，卖家已收到 " + strconv.Itoa(total) + "G币"})
}

// OrderCancel 取消订单（买家，2→5 退款）
func (h *StoreHandler) OrderCancel(c *gin.Context) {
	uid := middleware.GetUID(c)
	var o model.ShopOrder
	if err := h.DB.Where("id = ? AND buyer_id = ?", c.Param("id"), uid).First(&o).Error; err != nil {
		resp.ParamError(c, "订单不存在")
		return
	}
	if o.Status != 2 {
		resp.ParamError(c, "卖家已发货，订单不可取消")
		return
	}
	refund := o.Price * o.Amount
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&o).Update("status", 5).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).Where("id = ?", uid).
			Update("coins", gorm.Expr("coins + ?", refund)).Error; err != nil {
			return err
		}
		return tx.Model(&model.ShopGoods{}).Where("id = ?", o.GoodsID).Updates(map[string]interface{}{
			"amount": gorm.Expr("amount + ?", o.Amount), "sales": gorm.Expr("sales - ?", o.Amount),
		}).Error
	})
	if err != nil {
		resp.ParamError(c, "操作失败，请重试")
		return
	}
	addWalletLog(h.DB, uid, "shop", "取消订单退款：「"+o.GoodsName+"」", "coins", refund)
	resp.OK(c, gin.H{"msg": "订单已取消，" + strconv.Itoa(refund) + "G币已退回"})
}

// GoodsComments 商品评价列表
func (h *StoreHandler) GoodsComments(c *gin.Context) {
	gid, _ := strconv.Atoi(c.Param("id"))
	var list []model.ShopComment
	h.DB.Where("goods_id = ? AND status = 1", gid).Order("id DESC").Limit(50).Find(&list)
	uids := map[uint]bool{}
	for _, m := range list {
		uids[m.UserID] = true
	}
	nick := h.nickMap(uids)
	out := make([]gin.H, 0, len(list))
	for _, m := range list {
		out = append(out, gin.H{"id": m.ID, "user_id": m.UserID, "nick": nick[m.UserID],
			"d_type": m.DType, "content": m.Content, "reply": m.Reply,
			"time_txt": m.CreatedAt.Format("2006-01-02 15:04")})
	}
	resp.OK(c, out)
}

// CommentAdd 评价商品（买家、订单完成且未评过）
func (h *StoreHandler) CommentAdd(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		OrderID uint   `json:"order_id" binding:"required"`
		DType   int    `json:"d_type"`
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写评价内容")
		return
	}
	if req.DType < 1 || req.DType > 3 {
		req.DType = 1
	}
	var o model.ShopOrder
	if err := h.DB.Where("id = ? AND buyer_id = ?", req.OrderID, uid).First(&o).Error; err != nil {
		resp.ParamError(c, "订单不存在")
		return
	}
	if o.Status != 4 {
		resp.ParamError(c, "订单完成后才能评价！")
		return
	}
	var n int64
	h.DB.Model(&model.ShopComment{}).Where("order_id = ?", o.ID).Count(&n)
	if n > 0 {
		resp.ParamError(c, "该订单已评价过！")
		return
	}
	m := model.ShopComment{GoodsID: o.GoodsID, OrderID: o.ID, UserID: uid, DType: req.DType, Content: strings.TrimSpace(req.Content)}
	h.DB.Create(&m)
	h.DB.Model(&model.ShopGoods{}).Where("id = ?", o.GoodsID).Update("comment", gorm.Expr("comment + ?", 1))
	resp.OK(c, gin.H{"msg": "评价成功"})
}

// ReplyComment 店主回复评价
func (h *StoreHandler) ReplyComment(c *gin.Context) {
	uid := middleware.GetUID(c)
	var m model.ShopComment
	if err := h.DB.First(&m, c.Param("id")).Error; err != nil {
		resp.ParamError(c, "评价不存在")
		return
	}
	var g model.ShopGoods
	h.DB.First(&g, m.GoodsID)
	if g.UserID != uid {
		resp.ParamError(c, "只有店主可以回复")
		return
	}
	var req struct {
		Reply string `json:"reply" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写回复内容")
		return
	}
	h.DB.Model(&m).Update("reply", strings.TrimSpace(req.Reply))
	resp.OK(c, gin.H{"msg": "回复成功"})
}
