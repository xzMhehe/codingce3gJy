package ezfy

import (
	"os"
	"strings"
	"testing"
)

// ============ 二战 SSO 占位页「点此进入」兜底守卫 ============
//
// ★ 2026-10-18 线上问题：从家园进二战（/games/ezfy?sso_token=…）时，SSO 期间显示的是
// 「正在进入二战征途-【2区】诺曼底登陆…」占位卡片；/api/ezfy/sso 走跨机 HTTP（axios 15s 才超时），
// 偶尔会一直卡在这张卡上，玩家没有任何可点的东西 → 用户要求「加个手动跳转」。
//
// 口径（别改回去）：
//   - 占位卡在 SSO 超过 5 秒后**多出一行「点此进入」**（不改变原有标题/文案/字号）；
//   - 点击走 manualEnter()：已有二战 token → 直接刷新进游戏；没有家园 token → 刷新露出登录框
//     （**绝不自动重试**，否则会变成死循环刷新）；其余重走一次 SSO；
//   - 成功/失败都要清掉那个 5 秒定时器（否则占位消失后 ssoStuck 还在乱置位）。
func TestSsoStuckManualEnterContract(t *testing.T) {
	b, err := os.ReadFile("../../../../web/src/views/ezfy/Ezfy.vue")
	if err != nil {
		t.Fatalf("读前端文件失败: %v", err)
	}
	shell := string(b)

	// ① 占位卡里必须有可点的手动入口（且绑到 manualEnter）
	if !strings.Contains(shell, `@click="manualEnter"`) || !strings.Contains(shell, "点此进入") {
		t.Fatal("SSO 占位页没有「点此进入」手动跳转 —— 卡住时玩家无路可走")
	}
	// ② 只在卡住时才露出（否则会让人误以为要手动点）
	if !strings.Contains(shell, `v-if="ssoStuck"`) {
		t.Fatal("「点此进入」没有 v-if=\"ssoStuck\" 控制 → 正常进服也会一直显示")
	}
	// ③ 兜底定时器必须在 trySsoLogin 里起
	if !strings.Contains(shell, "this.ssoStuckTimer = setTimeout(() => { this.ssoStuck = true }, 5000)") {
		t.Fatal("trySsoLogin 没有 5 秒 ssoStuck 兜底定时器 → 卡住时永远不露出手动入口")
	}
	// ④ 成功/失败都要清定时器
	if strings.Count(shell, "this.clearSsoStuck()") < 3 {
		t.Fatal("clearSsoStuck 调用点不足（进入/成功失败/手动点击都要清），会留下野定时器")
	}
	// ⑤ 手动跳转必须存在，且分支齐全（有 token→刷新；无家园 token→刷新露出登录框；否则重走 SSO）
	if !strings.Contains(shell, "manualEnter ()") {
		t.Fatal("缺 manualEnter() 方法定义 → 模板点上去会报错")
	}
	if !strings.Contains(shell, "if (this.$store.getters.isEzfyLogin) { window.location.reload(); return }") {
		t.Fatal("manualEnter 没有「已有二战 token 直接刷新进游戏」分支")
	}
	if !strings.Contains(shell, "if (!this.$store.state.token) { window.location.reload(); return }") {
		t.Fatal("manualEnter 缺「没有家园 token 就刷新露出登录框」分支 → 会死循环刷新")
	}
}
