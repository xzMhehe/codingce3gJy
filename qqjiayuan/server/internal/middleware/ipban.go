package middleware

import (
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// IPBan 全站 IP 封禁：命中封禁名单的请求一律 302 到「服务不可用」页（含管理端）
type IPBan struct {
	db   *gorm.DB
	list atomic.Value // map[string]struct{}
}

func NewIPBan(db *gorm.DB) *IPBan {
	m := &IPBan{db: db}
	m.list.Store(map[string]struct{}{})
	m.reload()
	go func() {
		for range time.Tick(30 * time.Second) {
			m.reload()
		}
	}()
	return m
}

func (m *IPBan) reload() {
	var ips []string
	m.db.Model(&model.IPBan{}).Pluck("ip", &ips)
	set := make(map[string]struct{}, len(ips))
	for _, ip := range ips {
		set[ip] = struct{}{}
	}
	m.list.Store(set)
}

// Refresh 封禁/解封后立即重载，不必等 30 秒轮询
func (m *IPBan) Refresh() { m.reload() }

func (m *IPBan) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		// 服务不可用页自身放行，避免重定向死循环
		if p == "/503.html" || p == "/admin-ui/503.html" {
			c.Next()
			return
		}
		set, _ := m.list.Load().(map[string]struct{})
		if _, hit := set[c.ClientIP()]; hit {
			target := "/503.html"
			if strings.HasPrefix(p, "/admin-ui") || strings.HasPrefix(p, "/api/admin") {
				target = "/admin-ui/503.html"
			}
			c.Redirect(http.StatusFound, target)
			c.Abort()
			return
		}
		c.Next()
	}
}
