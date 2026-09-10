<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="按号码/昵称搜索" clearable style="width:220px"
                  @keyup.enter.native="page = 1; load()" @clear="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" align="center" />
        <el-table-column label="书籍" min-width="160" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.book || '—' }}</template>
        </el-table-column>
        <el-table-column label="书评人" min-width="110" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.nick || row.user_id }}</template>
        </el-table-column>
        <el-table-column label="评分" width="120" align="center">
          <template slot-scope="{row}"><span style="color:#e6a23c">{{ stars(row.score) }}</span></template>
        </el-table-column>
        <el-table-column prop="content" label="内容" min-width="260" show-overflow-tooltip />
        <el-table-column label="状态" width="80" align="center">
          <template slot-scope="{row}">
            <el-tag :type="row.visible ? 'success' : 'info'" size="mini">{{ row.visible ? '显示' : '隐藏' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="时间" width="160" align="center">
          <template slot-scope="{row}">{{ fmt(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="110" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" :type="row.visible ? 'warning' : 'success'" plain @click="toggle(row)">{{ row.visible ? '隐藏' : '显示' }}</el-button>
            <el-button size="mini" type="danger" icon="el-icon-delete" circle title="删除" @click="del(row)" />
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
  name: 'AdminBookComments',
  data () { return { list: [], total: 0, page: 1, size: 15, loading: false, word: '' } },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/book-comments', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    toggle (row) {
      api.put('/admin/book-comments/' + row.id, { status: row.visible ? 0 : 1 }).then(r => {
        if (r.code === 0) { this.$message.success('已保存'); this.load() } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm('确定删除该条书评？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/book-comments/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    stars (n) { return '★★★★★☆☆☆☆☆'.slice(5 - (n || 5), 10 - (n || 5)) },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>
