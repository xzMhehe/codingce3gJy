# 二战风云 交易所「向系统出售资源」功能

## Context（背景）

当前交易所只有「挂单出售」（玩家之间/系统挂单）玩法。玩家没有多余的资源快速换成黄金的通道。

本次新增「**向系统出售资源**」：玩家把资源**直接卖给系统**（不走挂单 / 不产生订单行），系统按**可配置的回收比例**换成黄金，直接扣城市资源、加城市黄金。玩家拿到的黄金还要扣 **10% 手续费**。

已与用户确认的定价口径（**每 100 单位 → N 黄金**）：
- 粮食 100:10、钢铁 100:10、石油 100:20、稀矿 100:25（默认值，可在「交易行维护」后台改）
- 玩家实得黄金 = 系统回收黄金 × 0.9（扣 10% 手续费）

**黄金超上限的提醒规则**（用户明确要求）：
- 加黄金时若城市已有黄金 + 本次到账会超过「黄金资源最大值」配置，超出部分会丢失 → **必须提醒玩家**；
- 没超过 → **不提醒**。

> 参考已确认的既有口径：黄金仍受 `res_max_gold`（资源最大值，默认 21 亿）硬上限约束，加金走 `ezfyAddResMax` 同类封顶逻辑。

## 改动清单

### 1. 后端：配置字段（`server/internal/model/ezfy.go`）
在 `EzfyCfgLimit` 末尾追加 4 个字段，存「每 100 单位 → N 黄金」：
- `SysSellFood  int`（默认 10）
- `SysSellSteel int`（默认 10）
- `SysSellOil   int`（默认 20）
- `SysSellRare  int`（默认 25）

沿用 `gorm:"default:...;comment:..." json:...` 风格。

### 2. 后端：读取助手（`server/internal/handler/ezfy_geo.go`）
新增 `ezfySysSellRatio(esType int) int`，按 es_type(1粮/2钢/3油/4稀) 返回 N（每100单位黄金），`<=0` 回落默认（10/10/20/25）。与 `ezfySellPriceMax` 同风格。

### 3. 后端：管理端配置读写（`server/internal/handler/ezfy_admin_round9.go`）
复用 `GET/PUT /admin/ezfy-build-limit`：
- **Get**（约 L24-192）：默认值字面量里补 4 个字段的默认；后续 `First(&lim,1)` 已存在。新增一个兜底：4 个字段 `<=0` 一律归一化到各自默认。
- **Update**（L198-895）：
  - 入参 struct 补 4 个 `*int`（`sys_sell_food/steel/oil/rare`）；
  - 校验 0~100000（防呆）；
  - 命中后写 `lim` 对应字段；
  - 最后 L848 的 `Updates(map)` 里补显式写入 4 个列（避免 GORM 零值被吞）。

### 4. 后端：新出售接口 + 列表下发比例（`server/internal/handler/ezfy_chat_exchange.go`）
- **ExchangeSysSell** `POST /games/ezfy/exchange/sys-sell`，body `{es_type, es_count}`：
  1. 校验 es_type ∈ 1-4、es_count > 0；
  2. `city := getOrCreateCity(uid)` + `calcResource(&city)`；
  3. `stock < es_count` → `resp.ParamError`（同 `ExchangeSell` 口径）；
  4. 从 `city` 扣资源（food/steel/oil/rare）；
  5. `base := es_count * ezfySysSellRatio(es_type) / 100`（整数除法），`received := base * 90 / 100`（即回收价×0.9，扣 10% 手续费）；
  6. **黄金封顶与丢量判断**：`goldMax := ezfyResMaxOf("gold")`；`before := city.Gold`；`after := min(goldMax, before + received)`；`lost := (before + received) - after`；`city.Gold = after`；
  7. `saveCityRes(&city)`；
  8. 返回：`msg`（含得到多少黄金、扣了多少手续费）、`received_gold`、`fee`、`gold_lost`（bool）、`lost_gold`（数值）。`gold_lost==true` 时 msg 里注明「超出黄金上限 X 丢失」。
- **ExchangeList**（L461-529）：在返回 map 里追加：
  - `"sys_sell_ratio"`: {1:…, 2:…, 3:…, 4:…}（各资源每100单位黄金，供前端展示）
  - `"sys_sell_fee"`: 10（手续费百分比）
  - `"gold_max"`: `ezfyResMaxOf("gold")`（前端可判断是否临近上限）

### 5. 后端：注册路由（`server/internal/router/router.go` L663-666 附近）
`ezfyG.POST("/exchange/sys-sell", ezfyH.ExchangeSysSell)`。

### 6. 管理端 UI（`admin-web/src/components/admin/AdminEzfyExchange.vue`）
新增第三个 `el-tab-pane`「**系统回收比例**」：
- 4 个 `el-input-number`（粮食/钢铁/石油/稀矿，每100单位黄金）；
- 旁边灰字说明：玩家卖 100 单位资源 → N 黄金，实得再扣 10% 手续费；
- 进入 tab 时 `GET /admin/ezfy-build-limit` 拉取，保存时 `PUT /admin/ezfy-build-limit`（只带这 4 个字段）。

### 7. 游戏端 UI（`web/src/views/Ezfy.vue`）
在交易所「挂单出售」面板下方新增「**向系统出售**」面板：
- 资源类型下拉（复用现有 `sellType` 逻辑）、数量输入 `sellSysCount`；
- 服务器下发的 `sys_sell_ratio` 实时计算：应得黄金 = 数量×ratio/100，实得 = 应得×0.9；展示两者；
- 按钮 `doExchangeSysSell()` → `POST /games/ezfy/exchange/sys-sell`；
  - 成功后按返回值 `alert`：若 `gold_lost` true，额外提示「超出黄金上限 X 已丢失」；
  - 成功后刷新 `loadExchange()` / `load()`（黄金与城市资源数字都要更新）。
- `data()` 里加 `sellSysCount: 0`、`sysSellRatio: {}`、`sysSellFee: 10`、`goldMax: 0`；`loadExchange()` 里读回后端下发字段。

## 复用的既有工具
- `ezfyResMaxOf(res)` — `ezfy_geo.go`：取「资源最大值」，直接用于黄金上限判断。
- `ezfyAddResMax` 用到的 `ezfyClampRes` 思路；黄金加金按本方案第 4 步手动 min 计算即可（便于拿 lost 数值）。
- `getOrCreateCity` / `calcResource` / `saveCityRes` / `resp.ParamError` / `resp.OK` — 既有交易所接口同款。

## 验证（端到端）
1. 启动 server（Go）+ web（npm run dev）+ admin-web。
2. 管理端「交易行维护 → 系统回收比例」：确认默认 10/10/20/25 可读写并保存。
3. 游戏端交易所，选石油、数量 100 → 应得 20、实得 18（费 2）；确认「向系统出售」成功，城市黄金 +18、石油 -100、手工挂单不变。
4. 把城市黄金调到接近 `res_max_gold`（管理端发资源），再出售较大数量 → 应收到账被截断，界面提示「超出黄金上限 X 已丢失」。
5. 正常数量（未超上限）出售 → 无丢失提示。
6. 数量超过城市资源库存 → 返回「资源不足」。