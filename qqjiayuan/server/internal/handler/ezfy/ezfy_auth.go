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

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/authutil"
	"qqjiayuan/server/pkg/resp"
)

type ezfyLoginReq struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
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

	var acct model.EzfyAccount
	found := h.DB.Where("game_id = ?", gameID).First(&acct).Error == nil
	bound := false
	if found {
		// 二战自有账号 → 二战密码校验
		if bcrypt.CompareHashAndPassword([]byte(acct.Password), []byte(req.Password)) != nil {
			resp.ParamError(c, "密码不对哦，再想想")
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
			resp.ParamError(c, "密码不对哦，再想想")
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
	resp.OK(c, gin.H{"token": token, "game_id": acct.GameID, "nickname": acct.Nickname, "bound": bound})
}