package handler

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 组队（复刻 xy111.php：上限4人，队长踢人/解散，队员离开）+ 国战（复刻 gz.php/gczc.php：
// 按星期轮换战场国家，整点后30分钟开战；报名帮派防守权杖，守满5分钟国家积分+10；
// 国战内奸每5分钟来袭，防守方击杀+1积分；进攻方战胜神兽守卫夺下权杖）

const hxTeamMax = 4        // 队伍人数上限
const hxGzHoldSec = 300    // 防守5分钟结算一次国家积分
const hxGzNeijianSec = 300 // 内奸每5分钟前来

// hxGzZc 按星期返回战场国家（原版：周日祭赛国…周六休整）
func hxGzZc(weekday int) (int, string) {
	switch weekday {
	case 1:
		return 1, "傲来国"
	case 2:
		return 2, "宝象国"
	case 3:
		return 3, "乌鸡国"
	case 4:
		return 4, "女儿国"
	case 5:
		return 5, "车迟国"
	case 0:
		return 7, "祭赛国"
	default:
		return 6, "休整"
	}
}

// hxGzWarOf 取（或初始化）今日战局
func (h *HxxyHandler) hxGzWarOf(today string, zcID int) *model.HxxyGzWar {
	var w model.HxxyGzWar
	if err := h.DB.Where("war_date = ?", today).First(&w).Error; err != nil {
		w = model.HxxyGzWar{WarDate: today, ZcID: zcID}
		h.DB.Create(&w)
	}
	return &w
}

// hxMyGang 我的帮派（ID/名称/角色）
func (h *HxxyHandler) hxMyGang(p *model.HxxyPlayer) (uint, string, int) {
	var gm model.HxxyGangMember
	if err := h.DB.Where("player_id = ?", p.ID).First(&gm).Error; err != nil {
		return 0, "", -1
	}
	var g model.HxxyGang
	if err := h.DB.First(&g, gm.GangID).Error; err != nil {
		return 0, "", -1
	}
	return g.ID, g.Name, gm.Role
}

// hxGzAddScore 国家积分+个人积分
func (h *HxxyHandler) hxGzAddScore(p *model.HxxyPlayer, gangID uint, gangName string, gj int, gr int) {
	if gj > 0 && gangID > 0 {
		var s model.HxxyGzScore
		if err := h.DB.Where("gang_id = ?", gangID).First(&s).Error; err != nil {
			s = model.HxxyGzScore{GangID: gangID, GangName: gangName}
			h.DB.Create(&s)
		}
		h.DB.Model(&model.HxxyGzScore{}).Where("gang_id = ?", gangID).UpdateColumn("total", s.Total+gj)
	}
	if gr > 0 {
		var gp model.HxxyGzPlayer
		if err := h.DB.Where("player_id = ?", p.ID).First(&gp).Error; err != nil {
			gp = model.HxxyGzPlayer{PlayerID: p.ID, Name: p.Name, GangName: gangName}
			h.DB.Create(&gp)
		}
		h.DB.Model(&model.HxxyGzPlayer{}).Where("player_id = ?", p.ID).UpdateColumn("total", gp.Total+gr)
	}
}

