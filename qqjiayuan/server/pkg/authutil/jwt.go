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

func GenerateToken(userID uint, nickname, secret string, expireHours int) (string, error) {
	return generateToken(userID, nickname, secret, homeIssuer, expireHours)
}

// GenerateEzfyToken 签发二战专属 token（Issuer=EzfyIssuer）。
func GenerateEzfyToken(userID uint, nickname, secret string, expireHours int) (string, error) {
	return generateToken(userID, nickname, secret, EzfyIssuer, expireHours)
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

func parseToken(tokenStr, secret, issuer string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		if claims.Issuer != issuer {
			return nil, errors.New("token 类型不符")
		}
		return claims, nil
	}
	return nil, errors.New("token 无效")
}
