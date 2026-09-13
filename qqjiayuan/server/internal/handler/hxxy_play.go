package handler

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 玩法扩展（复刻原版：挑战=通天塔 / 擂台=天下第一武道大会 / 娱乐=疯狂摇一摇 / 腾云）

const hxArenaDaily = 5 // 每日比武次数上限（照抄原版擂台每日5次）

// hxPlayerRankRow 比武排行行
type hxPlayerRankRow struct {
	PlayerID uint `json:"player_id"`
	Wins     int  `json:"wins"`
}

// ArenaInfo 天下第一武道大会（复刻 xy402.php：前十排行+查看+比武+每日5次）
func (h *HxxyHandler) ArenaInfo(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	// 比武胜场排行（pvp 战报 result=2 计胜）
	var rows []hxPlayerRankRow
	h.DB.Raw(`SELECT player_id, COUNT(*) AS wins FROM hxxy_battle_logs
		WHERE type = 'pvp' AND result = 2 GROUP BY player_id ORDER BY wins DESC, player_id ASC LIMIT 10`).
		Scan(&rows)
	rank := []gin.H{}
	myRank := 0
	for i, r := range rows {
		if r.PlayerID == p.ID {
			myRank = i + 1
		}
		var tp model.HxxyPlayer
		name, level := "未知", 0
		if err := h.DB.First(&tp, r.PlayerID).Error; err == nil {
			name, level = tp.Name, tp.Level
		}
		rank = append(rank, gin.H{"rank": i + 1, "player_id": r.PlayerID, "name": name, "level": level, "wins": r.Wins})
	}
	// 我的胜场
	var myWins int64
	h.DB.Model(&model.HxxyBattleLog{}).Where("player_id = ? AND type = 'pvp' AND result = 2", p.ID).Count(&myWins)
	resp.OK(c, gin.H{
		"rank":   rank,
		"me":     gin.H{"rank": myRank, "wins": myWins, "today": p.DayArena, "limit": hxArenaDaily},
	})
}

// ArenaFight 发起比武（PvP，敌方为对方玩家实时属性快照）
func (h *HxxyHandler) ArenaFight(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	if p.HP <= 0 {
		resp.ParamError(c, "你已身受重伤，先恢复气血再比武！")
		return
	}
	if b := h.hxOpenBattle(p.ID); b != nil {
		resp.OK(c, h.hxBattleView(p, b))
		return
	}
	if p.DayArena >= hxArenaDaily {
		resp.ParamError(c, fmt.Sprintf("今日比武次数已用完（每日%d次），明天再来！", hxArenaDaily))
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
		resp.ParamError(c, "不能和自己比武")
		return
	}
	var target model.HxxyPlayer
	if err := h.DB.First(&target, in.TargetID).Error; err != nil {
		resp.ParamError(c, "没有这个玩家")
		return
	}
	ta := h.hxAttrs(&target)
	enemy := hxEnemy{
		ID: target.ID, Name: target.Name, Level: target.Level,
		HP: ta.MaxHP, MaxHP: ta.MaxHP, MP: ta.MaxMP, MaxMP: ta.MaxMP,
		Atk: ta.Atk, Mg: ta.Mg, Def: ta.Def, Mf: ta.Def,
		Bg: ta.Bg, Hg: ta.Hg, Lg: ta.Lg, Bf: ta.Bf, Hf: ta.Hf, Lf: ta.Lf,
		Kind: 0, Type: "pvp",
	}
	if target.HP > 0 && target.HP < enemy.MaxHP {
		enemy.HP = target.HP
	}
	self := h.hxSelfSnap(p)
	logs := []string{fmt.Sprintf("你向%s（%d级）发起了比武挑战！", hxWName(target.Name), target.Level)}
	b := model.HxxyBattle{
		PlayerID: p.ID, Type: "pvp", EnemyID: target.ID, EnemyName: target.Name,
		Round: 1, Status: 1, Enemy: hxJSON(enemy), Self: hxJSON(self), Log: hxJSON(logs),
	}
	h.DB.Create(&b)
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).UpdateColumn("day_arena", p.DayArena+1)
	resp.OK(c, h.hxBattleView(p, &b))
}