// GzInfo 国战信息页（战场/倒计时/我的阵营/积分榜）
func (h *HxxyHandler) GzInfo(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	now := time.Now()
	today := now.Format("2006-01-02")
	zcID, zcName := hxGzZc(int(now.Weekday()))
	w := h.hxGzWarOf(today, zcID)
	gangID, gangName, role := h.hxMyGang(p)

	// 开战判定：整点后30分钟内（原版 gczc.php 计时逻辑）
	minute := now.Minute()
	inWar := minute < 30 && zcID != 6
	leftSec := 0
	if inWar {
		leftSec = 30*60 - minute*60 - now.Second()
	}

	// 防守结算：权杖被防守方占据满5分钟 → 国家积分+10（原版 gz06/gczc 防守5分钟成功）
	if inWar && w.HolderGangID > 0 && w.HoldAt > 0 && time.Now().Unix()-w.HoldAt >= hxGzHoldSec {
		var s model.HxxyGzScore
		if err := h.DB.Where("gang_id = ?", w.HolderGangID).First(&s).Error; err != nil {
			s = model.HxxyGzScore{GangID: w.HolderGangID, GangName: w.HolderGangName}
			h.DB.Create(&s)
		}
		h.DB.Model(&model.HxxyGzScore{}).Where("gang_id = ?", w.HolderGangID).UpdateColumn("total", s.Total+10)
		h.hxWorldNotice(fmt.Sprintf("【国战】%s防守5分钟成功！！获得国家积分+10，请进攻国抓紧时间拿下权杖", w.HolderGangName))
		h.DB.Model(&model.HxxyGzWar{}).Where("id = ?", w.ID).UpdateColumn("hold_at", time.Now().Unix())
		w.HoldAt = time.Now().Unix()
	}

	// 内奸刷新：开战期间每5分钟前来（原版 21:05~21:25 内奸机制简化为战场内循环出现）
	neijian := ""
	if inWar && w.DefGangID > 0 {
		if w.NeijianAt == 0 || time.Now().Unix()-w.NeijianAt >= hxGzNeijianSec {
			name := fmt.Sprintf("国战内奸·%s", []string{"探子", "细作", "奸细", "叛徒", "密探"}[rand.Intn(5)])
			h.DB.Model(&model.HxxyGzWar{}).Where("id = ?", w.ID).Updates(map[string]interface{}{"neijian_at": time.Now().Unix(), "neijian_name": name})
			w.NeijianAt, w.NeijianName = time.Now().Unix(), name
		}
		if w.NeijianName != "" && time.Now().Unix()-w.NeijianAt < hxGzNeijianSec {
			neijian = w.NeijianName
		}
	}

	// 防守倒计时
	holdLeft := 0
	if inWar && w.HolderGangID > 0 && w.HoldAt > 0 {
		holdLeft = hxGzHoldSec - int(time.Now().Unix()-w.HoldAt)
		if holdLeft < 0 {
			holdLeft = 0
		}
	}

	// 积分榜
	var gsc []model.HxxyGzScore
	h.DB.Order("total DESC").Limit(10).Find(&gsc)
	var psc []model.HxxyGzPlayer
	h.DB.Order("total DESC").Limit(10).Find(&psc)
	var myGP model.HxxyGzPlayer
	h.DB.Where("player_id = ?", p.ID).First(&myGP)

	resp.OK(c, gin.H{
		"zc_id": zcID, "zc_name": zcName, "in_war": inWar, "left_sec": leftSec,
		"war": gin.H{"def_gang_id": w.DefGangID, "def_gang_name": w.DefGangName,
			"holder_gang_id": w.HolderGangID, "holder_gang_name": w.HolderGangName,
			"hold_left": holdLeft, "neijian": neijian},
		"my_gang_id": gangID, "my_gang_name": gangName, "my_role": role,
		"my_side": func() string {
			if gangID == 0 || w.DefGangID == 0 {
				return ""
			}
			if gangID == w.DefGangID {
				return "def"
			}
			return "atk"
		}(),
		"my_score":    myGP.Total,
		"gang_scores": gsc,
		"my_gang_score": func() int {
			for _, s := range gsc {
				if s.GangID == gangID {
					return s.Total
				}
			}
			return 0
		}(),
		"player_scores": psc,
	})
}

// GzSignup 帮主报名今日战场（成为防守方）
func (h *HxxyHandler) GzSignup(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	now := time.Now()
	zcID, zcName := hxGzZc(int(now.Weekday()))
	if zcID == 6 {
		resp.ParamError(c, "今日国战休整，明天再来！")
		return
	}
	gangID, gangName, role := h.hxMyGang(p)
	if gangID == 0 {
		resp.ParamError(c, "你还没有加入国家（帮派），无法参加国战！")
		return
	}
	if role != 2 {
		resp.ParamError(c, "只有帮主才能报名国战！")
		return
	}
	w := h.hxGzWarOf(now.Format("2006-01-02"), zcID)
	if w.DefGangID == gangID {
		resp.ParamError(c, "你的国家已报名今日国战！")
		return
	}
	if w.DefGangID > 0 {
		resp.ParamError(c, fmt.Sprintf("今日防守方已是【%s】，只能进攻！", w.DefGangName))
		return
	}
	holdAt := int64(0)
	if now.Minute() < 30 {
		holdAt = time.Now().Unix() // 开战中报名立即开始防守计时
	}
	h.DB.Model(&model.HxxyGzWar{}).Where("id = ?", w.ID).Updates(map[string]interface{}{
		"def_gang_id": gangID, "def_gang_name": gangName,
		"holder_gang_id": gangID, "holder_gang_name": gangName, "hold_at": holdAt})
	h.hxWorldNotice(fmt.Sprintf("【国战】【%s】已报名%s国战，成为防守方！", gangName, zcName))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("报名成功！【%s】成为今日%s防守方", gangName, zcName)})
}

