package ezfy

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestPvpRankSwitchGatesAccumulation —— 防回归：风云榜统计开关必须**在写榜入口拦截**。
//
// 背景（2026-10-14 用户需求）：「风云榜 tab 能够选择开启关闭，关闭就是不进行统计」。
// 关闭靠 ezfy_cfg_limit.pvp_rank_on = 0，读取点是 ezfyPvpRankOn()。
//
// 风险：日后有人在 accumPvpWeek 之外新增写榜路径（或重构时把门禁挪走），
// 关闭开关就形同虚设 —— 表现是「后台明明关了，榜单还在涨」，很难当场发现。
// 所以这里断言：
//  1. accumPvpWeek 函数体里出现 ezfyPvpRankOn() 且**早于**任何写库调用；
//  2. ezfyPvpRankOn 读的是缓存里 limit.PvpRankOn（不是零值结构体）。
//
// 纯静态扫描，不需要 DB。
func TestPvpRankSwitchGatesAccumulation(t *testing.T) {
	path := filepath.Join(".", "ezfy_pvp_rank.go")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读 %s 失败: %v", path, err)
	}
	s := string(src)

	// 1) 取 accumPvpWeek 函数体
	body := funcBody(s, "func (h *EzfyHandler) accumPvpWeek(")
	if body == "" {
		t.Fatal("找不到 accumPvpWeek 函数体（重构后请同步更新本断言）")
	}
	gate := strings.Index(body, "ezfyPvpRankOn()")
	if gate < 0 {
		t.Fatal("accumPvpWeek 里没有统计开关门禁（ezfyPvpRankOn()）—— 关闭开关将无效")
	}
	// 门禁必须早于第一条写榜语句
	for _, write := range []string{"accumPvpWeekOnce(", "INSERT INTO ezfy_pvp_week", "INSERT INTO ezfy_pvp_foe"} {
		if i := strings.Index(body, write); i >= 0 && i < gate {
			t.Fatalf("写榜语句 %q 出现在开关门禁之前 —— 关闭开关也会写榜", write)
		}
	}

	// 2) 开关读取必须走缓存 limit，且不是「零值误判」
	onBody := funcBody(s, "func ezfyPvpRankOn()")
	if onBody == "" {
		t.Fatal("找不到 ezfyPvpRankOn 函数体")
	}
	if !strings.Contains(onBody, "ezfyCfg.limit.PvpRankOn") {
		t.Fatal("ezfyPvpRankOn 未读 ezfyCfg.limit.PvpRankOn")
	}
	if !strings.Contains(onBody, "ezfyCfg.ready()") {
		t.Fatal("ezfyPvpRankOn 缺 ready() 保护 —— 配置未加载时会读到零值 0，把「开」误判成「关」")
	}
}

// TestPvpExcludeAppliedToRankQuery —— 防回归：「不参与排名」名单必须下推到榜单 SQL。
//
// 只在 Go 侧过滤（取完再筛）会导致「徽标有名次、列表被截断成不足 N 名」这类隐蔽 bug，
// 且名单变大后名次口径会漂移。这里断言过滤发生在 ORDER/LIMIT 之前的 WHERE 里。
func TestPvpExcludeAppliedToRankQuery(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(".", "ezfy_pvp_rank.go"))
	if err != nil {
		t.Fatalf("读 ezfy_pvp_rank.go 失败: %v", err)
	}
	body := funcBody(string(src), "func ezfyPvpRankRows(")
	if body == "" {
		t.Fatal("找不到 ezfyPvpRankRows 函数体")
	}
	excl := strings.Index(body, "user_id NOT IN ?")
	if excl < 0 {
		t.Fatal("ezfyPvpRankRows 未在 SQL 层排除「不参与排名」名单")
	}
	lim := strings.Index(body, ".Limit(limit)")
	if lim >= 0 && lim < excl {
		t.Fatal("LIMIT 出现在排除条件之前 —— 会先截断再过滤，导致榜单缺人")
	}
}

// funcBody 粗略取出 Go 函数体（从 signature 起，按大括号配平到函数结束）。
// 只在测试里做静态检查用，遇到字符串/注释里的括号可能失真 —— 对本文件足够。
func funcBody(src, sig string) string {
	i := strings.Index(src, sig)
	if i < 0 {
		return ""
	}
	j := strings.Index(src[i:], "{")
	if j < 0 {
		return ""
	}
	start := i + j
	depth := 0
	inLine := false
	inBlock := false
	for k := start; k < len(src); k++ {
		switch {
		case inLine:
			if src[k] == '\n' {
				inLine = false
			}
			continue
		case inBlock:
			if src[k] == '*' && k+1 < len(src) && src[k+1] == '/' {
				inBlock = false
				k++
			}
			continue
		}
		switch src[k] {
		case '/':

			if k+1 < len(src) && src[k+1] == '/' {
				inLine = true
				k++
				continue
			}
			if k+1 < len(src) && src[k+1] == '*' {
				inBlock = true
				k++
				continue
			}
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[start : k+1]
			}
		}
	}
	return src[start:]
}

// 保证 regexp 被使用（本文件保留给后续更严格的断言扩展）。
var _ = regexp.MustCompile
