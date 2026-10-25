// 战力榜口径（★ 2026-10-25 重做兵种分）
//
// 用户反馈：「战力榜 兵种那个不合理 —— 有人侦察机多 排名第一，应该按兵种占用人口、数量来算；
// 注意不要太大，现在都有 20 亿兵的，这种兵种分算到 5 万左右合理」。
//
// 旧口径的根因：兵种「质量」= (生命+防御+三系攻击+速度+射程)/100，而**侦察机速度 5000**（全兵种最高），
// 质量算出来 50.95，比战列舰(45.88)还高；再乘 count^0.8 —— 谁侦察机堆得多谁第一。
//
// 新口径：兵种分只看「占用人口 × 数量」，**纯人口口径、不加任何其它加成**
// （加成会翻转名次 —— 兵多的人反而排后面，就是用户抱怨的那类不合理）。
//   · 侦察机 1 人口 / 航母 55 人口 → 20 亿侦察机 ≈ 3.6 亿航母，堆垃圾兵再也刷不到第一；
//   · 人口占用总量 popSum 走**饱和曲线**，恒不超过上限（默认 5 万）——
//     20 亿兵也不会把战力顶成天文数字（用户：『算到 5 万左右合理，别太大』）。
//
// 战力 = 科技战力 + 建筑战力 + 兵种战力，三项**直接相加**：
// 科技/建筑各自 ^power_compress_pow 压缩；兵种分本身已封顶，不能再套一层压缩（否则总和被压成几百）。
package ezfy

import "math"

const (
	// ezfyPowerTroopCap 兵种分上限（用户口径：5 万左右）。
	ezfyPowerTroopCap = 50000.0
	// ezfyPowerTroopHalf 半程人口：人口占用总量达到该值时兵种分 = 上限的一半。
	// 80 亿人口 ≈ 1.5 亿航母 / 80 亿步兵。当前全服第一约 700 亿人口 → 约 4.7 万分
	// （贴近上限但不爆表，且前几名之间仍有明显差距，不会全员并列 5 万）。
	ezfyPowerTroopHalf = 8e9
)

// ezfyTroopPop 兵种占用人口（配置缺失兜底 1，避免出现 0 人口兵种把分算成 0）。
func ezfyTroopPop(id int) float64 {
	if t := ezfyCfg.troop(id); t != nil && t.Pop > 0 {
		return float64(t.Pop)
	}
	return 1
}

// ezfyTroopPower 兵种战力（★ 唯一实现）：饱和曲线，严格单调递增、恒 < 上限。
//
//	troop = 上限 × popSum / (popSum + 半程人口)
//
//	popSum=0（没兵）→ 0；popSum=半程(80 亿) → 2.5 万；popSum→∞ 逼近 5 万。
func ezfyTroopPower(popSum float64) float64 {
	if popSum <= 0 {
		return 0
	}
	return ezfyPowerTroopCap * popSum / (popSum + ezfyPowerTroopHalf)
}

// ezfyPowerParts 三项原始分 → 展示战力（★ 唯一实现：玩家端战力榜 / 管理端战力榜 / 军团榜共用）。
//
//	power = techRaw^cp + buildRaw^cp + 兵种分      tech/build/troop 三列之和 == power
//
// 旧实现是 (tech+build+troop)^cp 再按各分项占比归一化拆明细；现在兵种分已自带压缩与封顶，
// 直接相加即可（明细天然就是三项本身，不用再归一化）。
func ezfyPowerParts(techRaw, buildRaw, popSum float64, cp float64) (power, tech, build, troop float64) {
	if cp <= 0 {
		cp = 0.5
	}
	tech = math.Pow(techRaw, cp)
	build = math.Pow(buildRaw, cp)
	troop = ezfyTroopPower(popSum)
	return tech + build + troop, tech, build, troop
}
