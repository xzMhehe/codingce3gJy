package ezfy

// ★★ 2026-10-09 用户规则静态断言：
//   ① 「玩家相同城市同名军官只能有一个」—— 所有会产生新军官行/改名的入口都要过
//      ezfyOfficerNameTaken；
//   ② 「玩家城市军官数量不能超过当前城市参谋部等级」—— 派遣到目标城也要卡容量。
//
// 这些入口历史上都是**漏卡**的：
//   - 野地/寇城/活动守将 createCaptiveOfficer：无「是否已拥有」判断 → 同一守将连俘 36 次；
//   - 改名 OfficerRename：无同城重名校验 → 改名即制造重名；
//   - 军校招募 hireOfficerDraft / 俘虏收编 recruitCaptive：不查同城同名；
//   - 派遣 OfficerDispatch：不卡目标城容量/重名。
//
// 本测试直接读源码断言，防止后续重构把这些卡控改掉。

import (
	"os"
	"strings"
	"testing"
)

func readOfficerSrc(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("ezfy_officer.go")
	if err != nil {
		t.Fatalf("读取 ezfy_officer.go 失败: %v", err)
	}
	return string(b)
}

// 统一口径函数必须存在（按 city_id + name 查重，可排除自己）
func TestOfficerNameTakenHelperExists(t *testing.T) {
	src := readOfficerSrc(t)
	if !strings.Contains(src, "func (h *EzfyHandler) ezfyOfficerNameTaken(cityId uint, name string, excludeOfficerId uint) bool") {
		t.Fatal("缺少 ezfyOfficerNameTaken 统一同名判定函数")
	}
	// 口径必须是 city_id + name（不是 user_id）
	if !strings.Contains(src, `Where("city_id = ? AND name = ?", cityId, name)`) {
		t.Fatal("ezfyOfficerNameTaken 口径必须是 city_id + name")
	}
}

// ① 野地/活动守将俘虏入口（createCaptiveOfficer）必须查同城同名
func TestCaptiveOfficerRejectsDuplicateName(t *testing.T) {
	src := readOfficerSrc(t)
	idx := strings.Index(src, "func (h *EzfyHandler) createCaptiveOfficer(")
	if idx < 0 {
		t.Fatal("找不到 createCaptiveOfficer")
	}
	body := src[idx:]
	if end := strings.Index(body, "\nfunc "); end > 0 {
		body = body[:end]
	}
	if !strings.Contains(body, "h.ezfyOfficerNameTaken(city.ID, g.Name, 0)") {
		t.Fatal("createCaptiveOfficer 必须查同城同名（否则同一守将被反复俘 → 军官超编）")
	}
}

// ② 改名入口（OfficerRename）必须查同城同名（改名前，排除自己）
func TestOfficerRenameRejectsDuplicateName(t *testing.T) {
	src := readOfficerSrc(t)
	idx := strings.Index(src, "func (h *EzfyHandler) OfficerRename(")
	if idx < 0 {
		t.Fatal("找不到 OfficerRename")
	}
	body := src[idx:]
	if end := strings.Index(body, "\nfunc "); end > 0 {
		body = body[:end]
	}
	if !strings.Contains(body, "h.ezfyOfficerNameTaken(city.ID, name, o.ID)") {
		t.Fatal("OfficerRename 必须查同城同名（排除自己），否则改名即制造重名")
	}
}

// ②b 改名入口必须拒绝「在外采集/出征中」的军官
//
// ★ 2026-10-23 用户规则：在外边采集的军官不允许修改军官名字。
// 命令表 ezfy_order.officer 存的是军官名，改名后按旧名找不回军官 → 归队解不掉出征态，
//
//	officerList 自愈还会把 status 改回 0，导致同一军官被重复派出。
//
// ★ 2026-10-24 按军官 **id** 判断在外（同名军官互不误伤），文案保留「归队后才能改名」。
func TestOfficerRenameRejectsBusyOfficer(t *testing.T) {
	src := readOfficerSrc(t)
	idx := strings.Index(src, "func (h *EzfyHandler) OfficerRename(")
	if idx < 0 {
		t.Fatal("找不到 OfficerRename")
	}
	body := src[idx:]
	if end := strings.Index(body, "\nfunc "); end > 0 {
		body = body[:end]
	}
	if !strings.Contains(body, "o.Status == 1 || h.officerBusyOrderByID(city.ID, o.Name, uint(o.ID))") {
		t.Fatal("OfficerRename 必须按军官 id 拒绝在外采集/出征中的军官（status=1 或仍有未结束命令）")
	}
	if !strings.Contains(body, "归队后才能改名") {
		t.Fatal("OfficerRename 拒绝文案必须提示「归队后才能改名」")
	}
}

