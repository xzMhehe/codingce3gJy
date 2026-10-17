// 管理端 · 统帅称号模块（★ 2026-10-17）
//
// 用户需求（原话）：「二战管理端加个模块 统帅称号，称号管理员可添加、称号名称、称号类型、
// 称号有效期、称号创建日期、已有称号玩家」。
//
// 管理端只管**数据本身**：
//
//	· 称号定义（ezfy_title）：名称 / 类型 / 有效期(天) / 创建日期 / 启用状态；
//	· 授予玩家（ezfy_title_grant）：手动发放、查看「已有称号玩家」、撤销；
//	· 玩家端展示与风云榜每日第一自动发放见 `ezfy_title.go`。
//
// 权限：module:ezfyTitles（菜单「统帅称号」）。
package ezfy

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// AdminEzfyTitles GET /admin/ezfy-titles —— 称号列表（带持有人数）
//
// 参数：word（称号名称模糊，可选）。
//
// ★★ 2026-10-17 卡顿优化（用户反馈「统帅称号 查询卡顿」）：
//
//	原实现是 **3 次跨 WAN 往返** —— COUNT(*) → 分页 SELECT → 持有人数聚合
//	（线上二战库是远端 RDS，单次 20~40ms，点菜单/翻页固定 100ms+）。
//	但称号是运营**手动创建**的，全服也就几条到几十条，分页本身就是多余的：
//	现在**一次 SQL 全量返回**（LEFT JOIN 预聚合持有人数，上限 500 条防意外），
//	前端本地做搜索 + 分页 → 「查询」不再发请求，翻页零延迟。
func (h *EzfyAdmin) AdminEzfyTitles(c *gin.Context) {
	word := strings.TrimSpace(c.Query("word"))
	now := time.Now().UnixMilli()

	type titleRow struct {
		ID           uint      `gorm:"column:id"`
		Name         string    `gorm:"column:name"`
		Type         int       `gorm:"column:type"`
		ValidDays    int       `gorm:"column:valid_days"`
		Remark       string    `gorm:"column:remark"`
		Status       int       `gorm:"column:status"`
		CreatedAt    time.Time `gorm:"column:created_at"`
		HolderTotal  int64     `gorm:"column:holder_total"`
		HolderActive int64     `gorm:"column:holder_active"`
	}
	var rows []titleRow
	// 持有人数用**预聚合子查询** LEFT JOIN：一次往返拿全，不再逐称号查一次。
	// 子查询只扫 ezfy_title_grant 的 (title_id, expire_at) 覆盖索引（见 EnsureEzfyIndexes）。
	q := h.DB.Table("ezfy_title AS t").
		Select("t.id AS id, t.name AS name, t.type AS type, t.valid_days AS valid_days, "+
			"t.remark AS remark, t.status AS status, t.created_at AS created_at, "+
			"COALESCE(g.total, 0) AS holder_total, COALESCE(g.active, 0) AS holder_active").
		Joins("LEFT JOIN (SELECT title_id, COUNT(*) AS total, "+
			"SUM(CASE WHEN expire_at = 0 OR expire_at > ? THEN 1 ELSE 0 END) AS active "+
			"FROM ezfy_title_grant GROUP BY title_id) AS g ON g.title_id = t.id", now)
	if word != "" {
		q = q.Where("t.name LIKE ?", "%"+word+"%")
	}
	q.Order("t.id DESC").Limit(500).Scan(&rows)

	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		out = append(out, gin.H{
			"id": r.ID, "name": r.Name,
			"type": r.Type, "type_name": ezfyTitleTypeName(r.Type),
			"valid_days":    r.ValidDays,
			"valid_text":    ezfyTitleValidText(r.ValidDays),
			"remark":        r.Remark,
			"status":        r.Status,
			"created_at":    r.CreatedAt.Format("2006-01-02 15:04:05"),
			"holder_total":  r.HolderTotal,
			"holder_active": r.HolderActive,
		})
	}
	// page/size 保留字段（老前端兼容），但已不再分页：一次给全，前端本地切页
	resp.OK(c, gin.H{"list": out, "total": len(out), "page": 1, "size": len(out), "types": ezfyTitleTypes()})
}