// GzRod 进攻方夺权杖（战胜神兽守卫即占领）
func (h *HxxyHandler) GzRod(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	if p.HP <= 0 {
		resp.ParamError(c, "你已身受重伤，先恢复气血再参战！")
		return
	}
	if b := h.hxOpenBattle(p.ID); b != nil {
		resp.ParamError(c, "你正在战斗中")
		return
	}
	now := time.Now()
	zcID, zcName := hxGzZc(int(now.Weekday()))
	if now.Minute() >= 30 {
		resp.ParamError(c, "当前不是国战时间（每整点后30分钟开战）")
		return
	}
	w := h.hxGzWarOf(now.Format("2006-01-02"), zcID)
	if w.DefGangID == 0 {
		resp.ParamError(c, "今日还没有国家防守，无需进攻")
		return
	}
	gangID, gangName, _ := h.hxMyGang(p)
	if gangID == 0 {
		resp.ParamError(c, "你还没有加入国家（帮派），无法参加国战！")
		return
	}
	if gangID == w.DefGangID {
		resp.ParamError(c, "你是防守方，守住权杖即可，无需夺权杖！")
		return
	}
	lv := p.Level + 10
	if lv < 20 {
		lv = 20
	}
	f := float64(lv)
	enemy := hxEnemy{
		ID: 1, Name: fmt.Sprintf("【%s】神兽守卫", zcName), Level: lv,
		MaxHP: int(math.Pow(f+15, 2)*2.6) + 800, MP: int(f) * 10, MaxMP: int(f) * 10,
		Atk: int((f+1)*(f+2)*3) + 400, Mg: int((f+1)*(f+2)*3) + 400,
		Def: int(math.Pow(f+1, 2)*2.2) + 250, Mf: int(math.Pow(f+1, 2)*2.2) + 250,
		Kind: 0, Type: "gz_rod",
	}
	enemy.HP = enemy.MaxHP
	self := h.hxSelfSnap(p)
	logs := []string{fmt.Sprintf("你随【%s】进攻%s，向神兽守卫发起猛攻，夺下权杖！", gangName, zcName)}
	b := &model.HxxyBattle{
		PlayerID: p.ID, Type: "gz", EnemyID: 1, EnemyName: enemy.Name,
		Round: 0, Status: 1, Enemy: hxJSON(enemy), Self: hxJSON(self), Log: hxJSON(logs),
	}
	h.DB.Create(b)
	resp.OK(c, h.hxBattleView(p, b))
}

