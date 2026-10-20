package ezfy

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
)

// ezfyMyWars 当前玩家进行中的个人宣战（宣战待生效 / 交战中）。
//
// ★★ 2026-10-20 玩家反馈「不知道自己跟谁是对战中、跟谁是宣战状态」：
//
//	原来宣战状态只在「地图 → 点对方城市」时才看得到，玩家切走后就找不到对手了。
//	统帅页现在直接列出全部进行中的宣战/交战（攻守双方都算），每行带对方昵称、
//	当前城市与坐标，点名字还能进对方统帅页。
//
// 口径：
//   - status IN (1,2) 且 expire_time > now（过期的老记录不再展示）；
//   - 展示状态按 effect_time 现算（库里 status 可能还是 1，但到点其实已经开战）；
//   - 城市取对方 profile.current_city_id，无效时回落该玩家 id 最小的城。
func (h *EzfyHandler) ezfyMyWars(uid uint) []gin.H {
	now := time.Now().UnixMilli()
	var wars []model.EzfyWar
	h.DB.Where("status IN (1,2) AND expire_time > ? AND (atk_user_id = ? OR def_user_id = ?)", now, uid, uid).
		Order("id DESC").Find(&wars)
	if len(wars) == 0 {
		return []gin.H{}
	}

	// 去重后的对手 uid（一场战争只查一次档案/城市）
	others := make([]uint, 0, len(wars))
	seen := make(map[uint]bool, len(wars))
	for _, w := range wars {
		other := w.AtkUserId
		if other == uid {
			other = w.DefUserId
		}
		if other == 0 || seen[other] {
			continue
		}
		seen[other] = true
		others = append(others, other)
	}
	if len(others) == 0 {
		return []gin.H{}
	}

	// 对手档案（昵称 + 当前城）
	var profiles []model.EzfyProfile
	h.DB.Select("user_id, nickname, current_city_id").Where("user_id IN ?", others).Find(&profiles)
	profileByUser := make(map[uint]model.EzfyProfile, len(profiles))
	for _, p := range profiles {
		profileByUser[p.UserID] = p
	}

	// 对手城市一次取齐：优先 current_city_id，无效时回落 id 最小的城
	var cities []model.EzfyCity
	h.DB.Select("id, user_id, name, x, y").Where("user_id IN ?", others).Order("id ASC").Find(&cities)
	cityByID := make(map[uint]model.EzfyCity, len(cities))
	cityByUser := make(map[uint]model.EzfyCity, len(cities))
	for _, c := range cities {
		cityByID[c.ID] = c
		if _, ok := cityByUser[c.UserID]; !ok {
			cityByUser[c.UserID] = c
		}
	}
	for _, p := range profiles {
		if p.CurrentCityId <= 0 {
			continue
		}
		if c, ok := cityByID[uint(p.CurrentCityId)]; ok {
			cityByUser[p.UserID] = c
		}
	}

	// 昵称兜底：二战档案没有昵称时取家园昵称（批量，避免 N+1）
	var missing []uint
	for _, other := range others {
		if profileByUser[other].Nickname == "" {
			missing = append(missing, other)
		}
	}
	homeNick := map[uint]string{}
	if len(missing) > 0 {
		var us []model.User
		h.home().Select("id, nickname").Where("id IN ?", missing).Find(&us)
		for _, u := range us {
			homeNick[u.ID] = u.Nickname
		}
	}

	rows := make([]gin.H, 0, len(wars))
	for _, w := range wars {
		mineIsAtk := w.AtkUserId == uid
		other := w.DefUserId
		if !mineIsAtk {
			other = w.AtkUserId
		}
		if other == 0 || other == uid {
			continue
		}

		name := profileByUser[other].Nickname
		if name == "" {
			name = homeNick[other]
		}
		if name == "" {
			name = fmt.Sprintf("玩家%d", other)
		}

		status := 1
		leftHours := (w.EffectTime - now + 3599999) / 3600000
		if now >= w.EffectTime {
			status = 2
			leftHours = (w.ExpireTime - now + 3599999) / 3600000
		}
		if leftHours < 0 {
			leftHours = 0
		}

		c := cityByUser[other]
		rows = append(rows, gin.H{
			"id":          w.ID,
			"mine_is_atk": mineIsAtk,
			"status":      status,
			"user_id":     other,
			"nickname":    name,
			"city_id":     c.ID,
			"city_name":   c.Name,
			"x":           c.X,
			"y":           c.Y,
			"hours_left":  leftHours,
			"effect_time": w.EffectTime,
			"expire_time": w.ExpireTime,
		})
	}
	return rows
}
