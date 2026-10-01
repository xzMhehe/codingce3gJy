package main

import (
	"fmt"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 验证 ezfyCityReportCond：老战报(city_id=0) 能否通过标题坐标反查订单归属城市
func main() {
	dsn := os.Getenv("MYDSN")
	if dsn == "" {
		dsn = "root:1234567890@tcp(127.0.0.1:3306)/qq_jiayuan?charset=utf8mb4&parseTime=True&loc=Local"
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	uid := uint(35806363)
	// 该用户的全部城市
	var cities []struct {
		ID   uint
		Name string
	}
	db.Table("ezfy_city").Select("id, name").Where("user_id = ?", uid).Find(&cities)
	fmt.Printf("用户 %d 城市 %d 座\n", uid, len(cities))

	var totalR int64
	db.Table("ezfy_report").Where("user_id = ?", uid).Count(&totalR)
	fmt.Printf("该用户战报总数 = %d\n", totalR)

	var zeroCityR int64
	db.Table("ezfy_report").Where("user_id = ? AND city_id = 0", uid).Count(&zeroCityR)

	// 新条件（与线上一致）：按坐标反查该城订单
	for _, c := range cities {
		var n int64
		db.Table("ezfy_report").
			Where("user_id = ?", uid).
			Where(`(city_id = ? OR (city_id = 0 AND EXISTS (
				SELECT 1 FROM ezfy_order o
				WHERE o.user_id = ezfy_report.user_id AND o.city_id = ?
				  AND ezfy_report.title LIKE CONCAT('%%', CONCAT(CONCAT('(', o.target_x), CONCAT(',', CONCAT(o.target_y, ')'))), '%%')
			)))`, c.ID, c.ID).
			Count(&n)
		fmt.Printf("  城市#%d %s → 新条件可见战报 = %d\n", c.ID, c.Name, n)
	}
	fmt.Printf("city_id=0 的老战报 = %d（其中能通过坐标反查归属的上面已计入各城市）\n", zeroCityR)
}
