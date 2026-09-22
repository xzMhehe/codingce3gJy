package handler

import (
	"fmt"
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

// activeActivity 取当前进行中的某类活动(同类多个时取加成最大的)
func (h *EzfyHandler) activeActivity(actType int) *model.EzfyActivity {
	now := time.Now().UnixMilli()
	var list []model.EzfyActivity
	h.DB.Where("type = ? AND status = 1 AND start_time <= ? AND end_time > ?", actType, now, now).
		Order("param DESC").Limit(1).Find(&list)
	if len(list) == 0 {
		return nil
	}
	return &list[0]
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
// ★ 用户要求「加个征兵资源消耗开关，默认开；关了的话征兵不消耗资源」→
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
