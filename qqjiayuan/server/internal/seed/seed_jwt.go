package seed

import (
	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// seedJwt 精武堂基础数据（幂等：按 name 去重插入）
func seedJwt(db *gorm.DB) {
	seedJwtSkills(db)
	seedJwtItems(db)
}

func seedJwtSkills(db *gorm.DB) {
	skills := []model.JwtSkill{
		{Name: "排山倒海", Act: 1, Level: 1, Price: 1000, Coef: 100, Hit: 90, Crit: 10, Desc: "排山倒海", WeaponReq: "无限制"},
		{Name: "降龙十八掌", Act: 1, Level: 5, Price: 20000, Coef: 5000, Hit: 85, Crit: 15, Desc: "降龙十八掌威力巨猛", WeaponReq: "无限制"},
		{Name: "剑影留痕", Act: 0, Level: 5, Price: 30000, PAtk: 20, PHp: 50, Desc: "被动：攻击+20 气血+50"},
		{Name: "御剑通灵", Act: 0, Level: 5, Price: 30000, PDef: 20, PMp: 50, Desc: "被动：防御+20 气力+50"},
		{Name: "人剑合一", Act: 0, Level: 5, Price: 30000, PHp: 100, Crit: 10, Desc: "被动：气血+100 暴击+10%"},
		{Name: "潇湘剑雨", Act: 1, Level: 10, Price: 10, Coef: 3000, Hit: 88, Crit: 20, Desc: "潇湘剑雨剑气纵横", WeaponReq: "无限制", Currency: "yuanbao"},
	}
	for i := range skills {
		var n int64
		db.Model(&model.JwtSkill{}).Where("name = ?", skills[i].Name).Count(&n)
		if n == 0 {
			db.Create(&skills[i])
		} else {
			db.Model(&model.JwtSkill{}).Where("name = ?", skills[i].Name).Updates(map[string]interface{}{
				"act": skills[i].Act, "level": skills[i].Level, "price": skills[i].Price,
				"coef": skills[i].Coef, "hit": skills[i].Hit, "crit": skills[i].Crit,
				"p_atk": skills[i].PAtk, "p_def": skills[i].PDef, "p_hp": skills[i].PHp, "p_mp": skills[i].PMp,
				"currency": skills[i].Currency, "desc": skills[i].Desc,
			})
		}
	}
}

func seedJwtItems(db *gorm.DB) {
	// 商店（G币 coins / 元宝 yuanbao）
	items := []model.JwtItem{
		// 材料
		{Name: "黄金", Cat: "material", Src: "shop", Price: 100, Currency: "yuanbao", Desc: "锻造高级装备的贵重金属"},
		{Name: "白鳞", Cat: "material", Src: "shop", Price: 60, Currency: "yuanbao", Desc: "锻造材料"},
		{Name: "翡翠", Cat: "material", Src: "shop", Price: 60, Currency: "yuanbao", Desc: "锻造材料"},
		{Name: "鹿皮", Cat: "material", Src: "shop", Price: 60, Currency: "yuanbao", Desc: "锻造材料"},
		{Name: "猛虎皮", Cat: "material", Src: "shop", Price: 90, Currency: "yuanbao", Desc: "锻造材料"},
		{Name: "长绒毛皮", Cat: "material", Src: "shop", Price: 90, Currency: "yuanbao", Desc: "锻造材料"},
		{Name: "短绒毛皮", Cat: "material", Src: "shop", Price: 40, Currency: "yuanbao", Desc: "锻造材料"},
		{Name: "麒麟火", Cat: "material", Src: "shop", Price: 90, Currency: "yuanbao", Desc: "锻造材料"},
		{Name: "普通蚕丝", Cat: "material", Src: "shop", Price: 40, Currency: "yuanbao", Desc: "锻造材料"},
		{Name: "普通皮革", Cat: "material", Src: "shop", Price: 40, Currency: "yuanbao", Desc: "锻造材料"},
		{Name: "三等棉布", Cat: "material", Src: "shop", Price: 90, Currency: "yuanbao", Desc: "锻造材料"},
		{Name: "四等棉布", Cat: "material", Src: "shop", Price: 60, Currency: "yuanbao", Desc: "锻造材料"},
		{Name: "五等棉布", Cat: "material", Src: "shop", Price: 40, Currency: "yuanbao", Desc: "锻造材料"},
		{Name: "白鳞发带图", Cat: "material", Src: "shop", Price: 90, Currency: "yuanbao", Desc: "白鳞发带图纸"},
		{Name: "沧月薄衣图", Cat: "material", Src: "shop", Price: 150, Currency: "yuanbao", Desc: "沧月薄衣图纸"},
		{Name: "飞龙腰带图", Cat: "material", Src: "shop", Price: 150, Currency: "yuanbao", Desc: "飞龙腰带图纸"},
		{Name: "翡翠项链图", Cat: "material", Src: "shop", Price: 90, Currency: "yuanbao", Desc: "翡翠项链图纸"},
		{Name: "锋灵剑模", Cat: "material", Src: "shop", Price: 70, Currency: "yuanbao", Desc: "锋灵剑模具"},
		{Name: "锋灵剑图", Cat: "material", Src: "shop", Price: 90, Currency: "yuanbao", Desc: "锋灵剑图纸"},
		{Name: "疾影闪巾图", Cat: "material", Src: "shop", Price: 60, Currency: "yuanbao", Desc: "疾影闪巾图纸"},
		{Name: "流光剑模", Cat: "material", Src: "shop", Price: 120, Currency: "yuanbao", Desc: "流光剑模具"},
		{Name: "流光剑图", Cat: "material", Src: "shop", Price: 150, Currency: "yuanbao", Desc: "流光剑图纸"},
		{Name: "鹿皮靴图", Cat: "material", Src: "shop", Price: 90, Currency: "yuanbao", Desc: "鹿皮靴图纸"},
		{Name: "猛虎靴图", Cat: "material", Src: "shop", Price: 150, Currency: "yuanbao", Desc: "猛虎靴图纸"},
		{Name: "麒麟冠图", Cat: "material", Src: "shop", Price: 150, Currency: "yuanbao", Desc: "麒麟冠图纸"},
		{Name: "嵌甲靴图", Cat: "material", Src: "shop", Price: 60, Currency: "yuanbao", Desc: "嵌甲靴图纸"},
		{Name: "松纹剑模", Cat: "material", Src: "shop", Price: 50, Currency: "yuanbao", Desc: "松纹剑模具"},
		{Name: "松纹剑图", Cat: "material", Src: "shop", Price: 60, Currency: "yuanbao", Desc: "松纹剑图纸"},
		{Name: "耀瞳华服图", Cat: "material", Src: "shop", Price: 60, Currency: "yuanbao", Desc: "耀瞳华服图纸"},
		{Name: "幽冥冷衫图", Cat: "material", Src: "shop", Price: 90, Currency: "yuanbao", Desc: "幽冥冷衫图纸"},
		{Name: "诸葛之魂图", Cat: "material", Src: "shop", Price: 60, Currency: "yuanbao", Desc: "诸葛之魂图纸"},
		{Name: "帮派令旗", Cat: "other", Src: "shop", Price: 200, Currency: "yuanbao", Desc: "创建帮派所需的令旗"},
		// 比武材料（src=battle，仅比武胜利掉落，不出售）
		{Name: "丝麻线", Cat: "material", Src: "battle", Price: 0, Desc: "比武材料"},
		{Name: "精铁", Cat: "material", Src: "battle", Price: 0, Desc: "比武材料"},
		{Name: "精钢", Cat: "material", Src: "battle", Price: 0, Desc: "比武材料"},
		{Name: "白纱", Cat: "material", Src: "battle", Price: 0, Desc: "比武材料"},
		{Name: "粗铁", Cat: "material", Src: "battle", Price: 0, Desc: "比武材料"},
		{Name: "竹炭纤维", Cat: "material", Src: "battle", Price: 0, Desc: "比武材料"},
		{Name: "竹丝线", Cat: "material", Src: "battle", Price: 0, Desc: "比武材料"},
		{Name: "黄麻线", Cat: "material", Src: "battle", Price: 0, Desc: "比武材料"},
		{Name: "纤维绳", Cat: "material", Src: "battle", Price: 0, Desc: "比武材料"},
		{Name: "寒铁石", Cat: "material", Src: "battle", Price: 0, Desc: "比武材料"},
		{Name: "尼龙线", Cat: "material", Src: "battle", Price: 0, Desc: "比武材料"},
		{Name: "白玉", Cat: "material", Src: "battle", Price: 0, Desc: "比武材料"},
		{Name: "亚麻线", Cat: "material", Src: "battle", Price: 0, Desc: "比武材料"},
		{Name: "石英", Cat: "material", Src: "battle", Price: 0, Desc: "比武材料"},
		// 药品
		{Name: "气血丸1号", Cat: "medicine", Src: "shop", Price: 1000, RecoverHp: 50, Desc: "恢复50点气血"},
		{Name: "气血丸2号", Cat: "medicine", Src: "shop", Price: 2000, RecoverHp: 150, Desc: "恢复150点气血"},
		{Name: "气血丸3号", Cat: "medicine", Src: "shop", Price: 5000, RecoverHp: 500, Desc: "恢复500点气血"},
		{Name: "气力丸1号", Cat: "medicine", Src: "shop", Price: 200, RecoverMp: 30, Desc: "恢复30点气力"},
		{Name: "气力丸2号", Cat: "medicine", Src: "shop", Price: 400, RecoverMp: 80, Desc: "恢复80点气力"},
		{Name: "气力丸3号", Cat: "medicine", Src: "shop", Price: 600, RecoverMp: 200, Desc: "恢复200点气力"},
		// 武器
		{Name: "木剑", Cat: "weapon", Src: "shop", Price: 10000, Atk: 8, Desc: "新手佩剑"},
		{Name: "铁剑", Cat: "weapon", Src: "shop", Price: 50000, Atk: 20, Desc: "锋利铁剑"},
		{Name: "金剑", Cat: "weapon", Src: "shop", Price: 100000, Atk: 40, Desc: "削铁如泥"},
		// 防具(头盔)
		{Name: "铁盾", Cat: "helmet", Src: "shop", Price: 10000, Def: 8, Hp: 20, Desc: "铁制护盾"},
		{Name: "钢盾", Cat: "helmet", Src: "shop", Price: 100000, Def: 20, Hp: 60, Desc: "钢制护盾"},
		{Name: "铜盾", Cat: "helmet", Src: "shop", Price: 1000000, Def: 40, Hp: 150, Desc: "铜制护盾"},
		// 盔甲
		{Name: "晦光华服", Cat: "armor", Src: "shop", Price: 100, Currency: "yuanbao", Def: 12, Hp: 50, Desc: "夜行华服"},
		{Name: "追影风衣", Cat: "armor", Src: "shop", Price: 300, Currency: "yuanbao", Def: 20, Hp: 90, Spd: 5, Desc: "轻灵风衣"},
		{Name: "锦绣华服", Cat: "armor", Src: "shop", Price: 600, Currency: "yuanbao", Def: 30, Hp: 150, Spd: 10, Desc: "锦绣华服"},
		// 战鞋
		{Name: "草鞋", Cat: "shoes", Src: "shop", Price: 10000, Spd: 3, Desc: "轻便草鞋"},
		{Name: "布鞋", Cat: "shoes", Src: "shop", Price: 100, Currency: "yuanbao", Spd: 6, Desc: "舒适布鞋"},
		{Name: "乌皮靴", Cat: "shoes", Src: "shop", Price: 400, Currency: "yuanbao", Spd: 12, Desc: "乌皮战靴"},
		// 配饰(项链/手镯/戒指/勋章)
		{Name: "新手项链", Cat: "necklace", Src: "shop", Price: 10000, Hp: 30, Mp: 20, Desc: "新手项链"},
		{Name: "新手手镯", Cat: "bracelet", Src: "shop", Price: 10000, Mp: 30, Spd: 2, Desc: "新手手镯"},
		{Name: "草环戒指", Cat: "ring", Src: "shop", Price: 10000, Atk: 5, Def: 5, Desc: "草环戒指"},
		{Name: "新手勋章", Cat: "medal", Src: "shop", Price: 10000, Hp: 20, Atk: 3, Desc: "新手勋章"},

		// ---------- 锻造产物（src=forge，图纸+材料合成） ----------
		{Name: "耀瞳华服", Cat: "armor", Src: "forge", Level: 30, Fee: 200, Mats: "耀瞳华服图x1,五等棉布x1,丝麻线x3", Def: 40, Hp: 200, Desc: "锻造盔甲"},
		{Name: "松纹剑", Cat: "weapon", Src: "forge", Level: 30, Fee: 200, Mats: "松纹剑图x1,松纹剑模x1,精铁x3", Atk: 45, Desc: "锻造武器"},
		{Name: "疾影闪巾", Cat: "helmet", Src: "forge", Level: 30, Fee: 150, Mats: "疾影闪巾图x1,普通蚕丝x1,白纱x2", Def: 30, Hp: 120, Spd: 8, Desc: "锻造头盔"},
		{Name: "嵌甲靴", Cat: "shoes", Src: "forge", Level: 30, Fee: 150, Mats: "嵌甲靴图x1,普通皮革x1,粗铁x3", Spd: 18, Def: 18, Desc: "锻造战鞋"},
		{Name: "诸葛之魂", Cat: "necklace", Src: "forge", Level: 30, Fee: 200, Mats: "诸葛之魂图x1,短绒毛皮x1,竹炭纤维x2", Hp: 180, Mp: 120, Desc: "锻造项链"},
		{Name: "锋灵剑", Cat: "weapon", Src: "forge", Level: 40, Fee: 200, Mats: "锋灵剑图x1,锋灵剑模x1,精钢x3", Atk: 70, Desc: "锻造武器"},
		{Name: "幽冥冷衫", Cat: "armor", Src: "forge", Level: 40, Fee: 200, Mats: "幽冥冷衫图x1,四等棉布x1,竹丝线x3", Def: 60, Hp: 320, Desc: "锻造盔甲"},
		{Name: "白鳞发带", Cat: "helmet", Src: "forge", Level: 40, Fee: 150, Mats: "白鳞发带图x1,白鳞x1,白纱x2", Def: 45, Hp: 180, Desc: "锻造头盔"},
		{Name: "鹿皮靴", Cat: "shoes", Src: "forge", Level: 40, Fee: 150, Mats: "鹿皮靴图x1,鹿皮x1,黄麻线x2", Spd: 28, Def: 26, Desc: "锻造战鞋"},
		{Name: "翡翠项链", Cat: "necklace", Src: "forge", Level: 40, Fee: 200, Mats: "翡翠项链图x1,翡翠x1,纤维绳x2", Hp: 280, Mp: 180, Atk: 15, Desc: "锻造项链"},
		{Name: "流光剑", Cat: "weapon", Src: "forge", Level: 50, Fee: 300, Mats: "流光剑图x1,流光剑模x1,寒铁石x3", Atk: 110, Spd: 12, Desc: "锻造武器"},
		{Name: "沧月薄衣", Cat: "armor", Src: "forge", Level: 50, Fee: 300, Mats: "沧月薄衣图x1,三等棉布x1,尼龙线x3", Def: 90, Hp: 480, Desc: "锻造盔甲"},
		{Name: "麒麟冠", Cat: "helmet", Src: "forge", Level: 50, Fee: 250, Mats: "麒麟冠图x1,麒麟火x1,白玉x2", Def: 70, Hp: 260, Spd: 12, Desc: "锻造头盔"},
		{Name: "猛虎靴", Cat: "shoes", Src: "forge", Level: 50, Fee: 250, Mats: "猛虎靴图x1,猛虎皮x1,亚麻线x2", Spd: 40, Def: 40, Desc: "锻造战鞋"},
		{Name: "飞龙腰带", Cat: "necklace", Src: "forge", Level: 50, Fee: 300, Mats: "飞龙腰带图x1,长绒毛皮x1,石英x2", Hp: 420, Mp: 260, Atk: 25, Spd: 15, Desc: "锻造项链"},
	}
	for i := range items {
		var n int64
		db.Model(&model.JwtItem{}).Where("name = ?", items[i].Name).Count(&n)
		if n == 0 {
			db.Create(&items[i])
		} else {
			db.Model(&model.JwtItem{}).Where("name = ?", items[i].Name).Updates(map[string]interface{}{
				"cat": items[i].Cat, "src": items[i].Src, "price": items[i].Price,
				"currency": items[i].Currency, "level": items[i].Level, "atk": items[i].Atk,
				"def": items[i].Def, "hp": items[i].Hp, "mp": items[i].Mp, "spd": items[i].Spd,
				"hit": items[i].Hit, "crit": items[i].Crit, "dodge": items[i].Dodge,
				"recover_hp": items[i].RecoverHp, "recover_mp": items[i].RecoverMp,
				"mats": items[i].Mats, "fee": items[i].Fee, "desc": items[i].Desc,
			})
		}
	}
}