package ezfy

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云管理端 —— 活动野地配置（地图管理「活动野地」tab）
//
// ★ 2026-09-29 活动野地配置不友好，优化成「按坐标列表管理」，
//   独立成 tab，且要能区别于普通野地（启用开关）+ 可配置活动（等级/守军/奖励）。
//
// 数据表 ezfy_act_wild：每个坐标一条。enabled=1 → 该格按活动野地玩法
//   （守军/奖励/等级用配置，缺省回退代码默认）；enabled=0 → 该格不是活动目标（普通野地）。
// 运行时判定走 ezfy_act_wild 优先 → mark 覆盖 → 坐标哈希（见 ezfyActTargetType）。

var ezfyActWildFields = map[string]string{
	"x": "int", "y": "int", "enabled": "int",
	"level": "int", "troops": "string", "res": "int64",
	"gold": "int64", "prestige": "int", "jewel": "string", "des": "string",
	"officer_id": "int", "treasures": "string", "capture_rate": "int", "max_capture": "int",
	// ★ 2026-10-24 守军等级 / 被俘等级 / 守军行为（按兵种 JSON map）
	"def_level": "int", "captive_level": "int", "def_move_dir": "string",
	// ★ 2026-10-25 兜底被俘次数（至少打赢多少次才可能被俘；0 = 直接按概率）
	"min_capture_wins": "int",
}

