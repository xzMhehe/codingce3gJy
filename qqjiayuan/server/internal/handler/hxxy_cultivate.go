package handler

// 人物修炼模块（复刻原版 xy427/428/432/433 + zbdz/xlms.php + wj/xlxx01.php）
// 四条修炼线：1血 2攻 3魔 4防；升级消耗修炼经验+银两+西游声望（出窍起加金豆）
// 修炼经验来源：开启修炼开关后，战斗经验存入修炼经验（原版 xy052 修炼经验开关）

import (
	"fmt"
	"math"

	"github.com/gin-gonic/gin"

	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/resp"
)

// hxXlSlotName 修炼线名称
var hxXlSlotName = map[int]string{1: "血", 2: "攻", 3: "魔", 4: "防"}

// hxXlRealm 下一级 xldj 的境界与升级消耗（复刻 xlms.php）
// 返回：境界名, 层, 银两, 声望, 金豆, 是否封顶
func hxXlRealm(xldj int) (string, int, int64, int64, int64, bool) {
	if xldj >= 541 {
		return "天尊", 20, 0, 0, 0, true
	}
	type realm struct {
		lo, hi          int
		name            string
		silver          int64
		swBase, swStep  int64
		beanBase        int64
	}
	rs := []realm{
		{1, 20, "炼气", 1000, 4000, 1000, 0},
		{21, 40, "筑基", 50000, 5000, 2000, 0},
		{41, 60, "开光", 1000000, 10000, 3000, 0},
		{61, 80, "金丹", 2000000, 20000, 4000, 0},
		{81, 100, "元婴", 5000000, 30000, 5000, 0},
		{101, 120, "出窍", 10000000, 40000, 6000, 1},
		{121, 140, "合体", 20000000, 50000, 7000, 5},
		{141, 160, "渡劫", 50000000, 60000, 8000, 10},
		{161, 180, "寂灭", 100000000, 70000, 9000, 15},
		{181, 200, "大乘", 200000000, 100000, 10000, 30},
		{201, 220, "仙人", 400000000, 100000, 10000, 30},
		{221, 240, "散仙", 600000000, 120000, 12000, 40},
		{241, 260, "上仙", 800000000, 140000, 14000, 50},
		{261, 280, "地仙", 1000000000, 160000, 16000, 60},
		{281, 300, "天仙", 1200000000, 180000, 18000, 70},
		{301, 320, "金仙", 1400000000, 200000, 20000, 80},
		{321, 340, "玄仙", 1600000000, 220000, 22000, 90},
		{341, 360, "初神", 1800000000, 240000, 24000, 100},
		{361, 380, "偏神", 2000000000, 260000, 26000, 110},
		{381, 400, "神人", 2200000000, 280000, 28000, 120},
		{401, 420, "准神", 2400000000, 300000, 30000, 110},
		{421, 440, "神使", 2600000000, 320000, 32000, 120},
		{441, 460, "神将", 2800000000, 340000, 34000, 130},
		{461, 480, "上神", 3000000000, 360000, 36000, 140},
		{481, 500, "天神", 3200000000, 380000, 38000, 150},
		{501, 520, "神王", 3400000000, 400000, 40000, 160},
		{521, 540, "天尊", 4000000000, 500000, 50000, 200},
	}
	for _, r := range rs {
		if xldj >= r.lo && xldj <= r.hi {
			layer := xldj - r.lo + 1
			beans := int64(0)
			if r.beanBase > 0 {
				beans = r.beanBase + int64(layer) // 原版：出窍起才需要金豆（beanBase=0 的境界金豆为0）
			}
			return r.name, layer, r.silver, r.swBase + int64(layer)*r.swStep, beans, false
		}
	}
	return "天尊", 20, 0, 0, 0, true
}

// hxXlExpNeed 升到 xldj 级所需修炼经验（复刻 xlms.php：xlxq=(xldj+1)³(xldj+2)+200）
func hxXlExpNeed(xldj int) int {
	l := float64(xldj)
	return int(math.Pow(l+1, 3)*(l+2)) + 200
}

// hxXlBonus 修炼加成（复刻 xlxx01.php）
// 血:(lv+20)³+800 攻:(lv+1)²×80+200 魔:(lv+1)²×80+200 防:(lv+1)²×70+200
func hxXlBonus(slot, lv int) int64 {
	if lv < 1 {
		return 0
	}
	switch slot {
	case 1:
		return int64(math.Pow(float64(lv+20), 3)) + 800
	case 2:
		return int64(math.Pow(float64(lv+1), 2))*80 + 200
	case 3:
		return int64(math.Pow(float64(lv+1), 2))*80 + 200
	case 4:
		return int64(math.Pow(float64(lv+1), 2))*70 + 200
	}
	return 0
}

// hxXlTracks 玩家四线修炼视图
func (h *HxxyHandler) hxXlTracks(p *model.HxxyPlayer) []gin.H {
	lvs := [4]int{p.XlLv1, p.XlLv2, p.XlLv3, p.XlLv4}
	tracks := []gin.H{}
	for i := 1; i <= 4; i++ {
		lv := lvs[i-1]
		next := lv
		capped := false
		realmName := ""
		layer, silver, sw, beans := 0, int64(0), int64(0), int64(0)
		if lv < 541 {
			next = lv + 1
			realmName, layer, silver, sw, beans, _ = hxXlRealm(next)
		} else {
			realmName, layer, _, _, _, capped = hxXlRealm(lv)
		}
		tr := gin.H{
			"slot": i, "name": hxXlSlotName[i], "lv": lv,
			"bonus": hxXlBonus(i, lv), "capped": capped,
			"realm": realmName, "layer": layer,
		}
		if !capped {
			tr["need"] = gin.H{"exp": hxXlExpNeed(next), "silver": silver, "sw": sw, "beans": beans}
		}
		tracks = append(tracks, tr)
	}
	return tracks
}

