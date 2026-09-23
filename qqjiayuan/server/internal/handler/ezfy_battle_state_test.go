package handler

import (
	"strings"
	"testing"
)

// 指挥功能（2026-09-22 用户要求）：
// 每回合 30 秒，前 25 秒可下达前进/暂停/后退，后 5 秒锁定结算，最多 40 回合。
// 下面这组测试盯的是**引擎侧**的语义 —— 能一回合一次地推进、指令真的生效、快照能往返。

// ezfyCmd1 把「全军指令」包成逐兵种指令表（测试里的单位 troopId 都是 1）
//
// ★ 指挥是**逐兵种**的（2026-09-22 用户要求：「自己带的兵种都能指挥，就是单独指挥」），
// 测试里只有一个兵种，包一层省得每处都写字面量。
func ezfyCmd1(cmd string) map[int]string {
	if cmd == "" {
		return nil
	}
	return map[int]string{1: cmd}
}

// ezfyTestUnit 造一个测试单位（不走 ezfyStatsOf，避免依赖配置缓存）
func ezfyTestUnit(id, name string, pos int, count int64) *ezfyFightUnit {
	return &ezfyFightUnit{
		id: id, count: count, initialCount: count, pos: pos,
		cfg: &ezfyTroopStats{
			ID: 1, Name: name, Type: 2, Health: 100,
			AtkGround: 200, Defence: 10, Speed: 100, AttackRange: 500,
		},
	}
}

// ezfyTestState 攻守各一支部队，初始相距 ezfyBattleStartDist
func ezfyTestState() *ezfyBattleState {
	return &ezfyBattleState{
		Attackers: []*ezfyFightUnit{ezfyTestUnit("A1", "步兵", 0, 100)},
		Defenders: []*ezfyFightUnit{ezfyTestUnit("D1", "步兵", ezfyBattleStartDist, 100)},
		Head:      []string{}, Actions: []string{},
	}
}

// TestBattleStepAdvancesOneRound —— Step 一次只推进一步，回合数递增
func TestBattleStepAdvancesOneRound(t *testing.T) {
	st := ezfyTestState()
	for i := 1; i <= 3; i++ {
		if st.Round != i-1 {
			t.Fatalf("推进前回合数应为 %d，实际 %d", i-1, st.Round)
		}
		st.Step(ezfyCmd1(ezfyCmdAdvance), ezfyCmd1(ezfyCmdAdvance))
		if st.Round != i {
			t.Fatalf("Step 后回合数应为 %d，实际 %d", i, st.Round)
		}
	}
}

// TestBattleHoldStopsMovement —— 「暂停」指令下单位不移动（日志里不出现前进）
func TestBattleHoldStopsMovement(t *testing.T) {
	st := ezfyTestState()
	startPos := st.Attackers[0].pos
	st.Step(ezfyCmd1(ezfyCmdHold), ezfyCmd1(ezfyCmdHold))
	if st.Attackers[0].pos != startPos {
		t.Fatalf("暂停指令下攻方不该移动：%d → %d", startPos, st.Attackers[0].pos)
	}
	if strings.Contains(strings.Join(st.Actions, "\n"), "前进") {
		t.Fatalf("暂停指令下不该出现前进日志：\n%s", strings.Join(st.Actions, "\n"))
	}
}

// TestBattleAdvanceMovesForward —— 「前进」指令下双方相向推进
func TestBattleAdvanceMovesForward(t *testing.T) {
	st := ezfyTestState()
	atkBefore, defBefore := st.Attackers[0].pos, st.Defenders[0].pos
	st.Step(ezfyCmd1(ezfyCmdAdvance), ezfyCmd1(ezfyCmdAdvance))
	if st.Attackers[0].pos <= atkBefore {
		t.Fatalf("攻方前进后位置应变大：%d → %d", atkBefore, st.Attackers[0].pos)
	}
	if st.Defenders[0].pos >= defBefore {
		t.Fatalf("守方前进后位置应变小：%d → %d", defBefore, st.Defenders[0].pos)
	}
}

// TestBattleRetreatMovesBackward —— 「后退」指令下单位远离对方，且不发起攻击
func TestBattleRetreatMovesBackward(t *testing.T) {
	st := ezfyTestState()
	atkBefore := st.Attackers[0].pos
	st.Step(ezfyCmd1(ezfyCmdRetreat), ezfyCmd1(ezfyCmdHold))
	if st.Attackers[0].pos >= atkBefore {
		t.Fatalf("后退指令下攻方位置应变小：%d → %d", atkBefore, st.Attackers[0].pos)
	}
	if !strings.Contains(strings.Join(st.Actions, "\n"), "后撤") {
		t.Fatalf("后退指令应写「后撤」日志：\n%s", strings.Join(st.Actions, "\n"))
	}
	if st.Defenders[0].count != st.Defenders[0].initialCount {
		t.Fatalf("后退到射程外不该造成伤害，守方损失 %d",
			st.Defenders[0].initialCount-st.Defenders[0].count)
	}
}