// AdminEzfyActWildList GET /admin/ezfy-act-wilds —— 活动野地配置列表（分页）
func (h *EzfyAdmin) AdminEzfyActWildList(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	word := strings.TrimSpace(c.Query("word"))
	enabled := atoiOr(c.Query("enabled"), -1)
	q := h.DB.Model(&model.EzfyActWild{})
	if word != "" {
		if parts := strings.SplitN(word, ",", 2); len(parts) == 2 {
			if x, e1 := strconv.Atoi(strings.TrimSpace(parts[0])); e1 == nil {
				if y, e2 := strconv.Atoi(strings.TrimSpace(parts[1])); e2 == nil {
					q = q.Where("x = ? AND y = ?", x, y)
				}
			}
		} else {
			q = q.Where("des LIKE ?", "%"+word+"%")
		}
	}
	if enabled >= 0 {
		q = q.Where("enabled = ?", enabled)
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyActWild
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)

	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		oName := ""
		oStar := 0
		if g := ezfyCfg.general(r.OfficerId); g != nil {
			oName = g.Name
			oStar = g.Star
		}
		out = append(out, gin.H{
			"id": r.ID, "x": r.X, "y": r.Y, "enabled": r.Enabled,
			"level": r.Level, "troops": r.Troops, "res": r.Res,
			"gold": r.Gold, "prestige": r.Prestige, "jewel": r.Jewel, "des": r.Des,
			"officer_id": r.OfficerId, "officer_name": oName, "officer_star": oStar,
			"treasures": r.Treasures, "capture_rate": r.CaptureRate, "max_capture": r.MaxCapture,
			// ★ 2026-10-24 守军等级 + 被俘等级 + 守军行为（按兵种 JSON map，编辑弹窗回填用）
			"def_level": r.DefLevel, "captive_level": r.CaptiveLevel, "def_move_dir": r.DefMoveDir,
			// ★ 2026-10-25 兜底被俘次数（**仅管理端**，用户端不下发）
			"min_capture_wins": r.MinCaptureWins,
			// 当前生效判定（enabled=1 才算活动目标）
			"eff_act": (r.Enabled == 1),
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyActWildSave POST /admin/ezfy-act-wilds —— 新增/更新某坐标的活动野地配置（按 x,y 幂等）
func (h *EzfyAdmin) AdminEzfyActWildSave(c *gin.Context) {
	var in map[string]interface{}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	vals := xyPickVals(in, ezfyActWildFields)
	x, _ := vals["x"].(int)
	y, _ := vals["y"].(int)
	if x <= 0 || y <= 0 {
		resp.ParamError(c, "请填写坐标 x / y（世界范围 ~50~450）")
		return
	}
	if msg := checkActWildVals(vals); msg != "" {
		resp.ParamError(c, msg)
		return
	}
	// 未填字段补默认
	if _, ok := vals["enabled"]; !ok {
		vals["enabled"] = 0
	}
	if _, ok := vals["level"]; !ok {
		vals["level"] = 0
	}
	if _, ok := vals["res"]; !ok {
		vals["res"] = 0
	}
	if _, ok := vals["gold"]; !ok {
		vals["gold"] = 0
	}
	if _, ok := vals["prestige"]; !ok {
		vals["prestige"] = 0
	}
	if s, _ := vals["troops"].(string); s == "" {
		vals["troops"] = ""
	}
	if s, _ := vals["jewel"].(string); s == "" {
		vals["jewel"] = ""
	}
	if s, _ := vals["des"].(string); s == "" {
		vals["des"] = ""
	}
	if _, ok := vals["officer_id"]; !ok {
		vals["officer_id"] = 0
	}
	if s, _ := vals["treasures"].(string); s == "" {
		vals["treasures"] = ""
	}
	if _, ok := vals["capture_rate"]; !ok {
		vals["capture_rate"] = 0
	}
	if _, ok := vals["max_capture"]; !ok {
		vals["max_capture"] = 1 // 默认同一玩家可抓 1 次
	}
	// ★ 2026-10-24 守军微调默认值：等级/被俘等级 0 = 用军官池默认；守军行为空 = 全部默认前进
	for _, k := range []string{"def_level", "captive_level"} {
		if _, ok := vals[k]; !ok {
			vals[k] = 0
		}
	}
	// ★ 2026-10-25 兜底被俘次数默认 0 = 第一次打赢就按概率（原行为）
	if _, ok := vals["min_capture_wins"]; !ok {
		vals["min_capture_wins"] = 0
	}
	if _, ok := vals["def_move_dir"]; !ok {
		vals["def_move_dir"] = ""
	}
	// ★★ 2026-10-05 修复「编辑会新增一个」：
	//
	//	原来**只按 (x,y) upsert**（不带 id）。在「编辑」对话框里改一下坐标再保存，
	//	就会在新坐标**新增一条**、而旧坐标那条仍然留着 —— 玩家看到的是「编辑变成了新增」。
	//	现在：带了 id 就**按 id 更新那一行**（坐标跟着一起改，等于「移动这条配置」）；
	//	没带 id（新增）才按 (x,y) upsert（同坐标再存一次仍然幂等更新，不会重复）。
	//
	//	两种情况都做「目标坐标已被别的行占用」检查 —— 否则会出现同坐标两条，
	//	而运行时配置是按坐标建 map 的，谁后读到谁生效，表现为「配置时灵时不灵」。
	if id := ezfyAnyInt(in["id"]); id > 0 {
		var cur model.EzfyActWild
		if err := h.DB.First(&cur, id).Error; err != nil {
			resp.NotFound(c, "该活动野地配置不存在（可能已被删除），请刷新后重试")
			return
		}
		var dup int64
		h.DB.Model(&model.EzfyActWild{}).Where("x = ? AND y = ? AND id <> ?", x, y, id).Count(&dup)
		if dup > 0 {
			resp.ParamError(c, fmt.Sprintf("坐标 (%d,%d) 已存在另一条活动野地配置，请先删除或改用其它坐标", x, y))
			return
		}
		if err := h.DB.Model(&model.EzfyActWild{}).Where("id = ?", id).Updates(vals).Error; err != nil {
			resp.ParamError(c, "保存失败："+err.Error())
			return
		}
		h.ezfyReload()
		ok := "关闭（普通野地）"
		if v, _ := vals["enabled"].(int); v == 1 {
			ok = "启用（活动野地）"
		}
		resp.OK(c, gin.H{"msg": fmt.Sprintf("坐标 (%d,%d) 已保存并立即生效：%s", x, y, ok)})
		return
	}
	if err := h.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "x"}, {Name: "y"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"enabled", "level", "troops", "res", "gold", "prestige", "jewel", "des", "officer_id",
			"treasures", "capture_rate", "max_capture",
			"def_level", "captive_level", "def_move_dir", "min_capture_wins",
		}),
	}).Model(&model.EzfyActWild{}).Create(vals).Error; err != nil {
		resp.ParamError(c, "保存失败："+err.Error())
		return
	}
	h.ezfyReload()
	ok := "关闭（普通野地）"
	if v, _ := vals["enabled"].(int); v == 1 {
		ok = "启用（活动野地）"
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("坐标 (%d,%d) 已保存并立即生效：%s", x, y, ok)})
}

