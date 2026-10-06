# 军官技能随等级自动升级(二战风云 Ezfy)

## Context

用户需求:军官已学的技能随军官等级自动升级,技能分 1/2/3 级:

- 军官等级 <50 → 技能 1 级(= 现有效果,即当前一切现状);
- 50 ≤ 等级 < 100 → 2 级,数值效果 = 1 级的 **2 倍**;
- 等级 ≥ 100 → 3 级,数值效果 = 1 级的 **3 倍**;
- **加成类技能**统乘倍率(用户确认「战斗+辅助全乘」:攻防类/速度/黄金眼掠夺/机械改造恢复全部 ×N);
- **绝地反击**(反击类,非数值倍率):1 级 = 前 1 回合,2 级 = 前 2 回合,3 级 = 前 3 回合;
- 学习时按当前军官等级**自然定级**(100 级学 → 直接 3 级,50+ 级学 → 2 级)—— 技能等级**动态推导、不落库**;
- 前端:军官详情「技能」tab 显示当前技能等级(Lv.N 效果×N);战报军官描述带 Lv.N。

关键事实(已探明):
- 技能等级唯一输入是军官/守将当前等级,`ezfySkillLevelOf(level)` 一个函数搞定;
- 快照 JSON 里 `AtkCounter/DefCounter` 是 bool,改 int 会破坏老快照反序列化 → **保留 bool,新增 Rounds int 字段**,老快照 true → 回退 1 回合;
- `ezfyPageCache*` 缓存已禁用(空实现),detail/skills 接口加字段无需失效处理;
- `officerBattleDesc` 仅战报使用,改它不影响 detail;
- 「机械改造」的油耗/回收文案代码里从未实现(仅伤兵恢复 healTech 接入),本次只对 healTech ×N,不新接玩法。

## 实现步骤

### 1. 核心函数(ezfy_officer.go,放 officerSkillBattleBonus 上方)

```go
// ezfySkillLevelOf 按等级推技能等级:<50→1, 50~99→2, ≥100→3
func ezfySkillLevelOf(level int) int {
    if level < 50 { return 1 }
    if level < 100 { return 2 }
    return 3
}
func (h *EzfyHandler) officerSkillLevel(o *model.EzfyOfficer) int {
    if o == nil { return 1 }
    return ezfySkillLevelOf(o.Level)
}
func (h *EzfyHandler) officerSkillScale(o *model.EzfyOfficer) int { return h.officerSkillLevel(o) }
func (h *EzfyHandler) officerCounterRounds(o *model.EzfyOfficer) int {
    if o != nil && h.officerHasSkill(o, "绝地反击") { return h.officerSkillLevel(o) }
    return 0
}
func (h *EzfyHandler) officerSpeedSkillBonus(o *model.EzfyOfficer) int {
    if o != nil && h.officerSpeedSkill(o) { return 10 * h.officerSkillScale(o) }
    return 0
}
```

ezfySkillEffectText 后新增等级感知文本(绝地反击按回合数化,加成类**直接换算最终数值**):
```go
// ★ 2026-10-06 用户反馈：「攻击力+30% (效果×3)」太 low → 加成类直接放大数值嵌入文案
//   （Lv.3 尖兵突击 = 「攻击力+90%」）；绝地反击按「前N回合反击」。
func ezfySkillEffectTextAt(skill string, lv int) string {
    switch skill {
    case "绝地反击": return "前" + strconv.Itoa(lv) + "回合反击"
    case "尖兵突击": return "攻击力+" + strconv.Itoa(30*lv) + "%"
    case "弧形防御": return "防御力+" + strconv.Itoa(30*lv) + "%"
    case "火炮控制": return "陆军装甲攻击+" + strconv.Itoa(10*lv)
    case "坦克突袭": return "陆军速度+" + strconv.Itoa(10*lv) + "%"
    case "四指编队": return "空军对空攻击+" + strconv.Itoa(15*lv) + "%"
    case "闪电袭击": return "空军速度+" + strconv.Itoa(10*lv) + "%"
    case "狼群战术": return "海军对海攻击+" + strconv.Itoa(15*lv) + "%"
    case "越岛战术": return "海军速度+" + strconv.Itoa(10*lv) + "%"
    case "弹幕支援": return "城防攻击范围+" + strconv.Itoa(10*lv) + "%"
    case "黄金眼": return "侦查等级+" + strconv.Itoa(lv)
    case "机械改造": return "回收率+" + strconv.Itoa(10*lv) + "%, 出征油耗-" + strconv.Itoa(10*lv) + "%"
    default: return ""
    }
}
```