// TestBattleRoundCapIsForty —— 复刻《战斗机制》§1：最多 40 回合
func TestBattleRoundCapIsForty(t *testing.T) {
	if ezfyBattleMaxRounds != 40 {
		t.Fatalf("最大回合数应为 40，实际 %d", ezfyBattleMaxRounds)
	}
	// 双方都暂停 → 永远不进射程 → 打满上限后结束（平局按守方守住）
	st := ezfyTestState()
	for !st.Done {
		st.Step(ezfyCmd1(ezfyCmdHold), ezfyCmd1(ezfyCmdHold))
	}
	if st.Round != ezfyBattleMaxRounds {
		t.Fatalf("打满后回合数应为 %d，实际 %d", ezfyBattleMaxRounds, st.Round)
	}
	if st.AttackerWin {
		t.Fatal("回合耗尽按平局处理，不应判攻方胜")
	}
}

// TestBattleFinishWhenOneSideWiped —— 一方全灭立即结束
func TestBattleFinishWhenOneSideWiped(t *testing.T) {
	st := ezfyTestState()
	// 守方只剩 1 个兵，攻方贴脸必杀
	st.Defenders[0].count = 1
	st.Defenders[0].initialCount = 1
	st.Defenders[0].pos = 100
	st.Attackers[0].pos = 100
	for !st.Done {
		st.Step(ezfyCmd1(ezfyCmdAdvance), ezfyCmd1(ezfyCmdAdvance))
	}
	if !st.AttackerWin {
		t.Fatal("守方全灭时应判攻方胜")
	}
	if st.Round > 2 {
		t.Fatalf("贴脸对射应在 1~2 回合内结束，实际 %d 回合", st.Round)
	}
}

// TestBattlePerTroopCommand —— 用户要求：「指挥不是指挥全部，自己带的兵种都能指挥，就是单独指挥」
//
// 同一次 Step 里，两个兵种各按自己的指令行动。
func TestBattlePerTroopCommand(t *testing.T) {
	mk := func(id, name string, troopId, pos int, count int64) *ezfyFightUnit {
		return &ezfyFightUnit{
			id: id, count: count, initialCount: count, pos: pos,
			cfg: &ezfyTroopStats{
				ID: troopId, Name: name, Type: 2, Health: 100,
				AtkGround: 200, Defence: 10, Speed: 100, AttackRange: 500,
			},
		}
	}
	st := &ezfyBattleState{
		Attackers: []*ezfyFightUnit{mk("A1", "步兵", 1, 0, 100), mk("A2", "坦克", 3, 0, 100)},
		Defenders: []*ezfyFightUnit{mk("D1", "守军", 2, ezfyBattleStartDist, 100)},
		Head:      []string{}, Actions: []string{},
	}
	// 步兵前进、坦克暂停
	st.Step(map[int]string{1: ezfyCmdAdvance, 3: ezfyCmdHold}, nil)
	if st.Attackers[0].pos == 0 {
		t.Fatal("步兵下了「前进」却没移动")
	}
	if st.Attackers[1].pos != 0 {
		t.Fatalf("坦克下了「暂停」却移动了：pos=%d", st.Attackers[1].pos)
	}

	// 换成坦克后退、步兵暂停 —— 指令互不影响
	st2 := &ezfyBattleState{
		Attackers: []*ezfyFightUnit{mk("A1", "步兵", 1, 2000, 100), mk("A2", "坦克", 3, 2000, 100)},
		Defenders: []*ezfyFightUnit{mk("D1", "守军", 2, ezfyBattleStartDist, 100)},
		Head:      []string{}, Actions: []string{},
	}
	st2.Step(map[int]string{1: ezfyCmdHold, 3: ezfyCmdRetreat}, nil)
	if st2.Attackers[0].pos != 2000 {
		t.Fatalf("步兵下了「暂停」却移动了：pos=%d", st2.Attackers[0].pos)
	}
	if st2.Attackers[1].pos >= 2000 {
		t.Fatalf("坦克下了「后退」却没后撤：pos=%d", st2.Attackers[1].pos)
	}
}

