// 魔法花园 · 独立前后端（标准库实现，零第三方依赖）
// 单独运行：cd games/magic-garden/server && go run main.go  → http://localhost:8090
// 数据保存在内存中（按昵称隔离），简化演示；可自由扩展为 SQLite/MySQL。
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ---- 作物定义 ----
type crop struct {
	Name string `json:"name"`
	Seed int    `json:"seed"`
	Secs int    `json:"secs"`
	Sell int    `json:"sell"`
}

var crops = []crop{
	{"向日葵", 5, 30, 12},
	{"玫瑰花", 10, 60, 25},
	{"郁金香", 20, 120, 55},
	{"月光花", 40, 240, 115},
}

const plotCount = 4

// ---- 花园状态（内存存储） ----
type plot struct {
	Crop   string    `json:"crop"`
	SeedAt time.Time `json:"-"`
}
type state struct {
	mu    sync.Mutex
	plots [plotCount]plot
	coins int
}

var (
	mu    sync.Mutex
	users = map[string]*state{}
)

func getOr(nick string) *state {
	mu.Lock()
	defer mu.Unlock()
	u, ok := users[nick]
	if !ok {
		u = &state{coins: 100}
		users[nick] = u
	}
	return u
}

func plantRemain(p plot) int {
	if p.Crop == "" {
		return 0
	}
	secs := 0
	for _, c := range crops {
		if c.Name == p.Crop {
			secs = c.Secs
			break
		}
	}
	remain := secs - int(time.Since(p.SeedAt).Seconds())
	if remain < 0 {
		return 0
	}
	return remain
}

// ---- API ----
func stateOf(u *state) map[string]interface{} {
	u.mu.Lock()
	defer u.mu.Unlock()
	out := make([]ginish, 0, 4)
	for i, p := range u.plots {
		st := "empty"
		remain := 0
		if p.Crop != "" {
			st = "growing"
			remain = plantRemain(p)
			if remain == 0 {
				st = "ripe"
			}
		}
		out = append(out, ginish{"index": i, "crop": p.Crop, "status": st, "remain": remain})
	}
	return map[string]interface{}{"plots": out, "coins": u.coins, "crops": crops}
}

type ginish map[string]interface{}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

func full(cropName string) (crop, bool) {
	for _, c := range crops {
		if c.Name == cropName {
			return c, true
		}
	}
	return crop{}, false
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	mux := http.NewServeMux()

	// /api/state?nick=xxx
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		nick := r.URL.Query().Get("nick")
		if nick == "" {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "请提供昵称", "data": nil})
			return
		}
		writeJSON(w, map[string]interface{}{"code": 0, "msg": "ok", "data": stateOf(getOr(nick))})
	})

	// /api/plant {nick,index,crop}
	mux.HandleFunc("/api/plant", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Nick  string `json:"nick"`
			Index int    `json:"index"`
			Crop  string `json:"crop"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Nick == "" || req.Index < 0 || req.Index >= plotCount {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "参数有误", "data": nil})
			return
		}
		c, ok := full(req.Crop)
		if !ok {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "作物不存在", "data": nil})
			return
		}
		u := getOr(req.Nick)
		u.mu.Lock()
		defer u.mu.Unlock()
		if u.plots[req.Index].Crop != "" {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "这块地已经种上了", "data": nil})
			return
		}
		if u.coins < c.Seed {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "金币不足", "data": nil})
			return
		}
		u.coins -= c.Seed
		u.plots[req.Index] = plot{Crop: c.Name, SeedAt: time.Now()}
		writeJSON(w, map[string]interface{}{"code": 0, "msg": "ok", "data": nil})
	})

	// /api/harvest {nick,index}
	mux.HandleFunc("/api/harvest", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Nick  string `json:"nick"`
			Index int    `json:"index"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Nick == "" || req.Index < 0 || req.Index >= plotCount {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "参数有误", "data": nil})
			return
		}
		u := getOr(req.Nick)
		u.mu.Lock()
		defer u.mu.Unlock()
		p := u.plots[req.Index]
		if p.Crop == "" || plantRemain(p) > 0 {
			writeJSON(w, map[string]interface{}{"code": 400, "msg": "还没成熟", "data": nil})
			return
		}
		sell := 0
		for _, c := range crops {
			if c.Name == p.Crop {
				sell = c.Sell
				break
			}
		}
		u.coins += sell
		u.plots[req.Index] = plot{}
		writeJSON(w, map[string]interface{}{"code": 0, "msg": "ok", "data": map[string]interface{}{"gain": sell, "coins": u.coins}})
	})

	// 静态前端
	webDir := "web"
	if d, err := os.Stat(webDir); err != nil || !d.IsDir() {
		webDir = filepath.Join("..", "web")
	}
	mux.Handle("/", http.FileServer(http.Dir(webDir)))

	addr := ":" + port
	fmt.Println("======================================")
	fmt.Println("  魔法花园（独立版）已启动  http://localhost" + addr)
	fmt.Println("  单独运行完毕，浏览器打开即可游玩")
	fmt.Println("======================================")
	log.Fatal(http.ListenAndServe(addr, mux))
}

// 随机昵称种子（前端默认昵称用）
func randNick() string {
	b := make([]byte, 4)
	rand.Read(b)
	return "花友" + hex.EncodeToString(b)[:6]
}
