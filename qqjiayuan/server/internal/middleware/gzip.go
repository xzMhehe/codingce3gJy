package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// GzipStatic 对前端/管理端静态资源做 gzip，并给静态资源补缓存头。
//
// ★ 2026-10-23 性能：web/dist 的主站 JS/CSS 走 /js、/css，原来只压 /assets、
// /admin-ui，导致 1.8MB JS 明文下载（1Mbps 小带宽要几十秒）。现在按扩展名压缩，
// 压缩器用 sync.Pool 复用 + BestSpeed，避免每请求新建 writer 和默认级别的高 CPU。
func GzipStatic() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Next()
			return
		}
		p := c.Request.URL.Path
		if isHashedAsset(p) {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		} else if strings.HasSuffix(p, ".html") || strings.HasSuffix(p, "/") {
			c.Header("Cache-Control", "no-cache")
		}

		if c.Request.Method != http.MethodGet ||
			!strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") ||
			!isCompressiblePath(p) {
			c.Next()
			return
		}

		gw := &gzipResponseWriter{ResponseWriter: c.Writer}
		c.Writer = gw
		c.Next()
		_ = gw.Close()
	}
}

var gzipWriterPool = sync.Pool{
	New: func() any {
		gz, _ := gzip.NewWriterLevel(io.Discard, gzip.BestSpeed)
		return gz
	},
}

type gzipResponseWriter struct {
	gin.ResponseWriter
	gz      *gzip.Writer
	decided bool
	gzipOn  bool
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	// gin 的静态文件处理器会先 WriteHeader(404)，真正找到文件后再改成 200，
	// 所以只有 200 且尚未写出时才决定是否压缩。
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
		gz := gzipWriterPool.Get().(*gzip.Writer)
		gz.Reset(w.ResponseWriter)
		w.gz = gz
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
	if !w.gzipOn {
		return nil
	}
	err := w.gz.Close()
	w.gz.Reset(io.Discard)
	gzipWriterPool.Put(w.gz)
	w.gz = nil
	w.gzipOn = false
	return err
}

func isCompressiblePath(p string) bool {
	if strings.HasPrefix(p, "/api/") {
		return false
	}
	switch strings.ToLower(path.Ext(p)) {
	case ".js", ".mjs", ".css", ".html", ".htm", ".json", ".svg", ".xml", ".txt", ".map":
		return true
	}
	return false
}

func isHashedAsset(p string) bool {
	return strings.HasPrefix(p, "/js/") ||
		strings.HasPrefix(p, "/css/") ||
		strings.HasPrefix(p, "/assets/") ||
		strings.HasPrefix(p, "/admin-ui/js/") ||
		strings.HasPrefix(p, "/admin-ui/css/") ||
		strings.HasPrefix(p, "/admin-ui/fonts/") ||
		strings.HasPrefix(p, "/admin-ui/img/")
}

func isCompressibleType(ct string) bool {
	return strings.HasPrefix(ct, "text/") ||
		strings.Contains(ct, "javascript") ||
		strings.Contains(ct, "json") ||
		strings.Contains(ct, "xml") ||
		strings.Contains(ct, "svg")
}
