package authutil

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   uint   `json:"uid"`
	Nickname string `json:"uname"`
	jwt.RegisteredClaims
}

// homeIssuer 家园 token 的签发者标识。
const homeIssuer = "qqjiayuan"

// EzfyIssuer 二战征途独立 token 的签发者标识。
//
// ★ 2026-10-12 二战与家园账号体系分离：二战 token 与家园 token 用同一签名密钥、
//   但用不同 Issuer 区分。二战专属中间件只认 EzfyIssuer，家园中间件只认 homeIssuer，
//   互相不能冒用 → 二战登录态彻底独立。
const EzfyIssuer = "qqjiayuan-ezfy"

// AdminIssuer 管理端（admin-web）独立 token 的签发者标识。
//
// ★★ 2026-10-19 管理端与用户端 token 分离：
//   - 同一套账号体系、同一签名密钥，但**用 Issuer 区分「哪个端登录的」**；
//   - 用户端（家园 web / 二战用户端）只认 homeIssuer，管理端只认 AdminIssuer；
//   - 两条会话作废线也分开：用户端 `users.session_at`、管理端 `users.admin_session_at`
//     → 管理端登录/退出**不再顶掉用户端**（原现象：管理端一登录，玩家端被迫下线）。
const AdminIssuer = "qqjiayuan-admin"

func GenerateToken(userID uint, nickname, secret string, expireHours int) (string, error) {
	return generateToken(userID, nickname, secret, homeIssuer, expireHours)
}

// GenerateEzfyToken 签发二战专属 token（Issuer=EzfyIssuer）。
func GenerateEzfyToken(userID uint, nickname, secret string, expireHours int) (string, error) {
	return generateToken(userID, nickname, secret, EzfyIssuer, expireHours)
}

// GenerateAdminToken 签发管理端专属 token（Issuer=AdminIssuer）。
func GenerateAdminToken(userID uint, nickname, secret string, expireHours int) (string, error) {
	return generateToken(userID, nickname, secret, AdminIssuer, expireHours)
}

func generateToken(userID uint, nickname, secret, issuer string, expireHours int) (string, error) {
	claims := Claims{
		UserID:   userID,
		Nickname: nickname,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseToken(tokenStr, secret string) (*Claims, error) {
	return parseToken(tokenStr, secret, homeIssuer)
}

// ParseEzfyToken 解析二战 token 并校验签发者必须是二战 Issuer。
func ParseEzfyToken(tokenStr, secret string) (*Claims, error) {
	return parseToken(tokenStr, secret, EzfyIssuer)
}

// ParseAnyToken 解析「家园端 + 管理端」两类 token（二战 token 不在此列）。
//
// ★★ 2026-10-19 管理端与用户端 token 分离：管理端接口（/api/admin/*）与用户端接口
//   共用同一批 handler/中间件，所以 JWTAuth 必须两类都认；**真正的隔离在会话作废线上**
//   （见 middleware.JWTAuth：home token 校 users.session_at，admin token 校 admin_session_at），
//   互不顶线 = 管理端登录不会把玩家端踢下线，反之亦然。
//   返回的 claims.Issuer 可用于区分来源端（Logout 据此决定顶哪条线）。
func ParseAnyToken(tokenStr, secret string) (*Claims, error) {
	return parseToken(tokenStr, secret, homeIssuer, AdminIssuer)
}

func parseToken(tokenStr, secret string, issuers ...string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		for _, issuer := range issuers {
			if claims.Issuer == issuer {
				return claims, nil
			}
		}
		return nil, errors.New("token 类型不符")
	}
	return nil, errors.New("token 无效")
}
