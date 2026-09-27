package handler

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// ============ 为爱发电卡（2026-09-27 用户新需求） ============
//
// 用户规则：
//   - 为爱发电卡 30 天、每天领 150 钻石；为爱发电高级卡 30 天、每天领 200 钻石。
//   - 卡片由管理端发放即生效（不进入背包、无需使用激活）。
//   - 玩家在「任务」面板的「为爱发电卡」tab 领取每日钻石。
//   - 发放当天即可领第 1 天；之后每天 +1 天，总领取天数封顶 TotalDays(30)。
//   - 漏领累加：领取时按「自激活起的应领天数 - 已领天数」一次性发放。
//   - 多张卡可叠加领取：普通 + 高级可同时各领各的。

const (
	// ezfyItemTypeLoveCard ezfy_cfg_item.item_type：22 为爱发电卡 / 23 为爱发电高级卡
	ezfyItemTypeLoveCard    = 22
	ezfyItemTypeLoveCardPro = 23
	// ezfyLoveCardDayMS 为爱发电卡 1 天的毫秒数
	ezfyLoveCardDayMS = int64(24 * 3600 * 1000)
	// ezfyLoveCardTotalDays 每张为爱发电卡总有效天数
	ezfyLoveCardTotalDays = 30
)

// isLoveCardItem 是否「为爱发电卡」类道具（管理端发放时据此识别激活卡片）。
func isLoveCardItem(itemType int) bool {
	return itemType == ezfyItemTypeLoveCard || itemType == ezfyItemTypeLoveCardPro
}

// addDiamond 给玩家钻石累计（无条件累加，仅依赖 ezfy_profile.diamond）。
func (h *EzfyHandler) addDiamond(uid uint, amount int64) error {
	prof := h.ensureProfile(uid)
	return h.DB.Model(&model.EzfyProfile{}).Where("user_id = ?", uid).
		Update("diamond", prof.Diamond+amount).Error
}

// createLoveCard 管理端发放为爱发电卡。
//
// 2026-09-27 规则：同类型卡「累加时间」——若玩家已有同一类型的卡且仍有未领天数，
// 则将天数续到那张卡上（total_days += 30×张数），不新建；否则才新建一张（发放即生效）。
func (h *EzfyHandler) createLoveCard(uid uint, cfg *model.EzfyCfgItem, count int) {
	if count <= 0 {
		return
	}
	nowMS := time.Now().UnixMilli()
	daily := cfg.Param1
	if daily <= 0 {
		daily = 0
	}
	extra := ezfyLoveCardTotalDays * count
	// 续在同类型、仍有剩余天数的卡上（累加时间）
	var cur model.EzfyLoveCard
	h.DB.Model(&model.EzfyLoveCard{}).
		Where("user_id = ? AND cfg_id = ? AND claimed_days < total_days", uid, cfg.ID).
		Order("created_at ASC, id ASC").First(&cur)
	if cur.ID > 0 {
		h.DB.Model(&model.EzfyLoveCard{}).Where("id = ?", cur.ID).
			Updates(map[string]interface{}{
				"total_days": cur.TotalDays + extra,
				"end_time":   cur.EndTime + int64(extra)*ezfyLoveCardDayMS,
			})
		return
	}
	// 无则新建
	for i := 0; i < count; i++ {
		h.DB.Create(&model.EzfyLoveCard{
			UserId:       uid,
			CfgId:        cfg.ID,
			Name:         cfg.Name,
			DailyDiamond: daily,
			StartTime:    nowMS,
			EndTime:      nowMS + ezfyLoveCardTotalDays*ezfyLoveCardDayMS,
			TotalDays:    ezfyLoveCardTotalDays,
		})
	}
}

// loveCardClaimable 某张卡当前可领天数（发放当天即可领第1天，之后每天+1，封顶总天数）。
func loveCardClaimable(c *model.EzfyLoveCard, nowMS int64) int {
	if c.TotalDays <= 0 {
		return 0
	}
	slots := int((nowMS-c.StartTime)/ezfyLoveCardDayMS) + 1 // 当天算第1天
	if slots < 0 {
		slots = 0
	}
	if slots > c.TotalDays {
		slots = c.TotalDays
	}
	claimable := slots - c.ClaimedDays
	if claimable < 0 {
		return 0
	}
	return claimable
}

