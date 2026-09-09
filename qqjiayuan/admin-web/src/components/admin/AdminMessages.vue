<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar"><span class="help-line">家信列表（站内私信）</span></div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="sender_id" label="发信号码" width="100" />
        <el-table-column prop="from_nick" label="发信人" min-width="110" />
        <el-table-column prop="receiver_id" label="收信号码" width="100" />
        <el-table-column prop="to_nick" label="收信人" min-width="110" />
        <el-table-column prop="content" label="内容" min-width="240" show-overflow-tooltip />
        <el-table-column label="已读" width="80" align="center">
          <template slot-scope="{row}">
            <el-tag :type="row.is_read ? 'info' : 'warning'" size="mini">{{ row.is_read ? '已读' : '未读' }}</el-tag>
          </template>
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
  name: 'AdminMessages',
  data () { return { list: [], total: 0, page: 1, size: 15, loading: false } },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/messages', { params: { page: this.page, size: this.size } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm('删除这条家信？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/messages/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>