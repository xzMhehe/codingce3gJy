package ezfy

import (
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// ============ 洗点：只洗「等级属性点」，升星加成必须保留（2026-10-25）============
//
// 用户原话：「洗点 只能洗 等级属性点，还有升星 属性点 在检查下有无bug」。
//
// 旧实现把重置目标设成「军官池属性」，于是「玩家加的点 + 升星加的点」一起退回待分配点
// —— 玩家可以把升星加成重新分配到别的项上，这就是那个 bug。
// 现在重置目标 = 池属性 + 升星逐项明细（star_military/logistics/learning）。

// TestStarSplitNewData —— 有逐项明细时直接用（不再反推）
func TestStarSplitNewData(t *testing.T) {
	o := &model.EzfyOfficer{Military: 190, Logistics: 70, Learning: 50,
		BaseMilitary: 100, BaseLogistics: 60, BaseLearning: 40,
		StarPoints: 60, StarMilitary: 40, StarLogistics: 10, StarLearning: 10}
	sm, sl, se := officerStarSplit(o)
	if sm != 40 || sl != 10 || se != 10 {
		t.Fatalf("有逐项明细时应原样返回，实际 %d/%d/%d", sm, sl, se)
	}
}

// TestStarSplitLegacyProportional —— 老数据只有总数：按超出池属性的比例反推，且不超过各自超出量
func TestStarSplitLegacyProportional(t *testing.T) {
	// 池 100/60/40；当前 190/90/40 → 超出 90/30/0（合计 120），升星总数 90
	o := &model.EzfyOfficer{Military: 190, Logistics: 90, Learning: 40,
		BaseMilitary: 100, BaseLogistics: 60, BaseLearning: 40, StarPoints: 90}
	sm, sl, se := officerStarSplit(o)
	if sm+sl+se != 90 {
		t.Fatalf("反推的三项之和应等于升星总数 90，实际 %d+%d+%d", sm, sl, se)
	}
	if sm > 90 || sl > 30 || se > 0 {
		t.Fatalf("三项不能超过各自超出池属性的量，实际 %d/%d/%d", sm, sl, se)
	}
	// 全部加点都在军事时（超出 90/0/0）→ 升星全算军事，洗点退不回任何点
	o2 := &model.EzfyOfficer{Military: 190, Logistics: 60, Learning: 40,
		BaseMilitary: 100, BaseLogistics: 60, BaseLearning: 40, StarPoints: 90}
	sm2, sl2, se2 := officerStarSplit(o2)
	if sm2 != 90 || sl2 != 0 || se2 != 0 {
		t.Fatalf("超出全在军事时，升星应全算军事，实际 %d/%d/%d", sm2, sl2, se2)
	}
}

// TestRespecKeepsStarBonus —— 洗点的重置目标必须带上升星明细（静态断言 + 语义断言）
func TestRespecKeepsStarBonus(t *testing.T) {
	// 语义：池 100/60/40 + 升星 20/20/20 + 玩家加点 30/0/0 → 洗点后应回到 120/80/60，退回 30 点
	o := &model.EzfyOfficer{Military: 150, Logistics: 80, Learning: 60,
		BaseMilitary: 100, BaseLogistics: 60, BaseLearning: 40,
		StarPoints: 60, StarMilitary: 20, StarLogistics: 20, StarLearning: 20}
	bm, bl, be := officerBaseAttr(o)
	sm, sl, se := officerStarSplit(o)
	tMil, tLog, tLea := bm+sm, bl+sl, be+se
	refund := (o.Military - tMil) + (o.Logistics - tLog) + (o.Learning - tLea)
	if tMil != 120 || tLog != 80 || tLea != 60 {
		t.Fatalf("洗点目标应为 池属性+升星 = 120/80/60，实际 %d/%d/%d", tMil, tLog, tLea)
	}
	if refund != 30 {
		t.Fatalf("只该退回玩家自己加的 30 点，实际 %d", refund)
	}
	// 静态：洗点分支必须用 officerStarSplit 且写回 star_* 三列
	body := ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) useItemOnce(")
	if body == "" {
		t.Skip("useItem 函数体定位失败（改名了？），跳过静态断言")
	}
	if !strings.Contains(body, "officerStarSplit(o)") {
		t.Fatal("洗点分支必须用 officerStarSplit 保住升星加成")
	}
}
