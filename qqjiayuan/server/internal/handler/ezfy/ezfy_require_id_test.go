package ezfy

import (
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// TestTroopRequireMsgOnlyTrustsIDs 行为验证：require 文本与 require_ids 打架时，以 ID 为准。
//
// 造一条「文本写军工厂 9 级、结构化 ID 写 b14:3 + t6:2」的配置：
// 军工厂 3 级 + 科技 6 满级 → 必须放行（证明 9 级那行文本根本没参与校验）。
func TestTroopRequireMsgOnlyTrustsIDs(t *testing.T) {
	h := &EzfyHandler{}
	cfg := &model.EzfyCfgTroop{ID: 99, Name: "测试兵", Require: "军工厂(9级)", RequireIds: "b14:3,t6:2"}

	if msg := h.ezfyTroopRequireMsg(cfg, map[int]int{14: 1}, map[int]int{6: 10}); msg == "" {
		t.Fatal("军工厂 1 级 < ID 要求 3 级，应该被拦（说明没在读 require_ids）")
	}
	if msg := h.ezfyTroopRequireMsg(cfg, map[int]int{14: 3}, map[int]int{6: 1}); msg == "" {
		t.Fatal("科技 1 级 < ID 要求 2 级，应该被拦")
	}
	// ★ 关键断言：军工厂 3 级就放行 —— 文本里写的是 9 级，说明校验只认 ID
	if msg := h.ezfyTroopRequireMsg(cfg, map[int]int{14: 3}, map[int]int{6: 2}); msg != "" {
		t.Fatalf("军工厂 3 级已满足 ID 要求（b14:3），却被拦：%q", msg)
	}

	// 反向：文本满足、ID 不满足 → 必须拦（旧逻辑会放行）
	cfg2 := &model.EzfyCfgTroop{ID: 98, Name: "测试兵2", Require: "军工厂(1级)", RequireIds: "b14:9"}
	if msg := h.ezfyTroopRequireMsg(cfg2, map[int]int{14: 5}, map[int]int{}); msg == "" {
		t.Fatal("ID 要求军工厂 9 级、当前 5 级，必须拦（文本写的 1 级不该被采信）")
	}
}

// ★ 2026-10-31 护栏：训练门槛必须**按结构化 ID** 校验，不能再靠「require 文本里的名字」。
//
// 事故模型：ezfy_cfg_troop.require 是展示文本，改造前校验用正则取出名字去
// buildingByName / techByName 反查 id；名字一改（本次科技改名）、打错、全角半角不一致
// 就查不到 → continue 跳过 → 该条门槛等于不存在，玩家不研究科技也能造高级兵，且无日志。
//
// 这里用源码静态断言守住三处，任何一处被改回「按名字」立刻红。
func TestTrainRequireUsesStructuredIDs(t *testing.T) {
	// ① 校验主体读 require_ids
	msg := ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) ezfyTroopRequireMsg(")
	if !strings.Contains(msg, "ezfyreq.ParseIDs(cfg.RequireIds)") {
		t.Fatalf("ezfyTroopRequireMsg 没有读 require_ids（按 ID 校验）：\n%s", msg)
	}
	if !strings.Contains(msg, "ezfyreq.KindBuilding") {
		t.Fatalf("ezfyTroopRequireMsg 没有区分建筑/科技来源（两张表 ID 会重叠）：\n%s", msg)
	}

	// ② 训练入口必须走这个校验
	train := ezfyFuncBody(t, "ezfy.go", "func (h *EzfyHandler) trainTroopLocked(")
	if !strings.Contains(train, "h.ezfyTroopRequireMsg(") {
		t.Fatalf("trainTroopLocked 没调 ezfyTroopRequireMsg，训练门槛可能被绕过：\n%s", train)
	}
	if strings.Contains(train, "ezfyRequirePattern.FindAllStringSubmatch") {
		t.Fatalf("trainTroopLocked 里还残留「按名字解析 require」的旧逻辑：\n%s", train)
	}
}

// 管理端保存兵种配置时，必须把 require 文本重算成 require_ids（唯一写入口）。
func TestAdminTroopSaveRecalcsRequireIDs(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy_admin_cfg.go", "func (h *EzfyAdmin) AdminEzfyTroopCfgUpdate(")
	if !strings.Contains(body, "ezfyreq.Parse(") {
		t.Fatalf("AdminEzfyTroopCfgUpdate 没有解析 require 文本：\n%s", body)
	}
	if !strings.Contains(body, `vals["require_ids"] = ezfyreq.Format(refs)`) {
		t.Fatalf("AdminEzfyTroopCfgUpdate 没有同步写 require_ids（保存后门槛会失效）：\n%s", body)
	}
	if !strings.Contains(body, "无法识别的建筑/科技名") {
		t.Fatalf("AdminEzfyTroopCfgUpdate 没有对错别字报错（会静默保存一条无效门槛）：\n%s", body)
	}
}

// 「军工厂等级」也要按 ID 取，不能靠文本里出现「军工厂」三个字。
func TestNeedFactoryUsesStructuredIDs(t *testing.T) {
	body := ezfyFuncBody(t, "ezfy.go", "func ezfyNeedFactoryLevelOf(")
	if !strings.Contains(body, "ezfyreq.BuildingLevel(refs, ezfyFactoryBuildingID)") {
		t.Fatalf("ezfyNeedFactoryLevelOf 没有按建筑 ID 取军工厂等级：\n%s", body)
	}
}
