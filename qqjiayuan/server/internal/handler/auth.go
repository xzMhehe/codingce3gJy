package handler

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/authutil"
	"qqjiayuan/server/pkg/resp"
)

type AuthHandler struct {
	DB     *gorm.DB
	Secret string
	ExpH   int
}

type regReq struct {
	Nickname string `json:"nickname" binding:"required,min=1,max=20"`
	Password string `json:"password" binding:"required,min=6,max=20"`
	Gender   int    `json:"gender"`
	Invite   string `json:"invite"` // 邀请码（邀请开通家园）
}

// 注册：昵称+性别+密码，系统自动分配家园号码（靓号）
func (h *AuthHandler) Register(c *gin.Context) {
	var req regReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "昵称或密码格式不对：昵称1-20字，密码6-20位")
		return
	}
	if req.Gender != 1 && req.Gender != 2 {
		req.Gender = 1
	}
	var exists int64
	h.DB.Model(&model.User{}).Where("nickname = ?", req.Nickname).Count(&exists)
	if exists > 0 {
		resp.ParamError(c, "这个昵称已经被注册了，换一个吧")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		resp.ServerError(c, err)
		return
	}
	// 注册配置：新人礼包金币（系统配置 reg_coins，默认 100）
	regCoins := 100
	var regCoinsStr string
	h.DB.Raw("SELECT value FROM settings WHERE `key` = 'reg_coins'").Scan(&regCoinsStr)
	if n, err := strconv.Atoi(strings.TrimSpace(regCoinsStr)); err == nil && n >= 0 && n <= 100000 {
		regCoins = n
	}
	user := model.User{
		Nickname: req.Nickname,
		Password: string(hash),
		Gender:   req.Gender,
		Coins:    regCoins,
		Level:    1,
		Config:   "10,1200,1500,1200,0", // 诺哈 wap_user.config 默认值
		AddIP:    c.ClientIP(),
		LastIP:   c.ClientIP(),
	}
	if err := h.DB.Create(&user).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	h.DB.Model(&user).Update("username", fmt.Sprintf("%d", user.ID))
	userLog(h.DB, user.ID, "注册成功", "家园号码 "+user.Username, c.ClientIP())

	var member model.Role
	h.DB.Where("code = ?", "member").First(&member)
	h.DB.Model(&user).Association("Roles").Append(&member)

	// 邀请开通家园：记录邀请关系 + 双方奖励（诺哈 promo 推荐奖励）
	if code := strings.TrimSpace(req.Invite); code != "" {
		applyInvite(h.DB, &user, code)
	}

	h.DB.Create(&model.Notification{
		UserID: user.ID, Type: "system",
		Title:   "欢迎来到家园社区",
		Content: fmt.Sprintf("你的家园号码是 %d，请牢记！新人礼包100金币已到账。多逛论坛多回帖，经验等级蹭蹭涨。", user.ID),
	})
	token, _ := authutil.GenerateToken(user.ID, user.Nickname, h.Secret, h.ExpH)
	resp.OK(c, gin.H{"token": token, "user_id": user.ID, "username": user.Username, "nickname": user.Nickname})
}

type loginReq struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请输入账号和密码")
		return
	}
	var user model.User
	if err := h.DB.Preload("Roles").Where("username = ? OR nickname = ?", req.Name, req.Name).First(&user).Error; err != nil {
		resp.ParamError(c, "账号不存在，先免费注册一个吧")
		return
	}
	if user.Status == 0 {
		userLog(h.DB, user.ID, "登陆失败", "账号已被封禁", c.ClientIP())
		resp.Forbidden(c, "该账号已被封禁，如有疑问请联系客服")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		userLog(h.DB, user.ID, "登陆失败", "密码错误", c.ClientIP())
		resp.ParamError(c, "密码不对哦，再想想")
		return
	}
	token, err := authutil.GenerateToken(user.ID, user.Nickname, h.Secret, h.ExpH)
	if err != nil {
		resp.ServerError(c, err)
		return
	}
	now := gorm.Expr("NOW()")
	h.DB.Model(&user).Updates(map[string]interface{}{"last_login_at": now, "last_active_at": now, "last_ip": c.ClientIP()})
	userLog(h.DB, user.ID, "登陆成功", "欢迎回来", c.ClientIP())
	resp.OK(c, gin.H{"token": token, "user": h.userBrief(user)})
}

