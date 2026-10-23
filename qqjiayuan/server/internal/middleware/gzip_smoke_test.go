package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGzipStaticSmoke(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	body := strings.Repeat("console.log('hello world');\n", 400)
	jsPath := filepath.Join(dir, "app.1234abcd.js")
	if err := os.WriteFile(jsPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	htmlPath := filepath.Join(dir, "index.html")
	if err := os.WriteFile(htmlPath, []byte("<html>hi</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	imgPath := filepath.Join(dir, "picture.jpg")
	if err := os.WriteFile(imgPath, bytes.Repeat([]byte{0xFF}, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.Use(GzipStatic())
	r.StaticFile("/js/app.1234abcd.js", jsPath)
	r.StaticFile("/index.html", htmlPath)
	r.StaticFile("/static/picture.jpg", imgPath)

	do := func(method, target string, gz bool) *http.Response {
		req := httptest.NewRequest(method, target, nil)
		if gz {
			req.Header.Set("Accept-Encoding", "gzip")
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Result()
	}

	res := do(http.MethodGet, "/js/app.1234abcd.js", true)
	if got := res.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("hashed js not gzipped, encoding=%q", got)
	}
	if got := res.Header.Get("Cache-Control"); !strings.Contains(got, "immutable") {
		t.Fatalf("hashed js cache-control=%q", got)
	}
	zr, err := gzip.NewReader(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != body {
		t.Fatalf("gzip roundtrip mismatch: %d vs %d", len(plain), len(body))
	}

	res = do(http.MethodGet, "/js/app.1234abcd.js", false)
	if res.Header.Get("Content-Encoding") != "" {
		t.Fatal("must not gzip without Accept-Encoding")
	}
	if res.ContentLength != int64(len(body)) {
		t.Fatalf("plain content-length=%d want %d", res.ContentLength, len(body))
	}

	res = do(http.MethodHead, "/js/app.1234abcd.js", true)
	if res.Header.Get("Content-Encoding") != "" {
		t.Fatal("HEAD must not be gzipped")
	}

	res = do(http.MethodGet, "/static/picture.jpg", true)
	if res.Header.Get("Content-Encoding") != "" {
		t.Fatal("jpg must not be gzipped")
	}
	if cc := res.Header.Get("Cache-Control"); cc != "" {
		t.Fatalf("jpg should keep original caching, got %q", cc)
	}

	res = do(http.MethodGet, "/index.html", true)
	if res.Header.Get("Cache-Control") != "no-cache" {
		t.Fatalf("html cache-control=%q", res.Header.Get("Cache-Control"))
	}
}
