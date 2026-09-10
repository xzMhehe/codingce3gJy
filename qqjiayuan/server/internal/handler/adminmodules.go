package handler

import (
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// wordUserIDs word 搜索过滤：纯数字=按家园号精确；否则按昵称模糊，返回用户ID集合。
// 返回 nil 表示无过滤条件。
func (h *AdminHandler) wordUserIDs(c *gin.Context) []uint {
	word := strings.TrimSpace(c.Query("word"))
	if word == "" {
		return nil
	}
	var ids []uint
	if uid, err := strconv.Atoi(word); err == nil {
		return []uint{uint(uid)}
	}
	h.DB.Model(&model.User{}).Where("nickname LIKE ?", "%"+word+"%").Pluck("id", &ids)
	return ids
}

// adminmodules.go 管理后台模块补齐（对齐诺哈三代后台菜单：
// 家园管理/信息管理/文章管理/留言管理/商城管理/货币管理/会员推荐/帖子回收站）

// ---- 家园管理 home/ ----

// AdminHomes 家园列表（活跃点/心情数/访客/留言统计）
func (h *AdminHandler) AdminHomes(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := c.Query("word")
	q := h.DB.Model(&model.Home{}).
		Joins("LEFT JOIN users ON homes.user_id = users.id")
	if word != "" {
		q = q.Where("users.nickname LIKE ? OR users.username = ?", "%"+word+"%", word)
	}
	var total int64
	q.Count(&total)
	type homeRow struct {
		model.Home
		Nickname string `json:"nickname"`
		Username string `json:"username"`
		Color    string `json:"color"`
	}
	var homes []model.Home
	q.Select("homes.*").Order("homes.point DESC").Offset(offset).Limit(size).Find(&homes)
	out := make([]homeRow, 0, len(homes))
	for _, hm := range homes {
		var u model.User
		h.DB.Select("nickname,username,color").First(&u, hm.UserID)
		out = append(out, homeRow{Home: hm, Nickname: u.Nickname, Username: u.Username, Color: u.Color})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminVisitors 家园访客（可按 user_id 过滤）
func (h *AdminHandler) AdminVisitors(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	q := h.DB.Model(&model.Visitor{})
	if uid, err := strconv.Atoi(c.Query("user_id")); err == nil && uid > 0 {
		q = q.Where("owner_id = ?", uid)
	}
	var total int64
	q.Count(&total)
	type visRow struct {
		model.Visitor
		OwnerNick string `json:"owner_nick"`
		VisitNick string `json:"visit_nick"`
	}
	var list []model.Visitor
	q.Order("created_at DESC").Offset(offset).Limit(size).Find(&list)
	out := make([]visRow, 0, len(list))
	for _, v := range list {
		var o, vi model.User
		h.DB.Select("nickname").First(&o, v.OwnerID)
		h.DB.Select("nickname").First(&vi, v.UserID)
		out = append(out, visRow{Visitor: v, OwnerNick: o.Nickname, VisitNick: vi.Nickname})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// ---- 信息管理 message/（新鲜事 + 通知） ----

// AdminHomeNews 新鲜事列表
func (h *AdminHandler) AdminHomeNews(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	uid, _ := strconv.Atoi(c.Query("user_id"))
	q := h.DB.Model(&model.HomeNews{})
	if uid > 0 {
		q = q.Where("user_id = ?", uid)
	}
	var total int64
	q.Count(&total)
	type newsRow struct {
		model.HomeNews
		Nickname string `json:"nickname"`
	}
	var list []model.HomeNews
	q.Order("created_at DESC").Offset(offset).Limit(size).Find(&list)
	out := make([]newsRow, 0, len(list))
	for _, n := range list {
		var u model.User
		h.DB.Select("nickname").First(&u, n.UserID)
		out = append(out, newsRow{HomeNews: n, Nickname: u.Nickname})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminHomeNewsDel 删除新鲜事
func (h *AdminHandler) AdminHomeNewsDel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.HomeNews{}, id)
	resp.OK(c, nil)
}

// AdminNotifications 通知列表（信息管理：家信）
func (h *AdminHandler) AdminNotifications(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	q := h.DB.Model(&model.Notification{})
	var total int64
	q.Count(&total)
	type notiRow struct {
		model.Notification
		Nickname string `json:"nickname"`
	}
	var list []model.Notification
	q.Order("created_at DESC").Offset(offset).Limit(size).Find(&list)
	out := make([]notiRow, 0, len(list))
	for _, n := range list {
		var u model.User
		h.DB.Select("nickname").First(&u, n.UserID)
		out = append(out, notiRow{Notification: n, Nickname: u.Nickname})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminNotificationDel 删除通知
func (h *AdminHandler) AdminNotificationDel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.Notification{}, id)
	resp.OK(c, nil)
}

// ---- 会员管理 user/（会员推荐） ----

// AdminInvites 会员推荐（邀请）列表
func (h *AdminHandler) AdminInvites(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	q := h.DB.Model(&model.Invite{})
	if ids := h.wordUserIDs(c); ids != nil {
		q = q.Where("user_id IN ? OR used_uid IN ?", ids, ids)
	}
	var total int64
	q.Count(&total)
	type invRow struct {
		model.Invite
		InviterNick string `json:"inviter_nick"`
		UsedNick    string `json:"used_nick"`
	}
	var list []model.Invite
	q.Order("created_at DESC").Offset(offset).Limit(size).Find(&list)
	out := make([]invRow, 0, len(list))
	for _, iv := range list {
		var a, b model.User
		h.DB.Select("nickname").First(&a, iv.UserID)
		usedNick := ""
		if iv.UsedUID > 0 {
			h.DB.Select("nickname").First(&b, iv.UsedUID)
			usedNick = b.Nickname
		}
		out = append(out, invRow{Invite: iv, InviterNick: a.Nickname, UsedNick: usedNick})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminInviteCreate 生成邀请码（归属指定会员，0=系统生成）
func (h *AdminHandler) AdminInviteCreate(c *gin.Context) {
	var req struct {
		UserID uint `json:"user_id"`
	}
	c.ShouldBindJSON(&req)
	if req.UserID > 0 {
		var u model.User
		if err := h.DB.First(&u, req.UserID).Error; err != nil {
			resp.ParamError(c, "邀请人不存在")
			return
		}
	}
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	for i := 0; i < 5; i++ {
		b := make([]byte, 8)
		for j := range b {
			b[j] = letters[rand.Intn(len(letters))]
		}
		code := string(b)
		var n int64
		h.DB.Model(&model.Invite{}).Where("code = ?", code).Count(&n)
		if n > 0 {
			continue
		}
		iv := model.Invite{UserID: req.UserID, Code: code}
		h.DB.Create(&iv)
		resp.OK(c, gin.H{"msg": "邀请码 " + code + " 已生成", "code": code})
		return
	}
	resp.ParamError(c, "生成失败，请重试")
}

// AdminInviteDelete 删除邀请码（未使用的才可删）
func (h *AdminHandler) AdminInviteDelete(c *gin.Context) {
	var iv model.Invite
	if err := h.DB.First(&iv, c.Param("id")).Error; err != nil {
		resp.ParamError(c, "邀请码不存在")
		return
	}
	if iv.Status == 1 {
		resp.ParamError(c, "已使用的邀请码不能删除")
		return
	}
	h.DB.Delete(&iv)
	resp.OK(c, gin.H{"msg": "已删除"})
}

// ---- 货币管理 money/（奖罚流水） ----

// AdminWalletLogs 钱包流水
func (h *AdminHandler) AdminWalletLogs(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	uid, _ := strconv.Atoi(c.Query("user_id"))
	q := h.DB.Model(&model.WalletLog{})
	if uid > 0 {
		q = q.Where("user_id = ?", uid)
	}
	var total int64
	q.Count(&total)
	type logRow struct {
		model.WalletLog
		Nickname string `json:"nickname"`
	}
	var list []model.WalletLog
	q.Order("created_at DESC").Offset(offset).Limit(size).Find(&list)
	out := make([]logRow, 0, len(list))
	for _, l := range list {
		var u model.User
		h.DB.Select("nickname").First(&u, l.UserID)
		out = append(out, logRow{WalletLog: l, Nickname: u.Nickname})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// ---- 社区管理 bbs/（帖子回收站） ----

// AdminThreadRecycle 回收站（已删帖子）
func (h *AdminHandler) AdminThreadRecycle(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	q := h.DB.Model(&model.Thread{}).Where("status = 0")
	var total int64
	q.Count(&total)
	type thRow struct {
		model.Thread
		Nickname string `json:"nickname"`
	}
	var list []model.Thread
	q.Order("updated_at DESC").Offset(offset).Limit(size).Find(&list)
	out := make([]thRow, 0, len(list))
	for _, th := range list {
		var u model.User
		h.DB.Select("nickname").First(&u, th.UserID)
		out = append(out, thRow{Thread: th, Nickname: u.Nickname})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminThreadRestore 恢复帖子
func (h *AdminHandler) AdminThreadRestore(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Model(&model.Thread{}).Where("id = ?", id).Update("status", 1)
	resp.OK(c, nil)
}

// ---- 商城管理 shop/（C2C） ----

// AdminShops 店铺列表
func (h *AdminHandler) AdminShops(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := c.Query("word")
	q := h.DB.Model(&model.Shop{}).Joins("LEFT JOIN users ON shops.user_id = users.id")
	if word != "" {
		q = q.Where("shops.name LIKE ? OR users.nickname LIKE ? OR users.username = ?", "%"+word+"%", "%"+word+"%", word)
	}
	var total int64
	q.Count(&total)
	type shopRow struct {
		model.Shop
		Nickname string `json:"nickname"`
		Username string `json:"username"`
		GoodsCnt int64  `json:"goods_cnt"`
	}
	var shops []model.Shop
	q.Select("shops.*").Order("shops.id ASC").Offset(offset).Limit(size).Find(&shops)
	out := make([]shopRow, 0, len(shops))
	for _, s := range shops {
		var u model.User
		h.DB.Select("nickname,username").First(&u, s.UserID)
		var cnt int64
		h.DB.Model(&model.ShopGoods{}).Where("user_id = ?", s.UserID).Count(&cnt)
		out = append(out, shopRow{Shop: s, Nickname: u.Nickname, Username: u.Username, GoodsCnt: cnt})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminShopGoods 商品列表（全量，含下架）
func (h *AdminHandler) AdminShopGoods(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	uid, _ := strconv.Atoi(c.Query("user_id"))
	q := h.DB.Model(&model.ShopGoods{})
	if uid > 0 {
		q = q.Where("user_id = ?", uid)
	}
	var total int64
	q.Count(&total)
	type goodsRow struct {
		model.ShopGoods
		Seller string `json:"seller"`
	}
	var list []model.ShopGoods
	q.Order("created_at DESC").Offset(offset).Limit(size).Find(&list)
	out := make([]goodsRow, 0, len(list))
	for _, g := range list {
		var u model.User
		h.DB.Select("nickname").First(&u, g.UserID)
		out = append(out, goodsRow{ShopGoods: g, Seller: u.Nickname})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminShopGoodsStatus 商品上下架
func (h *AdminHandler) AdminShopGoodsStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || (req.Status != 0 && req.Status != 1) {
		resp.ParamError(c, "status 只能是 0（下架）或 1（上架）")
		return
	}
	h.DB.Model(&model.ShopGoods{}).Where("id = ?", id).Update("status", req.Status)
	resp.OK(c, nil)
}

// AdminShopGoodsDel 删除商品
func (h *AdminHandler) AdminShopGoodsDel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.ShopGoods{}, id)
	resp.OK(c, nil)
}

// AdminShopOrders 订单列表
func (h *AdminHandler) AdminShopOrders(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	status, _ := strconv.Atoi(c.Query("status"))
	q := h.DB.Model(&model.ShopOrder{})
	if status > 0 {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	type orderRow struct {
		model.ShopOrder
		BuyerNick  string `json:"buyer_nick"`
		SellerNick string `json:"seller_nick"`
	}
	var list []model.ShopOrder
	q.Order("created_at DESC").Offset(offset).Limit(size).Find(&list)
	out := make([]orderRow, 0, len(list))
	for _, o := range list {
		var b, s model.User
		h.DB.Select("nickname").First(&b, o.BuyerID)
		h.DB.Select("nickname").First(&s, o.SellerID)
		out = append(out, orderRow{ShopOrder: o, BuyerNick: b.Nickname, SellerNick: s.Nickname})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminShopComments 商品评论
func (h *AdminHandler) AdminShopComments(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	q := h.DB.Model(&model.ShopComment{})
	var total int64
	q.Count(&total)
	type cmtRow struct {
		model.ShopComment
		User      string `json:"user"`
		GoodsName string `json:"goods_name"`
	}
	var list []model.ShopComment
	q.Order("created_at DESC").Offset(offset).Limit(size).Find(&list)
	out := make([]cmtRow, 0, len(list))
	for _, cm := range list {
		var u model.User
		h.DB.Select("nickname").First(&u, cm.UserID)
		var g model.ShopGoods
		h.DB.Select("name").First(&g, cm.GoodsID)
		out = append(out, cmtRow{ShopComment: cm, User: u.Nickname, GoodsName: g.Name})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminShopCommentDel 删除商品评论
func (h *AdminHandler) AdminShopCommentDel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.ShopComment{}, id)
	resp.OK(c, nil)
}

// ---- 文章管理 article/ ----

// AdminSiteArticles 文章列表
func (h *AdminHandler) AdminSiteArticles(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	q := h.DB.Model(&model.SiteArticle{}).Where("status = 1")
	var total int64
	q.Count(&total)
	type artRow struct {
		model.SiteArticle
		Author string `json:"author"`
	}
	var list []model.SiteArticle
	q.Order("created_at DESC").Offset(offset).Limit(size).Find(&list)
	out := make([]artRow, 0, len(list))
	for _, a := range list {
		var u model.User
		h.DB.Select("nickname").First(&u, a.UserID)
		out = append(out, artRow{SiteArticle: a, Author: u.Nickname})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminSiteArticleDel 删除文章
func (h *AdminHandler) AdminSiteArticleDel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Model(&model.SiteArticle{}).Where("id = ?", id).Update("status", 0)
	resp.OK(c, nil)
}

// AdminArticleCategories 文章分类列表
func (h *AdminHandler) AdminArticleCategories(c *gin.Context) {
	var cats []model.SiteArticleCategory
	h.DB.Order("sort ASC, id ASC").Find(&cats)
	resp.OK(c, cats)
}

// AdminArticleCategoryCreate 新建分类
func (h *AdminHandler) AdminArticleCategoryCreate(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required,max=30"`
		Sort int    `json:"sort"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "分类名必填（30字内）")
		return
	}
	h.DB.Create(&model.SiteArticleCategory{Name: req.Name, Sort: req.Sort})
	resp.OK(c, nil)
}

// AdminArticleCategoryUpdate 更新分类
func (h *AdminHandler) AdminArticleCategoryUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Name string `json:"name" binding:"required,max=30"`
		Sort int    `json:"sort"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "分类名必填")
		return
	}
	h.DB.Model(&model.SiteArticleCategory{}).Where("id = ?", id).Updates(map[string]interface{}{"name": req.Name, "sort": req.Sort})
	resp.OK(c, nil)
}

// AdminArticleCategoryDel 删除分类
func (h *AdminHandler) AdminArticleCategoryDel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.SiteArticleCategory{}, id)
	resp.OK(c, nil)
}

// ---- 留言管理 guest/ ----

// AdminGuestbook 留言本列表（含私密内容）
func (h *AdminHandler) AdminGuestbook(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	q := h.DB.Model(&model.GuestBook{}).Where("status = 1")
	var total int64
	q.Count(&total)
	var list []model.GuestBook
	q.Order("created_at DESC").Offset(offset).Limit(size).Find(&list)
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": list})
}

// AdminGuestbookDel 删除留言
func (h *AdminHandler) AdminGuestbookDel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Model(&model.GuestBook{}).Where("id = ?", id).Update("status", 0)
	resp.OK(c, nil)
}

// ---- 会员管理 user/ 子页（对齐诺哈：会员证件/联系/地址/密保/日志） ----

// AdminUserDocu 会员证件列表
func (h *AdminHandler) AdminUserDocu(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	q := h.DB.Model(&model.UserDocument{})
	if ids := h.wordUserIDs(c); ids != nil {
		q = q.Where("user_id IN ?", ids)
	}
	var total int64
	q.Count(&total)
	type row struct {
		model.UserDocument
		Nickname string `json:"nickname"`
		Username string `json:"username"`
		Number   string `json:"-"`
	}
	var list []model.UserDocument
	q.Order("updated_at DESC").Offset(offset).Limit(size).Find(&list)
	out := make([]gin.H, 0, len(list))
	for _, d := range list {
		var u model.User
		h.DB.Select("nickname,username").First(&u, d.UserID)
		num := d.Number
		if len(num) > 6 {
			num = num[:3] + "***********" + num[len(num)-3:]
		}
		out = append(out, gin.H{"id": d.ID, "user_id": d.UserID, "nickname": u.Nickname, "username": u.Username,
			"type": d.Type, "real_name": d.RealName, "number": num, "updated_at": d.UpdatedAt})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminUserDocuDelete 删除会员证件
func (h *AdminHandler) AdminUserDocuDelete(c *gin.Context) {
	var d model.UserDocument
	if err := h.DB.First(&d, c.Param("id")).Error; err != nil {
		resp.ParamError(c, "证件不存在")
		return
	}
	h.DB.Delete(&d)
	resp.OK(c, gin.H{"msg": "证件已删除"})
}

// AdminUserContacts 会员联系列表
func (h *AdminHandler) AdminUserContacts(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	q := h.DB.Model(&model.UserContact{})
	if ids := h.wordUserIDs(c); ids != nil {
		q = q.Where("user_id IN ?", ids)
	}
	var total int64
	q.Count(&total)
	type row struct {
		model.UserContact
		Nickname string `json:"nickname"`
		Username string `json:"username"`
	}
	var list []model.UserContact
	q.Order("updated_at DESC").Offset(offset).Limit(size).Find(&list)
	out := make([]row, 0, len(list))
	for _, ct := range list {
		var u model.User
		h.DB.Select("nickname,username").First(&u, ct.UserID)
		out = append(out, row{UserContact: ct, Nickname: u.Nickname, Username: u.Username})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminUserContactUpdate 编辑会员联系方式（QQ/邮箱/手机）
func (h *AdminHandler) AdminUserContactUpdate(c *gin.Context) {
	var ct model.UserContact
	if err := h.DB.First(&ct, c.Param("id")).Error; err != nil {
		resp.ParamError(c, "联系方式不存在")
		return
	}
	var req struct {
		QQ    string `json:"qq"`
		Mail  string `json:"mail"`
		Phone string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	h.DB.Model(&ct).Updates(map[string]interface{}{"qq": req.QQ, "mail": req.Mail, "phone": req.Phone})
	resp.OK(c, gin.H{"msg": "联系方式已保存"})
}

// AdminUserAddresses 会员地址列表
func (h *AdminHandler) AdminUserAddresses(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	q := h.DB.Model(&model.UserAddress{})
	if ids := h.wordUserIDs(c); ids != nil {
		q = q.Where("user_id IN ?", ids)
	}
	var total int64
	q.Count(&total)
	type row struct {
		model.UserAddress
		Nickname string `json:"nickname"`
		Username string `json:"username"`
	}
	var list []model.UserAddress
	q.Order("updated_at DESC").Offset(offset).Limit(size).Find(&list)
	out := make([]row, 0, len(list))
	for _, a := range list {
		var u model.User
		h.DB.Select("nickname,username").First(&u, a.UserID)
		out = append(out, row{UserAddress: a, Nickname: u.Nickname, Username: u.Username})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminUserAddressUpdate 编辑会员通信地址（故乡/现居）
func (h *AdminHandler) AdminUserAddressUpdate(c *gin.Context) {
	var a model.UserAddress
	if err := h.DB.First(&a, c.Param("id")).Error; err != nil {
		resp.ParamError(c, "地址不存在")
		return
	}
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	fields := []string{"home_nation", "home_prov", "home_city", "home_dist", "home_addr", "home_zip",
		"live_nation", "live_prov", "live_city", "live_dist", "live_addr", "live_zip"}
	up := map[string]interface{}{}
	for _, f := range fields {
		if v, ok := req[f]; ok {
			up[f] = strings.TrimSpace(v)
		}
	}
	if len(up) > 0 {
		h.DB.Model(&a).Updates(up)
	}
	resp.OK(c, gin.H{"msg": "地址已保存"})
}

// AdminUserProtections 会员密保列表
func (h *AdminHandler) AdminUserProtections(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	q := h.DB.Model(&model.UserProtection{})
	if ids := h.wordUserIDs(c); ids != nil {
		q = q.Where("user_id IN ?", ids)
	}
	var total int64
	q.Count(&total)
	type row struct {
		UserID      uint      `json:"user_id"`
		Nickname    string    `json:"nickname"`
		Username    string    `json:"username"`
		Issue       int       `json:"issue"`
		HasProtect  bool      `json:"has_protect"`
		UpdatedAt   time.Time `json:"updated_at"`
	}
	var list []model.UserProtection
	q.Order("updated_at DESC").Offset(offset).Limit(size).Find(&list)
	out := make([]row, 0, len(list))
	for _, p := range list {
		var u model.User
		h.DB.Select("nickname,username").First(&u, p.UserID)
		out = append(out, row{UserID: p.UserID, Nickname: u.Nickname, Username: u.Username, Issue: p.Issue, HasProtect: true, UpdatedAt: p.UpdatedAt})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminUserProtectionDelete 清除会员密保（对齐诺哈 admin/user/protec 删除）
func (h *AdminHandler) AdminUserProtectionDelete(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	if uid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	h.DB.Where("user_id = ?", uid).Delete(&model.UserProtection{})
	resp.OK(c, gin.H{"msg": "密保已清除"})
}

// AdminUserLogs 会员日志列表（登录/操作）
func (h *AdminHandler) AdminUserLogs(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	uid, _ := strconv.Atoi(c.Query("user_id"))
	q := h.DB.Model(&model.UserLog{})
	if uid > 0 {
		q = q.Where("user_id = ?", uid)
	} else if ids := h.wordUserIDs(c); ids != nil {
		q = q.Where("user_id IN ?", ids)
	}
	var total int64
	q.Count(&total)
	type row struct {
		model.UserLog
		Nickname string `json:"nickname"`
	}
	var list []model.UserLog
	q.Order("created_at DESC").Offset(offset).Limit(size).Find(&list)
	out := make([]row, 0, len(list))
	for _, l := range list {
		var u model.User
		h.DB.Select("nickname").First(&u, l.UserID)
		out = append(out, row{UserLog: l, Nickname: u.Nickname})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// ---- 信息管理 message/（家信 = 站内私信，对齐诺哈家信列表） ----

// AdminMessages 家信列表
func (h *AdminHandler) AdminMessages(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	q := h.DB.Model(&model.PrivateMessage{})
	var total int64
	q.Count(&total)
	type row struct {
		model.PrivateMessage
		FromNick string `json:"from_nick"`
		ToNick   string `json:"to_nick"`
	}
	var list []model.PrivateMessage
	q.Order("created_at DESC").Offset(offset).Limit(size).Find(&list)
	out := make([]row, 0, len(list))
	for _, m := range list {
		var f, t model.User
		h.DB.Select("nickname").First(&f, m.SenderID)
		h.DB.Select("nickname").First(&t, m.ReceiverID)
		out = append(out, row{PrivateMessage: m, FromNick: f.Nickname, ToNick: t.Nickname})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminMessageDel 删除家信
func (h *AdminHandler) AdminMessageDel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.PrivateMessage{}, id)
	resp.OK(c, nil)
}

// ---- 书城管理 book/（小说列表） ----

// AdminBooks 小说列表
func (h *AdminHandler) AdminBooks(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := c.Query("word")
	q := h.DB.Model(&model.Book{})
	if word != "" {
		q = q.Where("title LIKE ? OR author LIKE ?", "%"+word+"%", "%"+word+"%")
	}
	var total int64
	q.Count(&total)
	var list []model.Book
	q.Order("id ASC").Offset(offset).Limit(size).Find(&list)
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": list})
}

// AdminBookUpdate 更新小说（标题/作者/分类/状态/简介）
func (h *AdminHandler) AdminBookUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Title    string `json:"title" binding:"required,max=100"`
		Author   string `json:"author" binding:"max=50"`
		Category string `json:"category" binding:"max=20"`
		Status   string `json:"status" binding:"max=10"`
		Intro    string `json:"intro" binding:"max=500"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "书名必填")
		return
	}
	h.DB.Model(&model.Book{}).Where("id = ?", id).Updates(map[string]interface{}{
		"title": req.Title, "author": req.Author, "category": req.Category, "status": req.Status, "intro": req.Intro,
	})
	resp.OK(c, nil)
}

// AdminBookDel 删除小说
func (h *AdminHandler) AdminBookDel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.Book{}, id)
	resp.OK(c, nil)
}

// ---- 系统配置 config/（站点设置 KV） ----

// AdminSiteConfig 站点设置列表
func (h *AdminHandler) AdminSiteConfig(c *gin.Context) {
	var list []model.Setting
	h.DB.Where("`key` NOT LIKE 'blog\\_cat\\_%' AND `key` NOT LIKE 'invite\\_%'").Order("`key` ASC").Find(&list)
	if list == nil {
		list = []model.Setting{}
	}
	resp.OK(c, list)
}

// AdminSiteConfigSave 保存站点设置（KV upsert，支持批量）
func (h *AdminHandler) AdminSiteConfigSave(c *gin.Context) {
	var req []model.Setting
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数有误")
		return
	}
	for _, s := range req {
		if s.Key == "" {
			continue
		}
		h.DB.Where(&model.Setting{Key: s.Key}).Assign(model.Setting{Value: s.Value}).FirstOrCreate(&model.Setting{Key: s.Key, Value: s.Value})
	}
	resp.OK(c, nil)
}
