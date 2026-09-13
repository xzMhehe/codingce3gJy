# 幻想西游完整复刻进 QQ家园 — 实施计划

## 一、背景与目标

将 `西游一键端24082201\phpstudy_pro\WWW`（PHP WAP 版"幻想西游"）**完整复刻**为家园项目（`qqjiayuan`，Go+Vue 前后端分离）内的一个游戏模块，集成方式与精武堂一致：

- 嵌套进家园游戏大厅，登录家园账号即可进入
- 新增数据表统一前缀 `hxxy_`，游戏用户表与家园 `users` 表通过 `user_id` 关联
- 游戏内独立货币：银两 + 金豆（与 G 币不通）
- 前端保持原版 WAP 文字链交互风格（与精武堂页面风格一致）
- 提取的数据/UI 文案中**过滤掉"诺哈"等外站广告内容**（项目硬约束）

## 二、原版调研结论（复刻依据）

### 2.1 数据规模（全部可提取）
| 数据 | 来源 | 规模 |
|---|---|---|
| 地图节点 | `fqxy/xdt/*.php`（每个文件是一段纯 JSON 字符串） | 90 个区域约 1800 节点，含上下左右出口+跳转链接 |
| NPC 属性 | `fqxy/npc/npcxx01~14.php`（按 `$npcc==id` 分支赋值 `$nname/$ndj/$nhp/$ngj...`） | 约 2000 个 NPC |
| NPC 放置 | `fqxy/map/*.php` 中 `$clj[]=10; $npc[]=N;`（攻击链接，上下文含节点/难度） | 按区域提取刷怪表 |
| 物品 | `data/xyy.sql` 表 `wpxx` | 1108 条 |
| 装备 | `data/xyy.sql` 表 `zbxx` | 760 条（含 hp/gj/mg/fy/冰火雷攻防/门派/部位） |
| 技能 | `fqxy/data/jnxx.php`（jnid/分类/名字/倍率 shxs） | 全部门派技能+普攻+捕捉/查看 |
| 宠物 | `fqxy/cw/cwxx01.php` 等（cwid/基础属性/星级/品质） | 宠物种族表 |
| BOSS | `fqxy/npc/bossxx01~8.php` | 按 id 段分文件 |
| 门派 | `fqxy/template/xy295.php` | 5 门派：1将军府 2龙宫 3月宫(限女) 4方寸山 5普陀山(限男) |

### 2.2 核心公式（照抄原版）
- **基础属性（ztt.php，按门派）**：`maxhp=(lv+15)²×K`（将军府2/龙宫2/月宫4/方寸山2/普陀山3）；`maxmp=⌈(lv+30)(lv+20)/4⌉`；`atk=(lv+1)(lv+2)×K+300`（将军府4，余3）；`def=(lv+1)²×K+200`（龙宫3，余2）；`mg=(lv+1)(lv+2)×K+300`（方寸山4/普陀山4，余3）；普通攻击值=⌈max/1.3⌉；元素攻防=等级+装备加成
- **升级（ini_pz04/05.php）**：所需经验 `(lv+1)³(lv+2)+200`；升级后 等级+1、背包容量+50、属性重算回满；修炼经验上限 `(lv+1)⁴(lv+2)+100`
- **伤害（ltpk03.php）**：元素差 `gg1=Σ(攻-防)`（冰火雷）三支——优势：`((atk+100)×(1+gg1/300)−def)×技能倍率×1.3`；劣势：`(atk−(def+100)×(1+|gg1|/300))×倍率×1.1`；均势：`((atk+100)−def)×倍率×1.2`；10% 暴击 `×(1+rand(1,20)/10)`；最终伤害=`rand(⌈s/2⌉,s)`。法术门派（月宫/方寸山/普陀山）以魔攻作 atk
- **宠物成长（cwztt.php）**：`hp=((lv+15)²×2+base)×系数`、`mp=((lv+30)(lv+20)/4+base)×系数`、`atk/mg=((lv+1)(lv+2)×3+base)×系数+300`、`def=((lv+1)²×2+base)×系数+200`，系数含星级/变异/品质
- **装备加成（ztt.php）**：基础+装备+星级(经 xj.php)+镶嵌宝石+头衔+丹药+住宅家具+修炼+套装+星盘 逐项累加

## 三、总体架构

