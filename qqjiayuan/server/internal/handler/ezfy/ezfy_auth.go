package ezfy

// 二战征途独立登录（★ 2026-10-12 二战与家园账号体系分离）。
//
//	二战登录不再走家园 /api/auth/login，玩家用「家园号 + 家园密码」首登，
//	校验通过后把家园密码 hash 拷进二战自有账号表 ezfy_account 并签发二战专属 token；
//	之后二战自助管密码、不再依赖家园账号（为二战/家园拆开部署做准备）。

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/authutil"
	"qqjiayuan/server/pkg/captcha"
	"qqjiayuan/server/pkg/loginlimiter"
	"qqjiayuan/server/pkg/resp"
)

type ezfyLoginReq struct {
	Name      string `json:"name" binding:"required"`
	Password  string `json:"password" binding:"required"`
	CaptchaID string `json:"captcha_id"` // 图形验证码 ID（密码输错满 3 次后必填）
	Captcha   string `json:"captcha"`    // 图形验证码答案（算式计算结果）
}

// homeUIDFromAuth 从 Authorization 里取**家园** token 的 uid。
// 无 token / 不是家园 Issuer / 已过期 / 已作废 → 返回 0（表示「家园没登录」，调用方放行）。
// ★ 只用于二战登录页的「不许换号」卡控，不参与任何鉴权。
func homeUIDFromAuth(c *gin.Context, secret string) uint64 {
	raw := strings.TrimSpace(c.GetHeader("Authorization"))
	if raw == "" {
		return 0
	}
	tok := strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
	if tok == "" {
		return 0
	}
	claims, err := authutil.ParseToken(tok, secret)
	if err != nil || claims == nil || claims.UserID == 0 {
		return 0
	}
	return uint64(claims.UserID)
}

