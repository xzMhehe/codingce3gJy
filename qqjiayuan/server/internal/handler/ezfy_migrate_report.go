package handler

import (
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/model"
)

// 一次性数据迁移：修复历史战报里的「野地N级」
//
// 背景：
//   - 老代码生成战报标题时一律写「野地N级」，看不出具体地形（平原/丘陵/森林…），
//     也没有坐标。用户反馈「野地8级 也不对，应该展示具体的野地类型 如 平原8级（坐标）」。
//   - 现在的生成逻辑已经改成「地形名+等级」（见 ezfy_order.go 的 targetName），
//     但**存量数据**还是老样子，列表里翻到旧战报仍是「野地N级」。这里把存量也修一遍。
//
// 做法：
//   从战报正文里把坐标捞出来（正文形如「目的地:野地2级(432,259)」
//   或「对 野地2级[ 432，259 ]进行了掠夺」），再用坐标哈希重算真实地形名，
//   回写标题与正文里的「野地N级」→「平原N级」。
//
// 幂等：处理过的标题不再含「野地」，后续启动是空操作。
var ezfyReportMigrateOnce sync.Once

// 目的地:野地2级(432,259)  —— 中文/英文括号都兼容
var ezfyReportDestRe = regexp.MustCompile(`目的地:野地\d+级[（(]\s*(-?\d+)\s*[,，]\s*(-?\d+)\s*[)）]`)

// 对 野地2级[ 432，259 ] / 对野地8级[328，61]
var ezfyReportBodyRe = regexp.MustCompile(`野地\d+级\s*[\[［]\s*(-?\d+)\s*[,，]\s*(-?\d+)\s*[\]］]`)

// 采集报告: 「派遣部队在野地10级(262,239)完成一次采集结算」
var ezfyReportCollectRe = regexp.MustCompile(`野地\d+级[（(]\s*(-?\d+)\s*[,，]\s*(-?\d+)\s*[)）]`)

// 标题/正文里的「野地N级」（用于整体替换成地形名）
var ezfyReportWildRe = regexp.MustCompile(`野地(\d+)级`)

// ezfyMigrateReportTitles 把存量战报里的「野地N级」改成具体地形名（幂等）
func ezfyMigrateReportTitles(db *gorm.DB) {
	var reports []model.EzfyReport
	// 只捞需要处理的：标题含「野地N级」
	db.Where("title LIKE ?", "%野地%级%").Find(&reports)
	fixed := 0
	for i := range reports {
		r := &reports[i]
		// ① 活动目标战报：标题形如「活动野地3级战斗报告: 活动野地」——
		//    冒号后面缺等级和坐标（老代码用了不带等级的 typeName）。补全成
		//    「活动野地3级战斗报告: 活动野地3级(x,y)」。
		if strings.HasPrefix(r.Title, "活动") || strings.HasPrefix(r.Title, "特殊") {
			if actTitle, ok := ezfyFixActReportTitle(r.Content, r.Title); ok {
				db.Model(&model.EzfyReport{}).Where("id = ?", r.ID).
					Update("title", actTitle)
				fixed++
			}
			continue
		}
		// ② 普通野地：「野地N级」→ 具体地形名（从正文捞坐标重算地形）
		//    注意：采集报告的坐标只写在**标题**里（正文没有坐标），所以标题也要一起找。
		x, y, ok := ezfyReportCoordOf(r.Content)
		if !ok {
			x, y, ok = ezfyReportCoordOf(r.Title)
		}
		if !ok {
			continue
		}
		name := ezfyTerrainNameEx(x, y)
		if name == "" || name == "未知" {
			continue
		}
		// ★ 注意：Go 的 ReplaceAllString 里 "$1级" 会被当成组名 "1级"（空），必须写 "${1}"。
		newTitle := ezfyReportWildRe.ReplaceAllString(r.Title, name+"${1}级")
		newContent := ezfyReportWildRe.ReplaceAllString(r.Content, name+"${1}级")
		if newTitle == r.Title && newContent == r.Content {
			continue
		}
		db.Model(&model.EzfyReport{}).Where("id = ?", r.ID).
			Updates(map[string]interface{}{"title": newTitle, "content": newContent})
		fixed++
	}
	if fixed > 0 {
		log.Printf("ezfy 战报标题迁移: %d 条战报标题已修复（野地地形名 / 活动目标等级坐标）", fixed)
	}
}

// 活动战报标题：把「活动野地3级战斗报告: 活动野地」补成「…: 活动野地3级(x,y)」
var ezfyActReportTitleRe = regexp.MustCompile(`^(活动[^:：]*\d+级)(战斗报告)[:：]\s*(.+)$`)
var ezfyActReportBodyRe = regexp.MustCompile(`对\s*(活动[^\[\[]*?)\s*[\[\[]\s*(-?\d+)\s*[,，]\s*(-?\d+)\s*[\]\]]`)

func ezfyFixActReportTitle(content, title string) (string, bool) {
	// 冒号后已经带坐标系列就跳过（幂等）
	if strings.Contains(title, "(") {
		return title, false
	}
	x, y, ok := ezfyReportCoordOf(content)
	if !ok {
		x, y, ok = ezfyReportCoordOf(title)
	}
	if !ok {
		return title, false
	}
	m := ezfyActReportTitleRe.FindStringSubmatch(title)
	if len(m) != 4 {
		return title, false
	}
	full := m[1] // 活动野地3级
	return full + m[2] + ": " + full + "(" + strconv.Itoa(x) + "," + strconv.Itoa(y) + ")", true
}

// ezfyReportCoordOf 从战报正文里解析目标坐标
func ezfyReportCoordOf(content string) (int, int, bool) {
	for _, re := range []*regexp.Regexp{ezfyReportDestRe, ezfyReportBodyRe, ezfyReportCollectRe} {
		if m := re.FindStringSubmatch(content); len(m) == 3 {
			x, e1 := strconv.Atoi(m[1])
			y, e2 := strconv.Atoi(m[2])
			if e1 == nil && e2 == nil {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}
