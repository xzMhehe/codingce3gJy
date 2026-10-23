package seed

import (
	"os"
	"strings"
	"testing"
)

// TestEzfyMallStockSeedsUnlimited 商城道具的种子库存必须全部是无限（-1）。
//
// ★ 2026-10-22 用户规则：「商城道具库存都改成无限」。
// 种子文件里的 Stock 若回落到有限值，每次重启 batch(UpdateAll) 都会把线上库存改回有限，
// 所以这里把「所有 Stock 字段都必须是 -1」作为防回归断言。
func TestEzfyMallStockSeedsUnlimited(t *testing.T) {
	for _, file := range []string{"ezfy_cfg_gen.go", "ezfy.go"} {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("读种子文件失败 %s: %v", file, err)
		}
		src := strings.ReplaceAll(string(b), "\r\n", "\n")
		for i, line := range strings.Split(src, "\n") {
			if !strings.Contains(line, "Stock:") {
				continue
			}
			if !strings.Contains(line, "Stock: -1") {
				t.Fatalf("%s:%d 商城道具种子库存不是无限(-1)：%s", file, i+1, line)
			}
		}
	}
	// 基础道具切片再直接断言一次，避免上面的文本扫描漏掉多行/格式变化。
	for i := range ezfyEzfyCfgItem {
		if ezfyEzfyCfgItem[i].Stock != -1 {
			t.Fatalf("商城道具「%s」(ID=%d) 库存=%d，应为 -1（无限）",
				ezfyEzfyCfgItem[i].Name, ezfyEzfyCfgItem[i].ID, ezfyEzfyCfgItem[i].Stock)
		}
	}
}

// TestEzfyMigrationActDescription 移民法案（ItemType 31）人口满时加的是临时空闲人口，
// 商城说明不能再写成「空闲人口达上限后无法使用」。
func TestEzfyMigrationActDescription(t *testing.T) {
	for i := range ezfyEzfyCfgItem {
		it := ezfyEzfyCfgItem[i]
		if it.ID != 1042 {
			continue
		}
		if !strings.Contains(it.Description, "临时空闲") {
			t.Fatalf("移民法案说明未写明临时空闲人口：%s", it.Description)
		}
		if strings.Contains(it.Description, "无法使用") {
			t.Fatalf("移民法案说明仍是旧的「无法使用」口径：%s", it.Description)
		}
		return
	}
	t.Fatal("种子中未找到移民法案（ID=1042）")
}
