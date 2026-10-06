package ezfy

import "testing"

// 计谋伪装假数量（2026-10-06 恫疑虚喝/隐真示假）：
//   - 恫疑虚喝(fakeKind=3)：固定 1亿（100000000）吓唬敌人；
//   - 隐真示假(fakeKind=4)：1000 内随机小数量，隐藏实力。
func TestEzfySchemeFakeCount(t *testing.T) {
	if c := ezfySchemeFakeCount(3); c != 100000000 {
		t.Fatalf("恫疑虚喝假数量应为固定1亿(100000000), 实际 %d", c)
	}
	for i := 0; i < 1000; i++ {
		c := ezfySchemeFakeCount(4)
		if c < 1 || c > 1000 {
			t.Fatalf("隐真示假假数量应落在[1,1000], 实际 %d", c)
		}
	}
}
