package ezfy

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

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

var (
	errLoveCardSettled   = errors.New("love card already settled")
	errLoveCardNoProfile = errors.New("love card player profile missing")
)

// AdminEzfyLoveCardComplete POST /admin/ezfy-love-cards/:id/complete —— 管理端「完成」：
// 把该卡**剩余未领天数**折算成钻石一次性发给玩家，并把卡标记为已领完。
//
// ★ 2026-10-21 用户需求：操作列加「完成」，未领的钻石一次性全部发放给用户。
// ★ 与玩家每日领取共用同一把并发锁：条件更新 claimed_days 仍等于读取值才生效，
//
//	GM 点「完成」和玩家同时领取时只有一方能成功，不会重复发钻石。
func (h *EzfyAdmin) AdminEzfyLoveCardComplete(c *gin.Context) {
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
	days, gain := loveCardSettle(&love)
	if days <= 0 {
		resp.OK(c, gin.H{"msg": "该卡已全部领取，无需再发放", "days": 0, "diamond_gain": 0})
		return
	}
	if gain <= 0 {
		resp.ParamError(c, "卡片每日钻石配置异常，无法完成")
		return
	}

	// 卡片已存在说明玩家档案通常已建；这里仍兜底补建，避免 Update 命中 0 行后漏发。
	ez := h.ezfyH()
	ez.ensureProfile(love.UserId)

	nowMS := time.Now().UnixMilli()
	var balance int64
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.EzfyLoveCard{}).
			Where("id = ? AND claimed_days = ?", love.ID, love.ClaimedDays).
			Updates(map[string]interface{}{"claimed_days": love.TotalDays, "last_claim_time": nowMS})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errLoveCardSettled
		}
		// 钻石原子累加 + 同事务写流水，保证「标记完成」和「到账」要么都成功、要么都回滚。
		pres := tx.Model(&model.EzfyProfile{}).Where("user_id = ?", love.UserId).
			Update("diamond", gorm.Expr("diamond + ?", gain))
		if pres.Error != nil {
			return pres.Error
		}
		if pres.RowsAffected != 1 {
			return errLoveCardNoProfile
		}
		if err := tx.Model(&model.EzfyProfile{}).Where("user_id = ?", love.UserId).
			Pluck("diamond", &balance).Error; err != nil {
			return err
		}
		return tx.Create(&model.EzfyDiamondLog{
			UserId: love.UserId, Change: gain, Balance: balance,
			Reason: "为爱发电卡管理端完成发放",
		}).Error
	})
	if errors.Is(err, errLoveCardSettled) {
		resp.ParamError(c, "该卡已被领取或处理，请刷新后重试")
		return
	}
	if errors.Is(err, errLoveCardNoProfile) {
		resp.ParamError(c, "玩家档案不存在，无法发放")
		return
	}
	if err != nil {
		resp.ServerError(c, err)
		return
	}

	love.ClaimedDays = love.TotalDays
	love.LastClaimTime = nowMS
	msg := fmt.Sprintf("已完成「%s」：剩余 %d 天共 %d 钻石已发给玩家(ID:%d)", love.Name, days, gain, love.UserId)
	resp.OK(c, gin.H{
		"msg": msg, "days": days, "diamond_gain": gain,
		"diamond_total": balance, "card": loveCardView(&love),
	})
}

// ============ 二战风云管理端 · 为爱发电卡（发放/领取情况）============

// AdminEzfyLoveCardOptions GET /admin/ezfy-love-cards/options —— 专属维护页的卡片下拉选项
// （卡片不在通用「数据管理→道具配置 / 发放道具」面展示，这里单独取）。
func (h *EzfyAdmin) AdminEzfyLoveCardOptions(c *gin.Context) {
	// ★ 2026-10-10 为爱发电卡已彻底脱离道具体系（不再写入 ezfy_cfg_item），
	//   下拉选项直接取自内置 loveCardCatalog，不再依赖道具配置表。
	opts := make([]gin.H, 0, len(loveCardCatalog))
	for i := range loveCardCatalog {
		it := &loveCardCatalog[i]
		opts = append(opts, gin.H{"id": it.ID, "name": it.Name, "daily_diamond": it.Daily})
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
