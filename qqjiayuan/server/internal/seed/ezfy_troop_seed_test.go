package seed

import (
	"testing"

	"qqjiayuan/server/internal/model"
)

// TestEzfyNavalBomberTroop 新增兵种「海军轰炸机」(id 21) 的配置防回归。
//
// ★ 2026-10-22 用户需求：按原版兵种表（兵种.png 第12行）新增海军轰炸机，
// 训练时间改成 1分30秒，其余 20 个兵种一律不动。
//
// 这里锁住三件容易改错的事：
//   - id 必须是 21（1~20 是老兵种固定 id，玩家已训部队按 troop_id 落库，不能挤占）；
//   - 「对城墙」那一列就是对防 → AtkDef=40（用户特别提醒过）；
//   - 训练时间 90 秒（原版 130 秒）。
func TestEzfyNavalBomberTroop(t *testing.T) {
	var got *model.EzfyCfgTroop
	for i := range ezfyEzfyCfgTroop {
		if ezfyEzfyCfgTroop[i].ID == ezfyNavalBomberTroopID {
			got = &ezfyEzfyCfgTroop[i]
			break
		}
	}
	if got == nil {
		t.Fatalf("兵种种子里找不到海军轰炸机(id=%d)", ezfyNavalBomberTroopID)
	}
	if got.Name != "海军轰炸机" {
		t.Errorf("通用名 = %q, want 海军轰炸机", got.Name)
	}
	if got.NameAlly == "" || got.NameAxis == "" {
		t.Errorf("阵营名必须都填（同盟国=%q 轴心国=%q）", got.NameAlly, got.NameAxis)
	}
	if got.Type != 3 {
		t.Errorf("Type = %d, want 3(空军)", got.Type)
	}
	if got.TrainTime != 90 {
		t.Errorf("训练时间 = %d 秒, want 90（1分30秒）", got.TrainTime)
	}
	// 原版兵种表数值（兵种.png 第12行；「对城墙」= 对防 AtkDef）
	cases := []struct {
		name string
		got  int
		want int
	}{
		{"生命", got.Health, 130},
		{"防御", got.Defence, 21},
		{"对地", got.AtkGround, 25},
		{"对空", got.AtkAir, 15},
		{"对海", got.AtkSea, 75},
		{"对城墙(对防)", got.AtkDef, 40},
		{"射程", got.AttackRange, 100},
		{"速度", got.Speed, 1500},
		{"负重", got.Carry, 50},
		{"油耗", got.OilKeep, 100},
		{"人口", got.Pop, 10},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
	// 造价与建造条件必须填全（空的话训练页会显示 0 消耗 / 无门槛）
	if got.Food <= 0 || got.Steel <= 0 || got.Oil <= 0 || got.Rare <= 0 {
		t.Errorf("造价没填全: 粮%d 钢%d 油%d 稀矿%d", got.Food, got.Steel, got.Oil, got.Rare)
	}
	if got.Require == "" {
		t.Error("Require 不能为空")
	}
	if got.Icon == "" {
		t.Error("Icon 不能为空")
	}
}

// TestEzfyTroopSeedIDsUnique 兵种 id 不能重复，且 1~20 老兵种必须原样都在。
func TestEzfyTroopSeedIDsUnique(t *testing.T) {
	seen := map[int]string{}
	for _, tr := range ezfyEzfyCfgTroop {
		if prev, ok := seen[tr.ID]; ok {
			t.Fatalf("兵种 id=%d 重复：%q 与 %q", tr.ID, prev, tr.Name)
		}
		seen[tr.ID] = tr.Name
	}
	// 老兵种 1~20 必须都在（新增兵种不能删/改号）
	for id := 1; id <= 20; id++ {
		if _, ok := seen[id]; !ok {
			t.Fatalf("老兵种 id=%d 不见了（新增兵种不能改动 1~20）", id)
		}
	}
}
