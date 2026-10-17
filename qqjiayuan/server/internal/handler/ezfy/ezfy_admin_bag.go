// 管理端 · 玩家背包维护（★ 2026-10-17）
//
// 用户需求（原话）：「玩家道具好像没有维护得地方吧 → 管理系统 玩家信息管理 优化功能，
// 能编辑玩家背包、编辑数量、物理删除玩家道具」。
//
// 玩家信息管理 → 玩家详情 → 背包 tab 里直接维护（权限 module:ezfyPlayers）：
//
//	· 添加道具：选道具 + 数量 → **累加**进背包（已有则 count + N）；
//	· 编辑数量：把该道具**设为**指定数量（覆盖，不累加）；
//	· 物理删除：DELETE 该 (user_id,cfg_id) 行 —— ezfy_item 本来就是物理表，删了就是没有。
//
// ★ 与玩家端 addItem 的口径差异：管理端**不受** `ezfy_cfg_limit.bag_item_max` 单件上限约束
//
//	（修正数据以管理员为准，与「发放道具」ezfyGrantItemToPlayers 的无视上限口径一致）。
//	但设一个 10 亿的硬上限，防手误填天文数字把 int 撑爆 / 背包行变得无法阅读。
//
// ★ 一律走**单条 upsert / DELETE**，不「先读快照 → 校验 → 写回」：
//
//	玩家可能同时在游戏里用道具，读-改-写会把这期间的消耗覆盖掉。
package ezfy

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// ezfyAdminBagMaxCount 管理端可设置的背包单件数量硬上限（防手误，非玩法上限）。
const ezfyAdminBagMaxCount = 1000000000

// AdminEzfyPlayerItemSet POST /admin/ezfy-players/:id/item —— 添加道具 / 编辑数量
//
// body: {cfg_id, count, mode}
//
//	mode = "add"（累加，添加道具用）| "set"（直接设为，编辑数量用；不传按 set）
//	count = 0 且 mode=set → 等价于物理删除（前端主要走 DELETE 接口）
func (h *EzfyAdmin) AdminEzfyPlayerItemSet(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	if uid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var in struct {
		CfgID int    `json:"cfg_id"`
		Count int    `json:"count"`
		Mode  string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.CfgID <= 0 {
		resp.ParamError(c, "请选择道具")
		return
	}
	if in.Count < 0 || in.Count > ezfyAdminBagMaxCount {
		resp.ParamError(c, fmt.Sprintf("数量请填 0~%d", ezfyAdminBagMaxCount))
		return
	}
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	h.cfgs()
	var cfg model.EzfyCfgItem
	if err := h.DB.First(&cfg, in.CfgID).Error; err != nil {
		resp.ParamError(c, "道具不存在："+strconv.Itoa(in.CfgID))
		return
	}
	nick := ezfyNickByUid(h.DB, h.HomeDB, uint(uid))
	mode := strings.TrimSpace(in.Mode)

	// set 到 0 = 删除（避免背包里留一堆 count=0 的僵尸行）
	if mode == "set" && in.Count == 0 {
		if err := h.DB.Exec("DELETE FROM ezfy_item WHERE user_id = ? AND cfg_id = ?", uid, in.CfgID).Error; err != nil {
			resp.ServerError(c, err)
			return
		}
		resp.OK(c, gin.H{"msg": fmt.Sprintf("已删除「%s」的【%s】", nick, cfg.Name), "count": 0})
		return
	}
	if mode == "add" {
		if in.Count <= 0 {
			resp.ParamError(c, "添加数量必须大于 0")
			return
		}
		// 原子累加：已有行直接 count+N，没有则插入（唯一索引 uk_user_item）
		if err := h.DB.Exec(`INSERT INTO ezfy_item (user_id, cfg_id, count) VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE count = count + ?`, uid, in.CfgID, in.Count, in.Count).Error; err != nil {
			resp.ServerError(c, err)
			return
		}
	} else {
		if in.Count <= 0 {
			resp.ParamError(c, "数量必须大于 0（要清空请用删除）")
			return
		}
		// 原子覆盖
		if err := h.DB.Exec(`INSERT INTO ezfy_item (user_id, cfg_id, count) VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE count = ?`, uid, in.CfgID, in.Count, in.Count).Error; err != nil {
			resp.ServerError(c, err)
			return
		}
	}
	// 回读最新数量（前端提示 + 刷新用）
	var now model.EzfyItem
	h.DB.Where("user_id = ? AND cfg_id = ?", uid, in.CfgID).First(&now)
	msg := fmt.Sprintf("已把「%s」的【%s】数量设为 %d", nick, cfg.Name, now.Count)
	if mode == "add" {
		msg = fmt.Sprintf("已给「%s」添加【%s】×%d（现有 %d）", nick, cfg.Name, in.Count, now.Count)
	}
	resp.OK(c, gin.H{"msg": msg, "count": now.Count, "cfg_id": in.CfgID, "item_name": cfg.Name})
}

// AdminEzfyPlayerItemDelete DELETE /admin/ezfy-players/:id/item/:cfg_id —— 物理删除背包道具
//
// 只删这一行（user_id + cfg_id），玩家其余道具不受影响；不可恢复。
func (h *EzfyAdmin) AdminEzfyPlayerItemDelete(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	cfgID, _ := strconv.Atoi(c.Param("cfg_id"))
	if uid <= 0 || cfgID <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var row model.EzfyItem
	if err := h.DB.Where("user_id = ? AND cfg_id = ?", uid, cfgID).First(&row).Error; err != nil {
		resp.NotFound(c, "该玩家背包里没有这件道具")
		return
	}
	h.cfgs()
	name := ""
	if cfg := ezfyCfg.item(cfgID); cfg != nil {
		name = cfg.Name
	}
	if name == "" {
		name = "道具#" + strconv.Itoa(cfgID)
	}
	nick := ezfyNickByUid(h.DB, h.HomeDB, uint(uid))
	if err := h.DB.Exec("DELETE FROM ezfy_item WHERE user_id = ? AND cfg_id = ?", uid, cfgID).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	resp.OK(c, gin.H{
		"msg":   fmt.Sprintf("已物理删除「%s」的【%s】×%d", nick, name, row.Count),
		"count": row.Count, "cfg_id": cfgID, "item_name": name,
	})
}

// AdminEzfyPlayerItemOptions GET /admin/ezfy-player-items/options —— 背包「添加道具」下拉选项
//
// ★ 复用 AdminEzfyItemGrantOptions 的实现，但挂在 module:ezfyPlayers 权限下：
//
//	只有「玩家信息管理」权限、没有「数据管理」权限的管理员也能在这里加道具。
//	（直接复用 /admin/ezfy-item-grant/options 会 403 —— 那条在 module:ezfyData 下。）
func (h *EzfyAdmin) AdminEzfyPlayerItemOptions(c *gin.Context) {
	h.AdminEzfyItemGrantOptions(c)
}
