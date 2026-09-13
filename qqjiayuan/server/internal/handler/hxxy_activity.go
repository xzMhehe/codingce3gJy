package handler

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 活动模块（复刻原版活动页 xy404：活动列表+领取）
// 1. 七日登录礼 login7：每天可领 1 次，第 1~7 天奖励递增，领满 7 天为一轮重新开始
// 2. 每日活跃 daily：签到20 + 副本30 + 比武30 + 狩猎10只20，50/100 分可领宝箱
// 3. 双倍经验时段 exp2x：12:00-14:00 / 19:00-21:00 战斗经验翻倍（管理端可开关）

// hxLogin7Rewards 七日登录礼奖励表（第N天: 银两, 金豆）
var hxLogin7Rewards = [][2]int64{
	{500, 0}, {800, 0}, {1000, 2}, {1500, 2}, {2000, 3}, {3000, 4}, {5000, 10},
}

// hxExp2xNow 当前是否双倍经验时段（settings 键 hxxy_exp2x，默认开）
func (h *HxxyHandler) hxExp2xNow() bool {
	var val string
	h.DB.Model(&model.Setting{}).Select("`value`").Where("`key` = 'hxxy_exp2x'").Scan(&val)
	if val == "0" {
		return false
	}
	hh := time.Now().Hour()
	return (hh >= 12 && hh < 14) || (hh >= 19 && hh < 21)
}

// hxActiveScore 今日活跃度（签到20 + 副本30 + 比武30 + 狩猎10只20）
func (h *HxxyHandler) hxActiveScore(p *model.HxxyPlayer) int {
	score := 0
	if p.DaySignin > 0 {
		score += 20
	}
	if p.DayDungeon > 0 {
		score += 30
	}
	if p.DayArena > 0 {
		score += 30
	}
	if p.DayHunt >= 10 {
		score += 20
	}
	return score
}

// Activities 活动列表
func (h *HxxyHandler) Activities(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	today := time.Now().Format("2006-01-02")
	// 七日登录礼：累计领取次数 → 第 N 天
	var loginCnt int64
	h.DB.Model(&model.HxxyActivityLog{}).Where("player_id = ? AND act = 'login7'", p.ID).Count(&loginCnt)
	loginDay := int(loginCnt)%7 + 1
	loginDone := false
	var todayLogin int64
	h.DB.Model(&model.HxxyActivityLog{}).Where("player_id = ? AND act = 'login7' AND day = ?", p.ID, today).Count(&todayLogin)
	loginDone = todayLogin > 0
	rw := hxLogin7Rewards[loginDay-1]
	// 每日活跃
	score := h.hxActiveScore(p)
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
	resp.OK(c, gin.H{
		"login7": gin.H{"day": loginDay, "money": rw[0], "beans": rw[1], "claimed": loginDone,
			"days": hxLogin7Rewards},
		"daily":    gin.H{"score": score, "tiers": daily, "signed": p.DaySignin > 0, "dungeon": p.DayDungeon > 0, "arena": p.DayArena > 0, "hunt": p.DayHunt},
		"exp2x":    gin.H{"on": h.hxExp2xNow(), "windows": "12:00-14:00、19:00-21:00"},
	})
}

// ActivityClaim 领取活动奖励（act: login7 / daily50 / daily100）
func (h *HxxyHandler) ActivityClaim(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		Act  string `json:"act"`
		Tier int    `json:"tier"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Act == "" {
		resp.ParamError(c, "参数错误")
		return
	}
	today := time.Now().Format("2006-01-02")
	var existed int64
	h.DB.Model(&model.HxxyActivityLog{}).Where("player_id = ? AND act = ? AND day = ?", p.ID, in.Act, today).Count(&existed)
	if existed > 0 {
		resp.ParamError(c, "今日已领取过该活动奖励")
		return
	}
	msg := ""
	switch in.Act {
	case "login7":
		var cnt int64
		h.DB.Model(&model.HxxyActivityLog{}).Where("player_id = ? AND act = 'login7'", p.ID).Count(&cnt)
		day := int(cnt)%7 + 1
		rw := hxLogin7Rewards[day-1]
		h.hxWallet(p, "money", rw[0], "七日登录礼第"+fmt.Sprint(day)+"天")
		if rw[1] > 0 {
			h.hxWallet(p, "beans", rw[1], "七日登录礼")
		}
		msg = fmt.Sprintf("领取七日登录礼第%d天奖励：%d银两", day, rw[0])
		if rw[1] > 0 {
			msg += fmt.Sprintf(" + %d金豆", rw[1])
		}
	case "daily50", "daily100":
		need := 50
		if in.Act == "daily100" {
			need = 100
		}
		score := h.hxActiveScore(p)
		if score < need {
			resp.ParamError(c, fmt.Sprintf("今日活跃度不足（%d/%d）", score, need))
			return
		}
		money, beans := int64(1000), 0
		if need == 100 {
			money, beans = 3000, 5
		}
		h.hxWallet(p, "money", money, "每日活跃奖励")
		if beans > 0 {
			h.hxWallet(p, "beans", int64(beans), "每日活跃奖励")
		}
		msg = fmt.Sprintf("领取每日活跃%d分宝箱：%d银两", need, money)
		if beans > 0 {
			msg += fmt.Sprintf(" + %d金豆", beans)
		}
	default:
		resp.ParamError(c, "未知活动")
		return
	}
	h.DB.Create(&model.HxxyActivityLog{PlayerID: p.ID, Act: in.Act, Day: today, Tier: in.Tier})
	resp.OK(c, gin.H{"msg": msg})
}
