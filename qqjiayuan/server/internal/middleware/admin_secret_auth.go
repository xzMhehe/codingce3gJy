package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminSecretAuth ★ 2026-10-17 二战独立部署：二战服管理端路由的组级鉴权中间件。
//
// 只校验内部密钥 header `X-Ezfy-Admin-Secret` 是否等于 cfg.Jwt.Secret：
//   - home 侧二战管理端反向代理（router.ezfyAdminProxy）会注入该 header（并去掉玩家
//     Authorization），普通外部请求拿不到 → 403；
//   - RBAC（permissions/roles 表）在家园库，由 home 侧 authed jwtM + perm() 驱动，
//     二战服不再做权限点校验。
//
// 校验失败一律 403（不区分「缺 header」与「header 错误」，避免探测）。
func AdminSecretAuth(secret string) gin.HandlerFunc {
	if secret == "" {
		// 配置缺失属于部署事故，直接拒绝全部二战管理端请求比「裸奔」安全。
		return func(c *gin.Context) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": "二战管理端密钥未配置", "data": nil})
		}
	}
	return func(c *gin.Context) {
		if c.GetHeader("X-Ezfy-Admin-Secret") != secret {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": "二战管理端密钥校验失败", "data": nil})
			return
		}
		c.Next()
	}
}