// hxPvpSettle 比武结算（胜者拿败者10%银两，上限1000；主动挑战者胜+1恶名）
func (h *HxxyHandler) hxPvpSettle(p *model.HxxyPlayer, b *model.HxxyBattle, enemy *hxEnemy, result int) {
	var target model.HxxyPlayer
	if err := h.DB.First(&target, enemy.ID).Error; err != nil {
		return
	}
	win := result == 2
	loser := &target
	winner := p
	if !win {
		loser = p
		winner = &target
	}
	pot := loser.Money / 10
	if pot > 1000 {
		pot = 1000
	}
	if pot > 0 {
		h.hxWallet(loser, "money", -pot, "比武败给【"+winner.Name+"】")
		h.hxWallet(winner, "money", pot, "比武胜【"+loser.Name+"】")
	}
	if win {
		h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).UpdateColumn("emz", p.Emz+1)
	}
	// 对方也记一条战报（武道会胜场统计用）
	res := 3
	if !win {
		res = 2
	}
	h.DB.Create(&model.HxxyBattleLog{PlayerID: target.ID, Type: "pvp", EnemyName: p.Name,
		Result: res, Round: b.Round, Money: int(pot), Loot: "[]"})
	if win {
		h.hxNotify(p.ID, fmt.Sprintf("你在比武中击败了【%s】，赢得 %d 银两！", target.Name, pot))
		h.hxNotify(target.ID, fmt.Sprintf("你被【%s】击败，损失 %d 银两！", p.Name, pot))
	} else {
		h.hxNotify(p.ID, fmt.Sprintf("你在比武中败给了【%s】，被夺走 %d 银两。", target.Name, pot))
		h.hxNotify(target.ID, fmt.Sprintf("你击退了【%s】的比武挑战，赢得 %d 银两！", p.Name, pot))
	}
}

// TowerInfo 通天塔信息
func (h *HxxyHandler) TowerInfo(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	resp.OK(c, gin.H{"floor": p.TowerFloor, "best": p.TowerBest})
}

// TowerStart 通天塔挑战（敌方按层数成长，胜利上楼，战败/逃跑重置）
func (h *HxxyHandler) TowerStart(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	if p.HP <= 0 {
		resp.ParamError(c, "你已身受重伤，先恢复气血再挑战！")
		return
	}
	if b := h.hxOpenBattle(p.ID); b != nil {
		resp.ParamError(c, "你正在战斗中")
		return
	}
	floor := p.TowerFloor + 1
	lv := p.Level + floor
	if lv < 1 {
		lv = 1
	}
	f := float64(lv)
	enemy := hxEnemy{
		ID: uint(floor), Name: fmt.Sprintf("通天塔守卫·第%d层", floor), Level: lv,
		MaxHP: int(math.Pow(f+15, 2) * 2.2), MP: int(f) * 10, MaxMP: int(f) * 10,
		Atk: int((f+1)*(f+2)*3) + 250, Mg: int((f+1)*(f+2)*3) + 250,
		Def: int(math.Pow(f+1, 2)*2.2) + 180, Mf: int(math.Pow(f+1, 2)*2.2) + 180,
		Kind: 0, Type: "tower",
		ExpReward: lv*lv*15 + 80, MoneyReward: int64(lv)*40 + 30,
	}
	enemy.HP = enemy.MaxHP
	self := h.hxSelfSnap(p)
	logs := []string{fmt.Sprintf("你踏上了通天塔第 %d 层，%s拦住了去路！", floor, hxWName(enemy.Name))}
	b := model.HxxyBattle{
		PlayerID: p.ID, Type: "tower", EnemyID: uint(floor), EnemyName: enemy.Name,
		Round: 1, Status: 1, Enemy: hxJSON(enemy), Self: hxJSON(self), Log: hxJSON(logs),
	}
	h.DB.Create(&b)
	resp.OK(c, h.hxBattleView(p, &b))
}

// hxTowerSettle 通天塔结算（胜利层数+1并刷新最高，战败/逃跑重置）
func (h *HxxyHandler) hxTowerSettle(p *model.HxxyPlayer, result int) {
	if result == 2 {
		next := p.TowerFloor + 1
		updates := map[string]interface{}{"tower_floor": next}
		if next > p.TowerBest {
			updates["tower_best"] = next
		}
		h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Updates(updates)
		h.hxNotify(p.ID, fmt.Sprintf("通天塔第 %d 层通过，继续向上！", next))
	} else {
		h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update("tower_floor", 0)
		if result == 3 {
			h.hxNotify(p.ID, "通天塔挑战失败，已送回第一层。")
		}
	}
}

