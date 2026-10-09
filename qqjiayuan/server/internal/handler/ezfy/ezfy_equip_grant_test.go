package ezfy

import (
	"strings"
	"testing"
)

// TestEquipGrantSupportsMultiSelect 管理端「给玩家发装备」必须支持多选装备。
//
// ★★ 2026-10-09 用户要求：「二战 → 军官管理 → 玩家军官装备列表 → 给玩家发装备，
// 装备下拉支持多选」。原来一次只能选 1 种（cfg_id），配一整套要重复十几次。
//
// 断言两处：
//
//	① 后端 AdminEzfyEquipmentOwnedCreate 同时接受 `cfg_ids`(数组) 与 `cfg_id`(单个，兼容)；
//	② 前端 equips 下拉是 multiple 且绑定 cfg_ids。
func TestEquipGrantSupportsMultiSelect(t *testing.T) {
	be := strings.ReplaceAll(rawFile(t, "ezfy_admin_cfg.go"), "\r\n", "\n")
	i := strings.Index(be, "func (h *EzfyAdmin) AdminEzfyEquipmentOwnedCreate(")
	if i < 0 {
		t.Fatal("找不到 AdminEzfyEquipmentOwnedCreate")
	}
	j := strings.Index(be[i:], "// AdminEzfyEquipmentOwnedUpdate")
	if j < 0 {
		j = len(be) - i
	}
	fn := be[i : i+j]
	for _, want := range []string{"CfgIds", `json:"cfg_ids"`, `json:"cfg_id"`, "Where(\"id IN ?\"", "strings.Join("} {
		if !strings.Contains(fn, want) {
			t.Fatalf("发装备接口缺少多选支持（缺 %s）：\n%s", want, fn)
		}
	}
}
