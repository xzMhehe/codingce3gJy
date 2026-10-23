package ezfy

import (
	"strings"
	"testing"
)

// ============ 秘密策反（2026-10-23） ============
//
// 规则：目标城守忠诚 −= 用计军官有效学识/10（最高 99），结果最低保留 1。
// 用户给出的验收点：
//   88 - 99 → 1
//   88 - 88 → 1
//   88 - 48 → 40
// 重复发动也仍然按 1 点兜底，不能出现 0 或负数。

// TestEzfySchemeSecretAlert 中计方要「看得见」：
//   - 被别人发动的计谋战报标题是「计谋: …」，必须进**军情警讯(cat1)**并点亮顶部「军情」感叹号；
//   - 自己发动的「计谋发动: …」是收益记录，不能混进警讯（否则角标永远闪）。
//   - 标题必须带目标城市坐标（军情警讯列表只显示标题，重名城要能区分）。
func TestEzfySchemeSecretAlert(t *testing.T) {
	cat1 := ezfyReportCategoryCond(1)
	if !strings.Contains(cat1, "'计谋:%'") {
		t.Fatalf("军情警讯(cat1)口径漏了「计谋:%%」→ 中计玩家看不到提示：\n%s", cat1)
	}
	if strings.Contains(cat1, "计谋发动") {
		t.Fatalf("军情警讯(cat1)口径混入了自己发动的「计谋发动」：\n%s", cat1)
	}
	if cond := ezfyReportAlertCond(); !strings.Contains(cond, "计谋") {
		t.Fatalf("顶部「军情」感叹号口径没算计谋战报：\n%s", cond)
	}
	// ★ 三处口径必须同步：SQL(cat1) / Go 归类（列表复核）/ 徽标统计。
	//   只改 SQL 不改 Go 归类 = 「徽标有数、列表为空」（本次实测踩过）。
	if got := ezfyReportCategory("计谋: 秘密策反 主城(261,129)"); got != 1 {
		t.Fatalf("「计谋: …」应归军情警讯(1), 实际 %d", got)
	}
	if got := ezfyReportCategory("计谋: 先发制人 主城(261,129)"); got != 1 {
		t.Fatalf("「计谋: …」应归军情警讯(1), 实际 %d", got)
	}
	if got := ezfyReportCategory("计谋发动: 秘密策反"); got != 3 {
		t.Fatalf("自己发动的「计谋发动: …」应留在其他(3), 实际 %d", got)
	}
	src := ezfyStripCommentLines(rawFile(t, "ezfy_scheme.go"))
	for _, want := range []string{
		`"计谋: 秘密策反 %s(%d,%d)"`,
		`"计谋: 先发制人 %s(%d,%d)"`,
		// 用计军官必须在本城在职（非俘虏、未出征）
		"is_captive = 0 AND status = 0",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("ezfy_scheme.go 缺少「%s」", want)
		}
	}
}

func TestEzfySchemeSecretDrop(t *testing.T) {
	cases := []struct {
		learning, want int
	}{
		{0, 1},  // 学识不足 10：至少降 1
		{9, 1},  // 同上
		{10, 1}, // 整数除法
		{480, 48},
		{489, 48},
		{880, 88},
		{990, 99},
		{5000, 99}, // 封顶 99
	}
	for _, c := range cases {
		if got := ezfySchemeSecretDrop(c.learning); got != c.want {
			t.Fatalf("学识 %d 应降低 %d 点, 实际 %d", c.learning, c.want, got)
		}
	}
}

func TestEzfySchemeSecretLoyalty(t *testing.T) {
	cases := []struct {
		loyalty, drop, want int
	}{
		{88, 99, 1},  // 超过原忠诚 → 最低 1
		{88, 88, 1},  // 刚好归零 → 最低 1
		{88, 48, 40}, // 正常扣减
		{5, 48, 1},   // 不能为负
		{1, 99, 1},   // 重复发动仍触发最低 1 点兜底
		{100, 11, 89},
	}
	for _, c := range cases {
		if got := ezfySchemeSecretLoyalty(c.loyalty, c.drop); got != c.want {
			t.Fatalf("忠诚 %d 降低 %d 应剩 %d, 实际 %d", c.loyalty, c.drop, c.want, got)
		}
	}
}

