package handler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 好友/聊天/帮派/结婚/住宅/摆摊

// hxNameByID 玩家名
func (h *HxxyHandler) hxNameByID(id uint) string {
	var n string
	h.DB.Model(&model.HxxyPlayer{}).Select("name").Where("id = ?", id).Scan(&n)
	return n
}

// Friends 好友/黑名单列表（复刻 xy114 好友页 + xy116 黑名单页：单向关系，status 1好友 2黑名单）
func (h *HxxyHandler) Friends(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var fs []model.HxxyFriend
	h.DB.Where("player_id = ?", p.ID).Order("id").Find(&fs)
	friends, blacks := []gin.H{}, []gin.H{}
	for _, f := range fs {
		name := h.hxNameByID(f.FriendID)
		if f.Status == 2 {
			blacks = append(blacks, gin.H{"player_id": f.FriendID, "name": name})
		} else {
			friends = append(friends, gin.H{"player_id": f.FriendID, "name": name})
		}
	}
	resp.OK(c, gin.H{"friends": friends, "blacks": blacks})
}

// FriendAdd 加为好友（复刻 xy100：单向直接成为好友，无申请同意流程）
func (h *HxxyHandler) FriendAdd(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		PlayerID uint   `json:"player_id"`
		Name     string `json:"name"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.PlayerID == 0 && in.Name != "" {
		h.DB.Model(&model.HxxyPlayer{}).Select("id").Where("name = ?", in.Name).Scan(&in.PlayerID)
	}
	if in.PlayerID == 0 || in.PlayerID == p.ID {
		resp.ParamError(c, "找不到该玩家")
		return
	}
	var t model.HxxyPlayer
	if err := h.DB.First(&t, in.PlayerID).Error; err != nil {
		resp.ParamError(c, "找不到该玩家")
		return
	}
	var f model.HxxyFriend
	if err := h.DB.Where("player_id = ? AND friend_id = ?", p.ID, t.ID).First(&f).Error; err == nil {
		if f.Status == 2 {
			resp.ParamError(c, fmt.Sprintf("对不起！玩家：%s在你的黑名单内需要移除后才能加友", t.Name))
		} else {
			resp.ParamError(c, fmt.Sprintf("对不起！玩家：%s已经是你的好友了", t.Name))
		}
		return
	}
	h.DB.Create(&model.HxxyFriend{PlayerID: p.ID, FriendID: t.ID, Status: 1})
	resp.OK(c, gin.H{"msg": fmt.Sprintf("恭喜你！你和%s成为了好友", t.Name)})
}

// FriendBlack 拉入黑名单（复刻 xy104：好友则降为黑名单，非好友直接入黑名单）
func (h *HxxyHandler) FriendBlack(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		PlayerID uint `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.PlayerID == 0 || in.PlayerID == p.ID {
		resp.ParamError(c, "参数错误")
		return
	}
	var t model.HxxyPlayer
	if err := h.DB.First(&t, in.PlayerID).Error; err != nil {
		resp.ParamError(c, "找不到该玩家")
		return
	}
	var f model.HxxyFriend
	if err := h.DB.Where("player_id = ? AND friend_id = ?", p.ID, t.ID).First(&f).Error; err == nil {
		if f.Status == 2 {
			resp.ParamError(c, fmt.Sprintf("对不起！玩家：%s已经在你的黑名单内", t.Name))
			return
		}
		h.DB.Model(&model.HxxyFriend{}).Where("id = ?", f.ID).Update("status", 2)
	} else {
		h.DB.Create(&model.HxxyFriend{PlayerID: p.ID, FriendID: t.ID, Status: 2})
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("恭喜你！成功将%s拉进了黑名单", t.Name)})
}

