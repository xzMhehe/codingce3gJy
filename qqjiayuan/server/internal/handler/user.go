package handler

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
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
	"golang.org/x/crypto/bcrypt"
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
	// 通信地址（诺哈 wap_user_address：故乡/现居，公开展示省市）
	var addr model.UserAddress
	h.DB.Where("user_id = ?", user.ID).First(&addr)
	isMe := middleware.GetUID(c) == user.ID

	// 业务图标（诺哈 profile.asp）：靓号/身份证/QQ/手机/邮箱
	var docuCount int64
	h.DB.Model(&model.UserDocument{}).Where("user_id = ?", user.ID).Count(&docuCount)
	var ct model.UserContact
	h.DB.Where("user_id = ?", user.ID).First(&ct)

	// 贵族身份（复刻诺哈 my_vip.asp：图标/等级/成长值/成长速度/开通时间/到期时间）
	var levels []model.NobleLevel
	h.DB.Order("id ASC").Find(&levels)
	now := time.Now()
	blueLv := model.NobleLvOf(levels, user.BlueExp)
	qqLv := model.NobleLvOf(levels, user.QqExp)
	blueSpeed := user.BlueSpeed
	if blueSpeed <= 0 {
		blueSpeed = logSpeedOf(h.DB, user.ID, "blue")
	}
	qqSpeed := user.QqSpeed
	if qqSpeed <= 0 {
		qqSpeed = logSpeedOf(h.DB, user.ID, "qq")
	}
	nobleInfo := gin.H{
		"blue": gin.H{
			"lv": blueLv, "exp": user.BlueExp, "speed": blueSpeed,
			"active":  user.BlueEnd != nil && user.BlueEnd.After(now),
			"icon":    iconOrEmpty(levels, blueLv, "blue"),
			"start":   user.BlueStart, "end": user.BlueEnd,
			"days_left": nobleDaysLeft(user.BlueEnd),
		},
		"qq": gin.H{
			"lv": qqLv, "exp": user.QqExp, "speed": qqSpeed,
			"active":  user.QqEnd != nil && user.QqEnd.After(now),
			"icon":    iconOrEmpty(levels, qqLv, "qq"),
			"start":   user.QqStart, "end": user.QqEnd,
			"days_left": nobleDaysLeft(user.QqEnd),
		},
	}

	resp.OK(c, gin.H{
		"id": user.ID, "username": user.Username, "nickname": user.Nickname,
		"gender": user.Gender, "signature": user.Signature, "color": user.Color,
		"avatar": user.Avatar, "avatar_base64": user.AvatarBase64, "level": user.Level, "exp": user.Exp, "coins": user.Coins,
		"level_icon": user.LevelIcon, "level_title": user.LevelTitle,
		"noble": user.Noble, "qq_lv": user.QqLv, "blue_lv": user.BlueLv,
		"noble_info": nobleInfo,
		"partner_id": user.PartnerID, "partner_name": partnerName,
		"baby_name": user.BabyName, "achieve": user.Achieve, "achieve_level": user.AchieveLevel,
		"priv_id": user.PrivID, "priv": user.Priv,
		"online":     online,
		"active_days": user.ActiveDays, "home_level": hl, "home_next_days": nextDays,
		"family": familyName, "mood": mood, "city": user.City,
		// 诺哈 wap_user 字段
		"age": user.Age, "birth_year": user.BirthYear, "birth_month": user.BirthMonth, "birth_day": user.BirthDay,
		"birth_type": user.BirthType, "solar": user.Solar, "lunar": user.Lunar,
		"hours": user.Hours, "friend_policy": user.FriendPolicy,
		"introduction": user.Introduction,
		// 通信地址（省市对所有人可见，详细地址仅本人）
		"home_prov": addr.HomeProv, "home_city": addr.HomeCity,
		"live_prov": addr.LiveProv, "live_city": addr.LiveCity,
		"address": addressOut(&addr, isMe),
		// 联系方式仅本人可见（诺哈 contact 仅本人/管理员可见）
		"contact": contactOut(h.DB, user.ID, isMe),
		// 业务图标（公开标志，具体内容仅本人可见）
		"paid": user.Paid, "has_docu": docuCount > 0,
		"has_qq": ct.QQ != "", "has_phone": ct.Phone != "", "has_mail": ct.Mail != "",
		"created_at": user.CreatedAt, "last_login_at": user.LastLoginAt, "last_active_at": user.LastActiveAt,
		"thread_count": threadCount, "reply_count": replyCount, "sign_days": signDays,
		"badges": badges, "roles": roles, "duties": duties, "threads": threads,
	})
}