// 找回资料：凭昵称模糊找回家园号码（演示站的情怀功能）
func (h *AuthHandler) FindAccount(c *gin.Context) {
	word := strings.TrimSpace(c.Query("nickname"))
	if word == "" {
		resp.ParamError(c, "请输入你的昵称")
		return
	}
	var users []model.User
	h.DB.Where("nickname LIKE ?", "%"+word+"%").Limit(10).Find(&users)
	out := make([]gin.H, 0)
	for _, u := range users {
		out = append(out, gin.H{"id": u.ID, "username": u.Username, "nickname": maskNickname(u.Nickname), "created_at": u.CreatedAt.Format("2006-01-02")})
	}
	resp.OK(c, out)
}

// 找回号码：凭联系方式（QQ/邮箱/手机，任填其一）查回家园号码（诺哈 login/user.asp）
func (h *AuthHandler) FindByContact(c *gin.Context) {
	qq := strings.TrimSpace(c.Query("qq"))
	mail := strings.TrimSpace(c.Query("mail"))
	phone := strings.TrimSpace(c.Query("phone"))
	if qq == "" && mail == "" && phone == "" {
		resp.ParamError(c, "请填写QQ、邮箱或手机号之一")
		return
	}
	var q *gorm.DB
	first := true
	add := func(sql string, v interface{}) {
		if first {
			q = h.DB.Where(sql, v)
			first = false
		} else {
			q = q.Or(sql, v)
		}
	}
	if qq != "" {
		add("qq = ?", qq)
	}
	if mail != "" {
		add("mail = ?", mail)
	}
	if phone != "" {
		add("phone = ?", phone)
	}
	var contacts []model.UserContact
	q.Limit(10).Find(&contacts)
	out := make([]gin.H, 0)
	for _, ct := range contacts {
		var u model.User
		if err := h.DB.First(&u, ct.UserID).Error; err != nil {
			continue
		}
		out = append(out, gin.H{"id": u.ID, "username": u.Username, "nickname": maskNickname(u.Nickname), "created_at": u.CreatedAt.Format("2006-01-02")})
	}
	resp.OK(c, out)
}

// 找回号码：凭真实姓名+证件号码查回家园号码（诺哈 login/user.asp act=4 查 wap_user_docu）
func (h *AuthHandler) FindByDocument(c *gin.Context) {
	name := strings.TrimSpace(c.Query("name"))
	number := strings.TrimSpace(c.Query("number"))
	if name == "" || number == "" {
		resp.ParamError(c, "请填写真实姓名和证件号码")
		return
	}
	var docs []model.UserDocument
	h.DB.Where("real_name = ? AND number = ?", name, number).Limit(10).Find(&docs)
	out := make([]gin.H, 0)
	for _, d := range docs {
		var u model.User
		if err := h.DB.First(&u, d.UserID).Error; err != nil {
			continue
		}
		out = append(out, gin.H{"id": u.ID, "username": u.Username, "nickname": maskNickname(u.Nickname), "created_at": u.CreatedAt.Format("2006-01-02")})
	}
	resp.OK(c, out)
}