// AdminEzfyActWildToggle POST /admin/ezfy-act-wilds/:id/toggle —— 切换启用开关
func (h *EzfyAdmin) AdminEzfyActWildToggle(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var a model.EzfyActWild
	if err := h.DB.First(&a, id).Error; err != nil {
		resp.NotFound(c, "该活动野地配置不存在")
		return
	}
	a.Enabled = 1 - a.Enabled
	h.DB.Model(&model.EzfyActWild{}).Where("id = ?", id).Update("enabled", a.Enabled)
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": fmt.Sprintf("坐标 (%d,%d) 已%s", a.X, a.Y, map[bool]string{true: "启用为活动野地", false: "关闭（按普通野地）"}[a.Enabled == 1])})
}

// AdminEzfyActWildDelete DELETE /admin/ezfy-act-wilds/:id —— 删除配置，恢复按哈希/mark 判定
func (h *EzfyAdmin) AdminEzfyActWildDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var a model.EzfyActWild
	if err := h.DB.First(&a, id).Error; err != nil {
		resp.NotFound(c, "该活动野地配置不存在")
		return
	}
	h.DB.Delete(&model.EzfyActWild{}, id)
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": fmt.Sprintf("坐标 (%d,%d) 的活动野地配置已删除，恢复默认判定", a.X, a.Y)})
}

// AdminEzfyActWildBatchDelete POST /admin/ezfy-act-wilds/batch-delete  {ids:[...]}
//
// ★ 2026-10-05 「管理端删除做好批量删除」。物理删除（模型无 DeletedAt）。
func (h *EzfyAdmin) AdminEzfyActWildBatchDelete(c *gin.Context) {
	ids := ezfyBatchIDs(c)
	if len(ids) == 0 {
		resp.ParamError(c, "请先勾选要删除的记录")
		return
	}
	res := h.DB.Where("id IN ?", ids).Delete(&model.EzfyActWild{})
	if res.Error != nil {
		resp.ParamError(c, "批量删除失败："+res.Error.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已物理删除 %d 条活动野地配置（勾选 %d 条）", res.RowsAffected, len(ids)),
		"deleted": res.RowsAffected})
}

