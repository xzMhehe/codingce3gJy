package database

import (
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"qqjiayuan/server/internal/config"
)

// EnsureDatabase 建库（幂等）。
//
// 目标库不存在时，先用不带库名的连接串连到 MySQL 实例，再执行 CREATE DATABASE。
// 返回 nil 表示库已就绪；返回错误通常是账号没有 CREATE 权限，或 MySQL 不可达。
func EnsureDatabase(cfg *config.MysqlConfig) error {
	name := strings.TrimSpace(cfg.DBName)
	if name == "" {
		return fmt.Errorf("配置里的 dbname 为空")
	}
	// 库名要拼进 SQL，做一次字符白名单校验，避免反引号等字符破坏语句
	for _, r := range name {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-'
		if !ok {
			return fmt.Errorf("dbname 含非法字符: %q", name)
		}
	}

	// 建库阶段不打印 SQL 日志，避免干扰命令行输出
	inst, err := gorm.Open(mysql.Open(cfg.DSNWithoutDB()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return err
	}
	if sqlDB, err := inst.DB(); err == nil {
		defer sqlDB.Close()
	}

	stmt := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		name,
	)
	if err := inst.Exec(stmt).Error; err != nil {
		return err
	}
	return nil
}

func Init(cfg *config.MysqlConfig) *gorm.DB {
	logCfg := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: 500 * time.Millisecond,
			LogLevel:      logger.Warn,
			Colorful:      true,
		},
	)
	dial := func() (*gorm.DB, error) {
		return gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
			Logger: logCfg,
			// 关联关系由应用层维护，迁移时不建数据库外键（避免存量数据建 FK 失败）
			DisableForeignKeyConstraintWhenMigrating: true,
		})
	}

	db, err := dial()
	if err != nil {
		// 最常见的失败原因之一：目标库还不存在。先尝试建库再重连一次。
		// 建库也失败（如账号无 CREATE 权限）时保留原始错误，方便定位。
		if derr := EnsureDatabase(cfg); derr == nil {
			db, err = dial()
		}
		if err != nil {
			log.Fatalf("连接 MySQL 失败: %v", err)
		}
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("获取底层连接失败: %v", err)
	}
	// ★ 2026-10-04 连接池（database/sql 标准池 = Go 生态主流）显式调优：
	//   · 双机共享同一 RDS → 单机 MaxOpenConns 50（两台合计 100），避免合计打爆
	//     RDS 的 max_connections（连接被拒 = 请求无限排队 = 页面「一直加载中」）；
	//   · MaxIdleConns 20 减少连接抖动（原来 10 偏小，热接口并发高时会频繁新建连接）；
	//   · ConnMaxLifetime 30m / ConnMaxIdleTime 10m 主动回收，避免 MySQL
	//     wait_timeout 掐掉空闲连接导致「connection was killed」类报错。
	sqlDB.SetMaxIdleConns(20)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)
	// 建连后立刻探活：把「DSN 能 Open 但实际连不上」的故障提前暴露，而不是等到首个请求卡住
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("MySQL 探活失败: %v", err)
	}
	fmt.Println("MySQL 连接成功:", cfg.DBName)
	return db
}
