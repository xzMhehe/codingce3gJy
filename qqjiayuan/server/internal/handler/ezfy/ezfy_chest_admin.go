// 二战风云 管理端「发放宝箱」：给玩家发待开宝箱（入 ezfy_user_chest，玩家自己开）
//
// ★ 2026-10-14 用户需求「宝箱也能发放给玩家加个功能」：在「套装装备配置（宝箱）」页发放。
//   与「发放道具」同款：player 支持玩家昵称或游戏ID(user_id)，发放入持有表 + 站内通知。
package ezfy

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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
// 目标玩家进入「我的宝箱」（ezfy_user_chest）：已有的宝箱数量累加（唯一键 user_id+chest_id）。
// ⚠️ 与发放道具口径一致：不校验数量上限（运营发放本就该给多少算多少）。
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
	for _, it := range in.Items {
		if it.ChestID <= 0 || it.Count <= 0 {
			continue
		}
		var chest model.EzfyCfgChest
		if err := h.DB.First(&chest, it.ChestID).Error; err != nil {
			resp.ParamError(c, "宝箱不存在："+strconv.Itoa(it.ChestID))
			return
		}
		// 原子 upsert：已有该宝箱 → count 累加；没有 → 新建一行（唯一键 user_id+chest_id）
		h.DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "chest_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"count": gorm.Expr("count + ?", it.Count),
			}),
		}).Create(&model.EzfyUserChest{UserID: p.UserID, ChestId: it.ChestID, Count: it.Count})
		names += fmt.Sprintf(" 【%s】×%d", chest.Name, it.Count)
	}
	if names == "" {
		resp.ParamError(c, "请添加要发放的宝箱")
		return
	}
	h.DB.Create(&model.EzfyNotice{UserId: p.UserID, Title: "管理员发放宝箱",
		Content: "管理员发放宝箱：" + strings.TrimSpace(names) + "，请在「我的宝箱」中开启，请查收。"})
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已发放给玩家「%s」(ID:%d)%s", p.Nickname, p.UserID, names)})
}