package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云 管理端 —— 宣战管理（本轮新增）
//
// 背景：玩家侧宣战（DeclareWar）有 24 小时延迟生效、48 小时有效期，
// 管理端原先只能看城市/军团，**看不到任何宣战记录**，出问题只能查库。
// 这里补一个完整的维护页 + 一键操作：
//
//	1. 列表：所有宣战记录（含双方昵称/号码、状态、剩余时间）
//	2. 一键生效：把「宣战待生效」立即变成「交战中」（跳过 24 小时等待）
//	3. 一键完成/结束：把记录直接置为已结束（等同到期）
//	4. 新建宣战：管理员代玩家发起，可选择「立即生效」
//	5. 删除记录：清掉误建的历史数据
//
// ★ 状态约定（与 model.EzfyWar 严格一致，别自创）：
//
//	1 = 宣战待生效   2 = 交战中   3 = 已结束（管理员主动结束 / 到期归档）
//
// ★ 玩家侧 getWar 只认 status IN (1,2)，所以「结束」写 3 就自然失效，
//   不需要物理删除；到期是靠 ExpireTime 判定的，此处不重复实现。

// ezfyWarStatusText 状态中文名
func ezfyWarStatusText(s int) string {
	switch s {
	case 1:
		return "宣战待生效"
	case 2:
		return "交战中"
	case 3:
		return "已结束"
	}
	return "未知"
}

// ezfyWarLiveStatus 结合时间戳算出「当前实际状态」
//
// 库里 status 可能是 1，但 EffectTime 早就过了 —— 玩家侧 warStatus()
// 是惰性刷新的（有人访问才改库），管理端列表直接读库会显示成
// 「宣战待生效」但实际上早就打起来了。这里统一按时间算真实状态，
// 避免管理员被陈旧数据误导。
func ezfyWarLiveStatus(w *model.EzfyWar) int {
	if w.Status == 3 {
		return 3
	}
	now := time.Now().UnixMilli()
	if w.ExpireTime > 0 && now >= w.ExpireTime {
		return 3
	}
	if w.EffectTime > 0 && now >= w.EffectTime {
		return 2
	}
	return 1
}

// AdminEzfyWars GET /admin/ezfy-wars
//
// 查询参数：word（按昵称/家园号码模糊）、status（0 全部 / 1 待生效 / 2 交战中 / 3 已结束）
func (h *AdminHandler) AdminEzfyWars(c *gin.Context) {
	h.ezfyH().cfgs()
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	status := atoiOr(c.Query("status"), 0)

	q := h.DB.Model(&model.EzfyWar{})
	if status >= 1 && status <= 3 {
		// ★ 按「真实状态」过滤不能只靠 SQL：status=3 可能是到期导致的，
		//   所以先把候选收窄到时间维度，再在内存里精筛。
		now := time.Now().UnixMilli()
		switch status {
		case 1:
			q = q.Where("status = 1 AND effect_time > ?", now)
		case 2:
			q = q.Where("status IN (1,2) AND effect_time <= ? AND expire_time > ?", now, now)
		case 3:
			q = q.Where("status = 3 OR expire_time <= ?", now)
		}
	}
	if word != "" {
		// 先按昵称/家园号码定位玩家，再用其 uid 过滤
		var uids []uint
		h.DB.Model(&model.EzfyProfile{}).Where("nickname LIKE ?", "%"+word+"%").
			Pluck("user_id", &uids)
		var ids []uint
		h.DB.Model(&model.User{}).Where("username LIKE ?", "%"+word+"%").
			Pluck("id", &ids)
		uids = append(uids, ids...)
		if len(uids) == 0 {
			resp.OK(c, gin.H{"list": []interface{}{}, "total": 0, "page": page, "size": size})
			return
		}
		q = q.Where("atk_user_id IN ? OR def_user_id IN ?", uids, uids)
	}

	var total int64
	q.Count(&total)
	var rows []model.EzfyWar
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)

	type rowOut struct {
		model.EzfyWar
		AtkNick   string `json:"atk_nick"`
		AtkNum    string `json:"atk_num"`
		DefNick   string `json:"def_nick"`
		DefNum    string `json:"def_num"`
		LiveState int    `json:"live_status"`      // 按时间算出的真实状态
		StateText string `json:"state_text"`       // 真实状态中文
		LeftText  string `json:"left_text"`        // 剩余时间说明
		DeclareAt string `json:"declare_at_text"`  // 宣战时间（可读）
		EffectAt  string `json:"effect_at_text"`   // 生效时间（可读）
		ExpireAt  string `json:"expire_at_text"`   // 到期时间（可读）
	}
	out := []rowOut{}
	for i := range rows {
		w := rows[i]
		an, anum := h.ezfyAdminName(w.AtkUserId)
		dn, dnum := h.ezfyAdminName(w.DefUserId)
		live := ezfyWarLiveStatus(&w)
		out = append(out, rowOut{
			EzfyWar: w,
			AtkNick: an, AtkNum: anum, DefNick: dn, DefNum: dnum,
			LiveState: live, StateText: ezfyWarStatusText(live),
			LeftText:  ezfyWarLeftText(&w, live),
			DeclareAt: ezfyFmtMs(w.DeclareTime),
			EffectAt:  ezfyFmtMs(w.EffectTime),
			ExpireAt:  ezfyFmtMs(w.ExpireTime),
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size,
		"delay_hours": ezfyWarDelayHours, "duration_hours": ezfyWarDurationHours})
}

