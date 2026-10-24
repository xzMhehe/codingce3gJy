package seed

import (
	"log"
	"strconv"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// ★★ 2026-10-24 存量在途订单 officer_id 一次性回填（幂等，settings 标记位只跑一次）。
//
// 背景：命令表 ezfy_order.officer 历史上只存军官**名字**（不存 id），历史遗留的同城同名军官
// （线上实测 William·Rogers×4 等）让所有按名字关联的运行逻辑互相误伤 —— 出征中的「张三」
// 会让在城的另一个「张三」改名/出征全被拦（用户反馈的 bug 根源）。
// 2026-10-24 起新订单落库同时写 officer_id；本迁移把存量**在途**订单（与 orderListByCity 同口径：
// status IN (0,1,2,5,6)，或 status=3 的驻守增援）关联军官并回填 officer_id —— 回填后运行时
// 全部改走 id 判断，**彻底消灭按名字关联**（连同名字兜底一起删掉，见 ezfy_officer.go）。
//
// ★ 线上数据安全（1区/2区）：
//   - 标记位是**按库**的 settings 行，两个区各自独立库则各自启动时各跑一次；共享库则只跑一次；
//   - 第一步纯 SQL JOIN 一次 UPDATE（内存零负担，不整表拉进内存），只处理「同城同名唯一」；
//   - 第二步只对**剩余少量**歧义行（同名多名 / 无匹配）在 Go 里消歧，数据量极小；
//   - 全程只 UPDATE officer_id 一列，不碰任何玩家业务数据；失败/中断重启后 WHERE officer_id=0 续跑；
//   - 回滚预案：若线上发现回填异常，可 `UPDATE ezfy_order SET officer_id=0 WHERE officer_id>0 AND
//     (status IN (0,1,2,5,6) OR (status=3 AND order_type=6 AND IFNULL(result,'')=''))` 回退，
//     并 DELETE 标记位后重启重跑（新代码对 id=0 不再按名字关联，最坏退化成「孤儿订单不占外出」，
//     不会崩数据、不影响其他玩家）。
//
// 匹配规则：
//   - 名字唯一 → 第一步 SQL 直接回填；
//   - 同名多名 → 第二步优先挑「出征中(status=1)」那位（在途命令的带队军官应当正处于出征态）；
//     再不行按确定性次序兜底（非俘虏 > 无职位 > 等级高 > 更新时间新 > id 小）；
//   - 无同名军官（军官已改名/删除的孤儿订单）→ 留 officer_id=0 并打日志，运行时不按名字关联
//     （名字也关联不到，行为与「无兜底」一致，订单正常返航结算，只是带队军官加成归零）。
func EnsureEzfyOrderOfficerIdBackfill(db *gorm.DB) {
	if db == nil || !db.Migrator().HasTable("ezfy_order") || !db.Migrator().HasColumn("ezfy_order", "officer_id") {
		return
	}
	var done int64
	db.Model(&model.Setting{}).Where("`key` = ?", ezfyOrderOfficerIdBackfillKey).Count(&done)
	if done > 0 {
		return
	}
	// ★ 第一步：同城同名唯一 → 纯 SQL JOIN 一次回填（不把整表拉进内存，线上大库也安全）。
	// 同名多名/无匹配的行 UPDATE 不动（JOIN 匹配不上），自然留给第二步。
	// 语句内 IFNULL 兼容 MySQL 8（result 列允许 NULL）。
	res := db.Exec(`UPDATE ezfy_order o
		JOIN (SELECT MIN(oc.id) AS id, oc.city_id, oc.name
			FROM ezfy_officer oc
			GROUP BY oc.city_id, oc.name HAVING COUNT(*) = 1) s
			ON s.city_id = o.city_id AND s.name = o.officer
		SET o.officer_id = s.id
		WHERE o.officer_id = 0 AND o.officer != ''
		  AND (o.status IN (0,1,2,5,6) OR (o.status = 3 AND o.order_type = 6 AND IFNULL(o.result,'') = ''))`)
	if res.Error != nil {
		log.Printf("[seed] 在途订单 officer_id 回填(第一步 SQL)失败: %v", res.Error)
		return
	}
	step1 := res.RowsAffected
	// ★ 第二步：只取剩余的歧义行（同名多名 / 无匹配），数据量极小，全内存关联。
	var orders []model.EzfyOrder
	db.Select("id, city_id, officer, officer_id").
		Where("officer_id = 0 AND officer != '' AND (status IN (0,1,2,5,6) OR (status = 3 AND order_type = 6 AND IFNULL(result,'') = ''))").
		Find(&orders)
	if len(orders) == 0 {
		log.Printf("[seed] 在途订单 officer_id 回填完成: SQL 回填 %d 条, 无歧义行", step1)
		markOrderOfficerBackfillDone(db)
		return
	}
	// 歧义行涉及的军官：按 city_id 一次取回（城市数远小于订单数）
	cityIDs := map[int64]bool{}
	for _, ord := range orders {
		cityIDs[ord.CityId] = true
	}
	ids := make([]int64, 0, len(cityIDs))
	for cid := range cityIDs {
		ids = append(ids, cid)
	}
	var officers []model.EzfyOfficer
	db.Select("id, city_id, name, status, position, is_captive, level, update_time").
		Where("city_id IN ?", ids).Find(&officers)
	byKey := map[string][]model.EzfyOfficer{}
	for i := range officers {
		o := officers[i]
		key := strconv.FormatInt(o.CityId, 10) + "|" + o.Name
		byKey[key] = append(byKey[key], o)
	}
	type backfillHit struct{ orderID, officerID uint }
	hits := []backfillHit{}
	dupN := 0
	missed := []model.EzfyOrder{}
	for _, ord := range orders {
		cand := byKey[strconv.FormatInt(ord.CityId, 10)+"|"+ord.Officer]
		switch {
		case len(cand) == 0:
			missed = append(missed, ord)
		case len(cand) == 1:
			// SQL 第一步没处理到的唯一名行（并发间隙）直接回填
			hits = append(hits, backfillHit{ord.ID, cand[0].ID})
		default:
			if p := pickOrderOfficer(cand); p != nil {
				hits = append(hits, backfillHit{ord.ID, p.ID})
				dupN++
			} else {
				missed = append(missed, ord)
			}
		}
	}
	for _, hit := range hits {
		db.Model(&model.EzfyOrder{}).Where("id = ? AND officer_id = 0", hit.orderID).Update("officer_id", hit.officerID)
	}
	log.Printf("[seed] 在途订单 officer_id 回填完成: SQL 回填 %d, 歧义消歧 %d, 无匹配 %d (共处理 %d)",
		step1, dupN, len(missed), step1+int64(len(hits)))
	for _, ord := range missed {
		log.Printf("[seed] 在途订单无同名军官可关联（军官已改名/删除）保留 officer_id=0: order#%d city=%d name=%q", ord.ID, ord.CityId, ord.Officer)
	}
	markOrderOfficerBackfillDone(db)
}

// pickOrderOfficer 同名候选里挑「最可能是带队军官」的那位：
// 出征中(status=1) > 非俘虏 > 无职位(可带队) > 等级高 > 更新时间新 > id 小。
func pickOrderOfficer(cand []model.EzfyOfficer) *model.EzfyOfficer {
	for i := range cand {
		if cand[i].Status == 1 {
			return &cand[i]
		}
	}
	best := &cand[0]
	for i := 1; i < len(cand); i++ {
		if orderOfficerBetter(&cand[i], best) {
			best = &cand[i]
		}
	}
	return best
}

func orderOfficerBetter(c, b *model.EzfyOfficer) bool {
	cb := orderOfficerScore(c)
	bb := orderOfficerScore(b)
	if cb != bb {
		return cb > bb
	}
	if c.Level != b.Level {
		return c.Level > b.Level
	}
	if !c.UpdateTime.Equal(b.UpdateTime) {
		return c.UpdateTime.After(b.UpdateTime)
	}
	return c.ID < b.ID
}

// orderOfficerScore 分数越高越可能是带队军官：非俘虏 > 无职位(可带队) > 在职。
func orderOfficerScore(o *model.EzfyOfficer) int {
	s := 0
	if o.IsCaptive != 1 {
		s += 4
	}
	if o.Position == 0 {
		s += 2
	}
	if o.Status == 0 {
		s += 1
	}
	return s
}

func markOrderOfficerBackfillDone(db *gorm.DB) {
	_ = db.Exec("INSERT IGNORE INTO settings (`key`, `value`) VALUES (?, ?)", ezfyOrderOfficerIdBackfillKey, "1").Error
}
