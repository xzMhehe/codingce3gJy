<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model.number="userId" placeholder="按用户号码过滤" clearable style="width:200px" @keyup.enter.native="search" @clear="search" />
        <el-button type="primary" icon="el-icon-search" @click="search">查询</el-button>
        <div class="grow" />
        <span class="help-line">新鲜事动态</span>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="user_id" label="号码" width="90" />
        <el-table-column prop="nickname" label="用户" min-width="110" />
        <el-table-column prop="content" label="动态" min-width="260" show-overflow-tooltip />
        <el-table-column label="类型" width="90" align="center">
          <template slot-scope="{row}">{{ typeName(row.ntype) }}</template>
        </el-table-column>
        <el-table-column label="时间" width="170"><template slot-scope="{row}">{{ fmt(row.created_at) }}</template></el-table-column>
        <el-table-column label="操作" width="90" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="danger" plain @click="del(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page"
                     @current-change="p => { page = p; load() }" style="margin-top:14px;text-align:right" />
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminHomeNews',
  data () { return { list: [], total: 0, page: 1, size: 15, userId: '', loading: false } },
  mounted () { this.load() },
  methods: {
    search () { this.page = 1; this.load() },
    load () {
      this.loading = true
      api.get('/admin/home-news', { params: { page: this.page, size: this.size, user_id: this.userId || undefined } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm('删除这条新鲜事？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/home-news/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    typeName (t) {
      return { 1: '发帖', 2: '回帖', 101: '心情', 102: '日志', 103: '照片', 104: '留言', 105: '文章' }[t] || '其他'
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>