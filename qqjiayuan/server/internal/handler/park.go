package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/middleware"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// 抢车位（对齐诺哈三代 ASP 版 wap/game/car 玩法）
// 停车收入 = 停车整小时数 * 每小时盈利；税(国库) = 总收入/5；贡献 = 总收入/10；经验 = 停车小时数
// 停车超过 12 小时收入入国库，车辆变流动车；等级 = 经验/100；每人最多 10 辆车
const (
	parkStopHours = 12
	parkMaxCars   = 10
	parkLevelStep = 100
	parkInitSpots = 3
	parkSubsidy   = 1000 // 开通补助(对齐 help.asp 文案)
)

type ParkHandler struct{ DB *gorm.DB }

// ---------- 内部工具 ----------

// parkOf 玩家数据（懒创建，对齐 car_add.asp 开通逻辑）
func (h *ParkHandler) parkOf(uid uint) *model.ParkUser {
	var p model.ParkUser
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err != nil {
		u := h.userBrief(uid)
		p = model.ParkUser{UserID: uid}
		h.DB.Create(&p)
		for i := 1; i <= parkInitSpots; i++ {
			h.DB.Create(&model.CarStop{UserID: uid, Sort: i})
		}
		h.DB.Create(&model.CarLog{UserID: uid, Name: u.Nickname})
		// 开通补助（对齐 help.asp "系统赠送1000G生活补助费"）
		h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins + ?", parkSubsidy))
		addWalletLog(h.DB, uid, "park", "抢车位开通补助", "coins", parkSubsidy)
		h.carSend(0, uid, "欢迎来到抢车位，系统赠送了你 "+strconv.Itoa(parkInitSpots)+" 个车位和 "+strconv.Itoa(parkSubsidy)+"G 生活补助费，快去车市提一辆dream car吧！")
	}
	return &p
}

func (h *ParkHandler) userBrief(uid uint) *model.User {
	var u model.User
	h.DB.Select("id,username,nickname,coins").First(&u, uid)
	return &u
}

// nickMap 批量取昵称
func (h *ParkHandler) nickMap(uidSet map[uint]bool) map[uint]string {
	m := make(map[uint]string, len(uidSet))
	if len(uidSet) == 0 {
		return m
	}
	ids := make([]uint, 0, len(uidSet))
	for uid := range uidSet {
		ids = append(ids, uid)
	}
	var users []model.User
	h.DB.Select("id, nickname").Where("id IN ?", ids).Find(&users)
	for _, u := range users {
		m[u.ID] = u.Nickname
	}
	return m
}

func (h *ParkHandler) carSend(from, to uint, content string) {
	h.DB.Create(&model.CarMsg{UserID: to, FID: from, Content: content})
}

// carByID 车市车辆（status=0 上架才有效）
func (h *ParkHandler) carByID(id uint) *model.CarShop {
	var s model.CarShop
	if err := h.DB.First(&s, id).Error; err != nil {
		return nil
	}
	return &s
}

// parkLevel 等级 = 经验/100（对齐 CarPoint/100）
func parkLevel(point int) int { return point / parkLevelStep }

// parkCalc 停车收入（对齐 seal/favor 计算）
// hours=停车整小时数 total=hours*每小时盈利 tax=total/5 net=total-tax contri=total/10
func parkCalc(parked time.Duration, moneyPerHour int) (hours, total, tax, net, contri int) {
	hours = int(parked.Hours())
	total = hours * moneyPerHour
	tax = total / 5
	net = total - tax
	contri = total / 10
	return
}

// clearSpot 清空车位并让车辆回到流动状态
func (h *ParkHandler) clearSpot(stop *model.CarStop) {
	h.DB.Model(&model.CarStop{}).Where("id = ?", stop.ID).
		Updates(map[string]interface{}{"gar_id": 0, "car_id": 0, "owner": 0, "stop_time": nil})
	if stop.GarID > 0 {
		h.DB.Model(&model.CarGarage{}).Where("id = ?", stop.GarID).
			Updates(map[string]interface{}{"stop_id": 0, "new_time": time.Now()})
	}
	stop.GarID, stop.CarID, stop.Owner, stop.StopTime = 0, 0, 0, nil
}

