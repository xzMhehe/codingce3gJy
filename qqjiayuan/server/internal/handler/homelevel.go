package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type HomeLevelHandler struct{ DB *gorm.DB }

// 家园等级表：升级所需活跃天数（对齐参考站 column/20）
var homeLevels = []struct {
	Lv   int
	Days float64
}{
	{1, 5}, {2, 12}, {3, 21}, {4, 32}, {5, 45}, {6, 60}, {7, 77}, {8, 96},
	{9, 117}, {10, 140}, {11, 165}, {12, 192}, {13, 221}, {14, 252}, {15, 285}, {16, 320}, {17, 357},
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

// 我的家园等级（等级=用户数据库等级，图标按性别 男1_女2）
func (h *HomeLevelHandler) View(c *gin.Context) {
	uid := middleware.GetUID(c)
	var u model.User
	h.DB.First(&u, uid)
	lv := u.Level
	if lv < 1 {
		lv = 1
	}
	if lv > 17 {
		lv = 17
	}
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

	resp.OK(c, gin.H{
		"active_days": u.ActiveDays, "level": lv, "icon": icon,
		"next_days": next, "gender": u.Gender, "levels": homeLevels,
	})
}

func pad2(n int) string { return strconv.Itoa(n) }
