package ezfy

import (
	"strings"
	"testing"
	"time"

	"qqjiayuan/server/internal/model"
)

// ============ 活动目标「每城每日攻打上限」守卫（★ 2026-10-30 用户需求）============
//
// 用户要求：「活动目标每个城市每天可以打 50 次，二战系统配置可调；超过后该活动目标
// 按普通野地处理」（脚本反复刷活动资源）。
//
// 口径（唯一实现见 ezfy_activity_target.go）：
//   - 按**出发城市** + **自然日**统计掠夺(2)/征服(3)订单，侦查(1)不占次数；
//   - 默认 50，0 = 不限；
//   - 下单时把 ActType 固化在订单上；到达时按订单 id 排名二次卡控，
//     防止并发下单同时穿过创建时的读校验；
//   - 次数用完后地图/详情不再标活动，战斗走普通野地/寇城结算。
//
// 本文件守住：纯判定、配置 0 语义、下单/到达/展示接线、管理端字段与补列。

// TestActDailyLimitHelpers 纯判定：默认值、0 = 不限、自然日边界。
func TestActDailyLimitHelpers(t *testing.T) {
	if ezfyActDailyLimitDef != 50 {
		t.Fatalf("默认每日上限应为 50，实际 %d", ezfyActDailyLimitDef)
	}
	cases := []struct {
		limit int
		rank  int64
		want  bool
	}{
		{50, 1, true},
		{50, 50, true},
		{50, 51, false},
		{1, 1, true},
		{1, 2, false},
		{0, 1, true},
		{0, 999999, true}, // 0 = 不限
		{-1, 999999, true},
	}
	for _, c := range cases {
		if got := ezfyActDailyAllow(c.limit, c.rank); got != c.want {
			t.Errorf("ezfyActDailyAllow(%d, %d) = %v，期望 %v", c.limit, c.rank, got, c.want)
		}
	}

	// 自然日 [00:00, 次日 00:00)，服务器本地时区
	now := time.Date(2026, 10, 30, 15, 4, 5, 0, time.Local).UnixMilli()
	start, end := ezfyActDailyRange(now)
	wantStart := time.Date(2026, 10, 30, 0, 0, 0, 0, time.Local).UnixMilli()
	wantEnd := time.Date(2026, 10, 31, 0, 0, 0, 0, time.Local).UnixMilli()
	if start != wantStart || end != wantEnd {
		t.Fatalf("自然日区间错误: got [%d, %d)，期望 [%d, %d)", start, end, wantStart, wantEnd)
	}
	// 边界：前一天最后一毫秒属于前一天，当天 0 点属于当天
	if s, _ := ezfyActDailyRange(wantStart - 1); s != wantStart-24*time.Hour.Milliseconds() {
		t.Fatalf("前一天最后一毫秒应落到前一天，实际 start=%d", s)
	}
	if s, _ := ezfyActDailyRange(wantStart); s != wantStart {
		t.Fatalf("当天 0 点应落到当天，实际 start=%d", s)
	}
	if s, _ := ezfyActDailyRange(wantEnd); s != wantEnd {
		t.Fatalf("次日 0 点应落到次日，实际 start=%d", s)
	}
}

// TestActDailyLimitConfigZeroMeansUnlimited 配置读取：0 是「不限」，负数/未加载才回落 50。
func TestActDailyLimitConfigZeroMeansUnlimited(t *testing.T) {
	oldLoaded := ezfyCfg.loaded
	oldLimit := ezfyCfg.limit.ActDailyLimit
	defer func() {
		ezfyCfg.loaded = oldLoaded
		ezfyCfg.limit.ActDailyLimit = oldLimit
	}()

	ezfyCfg.loaded = true
	for _, c := range []struct {
		cfg  int
		want int
	}{
		{50, 50},
		{1, 1},
		{0, 0}, // 0 = 不限，必须原样返回，不能当「未配置」回落 50
		{-1, 50},
	} {
		ezfyCfg.limit.ActDailyLimit = c.cfg
		if got := ezfyActDailyLimit(); got != c.want {
			t.Errorf("ActDailyLimit=%d 时 ezfyActDailyLimit()=%d，期望 %d", c.cfg, got, c.want)
		}
	}

	// 配置缓存没加载过 → 回落默认 50（不能读零值当成「不限」）
	ezfyCfg.loaded = false
	if got := ezfyActDailyLimit(); got != ezfyActDailyLimitDef {
		t.Fatalf("配置未加载时应回落默认 %d，实际 %d", ezfyActDailyLimitDef, got)
	}
}