// GzNeijian 防守方击杀内奸
func (h *HxxyHandler) GzNeijian(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	if p.HP <= 0 {
		resp.ParamError(c, "你已身受重伤，先恢复气血再参战！")
		return
	}
	if b := h.hxOpenBattle(p.ID); b != nil {
		resp.ParamError(c, "你正在战斗中")
		return
	}
	now := time.Now()
	zcID, _ := hxGzZc(int(now.Weekday()))
	if now.Minute() >= 30 {
		resp.ParamError(c, "当前不是国战时间（每整点后30分钟开战）")
		return
	}
	w := h.hxGzWarOf(now.Format("2006-01-02"), zcID)
	gangID, _, _ := h.hxMyGang(p)
	if w.DefGangID == 0 || gangID != w.DefGangID {
		resp.ParamError(c, "只有防守方才能击杀国战内奸！")
		return
	}
	if w.NeijianName == "" || time.Now().Unix()-w.NeijianAt >= hxGzNeijianSec {
		resp.ParamError(c, "内奸还没有出现，稍等！")
		return
	}
	lv := p.Level + 3
	f := float64(lv)
	enemy := hxEnemy{
		ID: 2, Name: w.NeijianName, Level: lv,
		MaxHP: int(math.Pow(f+15, 2) * 2), MP: int(f) * 10, MaxMP: int(f) * 10,
		Atk: int((f+1)*(f+2)*3) + 300, Mg: int((f+1)*(f+2)*3) + 300,
		Def: int(math.Pow(f+1, 2)*2) + 200, Mf: int(math.Pow(f+1, 2)*2) + 200,
		Kind: 0, Type: "gz_neijian",
	}
	enemy.HP = enemy.MaxHP
	self := h.hxSelfSnap(p)
	logs := []string{fmt.Sprintf("【%s】混入了国战队伍，你上前将其拿下！", w.NeijianName)}
	b := &model.HxxyBattle{
		PlayerID: p.ID, Type: "gz", EnemyID: 2, EnemyName: enemy.Name,
		Round: 0, Status: 1, Enemy: hxJSON(enemy), Self: hxJSON(self), Log: hxJSON(logs),
	}
	h.DB.Create(&b)
	resp.OK(c, h.hxBattleView(p, b))
}

// hxGzSettle 国战战斗结算（在 hxFinishBattle 中按 Type 分发）
func (h *HxxyHandler) hxGzSettle(p *model.HxxyPlayer, b *model.HxxyBattle, enemy *hxEnemy, result int) {
	if result != 2 {
		return // 战败/逃跑无惩罚
	}
	gangID, gangName, _ := h.hxMyGang(p)
	now := time.Now()
	zcID, _ := hxGzZc(int(now.Weekday()))
	w := h.hxGzWarOf(now.Format("2006-01-02"), zcID)
	switch enemy.Type {
	case "gz_rod":
		if gangID == 0 {
			return
		}
		h.DB.Model(&model.HxxyGzWar{}).Where("id = ?", w.ID).Updates(map[string]interface{}{
			"holder_gang_id": gangID, "holder_gang_name": gangName, "hold_at": time.Now().Unix()})
		h.hxGzAddScore(p, gangID, gangName, 0, 5)
		h.hxWorldNotice(fmt.Sprintf("【国战】【%s】夺下了国家权杖！开始防守计时", gangName))
		h.hxNotify(p.ID, "你击败神兽守卫，夺下权杖！个人积分+5")
	case "gz_neijian":
		if gangID == 0 || gangID != w.DefGangID {
			return
		}
		h.hxGzAddScore(p, gangID, gangName, 1, 1)
		h.DB.Model(&model.HxxyGzWar{}).Where("id = ?", w.ID).Updates(map[string]interface{}{"neijian_name": "", "neijian_at": 0})
		h.hxNotify(p.ID, "你击杀了国战内奸！国家积分+1，个人积分+1")
	}
}

// hxWorldNotice 全服公告（所有玩家系统消息）
func (h *HxxyHandler) hxWorldNotice(content string) {
	var players []model.HxxyPlayer
	h.DB.Select("id").Find(&players)
	for _, pl := range players {
		h.hxNotify(pl.ID, content)
	}
}

// ==================== 组队 ====================

// hxTeamBrief 我的队伍简要（无队返回 nil）
func (h *HxxyHandler) hxTeamBrief(p *model.HxxyPlayer) gin.H {
	var tm model.HxxyTeamMember
	if err := h.DB.Where("player_id = ?", p.ID).First(&tm).Error; err != nil {
		return nil
	}
	var team model.HxxyTeam
	if err := h.DB.First(&team, tm.TeamID).Error; err != nil {
		h.DB.Where("id = ?", tm.ID).Delete(&model.HxxyTeamMember{})
		return nil
	}
	var members []model.HxxyTeamMember
	h.DB.Where("team_id = ?", team.ID).Order("id ASC").Find(&members)
	ms := []gin.H{}
	for _, m := range members {
		isLeader := m.PlayerID == team.LeaderID
		var lp model.HxxyPlayer
		level, vip := 0, 0
		if h.DB.First(&lp, m.PlayerID).Error == nil {
			level, vip = lp.Level, lp.Vip
		}
		ms = append(ms, gin.H{"player_id": m.PlayerID, "name": m.Name, "level": level, "vip": vip,
			"is_leader": isLeader, "is_me": m.PlayerID == p.ID})
	}
	return gin.H{"team_id": team.ID, "leader_id": team.LeaderID, "leader_name": team.LeaderName,
		"count": len(members), "max": hxTeamMax, "members": ms, "is_leader": team.LeaderID == p.ID}
}