// logSpeedOf 按最近一次开通方案日志取成长速度（点/天），无则默认 10
func logSpeedOf(db *gorm.DB, uid uint, typ string) int {
	var log model.WalletLog
	if err := db.Where("user_id = ? AND kind = ?", uid, typ+"_open").Order("id DESC").First(&log).Error; err == nil {
		if speed, e := strconv.Atoi(log.Remark); e == nil && speed > 0 {
			return speed
		}
	}
	return 10
}

// iconOrEmpty 取贵宾图标：0 级返回空（未开通不显示图标）
func iconOrEmpty(levels []model.NobleLevel, lv int, typ string) string {
	if lv <= 0 {
		return ""
	}
	return model.NobleIconOf(levels, lv, typ)
}

// nobleDaysLeft 剩余天数（已过期返回 0）
func nobleDaysLeft(end *time.Time) int {
	if end == nil {
		return 0
	}
	d := int(end.Sub(time.Now()).Hours()/24) + 1
	if d < 0 {
		return 0
	}
	return d
}

// addressOut 通信地址输出：详细地址/邮编仅本人可见
func addressOut(a *model.UserAddress, isMe bool) gin.H {
	out := gin.H{
		"home_nation": a.HomeNation, "home_prov": a.HomeProv, "home_city": a.HomeCity,
		"live_nation": a.LiveNation, "live_prov": a.LiveProv, "live_city": a.LiveCity,
	}
	if isMe {
		out["home_dist"] = a.HomeDist
		out["home_addr"] = a.HomeAddr
		out["home_zip"] = a.HomeZip
		out["live_dist"] = a.LiveDist
		out["live_addr"] = a.LiveAddr
		out["live_zip"] = a.LiveZip
	}
	return out
}

