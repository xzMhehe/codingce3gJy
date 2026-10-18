package ezfy

import "testing"

// 护栏：风云榜奖励「空档不发放」的判定（★ 2026-10-18）。
//
// 为什么要守：四档里管理员通常只填其中几档，空档如果也发，玩家会收到一封
// 「恭喜获得第 2 名」但附件为空的邮件 —— 反而像 bug。这里保证空档被识别出来跳过。
func TestEzfyRewardTierEmpty(t *testing.T) {
	var empty ezfyRewardTier
	if !ezfyRewardTierEmpty(empty) {
		t.Fatal("零值档位应判定为空档")
	}

	// 只填钻石
	var d ezfyRewardTier
	d.Diamond = 100
	if ezfyRewardTierEmpty(d) {
		t.Fatal("有钻石不应判定为空档")
	}

	// 只填宝箱（数量 0 视为没填）
	var c ezfyRewardTier
	c.Chests = append(c.Chests, struct {
		ChestID int `json:"chest_id"`
		Count   int `json:"count"`
	}{ChestID: 1, Count: 0})
	if !ezfyRewardTierEmpty(c) {
		t.Fatal("宝箱数量为 0 应判定为空档")
	}
	c.Chests[0].Count = 2
	if ezfyRewardTierEmpty(c) {
		t.Fatal("有宝箱不应判定为空档")
	}

	// 只填道具
	var it ezfyRewardTier
	it.Items = append(it.Items, struct {
		CfgID int `json:"cfg_id"`
		Count int `json:"count"`
	}{CfgID: 1001, Count: 5})
	if ezfyRewardTierEmpty(it) {
		t.Fatal("有道具不应判定为空档")
	}
}

// 附件类型的展示名（邮件正文/管理端对账用）。
func TestEzfyMailAttachKindName(t *testing.T) {
	cases := map[int]string{
		ezfyAttachItem:    "道具",
		ezfyAttachChest:   "宝箱",
		ezfyAttachDiamond: "钻石",
	}
	for kind, want := range cases {
		if got := ezfyMailAttachKindName(kind); got != want {
			t.Errorf("kind=%d 名称=%q, want %q", kind, got, want)
		}
	}
}

// 护栏：领取限流（★ 2026-10-18 用户「领取的时候注意防抖，防刷领取」）。
//
// 注意：这里测的是**进程内限流**（挡连点/脚本高频），真正的「不重复发奖」是 DB 条件更新
// `WHERE claimed = 0`，不在本用例范围。
func TestEzfyMailClaimAllow(t *testing.T) {
	uid := uint(990001) // 用独立 uid，避免和别的用例互相污染
	now := int64(1_000_000)

	if ok, _ := ezfyMailClaimAllow(uid, now); !ok {
		t.Fatal("首次领取应放行")
	}
	// 100ms 内连点 → 拒
	if ok, msg := ezfyMailClaimAllow(uid, now+100); ok || msg == "" {
		t.Fatalf("300ms 内连点应被拒并给出提示，实际 ok=%v msg=%q", ok, msg)
	}
	// 超过 300ms → 放行
	if ok, _ := ezfyMailClaimAllow(uid, now+400); !ok {
		t.Fatal("间隔超过 300ms 应放行")
	}

	// 一分钟窗口上限 60 次
	uid2 := uint(990002)
	base := int64(2_000_000)
	allowed := 0
	for i := 0; i < 70; i++ {
		if ok, _ := ezfyMailClaimAllow(uid2, base+int64(i)*400); ok {
			allowed++
		}
	}
	if allowed != ezfyMailClaimWinMax {
		t.Fatalf("一分钟内应最多放行 %d 次，实际 %d 次", ezfyMailClaimWinMax, allowed)
	}
	// 跨过一分钟窗口后恢复
	if ok, _ := ezfyMailClaimAllow(uid2, base+ezfyMailClaimWinMs+1000); !ok {
		t.Fatal("跨过一分钟窗口后应恢复放行")
	}
}