// FriendRemove 删除好友/移出黑名单（复刻 xy115/xy117：直接删除记录，按原状态区分文案）
func (h *HxxyHandler) FriendRemove(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		PlayerID uint `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.PlayerID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var f model.HxxyFriend
	if err := h.DB.Where("player_id = ? AND friend_id = ?", p.ID, in.PlayerID).First(&f).Error; err != nil {
		resp.ParamError(c, "该玩家不在你的列表内")
		return
	}
	name := h.hxNameByID(f.FriendID)
	h.DB.Delete(&f)
	if f.Status == 2 {
		resp.OK(c, gin.H{"msg": fmt.Sprintf("你将：%s移除了黑名单", name)})
	} else {
		resp.OK(c, gin.H{"msg": fmt.Sprintf("你删除了好友：%s", name)})
	}
}

// ChatList 世界聊天（最近 50 条）
func (h *HxxyHandler) ChatList(c *gin.Context) {
	var chats []model.HxxyChat
	h.DB.Order("id DESC").Limit(50).Find(&chats)
	// 反转为正序
	for i, j := 0, len(chats)-1; i < j; i, j = i+1, j-1 {
		chats[i], chats[j] = chats[j], chats[i]
	}
	resp.OK(c, gin.H{"chats": chats})
}

// ChatPost 发言
func (h *HxxyHandler) ChatPost(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	// 管理员禁言检查
	if p.MuteUntil > time.Now().Unix() {
		resp.ParamError(c, "你已被禁言，无法发言")
		return
	}
	var in struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	in.Content = strings.TrimSpace(in.Content)
	if in.Content == "" {
		resp.ParamError(c, "不能发送空消息")
		return
	}
	if len([]rune(in.Content)) > 100 {
		in.Content = trimStr(in.Content, 100)
	}
	h.DB.Create(&model.HxxyChat{PlayerID: p.ID, Name: p.Name, Content: in.Content})
	resp.OK(c, gin.H{"msg": "发言成功"})
}

// hxNotify 写玩家系统消息（首页消息区展示，复刻原版 [系统] 动态）
func (h *HxxyHandler) hxNotify(playerID uint, content string) {
	if playerID == 0 || content == "" {
		return
	}
	content = trimStr(content, 200)
	h.DB.Create(&model.HxxyMsg{PlayerID: playerID, FromName: "系统", Kind: "sys", Content: content})
}

// hxWorldMsg 全服公告（复刻原版 msgg02.php：写给所有玩家）
func (h *HxxyHandler) hxWorldMsg(content string) {
	if content == "" {
		return
	}
	content = trimStr(content, 200)
	var ids []uint
	h.DB.Model(&model.HxxyPlayer{}).Pluck("id", &ids)
	for _, id := range ids {
		h.DB.Create(&model.HxxyMsg{PlayerID: id, FromName: "系统", Kind: "sys", Content: content})
	}
}

// Home 首页数据（未读消息 + 同地地图附近玩家，复刻原版 xy002.php 消息区与附近玩家）
func (h *HxxyHandler) Home(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var msgs []model.HxxyMsg
	h.DB.Where("player_id = ? AND is_read = 0", p.ID).Order("id ASC").Limit(10).Find(&msgs)
	if len(msgs) > 0 {
		h.DB.Model(&model.HxxyMsg{}).Where("player_id = ? AND is_read = 0", p.ID).Update("is_read", 1)
	}
	// 附近玩家（同地图同节点）
	var near []model.HxxyPlayer
	h.DB.Where("map_x = ? AND map_y = ? AND id != ?", p.MapX, p.MapY, p.ID).Order("id").Limit(8).Find(&near)
	list := []gin.H{}
	for _, n := range near {
		// 复刻 fjwj.php：附近玩家显示 名字【国家】（职务）
		gangName, gangRole := "", ""
		np := n
		if g, mem := h.hxGangOf(&np); g != nil && mem != nil {
			gangName = g.Name
			gangRole = hxGangRoleNames[mem.Role]
		}
		list = append(list, gin.H{"player_id": n.ID, "name": n.Name, "level": n.Level,
			"sect_name": hxSectNames[n.Sect], "gang_name": gangName, "gang_role": gangRole, "vip_lv": n.VipLv})
	}
	// 组队邀请（复刻原版 yq1.php：邀请直接显示在首页）
	var invites []model.HxxyTeamInvite
	h.DB.Where("to_id = ? AND status = 0", p.ID).Order("id DESC").Limit(5).Find(&invites)
	inv := []gin.H{}
	for _, iv := range invites {
		inv = append(inv, gin.H{"id": iv.ID, "from_id": iv.FromID, "from_name": iv.FromName})
	}
	// 国家(帮派)邀请（复刻原版 yq2.php）
	var ginvs []model.HxxyGangInvite
	h.DB.Where("to_id = ? AND status = 0", p.ID).Order("id DESC").Limit(3).Find(&ginvs)
	glist := []gin.H{}
	for _, iv := range ginvs {
		glist = append(glist, gin.H{"id": iv.ID, "gang_id": iv.GangID, "gang_name": iv.GangName, "from_id": iv.FromID, "from_name": iv.FromName})
	}
	// 住宅参观邀请（复刻原版 yq3.php）
	var hinvs []model.HxxyHouseInvite
	h.DB.Where("to_id = ? AND status = 0", p.ID).Order("id DESC").Limit(3).Find(&hinvs)
	hlist := []gin.H{}
	for _, iv := range hinvs {
		hlist = append(hlist, gin.H{"id": iv.ID, "from_id": iv.FromID, "from_name": iv.FromName})
	}
	// 结婚邀请（复刻原版 yq4.php：求婚直接显示在首页）
	var marriageInvite gin.H
	var mi model.HxxyMarriage
	if err := h.DB.Where("player_b = ? AND status = 1", p.ID).Order("id DESC").First(&mi).Error; err == nil {
		marriageInvite = gin.H{"id": mi.ID, "from_id": mi.PlayerA, "from_name": h.hxNameByID(mi.PlayerA)}
	}
	// 导航动态标红：未签到 / 有可提交任务
	var questReady int64
	h.DB.Model(&model.HxxyPlayerQuest{}).Where("player_id = ? AND status = 2", p.ID).Count(&questReady)
	resp.OK(c, gin.H{"msgs": msgs, "nearby": list, "notices": hxStaticNotices(), "team_invites": inv,
		"gang_invites": glist, "house_invites": hlist, "marriage_invite": marriageInvite,
		"today_signed": p.DaySignin > 0, "quest_ready": questReady})
}

// hxStaticNotices 首页定时活动公告（复刻原版 msgg03.php：按当前时间显示活动预告/进行中红字）
func hxStaticNotices() []string {
	now := time.Now()
	hh, mm := now.Hour(), now.Minute()
	var out []string
	add := func(s string) { out = append(out, s) }
	zcName := func() string {
		switch time.Now().Weekday() {
		case 1:
			return "傲来国"
		case 2:
			return "宝象国"
		case 3:
			return "乌鸡国"
		case 4:
			return "女儿国"
		case 5:
			return "车迟国"
		case 0:
			return "祭赛国"
		}
		return ""
	}()
	switch {
	// 天降秘宝·长安场 21:00-22:25
	case hh == 21 && mm < 30:
		add("爱的玩家【第一波天降秘宝】将在【半小时】后火热开启！！请做好准备哦~~")
	case hh == 21 && mm >= 30 && mm < 40:
		add("【第一波天降秘宝】（持续10分钟）火热开启中！！请前往长安各个主街道寻找秘宝吧~~~")
	case hh == 21 && mm == 40:
		add("爱的玩家【第二波天降秘宝】将在【10分钟】后火热开启！！请做好准备哦~~")
	case hh == 21 && mm >= 50:
		add("【第二波天降秘宝】（持续10分钟）火热开启中！！请前往长安各个主街道寻找秘宝吧~~~")
	case hh == 22 && mm < 10:
		add("爱的玩家【第三波天降秘宝】将在【" + fmtMin(10-mm) + "】后火热开启！！秘宝将在长安主街道随机掉落有缘人皆可得之~~")
	case hh == 22 && mm >= 10 && mm < 20:
		add("【第三波天降秘宝】（持续10分钟）火热开启中！！请前往长安各个主街道寻找秘宝吧~~~")
	case hh == 22 && mm >= 20 && mm <= 25:
		add("【天降秘宝】活动已结束！！请明天再来哦~~~~~~")
	// 天降秘宝·天宫场 12:00-12:50
	case hh == 11 && mm >= 30:
		add("爱的玩家【第一波天降秘宝】将在【" + fmtMin(60-mm) + "】后在天宫火热开启！！请做好准备哦~~")
	case hh == 12 && mm < 20:
		add("【第一波天降秘宝】（持续10分钟）火热开启中！！请前往天宫各个主街道寻找秘宝吧~~~")
	case hh == 12 && mm >= 20 && mm < 30:
		add("【第二波天降秘宝】（持续10分钟）火热开启中！！请前往天宫主街道寻找秘宝吧~~~")
	case hh == 12 && mm >= 40 && mm < 50:
		add("【第三波天降秘宝】（持续10分钟）火热开启中！！请前往天宫街道寻找秘宝吧~~~")
	case hh == 13 && mm >= 50:
		add("【天降秘宝】活动已结束！！请明天再来哦~~~~~~")
	// 采花大盗 15:00-15:30
	case hh == 14 && mm >= 50:
		add("爱的玩家【采花大盗】将在【" + fmtMin(60-mm) + "】后火热开启！！请做好准备哦~~位置在玄武大街百花仙子处")
	case hh == 15 && mm < 30:
		add("【采花大盗】活动正在火热进行中，谁会是今天的花花公子了？")
	// 幸运女神与财神 11:00-11:10
	case hh == 10 && mm >= 50:
		add("幸运女神和财神爷将在11:00-11:10降临到金山当中，把握机会~~机不在失，失不再来哦~~")
	case hh == 11 && mm <= 10:
		add("幸运女神和财神时间火热开启中11:11溜走哦（金山挖宝有几率触发幸运女神与财神奖励丰厚）")
	// 国战 20:55 预告 / 21:00 开启
	case hh == 20 && mm >= 55 && zcName != "":
		add("【" + zcName + "】国战将于21:00开始,请还没报名的国家前往长安城-封榜堂,大宰相.房玄龄处进行报名")
	case hh == 21 && mm == 0 && zcName != "":
		add("【" + zcName + "】国战已经火热开启了,请各位仙友前往战场争夺属于国家的荣誉吧！")
	// 数据备份提醒 23:45-23:49
	case hh == 23 && mm >= 45 && mm <= 49:
		add("(づ￣3￣)づ╭❤～爱的玩家为了保证玩家的数据安全，服务器【" + fmtMin(50-mm) + "】后即将进入玩家数据备份！！给您造成的不便请谅解~~")
	}
	return out
}

func fmtMin(m int) string {
	if m >= 30 {
		return "半小时"
	}
	return strconv.Itoa(m) + "分钟"
}

// PMSend 私聊发送（写入对方消息区，复刻原版 [私聊]）
func (h *HxxyHandler) PMSend(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	if p.MuteUntil > time.Now().Unix() {
		resp.ParamError(c, "你已被禁言，无法私聊")
		return
	}
	var in struct {
		ToID    uint   `json:"to_id"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.ToID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.ToID == p.ID {
		resp.ParamError(c, "不能给自己发私聊")
		return
	}
	in.Content = strings.TrimSpace(in.Content)
	if in.Content == "" {
		resp.ParamError(c, "不能发送空消息")
		return
	}
	var to model.HxxyPlayer
	if err := h.DB.First(&to, in.ToID).Error; err != nil {
		resp.ParamError(c, "没有这个玩家")
		return
	}
	h.DB.Create(&model.HxxyMsg{PlayerID: to.ID, FromID: p.ID, FromName: p.Name, Kind: "pv",
		Content: trimStr(in.Content, 100)})
	resp.OK(c, gin.H{"msg": "私聊消息已发送"})
}

// PMList 私聊记录（与某人的双向消息，打开即读）
func (h *HxxyHandler) PMList(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	var to model.HxxyPlayer
	if err := h.DB.First(&to, id).Error; err != nil {
		resp.ParamError(c, "没有这个玩家")
		return
	}
	var msgs []model.HxxyMsg
	h.DB.Where("(player_id = ? AND from_id = ?) OR (player_id = ? AND from_id = ?)",
		p.ID, id, id, p.ID).Order("id ASC").Limit(50).Find(&msgs)
	// 对方发给我的标已读
	h.DB.Model(&model.HxxyMsg{}).
		Where("player_id = ? AND from_id = ? AND kind = 'pv' AND is_read = 0", p.ID, id).Update("is_read", 1)
	resp.OK(c, gin.H{
		"player": gin.H{"player_id": to.ID, "name": to.Name, "level": to.Level, "sect_name": hxSectNames[to.Sect]},
		"msgs":   msgs,
	})
}

// ================= 国家系统（复刻原版 all_bp：xy171 创建 / xy172 主页 / xy175 成员 / xy176-179 任命罢免 /
// xy182 退出 / xy184 踢出 / xy185 捐献 / xy186 国家商城 / xy595 升级 / xy605 兑换）=================