// hxFunField 摇一摇场次配置
var hxFunFields = map[string]struct {
	Cur  string
	Name string
	Bet  int64
}{
	"m1": {"money", "疯狂摇一摇(银两·黄金场)", 100},
	"m2": {"money", "疯狂摇一摇(银两·铂金场)", 1000},
	"b1": {"beans", "疯狂摇一摇(金豆·黄金场)", 10},
	"b2": {"beans", "疯狂摇一摇(金豆·铂金场)", 100},
}

// FunRoll 疯狂摇一摇（复刻 xy403.php 娱乐场：三宫格，三个相同×8，两个相同×2）
func (h *HxxyHandler) FunRoll(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		Field string `json:"field"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	f, ok := hxFunFields[in.Field]
	if !ok {
		resp.ParamError(c, "没有这个场次")
		return
	}
	if f.Cur == "money" && p.Money < f.Bet {
		resp.ParamError(c, fmt.Sprintf("需要 %d 银两，银两不足", f.Bet))
		return
	}
	if f.Cur == "beans" && int64(p.Beans) < f.Bet {
		resp.ParamError(c, fmt.Sprintf("需要 %d 金豆，金豆不足", f.Bet))
		return
	}
	h.hxWallet(p, f.Cur, -f.Bet, "娱乐·"+f.Name)
	symbols := []string{"金", "木", "水", "火", "土"}
	roll := []string{symbols[rand.Intn(len(symbols))], symbols[rand.Intn(len(symbols))], symbols[rand.Intn(len(symbols))]}
	win := int64(0)
	if roll[0] == roll[1] && roll[1] == roll[2] {
		win = f.Bet * 8
	} else if roll[0] == roll[1] || roll[1] == roll[2] || roll[0] == roll[2] {
		win = f.Bet * 2
	}
	msg := "很遗憾，没有摇中……"
	if win > 0 {
		h.hxWallet(p, f.Cur, win, "娱乐·"+f.Name+"中奖")
		msg = fmt.Sprintf("恭喜摇中，赢得 %d %s！", win, map[string]string{"money": "银两", "beans": "金豆"}[f.Cur])
	}
	resp.OK(c, gin.H{"symbols": roll, "win": win, "msg": msg})
}

// teyunDests 复刻原版 xy476.php 腾云目的地（区域名 → 传送坐标，坐标取自地图传送NPC teles）
var teyunDests = []struct{ Cat, Name string; Dtx, Dty int }{
	// 门派区域
	{"门派区域", "龙宫", 2, 1}, {"门派区域", "月宫", 7, 0}, {"门派区域", "普陀", 13, 0},
	{"门派区域", "方寸", 10, 0}, {"门派区域", "将军", 1, 72}, {"门派区域", "地府", 22, 0},
	// 主城区域
	{"主城区域", "长安城", 1, 0}, {"主城区域", "傲来国", 24, 0}, {"主城区域", "祭赛国", 25, 0},
	{"主城区域", "天宫", 23, 0}, {"主城区域", "东海海岸", 24, 7},
	// 野外区域
	{"野外区域", "城南荒野", 1, 122}, {"野外区域", "慈恩寺", 1, 86}, {"野外区域", "昆仑山区", 1, 104},
	{"野外区域", "雪山迷宫", 44, 12}, {"野外区域", "泾水河", 5, 12}, {"野外区域", "海底莽林", 3, 0},
	// 副本区域
	{"副本区域", "双叉岭", 35, 12}, {"副本区域", "黑松林", 36, 12}, {"副本区域", "白骨洞", 38, 12},
	{"副本区域", "金塔", 37, 12}, {"副本区域", "平顶山", 39, 12}, {"副本区域", "老君炉", 34, 12},
	{"副本区域", "斩妖台", 33, 12}, {"副本区域", "蟠桃园", 32, 12}, {"副本区域", "紫竹林", 14, 0},
	{"副本区域", "兜率宫", 23, 24}, {"副本区域", "斩妖台", 23, 29}, {"副本区域", "银河", 31, 12},
}

// TeyunList 腾云目的地（按原版 xy476.php 分类，消耗腾云符传送到各区域）
func (h *HxxyHandler) TeyunList(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	// 腾云符余量
	fuCount := h.hxItemCount(p.ID, "腾云符")
	list := []gin.H{}
	seen := map[string]bool{}
	for _, d := range teyunDests {
		if d.Dtx == p.MapX && d.Dty == p.MapY {
			continue
		}
		key := fmt.Sprintf("%d-%d", d.Dtx, d.Dty)
		if seen[key] {
			continue // 去重（如两个斩妖台取其一）
		}
		seen[key] = true
		list = append(list, gin.H{"cat": d.Cat, "name": d.Name, "dtx": d.Dtx, "dty": d.Dty})
	}
	resp.OK(c, gin.H{"fu": fuCount, "list": list})
}

// hxItemCount 背包内指定名称物品数量
func (h *HxxyHandler) hxItemCount(playerID uint, name string) int {
	var cnt int64
	h.DB.Raw(`SELECT IFNULL(SUM(b.count),0) FROM hxxy_bag b JOIN hxxy_items i ON i.id = b.ref_id
		WHERE b.player_id = ? AND b.kind = 'item' AND b.store = 0 AND i.name = ?`, playerID, name).Scan(&cnt)
	return int(cnt)
}

// TeyunGo 腾云传送（消耗1张腾云符）
func (h *HxxyHandler) TeyunGo(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		Dtx int `json:"dtx"`
		Dty int `json:"dty"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	node := h.hxNode(in.Dtx, in.Dty)
	if node == nil {
		resp.ParamError(c, "没有这个地方")
		return
	}
	if node.Dtx == p.MapX && node.Dty == p.MapY {
		resp.ParamError(c, "你已在此地")
		return
	}
	var fu model.HxxyItem
	if err := h.DB.Where("name = ?", "腾云符").First(&fu).Error; err != nil {
		resp.ParamError(c, "商店暂未出售腾云符，无法腾云")
		return
	}
	var bag model.HxxyBag
	if err := h.DB.Where("player_id = ? AND kind = 'item' AND ref_id = ? AND store = 0", p.ID, fu.ID).
		Order("id").First(&bag).Error; err != nil {
		resp.ParamError(c, "没有腾云符，无法腾云（可在商店购买）")
		return
	}
	if !h.hxBagSub(p.ID, bag.ID, 1) {
		resp.ParamError(c, "没有腾云符，无法腾云（可在商店购买）")
		return
	}
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Updates(map[string]interface{}{"map_x": node.Dtx, "map_y": node.Dty})
	p.MapX, p.MapY = node.Dtx, node.Dty
	resp.OK(c, gin.H{"msg": fmt.Sprintf("腾云驾雾，转眼便到了【%s】！", node.Name), "player": h.hxPlayerBrief(p)})
}

