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

// MoneyShopHandler 货币商店（复刻诺哈 wap_money_shop：花一种货币买另一种货币礼包）
type MoneyShopHandler struct{ DB *gorm.DB }

func validMoneyType(t string) bool {
	return t == "coins" || t == "yuanbao" || t == "jinzuan" || t == "youquan"
}

// moneyField 货币类型 → users 表字段
func moneyField(t string) string {
	switch t {
	case "yuanbao":
		return "yuanbao"
	case "jinzuan":
		return "jinzuan"
	case "youquan":
		return "youquan"
	default:
		return "coins"
	}
}

// moneyBalance 用户当前余额
func moneyBalance(u *model.User, t string) int {
	switch t {
	case "yuanbao":
		return u.YuanBao
	case "jinzuan":
		return u.JinZuan
	case "youquan":
		return u.YouQuan
	default:
		return u.Coins
	}
}

// 公开：货币商店列表（复刻诺哈 shop_list.asp：平铺编号，ID倒序，每页10条）
func (h *MoneyShopHandler) List(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	q := h.DB.Model(&model.MoneyShop{}).Where("status = 1 AND end_time > ?", time.Now())
	var total int64
	q.Count(&total)
	var list []model.MoneyShop
	q.Order("id DESC").Offset(offset).Limit(size).Find(&list)
	uid := middleware.GetUID(c)
	balances := gin.H{"coins": -1, "yuanbao": -1, "jinzuan": -1, "youquan": -1}
	if uid > 0 {
		var u model.User
		if h.DB.First(&u, uid).Error == nil {
			balances = gin.H{"coins": u.Coins, "yuanbao": u.YuanBao, "jinzuan": u.JinZuan, "youquan": u.YouQuan}
		}
	}
	resp.OK(c, gin.H{"list": list, "total": total, "page": page, "size": size, "balances": balances})
}

// 公开：货币商品详情（复刻诺哈 shop.asp）
func (h *MoneyShopHandler) Detail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var s model.MoneyShop
	if err := h.DB.Where("status = 1 AND end_time > ?", time.Now()).First(&s, id).Error; err != nil {
		resp.NotFound(c, "商品不存在或已下架")
		return
	}
	resp.OK(c, s)
}

// 购买货币（复刻诺哈 shop_buy_ok.asp：支付密码确认，扣支付货币入卖出货币，扣库存加销量）
func (h *MoneyShopHandler) Buy(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Num     int    `json:"num"`
		PayPass string `json:"pay_pass"`
	}
	_ = c.ShouldBindJSON(&req)
	var s model.MoneyShop
	if err := h.DB.First(&s, id).Error; err != nil {
		resp.NotFound(c, "商品不存在")
		return
	}
	if s.Status == 0 {
		resp.ParamError(c, "商品已下架")
		return
	}
	if s.EndTime.Before(time.Now()) {
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
	if req.Num > s.Stock {
		resp.ParamError(c, "购买数量超过库存数量！")
		return
	}
	if !verifyTime("msbuy_"+strconv.FormatUint(uint64(uid), 10), 3) {
		resp.ParamError(c, "操作太频繁了，歇 3 秒再来")
		return
	}
	total := s.Price * req.Num
	var u model.User
	if err := h.DB.First(&u, uid).Error; err != nil {
		resp.Unauthorized(c, "请先登录")
		return
	}
	if moneyBalance(&u, s.PType) < total {
		resp.ParamError(c, "您的"+currencyName(s.PType)+"不足！")
		return
	}
	// 扣支付货币 + 收卖出货币（诺哈 Update_Money 两笔）
	h.DB.Model(&model.User{}).Where("id = ?", uid).
		Update(moneyField(s.PType), gorm.Expr(moneyField(s.PType)+" - ?", total))
	h.DB.Model(&model.User{}).Where("id = ?", uid).
		Update(moneyField(s.MType), gorm.Expr(moneyField(s.MType)+" + ?", s.Money*req.Num))
	addWalletLog(h.DB, uid, "buy", "购买货币「"+s.Name+"」×"+strconv.Itoa(req.Num), s.PType, -total)
	addWalletLog(h.DB, uid, "buy", "购买货币「"+s.Name+"」到账", s.MType, s.Money*req.Num)
	// 扣库存加销量（诺哈 shop_buy_ok）
	h.DB.Model(&model.MoneyShop{}).Where("id = ?", s.ID).Updates(map[string]interface{}{
		"stock": gorm.Expr("stock - ?", req.Num), "sales": gorm.Expr("sales + ?", req.Num),
	})
	h.DB.First(&u, uid)
	resp.OK(c, gin.H{"name": s.Name, "num": req.Num, "gained": s.Money * req.Num,
		"coins": u.Coins, "yuanbao": u.YuanBao, "jinzuan": u.JinZuan, "youquan": u.YouQuan})
}

