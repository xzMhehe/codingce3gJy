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

type UserHandler struct {
	DB *gorm.DB
}

// 通用分页参数：page 从 1 起，size 默认 defSize（1~100），返回 page、offset、size
func pageOf(c *gin.Context, defSize int) (int, int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", strconv.Itoa(defSize)))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = defSize
	}
	return page, (page - 1) * size, size
}

// 他人主页：资料 + 最新发帖/回帖统计
func (h *UserHandler) Profile(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var user model.User
	if err := h.DB.First(&user, id).Error; err != nil {
		resp.NotFound(c, "这位友友不见了")
		return
	}
	var threadCount, replyCount, signDays int64
	h.DB.Model(&model.Thread{}).Where("user_id = ? AND status = 1", user.ID).Count(&threadCount)
	h.DB.Model(&model.Reply{}).Where("user_id = ? AND status = 1", user.ID).Count(&replyCount)
	h.DB.Model(&model.SignIn{}).Where("user_id = ?", user.ID).Count(&signDays)

	var threads []model.Thread
	h.DB.Preload("Board").Where("user_id = ? AND status = 1", user.ID).
		Order("created_at DESC").Limit(10).Find(&threads)

	var badges []model.Badge
	h.DB.Model(&user).Association("Badges").Find(&badges)
	var roles []model.Role
	h.DB.Model(&user).Association("Roles").Find(&roles)
	online := user.LastActiveAt != nil && time.Since(*user.LastActiveAt) < 10*time.Minute

	// 婚恋状态：伴侣昵称
	partnerName := ""
	if user.PartnerID > 0 {
		var p model.User
		if err := h.DB.First(&p, user.PartnerID).Error; err == nil {
			partnerName = p.Nickname
		}
	}
	// 所属家族
	familyName := ""
	var fm model.FamilyMember
	if err := h.DB.Where("user_id = ?", user.ID).First(&fm).Error; err == nil {
		var fam model.Family
		if err := h.DB.First(&fam, fm.FamilyID).Error; err == nil {
			familyName = fam.Name
		}
	}
	// 最新心情
	mood := ""
	var m model.Mood
	if err := h.DB.Where("user_id = ?", user.ID).Order("created_at DESC").First(&m).Error; err == nil {
		mood = m.Content
	}
	// 家园等级
	hl := homeLevelOf(user.ActiveDays)
	nextDays := 0
	for _, l := range homeLevels {
		if l.Lv == hl+1 {
			nextDays = int(l.Days)
			break
		}
	}
	// 社区职务：职务类马甲（演示站：1.<图标>公坛协管员 2.家族版主）
	dutyIcons := map[string]bool{"706.jpg": true, "704.gif": true, "3.gif": true, "501.gif": true}
	duties := []gin.H{}
	for _, b := range badges {
		if dutyIcons[b.Icon] {
			duties = append(duties, gin.H{"name": b.Name, "icon": b.Icon})
		}
	}

	resp.OK(c, gin.H{
		"id": user.ID, "username": user.Username, "nickname": user.Nickname,
		"gender": user.Gender, "signature": user.Signature, "color": user.Color,
		"avatar": user.Avatar, "level": user.Level, "exp": user.Exp, "coins": user.Coins,
		"level_icon": user.LevelIcon, "level_title": user.LevelTitle,
		"noble": user.Noble, "partner_id": user.PartnerID, "partner_name": partnerName,
		"baby_name": user.BabyName, "achieve": user.Achieve, "achieve_level": user.AchieveLevel,
		"priv_id": user.PrivID, "priv": user.Priv,
		"online":     online,
		"active_days": user.ActiveDays, "home_level": hl, "home_next_days": nextDays,
		"family": familyName, "mood": mood, "city": user.City,
		"created_at": user.CreatedAt, "last_login_at": user.LastLoginAt,
		"thread_count": threadCount, "reply_count": replyCount, "sign_days": signDays,
		"badges": badges, "roles": roles, "duties": duties, "threads": threads,
	})
}

type profileReq struct {
	Signature    string `json:"signature" binding:"max=120"`
	Gender       int    `json:"gender"`
	Nickname     string `json:"nickname" binding:"min=1,max=12"`
	Avatar       string `json:"avatar" binding:"max=100"`
	City         string `json:"city" binding:"max=30"`
	Age          int    `json:"age"`
	BirthYear    int    `json:"birth_year"`
	BirthMonth   int    `json:"birth_month"`
	BirthDay     int    `json:"birth_day"`
	Introduction string `json:"introduction" binding:"max=200"`
}

func (h *UserHandler) UpdateMe(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req profileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "资料格式不对：昵称1-12字，签名120字以内，年龄/生日请填写数字")
		return
	}
	if req.Gender != 1 && req.Gender != 2 {
		req.Gender = 1
	}
	var user model.User
	h.DB.First(&user, uid)
	// 想改名？家园传统：改名要谨慎
	if req.Nickname != user.Nickname {
		var n int64
		h.DB.Model(&model.User{}).Where("nickname = ? AND id <> ?", req.Nickname, uid).Count(&n)
		if n > 0 {
			resp.ParamError(c, "这个昵称已经有友友用啦")
			return
		}
	}
	// 年龄 / 生日 互相推断：填了完整生日 → 算年龄；只填年龄 → 用 now-year 推年份（保留原月日或默认当前月日）
	age := req.Age
	by, bm, bd := req.BirthYear, req.BirthMonth, req.BirthDay
	if by > 0 && bm > 0 && bd > 0 {
		now := time.Now()
		calc := now.Year() - by
		if int(now.Month()) < bm || (int(now.Month()) == bm && now.Day() < bd) {
			calc--
		}
		if calc > 0 {
			age = calc
		}
	} else if age > 0 && (by == 0 || bm == 0 || bd == 0) {
		now := time.Now()
		if by == 0 {
			by = now.Year() - age
		}
		if bm == 0 {
			bm = user.BirthMonth
			if bm == 0 {
				bm = int(now.Month())
			}
		}
		if bd == 0 {
			bd = user.BirthDay
			if bd == 0 {
				bd = now.Day()
			}
		}
	}
	h.DB.Model(&user).Updates(map[string]interface{}{
		"nickname": req.Nickname, "signature": req.Signature,
		"gender": req.Gender, "avatar": req.Avatar, "city": req.City,
		"age": age, "birth_year": by, "birth_month": bm, "birth_day": bd,
		"introduction": req.Introduction,
	})
	resp.OK(c, nil)
}