// PlayerView 玩家资料（查看/私聊/加好友/比武 入口）
func (h *HxxyHandler) PlayerView(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	var t model.HxxyPlayer
	if err := h.DB.First(&t, id).Error; err != nil {
		resp.ParamError(c, "没有这个玩家")
		return
	}
	var wins int64
	h.DB.Model(&model.HxxyBattleLog{}).Where("player_id = ? AND type = 'pvp' AND result = 2", t.ID).Count(&wins)
	gangName := ""
	var mem model.HxxyGangMember
	if err := h.DB.Where("player_id = ?", t.ID).First(&mem).Error; err == nil {
		var g model.HxxyGang
		if err2 := h.DB.First(&g, mem.GangID).Error; err2 == nil {
			gangName = g.Name
		}
	}
	titleName := ""
	if t.TitleID > 0 {
		var ti model.HxxyTitle
		if err := h.DB.First(&ti, t.TitleID).Error; err == nil {
			titleName = ti.Name
		}
	}
	nodeName := "未知之地"
	if n := h.hxNode(t.MapX, t.MapY); n != nil {
		// 所在地 = 地图区域名·节点名（如 长安城·朱雀大街）
		nodeName = n.Name
		if area := h.hxMapName(t.MapX); area != "" {
			nodeName = area + "·" + n.Name
		}
	}
	// 配偶（复刻 xy093：配偶：X/暂无）
	spouseName := ""
	var mg model.HxxyMarriage
	if err := h.DB.Where("(player_a = ? OR player_b = ?) AND status = 2", t.ID, t.ID).Order("id DESC").First(&mg).Error; err == nil {
		if mg.PlayerA == t.ID {
			spouseName = h.hxNameByID(mg.PlayerB)
		} else {
			spouseName = h.hxNameByID(mg.PlayerA)
		}
	}
	// 住宅（复刻 xy093：住宅：X/暂无）
	houseName := ""
	var hsCount int64
	h.DB.Model(&model.HxxyHouse{}).Where("player_id = ?", t.ID).Count(&hsCount)
	if hsCount > 0 {
		houseName = t.Name + "的住宅"
	}
	resp.OK(c, gin.H{
		"player_id": t.ID, "name": t.Name, "sex": t.Sex, "level": t.Level,
		"sect_name": hxSectNames[t.Sect], "gang": gangName, "title": titleName,
		"emz": t.Emz, "wins": wins, "tower_best": t.TowerBest, "node_name": nodeName,
		"spouse": spouseName, "house": houseName,
		"is_me": t.ID == p.ID,
	})
}

