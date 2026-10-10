package seed

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestEzfyItemSeedIDsOutOfEquipRange —— 防回归：道具种子的 cfg_id 必须落在 **1001+**，
// 不能再回到 1~35 —— 那一段是装备配置 `ezfy_cfg_equipment`（武器/防具/饰品/珠宝）的号段。
//
// 背景（线上事故 2026-10-10）：道具与装备两张表各自从 1 开始编号 → 数值 ID 重叠
// （30 = 建筑加速80% 也 = 黑曜石戒指）。任何「按 ID 区间猜身份」的代码都会误伤玩家数据
// （ezfyMigrateTreasureBag 按 27~35 把玩家真实加速道具当误发宝物删掉换成珠宝，每次部署清空一次）。
// 修复：道具全体 +1000（见 ezfy_item_id_migrate.go），本断言防止以后新增道具又取小号。
//
// 判据：种子里的道具行一定同时含 `ID:` 和 `ItemType:`（其它配置表用的是 Type/Level/…，不含 ItemType）。
func TestEzfyItemSeedIDsOutOfEquipRange(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("扫描源码失败: %v", err)
	}
	idRe := regexp.MustCompile(`\{ID:\s*(\d+),`)
	checked := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("读源码失败 %s: %v", f, err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if !strings.Contains(line, "ItemType:") {
				continue
			}
			m := idRe.FindStringSubmatch(line)
			if m == nil {
				t.Fatalf("%s:%d 含 ItemType: 却找不到 `{ID: N,`（判据失效，请更新本测试）：\n%s",
					f, i+1, line)
			}
			id, _ := strconv.Atoi(m[1])
			checked++
			if id < 1001 {
				t.Fatalf("%s:%d 道具种子 ID=%d 落在装备号段(1~35)内 —— 会与 ezfy_cfg_equipment 撞号：\n%s",
					f, i+1, id, line)
			}
		}
	}
	if checked == 0 {
		t.Fatal("一条道具种子都没扫到（判据 `ItemType:` 失效？），本断言已失去意义")
	}
	t.Logf("已校验 %d 条道具种子 ID 均 >= 1001", checked)
}

// TestEzfyShiftDropItemsJSON —— 野地「商城道具掉落」JSON 改号逻辑（纯函数，无需 DB）。
func TestEzfyShiftDropItemsJSON(t *testing.T) {
	cases := []struct {
		in        string
		want      string
		changed   bool
		validJSON bool
	}{
		// 老号 → +1000
		{`[[1,1,2]]`, `[[1001,1,2]]`, true, true},
		{`[[1,1,2],[19,1,2],[2,1,2],[13,1,2],[24,1,2]]`,
			`[[1001,1,2],[1019,1,2],[1002,1,2],[1013,1,2],[1024,1,2]]`, true, true},
		// 已经是新号 → 不动
		{`[[1028,1,2]]`, `[[1028,1,2]]`, false, true},
		// 超出旧号上限(41)的 id 不动（例如误填的大号）
		{`[[999,1,2]]`, `[[999,1,2]]`, false, true},
		// 历史文本值（不是 JSON）→ 原样返回、标记非法
		{`珠宝(平原)`, `珠宝(平原)`, false, false},
		{``, ``, false, false},
	}
	for _, c := range cases {
		got, changed, ok := ezfyShiftDropItemsJSON(c.in)
		if got != c.want || changed != c.changed || ok != c.validJSON {
			t.Fatalf("ezfyShiftDropItemsJSON(%q) = (%q, %v, %v)，期望 (%q, %v, %v)",
				c.in, got, changed, ok, c.want, c.changed, c.validJSON)
		}
	}
}
