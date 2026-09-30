# 二战风云 新手任务「首个城池」助力

## Context（背景）

玩家对新手任务有诉求：希望建出首个城池后能领一份「发展助力」奖励——**各资源 2000 万 + 黄金 2000 万**，整个生命周期只能领一次，领完标记「已领取」。

已与用户确认：
- 奖励 = 粮食/钢铁/石油/稀矿 **各 2000 万**，黄金也 **2000 万**。
- 触发 = **有首城即领**：所有零星玩家（老玩家也一样）只要拥有首个城池即可领，不做「仅新号」限制。

## 现有机制（复用的点）

二战任务系统完全数据驱动，由配置表 `ezfy_cfg_task` + 玩家进度表 `ezfy_task` 驱动：

- **配置 seed**（`server/internal/seed/ezfy_cfg_gen.go` 的 `ezfyEzfyCfgTask`）→ 用 `batchKeep`（insert-only）种入 `ezfy_cfg_task`。新增一行 ID 即可在下次启动被插入，不影响表里已存在行。
- **玩家进度**：`initTasks`（`ezfy.go`）会为**缺少记录的所有配置**补建 `ezfy_task` 行（`status=0`）。所以老玩家上线后也会自动出现新任务，领完后 `taskAward` 置 `status=2`（已领取），一次性任务（ResetType 0）不再重置 →「只能领一次」天然成立。
- **状态型任务自动完成**：任务类型在 `ezfyStateTaskTypes`（`ezfy.go:2638`）里即会走 `Tasks()` 的自动同步 + `calcStateValue`——条件满足时 `status` 自动置 1（可领取）。
- **奖励口径**：`ezfyTaskRewardRes` 对新手任务（TypeId=1）把**粮钢油稀 ×1000**（展示与发奖同口径）；**黄金不乘**。发奖走 `taskAward` 内的 `giveResNoCap`（不受仓储上限，仍受「资源最大值」21 亿约束）。

## 改动清单

### 1. 新增任务配置（`server/internal/seed/ezfy_cfg_gen.go`）
在 `ezfyEzfyCfgTask` 数组末尾（ID 19 之后）追加一行：

```go
{ID: 20, Name: "首个城池·发展助力", TaskType: "has_city", Target: 1,
 RewardFood: 20000, RewardSteel: 20000, RewardOil: 20000, RewardRare: 20000,
 RewardGold: 20000000, RewardPrestige: 0,
 SortNo: 0, TypeId: 1, Status: 1},
```

说明：
- `RewardFood/Steel/Oil/Rare = 20000` → 经新手任务 ×1000 后实得 **2000 万**。
- `RewardGold = 20000000` 直接 = **2000 万**（黄金不乘）。
- `TaskType: "has_city"` 对应「首个城池」状态条件；`TypeId: 1` 归入「新手任务」，既有 ×1000 加成自动生效。
- `SortNo: 0` 让它在新手任务里排最前。

### 2. 注册状态任务类型（`server/internal/handler/ezfy.go`）
- `ezfyStateTaskTypes`（L2638）加入 `"has_city": true`，使该任务走自动完成同步。
- `calcStateValue`（L2722）加 case：`case "has_city": return 1`（只要拥有城池即满足 Target=1；所有二战玩家都必有首城）。

### 3. 管理端字典文案（`admin-web/src/components/admin/AdminEzfyData.vue`）
`taskAction` 字典（L262）加 `has_city: { n: '拥有首个城池' }`，否则任务列表里该任务的行为列显示原始英文代码。

> 游戏端（`web/src/views/Ezfy.vue`）无需改动：任务列表/领奖按钮/「已领取」状态都是根据后端 `/tasks` 返回动态渲染的。

## 不涉及
- 不新增数据库表、不新增接口（复用 `/tasks`、`/tasks/award`）。
- 不改奖励上限逻辑（`giveResNoCap` 已按 21 亿封顶，2000 万远低于上限）。

## 验证（端到端）
1. `go build ./...` 编译 server 通过；`npm run build` 编译 admin-web 通过。
2. 重启 server → seed 插入 `ezfy_cfg_task` ID=20。
3. 用站长账号（id 10000，已有城池）进游戏「任务 → 新手任务」：
   - 出现「首个城池·发展助力」，`current/target = 1/1`，状态「可领取」，奖励显示粮钢油稀各 2000 万 + 黄金 2000 万。
   - 点领奖 → 城市各资源 + 2000 万、黄金 + 2000 万，任务变「已领取」，刷新后不再可领（只此一次）。
4. 用新建小号验证：建首城后同样出现并领一次，之后保持「已领取」。
5. 管理端「数据管理 → 任务配置」该行显示行为「拥有首个城池」。