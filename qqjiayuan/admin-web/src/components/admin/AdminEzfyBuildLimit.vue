<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <span>二战系统配置</span>
        <el-button size="mini" type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>
      <el-form label-width="180px" size="small" style="max-width:620px">
        <!-- ★ 说明文字一律走「标题后 i 图标 + 悬停提示」，不再以正文形式占版面 -->
        <el-divider content-position="left">建筑数量上限</el-divider>
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

        <el-divider content-position="left">其他上限</el-divider>
        <el-form-item>
          <template slot="label">首页公告展示条数<el-tooltip placement="top" :content="tips.notice_home_count"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
          <el-input-number v-model.number="form.notice_home_count" :min="0" :max="10" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item>
          <template slot="label">出征集结令单次上限<el-tooltip placement="top" :content="tips.gather_max_per_order"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
          <el-input-number v-model.number="form.gather_max_per_order" :min="1" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item>
          <template slot="label">商城单次购买上限<el-tooltip placement="top" :content="tips.mall_buy_max"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
          <el-input-number v-model.number="form.mall_buy_max" :min="1" :max="999999" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item>
          <template slot="label">单城兵力上限<el-tooltip placement="top" :content="tips.troop_max"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
          <el-input-number v-model.number="form.troop_max" :min="1" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item>
          <template slot="label">伤兵存活天数<el-tooltip placement="top" :content="tips.wound_expire_days"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
          <el-input-number v-model.number="form.wound_expire_days" :min="1" :max="3650" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item>
          <template slot="label">采集结算周期(小时)<el-tooltip placement="top" :content="tips.dispatch_period_h"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
          <el-input-number v-model.number="form.dispatch_period_h" :min="1" :max="720" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item>
          <template slot="label">出征速度加成(%)<el-tooltip placement="top" :content="tips.march_speed_bonus"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
          <el-input-number v-model.number="form.march_speed_bonus" :min="0" :max="1000" :step="10" :precision="1" controls-position="right" style="width:180px" />
        </el-form-item>

        <el-divider content-position="left">战斗 / 经济数值</el-divider>
        <el-form-item>
          <template slot="label">征服单次扣民心<el-tooltip placement="top" :content="tips.conquer_feelings_max"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
          <el-input-number v-model.number="form.conquer_feelings_max" :min="1" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item>
          <template slot="label">掠夺单次扣民心<el-tooltip placement="top" :content="tips.loot_feelings"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
          <el-input-number v-model.number="form.loot_feelings" :min="1" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item>
          <template slot="label">军官工资系数<el-tooltip placement="top" :content="tips.officer_salary_per_level"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
          <el-input-number v-model.number="form.officer_salary_per_level" :min="1" controls-position="right" style="width:180px" />
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
          <template slot="label">野地兵力倍数<el-tooltip placement="top" :content="tips.wild_troop_mult"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
          <el-input-number v-model.number="form.wild_troop_mult" :min="0.01" :step="0.1" :precision="2" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item>
          <template slot="label">训练加速黄金倍率(%)<el-tooltip placement="top" :content="tips.speed_train_rate"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
          <el-input-number v-model.number="form.speed_train_rate" :min="0.01" :max="100" :step="1" :precision="2" controls-position="right" style="width:180px" />
        </el-form-item>

        <el-divider content-position="left">玩法开关</el-divider>
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
        <el-form-item>
          <template slot="label">军队耗粮<el-tooltip placement="top" :content="tips.food_upkeep_on"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
          <el-switch v-model="form.food_upkeep_on" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关" />
        </el-form-item>
        <el-form-item>
          <template slot="label">出征油耗<el-tooltip placement="top" :content="tips.march_oil_on"><i class="el-icon-info cfg-tip" /></el-tooltip></template>
          <el-switch v-model="form.march_oil_on" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关" />
        </el-form-item>

        <el-divider content-position="left">军官升星</el-divider>
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
          <el-button type="primary" icon="el-icon-check" :loading="saving" @click="save">保存并立即生效</el-button>
        </el-form-item>
      </el-form>
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
      form: {
        military_max: 33, resource_max: 33, house_max: 10, factory_max: 0,
        notice_home_count: 1, gather_max_per_order: 50, mall_buy_max: 9999,
        troop_max: 1000000000, wound_expire_days: 5, dispatch_period_h: 4,
        march_speed_bonus: 0,
        conquer_feelings_max: 2, loot_feelings: 2,
        officer_salary_per_level: 2, wound_heal_divisor: 100,
        wild_troop_mult: 1, speed_train_rate: 100, wound_heal_rate: 100,
        recruit_cost_on: 1, food_upkeep_on: 1, march_oil_on: 1, war_require_on: 1, march_cap_on: 1,
        officer_star_up_on: 1,
        officer_star_chance: 20,
        officer_star_attr_gain: 10, officer_star_max: 5
      },
      // ★ 各配置项的悬停说明（鼠标移到标题后的 i 图标上显示）
      //   文案口径以后端 model/ezfy.go 的 EzfyCfgLimit 注释为准，改逻辑时同步改这里
      tips: {
        military_max: '军事区建筑（司令部、参谋部、军工厂、围墙等）最多可建数量，默认 33',
        resource_max: '资源区建筑（农田、炼钢厂、石油基地、稀矿厂）最多可建数量，默认 33',
        house_max: '民居最多可建数量，默认 10',
        factory_max: '军工厂最多可建数量，填 0 = 不限',
        notice_home_count: '游戏首页公告栏展示的公告条数，填 0 = 首页不展示，默认 1',
        gather_max_per_order: '单次出征最多可使用几个集结令，默认 50',
        mall_buy_max: '商城一次最多可购买的数量（下限恒为 1），默认 9999',
        troop_max: '单座城市的兵力上限（含训练队列中尚未出厂的新兵），超出将拒绝训练或恢复，默认 10 亿',
        wound_expire_days: '伤兵在营超过该天数自动消失；按最后一次入营时间计算，期间有新伤兵入营会顺延，默认 5 天',
        dispatch_period_h: '驻守采集每满该小时数结算一期（宝物+资源），默认 4 小时',
        march_speed_bonus: '出征行军速度加成（%）：100 = 行军时间减半。节假日调高让玩家队伍走快点，0 = 无加成',
        conquer_feelings_max: '征服成功时最多扣掉目标多少民心（按幸存兵力动态计算，不超过此值），默认 2',
        loot_feelings: '掠夺成功时固定扣掉目标多少民心，默认 2',
        officer_salary_per_level: '每名军官每小时消耗「等级 × 该值」黄金，随资源结算一并扣除，默认 2',
        wound_heal_divisor: '恢复 1 个伤兵消耗「该兵种总造价 ÷ 该值」黄金（最低 1 黄金），默认 100',
        wound_heal_rate: '在上一条算出的恢复费用上再打折：100 = 原价、50 = 半价，默认 100',
        wild_troop_mult: '野地 / 海野 / 寇城守军兵力 = 配置值 × 该倍数（可填小数，2 = 翻倍），默认 1',
        speed_train_rate: '训练一键加速费用 = 剩余秒数 × 10 × 倍率 ÷ 100，100 = 原价、50 = 半价，默认 100',
        war_require_on: '开 = 必须向对方宣战才能掠夺 / 征服其城市；关 = 无需宣战即可直接进攻',
        march_cap_on: '开 = 出征兵力受司令部等级上限限制；关 = 不限兵力，随便带多少',
        recruit_cost_on: '开 = 征兵消耗资源并占用空闲人口；关 = 不消耗资源、也不占人口',
        food_upkeep_on: '开 = 城内军队每小时扣除粮食；关 = 不扣',
        march_oil_on: '开 = 出征消耗石油；关 = 不消耗',
        officer_star_up_on: '开 = 可用星级徽章给军官升星；关 = 关闭升星功能',
        officer_star_chance: '每次升星的成功概率，失败同样消耗 1 枚星级徽章，默认 20',
        officer_star_attr_gain: '每升 1 星，军官三维属性各 +N，默认 10',
        officer_star_max: '军官最高可升到的星级，默认 5'
      }
    }
  },
  mounted () { this.load() },
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
            notice_home_count: r.data.notice_home_count === undefined || r.data.notice_home_count === null
              ? 1 : r.data.notice_home_count,
            gather_max_per_order: pos(r.data.gather_max_per_order, 50),
            mall_buy_max: pos(r.data.mall_buy_max, 9999),
            troop_max: pos(r.data.troop_max, 1000000000),
            wound_expire_days: pos(r.data.wound_expire_days, 5),
            dispatch_period_h: pos(r.data.dispatch_period_h, 4),
            march_speed_bonus: (r.data.march_speed_bonus === undefined || r.data.march_speed_bonus === null)
              ? 0 : Number(r.data.march_speed_bonus),
            conquer_feelings_max: pos(r.data.conquer_feelings_max, 2),
            loot_feelings: pos(r.data.loot_feelings, 2),
            officer_salary_per_level: pos(r.data.officer_salary_per_level, 2),
            wound_heal_divisor: pos(r.data.wound_heal_divisor, 100),
            wild_troop_mult: pos(Number(r.data.wild_troop_mult), 1),
            speed_train_rate: pos(Number(r.data.speed_train_rate), 100),
            wound_heal_rate: pos(Number(r.data.wound_heal_rate), 100),
            recruit_cost_on: sw(r.data.recruit_cost_on),
            food_upkeep_on: sw(r.data.food_upkeep_on),
            march_oil_on: sw(r.data.march_oil_on),
            war_require_on: sw(r.data.war_require_on),
            march_cap_on: sw(r.data.march_cap_on),
            officer_star_up_on: sw(r.data.officer_star_up_on),
            officer_star_chance: pos(r.data.officer_star_chance, 20),
            officer_star_attr_gain: pos(r.data.officer_star_attr_gain, 10),
            officer_star_max: pos(r.data.officer_star_max, 5)
          }
        } else this.$message.error(r.msg)
      })
    },
    save () {
      this.saving = true
      api.put('/admin/ezfy-build-limit', this.form).then(r => {
        this.saving = false
        if (r.code === 0) this.$message.success(r.msg || (r.data && r.data.msg) || '已保存')
        else this.$message.error(r.msg)
      })
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
</style>