// 赠送预览（复刻诺哈 shop_send.asp 确认页数据：校验商品/号码/数量/库存，不扣款不验密码）
func (h *MoneyShopHandler) SendPreview(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	to := strings.TrimSpace(c.Query("to"))
	num, _ := strconv.Atoi(c.Query("num"))
	var s model.MoneyShop
	if err := h.DB.Where("status = 1 AND end_time > ?", time.Now()).First(&s, id).Error; err != nil {
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
	if num > s.Stock {
		resp.ParamError(c, "赠送数量超过库存数量！")
		return
	}
	var toU model.User
	if err := h.DB.Where("username = ?", to).First(&toU).Error; err != nil {
		resp.ParamError(c, "会员号码不正确！")
		return
	}
	resp.OK(c, gin.H{
		"name": s.Name, "nickname": toU.Nickname, "username": toU.Username,
		"num": num, "amount": s.Money * num, "currency_name": currencyName(s.MType),
		"total": s.Price * num, "pay_currency_name": currencyName(s.PType),
	})
}

// 赠送货币（复刻诺哈 shop_send_ok.asp：扣赠送方货币、接收方直接到账、扣库存加销量、站内信）
func (h *MoneyShopHandler) Send(c *gin.Context) {
	uid := middleware.GetUID(c)
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		To      string `json:"to"`
		Num     int    `json:"num"`
		PayPass string `json:"pay_pass"`
	}
	_ = c.ShouldBindJSON(&req)
	var s model.MoneyShop
	if err := h.DB.First(&s, id).Error; err != nil {
		resp.NotFound(c, "商品不存在")
		return
	}
	if s.Status == 0 {
		resp.ParamError(c, "商品已下架")
		return
	}
	if s.EndTime.Before(time.Now()) {
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
	if req.Num > s.Stock {
		resp.ParamError(c, "赠送数量超过库存数量！")
		return
	}
	if !verifyTime("mssend_"+strconv.FormatUint(uint64(uid), 10), 3) {
		resp.ParamError(c, "操作太频繁了，歇 3 秒再来")
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
	total := s.Price * req.Num
	var u model.User
	if err := h.DB.First(&u, uid).Error; err != nil {
		resp.Unauthorized(c, "请先登录")
		return
	}
	if moneyBalance(&u, s.PType) < total {
		resp.ParamError(c, "您的"+currencyName(s.PType)+"不足！")
		return
	}
	// 扣赠送方支付货币
	h.DB.Model(&model.User{}).Where("id = ?", uid).
		Update(moneyField(s.PType), gorm.Expr(moneyField(s.PType)+" - ?", total))
	addWalletLog(h.DB, uid, "buy", "赠送货币「"+s.Name+"」×"+strconv.Itoa(req.Num)+" 给"+toU.Nickname+"("+toU.Username+")", s.PType, -total)
	// 接收方货币直接到账（诺哈 Update_Money uid）
	h.DB.Model(&model.User{}).Where("id = ?", toU.ID).
		Update(moneyField(s.MType), gorm.Expr(moneyField(s.MType)+" + ?", s.Money*req.Num))
	addWalletLog(h.DB, toU.ID, "buy", "收到"+u.Nickname+"赠送的货币", s.MType, s.Money*req.Num)
	// 扣库存加销量
	h.DB.Model(&model.MoneyShop{}).Where("id = ?", s.ID).Updates(map[string]interface{}{
		"stock": gorm.Expr("stock - ?", req.Num), "sales": gorm.Expr("sales + ?", req.Num),
	})
	// 站内信（诺哈 MessageSend：恭喜！赠送了您N(货币名)。）
	h.DB.Create(&model.PrivateMessage{SenderID: uid, ReceiverID: toU.ID,
		Content: "恭喜！赠送了您" + strconv.Itoa(s.Money*req.Num) + "(" + currencyName(s.MType) + ")。"})
	h.DB.First(&u, uid)
	resp.OK(c, gin.H{"name": s.Name, "num": req.Num,
		"coins": u.Coins, "yuanbao": u.YuanBao, "jinzuan": u.JinZuan, "youquan": u.YouQuan})
}

// ---- 后台管理 ----

func (h *MoneyShopHandler) AdminList(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	q := h.DB.Model(&model.MoneyShop{})
	var total int64
	q.Count(&total)
	var list []model.MoneyShop
	q.Order("id DESC").Offset(offset).Limit(size).Find(&list)
	resp.OK(c, gin.H{"list": list, "total": total, "page": page, "size": size})
}

type moneyShopReq struct {
	Name     string     `json:"name" binding:"required"`
	MType    string     `json:"mtype" binding:"required"`
	Money    int        `json:"money"`
	PType    string     `json:"ptype" binding:"required"`
	Price    int        `json:"price"`
	Stock    int        `json:"stock"`
	Sales    int        `json:"sales"`
	Status   int        `json:"status"`
	EndTime  *time.Time `json:"end_time"`
}

func (h *MoneyShopHandler) AdminCreate(c *gin.Context) {
	var req moneyShopReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "名称与货币类型必填")
		return
	}
	if !validMoneyType(req.MType) || !validMoneyType(req.PType) {
		resp.ParamError(c, "货币类型不正确")
		return
	}
	if req.Status == 0 {
		req.Status = 1
	}
	var end time.Time
	if req.EndTime != nil {
		end = req.EndTime.Truncate(time.Second)
	}
	h.DB.Create(&model.MoneyShop{Name: req.Name, MType: req.MType, Money: req.Money,
		PType: req.PType, Price: req.Price, Stock: req.Stock, Sales: req.Sales,
		Status: req.Status, AddTime: time.Now(), EndTime: end})
	resp.OK(c, nil)
}

func (h *MoneyShopHandler) AdminUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req moneyShopReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数有误")
		return
	}
	if !validMoneyType(req.MType) || !validMoneyType(req.PType) {
		resp.ParamError(c, "货币类型不正确")
		return
	}
	var end time.Time
	if req.EndTime != nil {
		end = req.EndTime.Truncate(time.Second)
	}
	h.DB.Model(&model.MoneyShop{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name": req.Name, "mtype": req.MType, "money": req.Money, "ptype": req.PType,
		"price": req.Price, "stock": req.Stock, "sales": req.Sales, "status": req.Status, "end_time": end,
	})
	resp.OK(c, nil)
}

func (h *MoneyShopHandler) AdminDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	h.DB.Delete(&model.MoneyShop{}, id)
	resp.OK(c, nil)
}
