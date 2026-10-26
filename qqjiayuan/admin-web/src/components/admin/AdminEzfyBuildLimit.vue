<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <span>二战系统配置</span>
        <div>
          <span class="head-hint">各 tab 独立保存；「战斗公式」页改完即时生效</span>
          <el-button size="mini" type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
        </div>
      </div>
      <!-- ★ 2026-09-25 「二战系统配置也做成 tab，相同类别的在同一个 tab」：
           原来是一张长表单 + el-divider 分段，滚起来很长、找一项要翻半天。
            现在按**类别**分成多个 tab；★ 2026-10-18 改为**分 tab 独立保存**：
           每个 tab 底部有自己的「保存本类」按钮，只提交该分类字段（整体保存太慢）。 -->
      <!-- ★ 2026-10-26 用户反馈「活动目标掉道具冷却(秒) 和 i 的位置不美观」：
           标签宽度 180 → 210px，并配合下方 CSS 的 `white-space: nowrap`，
           让最长的标签（活动目标掉道具冷却(秒)）与 i 图标**始终同一行**，不再折行。 -->
      <el-form label-width="210px" size="small" :style="{ maxWidth: activeTab === 'formula' ? '1600px' : '680px' }">
        <el-tabs v-model="activeTab">
          <!-- ① 建筑与上限 -->
          <el-tab-pane label="建筑与上限" name="build">
            <el-form-item>
              <template slot="label">军事区数量上限<el-tooltip placement="top" :content="tips.military_max"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.military_max" :min="1" :max="999" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">资源区数量上限<el-tooltip placement="top" :content="tips.resource_max"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.resource_max" :min="1" :max="999" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">民居数量上限<el-tooltip placement="top" :content="tips.house_max"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.house_max" :min="1" :max="999" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">军工厂数量上限<el-tooltip placement="top" :content="tips.factory_max"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.factory_max" :min="0" :max="999" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-10-22 每城建造队列基础上限（默认 5）：军事区/资源区共用，
                 「征募告示」道具可在基础上再叠加临时队列（最多 10 个，24 小时）。 -->
            <el-form-item>
              <template slot="label">建造队列基础上限<el-tooltip placement="top" :content="tips.build_queue_max"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.build_queue_max" :min="1" :max="999" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">首页公告展示条数<el-tooltip placement="top" :content="tips.notice_home_count"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.notice_home_count" :min="0" :max="10" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">出征集结令单次上限<el-tooltip placement="top" :content="tips.gather_max_per_order"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.gather_max_per_order" :min="1" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-09-28 挂单出售单价上限（黄金/单位，默认 100 = 1:100 卡控） -->
            <el-form-item>
              <template slot="label">挂单出售单价上限<el-tooltip placement="top" :content="tips.sell_price_max"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.sell_price_max" :min="0" :max="100000" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-10-22 系统回收（资源直接卖系统）手续费 + 每日每种资源上限 -->
            <el-form-item>
              <template slot="label">系统回收手续费(%)<el-tooltip placement="top" :content="tips.sys_sell_fee"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.sys_sell_fee" :min="1" :max="90" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">系统回收每日上限<el-tooltip placement="top" :content="tips.sys_sell_daily_limit"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.sys_sell_daily_limit" :min="0" :max="10000000000" :step="100000" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-09-28 军官军事加成：每点累加出征上限 + 每点加速（二战系统配置可调） -->
            <el-form-item>
              <template slot="label">军官军事每点上限<el-tooltip placement="top" :content="tips.officer_cap_per_military"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.officer_cap_per_military" :min="0" :max="100000" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">军官军事每点加速<el-tooltip placement="top" :content="tips.officer_speed_per_military"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.officer_speed_per_military" :min="0" :max="10" :step="0.1" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">商城单次购买上限<el-tooltip placement="top" :content="tips.mall_buy_max"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.mall_buy_max" :min="1" :max="999999" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">单件道具上限<el-tooltip placement="top" :content="tips.bag_item_max"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.bag_item_max" :min="1" :max="999999" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">宝物库存上限<el-tooltip placement="top" :content="tips.treasure_max"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.treasure_max" :min="1" :max="999999" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-09-28 军校刷新周期：按小时(默认)/按天 切换 -->
            <el-form-item>
              <template slot="label">军校刷新周期<el-tooltip placement="top" :content="tips.recruit_cycle_mode"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-select v-model="form.recruit_cycle_mode" style="width:180px">
                <el-option :value="2" label="按小时" />
                <el-option :value="1" label="按天" />
              </el-select>
            </el-form-item>
            <!-- ★ 2026-10-08 新玩家落地洲：默认欧洲，可切换；★ 2026-10-26 新增「随机洲落」(0) -->
            <el-form-item>
              <template slot="label">新玩家落地洲<el-tooltip placement="top" :content="tips.default_continent"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-select v-model="form.default_continent" style="width:180px">
                <el-option :value="0" label="随机洲落" />
                <el-option :value="1" label="欧洲" />
                <el-option :value="2" label="亚洲" />
                <el-option :value="3" label="非洲" />
                <el-option :value="4" label="北美洲" />
                <el-option :value="5" label="南美洲" />
                <el-option :value="6" label="大洋洲" />
                <el-option :value="7" label="南极洲" />
              </el-select>
            </el-form-item>
            <!-- ★ 2026-10-18 分 tab 保存：只提交本类字段，立即生效 -->
            <div class="tab-save-bar">
              <span class="save-hint">仅保存「建筑与上限」分类</span>
              <el-button type="primary" size="small" icon="el-icon-check" :loading="saving" @click="saveTab('build')">保存本类</el-button>
            </div>
          </el-tab-pane>

          <!-- ② 兵力与伤兵 -->
          <el-tab-pane label="兵力与伤兵" name="troop">
            <el-form-item>
              <template slot="label">单城兵力上限<el-tooltip placement="top" :content="tips.troop_max"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.troop_max" :min="1" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">伤兵存活天数<el-tooltip placement="top" :content="tips.wound_expire_days"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.wound_expire_days" :min="1" :max="3650" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">伤兵恢复系数<el-tooltip placement="top" :content="tips.wound_heal_divisor"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.wound_heal_divisor" :min="1" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">伤兵恢复黄金折扣率(%)<el-tooltip placement="top" :content="tips.wound_heal_rate"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.wound_heal_rate" :min="0.01" :max="100" :step="1" :precision="2" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">伤兵营钻石恢复(钻/次)<el-tooltip placement="top" :content="tips.heal_all_diamond"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.heal_all_diamond" :min="1" controls-position="right" style="width:180px" />
            </el-form-item>
            <div class="tab-save-bar">
              <span class="save-hint">仅保存「兵力与伤兵」分类</span>
              <el-button type="primary" size="small" icon="el-icon-check" :loading="saving" @click="saveTab('troop')">保存本类</el-button>
            </div>
          </el-tab-pane>

          <!-- ③ 野地与行军 -->
          <el-tab-pane label="野地与行军" name="wild">
            <el-form-item>
              <template slot="label">野地兵力倍数<el-tooltip placement="top" :content="tips.wild_troop_mult"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.wild_troop_mult" :min="0.01" :step="0.1" :precision="2" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">野地获取资源倍率<el-tooltip placement="top" :content="tips.wild_res_mult"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.wild_res_mult" :min="0.01" :step="0.5" :precision="2" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★★ 2026-10-21 采集新公式 F = 部队总兵力 × 整点小时 × 野地加成% × 后勤 ÷ 2 -->
            <el-alert type="info" :closable="false" show-icon style="margin-bottom:14px"
              title="采集公式：F = 部队兵力总数 × 整点小时 × 野地加成% × 军官后勤 ÷ 除数（16:59 按 16 小时算；海野产出再 × 海野采集倍率，默认 3）。其中「部队兵力总数」= 出征部队所有兵种数量之和，不同兵种差异只体现在负重" />
            <el-form-item>
              <template slot="label">每级野地加成(%)<el-tooltip placement="top" :content="tips.gather_wild_bonus_pct"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.gather_wild_bonus_pct" :min="0.01" :step="0.5" :precision="2" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">采集公式除数<el-tooltip placement="top" :content="tips.gather_formula_divisor"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.gather_formula_divisor" :min="0.01" :step="0.5" :precision="2" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-10-21 海野采集倍率：海野（海底森林/岛屿/渔场/矿礁）产出 = 公式结果 × 倍率，默认 3 -->
            <el-form-item>
              <template slot="label">海野采集倍率<el-tooltip placement="top" :content="tips.gather_sea_bonus_mult"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.gather_sea_bonus_mult" :min="0.01" :step="0.5" :precision="2" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-09-26 城市资源产量倍率：默认 1，**0 合法**（产量归零）→ :min="0" -->
            <!-- ★ 2026-10-05 拆成「资源产量加成倍率」+「黄金产量加成倍率」两个独立配置 -->
            <el-form-item>
              <template slot="label">资源产量加成倍率<el-tooltip placement="top" :content="tips.res_prod_mult"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.res_prod_mult" :min="0" :step="0.5" :precision="2" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">黄金产量加成倍率<el-tooltip placement="top" :content="tips.gold_prod_mult"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.gold_prod_mult" :min="0" :step="0.5" :precision="2" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-09-28 市长产量加成倍率：默认 1，**0 合法**（关闭市长加成）→ :min="0" -->
            <el-form-item>
              <template slot="label">市长加成倍率<el-tooltip placement="top" :content="tips.mayor_gain_mult"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.mayor_gain_mult" :min="0" :step="0.5" :precision="2" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">采集结算周期(小时)<el-tooltip placement="top" :content="tips.dispatch_period_h"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.dispatch_period_h" :min="1" :max="720" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">出征速度加成(%)<el-tooltip placement="top" :content="tips.march_speed_bonus"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.march_speed_bonus" :min="0" :max="1000" :step="10" :precision="1" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-10-11 打野地掉宝冷却（小时）：默认 2，**0 合法 = 关闭冷却** → :min="0" -->
            <el-form-item>
              <template slot="label">打野地掉宝冷却(小时)<el-tooltip placement="top" :content="tips.wild_treasure_cooldown_h"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.wild_treasure_cooldown_h" :min="0" :max="720" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-10-26 打野地掉商城道具冷却（秒）：默认 1800，**0 合法 = 关闭冷却** → :min="0" -->
            <el-form-item>
              <template slot="label">打野地掉道具冷却(秒)<el-tooltip placement="top" :content="tips.wild_item_cooldown_sec"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.wild_item_cooldown_sec" :min="0" :max="2592000" :step="60" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-10-24 活动目标掉落冷却（秒）：默认 7200 = 2 小时，**0 合法 = 关闭冷却** → :min="0" -->
            <el-form-item>
              <template slot="label">活动目标掉宝冷却(秒)<el-tooltip placement="top" :content="tips.act_treasure_cooldown_sec"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.act_treasure_cooldown_sec" :min="0" :max="86400" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">活动目标掉道具冷却(秒)<el-tooltip placement="top" :content="tips.act_item_cooldown_sec"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.act_item_cooldown_sec" :min="0" :max="86400" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-10-12 迁城冷却（小时）：默认 24，**0 合法 = 关闭迁城冷却** → :min="0" -->
            <el-form-item>
              <template slot="label">迁城冷却(小时)<el-tooltip placement="top" :content="tips.move_cool_h"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.move_cool_h" :min="0" :max="720" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-10-09 原「战斗掉落·中级/高级/特殊宝物概率(%) + 活动野地掉宝概率(%)」
                 四个配置已删除：掉落**全部走手动配置** ——
                 野地/寇城 →「地图管理 → 野地类型 → 掉落宝物/掉落道具」，
                 活动野地 →「活动野地配置 → 必掉宝物」。 -->
            <div class="tab-save-bar">
              <span class="save-hint">仅保存「野地与行军」分类</span>
              <el-button type="primary" size="small" icon="el-icon-check" :loading="saving" @click="saveTab('wild')">保存本类</el-button>
            </div>
          </el-tab-pane>

          <!-- ④ 战斗与经济 -->
          <el-tab-pane label="战斗与经济" name="battle">
            <!-- ★ 2026-10-13 回合战斗公式（可调，默认 = 移植前的硬编码值：40 / 6000 / 10000 / 50 / 50） -->
            <el-divider content-position="left">回合战斗公式</el-divider>
            <!-- ★ 2026-10-23 战斗公式按套维护：选了公式套时，下方 11 项内置数值不生效 -->
            <el-form-item>
              <template slot="label">战斗公式来源<el-tooltip placement="top" :content="formulaSourceTip"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-select v-model="activeFormulaId" :loading="formulaSaving" style="width:280px" @change="applyFormula">
                <el-option label="内置配置（下方数值）" :value="0" />
                <el-option v-for="f in formulas" :key="f.id" :label="f.name" :value="f.id" />
              </el-select>
            </el-form-item>
            <el-form-item>
              <template slot="label">战斗最大回合数<el-tooltip placement="top" :content="tips.battle_max_rounds"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.battle_max_rounds" :min="1" :max="200" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">战场初始距离<el-tooltip placement="top" :content="tips.battle_start_dist"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.battle_start_dist" :min="1000" :max="50000" :step="100" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">战斗最大后退距离<el-tooltip placement="top" :content="tips.battle_retreat_max"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.battle_retreat_max" :min="1000" :max="100000" :step="100" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-10-24 家园版公式不使用下面 6 项：选中的公式来源为家园版时隐藏，避免管理员误配 -->
            <template v-if="!activeFormulaIsHome">
            <el-form-item>
              <template slot="label">暴击基础伤害加成(%)<el-tooltip placement="top" :content="tips.crit_base_bonus_pct"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.crit_base_bonus_pct" :min="1" :max="500" controls-position="right" style="width:180px" />
            </el-form-item>
            </template>
            <el-form-item>
              <template slot="label">驻军溃败撤退阈值(%)<el-tooltip placement="top" :content="tips.garrison_break_pct"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.garrison_break_pct" :min="1" :max="100" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-10-13 伤害公式系数 + 每回合时长（默认 = 移植前的硬编码值） -->
            <el-form-item>
              <template slot="label">每回合时长(毫秒)<el-tooltip placement="top" :content="tips.battle_round_ms"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.battle_round_ms" :min="25000" :max="120000" :step="1000" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-10-24 以下 5 项仅原版公式使用，家园版隐藏 -->
            <template v-if="!activeFormulaIsHome">
            <el-form-item>
              <template slot="label">伤害攻击系数<el-tooltip placement="top" :content="tips.atk_coeff"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.atk_coeff" :min="1" :max="100" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">伤害攻击加值<el-tooltip placement="top" :content="tips.atk_base"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.atk_base" :min="1" :max="100000" :step="100" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">伤害防御系数<el-tooltip placement="top" :content="tips.def_coeff"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.def_coeff" :min="1" :max="100" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">伤害防御加值<el-tooltip placement="top" :content="tips.def_base"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.def_base" :min="1" :max="100000" :step="10" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">最小伤害系数<el-tooltip placement="top" :content="tips.min_dmg_coeff"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.min_dmg_coeff" :min="1" :max="1000" controls-position="right" style="width:180px" />
            </el-form-item>
            </template>
            <el-form-item>
              <template slot="label">征服单次扣民心<el-tooltip placement="top" :content="tips.conquer_feelings_max"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.conquer_feelings_max" :min="1" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">掠夺单次扣民心<el-tooltip placement="top" :content="tips.loot_feelings"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.loot_feelings" :min="1" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-10-07 赎城金额（被占城市原主人赎回押金，同意后归占领方） -->
            <el-form-item>
              <template slot="label">赎城金额(钻石)<el-tooltip placement="top" :content="tips.ransom_cost"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.ransom_cost" :min="1" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-09-28 用户规则：安抚固定花费 + 冷却，四个值都可配 -->
            <el-divider content-position="left">安抚（民心/民怨）</el-divider>
            <el-form-item>
              <template slot="label">安抚花费黄金<el-tooltip placement="top" :content="tips.placate_gold"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.placate_gold" :min="1" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">安抚降低民怨<el-tooltip placement="top" :content="tips.placate_grievance"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.placate_grievance" :min="1" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">安抚提升民心<el-tooltip placement="top" :content="tips.placate_feelings"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.placate_feelings" :min="1" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">安抚冷却(分钟)<el-tooltip placement="top" :content="tips.placate_cooldown_min"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.placate_cooldown_min" :min="1" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-10-24 民心平滑回归（秒级）：上升/下降独立配置，默认均 3600 秒；恢复上限受「民心+民怨+税率=100」约束 -->
            <el-form-item>
              <template slot="label">民心回归上升(秒)<el-tooltip placement="top" :content="tips.feelings_recover_seconds"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.feelings_recover_seconds" :min="1" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">民心回归下降(秒)<el-tooltip placement="top" :content="tips.feelings_drop_seconds"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.feelings_drop_seconds" :min="1" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">军官工资系数<el-tooltip placement="top" :content="tips.officer_salary_per_level"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.officer_salary_per_level" :min="1" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-10-17 训练一键加速开关：开（默认）= 训练页展示并使用两个一键加速按钮；关 = 前端隐藏 + 后端拒绝 -->
            <el-form-item>
              <template slot="label">训练一键加速<el-tooltip placement="top" :content="tips.speed_train_on"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-switch v-model="form.speed_train_on" :active-value="1" :inactive-value="0" active-text="开启" inactive-text="关闭" />
            </el-form-item>
            <el-form-item>
              <template slot="label">训练加速黄金倍率(%)<el-tooltip placement="top" :content="tips.speed_train_rate"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.speed_train_rate" :min="0.01" :max="100" :step="1" :precision="2" controls-position="right" style="width:180px" />
            </el-form-item>
            <!-- ★ 2026-09-27：资源最大值（每项资源唯一硬上限），产量/获取统一累加到该值，默认 21 亿 -->
            <el-divider content-position="left">资源最大值（每项资源唯一硬上限，默认 21 亿）</el-divider>
            <el-form-item>
              <template slot="label">粮食最大值<el-tooltip placement="top" :content="tips.res_max_food"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.res_max_food" :min="1" :max="1000000000000" :step="1000000000" :precision="0" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">钢铁最大值<el-tooltip placement="top" :content="tips.res_max_steel"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.res_max_steel" :min="1" :max="1000000000000" :step="1000000000" :precision="0" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">石油最大值<el-tooltip placement="top" :content="tips.res_max_oil"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.res_max_oil" :min="1" :max="1000000000000" :step="1000000000" :precision="0" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">稀有矿最大值<el-tooltip placement="top" :content="tips.res_max_rare"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.res_max_rare" :min="1" :max="1000000000000" :step="1000000000" :precision="0" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">黄金最大值<el-tooltip placement="top" :content="tips.res_max_gold"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.res_max_gold" :min="1" :max="1000000000000" :step="1000000000" :precision="0" controls-position="right" style="width:180px" />
            </el-form-item>
            <div class="tab-save-bar">
              <span class="save-hint">仅保存「战斗与经济」分类</span>
              <el-button type="primary" size="small" icon="el-icon-check" :loading="saving" @click="saveTab('battle')">保存本类</el-button>
            </div>
          </el-tab-pane>

          <!-- ⑤ 玩法开关 -->
          <el-tab-pane label="玩法开关" name="switch">
            <!-- ★ 2026-10-22 驻军 [一键采集]/[一键收获]/[一键召回] 的操作范围：
                 默认只处理当前城市（与军情→驻军列表一致），可切换为账号下全部城市。 -->
            <el-form-item>
              <template slot="label">驻军批量操作范围<el-tooltip placement="top" :content="tips.garrison_batch_scope"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-select v-model="form.garrison_batch_scope" style="width:180px">
                <el-option label="当前城市（默认）" :value="1" />
                <el-option label="全部城市" :value="2" />
              </el-select>
            </el-form-item>
            <el-form-item>
              <template slot="label">宣战功能<el-tooltip placement="top" :content="tips.war_require_on"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-switch v-model="form.war_require_on" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关" />
            </el-form-item>
            <el-form-item>
              <template slot="label">出征上限<el-tooltip placement="top" :content="tips.march_cap_on"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-switch v-model="form.march_cap_on" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关" />
            </el-form-item>
            <el-form-item>
              <template slot="label">征兵消耗资源<el-tooltip placement="top" :content="tips.recruit_cost_on"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-switch v-model="form.recruit_cost_on" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关" />
            </el-form-item>
            <!-- ★ 2026-10-21 造兵时间公式：floor(面板 × 250/(250+市长军事) × 10/(10+军训艺术等级))，默认吃市长加成 -->
            <el-form-item>
              <template slot="label">造兵市长军事加成<el-tooltip placement="top" :content="tips.train_mayor_military_on"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-switch v-model="form.train_mayor_military_on" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关" />
            </el-form-item>
            <el-form-item>
              <template slot="label">装备/套装军事计入<el-tooltip placement="top" :content="tips.train_mayor_equip_on"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-switch v-model="form.train_mayor_equip_on" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关" :disabled="form.train_mayor_military_on !== 1" />
            </el-form-item>
            <el-form-item>
              <template slot="label">军队耗粮<el-tooltip placement="top" :content="tips.food_upkeep_on"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-switch v-model="form.food_upkeep_on" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关" />
            </el-form-item>
            <el-form-item>
              <template slot="label">出征油耗<el-tooltip placement="top" :content="tips.march_oil_on"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-switch v-model="form.march_oil_on" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关" />
            </el-form-item>
            <!-- ★ 2026-10-22 人口上限只由民居容量（pop_max）决定：原「民居容量限制」
                 「允许长期超过民居上限」「召集人口上限」三项配置已删除。
                 ★ 2026-10-23 超限人口改为下一次懒结算直接夹平，原「超限人口回落速度」配置也已删除。 -->
            <!-- ★ 2026-09-26 「花费 10万粮食 召集 10万人口也要能配置，现在是写死的」 -->
            <el-form-item>
              <template slot="label">召集消耗粮食<el-tooltip placement="top" :content="tips.convene_food_cost"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model="form.convene_food_cost" :min="1" :max="1000000000" :step="10000" controls-position="right" style="width: 200px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">召集人口比例<el-tooltip placement="top" :content="tips.convene_pop_pct"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.convene_pop_pct" :min="1" :max="1000000000" :step="1" controls-position="right" style="width: 200px" />
            </el-form-item>
            <!-- ★ 2026-10-23 召集间隔可配：方便调整节奏；测试时可临时改小 -->
            <el-form-item>
              <template slot="label">普通召集间隔<el-tooltip placement="top" :content="tips.convene_cooldown_seconds"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.convene_cooldown_seconds" :min="1" :max="1000000000" :step="1" controls-position="right" style="width: 200px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">人口满时召集间隔<el-tooltip placement="top" :content="tips.convene_cap_cooldown_seconds"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.convene_cap_cooldown_seconds" :min="1" :max="1000000000" :step="10" controls-position="right" style="width: 200px" />
            </el-form-item>
            <!-- ★ 2026-10-22 「空闲人口恢复满时间」：训练消耗人口后按配置秒数线性恢复满 -->
            <el-form-item>
              <template slot="label">空闲人口恢复满时间<el-tooltip placement="top" :content="tips.pop_recover_seconds"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.pop_recover_seconds" :min="1" :max="1000000000" :step="10" controls-position="right" style="width: 200px" />
            </el-form-item>
            <!-- ★ 2026-10-23 临时空闲人口回落窗口：人口满后召集/移民法案产生的临时空闲，按此时间线性回落 -->
            <el-form-item>
              <template slot="label">临时空闲回落时间<el-tooltip placement="top" :content="tips.pop_extra_decay_seconds"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.pop_extra_decay_seconds" :min="1" :max="1000000000" :step="10" controls-position="right" style="width: 200px" />
            </el-form-item>
            <!-- ★ 2026-10-26 采集宝物播报开关：默认关 = 采集掉宝不播世界频道、也不写播报库（ezfy_chat） -->
            <el-form-item>
              <template slot="label">采集宝物播报<el-tooltip placement="top" :content="tips.loot_broadcast_on"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-switch v-model="form.loot_broadcast_on" :active-value="1" :inactive-value="0" active-text="播报" inactive-text="不播报" />
            </el-form-item>
            <!-- ★ 2026-10-14 风云榜统计开关（与「排行榜 → 风云榜」页的开关同一项，两处都能改） -->
            <el-form-item>
              <template slot="label">风云榜统计<el-tooltip placement="top" :content="tips.pvp_rank_on"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-switch v-model="form.pvp_rank_on" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关" />
            </el-form-item>
            <div class="tab-save-bar">
              <span class="save-hint">仅保存「玩法开关」分类</span>
              <el-button type="primary" size="small" icon="el-icon-check" :loading="saving" @click="saveTab('switch')">保存本类</el-button>
            </div>
          </el-tab-pane>

          <!-- ⑥ 军官升星 -->
          <el-tab-pane label="军官升星" name="star">
            <el-form-item>
              <template slot="label">升星功能<el-tooltip placement="top" :content="tips.officer_star_up_on"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-switch v-model="form.officer_star_up_on" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关" />
            </el-form-item>
            <el-form-item>
              <template slot="label">升星成功率(%)<el-tooltip placement="top" :content="tips.officer_star_chance"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.officer_star_chance" :min="1" :max="100" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">每星三维加成<el-tooltip placement="top" :content="tips.officer_star_attr_gain"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.officer_star_attr_gain" :min="1" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">军官星级上限<el-tooltip placement="top" :content="tips.officer_star_max"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.officer_star_max" :min="1" :max="100" controls-position="right" style="width:180px" />
            </el-form-item>
            <el-form-item>
              <template slot="label">招生简章出五星军官概率<el-tooltip placement="top" :content="tips.recruit_five_star_rate"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-input-number v-model.number="form.recruit_five_star_rate" :min="1" :max="100" controls-position="right" style="width:180px" />
            </el-form-item>
            <div class="tab-save-bar">
              <span class="save-hint">仅保存「军官升星」分类</span>
              <el-button type="primary" size="small" icon="el-icon-check" :loading="saving" @click="saveTab('star')">保存本类</el-button>
            </div>
          </el-tab-pane>

          <!-- ⑦ 战斗公式（★ 2026-10-23 按套维护 + 当前服选择） -->
          <el-tab-pane label="战斗公式" name="formula">
            <el-form-item style="margin-top:16px">
              <template slot="label">本服生效公式<el-tooltip placement="top" :content="formulaTabTip"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
              <el-select v-model="activeFormulaId" :loading="formulaSaving" style="width:300px" @change="applyFormula">
                <el-option label="内置配置（战斗与经济页里的数值）" :value="0" />
                <el-option v-for="f in formulas" :key="f.id" :label="f.name" :value="f.id" />
              </el-select>
              <el-button size="small" icon="el-icon-refresh" style="margin-left:8px" @click="loadFormulas">刷新</el-button>
            </el-form-item>
            <div class="formula-toolbar">
              <el-button type="primary" size="small" icon="el-icon-plus" @click="openFormulaDialog()">新建公式套</el-button>
              <el-tooltip placement="top" :content="formulaToolbarTip"><i class="el-icon-info cfg-tip" /></el-tooltip>
            </div>
            <el-table :data="formulas" size="small" border style="margin-top:10px; width:100%">
              <el-table-column prop="id" label="ID" width="60" align="center" />
              <el-table-column label="名称" min-width="150">
                <template slot-scope="scope">
                  <span>{{ scope.row.name }}</span>
                  <el-tag v-if="scope.row.id === activeFormulaId" size="mini" type="success" style="margin-left:6px">生效中</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="remark" label="说明" min-width="180" show-overflow-tooltip />
              <el-table-column label="伤害算法" width="100" align="center">
                <template slot-scope="scope">
                  <el-tag size="mini" :type="Number(scope.row.damage_formula) === 1 ? 'warning' : ''">
                    {{ Number(scope.row.damage_formula) === 1 ? '家园版' : '原版' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="加成明细" width="90" align="center">
                <template slot-scope="scope">
                  <el-tag size="mini" :type="Number(scope.row.show_battle_bonus_detail) === 0 ? 'info' : 'success'">
                    {{ Number(scope.row.show_battle_bonus_detail) === 0 ? '关闭' : '展示' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column v-for="f in formulaFields" :key="f.key" :label="f.short" width="110" align="center">
                <template slot-scope="scope">{{ formulaCell(scope.row, f) }}</template>
              </el-table-column>
              <el-table-column label="操作" width="210" align="center" fixed="right">
                <template slot-scope="scope">
                  <el-button type="text" size="mini" :disabled="scope.row.id === activeFormulaId" @click="applyFormula(scope.row.id)">使用</el-button>
                  <el-button type="text" size="mini" @click="openFormulaDialog(scope.row, true)">复制</el-button>
                  <el-button type="text" size="mini" @click="openFormulaDialog(scope.row)">编辑</el-button>
                  <el-button type="text" size="mini" style="color:#f56c6c" :disabled="scope.row.id === activeFormulaId" @click="deleteFormula(scope.row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </el-form>

      <!-- ★ 2026-10-23 战斗公式套：新建/复制/编辑弹窗 -->
      <el-dialog :title="formulaEditId ? '编辑战斗公式套' : '新建战斗公式套'" :visible.sync="formulaDialog" width="760px" append-to-body>
        <el-form label-width="170px" size="small">
          <el-form-item label="公式套名称" required>
            <el-input v-model="formulaForm.name" maxlength="50" show-word-limit placeholder="如：原版 / 快节奏 / 测试服" style="width:320px" />
          </el-form-item>
          <el-form-item label="说明">
            <el-input v-model="formulaForm.remark" maxlength="500" show-word-limit placeholder="选填，方便区分各套公式" style="width:520px" />
          </el-form-item>
          <el-form-item>
            <template slot="label">伤害算法<el-tooltip placement="top" :content="tips.damage_formula"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
            <el-radio-group v-model.number="formulaForm.damage_formula">
              <el-radio :label="0">原版（攻防系数）</el-radio>
              <el-radio :label="1">家园版（图片公式）</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item>
            <template slot="label">加成明细展示<el-tooltip placement="top" :content="tips.show_battle_bonus_detail"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
            <el-radio-group v-model.number="formulaForm.show_battle_bonus_detail">
              <el-radio :label="1">展示（默认）</el-radio>
              <el-radio :label="0">关闭</el-radio>
            </el-radio-group>
          </el-form-item>
          <!-- ★ 2026-10-24 公式预览：新建/编辑时直观展示当前公式（原版代入实时数值），便于管理员理解各参数作用 -->
          <el-alert :title="formulaPreview.title" type="info" :closable="false" show-icon style="margin:2px 0 10px">
            <div class="fx-preview">
              <div class="fx-line">{{ formulaPreview.formula }}</div>
              <div class="fx-line">{{ formulaPreview.min }}</div>
              <div v-if="formulaPreview.note" class="fx-note">{{ formulaPreview.note }}</div>
            </div>
          </el-alert>
          <el-form-item v-for="f in visibleFormulaFields" :key="f.key">
            <template slot="label">{{ f.label }}<el-tooltip placement="top" :content="tips[f.tip]"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
            <el-input-number v-model.number="formulaForm[f.key]" :min="f.min" :max="f.max" :step="f.step" controls-position="right" style="width:180px" />
          </el-form-item>
        </el-form>
        <span slot="footer">
          <el-button @click="formulaDialog = false">取消</el-button>
          <el-button type="primary" :loading="formulaSaving" @click="saveFormula">保存</el-button>
        </span>
      </el-dialog>
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminEzfyBuildLimit',
  data () {
    return {
      saving: false,
      // ★ 2026-09-25：配置按类别分了 tab，记住当前在哪个 tab（刷新页面回到第一个）
      activeTab: 'build',
      form: {
        military_max: 33, resource_max: 33, house_max: 10, factory_max: 0,
        // ★ 2026-10-22 每城建造队列基础上限（默认 5）
        build_queue_max: 5,
        notice_home_count: 1, gather_max_per_order: 50, sell_price_max: 100, mall_buy_max: 999,
        // ★ 2026-10-22 系统回收手续费（默认 30%）+ 每日每种资源上限（★ 2026-10-26 默认 0 = 不限）
        sys_sell_fee: 30, sys_sell_daily_limit: 0,
        bag_item_max: 5000,
        treasure_max: 5000,
        officer_cap_per_military: 2000, officer_speed_per_military: 0.1,
        troop_max: 1000000000, wound_expire_days: 5, dispatch_period_h: 4,
        march_speed_bonus: 0,
        conquer_feelings_max: 2, loot_feelings: 2, ransom_cost: 500,
        // ★ 2026-10-13 回合战斗公式（默认 = 移植前的硬编码值：40 / 6000 / 10000 / 50 / 50）
        battle_max_rounds: 40, battle_start_dist: 6000, battle_retreat_max: 10000,
        crit_base_bonus_pct: 50, garrison_break_pct: 50,
        // ★ 2026-10-13 伤害公式系数 + 每回合时长（默认 = 移植前的硬编码值：30000 / 5 / 1000 / 3 / 10 / 5）
        battle_round_ms: 30000, atk_coeff: 5, atk_base: 1000, def_coeff: 3, def_base: 10, min_dmg_coeff: 5,
        // ★ 2026-09-28 安抚参数（黄金 / 民怨- / 民心+ / 冷却分钟），默认 5万 / 2 / 1 / 15
        placate_gold: 50000, placate_grievance: 2, placate_feelings: 1, placate_cooldown_min: 15,
        // ★ 2026-10-24 民心平滑回归（秒级）：上升/下降默认均 3600 秒
        feelings_recover_seconds: 3600, feelings_drop_seconds: 3600,
        officer_salary_per_level: 20, wound_heal_divisor: 100,
        // ★ 野地兵力倍数 / 野地获取资源倍率（都允许小数，默认 1 = 原样）
        wild_troop_mult: 1, wild_res_mult: 1,
        // ★ 2026-09-26 城市资源产量倍率（默认 1；**0 合法 = 产量归零**）
        res_prod_mult: 1,
        // ★ 2026-10-05 黄金产量加成倍率（与资源倍率拆开，默认 1；0 合法 = 黄金产量归零）
        gold_prod_mult: 1,
        // ★ 2026-09-28 市长加成倍率（默认 1；0 合法 = 关闭）
        mayor_gain_mult: 1,
        // ★ 2026-10-21 采集新公式：每级野地 3% / 除数 2 / 海野倍率 3
        //   ⚠️ 2026-10-21 口径修正：M = 部队所有兵种数量之和（不再限定卡车兵种），
        //   gather_truck_troop_id 字段已废弃，不再展示/管理。
        gather_wild_bonus_pct: 3, gather_formula_divisor: 2,
        gather_sea_bonus_mult: 3,
        // ★ 2026-10-11 打野地掉宝冷却小时数（默认 2；**0 合法 = 关闭冷却**）
        wild_treasure_cooldown_h: 2,
        // ★ 2026-10-26 打野地掉商城道具冷却**秒数**（默认 1800 = 30 分钟；**0 合法 = 关闭冷却**）
        wild_item_cooldown_sec: 1800,
        // ★ 2026-10-24 活动目标掉落冷却秒数（默认 7200 = 2 小时；**0 合法 = 关闭冷却**）
        act_treasure_cooldown_sec: 7200, act_item_cooldown_sec: 7200,
        // ★ 2026-10-12 迁城冷却小时数（默认 24；**0 合法 = 关闭迁城冷却**）
        move_cool_h: 24,
        // ★ 2026-09-28 军校刷新周期（1=按天 2=按小时，默认按小时）
        recruit_cycle_mode: 2,
        // ★ 2026-10-08 新玩家落地洲（默认欧洲；★ 2026-10-26 新增 0 = 随机洲落）
        default_continent: 1,
        // ★ 2026-10-22 驻军批量操作范围（1=当前城市 默认，2=全部城市）
        garrison_batch_scope: 1,
        speed_train_rate: 100, wound_heal_rate: 100, heal_all_diamond: 100,
        speed_train_on: 1,
        // ★ 2026-10-21 造兵时间：市长军事加成 / 装备套装军事计入（默认都开）
        train_mayor_military_on: 1, train_mayor_equip_on: 1,
        // ★ 2026-09-27：资源最大值（每项资源唯一硬上限），默认 21 亿 = 2100000000
        res_max_food: 2100000000, res_max_steel: 2100000000, res_max_oil: 2100000000,
        res_max_rare: 2100000000, res_max_gold: 2100000000,
        recruit_cost_on: 1, food_upkeep_on: 1, march_oil_on: 1, war_require_on: 1, march_cap_on: 1,
        // ★ 2026-09-26：召集消耗粮食 / 召集人口比例（默认 10 万粮食 / 10%；原来写死固定人口）
        //   ★ 2026-10-22 人口上限只由民居容量决定，原三项人口上限配置已删除。
        convene_food_cost: 100000, convene_pop_pct: 10,
        // ★ 2026-10-23 召集间隔（秒）：普通默认 3 秒 / 人口满默认 300 秒
        convene_cooldown_seconds: 3, convene_cap_cooldown_seconds: 300,
        // ★ 2026-10-22：空闲人口恢复满时间（默认 1200 秒；训练消耗人口后按此时间线性恢复）
        pop_recover_seconds: 1200,
        // ★ 2026-10-23：临时空闲人口回落时间（默认 300 秒 = 5 分钟）
        pop_extra_decay_seconds: 300,
        // ★ 2026-10-14 风云榜统计开关（1 = 开：战斗结算写周榜；0 = 关：完全不统计）
        pvp_rank_on: 1,
        // ★ 2026-10-26 采集宝物播报开关（1 = 播报；0 = 不播报也不入播报库，**默认 0 关**）
        loot_broadcast_on: 0,
        officer_star_up_on: 1,
        officer_star_chance: 20,
        officer_star_attr_gain: 10, officer_star_max: 5,
        // ★ 2026-09-30 招生简章出五星军官概率（默认 1 = 1%，100 = 必出）
        recruit_five_star_rate: 1,
      },
      // ★ 2026-10-23 战斗公式套：列表 / 当前生效 ID / 弹窗状态
      formulas: [],
      activeFormulaId: 0,
      formulaSaving: false,
      formulaDialog: false,
      formulaEditId: 0,
      formulaForm: {
        name: '', remark: '', damage_formula: 0, show_battle_bonus_detail: 1,
        battle_max_rounds: 40, battle_start_dist: 6000, battle_retreat_max: 10000,
        crit_base_bonus_pct: 50, garrison_break_pct: 50, battle_round_ms: 30000,
        atk_coeff: 5, atk_base: 1000, def_coeff: 3, def_base: 10, min_dmg_coeff: 5,
      },
      // 公式字段定义：表格列 + 弹窗输入共用（tips 复用下面同名的说明文案）
      formulaFields: [
        { key: 'battle_max_rounds', label: '战斗最大回合数', short: '最大回合', min: 1, max: 200, step: 1, tip: 'battle_max_rounds' },
        { key: 'battle_start_dist', label: '战场初始距离', short: '初始距离', min: 1000, max: 50000, step: 100, tip: 'battle_start_dist' },
        { key: 'battle_retreat_max', label: '战斗最大后退距离', short: '最大后退', min: 1000, max: 100000, step: 100, tip: 'battle_retreat_max' },
        { key: 'crit_base_bonus_pct', label: '暴击基础伤害加成(%)', short: '暴击加成%', min: 1, max: 500, step: 1, tip: 'crit_base_bonus_pct' },
        { key: 'garrison_break_pct', label: '驻军溃败撤退阈值(%)', short: '驻军溃败%', min: 1, max: 100, step: 1, tip: 'garrison_break_pct' },
        { key: 'battle_round_ms', label: '每回合时长(毫秒)', short: '回合时长', min: 25000, max: 120000, step: 1000, tip: 'battle_round_ms' },
        { key: 'atk_coeff', label: '伤害攻击系数', short: '攻击系数', min: 1, max: 100, step: 1, tip: 'atk_coeff' },
        { key: 'atk_base', label: '伤害攻击加值', short: '攻击加值', min: 1, max: 100000, step: 100, tip: 'atk_base' },
        { key: 'def_coeff', label: '伤害防御系数', short: '防御系数', min: 1, max: 100, step: 1, tip: 'def_coeff' },
        { key: 'def_base', label: '伤害防御加值', short: '防御加值', min: 1, max: 100000, step: 10, tip: 'def_base' },
        { key: 'min_dmg_coeff', label: '最小伤害系数', short: '最小伤害', min: 1, max: 1000, step: 1, tip: 'min_dmg_coeff' },
      ],
      // ★ 2026-10-18 分 tab 保存：每个 tab 所属字段清单，保存时只提交当前分类（后端按字段判空部分更新）
      fieldGroups: {
        build: ['military_max', 'resource_max', 'house_max', 'factory_max', 'build_queue_max', 'notice_home_count',
          'gather_max_per_order', 'sell_price_max', 'sys_sell_fee', 'sys_sell_daily_limit',
          'officer_cap_per_military', 'officer_speed_per_military',
          'mall_buy_max', 'bag_item_max', 'treasure_max', 'recruit_cycle_mode', 'default_continent'],
        troop: ['troop_max', 'wound_expire_days', 'wound_heal_divisor', 'wound_heal_rate', 'heal_all_diamond'],
        wild: ['wild_troop_mult', 'wild_res_mult', 'gather_wild_bonus_pct',
          'gather_formula_divisor', 'gather_sea_bonus_mult', 'res_prod_mult', 'gold_prod_mult', 'mayor_gain_mult', 'dispatch_period_h',
          'march_speed_bonus', 'wild_treasure_cooldown_h', 'wild_item_cooldown_sec', 'act_treasure_cooldown_sec', 'act_item_cooldown_sec', 'move_cool_h'],
        battle: ['battle_max_rounds', 'battle_start_dist', 'battle_retreat_max', 'crit_base_bonus_pct',
          'garrison_break_pct', 'battle_round_ms', 'atk_coeff', 'atk_base', 'def_coeff', 'def_base',
          'min_dmg_coeff', 'conquer_feelings_max', 'loot_feelings', 'ransom_cost', 'placate_gold',
          'placate_grievance', 'placate_feelings', 'placate_cooldown_min', 'feelings_recover_seconds', 'feelings_drop_seconds', 'officer_salary_per_level',
          'speed_train_on', 'speed_train_rate', 'res_max_food', 'res_max_steel', 'res_max_oil',
          'res_max_rare', 'res_max_gold'],
        switch: ['war_require_on', 'march_cap_on', 'recruit_cost_on', 'train_mayor_military_on',
          'train_mayor_equip_on', 'food_upkeep_on', 'march_oil_on',
          'convene_food_cost', 'convene_pop_pct', 'convene_cooldown_seconds',
          'convene_cap_cooldown_seconds', 'pop_recover_seconds', 'pop_extra_decay_seconds',
          'pvp_rank_on', 'garrison_batch_scope', 'loot_broadcast_on'],
        star: ['officer_star_up_on', 'officer_star_chance', 'officer_star_attr_gain', 'officer_star_max',
          'recruit_five_star_rate'],
      },
      // ★ 各配置项的悬停说明（鼠标移到标题后的 i 图标上显示）
      //   文案口径以后端 model/ezfy.go 的 EzfyCfgLimit 注释为准，改逻辑时同步改这里
      tips: {
        military_max: '军事区建筑（司令部、参谋部、军工厂、围墙等）可建造数量上限，默认 36。',
        resource_max: '资源区建筑（农田、炼钢厂、石油基地、稀矿厂）可建造数量上限，默认 36。',
        house_max: '民居可建造数量上限，默认 33。',
        factory_max: '军工厂可建造数量上限，0 = 不限。',
        build_queue_max: '每座城市同时建造或升级的数量上限（军事区、资源区共用），默认 5；「征募告示」可临时增加，最多叠加 10 个。',
        notice_home_count: '首页公告栏展示条数，0 = 不展示，默认 1。',
        gather_max_per_order: '单次出征最多可使用的集结令数量，默认 99。',
        sell_price_max: '挂单出售资源的单价上限（黄金/单位）：100 = 每单位最多卖 100 黄金，默认 100。',
        sys_sell_fee: '资源直接卖给系统时收取的手续费百分比，默认 30%。' +
          '低于 24% 时商城资源包可低买高卖套利，建议保持 30%。',
        sys_sell_daily_limit: '每个玩家每日每种资源可卖给系统的数量上限，默认 0 = 不限；' +
          '填大于 0 时按自然日重置，超出后当日不可再卖。',
        officer_cap_per_military: '出征军官每 1 点军事属性增加的出征上限，默认 2000。',
        officer_speed_per_military: '出征军官每 1 点军事属性增加的行军速度百分比，默认 0.1（1000 点军事约提升 1 倍速度）。',
        mall_buy_max: '商城单次可购买数量上限，默认 999。',
        bag_item_max: '单个道具的持有上限（同种道具合计），超出后购买与掉落不再入包，默认 5000。',
        treasure_max: '宝物总库存上限（所有采集宝物合计），超出后停止采集、掉落不入包，默认 5000。',
        troop_max: '单座城市兵力上限（含训练队列中未出厂的新兵），默认 50 亿。',
        wound_expire_days: '伤兵在营存活天数，超时自动消失；有新伤兵入营则重新计时，默认 3 天。',
        dispatch_period_h: '驻守采集每满该小时数结算一期（资源与宝物），默认 4 小时。',
        march_speed_bonus: '出征行军速度加成百分比：100 = 行军时间减半，默认 0（无加成）。',
        conquer_feelings_max: '征服成功时最多扣除目标多少民心，默认 5。',
        loot_feelings: '掠夺成功时固定扣除目标多少民心，默认 3。',
        feelings_recover_seconds: '民心低于上限时，按该秒数平滑回升至上限，默认 3600 秒。' +
          '上限受「民心 + 民怨 + 税率 = 100」限制：民怨未清除时民心无法回满。',
        feelings_drop_seconds: '民心高于上限（上调税率或民怨上升）时，按该秒数平滑下降至上限，默认 3600 秒。',
        // ★ 2026-10-23 战斗公式来源（按套维护）
        battle_formula_id: '本服生效的战斗公式：内置配置 = 使用下方各项数值；选公式套 = 整套覆盖下方数值，立即生效。' +
          '公式套在「战斗公式」页新建、编辑、复制。',
        damage_formula: '伤害算法。原版 = (攻击×攻击系数+加值) ÷ (防御×(1+防御%)×防御系数+加值) × 数量，再乘攻击加成；' +
          '家园版 = 数量×基础伤害²×(1+攻击%) ÷ [基础防御×(1+防御%)]，不使用暴击加成、攻防系数与最小伤害系数（选中后自动隐藏）。' +
          '城墙耐久仅征服战扣除。',
        show_battle_bonus_detail: '回合行动日志是否展示加成来源明细（军官、科技、装备各占多少）。' +
          '关闭后只显示加成总百分比，原始数据仍完整存档。',
        // ★ 2026-10-13 回合战斗公式（可调，默认 = 移植前的硬编码值）
        battle_max_rounds: '单场战斗最多回合数，打满仍未分胜负按平局处理，默认 40。',
        battle_start_dist: '战场初始距离：攻方从 0 出发、守方站在该距离处，默认 6000。调小接火更快、回合更少。',
        battle_retreat_max: '双方「后退」指令的最远退距，超出后夹回，默认 10000。',
        crit_base_bonus_pct: '装备仅配置暴击几率、未配置暴击伤害时使用的暴击伤害加成，默认 50（暴击伤害 ×1.5）。',
        garrison_break_pct: '盟军驻军战的溃败撤退阈值：守方剩余兵力跌破初始兵力的该百分比即判负并返航，默认 50。',
        // ★ 2026-10-13 伤害公式系数 + 每回合时长
        battle_round_ms: '指挥室实时战斗每回合时长（毫秒），默认 30000（前 25 秒可下指令、后 5 秒锁定）；最小 25000。',
        atk_coeff: '伤害公式的攻击系数，默认 5。调大则攻击方伤害整体提高。',
        atk_base: '伤害公式的攻击加值，默认 1000。攻击力很低时靠它保底打出伤害。',
        def_coeff: '伤害公式的防御系数，默认 3。调大则防御属性削减伤害更明显，战斗更持久。',
        def_base: '伤害公式的防御加值，默认 10。防止防御为 0 时公式异常。',
        min_dmg_coeff: '最小伤害系数，默认 5：单支部队最小伤害 = 兵力数量 × 该值。',
        ransom_cost: '被占城市原主人赎城需先支付的钻石押金（拒绝或撤销自动退回，同意后归占领方），默认 500。',
        // ★ 2026-09-28 安抚：民心 + 税率 = 100 为基准，安抚只是临时顶一下民心、压一压民怨
        placate_gold: '每次安抚消耗的黄金，默认 5 万。',
        placate_grievance: '每次安抚降低的民怨值，默认 2。',
        placate_feelings: '每次安抚提升的民心值，默认 1。',
        placate_cooldown_min: '同一城市两次安抚的最短间隔（分钟），默认 15。',
        officer_salary_per_level: '每名军官每小时消耗「等级 × 该值」黄金，随资源结算扣除，默认 20。',
        wound_heal_divisor: '恢复 1 个伤兵消耗「该兵种总造价 ÷ 该值」黄金（最低 1），默认 50。',
        wound_heal_rate: '伤兵恢复费用的折扣率：100 = 原价，50 = 半价，默认 100。',
        heal_all_diamond: '伤兵营「一键恢复全部伤兵」的钻石单价（每次固定收取，与数量无关），默认 100。',
        wild_troop_mult: '野地、海野、寇城守军的兵力倍率，默认 10；支持小数（0.5 = 减半）。',
        wild_res_mult: '野地、海野、寇城战斗胜利的战利品资源倍率，默认 10；不含驻守采集产出。',
        // ★★ 2026-10-21 采集新公式：F = 部队总兵力 × 整点小时 × 野地加成% × 后勤 ÷ 除数
        //   （M = 出征部队所有兵种数量之和，兵种差异只体现在负重；卡片兵种 ID 配置已废弃，不再展示）
        gather_wild_bonus_pct: '每 1 级同资源野地提供的采集加成百分比，默认 3。' +
          '例：名下共 14 级粮食野地 → 粮食野地加成 14 × 3% = 42%。',
        gather_formula_divisor: '采集公式的除数，默认 2；数值越大产量越低。',
        // ★ 2026-10-21 海野采集倍率（简约提示：玩家端不展示倍率细节）
        gather_sea_bonus_mult: '海野采集产量倍率：海野产出 = 公式结果 × 该值，默认 3；陆地野地不生效。',
        // ★ 2026-09-26 城市资源产量倍率
        // ★ 2026-10-05 拆开：只作用于粮/钢/油/稀矿（黄金走 gold_prod_mult）
        res_prod_mult: '城市每小时粮、钢、油、稀矿的产量倍率，默认 1；0 = 停产。',
        // ★ 2026-10-05 黄金产量加成倍率（与资源倍率拆开）
        gold_prod_mult: '城市每小时黄金产量倍率，默认 1；0 = 停产。',
        // ★ 2026-09-28 市长加成倍率
        mayor_gain_mult: '市长产量加成倍率，默认 1；0 = 关闭市长加成。',
        // ★ 2026-09-28 军校刷新周期
        recruit_cycle_mode: '军校免费刷新次数的重置周期：按小时（默认）每小时重置；按天每日 0 点重置。',
        // ★ 2026-10-11 打野地掉宝冷却
        wild_treasure_cooldown_h: '打野地、寇城掉落宝物的玩家冷却（小时），默认 2；' +
          '冷却期内再次打地不掉宝物，0 = 关闭冷却。',
        // ★ 2026-10-26 打野地掉道具冷却（单位由小时改为秒）
        wild_item_cooldown_sec: '打野地、寇城掉落商城道具的玩家冷却（秒），默认 1800；' +
          '冷却期内再次打地不掉道具，0 = 关闭冷却。',
        // ★ 2026-10-24 活动目标掉落冷却（秒）
        act_treasure_cooldown_sec: '活动目标（活动野地、活动寇城、特殊城市）掉落必掉宝物的玩家冷却（秒），默认 7200；0 = 关闭冷却。',
        act_item_cooldown_sec: '活动目标掉落商城道具的玩家冷却（秒），默认 7200；0 = 关闭冷却。',
        // ★ 2026-10-12 迁城冷却
        move_cool_h: '迁城后该城市再次迁城需等待的小时数，默认 24；其他城市不受影响，0 = 关闭冷却。',
        // ★ 2026-10-08 新玩家落地洲（★ 2026-10-26 新增「随机洲落」）
        default_continent: '新玩家首次建城的落地洲，默认欧洲；选「随机洲落」则每次在七大洲中随机选择。',
        // ★ 2026-10-22 驻军批量操作范围
        garrison_batch_scope: '「一键采集、一键收获、一键召回」的处理范围：当前城市（默认）只处理本城出发的驻军；' +
          '全部城市一次处理账号下所有城市的驻军。',
        speed_train_rate: '训练一键加速费用倍率：费用 = 剩余秒数 × 10 × 该值 ÷ 100，默认 0.1；100 = 原价，50 = 半价。',
        speed_train_on: '训练一键加速开关：开 = 展示并使用加速按钮；关 = 按钮隐藏且接口拒绝。',
        // ★ 2026-09-27：资源最大值 = **每项资源唯一硬上限**，产量/获取统一累加到该值为止（不再看仓储）
        res_max_food: '粮食的唯一硬上限，默认 21 亿（2100000000）；产量与一切获取方式累加至此为止，范围 1 ~ 1 万亿。',
        res_max_steel: '钢铁的唯一硬上限，默认 21 亿（2100000000）；产量与一切获取方式累加至此为止，范围 1 ~ 1 万亿。',
        res_max_oil: '石油的唯一硬上限，默认 21 亿（2100000000）；产量与一切获取方式累加至此为止，范围 1 ~ 1 万亿。',
        res_max_rare: '稀有矿的唯一硬上限，默认 21 亿（2100000000）；产量与一切获取方式累加至此为止，范围 1 ~ 1 万亿。',
        res_max_gold: '黄金的唯一硬上限，默认 21 亿（2100000000）；产量与一切获取方式累加至此为止，范围 1 ~ 1 万亿。',
        war_require_on: '开 = 掠夺、征服他城需先宣战；关 = 无需宣战。',
        march_cap_on: '开 = 出征兵力受司令部等级上限限制；关 = 不限兵力。',
        recruit_cost_on: '开 = 征兵消耗资源并占用空闲人口；关 = 均不消耗。',
        train_mayor_military_on: '开 = 造兵时间按「面板时间 × 250/(250+市长军事) × 10/(10+军训艺术等级)」向下取整计算；' +
          '关 = 市长军事按 0 计。已开始的训练队列不受影响。',
        train_mayor_equip_on: '开 = 市长军事含装备与套装加成；关 = 只计军官自身军事。仅在「造兵市长军事加成」开启时生效。',
        food_upkeep_on: '开 = 城内军队每小时扣除粮食；关 = 不扣。',
        march_oil_on: '开 = 出征消耗石油；关 = 不消耗。',
        convene_food_cost: '每次召集消耗的粮食数量，默认 10 万（100000）。',
        convene_pop_pct: '每次召集获得的人口 = 当前民居上限 × 该百分比，默认 10%；人口已满时转为临时空闲人口，可超过 100。',
        convene_cooldown_seconds: '普通状态下两次召集的最短间隔（秒），默认 3。',
        convene_cap_cooldown_seconds: '人口达到民居上限后，两次召集的最短间隔（秒），默认 300。',
        pop_recover_seconds: '训练占用的空闲人口恢复满所需时间（秒），默认 1200；有新训练时重新计时。',
        pop_extra_decay_seconds: '临时空闲人口回落到 0 所需时间（秒），默认 300；每次新召集重新计时。',
        pvp_rank_on: '开 = 玩家对战结算计入风云榜；关 = 不统计（历史周榜保留）。与「排行榜 → 风云榜」页顶部开关为同一项配置。',
        loot_broadcast_on: '开 = 采集获得宝物时在世界频道播报；关 = 不播报，也不写入播报记录（默认关）。',
        officer_star_up_on: '开 = 可用星级徽章为军官升星；关 = 关闭该功能。',
        officer_star_chance: '每次升星的成功概率（%），失败同样消耗 1 枚星级徽章，默认 20。',
        officer_star_attr_gain: '每升 1 星，军事、后勤、学识各自随机 +1 ~ 该值，默认 10。',
        officer_star_max: '军官可达到的最高星级，默认 5。',
        // ★ 2026-09-30 招生简章出五星军官概率
        recruit_five_star_rate: '使用招生简章刷新军校时出现 5 星军官的概率（%），默认 1；100 = 必出。',
      }
    }
  },
  computed: {
    activeFormulaName () {
      if (!this.activeFormulaId) return '内置配置'
      const f = this.formulas.find(x => Number(x.id) === Number(this.activeFormulaId))
      return f ? f.name : '内置配置（所选公式套已不存在）'
    },
    formulaSourceTip () {
      const extra = this.activeFormulaIsHome
        ? '当前为家园版，原版专用的 6 项数值不生效并已隐藏。'
        : '内置配置与所选公式套均使用下方全部数值。'
      return '当前生效：' + this.activeFormulaName + '。选择公式套后，下方各项数值不再生效，以所选套为准。' + extra
    },
    // ★ 2026-10-24 当前生效公式是否为家园版：是则隐藏原版专用的 6 项配置
    activeFormulaIsHome () {
      if (!this.activeFormulaId) return false
      const f = this.formulas.find(x => Number(x.id) === Number(this.activeFormulaId))
      return !!(f && Number(f.damage_formula) === 1)
    },
    formulaTabTip () {
      return '战斗公式按套维护，本服选择其中一套生效；「内置配置」使用「战斗与经济」页的各项数值。切换后立即生效。'
    },
    formulaToolbarTip () {
      return '共 ' + this.formulas.length + ' 套，当前生效：' + this.activeFormulaName + '。使用或保存后立即生效。'
    },
    visibleFormulaFields () {
      if (Number(this.formulaForm.damage_formula) !== 1) return this.formulaFields
      const skip = ['crit_base_bonus_pct', 'atk_coeff', 'atk_base', 'def_coeff', 'def_base', 'min_dmg_coeff']
      return this.formulaFields.filter(f => skip.indexOf(f.key) < 0)
    },
    // ★ 2026-10-24 公式预览：按所选伤害算法生成公式文本，原版代入实时参数值
    formulaPreview () {
      if (Number(this.formulaForm.damage_formula) === 1) {
        return {
          title: '家园版伤害公式（图片公式）',
          formula: '伤害 = 数量 × 基础伤害² × (1 + 攻加成%) ÷ [基础防御 × (1 + 防加成%)]',
          // ★ 2026-10-24 家园版无最小伤害：伤害完全按公式结算，高防时可为 0
          //   （原「最小伤害 = 数量（每名士兵至少造成 1 点伤害）」会让 1 架侦察机
          //    对 10 万航母编队也至少削 1 点，用户认为不合理，已移除）
          min: '无最小伤害（伤害按公式结算，打不动高防目标时为 0）',
          note: '本算法不使用：暴击基础伤害加成 / 攻击系数 / 攻击加值 / 防御系数 / 防御加值 / 最小伤害系数（已隐藏）'
        }
      }
      const v = this.formulaForm
      return {
        title: '原版伤害公式（攻防系数）',
        formula: '伤害 = (攻 × ' + v.atk_coeff + ' + ' + v.atk_base + ') ÷ [守防 × (1+防加成%) × ' + v.def_coeff +
          ' + ' + v.def_base + '] × 数量 × (1+攻加成%)',
        min: '最小伤害 = 数量 × ' + v.min_dmg_coeff + '（单次伤害低于该值时按它结算）',
        note: '调整上方参数即可直观看到公式变化；攻加成% / 防加成% 来自兵种与军官属性'
      }
    }
  },
  mounted () { this.load(); this.loadFormulas() },
  methods: {
    load () {
      api.get('/admin/ezfy-build-limit').then(r => {
        if (r.code === 0) {
          const pos = (v, def) => (v === undefined || v === null || v <= 0) ? def : v
          const sw = (v) => (v === undefined || v === null) ? 1 : (Number(v) === 0 ? 0 : 1)
          this.form = {
            military_max: r.data.military_max,
            resource_max: r.data.resource_max,
            house_max: r.data.house_max,
            factory_max: r.data.factory_max,
            // ★ 2026-10-22 建造队列基础上限（0 无意义 → 回落 5）
            build_queue_max: pos(r.data.build_queue_max, 5),
            notice_home_count: r.data.notice_home_count === undefined || r.data.notice_home_count === null
              ? 1 : r.data.notice_home_count,
            gather_max_per_order: pos(r.data.gather_max_per_order, 50),
            sell_price_max: pos(r.data.sell_price_max, 100),
            // ★ 2026-10-22 系统回收手续费（0/越界无意义 → 默认 30）
            sys_sell_fee: pos(r.data.sys_sell_fee, 30),
            // ★ 2026-10-22 每日每种资源上限：**0 合法 = 不限**（★ 2026-10-26 起默认就是 0），
            //   不能用 pos()（会把 0 改回 100 万）
            sys_sell_daily_limit: (r.data.sys_sell_daily_limit === undefined || r.data.sys_sell_daily_limit === null ||
              isNaN(Number(r.data.sys_sell_daily_limit)))
              ? 0 : Math.max(0, Number(r.data.sys_sell_daily_limit)),
            officer_cap_per_military: pos(r.data.officer_cap_per_military, 2000),
            officer_speed_per_military: (r.data.officer_speed_per_military === undefined || r.data.officer_speed_per_military === null)
              ? 0.1 : r.data.officer_speed_per_military,
            mall_buy_max: pos(r.data.mall_buy_max, 999),
            bag_item_max: pos(r.data.bag_item_max, 5000),
            treasure_max: pos(r.data.treasure_max, 5000),
            troop_max: pos(r.data.troop_max, 1000000000),
            wound_expire_days: pos(r.data.wound_expire_days, 5),
            heal_all_diamond: pos(r.data.heal_all_diamond, 100),
            dispatch_period_h: pos(r.data.dispatch_period_h, 4),
            march_speed_bonus: (r.data.march_speed_bonus === undefined || r.data.march_speed_bonus === null)
              ? 0 : Number(r.data.march_speed_bonus),
            conquer_feelings_max: pos(r.data.conquer_feelings_max, 2),
            loot_feelings: pos(r.data.loot_feelings, 2),
            ransom_cost: pos(r.data.ransom_cost, 500),
            // ★ 2026-10-13 回合战斗公式：0 无意义 → pos 回落各自默认
            battle_max_rounds: pos(r.data.battle_max_rounds, 40),
            battle_start_dist: pos(r.data.battle_start_dist, 6000),
            battle_retreat_max: pos(r.data.battle_retreat_max, 10000),
            crit_base_bonus_pct: pos(r.data.crit_base_bonus_pct, 50),
            garrison_break_pct: pos(r.data.garrison_break_pct, 50),
            // ★ 2026-10-13 伤害公式系数 + 每回合时长：0 无意义 → pos 回落各自默认
            battle_round_ms: pos(r.data.battle_round_ms, 30000),
            atk_coeff: pos(r.data.atk_coeff, 5),
            atk_base: pos(r.data.atk_base, 1000),
            def_coeff: pos(r.data.def_coeff, 3),
            def_base: pos(r.data.def_base, 10),
            min_dmg_coeff: pos(r.data.min_dmg_coeff, 5),
            // ★ 2026-09-28 安抚参数：0 无意义 → pos 各自回落默认
            placate_gold: pos(Number(r.data.placate_gold) || 0, 50000),
            placate_grievance: pos(r.data.placate_grievance, 2),
            placate_feelings: pos(r.data.placate_feelings, 1),
            // ★ 2026-10-24 民心平滑回归（秒级）：0 无意义 → pos 各自回落默认 3600
            feelings_recover_seconds: pos(r.data.feelings_recover_seconds, 3600),
            feelings_drop_seconds: pos(r.data.feelings_drop_seconds, 3600),
            placate_cooldown_min: pos(r.data.placate_cooldown_min, 15),
            officer_salary_per_level: pos(r.data.officer_salary_per_level, 2),
            wound_heal_divisor: pos(r.data.wound_heal_divisor, 100),
            wild_troop_mult: pos(Number(r.data.wild_troop_mult), 1),
            wild_res_mult: pos(Number(r.data.wild_res_mult), 1),
            // ★ 2026-10-21 采集新公式参数：0 无意义 → pos 回落默认
            gather_wild_bonus_pct: pos(Number(r.data.gather_wild_bonus_pct), 3),
            gather_formula_divisor: pos(Number(r.data.gather_formula_divisor), 2),
            // ★ 2026-10-21 海野采集倍率：0 无意义 → pos 回落默认 3
            gather_sea_bonus_mult: pos(Number(r.data.gather_sea_bonus_mult), 3),
            // ★ 2026-10-11 打野地掉宝冷却：**0 合法 = 关闭冷却**，不能用 pos()（会把 0 改回 2）
            wild_treasure_cooldown_h: (r.data.wild_treasure_cooldown_h === undefined || r.data.wild_treasure_cooldown_h === null)
              ? 2 : Number(r.data.wild_treasure_cooldown_h),
            // ★ 2026-10-26 打野地掉道具冷却（秒）：**0 合法 = 关闭冷却**，不能用 pos()（会把 0 改回 1800）
            wild_item_cooldown_sec: (r.data.wild_item_cooldown_sec === undefined || r.data.wild_item_cooldown_sec === null)
              ? 1800 : Number(r.data.wild_item_cooldown_sec),
            // ★ 2026-10-24 活动目标掉落冷却：**0 合法 = 关闭冷却**，不能用 pos()（会把 0 改回 7200）
            act_treasure_cooldown_sec: (r.data.act_treasure_cooldown_sec === undefined || r.data.act_treasure_cooldown_sec === null)
              ? 7200 : Number(r.data.act_treasure_cooldown_sec),
            act_item_cooldown_sec: (r.data.act_item_cooldown_sec === undefined || r.data.act_item_cooldown_sec === null)
              ? 7200 : Number(r.data.act_item_cooldown_sec),
            // ★ 2026-10-12 迁城冷却：**0 合法 = 关闭迁城冷却**，同样不能用 pos()（会把 0 改回 24）
            move_cool_h: (r.data.move_cool_h === undefined || r.data.move_cool_h === null)
              ? 24 : Number(r.data.move_cool_h),
            // ★ 2026-09-28 军校刷新周期：1=按天 2=按小时，非法回落按小时
            recruit_cycle_mode: (r.data.recruit_cycle_mode === 1) ? 1 : 2,
            // ★ 2026-10-08 新玩家落地洲：0~7（★ 0 = 随机洲落），非法回落欧洲
            default_continent: (Number(r.data.default_continent) >= 0 && Number(r.data.default_continent) <= 7)
              ? Number(r.data.default_continent) : 1,
            // ★ 2026-10-22 驻军批量操作范围：只有明确 2 才是全部城市，老库 0 / 非法值都回落当前城市
            garrison_batch_scope: Number(r.data.garrison_batch_scope) === 2 ? 2 : 1,
            // ★ 2026-09-28 市长加成倍率：**0 合法 = 关闭**, 不能用 pos()（会把 0 改回 1）
            mayor_gain_mult: (r.data.mayor_gain_mult === undefined || r.data.mayor_gain_mult === null)
              ? 1 : Number(r.data.mayor_gain_mult),
            // ★ 2026-09-26 产量倍率：**不能用 pos()**（它把 <=0 回落成默认值，会把玩家设的 0 改回 1）
            res_prod_mult: (r.data.res_prod_mult === undefined || r.data.res_prod_mult === null)
              ? 1 : Number(r.data.res_prod_mult),
            // ★ 2026-10-05 黄金产量加成倍率：同样 0 合法，不能用 pos()
            gold_prod_mult: (r.data.gold_prod_mult === undefined || r.data.gold_prod_mult === null)
              ? 1 : Number(r.data.gold_prod_mult),
            speed_train_rate: pos(Number(r.data.speed_train_rate), 100),
            speed_train_on: r.data.speed_train_on === undefined || r.data.speed_train_on === null
              ? 1 : Number(r.data.speed_train_on),
            wound_heal_rate: pos(Number(r.data.wound_heal_rate), 100),
            // ★ 2026-09-27：资源最大值；非数字 / 0 / 负数一律回落 21 亿
            res_max_food: pos(Number(r.data.res_max_food) || 0, 2100000000),
            res_max_steel: pos(Number(r.data.res_max_steel) || 0, 2100000000),
            res_max_oil: pos(Number(r.data.res_max_oil) || 0, 2100000000),
            res_max_rare: pos(Number(r.data.res_max_rare) || 0, 2100000000),
            res_max_gold: pos(Number(r.data.res_max_gold) || 0, 2100000000),
            recruit_cost_on: sw(r.data.recruit_cost_on),
            // ★ 2026-10-21 造兵时间：0 合法 = 关，不能用 pos()（会把 0 改回 1）
            train_mayor_military_on: sw(r.data.train_mayor_military_on),
            train_mayor_equip_on: sw(r.data.train_mayor_equip_on),
            food_upkeep_on: sw(r.data.food_upkeep_on),
            march_oil_on: sw(r.data.march_oil_on),
            war_require_on: sw(r.data.war_require_on),
            march_cap_on: sw(r.data.march_cap_on),
            convene_food_cost: pos(r.data.convene_food_cost, 100000),
            convene_pop_pct: Math.min(1000000000, Math.max(1, Number(r.data.convene_pop_pct) || 10)),
            // ★ 2026-10-23 召集间隔（0/未配置 → 回落默认 3 秒 / 300 秒）
            convene_cooldown_seconds: Math.min(1000000000, Math.max(1, Number(r.data.convene_cooldown_seconds) || 3)),
            convene_cap_cooldown_seconds: Math.min(1000000000, Math.max(1, Number(r.data.convene_cap_cooldown_seconds) || 300)),
            // ★ 2026-10-22 空闲人口恢复满时间（0/未配置 → 回落默认 1200 秒）
            pop_recover_seconds: Math.min(1000000000, Math.max(1, Number(r.data.pop_recover_seconds) || 1200)),
            // ★ 2026-10-23 临时空闲人口回落时间（0/未配置 → 回落默认 300 秒）
            pop_extra_decay_seconds: Math.min(1000000000, Math.max(1, Number(r.data.pop_extra_decay_seconds) || 300)),
            // ★ 2026-10-14 风云榜统计开关：**0 合法 = 关闭统计**，不能用 pos()（会把 0 改回 1）
            pvp_rank_on: (r.data.pvp_rank_on === undefined || r.data.pvp_rank_on === null)
              ? 1 : Number(r.data.pvp_rank_on),
            // ★ 2026-10-26 采集宝物播报开关：**0 合法 = 关**（默认关），不能用 pos()（会把 0 改回 1）
            loot_broadcast_on: (r.data.loot_broadcast_on === undefined || r.data.loot_broadcast_on === null)
              ? 0 : Number(r.data.loot_broadcast_on),
            officer_star_up_on: sw(r.data.officer_star_up_on),
            officer_star_chance: pos(r.data.officer_star_chance, 20),
            officer_star_attr_gain: pos(r.data.officer_star_attr_gain, 10),
            officer_star_max: pos(r.data.officer_star_max, 5),
            // ★ 2026-09-30 招生简章出五星军官概率（默认 1 = 1%）
            recruit_five_star_rate: pos(r.data.recruit_five_star_rate, 1),
          }
        } else this.$message.error(r.msg)
      })
    },
    saveTab (tab) {
      // ★ 2026-10-18 分 tab 保存：只提交当前 tab 所属字段，后端逐字段判空 → 天然支持部分更新
      const payload = {}
      ;(this.fieldGroups[tab] || []).forEach(k => { payload[k] = this.form[k] })
      this.saving = true
      api.put('/admin/ezfy-build-limit', payload).then(r => {
        this.saving = false
        if (r.code === 0) this.$message.success(r.msg || (r.data && r.data.msg) || '已保存')
        else this.$message.error(r.msg)
      })
    },

    // ===== ★ 2026-10-23 战斗公式套 =====
    emptyFormula () {
      return {
        name: '', remark: '', damage_formula: 0, show_battle_bonus_detail: 1,
        battle_max_rounds: 40, battle_start_dist: 6000, battle_retreat_max: 10000,
        crit_base_bonus_pct: 50, garrison_break_pct: 50, battle_round_ms: 30000,
        atk_coeff: 5, atk_base: 1000, def_coeff: 3, def_base: 10, min_dmg_coeff: 5
      }
    },
    formulaCell (row, f) {
      if (Number(row.damage_formula) === 1 &&
        ['crit_base_bonus_pct', 'atk_coeff', 'atk_base', 'def_coeff', 'def_base', 'min_dmg_coeff'].indexOf(f.key) >= 0) {
        return '—'
      }
      return row[f.key]
    },
    loadFormulas () {
      api.get('/admin/ezfy-battle-formulas').then(r => {
        if (r.code === 0) {
          this.formulas = r.data.list || []
          this.activeFormulaId = Number(r.data.active_id) || 0
        } else this.$message.error(r.msg)
      })
    },
    // 选择公式套（0 = 内置配置）：立即保存到本服并生效
    applyFormula (id) {
      const n = Number(id) || 0
      this.formulaSaving = true
      api.post('/admin/ezfy-battle-formulas/' + n + '/apply').then(r => {
        this.formulaSaving = false
        if (r.code === 0) {
          this.activeFormulaId = n
          this.$message.success(r.msg || (r.data && r.data.msg) || '已切换')
        } else {
          this.$message.error(r.msg)
          this.loadFormulas()
        }
      }).catch(() => { this.formulaSaving = false; this.loadFormulas() })
    },
    openFormulaDialog (row, copy) {
      const f = this.emptyFormula()
      if (row) {
        this.formulaFields.forEach(x => {
          if (row[x.key] !== undefined && row[x.key] !== null) f[x.key] = Number(row[x.key])
        })
        f.damage_formula = Number(row.damage_formula) === 1 ? 1 : 0
        f.show_battle_bonus_detail = Number(row.show_battle_bonus_detail) === 0 ? 0 : 1
        f.name = copy ? (row.name + ' 副本').slice(0, 50) : row.name
        f.remark = row.remark || ''
      }
      this.formulaEditId = (row && !copy) ? row.id : 0
      this.formulaForm = f
      this.formulaDialog = true
    },
    saveFormula () {
      const name = String(this.formulaForm.name || '').trim()
      if (!name) { this.$message.warning('请填写公式套名称'); return }
      const payload = Object.assign({}, this.formulaForm, { name })
      const req = this.formulaEditId
        ? api.put('/admin/ezfy-battle-formulas/' + this.formulaEditId, payload)
        : api.post('/admin/ezfy-battle-formulas', payload)
      this.formulaSaving = true
      req.then(r => {
        this.formulaSaving = false
        if (r.code === 0) {
          this.formulaDialog = false
          this.$message.success(r.msg || (r.data && r.data.msg) || '已保存')
          this.loadFormulas()
        } else this.$message.error(r.msg)
      }).catch(() => { this.formulaSaving = false })
    },
    deleteFormula (row) {
      this.$confirm('删除公式套「' + row.name + '」？删除后不可恢复。', '提示', { type: 'warning' })
        .then(() => api.delete('/admin/ezfy-battle-formulas/' + row.id))
        .then(r => {
          if (r && r.code === 0) {
            this.$message.success(r.msg || '已删除')
            this.loadFormulas()
          } else if (r) this.$message.error(r.msg)
        })
        .catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.card-head { display: flex; justify-content: space-between; align-items: center; }
/* 配置标题后的说明图标：悬停显示提示，不占版面 */
.cfg-tip { margin-left: 3px; padding: 1px 3px; color: #909399; font-size: 13px; cursor: help; vertical-align: -1px; }
.cfg-tip:hover { color: #409eff; }
/* ★ 2026-10-26 用户反馈「活动目标掉道具冷却(秒) 和 i 的位置不美观」：
   标签文字强制单行，i 图标永远跟在标题后面同一行（配合 label-width 210px，不再折到第二行）。 */
.farm-admin ::v-deep .el-form-item__label { white-space: nowrap; }
.unit-hint { margin-left: 8px; font-size: 12px; color: #909399; }
/* ★ 2026-09-25 改成 tab 后新增的两个小样式 */
.head-hint { font-size: 12px; color: #909399; margin-right: 10px; }
/* ★ 2026-10-18 分 tab 保存条：放在每个 tab 底部，只保存当前分类 */
.tab-save-bar {
  display: flex; align-items: center; justify-content: space-between;
  max-width: 680px; margin-top: 4px; padding-top: 8px;
  border-top: 1px dashed #dcdfe6;
}
.save-hint { font-size: 12px; color: #909399; }
/* ★ 2026-10-23 战斗公式套：工具栏 + 空态 */
.formula-toolbar { display: flex; align-items: center; margin-top: 4px; }
.formula-toolbar .save-hint { margin-left: 12px; }
.formula-empty {
  margin-top: 12px; padding: 18px; text-align: center; font-size: 13px; color: #909399;
  background: #fafafa; border: 1px dashed #dcdfe6; border-radius: 4px;
}
/* ★ 2026-10-24 战斗公式弹窗里的公式预览 */
.fx-preview { font-size: 13px; line-height: 1.9; }
.fx-line { font-weight: 600; color: #303133; }
.fx-note { font-weight: 400; color: #909399; }
/* ★ 2026-10-24 7 个 tab 默认间距放不下（表单 maxWidth 660px），压缩间距让「战斗公式」tab 直接可见，无需点溢出箭头 */
.farm-admin ::v-deep .el-tabs__item {
  padding: 0 12px;
  font-size: 13px;
}
</style>