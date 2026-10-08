// 战报 type_name 一次性回填工具（上线后每个环境各跑一次，可重复执行）
//
// 背景：ezfy_report 新增持久化列 type_name，写战报时已由 addReport 落库；
// 存量行此列为空，管理端「战报查询」类型过滤需要它，故用本工具按 id 分批回填。
//
// 用法（在含 config.yaml 的目录下）：
//
//	go run ./cmd/backfilltype
//	go run ./cmd/backfilltype -config ../config.yaml
package main

import (
	"flag"
	"fmt"
	"log"

	"qqjiayuan/server/internal/config"
	"qqjiayuan/server/internal/handler/ezfy"
	"qqjiayuan/server/internal/model"
	"qqjiayuan/server/pkg/database"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "配置文件路径")
	batchSize := flag.Int("batch", 2000, "每批读取行数")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}
	db := database.Init(&cfg.Mysql)

	var scanned, updated int64
	var rows []model.EzfyReport
	lastID := uint(0)
	current := model.EzfyReport{}

	for {
		rows = rows[:0]
		if err := db.Where("id > ?", lastID).Order("id ASC").Limit(*batchSize).Find(&rows).Error; err != nil {
			log.Fatalf("分页读取失败: %v", err)
		}
		if len(rows) == 0 {
			break
		}
		lastID = rows[len(rows)-1].ID

		// 按目标类型汇总待更新 id，一次一个类型批量 UPDATE，避免逐行写。
		byType := map[string][]uint{}
		for _, r := range rows {
			scanned++
			tn := ezfy.EzfyReportTypeName(r.ReportType, r.Title)
			if tn == r.TypeName {
				continue
			}
			byType[tn] = append(byType[tn], r.ID)
		}
		for tn, ids := range byType {
			res := db.Model(&current).Where("id IN ?", ids).Update("type_name", tn)
			if res.Error != nil {
				log.Fatalf("回填 type_name=%s 失败: %v", tn, res.Error)
			}
			updated += res.RowsAffected
		}

		if len(rows) < *batchSize {
			break
		}
	}
	fmt.Printf("扫描 %d 条, 更新 %d 条\n", scanned, updated)
}