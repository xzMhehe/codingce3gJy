package ezfy

import (
	"testing"

	"qqjiayuan/server/internal/model"
)

// TestTypeDefTxtHiddenWhenTroopAbsent —— 「兵种专属防御(城防+N%)」**该方没有这个兵种就不展示**。
//
// ★★ 2026-10-11 用户口径：「假如当前兵种不是城防兵种就别展示，不然容易有歧义」。
//
//	玩家反馈「当时没有城防单位，却展示了城防+20%」→ 排查线上战场 83068 时发现守方确实带了
//	防空炮(type=4)，但过滤写成 `len(present) > 0 && !present[k]` 有个兜底：
//	**present 为空(nil/空 map)时会「不过滤、全部列出」** → 真的没带城防兵也会显示 20%。
func TestTypeDefTxtHiddenWhenTroopAbsent(t *testing.T) {
	tb := ezfyTypeBonus{Def: map[int]int{ezfyTroopTypeCity: 20}}

	cases := []struct {
		name    string
		present map[int]bool
		want    string
	}{
		{"present=nil（无兵种信息）", nil, ""},
		{"present=空 map", map[int]bool{}, ""},
		{"present={海,陆,空}（没有城防兵）", map[int]bool{1: true, 2: true, 3: true}, ""},
		{"present={城防}", map[int]bool{ezfyTroopTypeCity: true}, " 兵种专属防御(城防+20%)"},
	}
	for _, c := range cases {
		if got := ezfyTypeDefTxt(tb, c.present); got != c.want {
			t.Errorf("%s：期望 %q，实际 %q", c.name, c.want, got)
		}
	}

	// 速度同理（喷气引擎=空军速度，没带空军就不该展示）
	sp := ezfyTypeBonus{Speed: map[int]int{ezfyTroopTypeAir: 30}}
	if got := ezfyTypeSpeedTxt(sp, map[int]bool{}); got != "" {
		t.Errorf("没有空军时不应展示兵种专属速度，实际 %q", got)
	}
	if got := ezfyTypeSpeedTxt(sp, map[int]bool{ezfyTroopTypeAir: true}); got == "" {
		t.Error("带空军时应展示兵种专属速度")
	}
}

// TestCityDefBonusOnlyAppliesToCityTroops —— 不展示的那一截**也不能偷偷给加成**。
//
// ★★ 2026-10-11 用户口径：「不显示的也不要给防御加成哈」。
//
//	掩体防御(16) 改成「只对城防单位生效」后，守方这一截按**被攻击单位的兵种 type** 取
//	（`st.DefType.defOf(target.cfg.Type)`）：没带城防兵时任何单位都吃不到，
//	与「战斗加成行不展示这一截」保持一致 —— 不允许「不显示却生效」。
func TestCityDefBonusOnlyAppliesToCityTroops(t *testing.T) {
	saved := ezfyCfg.troops
	defer func() { ezfyCfg.troops = saved }()
	ezfyCfg.troops = map[int]model.EzfyCfgTroop{
		1:  {ID: 1, Name: "步兵", Type: 2, Health: 100, AtkGround: 20, Defence: 10, Speed: 500, AttackRange: 100},
		17: {ID: 17, Name: "碉堡", Type: 4, Health: 200, AtkGround: 30, Defence: 50, Speed: 0, AttackRange: 500},
	}
	cityDef := ezfyTypeBonus{Def: map[int]int{ezfyTroopTypeCity: 20}}

	defLoss := func(defUnits []ezfyUnitGroup, defType ezfyTypeBonus) int64 {
		r := ezfySimulate([]ezfyUnitGroup{{TroopId: 1, Count: 100000}}, defUnits,
			0, 0, 0, 0, 0, 0, 0,
			ezfyBattleBonus{}, ezfyBattleBonus{},
			"", "", 0, 0,
			map[int]int{}, map[int]int{}, map[int]int{}, map[int]int{},
			ezfyBattleExtra{DefType: defType})
		var n int64
		for _, g := range r.DefenderLosses {
			n += g.Count
		}
		return n
	}

	// ① 守方全是陆军(type=2)：这一截对它不生效 → 加不加 20% 损失必须完全一样
	army := []ezfyUnitGroup{{TroopId: 1, Count: 100000}}
	a0, a1 := defLoss(army, ezfyTypeBonus{}), defLoss(army, cityDef)
	if a0 != a1 {
		t.Fatalf("守方没有城防兵却吃到了城防专属防御：无加成损失 %d / 有加成损失 %d", a0, a1)
	}
	if a0 == 0 {
		t.Fatal("用例无效：守方损失为 0，无法判断是否加成")
	}
	t.Logf("守方全陆军：无加成 %d = 有加成 %d ✓（不生效）", a0, a1)

	// ② 守方带城防兵(type=4)：这一截生效 → 两次损失必须不同（证明用例有区分度）
	withCity := []ezfyUnitGroup{{TroopId: 17, Count: 100000}}
	c0, c1 := defLoss(withCity, ezfyTypeBonus{}), defLoss(withCity, cityDef)
	if c0 == c1 {
		t.Fatalf("守方带城防兵，城防专属防御却没生效：无加成 %d / 有加成 %d", c0, c1)
	}
	t.Logf("守方带碉堡：无加成 %d ≠ 有加成 %d ✓（生效）", c0, c1)
}
