package handler

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/authutil"
	"qqjiayuan/server/pkg/captcha"
	"qqjiayuan/server/pkg/loginlimiter"
	"qqjiayuan/server/pkg/resp"
)

type AuthHandler struct {
	DB     *gorm.DB
	Secret string
	ExpH   int
	// ★ 2026-10-12 二战账号库（`ezfy_account` 在二战库 qq_ezzt；单库模式下 == DB）。
	//   只用于「家园改/重置密码 → 同步二战密码」这条**临时**通道，见 syncEzfyPassword。
	EzfyDB *gorm.DB
}

// syncEzfyPassword ★ 2026-10-12 临时措施（用户：「二战征途还没脱离家园，家园里面重置密码的时候
// 给征途也同步下暂时」）：家园侧改密码/重置密码成功后，把新的 bcrypt hash 同步到二战账号表。
//
// 背景：二战现在是独立登录（`ezfy_account` 自己存一份密码 hash，首登时从家园拷过来），
// 但二战**还没真正脱离家园** —— 玩家在家园重置密码后，二战那边还是旧密码，
// 会出现「用家园新密码登不进二战、只能用旧密码」的错位。所以过渡期在家园侧同步一次。
//
// 规则：
//   - 只更新**已存在**的二战账号行（`game_id = 家园号`）；没玩过二战的玩家不会因此被建档；
//   - 二战库没配（单库模式下就是家园库）或表不存在 → 静默跳过，绝不影响家园改密码本身；
//   - 玩家在二战里自己改过密码的，家园再重置会把它覆盖回家园密码 —— 过渡期可接受。
//
// ⚠️ 等二战彻底拆开独立部署后，**应删掉本函数与全部调用点**（届时二战完全自助管密码）。
func syncEzfyPassword(ezfyDB *gorm.DB, uid uint, hash string) {
	if ezfyDB == nil || uid == 0 || hash == "" {
		return
	}
	if !ezfyDB.Migrator().HasTable("ezfy_account") {
		return
	}
	ezfyDB.Model(&model.EzfyAccount{}).Where("game_id = ?", uid).Update("password", hash)
}

type regReq struct {
	Nickname  string `json:"nickname" binding:"required,min=2,max=20"`
	Password  string `json:"password" binding:"required,min=6,max=20"`
	Gender    int    `json:"gender"`
	Invite    string `json:"invite"`     // 邀请码（邀请开通家园）
	CaptchaID string `json:"captcha_id"` // 图形验证码 ID
	Captcha   string `json:"captcha"`    // 图形验证码答案（算式计算结果）
}

// nickRule 昵称卡控（★ 2026-10-16，与前端 Register.vue 同口径）：
// 2-20 个字符；不能是纯数字（含夹空白）；不能含 emoji（主流表情区块 + 变体选择符 + ZWJ 表情序列）。
// 主流特殊字符（♥✿★ 等文本符号）不拦。合法返回空串，非法返回用户可读的提示。
func nickRule(s string) string {
	n := utf8.RuneCountInString(s)
	if n < 2 {
		return "昵称至少2个字符"
	}
	if n > 20 {
		return "昵称最多20个字符"
	}
	digitOnly := true
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		if r < '0' || r > '9' {
			digitOnly = false
			break
		}
	}
	if digitOnly && n > 0 {
		return "昵称不能是纯数字"
	}
	for _, r := range s {
		if (r >= 0x1F000 && r <= 0x1FAFF) || r == 0x200D || r == 0xFE0F || r == 0xFE0E {
			return "昵称不能包含表情符号"
		}
	}
	return ""
}