// ③ 军校招募 hireOfficerDraft 必须查同城同名
func TestHireOfficerDraftRejectsDuplicateName(t *testing.T) {
	src := readOfficerSrc(t)
	idx := strings.Index(src, "func (h *EzfyHandler) hireOfficerDraft(")
	if idx < 0 {
		t.Fatal("找不到 hireOfficerDraft")
	}
	body := src[idx:]
	if end := strings.Index(body, "\nfunc "); end > 0 {
		body = body[:end]
	}
	if !strings.Contains(body, "h.ezfyOfficerNameTaken(city.ID, pick.Name, 0)") {
		t.Fatal("hireOfficerDraft 必须查同城同名")
	}
}

// ④ 俘虏收编 recruitCaptive 必须查同城同名（排除自己）——
//    但 PvP 抢来的俘虏（source=4 抢玩家获取）豁免同名（2026-10-24 用户规则「抢别人的军官不受同名卡控」）
func TestRecruitCaptiveRejectsDuplicateName(t *testing.T) {
	src := readOfficerSrc(t)
	idx := strings.Index(src, "func (h *EzfyHandler) recruitCaptive(")
	if idx < 0 {
		t.Fatal("找不到 recruitCaptive")
	}
	body := src[idx:]
	if end := strings.Index(body, "\nfunc "); end > 0 {
		body = body[:end]
	}
	if !strings.Contains(body, "h.ezfyOfficerNameTaken(city.ID, o.Name, o.ID)") {
		t.Fatal("recruitCaptive 必须查同城同名（排除自己）")
	}
	// ★ 2026-10-24 PvP 来源（source=4 抢玩家获取）的俘虏允许同名收编
	if !strings.Contains(body, "o.Source != model.EzfyOfficerSourcePvp && h.ezfyOfficerNameTaken(city.ID, o.Name, o.ID)") {
		t.Fatal("recruitCaptive 必须豁免 PvP 来源俘虏的同名限制（否则抢来的同名将无法收编）")
	}
}

// ⑤ PvP 归降 defectDefenderOfficers 收押**不得查攻方城同名**
//
// ★ 2026-10-24 用户规则「抢别人的军官（普通、名将）不受同名卡控」：
//   2026-10-09 曾在 PvP 收押前查同名（同名即放弃收押，玩家反馈「隆美尔忠诚归零却未收押」），
//   与 2026-10-05 的「玩家抢玩家的名将不受重复卡控」矛盾 → 已移除。此测试防止再把它加回来。
func TestDefectDefenderAllowsDuplicateName(t *testing.T) {
	src := readOfficerSrc(t)
	idx := strings.Index(src, "func (h *EzfyHandler) defectDefenderOfficers(")
	if idx < 0 {
		t.Fatal("找不到 defectDefenderOfficers")
	}
	body := src[idx:]
	if end := strings.Index(body, "\nfunc "); end > 0 {
		body = body[:end]
	}
	if strings.Contains(body, "h.ezfyOfficerNameTaken(atkCity.ID, o.Name, 0)") {
		t.Fatal("defectDefenderOfficers 不得查攻方城同名（PvP 抢将不受同名卡控）")
	}
}

// ⑥ 派遣 OfficerDispatch 必须卡目标城「军官位 ≤ 参谋部等级」+ 同城同名
func TestOfficerDispatchEnforcesTargetCityCap(t *testing.T) {
	src := readOfficerSrc(t)
	idx := strings.Index(src, "func (h *EzfyHandler) OfficerDispatch(")
	if idx < 0 {
		t.Fatal("找不到 OfficerDispatch")
	}
	body := src[idx:]
	if end := strings.Index(body, "\nfunc "); end > 0 {
		body = body[:end]
	}
	if !strings.Contains(body, "h.officerCapacity(dst.ID)") || !strings.Contains(body, "h.officerCount(dst.ID)") {
		t.Fatal("OfficerDispatch 必须卡目标城参谋部容量（否则多城互调可堆超编）")
	}
	if !strings.Contains(body, "h.ezfyOfficerNameTaken(dst.ID, o.Name, 0)") {
		t.Fatal("OfficerDispatch 必须查目标城同名")
	}
}