// Cultivate 修炼主页（复刻 xy427）
func (h *HxxyHandler) Cultivate(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	resp.OK(c, gin.H{
		"switch": p.XiulianSwitch, "exp": p.XiulianExp, "sw": p.Sw,
		"tracks": h.hxXlTracks(p),
	})
}

// CultivateToggle 修炼经验开关（复刻 xy052 原版文案）
func (h *HxxyHandler) CultivateToggle(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	sw, msg := 0, ""
	if p.XiulianSwitch == 0 {
		sw = 1
		msg = "你打开了修炼经验将获得修炼经验(关闭后获得经验)"
	} else {
		msg = "你关闭了修炼经验将获得经验(开启后获得修炼经验)"
	}
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).Update("xiulian_switch", sw)
	p.XiulianSwitch = sw
	resp.OK(c, gin.H{"msg": msg, "switch": sw})
}

// CultivateUpgrade 开始修炼（升级指定线，复刻 xy433）
func (h *HxxyHandler) CultivateUpgrade(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	var in struct {
		Slot int `json:"slot"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Slot < 1 || in.Slot > 4 {
		resp.ParamError(c, "参数错误")
		return
	}
	name := hxXlSlotName[in.Slot]
	cur := [4]int{p.XlLv1, p.XlLv2, p.XlLv3, p.XlLv4}[in.Slot-1]
	if cur >= 541 {
		resp.ParamError(c, "已达到至高无上的境界了")
		return
	}
	next := cur + 1
	_, _, silver, sw, beans, _ := hxXlRealm(next)
	expNeed := hxXlExpNeed(next)
	if p.XiulianExp < expNeed || p.Money < silver || p.Sw < sw || p.Beans < int(beans) {
		resp.ParamError(c, fmt.Sprintf("对不起！！【人物修炼（%s）】失败！！需要：修炼经验%d，银两%d，西游声望%d%s",
			name, expNeed, silver, sw, func() string {
				if beans > 0 {
					return fmt.Sprintf("，〖金豆〗x%d", beans)
				}
				return ""
			}()))
		return
	}
	// 扣消耗
	p.XiulianExp -= expNeed
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).UpdateColumn("xiulian_exp", p.XiulianExp)
	h.hxWallet(p, "money", -silver, "人物修炼（"+name+"）")
	if sw > 0 {
		h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).UpdateColumn("sw", p.Sw-sw)
		p.Sw -= sw
	}
	if beans > 0 {
		h.hxWallet(p, "beans", -beans, "人物修炼（"+name+"）")
	}
	// 升级
	col := fmt.Sprintf("xl_lv%d", in.Slot)
	newLv := cur
	if cur == 0 {
		newLv = cur + 2 // 原版：0 级升完直接 2 级
	} else {
		newLv = cur + 1
	}
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).UpdateColumn(col, newLv)
	switch in.Slot {
	case 1:
		p.XlLv1 = newLv
	case 2:
		p.XlLv2 = newLv
	case 3:
		p.XlLv3 = newLv
	case 4:
		p.XlLv4 = newLv
	}
	msg := fmt.Sprintf("恭喜你！！【人物修炼（%s）】成功，实力大幅度提升 失去：修炼经验%d，银两%d，西游声望%d%s",
		name, expNeed, silver, sw, func() string {
			if beans > 0 {
				return fmt.Sprintf("，〖金豆〗x%d", beans)
			}
			return ""
		}())
	resp.OK(c, gin.H{"msg": msg, "lv": newLv})
}

// CultivateExchangeDan 【一键】各类修炼丹兑换修炼经验（复刻 xy647.php）
// 625 〖1亿修炼经验丹〗=1亿 / 626 〖5亿修炼经验丹〗=5亿 / 627 〖10亿修炼经验丹〗=10亿，一键全部兑换
func (h *HxxyHandler) CultivateExchangeDan(c *gin.Context) {
	p := h.hxPlayer(c)
	if p == nil {
		resp.ParamError(c, "请先创建角色")
		return
	}
	dans := map[uint]int64{625: 100000000, 626: 500000000, 627: 1000000000}
	var bags []model.HxxyBag
	h.DB.Where("player_id = ? AND store = 0 AND kind = 'item' AND ref_id IN ?", p.ID, []uint{625, 626, 627}).Find(&bags)
	var gain int64
	for _, b := range bags {
		per, ok := dans[b.RefID]
		if !ok || b.Count <= 0 {
			continue
		}
		gain += per * int64(b.Count)
		h.DB.Delete(&model.HxxyBag{}, b.ID)
	}
	if gain <= 0 {
		resp.OK(c, gin.H{"msg": "对不起,你没有修炼丹可兑换"})
		return
	}
	h.DB.Model(&model.HxxyPlayer{}).Where("id = ?", p.ID).UpdateColumn("xiulian_exp", p.XiulianExp+int(gain))
	p.XiulianExp += int(gain)
	resp.OK(c, gin.H{"msg": fmt.Sprintf("获得：%d修炼经验", gain), "gain": gain})
}