// GiveMoney 赠银（复刻 xy537：1000两起，2%手续费最低1两，对方收全额并收私聊）
func (h *HxxyHandler) GiveMoney(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		ToID   uint  `json:"to_id"`
		Amount int64 `json:"amount"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.ToID == 0 {
		resp.ParamError(c, "输入有误，或者不能为空")
		return
	}
	if in.Amount < 1000 || in.Amount > 99999999999 {
		resp.ParamError(c, "赠送银两必须大于1000两并且小于999亿9999万9999两~~")
		return
	}
	fee := in.Amount * 2 / 100
	if fee < 1 {
		fee = 1
	}
	if p.Money < in.Amount+fee {
		resp.ParamError(c, fmt.Sprintf("对不起！！赠送%s两不足~~\n附带：（%s两手续费用）", hxSilverText(in.Amount), hxSilverText(fee)))
		return
	}
	var t model.HxxyPlayer
	if err := h.DB.First(&t, in.ToID).Error; err != nil {
		resp.ParamError(c, "对方ID有误~~请重新赠送")
		return
	}
	h.hxWallet(p, "money", -(in.Amount+fee), "赠银给"+t.Name)
	h.hxWallet(&t, "money", in.Amount, "收到"+p.Name+"赠银")
	h.hxNotify(t.ID, "打赏了你"+hxSilverText(in.Amount)+"银两，快去抱抱大佬大腿求包养吧！！")
	resp.OK(c, gin.H{"msg": fmt.Sprintf("你成功赠送%s给了%s(%d)\n附带：（%s两手续费用）", hxSilverText(in.Amount), t.Name, t.ID, hxSilverText(fee))})
}

// GiveItem 赠物（复刻 xy538→xy547：从行囊选物品按数量赠送，绑定物品不可赠）
func (h *HxxyHandler) GiveItem(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		ToID  uint `json:"to_id"`
		BagID uint `json:"bag_id"`
		Count int  `json:"count"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.ToID == 0 || in.BagID == 0 {
		resp.ParamError(c, "输入有误，或者不能为空")
		return
	}
	var t model.HxxyPlayer
	if err := h.DB.First(&t, in.ToID).Error; err != nil {
		resp.ParamError(c, "对方ID有误~~请重新赠送")
		return
	}
	var b model.HxxyBag
	if err := h.DB.Where("id = ? AND player_id = ? AND store = 0", in.BagID, p.ID).First(&b).Error; err != nil {
		resp.ParamError(c, "物品不存在")
		return
	}
	if b.Bind == 1 {
		resp.ParamError(c, "对不起！绑定物品不能赠送")
		return
	}
	if in.Count <= 0 {
		resp.ParamError(c, "输入有误，或者不能为空")
		return
	}
	if in.Count > b.Count {
		in.Count = b.Count
	}
	name := ""
	if b.Kind == "item" {
		var it model.HxxyItem
		h.DB.First(&it, b.RefID)
		name = it.Name
	} else {
		var e model.HxxyEquip
		h.DB.First(&e, b.RefID)
		name = e.Name
	}
	if b.Count == in.Count {
		h.DB.Delete(&model.HxxyBag{}, b.ID)
	} else {
		h.DB.Model(&model.HxxyBag{}).Where("id = ?", b.ID).Update("count", b.Count-in.Count)
	}
	h.hxBagAdd(&t, b.Kind, b.RefID, in.Count, b.Bind)
	h.hxNotify(t.ID, fmt.Sprintf("赠送了你%sx%d，快去看看吧！", name, in.Count))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("你成功赠送%sx%d给了%s(%d)", name, in.Count, t.Name, t.ID)})
}
