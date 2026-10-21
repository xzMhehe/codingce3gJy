package ezfy

import (
	"testing"

	"qqjiayuan/server/internal/model"
)

// TestEzfyFactorySlotsLegacySpread 老队列（FactoryId=0）按兵种需求虚拟归属：
// 低级厂先被填满，满厂不再参与「X座平分」。
func TestEzfyFactorySlotsLegacySpread(t *testing.T) {
	saved := ezfyCfg.troops
	defer func() { ezfyCfg.troops = saved }()
	ezfyCfg.troops = map[int]model.EzfyCfgTroop{
		1: {ID: 1, Type: 2, Require: ""},
		2: {ID: 2, Type: 2, Require: "军工厂(5级)"},
	}
	factories := []model.EzfyCityBuilding{
		{ID: 11, BuildingId: ezfyFactoryBuildingID, Level: 1, Status: 0},
		{ID: 12, BuildingId: ezfyFactoryBuildingID, Level: 1, Status: 0},
		{ID: 13, BuildingId: ezfyFactoryBuildingID, Level: 9, Status: 0},
	}
	// 前两条低级兵 → 两个 1 级厂各 1；高级兵只能用 9 级厂；再一条低级兵 → 9 级厂。
	queues := []model.EzfyTrainQueue{
		{TroopId: 1, Count: 10},
		{TroopId: 1, Count: 10},
		{TroopId: 2, Count: 10},
		{TroopId: 1, Count: 10},
	}
	slots := ezfyFactorySlots(factories, queues)
	byID := map[uint]ezfyFactorySlot{}
	for _, s := range slots {
		byID[s.ID] = s
	}
	if got := byID[11].Used; got != 1 {
		t.Fatalf("厂11 used=%d, want 1", got)
	}
	if got := byID[12].Used; got != 1 {
		t.Fatalf("厂12 used=%d, want 1", got)
	}
	if got := byID[13].Used; got != 2 {
		t.Fatalf("厂13 used=%d, want 2", got)
	}

	eligible, available, free := ezfyFactoryAvailability(slots, 0)
	if eligible != 3 || available != 1 || free != 7 {
		t.Fatalf("低级兵可用厂 = eligible/available/free %d/%d/%d, want 3/1/7",
			eligible, available, free)
	}
	// 高级兵只能用 9 级厂，当前占 2，仍有 7 个队列位。
	eligible, available, free = ezfyFactoryAvailability(slots, 5)
	if eligible != 1 || available != 1 || free != 7 {
		t.Fatalf("高级兵可用厂 = eligible/available/free %d/%d/%d, want 1/1/7",
			eligible, available, free)
	}
}

// TestEzfyFactorySlotsHonorsFactoryId 已写归属的新队列必须按 FactoryId 计数，
// 不能再被虚拟分配挪到别的厂。
func TestEzfyFactorySlotsHonorsFactoryId(t *testing.T) {
	saved := ezfyCfg.troops
	defer func() { ezfyCfg.troops = saved }()
	ezfyCfg.troops = map[int]model.EzfyCfgTroop{
		1: {ID: 1, Type: 2, Require: ""},
	}
	factories := []model.EzfyCityBuilding{
		{ID: 21, BuildingId: ezfyFactoryBuildingID, Level: 1, Status: 0},
		{ID: 22, BuildingId: ezfyFactoryBuildingID, Level: 3, Status: 0},
	}
	queues := []model.EzfyTrainQueue{
		{TroopId: 1, Count: 10, FactoryId: 22},
		{TroopId: 1, Count: 10}, // 老队列：只能去 21
	}
	slots := ezfyFactorySlots(factories, queues)
	byID := map[uint]ezfyFactorySlot{}
	for _, s := range slots {
		byID[s.ID] = s
	}
	if got := byID[22].Used; got != 1 {
		t.Fatalf("归属厂22 used=%d, want 1", got)
	}
	if got := byID[21].Used; got != 1 {
		t.Fatalf("老队列厂21 used=%d, want 1", got)
	}
}

// TestEzfyFactoryFullFactoryExcluded 用户场景：3 座 1 级厂，其中 1 座已有 1 条队列，
// 它已经满了，[全部工厂] 只能按剩下 2 座平分。
func TestEzfyFactoryFullFactoryExcluded(t *testing.T) {
	saved := ezfyCfg.troops
	defer func() { ezfyCfg.troops = saved }()
	ezfyCfg.troops = map[int]model.EzfyCfgTroop{
		1: {ID: 1, Type: 2, Require: ""},
	}
	factories := []model.EzfyCityBuilding{
		{ID: 41, BuildingId: ezfyFactoryBuildingID, Level: 1, Status: 0},
		{ID: 42, BuildingId: ezfyFactoryBuildingID, Level: 1, Status: 0},
		{ID: 43, BuildingId: ezfyFactoryBuildingID, Level: 1, Status: 0},
	}
	slots := ezfyFactorySlots(factories, []model.EzfyTrainQueue{
		{TroopId: 1, Count: 10, FactoryId: 41},
	})
	eligible, available, free := ezfyFactoryAvailability(slots, 0)
	if eligible != 3 || available != 2 || free != 2 {
		t.Fatalf("availability = %d/%d/%d, want 3/2/2", eligible, available, free)
	}
	picked := ezfyFactoryPick(slots, 0, available)
	if len(picked) != 2 || picked[0].ID != 42 || picked[1].ID != 43 {
		t.Fatalf("picked = %+v, want only 42,43", picked)
	}
}

// TestEzfyFactoryPickSkipsFullAndUpgrading 选厂时跳过满厂/升级中的厂；
// 数量不足时优先高等级厂，低级厂留到以后继续参与平分。
func TestEzfyFactoryPickSkipsFullAndUpgrading(t *testing.T) {
	slots := []ezfyFactorySlot{
		{ID: 31, Level: 1, Used: 0, Ready: true},
		{ID: 32, Level: 9, Used: 9, Ready: true},  // 满
		{ID: 33, Level: 5, Used: 0, Ready: false}, // 升级中
		{ID: 34, Level: 2, Used: 0, Ready: true},
	}
	picked := ezfyFactoryPick(slots, 0, 2)
	if len(picked) != 2 {
		t.Fatalf("picked len=%d, want 2", len(picked))
	}
	if picked[0].ID != 34 || picked[1].ID != 31 {
		t.Fatalf("picked IDs=%d,%d, want 34,31", picked[0].ID, picked[1].ID)
	}
	eligible, available, free := ezfyFactoryAvailability(slots, 0)
	if eligible != 3 || available != 2 || free != 3 {
		t.Fatalf("availability = %d/%d/%d, want 3/2/3", eligible, available, free)
	}
}