// hxGangRoleNames 职务表（复刻原版：0成员 1君主 2辅助 3军机 4财政 5工部 6外交 7军团长）
var hxGangRoleNames = map[int]string{
	0: "成员", 1: "君主", 2: "辅助大臣", 3: "军机大臣", 4: "财政大臣", 5: "工部大臣", 6: "外交大臣", 7: "军团长",
}

func hxGangRoleName(role int) string {
	if n, ok := hxGangRoleNames[role]; ok {
		return n
	}
	return "成员"
}

// hxGangUpReq 升级消耗（xy595.php：当前等级→[国家资金,国家经验,国家声望]）
func hxGangUpReq(level int) (int64, int64, int64) {
	switch level {
	case 1:
		return 10000000000, 5000, 5000
	case 2:
		return 20000000000, 10000, 10000
	case 3:
		return 30000000000, 15000, 15000
	case 4:
		return 40000000000, 20000, 20000
	case 5:
		return 50000000000, 20000, 20000
	case 6:
		return 60000000000, 25000, 250000
	case 7:
		return 70000000000, 30000, 30000
	case 8:
		return 80000000000, 50000, 50000
	case 9:
		return 99999999999, 100000, 100000
	}
	return 0, 0, 0 // 10级封顶
}

// hxGangUpCap 升级后上限（xy595.php：新等级→[人数上限,经验上限]）
func hxGangUpCap(level int) (int, int64) {
	switch level {
	case 2:
		return 40, 10000
	case 3:
		return 60, 15000
	case 4:
		return 80, 20000
	case 5:
		return 100, 25000
	case 6:
		return 150, 25000
	case 7:
		return 200, 30000
	case 8:
		return 300, 50000
	case 9:
		return 400, 100000
	case 10:
		return 500, 150000
	}
	return 20, 5000
}

// hxGangMallItem 国家商城商品（xy605.php 35件：商城等级/物品id/所需贡献/所需银两）
type hxGangMallItem struct {
	Level        int
	ItemID       uint
	Contribution int
	Silver       int64
}

var hxGangMall = []hxGangMallItem{
	// 1级商城（xy186）
	{1, 5, 100, 100000000}, {1, 162, 200, 200000000}, {1, 163, 200, 200000000},
	{1, 164, 200, 200000000}, {1, 168, 200, 200000000}, {1, 169, 200, 200000000}, {1, 170, 200, 200000000},
	// 2级商城（xy596）
	{2, 302, 50, 100000000}, {2, 303, 50, 100000000}, {2, 314, 50, 100000000}, {2, 398, 50, 100000000},
	// 3级商城（xy597）
	{3, 304, 100, 200000000}, {3, 315, 100, 200000000}, {3, 399, 100, 200000000},
	// 4级商城（xy598）
	{4, 305, 150, 200000000}, {4, 316, 200, 400000000}, {4, 126, 100, 200000000},
	// 5级商城（xy599）
	{5, 306, 200, 400000000}, {5, 317, 500, 1000000000}, {5, 307, 200, 500000000},
	// 6级商城（xy600）
	{6, 175, 500, 500000000}, {6, 318, 1000, 5000000000}, {6, 401, 100, 100000000},
	// 7级商城（xy601）
	{7, 400, 500, 500000000}, {7, 319, 3000, 10000000000}, {7, 423, 50, 100000000},
	// 8级商城（xy602）
	{8, 427, 200, 500000000}, {8, 454, 1000, 1000000000}, {8, 494, 100, 200000000},
	// 9级商城（xy603）
	{9, 455, 3000, 5000000000}, {9, 625, 1000, 100000000}, {9, 128, 1000, 100000000},
	// 10级商城（xy604）
	{10, 626, 3000, 300000000}, {10, 627, 6000, 600000000}, {10, 127, 1000, 100000000},
}

// hxSilverText 银两转汉字（复刻 wp/ylxx.php：X亿X万X两）
func hxSilverText(v int64) string {
	if v <= 0 {
		return "0两"
	}
	s := strconv.FormatInt(v, 10)
	n := len(s)
	var y, w, l int64
	if n >= 9 {
		y, _ = strconv.ParseInt(s[:n-8], 10, 64)
		w, _ = strconv.ParseInt(s[n-8:n-4], 10, 64)
		l, _ = strconv.ParseInt(s[n-4:], 10, 64)
	} else if n >= 5 {
		w, _ = strconv.ParseInt(s[:n-4], 10, 64)
		l, _ = strconv.ParseInt(s[n-4:], 10, 64)
	} else {
		l = v
	}
	out := ""
	if y > 0 {
		out += strconv.FormatInt(y, 10) + "亿"
	}
	if w > 0 {
		out += strconv.FormatInt(w, 10) + "万"
	}
	if l > 0 {
		out += strconv.FormatInt(l, 10)
	}
	return out + "两"
}

// hxGangOf 玩家的国家与成员记录（无国家返回 nil, nil）
func (h *HxxyHandler) hxGangOf(p *model.HxxyPlayer) (*model.HxxyGang, *model.HxxyGangMember) {
	var mem model.HxxyGangMember
	if err := h.DB.Where("player_id = ?", p.ID).First(&mem).Error; err != nil {
		return nil, nil
	}
	var g model.HxxyGang
	if err := h.DB.First(&g, mem.GangID).Error; err != nil {
		return nil, nil
	}
	return &g, &mem
}

// hxGangCount 国家成员数
func (h *HxxyHandler) hxGangCount(gangID uint) int64 {
	var cnt int64
	h.DB.Model(&model.HxxyGangMember{}).Where("gang_id = ?", gangID).Count(&cnt)
	return cnt
}

// hxGangReward 国家奖励（复刻 yxpz/gjgx_pz.php：个人贡献+国家经验（受上限）+国家声望）
func (h *HxxyHandler) hxGangReward(p *model.HxxyPlayer, contrib, exp, sw int64) {
	g, mem := h.hxGangOf(p)
	if g == nil || mem == nil {
		return
	}
	if contrib > 0 {
		h.DB.Model(&model.HxxyGangMember{}).Where("id = ?", mem.ID).Updates(map[string]interface{}{
			"contribution": gorm.Expr("contribution + ?", contrib),
			"total_contribution": gorm.Expr("total_contribution + ?", contrib)})
	}
	if exp > 0 {
		add := exp
		if g.Exp+add > g.ExpMax {
			add = g.ExpMax - g.Exp // 达上限后不再累积，需升级国家
		}
		if add > 0 {
			h.DB.Model(&model.HxxyGang{}).Where("id = ?", g.ID).UpdateColumn("exp", gorm.Expr("exp + ?", add))
		}
	}
	if sw > 0 {
		h.DB.Model(&model.HxxyGang{}).Where("id = ?", g.ID).UpdateColumn("sw", gorm.Expr("sw + ?", sw))
	}
}

// hxMallItemName 商城物品名（物品表优先，其次装备表）
func (h *HxxyHandler) hxMallItemName(refID uint) string {
	var it model.HxxyItem
	if err := h.DB.Select("name").First(&it, refID).Error; err == nil {
		return it.Name
	}
	var e model.HxxyEquip
	if err := h.DB.Select("name").First(&e, refID).Error; err == nil {
		return e.Name
	}
	return fmt.Sprintf("物品#%d", refID)
}

// hxGangBrief 国家信息序列化（主页/成员列表共用）
func (h *HxxyHandler) hxGangBrief(g *model.HxxyGang, mem *model.HxxyGangMember) gin.H {
	var ms []model.HxxyGangMember
	h.DB.Where("gang_id = ?", g.ID).Order("role DESC, total_contribution DESC").Find(&ms)
	members := []gin.H{}
	officials := gin.H{}
	for _, m := range ms {
		name := h.hxNameByID(m.PlayerID)
		members = append(members, gin.H{
			"player_id": m.PlayerID, "name": name, "role": m.Role, "role_name": hxGangRoleName(m.Role),
			"contribution": m.Contribution, "total_contribution": m.TotalContribution})
		if m.Role >= 2 {
			officials[fmt.Sprintf("role%d", m.Role)] = gin.H{"player_id": m.PlayerID, "name": name}
		}
	}
	roleName := ""
	if mem != nil {
		roleName = hxGangRoleName(mem.Role)
	}
	return gin.H{
		"gang_id": g.ID, "name": g.Name, "level": g.Level,
		"founder_name": g.FounderName, "leader_id": g.LeaderID, "leader_name": h.hxNameByID(g.LeaderID),
		"member_count": len(members), "member_max": g.MemberMax,
		"exp": g.Exp, "exp_max": g.ExpMax, "money": g.Money, "sw": g.Sw,
		"officials": officials, "members": members,
		"role": mem.Role, "role_name": roleName,
		"contribution": mem.Contribution, "total_contribution": mem.TotalContribution,
		"is_monarch": mem != nil && mem.Role == 1,
		"can_manage": mem != nil && (mem.Role == 1 || mem.Role == 2), // 君主/辅助可罢免官职
		"can_invite": mem != nil && (mem.Role == 1 || mem.Role == 2),
	}
}

