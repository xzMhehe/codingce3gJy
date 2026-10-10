package ezfy

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestTreasureBagMigrationNotWiredIntoStartup —— 防回归：绝不能再把「宝物签到误发 → 装备表」
// 的一次性迁移（ezfyMigrateTreasureBag / ezfyTreasureBagOnce）挂回请求 / 启动路径。
//
// 线上事故（2026-10-10，玩家反馈「道具丢失，怀疑重新部署导致」+「打地掉落的道具也丢」）：
//
//	该迁移本意是把 2026-09-28 之前「宝物签到」误发进 ezfy_item 的残留转成装备，
//	但它用 sync.Once 包裹，而 sync.Once 只对**进程内**有效 —— 每次重启 / 重新部署后
//	首个请求都会重跑。而 ezfy_item.cfg_id 的 27~35 区间**已被真实道具占用**
//	（28~36 = 建筑/训练/科技加速），与装备配置（珠宝 25~35）ID 完全重叠 →
//	玩家花钱买 / 打野地掉落的加速道具，**每次部署都被当成「误发宝物」删掉**，
//	再换成一件珠宝装备。
//
// 线上特征（可用于确认事故）：ezfy_equipment 里珠宝（cfg 27~35）按天出现巨大簇，
// 例如某天某小时一次性生成 2000+ 件珠宝；且 ezfy_item 里 cfg 28~36 的存量远少于
// ezfy_diamond_logs 的「商城购买」件数。
//
// 修复：永久摘除该调用（函数体保留在 ezfy_rank_treasure.go 仅作参考，禁止调用）。
func TestTreasureBagMigrationNotWiredIntoStartup(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) cfgs() {")
	// 只看真实代码：注释里为了说明事故原因会提到这两个标识符，不算「挂回来」。
	code := stripGoCommentLines(body)
	for _, banned := range []string{"ezfyTreasureBagOnce", "ezfyMigrateTreasureBag"} {
		if strings.Contains(code, banned) {
			t.Fatalf("cfgs() 又把危险的启动迁移 %s 挂回来了 —— 它会在每次重启/部署时"+
				"把玩家真实道具(加速道具 cfg 28~36)当误发宝物删除并换成珠宝装备：\n%s", banned, body)
		}
	}
}

// TestTreasureBagMigrationGuardKept —— 万一有人误调用该函数，函数内的守卫必须仍在：
// 「能在道具配置表(ezfy_cfg_item)查到该 cfg_id 的，都是真实道具，一律跳过」。
// 这是最后一道防线（防止把真实道具当残留迁移删除）。
func TestTreasureBagMigrationGuardKept(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_rank_treasure.go", "func ezfyMigrateTreasureBag(")
	if !strings.Contains(body, "ezfyCfg.item(it.CfgId) != nil") {
		t.Fatalf("ezfyMigrateTreasureBag 丢了「真实道具跳过」守卫 —— 误调用会删玩家道具：\n%s", body)
	}
}

// TestNoCrossTableIDGuess —— 防回归：禁止「按 cfg_id 区间猜身份」的跨表解析。
//
// 背景：`ezfy_cfg_item`（道具）与 `ezfy_cfg_equipment`（装备）是**两张互不相关的配置表，
// 各自从 1 开始编号** → 数值 ID 大面积重叠：
//
//	ezfy_cfg_item      : 1~41        （30 = 建筑加速80%）
//	ezfy_cfg_equipment : 1~35 + 101… （30 = 黑曜石戒指）
//
// 于是「ID 30」既可能是道具也可能是装备 —— 光看数字判断不了身份。任何
// 「ID 落在某区间 → 当成装备/道具」的写法，都会在 ID 区间被复用时误伤真实数据。
// 历史事故（2026-10-10）：ezfyMigrateTreasureBag 按 `cfg_id 27~35` 把玩家真实加速道具
// 当「误发宝物」删掉换成珠宝装备，而且每次部署都重跑一次。
//
// 本断言扫描包内全部非测试 .go：若同一函数里**既有 cfg_id 区间比较、又有 equipments[] 查表**，
// 即视为可疑 → 必须显式加入下方 allow 白名单。新增白名单项前请先想清楚：
// 这个 ID 区间以后会不会被另一张表复用？
func TestNoCrossTableIDGuess(t *testing.T) {
	// 白名单：目前仅保留「已停用、禁止再调用」的历史迁移函数（见 ezfy.go 里的摘除说明）。
	allow := map[string]bool{
		"ezfyMigrateTreasureBag": true,
	}
	// cfg_id 与**非零数字**做区间比较（SQL 字符串 / Go 比较）才算可疑；
	// `CfgId <= 0` 这类「参数校验」不算。
	cfgRangeRe := regexp.MustCompile(`(?i)cfg_?id\s*(>=|<=|>|<)\s*[1-9][0-9]*`)
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("扫描源码失败: %v", err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("读源码失败 %s: %v", f, err)
		}
		for name, chunk := range goFuncChunks(string(b)) {
			if allow[name] {
				continue
			}
			code := stripGoCommentLines(chunk)
			if !strings.Contains(code, "equipments[") {
				continue
			}
			hit := cfgRangeRe.FindString(code)
			if hit == "" {
				continue
			}
			t.Fatalf("%s 的函数 %s 同时出现了「cfg_id 区间判断(%q)」和「equipments[] 查表」——\n"+
				"这是「按 ID 区间猜身份」的写法：ezfy_cfg_item 与 ezfy_cfg_equipment 的 ID 重叠\n"+
				"（如 ID 30 既是道具「建筑加速80%%」也是装备「黑曜石戒指」），ID 区间被复用后会误删/误发玩家数据。\n"+
				"请改为带来源的解析（item(id) / equipment(id)，或靠 Kind/item_type 区分）。\n"+
				"确属历史迁移且已停用 → 显式加入本测试的 allow 白名单。", f, name, hit)
		}
	}
}

// goFuncChunks 粗略按 "\nfunc " 切分 Go 源码，返回「函数名 → 函数体」。
// 仅用于静态断言（不解析 AST）：方法会剥掉接收者，取真实函数名。
func goFuncChunks(src string) map[string]string {
	out := map[string]string{}
	parts := strings.Split(src, "\nfunc ")
	for i := 1; i < len(parts); i++ {
		s := parts[i]
		p := 0
		if strings.HasPrefix(s, "(") {
			j := strings.Index(s, ")")
			if j < 0 {
				continue
			}
			p = j + 1
		}
		for p < len(s) && (s[p] == ' ' || s[p] == '\t') {
			p++
		}
		k := p
		for k < len(s) && (s[k] == '_' ||
			(s[k] >= 'a' && s[k] <= 'z') || (s[k] >= 'A' && s[k] <= 'Z') || (s[k] >= '0' && s[k] <= '9')) {
			k++
		}
		if k > p {
			out[s[p:k]] = s
		}
	}
	return out
}
