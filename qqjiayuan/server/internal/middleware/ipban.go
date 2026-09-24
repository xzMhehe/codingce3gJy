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
		ip := c.ClientIP()
		if _, hit := set[ip]; hit {
			// ★ bug 修复（「没有人封禁却展示 503」）：命中缓存名单后必须回库二次确认。
			//   站长按 503 页提示直接在库上 DELETE FROM ip_bans 解封时，本中间件 30 秒
			//   轮询窗口内仍持有旧名单，会把解封后的请求误判为封禁。回库确认后：
			//   库里确无记录 → 同步清缓存并放行，从根上消除假阳性。
			var n int64
			if err := m.db.Model(&model.IPBan{}).Where("ip = ?", ip).Count(&n).Error; err == nil && n == 0 {
				m.reload()
				c.Next()
				return
			}
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
