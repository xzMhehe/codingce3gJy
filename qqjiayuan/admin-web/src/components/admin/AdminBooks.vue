<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="按书名/作者搜索" prefix-icon="el-icon-search" clearable style="width:220px" @keyup.enter.native="search" @clear="search" />
        <el-button type="primary" icon="el-icon-search" @click="search">搜索</el-button>
        <div class="grow" />
        <span class="help-line">小说列表</span>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="书名" min-width="180">
          <template slot-scope="{row}"><el-input v-model="row.title" size="small" maxlength="60" /></template>
        </el-table-column>
        <el-table-column label="作者" min-width="120">
          <template slot-scope="{row}"><el-input v-model="row.author" size="small" maxlength="30" /></template>
        </el-table-column>
        <el-table-column label="分类" min-width="100">
          <template slot-scope="{row}"><el-input v-model="row.category" size="small" maxlength="20" /></template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template slot-scope="{row}">
            <el-select v-model="row.status" size="small">
              <el-option label="连载" value="连载" />
              <el-option label="完结" value="完结" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column prop="views" label="点击" width="80" align="center" />
        <el-table-column label="操作" width="150" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain @click="save(row)">保存</el-button>
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
  name: 'AdminBooks',
  data () { return { list: [], total: 0, page: 1, size: 10, word: '', loading: false } },
  mounted () { this.load() },
  methods: {
    search () { this.page = 1; this.load() },
    load () {
      this.loading = true
      api.get('/admin/books', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    save (row) {
      api.put('/admin/books/' + row.id, { title: row.title, author: row.author, category: row.category, status: row.status, intro: row.intro || '' }).then(r => {
        if (r.code === 0) this.$message.success('已保存'); else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm(`删除小说「${row.title}」？`, '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/books/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>