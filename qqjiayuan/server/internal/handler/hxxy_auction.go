package handler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 全区拍卖场（复刻原版 xy489/xy490-497/pmgmwp02/all_pm 表）：
// 〖全区拍卖场〗→ 我的拍卖 → ☆☆☆☆☆☆☆☆ → 分类页签（书卷◎材料◎装备/商城◎丹药◎任务/农场◎宝箱◎宝石）
// → 全区玩家拍卖如下：编号列表 → 点名进详情按数量购买，买方付 1% 手续费（最低1两），10天过期自动下架

// hxAuctionTabs 拍卖分类页签（复刻 xy489 页签顺序）
var hxAuctionTabs = []struct {
	K   string
	N   string
	Cat int // item category；equip=100 表示装备表；-1 表示空分类占位
	Kind string
}{
	{"scroll", "书卷", 1, "item"},
	{"material", "材料", 4, "item"},
	{"equip", "装备", 100, "equip"},
	{"mall", "商城", 6, "item"},
	{"pill", "丹药", 5, "item"},
	{"quest", "任务", 3, "item"},
	{"farm", "农场", 7, "item"},
	{"box", "宝箱", 8, "item"},
	{"gem", "宝石", 2, "item"},
}

// hxAuctionExpire 过期拍卖下架（复刻 xy489：超10天删除并退回物品）
func (h *HxxyHandler) hxAuctionExpire() {
	cutoff := time.Now().AddDate(0, 0, -10)
	var olds []model.HxxyAuction
	h.DB.Where("status = 1 AND created_at < ?", cutoff).Find(&olds)
	for _, a := range olds {
		var b model.HxxyBag
		if err := h.DB.Where("id = ? AND store = 4", a.BagID).First(&b).Error; err == nil {
			h.DB.Model(&model.HxxyBag{}).Where("id = ?", b.ID).Update("store", 0)
		}
		h.DB.Model(&model.HxxyAuction{}).Where("id = ?", a.ID).Update("status", 3)
	}
}

// AuctionList 拍卖场列表
func (h *HxxyHandler) AuctionList(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	h.hxAuctionExpire()
	tab := c.DefaultQuery("tab", "scroll")
	cat, kind := -2, "item"
	for _, t := range hxAuctionTabs {
		if t.K == tab {
			cat, kind = t.Cat, t.Kind
		}
	}
	var list []model.HxxyAuction
	q := h.DB.Where("status = 1")
	if kind == "equip" {
		q = q.Where("kind = 'equip'")
	} else {
		q = q.Where("kind = 'item' AND category = ?", cat)
	}
	q.Order("id DESC").Limit(50).Find(&list)
	out := []gin.H{}
	for _, a := range list {
		var sp model.HxxyPlayer
		seller := ""
		if err := h.DB.First(&sp, a.SellerID).Error; err == nil {
			seller = sp.Name
		}
		out = append(out, gin.H{"auction_id": a.ID, "name": a.Name, "count": a.Count,
			"price": a.Price, "seller": seller, "seller_id": a.SellerID, "created_at": a.CreatedAt})
	}
	resp.OK(c, gin.H{"tab": tab, "list": out})
}

// AuctionMine 我的拍卖
func (h *HxxyHandler) AuctionMine(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var list []model.HxxyAuction
	h.DB.Where("seller_id = ? AND status = 1", p.ID).Order("id DESC").Find(&list)
	out := []gin.H{}
	for _, a := range list {
		out = append(out, gin.H{"auction_id": a.ID, "name": a.Name, "count": a.Count, "price": a.Price})
	}
	resp.OK(c, gin.H{"list": out})
}

