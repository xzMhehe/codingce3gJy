package seed

import "testing"

// TestPoolStar5SideAttrsRule —— 五星普通军官：主属性 >=170 时，另外两项必须落在 [60,100]
//
// ★ 2026-09-26 用户规则（阈值先定 180，同日改成 170）。主属性按 军事 > 后勤 > 学识 优先级确定。
func TestPoolStar5SideAttrsRule(t *testing.T) {
	n, hit := 0, 0
	for _, o := range buildEzfyPoolOfficers() {
		if o.Star != 5 {
			continue
		}
		n++
		mil, log, lea := o.Military, o.Logistics, o.Learning
		check := func(main int, a, b int, which string) {
			hit++
			if a > 100 || a < 60 || b > 100 || b < 60 {
				t.Fatalf("%s 主属性(%s)=%d，但另两项 %d/%d 不在 [60,100]",
					o.Name, which, main, a, b)
			}
		}
		switch {
		case mil >= 170:
			check(mil, log, lea, "军事")
		case log >= 170:
			check(log, mil, lea, "后勤")
		case lea >= 170:
			check(lea, mil, log, "学识")
		}
	}
	t.Logf("五星 %d 名，其中 %d 名触发了主属性保护规则", n, hit)
}
