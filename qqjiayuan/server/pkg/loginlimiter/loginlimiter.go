// Package loginlimiter 登录失败计数
//
// 同一账号在 15 分钟窗口内密码输错满 3 次后，登录必须通过图形验证码。
// 只对「账号存在且密码错误」计数（账号不存在不计数，避免被任意字符串刷爆内存）；
// 登录成功后清零。与验证码一致，全部存进程内存，不依赖 Redis。
package loginlimiter

import (
	"sync"
	"time"
)

const (
	window  = 15 * time.Minute // 失败计数窗口
	maxFail = 3                // 窗口内输错满 3 次触发验证码
)

type entry struct {
	fails []time.Time
}

var (
	mu    sync.Mutex
	store = map[string]*entry{}
)

// NeedCaptcha 该账号当前是否已触发验证码门槛
func NeedCaptcha(name string) bool {
	mu.Lock()
	defer mu.Unlock()
	e := store[name]
	if e == nil {
		return false
	}
	sweepLocked(time.Now())
	return len(e.fails) >= maxFail
}

// Fail 记录一次密码错误
func Fail(name string) {
	mu.Lock()
	defer mu.Unlock()
	now := time.Now()
	sweepLocked(now)
	e := store[name]
	if e == nil {
		e = &entry{}
		store[name] = e
	}
	e.fails = append(e.fails, now)
}

// Success 登录成功，清零该账号的失败记录
func Success(name string) {
	mu.Lock()
	delete(store, name)
	mu.Unlock()
}

// sweepLocked 丢弃窗口外的失败记录，顺带清掉已无窗口内失败的账号（调用方需已持锁）
func sweepLocked(now time.Time) {
	cut := now.Add(-window)
	for k, e := range store {
		kept := e.fails[:0]
		for _, t := range e.fails {
			if t.After(cut) {
				kept = append(kept, t)
			}
		}
		e.fails = kept
		if len(e.fails) == 0 {
			delete(store, k)
		}
	}
}
