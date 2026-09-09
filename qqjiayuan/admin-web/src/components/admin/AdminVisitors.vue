<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model.number="userId" placeholder="按家园主人号码过滤" clearable style="width:200px" @keyup.enter.native="search" @clear="search" />
        <el-button type="primary" icon="el-icon-search" @click="search">查询</el-button>
        <div class="grow" />
        <span class="help-line">家园访客记录</span>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="owner_nick" label="家园主人" min-width="130" />
        <el-table-column prop="visit_nick" label="访客" min-width="130" />
        <el-table-column label="访问时间" width="180"><template slot-scope="{row}">{{ fmt(row.created_at) }}</template></el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page"
                     @current-change="p => { page = p; load() }" style="margin-top:14px;text-align:right" />
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminVisitors',
  data () { return { list: [], total: 0, page: 1, size: 15, userId: '', loading: false } },
  mounted () { this.load() },
  methods: {
    search () { this.page = 1; this.load() },
    load () {
      this.loading = true
      api.get('/admin/visitors', { params: { page: this.page, size: this.size, user_id: this.userId || undefined } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>