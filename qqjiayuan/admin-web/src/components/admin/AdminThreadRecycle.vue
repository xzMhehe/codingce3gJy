<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <span class="help-line">已删除帖子</span>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="90" />
        <el-table-column prop="title" label="标题" min-width="220" show-overflow-tooltip />
        <el-table-column prop="nickname" label="楼主" min-width="110" />
        <el-table-column prop="view_count" label="浏览" width="80" align="center" />
        <el-table-column prop="reply_count" label="回复" width="80" align="center" />
        <el-table-column label="删除时间" width="170"><template slot-scope="{row}">{{ fmt(row.updated_at) }}</template></el-table-column>
        <el-table-column label="操作" width="110" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="success" plain @click="restore(row)">恢复</el-button>
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
  name: 'AdminThreadRecycle',
  data () { return { list: [], total: 0, page: 1, size: 10, loading: false } },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/threads/recycle', { params: { page: this.page, size: this.size } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    restore (row) {
      this.$confirm(`恢复帖子「${row.title}」？`, '提示', { type: 'info' }).then(() => {
        api.put('/admin/threads/' + row.id + '/restore').then(r => {
          if (r.code === 0) { this.$message.success('已恢复'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>