// TestActOrderTypeResolution 订单活动类型：新单以 ActType 为准，存量单才按坐标兜底。
func TestActOrderTypeResolution(t *testing.T) {
	h := &EzfyHandler{}
	if got := h.ezfyOrderActType(1, nil); got != 0 {
		t.Fatalf("nil 订单应返回 0，实际 %d", got)
	}
	// 新订单：0 就是 0（次数用完后按普通目标处理），不能被坐标重新判成活动
	if got := h.ezfyOrderActType(1, &model.EzfyOrder{ActType: 0, OrderType: 2, TargetX: 1, TargetY: 1}); got != 0 {
		t.Fatalf("ActType=0 的订单必须按普通目标处理，实际 %d", got)
	}
	if got := h.ezfyOrderActType(1, &model.EzfyOrder{ActType: ezfyActCity, OrderType: 3}); got != ezfyActCity {
		t.Fatalf("ActType=%d 的订单应原样返回，实际 %d", ezfyActCity, got)
	}
	// 存量单（-1）也不必为采集/运输类查询坐标
	if got := h.ezfyOrderActType(1, &model.EzfyOrder{ActType: -1, OrderType: 5}); got != 0 {
		t.Fatalf("采集/运输订单不应判成活动目标，实际 %d", got)
	}
	// 存量在途订单不做配额追溯（保持部署瞬间的原行为）
	if !h.ezfyActOrderWithinDailyLimit(&model.EzfyOrder{ActType: -1, OrderType: 2}) {
		t.Fatal("存量订单 ActType=-1 不应被每日上限追溯拦截")
	}
}

