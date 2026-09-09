package handler

import (
	"math"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type AchieveHandler struct{ DB *gorm.DB }

func pct(cur, target int) float64 {
	if target <= 0 {
		return 0
	}
	return math.Min(100, float64(cur)/float64(target)*100)
}

// 成就大厅
func (h *AchieveHandler) View(c *gin.Context) {
	uid := middleware.GetUID(c)
	var u model.User
	h.DB.First(&u, uid)

	var friends, chatCount, sign, threadCount, replyCount int64
	h.DB.Model(&model.Friendship{}).Where("user_id = ? AND status = 1", uid).Count(&friends)
	h.DB.Model(&model.ChatMessage{}).Where("user_id = ?", uid).Count(&chatCount)
	h.DB.Model(&model.SignIn{}).Where("user_id = ?", uid).Count(&sign)
	h.DB.Model(&model.Thread{}).Where("user_id = ? AND status = 1", uid).Count(&threadCount)
	h.DB.Model(&model.Reply{}).Where("user_id = ? AND status = 1", uid).Count(&replyCount)

	// 家族/伴侣/花园种植
	family := "无"
	var fm model.FamilyMember
	if err := h.DB.Where("user_id = ?", uid).First(&fm).Error; err == nil {
		var f model.Family
		if err := h.DB.First(&f, fm.FamilyID).Error; err == nil {
			family = f.Name
		}
	}
	partner := u.PartnerID > 0
	var garden int64
	h.DB.Model(&model.GardenPlot{}).Where("user_id = ? AND status = 1", uid).Count(&garden)

	// 总排名（按成就点）
	var rank int64
	h.DB.Model(&model.User{}).Where("achieve > ?", u.Achieve).Count(&rank)
	rank++

	interact := int64(friends)

	// 成就分类
	cats := []gin.H{
		{"name": "家园成就", "items": []gin.H{
			{"name": "家园新手", "percent": pct(int(threadCount), 1), "icon": "101_u.gif"},
			{"name": "后起之秀", "percent": pct(int(replyCount), 10), "icon": "101_u.gif"},
		}},
		{"name": "魔法花园", "items": []gin.H{
			{"name": "魔法花童", "percent": pct(int(garden), 1), "icon": "202_u.jpg"},
			{"name": "魔法花王", "percent": pct(int(garden), 50), "icon": "202_u.jpg"},
		}},
		{"name": "论坛成就", "items": []gin.H{
			{"name": "魔法义工", "percent": pct(int(replyCount), 100), "icon": "301_u.gif"},
			{"name": "魔法学徒", "percent": pct(int(threadCount), 10), "icon": "301_u.gif"},
		}},
		{"name": "家族成就", "items": []gin.H{
			{"name": "找到组织", "percent": boolPct(family != "无"), "icon": "401_u.gif"},
			{"name": "家族新人", "percent": boolPct(family != "无"), "icon": "401_u.gif"},
		}},
		{"name": "互动成就", "items": []gin.H{
			{"name": "单枪匹马", "percent": pct(int(friends), 1), "icon": "501_u.gif"},
			{"name": "成双成对", "percent": boolPct(partner), "icon": "501_u.gif"},
		}},
		{"name": "农场成就", "items": []gin.H{
			{"name": "初入农场", "percent": 0, "icon": "202_u.jpg"},
			{"name": "农场新星", "percent": 0, "icon": "202_u.jpg"},
		}},
		{"name": "聊天成就", "items": []gin.H{
			{"name": "聊客新手", "percent": pct(int(chatCount), 1), "icon": "301_u.gif"},
			{"name": "聊坛新星", "percent": pct(int(chatCount), 100), "icon": "301_u.gif"},
		}},
	}

	resp.OK(c, gin.H{
		"achieve": u.Achieve, "interact": interact, "sign_days": sign,
		"friend_rank": "未入榜", "total_rank": rank,
		"categories": cats, "family": family,
	})
}

func boolPct(b bool) float64 {
	if b {
		return 100
	}
	return 0
}
