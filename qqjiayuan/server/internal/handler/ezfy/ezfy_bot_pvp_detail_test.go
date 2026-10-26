package ezfy

import (
	"os"
	"strings"
	"testing"
)

// TestPvpDetailReportLinked 风云榜「积分明细」必须能关联到本场战报。
//
// ★ 2026-10-26 用户需求「管理端能看所有人的积分明细（并且能关联查看本场战报）」：
//   - 明细在战斗判定后先写、战报在结算末尾才写 → 必须**回填** report_id；
//   - addReport 要返回新战报 id（否则回填无从下手）；
//   - 结算末尾要把攻/守双方的战报 id 各写回对应明细行。
func TestPvpDetailReportLinked(t *testing.T) {
	// ① addReport 返回战报 id
	order := rawFile(t, "ezfy_order.go")
	if !strings.Contains(order, "detailAndOrder ...interface{}) uint {") {
		t.Fatal("addReport 没有返回新战报 id（积分明细无法回填战报）")
	}
	if !strings.Contains(order, "return r.ID") {
		t.Fatal("addReport 没有把新战报 id 返回出去")
	}
	// ② 明细行 id 从写榜函数返回
	rank := rawFile(t, "ezfy_pvp_rank.go")
	for _, want := range []string{
		"func (h *EzfyHandler) accumPvpWeekOnce(",
		"return d.ID",
		"func (h *EzfyHandler) ezfyPvpAttachReport(detailID, reportID uint)",
	} {
		if !strings.Contains(rank, want) {
			t.Fatalf("积分明细没有产出/回填明细 id：缺少 %q", want)
		}
	}
	// ③ 结算里回填（攻守各一条）
	body := ezfyFuncBody(t, "ezfy_order.go", "func (h *EzfyHandler) processArrive(")
	if !strings.Contains(body, "pvpAtkDetailID, pvpDefDetailID = h.accumPvpWeek(") {
		t.Fatal("processArrive 没有接住攻守双方的明细 id")
	}
	if strings.Count(body, "h.ezfyPvpAttachReport(") < 2 {
		t.Fatal("processArrive 没有把战报 id 回填给双方明细")
	}
	// ④ 明细表必须有 report_id 字段（model 在上一级目录）
	mdl := strings.ReplaceAll(mustReadFile(t, "../../model/ezfy.go"), "\r\n", "\n")
	if !strings.Contains(mdl, `json:"report_id"`) {
		t.Fatal("EzfyPvpDetail 缺少 report_id 字段")
	}
}

// mustReadFile 读任意相对路径源码（用于跨包断言 model 定义）。
func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读源码失败 %s: %v", path, err)
	}
	return string(b)
}

// TestPvpLockOnlyCurrentWeek 锁分只锁当期。
//
// ★ 2026-10-26 用户规则「锁分操作只锁当期」：
//   - 管理端锁分接口只接受本周（历史周直接拒）；
//   - 展示用的 lock_week 只在「存的周 == 本周」时返回，跨周自动失效。
func TestPvpLockOnlyCurrentWeek(t *testing.T) {
	admin := rawFile(t, "ezfy_admin_rank.go")
	lockBody := ezfyFuncBody(t, "ezfy_admin_rank.go", "func (h *EzfyAdmin) AdminEzfyPvpRankLock(")
	if !strings.Contains(lockBody, "只能锁当期") {
		t.Fatalf("锁分接口没有限制「只能锁当期」：\n%s", lockBody)
	}
	if !strings.Contains(lockBody, "curWeek := ezfyPvpWeekOf(time.Now())") {
		t.Fatalf("锁分接口没有强制取当前自然周：\n%s", lockBody)
	}
	if strings.Contains(lockBody, "lockWeek = strings.TrimSpace(in.Week)") {
		t.Fatal("锁分接口仍在用请求里的 week（可以锁历史周）")
	}
	_ = admin

	// 展示口径：跨周视为未锁定
	body := ezfyFuncBody(t, "ezfy_pvp_rank.go", "func ezfyPvpRankLockWeek() string {")
	if !strings.Contains(body, "w != ezfyPvpWeekOf(time.Now())") {
		t.Fatalf("ezfyPvpRankLockWeek 没有做「跨周自动失效」：\n%s", body)
	}
}

