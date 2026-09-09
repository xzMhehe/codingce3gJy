<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <span class="help-line">站点配置</span>
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="addRow">新增配置</el-button>
        <el-button type="success" icon="el-icon-check" @click="saveAll">保存全部</el-button>
      </div>
      <el-table :data="rows" v-loading="loading" stripe>
        <el-table-column label="配置键" min-width="200">
          <template slot-scope="{row}"><el-input v-model="row.key" size="small" :disabled="row._exists" placeholder="如 reg_coins" /></template>
        </el-table-column>
        <el-table-column label="配置值" min-width="260">
          <template slot-scope="{row}"><el-input v-model="row.value" size="small" /></template>
        </el-table-column>
        <el-table-column label="说明" min-width="180">
          <template slot-scope="{row}"><span class="help-line">{{ hintOf(row.key) }}</span></template>
        </el-table-column>
        <el-table-column label="操作" width="90" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="danger" plain :disabled="row._exists" @click="removeRow(row)">移除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="help-line" style="margin-top:10px">
        常用键：reg_coins=注册赠送G币；site_announce=广场公告语。保存后立即生效。
      </div>
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

const HINTS = {
  reg_coins: '注册新人礼包 G币 数量（0-100000）',
  site_announce: '广场顶部公告语（留空不显示）'
}

export default {
  name: 'AdminSiteConfig',
  data () { return { rows: [], loading: false } },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/site-config').then(r => {
        this.loading = false
        if (r.code === 0) {
          this.rows = (r.data || []).map(s => ({ key: s.key, value: s.value, _exists: true }))
        } else this.$message.error(r.msg)
      })
    },
    addRow () { this.rows.push({ key: '', value: '', _exists: false }) },
    removeRow (row) {
      const i = this.rows.indexOf(row)
      if (i >= 0) this.rows.splice(i, 1)
    },
    saveAll () {
      const payload = this.rows.filter(r => r.key && r.key.trim()).map(r => ({ key: r.key.trim(), value: r.value }))
      api.put('/admin/site-config', payload).then(r => {
        if (r.code === 0) { this.$message.success('配置已保存'); this.load() } else this.$message.error(r.msg)
      })
    },
    hintOf (k) { return HINTS[k] || '' }
  }
}
</script>