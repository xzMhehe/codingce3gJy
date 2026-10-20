package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// GzipStatic 对管理端 / 前端静态资源（JS/CSS/HTML/JSON/SVG）做 gzip 压缩，
// 并给带 hash 的构建产物加长期缓存头。
//
// ★ 2026-10-20 管理端首屏慢：admin-web 的 JS/CSS 未压缩时约 2.4MB，
// 走公网 IP 加载要好几秒。这里在 Go 侧统一 gzip（浏览器带 Accept-Encoding 才压），
// 同时 hash 资源一年强缓存，index.html 仍 no-cache，重部署后能立即拿到新资源。
func GzipStatic() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet || !strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}
		p := c.Request.URL.Path
		if !isCompressiblePath(p) {
			c.Next()
			return
		}

		// 构建产物文件名带 content hash，可安全长期缓存；HTML 必须每次回源校验。
		if isHashedAsset(p) {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		} else if strings.HasSuffix(p, ".html") || strings.HasSuffix(p, "/") {
			c.Header("Cache-Control", "no-cache")
		}

		gw := &gzipResponseWriter{ResponseWriter: c.Writer, gz: gzip.NewWriter(c.Writer)}
		c.Writer = gw
		c.Next()
		_ = gw.Close()
	}
}

type gzipResponseWriter struct {
	gin.ResponseWriter
	gz      *gzip.Writer
	decided bool
	gzipOn  bool
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	// ★ 注意：gin 的静态文件处理器会先 WriteHeader(404)，真正找到文件后再改成 200
	//   （gin.responseWriter.WriteHeader 在未真正写出前允许改状态码）。
	//   这里不能把第一次非 200 当成最终响应，否则静态资源永远压不到。
	if code == http.StatusOK && !w.decided && !w.ResponseWriter.Written() {
		w.decide(code)
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.decided {
		w.decide(w.ResponseWriter.Status())
	}
	if w.gzipOn {
		return w.gz.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func (w *gzipResponseWriter) decide(code int) {
	w.decided = true
	ct := strings.ToLower(w.Header().Get("Content-Type"))
	if code == http.StatusOK && w.Header().Get("Content-Encoding") == "" && isCompressibleType(ct) {
		w.gzipOn = true
		w.Header().Del("Content-Length")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
	}
}

func (w *gzipResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *gzipResponseWriter) Close() error {
	if w.gzipOn {
		return w.gz.Close()
	}
	return nil
}

func isCompressiblePath(p string) bool {
	return strings.HasPrefix(p, "/admin-ui/") || strings.HasPrefix(p, "/assets/")
}

func isHashedAsset(p string) bool {
	return strings.HasPrefix(p, "/admin-ui/js/") ||
		strings.HasPrefix(p, "/admin-ui/css/") ||
		strings.HasPrefix(p, "/admin-ui/fonts/") ||
		strings.HasPrefix(p, "/admin-ui/img/") ||
		strings.HasPrefix(p, "/assets/")
}

func isCompressibleType(ct string) bool {
	return strings.HasPrefix(ct, "text/") ||
		strings.Contains(ct, "javascript") ||
		strings.Contains(ct, "json") ||
		strings.Contains(ct, "xml") ||
		strings.Contains(ct, "svg")
}
