package handler

import (
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

type UserHandler struct {
	DB        *gorm.DB
	StaticDir string // 如 ../web/dist/static
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

// ---- 我的头像（复刻参考站 /home/face.html） ----

// 推荐头像文件名规则：static/picture 下长数字命名的图片（参考站头像素材）
var avatarPresetRe = regexp.MustCompile(`^\d{6,}\.(jpg|jpeg|gif|png)$`)

// 当前头像
func (h *UserHandler) MyAvatar(c *gin.Context) {
	uid := middleware.GetUID(c)
	var u model.User
	if err := h.DB.First(&u, uid).Error; err != nil {
		resp.NotFound(c, "用户不存在")
		return
	}
	resp.OK(c, gin.H{"avatar": u.Avatar, "avatar_base64": u.AvatarBase64})
}

// 上传自定义头像：base64(data URI) 存库，单独字段 avatar_base64
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Data string `json:"data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择图片")
		return
	}
	if !strings.HasPrefix(req.Data, "data:image/") || !strings.Contains(req.Data, ";base64,") {
		resp.ParamError(c, "仅支持 JPG/PNG/GIF/WEBP 图片")
		return
	}
	idx := strings.Index(req.Data, ";base64,")
	raw := req.Data[idx+len(";base64,"):]
	if len(raw)*3/4 > 500*1024 { // base64 解码后约 500KB 上限
		resp.ParamError(c, "图片不能超过 500KB，请压缩后上传")
		return
	}
	if len(raw) < 16 {
		resp.ParamError(c, "图片数据不完整")
		return
	}
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("avatar_base64", req.Data)
	resp.OK(c, nil)
}

// 推荐头像列表（分页）
func (h *UserHandler) AvatarPresets(c *gin.Context) {
	_, offset, size := pageOf(c, 12)
	var files []string
	if entries, err := os.ReadDir(filepath.Join(h.StaticDir, "picture")); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if avatarPresetRe.MatchString(e.Name()) {
				files = append(files, e.Name())
			}
		}
	}
	sort.Strings(files)
	total := len(files)
	start := offset
	if start > total {
		start = total
	}
	end := offset + size
	if end > total {
		end = total
	}
	resp.OK(c, gin.H{"total": total, "page": offset/size + 1, "size": size, "list": files[start:end]})
}

// 设置推荐头像（点击图片即设为头像）
func (h *UserHandler) SetPresetAvatar(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		File string `json:"file" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择头像")
		return
	}
	if !avatarPresetRe.MatchString(req.File) {
		resp.ParamError(c, "头像不存在")
		return
	}
	if _, err := os.Stat(filepath.Join(h.StaticDir, "picture", req.File)); err != nil {
		resp.ParamError(c, "头像不存在")
		return
	}
	// 设置推荐头像：写 avatar 文件名，同时清掉自定义 base64（避免优先级混乱）
	h.DB.Model(&model.User{}).Where("id = ?", uid).Updates(map[string]interface{}{
		"avatar": req.File, "avatar_base64": "",
	})
	resp.OK(c, nil)
}

// 选项2：取QQ头像变为社区头像（复刻参考站 QQtx.aspx，拉 qlogo CDN 转 base64 存库）
var qqNumRe = regexp.MustCompile(`^\d{5,12}$`)

func (h *UserHandler) QqAvatar(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		QQ string `json:"qq" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请输入QQ号")
		return
	}
	qq := strings.TrimSpace(req.QQ)
	if !qqNumRe.MatchString(qq) {
		resp.ParamError(c, "QQ号格式不对（5-12位数字）")
		return
	}
	// 拉取QQ头像（100x100）
	client := &http.Client{Timeout: 8 * time.Second}
	httpResp, err := client.Get("https://q1.qlogo.cn/g?b=qq&nk=" + qq + "&s=100")
	if err != nil {
		resp.ParamError(c, "获取QQ头像失败，请稍后再试")
		return
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		resp.ParamError(c, "获取QQ头像失败，请检查QQ号")
		return
	}
	data, err := io.ReadAll(io.LimitReader(httpResp.Body, 600*1024))
	if err != nil || len(data) < 100 {
		resp.ParamError(c, "获取QQ头像失败")
		return
	}
	ct := httpResp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		ct = sniffImageType(data) // 部分 CDN 不返回 Content-Type，按魔数识别
	}
	uri := "data:" + ct + ";base64," + base64.StdEncoding.EncodeToString(data)
	h.DB.Model(&model.User{}).Where("id = ?", uid).Updates(map[string]interface{}{
		"avatar_base64": uri, "avatar": "",
	})
	resp.OK(c, nil)
}

// sniffImageType 按文件魔数识别图片类型
func sniffImageType(data []byte) string {
	switch {
	case len(data) > 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return "image/jpeg"
	case len(data) > 8 && string(data[0:4]) == "\x89PNG":
		return "image/png"
	case len(data) > 6 && string(data[0:6]) == "GIF89a" || (len(data) > 6 && string(data[0:6]) == "GIF87a"):
		return "image/gif"
	case len(data) > 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}
