package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"qqjiayuan/server/internal/seed"
)

// 一次性工具：对共享库直接跑 repairEquipSnapshots（与服务器启动时同款幂等逻辑），
// 把存量装备行六项战斗属性对齐配置池、重建军官 equipment JSON。
func main() {
	dsn := "root:Mzd980625@@#@tcp(rm-uf638q54c9ohpaf6i4o.mysql.cn-shanghai.rds.aliyuncs.com:3306)/qq_jiayuan?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	seed.EnsureEzfyEquipSnapshots(db)
	fmt.Println("repair done")
}
