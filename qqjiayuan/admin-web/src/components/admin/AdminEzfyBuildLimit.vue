<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <span>建筑数量上限 / 全局参数配置</span>
        <el-button size="mini" type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>
      <div class="toolbar">
        <span class="td-sub">军事区 = 总建筑类型 2/3/4；资源区 = 类型 1。两者<b>分开计数</b>，默认各 33。</span>
      </div>
      <el-form label-width="150px" size="small" style="max-width:620px">
        <el-form-item label="军事区数量上限">
          <el-input-number v-model.number="form.military_max" :min="1" :max="999" controls-position="right" style="width:180px" />
          <span class="td-sub">默认 33</span>
        </el-form-item>
        <el-form-item label="资源区数量上限">
          <el-input-number v-model.number="form.resource_max" :min="1" :max="999" controls-position="right" style="width:180px" />
          <span class="td-sub">默认 33</span>
        </el-form-item>
        <el-form-item label="民居数量上限">
          <el-input-number v-model.number="form.house_max" :min="1" :max="999" controls-position="right" style="width:180px" />
          <span class="td-sub">默认 10</span>
        </el-form-item>
        <el-form-item label="军工厂数量上限">
          <el-input-number v-model.number="form.factory_max" :min="0" :max="999" controls-position="right" style="width:180px" />
          <span class="td-sub"><b>0 = 不限数量</b>（默认，只要军事区上限没到就能一直建）</span>
        </el-form-item>
        <el-form-item label="首页公告展示条数">
          <el-input-number v-model.number="form.notice_home_count" :min="0" :max="10" controls-position="right" style="width:180px" />
          <span class="td-sub"><b>默认 1 条</b>；填 0 表示首页不展示公告（公告页仍可看全部）</span>
        </el-form-item>
        <el-form-item label="出征集结令单次上限">
          <el-input-number v-model.number="form.gather_max_per_order" :min="1" controls-position="right" style="width:180px" />
          <span class="td-sub"><b>默认 50 个</b>；玩家出征时单次最多使用的集结令个数（每个 +10 万出征上限）。<b>不设上限，填多少就多少</b></span>
        </el-form-item>
        <el-form-item label="商城单次购买上限">
          <el-input-number v-model.number="form.mall_buy_max" :min="1" :max="999999" controls-position="right" style="width:180px" />
          <span class="td-sub"><b>默认 9999 个</b>（原来是写死的 1~99）；玩家在【商城】买道具时单次最多买几个，
            下限恒为 1。<b>仍受道具库存限制</b>，所以实际能买多少是「本值」和「库存」里较小的那个</span>
        </el-form-item>

        <el-divider content-position="left">战斗 / 经济数值</el-divider>

        <el-form-item label="征服单次扣民心">
          <el-input-number v-model.number="form.conquer_feelings_max" :min="1" controls-position="right" style="width:180px" />
          <span class="td-sub"><b>默认 2</b>；征服(3) 单次最多打掉目标多少民心。民心降到 0 才能占领，
            所以这个值越小，攻城越难。<b>调大会显著降低征服难度</b></span>
        </el-form-item>
        <el-form-item label="掠夺单次扣民心">
          <el-input-number v-model.number="form.loot_feelings" :min="1" controls-position="right" style="width:180px" />
          <span class="td-sub"><b>默认 2</b>；掠夺(2) 每次固定扣目标多少民心（民怨同步 +同样数值）</span>
        </el-form-item>
        <el-form-item label="军官工资系数">
          <el-input-number v-model.number="form.officer_salary_per_level" :min="1" controls-position="right" style="width:180px" />
          <span class="td-sub">黄金 / 每级 / 每小时。<b>默认 2</b>：一名 20 级军官每小时消耗 40 黄金。
            随资源懒结算一起扣，离线期间照样扣（俘虏不发工资）</span>
        </el-form-item>
        <el-form-item label="伤兵恢复系数">
          <el-input-number v-model.number="form.wound_heal_divisor" :min="1" controls-position="right" style="width:180px" />
          <span class="td-sub"><b>默认 100</b>；恢复 1 个伤兵消耗「该兵种总造价 ÷ 该值」黄金（最低 1）。
            <b>值越大越便宜</b>。例：系数 100 时步兵 3 金/个、航母 625 金/个</span>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="el-icon-check" :loading="saving" @click="save">保存并立即生效</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" class="box">
      <div slot="header" class="card-head"><span>建筑等级上限（固定规则，无需配置）</span></div>
      <el-table :data="levelRules" stripe border size="small">
        <el-table-column prop="name" label="建筑" min-width="180" />
        <el-table-column prop="max" label="最高等级" width="140" align="center" />
        <el-table-column prop="note" label="说明" min-width="360" show-overflow-tooltip />
      </el-table>
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
        conquer_feelings_max: 2, loot_feelings: 2,
        officer_salary_per_level: 2, wound_heal_divisor: 100
      },
      levelRules: [
        { name: '市政厅', max: '10 级', note: '市政厅最高 10 级' },
        { name: '民居', max: '12 级', note: '最多比市政厅高 1 级；市政厅满级(10)时民居可升到 12 级' },
        { name: '参谋部 / 司令部', max: '12 级', note: '参谋部、司令部上限 12 级' },
        { name: '其他建筑', max: '10 级', note: '农田/炼钢厂/石油基地/稀矿厂/围墙/科研中心/军校/交易所/仓库/军工厂/联络中心…' }
      ]
    }
  },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/admin/ezfy-build-limit').then(r => {
        if (r.code === 0) {
          // 0 / null 一律回落默认值（这些值 0 都无意义）
          const pos = (v, def) => (v === undefined || v === null || v <= 0) ? def : v
          this.form = {
            military_max: r.data.military_max,
            resource_max: r.data.resource_max,
            house_max: r.data.house_max,
            factory_max: r.data.factory_max,
            notice_home_count: r.data.notice_home_count === undefined || r.data.notice_home_count === null
              ? 1 : r.data.notice_home_count,
            gather_max_per_order: pos(r.data.gather_max_per_order, 50),
            mall_buy_max: pos(r.data.mall_buy_max, 9999),
            conquer_feelings_max: pos(r.data.conquer_feelings_max, 2),
            loot_feelings: pos(r.data.loot_feelings, 2),
            officer_salary_per_level: pos(r.data.officer_salary_per_level, 2),
            wound_heal_divisor: pos(r.data.wound_heal_divisor, 100)
          }
        } else this.$message.error(r.msg)
      })
    },
    save () {
      this.saving = true
      api.put('/admin/ezfy-build-limit', this.form).then(r => {
        this.saving = false
        // ★ resp.OK 现在会把 data.msg 提到顶层，优先读顶层
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