// AuctionSell 上架拍卖（复刻 pmsjwp02：单价≥1000、上限、绑定拦截、1%手续费最低1两先扣）
func (h *HxxyHandler) AuctionSell(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		BagID uint  `json:"bag_id"`
		Count int   `json:"count"`
		Price int64 `json:"price"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.BagID == 0 {
		resp.ParamError(c, "输入有误请重新输入")
		return
	}
	if in.Count <= 0 {
		resp.ParamError(c, "拍卖数量输入有误请重新输入")
		return
	}
	if in.Price <= 0 {
		resp.ParamError(c, "拍卖价格输入有误请重新输入")
		return
	}
	if in.Price < 1000 {
		resp.ParamError(c, "拍卖单价必须在1000银两上")
		return
	}
	if in.Price > 99999999999 {
		resp.ParamError(c, "拍卖单价超过最大银两限制")
		return
	}
	var b model.HxxyBag
	if err := h.DB.Where("id = ? AND player_id = ? AND store = 0", in.BagID, p.ID).First(&b).Error; err != nil {
		resp.ParamError(c, "物品不存在")
		return
	}
	if b.Bind == 1 {
		resp.ParamError(c, "对不起！绑定物品不能进行拍卖")
		return
	}
	if b.Count < in.Count {
		resp.ParamError(c, "拍卖数量输入有误请重新输入")
		return
	}
	// 拍卖手续费：数量×单价×1%，最低1两，上架时先扣（复刻原版）
	fee := in.Price * int64(in.Count) / 100
	if fee < 1 {
		fee = 1
	}
	if p.Money < fee {
		resp.ParamError(c, "拍卖手续费不足")
		return
	}
	name, cat := "", 0
	if b.Kind == "item" {
		var it model.HxxyItem
		h.DB.First(&it, b.RefID)
		name, cat = it.Name, it.Category
	} else {
		var e model.HxxyEquip
		h.DB.First(&e, b.RefID)
		name, cat = e.Name, 100
	}
	h.hxWallet(p, "money", -fee, "拍卖【"+name+"】手续费")
	if b.Count == in.Count {
		h.DB.Model(&model.HxxyBag{}).Where("id = ?", b.ID).Update("store", 4)
	} else {
		h.DB.Model(&model.HxxyBag{}).Where("id = ?", b.ID).Update("count", b.Count-in.Count)
		h.DB.Create(&model.HxxyBag{PlayerID: p.ID, Kind: b.Kind, RefID: b.RefID, Count: in.Count, Bind: b.Bind, Store: 4, Extra: b.Extra})
	}
	h.DB.Create(&model.HxxyAuction{SellerID: p.ID, BagID: b.ID, Kind: b.Kind, RefID: b.RefID,
		Name: name, Category: cat, Count: in.Count, Price: in.Price})
	resp.OK(c, gin.H{"msg": fmt.Sprintf("手续费：%s银两\n你以每件%s两的价格拍卖了%sx%d", hxSilverText(fee), hxSilverText(in.Price), name, in.Count)})
}

// AuctionBuy 拍卖购买（复刻 pmgmwp02：按数量购买，买方付 1% 手续费最低1两）
func (h *HxxyHandler) AuctionBuy(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		AuctionID uint `json:"auction_id"`
		Count     int  `json:"count"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.AuctionID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var a model.HxxyAuction
	if err := h.DB.Where("id = ? AND status = 1", in.AuctionID).First(&a).Error; err != nil {
		resp.ParamError(c, "对不起！！该物品已被买走或者下架了")
		return
	}
	if a.SellerID == p.ID {
		resp.ParamError(c, "不能购买自己拍卖的物品")
		return
	}
	if in.Count <= 0 || in.Count > a.Count {
		in.Count = a.Count
	}
	total := a.Price * int64(in.Count)
	fee := total / 100
	if fee < 1 {
		fee = 1
	}
	if p.Money < total+fee {
		resp.ParamError(c, fmt.Sprintf("对不起！你银两不足！(附带%d两手续费)", fee))
		return
	}
	var b model.HxxyBag
	if err := h.DB.Where("id = ? AND store = 4", a.BagID).First(&b).Error; err != nil {
		resp.ParamError(c, "对不起！！该物品已被买走或者下架了")
		return
	}
	h.hxWallet(p, "money", -(total+fee), "拍卖购买【"+a.Name+"】x"+strconv.Itoa(in.Count)+"（含手续费）")
	h.hxBagAdd(p, a.Kind, a.RefID, in.Count, b.Bind)
	if in.Count >= a.Count {
		h.DB.Delete(&model.HxxyBag{}, b.ID)
		h.DB.Model(&model.HxxyAuction{}).Where("id = ?", a.ID).Update("status", 2)
	} else {
		h.DB.Model(&model.HxxyBag{}).Where("id = ?", b.ID).Update("count", b.Count-in.Count)
		h.DB.Model(&model.HxxyAuction{}).Where("id = ?", a.ID).Update("count", a.Count-in.Count)
	}
	var seller model.HxxyPlayer
	if err := h.DB.First(&seller, a.SellerID).Error; err == nil {
		h.hxWallet(&seller, "money", total, "拍卖售出【"+a.Name+"】")
		h.hxNotify(seller.ID, fmt.Sprintf("买走了你拍卖的%sx%d，获得%s银两", a.Name, in.Count, hxSilverText(total)))
	}
	// 复刻 pmgmwp02 成功文案（红字两行）
	resp.OK(c, gin.H{"msg": fmt.Sprintf("你用了%s两，购买%sx%d\n(附带%d两手续费)", hxSilverText(total+fee), a.Name, in.Count, fee)})
}

// AuctionCancel 拍卖下架（物品退回背包）
func (h *HxxyHandler) AuctionCancel(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		AuctionID uint `json:"auction_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.AuctionID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var a model.HxxyAuction
	if err := h.DB.Where("id = ? AND seller_id = ? AND status = 1", in.AuctionID, p.ID).First(&a).Error; err != nil {
		resp.ParamError(c, "拍卖记录不存在")
		return
	}
	var b model.HxxyBag
	if err := h.DB.Where("id = ? AND store = 4", a.BagID).First(&b).Error; err == nil {
		h.DB.Model(&model.HxxyBag{}).Where("id = ?", b.ID).Update("store", 0)
	}
	h.DB.Model(&model.HxxyAuction{}).Where("id = ?", a.ID).Update("status", 3)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("你下架了拍卖的%sx%d，物品已退回行囊", a.Name, a.Count)})
}

// hxNotify 供本文件使用的占位说明：实际通知走 hxxy.go 的 hxNotify
var _ = json.Marshal
