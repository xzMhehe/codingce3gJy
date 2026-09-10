<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="按书名/作者搜索" prefix-icon="el-icon-search" clearable style="width:220px" @keyup.enter.native="search" @clear="search" />
        <el-button type="primary" icon="el-icon-search" @click="search">搜索</el-button>
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openCreate">新增书籍</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" align="center" />
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
        <el-table-column label="操作" width="150" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain @click="save(row)">保存</el-button>
            <el-button size="mini" type="danger" plain @click="del(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page"
                     @current-change="p => { page = p; load() }" style="margin-top:14px;text-align:right" />
    </el-card>

    <el-dialog title="新增书籍" :visible.sync="dlg" width="480px" :close-on-click-modal="false">
      <el-form label-width="70px" size="small">
        <el-form-item label="书名"><el-input v-model.trim="form.title" maxlength="60" style="width:300px" /></el-form-item>
        <el-form-item label="作者"><el-input v-model.trim="form.author" maxlength="30" style="width:300px" /></el-form-item>
        <el-form-item label="分类">
          <el-select v-model="form.category" style="width:160px" allow-create filterable>
            <el-option v-for="c in categories" :key="c" :value="c" :label="c" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio label="连载">连载</el-radio>
            <el-radio label="完结">完结</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="标记">
          <el-checkbox v-model="form.recommend">强力推荐</el-checkbox>
          <el-checkbox v-model="form.newBook">新书上架</el-checkbox>
        </el-form-item>
        <el-form-item label="简介"><el-input v-model="form.intro" type="textarea" :rows="3" maxlength="300" /></el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="dlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminBooks',
  data () {
    return {
      list: [], total: 0, page: 1, size: 10, word: '', loading: false,
      categories: ['武侠', '言情', '都市', '灵异'],
      dlg: false, saving: false,
      form: { title: '', author: '', category: '武侠', status: '连载', recommend: false, newBook: false, intro: '' }
    }
  },
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
    openCreate () {
      this.form = { title: '', author: '', category: this.categories[0], status: '连载', recommend: false, newBook: false, intro: '' }
      this.dlg = true
    },
    save () {
      if (!this.form.title || !this.form.author) { this.$message.warning('书名和作者必填'); return }
      this.saving = true
      api.post('/admin/books', {
        title: this.form.title, author: this.form.author, category: this.form.category,
        status: this.form.status, intro: this.form.intro,
        recommend: this.form.recommend ? 1 : 0, new_book: this.form.newBook ? 1 : 0
      }).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.$message.success(r.data.msg)
          this.dlg = false
          this.load()
        } else this.$message.error(r.msg)
      })
    },
    save (row) {
      api.put('/admin/books/' + row.id, { title: row.title, author: row.author, category: row.category, status: row.status, intro: row.intro || '' }).then(r => {
        if (r.code === 0) this.$message.success('已保存'); else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm(`删除小说「${row.title}」？将同时删除其章节与书评`, '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/books/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>
