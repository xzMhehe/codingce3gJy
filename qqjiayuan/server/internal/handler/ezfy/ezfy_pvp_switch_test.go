package ezfy

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestPvpRankSwitchGatesAccumulation —— 防回归：风云榜统计开关必须**在写榜入口拦截**。
//
// 背景（2026-10-14 用户需求）：「风云榜 tab 能够选择开启关闭，关闭就是不进行统计」。
// 关闭靠 ezfy_cfg_limit.pvp_rank_on = 0，读取点是 ezfyPvpRankOn()。
//
// 风险：日后有人在 accumPvpWeek 之外新增写榜路径（或重构时把门禁挪走），
// 关闭开关就形同虚设 —— 表现是「后台明明关了，榜单还在涨」，很难当场发现。
// 所以这里断言：
//  1. accumPvpWeek 函数体里出现 ezfyPvpRankOn() 且**早于**任何写库调用；
//  2. ezfyPvpRankOn 读的是缓存里 limit.PvpRankOn（不是零值结构体）。
//
// 纯静态扫描，不需要 DB。
func TestPvpRankSwitchGatesAccumulation(t *testing.T) {
	path := filepath.Join(".", "ezfy_pvp_rank.go")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读 %s 失败: %v", path, err)
	}
	s := string(src)

	// 1) 取 accumPvpWeek 函数体
	body := funcBody(s, "func (h *EzfyHandler) accumPvpWeek(")
	if body == "" {
		t.Fatal("找不到 accumPvpWeek 函数体（重构后请同步更新本断言）")
	}
	gate := strings.Index(body, "ezfyPvpRankOn()")
	if gate < 0 {
		t.Fatal("accumPvpWeek 里没有统计开关门禁（ezfyPvpRankOn()）—— 关闭开关将无效")
	}
	// 门禁必须早于第一条写榜语句
	for _, write := range []string{"accumPvpWeekOnce(", "INSERT INTO ezfy_pvp_week", "INSERT INTO ezfy_pvp_foe"} {
		if i := strings.Index(body, write); i >= 0 && i < gate {
			t.Fatalf("写榜语句 %q 出现在开关门禁之前 —— 关闭开关也会写榜", write)
		}
	}

	// 2) 开关读取必须走缓存 limit，且不是「零值误判」
	onBody := funcBody(s, "func ezfyPvpRankOn()")
	if onBody == "" {
		t.Fatal("找不到 ezfyPvpRankOn 函数体")
	}
	if !strings.Contains(onBody, "ezfyCfg.limit.PvpRankOn") {
		t.Fatal("ezfyPvpRankOn 未读 ezfyCfg.limit.PvpRankOn")
	}
	if !strings.Contains(onBody, "ezfyCfg.ready()") {
		t.Fatal("ezfyPvpRankOn 缺 ready() 保护 —— 配置未加载时会读到零值 0，把「开」误判成「关」")
	}
}

// TestPvpExcludeAppliedToRankQuery —— 防回归：「不参与排名」名单必须下推到榜单 SQL。
//
// 只在 Go 侧过滤（取完再筛）会导致「徽标有名次、列表被截断成不足 N 名」这类隐蔽 bug，
// 且名单变大后名次口径会漂移。这里断言过滤发生在 ORDER/LIMIT 之前的 WHERE 里。
func TestPvpExcludeAppliedToRankQuery(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(".", "ezfy_pvp_rank.go"))
	if err != nil {
		t.Fatalf("读 ezfy_pvp_rank.go 失败: %v", err)
	}
	body := funcBody(string(src), "func ezfyPvpRankRows(")
	if body == "" {
		t.Fatal("找不到 ezfyPvpRankRows 函数体")
	}
	excl := strings.Index(body, "user_id NOT IN ?")
	if excl < 0 {
		t.Fatal("ezfyPvpRankRows 未在 SQL 层排除「不参与排名」名单")
	}
	lim := strings.Index(body, ".Limit(limit)")
	if lim >= 0 && lim < excl {
		t.Fatal("LIMIT 出现在排除条件之前 —— 会先截断再过滤，导致榜单缺人")
	}
}