// 找回密码第一步：按会员号码查询可用的找回渠道（脱敏展示，诺哈 login/pass.asp act=1~5）
func (h *AuthHandler) RepassChannels(c *gin.Context) {
	username := strings.TrimSpace(c.Query("username"))
	if username == "" {
		resp.ParamError(c, "请输入会员号码")
		return
	}
	var user model.User
	if err := h.DB.Where("username = ?", username).First(&user).Error; err != nil {
		resp.NotFound(c, "会员号码不存在")
		return
	}
	var ct model.UserContact
	h.DB.Where("user_id = ?", user.ID).First(&ct)
	var p model.UserProtection
	h.DB.Where("user_id = ?", user.ID).First(&p)
	issue := ""
	if p.ID > 0 && p.Issue >= 1 && p.Issue <= len(model.ProtectionQuestions) {
		issue = model.ProtectionQuestions[p.Issue-1]
	}
	resp.OK(c, gin.H{
		"nickname":       maskNickname(user.Nickname),
		"has_qq":         ct.QQ != "", "qq": maskQQ(ct.QQ),
		"has_mail":       ct.Mail != "", "mail": maskMail(ct.Mail),
		"has_phone":      ct.Phone != "", "phone": maskPhone(ct.Phone),
		"has_protection": p.ID > 0, "issue": issue,
	})
}

type repassQQReq struct {
	Username string `json:"username" binding:"required"`
	QQ       string `json:"qq" binding:"required,min=5,max=16"`
}

// 找回密码：通过QQ（诺哈 pass.asp act=6：QQ验证通过后把密码重置为QQ号）
func (h *AuthHandler) RepassByQQ(c *gin.Context) {
	var req repassQQReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请输入会员号码和QQ号")
		return
	}
	var user model.User
	if err := h.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		resp.NotFound(c, "会员号码不存在")
		return
	}
	var ct model.UserContact
	h.DB.Where("user_id = ?", user.ID).First(&ct)
	if ct.QQ == "" {
		resp.ParamError(c, "该号码没有绑定QQ")
		return
	}
	if ct.QQ != req.QQ {
		resp.ParamError(c, "QQ号码不正确")
		return
	}
	if len(req.QQ) < 6 {
		resp.ParamError(c, "QQ号不足6位，无法作为新密码，请改用其他方式找回")
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.QQ), bcrypt.DefaultCost)
	h.DB.Model(&user).Update("password", string(hash))
	userLog(h.DB, user.ID, "重置密码", "通过QQ找回", c.ClientIP())
	resp.OK(c, gin.H{"username": user.Username, "password": req.QQ})
}

type repassMailReq struct {
	Username string `json:"username" binding:"required"`
	Mail     string `json:"mail" binding:"required,max=50"`
	NewPwd   string `json:"new_password" binding:"required,min=6,max=20"`
}

// 找回密码：通过邮箱（诺哈 pass.asp act=7 发邮件链接；本项目无邮件服务，校验通过后直接设新密码）
func (h *AuthHandler) RepassByMail(c *gin.Context) {
	var req repassMailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请输入会员号码、邮箱和新密码（6-20位）")
		return
	}
	var user model.User
	if err := h.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		resp.NotFound(c, "会员号码不存在")
		return
	}
	var ct model.UserContact
	h.DB.Where("user_id = ?", user.ID).First(&ct)
	if ct.Mail == "" {
		resp.ParamError(c, "该号码没有绑定邮箱")
		return
	}
	if ct.Mail != req.Mail {
		resp.ParamError(c, "邮箱不正确")
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPwd), bcrypt.DefaultCost)
	h.DB.Model(&user).Update("password", string(hash))
	userLog(h.DB, user.ID, "重置密码", "通过邮箱找回", c.ClientIP())
	resp.OK(c, gin.H{"username": user.Username})
}

type repassProtReq struct {
	Username string `json:"username" binding:"required"`
	Answer   string `json:"answer" binding:"required,max=30"`
}

// 找回密码：通过密保问题（诺哈 pass.asp act=8：答案全部比对通过后重置为随机8位密码）
func (h *AuthHandler) RepassByProtection(c *gin.Context) {
	var req repassProtReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请输入会员号码和密保答案")
		return
	}
	var user model.User
	if err := h.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		resp.NotFound(c, "会员号码不存在")
		return
	}
	var p model.UserProtection
	h.DB.Where("user_id = ?", user.ID).First(&p)
	if p.ID == 0 {
		resp.ParamError(c, "该号码没有设置密码保护")
		return
	}
	sum := md5.Sum([]byte(strings.TrimSpace(req.Answer)))
	if p.Answer != hex.EncodeToString(sum[:]) {
		resp.ParamError(c, "密保答案错误！")
		return
	}
	pass := randomPassword(8)
	hash, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	h.DB.Model(&user).Update("password", string(hash))
	userLog(h.DB, user.ID, "重置密码", "通过密保找回", c.ClientIP())
	resp.OK(c, gin.H{"username": user.Username, "password": pass})
}