// TeamInfo 队伍页（我的队伍 + 待处理邀请）
func (h *HxxyHandler) TeamInfo(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var invites []model.HxxyTeamInvite
	h.DB.Where("to_id = ? AND status = 0", p.ID).Order("id DESC").Find(&invites)
	il := []gin.H{}
	for _, iv := range invites {
		il = append(il, gin.H{"id": iv.ID, "from_name": iv.FromName, "from_id": iv.FromID, "created_at": iv.CreatedAt})
	}
	resp.OK(c, gin.H{"team": h.hxTeamBrief(p), "invites": il})
}

// TeamCreate 创建队伍（自己当队长）
func (h *HxxyHandler) TeamCreate(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	if t := h.hxTeamBrief(p); t != nil {
		resp.ParamError(c, "你已经在队伍中了！")
		return
	}
	team := model.HxxyTeam{LeaderID: p.ID, LeaderName: p.Name}
	h.DB.Create(&team)
	h.DB.Create(&model.HxxyTeamMember{TeamID: team.ID, PlayerID: p.ID, Name: p.Name})
	resp.OK(c, gin.H{"msg": "队伍创建成功！去邀请好友加入吧（玩家资料页点[组队]）", "team": h.hxTeamBrief(p)})
}

// TeamInvite 邀请组队（玩家资料页入口）
func (h *HxxyHandler) TeamInvite(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		TargetID uint `json:"target_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.TargetID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.TargetID == p.ID {
		resp.ParamError(c, "不能邀请自己")
		return
	}
	var target model.HxxyPlayer
	if err := h.DB.First(&target, in.TargetID).Error; err != nil {
		resp.ParamError(c, "没有这个玩家")
		return
	}
	// 我的队伍：没队自动建队
	t := h.hxTeamBrief(p)
	var teamID uint
	var leaderID uint
	if t == nil {
		team := model.HxxyTeam{LeaderID: p.ID, LeaderName: p.Name}
		h.DB.Create(&team)
		h.DB.Create(&model.HxxyTeamMember{TeamID: team.ID, PlayerID: p.ID, Name: p.Name})
		teamID, leaderID = team.ID, p.ID
	} else {
		teamID = t["team_id"].(uint)
		leaderID = t["leader_id"].(uint)
		if t["count"].(int) >= hxTeamMax {
			resp.ParamError(c, "邀请失败！队伍已满员")
			return
		}
		if leaderID != p.ID {
			resp.ParamError(c, "只有队长才能邀请玩家组队！")
			return
		}
	}
	// 对方是否已有队伍
	var tm model.HxxyTeamMember
	if h.DB.Where("player_id = ?", target.ID).First(&tm).Error == nil {
		resp.ParamError(c, "对方已经有队伍了！")
		return
	}
	// 是否已邀请过（待处理）
	var cnt int64
	h.DB.Model(&model.HxxyTeamInvite{}).Where("team_id = ? AND to_id = ? AND status = 0", teamID, target.ID).Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "你已经邀请过该玩家了！请等待对方同意")
		return
	}
	h.DB.Create(&model.HxxyTeamInvite{TeamID: teamID, FromID: p.ID, FromName: p.Name, ToID: target.ID, ToName: target.Name})
	h.hxNotify(target.ID, fmt.Sprintf("【%s】邀请你加入队伍！请到[队伍]页查看并同意", p.Name))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("你向玩家【%s】发起了组队邀请，请等待对方同意", target.Name)})
}

// TeamAgree 同意邀请
func (h *HxxyHandler) TeamAgree(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		InviteID uint `json:"invite_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.InviteID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var iv model.HxxyTeamInvite
	if err := h.DB.First(&iv, in.InviteID).Error; err != nil || iv.ToID != p.ID || iv.Status != 0 {
		resp.ParamError(c, "没有这条邀请")
		return
	}
	h.DB.Model(&model.HxxyTeamInvite{}).Where("id = ?", iv.ID).Update("status", 1)
	if h.hxTeamBrief(p) != nil {
		resp.ParamError(c, "你已经有队伍了，无法加入！")
		return
	}
	var cnt int64
	h.DB.Model(&model.HxxyTeamMember{}).Where("team_id = ?", iv.TeamID).Count(&cnt)
	if cnt >= hxTeamMax {
		resp.ParamError(c, "队伍已满员！")
		return
	}
	var team model.HxxyTeam
	if err := h.DB.First(&team, iv.TeamID).Error; err != nil {
		resp.ParamError(c, "队伍已解散")
		return
	}
	h.DB.Create(&model.HxxyTeamMember{TeamID: iv.TeamID, PlayerID: p.ID, Name: p.Name})
	h.hxNotify(team.LeaderID, fmt.Sprintf("【%s】同意了你的组队邀请，已加入队伍！", p.Name))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("你已加入【%s】的队伍！", team.LeaderName), "team": h.hxTeamBrief(p)})
}

