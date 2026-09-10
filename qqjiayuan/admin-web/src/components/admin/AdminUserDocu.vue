<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="按号码/昵称搜索" prefix-icon="el-icon-search" clearable
                  style="width:220px" @keyup.enter.native="search" @clear="search" />
        <el-button type="primary" icon="el-icon-search" @click="search">搜索</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="user_id" label="号码" width="90" align="center" />
        <el-table-column prop="nickname" label="昵称" min-width="120" show-overflow-tooltip />
        <el-table-column label="类型" width="90" align="center">
          <template slot-scope="{row}">{{ row.type === 1 ? '身份证' : '其他' }}</template>
        </el-table-column>
        <el-table-column prop="real_name" label="姓名" min-width="110" show-overflow-tooltip />
        <el-table-column prop="number" label="证件号" min-width="180" show-overflow-tooltip />
        <el-table-column label="更新时间" width="160" align="center">
          <template slot-scope="{row}">{{ fmt(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="80" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="del(row)" />
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
  name: 'AdminUserDocu',
  data () { return { list: [], total: 0, page: 1, size: 15, loading: false, word: '' } },
  mounted () { this.load() },
  methods: {
    search () { this.page = 1; this.load() },
    load () {
      this.loading = true
      api.get('/admin/user-docu', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm('确定删除「' + row.nickname + '」的实名证件吗？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/user-docu/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>
