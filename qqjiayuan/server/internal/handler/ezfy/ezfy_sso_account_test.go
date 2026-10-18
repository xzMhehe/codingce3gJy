package ezfy

// ============ 家园 ⇄ 二战 登录态一致性守卫 ============
//
// ★★ 2026-10-18 线上问题（玩家反馈 + 用户实测）：
//  ① 「家园 A 登录了，进二战一区/二区还要再登录一次」
//     根因：Ezfy.vue::ssoFromUrl() 跨端口跳转后调 `commit('setToken', ssoToken)`，
//     但 store/index.js 里**根本没有 setToken 这个 mutation** —— Vuex 遇到未知 mutation
//     只打 warning 不写值 → state.token 仍是空 → trySsoLogin() 首行
//     `if (!this.$store.state.token) return` 直接返回 → 必然停在登录框。
//  ② 「A 账户登录家园，还能用 B 账户登录二战」= 账号串台。
//     根因：二战登录页允许随便输家园号，后端也不校验「当前家园会话是谁」。
//
// 口径（别改回去）：
//  · 前端所有 `commit('xxx')` 的 xxx 必须真的在 store 的 mutations 里定义过（下面第 ① 条自动扫）；
//  · 家园 token 要有**跨端口**副本（cookie）—— localStorage 按 origin 隔离，8080 登录的
//    token 在 8081/8082 读不到，「只登二战家园不跟着登」也是同一个根因；
//  · 二战登录页在家园已登录时**锁定为当前家园号**（只读），后端 EzfyLogin 再按
//    Authorization 里的家园 token 做一次 uid 一致性校验（防绕过前端）；
//  · 有 ezfy_token 却没有 jy_token（localStorage+cookie 都没有）= 家园已退出 → 清二战登录态。

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const (
	storePath = "../../../../web/src/store/index.js"
	ezfyVue   = "../../../../web/src/views/ezfy/Ezfy.vue"
	webSrcDir = "../../../../web/src"
)

// TestEveryCommittedMutationExists 扫全前端：`commit('xxx')` 的 xxx 必须在 store mutations 里定义。
//
//	这条就是本次事故的直接护栏 —— 名字写错/漏定义时，Vuex 只 warning，功能静默失效，
//	代码 review 和编译都发现不了。
func TestEveryCommittedMutationExists(t *testing.T) {
	sb, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("读 store 失败: %v", err)
	}
	store := string(sb)

	// 截 mutations 块（到 actions 之前）
	i := strings.Index(store, "mutations: {")
	j := strings.Index(store, "actions: {")
	if i < 0 || j < 0 || j < i {
		t.Fatal("store/index.js 结构变了：找不到 mutations/actions 块")
	}
	block := store[i:j]

	// 形如 `    setToken (s, token) {` / `    logout (s) {`
	reDef := regexp.MustCompile(`(?m)^\s{2,6}([A-Za-z_$][\w$]*)\s*\(`)
	defined := map[string]bool{}
	for _, m := range reDef.FindAllStringSubmatch(block, -1) {
		defined[m[1]] = true
	}
	if len(defined) == 0 {
		t.Fatal("没能从 store mutations 里解析出任何 mutation 名（正则失效？）")
	}

	// 扫 web/src 下所有 commit('name') / commit("name")
	reCommit := regexp.MustCompile(`commit\(\s*['"]([A-Za-z_$][\w$]*)['"]`)
	var missing []string
	seen := map[string]bool{}
	_ = filepath.WalkDir(webSrcDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(p, ".js") && !strings.HasSuffix(p, ".vue") {
			return nil
		}
		b, e := os.ReadFile(p)
		if e != nil {
			return nil
		}
		for _, m := range reCommit.FindAllStringSubmatch(string(b), -1) {
			name := m[1]
			if defined[name] || seen[name+"|"+p] {
				continue
			}
			seen[name+"|"+p] = true
			missing = append(missing, name+" @ "+filepath.Base(p))
		}
		return nil
	})
	if len(missing) > 0 {
		t.Fatalf("commit 了不存在的 mutation（Vuex 只会 warning，功能静默失效）: %v\n"+
			"store 里已定义: %v", missing, keysOf(defined))
	}
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestHomeTokenCrossPortChannel 家园 token 必须有跨端口通道（cookie），退出时必须清。
func TestHomeTokenCrossPortChannel(t *testing.T) {
	b, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("读 store 失败: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "COOKIE_TOKEN") || !strings.Contains(s, "writeCookie") {
		t.Fatal("store 没有 cookie 通道 —— localStorage 按 origin 隔离，8081/8082 读不到家园 token，" +
			"「家园登录了进二战还要再登录」「只登二战家园不跟着登」都会复现")
	}
	// 初始化要「localStorage 优先、cookie 兜底」
	if !strings.Contains(s, "localStorage.getItem('jy_token') || readCookie(COOKIE_TOKEN)") {
		t.Fatal("state.token 初始化没有 cookie 兜底")
	}
	// logout 必须清 cookie（否则退了又被 cookie 读回来 = 幽灵登录态）
	logoutIdx := strings.Index(s, "logout (s) {")
	if logoutIdx < 0 {
		t.Fatal("找不到 logout mutation")
	}
	tail := s[logoutIdx:]
	if end := strings.Index(tail, "ezfyLogout (s)"); end > 0 {
		tail = tail[:end]
	}
	if !strings.Contains(tail, "writeCookie(COOKIE_TOKEN, '')") {
		t.Fatal("logout 没清 cookie → 家园退出后跨端口仍能读到旧 token（幽灵登录态）")
	}
}