// GangInfo 国家信息（复刻 xy172 主页：国家资料+成员+权限 + 国家列表）
func (h *HxxyHandler) GangInfo(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	myGang := gin.H{}
	g, mem := h.hxGangOf(p)
	if g != nil && mem != nil {
		myGang = h.hxGangBrief(g, mem)
	}
	var gs []model.HxxyGang
	h.DB.Order("level DESC, id ASC").Limit(20).Find(&gs)
	list := []gin.H{}
	for _, gg := range gs {
		list = append(list, gin.H{"gang_id": gg.ID, "name": gg.Name, "level": gg.Level,
			"member_count": h.hxGangCount(gg.ID), "member_max": gg.MemberMax})
	}
	resp.OK(c, gin.H{"my_gang": myGang, "gangs": list})
}

// GangCreate 创建国家（复刻 xy171/jlbp.php：国家名≤7字，需1亿银两+玄铁令x5）
func (h *HxxyHandler) GangCreate(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Name) == "" {
		resp.ParamError(c, "国家名不能为空")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if n := len([]rune(in.Name)); n > 7 {
		resp.ParamError(c, "国家名长度不能超过限制")
		return
	}
	if _, mem := h.hxGangOf(p); mem != nil {
		resp.ParamError(c, "对不起！你已经有国家了无法创建")
		return
	}
	var cnt int64
	h.DB.Model(&model.HxxyGang{}).Where("name = ?", in.Name).Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "你想要创建的"+in.Name+",已经存在请换个国家名字吧")
		return
	}
	const cost = 100000000 // 1亿银两
	if p.Money < cost {
		resp.ParamError(c, "对不起！建立国家需要银两1亿和玄铁令x5")
		return
	}
	// 玄铁令x5（物品266，跨多行统计）
	var tokens []model.HxxyBag
	h.DB.Where("player_id = ? AND kind = 'item' AND ref_id = 266 AND store = 0", p.ID).Order("id ASC").Find(&tokens)
	total := 0
	for _, b := range tokens {
		total += b.Count
	}
	if total < 5 {
		resp.ParamError(c, "对不起！建立国家需要银两1亿和玄铁令x5")
		return
	}
	need := 5
	for _, b := range tokens {
		if need <= 0 {
			break
		}
		take := b.Count
		if take > need {
			take = need
		}
		h.hxBagSub(p.ID, b.ID, take)
		need -= take
	}
	g := model.HxxyGang{Name: in.Name, Level: 1, FounderID: p.ID, FounderName: p.Name,
		LeaderID: p.ID, MemberMax: 20, ExpMax: 5000}
	h.DB.Create(&g)
	h.hxWallet(p, "money", -cost, "创建国家【"+in.Name+"】")
	h.DB.Create(&model.HxxyGangMember{GangID: g.ID, PlayerID: p.ID, Role: 1})
	resp.OK(c, gin.H{"msg": "恭喜你创建了国家" + in.Name})
}

// GangJoin 加入国家（列表直接加入，人数满则拒绝）
func (h *HxxyHandler) GangJoin(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		GangID uint `json:"gang_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.GangID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	if _, mem := h.hxGangOf(p); mem != nil {
		resp.ParamError(c, "你已有国家")
		return
	}
	var g model.HxxyGang
	if err := h.DB.First(&g, in.GangID).Error; err != nil {
		resp.ParamError(c, "国家不存在")
		return
	}
	if h.hxGangCount(g.ID) >= int64(g.MemberMax) {
		resp.ParamError(c, "该国家人数已满！")
		return
	}
	h.DB.Create(&model.HxxyGangMember{GangID: g.ID, PlayerID: p.ID})
	resp.OK(c, gin.H{"msg": fmt.Sprintf("欢迎加入国家【%s】！", g.Name)})
}

// GangLeave 退出国家（原版 xy182：君主无退出入口，仅成员/官员可退）
func (h *HxxyHandler) GangLeave(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	g, mem := h.hxGangOf(p)
	if g == nil || mem == nil {
		resp.ParamError(c, "你还未加入任何国家！！")
		return
	}
	if mem.Role == 1 {
		resp.ParamError(c, "现任君主无法退出国家！可解散国家")
		return
	}
	h.DB.Delete(&model.HxxyGangMember{}, mem.ID)
	resp.OK(c, gin.H{"msg": "你退出了" + g.Name}) // 复刻 xy340 文案
}

// GangDonate 捐献银两（复刻 wj/gjjx.php：单笔100万~100亿，100万银两=1贡献）
func (h *HxxyHandler) GangDonate(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		Amount int64 `json:"amount"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Amount <= 0 {
		resp.ParamError(c, "输入有误请重新输入")
		return
	}
	g, mem := h.hxGangOf(p)
	if g == nil || mem == nil {
		resp.ParamError(c, "你还未加入任何国家！！")
		return
	}
	if in.Amount < 1000000 {
		resp.ParamError(c, "每次最少捐献100万以上的银两哦")
		return
	}
	if in.Amount > 10000000000 {
		resp.ParamError(c, "每次最多捐献100亿以下银两哦")
		return
	}
	if p.Money < in.Amount {
		resp.ParamError(c, "你的银两不足不能进行捐献")
		return
	}
	contrib := in.Amount / 1000000
	h.hxWallet(p, "money", -in.Amount, "捐献国家【"+g.Name+"】")
	h.DB.Model(&model.HxxyGang{}).Where("id = ?", g.ID).UpdateColumn("money", gorm.Expr("money + ?", in.Amount))
	h.DB.Model(&model.HxxyGangMember{}).Where("id = ?", mem.ID).Updates(map[string]interface{}{
		"contribution": gorm.Expr("contribution + ?", contrib),
		"total_contribution": gorm.Expr("total_contribution + ?", contrib)})
	// 复刻 gjjx.php：恭喜你!为国家捐赠了X亿X万X两,获得国家贡献N点
	resp.OK(c, gin.H{"msg": fmt.Sprintf("恭喜你!为国家捐赠了%s,获得国家贡献%d点", hxSilverText(in.Amount), contrib)})
}

// GangMall 国家商城（复刻 xy186：1~10级商城页签，仅展示已达等级商品可兑换）
func (h *HxxyHandler) GangMall(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	g, mem := h.hxGangOf(p)
	if g == nil || mem == nil {
		resp.ParamError(c, "你还未加入任何国家！！")
		return
	}
	tabs := []gin.H{}
	for lv := 1; lv <= 10; lv++ {
		items := []gin.H{}
		for _, mi := range hxGangMall {
			if mi.Level == lv {
				items = append(items, gin.H{"item_id": mi.ItemID, "name": h.hxMallItemName(mi.ItemID),
					"contribution": mi.Contribution, "silver": mi.Silver})
			}
		}
		tabs = append(tabs, gin.H{"level": lv, "unlocked": g.Level >= lv, "items": items})
	}
	resp.OK(c, gin.H{"name": g.Name, "level": g.Level, "contribution": mem.Contribution, "tabs": tabs})
}

