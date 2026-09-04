<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="按号码/昵称搜索" prefix-icon="el-icon-search" clearable
                  style="width:220px" @keyup.enter.native="search" @clear="search" />
        <el-button type="primary" icon="el-icon-search" @click="search">搜索</el-button>
        <div class="grow" />
        <span class="txt-fade">设置用户的 家园等级/活跃天数/成就点/城市</span>
      </div>
      <el-table :data="list" v-loading="loading" stripe style="width:100%">
        <el-table-column prop="id" label="号码" width="100" header-align="center" />
        <el-table-column label="昵称" min-width="140">
          <template slot-scope="{row}"><b><font :color="row.color || '#333'">{{ row.nickname }}</font></b></template>
        </el-table-column>
        <el-table-column label="家园等级" min-width="120" header-align="center">
          <template slot-scope="{row}"><el-input-number v-model="row.level" size="small" :min="1" :max="99" controls-position="right" /></template>
        </el-table-column>
        <el-table-column label="活跃天数" min-width="130" header-align="center">
          <template slot-scope="{row}"><el-input-number v-model="row.active_days" size="small" :min="0" :max="99999" :precision="1" controls-position="right" /></template>
        </el-table-column>
        <el-table-column label="成就点" min-width="120" header-align="center">
          <template slot-scope="{row}"><el-input-number v-model="row.achieve" size="small" :min="0" :max="999999" controls-position="right" /></template>
        </el-table-column>
        <el-table-column label="城市" min-width="140">
          <template slot-scope="{row}"><el-input v-model="row.city" size="small" maxlength="30" /></template>
        </el-table-column>
        <el-table-column label="操作" width="100" header-align="center">
          <template slot-scope="{row}"><el-button size="mini" type="primary" plain @click="save(row)">保存</el-button></template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="10" :current-page="page"
                     @current-change="p => { page = p; load() }" style="margin-top:14px;text-align:right" />
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminHome',
  data () { return { list: [], total: 0, page: 1, word: '', loading: false } },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/users?page=' + this.page + (this.word ? '&word=' + encodeURIComponent(this.word) : '')).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total }
      })
    },
    search () { this.page = 1; this.load() },
    save (row) {
      api.put('/admin/users/' + row.id + '/home', { level: row.level, active_days: row.active_days, achieve: row.achieve, city: row.city }).then(r => {
        if (r.code === 0) this.$message.success('已保存') ; else this.$message.error(r.msg)
      })
    }
  }
}
</script>
