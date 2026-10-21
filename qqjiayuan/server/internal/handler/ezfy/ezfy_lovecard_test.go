package ezfy

import (
	"strings"
	"testing"

	"qqjiayuan/server/internal/model"
)

// TestLoveCardSettle 管理端「完成」结算口径：剩余未领天数 × 每日钻石。
func TestLoveCardSettle(t *testing.T) {
	cases := []struct {
		name     string
		card     *model.EzfyLoveCard
		wantDays int
		wantGain int64
	}{
		{"正常剩余25天", &model.EzfyLoveCard{TotalDays: 30, ClaimedDays: 5, DailyDiamond: 150}, 25, 3750},
		{"高级卡", &model.EzfyLoveCard{TotalDays: 30, ClaimedDays: 10, DailyDiamond: 200}, 20, 4000},
		{"已领完", &model.EzfyLoveCard{TotalDays: 30, ClaimedDays: 30, DailyDiamond: 150}, 0, 0},
		{"领取天数超过总天数", &model.EzfyLoveCard{TotalDays: 30, ClaimedDays: 40, DailyDiamond: 150}, 0, 0},
		{"总天数为0", &model.EzfyLoveCard{TotalDays: 0, ClaimedDays: 0, DailyDiamond: 150}, 0, 0},
		{"每日钻石为0", &model.EzfyLoveCard{TotalDays: 30, ClaimedDays: 5, DailyDiamond: 0}, 25, 0},
		{"每日钻石为负", &model.EzfyLoveCard{TotalDays: 30, ClaimedDays: 5, DailyDiamond: -100}, 25, 0},
		{"空卡", nil, 0, 0},
	}
	for _, c := range cases {
		days, gain := loveCardSettle(c.card)
		if days != c.wantDays || gain != c.wantGain {
			t.Errorf("%s: loveCardSettle() = (%d, %d), want (%d, %d)", c.name, days, gain, c.wantDays, c.wantGain)
		}
	}
}

// TestLoveCardCompleteWiring 管理端「完成」按钮 → 路由 → 条件更新 + 事务发钻石 的链路锁死。
func TestLoveCardCompleteWiring(t *testing.T) {
	router := rawFile(t, "../../router/router.go")
	if !strings.Contains(router, `admin.POST("/ezfy-love-cards/:id/complete"`) {
		t.Fatal("管理端路由缺少 /ezfy-love-cards/:id/complete")
	}

	admin := rawFile(t, "ezfy_admin_lovecard.go")
	for _, want := range []string{
		"func (h *EzfyAdmin) AdminEzfyLoveCardComplete(",
		`Where("id = ? AND claimed_days = ?", love.ID, love.ClaimedDays)`,
		`"claimed_days": love.TotalDays`,
		`gorm.Expr("diamond + ?", gain)`,
		"h.DB.Transaction(",
	} {
		if !strings.Contains(admin, want) {
			t.Fatalf("完成接口实现缺失（缺 %s）", want)
		}
	}

	ui := readFrontFile(t, "../../../../admin-web/src/components/admin/AdminEzfyLoveCard.vue")
	for _, want := range []string{"doComplete", "/complete", "剩余未领钻石一次性发放"} {
		if !strings.Contains(ui, want) {
			t.Fatalf("管理端「完成」按钮缺失（缺 %s）", want)
		}
	}
}
