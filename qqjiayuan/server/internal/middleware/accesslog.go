package middleware

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AccessLog 只记录出错和慢请求，不再把每个静态资源请求写进 journald。
//
// ★ 2026-10-23 性能：gin.Default() 会对每个 JS/图片请求写一行日志，小机器上
// 静态资源一多就是纯磁盘 I/O 浪费。常规 2xx API 不再记录，排障看 4xx/5xx 和慢请求即可。
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		if isStaticPath(p) {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		d := time.Since(start)
		status := c.Writer.Status()
		if status >= http.StatusBadRequest || d >= 500*time.Millisecond {
			log.Printf("[HTTP] %d %s %s %s %v",
				status, c.Request.Method, p, c.ClientIP(), d.Round(time.Millisecond))
		}
	}
}

func isStaticPath(p string) bool {
	return strings.HasPrefix(p, "/js/") ||
		strings.HasPrefix(p, "/css/") ||
		strings.HasPrefix(p, "/static/") ||
		strings.HasPrefix(p, "/assets/") ||
		strings.HasPrefix(p, "/admin-ui/") ||
		p == "/favicon.ico" ||
		p == "/robots.txt"
}
