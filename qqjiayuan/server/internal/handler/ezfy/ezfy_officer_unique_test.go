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

// ④ 俘虏收编 recruitCaptive 必须查同城同名（排除自己）
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
}

// ⑤ PvP 归降 defectDefenderOfficers 收押前必须查攻方城同名
func TestDefectDefenderRejectsDuplicateName(t *testing.T) {
	src := readOfficerSrc(t)
	idx := strings.Index(src, "func (h *EzfyHandler) defectDefenderOfficers(")
	if idx < 0 {
		t.Fatal("找不到 defectDefenderOfficers")
	}
	body := src[idx:]
	if end := strings.Index(body, "\nfunc "); end > 0 {
		body = body[:end]
	}
	if !strings.Contains(body, "h.ezfyOfficerNameTaken(atkCity.ID, o.Name, 0)") {
		t.Fatal("defectDefenderOfficers 收押前必须查攻方城同名")
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
