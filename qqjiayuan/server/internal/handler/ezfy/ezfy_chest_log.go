package ezfy

import "fmt"

// ezfyAggPrizeNames 把开箱结果按「奖品文案」聚合，供玩家播报 / 开箱流水共用。
//
// 旧实现有两个问题：
//  1. 同名奖品第一次写成「名×N」，后面 N-1 次又把同名逐条追加，
//     30 连会膨胀成「荣誉史记×1×11、荣誉史记×1、…」，很容易撞上 255 字符上限；
//  2. 与 255 字符硬截断叠加后，管理端开箱明细只剩前 24 条，后面的奖品查不到。
//
// 现在只输出一条「名×N」，顺序按首次出现；空结果由调用方决定是否显示「空箱」。
func ezfyAggPrizeNames(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	order := make([]string, 0, len(names))
	counts := make(map[string]int, len(names))
	for _, nm := range names {
		if counts[nm] == 0 {
			order = append(order, nm)
		}
		counts[nm]++
	}
	agg := make([]string, 0, len(order))
	for _, nm := range order {
		if counts[nm] > 1 {
			agg = append(agg, fmt.Sprintf("%s×%d", nm, counts[nm]))
		} else {
			agg = append(agg, nm)
		}
	}
	return agg
}
