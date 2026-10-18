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
