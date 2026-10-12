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
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

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

// EzfyLogin 二战独立登录。
//
//	path: POST /api/ezfy/login （公开路由，不走二战/家园认证）
//
//	game_id = 家园号(users.id)。
//	· 二战已有该账号 → 直接用二战密码比对。
//	· 首次进入（本表无记录）→ 用家园号去家园库校验家园密码，通过后把家园密码 hash
//	  拷入 ezfy_account 建档，并回带 bound=true。
//	返回二战专属 token（Issuer=qqjiayuan-ezfy），与家园 token 完全独立。
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
	// ★ 密码输错满 3 次（15 分钟窗口）→ 必须过图形验证码（与家园登录一致，防脚本刷密码）
	if loginlimiter.NeedCaptcha(name) {
		if strings.TrimSpace(req.Captcha) == "" {
			resp.ParamError(c, "密码错误次数较多，请输入验证码（图片算式的计算结果）")
			return
		}
		if !captcha.Verify(req.CaptchaID, req.Captcha) {
			resp.ParamError(c, "验证码不对哦，请点击图片刷新后重试")
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

	var acct model.EzfyAccount
	found := h.DB.Where("game_id = ?", gameID).First(&acct).Error == nil
	bound := false
	if found {
		// 二战自有账号 → 二战密码校验
		if bcrypt.CompareHashAndPassword([]byte(acct.Password), []byte(req.Password)) != nil {
			passFail()
			return
		}
	} else {
		// 首登：去家园库核对家园号 + 家园密码
		var user model.User
		if err := h.home().First(&user, gameID).Error; err != nil {
			resp.ParamError(c, "该家园号不存在，请先注册家园账号")
			return
		}
		if user.Status == 0 {
			resp.Forbidden(c, "该账号已被封禁，如有疑问请联系客服")
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
			passFail()
			return
		}
		// 校验通过 → 同步家园密码 hash 建档（此后二战自助管，不再依赖家园）
		acct = model.EzfyAccount{
			GameID:   uint(gameID),
			Password: user.Password,
			Nickname: user.Nickname,
			MSync:    1,
		}
		if err := h.DB.Create(&acct).Error; err != nil {
			log.Printf("二战独立账号建档失败 uid=%d: %v", gameID, err)
			resp.ServerError(c, err)
			return
		}
		bound = true
	}
	loginlimiter.Success(name)

	if acct.Nickname == "" {
		// 建档时带了昵称；这里兜底（理论不会为空）
		var cu model.User
		if err := h.home().First(&cu, acct.GameID).Error; err == nil {
			acct.Nickname = cu.Nickname
		}
	}
	token, err := authutil.GenerateEzfyToken(acct.GameID, acct.Nickname, h.Secret, h.ExpH)
	if err != nil {
		resp.ServerError(c, err)
		return
	}
	// ★ 2026-10-12 用户反馈：用家园号+家园密码登二战，点【家园】应已是家园已登录态。
	//   二战口令经过家园校验（或已建档的自有账号）→ 用同一签名秘密再签发一个**家园** token
	//   （Issuer=qqjiayuan，与二战 Issuer=qqjiayuan-ezfy 区分）一起返回，前端顺势写进家园 jy_token；
	//   两个 token 各自独立签名、独立过期，互不影响。
	homeToken, herr := authutil.GenerateToken(acct.GameID, acct.Nickname, h.Secret, h.ExpH)
	if herr != nil {
		resp.ServerError(c, herr)
		return
	}
	// ★ 2026-10-12 管理端「玩家信息管理」展示最后登陆时间用 → 登录成功落库
	h.DB.Model(&model.EzfyAccount{}).Where("game_id = ?", acct.GameID).Update("last_login_at", time.Now().UnixMilli())
	resp.OK(c, gin.H{"token": token, "home_token": homeToken, "game_id": acct.GameID, "nickname": acct.Nickname, "bound": bound})
}