// TestActDailyCountScope 计数口径：按城 + 自然日，只数掠夺/征服，侦查不占次数。
func TestActDailyCountScope(t *testing.T) {
	used := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_activity_target.go", "func (h *EzfyHandler) ezfyActDailyUsed("), "\r\n", "\n"))
	for _, want := range []string{
		"city_id = ?",
		"act_type > 0",
		"order_type IN (?, ?)",
		"start_time >= ? AND start_time < ?",
		"cityID, 2, 3, start, end",
		// ★★ 2026-10-30 名将活动目标不占配额 → 计数里必须跳过
		"h.ezfyActNamedTargetAt(rows[i].TargetX, rows[i].TargetY)",
	} {
		if !strings.Contains(used, want) {
			t.Fatalf("ezfyActDailyUsed 的计数口径缺少 %q：\n%s", want, used)
		}
	}

	rank := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_activity_target.go", "func (h *EzfyHandler) ezfyActDailyOrderRank("), "\r\n", "\n"))
	for _, want := range []string{"city_id = ?", "act_type > 0", "order_type IN (?, ?)", "id <= ?",
		"h.ezfyActNamedTargetAt(rows[i].TargetX, rows[i].TargetY)"} {
		if !strings.Contains(rank, want) {
			t.Fatalf("ezfyActDailyOrderRank 的排名口径缺少 %q：\n%s", want, rank)
		}
	}
}

// TestActDailyLimitCreateOrder 下单时必须固化 ActType，并在活动目标上检查本城配额。
func TestActDailyLimitCreateOrder(t *testing.T) {
	body := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) createOrder("), "\r\n", "\n"))

	for _, want := range []string{
		"ActType: actType",
		"h.ezfyActTargetType(targetX, targetY)",
		"h.ezfyActDailyLimitReached(int64(city.ID))",
		"!h.playerOwnsActWildGeneral(uid, targetX, targetY)",
		"targetType != 3",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("createOrder 缺少每日上限关键逻辑 %q：\n%s", want, body)
		}
	}
	// 必须先算 actType，再构造订单（否则订单永远落 0）
	iCalc := strings.Index(body, "actType := 0")
	iOrder := strings.Index(body, "order := model.EzfyOrder{")
	if iCalc < 0 || iOrder < 0 || iCalc > iOrder {
		t.Fatal("createOrder 必须先按每日上限计算 actType，再写入订单")
	}
	// 侦查(1)也要参与活动判定（次数用完后按普通野地回报），但不能占掠夺/征服的配额。
	if !strings.Contains(body, "orderType == 1 || orderType == 2 || orderType == 3") {
		t.Fatal("createOrder 的活动判定应覆盖侦查/掠夺/征服")
	}
}

// TestActDailyLimitArrivalRecheck 到达时必须按订单固化类型结算，并做 id 排名二次卡控。
func TestActDailyLimitArrivalRecheck(t *testing.T) {
	body := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) processArrive("), "\r\n", "\n"))

	for _, want := range []string{
		"act := h.ezfyOrderActType(uid, order)",
		"h.ezfyActOrderWithinDailyLimit(order)",
		"order.ActType = 0",
		`Update("act_type", 0)`,
		"h.processActivityBattle(uid, city, order, now, act)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("processArrive 缺少到达二次卡控 %q：\n%s", want, body)
		}
	}
	// 二次卡控必须在活动战斗结算之前
	iGate := strings.Index(body, "h.ezfyActOrderWithinDailyLimit(order)")
	iBattle := strings.Index(body, "h.processActivityBattle(uid, city, order, now, act)")
	if iGate < 0 || iBattle < 0 || iGate > iBattle {
		t.Fatal("每日上限二次卡控必须在 processActivityBattle 之前")
	}
}