// ezfyWarLeftText 剩余时间文案
func ezfyWarLeftText(w *model.EzfyWar, live int) string {
	now := time.Now().UnixMilli()
	switch live {
	case 1:
		h := (w.EffectTime - now + 3599999) / 3600000
		if h < 1 {
			h = 1
		}
		return fmt.Sprintf("约 %d 小时后开战", h)
	case 2:
		h := (w.ExpireTime - now) / 3600000
		if h < 0 {
			h = 0
		}
		return fmt.Sprintf("剩余约 %d 小时", h)
	}
	return "—"
}

// ezfyFmtMs 毫秒时间戳 → "2006-01-02 15:04"
func ezfyFmtMs(ms int64) string {
	if ms <= 0 {
		return "—"
	}
	return time.UnixMilli(ms).Format("2006-01-02 15:04")
}

// AdminEzfyWarCreate POST /admin/ezfy-wars
//
// 管理员代玩家发起宣战。
// instant=true 时立即生效（跳过 24 小时等待），适合内测时让玩家马上能打。
func (h *AdminHandler) AdminEzfyWarCreate(c *gin.Context) {
	var in struct {
		AtkUserId  uint `json:"atk_user_id"`
		DefUserId  uint `json:"def_user_id"`
		Instant    bool `json:"instant"`
		DelayHours int  `json:"delay_hours"` // 0 = 用默认 24
		KeepHours  int  `json:"keep_hours"`  // 0 = 用默认 48
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.AtkUserId == 0 || in.DefUserId == 0 {
		resp.ParamError(c, "请填写宣战方与被宣战方玩家ID")
		return
	}
	if in.AtkUserId == in.DefUserId {
		resp.ParamError(c, "不能对自己宣战")
		return
	}
	// 玩家必须存在（有游戏档案）
	var n int64
	h.DB.Model(&model.EzfyProfile{}).Where("user_id = ?", in.AtkUserId).Count(&n)
	if n == 0 {
		resp.ParamError(c, fmt.Sprintf("宣战方 %d 没有二战风云档案", in.AtkUserId))
		return
	}
	h.DB.Model(&model.EzfyProfile{}).Where("user_id = ?", in.DefUserId).Count(&n)
	if n == 0 {
		resp.ParamError(c, fmt.Sprintf("被宣战方 %d 没有二战风云档案", in.DefUserId))
		return
	}
	if h.ezfyH().getWar(in.AtkUserId, in.DefUserId) != nil {
		resp.ParamError(c, "双方已存在进行中的宣战记录（待生效或交战中）")
		return
	}

	delay := in.DelayHours
	if delay <= 0 {
		delay = ezfyWarDelayHours
	}
	if delay > 720 {
		resp.ParamError(c, "延迟小时数最大 720（30 天）")
		return
	}
	keep := in.KeepHours
	if keep <= 0 {
		keep = ezfyWarDurationHours
	}
	if keep > 8760 {
		resp.ParamError(c, "有效小时数最大 8760（1 年）")
		return
	}

	now := time.Now().UnixMilli()
	if in.Instant {
		delay = 0
	}
	// delay=0 时 EffectTime=now，玩家侧 warStatus() 会立即判定为交战中
	w := model.EzfyWar{
		AtkUserId: in.AtkUserId, DefUserId: in.DefUserId,
		Status:      1,
		DeclareTime: now,
		EffectTime:  now + int64(delay)*3600000,
		ExpireTime:  now + int64(delay+keep)*3600000,
	}
	if in.Instant {
		w.Status = 2
	}
	if err := h.DB.Create(&w).Error; err != nil {
		resp.ParamError(c, "创建失败："+err.Error())
		return
	}
	// 通知被宣战方（与玩家侧宣战保持一致的提示口径）
	an, _ := h.ezfyAdminName(in.AtkUserId)
	if an == "" {
		an = fmt.Sprintf("玩家%d", in.AtkUserId)
	}
	tip := fmt.Sprintf("【宣战】%s 向你宣战，%d 小时后生效，生效后 %d 小时内可互相掠夺/征服。",
		an, delay, keep)
	if in.Instant {
		tip = fmt.Sprintf("【宣战】%s 向你宣战，已立即生效，%d 小时内可互相掠夺/征服。", an, keep)
	}
	h.DB.Create(&model.EzfyNotice{UserId: in.DefUserId, Title: "宣战", Content: tip})

	resp.OK(c, gin.H{"msg": fmt.Sprintf("已创建宣战：%s(id%d) → %s(id%d)，%s",
		an, in.AtkUserId, mustNick(h, in.DefUserId), in.DefUserId, ezfyWarStatusText(w.Status)),
		"id": w.ID})
}

func mustNick(h *AdminHandler, uid uint) string {
	n, _ := h.ezfyAdminName(uid)
	if n == "" {
		return fmt.Sprintf("玩家%d", uid)
	}
	return n
}

// AdminEzfyWarFinish POST /admin/ezfy-wars/:id/finish
//
// 一键完成宣战：把记录置为「已结束」（status=3），双方立刻恢复和平。
// 这是「一键完成宣战功能」的核心实现 —— 无论当前是待生效还是交战中都能直接结束。
func (h *AdminHandler) AdminEzfyWarFinish(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var w model.EzfyWar
	if err := h.DB.First(&w, id).Error; err != nil {
		resp.NotFound(c, "宣战记录不存在")
		return
	}
	if w.Status == 3 {
		resp.ParamError(c, "该宣战已经是结束状态")
		return
	}
	now := time.Now().UnixMilli()
	// 同时把 ExpireTime 提前到当下：玩家侧 warStatus() 是靠 ExpireTime 判 0 的，
	// 只改 status 的话，如果 ExpireTime 还没到，getWar 仍会捞到它（status IN 1,2 才捞，
	// 但我们把 status 改成 3 后就捞不到了）—— 双保险，两边都改更稳。
	if err := h.DB.Model(&model.EzfyWar{}).Where("id = ?", w.ID).
		Updates(map[string]interface{}{"status": 3, "expire_time": now}).Error; err != nil {
		resp.ParamError(c, "操作失败："+err.Error())
		return
	}
	// 通知双方
	an, _ := h.ezfyAdminName(w.AtkUserId)
	dn, _ := h.ezfyAdminName(w.DefUserId)
	msg := fmt.Sprintf("【宣战结束】%s 与 %s 的战争状态已由管理员结束。", an, dn)
	h.DB.Create(&model.EzfyNotice{UserId: w.AtkUserId, Title: "宣战结束", Content: msg})
	h.DB.Create(&model.EzfyNotice{UserId: w.DefUserId, Title: "宣战结束", Content: msg})
	resp.OK(c, gin.H{"msg": "已结束该宣战（双方恢复和平）"})
}

// AdminEzfyWarEffect POST /admin/ezfy-wars/:id/effect
//
// 一键生效：把「宣战待生效」立即变成「交战中」，跳过剩余的等待时间。
// 与「一键完成」相对 —— 一个是马上开战，一个是马上停战。
func (h *AdminHandler) AdminEzfyWarEffect(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var w model.EzfyWar
	if err := h.DB.First(&w, id).Error; err != nil {
		resp.NotFound(c, "宣战记录不存在")
		return
	}
	live := ezfyWarLiveStatus(&w)
	if live == 2 {
		resp.ParamError(c, "该宣战已经在交战中")
		return
	}
	if live == 3 {
		resp.ParamError(c, "该宣战已结束，不能再生效（如需重新开战请新建）")
		return
	}
	now := time.Now().UnixMilli()
	// 生效时间拉到当下；如果到期时间比现在早，顺带按默认时长续上，
	// 避免「刚点生效就立刻过期」这种反直觉结果。
	expire := w.ExpireTime
	if expire <= now {
		expire = now + ezfyWarDurationHours*3600000
	}
	if err := h.DB.Model(&model.EzfyWar{}).Where("id = ?", w.ID).
		Updates(map[string]interface{}{"status": 2, "effect_time": now, "expire_time": expire}).Error; err != nil {
		resp.ParamError(c, "操作失败："+err.Error())
		return
	}
	an, _ := h.ezfyAdminName(w.AtkUserId)
	dn, _ := h.ezfyAdminName(w.DefUserId)
	msg := fmt.Sprintf("【宣战生效】%s 与 %s 已进入交战状态，可互相掠夺/征服。", an, dn)
	h.DB.Create(&model.EzfyNotice{UserId: w.AtkUserId, Title: "宣战生效", Content: msg})
	h.DB.Create(&model.EzfyNotice{UserId: w.DefUserId, Title: "宣战生效", Content: msg})
	resp.OK(c, gin.H{"msg": "已立即生效，双方进入交战状态"})
}

// AdminEzfyWarUpdate PUT /admin/ezfy-wars/:id
//
// 调整生效/到期时间（管理员排查问题时常用：延长内战、缩短等待）。
// 时间单位为毫秒时间戳；传 hours 更直观，这里两套都支持。
func (h *AdminHandler) AdminEzfyWarUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var w model.EzfyWar
	if err := h.DB.First(&w, id).Error; err != nil {
		resp.NotFound(c, "宣战记录不存在")
		return
	}
	var in struct {
		DelayHours *int `json:"delay_hours"` // 从现在起多久生效
		KeepHours  *int `json:"keep_hours"`  // 生效后持续多久
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	now := time.Now().UnixMilli()
	updates := map[string]interface{}{}
	if in.DelayHours != nil {
		d := *in.DelayHours
		if d < 0 || d > 720 {
			resp.ParamError(c, "生效延迟需在 0~720 小时之间")
			return
		}
		updates["effect_time"] = now + int64(d)*3600000
	}
	if in.KeepHours != nil {
		k := *in.KeepHours
		if k <= 0 || k > 8760 {
			resp.ParamError(c, "持续时间需在 1~8760 小时之间")
			return
		}
		base := w.EffectTime
		if v, ok := updates["effect_time"]; ok {
			base = v.(int64)
		}
		updates["expire_time"] = base + int64(k)*3600000
	}
	if len(updates) == 0 {
		resp.ParamError(c, "没有要修改的内容")
		return
	}
	// 改完时间后重算一次状态，避免「待生效」但时间已过期的尴尬
	fake := w
	if v, ok := updates["effect_time"]; ok {
		fake.EffectTime = v.(int64)
	}
	if v, ok := updates["expire_time"]; ok {
		fake.ExpireTime = v.(int64)
	}
	updates["status"] = ezfyWarLiveStatus(&fake)
	if err := h.DB.Model(&model.EzfyWar{}).Where("id = ?", w.ID).Updates(updates).Error; err != nil {
		resp.ParamError(c, "保存失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "已保存"})
}

// AdminEzfyWarDelete DELETE /admin/ezfy-wars/:id
func (h *AdminHandler) AdminEzfyWarDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var w model.EzfyWar
	if err := h.DB.First(&w, id).Error; err != nil {
		resp.NotFound(c, "宣战记录不存在")
		return
	}
	if err := h.DB.Delete(&model.EzfyWar{}, id).Error; err != nil {
		resp.ParamError(c, "删除失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"msg": "已删除宣战记录"})
}

// AdminEzfyWarFinishAll POST /admin/ezfy-wars/finish-all
//
// ★ 一键完成全部宣战：把当前所有未结束的宣战一次性结束。
// 内测收尾/服务器活动结束时常用，避免玩家之间还挂着一堆战争状态。
func (h *AdminHandler) AdminEzfyWarFinishAll(c *gin.Context) {
	now := time.Now().UnixMilli()
	var rows []model.EzfyWar
	h.DB.Where("status IN (1,2) AND expire_time > ?", now).Find(&rows)
	if len(rows) == 0 {
		resp.OK(c, gin.H{"msg": "当前没有进行中的宣战", "count": 0})
		return
	}
	if err := h.DB.Model(&model.EzfyWar{}).
		Where("status IN (1,2) AND expire_time > ?", now).
		Updates(map[string]interface{}{"status": 3, "expire_time": now}).Error; err != nil {
		resp.ParamError(c, "操作失败："+err.Error())
		return
	}
	// 通知涉及的玩家（去重）
	seen := map[uint]bool{}
	body := "【宣战结束】服务器已结束当前全部战争状态，所有玩家恢复和平。"
	for _, w := range rows {
		for _, uid := range []uint{w.AtkUserId, w.DefUserId} {
			if seen[uid] {
				continue
			}
			seen[uid] = true
			h.DB.Create(&model.EzfyNotice{UserId: uid, Title: "宣战结束", Content: body})
		}
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已结束 %d 条宣战记录，涉及 %d 名玩家", len(rows), len(seen)),
		"count": len(rows)})
}

// AdminEzfyWarEffectAll POST /admin/ezfy-wars/effect-all
//
// ★ 一键生效全部宣战：所有「待生效」的记录立即开战。
// 内测开局常见需求 —— 玩家宣战后不想等 24 小时，管理员一键全部放开。
func (h *AdminHandler) AdminEzfyWarEffectAll(c *gin.Context) {
	now := time.Now().UnixMilli()
	var rows []model.EzfyWar
	h.DB.Where("status = 1 AND effect_time > ? AND expire_time > ?", now, now).Find(&rows)
	if len(rows) == 0 {
		resp.OK(c, gin.H{"msg": "当前没有待生效的宣战", "count": 0})
		return
	}
	if err := h.DB.Model(&model.EzfyWar{}).
		Where("status = 1 AND effect_time > ? AND expire_time > ?", now, now).
		Updates(map[string]interface{}{"status": 2, "effect_time": now}).Error; err != nil {
		resp.ParamError(c, "操作失败："+err.Error())
		return
	}
	seen := map[uint]bool{}
	body := "【宣战生效】服务器已让所有待生效的宣战立即生效，可互相掠夺/征服。"
	for _, w := range rows {
		for _, uid := range []uint{w.AtkUserId, w.DefUserId} {
			if seen[uid] {
				continue
			}
			seen[uid] = true
			h.DB.Create(&model.EzfyNotice{UserId: uid, Title: "宣战生效", Content: body})
		}
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已让 %d 条宣战立即生效，涉及 %d 名玩家", len(rows), len(seen)),
		"count": len(rows)})
}
