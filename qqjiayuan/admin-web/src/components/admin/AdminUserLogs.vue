<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model.number="userId" placeholder="按用户号码过滤" clearable style="width:200px" @keyup.enter.native="search" @clear="search" />
        <el-button type="primary" icon="el-icon-search" @click="search">查询</el-button>
        <div class="grow" />
        <span class="help-line">会员登录/操作日志（对齐诺哈 会员日志）</span>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="user_id" label="号码" width="90" />
        <el-table-column prop="nickname" label="昵称" min-width="110" />
        <el-table-column prop="action" label="动作" width="110">
          <template slot-scope="{row}">
            <el-tag :type="row.action === '登陆成功' ? 'success' : (row.action === '登陆失败' ? 'danger' : 'info')" size="mini">{{ row.action }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="intro" label="详情" min-width="180" show-overflow-tooltip />
        <el-table-column prop="ip" label="IP" width="130" />
        <el-table-column label="时间" width="170"><template slot-scope="{row}">{{ fmt(row.created_at) }}</template></el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page"
                     @current-change="p => { page = p; load() }" style="margin-top:14px;text-align:right" />
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminUserLogs',
  data () { return { list: [], total: 0, page: 1, size: 15, userId: '', loading: false } },
  mounted () { this.load() },
  methods: {
    search () { this.page = 1; this.load() },
    load () {
      this.loading = true
      api.get('/admin/user-logs', { params: { page: this.page, size: this.size, user_id: this.userId || undefined } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>