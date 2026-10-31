package seed

// ★★ 2026-10-31 用户要求：线上 31 名名将**全部配置**组合技、每名**最多 1 个**，1/2 区一致。
// 这里守住「初始搭配」这张表：31 个 ID 一个不少、值必须是池子里存在的技能 ID。
// （map[int]int 本身保证每名最多一个；线上两区已按同一张表刷过。）

import "testing"

func TestDefaultComboCoversAllGenerals(t *testing.T) {
	if len(ezfyDefaultComboOfGeneral) != 31 {
		t.Fatalf("默认搭配应覆盖 31 名名将，实际 %d 条", len(ezfyDefaultComboOfGeneral))
	}
	skillOK := map[int]bool{}
	for _, s := range ezfyGeneralSkillSeeds {
		skillOK[s.ID] = true
	}
	for gid := 1; gid <= 31; gid++ {
		sid, ok := ezfyDefaultComboOfGeneral[gid]
		if !ok {
			t.Fatalf("名将 id=%d 没有默认组合技", gid)
		}
		if !skillOK[sid] {
			t.Fatalf("名将 id=%d 配了不存在的组合技 id=%d", gid, sid)
		}
	}
}

// TestDefaultComboUsesMostPool 31 名名将只配 28 个技能，允许少量复用；
// 这里只守「不要退化成只配 10 个」——至少覆盖 25 个不同技能。
func TestDefaultComboUsesMostPool(t *testing.T) {
	used := map[int]bool{}
	for _, sid := range ezfyDefaultComboOfGeneral {
		used[sid] = true
	}
	if len(used) < 25 {
		t.Fatalf("默认搭配只用到 %d 个组合技，覆盖过少（期望 ≥25）", len(used))
	}
}
