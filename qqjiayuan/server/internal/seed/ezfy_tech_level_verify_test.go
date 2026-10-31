package seed

import (
	"os"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"qqjiayuan/server/internal/model"
)

// 连真实库验证：管理端改过的 effect_value 不会被种子重启冲掉。
// 仅当设置了 EZFY_VERIFY_DSN 时执行（CI/无库环境自动跳过）。
func TestTechLevelSeedKeepsAdminValue(t *testing.T) {
	dsn := os.Getenv("EZFY_VERIFY_DSN")
	if dsn == "" {
		t.Skip("未设置 EZFY_VERIFY_DSN，跳过连库验证")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("连库失败: %v", err)
	}
	// 模拟管理端把战地医疗 Lv.10 改成 50
	db.Model(&model.EzfyCfgTechLevel{}).Where("tech_id = ? AND level = ?", 21, 10).
		Updates(map[string]interface{}{"effect_value": 50, "effect": "伤兵恢复+50%"})
	// 跑两次种子（= 两次重启）
	seedEzfyTechLevels(db)
	seedEzfyTechLevels(db)

	var l model.EzfyCfgTechLevel
	db.Where("tech_id = ? AND level = ?", 21, 10).First(&l)
	if l.EffectValue != 50 {
		t.Fatalf("管理端改的 effect_value 被种子冲掉：得到 %d，期望 50", l.EffectValue)
	}
	if l.Effect != "伤兵恢复+50%" {
		t.Fatalf("effect 文案被冲掉：得到 %q", l.Effect)
	}
	// 资源花费仍应被种子对齐（这条列不归管理端独占）
	if l.Food != 27500 {
		t.Fatalf("资源花费未被种子对齐：food=%d，期望 27500", l.Food)
	}
	// 清理：还原成内置默认
	db.Model(&model.EzfyCfgTechLevel{}).Where("tech_id = ? AND level = ?", 21, 10).
		Updates(map[string]interface{}{"effect_value": 20, "effect": "伤兵恢复+20%"})
}