### 2. activity_target.go 的 general 版

```go
func generalSkillLevel(g *model.EzfyCfgGeneral) int { if g == nil { return 1 }; return ezfySkillLevelOf(g.Level) }
func generalSkillScale(g *model.EzyfyCfgGeneral) int { return generalSkillLevel(g) }
func generalCounterRounds(g *model.EzfyCfgGeneral) int {
    if g != nil && generalHasSkill(g, "绝地反击") { return generalSkillLevel(g) }
    return 0
}
```

### 3. 加成点乘倍率

| 位置 | 现值 | 改为 |
|---|---|---|
| ezfy_officer.go `officerSkillBattleBonus` L1489 | 尖兵突击+30 / 火炮控制+10 / 四指编队·狼群战术+15 | 先取 `scale := h.officerSkillScale(o)`,case 改 `30*scale` 等 |
| ezfy_officer.go `officerGuardBonus` L1537 | 弧形防御+30 / 弹幕支援+10 | 先取 scale,case 改 `30*scale` / `10*scale`(属性部分 `officerGuardAttrBonus` **不加倍**) |
| ezfy_order.go L640 / L1215 | `if h.officerSpeedSkill(lead){ travelSec=travelSec*100/110 }` | `if s:=h.officerSpeedSkillBonus(lead); s>0 { travelSec=travelSec*100/int64(100+s) }` |
| ezfy_order.go L2438-2440 | `if h.officerSpeedSkill(leadOfficer){ atkSpeedBonus += 10 }` | `atkSpeedBonus += h.officerSpeedSkillBonus(leadOfficer)` |
| activity_target.go L319-321 | 同上 `atkSpeedBonus += 10` | `atkSpeedBonus += h.officerSpeedSkillBonus(leadOfficer)` |
| ezfy_order.go L2529 黄金眼 | `lootTech += 10` | `lootTech += 10 * h.officerSkillScale(leadOfficer)` |
| ezfy_order.go L3057 黄金眼 | `lootRate += 10` | `lootRate += 10 * h.officerSkillScale(leadOfficer)`(保留上限 50) |
| ezfy_order.go L2940 机械改造 | `healTech += 10` | `healTech += 10 * h.officerSkillScale(leadOfficer)` |
| activity_target.go L463 机械改造 | `healTech += 10` | `healTech += 10 * h.officerSkillScale(leadOfficer)` |
| activity_target.go `ezfyActWildDefBonus` L77-99 | 弧形防御+30 / 弹幕支援+10 / 速度+10 | 取 `scale := generalSkillScale(g)`,case 改 `30*scale` / `10*scale`,`speed = 10*scale`(`ezfyAttrToBonus(g.Learning)` 不加倍) |

### 4. 绝地反击回合数(ezfy_battle.go)

- 结构体 L131-133 保留 `AtkCounter/DefCounter bool`(快照兼容),新增 `AtkCounterRounds int / DefCounterRounds int`;
- `ezfyNewBattleState` L166-174:签名 `atkCounter, defCounter bool` → `atkCounterRounds, defCounterRounds int`;赋值 `AtkCounter: atkCounterRounds>0, DefCounter: defCounterRounds>0, AtkCounterRounds: atkCounterRounds, DefCounterRounds: defCounterRounds`;
- 触发 L636-669:改外层判定 `if dist <= rangeD && target.alive() {` 内:
  ```go
  counterRounds := st.AtkCounterRounds
  if isAtk { counterRounds = st.DefCounterRounds }
  if st.Round <= counterRounds && counterRounds > 0 { /* 原反击反体,删掉 counter bool 分支 */ }
  ```
  (`st.Round` 每回合 Step 开头 ++,`<=N` 精确覆盖前 N 回合);
