package ezfy

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// 建造队列上限与「征募告示」
//
// ★ 2026-10-22 用户规则：
//   - 军事区、资源区建造队列默认每个城市最多 5 个（二战系统配置可调整上限）；
//   - 使用「征募告示」后当前城市建造队列 +1，有效期 24 小时；
//   - 可叠加使用，最多叠加 10 个（即默认 5 + 10 = 最多 15 个队列）。
//
// 实现口径：
//   - 基础上限读 ezfy_cfg_limit.build_queue_max，缺行/0 回落 5；
//   - 临时加成存 ezfy_city_effect(effect_type=4)：
//     Param1 = 当前叠加层数（1~10），UntilTime = 到期毫秒时间戳；
//   - 每次使用叠加 1 层，并把有效期刷新为「最后一次使用起 24 小时」——
//     这样 10 个一起用不会变成 10 天，过期后层数整体失效；
//   - 建造/升级/一键升级都走同一套队列校验，且用城市级锁串行化，
//     避免并发请求同时通过「未满」校验后一起开工。

const (
	// ezfyBuildQueueDefaultMax 每座城市默认建造队列数
	ezfyBuildQueueDefaultMax = 5
	// ezfyBuildQueueStackMax 「征募告示」最多叠加层数
	ezfyBuildQueueStackMax = 10
	// ezfyBuildQueueHours 每层「征募告示」的有效小时数
	ezfyBuildQueueHours = 24
	// ezfyCityEffectBuildQueue 城市效果类型：4 = 建造队列 +N
	ezfyCityEffectBuildQueue = 4
)

// ezfyBuildQueueLocks 建造队列的并发锁：按城市 id 分片。
//
// 与训练/建造原锁分开，避免建造请求与训练请求互相等待；同一城市的
// 「检查队列数 → 开工 / 使用征募告示」必须串行，否则连点会突破上限。
var ezfyBuildQueueLocks [64]sync.Mutex

func ezfyBuildQueueLock(cityID uint) *sync.Mutex {
	return &ezfyBuildQueueLocks[cityID%64]
}

// ezfyBuildQueueState 当前城市的建造队列状态快照。
type ezfyBuildQueueState struct {
	Base      int   // 配置的基础上限
	Bonus     int   // 征募告示的有效层数（已过期时为 0）
	Max       int   // 当前实际上限 = Base + Bonus
	UntilTime int64 // 征募告示到期时间（毫秒；无加成为 0）
}

// ezfyBuildQueueStateOf 读取当前城市建造队列上限与征募告示加成。
func (h *EzfyHandler) ezfyBuildQueueStateOf(cityID uint) ezfyBuildQueueState {
	var e model.EzfyCityEffect
	err := h.DB.Where("city_id = ? AND effect_type = ?", cityID, ezfyCityEffectBuildQueue).
		First(&e).Error
	if err != nil {
		return ezfyBuildQueueCalc(ezfyLimit().BuildQueueMax, 0, 0, time.Now().UnixMilli())
	}
	return ezfyBuildQueueCalc(ezfyLimit().BuildQueueMax, e.Param1, e.UntilTime, time.Now().UnixMilli())
}

// ezfyBuildQueueCalc 纯计算：基础上限 + 未过期的征募告示层数 → 队列状态。
//
// ★ 拆成纯函数便于单测（不起库）：过期的层数整体失效（Bonus=0，Max=Base），
// 层数上限夹到 ezfyBuildQueueStackMax；基础上限 <=0 回落默认 5。
func ezfyBuildQueueCalc(base, stacks int, until, now int64) ezfyBuildQueueState {
	if base <= 0 {
		base = ezfyBuildQueueDefaultMax
	}
	st := ezfyBuildQueueState{Base: base, Max: base}
	if until > now && stacks > 0 {
		if stacks > ezfyBuildQueueStackMax {
			stacks = ezfyBuildQueueStackMax
		}
		st.Bonus = stacks
		st.Max = base + stacks
		st.UntilTime = until
	}
	return st
}

// ezfyBuildingQueueCount 当前城市正在建造/升级的建筑数量。
//
// 一行建筑只占一个队列：一键升级链式施工仍然只算 1 个队列。
func (h *EzfyHandler) ezfyBuildingQueueCount(cityID uint) int {
	var n int64
	h.DB.Model(&model.EzfyCityBuilding{}).
		Where("city_id = ? AND status <> 0", cityID).Count(&n)
	return int(n)
}

// ezfyBuildQueueFullMsg 队列满时返回提示；未满返回空串。
func (h *EzfyHandler) ezfyBuildQueueFullMsg(cityID uint) string {
	st := h.ezfyBuildQueueStateOf(cityID)
	n := h.ezfyBuildingQueueCount(cityID)
	if n < st.Max {
		return ""
	}
	if st.Bonus > 0 {
		return fmt.Sprintf("建造队列已满(%d/%d)，可等待队列完成后再建造", n, st.Max)
	}
	return fmt.Sprintf("建造队列已满(%d/%d)，可使用「征募告示」增加1个建造队列(24小时)", n, st.Max)
}

// ezfyAddBuildQueueEffect 使用 1 个「征募告示」：层数 +1，有效期刷新为 24 小时。
//
// 返回 (当前层数, 到期时间, 错误信息)。错误信息非空时表示未生效、不扣道具。
func (h *EzfyHandler) ezfyAddBuildQueueEffect(cityID uint) (int, int64, string) {
	lock := ezfyBuildQueueLock(cityID)
	lock.Lock()
	defer lock.Unlock()

	now := time.Now().UnixMilli()
	var e model.EzfyCityEffect
	err := h.DB.Where("city_id = ? AND effect_type = ?", cityID, ezfyCityEffectBuildQueue).
		First(&e).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, 0, "征募告示生效失败，请稍后重试"
	}

	stacks := 0
	if err == nil && e.UntilTime > now && e.Param1 > 0 {
		stacks = e.Param1
	}
	if stacks >= ezfyBuildQueueStackMax {
		return 0, 0, fmt.Sprintf("「征募告示」最多叠加%d个", ezfyBuildQueueStackMax)
	}
	stacks++
	until := now + int64(ezfyBuildQueueHours)*3600*1000

	if err == nil {
		res := h.DB.Model(&model.EzfyCityEffect{}).Where("id = ?", e.ID).
			Updates(map[string]interface{}{"param1": stacks, "until_time": until})
		if res.Error != nil {
			return 0, 0, "征募告示生效失败，请稍后重试"
		}
	} else {
		e = model.EzfyCityEffect{
			CityId:     int64(cityID),
			EffectType: ezfyCityEffectBuildQueue,
			Param1:     stacks,
			UntilTime:  until,
		}
		if createErr := h.DB.Create(&e).Error; createErr != nil {
			return 0, 0, "征募告示生效失败，请稍后重试"
		}
	}
	return stacks, until, ""
}