// TeamRefuse 拒绝邀请
func (h *HxxyHandler) TeamRefuse(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		InviteID uint `json:"invite_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.InviteID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var iv model.HxxyTeamInvite
	if err := h.DB.First(&iv, in.InviteID).Error; err != nil || iv.ToID != p.ID || iv.Status != 0 {
		resp.ParamError(c, "没有这条邀请")
		return
	}
	h.DB.Model(&model.HxxyTeamInvite{}).Where("id = ?", iv.ID).Update("status", 2)
	resp.OK(c, gin.H{"msg": "已拒绝该邀请"})
}

// TeamKick 队长踢人
func (h *HxxyHandler) TeamKick(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		TargetID uint `json:"target_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.TargetID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	t := h.hxTeamBrief(p)
	if t == nil || !t["is_leader"].(bool) {
		resp.ParamError(c, "只有队长才能踢人！")
		return
	}
	if in.TargetID == p.ID {
		resp.ParamError(c, "不能踢自己（解散队伍即可）")
		return
	}
	var tm model.HxxyTeamMember
	if err := h.DB.Where("team_id = ? AND player_id = ?", t["team_id"], in.TargetID).First(&tm).Error; err != nil {
		resp.ParamError(c, "他不在你的队伍里")
		return
	}
	h.DB.Delete(&tm)
	h.hxNotify(in.TargetID, fmt.Sprintf("你被队长【%s】踢出了队伍", p.Name))
	resp.OK(c, gin.H{"msg": "已将其踢出队伍", "team": h.hxTeamBrief(p)})
}

// TeamLeave 离开队伍（队长离开=解散）
func (h *HxxyHandler) TeamLeave(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	t := h.hxTeamBrief(p)
	if t == nil {
		resp.ParamError(c, "你还没有队伍")
		return
	}
	teamID := t["team_id"].(uint)
	if t["is_leader"].(bool) {
		// 队长解散：通知全部队员
		var members []model.HxxyTeamMember
		h.DB.Where("team_id = ?", teamID).Find(&members)
		for _, m := range members {
			if m.PlayerID != p.ID {
				h.hxNotify(m.PlayerID, fmt.Sprintf("队长【%s】解散了队伍", p.Name))
			}
		}
		h.DB.Where("team_id = ?", teamID).Delete(&model.HxxyTeamMember{})
		h.DB.Delete(&model.HxxyTeam{}, teamID)
		h.DB.Model(&model.HxxyTeamInvite{}).Where("team_id = ? AND status = 0", teamID).Update("status", 2)
		resp.OK(c, gin.H{"msg": "队伍已解散", "team": nil})
		return
	}
	h.DB.Where("team_id = ? AND player_id = ?", teamID, p.ID).Delete(&model.HxxyTeamMember{})
	var team model.HxxyTeam
	if h.DB.First(&team, teamID).Error == nil {
		h.hxNotify(team.LeaderID, fmt.Sprintf("【%s】离开了队伍", p.Name))
	}
	resp.OK(c, gin.H{"msg": "你已离开队伍", "team": h.hxTeamBrief(p)})
}