// spotInfo 车位视图（超时自动结算：收入入国库，车变流动车）
func (h *ParkHandler) spotInfo(stop model.CarStop, meUID uint, isSelf bool) gin.H {
	out := gin.H{"id": stop.ID, "sort": stop.Sort, "car_id": stop.CarID,
		"owner": stop.Owner, "empty": stop.Owner == 0}
	if stop.Owner == 0 {
		return out
	}
	car := h.carByID(stop.CarID)
	if car == nil {
		out["lost"] = true
		out["car_name"] = "汽车(" + strconv.Itoa(int(stop.CarID)) + ")参数丢失"
		return out
	}
	now := time.Now()
	parked := now.Sub(*stop.StopTime)
	hours, total, tax, net, contri := parkCalc(parked, car.Money)
	out["car_name"] = car.Name
	out["price"] = car.Price
	out["dtype"] = car.DType
	out["owner_nick"] = h.nickMap(map[uint]bool{stop.Owner: true})[stop.Owner]
	out["minutes"] = int(parked.Minutes())
	out["hours"] = hours
	out["over"] = parked > time.Duration(parkStopHours)*time.Hour
	out["total"] = total
	out["tax"] = tax
	out["net"] = net
	out["contri"] = contri
	if out["over"] == true {
		// 超 12 小时：自动清位（查看时结算，对齐 index.asp/owner.asp）
		h.clearSpot(&stop)
		out["empty"] = true
		out["over_msg"] = "停车时间已超过" + strconv.Itoa(parkStopHours) + "小时，车辆所得已入国库，车辆已变为流动车！"
		return out
	}
	// 操作权限：收车=停车人 贴封条=车位主人
	if stop.Owner == meUID {
		out["can_favor"] = true
	}
	if isSelf {
		out["can_seal"] = true
	}
	return out
}

// garageInfo 车库车辆视图（对齐 my_garage.asp）
func (h *ParkHandler) garageInfo(g model.CarGarage, uid uint) gin.H {
	out := gin.H{"id": g.ID, "car_id": g.CarID, "stop_id": g.StopID, "moving": g.StopID == 0}
	car := h.carByID(g.CarID)
	if car == nil {
		out["lost"] = true
		out["car_name"] = "汽车(" + strconv.Itoa(int(g.CarID)) + ")参数丢失"
		return out
	}
	out["car_name"] = car.Name
	out["price"] = car.Price
	out["dtype"] = car.DType
	if g.StopID == 0 {
		out["moving_msg"] = "流动中，小心被警察罚款哦！赶紧找个车位吧！"
		return out
	}
	var stop model.CarStop
	if err := h.DB.First(&stop, g.StopID).Error; err != nil {
		// 车位参数错误：重置为流动车
		h.DB.Model(&model.CarGarage{}).Where("id = ?", g.ID).
			Updates(map[string]interface{}{"stop_id": 0, "new_time": time.Now()})
		out["moving"] = true
		out["moving_msg"] = "车位参数错误，已重置为流动车。"
		return out
	}
	out["spot_uid"] = stop.UserID
	out["spot_nick"] = h.nickMap(map[uint]bool{stop.UserID: true})[stop.UserID]
	now := time.Now()
	if stop.Owner == 0 || stop.StopTime == nil {
		out["moving"] = true
		out["moving_msg"] = "车位已空置，已重置为流动车。"
		h.DB.Model(&model.CarGarage{}).Where("id = ?", g.ID).
			Updates(map[string]interface{}{"stop_id": 0, "new_time": now})
		return out
	}
	parked := now.Sub(*stop.StopTime)
	hours, total, tax, net, contri := parkCalc(parked, car.Money)
	out["minutes"] = int(parked.Minutes())
	out["hours"] = hours
	out["over"] = parked > time.Duration(parkStopHours)*time.Hour
	out["total"] = total
	out["tax"] = tax
	out["net"] = net
	out["contri"] = contri
	if out["over"] == true {
		out["over_msg"] = "停车时间已超过" + strconv.Itoa(parkStopHours) + "小时，车辆所得将入国库！"
	} else {
		out["can_favor"] = true // 车主可收车
	}
	return out
}

// ---------- 接口 ----------