// TestPvpExcludeBoardsIndependent —— 防回归：三个榜的「不参与排名」名单必须**互相独立**。
//
// 背景（2026-10-16 用户）：「风云榜和战力榜、军衔声望榜 都可以单独不参与排名，现在公用一个不对」。
// 实现：ezfy_pvp_exclude.board = pvp / power / prestige，三份名单各查各的。
//
// 风险：日后有人图省事把 board 参数去掉（又退回一份共用名单）—— 表现很隐蔽：
// 在风云榜标记的玩家会莫名从战力榜/军衔声望榜消失，运营要查半天。
// 所以这里断言：
//  1. 两个查询函数签名带 board，且 SQL 里有 board 条件；
//  2. 三处调用点分别传 ezfyBoardPvp / ezfyBoardPower / ezfyBoardPrestige（不是同一个）；
//  3. 模型里有 board 列 + 复合唯一键（同一玩家能在三个榜各排一次）；
//  4. 建表迁移有**一次性标记位**（否则每次重启会把「只在风云榜排除」的记录又复制到另外两榜）。
func TestPvpExcludeBoardsIndependent(t *testing.T) {
	read := func(p string) string {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("读 %s 失败: %v", p, err)
		}
		return string(b)
	}

	pvpSrc := read(filepath.Join(".", "ezfy_pvp_rank.go"))
	for _, sig := range []string{"func pvpExcludedSet(db *gorm.DB, board string)", "func pvpExcludedRows(db *gorm.DB, board string)"} {
		body := funcBody(pvpSrc, sig)
		if body == "" {
			t.Fatalf("找不到 %s 函数体（榜单名单必须按 board 隔离）", sig)
		}
		if !strings.Contains(body, "board = ?") {
			t.Fatalf("%s 没在 SQL 里按 board 过滤 —— 三个榜会互相影响", sig)
		}
	}

	// 调用点：玩家端 / 管理端三个榜各传自己的 board
	callers := map[string][]string{
		filepath.Join(".", "ezfy_api.go"):        {"pvpExcludedSet(h.DB, ezfyBoardPower)", "pvpExcludedSet(h.DB, ezfyBoardPrestige)"},
		filepath.Join(".", "ezfy_admin_rank.go"): {"pvpExcludedSet(h.DB, ezfyBoardPower)", "pvpExcludedSet(h.DB, ezfyBoardPrestige)", "pvpExcludedRows(h.DB, ezfyBoardPvp)"},
	}
	for path, wants := range callers {
		src := read(path)
		for _, w := range wants {
			if !strings.Contains(src, w) {
				t.Fatalf("%s 里找不到 %q —— 榜单名单没按 board 分开", filepath.Base(path), w)
			}
		}
	}

	// 模型：board 列 + 复合唯一键
	modelSrc := read(filepath.Join("..", "..", "model", "ezfy.go"))
	ms := funcBody(modelSrc, "type EzfyPvpExclude struct {")
	if ms == "" {
		t.Fatal("找不到 model.EzfyPvpExclude 定义")
	}
	if !strings.Contains(ms, "Board") || !strings.Contains(ms, "uk_excl_board_user") {
		t.Fatal("EzfyPvpExclude 缺 board 列 / 复合唯一键 uk_excl_board_user")
	}

	// 迁移：一次性标记位（防「启动迁移重跑」把单独取消的设置冲掉）
	seedSrc := read(filepath.Join("..", "..", "seed", "seed.go"))
	sb := funcBody(seedSrc, "func EnsureEzfyPvpExcludeTable(")
	if sb == "" {
		t.Fatal("找不到 seed.EnsureEzfyPvpExcludeTable 函数体")
	}
	if !strings.Contains(sb, "ezfyPvpExcludeBoardMigratedKey") {
		t.Fatal("EnsureEzfyPvpExcludeTable 没有一次性标记位 —— 每次重启都会把已取消的排除复制回来")
	}
	if !strings.Contains(sb, "uk_pvp_exclude_user") {
		t.Fatal("EnsureEzfyPvpExcludeTable 没删老唯一索引 uk_pvp_exclude_user —— 同一玩家无法在第二个榜标记")
	}
}

