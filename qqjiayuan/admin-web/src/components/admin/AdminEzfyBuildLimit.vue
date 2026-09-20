<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <span>建筑数量上限配置（军事区 / 资源区分开）</span>
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
      form: { military_max: 33, resource_max: 33, house_max: 10, factory_max: 0, notice_home_count: 1 },
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
          this.form = {
            military_max: r.data.military_max,
            resource_max: r.data.resource_max,
            house_max: r.data.house_max,
            factory_max: r.data.factory_max,
            notice_home_count: r.data.notice_home_count === undefined || r.data.notice_home_count === null
              ? 1 : r.data.notice_home_count
          }
        } else this.$message.error(r.msg)
      })
    },
    save () {
      this.saving = true
      api.put('/admin/ezfy-build-limit', this.form).then(r => {
        this.saving = false
        if (r.code === 0) this.$message.success(r.data.msg || '已保存')
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
