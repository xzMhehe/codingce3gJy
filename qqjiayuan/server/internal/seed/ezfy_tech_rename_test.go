package seed

import (
	"testing"

	"qqjiayuan/server/internal/ezfyreq"
)

// ★ 2026-10-31 科技改名护栏（规避原版名称版权）。
//
// 背景：`ezfy_cfg_troop.require` 是**展示文本**，但它是 require_ids 的解析来源
// （internal/ezfyreq.Parse 按名字查 id）。训练校验本身已改为只读 require_ids（按 ID），
// 所以名字对不上不会让门槛「被绕过」，但会让这条要求**写不进结构化 ID**（少一条）。
// 结论一样：「科技名」与「兵种 require 里引用的名字」必须严格一致。
//
// 本测试守住三条：
//  1. 科技名非空、互不重复；
//  2. 科技名不与建筑名撞车（否则 require 里同名时会被优先当建筑解析）；
//  3. require 里出现的每一个「名字(n级)」都必须是已存在的建筑或科技。
func TestEzfyTechRenameConsistency(t *testing.T) {
	buildingIDs := map[string]int{}
	for _, b := range ezfyEzfyCfgBuilding {
		buildingIDs[b.Name] = b.ID
	}

	techIDs := map[string]int{}
	for _, tech := range ezfyEzfyCfgTech {
		if tech.Name == "" {
			t.Fatalf("科技 id=%d 名字为空", tech.ID)
		}
		if _, dup := techIDs[tech.Name]; dup {
			t.Fatalf("科技名重复：%s（科技名必须唯一，否则按名反查 id 会错配）", tech.Name)
		}
		techIDs[tech.Name] = tech.ID
		if _, clash := buildingIDs[tech.Name]; clash {
			t.Fatalf("科技名「%s」与建筑名撞车：require 解析会优先命中建筑", tech.Name)
		}
	}

	// 用与线上**同一份**解析实现（internal/ezfyreq），别再自己编译一份正则
	for _, tr := range ezfyEzfyCfgTroop {
		if tr.Require == "" {
			continue
		}
		refs, unknown := ezfyreq.Parse(tr.Require, buildingIDs, techIDs)
		if len(unknown) > 0 {
			t.Fatalf("兵种 id=%d（%s）的 require 引用了不存在的名字 %v：\n"+
				"  建筑/科技改名后必须同步 ezfy_cfg_troop.require，否则这条要求写不进 require_ids。\n"+
				"  当前 require = %s", tr.ID, tr.Name, unknown, tr.Require)
		}
		if len(refs) == 0 {
			t.Fatalf("兵种 id=%d（%s）的 require 一个条件都解析不出来：%s", tr.ID, tr.Name, tr.Require)
		}
	}
}
