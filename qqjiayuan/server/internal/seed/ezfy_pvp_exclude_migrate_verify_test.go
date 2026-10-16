package seed

import (
	"os"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"qqjiayuan/server/internal/model"
)

// TestEzfyPvpExcludeBoardMigrate 连库验证：「不参与排名」名单拆榜迁移（2026-10-16）。
//
// 为什么必须连真库跑：这次迁移动的是**表结构**（加 board 列 / 删老唯一索引 / 复制老记录），
// 纯静态断言查不出 DDL 是否真的能执行、老唯一索引是否真的被删掉。
//
// 安全性：**不碰真实业务库** —— 自己在同一实例上建一个临时库 scratch，造出「老表结构 +
// 老数据」，跑完迁移断言后再 DROP DATABASE。仅当设置了 EZFY_VERIFY_DSN 时执行：
//
//	EZFY_VERIFY_DSN='root:pwd@tcp(127.0.0.1:3306)/qq_ezzt?charset=utf8mb4&parseTime=True&loc=Local' \
//	  go test ./internal/seed/ -run TestEzfyPvpExcludeBoardMigrate -v -timeout 300s
func TestEzfyPvpExcludeBoardMigrate(t *testing.T) {
	dsn := os.Getenv("EZFY_VERIFY_DSN")
	if dsn == "" {
		t.Skip("未设置 EZFY_VERIFY_DSN，跳过连库验证")
	}
	prefix, params := splitDSN(dsn)
	const scratch = "ezfy_excl_migrate_verify"

	// 1) 无库名连接 → 建临时库
	admin, err := gorm.Open(mysql.Open(prefix+params), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("连库失败: %v", err)
	}
	if err := admin.Exec("DROP DATABASE IF EXISTS " + scratch).Error; err != nil {
		t.Fatalf("清理临时库失败: %v", err)
	}
	if err := admin.Exec("CREATE DATABASE " + scratch + " DEFAULT CHARSET utf8mb4").Error; err != nil {
		t.Fatalf("建临时库失败: %v", err)
	}
	defer admin.Exec("DROP DATABASE IF EXISTS " + scratch)

	db, err := gorm.Open(mysql.Open(prefix+scratch+params), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("连临时库失败: %v", err)
	}
	// settings 表（迁移标记位要用）—— 与 RunEzfy 的顺序一致：先 AutoMigrate 再补
	if err := db.AutoMigrate(&model.Setting{}); err != nil {
		t.Fatalf("建 settings 表失败: %v", err)
	}
	// 2) 造出**拆分前的老表结构**（只有 user_id 唯一索引，没有 board）
	oldDDL := "CREATE TABLE ezfy_pvp_exclude (" +
		"id bigint unsigned NOT NULL AUTO_INCREMENT," +
		"user_id bigint unsigned DEFAULT NULL," +
		"nickname varchar(20) DEFAULT NULL," +
		"reason varchar(100) DEFAULT NULL," +
		"created_at datetime(3) DEFAULT NULL," +
		"updated_at datetime(3) DEFAULT NULL," +
		"PRIMARY KEY (id)," +
		"UNIQUE KEY uk_pvp_exclude_user (user_id)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"
	if err := db.Exec(oldDDL).Error; err != nil {
		t.Fatalf("建老表失败: %v", err)
	}
	if err := db.Exec("INSERT INTO ezfy_pvp_exclude (user_id, nickname, reason, created_at, updated_at) VALUES " +
		"(100, '甲', '刷分', NOW(), NOW()), (200, '乙', '', NOW(), NOW())").Error; err != nil {
		t.Fatalf("灌老数据失败: %v", err)
	}

	// 3) 跑迁移
	EnsureEzfyPvpExcludeTable(db)

	if !db.Migrator().HasColumn("ezfy_pvp_exclude", "board") {
		t.Fatal("迁移后没有 board 列")
	}
	if db.Migrator().HasIndex(&model.EzfyPvpExclude{}, "uk_pvp_exclude_user") {
		t.Fatal("老唯一索引 uk_pvp_exclude_user 还在 —— 同一玩家无法在第二个榜标记")
	}
	var n int64
	db.Model(&model.EzfyPvpExclude{}).Count(&n)
	if n != 6 {
		t.Fatalf("老记录(2 人)应复制成 2×3=6 行，实际 %d", n)
	}
	for _, b := range []string{"pvp", "power", "prestige"} {
		var c int64
		db.Model(&model.EzfyPvpExclude{}).Where("board = ?", b).Count(&c)
		if c != 2 {
			t.Fatalf("board=%s 应有 2 行（老记录复制过来），实际 %d", b, c)
		}
	}

	// 4) 幂等：再跑一次不能变多
	EnsureEzfyPvpExcludeTable(db)
	db.Model(&model.EzfyPvpExclude{}).Count(&n)
	if n != 6 {
		t.Fatalf("迁移重跑后行数变了：%d（应为 6，标记位必须挡住重复复制）", n)
	}

	// 5) 关键回归：管理员「只在风云榜排除」后重启，不能被迁移又复制回 power/prestige
	db.Exec("DELETE FROM ezfy_pvp_exclude WHERE board = ? AND user_id = ?", "power", 100)
	EnsureEzfyPvpExcludeTable(db)
	db.Model(&model.EzfyPvpExclude{}).Count(&n)
	if n != 5 {
		t.Fatalf("重启迁移把已删除的 power 记录又复制回来了（行数 %d，应为 5）", n)
	}

	// 6) 复合唯一键生效：同一玩家可以在三个榜各排一次，同一榜重复则冲突
	EnsureEzfyPvpExcludeTable(db)
	for _, b := range []string{"pvp", "power", "prestige"} {
		if err := db.Exec("INSERT INTO ezfy_pvp_exclude (user_id, board, nickname, reason, created_at, updated_at) VALUES (300, ?, '丙', '', NOW(), NOW())", b).Error; err != nil {
			t.Fatalf("同一玩家在 board=%s 插入失败（复合唯一键不对）: %v", b, err)
		}
	}
	if err := db.Exec("INSERT INTO ezfy_pvp_exclude (user_id, board, nickname, reason, created_at, updated_at) VALUES (300, 'pvp', '丙', '', NOW(), NOW())").Error; err == nil {
		t.Fatal("同一玩家在同一榜重复插入应冲突（board+user_id 唯一键没生效）")
	}
}

// splitDSN 把 DSN 拆成「库名前的部分」和「参数部分」：
//
//	root:pwd@tcp(host:3306)/qq_ezzt?charset=utf8mb4 → ("root:pwd@tcp(host:3306)/", "?charset=utf8mb4")
func splitDSN(dsn string) (prefix, params string) {
	head := dsn
	if i := strings.Index(dsn, "?"); i >= 0 {
		head, params = dsn[:i], dsn[i:]
	}
	i := strings.LastIndex(head, "/")
	if i < 0 {
		return head, params
	}
	return head[:i+1], params
}
