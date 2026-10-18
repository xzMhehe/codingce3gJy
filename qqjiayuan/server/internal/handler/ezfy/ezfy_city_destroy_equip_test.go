package ezfy

import (
	"strings"
	"testing"
)

// ★★ 装备与城市的关系（2026-10-18 定案）：
//
//	装备是**玩家背包资产**，只认两个外键 —— `user_id`（属于哪个账号）、
//	`officer_id`（穿在哪名军官身上，0 = 在背包里）。
//	`city_id` 是历史「获得时的出生城」，穿戴 / 俘虏转移 / 派遣到别城之后**都不会更新**，
//	把它当删除或迁移依据 = 把「出生地」当「存放地」，已经酿成两次线上事故：
//	  · 2026-10-13 弃城删装备，一次丢 305 件（35806411）
//	  · 2026-10-18 占城摧毁删装备，丢 3 件青天白日（35806363）
//
//	所以现在两处「清城市数据」的入口都**完全不碰 city_id**，只按军官解绑：
//	该城军官身上的装备放回背包（officer_id=0），军官随后随城删除。
//	`ezfy_equipment.city_id` 由 seed.EnsureEzfyEquipmentNoCity 全量清零并保持恒 0。

// TestCityDestroyKeepsEquipment 弃城（自己的城）不得删除/迁移装备，只按军官解绑。
func TestCityDestroyKeepsEquipment(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy.go"), "\r\n", "\n")
	body := funcBodyBetween(t, src, "ezfyDestroyCity", "func (h *EzfyHandler) RenameCity(")
	assertEquipUntouchedByCity(t, body, "ezfyDestroyCity")
}

// TestOccupyDestroyKeepsEquipment 占城摧毁（别人的城）同样不得删除/迁移装备。
//
// 本次事故就是这条路径：城 59 曾是 35806363 三件青天白日的「出生城」，
// 城被摧毁时 `DELETE FROM ezfy_equipment WHERE city_id = 59` 把别人账号里的装备删了。
func TestOccupyDestroyKeepsEquipment(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_chat_exchange.go"), "\r\n", "\n")
	body := funcBodyBetween(t, src, "deleteCityData", "func (h *EzfyHandler) WildlandFull(")
	assertEquipUntouchedByCity(t, body, "deleteCityData")
}

// TestBuildEquipmentNoCityId 新装备不再写 city_id（恒 0）。
func TestBuildEquipmentNoCityId(t *testing.T) {
	src := strings.ReplaceAll(rawFile(t, "ezfy_officer.go"), "\r\n", "\n")
	body := funcBodyBetween(t, src, "buildEquipment", "func (h *EzfyHandler) addEquipment(")
	if strings.Contains(body, "CityId:") {
		t.Fatal("buildEquipment 仍在写 city_id —— 装备是背包资产，不应记录出生城（否则按 city_id 删装备的旧逻辑又会命中）")
	}
}

// assertEquipUntouchedByCity 统一断言：函数体里不得按 city_id 删/迁装备，且必须在删军官前按军官解绑。
func assertEquipUntouchedByCity(t *testing.T, body, fn string) {
	t.Helper()
	if strings.Contains(body, "Delete(&model.EzfyEquipment{})") {
		t.Fatalf("%s 仍在删除玩家装备 —— 装备是账号级资产（按 user_id 查），任何城市操作都不得删", fn)
	}
	// 任何「EzfyEquipment + city_id」的写法都禁止：删和迁都不行（迁城同样是在拿出生地当存放地）
	for _, bad := range []string{
		`EzfyEquipment{}).Where("city_id = ?"`,
		`EzfyEquipment{})` + "\n" + `\t\tWhere("city_id = ?"`,
		`model.EzfyEquipment{}).Where("city_id = ?"`,
	} {
		if strings.Contains(body, bad) {
			t.Fatalf("%s 仍按 city_id 操作装备（%q）—— 必须改成按 officer_id 解绑", fn, bad)
		}
	}
	// 正确姿势：把本城军官身上的装备放回背包
	bind := `Where("officer_id IN (?)", h.DB.Model(&model.EzfyOfficer{}).Select("id").Where("city_id = ?", cid))`
	if fn == "deleteCityData" {
		bind = `Where("officer_id IN (?)", h.DB.Model(&model.EzfyOfficer{}).Select("id").Where("city_id = ?", cityId))`
	}
	if !strings.Contains(body, bind) {
		t.Fatalf("%s 没有把本城军官身上的装备放回背包（应按 officer_id IN (本城军官) 解绑）", fn)
	}
	if !strings.Contains(body, `Update("officer_id", 0)`) {
		t.Fatalf("%s 解绑装备时必须把 officer_id 置 0（军官已随城消失）", fn)
	}
	// 顺序：先解绑装备，再删军官 —— 反过来军官没了就查不到它身上的装备
	unbindAt := strings.Index(body, bind)
	delAt := strings.Index(body, "Delete(&model.EzfyOfficer{})")
	if unbindAt < 0 || delAt < 0 || unbindAt > delAt {
		t.Fatalf("%s 必须先解绑装备（officer_id=0）再删军官，否则装备会悬挂在已删除的军官上", fn)
	}
}

// funcBodyBetween 截取 `func ... <name>(` 到 `endMarker` 之间的源码（endMarker 为下一个函数的签名）。
func funcBodyBetween(t *testing.T, src, name, endMarker string) string {
	t.Helper()
	i := strings.Index(src, "func (h *EzfyHandler) "+name+"(")
	if i < 0 {
		t.Fatalf("找不到 %s", name)
	}
	j := strings.Index(src[i:], endMarker)
	if j < 0 {
		t.Fatalf("找不到 %s 的结束边界（%s）", name, endMarker)
	}
	return src[i : i+j]
}
