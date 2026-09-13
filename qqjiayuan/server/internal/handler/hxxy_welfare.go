package handler

// 福利中心（复刻原版 幻想西游福利中心 xy307/408/409/410/417/418/419/420）
// 1. 每日福利：神秘礼物（每天10份，越领等待越久，随机奖励）
// 2. 每日活跃：活跃度任务（签到/副本/比武/狩猎）可领宝箱（复用 Activities）
// 3. 每日签到：复用 SigninInfo / SigninClaim
// 4. 每日宣传：前端静态推广文案 + 一键复制
// 5. 贵族（黄金/铂金/钻石/至尊）：开通月卡后每日领取金豆+"XX贵族宝箱"

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// hxGiftWaits 神秘礼物每份等待秒数（第1份即领，之后递增）
var hxGiftWaits = []int{1, 300, 300, 300, 300, 600, 600, 600, 1200, 1200}

// hxNobleDef 贵族月卡定义
var hxNobleDef = []struct {
	Tier int
	Name string
	Price string
	Beans int
	Box   string
	Intro string
}{
	{1, "黄金贵族", "10元/30次", 2, "黄金贵族宝箱", "玩家开通【黄金贵族】后每日领取〖金豆〗x2，〖黄金贵族宝箱〗x1"},
	{2, "铂金贵族", "20元/30次", 6, "铂金贵族宝箱", "玩家开通【铂金贵族】后每日领取〖金豆〗x6，〖铂金贵族宝箱〗x1"},
	{3, "钻石皇族", "50元/30次", 15, "钻石皇族宝箱", "玩家开通【钻石皇族】后每日领取〖金豆〗x15，〖钻石皇族宝箱〗x1"},
	{4, "至尊皇族", "100元/30次", 30, "至尊皇族宝箱", "玩家开通【至尊皇族】后每日领取〖金豆〗x30，〖至尊皇族宝箱〗x1"},
}

// hxGiftTodayCount 今日神秘礼物已领份数
func (h *HxxyHandler) hxGiftTodayCount(playerID uint, day string) int {
	var cnt int64
	h.DB.Model(&model.HxxyActivityLog{}).Where("player_id = ? AND act = 'gift' AND day = ?", playerID, day).Count(&cnt)
	return int(cnt)
}

// hxGiftState 神秘礼物状态
func (h *HxxyHandler) hxGiftState(p *model.HxxyPlayer, today string) gin.H {
	done := h.hxGiftTodayCount(p.ID, today)
	if done >= 10 {
		return gin.H{"status": "done", "done": done, "total": 10, "name": "神秘礼物", "msg": "【神秘礼物】(请明日再来)"}
	}
	next := done + 1 // 第 N 份
	var remaining = 0
	if done > 0 {
		var last model.HxxyActivityLog
		h.DB.Where("player_id = ? AND act = 'gift' AND day = ?", p.ID, today).Order("created_at desc").First(&last)
		need := hxGiftWaits[done-1]
		elapsed := int(time.Since(last.CreatedAt).Seconds())
		remaining = need - elapsed
		if remaining < 0 {
			remaining = 0
		}
	}
	var ready = done == 0 || remaining == 0
	return gin.H{"status": "wait", "done": done, "total": 10, "next": next, "remaining": remaining, "ready": ready}
}

// Welfare 福利中心数据
func (h *HxxyHandler) Welfare(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	today := time.Now().Format("2006-01-02")
	gift := h.hxGiftState(p, today)
	score := h.hxActiveScore(p)
	// 活跃度任务
	tasks := []gin.H{
		{"name": "每日签到", "done": p.DaySignin > 0, "need": 1, "score": 20, "total": 20},
		{"name": "副本挑战", "done": p.DayDungeon > 0, "need": 1, "score": 30, "total": 30},
		{"name": "比武夺魁", "done": p.DayArena > 0, "need": 1, "score": 30, "total": 30},
		{"name": "狩猎妖魔", "done": p.DayHunt >= 10, "need": 10, "score": 20, "total": 20},
	}
	daily := []gin.H{}
	for _, tier := range []struct {
		need  int
		money int64
		beans int
		name  string
	}{{50, 1000, 0, "活跃宝箱(小)"}, {100, 3000, 5, "活跃宝箱(大)"}} {
		var done int64
		h.DB.Model(&model.HxxyActivityLog{}).Where("player_id = ? AND act = ? AND day = ?", p.ID, fmt.Sprintf("daily%d", tier.need), today).Count(&done)
		daily = append(daily, gin.H{"tier": tier.need, "name": tier.name, "money": tier.money, "beans": tier.beans,
			"claimed": done > 0, "can": score >= tier.need})
	}
	// 贵族
	nobles := []gin.H{}
	now := time.Now()
	for _, nd := range hxNobleDef {
		owned := false
		var last model.HxxyActivityLog
		if err := h.DB.Where("player_id = ? AND act = 'noble_own' AND tier = ?", p.ID, nd.Tier).Order("created_at desc").First(&last).Error; err == nil {
			exp, _ := time.Parse("2006-01-02", last.Day)
			owned = !exp.IsZero() && !exp.Before(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()))
		}
		var claimed int64
		if owned {
			h.DB.Model(&model.HxxyActivityLog{}).Where("player_id = ? AND act = 'noble_claim' AND tier = ? AND day = ?", p.ID, nd.Tier, today).Count(&claimed)
		}
		nobles = append(nobles, gin.H{"tier": nd.Tier, "name": nd.Name, "price": nd.Price, "beans": nd.Beans,
			"box": nd.Box, "intro": nd.Intro, "owned": owned, "claimed": claimed > 0})
	}
	resp.OK(c, gin.H{
		"gift":  gift,
		"active": gin.H{"score": score, "tasks": tasks, "tiers": daily, "total": 100},
		"vip":   gin.H{"level": p.VipLv},
		"nobles": nobles,
	})
}