// TestPvpKillCapSwitch —— 防回归：风云榜「灭敌分上限」开关必须真的生效。
//
// 背景（2026-10-29 用户需求）：「单灭敌加分上限加个开关，默认开有上限，关掉期间灭敌无上限」。
// 关闭靠 ezfy_cfg_limit.pvp_kill_cap_on = 0，读取点是 ezfyPvpKillCapOn()。
//
// 风险：日后有人重构 pvpBattlePoints 时把封顶写回「无条件截断」（开关只改了个 UI），
// 表现是「后台明明关了无上限，分还是卡在 200」——玩家端看不出来，运营要查半天。
// 所以这里断言：
//  1. pvpBattlePoints 里的截断受 ezfyPvpKillCapOn() 门禁（不是裸 if killPts > cap）；
//  2. ezfyPvpKillCapOn 读缓存 limit.PvpKillCapOn 且有 ready() 保护（0 合法，不能零值误判）；
//  3. 模型有列 + 迁移补列（只回填 NULL，不用 <= 0 兜底，否则关掉的又被打开）；
//  4. 管理端有开关接口（路由 + handler）。
func TestPvpKillCapSwitch(t *testing.T) {
	read := func(p string) string {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("读 %s 失败: %v", p, err)
		}
		return string(b)
	}
	pvpSrc := read(filepath.Join(".", "ezfy_pvp_rank.go"))

	// 1) 截断必须带开关门禁
	body := funcBody(pvpSrc, "func pvpBattlePoints(")
	if body == "" {
		t.Fatal("找不到 pvpBattlePoints 函数体（重构后请同步更新本断言）")
	}
	guard := strings.Index(body, "ezfyPvpKillCapOn()")
	if guard < 0 {
		t.Fatal("pvpBattlePoints 里没有灭敌分上限开关门禁（ezfyPvpKillCapOn()）—— 关掉开关仍会封顶")
	}
	capCut := strings.Index(body, "killPts = ezfyPvpKillPtCap")
	if capCut < 0 {
		t.Fatal("pvpBattlePoints 里找不到灭敌分封顶语句（killPts = ezfyPvpKillPtCap）")
	}
	if capCut < guard {
		t.Fatal("封顶语句出现在开关门禁之前 —— 关闭开关也会封顶")
	}

	// 2) 开关读取走缓存 limit + ready() 保护
	onBody := funcBody(pvpSrc, "func ezfyPvpKillCapOn()")
	if onBody == "" {
		t.Fatal("找不到 ezfyPvpKillCapOn 函数体")
	}
	if !strings.Contains(onBody, "ezfyCfg.limit.PvpKillCapOn") {
		t.Fatal("ezfyPvpKillCapOn 未读 ezfyCfg.limit.PvpKillCapOn")
	}
	if !strings.Contains(onBody, "ezfyCfg.ready()") {
		t.Fatal("ezfyPvpKillCapOn 缺 ready() 保护 —— 配置未加载时会读到零值 0，误判成「无上限」")
	}

	// 3) 模型列 + 迁移（0 合法：只回填 NULL）
	modelSrc := read(filepath.Join("..", "..", "model", "ezfy.go"))
	if !strings.Contains(modelSrc, "PvpKillCapOn") || !strings.Contains(modelSrc, "pvp_kill_cap_on") {
		t.Fatal("model.EzfyCfgLimit 缺 PvpKillCapOn / pvp_kill_cap_on 列")
	}
	seedSrc := read(filepath.Join("..", "..", "seed", "seed.go"))
	if !strings.Contains(seedSrc, "pvp_kill_cap_on int DEFAULT 1") {
		t.Fatal("seed 没有补列 pvp_kill_cap_on（老库会缺列）")
	}
	if strings.Contains(seedSrc, "pvp_kill_cap_on <= 0") {
		t.Fatal("seed 用 `pvp_kill_cap_on <= 0` 兜底 —— 会把管理员关掉的上限又打开")
	}

	// 4) 管理端接口
	adminSrc := read(filepath.Join(".", "ezfy_admin_rank.go"))
	if !strings.Contains(adminSrc, "func (h *EzfyAdmin) AdminEzfyPvpKillCap(") {
		t.Fatal("缺管理端开关 handler AdminEzfyPvpKillCap")
	}
	if !strings.Contains(adminSrc, `Update("pvp_kill_cap_on"`) {
		t.Fatal("AdminEzfyPvpKillCap 没有写 pvp_kill_cap_on")
	}
	routerSrc := read(filepath.Join("..", "..", "router", "router.go"))
	if !strings.Contains(routerSrc, "/ezfy-pvp-rank/kill-cap") {
		t.Fatal("路由缺 POST /admin/ezfy-pvp-rank/kill-cap")
	}
}

