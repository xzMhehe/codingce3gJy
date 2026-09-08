<template>
  <div>
    <el-tabs v-model="cur">
      <el-tab-pane label="文章管理" name="articles">
        <el-card shadow="never" class="box">
          <el-table :data="list" v-loading="loading" stripe>
            <el-table-column prop="id" label="ID" width="70" />
            <el-table-column prop="title" label="标题" min-width="200" show-overflow-tooltip />
            <el-table-column prop="author" label="作者" min-width="110" />
            <el-table-column prop="click" label="浏览" width="80" align="center" />
            <el-table-column prop="comment" label="评论" width="80" align="center" />
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
      </el-tab-pane>
      <el-tab-pane label="文章分类" name="cats">
        <el-card shadow="never" class="box">
          <div class="toolbar">
            <el-input v-model="catForm.name" placeholder="分类名" maxlength="30" style="width:200px" />
            <el-input-number v-model="catForm.sort" :min="0" :max="999" size="small" placeholder="排序" />
            <el-button type="primary" icon="el-icon-plus" @click="addCat">新增分类</el-button>
          </div>
          <el-table :data="cats" v-loading="loading" stripe>
            <el-table-column prop="id" label="ID" width="70" />
            <el-table-column label="分类名" min-width="180">
              <template slot-scope="{row}"><el-input v-model="row.name" size="small" maxlength="30" /></template>
            </el-table-column>
            <el-table-column label="排序" width="140">
              <template slot-scope="{row}"><el-input-number v-model="row.sort" size="small" :min="0" :max="999" controls-position="right" /></template>
            </el-table-column>
            <el-table-column label="操作" width="160" align="center">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain @click="saveCat(row)">保存</el-button>
                <el-button size="mini" type="danger" plain @click="delCat(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminSiteArticles',
  data () {
    return { cur: 'articles', list: [], total: 0, page: 1, size: 10, loading: false, cats: [], catForm: { name: '', sort: 0 } }
  },
  mounted () { this.load(); this.loadCats() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/site-articles', { params: { page: this.page, size: this.size } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    loadCats () {
      api.get('/admin/article-categories').then(r => { if (r.code === 0) this.cats = r.data || [] })
    },
    del (row) {
      this.$confirm(`删除文章「${row.title}」？`, '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/site-articles/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    addCat () {
      if (!this.catForm.name) { this.$message.warning('请填写分类名'); return }
      api.post('/admin/article-categories', this.catForm).then(r => {
        if (r.code === 0) { this.$message.success('已新增'); this.catForm = { name: '', sort: 0 }; this.loadCats() } else this.$message.error(r.msg)
      })
    },
    saveCat (row) {
      api.put('/admin/article-categories/' + row.id, { name: row.name, sort: row.sort }).then(r => {
        if (r.code === 0) this.$message.success('已保存'); else this.$message.error(r.msg)
      })
    },
    delCat (row) {
      this.$confirm(`删除分类「${row.name}」？`, '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/article-categories/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.loadCats() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>