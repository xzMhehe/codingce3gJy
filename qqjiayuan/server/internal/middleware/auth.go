package middleware

import (
	"database/sql"
	"strings"
	"sync"
	"sync/atomic"
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

// badgeCleanLastMs 上次执行「过期勋章清理」的毫秒时间戳。
// ★ 2026-10-02 线上 1核1G CPU 100%：这里原本每个请求都跑一次
//   DELETE FROM user_badges ...（~8 次/秒），MySQL 饱和时单条被放大到 30ms+，
//   累计算得上行 25% 单核。改为 60 秒节流：过期勋章晚 1 分钟消失，玩家无感知。
var badgeCleanLastMs int64 = 0

// ★★ 2026-10-05 性能：活跃时间「本次进程内已刷新」备忘（uid → 毫秒时间戳）。
//
//	原来每个请求都先 `SELECT last_active_at FROM users WHERE id = ?` 再判断要不要写 ——
//	这条 SELECT 在**所有**接口上都是第 1 个跨 WAN 往返（60 秒节流只省了 UPDATE，没省 SELECT）。
//	既然节流窗口就是 1 分钟，那就直接备忘「这个 uid 本进程 1 分钟内已刷新过」，
//	窗口内连 SELECT 都省掉 —— 每个请求少 1 个 RDS 往返。
//	多机部署下各进程独立刷新（活跃时间只是统计用途，允许各机各记一次）。
var (
	lastActiveMemoMu sync.Mutex
	lastActiveMemo   = map[uint]int64{}
)

const lastActiveThrottleMs = 60 * 1000

// lastActiveRecently 本进程 1 分钟内是否已为该 uid 刷新过活跃时间。
func lastActiveRecently(uid uint, nowMs int64) bool {
	lastActiveMemoMu.Lock()
	defer lastActiveMemoMu.Unlock()
	at, ok := lastActiveMemo[uid]
	if ok && nowMs-at < lastActiveThrottleMs {
		return true
	}
	if len(lastActiveMemo) > 65536 { // 防无限增长（与其它进程内备忘同一策略）
		lastActiveMemo = map[uint]int64{}
	}
	lastActiveMemo[uid] = nowMs
	return false
}

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

// ★★ 2026-10-15 家园 token 服务端作废（logout 接口写 `users.session_at`，这里按
// `token 的 iat < session_at → 401 作废` 判掉线）。家园 token 无状态，本地清了不算数，
// 脚本拿着旧 token 还能打接口 —— 退出时服务端把 session_at 顶到当前，旧 token 即失效。
//
// 性能：家园请求也走 1核1G + RDS，这里同样用 60s 进程内缓存；logout 接口写库后调
// SetHomeSessionAt **立即刷新本机缓存**（多机部署时其他节点最迟 60s 生效，与二战侧同策略）。
// 注意：只在「退出」时写 session_at，登录不写 → 家园多设备同时登录不受影响。
var (
	homeSessionCacheMu   sync.Mutex
	homeSessionCache     = map[uint]int64{}
	homeSessionCacheDone = map[uint]time.Time{}
)

const homeSessionCacheTTL = 60 * time.Second

// homeSessionAtOf 取玩家「最近退出登录」时刻（0 = 从未退出），带进程内缓存。
func homeSessionAtOf(db *gorm.DB, uid uint) int64 {
	homeSessionCacheMu.Lock()
	if at, ok := homeSessionCache[uid]; ok && time.Since(homeSessionCacheDone[uid]) < homeSessionCacheTTL {
		homeSessionCacheMu.Unlock()
		return at
	}
	homeSessionCacheMu.Unlock()
	var at int64
	// 用 Raw 而不是 model.User：表列不存在时 at 保持 0 = 从未退出，不影响登录。
	db.Raw("SELECT session_at FROM users WHERE id = ?", uid).Scan(&at)
	homeSessionCacheMu.Lock()
	if len(homeSessionCache) > 65536 { // 防无限增长（与其它进程内备忘同一策略）
		homeSessionCache = map[uint]int64{}
		homeSessionCacheDone = map[uint]time.Time{}
	}
	homeSessionCache[uid] = at
	homeSessionCacheDone[uid] = time.Now()
	homeSessionCacheMu.Unlock()
	return at
}

// SetHomeSessionAt 退出登录后立即刷新本机缓存（logout 接口调用），保证本机即时生效。
func SetHomeSessionAt(uid uint, at int64) {
	homeSessionCacheMu.Lock()
	homeSessionCache[uid] = at
	homeSessionCacheDone[uid] = time.Now()
	homeSessionCacheMu.Unlock()
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

		// ★★ 2026-10-15 家园退出 → 家园 token 作废：token 签发时刻早于「最近退出登录时刻」→ 401。
		if claims.IssuedAt != nil {
			if sessionAt := homeSessionAtOf(db, claims.UserID); sessionAt > 0 && claims.IssuedAt.UnixMilli() < sessionAt {
				resp.Unauthorized(c, "账号已在其他设备退出登录，请重新登录")
				c.Abort()
				return
			}
		}

		// 节流更新活跃时间（10 分钟内活跃视为在线）
		// ★ 2026-10-05 性能：本进程 1 分钟内已刷新过 → 连 SELECT 都跳过（见 lastActiveRecently）。
		if !lastActiveRecently(claims.UserID, time.Now().UnixMilli()) {
			var last sql.NullTime
			db.Raw("SELECT last_active_at FROM users WHERE id = ?", claims.UserID).Scan(&last)
			if !last.Valid || time.Since(last.Time) > time.Minute {
				now := time.Now()
				db.Model(&struct{}{}).Table("users").
					Where("id = ?", claims.UserID).
					Updates(map[string]interface{}{"last_active_at": now, "last_login_at": now})
				// 家园活跃天数：每天首次活跃 +1，连续登录 +0.2，超Q/蓝钻在有效期内每晚一天等级 +0.1/级（封顶 +1.0）
				var lastDate sql.NullString
				db.Raw("SELECT last_active_date FROM users WHERE id = ?", claims.UserID).Scan(&lastDate)
				today := now.Format("2006-01-02")
				lastDateStr := ""
				if lastDate.Valid {
					lastDateStr = lastDate.String
				}
				if lastDateStr != today {
					base := 1.0
					if lastDateStr == now.AddDate(0, 0, -1).Format("2006-01-02") {
						base = 1.2
					}
					var qqLv, blueLv int
					db.Raw("SELECT IFNULL(qq_lv, 0) FROM users WHERE id = ? AND qq_end > NOW()", claims.UserID).Scan(&qqLv)
					db.Raw("SELECT IFNULL(blue_lv, 0) FROM users WHERE id = ? AND blue_end > NOW()", claims.UserID).Scan(&blueLv)
					base += 0.1*float64(qqLv) + 0.1*float64(blueLv)
					if base > 2.2 {
						base = 2.2
					}
					db.Exec("UPDATE users SET active_days = active_days + ?, last_active_date = ? WHERE id = ?", base, today, claims.UserID)
				}
			}
		}
		// 清理已过期会员勋章（复刻诺哈：过期勋章自动消失）。
		// ★ 60 秒节流：不再每个请求都 DELETE（见 badgeCleanLastMs 注释）。
		if now := time.Now().UnixMilli(); now-atomic.LoadInt64(&badgeCleanLastMs) >= 60000 {
			atomic.StoreInt64(&badgeCleanLastMs, now)
			db.Exec("DELETE FROM user_badges WHERE expire_at IS NOT NULL AND expire_at < NOW()")
		}
		c.Next()
	}
}

