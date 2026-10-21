package ezfy

import (
	"reflect"
	"testing"
)

func TestEzfyAggPrizeNames(t *testing.T) {
	names := []string{
		"荣誉史记×1", "星级徽章×1", "荣誉史记×1",
		"荣誉史记×1", "军官洗点卡×1", "星级徽章×1",
	}
	got := ezfyAggPrizeNames(names)
	want := []string{
		"荣誉史记×1×3", "星级徽章×1×2", "军官洗点卡×1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("聚合结果不对:\n got=%v\nwant=%v", got, want)
	}
}

func TestEzfyAggPrizeNamesEmpty(t *testing.T) {
	if got := ezfyAggPrizeNames(nil); len(got) != 0 {
		t.Fatalf("空输入应返回空切片，实际 %v", got)
	}
}