// AdminEzfyActWildAttacks GET /admin/ezfy-act-wilds/:id/attacks —— 查看该活动野地（坐标）的被攻打记录（分页）
//
// ★ 2026-10-01 活动野地配置页新增「查看被打记录」，模态框展示。
//   攻打历史来自 ezfy_battle（活动野地/活动寇/特殊城市战斗都会 ezfyBattleStart 建行，
//   行不删除，按 target_x / target_y 反查即可）。
func (h *EzfyAdmin) AdminEzfyActWildAttacks(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var a model.EzfyActWild
	if err := h.DB.First(&a, id).Error; err != nil {
		resp.NotFound(c, "该活动野地配置不存在")
		return
	}
	page, offset, size := pageOf(c, 15)
	q := h.DB.Model(&model.EzfyBattle{}).Where("target_x = ? AND target_y = ?", a.X, a.Y)
	var total int64
	q.Count(&total)
	var rows []model.EzfyBattle
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)

	// 攻打者昵称批量取（ezfy_profile.user_id → nickname，找不到用玩家ID兜底）
	uids := make([]uint, 0, len(rows))
	for _, b := range rows {
		uids = append(uids, b.UserID)
	}
	names := map[uint]string{}
	if len(uids) > 0 {
		var ps []model.EzfyProfile
		h.DB.Select("user_id, nickname").Where("user_id IN ?", uids).Find(&ps)
		for _, p := range ps {
			names[p.UserID] = p.Nickname
		}
	}

	// ★ 2026-10-01 补「有没有俘虏军官」：俘虏情况写在战报正文里
	//   （capture 成功 → createCaptiveOfficer 返回「俘虏敌将:XXX(...)」，
	//   已到上限 → 正文写「已被你捕获达到上限，无法再次俘虏」）。
	//   关联方式：优先按 battle.OrderId 精确匹配战报（修复 addReport 的 uint 分支后
	//   新战报 order_id 都正确）；历史战报 order_id 全为 0，兜底按
	//   (user_id + 标题含坐标 + 创建时间区间) 匹配。
	capText := map[uint]string{} // key = battle.ID
	if len(rows) > 0 {
		// ① 精确匹配：order_id > 0 的战斗行
		matched := map[uint]bool{}
		oids := make([]int64, 0, len(rows))
		for _, b := range rows {
			if b.OrderId > 0 {
				oids = append(oids, b.OrderId)
			}
		}
		if len(oids) > 0 {
			var rps []model.EzfyReport
			h.DB.Select("id, order_id, user_id, title, content, created_at").
				Where("order_id IN ?", oids).Order("id DESC").Find(&rps)
			seen := map[int64]bool{}
			for _, r := range rps {
				if seen[r.OrderId] {
					continue
				}
				seen[r.OrderId] = true
				for _, b := range rows {
					if b.OrderId == r.OrderId {
						capText[b.ID] = captiveTextOf(r.Content)
						matched[b.ID] = true
						break
					}
				}
			}
		}
		// ② 兜底：未匹配的战斗行按 (user_id, 标题坐标) 分组，时间两指针归并
		var rest []model.EzfyBattle
		for _, b := range rows {
			if !matched[b.ID] {
				rest = append(rest, b)
			}
		}
		if len(rest) > 0 {
			conds := make([]string, 0, len(rest))
			args := make([]interface{}, 0, len(rest)*2)
			for _, b := range rest {
				conds = append(conds, "(user_id = ? AND title LIKE ?)")
				args = append(args, b.UserID, fmt.Sprintf("%%(%d,%d)%%", b.TargetX, b.TargetY))
			}
			var rps []model.EzfyReport
			h.DB.Select("id, user_id, title, content, created_at").
				Where("report_type = 3 AND ("+strings.Join(conds, " OR ")+")", args...).
				Order("created_at ASC").Find(&rps)
			// 战报按 (user_id, 坐标) 归组；坐标从标题尾部取，如 "(258,100)"
			repByKey := map[string][]model.EzfyReport{}
			for _, r := range rps {
				if k, ok := actWildCoordsKey(r.Title); ok {
					key := fmt.Sprintf("%d:%s", r.UserID, k)
					repByKey[key] = append(repByKey[key], r)
				}
			}
			battleByKey := map[string][]model.EzfyBattle{}
			var keyOrder []string
			for _, b := range rest {
				key := fmt.Sprintf("%d:(%d,%d)", b.UserID, b.TargetX, b.TargetY)
				if _, ok := battleByKey[key]; !ok {
					keyOrder = append(keyOrder, key)
				}
				battleByKey[key] = append(battleByKey[key], b)
			}
			for _, key := range keyOrder {
				bs := battleByKey[key]
				rs := repByKey[key]
				if len(rs) == 0 {
					continue
				}
				// 战斗与战报都按创建时间升序；战报 B 对应战斗 B：
				// 取第一条 created_at 落在 [battle.CreatedAt, 下一场战斗.CreatedAt] 的战报
				sort.Slice(bs, func(i, j int) bool { return bs[i].CreatedAt.Before(bs[j].CreatedAt) })
				sort.Slice(rs, func(i, j int) bool { return rs[i].CreatedAt.Before(rs[j].CreatedAt) })
				i := 0
				for bi := range bs {
					for i < len(rs) && rs[i].CreatedAt.Before(bs[bi].CreatedAt) {
						i++
					}
					var nextStart time.Time
					if bi+1 < len(bs) {
						nextStart = bs[bi+1].CreatedAt
					}
					if i < len(rs) && (nextStart.IsZero() || !rs[i].CreatedAt.After(nextStart)) {
						capText[bs[bi].ID] = captiveTextOf(rs[i].Content)
						i++
					}
				}
			}
		}
		// ③ ★★ 2026-10-28 战报只保留 3 天、运营会定期删（用户明确要求「别用战报关联查询」）→
		//   战报已经取不到的，用**抓取流水**兜底判断「这一场到底有没有抓到」。
		//   匹配条件：同一玩家 + 时间落在战斗开始后 10 分钟内 + 坐标一致（流水无坐标则不校验）。
		//   ★ 只认 kind=1/2（实时记录 / 战报补记，时间可信）；kind=3 军官补记的时间是
		//   军官最后变动时间，跟战斗对不上，参与匹配会误判。
		if a.OfficerId > 0 && len(capText) < len(rows) {
			uids2 := make([]uint, 0, len(rows))
			for _, b := range rows {
				if _, ok := capText[b.ID]; !ok {
					uids2 = append(uids2, b.UserID)
				}
			}
			if len(uids2) > 0 {
				var logs []model.EzfyActCaptureLog
				h.DB.Where("general_id = ? AND kind IN ? AND user_id IN ?", a.OfficerId,
					[]int{model.EzfyActCaptureKindBattle, model.EzfyActCaptureKindReport}, uids2).
					Find(&logs)
				for _, b := range rows {
					if _, ok := capText[b.ID]; ok {
						continue
					}
					for _, l := range logs {
						if l.UserId != b.UserID {
							continue
						}
						dt := l.CreatedAt.Sub(b.CreatedAt)
						if dt < -time.Minute || dt > 10*time.Minute {
							continue
						}
						if l.X != 0 && (l.X != b.TargetX || l.Y != b.TargetY) {
							continue
						}
						capText[b.ID] = "俘虏敌将（据抓取流水）"
						break
					}
				}
			}
		}
	}

	out := make([]gin.H, 0, len(rows))
	for _, b := range rows {
		result := "进行中"
		if b.Status == 2 {
			switch b.Win {
			case 1:
				result = "攻方胜"
			case 2:
				result = "攻方负"
			case 3:
				result = "平局"
			}
		}
		captive := "未俘虏"
		if t := capText[b.ID]; t != "" {
			captive = t
		}
		out = append(out, gin.H{
			"id":          b.ID,
			"user_id":     b.UserID,
			"player_name": names[b.UserID],
			"target_name": b.TargetName,
			"order_id":    b.OrderId,
			"status":      b.Status,
			"win":         b.Win,
			"result":      result,
			"captive":     captive,
			"created_at":  b.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	resp.OK(c, gin.H{"x": a.X, "y": a.Y, "list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyActWildCaptures GET /admin/ezfy-act-wilds/:id/captures —— 查看该活动野地守将的「抓取记录」（分页）
//
// ★ 2026-10-28 用户需求：地图管理 → 活动野地 → 操作里加「抓取记录」。
//
//	数据源 = ezfy_act_capture_log（每成功抓到一次守将记一条流水，只增不删），
//	这也是「可抓次数（max_capture）」的计数依据 —— 配 1 次的玩家在这里有 1 条就再也抓不到，
//	**哪怕那名军官后来被其他玩家抢走**。
//
// ⚠️ 流水按 (玩家, 守将) 存，不按坐标 —— 同一守将配在多个坐标时共用同一份计数，
// 所以这里按 `general_id` 查（含该守将在其它坐标被抓的记录），坐标列展示当时打的是哪一格。
func (h *EzfyAdmin) AdminEzfyActWildCaptures(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var a model.EzfyActWild
	if err := h.DB.First(&a, id).Error; err != nil {
		resp.NotFound(c, "该活动野地配置不存在（可能已被删除），请刷新后重试")
		return
	}
	page, offset, size := pageOf(c, 15)
	// ★ 守将名直接查库（不走 ezfyCfg 全局缓存）：管理端可能比任何玩家请求先到达，
	//   此时配置缓存还没懒加载 → ezfyCfg.general() 返回 nil → 名字空着（本地冷启动实测过）。
	oName := ""
	if a.OfficerId > 0 {
		var g model.EzfyCfgGeneral
		if err := h.DB.Select("id, name").First(&g, a.OfficerId).Error; err == nil {
			oName = g.Name
		}
	}
	base := gin.H{"x": a.X, "y": a.Y, "officer_id": a.OfficerId, "officer_name": oName,
		"max_capture": a.MaxCapture, "page": page, "size": size, "list": []gin.H{}, "total": 0}
	if a.OfficerId <= 0 {
		base["note"] = "该坐标没有配置守将，不会有抓取记录"
		resp.OK(c, base)
		return
	}
	// 表还没建（老库没跑建表）→ 别报 500，给空列表 + 说明
	if !h.DB.Migrator().HasTable(&model.EzfyActCaptureLog{}) {
		base["note"] = "抓取流水表 ezfy_act_capture_log 尚未创建，请重启服务自动补建"
		resp.OK(c, base)
		return
	}
	q := h.DB.Model(&model.EzfyActCaptureLog{}).Where("general_id = ?", a.OfficerId)
	var total int64
	q.Count(&total)
	var rows []model.EzfyActCaptureLog
	// ★ 按「抓取时间」倒序，不是按自增 id —— 补记行的 id 是回填顺序，跟抓取时间不一致
	//   （军官补记的 id 更大但时间更早），按 id 排会让运营看到的时间是乱的。
	q.Order("created_at DESC, id DESC").Offset(offset).Limit(size).Find(&rows)

	// 玩家昵称 / 城市名 批量取（避免 N+1）
	uids := make([]uint, 0, len(rows))
	cids := make([]int64, 0, len(rows))
	seenU := map[uint]bool{}
	seenC := map[int64]bool{}
	for _, r := range rows {
		if !seenU[r.UserId] {
			seenU[r.UserId] = true
			uids = append(uids, r.UserId)
		}
		if r.CityId > 0 && !seenC[r.CityId] {
			seenC[r.CityId] = true
			cids = append(cids, r.CityId)
		}
	}
	names := map[uint]string{}
	if len(uids) > 0 {
		var ps []model.EzfyProfile
		h.DB.Select("user_id, nickname").Where("user_id IN ?", uids).Find(&ps)
		for _, p := range ps {
			names[p.UserID] = p.Nickname
		}
	}
	cityNames := map[int64]string{}
	if len(cids) > 0 {
		var cs []model.EzfyCity
		h.DB.Select("id, name").Where("id IN ?", cids).Find(&cs)
		for _, cc := range cs {
			cityNames[int64(cc.ID)] = cc.Name
		}
	}

	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		kindName := "战斗抓取"
		switch r.Kind {
		case model.EzfyActCaptureKindReport:
			kindName = "战报补记"
		case model.EzfyActCaptureKindOfficer:
			kindName = "军官补记"
		}
		out = append(out, gin.H{
			"id":          r.ID,
			"user_id":     r.UserId,
			"player_name": names[r.UserId],
			"general_id":  r.GeneralId,
			"general_name": oName,
			"x":           r.X,
			"y":           r.Y,
			"city_id":     r.CityId,
			"city_name":   cityNames[r.CityId],
			"kind":        r.Kind,
			"kind_name":   kindName,
			"created_at":  r.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	base["list"] = out
	base["total"] = total
	resp.OK(c, base)
}

// captiveTextOf 从战报正文提取活动野地守将的俘虏情况
//
//	成功 → 原样带回「俘虏敌将:XXX(星级, 忠诚30) 可前往军校收编」行
//	已达上限 → 「已达捕获上限（无法再次俘虏）」
//	其他（战败/平局/概率未中/无参谋部）→ 空字符串（前端显示「未俘虏」）
func captiveTextOf(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "俘虏敌将:") {
			return line
		}
	}
	if strings.Contains(content, "无法再次俘虏") {
		return "已达捕获上限（无法再次俘虏）"
	}
	return ""
}

// actWildCoordsKey 从战报标题尾部提取坐标串（如 "(258,100)"），用于历史战报兜底匹配
func actWildCoordsKey(title string) (string, bool) {
	i := strings.LastIndex(title, "(")
	if i < 0 || !strings.HasSuffix(title, ")") {
		return "", false
	}
	return title[i:], true
}

// ezfyAnyInt 从 JSON 解析出来的任意值里取整数（前端数字是 float64，兼容字符串/整数）。
// 用于读取不在白名单里的字段（如 id）。
func ezfyAnyInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(n))
		return i
	}
	return 0
}

// checkActWildVals 校验活动野地配置值
func checkActWildVals(vals map[string]interface{}) string {
	if v, ok := vals["enabled"].(int); ok && v != 0 && v != 1 {
		return "启用开关只能是 0（关闭）/ 1（启用）"
	}
	if v, ok := vals["level"].(int); ok && v != 0 && (v < 1 || v > 3) {
		return "活动等级只能是 1~3（0 = 用默认）"
	}
	if v, ok := vals["res"].(int64); ok && v < 0 {
		return "资源奖励不能为负"
	}
	if v, ok := vals["gold"].(int64); ok && v < 0 {
		return "黄金奖励不能为负"
	}
	if v, ok := vals["prestige"].(int); ok && v < 0 {
		return "声望奖励不能为负"
	}
	// 守军 JSON 校验（范围格式 [[兵种id,最小,最大]]，兼容旧的 [[兵种id,数量]]）
	if s, ok := vals["troops"].(string); ok && strings.TrimSpace(s) != "" {
		if parseActWildTroops(s) == nil {
			return "守军配置格式不对，应为 [[兵种id,数量]] 或 [[兵种id,最小,最大]]"
		}
	}
	// 必掉宝物 JSON 校验 [[cfg_id,count],...]
	if s, ok := vals["treasures"].(string); ok && strings.TrimSpace(s) != "" {
		if parseActWildTreasures(s) == nil {
			return "宝物配置格式不对，应为 [[宝物id,数量],...]"
		}
	}
	// 守将被俘虏概率 0~100
	if v, ok := vals["capture_rate"].(int); ok && (v < 0 || v > 100) {
		return "被俘虏概率只能是 0~100（0 = 不俘虏，100 = 必俘虏）"
	}
	// 同一玩家可抓次数：0 = 不限；否则 >= 1
	if v, ok := vals["max_capture"].(int); ok && v < 0 {
		return "可抓次数不能为负（0 = 不限，>=1 为上限）"
	}
	// ★ 2026-10-25 兜底被俘次数不能为负（0 = 第一次打赢就按概率）
	if v, ok := vals["min_capture_wins"].(int); ok && v < 0 {
		return "兜底被俘次数不能为负（0 = 第一次打赢就按概率）"
	}
	// ★ 2026-10-24 守军行为（按兵种）：JSON map {"兵种ID":方向}，方向只能 1/0/-1
	if s, ok := vals["def_move_dir"].(string); ok && strings.TrimSpace(s) != "" && strings.TrimSpace(s) != "{}" {
		m := map[int]int{}
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			return "守军行为格式不对，应为 {\"兵种ID\":1前进/0停止/-1后退}"
		}
		for id, dir := range m {
			if dir != 0 && dir != 1 && dir != -1 {
				return fmt.Sprintf("兵种 %d 的守军行为只能是 1（前进）/ 0（停止）/ -1（后退）", id)
			}
		}
	}
	// 守将等级 / 被俘等级不能为负（0 = 用军官池默认）
	for _, k := range []string{"def_level", "captive_level"} {
		if v, ok := vals[k].(int); ok && v < 0 {
			return k + "不能为负（0 = 用军官池默认）"
		}
	}
	// 守将军官必须来自军官池（普通军官/名将都可选）
	if v, ok := vals["officer_id"].(int); ok && v > 0 {
		if ezfyCfg.general(v) == nil {
			return "守将军官不在军官池中，请重新选择"
		}
	}
	return ""
}