// TestEzfySchemeSecretWiring 源码接线断言：
//   - 目标取的是城守（ezfyPositionGuard），不是市长；
//   - 使用当前前端选中的 officer_id，且必须属于本城；
//   - 货架种子 ID=15 / Bullet=10 / Kind=6，并能给存量库幂等补建；
//   - 前端 Kind=6 显示坐标输入并传 officer_id。
func TestEzfySchemeSecretWiring(t *testing.T) {
	schemeSrc := ezfyStripCommentLines(rawFile(t, "ezfy_scheme.go"))
	for _, want := range []string{
		"ezfyPositionGuard",
		"officer_id",
		"ezfySchemeSecretDrop(learning)",
		"ezfySchemeSecretLoyalty(oldLoyalty, drop)",
		"最低保留1点",
	} {
		if !strings.Contains(schemeSrc, want) {
			t.Fatalf("ezfy_scheme.go 秘密策反接线缺少「%s」", want)
		}
	}
	// 只看 Kind=6 分支，避免误伤同文件里「先发制人」仍使用市长的逻辑。
	secretBlock := schemeSrc
	if i := strings.Index(secretBlock, "if sc.Kind == 6 {"); i >= 0 {
		secretBlock = secretBlock[i:]
	}
	if j := strings.Index(secretBlock, "if sc.Kind == 2 || sc.Kind == 3 {"); j >= 0 {
		secretBlock = secretBlock[:j]
	}
	if strings.Contains(secretBlock, "ezfyPositionMayor") {
		t.Fatal("秘密策反目标应改为城守(ezfyPositionGuard)，不能再用市长(ezfyPositionMayor)")
	}
	// 允许重复发动：不能再有「忠诚已降至最低」的前置拒绝。
	if strings.Contains(secretBlock, "忠诚已降至最低") {
		t.Fatal("秘密策反应允许重复发动，并在结算时按最低 1 点兜底")
	}

	seedSrc := ezfyStripCommentLines(rawFile(t, "../../seed/ezfy_officer_pool.go"))
	for _, want := range []string{
		`Name: "秘密策反"`,
		"Bullet: 10",
		"Kind: 6",
		"SortNo: 15",
	} {
		if !strings.Contains(seedSrc, want) {
			t.Fatalf("计谋种子缺少「%s」", want)
		}
	}
	ensureSrc := ezfyStripCommentLines(rawFile(t, "../../seed/seed.go"))
	for _, want := range []string{
		"ezfySchemeSecretSeed.ID",
		"db.Create(&row)",
	} {
		if !strings.Contains(ensureSrc, want) {
			t.Fatalf("EnsureEzfySchemeKinds 未幂等补建秘密策反（缺少「%s」）", want)
		}
	}

	playerVue := ezfyStripCommentLines(readFrontFile(t, "../../../../web/src/views/ezfy/Ezfy.vue"))
	for _, want := range []string{
		"s.kind === 1 || s.kind === 6",
		"body.officer_id = officer.id",
		"schemeSecretDrop",
	} {
		if !strings.Contains(playerVue, want) {
			t.Fatalf("Ezfy.vue 秘密策反前端缺少「%s」", want)
		}
	}
	academyVue := ezfyStripCommentLines(readFrontFile(t, "../../../../web/src/views/ezfy/modules/EzfyAcademy.vue"))
	for _, want := range []string{
		"s.kind === 1 || s.kind === 6",
		"用计军官",
		"预计降低",
	} {
		if !strings.Contains(academyVue, want) {
			t.Fatalf("EzfyAcademy.vue 秘密策反前端缺少「%s」", want)
		}
	}
	adminVue := ezfyStripCommentLines(readFrontFile(t, "../../../../admin-web/src/components/admin/AdminEzfyOfficers.vue"))
	for _, want := range []string{
		"6: '秘密策反'",
		`:label="6"`,
	} {
		if !strings.Contains(adminVue, want) {
			t.Fatalf("管理端计谋类型缺少「%s」", want)
		}
	}
}