// contactOut 联系方式（诺哈 wap_user_contact）：仅本人可见
func contactOut(db *gorm.DB, uid uint, isMe bool) gin.H {
	if !isMe {
		return nil
	}
	var ct model.UserContact
	db.Where("user_id = ?", uid).First(&ct)
	return gin.H{"qq": ct.QQ, "mail": ct.Mail, "phone": ct.Phone}
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
	BirthType    *int   `json:"birth_type"`  // 0阴历 1阳历（诺哈）
	Solar        string `json:"solar" binding:"max=20"`
	Lunar        string `json:"lunar" binding:"max=20"`
	FriendPolicy *int   `json:"friend_policy"` // 0允许 1验证 2拒绝（诺哈）
	PerPage      *int   `json:"per_page"`      // 每页帖子数 5-20（诺哈 config[0]）
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
	updates := map[string]interface{}{
		"nickname": req.Nickname, "signature": req.Signature,
		"gender": req.Gender, "avatar": req.Avatar, "city": req.City,
		"age": age, "birth_year": by, "birth_month": bm, "birth_day": bd,
		"introduction": req.Introduction,
		"solar":        req.Solar, "lunar": req.Lunar,
	}
	// 诺哈 wap_user 扩展字段
	if req.BirthType != nil {
		if *req.BirthType != 0 {
			*req.BirthType = 1
		}
		updates["birth_type"] = *req.BirthType
	}
	if req.FriendPolicy != nil {
		if *req.FriendPolicy < 0 || *req.FriendPolicy > 2 {
			*req.FriendPolicy = 0
		}
		updates["friend_policy"] = *req.FriendPolicy
	}
	if req.PerPage != nil {
		// 个性设置 config：只改第 0 段（每页帖子数 5-20），其余段保留（诺哈 config CSV）
		pp := *req.PerPage
		if pp < 5 {
			pp = 5
		}
		if pp > 20 {
			pp = 20
		}
		cfg := user.Config
		if cfg == "" {
			cfg = "10,1200,1500,1200,0"
		}
		parts := strings.Split(cfg, ",")
		if len(parts) < 5 {
			parts = []string{"10", "1200", "1500", "1200", "0"}
		}
		parts[0] = strconv.Itoa(pp)
		updates["config"] = strings.Join(parts, ",")
	}
	h.DB.Model(&user).Updates(updates)
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

// ============ 用户附属信息（对齐诺哈 contact/address/docu/protec/pass 家族） ============

// userLog 记录用户操作日志（诺哈 wap_user_log）
func userLog(db *gorm.DB, uid uint, action, intro, ip string) {
	db.Create(&model.UserLog{UserID: uid, Action: action, Intro: intro, IP: ip})
}

// ---- 通信地址（诺哈 address.asp：故乡/现居） ----

// AddressView 我的通信地址
func (h *UserHandler) AddressView(c *gin.Context) {
	uid := middleware.GetUID(c)
	var a model.UserAddress
	h.DB.Where("user_id = ?", uid).First(&a)
	resp.OK(c, a)
}

// AddressSave 保存通信地址
func (h *UserHandler) AddressSave(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req model.UserAddress
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "地址格式不对")
		return
	}
	var a model.UserAddress
	if err := h.DB.Where("user_id = ?", uid).First(&a).Error; err != nil {
		a = model.UserAddress{UserID: uid}
		h.DB.Create(&a)
	}
	h.DB.Model(&a).Updates(map[string]interface{}{
		"home_nation": req.HomeNation, "home_prov": req.HomeProv, "home_city": req.HomeCity,
		"home_dist": req.HomeDist, "home_addr": req.HomeAddr, "home_zip": req.HomeZip,
		"live_nation": req.LiveNation, "live_prov": req.LiveProv, "live_city": req.LiveCity,
		"live_dist": req.LiveDist, "live_addr": req.LiveAddr, "live_zip": req.LiveZip,
	})
	resp.OK(c, nil)
}

// ---- 实名证件（诺哈 docu：设置需登录密码确认） ----

// DocumentView 我的实名证件（号码脱敏）
func (h *UserHandler) DocumentView(c *gin.Context) {
	uid := middleware.GetUID(c)
	var d model.UserDocument
	h.DB.Where("user_id = ?", uid).First(&d)
	num := d.Number
	if len(num) > 6 {
		num = num[:3] + "***********" + num[len(num)-3:]
	}
	resp.OK(c, gin.H{"type": d.Type, "real_name": d.RealName, "number": num, "has_doc": d.ID > 0})
}