// 图形验证码：返回 {id, img}，img 为 data:image/png;base64 可直接给 <img src>
// （复刻参考站 /captcha：算式图片，点图片换一张）
func (h *AuthHandler) Captcha(c *gin.Context) {
	id, png, err := captcha.New()
	if err != nil {
		resp.ServerError(c, err)
		return
	}
	resp.OK(c, gin.H{"id": id, "img": "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)})
}

// 注册：昵称+性别+密码，系统自动分配家园号码（靓号）
func (h *AuthHandler) Register(c *gin.Context) {
	var req regReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "昵称或密码格式不对：昵称2-20字，密码6-20位")
		return
	}
	if req.Gender != 1 && req.Gender != 2 {
		req.Gender = 1
	}
	// ★ 2026-10-16 昵称卡控（与前端同口径）：去首尾空白后校验，防脚本直连绕过前端限制。
	req.Nickname = strings.TrimSpace(req.Nickname)
	if msg := nickRule(req.Nickname); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	// 先验验证码，避免脚本不填验证码就批量探测昵称
	if strings.TrimSpace(req.Captcha) == "" {
		resp.ParamError(c, "请输入验证码")
		return
	}
	if !captcha.Verify(req.CaptchaID, req.Captcha) {
		resp.ParamError(c, "验证码不对哦，请点击图片刷新后重试")
		return
	}
	ip := c.ClientIP()
	// 同一 IP 注册数量限制（系统配置 reg_ip_limit，默认 5，填 0 不限制）
	if limit := settingInt(h.DB, "reg_ip_limit", 5, 0, 1000); limit > 0 {
		var ipCount int64
		h.DB.Model(&model.User{}).Where("add_ip = ?", ip).Count(&ipCount)
		if ipCount >= int64(limit) {
			resp.ParamError(c, fmt.Sprintf("同一个IP最多只能注册%d个家园账号哦，如有疑问请联系客服", limit))
			return
		}
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
	regCoins := settingInt(h.DB, "reg_coins", 100, 0, 100000)
	user := model.User{
		Nickname: req.Nickname,
		Password: string(hash),
		Gender:   req.Gender,
		Coins:    regCoins,
		Level:    1,
		Color:    DefaultNickColor,      // 普通用户昵称默认蓝色（诺哈用户昵称展示）
		Config:   "10,1200,1500,1200,0", // 诺哈 wap_user.config 默认值
		AddIP:    ip,
		LastIP:   ip,
	}
	if err := h.DB.Create(&user).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	h.DB.Model(&user).Update("username", fmt.Sprintf("%d", user.ID))
	userLog(h.DB, user.ID, "注册成功", "家园号码 "+user.Username, ip)

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
	Name      string `json:"name" binding:"required"`
	Password  string `json:"password" binding:"required"`
	CaptchaID string `json:"captcha_id"` // 图形验证码 ID（密码输错满3次后必填）
	Captcha   string `json:"captcha"`    // 图形验证码答案（算式计算结果）
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请输入账号和密码")
		return
	}
	name := strings.TrimSpace(req.Name)
	// 同一账号在窗口期内密码输错满 3 次后，登录必须过图形验证码（先验验证码，避免被脚本刷密码）
	// ★★ 2026-10-13 修复「只弹提示、不显示验证码框」：
	//   这两个分支原来用 `resp.ParamError`（**不带 `need_captcha` 标记**），
	//   而前端 `Login.vue` 是**靠 `r.data.need_captcha` 才把验证码框亮出来**的。
	//   于是「输错 3 次 → 看到验证码 → 刷新/重进登录页（needCaptcha 被重置）→ 再登录」
	//   就只弹一句提示、**没有输入框也没有图片**，玩家直接卡死登不进去。
	//   统一改成 `resp.FailData(..., {"need_captcha": true})`，与下面的密码错误分支 / 二战登录同口径。
	if loginlimiter.NeedCaptcha(name) {
		if strings.TrimSpace(req.Captcha) == "" {
			resp.FailData(c, 400, "密码错误次数较多，请输入验证码", gin.H{"need_captcha": true})
			return
		}
		if !captcha.Verify(req.CaptchaID, req.Captcha) {
			resp.FailData(c, 400, "验证码不对哦，请点击图片刷新后重试", gin.H{"need_captcha": true})
			return
		}
	}
	var user model.User
	// ★ 2026-10-12 二战独立登录：家园号（=users.id，纯数字）也可直接登录。
	//   原来是 `username = name OR nickname = name`，纯数字家园号（区域号/数字UID）会查不到人。
	//   数字优先按 id 精确匹配（家园号即全局唯一主键），同时兼容 username/昵称。
	q := h.DB.Preload("Roles")
	if idv, err := strconv.ParseUint(name, 10, 64); err == nil {
		q = q.Where("id = ? OR username = ? OR nickname = ?", idv, name, name)
	} else {
		q = q.Where("username = ? OR nickname = ?", name, name)
	}
	if err := q.First(&user).Error; err != nil {
		resp.ParamError(c, "账号不存在，先免费注册一个吧")
		return
	}
	if user.Status == 0 {
		userLog(h.DB, user.ID, "登陆失败", "账号已被封禁", c.ClientIP())
		resp.Forbidden(c, "该账号已被封禁，如有疑问请联系客服")
		return
	}
	// ★ 2026-10-17 人机账号禁止登录（防止别人登录人机账号）：
	//   人机只由内部 botCall 驱动（JWT 直调业务接口，不走登录），真人登录一律拒绝。
	if user.IsBot == 1 {
		userLog(h.DB, user.ID, "登陆失败", "人机账号禁止登录", c.ClientIP())
		resp.Forbidden(c, "该账号不可登录")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		userLog(h.DB, user.ID, "登陆失败", "密码错误", c.ClientIP())
		loginlimiter.Fail(name)
		// 输错满 3 次：响应带 need_captcha 标记，前端立即弹出验证码框，下次起必须过验证码
		if loginlimiter.NeedCaptcha(name) {
			resp.FailData(c, 400, "密码错误次数较多，请先通过验证码再试", gin.H{"need_captcha": true})
			return
		}
		resp.ParamError(c, "密码不对哦，再想想")
		return
	}
	loginlimiter.Success(name)

	// ★★ 2026-10-18 单设备在线（用户：「一个账户只能在线一个设备」，多地登录=异常缓存，一并踢下线）：
	//   登录成功即把「最近签发时刻」顶到当前 → 该账号此前签发的**所有** token 立即作废，
	//   后登录的顶掉先登录的（其它设备 / 异常缓存里的旧 token 全部 401）。
	//   与「退出」(Logout) 同口径：家园 users.session_at + 二战 ezfy_account.session_at 一起顶。
	//   ★ session_at 必须在**签发 token 之前**写（iat 取自签发时刻，二者同秒才恒通过；
	//     若先签 token 再写 session_at，跨秒的瞬间刚签发的 token 就会立刻失效 = 登录即掉线）。
	nowSec := time.Now().Unix() * 1000
	h.DB.Model(&model.User{}).Where("id = ?", user.ID).Update("session_at", nowSec)
	middleware.SetHomeSessionAt(user.ID, nowSec)
	if h.EzfyDB != nil && h.EzfyDB.Migrator().HasTable("ezfy_account") {
		h.EzfyDB.Model(&model.EzfyAccount{}).Where("game_id = ?", user.ID).Update("session_at", nowSec)
		middleware.SetEzfySessionAt(user.ID, nowSec)
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

// Logout POST /api/auth/logout —— ★★ 2026-10-15 家园退出 → **家园 token 和二战 token 一起服务端作废**。
//
// 家园/二战 token 都是无状态 JWT，本地清了不算数（脚本拿着旧 token 还能打接口）。
// 作废手段 = 把两边「最近签发时刻」顶到当前（秒级 ×1000，与 JWT 的 iat 精度对齐）：
//
//	家园侧  users.session_at            → 中间件按 token.iat < session_at 判 401
//	二战侧  ezfy_account.session_at     → 二战中间件同样判 401
//
// 只写这两个时间戳，不做其它状态；玩家重新登录会签发新 token（iat >= session_at）→ 自然恢复。
// 多机部署时写库后各自刷新**本机**缓存即时生效，其它节点最迟 60s（与「踢下线」同策略）。
// 只在退出时写、登录不写 —— 家园多设备同时登录不受影响。
func (h *AuthHandler) Logout(c *gin.Context) {
	uid := middleware.GetUID(c)
	nowSec := time.Now().Unix() * 1000

	// 家园 token 作废
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("session_at", nowSec)
	middleware.SetHomeSessionAt(uid, nowSec)

	// 二战 token 一并作废（家园退出 → 二战也掉线）；没玩过二战的玩家（无 ezfy_account 行）静默跳过。
	// EzfyDB 是二战库（单库模式下就是家园库），表不存在时同样跳过，绝不影响家园退出本身。
	if h.EzfyDB != nil && h.EzfyDB.Migrator().HasTable("ezfy_account") {
		h.EzfyDB.Model(&model.EzfyAccount{}).Where("game_id = ?", uid).Update("session_at", nowSec)
		middleware.SetEzfySessionAt(uid, nowSec)
	}

	userLog(h.DB, uid, "登出", "家园退出", c.ClientIP())
	resp.OK(c, gin.H{"msg": "已退出登录"})
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
		"nickname": maskNickname(user.Nickname),
		"has_qq":   ct.QQ != "", "qq": maskQQ(ct.QQ),
		"has_mail": ct.Mail != "", "mail": maskMail(ct.Mail),
		"has_phone": ct.Phone != "", "phone": maskPhone(ct.Phone),
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
	syncEzfyPassword(h.EzfyDB, user.ID, string(hash)) // ★ 临时：同步二战密码（见 syncEzfyPassword）
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
	syncEzfyPassword(h.EzfyDB, user.ID, string(hash)) // ★ 临时：同步二战密码（见 syncEzfyPassword）
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
	syncEzfyPassword(h.EzfyDB, user.ID, string(hash)) // ★ 临时：同步二战密码（见 syncEzfyPassword）
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

// settingInt 读系统配置（settings 表）里的整数值；没配置或值非法时返回 def
func settingInt(db *gorm.DB, key string, def, min, max int) int {
	var s string
	db.Raw("SELECT value FROM settings WHERE `key` = ?", key).Scan(&s)
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < min || n > max {
		return def
	}
	return n
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
		"has_paypass":  user.PayPass != "",
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
	syncEzfyPassword(h.EzfyDB, user.ID, string(hash)) // ★ 临时：同步二战密码（见 syncEzfyPassword）
	resp.OK(c, nil)
}

func (h *AuthHandler) userBrief(u model.User) gin.H {
	return gin.H{"id": u.ID, "username": u.Username, "nickname": u.Nickname,
		"gender": u.Gender, "color": u.Color, "level": u.Level, "coins": u.Coins,
		"noble": u.Noble, "qq_end": u.QqEnd}
}