// View 我的停车场（对齐 index.asp）
func (h *ParkHandler) View(c *gin.Context) {
	uid := middleware.GetUID(c)
	p := h.parkOf(uid)
	u := h.userBrief(uid)

	var stops []model.CarStop
	h.DB.Where("user_id = ?", uid).Order("sort ASC, id ASC").Find(&stops)
	spots := make([]gin.H, 0, len(stops))
	for _, s := range stops {
		spots = append(spots, h.spotInfo(s, uid, true))
	}

	// 最新加入（对齐 index.asp TOP5）
	var logs []model.CarLog
	h.DB.Order("id DESC").Limit(5).Find(&logs)
	uidSet := map[uint]bool{}
	for _, l := range logs {
		uidSet[l.UserID] = true
	}
	nick := h.nickMap(uidSet)
	recent := make([]gin.H, 0, len(logs))
	for _, l := range logs {
		recent = append(recent, gin.H{"uid": l.UserID, "nick": nick[l.UserID],
			"time_txt": timeSince(l.CreatedAt)})
	}

	// 未读消息弹出（查看即置已读，对齐农场做法）
	var msgs []model.CarMsg
	h.DB.Where("user_id = ?", uid).Order("id DESC").Limit(10).Find(&msgs)
	h.DB.Model(&model.CarMsg{}).Where("user_id = ? AND status = 0", uid).Update("status", 1)
	uidSet = map[uint]bool{}
	for _, m := range msgs {
		if m.FID > 0 {
			uidSet[m.FID] = true
		}
	}
	fnick := h.nickMap(uidSet)
	msgViews := make([]gin.H, 0, len(msgs))
	for _, m := range msgs {
		from := "系统消息"
		if m.FID > 0 {
			from = fnick[m.FID]
		}
		msgViews = append(msgViews, gin.H{"id": m.ID, "fid": m.FID, "nick": from, "msg": m.Content,
			"time_txt": m.CreatedAt.Format("01-02 15:04")})
	}

	resp.OK(c, gin.H{
		"uid": uid, "nick": u.Nickname, "coins": u.Coins,
		"cars": p.Cars, "love": p.Love, "point": p.Point, "contri": p.Contri,
		"level": parkLevel(p.Point), "count": h.parkCount(),
		"spots": spots, "recent": recent, "msgs": msgViews,
	})
}

// parkCount 参与人数
func (h *ParkHandler) parkCount() int64 {
	var n int64
	h.DB.Model(&model.ParkUser{}).Count(&n)
	return n
}

// timeSince 简洁间隔文案（对齐 Interval_Time）
func timeSince(t time.Time) string {
	d := int(time.Since(t).Seconds())
	switch {
	case d < 60:
		return strconv.Itoa(d) + "秒钟"
	case d < 3600:
		return strconv.Itoa(d/60) + "分钟"
	case d < 86400:
		return strconv.Itoa(d/3600) + "小时"
	default:
		return strconv.Itoa(d/86400) + "天"
	}
}

// Owner 他人停车场（对齐 owner.asp，uid=0 看自己）
func (h *ParkHandler) Owner(c *gin.Context) {
	meUID := middleware.GetUID(c)
	uid, _ := strconv.Atoi(c.Query("uid"))
	if uid <= 0 {
		uid = int(meUID)
	}
	var target model.ParkUser
	if err := h.DB.Where("user_id = ?", uid).First(&target).Error; err != nil {
		resp.ParamError(c, "TA还没有开通抢车位！")
		return
	}
	u := h.userBrief(uint(uid))
	var stops []model.CarStop
	h.DB.Where("user_id = ?", uid).Order("sort ASC, id ASC").Find(&stops)
	spots := make([]gin.H, 0, len(stops))
	for _, s := range stops {
		spots = append(spots, h.spotInfo(s, meUID, uint(uid) == meUID))
	}
	resp.OK(c, gin.H{
		"uid": uint(uid), "nick": u.Nickname, "cars": target.Cars, "love": target.Love,
		"level": parkLevel(target.Point), "contri": target.Contri,
		"is_self": uint(uid) == meUID, "spots": spots,
	})
}