// DocumentSave 保存实名证件（需登录密码确认，对齐诺哈）
func (h *UserHandler) DocumentSave(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Type     int    `json:"type"`
		RealName string `json:"real_name" binding:"required,max=30"`
		Number   string `json:"number" binding:"required,max=30"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写证件类型、姓名和号码")
		return
	}
	var u model.User
	h.DB.First(&u, uid)
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)) != nil {
		resp.ParamError(c, "登录密码不对，无法设置证件")
		return
	}
	if req.Type != 1 {
		req.Type = 1
	}
	var d model.UserDocument
	if err := h.DB.Where("user_id = ?", uid).First(&d).Error; err != nil {
		d = model.UserDocument{UserID: uid}
		h.DB.Create(&d)
	}
	h.DB.Model(&d).Updates(map[string]interface{}{"type": req.Type, "real_name": req.RealName, "number": req.Number})
	userLog(h.DB, uid, "设置证件", "更新了实名证件信息", c.ClientIP())
	resp.OK(c, nil)
}

// ---- 密保问题（诺哈 protec：issue 编号 + MD5 答案） ----

// ProtectionView 密保状态与问题库
func (h *UserHandler) ProtectionView(c *gin.Context) {
	uid := middleware.GetUID(c)
	var p model.UserProtection
	h.DB.Where("user_id = ?", uid).First(&p)
	resp.OK(c, gin.H{
		"questions": model.ProtectionQuestions,
		"issue":     p.Issue, "has_protection": p.ID > 0,
	})
}

// ProtectionSave 设置密保问题
func (h *UserHandler) ProtectionSave(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Issue  int    `json:"issue" binding:"min=1"`
		Answer string `json:"answer" binding:"required,max=30"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请选择密保问题并填写答案")
		return
	}
	if req.Issue > len(model.ProtectionQuestions) {
		resp.ParamError(c, "密保问题不存在")
		return
	}
	sum := md5.Sum([]byte(strings.TrimSpace(req.Answer)))
	var p model.UserProtection
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err != nil {
		p = model.UserProtection{UserID: uid}
		h.DB.Create(&p)
	}
	h.DB.Model(&p).Updates(map[string]interface{}{"issue": req.Issue, "answer": hex.EncodeToString(sum[:])})
	userLog(h.DB, uid, "设置密保", "更新了密保问题", c.ClientIP())
	resp.OK(c, nil)
}

// ---- 支付密码（诺哈 wap_user_money.pass，独立于登录密码） ----

// PayPassView 是否已设置支付密码
func (h *UserHandler) PayPassView(c *gin.Context) {
	uid := middleware.GetUID(c)
	var u model.User
	h.DB.Select("pay_pass").First(&u, uid)
	resp.OK(c, gin.H{"has_paypass": u.PayPass != ""})
}

// PayPassSet 设置/修改支付密码（首次直接设；修改需原支付密码）
func (h *UserHandler) PayPassSet(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		Old string `json:"old"`
		New string `json:"new" binding:"required,min=6,max=20"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "支付密码需要 6-20 位")
		return
	}
	var u model.User
	h.DB.Select("id,pay_pass").First(&u, uid)
	if u.PayPass != "" {
		if bcrypt.CompareHashAndPassword([]byte(u.PayPass), []byte(req.Old)) != nil {
			resp.ParamError(c, "原支付密码不对")
			return
		}
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.New), bcrypt.DefaultCost)
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("pay_pass", string(hash))
	userLog(h.DB, uid, "支付密码", "修改了支付密码", c.ClientIP())
	resp.OK(c, nil)
}

// verifyPayPass 校验支付密码（购买/赠送等消费确认，复刻诺哈 VerifyPayPass），返回错误文案（空串=通过）
func verifyPayPass(db *gorm.DB, uid uint, payPass string) string {
	var u model.User
	db.Select("id,pay_pass").First(&u, uid)
	if u.PayPass == "" {
		return "您还未设置支付密码，请先到安全中心设置！"
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PayPass), []byte(payPass)) != nil {
		return "支付密码错误！"
	}
	return ""
}

// ---- 登录/操作日志（诺哈 log.asp） ----

// MyLogs 我的操作日志
func (h *UserHandler) MyLogs(c *gin.Context) {
	uid := middleware.GetUID(c)
	page, offset, size := pageOf(c, 20)
	q := h.DB.Model(&model.UserLog{}).Where("user_id = ?", uid)
	var total int64
	q.Count(&total)
	var list []model.UserLog
	q.Order("created_at DESC").Offset(offset).Limit(size).Find(&list)
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": list})
}
