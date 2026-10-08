package ezfy

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// TestCounterKillUsesEffectiveHp 「绝地反击」与普通攻击必须用**同一个有效生命**换算消灭数。
//
// ★★ 2026-10-08 用户反馈「为什么反击造成的伤害/战果比正常攻击还高」：
//
//	普通攻击用 `cur.hp`（有效生命 = 基础血量 ×(1+生命加成)），而反击原来用 `unit.cfg.Health`
//	（基础血量）→ 同一笔伤害下反击多消灭「1+生命加成」倍（实测 9067 → 15650，正好 1.73 倍）。
//	修复后：任意一行的「造成伤害 ÷ 消灭数」都必须恰好等于**被攻击方**的有效生命，
//	既不能是基础血量（135），也不能是理论伤害除以基础血量得出的混合值。
func TestCounterKillUsesEffectiveHp(t *testing.T) {
	saved := ezfyCfg.troops
	defer func() { ezfyCfg.troops = saved }()
	// 轰炸机：基础血量 135（与线上 ezfy_cfg_troop 一致）；射程给足，保证开局即可对射
	ezfyCfg.troops = map[int]model.EzfyCfgTroop{
		11: {ID: 11, Name: "轰炸机", Type: 3, Health: 135,
			AtkSea: 60, AtkGround: 90, AtkAir: 70, Defence: 40, Speed: 1200, AttackRange: 99999},
	}
	const baseHp = 135
	const atkLifeBonus = 73 // 攻方装备生命+73% → 攻方有效生命 = 135*1.73 = 233
	atkHp := baseHp * (100 + atkLifeBonus) / 100 // 233
	defHp := baseHp                              // 守方无生命加成

	st := ezfyNewBattleState([]ezfyUnitGroup{{TroopId: 11, Count: 200000}},
		[]ezfyUnitGroup{{TroopId: 11, Count: 200000}},
		634, 395, 575, 60, 50, // atkBonus, defBonus, defAtkBonus, atkSpeed, defSpeed
		30, 50, // atkRange, defRange
		ezfyBattleBonus{Hp: atkLifeBonus}, ezfyBattleBonus{},
		ezfyBattleBonus{}, ezfyBattleBonus{}, "", "",
		"Michael·Adams Lv.150 攻击加成+365% 防御加成+144%", "斯大林 Lv.350 攻击加成+325% 守军防御+100%",
		365, 325, 150, 180,
		nil, nil, nil, nil, nil,
		355, nil,
		map[int]int{}, map[int]int{}, map[int]int{}, map[int]int{},
		5, 6, 1, 2, // 双方都有「绝地反击」：攻方前 5 回合 / 守方前 6 回合
		ezfyTypeBonus{}, ezfyTypeBonus{})
	for i := 0; i < 3 && !st.Done; i++ {
		st.Step(nil, nil)
	}

	re := regexp.MustCompile(`造成(\d+)伤害, 消灭(\d+)个`)
	hitAtkLine, hitDefLine := 0, 0
	for _, ln := range st.Actions {
		m := re.FindStringSubmatch(ln)
		if m == nil {
			continue
		}
		dmg, _ := strconv.ParseInt(m[1], 10, 64)
		killed, _ := strconv.ParseInt(m[2], 10, 64)
		if killed <= 0 {
			t.Fatalf("消灭数为 0 的伤害行：%s", ln)
		}
		// 打攻方的行（守方攻击 / 守方反击）→ 有效生命 233；打守方的行 → 135
		targetsAtk := strings.Contains(ln, "攻方轰炸机")
		want := int64(defHp)
		who := "守方"
		if targetsAtk {
			want = int64(atkHp)
			who = "攻方"
		}
		if got := dmg / killed; got != want {
			t.Fatalf("「造成伤害÷消灭数」应等于%s有效生命 %d，实际 %d（%s）\n行：%s",
				who, want, got, ln, ln)
		}
		if targetsAtk {
			hitAtkLine++
		} else {
			hitDefLine++
		}
	}
	if hitAtkLine == 0 || hitDefLine == 0 {
		t.Fatalf("样本不足：打攻方 %d 行 / 打守方 %d 行（需要有攻击+反击两侧数据）", hitAtkLine, hitDefLine)
	}
	t.Logf("校验通过：打攻方 %d 行（有效生命 %d）/ 打守方 %d 行（有效生命 %d）",
		hitAtkLine, atkHp, hitDefLine, defHp)
	// 反击行必须存在（否则这条用例没覆盖到「绝地反击」）
	hasCounter := false
	for _, ln := range st.Actions {
		if strings.Contains(ln, "绝地反击") {
			hasCounter = true
			break
		}
	}
	if !hasCounter {
		t.Fatal("本场没有触发「绝地反击」，用例未覆盖反击路径")
	}
}
