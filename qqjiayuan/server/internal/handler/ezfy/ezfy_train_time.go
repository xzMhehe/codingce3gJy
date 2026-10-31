package ezfy

import "qqjiayuan/server/internal/model"

// ezfyTrainUnitSeconds 单个兵种的训练耗时（秒）。
//
// ★ 2026-10-21 用户公式（参考二战风云通用公式）：
//
//	T = floor(P * 250/(250+M) * 10/(10+L))
//
//	P = ezfy_cfg_troop.train_time（兵种初始值）
//	M = 该城现任市长的军事（主开关关掉时按 0 处理）
//	L = 动员训练科技等级（全账号共享）
//
// ★ 科技面板仍保留「造兵速度+2%/级」的原数值展示；但本公式按**等级 L**
// 取 10/(10+L)，所以 10 级时训练时间减半。两者口径不同，改公式时勿把 L 换成效果百分比。
//
// 先算单个兵再乘队列数量，保证与「面板时间」口径一致；不向上取整。
// 用整数等价式 floor(P*2500 / ((250+M)*(10+L))) 计算，避免浮点除法
// 在 1311 这类边界上出现 1310.999… → floor 成 1310 的误差。
func ezfyTrainUnitSeconds(panel, military, trainArtLevel int) int64 {
	if panel <= 0 {
		return 0
	}
	if military < 0 {
		military = 0
	}
	if trainArtLevel < 0 {
		trainArtLevel = 0
	}
	return int64(panel) * 2500 / ((int64(250) + int64(military)) * (int64(10) + int64(trainArtLevel)))
}

// mayorTrainMilitary 该城现任市长用于造兵计时的军事值。
//
//	① 主开关关 → 0（不查库，训练时间只按初始值/动员训练）；
//	② 装备开关关 → 只取军官自身军事（不含装备/套装）；
//	③ 装备开关开 → 取 officerEffective（自身 + 装备 + 套装）；
//	④ 没有市长 → 0。
//
// 只在**新建训练队列**时取一次并写进 EndTime；已有队列不会因市长/装备变化而重算。
func (h *EzfyHandler) mayorTrainMilitary(cityID uint) int {
	if !ezfyTrainMayorMilitaryOn() {
		return 0
	}
	var o model.EzfyOfficer
	if err := h.DB.Where("city_id = ? AND position = ? AND is_captive = 0", cityID, ezfyPositionMayor).
		First(&o).Error; err != nil {
		return 0
	}
	mil := o.Military
	if ezfyTrainMayorEquipOn() {
		mil, _, _ = h.officerEffective(&o)
	}
	if mil < 0 {
		return 0
	}
	return mil
}
