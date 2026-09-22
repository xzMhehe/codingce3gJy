// 家园社区 · 二战风云 —— 城池批量迁移工具（ezfymigrate）
//
// 用途：把玩家城池批量迁移到指定大洲。主要为「内测开荒」服务 ——
// 让分散在世界各地的城市统一搬到欧洲，玩家之间离得近，
// 才打得起征服 / 侦查 / 掠夺。
//
// ★ 这是**运维工具，不参与游戏运行**：
//   - 不写进 seed（不会每次启动自动跑）
//   - 不消耗玩家道具、不扣黄金（玩家侧迁城才走道具，见 ezfy_move.go MoveCity）
//   - 默认 --dry-run 只预演不落库，确认无误再加 --apply
//
// 用法示例：
//
//	# 1) 先看会发生什么（不落库，强烈建议先跑一次）
//	./ezfymigrate --all --dry-run
//
//	# 2) 全部城池迁到欧洲（默认洲）
//	./ezfymigrate --all --apply
//
//	# 3) 只迁某个玩家（按家园/游戏 ID）
//	./ezfymigrate --uid 10007 --apply
//
//	# 4) 只迁海城 / 只迁陆城
//	./ezfymigrate --all --sea-only --apply
//	./ezfymigrate --all --land-only --apply
//
//	# 5) 迁到亚洲
//	./ezfymigrate --all --continent 2 --apply
//
//	# 6) 指定配置与批次大小（分批提交，避免一次改动过大）
//	./ezfymigrate -config /opt/qqjiayuan/server/config.yaml --all --apply --limit 100
//
//	# 7) ★ 沿海迁城计划：把所有「建有航海协会」的城重新迁到沿海平原
//	#    （线上事故修复用：批量迁城把海城搬成了陆地城）
//	./ezfymigrate --coastal --dry-run
//	./ezfymigrate --coastal --apply --yes
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"gorm.io/gorm"

	"qqjiayuan/server/internal/config"
	"qqjiayuan/server/internal/handler"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/database"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "配置文件路径（同 dbinit / server）")
	uid := flag.Uint("uid", 0, "只迁移该玩家的城（家园/游戏 ID）；0 = 不限制")
	all := flag.Bool("all", false, "迁移全部玩家的城")
	continent := flag.Int("continent", 0, "目标大洲 ID：1欧洲 2亚洲 3非洲 4北美洲 5南美洲 6大洋洲 7南极洲（默认 0 = 欧洲）")
	seaOnly := flag.Bool("sea-only", false, "只迁移沿海城市（海城）")
	landOnly := flag.Bool("land-only", false, "只迁移内陆城市（非海城）")
	// ★ 沿海迁城计划：把「建有航海协会」的城重新迁到沿海平原。
	//   线上事故（2026-09-20）批量迁城把海城搬成了陆地城，用这个模式修回来。
	coastal := flag.Bool("coastal", false, "沿海迁城计划：把所有建有航海协会的城重新迁到沿海平原（可与其他范围参数组合）")
	coastalBuilding := flag.Int("coastal-building", 19, "「沿海城市」的判定建筑 ID（默认 19 = 航海协会）")
	stats := flag.Bool("stats", false, "只统计各大洲的空闲沿海平原/空地余量，不做任何迁移")
	// ★ 用户规则：「不够就给移动到亚洲的沿海平原」—— 目标洲放不下时自动退到备用洲。
	coastalFallback := flag.Int("coastal-fallback", 2, "沿海迁城计划的备用洲 ID（默认 2 = 亚洲；0 = 不启用备用洲）")
	apply := flag.Bool("apply", false, "真正落库；不加则只预演（dry-run）")
	dryRun := flag.Bool("dry-run", false, "显式预演（与不加 --apply 等价，方便脚本里写清楚意图）")
	limit := flag.Int("limit", 0, "最多迁移多少座城（0 = 不限）")
	yes := flag.Bool("yes", false, "跳过交互确认（适合脚本/自动化调用）")
	flag.Parse()

	if !*all && *uid == 0 && !*coastal && !*stats {
		log.Fatal("请指定迁移范围：--all（全部玩家）/ --uid <ID>（单个玩家）/ --coastal（所有航海协会城）/ --stats（只看余量）")
	}
	if *all && *uid != 0 {
		log.Fatal("--all 与 --uid 不能同时使用")
	}
	if *seaOnly && *landOnly {
		log.Fatal("--sea-only 与 --land-only 不能同时使用")
	}
	// --dry-run 与 --apply 同时给时，以 --dry-run 为准（宁可不改数据）
	if *dryRun {
		*apply = false
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("读取配置失败: %v（当前目录下需要有 config.yaml，或用 -config 指定）", err)
	}
	db := database.Init(&cfg.Mysql)
	h := &handler.EzfyHandler{DB: db}

	continentName := ezfyContinentLabel(db, *continent)

	fmt.Println("============================================================")
	fmt.Println("  二战风云 · 城池批量迁移")
	fmt.Println("============================================================")
	fmt.Printf("  MySQL      : %s:%d / %s\n", cfg.Mysql.Host, cfg.Mysql.Port, cfg.Mysql.DBName)
	if *all {
		fmt.Println("  迁移范围   : 全部玩家")
	} else {
		fmt.Printf("  迁移范围   : 玩家 %d\n", *uid)
	}
	fmt.Printf("  目标大洲   : %s\n", continentName)
	if *coastal {
		fmt.Printf("  迁移计划   : ★ 沿海迁城计划（判定建筑 ID %d = 航海协会，落点必须是沿海平原）\n", *coastalBuilding)
	}
	if *seaOnly {
		fmt.Println("  城市类型   : 仅沿海城市（海城）")
	} else if *landOnly {
		fmt.Println("  城市类型   : 仅内陆城市")
	} else {
		fmt.Println("  城市类型   : 全部")
	}
	if *limit > 0 {
		fmt.Printf("  单次上限   : %d 座\n", *limit)
	}
	if *apply {
		fmt.Println("  模式       : ★ 落库执行（apply）")
	} else {
		fmt.Println("  模式       : 预演（dry-run，不修改任何数据）")
	}
	fmt.Println("============================================================")
	fmt.Println()

	// 取待迁移的城
	var cities []model.EzfyCity
	if *stats {
		printCoastalStats(db)
		return
	}
	if *coastal {
		// 沿海迁城计划：范围 = 所有「建有航海协会」的城（不看它现在是不是海城）
		cities = h.EzfyCitiesWithBuilding(*coastalBuilding)
		if *uid != 0 {
			keep := make([]model.EzfyCity, 0, len(cities))
			for _, c := range cities {
				if c.UserID == *uid {
					keep = append(keep, c)
				}
			}
			cities = keep
		}
	} else {
		q := db.Model(&model.EzfyCity{}).Order("id ASC")
		if !*all {
			q = q.Where("user_id = ?", *uid)
		}
		if err := q.Find(&cities).Error; err != nil {
			log.Fatalf("查询城池失败: %v", err)
		}
	}
	if len(cities) == 0 {
		fmt.Println("没有找到匹配的城池，退出。")
		return
	}
	fmt.Printf("共查到 %d 座城池，开始生成迁移计划...\n\n", len(cities))

	var plan []handler.EzfyMovePlanItem
	if *coastal {
		plan = make([]handler.EzfyMovePlanItem, 0, len(cities))
		for i := range cities {
			ct := cities[i]
			plan = append(plan, handler.EzfyMovePlanItem{
				CityId: ct.ID, UserID: ct.UserID, Name: ct.Name,
				OldX: ct.X, OldY: ct.Y, OldRegion: ezfyRegionLabel(ct.X, ct.Y),
				IsSea: true,
			})
		}
	} else {
		plan = h.EzfyBuildMovePlan(cities, *continent, *seaOnly, *landOnly)
	}

	// 打印计划
	var toMove, skipped, failed int
	shown := 0
	for _, p := range plan {
		if *coastal {
			// 沿海迁城计划：不在目标洲、或脚下不是沿海平原 → 需要迁
			if !handler.EzfyCoastalNeedMoveAt(p.OldX, p.OldY, *continent) {
				skipped++
				continue
			}
			toMove++
			if shown < 200 {
				fmt.Printf("  [迁移] 城%-6d 玩家%-6d %-14s 航海协会城 %s(%d,%d) → 沿海平原(%s)\n",
					p.CityId, p.UserID, trimName(p.Name), p.OldRegion, p.OldX, p.OldY, continentName)
			}
			shown++
			continue
		}
		if p.NewX == 0 && p.NewY == 0 {
			failed++
			fmt.Printf("  [跳过] 城%-6d 玩家%-6d %-14s (%d,%d) %s  —— %s\n",
				p.CityId, p.UserID, trimName(p.Name), p.OldX, p.OldY, p.OldRegion, p.Reason)
			continue
		}
		if p.Reason != "" { // 已在目标洲等
			skipped++
			continue
		}
		toMove++
		if shown < 200 {
			kind := "内陆"
			if p.IsSea {
				kind = "海城"
			}
			fmt.Printf("  [迁移] 城%-6d 玩家%-6d %-14s %s(%d,%d) %s → %s(%d,%d) %s\n",
				p.CityId, p.UserID, trimName(p.Name), kind, p.OldX, p.OldY, p.OldRegion,
				kind, p.NewX, p.NewY, p.NewRegion)
		}
		shown++
	}
	if shown > 200 {
		fmt.Printf("  ... 另有 %d 条迁移记录未打印（避免刷屏）\n", shown-200)
	}

	fmt.Println()
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("  需迁移 : %d 座\n", toMove)
	if *coastal {
		fmt.Printf("  无需动 : %d 座（脚下已经是沿海平原）\n", skipped)
	} else {
		fmt.Printf("  无需动 : %d 座（已在目标大洲）\n", skipped)
	}
	fmt.Printf("  无法迁 : %d 座（目标洲内没找到可用空地）\n", failed)
	fmt.Println("------------------------------------------------------------")

	if toMove == 0 {
		fmt.Println("没有需要迁移的城市，退出。")
		return
	}

	if !*apply {
		fmt.Println()
		fmt.Println("【预演结束】以上均未落库。确认无误后加 --apply 真正执行。")
		return
	}

	// 交互确认（自动化调用可加 --yes 跳过）
	if !*yes {
		fmt.Printf("\n确认要把以上 %d 座城市真正迁移吗？输入 yes 继续：", toMove)
		var ans string
		fmt.Scanln(&ans)
		if strings.ToLower(strings.TrimSpace(ans)) != "yes" {
			fmt.Println("已取消，未修改任何数据。")
			return
		}
	}

	// 执行
	//
	// ★ 关键：逐城**实时重算**落点，不要用 plan 里预计算的 NewX/NewY。
	//   原因：plan 是一次性对着「原始地图」算的，而每迁一座城就多占一格，
	//   预计算的落点会自相撞（线上 147 城实测：预演全通过，apply 时
	//   71 座报 `该坐标已有城市`）。excluded 记录本批已占用坐标，双重保险。
	//   预演的坐标只作「预计会迁到哪片区域」参考，不构成承诺。
	fmt.Println()
	used := make(map[[2]int]bool, len(plan))
	// ★ 批次上下文：把「全城坐标」读一次，逐城复用（不再每城全表扫描）。
	//   配合 ezfy_coastal_index.go 的沿海平原缓存，批量迁城的 DB 读量从 O(n·N) 降到 O(N)。
	mctx := h.NewMoveCtx()
	done, errs, nomove := 0, 0, 0
	for _, p := range plan {
		if p.Reason != "" {
			continue // 已在目标洲 / 被 --sea-only、--land-only 过滤掉
		}
		nx, ny, msg := 0, 0, ""
		if *coastal {
			nx, ny, msg = h.EzfyMoveOneCityCoastalC(p.CityId, *continent, *coastalFallback, used, mctx)
		} else {
			nx, ny, msg = h.EzfyMoveOneCityC(p.CityId, *continent, used, mctx)
		}
		if msg != "" {
			fmt.Printf("  [失败] 城%-6d 玩家%-6d %s: %s\n", p.CityId, p.UserID, trimName(p.Name), msg)
			errs++
			continue
		}
		if nx == p.OldX && ny == p.OldY {
			nomove++
			continue // 已在目标洲（实际落点没变）
		}
		done++
		fmt.Printf("  [完成] 城%-6d 玩家%-6d %-14s (%d,%d) → (%d,%d) %s\n",
			p.CityId, p.UserID, trimName(p.Name), p.OldX, p.OldY, nx, ny, ezfyRegionLabel(nx, ny))
		if *limit > 0 && done >= *limit {
			break
		}
	}

	fmt.Println()
	fmt.Println("============================================================")
	fmt.Printf("  迁移完成：成功 %d 座，失败 %d 座", done, errs)
	if nomove > 0 {
		fmt.Printf("，无需迁移 %d 座", nomove)
	}
	fmt.Println()
	fmt.Println("============================================================")
	if errs > 0 {
		os.Exit(1)
	}
}