// TestPvpBattleCapConfig —— 防回归：单场分数上限 / 同对手周封顶必须**可配置**且默认 800 / 20000。
//
// 背景（2026-10-29 用户需求）：「风云榜加个单场分数上限配置，默认 800，可灵活调整」
// + 「pvp 对战同一个人积分上限也可以配置，默认 20000」。
// 读取点分别是 ezfyPvpBattleCap() / ezfyPvpFoeCap()（读 ezfy_cfg_limit.pvp_battle_cap / pvp_foe_cap）。
//
// 风险：日后重构 pvpBattlePoints / accumPvpWeekOnce 时把 800/20000 写死回去（配置项只改了个 UI）——
// 表现是「后台改了，分数还是卡在老值」，玩家端看不出原因。
func TestPvpBattleCapConfig(t *testing.T) {
	read := func(p string) string {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("读 %s 失败: %v", p, err)
		}
		return string(b)
	}
	pvpSrc := read(filepath.Join(".", "ezfy_pvp_rank.go"))

	// 1) 封顶值必须来自配置读取函数，不能是常量
	body := funcBody(pvpSrc, "func pvpBattlePoints(")
	if body == "" {
		t.Fatal("找不到 pvpBattlePoints 函数体")
	}
	if !strings.Contains(body, "ezfyPvpBattleCap()") || !strings.Contains(body, "ezfyPvpSmallCap()") {
		t.Fatal("pvpBattlePoints 的封顶没走 ezfyPvpBattleCap()/ezfyPvpSmallCap()（配置改了不生效）")
	}
	if strings.Contains(body, "ezfyPvpMaxPoint ") || strings.Contains(body, "ezfyPvpSmallMax ") {
		t.Fatal("pvpBattlePoints 里还在用写死的旧常量 ezfyPvpMaxPoint / ezfyPvpSmallMax")
	}
	// 同对手封顶同理
	onceBody := funcBody(pvpSrc, "func (h *EzfyHandler) accumPvpWeekOnce(")
	if onceBody == "" {
		t.Fatal("找不到 accumPvpWeekOnce 函数体")
	}
	if !strings.Contains(onceBody, "ezfyPvpFoeCap()") {
		t.Fatal("accumPvpWeekOnce 的同对手封顶没走 ezfyPvpFoeCap()（配置改了不生效）")
	}

	// 2) 读取函数：读缓存 limit + 非法值回落默认
	capBody := funcBody(pvpSrc, "func ezfyPvpBattleCap()")
	if capBody == "" {
		t.Fatal("找不到 ezfyPvpBattleCap 函数体")
	}
	if !strings.Contains(capBody, "ezfyCfg.limit.PvpBattleCap") || !strings.Contains(capBody, "ezfyPvpMaxPointDef") {
		t.Fatal("ezfyPvpBattleCap 未读 PvpBattleCap 或没有回落默认值（0 会把每场分打成 0）")
	}
	smallBody := funcBody(pvpSrc, "func ezfyPvpSmallCap()")
	if smallBody == "" || !strings.Contains(smallBody, "ezfyPvpBattleCap() / 2") {
		t.Fatal("ezfyPvpSmallCap 必须是「单场上限的一半」（默认 800 → 400）")
	}
	foeBody := funcBody(pvpSrc, "func ezfyPvpFoeCap()")
	if foeBody == "" {
		t.Fatal("找不到 ezfyPvpFoeCap 函数体")
	}
	if !strings.Contains(foeBody, "ezfyCfg.limit.PvpFoeCap") || !strings.Contains(foeBody, "ezfyPvpFoeCapDef") {
		t.Fatal("ezfyPvpFoeCap 未读 PvpFoeCap 或没有回落默认值（0 会把同对手分打成 0）")
	}

	// 3) 默认值仍是 800 / 400 / 20000（配置缺省时行为不变）—— 用正则容忍 gofmt 的等号对齐空格
	for _, re := range []string{
		`ezfyPvpMaxPointDef\s*=\s*int64\(800\)`,
		`ezfyPvpSmallMaxDef\s*=\s*int64\(400\)`,
		`ezfyPvpFoeCapDef\s*=\s*int64\(20000\)`,
	} {
		if !regexp.MustCompile(re).MatchString(pvpSrc) {
			t.Fatalf("默认值被改动（应匹配 %s）", re)
		}
	}

	// 4) 模型列 + 迁移 + 管理端接口
	modelSrc := read(filepath.Join("..", "..", "model", "ezfy.go"))
	if !strings.Contains(modelSrc, "pvp_battle_cap") || !strings.Contains(modelSrc, "pvp_foe_cap") {
		t.Fatal("model.EzfyCfgLimit 缺 pvp_battle_cap / pvp_foe_cap 列")
	}
	seedSrc := read(filepath.Join("..", "..", "seed", "seed.go"))
	if !strings.Contains(seedSrc, "pvp_battle_cap int DEFAULT 800") {
		t.Fatal("seed 没有补列 pvp_battle_cap（老库会缺列）")
	}
	if !strings.Contains(seedSrc, "pvp_foe_cap int DEFAULT 20000") {
		t.Fatal("seed 没有补列 pvp_foe_cap（老库会缺列）")
	}
	adminSrc := read(filepath.Join(".", "ezfy_admin_rank.go"))
	if !strings.Contains(adminSrc, "func (h *EzfyAdmin) AdminEzfyPvpCaps(") ||
		!strings.Contains(adminSrc, `"pvp_battle_cap"`) || !strings.Contains(adminSrc, `"pvp_foe_cap"`) {
		t.Fatal("缺管理端设置接口 AdminEzfyPvpCaps / 未写两个上限字段")
	}
	routerSrc := read(filepath.Join("..", "..", "router", "router.go"))
	if !strings.Contains(routerSrc, "/ezfy-pvp-rank/caps") {
		t.Fatal("路由缺 POST /admin/ezfy-pvp-rank/caps")
	}
}

// funcBody 粗略取出 Go 函数体（从 signature 起，按大括号配平到函数结束）。
// 只在测试里做静态检查用，遇到字符串/注释里的括号可能失真 —— 对本文件足够。
func funcBody(src, sig string) string {
	i := strings.Index(src, sig)
	if i < 0 {
		return ""
	}
	j := strings.Index(src[i:], "{")
	if j < 0 {
		return ""
	}
	start := i + j
	depth := 0
	inLine := false
	inBlock := false
	for k := start; k < len(src); k++ {
		switch {
		case inLine:
			if src[k] == '\n' {
				inLine = false
			}
			continue
		case inBlock:
			if src[k] == '*' && k+1 < len(src) && src[k+1] == '/' {
				inBlock = false
				k++
			}
			continue
		}
		switch src[k] {
		case '/':

			if k+1 < len(src) && src[k+1] == '/' {
				inLine = true
				k++
				continue
			}
			if k+1 < len(src) && src[k+1] == '*' {
				inBlock = true
				k++
				continue
			}
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[start : k+1]
			}
		}
	}
	return src[start:]
}

// 保证 regexp 被使用（本文件保留给后续更严格的断言扩展）。
var _ = regexp.MustCompile