// ezfyTitleValidText 有效期展示文案（0 = 永久）。
func ezfyTitleValidText(days int) string {
	if days <= 0 {
		return "永久"
	}
	return strconv.Itoa(days) + " 天"
}

// AdminEzfyTitleSave POST /admin/ezfy-titles —— 新增 / 编辑称号
//
// body: {id?, name, type, valid_days, remark, status}
// id 为空或 0 = 新增；否则更新（名称唯一，重名直接报错）。
func (h *EzfyAdmin) AdminEzfyTitleSave(c *gin.Context) {
	var in struct {
		ID        uint   `json:"id"`
		Name      string `json:"name"`
		Type      int    `json:"type"`
		ValidDays int    `json:"valid_days"`
		Remark    string `json:"remark"`
		Status    int    `json:"status"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		resp.ParamError(c, "请填写称号名称")
		return
	}
	if r := []rune(name); len(r) > 12 {
		resp.ParamError(c, "称号名称最多 12 个字")
		return
	}
	if in.Type < ezfyTitleTypeMerit || in.Type > ezfyTitleTypeHonor {
		resp.ParamError(c, "称号类型不正确")
		return
	}
	if in.ValidDays < 0 || in.ValidDays > 3650 {
		resp.ParamError(c, "有效期请填 0~3650 天（0 = 永久）")
		return
	}
	if in.Status != 0 {
		in.Status = 1
	}
	// 名称唯一（编辑时排除自己）
	var dup int64
	{
		q := h.DB.Model(&model.EzfyTitle{}).Where("name = ?", name)
		if in.ID > 0 {
			q = q.Where("id <> ?", in.ID)
		}
		q.Count(&dup)
	}
	if dup > 0 {
		resp.ParamError(c, "已存在同名称号「"+name+"」")
		return
	}

	remark := strings.TrimSpace(in.Remark)
	if r := []rune(remark); len(r) > 100 {
		remark = string(r[:100])
	}
	if in.ID > 0 {
		var t model.EzfyTitle
		if err := h.DB.First(&t, in.ID).Error; err != nil {
			resp.NotFound(c, "称号不存在")
			return
		}
		if err := h.DB.Model(&model.EzfyTitle{}).Where("id = ?", in.ID).Updates(map[string]interface{}{
			"name": name, "type": in.Type, "valid_days": in.ValidDays,
			"remark": remark, "status": in.Status,
		}).Error; err != nil {
			resp.ServerError(c, err)
			return
		}
		resp.OK(c, gin.H{"msg": "称号「" + name + "」已更新", "id": in.ID})
		return
	}
	t := model.EzfyTitle{
		Name: name, Type: in.Type, ValidDays: in.ValidDays,
		Remark: remark, Status: in.Status,
	}
	if err := h.DB.Create(&t).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	resp.OK(c, gin.H{"msg": "称号「" + name + "」已创建", "id": t.ID})
}

// AdminEzfyTitleDelete DELETE /admin/ezfy-titles/:id —— 删除称号（连带清掉该称号的全部持有记录）
//
// ★ 单事务：先删授予记录再删定义，避免留下指向不存在称号的孤儿行。
func (h *EzfyAdmin) AdminEzfyTitleDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var t model.EzfyTitle
	if err := h.DB.First(&t, id).Error; err != nil {
		resp.NotFound(c, "称号不存在")
		return
	}
	removed := int64(0)
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 先数（删完再数恒为 0），再删授予记录，最后删定义
		tx.Model(&model.EzfyTitleGrant{}).Where("title_id = ?", id).Count(&removed)
		if err := tx.Exec("DELETE FROM ezfy_title_grant WHERE title_id = ?", id).Error; err != nil {
			return err
		}
		return tx.Exec("DELETE FROM ezfy_title WHERE id = ?", id).Error
	})
	if err != nil {
		resp.ServerError(c, err)
		return
	}
	resp.OK(c, gin.H{"msg": "已删除称号「" + t.Name + "」", "removed": removed})
}

// AdminEzfyTitlePlayers GET /admin/ezfy-titles/:id/players —— 该称号的「已有称号玩家」
//
// 参数：page / size / word（昵称 / 用户ID / 游戏号）/ only（1=只看生效中）。
func (h *EzfyAdmin) AdminEzfyTitlePlayers(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var t model.EzfyTitle
	if err := h.DB.First(&t, id).Error; err != nil {
		resp.NotFound(c, "称号不存在")
		return
	}
	page, offset, size := pageOf(c, 10)
	word := strings.TrimSpace(c.Query("word"))
	only := strings.TrimSpace(c.DefaultQuery("only", "0")) == "1"
	now := time.Now().UnixMilli()

	q := h.DB.Model(&model.EzfyTitleGrant{}).Where("title_id = ?", id)
	if only {
		// ★ 2026-10-17 必须自己加括号：GORM 串联多个 Where 只按 AND 拼、**不会**给字符串条件补括号，
		//   写成 `expire_at = 0 OR expire_at > ?` 会变成 `title_id=? AND expire_at=0 OR ...`
		//   → 串味（把别的称号的未过期记录也带出来）+ OR 让索引失效全表扫。
		q = q.Where("(expire_at = 0 OR expire_at > ?)", now)
	}
	// ★ 搜索条件放在**一个 Where 内的括号**里：前面已有 title_id / only 条件，
	//   用 `q.Or(...)` 会被 SQL 的 AND 优先于 OR 串味（免战管理踩过同款坑）。
	//
	// ★★ 2026-10-17 卡顿优化：原来是 `user_id IN (SELECT ... FROM ezfy_profile WHERE ...)` 子查询，
	//   MySQL 会**对 grant 表的每一行**求值一次子查询（半连接也不一定物化），昵称 LIKE 更是全表扫。
	//   改成「先查 profile 拿 uid 列表 → 再 IN」：两次都走主键/索引，且昵称无匹配时直接返回空。
	if word != "" {
		if uid, err := strconv.Atoi(word); err == nil {
			ids := []uint{uint(uid)}
			// 数字既可能是用户ID也可能是游戏号（game_uid），两个都匹配
			var gids []uint
			h.DB.Model(&model.EzfyProfile{}).Where("game_uid = ?", uid).Limit(20).Pluck("user_id", &gids)
			ids = append(ids, gids...)
			q = q.Where("user_id IN ?", ids)
		} else {
			var uids []uint
			h.DB.Model(&model.EzfyProfile{}).Where("nickname LIKE ?", "%"+word+"%").Limit(500).Pluck("user_id", &uids)
			if len(uids) == 0 {
				resp.OK(c, gin.H{"list": []gin.H{}, "total": 0, "page": 1, "size": size,
					"title": gin.H{"id": t.ID, "name": t.Name, "type": t.Type, "type_name": ezfyTitleTypeName(t.Type),
						"valid_days": t.ValidDays, "valid_text": ezfyTitleValidText(t.ValidDays)}})
				return
			}
			q = q.Where("user_id IN ?", uids)
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyTitleGrant
	q.Order("id DESC").Offset(offset).Limit(size).Find(&rows)

	uids := make([]uint, 0, len(rows))
	for _, r := range rows {
		uids = append(uids, r.UserID)
	}
	// 游戏号 / 昵称（二战档案）批量取：昵称以**当前档案**为准，快照只做兜底
	// （玩家改名后后台仍显示旧名会让管理员对不上人）
	gameUID := map[uint]int64{}
	nickOf := map[uint]string{}
	if len(uids) > 0 {
		var profs []model.EzfyProfile
		h.DB.Select("user_id, game_uid, nickname").Where("user_id IN ?", uids).Find(&profs)
		for _, p := range profs {
			gameUID[p.UserID] = p.GameUID
			if p.Nickname != "" {
				nickOf[p.UserID] = p.Nickname
			}
		}
	}

	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		expired := r.ExpireAt > 0 && r.ExpireAt <= now
		expireText := "永久"
		if r.ExpireAt > 0 {
			expireText = time.UnixMilli(r.ExpireAt).Format("2006-01-02 15:04:05")
		}
		src := "管理端授予"
		if r.Source == "pvp_daily" {
			src = "风云榜每日第一"
		}
		nick := nickOf[r.UserID]
		if nick == "" {
			nick = r.Nickname
		}
		out = append(out, gin.H{
			"id": r.ID, "user_id": r.UserID, "nickname": nick,
			"game_uid": gameUID[r.UserID], "source": r.Source, "source_name": src,
			"created_at": r.CreatedAt.Format("2006-01-02 15:04:05"),
			"expire_at":  r.ExpireAt, "expire_text": expireText,
			"permanent": r.ExpireAt == 0, "expired": expired,
		})
	}
	resp.OK(c, gin.H{
		"list": out, "total": total, "page": page, "size": size,
		"title": gin.H{"id": t.ID, "name": t.Name, "type": t.Type, "type_name": ezfyTitleTypeName(t.Type),
			"valid_days": t.ValidDays, "valid_text": ezfyTitleValidText(t.ValidDays)},
	})
}

// AdminEzfyTitleGrant POST /admin/ezfy-titles/:id/grant —— 手动授予玩家
//
// body: {user_id, days?}；user_id 同时接受**用户ID**与**游戏号**（管理端统一口径），
// days 不传则用称号自身的有效期（0 = 永久）。
func (h *EzfyAdmin) AdminEzfyTitleGrant(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var t model.EzfyTitle
	if err := h.DB.First(&t, id).Error; err != nil {
		resp.NotFound(c, "称号不存在")
		return
	}
	var in struct {
		UserID uint `json:"user_id"`
		Days   *int `json:"days"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.UserID == 0 {
		resp.ParamError(c, "请填写玩家ID或游戏号")
		return
	}
	// 定位玩家：先按 user_id（家园号），再按 game_uid
	uid := in.UserID
	var prof model.EzfyProfile
	if err := h.DB.Where("user_id = ?", uid).First(&prof).Error; err != nil {
		if e2 := h.DB.Where("game_uid = ?", int64(in.UserID)).First(&prof).Error; e2 == nil {
			uid = prof.UserID
		} else {
			resp.NotFound(c, "玩家不存在")
			return
		}
	}
	days := t.ValidDays
	if in.Days != nil {
		if *in.Days < 0 || *in.Days > 3650 {
			resp.ParamError(c, "有效期请填 0~3650 天（0 = 永久）")
			return
		}
		days = *in.Days
	}
	expireAt := ezfyTitleExpireAt(days, time.Now())
	nick := ezfyNickByUid(h.DB, h.HomeDB, uid)
	if err := ezfyGrantTitle(h.DB, t.ID, uid, nick, "admin", expireAt); err != nil {
		resp.ServerError(c, err)
		return
	}
	msg := "已授予「" + nick + "」称号「" + t.Name + "」"
	if expireAt == 0 {
		msg += "（永久）"
	} else {
		msg += "（到期 " + time.UnixMilli(expireAt).Format("2006-01-02 15:04") + "）"
	}
	resp.OK(c, gin.H{"msg": msg, "expire_at": expireAt})
}

// AdminEzfyTitleRevoke DELETE /admin/ezfy-title-grants/:gid —— 撤销某条授予记录
//
// ★ 物理删除：撤销 = 玩家立刻失去该称号（软删除会让「生效中」口径还得再排一次 deleted_at）。
func (h *EzfyAdmin) AdminEzfyTitleRevoke(c *gin.Context) {
	gid, _ := strconv.Atoi(c.Param("gid"))
	if gid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var g model.EzfyTitleGrant
	if err := h.DB.First(&g, gid).Error; err != nil {
		resp.NotFound(c, "授予记录不存在")
		return
	}
	var t model.EzfyTitle
	h.DB.First(&t, g.TitleId)
	if err := h.DB.Exec("DELETE FROM ezfy_title_grant WHERE id = ?", gid).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	resp.OK(c, gin.H{"msg": "已撤销「" + g.Nickname + "」的称号「" + t.Name + "」"})
}
