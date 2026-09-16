package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type HomeLevelHandler struct{ DB *gorm.DB }

// 家园等级表：升级所需活跃天数（对齐参考站 column/20，最高 50 级）
var homeLevels = []struct {
	Lv   int
	Days float64
}{
	{1, 5}, {2, 12}, {3, 21}, {4, 32}, {5, 45}, {6, 60}, {7, 77}, {8, 96},
	{9, 117}, {10, 140}, {11, 165}, {12, 192}, {13, 221}, {14, 252}, {15, 285}, {16, 320}, {17, 357},
	{18, 396}, {19, 437}, {20, 480}, {21, 525}, {22, 572}, {23, 621}, {24, 672}, {25, 725},
	{26, 780}, {27, 837}, {28, 896}, {29, 957}, {30, 1020}, {31, 1085}, {32, 1152}, {33, 1221},
	{34, 1292}, {35, 1365}, {36, 1440}, {37, 1517}, {38, 1596}, {39, 1677}, {40, 1760},
	{41, 1845}, {42, 1932}, {43, 2021}, {44, 2112}, {45, 2205}, {46, 2300}, {47, 2397},
	{48, 2496}, {49, 2597}, {50, 2700},
}

func homeLevelOf(days float64) int {
	lv := 1
	for _, l := range homeLevels {
		if days >= l.Days {
			lv = l.Lv
		}
	}
	return lv
}

// homeLevelOfMinDays 家园等级对应的最低活跃天数（诺哈 1 级 5 天，最高 50 级 2700 天）。小于 1 返回 0。
func homeLevelOfMinDays(lv int) float64 {
	if lv <= 0 {
		return 0
	}
	for _, l := range homeLevels {
		if l.Lv == lv {
			return l.Days
		}
	}
	return 0
}

// homeLevelOfMaxDays 家园等级对应的最大活跃天数：下一级最低天数减 0.5（50 级封顶 2700）。非等级范围返回 0。
func homeLevelOfMaxDays(lv int) float64 {
	if lv <= 0 {
		return 0
	}
	for _, l := range homeLevels {
		if l.Lv == lv+1 {
			return l.Days - 0.5
		}
	}
	return homeLevelOfMinDays(50)
}

// 我的家园等级：按活跃天数计算（诺哈公式 n²+4n，与论坛等级 user.Level 分开），最高 50 级
func (h *HomeLevelHandler) View(c *gin.Context) {
	uid := middleware.GetUID(c)
	var u model.User
	h.DB.First(&u, uid)
	lv := homeLevelOf(u.ActiveDays)
	sex := "1"
	if u.Gender == 2 {
		sex = "2"
	}
	icon := "/static/picture/home_" + sex + "_" + pad2(lv) + ".gif"

	next := 0
	for _, l := range homeLevels {
		if l.Lv == lv+1 {
			next = int(l.Days)
			break
		}
	}

	// 今日活跃进度：基准 1 点/天，连续登录 1.2 点；超Q/蓝钻在有效期内每天 +0.1/级（封顶 +1.0）
	speed := 1.0
	now := time.Now()
	if u.LastActiveDate == "" || u.LastActiveDate == now.AddDate(0, 0, -1).Format("2006-01-02") {
		speed = 1.2
	}
	if u.QqEnd != nil && u.QqEnd.After(now) && u.QqLv > 0 {
		speed += 0.1 * float64(u.QqLv)
	}
	if u.BlueEnd != nil && u.BlueEnd.After(now) && u.BlueLv > 0 {
		speed += 0.1 * float64(u.BlueLv)
	}
	if speed > 2.2 {
		speed = 2.2
	}

	resp.OK(c, gin.H{
		"active_days": u.ActiveDays, "level": lv, "icon": icon,
		"next_days": next, "gender": u.Gender, "levels": homeLevels,
		"today_speed": speed,
	})
}

func pad2(n int) string { return strconv.Itoa(n) }
