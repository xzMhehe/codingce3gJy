# banbenA 分支 —— 只保留列出的功能，源码出售版本

## Context（背景）

用户要卖源码回本继续开发，新建本地分支 **`banbenA`**，供出售的分支只保留：**家园论坛（社区/论坛）、魔法花园(garden)、开心农场(farm)、精武堂(jwt)、用户端、管理端**。

已与用户确认：
- 三大游戏 **彻底删除**（二战风云 `ezfy` / 幻想西游·纵横四海 `hxxy` / 抢车位 `park`），含全部源码、路由、seed、前端。
- 游戏大厅里的 **未开发占位游戏条目一并清掉**，大厅只留 魔法花园/开心农场/精武堂。
- 数据库名在分支上改为 `banbenA`，并在本地 MySQL(已确认 3306 运行中) 新建 `banbenA` 库。

不删除：garden/farm/jwt 的任何代码；社区/用户/管理基础设施（user/auth/plaza/bbs/board/thread/sign/friend/message/chat/space/admin 等）。

## 步骤

### 1. 建分支
- `git checkout -b banbenA`（基于当前 main，本地分支）。

### 2. 后端删除整批文件（server/）
- `internal/handler/`：删 `ezfy_*.go`(约50个含测试)、`hxxy*.go`(12个)、`park.go`。
- `internal/model/`：删 `ezfy.go`、`hxxy.go`、`park.go`。
- `internal/seed/`：删 `ezfy.go`、`ezfy_cfg_gen.go`、`ezfy_officer_cfg.go`、`ezfy_officer_pool.go`、`ezfy_*_test.go`、`seed_hxxy.go`、整目录 `hxxy_data/`。
- `cmd/ezfymigrate/` 整目录删除（它依赖 EzfyHandler，不删则编译失败）。

### 3. 后端 `internal/router/router.go` 剔除路由
删（行号实测）：
- handler 声明：`59`(parkH)、`61`(hxH)、`62`(ezfyH)、`63-65`(注释+`go ezfyH.BgTickBattles()`)。
- 用户端游戏组：`341-354`(park)、`399-531`(hxxy)、`533-734`(ezfy)。
- admin 组：`1026-1034`(park管理)、`1060-1080`(hxxy/xy管理)、`1082-1390`(ezfy管理)。
- 保留 jwt/farm/garden 路由与「我的游戏」块。

### 4. 后端 `internal/seed/seed.go` 剔除建表/种子
- AutoMigrate：`64`(Park*)，`92-151`(hxxy + ezfy model)。
- Run() 内 二战 历史回填块 `156-485`。
- Run() 调用 `657`(seedParkData)、`659`(seedHxxy)、`660`(seedEzfy) 删除（658 seedJwt 保留）。
- `seedParkData` 函数体 `1153-1217` 整个删除（依赖已删 model.CarShop）。
- `seedRBAC` 权限 mod：`1739`(抢车位)、`1743-1761`(幻想西游+二战) 删除；可选删 `1771-1772`。
- `seedGameBoards`：`2420-2445` 只留 魔法花园/开心农场/精武堂 + 游戏综合反馈/研发/交流 功能板块，删其余占位；示例帖删 `2467-2468`。
- `seedGames`：`2595-2616` games 切片只留 魔法花园/开心农场/精武堂，删其余（狂抢车位/好友买卖/台球/猜数/大富翁/大话吹牛/水果乐园/幻想西游/二战风云 及永恒修仙/婚礼殿堂/家园宠物/全民猎马/家园股市/竞技场/砸金蛋/六合彩占位）。

### 5. 后端 `internal/handler/social.go`
删 `539-544`（引用已删 `model.EzfyProfile` 的 `ezfy_profile.game_uid` 找人逻辑）。

### 6. 数据库名 & 本地建库
- `server/config.yaml`：`mysql.dbname` `qq_jiayuan` → `banbenA`。
- 编译并启动一次（`go run .` 或 `cd server && go run ./cmd/dbinit`），`pkg/database.EnsureDatabase` 会自动 `CREATE DATABASE IF NOT EXISTS banbenA` 并建表迁移。无 mysql 客户端，用该方式建库。

### 7. 用户端 web/
- 删视图：`views/Ezfy.vue`、`views/Xiyou.vue`、`views/Park.vue`。
- `router/index.js`：删 `73`(/games/park)、`75`(/games/hxxy)、`76`(/games/ezfy)。
- `App.vue`：删 `immersive` 计算属性及其模板引用（对应 /games/ezfy）。
- `views/Games.vue`：删 `66` 指向 `/games/hxxy` 的「马年新区」推广 `<li>`。

### 8. 管理端 admin-web/
- 删 `components/admin/` 下 `AdminEzfy*.vue`(25)、`AdminPark*.vue`(2)、`AdminXy*.vue`(4)。
- `menu.js`：删 import 块 `62-97`；删菜单节点 `235-241`(g-park)、`252-260`(g-xy)、`261-297`(g-ezfy)。

### 9. 编译/构建验证
- 后端：`cd qqjiayuan/server && go build ./... && go build -o server .`（应无错误）。
- 用户端：`cd qqjiayuan/web && npm run build`。
- 管理端：`cd qqjiayuan/admin-web && npm run build`。
- 若编译报被删符号引用，回到对应文件补齐删除。

## 验证（端到端）

1. `go build ./...` 通过，无 `undefined: Ezfy/EzfyHandler/hxxy/Park/CarShop` 之类报错。
2. 启动后端连 `banbenA` 库，确认自动建库建表、种子只含保留四功能（论坛+三游戏）。
3. 用户端访问 `/games` 大厅：只见 魔法花园/开心农场/精武堂；无 ezfy/hxxy/park。
4. 管理端菜单：游戏管理下只有 魔法花园/开心农场/精武堂（+游戏大厅）。
5. DB 侧：`SHOW TABLES`（用 dbq 或应用日志）无 `ezfy_*`、`hxxy_*`、`park/car_*` 表。

## 交付
分支 `banbenA` 保留在本地，可被 `git switch banbenA` 切换使用；数据库 `banbenA` 已建好。部署打包（Linuxbushu 等）由用户自行按既有流程执行。