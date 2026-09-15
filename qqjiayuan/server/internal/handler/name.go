package handler

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// NameHandler 个性昵称（复刻 3GQQ家园社区 name.html/name_buy.html/name_send.html）
// 规则：购买后才能自定义昵称颜色；该页只管颜色；未购买/过期昵称展示默认蓝色
type NameHandler struct{ DB *gorm.DB }

// DefaultNickColor 普通用户昵称默认颜色（诺哈用户昵称蓝色）
const DefaultNickColor = "#004299"

// 颜色套餐（复刻原站 name_buy：100元宝/月、1000元宝/年）
var namePlans = []gin.H{
	{"id": 1, "name": "100元宝/月", "cost": 100, "days": 30},
	{"id": 2, "name": "1000元宝/年", "cost": 1000, "days": 365},
}

// NameColors 个性昵称预设色板
var NameColors = []string{
	"#004299", "#0066CC", "#3399FF", "#008000", "#009E00", "#8B4513", "#8B008B",
	"#800080", "#FF00FF", "#C00", "#FF4500", "#FF8C00", "#FFD700", "#996600",
	"#000000", "#666666", "#808080",
}

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{3,6}$`)

// nameOpened 个性昵称是否有效
func nameOpened(u *model.User, now time.Time) bool {
	return u.NameEnd != nil && u.NameEnd.After(now)
}

// NameInfo 个性昵称状态/设置页数据
func (h *NameHandler) NameInfo(c *gin.Context) {
	uid := middleware.GetUID(c)
	var u model.User
	if err := h.DB.First(&u, uid).Error; err != nil {
		resp.NotFound(c, "用户不存在")
		return
	}
	now := time.Now()
	opened := nameOpened(&u, now)
	daysLeft := 0
	if opened {
		daysLeft = int(u.NameEnd.Sub(now).Hours()/24) + 1
	} else if u.NameEnd != nil && u.Color != DefaultNickColor {
		// 过期回收：昵称颜色回默认蓝
		h.DB.Model(&model.User{}).Where("id = ?", uid).Update("color", DefaultNickColor)
		u.Color = DefaultNickColor
	}
	resp.OK(c, gin.H{
		"opened": opened, "days_left": daysLeft,
		"end": u.NameEnd, "color": strings.ToLower(u.Color),
		"default_color": DefaultNickColor, "colors": NameColors, "plans": namePlans,
	})
}

// NameBuy 购买个性昵称（pid 1=100元宝/月 2=1000元宝/年）
func (h *NameHandler) NameBuy(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		PID    int `json:"pid"`
		Amount int `json:"amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.PID == 0 || req.Amount < 1 {
		// 复刻原站报错文案
		resp.ParamError(c, "各项不能为空！")
		return
	}
	if req.Amount > 99 {
		req.Amount = 99
	}
	var cost, days int
	for _, p := range namePlans {
		if p["id"].(int) == req.PID {
			cost = p["cost"].(int) * req.Amount
			days = p["days"].(int) * req.Amount
		}
	}
	if cost == 0 {
		resp.ParamError(c, "各项不能为空！")
		return
	}
	var u model.User
	h.DB.First(&u, uid)
	if u.YuanBao < cost {
		resp.ParamError(c, "您的元宝不足！充值")
		return
	}
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("yuanbao", gorm.Expr("yuanbao - ?", cost))
	addWalletLog(h.DB, uid, "name_buy", "购买个性昵称"+strconv.Itoa(req.Amount)+"份", "yuanbao", -cost)
	h.nameExtend(uid, int64(days))
	userLog(h.DB, uid, "购买昵称", "购买个性昵称（"+strconv.Itoa(req.Amount)+"份）", c.ClientIP())
	resp.OK(c, gin.H{"msg": "购买成功！", "amount": req.Amount, "cost": cost})
}

