package ezfy

import (
	"strings"
	"testing"
)

// ============ 司令部「兵种战斗配置 → 防守数量」+「防守=后退」守卫 ============
//
// 用户规则（2026-10-28）：
//   - 每个兵种可单独设置「防守数量」，默认全部；玩家自定义，上限 = 系统配置「单城兵力上限」；
//   - 实际参战 = min(城内该兵种数量, 配置)：城内多于配置按配置打，少于配置按实际打；
//   - 该配置是**城市维度**（ezfy_city_target.city_id），每座城各一套，不是账号一套；
//   - 「防守」下拉新增「后退」，原「不参与防御」文案改成「不参与」，功能（def_move=-1）不变。
//
// 这几条守卫分别盯：夹取 / 战斗取值 / 城市维度 / 指令取值不撞车。规则再改时同步这里。

// TestDefCountOf 参战兵力 = min(城内, 配置)，配置 0/缺省 = 全部
func TestDefCountOf(t *testing.T) {
	caps := map[int]int64{11: 100, 12: 500}
	cases := []struct {
		name  string
		count int64
		tid   int
		want  int64
	}{
		{"城内多于配置 → 按配置", 1000, 11, 100},
		{"城内少于配置 → 按实际", 50, 11, 50},
		{"刚好等于配置", 100, 11, 100},
		{"没配 → 全部", 777, 13, 777},
		{"配置为 0 → 全部", 777, 12 - 12, 777}, // 0 号兵种没配
		{"城内为 0 → 0", 0, 11, 0},
		{"城内为负（脏数据）→ 0", -5, 11, 0},
	}
	for _, c := range cases {
		if got := defCountOf(c.count, caps, c.tid); got != c.want {
			t.Fatalf("%s: defCountOf(%d, caps, %d) = %d, want %d", c.name, c.count, c.tid, got, c.want)
		}
	}
	if got := defCountOf(777, nil, 11); got != 777 {
		t.Fatalf("caps 为 nil（没取到配置）时应当按全部打，got %d", got)
	}
}

// TestDefenderBuildAppliesDefCount —— 被攻击时守军必须按「防守数量」封顶
func TestDefenderBuildAppliesDefCount(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) processArrive(")
	if !strings.Contains(body, "defCaps = h.defCapMap(target.ID)") {
		t.Fatal("processArrive 必须取目标城的「兵种防守数量」配置（defCapMap）")
	}
	if !strings.Contains(body, "defCountOf(count, defCaps, tid)") {
		t.Fatal("构建守军时必须用 defCountOf 按「防守数量」封顶，否则配置形同虚设")
	}
	// 侦察机拦截空战同样走这个口径
	if !strings.Contains(body, "defCountOf(defMap[ezfyReconPlaneTroopID], defCaps, ezfyReconPlaneTroopID)") {
		t.Fatal("侦察机拦截空战也要按「防守数量」封顶（与主战场同口径）")
	}
}

// TestSaveTargetClampsDefCount —— 保存时服务端自己夹取（手机端可能粘贴大数）
func TestSaveTargetClampsDefCount(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_api.go", "func (h *EzfyHandler) SaveTarget(")
	for _, want := range []string{
		"def_count",         // 入参
		"req.DefCount < 0",  // 负数 → 0（全部）
		"ezfyTroopMaxCfg()", // 上限 = 系统配置「单城兵力上限」
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("SaveTarget 必须夹取防守数量：缺少 `%s`", want)
		}
	}
	// 保存失败必须回报（缺列 1054 的老坑：静默失败 = 玩家以为存上了）
	if !strings.Contains(body, "战斗配置保存失败") {
		t.Fatal("SaveTarget 的 Create/Updates 失败必须回错，不能忽略 error（缺列时就是静默失败）")
	}
}

// TestTargetsIsCityScoped —— 兵种战斗配置是城市维度：支持 ?city_id=，且按城市查
func TestTargetsIsCityScoped(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_api.go", "func (h *EzfyHandler) Targets(")
	if !strings.Contains(body, `c.Query("city_id")`) {
		t.Fatal("Targets 必须支持 ?city_id= 显式指定城市（多城时不能只认当前城）")
	}
	if !strings.Contains(body, "city_id = ?") {
		t.Fatal("Targets 必须按 city_id 查配置（每城一套）")
	}
}

// TestTargetDropdownUsesBaseTroopName —— 目标下拉用**基础兵种名**，不用阵营名
//
// 用户规则（2026-10-28）：「进攻目标 下拉改成 兵种名字，别用阵营名字」。
//
//	下拉里选的是**敌方兵种类型**（敌方可能是另一个阵营），用「虎式重型坦克」去指代
//	「重型坦克」会让人对不上；与战场指挥室 target_options 同口径（那边 2026-10-05 已统一）。
func TestTargetDropdownUsesBaseTroopName(t *testing.T) {
	army := ezfyStripCommentLines(readFrontFile(t, "../../../../web/src/views/ezfy/modules/EzfyArmy.vue"))
	if !strings.Contains(army, "{{ tt.base_name || tt.name }}") {
		t.Fatal("司令部「进攻/防守目标」下拉必须用基础兵种名（tt.base_name），不能直接用阵营名（tt.name）")
	}
	body := ezfyFuncBody(t, "ezfy_api.go", "func (h *EzfyHandler) Troops(")
	if !strings.Contains(body, `"base_name": t.Name`) {
		t.Fatal("/troops 必须下发 base_name（基础兵种名），否则前端拿不到不带阵营色彩的名字")
	}
}

// TestDefMoveBackAndNone —— 后退(-2) 与 避战(-1) 必须区分开
func TestDefMoveBackAndNone(t *testing.T) {
	if ezfyDefMoveNone != -1 {
		t.Fatalf("「不参与」的值必须保持 -1（线上存量数据），实际 %d", ezfyDefMoveNone)
	}
	if ezfyDefMoveBack != -2 {
		t.Fatalf("「后退」的值必须是 -2，实际 %d", ezfyDefMoveBack)
	}
	// 引擎：moveMap 值 <0 一律后退 → -2 天然生效；而 defExcludeSet 只认 -1，
	// 所以「后退」的兵种照常参战，只是每回合往后撤。
	if got := ezfyMoveDir(&ezfyFightUnit{cfg: &ezfyTroopStats{ID: 11}}, map[int]int{11: ezfyDefMoveBack}, ""); got != -1 {
		t.Fatalf("防守=后退(-2) 时引擎方向应为 -1（后退），实际 %d", got)
	}
	if got := ezfyMoveDir(&ezfyFightUnit{cfg: &ezfyTroopStats{ID: 11}}, map[int]int{11: 1}, ""); got != 1 {
		t.Fatalf("防守=前进(1) 时引擎方向应为 1，实际 %d", got)
	}
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) defExcludeSet(")
	if !strings.Contains(body, "ezfyDefMoveNone") || strings.Contains(body, "ezfyDefMoveBack") {
		t.Fatal("defExcludeSet 只能排除「不参与」(-1)，不能把「后退」(-2) 也排除掉")
	}
}