// TestWwiiLoginLocksHomeAccount 二战登录页必须锁定当前家园账号（前端锁 + 后端校验）。
func TestWwiiLoginLocksHomeAccount(t *testing.T) {
	b, err := os.ReadFile(ezfyVue)
	if err != nil {
		t.Fatalf("读 Ezfy.vue 失败: %v", err)
	}
	s := string(b)

	// 前端：computed + 只读绑定 + 登录时强制用 homeUid
	if !strings.Contains(s, "homeUid ()") {
		t.Fatal("缺 homeUid computed → 登录页无法锁定当前家园账号")
	}
	if !strings.Contains(s, `:readonly="!!homeUid"`) {
		t.Fatal("家园号输入框没有 :readonly=\"!!homeUid\" → 玩家能在二战登录页换成另一个账号（账号串台）")
	}
	if !strings.Contains(s, "(this.homeUid || this.wwiiName || '')") {
		t.Fatal("wwiiLogin 没有优先用 homeUid → 只读属性被绕过时仍能登别的号")
	}

	// 后端：EzfyLogin 必须做 uid 一致性校验
	gb, err := os.ReadFile("ezfy_auth.go")
	if err != nil {
		t.Fatalf("读 ezfy_auth.go 失败: %v", err)
	}
	g := string(gb)
	if !strings.Contains(g, "homeUIDFromAuth") {
		t.Fatal("ezfy_auth.go 缺 homeUIDFromAuth（家园 token 取 uid 的卡控）")
	}
	if !strings.Contains(g, "uid != 0 && uid != gameID") {
		t.Fatal("EzfyLogin 没有「当前家园账号 ≠ 要登录的家园号 → 拒绝」的卡控")
	}
}

// TestHomeLogoutKillsWwiiSession 家园退出后，二战页必须能自愈回登录框。
func TestHomeLogoutKillsWwiiSession(t *testing.T) {
	b, err := os.ReadFile(ezfyVue)
	if err != nil {
		t.Fatalf("读 Ezfy.vue 失败: %v", err)
	}
	s := string(b)
	// mounted 里的联动：有二战 token 但没有家园 token → 清二战登录态
	if !strings.Contains(s, "if (this.isLogin && !this.$store.state.token)") {
		t.Fatal("Ezfy.vue mounted 缺「有 ezfy_token 却无 jy_token → 清二战登录态」的联动，" +
			"家园退出后 8081/8082 仍会显示已登录")
	}
	// 必须在 loadZone / 进入游戏之前判定
	mi := strings.Index(s, "mounted () {")
	if mi < 0 {
		t.Fatal("找不到 mounted")
	}
	seg := s[mi : mi+1600]
	if !strings.Contains(seg, "if (this.isLogin && !this.$store.state.token)") {
		t.Fatal("家园退出联动的判定不在 mounted 开头（会被后续逻辑跳过）")
	}
}

// TestAccountSwitchNotLeaking 家园换人 / URL 带别的 sso_token 时，不能带着旧二战身份进游戏。
//
//	场景：A 玩过二战（8081 有 A 的 ezfy_token）→ 家园退出 A、B 登录 → 从家园点进二战。
//	若不做 uid 比对，8081 会直接以 A 的身份进游戏（二战页自己没有退出入口，玩家无从纠正）。
func TestAccountSwitchNotLeaking(t *testing.T) {
	sb, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("读 store 失败: %v", err)
	}
	store := string(sb)
	// uid 必须能从 token 解（跨端口时 user 对象读不到）
	if !strings.Contains(store, "export function jwtUid") {
		t.Fatal("store 没有导出 jwtUid —— 跨端口拿不到家园 uid，换号检测失效")
	}
	if !strings.Contains(store, "homeUid: s => jwtUid(s.token)") {
		t.Fatal("store 缺 homeUid getter（从 token 解 uid）")
	}

	b, err := os.ReadFile(ezfyVue)
	if err != nil {
		t.Fatalf("读 Ezfy.vue 失败: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "ezfyUid ()") {
		t.Fatal("Ezfy.vue 缺 ezfyUid computed（本地二战登录态属于哪个家园号）")
	}
	// 家园换人 → 作废旧二战 token
	if !strings.Contains(s, "this.homeUid !== this.ezfyUid") {
		t.Fatal("缺「家园换人 → 作废旧二战 token」的比对，A/B 会串台")
	}
	// URL 的 sso_token 与本地二战身份不同 → 必须以 sso_token 为准
	if !strings.Contains(s, "ssoUid === this.ezfyUid") {
		t.Fatal("ssoFromUrl 没比对 uid —— 本地残留 A 的 ezfy_token 时会跳过 SSO、以 A 进游戏")
	}
	if !strings.Contains(s, "stripSsoToken") {
		t.Fatal("缺 stripSsoToken（一次性 sso_token 要从地址栏清掉）")
	}
}
