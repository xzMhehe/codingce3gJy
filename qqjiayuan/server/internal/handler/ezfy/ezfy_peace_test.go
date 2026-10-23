package ezfy

import (
	"os"
	"strings"
	"testing"
)

// ============ 免战保护令「双向生效」+ 城市状态页口径 守卫（2026-10-15）============
//
// 用户规则（原话）：「免战城市的城市不能被人打，当前免战城市也不能打别人只能打野地」。
//   - 目标侧：别人打不到免战城 —— 出征拦截 + 到达结算两处早已实现（2026-09-27）。
//   - 出击侧：**当前城市**在免战期时，不能对**玩家城市**(targetType=3) 发起 侦查/掠夺/征服。
//     野地(1)/寇城(2) 不受限；运输(5)/增援(6)/派遣(8) 是自家/盟友城，也不受影响。
//
// 另两条「城市状态页数字不对」的口径（同日修复）：
//   - 「未读军情: N」原来统计**全部未读战报**（含部队返航/城市迁移/将领升级这类非军情播报）
//     → 改用与军情页两个 tab 徽标同口径的 `ezfyReportCounts`（军情警讯 + 战斗报告）。
//   - 「人口: 78250/18000」分母原来恒取民居容纳 → 2026-10-22 起人口上限只由民居
//     pop_max 决定；2026-10-23 起超过上限的老数据在下一次结算时立即夹平。

// TestCreateOrderBlocksAttackWhileInPeace —— 出击侧：免战城不得侦查/掠夺/征服玩家城市
func TestCreateOrderBlocksAttackWhileInPeace(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) createOrder(")
	if !strings.Contains(body, "h.hasCityEffect(city.ID, 2)") {
		t.Fatal("createOrder 缺少出击侧免战拦截：当前城市在免战期时不能主动打别人（hasCityEffect(city.ID, 2)）")
	}
	// 拦截范围必须是 侦查(1)/掠夺(2)/征服(3) 且目标为玩家城市(targetType=3)
	//   ★ 取判定点**前后各一段**窗口（前面是条件、后面是拦截文案）
	i := strings.Index(body, "h.hasCityEffect(city.ID, 2)")
	lo := i - 700
	if lo < 0 {
		lo = 0
	}
	hi := i + 500
	if hi > len(body) {
		hi = len(body)
	}
	seg := body[lo:hi]
	for _, want := range []string{"orderType == 1", "orderType == 2", "orderType == 3", "targetType == 3"} {
		if !strings.Contains(seg, want) {
			t.Fatalf("出击侧免战拦截的条件不完整（应含 %s）：\n%s", want, seg)
		}
	}
	if !strings.Contains(seg, "免战保护期") {
		t.Fatalf("出击侧免战拦截的文案应说明「当前城市处于免战保护期」：\n%s", seg)
	}
}

// TestCreateOrderStillBlocksPeaceTarget —— 目标侧：免战城不能被别人打（不得被误删）
func TestCreateOrderStillBlocksPeaceTarget(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) createOrder(")
	if !strings.Contains(body, "h.hasCityEffect(uint(targetId), 2)") {
		t.Fatal("createOrder 的目标侧免战拦截被弄丢了：免战城不能被掠夺/征服（hasCityEffect(uint(targetId), 2)）")
	}
}

// TestViewSendsProtectedUntil —— 城市状态页要展示免战时效 → /view 必须下发到期时间戳
func TestViewSendsProtectedUntil(t *testing.T) {
	view := strings.ReplaceAll(rawFile(t, "ezfy.go"), "\r\n", "\n")
	if !strings.Contains(view, `"protected_until": protectedUntil`) {
		t.Fatal("/view 必须下发 protected_until（城市状态页据此显示「[免战保护中] 剩余 …」）")
	}
	if !strings.Contains(view, "protectedUntil = e.UntilTime") {
		t.Fatal("免战 effect_type=2 的并行查询必须把 UntilTime 写进 protectedUntil（不能只下发布尔值）")
	}
}

// TestViewUnreadReportsUsesIntelScope —— 「未读军情」必须与军情页徽标同口径（军情警讯+战斗报告）
func TestViewUnreadReportsUsesIntelScope(t *testing.T) {
	view := strings.ReplaceAll(rawFile(t, "ezfy.go"), "\r\n", "\n")
	if !strings.Contains(view, "h.ezfyReportCounts(uid, 0)") {
		t.Fatal("/view 的 unread_reports 必须走 ezfyReportCounts(uid, 0)（军情警讯+战斗报告，全城口径）")
	}
	if strings.Contains(view, `Where("user_id = ? AND is_read = 0", uid).Count(&c)`) {
		t.Fatal("/view 的 unread_reports 又回退成「全部未读战报」了 —— 会把部队返航/城市迁移/将领升级也算成未读军情")
	}
	// 口径只能有一处实现：两个 tab 的统计函数
	if !strings.Contains(strings.ReplaceAll(rawFile(t, "ezfy_api.go"), "\r\n", "\n"), "func (h *EzfyHandler) ezfyReportCounts(") {
		t.Fatal("ezfyReportCounts 不见了（军情徽标与「未读军情」的共同口径）")
	}
}

// TestCityStatusPopulationCap —— 城市状态页人口分母必须用有效上限(popCap)，不能再恒取民居容纳
func TestCityStatusPopulationCap(t *testing.T) {
	// 前端源在 web/src/views/ezfy 下（测试工作目录是 server/internal/handler/ezfy → 上溯 4 级到仓库根）
	cityVue := readFrontFile(t, "../../../../web/src/views/ezfy/modules/EzfyCity.vue")
	line := ""
	for _, l := range strings.Split(strings.ReplaceAll(cityVue, "\r\n", "\n"), "\n") {
		if strings.Contains(l, "人口: {{ ezfy.city.pop }}") {
			line = l
			break
		}
	}
	if line == "" {
		t.Fatal("找不到城市状态页的「人口: …」那一行（EzfyCity.vue 结构变了？）")
	}
	if !strings.Contains(line, "ezfy.popCap") {
		t.Fatalf("城市状态页人口分母必须用 ezfy.popCap（有效上限），当前：%s", strings.TrimSpace(line))
	}
	if strings.Contains(line, "ezfy.city.pop_max") {
		t.Fatalf("城市状态页人口分母还在用民居容纳 city.pop_max —— 召集灵活配置开着时会显示「78250/18000」：%s", strings.TrimSpace(line))
	}

	ezfyVue := readFrontFile(t, "../../../../web/src/views/ezfy/Ezfy.vue")
	for _, want := range []string{"popCap ()", "peaceText ()", "protectedUntilMs"} {
		if !strings.Contains(ezfyVue, want) {
			t.Fatalf("Ezfy.vue 缺少 %s（城市状态页人口上限/免战时实效应的数据源）", want)
		}
	}
}

// readFrontFile 读用户端前端源码（静态断言用）。
func readFrontFile(t *testing.T, file string) string {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("读前端源码失败 %s: %v", file, err)
	}
	return string(b)
}
