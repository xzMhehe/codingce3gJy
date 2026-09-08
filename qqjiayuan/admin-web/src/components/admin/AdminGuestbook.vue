<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <div class="grow" />
        <span class="help-line">全站留言本（对齐诺哈 留言管理，含私密留言内容）</span>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="留言人" min-width="110" />
        <el-table-column label="留言内容" min-width="260" show-overflow-tooltip>
          <template slot-scope="{row}">
            {{ row.content }}<el-tag v-if="row.pass" size="mini" type="warning" style="margin-left:6px">私密</el-tag>
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
  name: 'AdminGuestbook',
  data () { return { list: [], total: 0, page: 1, size: 10, loading: false } },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/guestbook', { params: { page: this.page, size: this.size } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm('删除这条留言？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/guestbook/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>