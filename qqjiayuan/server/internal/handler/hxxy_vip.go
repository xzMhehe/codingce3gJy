package handler

// 会员（VIP等级）模块（复刻原版 会员中心 / VIP 福利 xy520/521）
// VIP等级 0~20（充值等级），等级越高每日可用银两兑换越多【万能果】与〖金豆〗
// 1. GET /vip/info 当前 VIP 等级 + 福利一览
// 2. POST /vip/exchange 每日一次按当前 VIP 等级兑换万能果+金豆（需 160 级，扣银两）
// 3. POST /vip/recharge 演示充值码（含 SVIP0~20 设置会员等级）

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// hxVipDef VIP 等级福利表：等级 → (万能果数量, 金豆数量, 所需银两)
var hxVipDef = []struct {
	Lv      int
	Cute    int // 万能果
	Beans   int // 金豆
	Price   int64
	ShowSilver string
}{
	{0, 1, 0, 20000000, "2000万"}, {1, 2, 1, 20000000, "2000万"},
	{2, 4, 3, 50000000, "5000万"}, {3, 4, 3, 60000000, "6000万"},
	{4, 5, 4, 70000000, "7000万"}, {5, 6, 5, 80000000, "8000万"},
	{6, 7, 6, 90000000, "9000万"}, {7, 8, 7, 100000000, "1亿"},
	{8, 9, 8, 100000000, "1亿"}, {9, 10, 9, 100000000, "1亿"},
	{10, 11, 10, 100000000, "1亿"}, {11, 12, 11, 200000000, "2亿"},
	{12, 13, 12, 200000000, "2亿"}, {13, 14, 13, 200000000, "2亿"},
	{14, 15, 14, 200000000, "2亿"}, {15, 16, 15, 200000000, "2亿"},
	{16, 17, 16, 200000000, "2亿"}, {17, 18, 17, 200000000, "2亿"},
	{18, 19, 18, 200000000, "2亿"}, {19, 20, 20, 200000000, "2亿"},
	{20, 30, 30, 200000000, "2亿"},
}

// hxVipDefAt 取指定等级的福利
func hxVipDefAt(lv int) (int, int, int64, string) {
	for _, d := range hxVipDef {
		if d.Lv == lv {
			return d.Cute, d.Beans, d.Price, d.ShowSilver
		}
	}
	return 1, 0, 20000000, "2000万"
}

// hxVipItemID 万能果物品 id（名字沾万能果即用，缺失返回 0）
func (h *HxxyHandler) hxVipItemID() uint {
	var it model.HxxyItem
	if err := h.DB.Where("name LIKE '%万能果%' AND category = 4").Order("id ASC").First(&it).Error; err == nil {
		return it.ID
	}
	return 0
}

// hxVipExpNeed vip 积分升级表（复刻 xy306：各 VIP 等级所需积分下限，lv0~20）
var hxVipExpNeed = []int{0, 10, 50, 100, 200, 400, 500, 600, 1000, 1500, 2000, 3000, 5000, 7000, 9000, 11000, 13000, 15000, 17000, 19000, 25000}

// VipInfo 会员中心数据
func (h *HxxyHandler) VipInfo(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	today := time.Now().Format("2006-01-02")
	var exchanged int64
	h.DB.Model(&model.HxxyActivityLog{}).Where("player_id = ? AND act = 'vip_exchange' AND day = ?", p.ID, today).Count(&exchanged)
	defs := []gin.H{}
	for _, d := range hxVipDef {
		defs = append(defs, gin.H{"lv": d.Lv, "cute": d.Cute, "beans": d.Beans, "price": d.Price, "silver": d.ShowSilver})
	}
	// vip 积分升级进度（复刻 xy306：差X点vip积分即可升级为VIPN级）
	cute, beans, _, silver := hxVipDefAt(p.VipLv)
	expDiff, nextLv := 0, 0
	if p.VipLv < 20 {
		nextLv = p.VipLv + 1
		expDiff = hxVipExpNeed[nextLv] - int(p.VipExp)
		if expDiff < 0 {
			expDiff = 0
		}
	}
	resp.OK(c, gin.H{
		"level": p.VipLv,
		"lvl160": p.Level >= 160,
		"exchanged": exchanged > 0,
		"defs": defs,
		"wanneng_id": h.hxVipItemID(),
		"vip_exp": p.VipExp, "exp_diff": expDiff, "next_lv": nextLv,
		"cur_cute": cute, "cur_beans": beans, "cur_silver": silver,
	})
}

// VipExchange 每日按当前 VIP 等级兑换【万能果】+〖金豆〗（扣银两，每日一次，需160级）
func (h *HxxyHandler) VipExchange(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	if p.Level < 160 {
		resp.ParamError(c, "对不起！小仙家等级还未满160级啊？伤不起~~伤不起~~")
		return
	}
	today := time.Now().Format("2006-01-02")
	var exchanged int64
	h.DB.Model(&model.HxxyActivityLog{}).Where("player_id = ? AND act = 'vip_exchange' AND day = ?", p.ID, today).Count(&exchanged)
	if exchanged > 0 {
		resp.ParamError(c, "对不起！你今日兑换过了")
		return
	}
	cute, beans, price, _ := hxVipDefAt(p.VipLv)
	if p.Money < price {
		resp.ParamError(c, "对不起！！你的银两不足来兑换")
		return
	}
	h.hxWallet(p, "money", -price, "VIP"+strconv.Itoa(p.VipLv)+"级兑换万能果")
	if beans > 0 {
		h.hxWallet(p, "beans", int64(beans), "VIP"+strconv.Itoa(p.VipLv)+"级兑换")
	}
	// 发万能果
	if id := h.hxVipItemID(); id > 0 {
		h.hxBagAdd(p, "item", id, cute, 0)
	}
	h.DB.Create(&model.HxxyActivityLog{PlayerID: p.ID, Act: "vip_exchange", Day: today})
	resp.OK(c, gin.H{"msg": fmt.Sprintf("恭喜你！兑换成功：万能果x%d%s", cute, func() string {
		if beans > 0 {
			return fmt.Sprintf(" + 金豆%d", beans)
		}
		return ""
	}())})
}