package ezfy

import (
	"strings"
	"testing"
)

// TestMustLeadOfficerTypes 出征类型「是否必须带军官」口径（★ 2026-10-30 用户规则）：
// 除派遣(8)外，出征（侦查/掠夺/征服/采集/运输/增援/驻守采集）一律必须带一名带队军官。
func TestMustLeadOfficerTypes(t *testing.T) {
	must := []int{1, 2, 3, 4, 5, 6, 7}
	for _, ot := range must {
		if !ezfyMustLeadOfficer(ot) {
			t.Fatalf("%s(%d) 必须带军官，但 ezfyMustLeadOfficer 返回 false", ezfyOrderTypeName(ot), ot)
		}
	}
	if ezfyMustLeadOfficer(8) {
		t.Fatal("派遣(8) 是唯一例外，不该强制带军官")
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