```
后端 qqjiayuan/server
├── tools/hxxy_extract/extract.php      # PHP CLI 提取脚本（用一键端自带 php.exe 运行）
├── internal/seed/hxxy_data/*.json      # 提取产物（go:embed 进 seed）
├── internal/seed/seed.go               # +SeedHxxy(db)：空表时灌入
├── internal/model/hxxy.go              # 全部 hxxy_ 模型（GORM，TableName 前缀）
├── internal/handler/hxxy/              # 游戏逻辑（按域拆分多个文件）
├── internal/handler/hxxy_admin.go      # 管理端接口
└── internal/router/router.go           # /api/games/hxxy/*（authed）+ /api/admin/hxxy/*
前端 qqjiayuan/web
├── src/views/Xiyou.vue                 # 游戏外壳（WAP 风格导航 + 主界面/地图/状态）
├── src/views/xy/*.vue                  # 战斗/背包/装备/技能/宠物/商店/银行/任务/副本/社交等子视图
└── src/router/index.js                 # /games/xy（meta.auth）
管理端 qqjiayuan/admin-web
├── src/menu.js                         # 「幻想西游」菜单组
└── src/views/AdminXy*.vue              # 玩家管理/数据管理/道具发放
```

复用现有模式（不新造轮子）：
- 路由注册、JWT 取当前用户：照 `router.go` authed 组 + `handler/jwt.go` 写法
- 商店扣费/写流水：参照 `handler/jwt.go` L367-414（改为银两）
- 前端单页多视图 + token 处理：照 `views/Jingwt.vue`
- 游戏大厅条目：`seed.go` 已有"幻想西游"论坛板块（L1881 附近），补 `games` 表记录 `path=/games/xy`

## 四、数据库设计（hxxy_ 前缀，均含自增 id）

- `hxxy_players`：user_id(唯一,关联 users.id)、name、sex、sect(1-5)、level、exp、xiulian_exp、xiulian_switch、hp、mp、money(银两)、bank(存款)、beans(金豆)、vip、vip_exp、title_id、bag_cap、wh_cap、emz(恶名)、map_x(dtx)、map_y(dty)、时间戳
- `hxxy_map_nodes`：node_id、dtx、dty、name、desc、up/down/left/right、*_jump、is_jump
- `hxxy_npcs`：npc_id、name、desc、take(被打语)、level、hp/max、mp/max、atk、mg、def、mf、bg/hg/lg/bf/hf/lf、kind(战斗/商店/功能)、shop_id
- `hxxy_spawns`：dtx、dty(0=区域随机池)、npc_id、difficulty(普通/困难)
- `hxxy_items`：wpxx 全字段 + effect(JSON: 恢复血蓝/加经验/属性丹/每日限用次数)
- `hxxy_equips`：zbxx 全字段（slot 部位、sect 限制、攻防元素、星级相关）
- `hxxy_bag`：player_id、kind(item/equip/gem/pet)、ref_id、count、bind、额外JSON(装备星级/孔/镶嵌/每日已用次数)
- `hxxy_skills`：jnxx 全字段 + 耗蓝/学习等级/门派限制；`hxxy_player_skills`：player_id、skill_id、level
- `hxxy_pet_species`：cwid、name、基础属性、星级、品质；`hxxy_pets`：player_id、species_id、name、level、exp、star、mutate、quality、fighting
- `hxxy_battles`：player_id、type(npc/pk/boss/dungeon)、round、敌方快照(JSON)、我方快照(JSON)、log(JSON)、status；`hxxy_battle_logs`：胜负/奖励/掉落历史
- `hxxy_quests` + `hxxy_player_quests`（状态/进度/计数）；`hxxy_dungeon_runs`（副本层数进度/当日次数）
- `hxxy_bosses`（boss 属性+刷新时间）；`hxxy_titles` + `hxxy_player_titles`
- `hxxy_gangs` + `hxxy_gang_members`；`hxxy_marriages`；`hxxy_houses`（家具 JSON）；`hxxy_friends`；`hxxy_chats`（世界聊天）；`hxxy_signins`；`hxxy_stalls`（挂售）；`hxxy_wallet_logs`（银两/金豆流水）

## 五、后端 API（/api/games/hxxy，JWT 鉴权）