// loveCards 玩家全部为爱发电卡（含已领完，按发放倒序）。
func (h *EzfyHandler) loveCards(uid uint) []model.EzfyLoveCard {
	var cards []model.EzfyLoveCard
	h.DB.Where("user_id = ?", uid).Order("created_at DESC, id DESC").Find(&cards)
	return cards
}

// loveCardView 单卡玩家端展示视图。
func loveCardView(c *model.EzfyLoveCard) gin.H {
	remaining := c.TotalDays - c.ClaimedDays
	if remaining < 0 {
		remaining = 0
	}
	return gin.H{
		"id":            c.ID,
		"name":          c.Name,
		"daily_diamond": c.DailyDiamond,
		"start_time":    c.StartTime,
		"end_time":      c.EndTime,
		"total_days":    c.TotalDays,
		"claimed_days":  c.ClaimedDays,
		"remaining":     remaining,
		"claimable":     loveCardClaimable(c, time.Now().UnixMilli()),
	}
}

// loveCardsView 玩家端展示视图（未发放为空数组）。
func (h *EzfyHandler) loveCardsView(uid uint) []gin.H {
	cards := h.loveCards(uid)
	out := make([]gin.H, 0, len(cards))
	for i := range cards {
		out = append(out, loveCardView(&cards[i]))
	}
	return out
}

// loveCardTotalClaimable 玩家所有卡合计可领天数（用于前端判断是否可领取）。
func (h *EzfyHandler) loveCardTotalClaimable(uid uint) int {
	total := 0
	nowMS := time.Now().UnixMilli()
	for _, c := range h.loveCards(uid) {
		total += loveCardClaimable(&c, nowMS)
	}
	return total
}

// LoveCard GET /games/ezfy/love-card —— 玩家查询为爱发电卡状态（未发放为空数组）。
func (h *EzfyHandler) LoveCard(c *gin.Context) {
	uid := middleware.GetUID(c)
	resp.OK(c, gin.H{"love_cards": h.loveCardsView(uid), "total_claimable": h.loveCardTotalClaimable(uid)})
}

// LoveCardClaim POST /games/ezfy/love-card/claim —— 领取每日钻石（多卡叠加、漏领累加、封顶总天数）。
func (h *EzfyHandler) LoveCardClaim(c *gin.Context) {
	uid := middleware.GetUID(c)
	nowMS := time.Now().UnixMilli()
	cards := h.loveCards(uid)
	if len(cards) == 0 {
		resp.OK(c, gin.H{"msg": "未持有为爱发电卡", "diamond_gain": 0})
		return
	}
	var totalGain int64
	login := []string{}
	for i := range cards {
		card := &cards[i]
		claimable := loveCardClaimable(card, nowMS)
		if claimable <= 0 {
			continue
		}
		gain := int64(claimable) * card.DailyDiamond
		if gain < 0 {
			gain = 0
		}
		h.DB.Model(&model.EzfyLoveCard{}).Where("id = ?", card.ID).
			Updates(map[string]interface{}{"claimed_days": card.ClaimedDays + claimable, "last_claim_time": nowMS})
		totalGain += gain
		login = append(login, fmt.Sprintf("%s领%d天(+%d)", card.Name, claimable, gain))
	}
	if totalGain == 0 {
		resp.OK(c, gin.H{"msg": "今日暂无可领取钻石, 明天再来吧", "diamond_gain": 0})
		return
	}
	if err := h.addDiamond(uid, totalGain); err != nil {
		resp.OK(c, gin.H{"msg": "钻石发放失败: " + err.Error(), "diamond_gain": 0})
		return
	}
	msg := "已领取: " + strings.Join(login, "; ")
	resp.OK(c, gin.H{
		"msg":             msg,
		"diamond_gain":    totalGain,
		"diamond_total":   h.ensureProfile(uid).Diamond,
		"love_cards":      h.loveCardsView(uid),
		"total_claimable": h.loveCardTotalClaimable(uid),
	})
}