func maskNickname(s string) string {
	n := []rune(s)
	if len(n) == 0 {
		return "*"
	}
	masked := string(n[0])
	if len(n) > 2 {
		masked += strings.Repeat("*", len(n)-2)
	}
	if len(n) > 1 {
		masked += string(n[len(n)-1])
	}
	return masked
}

// 脱敏规则对齐诺哈 pass.asp：QQ 首尾2位、邮箱名首2位、手机首3尾2
func maskQQ(s string) string {
	r := []rune(s)
	if len(r) <= 4 {
		return "**"
	}
	return string(r[:2]) + "*****" + string(r[len(r)-2:])
}

func maskMail(s string) string {
	parts := strings.SplitN(s, "@", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "***"
	}
	local := parts[0]
	if len(local) > 2 {
		local = local[:2]
	}
	return local + "***@" + parts[1]
}

func maskPhone(s string) string {
	r := []rune(s)
	if len(r) < 5 {
		return "******"
	}
	return string(r[:3]) + "******" + string(r[len(r)-2:])
}

func randomPassword(n int) string {
	const chars = "abcdefghjkmnpqrstuvwxyz23456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)[len(strconv.FormatInt(time.Now().UnixNano(), 10))-n:]
	}
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}

func (h *AuthHandler) Me(c *gin.Context) {
	uid := middleware.GetUID(c)
	var user model.User
	if err := h.DB.Preload("Roles").Preload("Badges").First(&user, uid).Error; err != nil {
		resp.NotFound(c, "用户不存在")
		return
	}
	resp.OK(c, gin.H{
		"id": user.ID, "username": user.Username, "nickname": user.Nickname,
		"gender": user.Gender, "signature": user.Signature, "color": user.Color,
		"avatar": user.Avatar, "coins": user.Coins, "exp": user.Exp, "level": user.Level,
		"yuanbao": user.YuanBao, "jinzuan": user.JinZuan, "youquan": user.YouQuan,
		"age": user.Age, "birth_year": user.BirthYear, "birth_month": user.BirthMonth, "birth_day": user.BirthDay,
		"birth_type": user.BirthType, "solar": user.Solar, "lunar": user.Lunar,
		"friend_policy": user.FriendPolicy, "config": user.Config, "hours": user.Hours,
		"has_paypass": user.PayPass != "",
		"introduction": user.Introduction, "city": user.City, "avatar_base64": user.AvatarBase64,
		"level_icon": user.LevelIcon, "level_title": user.LevelTitle,
		"noble": user.Noble, "qq_end": user.QqEnd, "qq_lv": user.QqLv,
		"roles": user.Roles, "badges": user.Badges, "priv": user.Priv,
		"perms": middleware.UserPermissionCodes(h.DB, uid),
	})
}

type pwdReq struct {
	Old string `json:"old" binding:"required"`
	New string `json:"new" binding:"required,min=6,max=20"`
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req pwdReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "新密码需要6-20位")
		return
	}
	var user model.User
	h.DB.First(&user, uid)
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Old)) != nil {
		resp.ParamError(c, "原密码不对哦")
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.New), bcrypt.DefaultCost)
	h.DB.Model(&user).Update("password", string(hash))
	resp.OK(c, nil)
}

func (h *AuthHandler) userBrief(u model.User) gin.H {
	return gin.H{"id": u.ID, "username": u.Username, "nickname": u.Nickname,
		"gender": u.Gender, "color": u.Color, "level": u.Level, "coins": u.Coins,
		"noble": u.Noble, "qq_end": u.QqEnd}
}