// EzfyLogin 二战独立登录。
//
//	path: POST /api/ezfy/login （公开路由，不走二战/家园认证）
//
//	game_id = 家园号(users.id)。★★ 2026-10-18 账号统一：**密码永远以家园账号为准**
//	（家园改密码后二战旧密码立即失效，校验通过后把家园 hash 同步回 ezfy_account），
//	彻底消除「家园账号 A 和二战账号 A 密码不一致」的分裂。
//	返回二战专属 token（Issuer=qqjiayuan-ezfy）+ 家园 token（Issuer=qqjiayuan）各一个。
func (h *EzfyHandler) EzfyLogin(c *gin.Context) {
	var req ezfyLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请输入家园号和密码")
		return
	}
	name := strings.TrimSpace(req.Name)
	if req.Password == "" {
		resp.ParamError(c, "请输入密码")
		return
	}
	gameID, err := strconv.ParseUint(name, 10, 64)
	if err != nil {
		resp.ParamError(c, "请输入正确的家园号（纯数字）")
		return
	}
	if gameID == 0 {
		resp.ParamError(c, "家园号不正确")
		return
	}
	// ★★ 2026-10-18 防账号串台（用户：「A 账户登录家园，现在还能 B 账户登录二战，这不乱了」）：
	//   前端 api 拦截器对 /ezfy/login（不在 /games/ezfy/ 前缀下）会自动带**家园** token。
	//   带了有效家园 token 就说明「家园当前登录着某个账号」，此时二战登录页只允许登这个账号；
	//   想换号必须先去家园退出。否则会写出「家园是 A、二战是 B」的分叉登录态 ——
	//   更糟的是 home_token 回传还会把家园的 jy_token 一并换成 B。
	//   ★ 解析失败（没带 token / 已过期）= 家园未登录 → 放行，允许玩家直接登二战。
	if uid := homeUIDFromAuth(c, h.Secret); uid != 0 && uid != gameID {
		resp.ParamError(c, "当前已登录家园账号 "+strconv.FormatUint(uid, 10)+"，请先退出家园再登录其他账号")
		return
	}
	// ★ 密码输错满 3 次（15 分钟窗口）→ 必须过图形验证码（与家园登录一致，防脚本刷密码）
	// ★★ 2026-10-13 同 auth.go 的修复：这里原来用 `resp.ParamError`（不带 `need_captcha` 标记），
	//   前端刷新登录页后 `needCaptcha` 被重置 → 只弹提示、不亮验证码框，玩家卡死。
	//   统一带 `need_captcha: true`。
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
	// 密码校验失败统一走这里：计数 + 满次数带 need_captcha 标记
	passFail := func() {
		loginlimiter.Fail(name)
		if loginlimiter.NeedCaptcha(name) {
			resp.FailData(c, 400, "密码错误次数较多，请先通过验证码再试", gin.H{"need_captcha": true})
			return
		}
		resp.ParamError(c, "密码不对哦，再想想")
	}

	// ★★ 2026-10-18 账号统一（用户：「家园账号和二战账号不一致，这不是乱了」）：
	//   二战**不再有自己的密码** —— 身份与口令永远以家园账号为准：
	//     · 校验一律走家园库 users.password（家园改密码后，二战旧密码立即失效）；
	//     · 校验通过后把家园密码 hash 同步回 ezfy_account，两边永远一致。
	//   ezfy_account 从此只承载「登录态」职责：session_at（单点登录）/ kick_at（踢下线）/ last_login_*。
	//   每个区各一份（qq_ezzt / qq_ezzt2），但**身份都是同一个家园账号 A**。
	var user model.User
	if err := h.home().First(&user, gameID).Error; err != nil {
		resp.ParamError(c, "该家园号不存在，请先注册家园账号")
		return
	}
	if user.Status == 0 {
		resp.Forbidden(c, "该账号已被封禁，如有疑问请联系客服")
		return
	}
	// ★ 人机账号禁止登录：家园侧 / 二战侧双重卡控，都放在密码校验之前（不给爆破面）
	if user.IsBot == 1 {
		resp.ParamError(c, "该账号无法登录")
		return
	}
	var acct model.EzfyAccount
	acctFound := h.DB.Where("game_id = ?", gameID).First(&acct).Error == nil
	if acctFound && acct.IsBot == 1 {
		resp.ParamError(c, "该账号无法登录")
		return
	}
	// 口令 = 家园密码（唯一口令）
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		passFail()
		return
	}

	// 密码/昵称同步回 ezfy_account（家园改密码后这里跟着更新，彻底消除「两个账号」）
	sync := map[string]interface{}{
		"password": user.Password,
		"nickname": user.Nickname,
		"m_sync":   1,
	}
	if acctFound {
		h.DB.Model(&model.EzfyAccount{}).Where("game_id = ?", gameID).Updates(sync)
	} else {
		acct = model.EzfyAccount{GameID: uint(gameID), Password: user.Password, Nickname: user.Nickname, MSync: 1}
		if err := h.DB.Create(&acct).Error; err != nil {
			log.Printf("二战独立账号建档失败 uid=%d: %v", gameID, err)
			resp.ServerError(c, err)
			return
		}
	}
	if acct.Nickname != user.Nickname {
		acct.Nickname = user.Nickname
	}
	bound := !acctFound

	loginlimiter.Success(name)

	if acct.Nickname == "" {
		// 建档时带了昵称；这里兜底（理论不会为空）
		var cu model.User
		if err := h.home().First(&cu, acct.GameID).Error; err == nil {
			acct.Nickname = cu.Nickname
		}
	}
	// ★★ 2026-10-15 单点登录卡控（只能一个 IP 在线，防脚本多 IP 同时刷号）：
	//   session_at 取「秒级」时间戳 ×1000 —— 与 JWT 的 iat 精度对齐（jwt/v5 秒级），
	//   中间件判据 = token.iat < session_at → 掉线。
	//   ★★ 2026-10-18 必须在**签发 token 之前**写：若先签 token 再写 session_at，
	//     跨秒的瞬间刚签发的 token 就 iat < session_at = 登录即掉线。
	//   写库后立即刷新本机缓存（多机部署时其他节点最迟 60s 生效，与「踢下线」同策略）。
	//   last_login_ip 留痕（配合排查「不同城市/IP 刷号」）。
	nowSec := time.Now().Unix() * 1000
	h.DB.Model(&model.EzfyAccount{}).Where("game_id = ?", acct.GameID).
		Updates(map[string]interface{}{"last_login_at": nowSec, "session_at": nowSec, "last_login_ip": c.ClientIP()})
	middleware.SetEzfySessionAt(acct.GameID, nowSec)

	token, err := authutil.GenerateEzfyToken(acct.GameID, acct.Nickname, h.Secret, h.ExpH)
	if err != nil {
		resp.ServerError(c, err)
		return
	}
	// ★ 2026-10-12 用家园号+家园密码登二战，点【家园】应已是家园已登录态。
	//   二战口令经过家园校验 → 用同一签名秘密再签发一个**家园** token
	//   （Issuer=qqjiayuan，与二战 Issuer=qqjiayuan-ezfy 区分）一起返回，前端顺势写进家园 jy_token；
	//   两个 token 各自独立签名、独立过期，互不影响。
	homeToken, herr := authutil.GenerateToken(acct.GameID, acct.Nickname, h.Secret, h.ExpH)
	if herr != nil {
		resp.ServerError(c, herr)
		return
	}
	resp.OK(c, gin.H{"token": token, "home_token": homeToken, "game_id": acct.GameID, "nickname": acct.Nickname, "bound": bound})
}

