<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <span class="txt-fade">控制社区广场各板块的显示/隐藏</span>
        <div class="grow" />
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="板块" min-width="220">
          <template slot-scope="{row}"><b>{{ row.name }}</b></template>
        </el-table-column>
        <el-table-column prop="key" label="标识" width="180" />
        <el-table-column label="显示" width="120" header-align="center">
          <template slot-scope="{row}">
            <el-switch :value="row.enabled === 1" active-color="#409eff" @change="v => toggle(row, v)" />
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="顺序" width="90" header-align="center" />
      </el-table>
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminPlazaSections',
  data () { return { list: [], loading: false } },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/plaza-sections').then(r => { this.loading = false; if (r.code === 0) this.list = r.data })
    },
    toggle (row, v) {
      api.put('/admin/plaza-sections/' + row.id, { enabled: v ? 1 : 0 }).then(r => {
        if (r.code === 0) this.load()
        else this.$message.error(r.msg)
      })
    }
  }
}
</script>
