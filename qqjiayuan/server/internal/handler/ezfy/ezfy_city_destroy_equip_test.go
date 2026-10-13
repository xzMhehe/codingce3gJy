package ezfy

import (
	"strings"
	"testing"
)

// TestCityDestroyKeepsEquipment 弃城**不得**删除账号背包里的装备/宝物。
//
// ★★ 2026-10-13 线上事故（用户 35806411，一次丢了 305 件）：
//
//	弃城 `DELETE FROM ezfy_equipment WHERE city_id = ?` 把「出生城 = 该城」的装备全删了，
//	而装备是**账号级**资产（`equipmentList(uid)` 就是 `WHERE user_id = ?` 全局查、界面不分城），
//	`city_id` 只是「获得时的出生城」且穿戴/转移到别城军官后**不更新**
//	（备份里就有 `city_id=68` 却穿在 142/91/92/552/71 等城军官身上的行）。
//	于是玩家主动弃一座城，账号背包里的宝物凭空消失；恢复时又因为按「军官」挑而漏了 281 件未穿戴宝物。
//
// 正确做法：**装备保留在账号背包** —— 只把 `officer_id` 卸下（军官已随城消失）、
// `city_id` 迁到玩家主城（且主城要排除被弃的那座）。
func TestCityDestroyKeepsEquipment(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy.go"), "\r\n", "\n")
	i := strings.Index(src, "func (h *EzfyHandler) ezfyDestroyCity(")
	if i < 0 {
		t.Fatal("找不到 ezfyDestroyCity")
	}
	j := strings.Index(src[i:], "func (h *EzfyHandler) RenameCity(")
	if j < 0 {
		t.Fatal("找不到 ezfyDestroyCity 的结束边界（RenameCity）")
	}
	body := src[i : i+j]
	if strings.Contains(body, "Delete(&model.EzfyEquipment{})") {
		t.Fatal("弃城仍在删除玩家装备 —— 装备是账号级资产（按 user_id 查），弃城不得删")
	}
	if !strings.Contains(body, `h.DB.Model(&model.EzfyEquipment{}).Where("city_id = ?", cid)`) {
		t.Fatal("弃城没有把该城装备迁到主城（应卸下 officer_id 并改 city_id）")
	}
	if !strings.Contains(body, `"officer_id": 0`) {
		t.Fatal("弃城迁装备时必须把 officer_id 置 0（军官已随城一起消失）")
	}
	if !strings.Contains(body, `"user_id = ? AND id <> ?", uid, cid`) {
		t.Fatal("弃城选主城时没有排除被弃的那座城（anyCity 可能正好返回它 → 等于没迁）")
	}
}
