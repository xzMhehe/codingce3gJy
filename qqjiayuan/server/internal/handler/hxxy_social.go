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

// Friends 好友列表
func (h *HxxyHandler) Friends(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var fs []model.HxxyFriend
	h.DB.Where("player_id = ? OR friend_id = ?", p.ID, p.ID).Find(&fs)
	friends, applies := []gin.H{}, []gin.H{}
	for _, f := range fs {
		other := f.FriendID
		if other == p.ID {
			other = f.PlayerID
		}
		name := h.hxNameByID(other)
		if f.Status == 2 {
			friends = append(friends, gin.H{"player_id": other, "name": name})
		} else if f.FriendID == p.ID {
			applies = append(applies, gin.H{"player_id": other, "name": name, "apply_id": f.ID})
		}
	}
	resp.OK(c, gin.H{"friends": friends, "applies": applies})
}

// FriendAdd 好友申请
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
	var cnt int64
	h.DB.Model(&model.HxxyFriend{}).Where("(player_id = ? AND friend_id = ?) OR (player_id = ? AND friend_id = ?)",
		p.ID, in.PlayerID, in.PlayerID, p.ID).Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "已是好友或申请中")
		return
	}
	h.DB.Create(&model.HxxyFriend{PlayerID: p.ID, FriendID: in.PlayerID, Status: 1})
	h.hxNotify(in.PlayerID, fmt.Sprintf("【%s】请求加你为好友，请到[好友]页处理。", p.Name))
	resp.OK(c, gin.H{"msg": "好友申请已发送！"})
}