// TestBotOfficerPostNotGatedByBuildings 人机城守/技能不能被军校·参谋部挡住。
//
// ★ 2026-10-26 两个线上 bug：「人机不会自己上城守」「人机军官不会学技能」——
// 根因是这两件事原来写在 botOfficer 里，而 botOfficer 开头就 return 掉了没建军校/参谋部的人机。
// 现在拆到 botOfficerPost（无建筑前置），且必须排在建筑/征兵之前。
func TestBotOfficerPostNotGatedByBuildings(t *testing.T) {
	post := ezfyFuncBody(t, "ezfy_bot.go", "func (h *EzfyHandler) botOfficerPost(")
	if post == "" {
		t.Fatal("缺少 botOfficerPost（城守/技能独立步骤）")
	}
	for _, bad := range []string{"ezfyBuildingAcademy", "ezfyBuildingStaff"} {
		if strings.Contains(post, bad) {
			t.Fatalf("botOfficerPost 里不该再校验 %s（会让没建建筑的人机永远不任命城守/不学技能）", bad)
		}
	}
	if !strings.Contains(post, "ezfyPositionGuard") {
		t.Fatal("botOfficerPost 没有任命城守")
	}
	if !strings.Contains(post, "saveOfficerSkills") {
		t.Fatal("botOfficerPost 没有给军官配技能")
	}
	// 城守不能把人机唯一的出征军官占掉
	if !strings.Contains(post, "freeCnt >= 2") {
		t.Fatal("任命城守没有「至少还剩一个能出征的军官」的保护")
	}
	// tick 里必须在建筑/征兵之前调用
	tick := ezfyFuncBody(t, "ezfy_bot.go", "func (h *EzfyHandler) botDecide(")
	if tick == "" {
		t.Skip("tick 主函数改名了，跳过顺序断言")
	}
	iPost := strings.Index(tick, "h.botOfficerPost(city)")
	iTrain := strings.Index(tick, "h.botTrain(city, troopTotal, cfg, 0)")
	if iPost < 0 {
		t.Fatal("tick 没有调用 botOfficerPost")
	}
	if iTrain > 0 && iPost > iTrain {
		t.Fatal("botOfficerPost 排在征兵之后（会被征兵长期抢占，城守/技能轮不到）")
	}
}

// TestBotTargetMinTroops 人机自动挑目标必须吃「玩家城市最小兵力」配置。
//
// ★ 2026-10-26 用户规则「配置按人机分而不是总的，默认最少 100000」：
//   - 默认值 100000（原全局 10000 已废弃为兜底）；
//   - 实际门槛读 EzfyBot.TargetMinTroops（每台人机各自一份，0 = 不限）；
//   - botPickTarget 的 SQL 下界断言保持不变。
func TestBotTargetMinTroops(t *testing.T) {
	src := rawFile(t, "ezfy_bot.go")
	if !strings.Contains(src, "const ezfyBotTargetMinTroopsDef int64 = 100000") {
		t.Fatal("最小兵力默认值必须是 100000")
	}
	// 真正生效的口径：botMilitary 把**该人机自己的**字段传进 botPickTarget
	if !strings.Contains(src, "h.botPickTarget(city, troopTotal, cfg, b.TargetMinTroops)") {
		t.Fatal("botMilitary 没有把人机自己的 target_min_troops 传给 botPickTarget（会退回全局口径）")
	}
	pick := ezfyFuncBody(t, "ezfy_bot.go", "func (h *EzfyHandler) botPickTarget(")
	if !strings.Contains(pick, "minTroops int64") {
		t.Fatalf("botPickTarget 没有接收按人机的最小兵力参数：\n%s", pick)
	}
	if !strings.Contains(pick, "AND COALESCE(t.troops, 0) >= ?") {
		t.Fatalf("botPickTarget 没有最小兵力下界：\n%s", pick)
	}
	// 新建人机必须显式落默认 100000（不靠列默认，避免口径漂移）
	if !strings.Contains(src, "TargetMinTroops: ezfyBotTargetMinTroopsDef") {
		t.Fatal("ezfyBotProvision 没有显式落默认目标最小兵力 100000")
	}
	// 管理端可配（按人机）
	adm := rawFile(t, "ezfy_admin_bot.go")
	for _, want := range []string{"target_min_troops", "TargetMinTroops"} {
		if !strings.Contains(adm, want) {
			t.Fatalf("管理端人机配置没有接通 %q", want)
		}
	}
}