// printCoastalStats 统计各大洲的「沿海平原 / 其他陆地」余量，帮助判断还放得下多少海城。
func printCoastalStats(db *gorm.DB) {
	names := map[int]string{1: "欧洲", 2: "亚洲", 3: "非洲", 4: "北美洲", 5: "南美洲", 6: "大洋洲", 7: "南极洲"}
	type st struct{ coastAll, coastFree, landAll, landFree int }
	m := map[int]*st{}
	occupied := map[[2]int]bool{}
	var cs []model.EzfyCity
	db.Select("x, y").Find(&cs)
	for _, c := range cs {
		occupied[[2]int{c.X, c.Y}] = true
	}
	for x := 0; x < 500; x++ {
		for y := 0; y < 500; y++ {
			cont := handler.EzfyContinentOf(x, y)
			if cont <= 0 {
				continue
			}
			s := m[cont]
			if s == nil {
				s = &st{}
				m[cont] = s
			}
			free := !occupied[[2]int{x, y}]
			if handler.EzfyTerrainAt(x, y) == handler.EzfyCoastalPlainValue {
				s.coastAll++
				if free {
					s.coastFree++
				}
			} else if handler.EzfyTerrainAt(x, y) != 8 {
				s.landAll++
				if free {
					s.landFree++
				}
			}
		}
	}
	fmt.Printf("  已有城池 %d 座（占用格）\n\n", len(cs))
	fmt.Println("  洲        沿海平原 空闲/总数      其他陆地 空闲/总数")
	for _, id := range []int{1, 2, 3, 4, 5, 6, 7} {
		s := m[id]
		if s == nil {
			continue
		}
		fmt.Printf("  %-6s    %6d / %-8d    %6d / %-8d\n", names[id], s.coastFree, s.coastAll, s.landFree, s.landAll)
	}
}