// Friends 可停车的玩家列表（对齐 friend.asp：按加入时间倒序分页）
func (h *ParkHandler) Friends(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	var total int64
	h.DB.Model(&model.ParkUser{}).Count(&total)
	var rows []model.ParkUser
	h.DB.Order("created_at DESC, id ASC").Offset(offset).Limit(size).Find(&rows)
	uidSet := map[uint]bool{}
	for _, p := range rows {
		uidSet[p.UserID] = true
	}
	nick := h.nickMap(uidSet)
	out := make([]gin.H, 0, len(rows))
	for _, p := range rows {
		// 每个玩家的空位数（前端提示可停）
		var empty int64
		h.DB.Model(&model.CarStop{}).Where("user_id = ? AND owner = 0", p.UserID).Count(&empty)
		out = append(out, gin.H{"uid": p.UserID, "nick": nick[p.UserID],
			"level": parkLevel(p.Point), "cars": p.Cars, "empty": empty})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// Shop 车市（对齐 shop.asp：dtype 分类，按盈利倒序）
func (h *ParkHandler) Shop(c *gin.Context) {
	dtype, _ := strconv.Atoi(c.DefaultQuery("dtype", "1"))
	if dtype < 1 || dtype > 5 {
		dtype = 1
	}
	page, offset, size := pageOf(c, 6)
	q := h.DB.Model(&model.CarShop{}).Where("dtype = ? AND status = 0", dtype)
	var total int64
	q.Count(&total)
	var cars []model.CarShop
	q.Order("money DESC, price ASC").Offset(offset).Limit(size).Find(&cars)
	resp.OK(c, gin.H{"dtype": dtype, "total": total, "page": page, "size": size, "list": cars})
}

// Buy 购买汽车（对齐 buy.asp）
func (h *ParkHandler) Buy(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	car := h.carByID(req.ID)
	if car == nil || car.Status != 0 {
		resp.ParamError(c, "无此汽车！")
		return
	}
	u := h.userBrief(uid)
	if u.Coins < car.Price {
		resp.ParamError(c, "您的G币不够买这辆车")
		return
	}
	p := h.parkOf(uid)
	if p.Cars >= parkMaxCars {
		resp.ParamError(c, "您的车太多了，每人最多只能拥有" + strconv.Itoa(parkMaxCars) + "辆车哦！")
		return
	}
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins - ?", car.Price))
	h.DB.Model(&model.ParkUser{}).Where("id = ?", p.ID).Update("cars", gorm.Expr("cars + 1"))
	h.DB.Create(&model.CarGarage{UserID: uid, CarID: car.ID, NewTime: time.Now()})
	addWalletLog(h.DB, uid, "park", "购买汽车:"+car.Name, "coins", -car.Price)
	p.Cars++
	resp.OK(c, gin.H{"msg": "购买成功，赶紧去抢个车位哦！", "cars": p.Cars, "coins": u.Coins - car.Price})
}

// Send 赠送汽车（对齐 send.asp）
func (h *ParkHandler) Send(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		ID  uint `json:"id" binding:"required"`
		UID uint `json:"uid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.UID == uid {
		resp.ParamError(c, "不能赠送给自己哦")
		return
	}
	car := h.carByID(req.ID)
	if car == nil || car.Status != 0 {
		resp.ParamError(c, "无此汽车！")
		return
	}
	var target model.ParkUser
	if err := h.DB.Where("user_id = ?", req.UID).First(&target).Error; err != nil {
		resp.ParamError(c, "TA还没有开通抢车位！")
		return
	}
	if target.Cars >= parkMaxCars {
		resp.ParamError(c, "TA的车太多了，每人最多只能拥有" + strconv.Itoa(parkMaxCars) + "辆车哦！")
		return
	}
	u := h.userBrief(uid)
	if u.Coins < car.Price {
		resp.ParamError(c, "您的G币不够买这辆车")
		return
	}
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins - ?", car.Price))
	h.DB.Model(&model.ParkUser{}).Where("id = ?", target.ID).Update("cars", gorm.Expr("cars + 1"))
	h.DB.Create(&model.CarGarage{UserID: req.UID, CarID: car.ID, NewTime: time.Now()})
	addWalletLog(h.DB, uid, "park", "赠送汽车:"+car.Name, "coins", -car.Price)
	me := h.userBrief(uid)
	h.carSend(uid, req.UID, me.Nickname+" 赠送了一辆["+car.Name+"]，赶紧去抢个车位哦！")
	resp.OK(c, gin.H{"msg": "赠送成功！", "coins": u.Coins - car.Price})
}

// Stop 停车（对齐 stop_car.asp：把流动车停进他人空车位）
func (h *ParkHandler) Stop(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		OID    uint `json:"oid" binding:"required"`
		StopID uint `json:"stop_id" binding:"required"`
		GarID  uint `json:"gar_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var gar model.CarGarage
	if err := h.DB.Where("id = ? AND user_id = ? AND stop_id = 0", req.GarID, uid).First(&gar).Error; err != nil {
		resp.ParamError(c, "请选择要停放的车辆！")
		return
	}
	var stop model.CarStop
	if err := h.DB.Where("id = ? AND user_id = ? AND owner = 0", req.StopID, req.OID).First(&stop).Error; err != nil {
		resp.ParamError(c, "这个车位已经有车了！")
		return
	}
	now := time.Now()
	h.DB.Model(&model.CarGarage{}).Where("id = ?", gar.ID).Update("stop_id", stop.ID)
	h.DB.Model(&model.CarStop{}).Where("id = ?", stop.ID).Updates(map[string]interface{}{
		"gar_id": gar.ID, "car_id": gar.CarID, "owner": uid, "stop_time": now})
	car := h.carByID(gar.CarID)
	name := ""
	if car != nil {
		name = car.Name
	}
	h.carSend(uid, stop.UserID, "有人把一辆["+name+"]停进了你的车位，记得贴封条赚罚金哦！")
	resp.OK(c, gin.H{"msg": "停车成功！收车不能超过" + strconv.Itoa(parkStopHours) + "小时哦！"})
}

// Favor 收车（对齐 favor_car.asp：停车人收取净收入）
func (h *ParkHandler) Favor(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		StopID uint `json:"stop_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	var stop model.CarStop
	if err := h.DB.Where("id = ? AND owner = ?", req.StopID, uid).First(&stop).Error; err != nil {
		resp.ParamError(c, "您已收过了！")
		return
	}
	parked := time.Now().Sub(*stop.StopTime)
	if parked > time.Duration(parkStopHours)*time.Hour {
		h.clearSpot(&stop)
		resp.OK(c, gin.H{"msg": "停车时间已超过" + strconv.Itoa(parkStopHours) + "小时，车辆所得入国库，车辆已变为流动车！"})
		return
	}
	car := h.carByID(stop.CarID)
	if car == nil {
		h.clearSpot(&stop)
		resp.ParamError(c, "汽车("+strconv.Itoa(int(stop.CarID))+")参数丢失")
		return
	}
	hours, _, _, net, contri := parkCalc(parked, car.Money)
	// 结算：净收入 + 经验 + 贡献，清位
	h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins + ?", net))
	p := h.parkOf(uid)
	h.DB.Model(&model.ParkUser{}).Where("id = ?", p.ID).Updates(map[string]interface{}{
		"point": gorm.Expr("point + ?", hours), "contri": gorm.Expr("contri + ?", contri)})
	addWalletLog(h.DB, uid, "park", "收车:"+car.Name, "coins", net)
	h.clearSpot(&stop)
	resp.OK(c, gin.H{"msg": "收车成功！本次收入" + strconv.Itoa(net) + "G币，贡献" + strconv.Itoa(contri) + "，经验" + strconv.Itoa(hours) + "！",
		"net": net, "contri": contri, "hours": hours})
}

// Seal 贴封条（对齐 seal.asp：车位主人按比例没收停车收入）
func (h *ParkHandler) Seal(c *gin.Context) {
	uid := middleware.GetUID(c)
	var req struct {
		StopID uint `json:"stop_id" binding:"required"`
		Ratio  int  `json:"ratio"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	ratio := clampInt(req.Ratio, 0, 10)
	var stop model.CarStop
	if err := h.DB.Where("id = ? AND user_id = ? AND owner > 0", req.StopID, uid).First(&stop).Error; err != nil {
		resp.ParamError(c, "动作太慢了，别人已经收车啦！")
		return
	}
	parked := time.Now().Sub(*stop.StopTime)
	if parked > time.Duration(parkStopHours)*time.Hour {
		h.clearSpot(&stop)
		resp.OK(c, gin.H{"msg": "停车时间已超过" + strconv.Itoa(parkStopHours) + "小时，车辆所得入国库，车辆已变为流动车！"})
		return
	}
	car := h.carByID(stop.CarID)
	if car == nil {
		h.clearSpot(&stop)
		resp.ParamError(c, "汽车("+strconv.Itoa(int(stop.CarID))+")参数丢失")
		return
	}
	hours, _, _, net, contri := parkCalc(parked, car.Money)
	// 按比例分割（对齐 seal.asp：贴条人拿 ratio/10，车主拿剩余）
	mine := net * ratio / 10
	theirs := net - mine
	expMine := hours * ratio / 10
	expTheirs := hours - expMine
	conMine := contri * ratio / 10
	conTheirs := contri - conMine
	if theirs > 0 {
		h.DB.Model(&model.User{}).Where("id = ?", stop.Owner).Update("coins", gorm.Expr("coins + ?", theirs))
		addWalletLog(h.DB, stop.Owner, "park", "被贴车:"+car.Name, "coins", theirs)
	}
	if mine > 0 {
		h.DB.Model(&model.User{}).Where("id = ?", uid).Update("coins", gorm.Expr("coins + ?", mine))
		addWalletLog(h.DB, uid, "park", "贴车:"+car.Name, "coins", mine)
	}
	// 经验/贡献分割
	pMe := h.parkOf(uid)
	if stop.Owner == uid {
		// 自己贴自己车位：经验/贡献全额归自己（对齐 ASP 两次 UPDATE 同一行）
		h.DB.Model(&model.ParkUser{}).Where("id = ?", pMe.ID).Updates(map[string]interface{}{
			"point": gorm.Expr("point + ?", expMine+expTheirs), "contri": gorm.Expr("contri + ?", conMine+conTheirs)})
	} else {
		h.DB.Model(&model.ParkUser{}).Where("id = ?", pMe.ID).Updates(map[string]interface{}{
			"point": gorm.Expr("point + ?", expMine), "contri": gorm.Expr("contri + ?", conMine)})
		pOwner := h.parkOf(stop.Owner)
		h.DB.Model(&model.ParkUser{}).Where("id = ?", pOwner.ID).Updates(map[string]interface{}{
			"point": gorm.Expr("point + ?", expTheirs), "contri": gorm.Expr("contri + ?", conTheirs)})
	}
	h.clearSpot(&stop)
	resp.OK(c, gin.H{"msg": "==贴车成功==", "mine": mine, "theirs": theirs,
		"exp_mine": expMine, "exp_theirs": expTheirs,
		"txt": "自己获得" + strconv.Itoa(mine) + "G币，对方获得" + strconv.Itoa(theirs) + "G币；自己经验+" + strconv.Itoa(expMine) + "，对方经验+" + strconv.Itoa(expTheirs)})
}

// Garage 车库（对齐 my_garage.asp / garage.asp，uid=0 看自己）
func (h *ParkHandler) Garage(c *gin.Context) {
	meUID := middleware.GetUID(c)
	uid, _ := strconv.Atoi(c.Query("uid"))
	if uid <= 0 {
		uid = int(meUID)
	}
	var rows []model.CarGarage
	h.DB.Where("user_id = ?", uid).Order("id DESC").Find(&rows)
	list := make([]gin.H, 0, len(rows))
	for _, g := range rows {
		list = append(list, h.garageInfo(g, uint(uid)))
	}
	resp.OK(c, gin.H{"uid": uint(uid), "is_self": uint(uid) == meUID, "list": list})
}

// Top 排行（对齐 car_top.asp：act 1爱心 2贡献 3经验）
func (h *ParkHandler) Top(c *gin.Context) {
	act, _ := strconv.Atoi(c.DefaultQuery("act", "1"))
	if act < 1 || act > 3 {
		act = 1
	}
	col := "love"
	if act == 2 {
		col = "contri"
	} else if act == 3 {
		col = "point"
	}
	var rows []model.ParkUser
	h.DB.Order(col + " DESC, id ASC").Limit(50).Find(&rows)
	uidSet := map[uint]bool{}
	for _, p := range rows {
		uidSet[p.UserID] = true
	}
	nick := h.nickMap(uidSet)
	out := make([]gin.H, 0, len(rows))
	for i, p := range rows {
		out = append(out, gin.H{"rank": i + 1, "uid": p.UserID, "nick": nick[p.UserID],
			"love": p.Love, "contri": p.Contri, "point": p.Point, "level": parkLevel(p.Point)})
	}
	resp.OK(c, gin.H{"act": act, "list": out})
}

// ---------- 管理端 ----------

// AdminCars 车市车辆列表（全量数组，支持分类/关键词过滤）
func (h *ParkHandler) AdminCars(c *gin.Context) {
	word := c.Query("word")
	dtype, _ := strconv.Atoi(c.Query("dtype"))
	q := h.DB.Model(&model.CarShop{})
	if word != "" {
		q = q.Where("name LIKE ?", "%"+word+"%")
	}
	if dtype >= 1 && dtype <= 5 {
		q = q.Where("dtype = ?", dtype)
	}
	var rows []model.CarShop
	q.Order("dtype ASC, money DESC, id ASC").Find(&rows)
	resp.OK(c, rows)
}

// AdminCarCreate 新增车辆
func (h *ParkHandler) AdminCarCreate(c *gin.Context) {
	var req struct {
		Name   string `json:"name" binding:"required,max=30"`
		Icon   string `json:"icon" binding:"max=10"`
		Price  int    `json:"price"`
		Money  int    `json:"money"`
		DType  int    `json:"dtype"`
		Status int    `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	if req.DType < 1 || req.DType > 5 {
		req.DType = 1
	}
	if req.Price < 0 {
		req.Price = 0
	}
	if req.Money < 0 {
		req.Money = 0
	}
	car := model.CarShop{Name: req.Name, Icon: req.Icon, Price: req.Price, Money: req.Money,
		DType: req.DType, Status: req.Status}
	h.DB.Create(&car)
	resp.OK(c, gin.H{"msg": "车辆已新增", "id": car.ID})
}

// AdminCarUpdate 编辑车辆
func (h *ParkHandler) AdminCarUpdate(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var car model.CarShop
	if err := h.DB.First(&car, id).Error; err != nil {
		resp.NotFound(c, "无此车辆")
		return
	}
	var req struct {
		Name   string `json:"name" binding:"max=30"`
		Icon   string `json:"icon" binding:"max=10"`
		Price  *int   `json:"price"`
		Money  *int   `json:"money"`
		DType  *int   `json:"dtype"`
		Status *int   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Icon != "" {
		updates["icon"] = req.Icon
	}
	if req.Price != nil && *req.Price >= 0 {
		updates["price"] = *req.Price
	}
	if req.Money != nil && *req.Money >= 0 {
		updates["money"] = *req.Money
	}
	if req.DType != nil && *req.DType >= 1 && *req.DType <= 5 {
		updates["dtype"] = *req.DType
	}
	if req.Status != nil {
		updates["status"] = clampInt(*req.Status, 0, 1)
	}
	if len(updates) > 0 {
		h.DB.Model(&car).Updates(updates)
	}
	resp.OK(c, gin.H{"msg": "车辆已更新"})
}

// AdminCarDelete 删除车辆（连同用户车库里的该车）
func (h *ParkHandler) AdminCarDelete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var car model.CarShop
	if err := h.DB.First(&car, id).Error; err != nil {
		resp.NotFound(c, "无此车辆")
		return
	}
	// 已停出的车先收回到流动状态
	var stops []model.CarStop
	h.DB.Where("car_id = ? AND owner > 0", id).Find(&stops)
	for _, s := range stops {
		h.clearSpot(&s)
	}
	var gars []model.CarGarage
	h.DB.Where("car_id = ?", id).Find(&gars)
	for _, g := range gars {
		h.DB.Model(&model.ParkUser{}).Where("user_id = ? AND cars > 0", g.UserID).Update("cars", gorm.Expr("cars - 1"))
	}
	h.DB.Where("car_id = ?", id).Delete(&model.CarGarage{})
	h.DB.Delete(&car)
	resp.OK(c, gin.H{"msg": "车辆已删除（用户车库中的该车已同步清除并扣减持有数）"})
}

// AdminUsers 抢车位用户列表
func (h *ParkHandler) AdminUsers(c *gin.Context) {
	page, offset, size := pageOf(c, 10)
	word := c.Query("word")
	sub := h.DB.Model(&model.User{}).Select("id")
	if word != "" {
		sub = sub.Where("username = ? OR nickname LIKE ?", word, "%"+word+"%")
	}
	var total int64
	h.DB.Model(&model.ParkUser{}).Where("user_id IN (?)", sub).Count(&total)
	var rows []model.ParkUser
	h.DB.Where("user_id IN (?)", sub).Order("point DESC, contri DESC, id ASC").Offset(offset).Limit(size).Find(&rows)
	uidSet := map[uint]bool{}
	for _, p := range rows {
		uidSet[p.UserID] = true
	}
	nick := h.nickMap(uidSet)
	out := make([]gin.H, 0, len(rows))
	for _, p := range rows {
		var garN, parkedN int64
		h.DB.Model(&model.CarGarage{}).Where("user_id = ?", p.UserID).Count(&garN)
		h.DB.Model(&model.CarStop{}).Where("user_id = ? AND owner > 0", p.UserID).Count(&parkedN)
		out = append(out, gin.H{"user_id": p.UserID, "nickname": nick[p.UserID],
			"cars": p.Cars, "love": p.Love, "point": p.Point, "level": parkLevel(p.Point),
			"contri": p.Contri, "garage": garN, "parked": parkedN})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}

// AdminUserEdit 编辑玩家数据（爱心/经验/贡献）
func (h *ParkHandler) AdminUserEdit(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	var p model.ParkUser
	if err := h.DB.Where("user_id = ?", uid).First(&p).Error; err != nil {
		resp.ParamError(c, "该用户还未开通抢车位")
		return
	}
	var req struct {
		Love   *int `json:"love"`
		Point  *int `json:"point"`
		Contri *int `json:"contri"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ParamError(c, "参数错误")
		return
	}
	updates := map[string]interface{}{}
	if req.Love != nil && *req.Love >= 0 {
		updates["love"] = *req.Love
	}
	if req.Point != nil && *req.Point >= 0 {
		updates["point"] = *req.Point
	}
	if req.Contri != nil && *req.Contri >= 0 {
		updates["contri"] = *req.Contri
	}
	if len(updates) > 0 {
		h.DB.Model(&p).Updates(updates)
	}
	resp.OK(c, gin.H{"msg": "玩家数据已更新"})
}

// AdminSpotClear 清空用户车位（收回占用车）
func (h *ParkHandler) AdminSpotClear(c *gin.Context) {
	uid, _ := strconv.Atoi(c.Param("uid"))
	id, _ := strconv.Atoi(c.Param("id"))
	var stop model.CarStop
	if err := h.DB.Where("id = ? AND user_id = ?", id, uid).First(&stop).Error; err != nil {
		resp.NotFound(c, "无此车位")
		return
	}
	if stop.Owner == 0 {
		resp.ParamError(c, "该车位本来就是空的")
		return
	}
	h.clearSpot(&stop)
	resp.OK(c, gin.H{"msg": "车位已清空，车辆回到流动状态"})
}

// AdminLogs 日志：type=join(加入记录) msg(游戏消息)
func (h *ParkHandler) AdminLogs(c *gin.Context) {
	page, offset, size := pageOf(c, 15)
	ty := c.DefaultQuery("type", "join")
	word := c.Query("word")
	if ty == "msg" {
		q := h.DB.Model(&model.CarMsg{})
		if word != "" {
			q = q.Where("user_id IN (SELECT id FROM users WHERE username = ? OR nickname LIKE ?) OR content LIKE ?", word, "%"+word+"%", "%"+word+"%")
		}
		var total int64
		q.Count(&total)
		var rows []model.CarMsg
		h.DB.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
		uidSet := map[uint]bool{}
		for _, r := range rows {
			uidSet[r.UserID] = true
			if r.FID > 0 {
				uidSet[r.FID] = true
			}
		}
		nick := h.nickMap(uidSet)
		out := make([]gin.H, 0, len(rows))
		for _, r := range rows {
			from := "系统"
			if r.FID > 0 {
				from = nick[r.FID]
			}
			out = append(out, gin.H{"id": r.ID, "to_uid": r.UserID, "to": nick[r.UserID], "from": from,
				"content": r.Content, "status": r.Status, "created_at": r.CreatedAt.Format("2006-01-02 15:04:05")})
		}
		resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
		return
	}
	// 加入记录
	q := h.DB.Model(&model.CarLog{})
	if word != "" {
		q = q.Where("name LIKE ? OR user_id IN (SELECT id FROM users WHERE username = ? OR nickname LIKE ?)", "%"+word+"%", word, "%"+word+"%")
	}
	var total int64
	q.Count(&total)
	var rows []model.CarLog
	h.DB.Order("id DESC").Offset(offset).Limit(size).Find(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		out = append(out, gin.H{"id": r.ID, "user_id": r.UserID, "name": r.Name,
			"created_at": r.CreatedAt.Format("2006-01-02 15:04:05")})
	}
	resp.OK(c, gin.H{"total": total, "page": page, "size": size, "list": out})
}
