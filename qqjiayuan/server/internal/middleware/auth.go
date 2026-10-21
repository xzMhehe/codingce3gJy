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
	// CtxScope token 来源端（claims.Issuer）：authutil 的 homeIssuer / AdminIssuer / EzfyIssuer。
	// ★★ 2026-10-19 管理端与用户端 token 分离：Logout 靠它决定「顶哪条会话作废线」。
	CtxScope = "token_scope"

	// OnlineScopeHome 家园端在线来源标签（users.last_online_scope）。
	// 管理端「在线查看」按这个字段把家园与二战各区在线人数分开统计。
	OnlineScopeHome = "家园"
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

// touchHomeOnline 把「在线」写回家园库 users（二战服请求也走这里）。
//
// ★★ 2026-10-20 用户反馈「管理端在线查看不准、数量少多了」：
// 二战独立部署后 EzfyJWTAuth 原来**完全不写家园库**，玩家在二战里玩再久，
// users.last_active_at 也停在家园最后一次请求 —— 30 分钟后就从管理端在线列表消失，
// 连二战自己的 ezfyUserOnline（战斗是否进指挥室）也会误判玩家离线。
//
// 这里按家园 JWT 同款 60 秒节流更新 last_active_at + last_online_scope，
// 二战各区分别写自己的区服标题，管理端即可把「家园 / 1区 / 2区」分开统计。
// 只写活跃时间与来源，不碰登录时间/活跃天数（那些仍只由家园登录逻辑维护）。
func touchHomeOnline(db *gorm.DB, uid uint, ip, scope string) {
	if db == nil || uid == 0 {
		return
	}
	if lastActiveRecently(uid, time.Now().UnixMilli()) {
		return
	}
	updates := map[string]interface{}{
		"last_active_at":    time.Now(),
		"last_online_scope": scope,
	}
	// 顺手刷新最后 IP：在线查看里的「封禁」直接用它，避免二战玩家显示成旧的家园登录 IP。
	if ip != "" {
		updates["last_ip"] = ip
	}
	db.Model(&struct{}{}).Table("users").Where("id = ?", uid).Updates(updates)
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
// ★★ 2026-10-19 管理端/用户端 token 分离后，作废线有**两条**（互不影响）：
//
//	用户端 home token  → users.session_at        （本函数下方 homeSessionAtOf）
//	管理端 admin token → users.admin_session_at  （下方 adminSessionAtOf）
//
// 两条线各自只作废本端的 token：管理端退出不再把玩家端踢下线，玩家端登录也不再踢管理端。
//
// ★★ 2026-10-21 管理端允许多设备同时在线：管理端**登录不写** admin_session_at，
//   只有**退出**才写（写一次 = 该账号此前签发的全部管理端 token 一起失效）。
//   所以这一列的含义是「管理端最近一次退出时刻」，不再是「最近登录时刻」。
//
// 性能：家园请求也走 1核1G + RDS，这里同样用 60s 进程内缓存；logout 接口写库后调
// SetHomeSessionAt / SetAdminSessionAt **立即刷新本机缓存**（多机部署时其他节点最迟 60s 生效，
// 与二战侧同策略）。
// 注意：家园「退出」时写 session_at，「登录」也写（2026-10-18 单设备在线）→ 家园同一账号
// 只允许一处在线；管理端是另一条线，不受此限制。
var (
	homeSessionCacheMu   sync.Mutex
	homeSessionCache     = map[uint]int64{}
	homeSessionCacheDone = map[uint]time.Time{}
)

const homeSessionCacheTTL = 60 * time.Second

// ★★ 2026-10-18 二战跨区校验专用 TTL（5s）：
//   家园服的 JWT 中间件走 homeSessionCacheTTL(60s) 没问题 —— 家园 logout 会调
//   SetHomeSessionAt **即时刷新本进程缓存**。但二战一区/二区是**别的进程**，刷不到，
//   60s 内旧 token 仍然可用（用户：「退出家园，二战 token 也要失效」+「还有缓存问题」）。
//   这里给二战侧单独一个短 TTL；同一进程内只会用到其中一种，共享 map 不冲突。
const homeSessionCacheTTLCross = 5 * time.Second

// homeSessionAtOf 取玩家「最近退出登录」时刻（0 = 从未退出），带进程内缓存。
func homeSessionAtOf(db *gorm.DB, uid uint) int64 {
	return homeSessionAtWith(db, uid, homeSessionCacheTTL)
}

// homeSessionAtWith 同上，但可指定缓存 TTL（二战跨区校验用短 TTL）。
func homeSessionAtWith(db *gorm.DB, uid uint, ttl time.Duration) int64 {
	homeSessionCacheMu.Lock()
	if at, ok := homeSessionCache[uid]; ok && time.Since(homeSessionCacheDone[uid]) < ttl {
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

// ★★ 2026-10-19 管理端独立作废线（users.admin_session_at）。
//
//	与 homeSessionCache 完全独立的两份缓存/两条线：
//	  用户端登录/退出 → users.session_at       → 只作废 home token
//	  管理端退出      → users.admin_session_at → 只作废 admin token
//	所以「管理端退出 → 玩家端被迫下线」不再发生（反之亦然）。
//	管理端登录不写这一列（2026-10-21）：多个管理端会话可并存，避免后登录顶掉先登录。
var (
	adminSessionCacheMu   sync.Mutex
	adminSessionCache     = map[uint]int64{}
	adminSessionCacheDone = map[uint]time.Time{}
)

// adminSessionAtOf 取玩家「管理端最近退出」时刻（0 = 从未退出），带进程内缓存。
func adminSessionAtOf(db *gorm.DB, uid uint) int64 {
	adminSessionCacheMu.Lock()
	if at, ok := adminSessionCache[uid]; ok && time.Since(adminSessionCacheDone[uid]) < homeSessionCacheTTL {
		adminSessionCacheMu.Unlock()
		return at
	}
	adminSessionCacheMu.Unlock()
	var at int64
	// Raw 而非 model.User：列不存在时 at 保持 0 = 从未，不影响管理端登录。
	db.Raw("SELECT admin_session_at FROM users WHERE id = ?", uid).Scan(&at)
	adminSessionCacheMu.Lock()
	if len(adminSessionCache) > 65536 {
		adminSessionCache = map[uint]int64{}
		adminSessionCacheDone = map[uint]time.Time{}
	}
	adminSessionCache[uid] = at
	adminSessionCacheDone[uid] = time.Now()
	adminSessionCacheMu.Unlock()
	return at
}

// SetAdminSessionAt 管理端退出后立即刷新本机缓存，保证本机即时生效。
func SetAdminSessionAt(uid uint, at int64) {
	adminSessionCacheMu.Lock()
	adminSessionCache[uid] = at
	adminSessionCacheDone[uid] = time.Now()
	adminSessionCacheMu.Unlock()
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
		// ★★ 2026-10-19 管理端与用户端 token 分离：两类 token 都认（Issuer 不同），
		//   下面按 Issuer 分别校各自的会话作废线 —— 这是「管理端登录不踢用户端」的关键。
		claims, err := authutil.ParseAnyToken(tokenStr, secret)
		if err != nil {
			resp.Unauthorized(c, "登录已过期，请重新登录")
			c.Abort()
			return
		}
		c.Set(CtxUID, claims.UserID)
		c.Set(CtxUName, claims.Nickname)
		c.Set(CtxScope, claims.Issuer)

		// ★★ 2026-10-19 按端分流校验（两条线互不影响）：
		//   管理端 token → users.admin_session_at（管理端退出作废线；登录不写，可多设备并存）
		//   用户端 token → users.session_at      （家园退出/单设备在线的线）
		if claims.IssuedAt != nil {
			var sessionAt int64
			if claims.Issuer == authutil.AdminIssuer {
				sessionAt = adminSessionAtOf(db, claims.UserID)
			} else {
				sessionAt = homeSessionAtOf(db, claims.UserID)
			}
			if sessionAt > 0 && claims.IssuedAt.UnixMilli() < sessionAt {
				if claims.Issuer == authutil.AdminIssuer {
					resp.Unauthorized(c, "管理端登录状态已失效，请重新登录")
				} else {
					resp.Unauthorized(c, "账号已在其他设备退出登录，请重新登录")
				}
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
					Updates(map[string]interface{}{
						"last_active_at": now, "last_login_at": now,
						"last_online_scope": OnlineScopeHome,
					})
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
//	登录态独立；但仍按 60 秒节流把在线时间/来源写回家园库（见 touchHomeOnline），
//	否则管理端在线查看会漏掉全部二战玩家（2026-10-20 用户反馈「在线数量少多了」）。
//	onlineScope = 本区区服标题（如「二战征途-【1区】红色警戒」），管理端按它分栏统计。
func EzfyJWTAuth(secret string, db *gorm.DB, homeDB *gorm.DB, onlineScope string) gin.HandlerFunc {
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
		// ★★ 2026-10-18 跨区统一作废源：家园 `users.session_at`。
		//   为什么需要：家园退出（/api/auth/logout）只更新**它自己配置的那一个**二战库
		//   （config 的 ezfy_mysql.dbname，通常是 1 区库）→ 2 区的 ezfy_account.session_at
		//   根本不会被碰 → 2 区 token 能一直活到 JWT 自然过期（用户：「退出家园，二战 token
		//   也要失效」在多区下是漏的）。二战服本来就挂着 HomeDB（账号/昵称都读家园库），
		//   顺带校一下家园的作废线就覆盖了所有区。
		//   ★ 复用家园那套 homeSessionAtOf（60s 进程内缓存，与家园 JWT 中间件同口径）；
		//   前端 logout 会同时清掉 jy_token（cookie + localStorage），二战页刷新即回登录框，
		//   这里的 60s 只是「不刷新页面也能作废」的服务端兜底。
		if homeDB != nil && claims.IssuedAt != nil {
			if sat := homeSessionAtWith(homeDB, claims.UserID, homeSessionCacheTTLCross); sat > 0 && claims.IssuedAt.UnixMilli() < sat {
				resp.Unauthorized(c, "家园账号已退出，请重新登录")
				c.Abort()
				return
			}
		}
		// ★★ 2026-10-20 二战在线也要算进家园 users.last_active_at：
		//   否则管理端在线查看（家园库）看不到二战玩家，数量比实际少很多。
		touchHomeOnline(homeDB, claims.UserID, c.ClientIP(), onlineScope)
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

// ★★ 2026-10-18 从 60s 收到 5s：家园服 / 二战一区 / 二战二区是**三个独立进程**，
//   `SetEzfySessionAt` 只能刷新「自己这个进程」的缓存。玩家在家园点退出时，一区/二区
//   进程的缓存还是旧值 → 60s 内旧二战 token 仍然可用（用户反馈的「退了还有缓存」）。
//   5s 的代价：每用户每 5s 最多一次查询（只在有请求时查），远小于 60s 的错误窗口。
const ezfyKickCacheTTL = 5 * time.Second

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

func GetUID(c *gin.Context) uint {	v, _ := c.Get(CtxUID)
	uid, _ := v.(uint)
	return uid
}

func GetUName(c *gin.Context) string {
	v, _ := c.Get(CtxUName)
	s, _ := v.(string)
	return s
}
