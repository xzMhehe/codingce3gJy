package ezfy

import (
	"os"
	"strings"
	"testing"
)

// ============ 二战 SSO 占位页「点此进入」守卫 ============
//
// ★ 2026-10-18 线上问题：从家园进二战（/games/ezfy?sso_token=…）时，SSO 期间显示的是
// 「正在进入二战征途-【2区】诺曼底登陆…」占位卡片；/api/ezfy/sso 走跨机 HTTP，偶尔会一直卡住，
// 玩家没有任何可点的东西。
//
// 口径（别改回去）：
//   - 「点此进入」**常显**，不做几秒延迟（用户：「点此进入能不能别几秒后，直接就有」）——
//     正常 SSO 几百毫秒就自动 reload，链接只是一闪而过；
//   - /ezfy/sso 必须带 **4s 超时**（原来吃 axios 全局 15s，卡住要干等 15s）；
//   - 手动点击走 manualEnter()：已有二战 token → 直接刷新进游戏；没有家园 token → 刷新露出登录框
//     （**绝不自动重试**，否则死循环刷新）；其余重走一次 SSO；
//   - 重走时必须靠 `ssoSeq` 序号丢弃过期响应 —— 否则并发的两个 SSO 各自签发 token，
//     后到的把 session_at 顶掉，玩家可能拿着已失效的 token 进游戏。
func TestSsoStuckManualEnterContract(t *testing.T) {
	b, err := os.ReadFile("../../../../web/src/views/ezfy/Ezfy.vue")
	if err != nil {
		t.Fatalf("读前端文件失败: %v", err)
	}
	shell := string(b)

	// ① 占位卡里必须有手动入口（且绑到 manualEnter）
	if !strings.Contains(shell, `@click="manualEnter"`) || !strings.Contains(shell, "点此进入") {
		t.Fatal("SSO 占位页没有「点此进入」手动跳转 —— 卡住时玩家无路可走")
	}
	// ② 必须是常显：不能再挂 v-if="ssoStuck"（用户明确要求别等几秒）
	if strings.Contains(shell, `v-if="ssoStuck"`) {
		t.Fatal(`「点此进入」还挂着 v-if="ssoStuck" —— 用户要求直接就有，不要延迟显示`)
	}
	if strings.Contains(shell, "ssoStuckTimer") {
		t.Fatal("还残留 ssoStuckTimer 延迟机制（应已改成常显）")
	}
	// ③ SSO 请求必须有 4s 超时，不能吃 axios 全局 15s
	if !strings.Contains(shell, "api.post('/ezfy/sso', {}, { timeout: 4000 })") {
		t.Fatal("/ezfy/sso 没有 4s 超时 —— 卡住时要等满 15s 才露出登录框")
	}
	// ④ 手动跳转必须存在，且分支齐全（有 token→刷新；无家园 token→刷新露出登录框；否则重走 SSO）
	if !strings.Contains(shell, "manualEnter ()") {
		t.Fatal("缺 manualEnter() 方法定义 → 模板点上去会报错")
	}
	if !strings.Contains(shell, "if (this.$store.getters.isEzfyLogin) { window.location.reload(); return }") {
		t.Fatal("manualEnter 没有「已有二战 token 直接刷新进游戏」分支")
	}
	if !strings.Contains(shell, "if (!this.$store.state.token) { window.location.reload(); return }") {
		t.Fatal("manualEnter 缺「没有家园 token 就刷新露出登录框」分支 → 会死循环刷新")
	}
	// ⑤ 并发保护：重走 SSO 必须用请求序号丢弃过期响应
	if !strings.Contains(shell, "const seq = ++this.ssoSeq") {
		t.Fatal("trySsoLogin 没有请求序号 —— 手动点击与自动 SSO 并发时，后到的响应会顶掉先到的 token")
	}
	if !strings.Contains(shell, "if (seq !== this.ssoSeq) return") {
		t.Fatal("trySsoLogin 没有丢弃过期响应 —— 玩家可能拿着被顶掉的 token 进游戏")
	}
}