// TestActDailyLimitDisplay 地图/详情/队列/军情都以「订单 ActType + 本城配额」为准。
func TestActDailyLimitDisplay(t *testing.T) {
	mapBody := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) MapView("), "\r\n", "\n"))
	for _, want := range []string{
		"actDailyBlocked()",
		// ★★ 2026-10-30 名将活动目标不卡每日上限 → 配额用完也照常显示为活动目标
		"h.ezfyActNamedTargetAt(x, y) || !actDailyBlocked()",
	} {
		if !strings.Contains(mapBody, want) {
			t.Fatalf("MapView 缺少配额展示拦截 %q", want)
		}
	}

	detailBody := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) WildlandView("), "\r\n", "\n"))
	for _, want := range []string{
		"if act := h.ezfyActTargetType(x, y); act > 0",
		// ★★ 名将活动目标：次数用完也展示活动详情（先于配额判定返回）
		"if h.ezfyActNamedTargetAt(x, y) {",
		"h.ezfyActDailyLimitReached(int64(curCity.ID))",
	} {
		if !strings.Contains(detailBody, want) {
			t.Fatalf("WildlandView 缺少配额展示拦截 %q", want)
		}
	}

	orderList := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) OrderList("), "\r\n", "\n"))
	if !strings.Contains(orderList, `"act_type"`) ||
		!strings.Contains(orderList, "h.ezfyOrderActType(uid, &o)") {
		t.Fatal("出征队列必须按订单固化的 ActType 下发活动标识")
	}

	dynamics := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_api.go", "func (h *EzfyHandler) ReportDynamics("), "\r\n", "\n"))
	if !strings.Contains(dynamics, "actType = h.ezfyOrderActType(uid, o)") {
		t.Fatal("军队动态必须按订单固化的 ActType 下发活动标识，不能按坐标重判")
	}
	if strings.Contains(dynamics, "h.ezfyActTargetType(o.TargetX, o.TargetY)") {
		t.Fatal("军队动态不能再用坐标重判活动目标（次数用完后在途单会漂移）")
	}
}

// TestActDailyLimitModelAndSeed 模型/补列：默认 50、0 合法、订单 ActType 不能挂 GORM default。
func TestActDailyLimitModelAndSeed(t *testing.T) {
	src := strings.ReplaceAll(readFrontFile(t, "../../model/ezfy.go"), "\r\n", "\n")
	if !strings.Contains(src, `gorm:"default:50;comment:每城每日活动目标攻击上限(0=不限)"`) {
		t.Fatal("model/ezfy.go 缺少 ActDailyLimit 默认 50 的字段")
	}
	// ActType 必须显式写 0（次数用完的单）→ 不能带 gorm default 标签
	orderStruct := ezfyFuncBody(t, "../../model/ezfy.go", "type EzfyOrder struct {")
	hasActType := false
	for _, line := range strings.Split(orderStruct, "\n") {
		if !strings.Contains(line, "ActType") || !strings.Contains(line, "gorm:") {
			continue
		}
		hasActType = true
		if !strings.Contains(line, `comment:活动类型 -1未知`) {
			t.Fatalf("EzfyOrder.ActType 字段注释不对：%s", strings.TrimSpace(line))
		}
		if strings.Contains(line, "default:") {
			t.Fatalf("ActType 不能挂 gorm default（0 会被跳过、无法落库）：%s", strings.TrimSpace(line))
		}
	}
	if !hasActType {
		t.Fatal("model/ezfy.go 缺少 EzfyOrder.ActType 字段")
	}

	seedSrc := readFrontFile(t, "../../seed/seed.go")
	for _, want := range []string{
		"ALTER TABLE ezfy_cfg_limit ADD COLUMN act_daily_limit int DEFAULT 50",
		"UPDATE ezfy_cfg_limit SET act_daily_limit = 50 WHERE act_daily_limit IS NULL",
		"ALTER TABLE ezfy_order ADD COLUMN act_type int NOT NULL DEFAULT -1",
		"UPDATE ezfy_order SET act_type = -1 WHERE act_type IS NULL",
		// ★ 通用补列（AutoMigrate）会建出**可空**列 → 历史行 NULL 会让 GORM 扫 int 直接报错，
		//   必须一次性回填 + 收紧约束（带 settings 标记，只跑一次）。
		"func ezfyOrderActTypeBackfillOnce(",
		"ezfyOrderActTypeBackfillOnce(db)",
		"MODIFY COLUMN act_type int NOT NULL DEFAULT -1",
		// ★ 每日上限的计数按 (city_id, start_time) 过滤 → 没索引就每次全表扫订单表
		"ALTER TABLE ezfy_order ADD KEY idx_city_start (city_id, start_time)",
	} {
		if !strings.Contains(seedSrc, want) {
			t.Fatalf("seed/seed.go 缺少补列/回填 %q", want)
		}
	}
}

// TestActDailyLimitAdminPlumbing 管理端：配置收发、0 校验、显式落库、前端字段缺一不可。
func TestActDailyLimitAdminPlumbing(t *testing.T) {
	admin := strings.ReplaceAll(readFrontFile(t, "ezfy_admin_round9.go"), "\r\n", "\n")
	for _, want := range []string{
		"ActDailyLimit: ezfyActDailyLimitDef",
		"ActDailyLimit *int `json:\"act_daily_limit\"`",
		"if *in.ActDailyLimit < 0 || *in.ActDailyLimit > 100000",
		"ALTER TABLE ezfy_cfg_limit ADD COLUMN act_daily_limit",
		`"act_daily_limit": lim.ActDailyLimit`,
	} {
		if !strings.Contains(admin, want) {
			t.Fatalf("ezfy_admin_round9.go 缺少每日上限配置收发 %q", want)
		}
	}
	// 0 = 不限是合法值 → GET 兜底只能拦负数
	adminSrc := ezfyStripCommentLines(admin)
	if !strings.Contains(adminSrc, "if lim.ActDailyLimit < 0") {
		t.Fatal("AdminEzfyBuildLimitGet 只能把负数回落默认，0 必须原样返回（不限）")
	}

	vue := strings.ReplaceAll(readFrontFile(t,
		"../../../../admin-web/src/components/admin/AdminEzfyBuildLimit.vue"), "\r\n", "\n")
	for _, want := range []string{
		"form.act_daily_limit",
		"act_daily_limit: 50",
		"act_daily_limit: '每座城市每天可攻击",
		"'act_daily_limit', 'move_cool_h'",
		"act_daily_limit: (r.data.act_daily_limit === undefined || r.data.act_daily_limit === null)",
	} {
		if !strings.Contains(vue, want) {
			t.Fatalf("AdminEzfyBuildLimit.vue 缺少每日上限字段 %q", want)
		}
	}
	// 0 = 不限，前端 min 必须是 0（不能从 1 起）
	if !strings.Contains(vue, `v-model.number="form.act_daily_limit" :min="0"`) {
		t.Fatal("活动目标每日上限输入框 min 必须是 0（0 = 不限）")
	}
}

// TestActNamedTargetNotLimitedByDailyLimit 名将活动目标（守将是**名将**）**不占也不受**每日上限。
//
// ★★ 2026-10-30 用户规则：「名将野地活动不要卡活动目标每日上限(次/城)，
// 名将只卡原来的 —— 抓到了最大次数才会变成普通地形」。
// ★ 用户补充：「仅仅是配置了名将的守将的」→ 只认守将 kind=2（名将）；
// 配普通军官(kind=1)的活动野地仍按普通活动目标受每日上限约束。
//
// 口径：
//   - 名将订单不计入当日配额（不挤掉普通活动目标的次数）；
//   - 名将订单到达时不被配额拦下（只受 max_capture / playerOwnsActWildGeneral 约束）；
//   - 地图/详情在本城次数用完后，名将格仍按活动目标展示。
func TestActNamedTargetNotLimitedByDailyLimit(t *testing.T) {
	// ① 名将判定：坐标级配置的守将**是名将**才算
	oldActWilds, oldGenerals := ezfyCfg.actWilds, ezfyCfg.generals
	ezfyCfg.mu.Lock()
	ezfyCfg.actWilds = map[int64]*model.EzfyActWild{
		ezfyTileKey(7, 7): {X: 7, Y: 7, Enabled: 1, Level: 1, OfficerId: 42}, // 名将
		ezfyTileKey(8, 8): {X: 8, Y: 8, Enabled: 1, Level: 1, OfficerId: 43}, // 普通军官
		ezfyTileKey(9, 9): {X: 9, Y: 9, Enabled: 1, Level: 1, OfficerId: 0},  // 无守将
	}
	ezfyCfg.mu.Unlock()
	ezfyCfg.generals = map[int]model.EzfyCfgGeneral{
		42: {ID: 42, Name: "名将甲", Kind: 2},
		43: {ID: 43, Name: "普通军官乙", Kind: 1},
	}
	defer func() {
		ezfyCfg.mu.Lock()
		ezfyCfg.actWilds = oldActWilds
		ezfyCfg.mu.Unlock()
		ezfyCfg.generals = oldGenerals
	}()

	h := &EzfyHandler{}
	if !h.ezfyActNamedTargetAt(7, 7) {
		t.Fatal("守将是名将(kind=2)的活动野地应判为名将活动目标")
	}
	if h.ezfyActNamedTargetAt(8, 8) {
		t.Fatal("守将是普通军官(kind=1)的活动野地不应判为名将活动目标（仍受每日上限）")
	}
	if h.ezfyActNamedTargetAt(9, 9) {
		t.Fatal("没配守将的活动野地不应判为名将活动目标")
	}
	// ② 名将订单到达时直接放行（不查库、不看配额）
	if !h.ezfyActOrderWithinDailyLimit(&model.EzfyOrder{
		ActType: ezfyActWild, OrderType: 2, TargetX: 7, TargetY: 7}) {
		t.Fatal("名将活动目标的订单不应被每日上限拦下")
	}

	// ③ 接线：下单/地图/详情都要给名将开口子
	create := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) createOrder("), "\r\n", "\n"))
	if !strings.Contains(create, "h.ezfyActNamedTargetAt(targetX, targetY) || !h.ezfyActDailyLimitReached") {
		t.Fatal("createOrder 里名将活动目标必须绕过每日上限")
	}
	gate := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_activity_target.go", "func (h *EzfyHandler) ezfyActOrderWithinDailyLimit("), "\r\n", "\n"))
	if !strings.Contains(gate, "if h.ezfyActNamedTargetAt(order.TargetX, order.TargetY) {") {
		t.Fatal("ezfyActOrderWithinDailyLimit 必须对名将活动目标短路放行")
	}
	// 名将判定必须复用 isGeneral（kind==2），不能只看「配了守将」
	named := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_activity_target.go", "func (h *EzfyHandler) ezfyActNamedTargetAt("), "\r\n", "\n"))
	if !strings.Contains(named, "ezfyCfg.isGeneral(oid)") {
		t.Fatal("名将判定必须用 isGeneral（kind==2），否则普通军官守将也会被放过")
	}
}

// TestActNamedGeneralSwitchesToNormalOnlyWhenFull 名将野地「抓满最大次数」才变普通地形。
//
// ★★ 2026-10-30 用户规则：「名将只卡原来的 —— 抓到了最大次数才会变成普通地形」。
//
//	原来 playerOwnsActWildGeneral 是「拥有即变普通地形」→ max_capture=3 时
//	抓到第 1 只该坐标就变普通野地，剩下 2 次永远用不掉。
//	现在名将（kind=2）且 max_capture>0 时按抓取流水条数判抓满；其余保持原口径。
func TestActNamedGeneralSwitchesToNormalOnlyWhenFull(t *testing.T) {
	body := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_activity_target.go", "func (h *EzfyHandler) playerOwnsActWildGeneral("), "\r\n", "\n"))
	for _, want := range []string{
		"g.Kind == 2 && aw.MaxCapture > 0", // 只对名将 + 配了次数
		"ezfyActCaptureFull(aw.MaxCapture, used)",
		"user_id = ? AND general_id = ?",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("playerOwnsActWildGeneral 缺少「抓满才变普通地形」逻辑 %q：\n%s", want, body)
		}
	}
	// ★ 地图渲染逐格调用 → 绝不能调 ezfyActCaptureUsed（无流水时它会**补记流水**，等于写库）
	if strings.Contains(body, "h.ezfyActCaptureUsed(") {
		t.Fatal("playerOwnsActWildGeneral 不能调 ezfyActCaptureUsed（会补记流水 = 写库），只读 count")
	}
}

// TestActDailyLimitAdminCopy 活动目标配置页文案已简化，且「0 = 全部」替换两处旧长文案。
func TestActDailyLimitAdminCopy(t *testing.T) {
	vue := strings.ReplaceAll(readFrontFile(t,
		"../../../../admin-web/src/components/admin/AdminEzfyMap.vue"), "\r\n", "\n")
	if !strings.Contains(vue, "维护活动野地 / 活动寇城 / 特殊城市的守军与奖励；坐标级活动野地配置优先，守军留空用内置默认。") {
		t.Fatal("活动目标配置页缺少简化后的说明文案")
	}
	if strings.Contains(vue, "维护三类活动目标") || strings.Contains(vue, "等级填 0 = 不限") {
		t.Fatal("活动目标配置页仍残留旧的长文案")
	}
	if strings.Count(vue, "0 = 全部") < 2 {
		t.Fatal("两处等级筛选都应显示「0 = 全部」")
	}
}