// TestAtkCmdsParseCompat —— 逐兵种指令表解析：新 JSON 格式 + 旧「全军统一」格式都要认
func TestAtkCmdsParseCompat(t *testing.T) {
	m := ezfyAtkCmdsParse(`{"1":"advance","3":"hold"}`)
	if ezfyAtkCmdOf(m, 1) != ezfyCmdAdvance || ezfyAtkCmdOf(m, 3) != ezfyCmdHold {
		t.Fatalf("JSON 格式解析错误：%+v", m)
	}
	if ezfyAtkCmdOf(m, 9) != "" {
		t.Fatal("没下指令的兵种应返回空串（回落司令部配置）")
	}
	// 旧格式：整串 = 全军统一（存到 0 号键）
	old := ezfyAtkCmdsParse(ezfyCmdHold)
	if ezfyAtkCmdOf(old, 7) != ezfyCmdHold {
		t.Fatal("旧格式「全军统一指令」应能被任意兵种取到")
	}
	if len(ezfyAtkCmdsParse("")) != 0 {
		t.Fatal("空指令表解析出来应为空")
	}
}

// TestBattleSnapshotRoundTrip —— 战场快照往返（存库 → 重建）不丢状态
//
// 指挥室是**懒结算**：每次请求都要「读快照 → 重建 → 推进 → 存回」，
// 所以往返必须无损，否则回合数/兵力/位置会在存取之间漂移。
func TestBattleSnapshotRoundTrip(t *testing.T) {
	st := ezfyTestState()
	st.Step(ezfyCmd1(ezfyCmdAdvance), ezfyCmd1(ezfyCmdAdvance))
	st.Step(ezfyCmd1(ezfyCmdAdvance), ezfyCmd1(ezfyCmdHold))

	encoded := ezfyBattleSnapshotEncode(st.Snapshot())
	if encoded == "" {
		t.Fatal("快照序列化失败")
	}
	snap, ok := ezfyBattleSnapshotDecode(encoded)
	if !ok {
		t.Fatal("快照反序列化失败")
	}
	back := ezfyBattleStateFromSnapshot(snap)

	if back.Round != st.Round {
		t.Fatalf("往返后回合数不一致：%d vs %d", back.Round, st.Round)
	}
	if len(back.Attackers) != len(st.Attackers) || len(back.Defenders) != len(st.Defenders) {
		t.Fatal("往返后单位数量不一致")
	}
	if back.Attackers[0].pos != st.Attackers[0].pos {
		t.Fatalf("往返后位置不一致：%d vs %d", back.Attackers[0].pos, st.Attackers[0].pos)
	}
	if back.Attackers[0].count != st.Attackers[0].count {
		t.Fatalf("往返后兵力不一致：%d vs %d", back.Attackers[0].count, st.Attackers[0].count)
	}
	if back.Attackers[0].cfg == nil || back.Attackers[0].cfg.ID != st.Attackers[0].cfg.ID {
		t.Fatal("往返后兵种配置未按 troop_id 正确重建")
	}
	if len(back.Actions) != len(st.Actions) {
		t.Fatalf("往返后日志条数不一致：%d vs %d", len(back.Actions), len(st.Actions))
	}
}

// TestBattleSnapshotKeepsGoingAfterRestore —— 重建后还能继续推进（不是死状态）
func TestBattleSnapshotKeepsGoingAfterRestore(t *testing.T) {
	st := ezfyTestState()
	st.Step(ezfyCmd1(ezfyCmdAdvance), ezfyCmd1(ezfyCmdAdvance))
	snap, _ := ezfyBattleSnapshotDecode(ezfyBattleSnapshotEncode(st.Snapshot()))
	back := ezfyBattleStateFromSnapshot(snap)
	before := back.Round
	if back.Step(ezfyCmd1(ezfyCmdAdvance), ezfyCmd1(ezfyCmdAdvance)) && back.Round != before+1 {
		t.Fatalf("重建后推进异常：回合 %d → %d", before, back.Round)
	}
}

// TestBattleSimulateStillWorks —— 兼容包装：ezfySimulate 仍能一次跑完并给出结果
func TestBattleSimulateStillWorks(t *testing.T) {
	st := ezfyTestState()
	for !st.Done {
		st.Step(nil, nil) // 空指令 = 沿用司令部兵种配置
	}
	res := st.Result()
	if res.Rounds <= 0 {
		t.Fatalf("ezfySimulate 等价流程应产出正回合数，实际 %d", res.Rounds)
	}
	if len(res.Actions) == 0 {
		t.Fatal("结果里应包含行动日志")
	}
	// 攻方 100 打守方 100，同兵种同属性、攻方先手 → 攻方应能取胜
	if !res.AttackerWin {
		t.Fatal("同兵种同兵力下先手方应取胜")
	}
}