// ezfyContinentLabel 把洲 ID 转成中文名（读库里能查到的城来反推，避免 import 循环）
//
// 说明：洲名与几何定义在 handler 包内部（ezfyContinentNames 未导出），
// 这里用一份等价映射，保证 CLI 输出与游戏内显示一致。
func ezfyContinentLabel(db *gorm.DB, id int) string {
	if id == 0 {
		id = 1 // 默认欧洲
	}
	names := map[int]string{
		1: "欧洲", 2: "亚洲", 3: "非洲", 4: "北美洲",
		5: "南美洲", 6: "大洋洲", 7: "南极洲",
	}
	if n, ok := names[id]; ok {
		return fmt.Sprintf("%s (ID %d)", n, id)
	}
	return fmt.Sprintf("未知 (ID %d)", id)
}

func trimName(s string) string {
	r := []rune(s)
	if len(r) <= 14 {
		return s
	}
	return string(r[:13]) + "…"
}

// ezfyRegionLabel 用洲名近似表示落点所属区域（CLI 拿不到 handler 内部的地名函数，够用即可）
func ezfyRegionLabel(x, y int) string {
	id := ezfyContinentOfExported(x, y)
	names := map[int]string{
		1: "欧洲", 2: "亚洲", 3: "非洲", 4: "北美洲",
		5: "南美洲", 6: "大洋洲", 7: "南极洲",
	}
	if n, ok := names[id]; ok {
		return n
	}
	return "大海"
}

// ezfyContinentOfExported 与 handler 内部 ezfyContinentOf 保持一致的洲判定
//
// 说明：handler 的 ezfyContinentOf 未导出，这里复制一份等价的几何判定，
// 只用于 CLI 日志显示；**游戏逻辑一律以 handler 内部实现为准**。
// 若哪天改了世界地图几何，记得同步这里（否则只是日志显示偏差，不影响数据）。
func ezfyContinentOfExported(x, y int) int {
	return handler.EzfyContinentOf(x, y)
}
