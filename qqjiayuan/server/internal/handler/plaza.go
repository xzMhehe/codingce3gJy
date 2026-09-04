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

type PlazaHandler struct {
	DB *gorm.DB
}

// 广场首页聚合数据：公告广播、在线人数、最新友友、社区头条、各频道最新帖、友友动态
func (h *PlazaHandler) Index(c *gin.Context) {
	db := h.DB

	var announcements []model.Announcement
	db.Where("type = ? AND status = 1", "notice").Order("created_at DESC").Limit(2).Find(&announcements)
	var broadcasts []model.Announcement
	db.Where("type = ? AND status = 1", "broadcast").Order("created_at DESC").Limit(1).Find(&broadcasts)
	var activities []model.Announcement
	db.Where("type = ? AND status = 1", "activity").Order("created_at DESC").Limit(1).Find(&activities)

	var onlineCount, userCount int64
	tenMinAgo := time.Now().Add(-10 * time.Minute)
	db.Model(&model.User{}).Where("last_active_at > ?", tenMinAgo).Count(&onlineCount)
	db.Model(&model.User{}).Count(&userCount)
	var newestUser model.User
	db.Order("id DESC").First(&newestUser)

	// T台秀：资历最深的友友（经验最高）
	var ttou model.User
	db.Where("status = 1").Order("exp DESC").First(&ttou)
	ttouOut := gin.H{}
	if ttou.ID > 0 {
		ttouOut = gin.H{"id": ttou.ID, "nickname": ttou.Nickname, "color": ttou.Color,
			"avatar": ttou.Avatar, "signature": ttou.Signature, "exp": ttou.Exp}
	}

	// 社区快报 = 最新发帖；家园活跃 = 最新被回复
	var quickThreads []model.Thread
	db.Preload("User").Preload("User.Badges").Where("status = 1").Order("created_at DESC").Limit(5).Find(&quickThreads)
	var activeThreads []model.Thread
	db.Preload("User").Preload("User.Badges").Where("status = 1").Order("IFNULL(last_reply_at, created_at) DESC").Limit(5).Find(&activeThreads)

	// 社区头条 = 精华帖
	var fineThreads []model.Thread
	db.Preload("User").Preload("User.Badges").Where("status = 1 AND is_fine = 1").Order("created_at DESC").Limit(3).Find(&fineThreads)

	// 频道最新帖：公共论坛 / 家族大厅 / 同城客栈
	var channels []model.Board
	db.Where("parent_id = 0").Order("sort ASC").Find(&channels)
	channelData := []gin.H{}
	for _, ch := range channels {
		var subs []model.Board
		db.Where("parent_id = ?", ch.ID).Order("sort ASC").Find(&subs)
		ids := []uint{ch.ID}
		for _, s := range subs {
			ids = append(ids, s.ID)
		}
		var latest []model.Thread
		db.Preload("User").Preload("User.Badges").Where("board_id IN ? AND status = 1", ids).
			Order("IFNULL(last_reply_at, created_at) DESC").Limit(5).Find(&latest)
		channelData = append(channelData, gin.H{"channel": ch, "subs": subs, "threads": latest})
	}

	// 友友动态：最新发帖/回帖流
	type Dynamic struct {
		ID        uint      `json:"id"`
		UserID    uint      `json:"user_id"`
		Nickname  string    `json:"nickname"`
		Color     string    `json:"color"`
		Action    string    `json:"action"`
		ThreadID  uint      `json:"thread_id"`
		Title     string    `json:"title"`
		CreatedAt time.Time `json:"created_at"`
	}
	dynamics := []Dynamic{}
	var newThreads []model.Thread
	db.Preload("User").Preload("User.Badges").Where("status = 1").Order("created_at DESC").Limit(5).Find(&newThreads)
	for _, t := range newThreads {
		name, color := "神秘友友", ""
		if t.User != nil {
			name, color = t.User.Nickname, t.User.Color
		}
		dynamics = append(dynamics, Dynamic{ID: t.ID * 10, UserID: t.UserID, Nickname: name, Color: color,
			Action: "发表了帖子", ThreadID: t.ID, Title: t.Title, CreatedAt: t.CreatedAt})
	}
	var newReplies []model.Reply
	db.Preload("User").Preload("User.Badges").Preload("Thread").Where("status = 1").Order("created_at DESC").Limit(10).Find(&newReplies)
	for _, r := range newReplies {
		name, color := "神秘友友", ""
		if r.User != nil {
			name, color = r.User.Nickname, r.User.Color
		}
		title := "未知帖子"
		if r.Thread != nil {
			title = r.Thread.Title
		}
		dynamics = append(dynamics, Dynamic{ID: r.ID*10 + 1, UserID: r.UserID, Nickname: name, Color: color,
			Action: "回复了帖子", ThreadID: r.ThreadID, Title: title, CreatedAt: r.CreatedAt})
	}
	// 按时间倒序取前 8 条
	for i := 0; i < len(dynamics); i++ {
		for j := i + 1; j < len(dynamics); j++ {
			if dynamics[j].CreatedAt.After(dynamics[i].CreatedAt) {
				dynamics[i], dynamics[j] = dynamics[j], dynamics[i]
			}
		}
	}
	if len(dynamics) > 8 {
		dynamics = dynamics[:8]
	}

	resp.OK(c, gin.H{
		"announcements": announcements, "broadcasts": broadcasts, "activities": activities,
		"online_count": onlineCount, "user_count": userCount,
		"newest_user":  gin.H{"id": newestUser.ID, "nickname": newestUser.Nickname},
		"fine_threads": fineThreads, "channels": channelData, "dynamics": dynamics,
		"quick_threads": quickThreads, "active_threads": activeThreads,
		"ttou": ttouOut,
	})
}

