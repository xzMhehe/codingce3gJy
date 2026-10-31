// Package ezfyreq 二战「需要条件」的结构化解析（require 文本 ↔ 结构化 ID）。
//
// ★ 为什么要有这个包（2026-10-31 科技改名后立的规矩）：
//
// `ezfy_cfg_troop.require` 是给人看的文本（如「军工厂(8级)枪械改良(8级)」），
// 而训练校验原来是**按名字**去 ezfyCfg.buildingByName / techByName 反查 id 的。
// 名字一改（改名、错别字、全角半角差异、管理端手写），查不到就**静默跳过**这条门槛 ——
// 玩家不研究科技也能造高级兵，且不报错、不写日志，属于最难发现的数据漏洞。
//
// 现在的口径：
//
//	① 文本只用于**展示**；
//	② 校验读 `ezfy_cfg_troop.require_ids`（如 "b14:1,t5:1"，b=建筑 t=科技）；
//	③ 文本 → ID 的转换只发生在这一处（Parse），种子回填与管理端保存都调它；
//	④ 建筑与科技分属两张表、ID 会重叠 → 每段必须带 kind 前缀，**禁止按 ID 区间猜身份**。
package ezfyreq

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Kind 引用来源。建筑与科技各自从 1 编号，光看数字无法判断身份，必须带前缀。
const (
	KindBuilding byte = 'b'
	KindTech     byte = 't'
)

// Ref 一条「需要 X 到 N 级」。
type Ref struct {
	Kind  byte // KindBuilding / KindTech
	ID    int
	Level int
}

// pattern 与改造前 handler 里的 ezfyRequirePattern 同口径（名字 + （N级））。
// ★ 那个旧变量 2026-10-31 已删除，正则以本包为准 —— 别再在别处再造一份。
// 同时吃半角 () 与全角 （）（历史配置两种都出现过）。
var pattern = regexp.MustCompile(`([^()（）]+)[（(]\s*(\d+)\s*级?\s*[）)]`)

// Parse 把 require 文本解析成结构化引用。
// unknown 返回「既不是建筑也不是科技」的名字（用于管理端保存时报错、种子回填时告警）。
func Parse(text string, buildingIDs, techIDs map[string]int) (refs []Ref, unknown []string) {
	refs = []Ref{}
	unknown = []string{}
	if strings.TrimSpace(text) == "" {
		return
	}
	for _, m := range pattern.FindAllStringSubmatch(text, -1) {
		name := strings.TrimSpace(m[1])
		lv, err := strconv.Atoi(m[2])
		if err != nil || lv <= 0 {
			unknown = append(unknown, m[0])
			continue
		}
		if id, ok := buildingIDs[name]; ok {
			refs = append(refs, Ref{Kind: KindBuilding, ID: id, Level: lv})
			continue
		}
		if id, ok := techIDs[name]; ok {
			refs = append(refs, Ref{Kind: KindTech, ID: id, Level: lv})
			continue
		}
		unknown = append(unknown, name)
	}
	return
}

// Format 结构化引用 → 存储字符串，如 "b14:1,t5:1"。
func Format(refs []Ref) string {
	parts := make([]string, 0, len(refs))
	for _, r := range refs {
		if r.ID <= 0 || r.Level <= 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%c%d:%d", r.Kind, r.ID, r.Level))
	}
	return strings.Join(parts, ",")
}

// ParseIDs 存储字符串 → 结构化引用。脏段直接丢弃（宁可少一条门槛也不 panic）。
func ParseIDs(s string) []Ref {
	refs := []Ref{}
	for _, seg := range strings.Split(s, ",") {
		seg = strings.TrimSpace(seg)
		if len(seg) < 4 {
			continue
		}
		kind := seg[0]
		if kind != KindBuilding && kind != KindTech {
			continue
		}
		rest := seg[1:]
		i := strings.Index(rest, ":")
		if i <= 0 {
			continue
		}
		id, err1 := strconv.Atoi(strings.TrimSpace(rest[:i]))
		lv, err2 := strconv.Atoi(strings.TrimSpace(rest[i+1:]))
		if err1 != nil || err2 != nil || id <= 0 || lv <= 0 {
			continue
		}
		refs = append(refs, Ref{Kind: kind, ID: id, Level: lv})
	}
	return refs
}

// BuildingLevel 取某个建筑被要求的等级（没有要求返回 0）。
func BuildingLevel(refs []Ref, buildingID int) int {
	for _, r := range refs {
		if r.Kind == KindBuilding && r.ID == buildingID {
			return r.Level
		}
	}
	return 0
}
