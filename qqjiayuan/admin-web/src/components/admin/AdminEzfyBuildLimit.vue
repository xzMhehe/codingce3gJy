<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <span>二战系统配置</span>
        <el-button size="mini" type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>
      <el-form label-width="180px" size="small" style="max-width:620px">
        <el-divider content-position="left">建筑数量上限</el-divider>
        <el-form-item label="军事区数量上限">
          <el-input-number v-model.number="form.military_max" :min="1" :max="999" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item label="资源区数量上限">
          <el-input-number v-model.number="form.resource_max" :min="1" :max="999" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item label="民居数量上限">
          <el-input-number v-model.number="form.house_max" :min="1" :max="999" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item label="军工厂数量上限">
          <el-input-number v-model.number="form.factory_max" :min="0" :max="999" controls-position="right" style="width:180px" />
        </el-form-item>

        <el-divider content-position="left">其他上限</el-divider>
        <el-form-item label="首页公告展示条数">
          <el-input-number v-model.number="form.notice_home_count" :min="0" :max="10" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item label="出征集结令单次上限">
          <el-input-number v-model.number="form.gather_max_per_order" :min="1" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item label="商城单次购买上限">
          <el-input-number v-model.number="form.mall_buy_max" :min="1" :max="999999" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item label="单城兵力上限">
          <el-input-number v-model.number="form.troop_max" :min="1" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item label="伤兵存活天数">
          <el-input-number v-model.number="form.wound_expire_days" :min="1" :max="3650" controls-position="right" style="width:180px" />
        </el-form-item>

        <el-divider content-position="left">战斗 / 经济数值</el-divider>
        <el-form-item label="征服单次扣民心">
          <el-input-number v-model.number="form.conquer_feelings_max" :min="1" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item label="掠夺单次扣民心">
          <el-input-number v-model.number="form.loot_feelings" :min="1" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item label="军官工资系数">
          <el-input-number v-model.number="form.officer_salary_per_level" :min="1" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item label="伤兵恢复系数">
          <el-input-number v-model.number="form.wound_heal_divisor" :min="1" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item label="伤兵恢复黄金折扣率(%)">
          <el-input-number v-model.number="form.wound_heal_rate" :min="0.01" :max="100" :step="1" :precision="2" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item label="野地兵力倍数">
          <el-input-number v-model.number="form.wild_troop_mult" :min="0.01" :step="0.1" :precision="2" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item label="训练加速黄金倍率(%)">
          <el-input-number v-model.number="form.speed_train_rate" :min="0.01" :max="100" :step="1" :precision="2" controls-position="right" style="width:180px" />
        </el-form-item>

        <el-divider content-position="left">玩法开关</el-divider>
        <el-form-item label="宣战功能">
          <el-switch v-model="form.war_require_on" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关" />
        </el-form-item>
        <el-form-item label="出征上限">
          <el-switch v-model="form.march_cap_on" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关" />
        </el-form-item>
        <el-form-item label="征兵消耗资源">
          <el-switch v-model="form.recruit_cost_on" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关" />
        </el-form-item>
        <el-form-item label="军队耗粮">
          <el-switch v-model="form.food_upkeep_on" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关" />
        </el-form-item>
        <el-form-item label="出征油耗">
          <el-switch v-model="form.march_oil_on" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关" />
        </el-form-item>

        <el-divider content-position="left">军官升星</el-divider>
        <el-form-item label="升星功能">
          <el-switch v-model="form.officer_star_up_on" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关" />
        </el-form-item>
        <el-form-item label="升星成功率(%)">
          <el-input-number v-model.number="form.officer_star_chance" :min="1" :max="100" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item label="每星三维加成">
          <el-input-number v-model.number="form.officer_star_attr_gain" :min="1" controls-position="right" style="width:180px" />
        </el-form-item>
        <el-form-item label="军官星级上限">
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
        troop_max: 1000000000, wound_expire_days: 5,
        conquer_feelings_max: 2, loot_feelings: 2,
        officer_salary_per_level: 2, wound_heal_divisor: 100,
        wild_troop_mult: 1, speed_train_rate: 100, wound_heal_rate: 100,
        recruit_cost_on: 1, food_upkeep_on: 1, march_oil_on: 1, war_require_on: 1, march_cap_on: 1,
        officer_star_up_on: 1,
        officer_star_chance: 20,
        officer_star_attr_gain: 10, officer_star_max: 5
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
</style>