// FriendAgree 同意好友
func (h *HxxyHandler) FriendAgree(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		ApplyID uint `json:"apply_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.ApplyID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var f model.HxxyFriend
	if err := h.DB.Where("id = ? AND friend_id = ? AND status = 1", in.ApplyID, p.ID).First(&f).Error; err != nil {
		resp.ParamError(c, "申请不存在")
		return
	}
	h.DB.Model(&model.HxxyFriend{}).Where("id = ?", f.ID).Update("status", 2)
	h.hxNotify(f.PlayerID, fmt.Sprintf("【%s】同意了你的好友申请，你们已成为好友。", p.Name))
	resp.OK(c, gin.H{"msg": "你们已成为好友！"})
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
		list = append(list, gin.H{"player_id": n.ID, "name": n.Name, "level": n.Level, "sect_name": hxSectNames[n.Sect]})
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
	resp.OK(c, gin.H{"msgs": msgs, "nearby": list, "notices": hxStaticNotices(), "team_invites": inv,
		"gang_invites": glist, "house_invites": hlist, "marriage_invite": marriageInvite})
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

// GangInfo 帮派信息（我的帮派 + 列表）
func (h *HxxyHandler) GangInfo(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var mem model.HxxyGangMember
	myGang := gin.H{}
	if err := h.DB.Where("player_id = ?", p.ID).First(&mem).Error; err == nil {
		var g model.HxxyGang
		if err := h.DB.First(&g, mem.GangID).Error; err == nil {
			var members []gin.H
			var ms []model.HxxyGangMember
			h.DB.Where("gang_id = ?", g.ID).Order("role DESC").Limit(50).Find(&ms)
			for _, m := range ms {
				role := "帮众"
				if m.Role == 2 {
					role = "帮主"
				} else if m.Role == 1 {
					role = "长老"
				}
				members = append(members, gin.H{"player_id": m.PlayerID, "name": h.hxNameByID(m.PlayerID), "role": role, "contribution": m.Contribution})
			}
			myGang = gin.H{"gang_id": g.ID, "name": g.Name, "level": g.Level, "notice": g.Notice,
				"money": g.Money, "role": map[int]string{0: "帮众", 1: "长老", 2: "帮主"}[mem.Role],
				"contribution": mem.Contribution, "members": members}
		}
	}
	var gs []model.HxxyGang
	h.DB.Order("level DESC").Limit(20).Find(&gs)
	list := []gin.H{}
	for _, g := range gs {
		list = append(list, gin.H{"gang_id": g.ID, "name": g.Name, "level": g.Level, "notice": g.Notice})
	}
	resp.OK(c, gin.H{"my_gang": myGang, "gangs": list})
}

// GangCreate 创建帮派（10000 银两）
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
		resp.ParamError(c, "请输入帮派名称")
		return
	}
	in.Name = trimStr(strings.TrimSpace(in.Name), 10)
	var mem model.HxxyGangMember
	if err := h.DB.Where("player_id = ?", p.ID).First(&mem).Error; err == nil {
		resp.ParamError(c, "你已有帮派，先退出再创建")
		return
	}
	var cnt int64
	h.DB.Model(&model.HxxyGang{}).Where("name = ?", in.Name).Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "帮派名已存在")
		return
	}
	const cost = 10000
	if p.Money < cost {
		resp.ParamError(c, fmt.Sprintf("创建帮派需要 %d 银两", cost))
		return
	}
	g := model.HxxyGang{Name: in.Name, LeaderID: p.ID, Notice: "本帮广纳贤士！"}
	h.DB.Create(&g)
	h.hxWallet(p, "money", -cost, "创建帮派【"+in.Name+"】")
	h.DB.Create(&model.HxxyGangMember{GangID: g.ID, PlayerID: p.ID, Role: 2})
	resp.OK(c, gin.H{"msg": fmt.Sprintf("帮派【%s】创建成功！你是帮主。", in.Name)})
}

// GangJoin 加入帮派
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
	var mem model.HxxyGangMember
	if err := h.DB.Where("player_id = ?", p.ID).First(&mem).Error; err == nil {
		resp.ParamError(c, "你已有帮派")
		return
	}
	var g model.HxxyGang
	if err := h.DB.First(&g, in.GangID).Error; err != nil {
		resp.ParamError(c, "帮派不存在")
		return
	}
	h.DB.Create(&model.HxxyGangMember{GangID: g.ID, PlayerID: p.ID})
	resp.OK(c, gin.H{"msg": fmt.Sprintf("欢迎加入【%s】！", g.Name)})
}

// GangLeave 退出帮派
func (h *HxxyHandler) GangLeave(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var mem model.HxxyGangMember
	if err := h.DB.Where("player_id = ?", p.ID).First(&mem).Error; err != nil {
		resp.ParamError(c, "你没有帮派")
		return
	}
	if mem.Role == 2 {
		// 帮主退出 → 转让给贡献最高的成员或解散
		var next model.HxxyGangMember
		if err := h.DB.Where("gang_id = ? AND player_id <> ?", mem.GangID, p.ID).Order("contribution DESC").First(&next).Error; err == nil {
			h.DB.Model(&model.HxxyGangMember{}).Where("id = ?", next.ID).Update("role", 2)
			h.DB.Model(&model.HxxyGang{}).Where("id = ?", mem.GangID).Update("leader_id", next.PlayerID)
			h.DB.Delete(&model.HxxyGangMember{}, mem.ID)
			resp.OK(c, gin.H{"msg": "你已退帮，帮主之位已转让。"})
			return
		}
		h.DB.Delete(&model.HxxyGang{}, mem.GangID)
		h.DB.Delete(&model.HxxyGangMember{}, mem.ID)
		resp.OK(c, gin.H{"msg": "帮派只剩你一人，已解散。"})
		return
	}
	h.DB.Delete(&model.HxxyGangMember{}, mem.ID)
	resp.OK(c, gin.H{"msg": "你已退出帮派。"})
}

// GangDonate 帮派捐献（1银两=1贡献）
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
		resp.ParamError(c, "请输入正确的金额")
		return
	}
	if p.Money < in.Amount {
		resp.ParamError(c, "银两不足")
		return
	}
	var mem model.HxxyGangMember
	if err := h.DB.Where("player_id = ?", p.ID).First(&mem).Error; err != nil {
		resp.ParamError(c, "你没有帮派")
		return
	}
	h.hxWallet(p, "money", -in.Amount, "帮派捐献")
	h.DB.Model(&model.HxxyGang{}).Where("id = ?", mem.GangID).UpdateColumn("money", gorm.Expr("money + ?", in.Amount))
	h.DB.Model(&model.HxxyGangMember{}).Where("id = ?", mem.ID).UpdateColumn("contribution", gorm.Expr("contribution + ?", in.Amount))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("捐献 %d 银两，贡献 +%d！", in.Amount, in.Amount)})
}

// GangInvite 邀请入帮（复刻原版 yq2.php 国家邀请：帮主/长老可发，邀请直接显示在对方首页）
func (h *HxxyHandler) GangInvite(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var mem model.HxxyGangMember
	if err := h.DB.Where("player_id = ?", p.ID).First(&mem).Error; err != nil {
		resp.ParamError(c, "你还没有帮派")
		return
	}
	if mem.Role < 1 {
		resp.ParamError(c, "只有帮主或长老才能邀请入帮")
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
	var otherMem model.HxxyGangMember
	if err := h.DB.Where("player_id = ?", in.PlayerID).First(&otherMem).Error; err == nil {
		resp.ParamError(c, "对方已有帮派")
		return
	}
	var g model.HxxyGang
	if err := h.DB.First(&g, mem.GangID).Error; err != nil {
		resp.ParamError(c, "帮派不存在")
		return
	}
	var cnt int64
	h.DB.Model(&model.HxxyGangInvite{}).Where("gang_id = ? AND to_id = ? AND status = 0", g.ID, in.PlayerID).Count(&cnt)
	if cnt > 0 {
		resp.ParamError(c, "已邀请过该玩家，等待对方处理")
		return
	}
	h.DB.Create(&model.HxxyGangInvite{GangID: g.ID, GangName: g.Name, FromID: p.ID, FromName: p.Name, ToID: in.PlayerID, ToName: h.hxNameByID(in.PlayerID)})
	h.hxNotify(in.PlayerID, fmt.Sprintf("【%s】邀请你加入帮派【%s】，请到首页处理。", p.Name, g.Name))
	resp.OK(c, gin.H{"msg": "邀请已发送！"})
}

// GangInviteAgree 同意入帮邀请（首页处理，复刻原版 cmd 180 接受）
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
	var mem model.HxxyGangMember
	if err := h.DB.Where("player_id = ?", p.ID).First(&mem).Error; err == nil {
		h.DB.Model(&model.HxxyGangInvite{}).Where("id = ?", iv.ID).Update("status", 2)
		resp.ParamError(c, "你已有帮派")
		return
	}
	h.DB.Create(&model.HxxyGangMember{GangID: iv.GangID, PlayerID: p.ID})
	h.DB.Model(&model.HxxyGangInvite{}).Where("id = ?", iv.ID).Update("status", 1)
	h.hxNotify(iv.FromID, fmt.Sprintf("【%s】接受了你的邀请，已加入帮派【%s】。", p.Name, iv.GangName))
	resp.OK(c, gin.H{"msg": fmt.Sprintf("欢迎加入【%s】！", iv.GangName)})
}

// GangInviteRefuse 拒绝入帮邀请（复刻原版 cmd 181 拒绝）
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
	h.hxNotify(iv.FromID, fmt.Sprintf("【%s】拒绝了你加入帮派【%s】的邀请。", p.Name, iv.GangName))
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

// StallSell 上架
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
	if err := c.ShouldBindJSON(&in); err != nil || in.BagID == 0 || in.Price <= 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	if in.Count <= 0 {
		in.Count = 1
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
	if b.Count < in.Count {
		resp.ParamError(c, "数量不足")
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
	// 挂售物品转移为 store=2
	if b.Count == in.Count {
		h.DB.Model(&model.HxxyBag{}).Where("id = ?", b.ID).Update("store", 2)
	} else {
		h.DB.Model(&model.HxxyBag{}).Where("id = ?", b.ID).Update("count", b.Count-in.Count)
		h.DB.Create(&model.HxxyBag{PlayerID: p.ID, Kind: b.Kind, RefID: b.RefID, Count: in.Count, Bind: b.Bind, Store: 2, Extra: b.Extra})
	}
	h.DB.Create(&model.HxxyStall{SellerID: p.ID, BagID: b.ID, Kind: b.Kind, RefID: b.RefID, Name: name, Count: in.Count, Price: in.Price})
	resp.OK(c, gin.H{"msg": fmt.Sprintf("【%s】×%d 已上架，售价 %d 银两。", name, in.Count, in.Price)})
}

// StallBuy 购买摊位物品
func (h *HxxyHandler) StallBuy(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		StallID uint `json:"stall_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.StallID == 0 {
		resp.ParamError(c, "参数错误")
		return
	}
	var s model.HxxyStall
	if err := h.DB.Where("id = ? AND status = 1", in.StallID).First(&s).Error; err != nil {
		resp.ParamError(c, "商品已售出或下架")
		return
	}
	if s.SellerID == p.ID {
		resp.ParamError(c, "不能购买自己的商品")
		return
	}
	if p.Money < s.Price {
		resp.ParamError(c, fmt.Sprintf("需要 %d 银两，银两不足", s.Price))
		return
	}
	// 校验卖方商品仍在
	var sb model.HxxyBag
	if err := h.DB.Where("id = ? AND store = 2", s.BagID).First(&sb).Error; err != nil {
		resp.ParamError(c, "商品已失效")
		return
	}
	// 交割
	h.hxWallet(p, "money", -s.Price, "摆摊购买【"+s.Name+"】")
	h.hxBagAdd(p, s.Kind, s.RefID, s.Count, sb.Bind)
	h.DB.Delete(&model.HxxyBag{}, sb.ID)
	h.DB.Model(&model.HxxyStall{}).Where("id = ?", s.ID).Update("status", 2)
	// 卖家收款（流水）
	var seller model.HxxyPlayer
	if err := h.DB.First(&seller, s.SellerID).Error; err == nil {
		h.hxWallet(&seller, "money", s.Price, "摆摊售出【"+s.Name+"】")
		h.hxNotify(seller.ID, fmt.Sprintf("你挂售的【%s】×%d 被【%s】买走，入账 %d 银两。", s.Name, s.Count, p.Name, s.Price))
	}
	resp.OK(c, gin.H{"msg": fmt.Sprintf("购买【%s】×%d 成功！", s.Name, s.Count)})
}

// StallCancel 下架
func (h *HxxyHandler) StallCancel(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		StallID uint `json:"stall_id"`
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
	h.DB.Model(&model.HxxyStall{}).Where("id = ?", s.ID).Update("status", 3)
	h.DB.Model(&model.HxxyBag{}).Where("id = ? AND store = 2", s.BagID).Update("store", 0)
	resp.OK(c, gin.H{"msg": "已下架，物品退回背包。"})
}

// jsonUnmarshalInto 宽松 JSON 解析
func jsonUnmarshalInto(s string, v interface{}) {
	if s != "" {
		json.Unmarshal([]byte(s), v)
	}
}