// EzfyJWTAuth 二战征途专属认证（★ 2026-10-12 账号体系与家园分离）。
//
//	只认二战签发的 token（Issuer=qqjiayuan-ezfy），家园 token 无法通过（反过亦然）。
//	校验通过后同样 set CtxUID/CtxUName —— 值就是「家园号(=二战游戏ID)」，
//	二战 handler 按 middleware.GetUID(c) 沿用既有查询，**零迁移**。
//	不写家园库（无活跃时间/勋章等家园逻辑），登录态彻底独立。
func EzfyJWTAuth(secret string, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			resp.Unauthorized(c, "请先登录")
			c.Abort()
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		claims, err := authutil.ParseEzfyToken(tokenStr, secret)
		if err != nil {
			resp.Unauthorized(c, "登录已过期，请重新登录")
			c.Abort()
			return
		}
		// ★ 2026-10-12 管理端「踢下线」 + ★★ 2026-10-15 单点登录（新登录顶掉旧登录）：
		// 两个时间戳合一查询（带 60s 进程内缓存），任一命中 → 401 作废旧 token。
		if db != nil && claims.IssuedAt != nil {
			kickAt, sessionAt := ezfySessionOf(db, claims.UserID)
			if kickAt > 0 && claims.IssuedAt.UnixMilli() <= kickAt {
				resp.Unauthorized(c, "账号已被强制下线，请重新登录")
				c.Abort()
				return
			}
			if sessionAt > 0 && claims.IssuedAt.UnixMilli() < sessionAt {
				resp.Unauthorized(c, "账号已在其他地点登录，请重新登录")
				c.Abort()
				return
			}
		}
		c.Set(CtxUID, claims.UserID)
		c.Set(CtxUName, claims.Nickname)
		c.Next()
	}
}

