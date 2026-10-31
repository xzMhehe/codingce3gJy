package ezfy

import (
	"strings"
	"testing"
)

// TestMustLeadOfficerTypes 出征类型「是否必须带军官」口径
// （★ 2026-10-30 用户规则，★ 2026-10-31 调整：运输(5) 也改成**可带可不带**）：
//
//	必带 = 侦查/掠夺/征服/采集/增援/驻守采集；运输(5)/派遣(8) 选填 ——
//	★ 「选填」= 不强制，但传了军官仍要生效（不是禁止带）。
func TestMustLeadOfficerTypes(t *testing.T) {
	must := []int{1, 2, 3, 4, 6, 7}
	for _, ot := range must {
		if !ezfyMustLeadOfficer(ot) {
			t.Fatalf("%s(%d) 必须带军官，但 ezfyMustLeadOfficer 返回 false", ezfyOrderTypeName(ot), ot)
		}
	}
	// ★ 2026-10-31 用户要求「运输、派遣不需要军官（可带可不带）」
	for _, ot := range []int{5, 8} {
		if ezfyMustLeadOfficer(ot) {
			t.Fatalf("%s(%d) 不该强制带军官（运输/派遣是可带可不带）", ezfyOrderTypeName(ot), ot)
		}
	}
}

// TestOptionalOfficerStillAllowed 运输/派遣是「**可带可不带**」，不是「不能带」：
//
//	UI 必须给「(可选)」提示、军官下拉不能被订单类型禁用、且保留「未指定」空选项。
func TestOptionalOfficerStillAllowed(t *testing.T) {
	mapVue := strings.ReplaceAll(readFrontFile(t,
		"../../../../web/src/views/ezfy/modules/EzfyMap.vue"), "\r\n", "\n")
	if !strings.Contains(mapVue, "(可选，随军调往目标城)") || !strings.Contains(mapVue, ">(可选)<") {
		t.Fatal("运输/派遣 的指挥军官应显示「(可选)」，别让玩家以为不能带")
	}
	if strings.Contains(mapVue, `:disabled="ezfy.orderType`) ||
		strings.Contains(mapVue, `:disabled="!ezfy.mustLeadOfficer`) {
		t.Fatal("军官下拉不能被订单类型禁用 —— 运输/派遣是「可带可不带」")
	}
	if !strings.Contains(mapVue, `<option value="0">未指定</option>`) {
		t.Fatal("军官下拉缺少「未指定」空选项（不带军官时必须能选它）")
	}
}

// TestMustLeadOfficerFrontendSync 前端 mustLeadOfficer 必须与后端同一口径：
//
//	★ 2026-10-31 用户要求「运输、派遣不需要军官」→ 前端数组也必须是 [1,2,3,4,6,7]，
//	否则会出现「前端拦着不让发 / 后端其实允许」的不一致。
func TestMustLeadOfficerFrontendSync(t *testing.T) {
	vue := strings.ReplaceAll(readFrontFile(t,
		"../../../../web/src/views/ezfy/Ezfy.vue"), "\r\n", "\n")
	if !strings.Contains(vue, "return [1, 2, 3, 4, 6, 7].indexOf(parseInt(this.orderType, 10)) >= 0") {
		t.Fatal("前端 mustLeadOfficer 数组必须与后端一致（运输5/派遣8 免军官）")
	}
	if strings.Contains(vue, "return [1, 2, 3, 4, 5, 6, 7].indexOf") {
		t.Fatal("前端 mustLeadOfficer 仍把运输(5) 算成必带军官")
	}
}

// TestCreateOrderRequiresLeadOfficer createOrder 里必须真的落这道校验，
// 且校验在「扣油/扣兵/写订单」之前（否则玩家会被扣了资源却出不去兵）。
func TestCreateOrderRequiresLeadOfficer(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) createOrder(")
	for _, want := range []string{
		`officer == "" && ezfyMustLeadOfficer(orderType)`,
		`请选择带队军官`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("createOrder 缺少「必须带军官」校验（缺 %s）", want)
		}
	}
	// 校验必须早于写库（订单 Create）
	iCheck := strings.Index(body, "ezfyMustLeadOfficer(orderType)")
	iCreate := strings.Index(body, "h.DB.Create(&order)")
	if iCheck < 0 || iCreate < 0 || iCheck > iCreate {
		t.Fatalf("「必须带军官」校验必须在写订单之前（check=%d create=%d）", iCheck, iCreate)
	}
}

// TestCreateOrderOfficerCAS 同一军官不能被两支出征队伍同时带走：
// 必须用**带条件的 UPDATE**（status 0→1）做原子抢占，而不是「先读校验再无条件写」——
// 连点/双开/同时打两个不同目标时，读校验会同时通过，只有 CAS 才能拦住。
func TestCreateOrderOfficerCAS(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) createOrder(")
	if !strings.Contains(body, `Where("id = ? AND status = 0", lead.ID)`) {
		t.Fatal("createOrder 缺少军官出征态的 CAS 抢占（Where id AND status=0）")
	}
	if !strings.Contains(body, "res.RowsAffected == 0") {
		t.Fatal("createOrder 的 CAS 必须检查 RowsAffected（否则抢不到也放行）")
	}
	// 抢占必须在写库之前：抢不到就直接返回，不能已经扣了油/兵/写了订单
	iCAS := strings.Index(body, "res.RowsAffected == 0")
	iCreate := strings.Index(body, "h.DB.Create(&order)")
	if iCAS < 0 || iCreate < 0 || iCAS > iCreate {
		t.Fatalf("CAS 抢占必须在写订单之前（cas=%d create=%d）", iCAS, iCreate)
	}
	// 抢占成功后不能再重复置出征态（白多一次跨 WAN 写）
	if strings.Contains(body, "h.officerGoOutByID(lead.ID, true)") {
		t.Fatal("CAS 已置出征态，收尾不应再调 officerGoOutByID(lead.ID, true)")
	}
}
