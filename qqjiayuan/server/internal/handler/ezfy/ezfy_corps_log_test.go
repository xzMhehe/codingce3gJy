package ezfy

import (
	"strings"
	"testing"
)

// ============ 军团页改版护栏（★ 2026-10-30 用户需求）============
//
// 用户要求：
//  ① 军团成员**分页** + **按职位排序**（军团长 > 副团长 > 参谋长 > 成员）；
//  ② 页签重排两行：信息/列表/聊天 + 外交/日志/商城；
//  ③ 「军团宣战」并入「军团外交」；
//  ④ 新增「军团日志」，记录进团 / 出团 / 任命（含撤职）。

// TestCorpsMembersPagedAndSorted 成员接口：职位排序 + 分页 + 踢人用全量列表。
func TestCorpsMembersPagedAndSorted(t *testing.T) {
	body := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_chat_exchange.go", "func (h *EzfyHandler) CorpsMembers("), "\r\n", "\n"))
	for _, want := range []string{
		"ezfyCorpsMembersSorted(members)", // 按职位排序（唯一实现）
		"pageOf(c, 20)",                   // 分页参数
		`"all_members": allViews`,         // 踢人下拉要能选到不在当前页的成员
		`"total": total, "page": page, "size": size`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("CorpsMembers 缺少分页/排序逻辑 %q：\n%s", want, body)
		}
	}
	// 排序权重唯一实现：军团长 > 副团长 > 参谋长 > 成员
	w := ezfyStripCommentLines(strings.ReplaceAll(
		ezfyFuncBody(t, "ezfy_corps_log.go", "func ezfyCorpsTitleWeight("), "\r\n", "\n"))
	for _, want := range []string{"if m.IsLeader == 1 {", "case ezfyCorpsTitleVice:", "case ezfyCorpsTitleChief:"} {
		if !strings.Contains(w, want) {
			t.Fatalf("ezfyCorpsTitleWeight 缺职位权重分支 %q：\n%s", want, w)
		}
	}
}

// TestCorpsLogWrites 日志必须覆盖：创建/入团/审核通过/退出/踢人/任命/撤职。
func TestCorpsLogWrites(t *testing.T) {
	api := ezfyStripCommentLines(strings.ReplaceAll(rawFile(t, "ezfy_api.go"), "\r\n", "\n"))
	for _, want := range []string{
		`h.corpsLogAdd(cp.ID, uid, 0, ezfyCorpsLogJoin, "", "创建军团")`,
		`h.corpsLogAdd(cp.ID, uid, 0, ezfyCorpsLogJoin, "", "加入军团")`,
		`h.corpsLogAdd(cp.ID, a.UserId, uid, ezfyCorpsLogJoin, "", "审核通过入团")`,
		`h.corpsLogAdd(mb.CorpsId, uid, 0, ezfyCorpsLogLeave, "", "主动退出")`,
		`h.corpsLogAdd(leader.CorpsId, req.UserId, uid, ezfyCorpsLogLeave, "", "被踢出军团")`,
	} {
		if !strings.Contains(api, want) {
			t.Fatalf("军团日志缺少记录点 %q", want)
		}
	}
	// 军团长解散：日志随军团一起清（不留孤儿行）
	if !strings.Contains(api, `h.DB.Where("corps_id = ?", mb.CorpsId).Delete(&model.EzfyCorpsLog{})`) {
		t.Fatal("军团长解散军团时必须清理 ezfy_corps_log")
	}
	lia := ezfyStripCommentLines(strings.ReplaceAll(rawFile(t, "ezfy_liaison.go"), "\r\n", "\n"))
	for _, want := range []string{
		`h.corpsLogAdd(cp.ID, req.UserId, uid, ezfyCorpsLogRevoke, mb.Title, "撤销职位")`,
		`h.corpsLogAdd(cp.ID, req.UserId, uid, ezfyCorpsLogAppoint, title, "任命职位")`,
	} {
		if !strings.Contains(lia, want) {
			t.Fatalf("军团任命/撤职没有写日志 %q", want)
		}
	}
}

// TestCorpsLogRouteAndPage 日志接口路由 + 用户端页签改版（宣战并入外交、新增日志）。
func TestCorpsLogRouteAndPage(t *testing.T) {
	router := readFrontFile(t, "../../router/router.go")
	if !strings.Contains(router, `ezfyG.GET("/corps/logs", ezfyH.CorpsLogs)`) {
		t.Fatal("路由缺少 GET /corps/logs")
	}
	vue := strings.ReplaceAll(readFrontFile(t,
		"../../../../web/src/views/ezfy/modules/EzfyCorps.vue"), "\r\n", "\n")
	for _, want := range []string{
		`ezfy.switchCorpsTab('log')`,  // 新增日志 tab
		"军团日志",                        // tab 文案
		`ezfy.corpsTab === 'log'`,     // 日志页模板
		`ezfy.corpsLogGo(-1)`,         // 日志分页
		`ezfy.corpsMemberGo(-1)`,      // 成员分页
		`ezfy.corpsMembersAll`,        // 踢人下拉用全量
		`ezfy.switchCorpsTab('diplomacy')`, // 外交 tab 仍在
	} {
		if !strings.Contains(vue, want) {
			t.Fatalf("EzfyCorps.vue 缺少 %q", want)
		}
	}
	// 宣战 tab 已并入外交：不允许再有独立 war 入口
	if strings.Contains(vue, `ezfy.switchCorpsTab('war')`) || strings.Contains(vue, `ezfy.corpsTab === 'war'`) {
		t.Fatal("「军团宣战」应并入「军团外交」，不能再有独立 war tab")
	}
	if !strings.Contains(vue, "<br/>") {
		t.Fatal("页签应排成两行（第一行 信息/列表/聊天，第二行 外交/日志/商城）")
	}
	// 外壳：切 tab 时日志按需加载 + 进页重置
	shell := strings.ReplaceAll(readFrontFile(t,
		"../../../../web/src/views/ezfy/Ezfy.vue"), "\r\n", "\n")
	for _, want := range []string{
		"if (tab === 'war') tab = 'diplomacy'", // 历史 war 调用兼容
		"if (!this.corpsLogLoaded) this.loadCorpsLogs(1)",
		"this.loadCorpsMembers(1)",
	} {
		if !strings.Contains(shell, want) {
			t.Fatalf("Ezfy.vue 缺少军团改版接线 %q", want)
		}
	}
}
