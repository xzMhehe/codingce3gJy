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
        <el-table-column label="配置键" min-width="220">
          <template slot-scope="{row}">
            <!-- 已存在的键只读展示 -->
            <span v-if="row._exists" class="exist-key">{{ row.key }}</span>
            <!-- 新增行：键从下拉选（只允许已知键，已占用的键禁用） -->
            <el-select v-else v-model="row.key" size="small" placeholder="选择配置项" filterable style="width:100%">
              <el-option v-for="opt in knownKeys" :key="opt.key" :label="opt.label" :value="opt.key" :disabled="isUsed(opt.key)" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="配置值" min-width="260">
          <template slot-scope="{row}"><el-input v-model="row.value" size="small" placeholder="填写该配置项的值" /></template>
        </el-table-column>
        <el-table-column label="说明" min-width="200">
          <template slot-scope="{row}"><span class="help-line">{{ hintOf(row.key) }}</span></template>
        </el-table-column>
        <el-table-column label="操作" width="90" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="danger" plain :disabled="row._exists" @click="removeRow(row)">移除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="help-line" style="margin-top:10px">
        从下拉选择要配置的项，填好值后点「保存全部」。删除后的配置再新增，可从下拉里重新选回。
      </div>
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

// 已知配置项（键 + 显示名 + 说明）。这是「站点设置」唯一允许配置的键集合，
// 新来的人从下拉选即可，不必记键名。新增后端会用到的键时同步补在这里。
const KNOWN_KEYS = [
  { key: 'reg_coins', label: '注册新人礼包G币数', hint: '新注册玩家赠送的 G币 数量（0-100000）' },
  { key: 'reg_ip_limit', label: '同IP注册限制', hint: '同一IP最多可注册账号数（默认5，填0不限制）' },
  { key: 'site_announce', label: '广场公告语', hint: '广场顶部公告语（留空不显示）' },
  { key: 'pretty_limit', label: '靓号转换次数', hint: '每人可转号次数（默认1，范围1-100）' },
  { key: 'anti_copy', label: '防复制/防保存开关', hint: '开启（1/留空）禁止复制保存网页；填 0 关闭' }
]

export default {
  name: 'AdminSiteConfig',
  data () { return { rows: [], loading: false, knownKeys: KNOWN_KEYS } },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/site-config').then(r => {
        this.loading = false
        if (r.code === 0) {
          // 只保留已知键的现有配置（未知/遗留键不再展示，避免误导新人）
          const known = KNOWN_KEYS.map(k => k.key)
          this.rows = (r.data || [])
            .filter(s => known.indexOf(s.key) !== -1)
            .map(s => ({ key: s.key, value: s.value, _exists: true }))
        } else this.$message.error(r.msg)
      })
    },
    addRow () { this.rows.push({ key: '', value: '', _exists: false }) },
    removeRow (row) {
      const i = this.rows.indexOf(row)
      if (i >= 0) this.rows.splice(i, 1)
    },
    // 该键是否已被某行占用（新增下拉里禁选已存在的键，避免重复）
    isUsed (key) {
      return this.rows.some(r => r._exists && r.key === key)
    },
    saveAll () {
      const payload = this.rows.filter(r => r.key && r.key.trim()).map(r => ({ key: r.key.trim(), value: r.value }))
      api.put('/admin/site-config', payload).then(r => {
        if (r.code === 0) { this.$message.success('配置已保存'); this.load() } else this.$message.error(r.msg)
      })
    },
    hintOf (k) {
      const hit = KNOWN_KEYS.find(x => x.key === k)
      return hit ? hit.hint : ''
    }
  }
}
</script>