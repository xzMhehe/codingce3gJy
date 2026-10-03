package ezfy

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// AdminEzfyLoveCardDelete DELETE /admin/ezfy-love-cards/:id —— 删除某条为爱发电卡记录。
func (h *EzfyAdmin) AdminEzfyLoveCardDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var love model.EzfyLoveCard
	if err := h.DB.First(&love, id).Error; err != nil {
		resp.NotFound(c, "记录不存在")
		return
	}
	if err := h.DB.Delete(&model.EzfyLoveCard{}, id).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	resp.OK(c, gin.H{"msg": "已删除「" + love.Name + "」(ID:" + strconv.FormatUint(uint64(love.UserId), 10) + ")"})
}

// ============ 二战风云管理端 · 为爱发电卡（发放/领取情况）============

// AdminEzfyLoveCardOptions GET /admin/ezfy-love-cards/options —— 专属维护页的卡片下拉选项
// （卡片不在通用「数据管理→道具配置 / 发放道具」面展示，这里单独取）。
func (h *EzfyAdmin) AdminEzfyLoveCardOptions(c *gin.Context) {
	var items []model.EzfyCfgItem
	h.DB.Where("item_type IN ?", []int{ezfyItemTypeLoveCard, ezfyItemTypeLoveCardPro}).
		Order("id ASC").Find(&items)
	opts := make([]gin.H, 0, len(items))
	for i := range items {
		it := items[i]
		opts = append(opts, gin.H{"id": it.ID, "name": it.Name, "daily_diamond": it.Param1})
	}
	resp.OK(c, gin.H{"list": opts, "total": len(opts)})
}

// AdminEzfyLoveCards GET /admin/ezfy-love-cards/list —— 查看为爱发电卡发放与领取情况。
// word 可按玩家昵称 / 游戏ID 过滤；返回卡片激活记录（含领取进度）。
func (h *EzfyAdmin) AdminEzfyLoveCards(c *gin.Context) {
	word := strings.TrimSpace(c.Query("word"))
	page := atoiDefault(c.Query("page"), 1)
	size := atoiDefault(c.Query("size"), 20)
	if size < 1 {
		size = 20
	}
	if size > 200 {
		size = 200
	}
	if page < 1 {
		page = 1
	}

	db := h.DB.Model(&model.EzfyLoveCard{})
	// 按玩家过滤：先解析目标 user_id 范围（昵称 → 多行；游戏ID → 单行）
	if word != "" {
		var uids []uint
		h.DB.Model(&model.EzfyProfile{}).
			Where("nickname LIKE ?", "%"+word+"%").Pluck("user_id", &uids)
		if uidInt, err := strconv.Atoi(word); err == nil {
			uids = append(uids, uint(uidInt))
		}
		if len(uids) == 0 {
			uids = []uint{0} // 命中空集，保证返回空结果
		}
		db = db.Where("user_id IN ?", uids)
	}

	var total int64
	db.Count(&total)
	var cards []model.EzfyLoveCard
	db.Order("created_at DESC, id DESC").
		Offset((page - 1) * size).Limit(size).Find(&cards)

	// 玩家昵称 map
	nickMap := map[uint]string{}
	if len(cards) > 0 {
		var ids []uint
		for _, c := range cards {
			ids = append(ids, c.UserId)
		}
		var ps []model.EzfyProfile
		h.DB.Select("user_id, nickname, game_uid").Where("user_id IN ?", ids).Find(&ps)
		for _, p := range ps {
			nickMap[p.UserID] = p.Nickname
		}
	}

	var out = make([]gin.H, 0, len(cards))
	for _, c := range cards {
		remaining := c.TotalDays - c.ClaimedDays
		if remaining < 0 {
			remaining = 0
		}
		out = append(out, gin.H{
			"id":              c.ID,
			"user_id":         c.UserId,
			"nickname":        nickMap[c.UserId],
			"name":            c.Name,
			"daily_diamond":   c.DailyDiamond,
			"start_time":      c.StartTime,
			"end_time":        c.EndTime,
			"total_days":      c.TotalDays,
			"claimed_days":    c.ClaimedDays,
			"remaining":       remaining,
			"last_claim_time": c.LastClaimTime,
			"created_at":      c.CreatedAt.UnixMilli(),
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}
