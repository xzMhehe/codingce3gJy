package seed

import (
	"os"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	"qqjiayuan/server/internal/model"
)

// TestStaffLevelSeedUpTo15 连真实库验证：参谋部建筑等级配置含 13/14/15 级。
//
// ★ 2026-10-14 用户：「参谋部等级上调到 15 级」。
//
//	上限（ezfyBuildingMaxLevel=15）与配置（ezfy_cfg_building_level 有 13/14/15 行）必须成对，
//	否则升级校验通过、`ezfyCfg.buildingLevel(10,13)` 却返回 nil → 报「配置缺失」升不上去。
//
// ★★ 注意：本测试**只跑建筑等级表这一条 batchKeep 语义的写入**，不调 `seedEzfy` ——
//	后者还含 `batch(UpdateAll)`（科技主表/道具表），在线上库上跑会覆盖管理端数据。
//
// 仅当设置了 EZFY_VERIFY_DSN 时执行（CI/无库环境自动跳过）。
//
//	EZFY_VERIFY_DSN='root:pwd@tcp(host:3306)/qq_ezzt?charset=utf8mb4&parseTime=True&loc=Local' \
//	  go test ./internal/seed/ -run TestStaffLevelSeedUpTo15 -v -timeout 300s
func TestStaffLevelSeedUpTo15(t *testing.T) {
	dsn := os.Getenv("EZFY_VERIFY_DSN")
	if dsn == "" {
		t.Skip("未设置 EZFY_VERIFY_DSN，跳过连库验证")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("连库失败: %v", err)
	}
	// 与 seedEzfy 里同一条策略：建筑等级表 = insert-only（DoNothing），只补缺不覆盖。
	apply := func() {
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).
			CreateInBatches(ezfyEzfyCfgBuildingLevel, 200).Error; err != nil {
			t.Fatalf("种子写入失败: %v", err)
		}
	}
	apply()
	// 参谋部(10) 必须有 13/14/15 三级
	for _, lv := range []int{13, 14, 15} {
		var row model.EzfyCfgBuildingLevel
		if err := db.Where("building_id = ? AND level = ?", 10, lv).First(&row).Error; err != nil {
			t.Fatalf("参谋部 %d 级配置缺失（种子没补上）：%v", lv, err)
		}
		if row.Capacity != int64(lv) {
			t.Fatalf("参谋部 %d 级容量应为 %d，实际 %d", lv, lv, row.Capacity)
		}
		if row.Food <= 0 || row.Steel <= 0 {
			t.Fatalf("参谋部 %d 级资源消耗异常：food=%d steel=%d", lv, row.Food, row.Steel)
		}
	}
	// 幂等：再跑一次不能重复插
	apply()
	var n int64
	db.Model(&model.EzfyCfgBuildingLevel{}).Where("building_id = ? AND level >= ?", 10, 13).Count(&n)
	if n != 3 {
		t.Fatalf("参谋部 13~15 级应为 3 行（幂等），实际 %d", n)
	}
	// ★ 回归护栏：种子 ID 不能与已有建筑撞号（撞号会被 DoNothing 静默跳过 → 数据永远补不上）
	var collide int64
	db.Model(&model.EzfyCfgBuildingLevel{}).
		Where("id IN ? AND building_id <> 10", []int{3010, 3011, 3012}).Count(&collide)
	if collide != 0 {
		t.Fatalf("参谋部新等级的 ID 与其它建筑撞号 %d 行（DoNothing 会静默跳过）", collide)
	}
}