// GangMallBuy 国家商城兑换（复刻 xy605：扣贡献+银两，物品入包）
func (h *HxxyHandler) GangMallBuy(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		ItemID uint `json:"item_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.ItemID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	g, mem := h.hxGangOf(p)
	if g == nil || mem == nil {
		resp.ParamError(c, "你还未加入任何国家！！")
		return
	}
	var mi *hxGangMallItem
	for i := range hxGangMall {
		if hxGangMall[i].ItemID == in.ItemID {
			mi = &hxGangMall[i]
			break
		}
	}
	if mi == nil {
		resp.ParamError(c, "该商品不存在")
		return
	}
	if g.Level < mi.Level {
		resp.ParamError(c, fmt.Sprintf("该商品需要%d级国家商城才能兑换！", mi.Level))
		return
	}
	if mem.Contribution < mi.Contribution {
		resp.ParamError(c, "对不起！！你的国家贡献不足！！")
		return
	}
	if p.Money < mi.Silver {
		resp.ParamError(c, "对不起！！你的银两不足！！")
		return
	}
	name := h.hxMallItemName(mi.ItemID)
	h.hxWallet(p, "money", -mi.Silver, "国家商城兑换【"+name+"】")
	h.DB.Model(&model.HxxyGangMember{}).Where("id = ?", mem.ID).UpdateColumn("contribution", gorm.Expr("contribution - ?", mi.Contribution))
	h.hxBagAdd(p, "item", mi.ItemID, 1, 1)
	// 复刻 xy605 文案
	resp.OK(c, gin.H{"msg": fmt.Sprintf("恭喜你！！兑换成功！！失去%d点国家贡献~~~", mi.Contribution)})
}

// GangAppoint 任命官员（复刻 xy176-178：仅君主，仅可任命成员，职务2-7各一名）
func (h *HxxyHandler) GangAppoint(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		PlayerID uint `json:"player_id"`
		Role     int  `json:"role"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.PlayerID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.Role < 2 || in.Role > 7 {
		resp.ParamError(c, "职务不合法")
		return
	}
	g, mem := h.hxGangOf(p)
	if g == nil || mem == nil {
		resp.ParamError(c, "你还未加入任何国家！！")
		return
	}
	if mem.Role != 1 {
		resp.ParamError(c, "只有君主才能任命官员！")
		return
	}
	var target model.HxxyGangMember
	if err := h.DB.Where("player_id = ? AND gang_id = ?", in.PlayerID, g.ID).First(&target).Error; err != nil {
		resp.ParamError(c, "对方不是你的国家成员")
		return
	}
	if target.Role != 0 {
		resp.ParamError(c, "该玩家已有职务，请先罢免原职务")
		return
	}
	h.DB.Model(&model.HxxyGangMember{}).Where("id = ?", target.ID).Update("role", in.Role)
	roleName := hxGangRoleName(in.Role)
	h.hxNotify(in.PlayerID, fmt.Sprintf("【%s】君主任命你为【%s】！", g.Name, roleName))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("任命成功！【%s】已成为【%s】", h.hxNameByID(in.PlayerID), roleName)})
}

// GangDismiss 罢免官职（复刻 xy179：君主/辅助大臣可罢免2-7职务，不可动君主）
func (h *HxxyHandler) GangDismiss(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		PlayerID uint `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.PlayerID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	g, mem := h.hxGangOf(p)
	if g == nil || mem == nil {
		resp.ParamError(c, "你还未加入任何国家！！")
		return
	}
	if mem.Role != 1 && mem.Role != 2 {
		resp.ParamError(c, "只有君主或辅助大臣才能罢免官职！")
		return
	}
	var target model.HxxyGangMember
	if err := h.DB.Where("player_id = ? AND gang_id = ?", in.PlayerID, g.ID).First(&target).Error; err != nil {
		resp.ParamError(c, "对方不是你的国家成员")
		return
	}
	if target.Role == 1 {
		resp.ParamError(c, "不能罢免君主！")
		return
	}
	if target.Role == 0 {
		resp.ParamError(c, "该玩家没有职务")
		return
	}
	if target.PlayerID == p.ID {
		resp.ParamError(c, "不能罢免自己！")
		return
	}
	old := hxGangRoleName(target.Role)
	h.DB.Model(&model.HxxyGangMember{}).Where("id = ?", target.ID).Update("role", 0)
	h.hxNotify(in.PlayerID, fmt.Sprintf("你被罢免了【%s】的职务。", old))
	// 复刻 xy179 文案
	resp.OK(c, gin.H{"msg": fmt.Sprintf("恭喜你！成功将%s罢免为【成员】", h.hxNameByID(in.PlayerID))})
}

// GangKick 踢出国家（复刻 xy184：君主/辅助可踢任意非君主，其他官员可踢非君主成员）
func (h *HxxyHandler) GangKick(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		PlayerID uint `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.PlayerID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	g, mem := h.hxGangOf(p)
	if g == nil || mem == nil {
		resp.ParamError(c, "你还未加入任何国家！！")
		return
	}
	if mem.Role < 1 {
		resp.ParamError(c, "只有官员才能踢人！")
		return
	}
	if in.PlayerID == p.ID {
		resp.ParamError(c, "不能踢自己！")
		return
	}
	var target model.HxxyGangMember
	if err := h.DB.Where("player_id = ? AND gang_id = ?", in.PlayerID, g.ID).First(&target).Error; err != nil {
		resp.ParamError(c, "对方不是你的国家成员")
		return
	}
	if target.Role == 1 {
		resp.ParamError(c, "不能踢出君主！")
		return
	}
	name := h.hxNameByID(in.PlayerID)
	h.DB.Delete(&model.HxxyGangMember{}, target.ID)
	h.hxNotify(in.PlayerID, fmt.Sprintf("你被踢出了国家【%s】。", g.Name))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("已将【%s】踢出国家！", name)})
}

// GangUpgrade 升级国家（复刻 xy595：扣国家资金/经验/声望，提升人数与经验上限）
func (h *HxxyHandler) GangUpgrade(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	g, mem := h.hxGangOf(p)
	if g == nil || mem == nil {
		resp.ParamError(c, "你还未加入任何国家！！")
		return
	}
	if g.Level >= 10 {
		resp.ParamError(c, "国家已达到最高等级！")
		return
	}
	needMoney, needExp, needSw := hxGangUpReq(g.Level)
	if g.Money < needMoney || g.Exp < needExp || g.Sw < needSw {
		resp.ParamError(c, fmt.Sprintf("对不起！！升级%s需要：国家资金%d，国家经验%d，国家声望%d",
			g.Name, needMoney, needExp, needSw))
		return
	}
	newLevel := g.Level + 1
	newMax, newExpMax := hxGangUpCap(newLevel)
	h.DB.Model(&model.HxxyGang{}).Where("id = ?", g.ID).Updates(map[string]interface{}{
		"level": newLevel, "member_max": newMax, "exp_max": newExpMax,
		"money": g.Money - needMoney, "exp": g.Exp - needExp, "sw": g.Sw - needSw})
	h.hxWorldNotice(fmt.Sprintf("【国家】恭喜！【%s】成功升级到了%d级！人口提升为%d！", g.Name, newLevel, newMax))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("恭喜你！！成功将%s升级到了%d级！！人口提升为%d人口！！", g.Name, newLevel, newMax)})
}

// GangDissolve 解散国家（复刻 xy173 确认 + xy341 执行：仅君主，且需先将子民流放，成员仅剩君主1人才能解散）
func (h *HxxyHandler) GangDissolve(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	g, mem := h.hxGangOf(p)
	if g == nil || mem == nil {
		resp.ParamError(c, "你还未加入任何国家！！")
		return
	}
	if mem.Role != 1 {
		resp.ParamError(c, "只有君主才能解散国家！")
		return
	}
	if cnt := h.hxGangCount(g.ID); cnt > 1 {
		resp.ParamError(c, "对不起！要解散国家"+g.Name+"需要将你国家的子民流放掉！！")
		return
	}
	h.DB.Where("gang_id = ?", g.ID).Delete(&model.HxxyGangMember{})
	h.DB.Delete(&model.HxxyGang{}, g.ID)
	resp.OK(c, gin.H{"msg": "你解散了" + g.Name}) // 复刻 xy341 文案
}

