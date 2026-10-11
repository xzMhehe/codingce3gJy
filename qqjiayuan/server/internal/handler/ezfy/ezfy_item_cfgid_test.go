package ezfy

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestNoHardcodedItemCfgID —— 防回归：handler 里**不得再硬编码道具 cfg_id**。
//
// 背景（线上事故 2026-10-10）：道具 ezfy_cfg_item 的 cfg_id 全体 +1000（1~41 → 1001~1041，
// 见 seed/ezfy_cfg_gen.go）。但 handler 里多处仍写死老号（建筑图纸 10 / 军官技能书 15 /
// 改名卡 17 / 军官改名卡 25 / 星级徽章 23 / 招生简章 13 / 迁城计划 20·21·22），
// 迁移后这些老号查不到 → 持有量恒 0 → 玩家反馈「背包里明明有道具，却提示没有」：
// 迁城、改名、学技能、升 10 级建筑全部哑掉。
//
// 规则：凡「认某一种固定道具」一律**按名字解析**（`ezfyCfg.itemIDByName`，见 ezfy_geo.go），
// 别写数字 ID。本断言扫描包内全部非测试 .go，发现 `ezfyXxxItemYyy = <数字>` 这类常量即失败。
//
// 允许：① 名字含 ItemType 的（那是 item_type 值 16/17/18…，不是 cfg_id）；
//
//	② 名字以 Def 结尾的默认值常量（如 ezfyBagItemMaxDef）；③ 注释里提到历史常量。
func TestNoHardcodedItemCfgID(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("扫描源码失败: %v", err)
	}
	re := regexp.MustCompile(`\b(ezfy\w*Item\w*)\s*=\s*(\d+)\b`)
	total := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("读源码失败 %s: %v", f, err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue // 注释里提到历史常量（如 `const ezfySchemeItemID = 24`）不算
			}
			for _, m := range re.FindAllStringSubmatch(line, -1) {
				total++
				name := m[1]
				if strings.Contains(name, "ItemType") {
					continue // item_type 值，不是 cfg_id
				}
				if strings.HasSuffix(name, "Def") {
					continue // 默认值常量
				}
				t.Fatalf("%s:%d 硬编码了道具 cfg_id（%s = %s）——\n"+
					"道具 cfg_id 已统一迁到 1001+，请改为按名字解析（ezfyCfg.itemIDByName）：\n%s",
					f, i+1, name, m[2], line)
			}
		}
	}
	if total == 0 {
		t.Fatal("一条 `ezfyXxxItemYyy = 数字` 都没扫到（判据失效？），本断言已失去意义")
	}
	t.Logf("已扫描 %d 处 item 相关常量，均未硬编码道具 cfg_id", total)
}
