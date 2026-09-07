package handler

import (
	"math"
	"strconv"
	"sync"
	"time"
)

// ============ 频控（对齐诺哈 VerifyTime：心情/留言/日志 15 秒，邀请 5 秒） ============

var (
	verifyMu  sync.Mutex
	verifyMap = map[string]time.Time{}
)

// verifyTime 校验频控：距离上次操作不足 seconds 秒返回 false
func verifyTime(key string, seconds int) bool {
	verifyMu.Lock()
	defer verifyMu.Unlock()
	now := time.Now()
	if t, ok := verifyMap[key]; ok && now.Sub(t) < time.Duration(seconds)*time.Second {
		return false
	}
	verifyMap[key] = now
	return true
}

// ============ 家园等级（诺哈公式：P=L²+4L → L=(√(16+4P)-4)/2） ============

func homePointLevel(point int) (lv int, need int) {
	if point < 0 {
		point = 0
	}
	lv = int((math.Sqrt(float64(16+4*point)) - 4) / 2)
	if lv < 0 {
		lv = 0
	}
	need = (lv+1)*(lv+1) + 4*(lv+1)
	return
}

func atoiDefault(s string, d int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return d
	}
	return n
}