// GangInvite 邀请入国（复刻原版 yq2.php：君主/辅助大臣可发，邀请显示在对方首页）
func (h *HxxyHandler) GangInvite(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	g, mem := h.hxGangOf(p)
	if g == nil || mem == nil {
		resp.ParamError(c, "你还未加入任何国家！！")
		return
	}
	if mem.Role != 1 && mem.Role != 2 {
		resp.ParamError(c, "只有君主或辅助大臣才能邀请入国")
		return
	}
	var in struct {
		PlayerID uint   `json:"player_id"`
		Name     string `json:"name"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.PlayerID == 0 && in.Name != "" {
		h.DB.Model(&model.HxxyPlayer{}).Select("id").Where("name = ?", in.Name).Scan(&in.PlayerID)
	}
	if in.PlayerID == 0 || in.PlayerID == p.ID {
		resp.ParamError(c, "找不到该玩家")
		return
	}
	if otherMem := h.DB.Where("player_id = ?", in.PlayerID).First(&model.HxxyGangMember{}); otherMem.Error == nil {
		resp.ParamError(c, "对方已有国家")
		return
	}
	var cnt int64
	h.DB.Model(&model.HxxyGangInvite{}).Where("gang_id = ? AND to_id = ? AND status = 0", g.ID, in.PlayerID).Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "已邀请过该玩家，等待对方处理")
		return
	}
	h.DB.Create(&model.HxxyGangInvite{GangID: g.ID, GangName: g.Name, FromID: p.ID, FromName: p.Name, ToID: in.PlayerID, ToName: h.hxNameByID(in.PlayerID)})
	h.hxNotify(in.PlayerID, fmt.Sprintf("【%s】邀请你加入国家【%s】，请到首页处理。", p.Name, g.Name))
	resp.OK(c, gin.H{"msg": "邀请已发送！"})
}

// GangInviteAgree 同意入国邀请（首页处理，复刻原版 cmd 180 接受）
func (h *HxxyHandler) GangInviteAgree(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		ID uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.ID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var iv model.HxxyGangInvite
	if err := h.DB.Where("id = ? AND to_id = ? AND status = 0", in.ID, p.ID).First(&iv).Error; err != nil {
		resp.ParamError(c, "邀请不存在或已处理")
		return
	}
	if _, mem := h.hxGangOf(p); mem != nil {
		h.DB.Model(&model.HxxyGangInvite{}).Where("id = ?", iv.ID).Update("status", 2)
		resp.ParamError(c, "你已有国家")
		return
	}
	var g model.HxxyGang
	if err := h.DB.First(&g, iv.GangID).Error; err != nil {
		resp.ParamError(c, "该国家已不存在")
		return
	}
	if h.hxGangCount(g.ID) >= int64(g.MemberMax) {
		resp.ParamError(c, "该国家人数已满！")
		return
	}
	h.DB.Create(&model.HxxyGangMember{GangID: iv.GangID, PlayerID: p.ID})
	h.DB.Model(&model.HxxyGangInvite{}).Where("id = ?", iv.ID).Update("status", 1)
	h.hxNotify(iv.FromID, fmt.Sprintf("【%s】接受了你的邀请，已加入国家【%s】。", p.Name, iv.GangName))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("欢迎加入国家【%s】！", iv.GangName)})
}

// GangInviteRefuse 拒绝入国邀请（复刻原版 cmd 181 拒绝）
func (h *HxxyHandler) GangInviteRefuse(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		ID uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.ID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var iv model.HxxyGangInvite
	if err := h.DB.Where("id = ? AND to_id = ? AND status = 0", in.ID, p.ID).First(&iv).Error; err != nil {
		resp.ParamError(c, "邀请不存在或已处理")
		return
	}
	h.DB.Model(&model.HxxyGangInvite{}).Where("id = ?", iv.ID).Update("status", 2)
	h.hxNotify(iv.FromID, fmt.Sprintf("【%s】拒绝了你加入国家【%s】的邀请。", p.Name, iv.GangName))
	resp.OK(c, gin.H{"msg": "已拒绝邀请"})
}

// MarriageInfo 结婚信息
func (h *HxxyHandler) MarriageInfo(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var m model.HxxyMarriage
	info := gin.H{"status": 0}
	if err := h.DB.Where("(player_a = ? OR player_b = ?) AND status IN (1,2)", p.ID, p.ID).Order("id DESC").First(&m).Error; err == nil {
		other := m.PlayerB
		if other == p.ID {
			other = m.PlayerA
		}
		info = gin.H{"status": m.Status, "other_id": other, "other_name": h.hxNameByID(other),
			"incoming": m.PlayerB == p.ID && m.Status == 1}
	}
	resp.OK(c, gin.H{"marriage": info})
}

// MarriagePropose 求婚（5000 银两）
func (h *HxxyHandler) MarriagePropose(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		PlayerID uint   `json:"player_id"`
		Name     string `json:"name"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.PlayerID == 0 && in.Name != "" {
		h.DB.Model(&model.HxxyPlayer{}).Select("id").Where("name = ?", in.Name).Scan(&in.PlayerID)
	}
	if in.PlayerID == 0 || in.PlayerID == p.ID {
		resp.ParamError(c, "找不到该玩家")
		return
	}
	var exist int64
	h.DB.Model(&model.HxxyMarriage{}).Where("(player_a = ? OR player_b = ?) AND status IN (1,2)", p.ID, p.ID).Count(&exist)
	if exist > 0 {
		resp.ParamError(c, "你已有婚约或已婚")
		return
	}
	h.DB.Model(&model.HxxyMarriage{}).Where("(player_a = ? OR player_b = ?) AND status IN (1,2)", in.PlayerID, in.PlayerID).Count(&exist)
	if exist > 0 {
		resp.ParamError(c, "对方已有婚约或已婚")
		return
	}
	const cost = 5000
	if p.Money < cost {
		resp.ParamError(c, fmt.Sprintf("求婚需要 %d 银两彩礼", cost))
		return
	}
	h.hxWallet(p, "money", -cost, "求婚彩礼")
	h.DB.Create(&model.HxxyMarriage{PlayerA: p.ID, PlayerB: in.PlayerID, Status: 1})
	h.hxNotify(in.PlayerID, fmt.Sprintf("【%s】向你求婚，请到[结婚]页处理。", p.Name))
	resp.OK(c, gin.H{"msg": "求婚成功！等待对方答应。"})
}

// MarriageAgree 答应求婚
func (h *HxxyHandler) MarriageAgree(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var m model.HxxyMarriage
	if err := h.DB.Where("player_b = ? AND status = 1", p.ID).Order("id DESC").First(&m).Error; err != nil {
		resp.ParamError(c, "没有待答应的求婚")
		return
	}
	h.DB.Model(&model.HxxyMarriage{}).Where("id = ?", m.ID).Update("status", 2)
	h.hxNotify(m.PlayerA, fmt.Sprintf("【%s】答应了你的求婚，恭喜你们喜结连理！", p.Name))
	resp.OK(c, gin.H{"msg": "恭喜你们喜结连理！"})
}

// MarriageDivorce 离婚
func (h *HxxyHandler) MarriageDivorce(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var m model.HxxyMarriage
	if err := h.DB.Where("(player_a = ? OR player_b = ?) AND status = 2", p.ID, p.ID).Order("id DESC").First(&m).Error; err != nil {
		resp.ParamError(c, "你还没有结婚")
		return
	}
	h.DB.Model(&model.HxxyMarriage{}).Where("id = ?", m.ID).Update("status", 3)
	resp.OK(c, gin.H{"msg": "你们已和平分手。"})
}

// MarriageRefuse 拒绝求婚（复刻原版 yq4.php：求婚邀请直接显示在首页）
func (h *HxxyHandler) MarriageRefuse(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		ID uint `json:"id"`
	}
	c.ShouldBindJSON(&in)
	var m model.HxxyMarriage
	q := h.DB.Where("player_b = ? AND status = 1", p.ID)
	if in.ID > 0 {
		q = q.Where("id = ?", in.ID)
	}
	if err := q.Order("id DESC").First(&m).Error; err != nil {
		resp.ParamError(c, "没有待处理的求婚")
		return
	}
	h.DB.Model(&model.HxxyMarriage{}).Where("id = ?", m.ID).Update("status", 4)
	h.hxNotify(m.PlayerA, fmt.Sprintf("【%s】拒绝了你的求婚。", p.Name))
	resp.OK(c, gin.H{"msg": "已拒绝对方的求婚"})
}

// House 住宅
func (h *HxxyHandler) House(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var house model.HxxyHouse
	furniture := []map[string]interface{}{}
	if err := h.DB.Where("player_id = ?", p.ID).First(&house).Error; err == nil && house.Furniture != "" {
		jsonUnmarshalInto(house.Furniture, &furniture)
	}
	// 可购家具（加成复刻原版住宅系统）
	shop := []gin.H{
		{"fid": 1, "name": "兵器架", "bonus": "atk", "val": 200, "price": 5000},
		{"fid": 2, "name": "练功石", "bonus": "atk", "val": 500, "price": 12000},
		{"fid": 3, "name": "屏风", "bonus": "def", "val": 200, "price": 5000},
		{"fid": 4, "name": "护院石狮", "bonus": "def", "val": 500, "price": 12000},
		{"fid": 5, "name": "檀香案", "bonus": "mg", "val": 300, "price": 8000},
		{"fid": 6, "name": "龙纹柱", "bonus": "mg", "val": 600, "price": 15000},
		{"fid": 7, "name": "雕花木床", "bonus": "hp", "val": 500, "price": 5000},
		{"fid": 8, "name": "聚灵阵盘", "bonus": "hp", "val": 1200, "price": 15000},
	}
	resp.OK(c, gin.H{"furniture": furniture, "shop": shop})
}