// EzfySSO ★ 2026-10-12 家园已登录 → 免密换取二战登录态。
//
// 「从家园登录页面登录了，再进二战征途还需要登录，不对」。
// 二战独立登录后二战页只认 ezfy_token，家园已登录的玩家进来仍被要求再输一次密码。
// 这里补一条「家园登录态换二战 token」的通道：
//
//	Authorization 里带**家园** token（Issuer=qqjiayuan，家园登录时签发）→ 校验通过即
//	视为身份可信（token 本身就是家园登录成功的凭证，无需再验密码）→ 找/建 ezfy_account
//	→ 签发二战 token（Issuer=qqjiayuan-ezfy）。
//
// ★ 只认家园 Issuer：authutil.ParseToken 内部校验，二战 token 拿不来这里自我续期。
// ★ 失败一律返回**业务码 400**（HTTP 200），刻意不用 401 —— 前端 axios 拦截器把
//   非二战接口的 401 当作「家园登录过期」会强制登出并跳家园 /login；而这里的语义
//   只是「家园没登录/失效」，玩家应当安静地看到二战自己的登录框。
//
//	path: POST /api/ezfy/sso （公开路由；前端在「家园已登录但还没有 ezfy_token」时调用）
func (h *EzfyHandler) EzfySSO(c *gin.Context) {
	raw := strings.TrimSpace(c.GetHeader("Authorization"))
	tokenStr := strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
	if tokenStr == "" {
		resp.ParamError(c, "未登录家园账号")
		return
	}
	claims, err := authutil.ParseToken(tokenStr, h.Secret)
	if err != nil || claims == nil || claims.UserID == 0 {
		resp.ParamError(c, "家园登录状态已失效，请用家园号和密码登录")
		return
	}
	uid := claims.UserID
	// ★★ 2026-10-18 性能（用户：「SSO 请求顺便优化性能，这系统人少，还不至于卡顿吧，才200多个人」）：
	//   家园库 `users` 和二战库 `ezfy_account` 是**两个库**，原来串行查 → 两次跨 WAN 往返；
	//   这里并发查，只花一次的时间。Select 只取用到的列，少传点数据。
	var user model.User
	var acct model.EzfyAccount
	var userErr, acctErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		userErr = h.home().Select("id, password, nickname, status, is_bot").First(&user, uid).Error
	}()
	go func() {
		defer wg.Done()
		acctErr = h.DB.Where("game_id = ?", uid).First(&acct).Error
	}()
	wg.Wait()
	if userErr != nil {
		resp.ParamError(c, "家园账号不存在")
		return
	}
	if user.Status == 0 {
		resp.Forbidden(c, "该账号已被封禁，如有疑问请联系客服")
		return
	}
	// ★ 2026-10-17 人机账号禁止通过 SSO 登录（家园 token 换二战 token 这条路也要堵）
	if user.IsBot == 1 {
		resp.ParamError(c, "该账号无法登录")
		return
	}
	// ★★ 2026-10-18 单点登录卡控（与 EzfyLogin 同口径）：SSO 换发新 token 同样顶掉旧 token。
	//   ★ 必须在**签发 token 之前**写（跨秒会登录即掉线）。
	nowSec := time.Now().Unix() * 1000
	// ★★ 2026-10-18 资料同步 + 登录痕迹**合并成一条写**（原来拆两条 UPDATE = 两次往返）。
	//   账号统一后密码/昵称永远以家园为准，这里顺带覆盖；session_at 同一条写掉。
	if acctErr == nil {
		h.DB.Model(&model.EzfyAccount{}).Where("game_id = ?", uid).
			Updates(map[string]interface{}{
				"password": user.Password, "nickname": user.Nickname, "m_sync": 1,
				"last_login_at": nowSec, "session_at": nowSec, "last_login_ip": c.ClientIP(),
			})
		if user.Nickname != "" {
			acct.Nickname = user.Nickname
		}
	} else {
		acct = model.EzfyAccount{
			GameID:      uid,
			Password:    user.Password, // 拷家园 hash（保持列语义，登录校验不再用它）
			Nickname:    user.Nickname,
			MSync:       1,
			LastLoginAt: nowSec,
			SessionAt:   nowSec,
			LastLoginIP: c.ClientIP(),
		}
		if err := h.DB.Create(&acct).Error; err != nil {
			log.Printf("二战 SSO 建档失败 uid=%d: %v", uid, err)
			resp.ServerError(c, err)
			return
		}
	}
	if acct.Nickname == "" {
		acct.Nickname = user.Nickname
	}
	middleware.SetEzfySessionAt(uid, nowSec)

	token, err := authutil.GenerateEzfyToken(acct.GameID, acct.Nickname, h.Secret, h.ExpH)
	if err != nil {
		resp.ServerError(c, err)
		return
	}
	resp.OK(c, gin.H{"token": token, "game_id": acct.GameID, "nickname": acct.Nickname})
}

