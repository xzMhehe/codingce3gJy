package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/pkg/authutil"
)

// OptionalAuth 可选认证：若请求携带有效 token 则设置 uid/uname，否则放行（不拦截）
// 用于公开接口中需要「若已登录则个性化」的场景（如帖子详情的投票状态、收藏状态）
func OptionalAuth(db *gorm.DB, secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth != "" {
			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			if claims, err := authutil.ParseToken(tokenStr, secret); err == nil {
				c.Set(CtxUID, claims.UserID)
				c.Set(CtxUName, claims.Nickname)
			}
		}
		c.Next()
	}
}