// HouseBuy 购买家具（加成并入住宅 JSON）
func (h *HxxyHandler) HouseBuy(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		FID uint `json:"fid"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.FID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	catalog := map[uint]struct {
		Name  string
		Bonus string
		Val   int
		Price int64
	}{
		1: {"兵器架", "atk", 200, 5000}, 2: {"练功石", "atk", 500, 12000},
		3: {"屏风", "def", 200, 5000}, 4: {"护院石狮", "def", 500, 12000},
		5: {"檀香案", "mg", 300, 8000}, 6: {"龙纹柱", "mg", 600, 15000},
		7: {"雕花木床", "hp", 500, 5000}, 8: {"聚灵阵盘", "hp", 1200, 15000},
	}
	f, ok := catalog[in.FID]
	if !ok {
		resp.ParamError(c, "没有这件家具")
		return
	}
	if p.Money < f.Price {
		resp.ParamError(c, fmt.Sprintf("需要 %d 银两", f.Price))
		return
	}
	var house model.HxxyHouse
	if err := h.DB.Where("player_id = ?", p.ID).First(&house).Error; err != nil {
		house = model.HxxyHouse{PlayerID: p.ID, Furniture: "[]"}
		h.DB.Create(&house)
	}
	var fs []map[string]interface{}
	jsonUnmarshalInto(house.Furniture, &fs)
	fs = append(fs, map[string]interface{}{"id": in.FID, "name": f.Name, "bonus": f.Bonus, "val": f.Val})
	nb := hxJSON(fs)
	h.DB.Model(&model.HxxyHouse{}).Where("id = ?", house.ID).Update("furniture", nb)
	h.hxWallet(p, "money", -f.Price, "购置家具【"+f.Name+"】")
	resp.OK(c, gin.H{"msg": fmt.Sprintf("购置【%s】成功！%s +%d", f.Name, map[string]string{"atk": "攻击", "def": "防御", "mg": "魔攻", "hp": "气血"}[f.Bonus], f.Val)})
}

// HouseInvite 邀请参观住宅（复刻原版 yq3.php：邀请直接显示在对方首页）
func (h *HxxyHandler) HouseInvite(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		PlayerID uint   `json:"player_id"`
		Name     string `json:"name"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.PlayerID == 0 && in.Name != "" {
		h.DB.Model(&model.HxxyPlayer{}).Select("id").Where("name = ?", in.Name).Scan(&in.PlayerID)
	}
	if in.PlayerID == 0 || in.PlayerID == p.ID {
		resp.ParamError(c, "找不到该玩家")
		return
	}
	var cnt int64
	h.DB.Model(&model.HxxyHouseInvite{}).Where("from_id = ? AND to_id = ? AND status = 0", p.ID, in.PlayerID).Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "已邀请过该玩家，等待对方处理")
		return
	}
	h.DB.Create(&model.HxxyHouseInvite{FromID: p.ID, FromName: p.Name, ToID: in.PlayerID, ToName: h.hxNameByID(in.PlayerID)})
	h.hxNotify(in.PlayerID, fmt.Sprintf("【%s】邀请你参观TA的住宅，请到首页处理。", p.Name))
	resp.OK(c, gin.H{"msg": "邀请已发送！"})
}

// hxHouseFurniture 读住宅家具 JSON
func (h *HxxyHandler) hxHouseFurniture(playerID uint) []map[string]interface{} {
	var house model.HxxyHouse
	fs := []map[string]interface{}{}
	if err := h.DB.Where("player_id = ?", playerID).First(&house).Error; err == nil && house.Furniture != "" {
		jsonUnmarshalInto(house.Furniture, &fs)
	}
	return fs
}

// HouseVisit 参观他人住宅
func (h *HxxyHandler) HouseVisit(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		PlayerID uint `json:"player_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.PlayerID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	resp.OK(c, gin.H{"owner_name": h.hxNameByID(in.PlayerID), "furniture": h.hxHouseFurniture(in.PlayerID)})
}

// HouseInviteAgree 同意参观邀请（首页处理，复刻原版 cmd 168 接受，直接进入对方住宅）
func (h *HxxyHandler) HouseInviteAgree(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		ID uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.ID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var iv model.HxxyHouseInvite
	if err := h.DB.Where("id = ? AND to_id = ? AND status = 0", in.ID, p.ID).First(&iv).Error; err != nil {
		resp.ParamError(c, "邀请不存在或已处理")
		return
	}
	h.DB.Model(&model.HxxyHouseInvite{}).Where("id = ?", iv.ID).Update("status", 1)
	h.hxNotify(iv.FromID, fmt.Sprintf("【%s】接受了邀请，正在参观你的住宅。", p.Name))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("你来到了%s的住宅。", iv.FromName),
		"owner_id": iv.FromID, "owner_name": iv.FromName, "furniture": h.hxHouseFurniture(iv.FromID)})
}

// HouseInviteRefuse 拒绝参观邀请（复刻原版 cmd 183 拒绝）
func (h *HxxyHandler) HouseInviteRefuse(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		ID uint `json:"id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.ID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var iv model.HxxyHouseInvite
	if err := h.DB.Where("id = ? AND to_id = ? AND status = 0", in.ID, p.ID).First(&iv).Error; err != nil {
		resp.ParamError(c, "邀请不存在或已处理")
		return
	}
	h.DB.Model(&model.HxxyHouseInvite{}).Where("id = ?", iv.ID).Update("status", 2)
	h.hxNotify(iv.FromID, fmt.Sprintf("【%s】拒绝了你参观住宅的邀请。", p.Name))
	resp.OK(c, gin.H{"msg": "已拒绝邀请"})
}

// Stalls 摆摊市场
func (h *HxxyHandler) Stalls(c *gin.Context) {
	var ss []model.HxxyStall
	h.DB.Where("status = 1").Order("id DESC").Limit(50).Find(&ss)
	list := []gin.H{}
	for _, s := range ss {
		list = append(list, gin.H{"stall_id": s.ID, "name": s.Name, "kind": s.Kind, "count": s.Count,
			"price": s.Price, "seller": h.hxNameByID(s.SellerID), "seller_id": s.SellerID})
	}
	resp.OK(c, gin.H{"stalls": list})
}