// EzfyLogout 二战登出：作废当前账号的二战 token。
//
// ★★ 2026-10-18 用户要求：「再二战进家园（底部家园导航）也要作废二战的 token，
// 家园再进二战就继续生成新的 token」。二战页没有独立退出入口（也不需要），
// 点底部「家园」就是唯一的出口 —— 那里顺手把二战 token 作废，回家园再进来时
// 由 EzfySSO 重新签发。这样二战 token 不会长期残留在浏览器里，换号（退家园 A →
// 登家园 B → 进二战）时也不可能拿着 A 的旧 token 继续玩。
//
// 实现：把 ezfy_account.session_at 顶到当前 → 该账号此前签发的二战 token 全部
// 按「iat < session_at」作废（与单点登录同一套判据）。没带 token / token 已失效
// 一律当「已退出」返回成功，前端照常跳转，不阻塞玩家回家园。
//
//	path: POST /api/ezfy/logout （公开路由）
func (h *EzfyHandler) EzfyLogout(c *gin.Context) {
	raw := strings.TrimSpace(c.GetHeader("Authorization"))
	tok := strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
	if tok == "" {
		resp.OK(c, gin.H{"msg": "已退出"})
		return
	}
	claims, err := authutil.ParseEzfyToken(tok, h.Secret)
	if err != nil || claims == nil || claims.UserID == 0 {
		resp.OK(c, gin.H{"msg": "已退出"})
		return
	}
	// ★★ 2026-10-18 作废线取「当前秒 + 1」：JWT 的 iat 只有**秒级**精度，
	//   若同秒内先签发再登出，`iat(T) < session_at(T)` 为 false → 旧 token 作废不掉。
	//   推到下一秒后：本秒签发的 token 一律 `iat < session_at` ✅ 作废；
	//   而登录/SSO 会**重新写** session_at=now（见 EzfyLogin/EzfySSO），新 token 不会被误杀。
	//   （家园侧不能这么干：家园登录不写 users.session_at，+1 秒会让「退出后同秒登录」直接掉线。）
	nowSec := (time.Now().Unix() + 1) * 1000
	h.DB.Model(&model.EzfyAccount{}).Where("game_id = ?", claims.UserID).
		Updates(map[string]interface{}{"session_at": nowSec})
	middleware.SetEzfySessionAt(claims.UserID, nowSec)
	resp.OK(c, gin.H{"msg": "已退出"})
}