| 域 | 端点 |
|---|---|
| 入口 | GET /status（无角色→需建角）；POST /create {name,sex,sect(性别校验)}；GET /state（当前节点信息+出口+本节点NPC/怪物+功能菜单）；GET /attrs |
| 地图 | POST /move {dir}；POST /jump {dtx,dty} |
| 战斗 | POST /battle/start {npc_id}；GET /battle/state；POST /battle/action {act: attack/skill/catch/flee/item, skill_id?}（回合制：我方出手→敌方出手，按 2.2 公式结算，写 hxxy_battles；胜利发经验/银两/掉落+升级判定） |
| 背包 | GET /bag；POST /bag/use（药品/丹药，含每日限次）；POST /bag/discard；GET /warehouse；POST /warehouse/deposit|takeout |
| 装备 | POST /equip/wear|takeoff；POST /equip/upgrade（星级强化，zbdz 材料）；POST /equip/hole（打孔）；POST /equip/gem（镶嵌）；GET /equip/:id 详情 |
| 技能 | GET /skills；POST /skills/learn|upgrade（消耗银两+等级） |
| 宠物 | GET /pets；POST /pets/fight|rest|free|rename；战斗中 catch 按原版捕捉判定；宠物随战斗获得经验并按 cwztt 公式成长 |
| 商店 | GET /shop/:npc_id；POST /shop/buy {ref_id,count}；POST /shop/sell（银两，写钱包流水） |
| 银行 | GET /bank；POST /bank/deposit|withdraw |
| 任务 | GET /quests；POST /quests/accept|submit（打怪计数/收集/对话三类通用引擎） |
| 副本 | GET /dungeons；POST /dungeons/enter|next|exit（大雁塔/兵马俑/碑林/小雁塔/冰风谷等，逐层战斗） |
| BOSS | GET /bosses；POST /bosses/challenge（世界 BOSS，带刷新时间） |
| 修炼 | GET /cultivate；POST /cultivate/toggle（开关，吃经验道具时走修炼经验池） |
| 社交 | 好友 add/agree/list；聊天 GET /chat?since_id / POST /chat；帮派 create/join/leave/donate/kick；结婚 propose/agree/divorce；住宅 GET /house + 家具摆放加成 |
| 其他 | 排行 GET /rank/:type(等级/财富/宠物)；签到 POST /signin（连签递增奖励）；头衔 list/wear；摆摊 list/sell/buy/cancel；VIP info/upgrade（金豆）；充值 POST /pay（演示金豆充值码）；星盘简单版（金豆点亮属性加成） |

管理端 /api/admin/hxxy：players 列表/详情/改属性/封禁、发道具（入背包）、items/equips/npcs/skills/spawns 分页查询、battle_logs 查询、钱包流水查询。

## 六、前端页面（WAP 文字链风格，与原版一致）

- `Xiyou.vue` 外壳：顶部返回家园/游戏名导航 + 底部菜单（人物/背包/技能/宠物/任务/更多），中间子视图切换（`v-if`，照 Jingwt.vue）
- `views/xy/`：`Main.vue`（地图描述+上下左右文字链+遇怪链接+NPC交互）、`Battle.vue`（回合战况文字滚动+普攻/技能/捕捉/逃跑/用药）、`Bag.vue`（背包/仓库 tab）、`Equip.vue`（穿戴/卸下/强化/打孔/镶嵌）、`Skills.vue`、`Pets.vue`、`Shop.vue`（列表→详情→确认三页，与原版一致）、`Bank.vue`、`Quests.vue`、`Dungeon.vue`、`Boss.vue`、`Social.vue`（好友/聊天/帮派/结婚）、`Misc.vue`（签到/VIP/头衔/住宅/摆摊/排行/充值/修炼）
- 建角页：5 门派介绍文案照抄 xy295.php（过滤广告），性别限制生效
- 所有列表单列左对齐，数值格式"当前-上限"风格与家园一致

## 七、实施步骤（顺序执行，每步可验证）

1. **数据提取**：写 `tools/hxxy_extract/extract.php`（用一键端 php.exe），产出 maps/npcs/spawns/items/equips/skills/pets/bosses/quests JSON 至 `internal/seed/hxxy_data/`；提取时过滤"诺哈"字样；输出各 JSON 条数校验（地图≈1800、物品≈1108、装备≈760、NPC 按 npcxx 覆盖数）
2. **模型与种子**：`model/hxxy.go` 全部表 + `seed.go` SeedHxxy（go:embed JSON，空表灌入）+ games 表补"幻想西游"条目
3. **后端核心**：建角/属性成长/地图行走/战斗引擎（NPC 战，公式照 2.2）+ battle 持久化
4. **后端系统**：背包/装备/技能/宠物/商店/银行 → 任务/副本/BOSS/修炼/头衔/帮派/结婚/住宅/好友/聊天/排行/签到/摆摊/VIP/充值
5. **前端**：Xiyou.vue + views/xy/* 全部视图 + 路由
6. **管理端**：menu.js + AdminXyPlayers/AdminXyData + 后端 admin 接口
7. **联调验证**（见下）

## 八、验证方式

1. `php extract.php` → 检查 JSON 条数与样例（无"诺哈"残留）
2. `cd server && go build ./... && go run .`（重建重启 8080），无报错；首次启动日志显示 hxxy 种子灌入条数
3. API 冒烟：登录测试号(10007/admin123) → POST /api/games/hxxy/create 建角 → /state 查地图 → /move 移动 → /battle/start+action 打怪升级 → /bag/use 用药 → /shop/buy 买药 → /equip/wear 穿装备
4. `cd web && npm run build` 通过；浏览器（Ctrl+F5 强刷）走通全流程：游戏大厅看到"幻想西游"→ 进入 → 建角选门派 → 长安城走路 → 打怪 → 捕捉宠物 → 买药穿装备 → 世界聊天
5. `cd admin-web && npm run build` 通过；/admin-ui/ 查看幻想西游菜单：玩家列表/发道具/数据查询
