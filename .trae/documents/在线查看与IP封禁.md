# 管理端「在线查看」+ IP 封禁（封禁后跳转「服务不可用」页）

## 一、需求

> 管理系统 不是有个在线查看统计，要能点击；点击后跳转查看在线用户或者是在线游客简单信息、以及 ip；可进行封禁，封禁后让用户重定向到默认的服务不可用网页。

拆成 4 件事：

1. 概览页「当前在线」统计卡片可点击 → 跳到「在线查看」页
2. 在线查看页：在线用户 + 在线游客的简单信息 + IP
3. 对 IP 封禁 / 解封
4. 被封 IP 访问站点 → 302 重定向到「服务不可用」页

## 二、现状分析（已核对源码）

| 关注点 | 现状 | 位置 |
| --- | --- | --- |
| 概览「当前在线」 | 纯 `div`，不可点击；数字 = 10 分钟内活跃**登录用户**（不含游客） | [Dashboard.vue](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/admin-web/src/views/Dashboard.vue#L18-L24)、[admin.go](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/handler/admin.go#L21-L29) |
| 管理端页面分发 | `Dashboard.vue` 按 `$route.query.tab` 从 `menu.js` 的 `pageMap` 取组件 | [Dashboard.vue](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/admin-web/src/views/Dashboard.vue#L44)、[menu.js](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/admin-web/src/menu.js#L290-L302) |
| 用户端在线列表 | 已有 `GET /api/online`：用户 + 游客混排、30 分钟窗口、10 条/页 | [plaza.go](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/handler/plaza.go#L211-L261) |
| 在线游客表 | `online_guests(ip PK, last_active_at)`，只在 `PlazaHandler.Index` 写入（游客 30 分钟滑动窗口） | [online_guest.go](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/model/online_guest.go)、[plaza.go](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/handler/plaza.go#L42-L47) |
| 登录用户 IP | `users.last_ip`（登录时写入，`json:"-"` 不外泄） | [auth.go](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/handler/auth.go#L160) |
| 权限 | RBAC `module:*` + `RequirePerm`；`modulePerms` 里加一条即自动授予 `admin` 角色 | [rbac.go](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/middleware/rbac.go)、[seed.go](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/seed/seed.go#L1398-L1460) |
| 全局中间件挂载点 | `r.Use(middleware.CORS())`（第 19 行），`db` 已在 `Setup` 入参里 | [router.go](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/router/router.go#L17-L19) |
| 静态页托管 | 生产：`NoRoute` 对含 `.` 的路径先 `os.Stat(WebDir+path)` 命中即返回（`/503.html` 可被直接命中）；管理端 `r.Static("/admin-ui", AdminWebDir)` | [router.go](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/router/router.go#L1440-L1477) |
| 前端 public 目录 | `web/public/`、`admin-web/public/` 均已存在（各有 `index.html`） | — |
| dev 代理 | web(8000) 代理 `/api`+`/admin-ui`；admin-web(8001) 代理 `/api`+`/static`；admin-web `publicPath=/admin-ui/` | [vue.config.js](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/web/vue.config.js)、[admin-web/vue.config.js](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/admin-web/vue.config.js) |
| axios 拦截器 | 成功分支直接 `resp => resp.data`；拿到 HTML 字符串时会静默返回字符串 | [web/src/api/index.js](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/web/src/api/index.js)、[admin-web/src/api/index.js](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/admin-web/src/api/index.js) |

**结论**：项目里**没有**任何 IP 封禁表 / 中间件，也**没有**「服务不可用」页，这两块需要新建；在线数据、权限体系、菜单分发、静态托管都可直接复用现有模式。

## 三、已确认的设计决策

| 决策点 | 选择 |
| --- | --- |
| 封禁生效范围 | **全站封禁（含管理端）** |
| 被封 IP 的响应 | **全部 302 重定向**（页面请求与 `/api/*` 一视同仁） |
| 概览「当前在线」口径 | **与在线查看页对齐**：登录用户 + 游客，30 分钟窗口 |
| 封禁期限 | 永久，直到手动解封（无到期时间字段） |
| 503 页实现 | 静态 HTML，放在 `web/public/503.html` 与 `admin-web/public/503.html`（自包含内联样式，不依赖任何外部资源） |

### 重定向目标规则（关键）

| 请求路径 | 重定向到 |
| --- | --- |
| `/503.html`、`/admin-ui/503.html` | **放行**（防死循环） |
| 以 `/admin-ui` 或 `/api/admin` 开头 | `/admin-ui/503.html` |
| 其余全部 | `/503.html` |

理由：admin-web 的 `publicPath` 是 `/admin-ui/`，dev(8001) 下 503 页在 `/admin-ui/503.html`；生产下 `r.Static("/admin-ui", ...)` 也命中同一路径。主站在两种模式下都在 `/503.html`。

### 已接受的副作用（必须在实现时写进 UI 提示）

「全站封禁 + 302」意味着：**如果把管理员自己当前所在 IP 封了，将无法再打开管理端，也无法调用解封接口**。恢复手段只有直接改数据库：

```sql
DELETE FROM ip_bans WHERE ip = '被误封的IP';
```

因此封禁按钮的确认弹窗必须明确写出这句话。同时**不做**「禁止封禁自己 IP」的后端硬校验（否则本机 `127.0.0.1` 环境下无法演示该功能）。

## 四、改动清单

### A. 后端

#### A1. 新增 `qqjiayuan/server/internal/model/ip_ban.go`

```go
package model

import "time"

// IPBan IP 封禁（管理端「在线查看」对在线用户/游客的 IP 一键封禁；存在记录即视为封禁）
type IPBan struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	IP        string    `gorm:"type:varchar(45);uniqueIndex:uk_ip_ban" json:"ip"`
	Reason    string    `gorm:"type:varchar(255)" json:"reason"`
	AdminID   uint      `json:"admin_id"`
	AdminName string    `gorm:"type:varchar(50)" json:"admin_name"`
	CreatedAt time.Time `json:"created_at"`
}

func (IPBan) TableName() string { return "ip_bans" }
```

#### A2. `qqjiayuan/server/internal/seed/seed.go`

- `AutoMigrate` 列表（第 86 行 `&model.OnlineGuest{}` 旁）加 `&model.IPBan{}`
- `modulePerms`（第 1401 行起）新增一条：`mod("概览", "在线查看", "online")`
  - `mod(group,name,key)` 会生成 `Code = "module:online"`；`admin` 角色通过 `moduleCodes` 循环自动获得，无需另改授权代码

#### A3. 新增 `qqjiayuan/server/internal/middleware/ipban.go`

要点：
- `atomic.Value` 缓存 `map[string]struct{}` 封禁 IP 集合，**30 秒**定时重载（避免每请求查库）
- 暴露 `Refresh()` 供封禁/解封后立即生效（否则要等 30 秒）
- `Handler()` 按上文「重定向目标规则」判定；命中即 `c.Redirect(http.StatusFound, target)` + `c.Abort()`
- 用 `c.ClientIP()` 取客户端 IP（与 `online_guests` 写入口径一致）

```go
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
```

#### A4. `qqjiayuan/server/internal/router/router.go`

1. 第 19 行 `r.Use(middleware.CORS())` 之后插入：

```go
	// 全站 IP 封禁（命中封禁名单的请求 302 到服务不可用页）
	banM := middleware.NewIPBan(db)
	r.Use(banM.Handler())
```

2. 第 34 行 `adminH := &handler.AdminHandler{DB: db}` 改为带上封禁器：

```go
	adminH := &handler.AdminHandler{DB: db, Ban: banM}
```

3. 管理端路由，紧挨第 849 行 `admin.GET("/stats", ...)` 之后新增：

```go
				// 在线查看 + IP 封禁
				admin.GET("/online", perm(db, "module:online"), adminH.OnlineView)
				admin.GET("/ip-bans", perm(db, "module:online"), adminH.IPBanList)
				admin.POST("/ip-bans", perm(db, "module:online"), adminH.IPBanAdd)
				admin.DELETE("/ip-bans/:id", perm(db, "module:online"), adminH.IPBanRemove)
```

#### A5. `qqjiayuan/server/internal/handler/admin.go`

1. `AdminHandler` 结构体加字段（需 `import "qqjiayuan/server/internal/middleware"`，该文件已 import）：

```go
type AdminHandler struct {
	DB  *gorm.DB
	Ban *middleware.IPBan
}
```

2. `Stats()` 的在线口径改为「用户 + 游客、30 分钟」：

```go
	halfHourAgo := time.Now().Add(-30 * time.Minute)
	var onlineUsers, onlineGuests int64
	h.DB.Model(&model.User{}).Where("last_active_at > ?", halfHourAgo).Count(&onlineUsers)
	h.DB.Model(&model.OnlineGuest{}).Where("last_active_at > ?", halfHourAgo).Count(&onlineGuests)
	online := onlineUsers + onlineGuests
```

3. 新增 `OnlineView`（用户 + 游客混排，按最后活跃倒序，分页；每行带 `banned`）：

```go
// OnlineView 在线查看：登录用户 + 在线游客混排（复刻用户端 /online 口径），带 IP 与封禁状态
func (h *AdminHandler) OnlineView(c *gin.Context) {
	page, offset, size := pageOf(c, 20)
	since := time.Now().Add(-30 * time.Minute)

	type uRow struct {
		UserID       uint
		Username     string
		Nickname     string
		Color        string
		IP           string
		LastActiveAt time.Time
	}
	var users []uRow
	h.DB.Model(&model.User{}).
		Select("id AS user_id, username, nickname, color, last_ip AS ip, last_active_at").
		Where("last_active_at > ?", since).Find(&users)
	var guests []model.OnlineGuest
	h.DB.Where("last_active_at > ?", since).Find(&guests)

	var bannedIPs []string
	h.DB.Model(&model.IPBan{}).Pluck("ip", &bannedIPs)
	banSet := map[string]bool{}
	for _, ip := range bannedIPs {
		banSet[ip] = true
	}

	type row struct {
		IsGuest      bool      `json:"is_guest"`
		UserID       uint      `json:"user_id"`
		Username     string    `json:"username"`
		Nickname     string    `json:"nickname"`
		Color        string    `json:"color"`
		IP           string    `json:"ip"`
		LastActiveAt time.Time `json:"last_active_at"`
		Banned       bool      `json:"banned"`
	}
	rows := make([]row, 0, len(users)+len(guests))
	for _, u := range users {
		rows = append(rows, row{UserID: u.UserID, Username: u.Username, Nickname: u.Nickname,
			Color: u.Color, IP: u.IP, LastActiveAt: u.LastActiveAt, Banned: banSet[u.IP]})
	}
	for _, g := range guests {
		rows = append(rows, row{IsGuest: true, IP: g.IP, LastActiveAt: g.LastActiveAt, Banned: banSet[g.IP]})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].LastActiveAt.After(rows[j].LastActiveAt) })
	total := len(rows)
	start, end := offset, offset+size
	if end > total { end = total }
	if start > end { start = end }
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": rows[start:end]})
}
```

> 需要 `import "sort"`（admin.go 现有 import：`math/rand`、`strconv`、`strings`、`time`、gin、bcrypt、gorm、middleware、model、resp）。

4. 新增封禁名单 3 个接口：

```go
// IPBanList 封禁名单（支持按 IP / 原因模糊搜索）
func (h *AdminHandler) IPBanList(c *gin.Context) {
	page, offset, size := pageOf(c, 20)
	word := c.Query("word")
	q := h.DB.Model(&model.IPBan{})
	if word != "" {
		q = q.Where("ip LIKE ? OR reason LIKE ?", "%"+word+"%", "%"+word+"%")
	}
	var total int64
	q.Count(&total)
	var list []model.IPBan
	q.Order("id DESC").Offset(offset).Limit(size).Find(&list)
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": list})
}

// IPBanAdd 封禁 IP（已存在则更新原因，不报错）
func (h *AdminHandler) IPBanAdd(c *gin.Context) {
	var req struct {
		IP     string `json:"ip" binding:"required"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "请填写要封禁的 IP")
		return
	}
	req.IP = strings.TrimSpace(req.IP)
	if req.IP == "" {
		resp.ParamError(c, "请填写要封禁的 IP")
		return
	}
	var admin model.User
	h.DB.Select("id, nickname, username").First(&admin, middleware.GetUID(c))
	name := admin.Nickname
	if name == "" {
		name = admin.Username
	}
	var old model.IPBan
	if err := h.DB.Where("ip = ?", req.IP).First(&old).Error; err == nil {
		h.DB.Model(&old).Updates(map[string]interface{}{"reason": req.Reason, "admin_id": admin.ID, "admin_name": name})
	} else {
		h.DB.Create(&model.IPBan{IP: req.IP, Reason: req.Reason, AdminID: admin.ID, AdminName: name})
	}
	if h.Ban != nil { h.Ban.Refresh() }
	resp.OK(c, gin.H{"ip": req.IP})
}

// IPBanRemove 解封（按 id）
func (h *AdminHandler) IPBanRemove(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var row model.IPBan
	if err := h.DB.First(&row, id).Error; err != nil {
		resp.NotFound(c, "封禁记录不存在")
		return
	}
	h.DB.Delete(&row)
	if h.Ban != nil { h.Ban.Refresh() }
	resp.OK(c, gin.H{"ip": row.IP})
}
```

### B. 前端（用户端 `web/`）

#### B1. 新增 `qqjiayuan/web/public/503.html`

自包含「服务不可用」页（内联 CSS、无外部资源、无「诺哈」字样）：

- 标题：`服务不可用`
- 主文案：`503 Service Unavailable` / `您的访问请求已被拒绝，无法继续访问本站。`
- 副文案：`如认为属于误封，请联系站点管理员处理。`
- 提供「刷新重试」按钮（`location.reload()`）

#### B2. `qqjiayuan/web/src/api/index.js`

在成功拦截器里加防呆：后端 302 后 axios 会跟随重定向并拿到 HTML 字符串，此时直接跳转到 503 页。

```js
api.interceptors.response.use(
  resp => {
    // 被 IP 封禁时后端 302 到 /503.html，axios 跟随重定向后拿到的是 HTML 字符串
    if (typeof resp.data === 'string' && /<html/i.test(resp.data)) {
      window.location.href = '/503.html'
      return { code: 503, msg: '服务不可用', data: null }
    }
    return resp.data
  },
  err => { /* 原逻辑不变 */ }
)
```

### C. 前端（管理端 `admin-web/`）

#### C1. 新增 `qqjiayuan/admin-web/public/503.html`

与 B1 同内容的副本（管理端 `publicPath=/admin-ui/`，该文件最终位于 `/admin-ui/503.html`）。

#### C2. `qqjiayuan/admin-web/src/api/index.js`

同上加防呆，跳转目标改为 `'/admin-ui/503.html'`。

#### C3. 新增 `qqjiayuan/admin-web/src/components/admin/AdminOnline.vue`

沿用 `AdminUsers.vue` 的骨架（`el-table` + `el-pagination` + `ElMessageBox` 确认 + `api.get/post/delete`）。两个区块：

**区块一：在线列表**
- 列：序号 / 类型（`用户` / `游客`）/ 名称（用户：`昵称(账号)`，可跳 `go('users')` 或直接展示文本；游客：`家园社区游客`）/ IP / 最后活跃时间 / 操作
- 操作列：`封禁` 按钮（已有 `banned` 时显示红色 `已封禁` 标签 + `解封` 按钮 → 需按 IP 查名单拿 id，或改为在弹窗内提示去下方名单解封）
  - **简化**：操作列只放「封禁」；解封统一在区块二的名单里做，避免按 IP 反查 id
- 顶部：`刷新` 按钮 + 总数提示
- 分页：`el-pagination`，`size=20`，`page` 存 `data`

**区块二：IP 封禁名单**
- 顶部：`word` 搜索框（IP/原因）+ `手动封禁` 按钮（弹窗填 IP + 原因）
- 列：ID / IP / 原因 / 操作人 / 封禁时间 / 操作（`解封`）
- 分页同上

**封禁确认弹窗文案（必须包含，防止自锁）**：

> 确定要封禁 IP `{ip}` 吗？
> 封禁后该 IP 将**无法访问全站（含本管理端）**，页面会跳转到「服务不可用」。
> 若误封了您自己的 IP，需在数据库执行：`DELETE FROM ip_bans WHERE ip='{ip}';`

#### C4. `qqjiayuan/admin-web/src/menu.js`

- 顶部 `import AdminOnline from './components/admin/AdminOnline.vue'`
- 菜单树中 `dashboard` 之后新增一条**顶级**菜单：

```js
  { key: 'online', name: '在线查看', icon: 'el-icon-view', component: AdminOnline, perm: 'module:online' },
```

（顶级项已有先例：`dashboard`。`pageMap` / `tabNames` 由文件末尾的 `walk()` 自动收集，无需另改。）

#### C5. `qqjiayuan/admin-web/src/views/Dashboard.vue`

「当前在线」卡片改为可点击：

```html
<div class="stat-card stat-card-link" @click="go('online')" title="点击查看在线用户/游客">
  <div class="ico ico-online"><i class="el-icon-cpu"></i></div>
  <div><div class="num">{{ stats.online || 0 }}</div><div class="lab">当前在线</div></div>
</div>
```

样式补 `.stat-card-link { cursor: pointer; }` 与 hover 反馈（`transform: translateY(-2px)` + 阴影，与 `.quick-item:hover` 一致）。

## 五、假设与取舍

1. **不做封禁到期时间**：需求未提及，永久封禁 + 手动解封即可。
2. **不做「禁止封禁自己 IP」后端校验**：本机环境只有 `127.0.0.1` 一个来源 IP，硬校验会让功能无法演示；改用弹窗强提示 + SQL 兜底。
3. **登录用户的 IP 取 `users.last_ip`**（登录时写入），非实时；这是项目现有口径（`online_guests` 才是实时 IP）。若需要更准，可后续在 `JWTAuth` 里同步刷新 `last_ip`，本次不做。
4. **在线查看用混排单列表**（复刻用户端 `/online`），不拆成两个 tab，也不加类型筛选。
5. **503 页做两份静态副本**（web / admin-web），因为两者 `publicPath` 与 dev server 端口不同；生产下分别落在 `/503.html` 与 `/admin-ui/503.html`。
6. **axios 防呆只加在拦截器**，不改业务代码；这是让「302 重定向」在 dev（页面由 dev server 直出、后端只拦到 `/api`）下也能真正落到 503 页的必要补充。

## 六、验证步骤

### 1. 编译 / 构建

```powershell
# 后端（主包在 server 根目录）
cd d:\mxz-code\github\codingce3gJy\qqjiayuan\server; go build -o server.exe .
# air 会自动重建；确认 8080 起来
# 用户端 / 管理端
cd d:\mxz-code\github\codingce3gJy\qqjiayuan\web; npm run build
cd d:\mxz-code\github\codingce3gJy\qqjiayuan\admin-web; npm run build
```

### 2. 权限落库检查

```sql
SELECT id, name, code FROM permissions WHERE code = 'module:online';
```

### 3. 接口冒烟（管理员 10000/admin123；注意登录字段是 `name`）

| 步骤 | 期望 |
| --- | --- |
| `GET /api/admin/stats` | `online` = 30 分钟内登录用户 + 游客 |
| `GET /api/admin/online` | 返回混排 `list`，含 `ip` / `is_guest` / `banned` |
| `POST /api/admin/ip-bans` `{"ip":"9.9.9.9","reason":"测试"}` | `code=0` |
| `GET /api/admin/ip-bans` | 名单含 9.9.9.9 |
| `DELETE /api/admin/ip-bans/:id` | `code=0`，名单清空 |

> 冒烟测试号：管理员用 **10000/admin123**（10007 无 `admin:access`）；PowerShell 发中文 JSON body 须用 `[System.Text.Encoding]::UTF8.GetBytes(...)`。

### 4. 重定向冒烟（**用伪造 IP，避免自锁**）

Gin 默认信任代理，`X-Forwarded-For` 会被 `ClientIP()` 采用：

```powershell
# 正常访问：200
curl.exe -i http://127.0.0.1:8080/api/health
# 先封 9.9.9.9（走管理端接口），再伪造该 IP 访问
curl.exe -i -H "X-Forwarded-For: 9.9.9.9" http://127.0.0.1:8080/
#   期望 302 + Location: /503.html
curl.exe -i -H "X-Forwarded-For: 9.9.9.9" http://127.0.0.1:8080/admin-ui/
#   期望 302 + Location: /admin-ui/503.html
curl.exe -i -H "X-Forwarded-For: 9.9.9.9" http://127.0.0.1:8080/503.html
#   期望 200（放行，无死循环）
curl.exe -i -H "X-Forwarded-For: 9.9.9.9" http://127.0.0.1:8080/api/online
#   期望 302（/api/* 同样被拦）
```

若 `X-Forwarded-For` 未被采用（`ClientIP()` 仍返回 127.0.0.1），则改用真封 `127.0.0.1` 验证，**验证后立刻执行**：

```powershell
mysql -h 127.0.0.1 -u root --password=1234567890 -D qq_jiayuan -e "DELETE FROM ip_bans WHERE ip='127.0.0.1'"
```

### 5. UI 验证

1. 管理端 `http://localhost:8001/admin-ui/` → 概览页「当前在线」卡片 hover 有反馈、可点击
2. 点击后进入 `?tab=online`「在线查看」，左侧菜单「在线查看」高亮
3. 列表展示在线用户/游客 + IP + 最后活跃时间；分页可用
4. 对某 IP 点「封禁」→ 确认弹窗出现（含自锁警告）→ 确定 → 该行变为「已封禁」
5. 「IP 封禁名单」区块出现该条记录；点「解封」→ 记录消失
6. 用户端 `http://localhost:8000/` 与 `/online` 仍正常（未被误封时）
