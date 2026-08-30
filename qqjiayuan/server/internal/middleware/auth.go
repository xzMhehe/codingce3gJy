package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/pkg/authutil"
	"qqjiayuan/server/pkg/resp"
)

const (
	CtxUID   = "uid"
	CtxUName = "uname"
)

// CORS 跨域
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", c.GetHeader("Origin"))
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// JWT 认证，并把活跃时间落库用于「在线人数」统计（节流 1 分钟）
func JWTAuth(db *gorm.DB, secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			resp.Unauthorized(c, "请先登录")
			c.Abort()
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		claims, err := authutil.ParseToken(tokenStr, secret)
		if err != nil {
			resp.Unauthorized(c, "登录已过期，请重新登录")
			c.Abort()
			return
		}
		c.Set(CtxUID, claims.UserID)
		c.Set(CtxUName, claims.Nickname)

		// 节流更新活跃时间（10 分钟内活跃视为在线）
		var last time.Time
		db.Raw("SELECT IFNULL(last_active_at, '2000-01-01') FROM users WHERE id = ?", claims.UserID).Scan(&last)
		if time.Since(last) > time.Minute {
			now := time.Now()
			db.Model(&struct{}{}).Table("users").
				Where("id = ?", claims.UserID).
				Updates(map[string]interface{}{"last_active_at": now, "last_login_at": now})
		}
		c.Next()
	}
}

func GetUID(c *gin.Context) uint {
	v, _ := c.Get(CtxUID)
	uid, _ := v.(uint)
	return uid
}

func GetUName(c *gin.Context) string {
	v, _ := c.Get(CtxUName)
	s, _ := v.(string)
	return s
}
