<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-select v-model="bookId" filterable placeholder="选择书籍" style="width:280px" @change="loadChapters">
          <el-option v-for="b in books" :key="b.id" :value="b.id" :label="`《${b.title}》 ${b.author}`" />
        </el-select>
        <el-button type="primary" icon="el-icon-plus" :disabled="!bookId" @click="openDlg(null)">新增章节</el-button>
      </div>
      <el-table :data="chapters" v-loading="loading" stripe>
        <el-table-column prop="sort" label="序号" width="70" align="center" />
        <el-table-column label="章节标题" min-width="220" show-overflow-tooltip>
          <template slot-scope="{row}">
            {{ row.title }}
            <el-tag v-if="row.vip === 1" type="danger" size="mini">VIP</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="内容预览" min-width="280" show-overflow-tooltip>
          <template slot-scope="{row}">{{ (row.content || '').slice(0, 40) || '—' }}</template>
        </el-table-column>
        <el-table-column label="更新时间" width="160" align="center">
          <template slot-scope="{row}">{{ fmt(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="110" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" icon="el-icon-edit" circle title="编辑" @click="openDlg(row)" />
            <el-button size="mini" type="danger" icon="el-icon-delete" circle title="删除" @click="del(row)" />
          </template>
        </el-table-column>
      </el-table>
      <div v-if="!bookId" class="txt-fade" style="padding:12px">请先选择一本书籍查看章节。</div>
    </el-card>

    <el-dialog :title="form.id ? '编辑章节' : '新增章节'" :visible.sync="dlg" width="640px" :close-on-click-modal="false">
      <el-form label-width="80px" size="small">
        <el-form-item label="章节标题">
          <el-input v-model.trim="form.title" maxlength="60" style="width:340px" />
        </el-form-item>
        <el-form-item label="序号">
          <el-input-number v-model.number="form.sort" :min="0" controls-position="right" />
          <el-checkbox v-model="isVip" style="margin-left:16px">VIP章节</el-checkbox>
        </el-form-item>
        <el-form-item label="正文">
          <el-input v-model="form.content" type="textarea" :rows="12" placeholder="章节正文，空行分段" />
        </el-form-item>
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
  name: 'AdminBookChapters',
  data () {
    return {
      books: [], bookId: null, chapters: [], loading: false,
      dlg: false, saving: false, isVip: false,
      form: { id: 0, title: '', content: '', sort: 0 }
    }
  },
  mounted () { this.loadBooks() },
  methods: {
    loadBooks () {
      api.get('/admin/books', { params: { page: 1, size: 500 } }).then(r => {
        if (r.code === 0) {
          this.books = r.data.list || []
          if (this.books.length && !this.bookId) {
            this.bookId = this.books[0].id
            this.loadChapters()
          }
        }
      })
    },
    loadChapters () {
      if (!this.bookId) return
      this.loading = true
      api.get(`/admin/books/${this.bookId}/chapters`).then(r => {
        this.loading = false
        if (r.code === 0) this.chapters = r.data
      })
    },
    openDlg (row) {
      if (row) {
        this.form = { id: row.id, title: row.title, content: row.content, sort: row.sort }
        this.isVip = row.vip === 1
      } else {
        const maxSort = this.chapters.reduce((m, c) => Math.max(m, c.sort || 0), 0)
        this.form = { id: 0, title: '', content: '', sort: maxSort + 1 }
        this.isVip = false
      }
      this.dlg = true
    },
    save () {
      if (!this.form.title) { this.$message.warning('请填写章节标题'); return }
      this.saving = true
      const body = { title: this.form.title, content: this.form.content, sort: this.form.sort, vip: this.isVip ? 1 : 0 }
      const call = this.form.id
        ? api.put('/admin/book-chapters/' + this.form.id, body)
        : api.post(`/admin/books/${this.bookId}/chapters`, body)
      call.then(r => {
        this.saving = false
        if (r.code === 0) { this.$message.success(r.data.msg || '已保存'); this.dlg = false; this.loadChapters() } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm('确定删除章节「' + row.title + '」？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/book-chapters/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.loadChapters() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>
