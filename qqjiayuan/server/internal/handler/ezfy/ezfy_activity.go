package ezfy

import (
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 节日活动（E1）
//
// 设计依据：`参考材料/开发文档/福利.txt`（26 条活动点子），原版 Java 只做了静态说明页。
// 本项目做成可配置、真实生效的活动：管理端「数据管理 → 节日活动」增删改，
// 开启后按类型给全服加成。
//
//	1 资源增产   —— 全资源产量 +Param%
//	2 造兵打折   —— 训练部队资源消耗 -Param%
//	3 建造加速   —— 建筑建造/升级耗时 -Param%
//	4 研究加速   —— 科技研究耗时 -Param%
//	5 声望加成   —— 战斗获得声望 +Param%
const (
	ezfyActProduce  = 1
	ezfyActTrain    = 2
	ezfyActBuild    = 3
	ezfyActTech     = 4
	ezfyActPrestige = 5
)

// ★★ 2026-10-05 性能：活动查询进程内 5 秒 TTL 缓存（修 N+1）。
//
//	事故形态：`/troops` 的兵种一览对**每个兵种**都调 `trainCostWithActivity` →
//	`actPct(ezfyActTrain)` → `activeActivity` → 一条 SQL。20 个兵种 = 20 条一模一样的
//	`SELECT * FROM ezfy_activity WHERE type = 2 ...`，实测单请求 22 条（跨 WAN 就是 22 个 RTT）。
//	`/view`、`/resources` 里 calcResource 与 getResourceCalc 也各查一次。
//
//	活动表是**全服共享、管理端低频改动**的数据，进程内缓存 5 秒完全安全：
//	管理端改完最多 5 秒生效（与「配置 30s 周期收敛」同量级）。多机部署下各机独立收敛，无一致性问题。
var (
	ezfyActCacheMu sync.Mutex
	ezfyActCache   = map[int]ezfyActCacheEntry{}
)

type ezfyActCacheEntry struct {
	act *model.EzfyActivity
	at  int64
}

const ezfyActCacheTTLMs = 5000

// ezfyActCacheInvalidate 管理端改动活动后立即失效（可选，不调也会 5 秒内自然收敛）。
func ezfyActCacheInvalidate() {
	ezfyActCacheMu.Lock()
	ezfyActCache = map[int]ezfyActCacheEntry{}
	ezfyActCacheMu.Unlock()
}

// activeActivity 取当前进行中的某类活动(同类多个时取加成最大的)
//
// ★ 5 秒进程内缓存：同一请求里 20 个兵种只查 1 次库（见上方说明）。
// 缓存的是「查询时刻的结论」，故把「是否已过期」也一并按 TTL 收敛 —— 活动起止最多滞后 5 秒。
func (h *EzfyHandler) activeActivity(actType int) *model.EzfyActivity {
	now := time.Now().UnixMilli()
	ezfyActCacheMu.Lock()
	if e, ok := ezfyActCache[actType]; ok && now-e.at < ezfyActCacheTTLMs {
		ezfyActCacheMu.Unlock()
		// 缓存命中的活动若刚好在这一瞬间过期，按「无活动」处理，避免多给 5 秒加成
		if e.act != nil && e.act.EndTime > now {
			return e.act
		}
		return nil
	}
	ezfyActCacheMu.Unlock()

	var list []model.EzfyActivity
	h.DB.Where("type = ? AND status = 1 AND start_time <= ? AND end_time > ?", actType, now, now).
		Order("param DESC").Limit(1).Find(&list)
	var got *model.EzfyActivity
	if len(list) > 0 {
		got = &list[0]
	}
	ezfyActCacheMu.Lock()
	ezfyActCache[actType] = ezfyActCacheEntry{act: got, at: now}
	ezfyActCacheMu.Unlock()
	return got
}

// actPct 活动加成百分比(无活动返回 0)
func (h *EzfyHandler) actPct(actType int) int {
	if a := h.activeActivity(actType); a != nil {
		return a.Param
	}
	return 0
}

func ezfyActTypeName(t int) string {
	switch t {
	case ezfyActProduce:
		return "资源增产"
	case ezfyActTrain:
		return "造兵打折"
	case ezfyActBuild:
		return "建造加速"
	case ezfyActTech:
		return "研究加速"
	case ezfyActPrestige:
		return "声望加成"
	}
	return "活动"
}

// trainCostWithActivity 训练/建造部队的资源消耗(含节日活动·造兵打折)
// 训练扣费与列表展示共用这一份计算, 避免两边算错。
//
// ★ 「加个征兵资源消耗开关，默认开；关了的话征兵不消耗资源」→
// 开关关掉时直接返回 0，**扣费与所有展示**（兵种列表 cost / 兵种详情 / 训练确认页）
// 都跟着变 0，不会出现「显示要 100 粮、实际不扣」的错位。
func (h *EzfyHandler) trainCostWithActivity(food, steel, oil, rare int64) (int64, int64, int64, int64) {
	if !ezfyRecruitCostOn() {
		return 0, 0, 0, 0
	}
	pct := h.actPct(ezfyActTrain)
	if pct <= 0 {
		return food, steel, oil, rare
	}
	k := int64(100 - pct)
	return food * k / 100, steel * k / 100, oil * k / 100, rare * k / 100
}

// techResearchMs 科技研究耗时(ms), 含节日活动·研究加速
func (h *EzfyHandler) techResearchMs(sec int) int64 {
	ms := int64(sec) * 1000
	if pct := h.actPct(ezfyActTech); pct > 0 {
		ms = ms * int64(100-pct) / 100
	}
	if ms < 1000 {
		ms = 1000
	}
	return ms
}

// ActivityInfo GET /games/ezfy/activity —— 活动列表(进行中 + 未开启)
func (h *EzfyHandler) ActivityInfo(c *gin.Context) {
	uid := middleware.GetUID(c)
	h.cfgs()
	h.getOrCreateCity(uid)
	now := time.Now().UnixMilli()
	var list []model.EzfyActivity
	h.DB.Order("status DESC, id ASC").Find(&list)
	views := make([]gin.H, 0, len(list))
	for _, a := range list {
		left := int64(0)
		if a.EndTime > now {
			left = (a.EndTime - now) / 1000
		}
		views = append(views, gin.H{
			"id": a.ID, "name": a.Name, "type": a.Type, "type_name": ezfyActTypeName(a.Type),
			"param": a.Param, "status": a.Status, "des": a.Des,
			"start_time": a.StartTime, "end_time": a.EndTime,
			"left_sec": left, "running": a.Status == 1 && a.StartTime <= now && a.EndTime > now,
			"effect": fmt.Sprintf("%s %d%%", ezfyActTypeName(a.Type), a.Param),
		})
	}
	resp.OK(c, gin.H{"activities": views, "now": now})
}