// NameSend 赠送个性昵称（复刻 name_send.html）
func (h *NameHandler) NameSend(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		To     string `json:"to"`
		PID    int    `json:"pid"`
		Amount int    `json:"amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if strings.TrimSpace(req.To) == "" || req.PID == 0 || req.Amount < 1 {
		resp.ParamError(c, "各项不能为空！")
		return
	}
	if req.Amount > 99 {
		req.Amount = 99
	}
	var cost, days int
	for _, p := range namePlans {
		if p["id"].(int) == req.PID {
			cost = p["cost"].(int) * req.Amount
			days = p["days"].(int) * req.Amount
		}
	}
	if cost == 0 {
		resp.ParamError(c, "各项不能为空！")
		return
	}
	var to model.User
	if err := h.DB.Where("username = ? OR nickname = ?", strings.TrimSpace(req.To), strings.TrimSpace(req.To)).First(&to).Error; err != nil {
		resp.NotFound(c, "对方号码不存在")
		return
	}
	if to.ID == uid {
		resp.ParamError(c, "不能赠送给自己")
		return
	}
	var me model.User
	h.DB.First(&me, uid)
	if me.YuanBao < cost {
		resp.ParamError(c, "您的元宝不足！充值")
		return
	}
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("yuanbao", gorm.Expr("yuanbao - ?", cost))
	addWalletLog(h.DB, uid, "name_send", "赠送个性昵称给"+to.Nickname+"("+to.Username+")", "yuanbao", -cost)
	addWalletLog(h.DB, to.ID, "name_send", "收到"+me.Nickname+"("+me.Username+")赠送的个性昵称", "yuanbao", 0)
	h.nameExtend(to.ID, int64(days))
	h.DB.Create(&model.Notification{UserID: to.ID, Type: "system", Title: "收到个性昵称",
		Content: me.Nickname + " 赠送了 " + strconv.Itoa(req.Amount) + " 份个性昵称，去「我的百宝箱>个性昵称」配置颜色吧！"})
	userLog(h.DB, uid, "赠送昵称", "赠送个性昵称给 "+to.Username, c.ClientIP())
	resp.OK(c, gin.H{"msg": "赠送成功！", "to": to.Nickname, "cost": cost})
}

// nameExtend 延长个性昵称有效期（未过期从原到期时间续，已过期/未开通从现在起）
func (h *NameHandler) nameExtend(uid uint, days int64) {
	var u model.User
	h.DB.First(&u, uid)
	now := time.Now()
	var end time.Time
	if nameOpened(&u, now) {
		end = u.NameEnd.AddDate(0, 0, int(days))
	} else {
		end = now.AddDate(0, 0, int(days))
	}
	updates := map[string]interface{}{"name_end": end}
	if u.NameStart == nil {
		updates["name_start"] = now
	}
	h.DB.Model(&model.User{}).Where("id = ?", uid).Updates(updates)
}

// NameSet 配置昵称颜色（仅已开通可用；只能改颜色，其他一律不给改）
// color 可传入单色（#06f），也可传入逐字颜色序列（逗号分隔，逐字变色，微盘站特色功能玩法）
func (h *NameHandler) NameSet(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Color  string   `json:"color"`
		Colors []string `json:"colors"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	colors := req.Colors
	if len(colors) == 0 {
		colors = []string{req.Color}
	}
	var valid []string
	for _, cc := range colors {
		cc = strings.TrimSpace(strings.ToLower(cc))
		if !hexColorRe.MatchString(cc) {
			resp.ParamError(c, "颜色格式不对")
			return
		}
		if len(valid) == 0 || valid[len(valid)-1] != cc {
			valid = append(valid, cc)
		}
	}
	color := strings.Join(valid, ",")
	var u model.User
	h.DB.First(&u, uid)
	if !nameOpened(&u, time.Now()) {
		// 复刻原站：未开通也要先购买
		resp.Fail(c, 200, 450, "您尚未开通个性昵称功能！")
		return
	}
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("color", color)
	userLog(h.DB, uid, "设置昵称颜色", color, c.ClientIP())
	resp.OK(c, gin.H{"msg": "设置成功！", "color": color})
}

// ---- 后台管理（管理后台 > 游戏管理>个性昵称） ----

// AdminNamePlayers 已开通/全部用户列表（word：号码精确/昵称模糊；only=open 只看开通）
func (h *NameHandler) AdminNameUsers(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.User{})
	if word != "" {
		if uid, err := strconv.Atoi(word); err == nil {
			q = q.Where("id = ?", uid)
		} else {
			q = q.Where("nickname LIKE ?", "%"+word+"%")
		}
	}
	switch c.Query("opened") {
	case "1":
		q = q.Where("name_end IS NOT NULL AND name_end > ?", time.Now())
	case "0":
		q = q.Where("name_end IS NULL OR name_end <= ?", time.Now())
	}
	var total int64
	q.Count(&total)
	var list []model.User
	q.Order("name_end IS NULL, id DESC").Offset(offset).Limit(size).Find(&list)
	out := make([]gin.H, 0, len(list))
	now := time.Now()
	for _, u := range list {
		opened := nameOpened(&u, now)
		days := 0
		if opened {
			days = int(u.NameEnd.Sub(now).Hours()/24) + 1
		}
		out = append(out, gin.H{
			"id": u.ID, "username": u.Username, "nickname": u.Nickname,
			"color": u.Color, "multi": strings.Contains(u.Color, ","),
			"name_start": u.NameStart, "name_end": u.NameEnd, "opened": opened, "days_left": days,
		})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminNameUserEdit 调整用户个性昵称：延长/缩短到期、设置或重置颜色、重置开通状态
func (h *NameHandler) AdminNameUserEdit(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	var u model.User
	if err := h.DB.First(&u, uint(uid)).Error; err != nil {
		resp.NotFound(c, "用户不存在")
		return
	}
	var req struct {
		AddDays int     `json:"add_days"`    // >0 续期
		End     *string `json:"end"`         // 直接指定到期时间（yyyy-MM-dd 或 RFC3339）
		Clear   bool    `json:"clear"`       // 重置：清空开通状态
		Color   *string `json:"color"`       // 指定颜色
		Reset   bool    `json:"reset_color"` // 颜色重置为默认蓝
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	updates := map[string]interface{}{}
	if req.AddDays != 0 {
		h.nameExtend(u.ID, int64(req.AddDays))
	}
	if req.End != nil {
		s := strings.TrimSpace(*req.End)
		if s == "" {
			updates["name_end"] = nil
			updates["name_start"] = nil
		} else {
			end, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
			if err != nil {
				if end, err = time.ParseInLocation("2006-01-02", s, time.Local); err != nil {
					end, err = time.Parse(time.RFC3339, s)
				}
			}
			if err != nil {
				resp.ParamError(c, "到期时间格式不对")
				return
			}
			updates["name_end"] = &end
		}
	}
	if req.Clear {
		updates["name_end"] = nil
		updates["name_start"] = nil
	}
	if req.Reset {
		updates["color"] = DefaultNickColor
	}
	if req.Color != nil {
		col := strings.ToLower(strings.TrimSpace(*req.Color))
		if !hexColorRe.MatchString(col) {
			resp.ParamError(c, "颜色格式不对")
			return
		}
		updates["color"] = col
	}
	if len(updates) > 0 {
		h.DB.Model(&model.User{}).Where("id = ?", uid).Updates(updates)
	}
	userLog(h.DB, u.ID, "管理调整", "个人昵称设置调整", c.ClientIP())
	resp.OK(c, gin.H{"msg": "OK"})
}
