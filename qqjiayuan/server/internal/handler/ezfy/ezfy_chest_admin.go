// 二战风云 管理端「发放宝箱」：给玩家发待开宝箱（玩家在邮箱点「领取」才入 ezfy_user_chest）
//
// ★ 2026-10-14 用户需求「宝箱也能发放给玩家加个功能」：在「套装装备配置（宝箱）」页发放。
// ★ 2026-10-24 用户需求「发宝箱也是 发到邮箱吧 然后 7 天内有效」：
//   与「发放道具」同口径，发放 = 系统邮件 + 附件（宝箱），支持单人 / 全体玩家，7 天过期作废。
package ezfy

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// AdminEzfyChestGrantOptions GET /admin/ezfy-chest-grant/options —— 发放宝箱弹窗的宝箱下拉选项。
func (h *EzfyAdmin) AdminEzfyChestGrantOptions(c *gin.Context) {
	var chests []model.EzfyCfgChest
	h.DB.Order("id ASC").Find(&chests)
	list := make([]gin.H, 0, len(chests))
	for _, ct := range chests {
		list = append(list, gin.H{"id": ct.ID, "name": ct.Name})
	}
	resp.OK(c, gin.H{"list": list, "total": len(list)})
}

// AdminEzfyChestGrant POST /admin/ezfy-chest-grant {player, items:[{chest_id, count}]}
//
// ★ 2026-10-24 与「发放道具」同口径：**改走系统邮件 + 附件**，不再直接写待开宝箱表。
//   玩家在邮箱点「领取」才到账（领取时按附件入 ezfy_user_chest），7 天不领过期作废。
// ⚠️ 不校验数量上限（运营发放本就该给多少算多少）。
func (h *EzfyAdmin) AdminEzfyChestGrant(c *gin.Context) {
	var in struct {
		Player string `json:"player"`
		Items  []struct {
			ChestID int `json:"chest_id"`
			Count   int `json:"count"`
		} `json:"items"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	player := strings.TrimSpace(in.Player)
	if player == "" {
		resp.ParamError(c, "请填写玩家昵称或游戏ID")
		return
	}
	if len(in.Items) == 0 {
		resp.ParamError(c, "请添加要发放的宝箱")
		return
	}
	var p model.EzfyProfile
	if uid, err := strconv.Atoi(player); err == nil {
		if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err != nil {
			resp.NotFound(c, "未找到该游戏ID对应的玩家，请先搜索确认")
			return
		}
	} else {
		if err := h.DB.Where("nickname = ?", player).First(&p).Error; err != nil {
			resp.NotFound(c, "未找到该昵称对应的玩家，请先搜索确认")
			return
		}
	}
	names := ""
	attaches := make([]model.EzfyMailAttach, 0, len(in.Items))
	for _, it := range in.Items {
		if it.ChestID <= 0 || it.Count <= 0 {
			continue
		}
		var chest model.EzfyCfgChest
		if err := h.DB.First(&chest, it.ChestID).Error; err != nil {
			resp.ParamError(c, "宝箱不存在："+strconv.Itoa(it.ChestID))
			return
		}
		attaches = append(attaches, model.EzfyMailAttach{
			Kind: ezfyAttachChest, CfgID: it.ChestID, Num: int64(it.Count), Name: chest.Name,
		})
		names += fmt.Sprintf(" 【%s】×%d", chest.Name, it.Count)
	}
	if names == "" || len(attaches) == 0 {
		resp.ParamError(c, "请添加要发放的宝箱")
		return
	}
	content := "管理员发放宝箱：" + strings.TrimSpace(names) +
		fmt.Sprintf("，请在 %d 天内到邮箱领取，过期作废。", ezfyMailAttachExpireDays)
	if _, err := ezfyMailSendBatch(h.DB, []ezfyMailDraft{{
		ReceiverID: p.UserID, Subject: "管理员发放宝箱", Content: content, Attaches: attaches,
	}}, ezfyMailAttachExpireDays); err != nil {
		resp.ServerError(c, fmt.Errorf("发放失败：%w", err))
		return
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已通过系统邮件发放给玩家「%s」(ID:%d)%s，玩家需在邮箱领取",
		p.Nickname, p.UserID, names)})
}

// AdminEzfyChestGrantTargets GET /admin/ezfy-chest-grant/targets —— 全员发放的「目标人数」预览。
//
// 与 AdminEzfyItemGrantTargets 同款：权限点挂在 module:ezfyData 下，
// 复用 /admin/ezfy-players（module:ezfyPlayers）会对只有数据管理权限的管理员 403。
func (h *EzfyAdmin) AdminEzfyChestGrantTargets(c *gin.Context) {
	var n int64
	h.DB.Model(&model.EzfyProfile{}).Count(&n)
	resp.OK(c, gin.H{"total": n})
}

// AdminEzfyChestGrantAll POST /admin/ezfy-chest-grant/all {items[]} —— 「发放宝箱」的**全员发放**。
//
// ★ 2026-10-24 用户需求「发放宝箱也要支持 全体玩家 发放 参考道具」。
//   发放对象 = 全部二战玩家档案（ezfy_profile 的所有 user_id），每人一封系统邮件 + 同一份宝箱附件，
//   7 天不领过期作废（与「发放道具」全员发放完全同口径）。
func (h *EzfyAdmin) AdminEzfyChestGrantAll(c *gin.Context) {
	var in struct {
		Items []struct {
			ChestID int `json:"chest_id"`
			Count   int `json:"count"`
		} `json:"items"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if len(in.Items) == 0 {
		resp.ParamError(c, "请添加要发放的宝箱")
		return
	}
	var uids []uint
	if err := h.DB.Model(&model.EzfyProfile{}).Pluck("user_id", &uids).Error; err != nil {
		resp.ServerError(c, fmt.Errorf("读取玩家列表失败：%w", err))
		return
	}
	if len(uids) == 0 {
		resp.NotFound(c, "还没有任何二战玩家档案，无法全员发放")
		return
	}
	names := ""
	attaches := make([]model.EzfyMailAttach, 0, len(in.Items))
	for _, it := range in.Items {
		if it.ChestID <= 0 || it.Count <= 0 {
			continue
		}
		var chest model.EzfyCfgChest
		if err := h.DB.First(&chest, it.ChestID).Error; err != nil {
			resp.ParamError(c, "宝箱不存在："+strconv.Itoa(it.ChestID))
			return
		}
		attaches = append(attaches, model.EzfyMailAttach{
			Kind: ezfyAttachChest, CfgID: it.ChestID, Num: int64(it.Count), Name: chest.Name,
		})
		names += fmt.Sprintf(" 【%s】×%d", chest.Name, it.Count)
	}
	if names == "" || len(attaches) == 0 {
		resp.ParamError(c, "请添加要发放的宝箱")
		return
	}
	content := "管理员发放宝箱：" + strings.TrimSpace(names) +
		fmt.Sprintf("，请在 %d 天内到邮箱领取，过期作废。", ezfyMailAttachExpireDays)
	drafts := make([]ezfyMailDraft, 0, len(uids))
	for _, uid := range uids {
		drafts = append(drafts, ezfyMailDraft{
			ReceiverID: uid, Subject: "管理员发放宝箱", Content: content, Attaches: attaches,
		})
	}
	if _, err := ezfyMailSendBatch(h.DB, drafts, ezfyMailAttachExpireDays); err != nil {
		resp.ServerError(c, fmt.Errorf("发放失败：%w", err))
		return
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已给全部 %d 名玩家发送系统邮件%s，玩家需在邮箱领取", len(uids), names), "players": len(uids)})
}

// AdminEzfyPlayerChestOptions GET /admin/ezfy-player-chests/options —— 玩家详情「添加宝箱」下拉选项。
//
// ★ 2026-10-18 挂 module:ezfyPlayers 权限：只有「玩家信息管理」权限、没有「数据管理」权限的管理员
// 也能在玩家详情里加宝箱（口径同 AdminEzfyPlayerItemOptions 复用道具下拉）。
func (h *EzfyAdmin) AdminEzfyPlayerChestOptions(c *gin.Context) {
	h.AdminEzfyChestGrantOptions(c)
}

// AdminEzfyPlayerChestSet POST /admin/ezfy-players/:id/chest —— 添加宝箱 / 编辑待开数量
//
// ★ 2026-10-18 用户需求「玩家详情 背包 也要能看到玩家的宝箱个数 并且能编辑」。
// 玩家详情 → 背包 tab 里的待开宝箱维护，口径与背包道具维护（ezfy_admin_bag.go）一致：
//
//	· mode = "add"：累加进待开宝箱（已有则 count + N，与发放宝箱同款 upsert）；
//	· mode = "set"：把该宝箱待开数量**直接设为**指定值（覆盖，不累加）；
//	· set 到 0 = 等价于删除该行（不残留 count=0 僵尸行）。
//
// 不做数量上限校验（运营修正数据以管理员为准），设一个 10 亿硬上限防手误。
func (h *EzfyAdmin) AdminEzfyPlayerChestSet(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	if uid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var in struct {
		ChestID int    `json:"chest_id"`
		Count   int    `json:"count"`
		Mode    string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.ChestID <= 0 {
		resp.ParamError(c, "请选择宝箱")
		return
	}
	if in.Count < 0 || in.Count > 1000000000 {
		resp.ParamError(c, "数量请填 0~1000000000")
		return
	}
	var chest model.EzfyCfgChest
	if err := h.DB.First(&chest, in.ChestID).Error; err != nil {
		resp.ParamError(c, "宝箱不存在："+strconv.Itoa(in.ChestID))
		return
	}
	nick := ezfyNickByUid(h.DB, h.HomeDB, uint(uid))
	mode := strings.TrimSpace(in.Mode)

	// set 到 0 = 删除（避免待开宝箱留 count=0 僵尸行）
	if mode == "set" && in.Count == 0 {
		if err := h.DB.Exec("DELETE FROM ezfy_user_chest WHERE user_id = ? AND chest_id = ?", uid, in.ChestID).Error; err != nil {
			resp.ServerError(c, err)
			return
		}
		resp.OK(c, gin.H{"msg": fmt.Sprintf("已删除「%s」的【%s】", nick, chest.Name), "count": 0})
		return
	}
	if in.Count <= 0 {
		resp.ParamError(c, "数量必须大于 0（要清空请用删除）")
		return
	}
	// 原子 upsert：add=累加 / set=覆盖（唯一键 user_id+chest_id，同发放宝箱）
	expr := "count = count + ?"
	if mode != "add" {
		expr = "count = ?"
	}
	if err := h.DB.Exec(`INSERT INTO ezfy_user_chest (user_id, chest_id, count) VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE `+expr+`, updated_at = NOW()`, uid, in.ChestID, in.Count, in.Count).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	var now model.EzfyUserChest
	h.DB.Where("user_id = ? AND chest_id = ?", uid, in.ChestID).First(&now)
	msg := fmt.Sprintf("已把「%s」的【%s】待开数量设为 %d", nick, chest.Name, now.Count)
	if mode == "add" {
		msg = fmt.Sprintf("已给「%s」添加【%s】×%d（现有 %d）", nick, chest.Name, in.Count, now.Count)
	}
	ezfyChestCacheDel(uint(uid))
	resp.OK(c, gin.H{"msg": msg, "count": now.Count, "chest_id": in.ChestID, "chest_name": chest.Name})
}

// AdminEzfyPlayerChestDelete DELETE /admin/ezfy-players/:id/chest/:chest_id —— 物理删除玩家待开宝箱
//
// 只删这一行（user_id + chest_id），玩家其余宝箱/道具不受影响；不可恢复。
func (h *EzfyAdmin) AdminEzfyPlayerChestDelete(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	chestID, _ := strconv.Atoi(c.Param("chest_id"))
	if uid <= 0 || chestID <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var row model.EzfyUserChest
	if err := h.DB.Where("user_id = ? AND chest_id = ?", uid, chestID).First(&row).Error; err != nil {
		resp.NotFound(c, "该玩家没有这个待开宝箱")
		return
	}
	name := "宝箱#" + strconv.Itoa(chestID)
	var chest model.EzfyCfgChest
	if err := h.DB.First(&chest, chestID).Error; err == nil {
		name = chest.Name
	}
	nick := ezfyNickByUid(h.DB, h.HomeDB, uint(uid))
	if err := h.DB.Exec("DELETE FROM ezfy_user_chest WHERE user_id = ? AND chest_id = ?", uid, chestID).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	ezfyChestCacheDel(uint(uid))
	resp.OK(c, gin.H{
		"msg": fmt.Sprintf("已物理删除「%s」的【%s】×%d", nick, name, row.Count),
	})
}