// WelfareGiftClaim 领取神秘礼物
func (h *HxxyHandler) WelfareGiftClaim(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	today := time.Now().Format("2006-01-02")
	st := h.hxGiftState(p, today)
	if st["status"] == "done" {
		resp.ParamError(c, "今日10份神秘礼物已领完，明日再来")
		return
	}
	if !st["ready"].(bool) {
		resp.ParamError(c, fmt.Sprintf("神秘礼物冷却中，还需等待%d秒", st["remaining"].(int)))
		return
	}
	idx := st["done"].(int) // 当前第 idx 份（0-based）
	if idx >= 10 {
		resp.ParamError(c, "今日神秘礼物已领完")
		return
	}
	// 随机奖励：银两 5000-50000，10% 概率金豆 1-5
	money := int64(5000 + rand.Intn(45000))
	msg := fmt.Sprintf("领取第%d份神秘礼物：银两%d", idx+1, money)
	h.hxWallet(p, "money", money, "神秘礼物第"+strconv.Itoa(idx+1)+"份")
	if rand.Intn(10) == 0 {
		b := 1 + rand.Intn(5)
		h.hxWallet(p, "beans", int64(b), "神秘礼物")
		msg += fmt.Sprintf(" + 金豆%d", b)
	}
	h.DB.Create(&model.HxxyActivityLog{PlayerID: p.ID, Act: "gift", Day: today, Tier: idx})
	resp.OK(c, gin.H{"msg": msg})
}

// WelfareNobleClaim 领取贵族每日奖励
func (h *HxxyHandler) WelfareNobleClaim(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		Tier int `json:"tier"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Tier < 1 || in.Tier > 4 {
		resp.ParamError(c, "参数错误")
		return
	}
	nd := hxNobleDef[in.Tier-1]
	today := time.Now().Format("2006-01-02")
	// 校验是否开通且未到期
	var last model.HxxyActivityLog
	if err := h.DB.Where("player_id = ? AND act = 'noble_own' AND tier = ?", p.ID, in.Tier).Order("created_at desc").First(&last).Error; err != nil {
		resp.ParamError(c, "亲！【"+nd.Name+"】已到期，或者未开通（在游戏左下角充值联系GM并告知开通月卡）")
		return
	}
	exp, _ := time.Parse("2006-01-02", last.Day)
	now := time.Now()
	if exp.IsZero() || exp.Before(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())) {
		resp.ParamError(c, "亲！【"+nd.Name+"】已到期，或者未开通（在游戏左下角充值联系GM并告知开通月卡）")
		return
	}
	var claimed int64
	h.DB.Model(&model.HxxyActivityLog{}).Where("player_id = ? AND act = 'noble_claim' AND tier = ? AND day = ?", p.ID, in.Tier, today).Count(&claimed)
	if claimed > 0 {
		resp.ParamError(c, "今日已领取【"+nd.Name+"】奖励，明天再来")
		return
	}
	h.hxWallet(p, "beans", int64(nd.Beans), nd.Name+"每日奖励")
	msg := fmt.Sprintf("领取【%s】每日奖励：金豆%d + 〖%s〗x1", nd.Name, nd.Beans, nd.Box)
	h.DB.Create(&model.HxxyActivityLog{PlayerID: p.ID, Act: "noble_claim", Day: today, Tier: in.Tier})
	resp.OK(c, gin.H{"msg": msg})
}