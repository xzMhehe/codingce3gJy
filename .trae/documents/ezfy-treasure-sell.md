# 二战风云：宝物出售给系统（统一 20 万黄金/件，收 10% 手续费）

## Context

用户需求：在宝物页 `games/ezfy?cur=treasure` 新增"采集宝物出售给系统"功能。经确认：

- 采集宝物（`ezfy_equipment` 表，ID 27-35 的 9 种珠宝）**没有品质差异**（全 Tier1、属性全 +5）。
- 用户明确：**统一售价 20 万黄金/件**，卖出后收 **10% 手续费**（玩家实得 18 万/件），货币为**黄金**。
- 只卖**未穿戴**的宝物（`officer_id = 0`，即背包里的采集宝物）。

当前宝物页只展示宝物名 × 数量，无任何出售操作，需要前后端一起补。

## 现状调研（已确认）

- 前端宝物页：`web/src/views/ezfy/modules/EzfyShop.vue` L69-81（`cur==='treasure'`），数据源 `ezfy.bagTreasures`（字段 cfg_id/name/count）。
- `getBagTreasures` 赋值处：`web/src/views/ezfy/Ezfy.vue:2098-2099` 进入宝物页调用 `loadBag()`。宝物聚合在后端 `server/internal/handler/ezfy/ezfy_api.go:2420-2440`（Bag handler），`ezfy_equipment` 按 cfg_id 分组、`officer_id=0`、仅 `ezfyCollectibleTreasureNames()` 的宝物。
- 可出售宝物识别：`ezfyCollectibleTreasureNames()`（`ezfy_geo.go:376`）返回 9 种采集宝物名集合。
- 可参考的出售实现：`ExchangeSysSell`（`ezfy_chat_exchange.go:840-915`）——加锁防连点、换算、扣 10% 手续费（常量 `ezfySysSellFeePct = 10` 在 L824）、黄金封顶处理。
- `saveCityRes(city)`（`ezfy.go:1834`）落库城市资源含 gold。
- `addEquipment(city, cfg)`（`ezfy_officer.go:827`）是宝物写库函数；宝物记录 `OfficerId` 字段（0=未穿戴）。
- `ezfyResMaxOf("gold")` 黄金上限工具。

## 实现步骤

### 1. 后端：新增宝物出售 handler

文件：`server/internal/handler/ezfy/ezfy_chat_exchange.go`（紧邻 `ExchangeSysSell` 放，或就近函数文件）。

- 新增常量：`ezfyTreasureSellPrice = 200000`（20 万黄金/件）。
- 新增 `func (h *EzfyHandler) ExchangeTreasureSell(c *gin.Context)`：
  1. 取 uid；`ezfyExchangeLock(uid)` 加锁（防连点重复出售），defer 解锁。
  2. 解析 body `{cfg_id int, count int}`；`cfg_id > 0 && count > 0`，非法返回 ParamError。
  3. 校验 `cfg_id` 属于采集宝物：`cfge, ok := ezfyCfg.equipments[cfg_id]` 且 `ezfyCollectibleTreasureNames()[cfge.Name]`，否则 ParamError("非可出售宝物")。
  4. 查持有量：`h.DB.Model(&model.EzfyEquipment{}).Where("user_id=? AND officer_id=0 AND cfg_id=?", uid, cfg_id).Count(&own)`；`own < count` → ParamError("宝物不足(现有N件)")。
  5. 删除 count 条未穿戴宝物记录（`DELETE ... WHERE user_id=? AND officer_id=0 AND cfg_id=? LIMIT count`——GORM 可先 `Limit(count).Delete`）。
  6. 计算金额：`base = count * ezfyTreasureSellPrice`；`received = base*(100-ezfySysSellFeePct)/100`；`fee = base - received`。
  7. 加黄金并封顶：仿照 `ExchangeSysSell` 用 `ezfyResMaxOf("gold")` 算 `lost`，`city.Gold = after`，`h.saveCityRes(&city)`。
  8. 返回 `{msg, received_gold, fee, gold_lost, lost_gold}`（msg 提示"出售宝物X×N 成功，获得 N 黄金（手续费已扣 N）"）。
  9. 宝物资质/训练队列等缓存：参照其它写操作在入口 `Del` 相关缓存（Bag 有玩家级缓存 `ezfyPageCacheGet(uid,"officers")` 等，出售后需失效宝物所属缓存，避免刷新仍显示旧数量）。

### 2. 后端：路由注册

文件：`server/internal/router/router.go` L681 后新增一行：
```go
ezfyG.POST("/exchange/treasure-sell", ezfyH.ExchangeTreasureSell)
```

### 3. 前端：宝物页加出售交互

文件：`web/src/views/ezfy/modules/EzfyShop.vue` L73-75（宝物 `v-for` 行内）。

- 显示售价提示：每种宝物行内标注"可售 20万黄金/件（到手 18万）"。
- 每行加 `[出售]` 按钮 → 打开一个出售面板（沿用现有 use-box 风格：数量 input + 合计 + [确认出售]/[取消]）。
- 出售面板 state：在 `Ezfy.vue` 的 data 加 `sellTreasure`（当前要卖的宝物对象）、`sellTreasureCount`；按钮点击设 `ezfy.sellTreasure = t`。
- 新增方法 `doSellTreasure()`：`api.post('/games/ezfy/exchange/treasure-sell', {cfg_id, count})`，成功后 `alert/msg` 提示返回 msg，关闭面板并 `ezfy.loadBag()` 刷新（宝物数量会变）。
- 底部按钮区保留现有入口。

## 验证

1. `cd qqjiayuan/server && go build ./...` 编译通过；`go vet ./internal/handler/ezfy/` 无告警。
2. 手工逻辑核对：
   - 无此类宝物时出售 → 返回"宝物不足"。
   - 有 N 件 → 出售 N 件后背包数量减 N，黄金 +received（受上限约束，超出丢量提示）。
   - 重复发送同一请求（连点）→ 加锁只成功一次。
   - 穿戴中的宝物（officer_id≠0）不计入可卖数、不可被出售。
3. 前端浏览器验证：宝物页显示每件宝物的"20万/件"提示 + [出售]按钮，选数量后实得=出售件×18万，出售成功后列表刷新、黄金变多。

## 交付

按用户部署约定：改动完成助手不碰服务器；打包 `cd qqjiayuan && ./pack-linxu.sh` → `Linuxbushu.tar.gz` + `md5` + 部署命令交给用户，部署由用户执行。