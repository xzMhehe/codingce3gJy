package ezfyreq

import (
	"strings"
	"testing"
)

var (
	testBuildings = map[string]int{"军工厂": 14, "停机坪": 18, "航海协会": 19, "围墙": 7}
	testTechs     = map[string]int{"枪械改良": 6, "增程弹药": 8, "动员训练": 5, "合金冶炼": 2}
)

// TestParseRequireText 文本解析：半角/全角括号都要吃，名字必须命中建筑或科技。
func TestParseRequireText(t *testing.T) {
	cases := []struct {
		text  string
		want  string
		kinds []byte
	}{
		{"军工厂(8级)枪械改良(8级)", "b14:8,t6:8", []byte{KindBuilding, KindTech}},
		{"围墙（6级）枪械改良(8级)永备工事（5级）", "b7:6,t6:8", []byte{KindBuilding, KindTech}}, // 永备工事不在测试表 → unknown
		{"军工厂(1级)", "b14:1", []byte{KindBuilding}},
		{"", "", nil},
	}
	for _, c := range cases {
		refs, unknown := Parse(c.text, testBuildings, testTechs)
		got := Format(refs)
		if c.text == "围墙（6级）枪械改良(8级)永备工事（5级）" {
			// 这条专门验 unknown 行为，不做 Format 全等
			if len(unknown) != 1 || unknown[0] != "永备工事" {
				t.Fatalf("%q 的 unknown = %v，期望 [永备工事]", c.text, unknown)
			}
			if got != "b7:6,t6:8" {
				t.Fatalf("%q 解析出 %q，期望 b7:6,t6:8", c.text, got)
			}
			continue
		}
		if got != c.want {
			t.Fatalf("%q 解析出 %q，期望 %q", c.text, got, c.want)
		}
		if len(unknown) != 0 {
			t.Fatalf("%q 不该有 unknown，得到 %v", c.text, unknown)
		}
	}
}

// TestParseIDsRoundTrip 结构化字符串往返：Format → ParseIDs 必须一模一样。
func TestParseIDsRoundTrip(t *testing.T) {
	refs := []Ref{
		{Kind: KindBuilding, ID: 14, Level: 8},
		{Kind: KindTech, ID: 6, Level: 8},
		{Kind: KindTech, ID: 12, Level: 1},
	}
	s := Format(refs)
	if s != "b14:8,t6:8,t12:1" {
		t.Fatalf("Format = %q", s)
	}
	back := ParseIDs(s)
	if len(back) != len(refs) {
		t.Fatalf("ParseIDs 条数 = %d，期望 %d（%q）", len(back), len(refs), s)
	}
	for i := range refs {
		if back[i] != refs[i] {
			t.Fatalf("第 %d 条 = %+v，期望 %+v", i, back[i], refs[i])
		}
	}
}

// TestParseIDsTolerant 脏数据（缺冒号/非数字/未知 kind/空段）只丢段，不 panic。
func TestParseIDsTolerant(t *testing.T) {
	got := ParseIDs("b14:8,,x9:1,b:t5:1,t0:3,t5:0,  t6:2 ")
	if s := Format(got); s != "b14:8,t6:2" {
		t.Fatalf("脏数据解析出 %q，期望 b14:8,t6:2", s)
	}
	if len(ParseIDs("")) != 0 {
		t.Fatal("空串应解析成空切片")
	}
}

// TestBuildingLevel 取建筑等级（军工厂门槛用）。
func TestBuildingLevel(t *testing.T) {
	refs := ParseIDs("b14:5,t6:3")
	if got := BuildingLevel(refs, 14); got != 5 {
		t.Fatalf("军工厂等级 = %d，期望 5", got)
	}
	if got := BuildingLevel(refs, 18); got != 0 {
		t.Fatalf("没要求的建筑应返回 0，得到 %d", got)
	}
	// 建筑与科技 ID 会重叠：b6 与 t6 是两条不同的引用，不能互相串
	refs2 := ParseIDs("b6:2,t6:9")
	if BuildingLevel(refs2, 6) != 2 {
		t.Fatalf("建筑 6 的等级被科技 6 串了：%+v", refs2)
	}
}

// TestFormatSkipsZero 非法等级（0/负）不落库。
func TestFormatSkipsZero(t *testing.T) {
	s := Format([]Ref{{Kind: KindTech, ID: 6, Level: 0}, {Kind: KindTech, ID: 0, Level: 3}})
	if s != "" {
		t.Fatalf("非法引用不该输出，得到 %q", s)
	}
	if strings.Contains(Format([]Ref{{Kind: KindTech, ID: 6, Level: 2}}), "t6:2") == false {
		t.Fatal("正常引用应输出 t6:2")
	}
}
