package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 二战风云管理端 —— 军衔配置维护 + 玩家军衔维护
//
// 军衔表复刻原版 rank/rankIndex.html：军衔等级 / 职位 / 需要声望 / **可建城数**。
// ★ 可建城数就是「玩家能拥有的城市数量上限」（军衔限制分城数量）。

// ============ 军衔配置 ============

// AdminEzfyRanks 军衔配置列表
func (h *AdminHandler) AdminEzfyRanks(c *gin.Context) {
	var rows []model.EzfyCfgRank
	h.DB.Order("id").Find(&rows)
	if len(rows) == 0 {
		rows = ezfyDefaultRanks()
	}
	// 每个军衔当前有多少玩家
	type cntAgg struct {
		RankIdx int
		Cnt     int64
	}
	var profs []model.EzfyProfile
	h.DB.Select("prestige").Find(&profs)
	cnt := map[int]int64{}
	for _, p := range profs {
		cnt[ezfyRankIndex(p.Prestige)]++
	}
	out := make([]gin.H, 0, len(rows))
	for i, r := range rows {
		out = append(out, gin.H{
			"id": r.ID, "level": i + 1, "name": r.Name, "post": r.Post,
			"need_prestige": r.NeedPrestige, "city_max": r.CityMax, "des": r.Des,
			"player_count": cnt[i],
		})
	}
	resp.OK(c, gin.H{"list": out, "total": len(out),
		"usage": "「可建城数」即该军衔下玩家能拥有的城市数量上限"})
}

// AdminEzfyRankUpdate 修改军衔配置
func (h *AdminHandler) AdminEzfyRankUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var r model.EzfyCfgRank
	if err := h.DB.First(&r, id).Error; err != nil {
		resp.NotFound(c, "军衔不存在")
		return
	}
	var in struct {
		Name         string `json:"name"`
		Post         string `json:"post"`
		NeedPrestige *int   `json:"need_prestige"`
		CityMax      *int   `json:"city_max"`
		Des          string `json:"des"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	updates := map[string]interface{}{}
	if s := strings.TrimSpace(in.Name); s != "" {
		updates["name"] = trimStr(s, 20)
	}
	if s := strings.TrimSpace(in.Post); s != "" {
		updates["post"] = trimStr(s, 20)
	}
	if in.NeedPrestige != nil {
		if *in.NeedPrestige < 0 {
			resp.ParamError(c, "需要声望不能为负")
			return
		}
		updates["need_prestige"] = *in.NeedPrestige
	}
	if in.CityMax != nil {
		if *in.CityMax < 1 {
			resp.ParamError(c, "可建城数至少为 1")
			return
		}
		updates["city_max"] = *in.CityMax
	}
	if in.Des != "" {
		updates["des"] = trimStr(in.Des, 200)
	}
	if len(updates) == 0 {
		resp.ParamError(c, "无可修改字段")
		return
	}
	if err := h.DB.Model(&model.EzfyCfgRank{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		resp.ParamError(c, "修改失败："+err.Error())
		return
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "军衔「" + r.Name + "」已保存"})
}

// AdminEzfyRankReset 恢复内置默认军衔表
func (h *AdminHandler) AdminEzfyRankReset(c *gin.Context) {
	for _, d := range ezfyDefaultRanks() {
		h.DB.Model(&model.EzfyCfgRank{}).Where("id = ?", d.ID).
			Updates(map[string]interface{}{
				"name": d.Name, "post": d.Post,
				"need_prestige": d.NeedPrestige, "city_max": d.CityMax,
			})
	}
	h.ezfyReload()
	resp.OK(c, gin.H{"msg": "军衔表已恢复默认"})
}

// ============ 玩家军衔 ============

// AdminEzfyRankPlayers 玩家军衔列表（含可建城数 / 实际城数）
func (h *AdminHandler) AdminEzfyRankPlayers(c *gin.Context) {
	page, offset, size := pageOf(c, 20)
	word := strings.TrimSpace(c.Query("word"))
	q := h.DB.Model(&model.EzfyProfile{})
	if word != "" {
		if n, err := strconv.Atoi(word); err == nil {
			q = q.Where("user_id = ? OR game_uid = ?", n, n)
		} else {
			q = q.Where("nickname LIKE ?", "%"+word+"%")
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.EzfyProfile
	q.Order("prestige DESC").Offset(offset).Limit(size).Find(&rows)

	out := make([]gin.H, 0, len(rows))
	for _, p := range rows {
		var cityCount int64
		h.DB.Model(&model.EzfyCity{}).Where("user_id = ?", p.UserID).Count(&cityCount)
		name, homeNum := h.ezfyAdminName(p.UserID)
		rank := ezfyRankOf(p.Prestige)
		out = append(out, gin.H{
			"user_id": p.UserID, "game_uid": p.GameUID,
			"nickname": p.Nickname, "player_name": name, "home_num": homeNum,
			"prestige": p.Prestige,
			"rank_id":  rank.ID, "rank_name": rank.Name, "rank_post": rank.Post,
			"city_max": ezfyRankCityMax(p.Prestige), "city_count": cityCount,
			"over_limit": cityCount > int64(ezfyRankCityMax(p.Prestige)),
		})
	}
	resp.OK(c, gin.H{"list": out, "total": total, "page": page, "size": size})
}

// AdminEzfyRankSetPlayer 把玩家设为指定军衔（写声望为该军衔门槛）
func (h *AdminHandler) AdminEzfyRankSetPlayer(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	var in struct {
		RankID int `json:"rank_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.RankID <= 0 {
		resp.ParamError(c, "请选择军衔")
		return
	}
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	rank := ezfyRankOf(0)
	found := false
	for _, r := range ezfyCfg.rankList() {
		if r.ID == in.RankID {
			rank, found = r, true
			break
		}
	}
	if !found {
		resp.ParamError(c, "军衔不存在")
		return
	}
	h.DB.Model(&model.EzfyProfile{}).Where("id = ?", p.ID).Update("prestige", rank.NeedPrestige)
	resp.OK(c, gin.H{"msg": "已将军衔设为「" + rank.Name + "」（声望 " +
		strconv.Itoa(rank.NeedPrestige) + "，可建城 " + strconv.Itoa(rank.CityMax) + " 座）"})
}

// AdminEzfyRankSetPrestige 直接设置玩家声望
func (h *AdminHandler) AdminEzfyRankSetPrestige(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("id"))
	var in struct {
		Prestige int `json:"prestige"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Prestige < 0 {
		resp.ParamError(c, "请填写不小于 0 的声望")
		return
	}
	var p model.EzfyProfile
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err != nil {
		resp.NotFound(c, "玩家不存在")
		return
	}
	h.DB.Model(&model.EzfyProfile{}).Where("id = ?", p.ID).Update("prestige", in.Prestige)
	resp.OK(c, gin.H{"msg": "声望已设为 " + strconv.Itoa(in.Prestige) +
		"，当前军衔「" + ezfyRankName(in.Prestige) + "」"})
}