- 快照:snapshot 结构体 L825-826 旁加 `AtkCounterRounds int json:"atk_counter_rounds"`/`DefCounterRounds int json:"def_counter_rounds"`;Snapshot() L874 旁补;FromSnapshot L884 前做回退 `if snap.AtkCounterRounds==0 && snap.AtkCounter { atkRounds=1 }`(**绝不动 bool 字段**);
- ctor 调用点:
  - ezfy_order.go L2797:`officerCounterRounds(leadOfficer), officerCounterRounds(cityGuard)`;
  - activity_target.go L366-367:`officerCounterRounds(leadOfficer), generalCounterRounds(defGeneral)`;
  - ezfySimulate L735 / ezfySimulateBreak L761:`false, false` → `0, 0`。

### 5. 野地/寇城守将绝地反击补接(ezfy_order.go)

现状 `g` 在 L2507 `if cfg.OfficerId>0 {}` 块内局部,野地守将绝地反击从未生效。
- L2463 附近加 `var defGeneral *model.EzfyCfgGeneral`;
- L2508 块内改为先取再判:`defGeneral = ezfyCfg.general(cfg.OfficerId)` → `if defGeneral != nil { ...原逻辑... }`;
- L2797 守方参数改 `generalCounterRounds(defGeneral)`。
(野地守将速度/射程加成未接,不在本次范围。)

### 6. 前端

- 后端 detail(ezfy_officer.go L2541-2549):循环前算 `lv := h.officerSkillScale(o)`,skillViews 加 `"level": lv`,`"effect"` 用 `ezfySkillEffectTextAt(s, lv)` 替换;
- EzfyAcademy.vue 技能 tab(L620-641):已学技能名后显示 `<span class="green">Lv.{{ s.level }}</span>`,effect 后可选 `效果×N`;可学技能表头/行加「学成=Lv.{{ officer.level }}」提示(可选);
- 战报 `officerBattleDesc`(L1571-1575):循环内取 `lv := h.officerSkillLevel(o)`,parts 改 `s+"(Lv."+strconv.Itoa(lv)+" "+ezfySkillEffectTextAt(s, lv)+")"`。

### 7. 测试(新建 ezfy_skill_level_test.go,仿现有静态断言风格)

1. `TestEzfySkillLevelOf`:边界 {0,1},{49,1},{50,2},{99,2},{100,3},{255,3};nil 兜底;
2. `TestSkillBattleBonusScaled`:ezfy_officer.go 全文含 `30 * scale`、`15 * scale`、`h.officerSkillScale(o)`;
3. `TestGuardBonusScaled`:officerGuardBonus 函数体含 `30 * scale`、`10 * scale`;
4. `TestSpeedSkillBonusScaled`:officerSpeedSkillBonus 函数体含 `10 * h.officerSkillScale(o)`;
5. `TestLootHealScaled`:ezfy_order.go 全文 `10 * h.officerSkillScale(leadOfficer)` 出现 ≥3 次;activity_target.go ≥1 次;
6. `TestActWildDefScaled`:ezfyActWildDefBonus 含 `generalSkillScale(g)`、`10 * scale`;
7. `TestCounterTriggerRounds`:ezfy_battle.go 全文含 `st.Round <= counterRounds`,不含 `st.Round == 1`;
8. `TestCounterSnapshotRounds`:含 `json:"atk_counter_rounds"` 与 `snap.DefCounter`(回退逻辑)。

## 验证

```bash
cd server && go build ./... && go vet ./internal/handler/ezfy/... && go test ./internal/handler/ezfy/ -count=1
cd ../web && npm run build   # 前端无语法错误
```

后端全绿 + 前端 build 通过后,重打包 `pack-linxu.sh` 出 Linuxbushu.tar.gz + 新 md5 交付(部署由用户执行)。

## 风险与注意事项

1. **lootRate 上限 50**(ezfy_order.go L3064):黄金眼 Lv.3 时掠夺率可能被截断,是否放开上限待用户确认;默认**保留上限**。
2. 快照兼容:老指挥室战场(存 `atk_counter:true`)回退 1 回合,行为与旧版一致;勿把 bool 字段改 int。
3. `officerSpeedSkill` bool 保留(作哨兵),业务点统一走 `officerSpeedSkillBonus`。
4. 机械改造油耗/回收文案未实现,本次不新接玩法。