// StallSell 上架（复刻 gssjwp02：单价≥1000、上限、绑定拦截、1%手续费最低1两先扣）
func (h *HxxyHandler) StallSell(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		BagID uint  `json:"bag_id"`
		Count int   `json:"count"`
		Price int64 `json:"price"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.BagID == 0 {
		resp.ParamError(c, "输入有误请重新输入")
		return
	}
	if in.Count <= 0 {
		resp.ParamError(c, "挂售数量输入有误请重新输入")
		return
	}
	if in.Price <= 0 {
		resp.ParamError(c, "挂售价格输入有误请重新输入")
		return
	}
	if in.Price < 1000 {
		resp.ParamError(c, "挂售单价必须在1000银两上")
		return
	}
	if in.Price > 99999999999 {
		resp.ParamError(c, "挂售单价超过最大银两限制")
		return
	}
	for _, bid := range h.hxEqBagIDs(p) {
		if bid == in.BagID {
			resp.ParamError(c, "已穿戴的装备不能上架")
			return
		}
	}
	var b model.HxxyBag
	if err := h.DB.Where("id = ? AND player_id = ? AND store = 0", in.BagID, p.ID).First(&b).Error; err != nil {
		resp.ParamError(c, "物品不存在")
		return
	}
	if b.Bind == 1 {
		resp.ParamError(c, "对不起！绑定物品不能进行挂售")
		return
	}
	if b.Count < in.Count {
		resp.ParamError(c, "挂售数量输入有误请重新输入")
		return
	}
	// 挂售手续费：数量×单价×1%，最低1两，上架时先扣（复刻原版）
	fee := in.Price * int64(in.Count) / 100
	if fee < 1 {
		fee = 1
	}
	if p.Money < fee {
		resp.ParamError(c, "挂售手续费不足")
		return
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
	h.hxWallet(p, "money", -fee, "挂售【"+name+"】手续费")
	// 挂售物品转移为 store=2
	if b.Count == in.Count {
		h.DB.Model(&model.HxxyBag{}).Where("id = ?", b.ID).Update("store", 2)
	} else {
		h.DB.Model(&model.HxxyBag{}).Where("id = ?", b.ID).Update("count", b.Count-in.Count)
		h.DB.Create(&model.HxxyBag{PlayerID: p.ID, Kind: b.Kind, RefID: b.RefID, Count: in.Count, Bind: b.Bind, Store: 2, Extra: b.Extra})
	}
	h.DB.Create(&model.HxxyStall{SellerID: p.ID, BagID: b.ID, Kind: b.Kind, RefID: b.RefID, Name: name, Count: in.Count, Price: in.Price})
	resp.OK(c, gin.H{"msg": fmt.Sprintf("手续费：%s银两\n你以每件%s两的价格挂售了%sx%d", hxSilverText(fee), hxSilverText(in.Price), name, in.Count)})
}

// StallBuy 购买摊位物品（复刻 gsgmbs02：可按数量购买，买方付 1% 手续费（最低1两），卖家收全额并收私聊）
func (h *HxxyHandler) StallBuy(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		StallID uint `json:"stall_id"`
		Count   int  `json:"count"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.StallID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var s model.HxxyStall
	if err := h.DB.Where("id = ? AND status = 1", in.StallID).First(&s).Error; err != nil {
		resp.ParamError(c, "该宝石已被下架或者被买走了！！")
		return
	}
	if s.SellerID == p.ID {
		resp.ParamError(c, "不能购买自己的商品")
		return
	}
	if in.Count <= 0 || in.Count > s.Count {
		in.Count = s.Count
	}
	total := s.Price * int64(in.Count)
	// 手续费 1%，最低 1 两（买方承担）
	fee := total / 100
	if fee < 1 {
		fee = 1
	}
	if p.Money < total+fee {
		resp.ParamError(c, "对不起！你银两不足！")
		return
	}
	// 校验卖方商品仍在
	var sb model.HxxyBag
	if err := h.DB.Where("id = ? AND store = 2", s.BagID).First(&sb).Error; err != nil {
		resp.ParamError(c, "该宝石已被下架或者被买走了！！")
		return
	}
	// 交割：买方付总价+手续费，入包
	h.hxWallet(p, "money", -(total+fee), "摆摊购买【"+s.Name+"】x"+strconv.Itoa(in.Count)+"（含手续费）")
	h.hxBagAdd(p, s.Kind, s.RefID, in.Count, sb.Bind)
	if in.Count >= s.Count {
		h.DB.Delete(&model.HxxyBag{}, sb.ID)
		h.DB.Model(&model.HxxyStall{}).Where("id = ?", s.ID).Update("status", 2)
	} else {
		// 部分购买：摊位与挂售背包行同步减库存
		h.DB.Model(&model.HxxyBag{}).Where("id = ?", sb.ID).Update("count", sb.Count-in.Count)
		h.DB.Model(&model.HxxyStall{}).Where("id = ?", s.ID).Update("count", s.Count-in.Count)
	}
	// 卖家收款全额 + 私聊通知（复刻 gsgmbs02 文案）
	var seller model.HxxyPlayer
	if err := h.DB.First(&seller, s.SellerID).Error; err == nil {
		h.hxWallet(&seller, "money", total, "摆摊售出【"+s.Name+"】")
		h.hxNotify(seller.ID, fmt.Sprintf("买走了你挂售的%sx%d，获得%s银两", s.Name, in.Count, hxSilverText(total)))
	}
	// 复刻 gsgmbs02 成功文案
	resp.OK(c, gin.H{"msg": fmt.Sprintf("你用了%s，购买%sx%d（附带%s手续费）", hxSilverText(total+fee), s.Name, in.Count, hxSilverText(fee))})
}

// StallCancel 下架（复刻 gsxjwp02：可部分下架，"你下架了XxN"）
func (h *HxxyHandler) StallCancel(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		StallID uint `json:"stall_id"`
		Count   int  `json:"count"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.StallID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var s model.HxxyStall
	if err := h.DB.Where("id = ? AND seller_id = ? AND status = 1", in.StallID, p.ID).First(&s).Error; err != nil {
		resp.ParamError(c, "商品不存在")
		return
	}
	if in.Count <= 0 || in.Count >= s.Count {
		// 全部下架
		h.DB.Model(&model.HxxyStall{}).Where("id = ?", s.ID).Update("status", 3)
		h.DB.Model(&model.HxxyBag{}).Where("id = ? AND store = 2", s.BagID).Update("store", 0)
		resp.OK(c, gin.H{"msg": fmt.Sprintf("你下架了%sx%d", s.Name, s.Count)})
		return
	}
	// 部分下架：摊位减量，退回对应数量到背包
	var b model.HxxyBag
	if err := h.DB.Where("id = ? AND store = 2", s.BagID).First(&b).Error; err == nil && b.Count >= in.Count {
		if b.Count == in.Count {
			h.DB.Model(&model.HxxyBag{}).Where("id = ?", b.ID).Update("store", 0)
		} else {
			h.DB.Model(&model.HxxyBag{}).Where("id = ?", b.ID).Update("count", b.Count-in.Count)
			h.DB.Create(&model.HxxyBag{PlayerID: p.ID, Kind: b.Kind, RefID: b.RefID, Count: in.Count, Bind: b.Bind, Store: 0, Extra: b.Extra})
		}
	}
	h.DB.Model(&model.HxxyStall{}).Where("id = ?", s.ID).Update("count", s.Count-in.Count)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("你下架了%sx%d", s.Name, in.Count)})
}

// StallsMine 我的挂售（复刻 xy219：按物品/装备/宝石分类，标题"我的挂售"+挂售容量）
func (h *HxxyHandler) StallsMine(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	kind := c.DefaultQuery("kind", "item") // item/equip/gem
	q := h.DB.Where("seller_id = ? AND status = 1", p.ID)
	switch kind {
	case "equip":
		q = q.Where("kind = 'equip'")
	case "gem":
		q = q.Where("kind = 'item' AND name LIKE '%宝石%'")
	default:
		q = q.Where("kind = 'item' AND name NOT LIKE '%宝石%'")
	}
	var ss []model.HxxyStall
	q.Order("id DESC").Find(&ss)
	list := []gin.H{}
	for _, s := range ss {
		list = append(list, gin.H{"stall_id": s.ID, "name": s.Name, "kind": s.Kind,
			"count": s.Count, "price": s.Price, "desc": h.hxItemDesc(s.Kind, s.RefID)})
	}
	var used int64
	h.DB.Model(&model.HxxyStall{}).Where("seller_id = ? AND status = 1", p.ID).Count(&used)
	resp.OK(c, gin.H{"stalls": list, "used": used, "capacity": 10})
}

// StallsOf 他人挂售列表（复刻 xy222："{名字}的挂售："，从玩家资料页进入）
func (h *HxxyHandler) StallsOf(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	sid, _ := strconv.Atoi(c.Param("id"))
	if sid <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var ss []model.HxxyStall
	h.DB.Where("seller_id = ? AND status = 1", uint(sid)).Order("id DESC").Limit(100).Find(&ss)
	list := []gin.H{}
	for _, s := range ss {
		list = append(list, gin.H{"stall_id": s.ID, "name": s.Name, "kind": s.Kind, "count": s.Count,
			"price": s.Price, "desc": h.hxItemDesc(s.Kind, s.RefID), "seller": h.hxNameByID(s.SellerID), "seller_id": s.SellerID})
	}
	var used int64
	h.DB.Model(&model.HxxyStall{}).Where("seller_id = ? AND status = 1", uint(sid)).Count(&used)
	resp.OK(c, gin.H{"seller_id": sid, "seller_name": h.hxNameByID(uint(sid)), "stalls": list, "used": used, "capacity": 10})
}

// hxItemDesc 物品/装备描述
func (h *HxxyHandler) hxItemDesc(kind string, refID uint) string {
	if kind == "equip" {
		var e model.HxxyEquip
		if err := h.DB.First(&e, refID).Error; err == nil {
			return e.Desc
		}
		return ""
	}
	var it model.HxxyItem
	if err := h.DB.First(&it, refID).Error; err == nil {
		return it.Desc
	}
	return ""
}

// jsonUnmarshalInto 宽松 JSON 解析
func jsonUnmarshalInto(s string, v interface{}) {
	if s != "" {
		json.Unmarshal([]byte(s), v)
	}
}