// ============ 二战「踢下线 + 单点登录」支持（★ 2026-10-12 / ★★ 2026-10-15）============
//
// 二战 token 是**无状态 JWT**，服务端没有会话可删，所以用 ezfy_account 里的两个时间戳作废旧 token：
//
//	kick_at    被踢时间戳(ms)：`token.iat <= kick_at` → 作废（管理端「踢下线」）。
//	session_at 最近签发 token 时刻(秒级×1000)：`token.iat < session_at` → 该 token 是旧登录
//	            签发的 → 作废（★★ 2026-10-15 单点登录：只能一个 IP 在线，其他地方登录前面必掉线）。
//
//	命中任一 → 拒绝（401，前端 axios 拦截器看到二战接口 401 会自动 ezfyLogout，玩家回到二战登录界面）。
//	玩家重新登录会签发新 token（iat >= session_at）→ 自然恢复，不需要额外的「解封」操作。
//
// 性能：本中间件跑在**每个二战请求**上，1核1G + RDS 跨 WAN 不能每次都查库 →
// 两个时间戳都走进程内缓存（TTL 60s）；登录/踢出接口写库后各自调 SetEzfySessionAt/SetEzfyKickAt
// **立即刷新本机缓存**，所以单机部署下即时生效（多机部署时其他节点最迟 60s 生效）。
var ezfyKickCache sync.Map // uid(uint) -> ezfyKickEntry

type ezfyKickEntry struct {
	kickAt    int64
	sessionAt int64
	at        time.Time
}

const ezfyKickCacheTTL = 60 * time.Second

// ezfySessionOf 取玩家「token 作废线」两项（kick_at / session_at），带进程内缓存。
func ezfySessionOf(db *gorm.DB, uid uint) (kickAt, sessionAt int64) {
	if v, ok := ezfyKickCache.Load(uid); ok {
		if e, ok2 := v.(ezfyKickEntry); ok2 && time.Since(e.at) < ezfyKickCacheTTL {
			return e.kickAt, e.sessionAt
		}
	}
	var row struct {
		KickAt    int64
		SessionAt int64
	}
	// 用 Raw 而不是 model.EzfyAccount：middleware 包不引 model，免得平添依赖（表不存在时两值保持 0）。
	db.Raw("SELECT IFNULL(kick_at,0) AS kick_at, IFNULL(session_at,0) AS session_at FROM ezfy_account WHERE game_id = ?", uid).Scan(&row)
	ezfyKickCache.Store(uid, ezfyKickEntry{kickAt: row.KickAt, sessionAt: row.SessionAt, at: time.Now()})
	return row.KickAt, row.SessionAt
}

// SetEzfyKickAt 踢下线后立即刷新缓存（管理端踢出接口调用），保证本机即时生效。
// 保留已缓存的 sessionAt（单点登录时刻），只更新 kick_at。
func SetEzfyKickAt(uid uint, kickAt int64) {
	sessionAt := int64(0)
	if v, ok := ezfyKickCache.Load(uid); ok {
		if e, ok2 := v.(ezfyKickEntry); ok2 {
			sessionAt = e.sessionAt
		}
	}
	ezfyKickCache.Store(uid, ezfyKickEntry{kickAt: kickAt, sessionAt: sessionAt, at: time.Now()})
}

// SetEzfySessionAt 登录签发新 token 后立即刷新缓存（单点登录：新登录顶掉旧登录），
// 保证本机即时生效。保留已缓存的 kickAt，只更新 session_at。
func SetEzfySessionAt(uid uint, sessionAt int64) {
	kickAt := int64(0)
	if v, ok := ezfyKickCache.Load(uid); ok {
		if e, ok2 := v.(ezfyKickEntry); ok2 {
			kickAt = e.kickAt
		}
	}
	ezfyKickCache.Store(uid, ezfyKickEntry{kickAt: kickAt, sessionAt: sessionAt, at: time.Now()})
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
