# 二战风云 军团：去重 + 军团聊天独立tab + 入团审核

## Context（背景）

玩家反馈军团页三处问题：

1. **军团信息 tab 里重复**：`军团信息` tab 下方同时又摆了「军团列表」表格和「军团聊天」面板，且它们其实已有各自独立的用途（列表给没入团的人浏览/申请，聊天是军团内部聊天），放进「军团信息」里显得冗余。
2. **军团聊天应独立成 tab**：希望聊天单独一个 tab，专门做军团内部聊天。
3. **入团需要审核**：目前 `CorpsJoin` 是点「加入」直接 `DB.Create` 入团（[ezfy_api.go L1012-1045](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/handler/ezfy_api.go#L1012-1045)），没有审核环节。希望军团可配置「审核开关」，默认**无需审核**；团长开启后，新成员申请需**军团长**审核通过才正式入团。

已与用户确认：
- 军团 tab 最终布局：**军团信息 | 军团列表 | 军团聊天 | 军团外交 | 军团宣战 | 军团商城**（军团列表、军团聊天都保留、各自成 tab）。
- 审核：军团可配置开关，默认**无需审核**（直接入团）；开启后需**军团长**审核。
- 审核人：**仅军团长**。

## 现有机制（复用的点）
- 军团数据模型 `EzfyCorps / EzfyCorpsMember / EzfyCorpsChat`（[ezfy.go L914-1031](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/model/ezfy.go#L914-1031)）。
- 军团接口已在 `ezfy_api.go`：`CorpsList/Create/Join/Leave/Kick/Notice/Chats/Chat`；`CorpsMembers` 在 `ezfy_chat_exchange.go`。路由见 [router.go L602-622](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/router/router.go#L602-622)。
- 军团页前端在 `web/src/views/Ezfy.vue`：tab 结构 L2250-2259、`corpsTab` 数据 L4335、`loadCorps()` L6601、`switchCorpsTab()` L8156、`doJoinCorps()` L8117、`doCorpsChat()` L8146。
- 权限口径：团长= `ezfyCorpsLeaderOf`；可邮件=团长或副团长（`ezfyCanMailCorps`）。审核仅团长，用 `IsLeader==1` / `LeaderUserId` 判定即可。

## 改动清单

### A. 后端：军团「审核开关」字段
在 `model.EzfyCorps`（[ezfy.go L914-924](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/model/ezfy.go#L914-924)）追加：
```go
NeedReview int `gorm:"default:0;comment:入团是否需审核(0=无需直接入团,1=需军团长审核)" json:"need_review"`
```
`ezfy_corps` 加入 AutoMigrate（[seed.go L118 已含](file:///d:/mxz-code/github/codingce3gJy/qqjiayuan/server/internal/seed/seed.go#L118)，GORM 自动补列）。seed 补列脚本（`seed.go` 附近，参照其它列）为老库 `ALTER ADD COLUMN need_review int DEFAULT 0`，避免老军团为空。

### B. 后端：新增「入团申请」模型 + 审核接口
新增模型 `EzfyCorpsApply`（表 `ezfy_corps_apply`）：
```
ID, CorpsId, UserId, Status(0待审 1通过 2拒绝), CreatedAt
```
加入 AutoMigrate seed 列表。

在 `ezfy_api.go`（或新建 `ezfy_corps_apply.go`，建议后者）新增 handler：

- **`CorpsApply`** `POST /corps/apply` `{corps_id}`：
  - 校验：未入团、军团结存在、联络中心≥1、人数未满（复用 `CorpsJoin` 现有校验片段）。
  - 读 `corps.NeedReview`：
    - `NeedReview==0` → 走**原来的直接入团**逻辑（建 `EzfyCorpsMember`、member_count+1）。
    - `NeedReview==1` → **幂等写申请**：若已有 status=0 的申请则提示「已申请，等待团长审核」；否则 `DB.Create(EzfyCorpsApply{...})`，提示「申请已提交，等待团长审核」。
  - 前端把原「[加入]」按钮改为「[申请]」，文案随结果。
- **`CorpsApplyList`** `GET /corps/apply` ：
  - 仅军团长：返回本人名下军团的 status=0 申请列表（含 `user_id`、昵称、`apply_id`、时间）。
- **`CorpsApplyHandle`** `POST /corps/apply/handle` `{apply_id, op}`（op：1=通过，2=拒绝）：
  - 仅军团长（`LeaderUserId==uid`）。
  - 通过 → **幂等**建 `EzfyCorpsMember` + `member_count+1`（复用入团校验），申请置 status=1。
  - 拒绝 → 申请置 status=2。
  - 校验军团人数上限；通过时若已入团则忽略。
- **`CorpsNeedReview`** `POST /corps/need-review` `{need_review}`：
  - 仅军团长，开/关 `corps.NeedReview`。

> 可选：原 `CorpsJoin`（直接入团）保留，前端不再调用；或改由 `CorpsApply` 统一入口。为最小改动，**前端改用 `/corps/apply`**，`CorpsJoin` 保留但不再从前端触发。

路由（`router.go` ezfyG 组）新增：
- `POST /corps/apply`
- `GET /corps/apply`
- `POST /corps/apply/handle`
- `POST /corps/need-review`

### C. 前端 tab 重构（`web/src/views/Ezfy.vue`）
1. **tab 栏**（L2253-2258）改为 6 项：军团信息 | 军团列表 | 军团聊天 | 军团外交 | 军团宣战 | 军团商城，映射 `corpsTab`：`info / list / chat / diplomacy / war / mall`。
2. **军团信息 tab**（L2262-2348）：
   - **删除**原「军团列表」面板（L2316-2328）和「军团聊天」面板（L2329-2340）。
   - 保留：我的军团信息 + 成员 + 任命 + 军团邮件 + 创建军团（未入团时）。
   - **新增「入团审核」区块（仅军团长）**：
     - 审核开关：`[开启审核]/[关闭审核]` → `doToggleNeedReview()`。
     - 待审申请列表：每行申请者昵称 + `[通过]/[拒绝]` → `doApplyHandle(apply, op)`；空则「(暂无待审申请)」。
   - **新增「入团申请」区块（仅未入团玩家 + 当前查看军团开启了审核）**：提示「该军团需军团长审核，申请后等待批复」。
3. **军团列表 tab**（新）：把原「军团列表」表格（L2316-2328）挪到这里，按钮改为 `[申请]`（`doJoinCorps(cp)` 改调 `/corps/apply`）。未入团才显示申请按钮。
4. **军团聊天 tab**（新）：把原「军团聊天」面板（L2329-2340）挪到这里；仅入团可见；复用 `loadCorps()` 的 chats 加载 + `doCorpsChat()` 发送。未入团提示「你还没有加入军团」。

### D. 前端数据/方法（`web/src/views/Ezfy.vue`）
- `data()`：
  - `corpsTab` 仍为字符串，值枚举改为 `info/list/chat/diplomacy/war/mall`。
  - 新增：`corpsApplies: []`、`myCorpsNeedReview: 0`、`myApplyStatus: 0`（0无申请/1待审/2通过/3拒绝，用于展示）。
- `switchCorpsTab(tab)`（L8156）：新增惰性加载——`list`→`loadCorps()`；`chat`→`loadCorps()`（拉 chats）；`info`→ 若团长补 `loadCorpsApplies()`。
- 新方法：`loadCorpsApplies()` GET `/corps/apply`；`doApplyHandle(a, op)` POST `/corps/apply/handle`；`doToggleNeedReview()` POST `/corps/need-review`。
- 改 `doJoinCorps(cp)` → POST `/corps/apply`，按返回 msg 或 `myApplyStatus` 更新按钮状态。
- `loadCorps()`（L6601）解析 `corps.list`/`my_corps` 时带上 `need_review`，存 `myCorpsNeedReview`；并刷新当前军团入团申请状态。

## 不涉及
- 不改世界聊天的「军团频道」（channel=2）——它和军团页聊天共用 `ezfy_corps_chat` 表但视角不同；本轮仅把军团页聊天挪进独立 tab，不动 chat 模块。
- 不新增管理端页面（入团审核是玩家侧玩法，可后续再加管理端）。

## 验证（端到端）
1. `go build ./...`、`npm run build`（web）通过。
2. 重启 server（AutoMigrate 建 `ezfy_corps_apply`、给 `ezfy_corps` 补 `need_review` 列）。
3. 团长号新建军团 → 军团信息 tab：只有「我的军团+成员+邮件+入团审核」，无军团列表/聊天重复；tab 显示 6 项。
4. 团长开关审核 → 小号在「军团列表」tab 点 `[申请]`；团长「军团信息 → 入团审核」看到申请 → `[通过]` 后小号入团；`[拒绝]` 则不入团且小号可重新申请。
5. 关闭审核时，小号点 `[申请]` 直接入团（无需团长操作）。
6. 军团聊天 tab：入团成员可收发，未入团提示加入。