// 公开：广场板块开关（前端按 enabled 显示）
func (h *PlazaHandler) Sections(c *gin.Context) {
	var list []model.PlazaSection
	h.DB.Order("sort ASC").Find(&list)
	resp.OK(c, list)
}

// 后台：广场板块列表
func (h *PlazaHandler) AdminSections(c *gin.Context) {
	var list []model.PlazaSection
	h.DB.Order("sort ASC").Find(&list)
	resp.OK(c, list)
}

type sectReq struct {
	Enabled int `json:"enabled"`
}

// 后台：更新板块开关
func (h *PlazaHandler) AdminSectionUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req sectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数有误")
		return
	}
	if req.Enabled != 0 {
		req.Enabled = 1
	}
	h.DB.Model(&model.PlazaSection{}).Where("id = ?", id).Update("enabled", req.Enabled)
	resp.OK(c, nil)
}

// 全站搜索：搜帖子标题 + 友友昵称
func (h *PlazaHandler) Search(c *gin.Context) {
	word := c.Query("word")
	if word == "" {
		resp.ParamError(c, "请输入搜索内容")
		return
	}
	like := "%" + word + "%"
	var threads []model.Thread
	h.DB.Preload("User").Preload("Board").
		Where("status = 1 AND (title LIKE ? OR content LIKE ?)", like, like).
		Order("created_at DESC").Limit(20).Find(&threads)
	var users []model.User
	h.DB.Where("nickname LIKE ?", like).Limit(10).Find(&users)
	resp.OK(c, gin.H{"threads": threads, "users": users})
}

// 签到
type SignHandler struct{ DB *gorm.DB }

func todayStr() string { return time.Now().Format("2006-01-02") }

func (h *SignHandler) Do(c *gin.Context) {
	uid := middleware.GetUID(c)
	date := todayStr()
	var exist model.SignIn
	if err := h.DB.Where("user_id = ? AND sign_date = ?", uid, date).First(&exist).Error; err == nil {
		resp.ParamError(c, "今天已经签到过啦，明天再来吧")
		return
	}
	var last model.SignIn
	consec := 1
	if err := h.DB.Where("user_id = ?", uid).Order("sign_date DESC").First(&last).Error; err == nil {
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		if last.SignDate == yesterday {
			consec = last.Consec + 1
		}
	}
	reward := 10 + consec
	if reward > 17 {
		reward = 17
	}
	sign := model.SignIn{UserID: uid, SignDate: date, Consec: consec, Reward: reward}
	if err := h.DB.Create(&sign).Error; err != nil {
		resp.ParamError(c, "今天已经签到过啦")
		return
	}
	addExpAndCoins(h.DB, uid, 20, reward, 2)
	var u model.User
	h.DB.First(&u, uid)
	resp.OK(c, gin.H{"consec": consec, "reward": reward, "coins": u.Coins, "exp": u.Exp, "level": u.Level})
}

// 签到状态与排行（连续天数榜）
func (h *SignHandler) Info(c *gin.Context) {
	uid := middleware.GetUID(c)
	var signedToday int64
	h.DB.Model(&model.SignIn{}).Where("user_id = ? AND sign_date = ?", uid, todayStr()).Count(&signedToday)

	var consec int64
	var last model.SignIn
	if err := h.DB.Where("user_id = ?", uid).Order("sign_date DESC").First(&last).Error; err == nil {
		if last.SignDate == todayStr() || last.SignDate == time.Now().AddDate(0, 0, -1).Format("2006-01-02") {
			consec = int64(last.Consec)
		}
	}
	var totalDays int64
	h.DB.Model(&model.SignIn{}).Where("user_id = ?", uid).Count(&totalDays)

	type rankRow struct {
		UserID   uint   `json:"user_id"`
		Nickname string `json:"nickname"`
		Color    string `json:"color"`
		Consec   int    `json:"consec"`
	}
	var rank []rankRow
	h.DB.Raw(`SELECT s.user_id, u.nickname, u.color, MAX(s.consec) AS consec
FROM sign_ins s JOIN users u ON u.id = s.user_id
GROUP BY s.user_id, u.nickname, u.color ORDER BY consec DESC LIMIT 10`).Scan(&rank)

	// 今日已签人数
	var todayCount int64
	h.DB.Model(&model.SignIn{}).Where("sign_date = ?", todayStr()).Count(&todayCount)

	resp.OK(c, gin.H{
		"signed_today": signedToday > 0, "consec": consec, "total_days": totalDays,
		"rank": rank, "today_count": todayCount,
	})
}
