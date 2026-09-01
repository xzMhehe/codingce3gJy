<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="按标题搜索" prefix-icon="el-icon-search" clearable
                  style="width:220px" @keyup.enter.native="search" @clear="search" />
        <el-button type="primary" icon="el-icon-search" @click="search">搜索</el-button>
        <div class="grow" />
        <el-tag size="small">置顶{{ topCount }}</el-tag>
        <el-tag type="success" size="small">精华{{ fineCount }}</el-tag>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="标题" min-width="240" show-overflow-tooltip>
          <template slot-scope="{row}">
            <el-tag v-if="row.is_top" type="danger" size="mini" style="margin-right:4px">顶</el-tag>
            <el-tag v-if="row.is_fine" type="success" size="mini" style="margin-right:4px">精</el-tag>
            {{ row.title }}
          </template>
        </el-table-column>
        <el-table-column label="板块" min-width="100" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.board ? row.board.name : '—' }}</template>
        </el-table-column>
        <el-table-column label="楼主" min-width="100" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.user ? row.user.nickname : '—' }}</template>
        </el-table-column>
        <el-table-column label="数据" width="110">
          <template slot-scope="{row}">{{ row.view_count }}阅/{{ row.reply_count }}回</template>
        </el-table-column>
        <el-table-column label="操作" width="340" fixed="right" header-align="center">
          <template slot-scope="{row}">
            <div class="ops">
              <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openDlg(row)">编辑</el-button>
              <el-button size="mini" @click="toggle(row, 'is_top')">{{ row.is_top ? '取消置顶' : '置顶' }}</el-button>
              <el-button size="mini" @click="toggle(row, 'is_fine')">{{ row.is_fine ? '取消精华' : '加精' }}</el-button>
              <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="del(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total"
                     :page-size="size" :current-page="page"
                     @current-change="p => { page = p; load() }"
                     style="margin-top:14px;text-align:right" />
    </el-card>

    <!-- 编辑帖子 模态框 -->
    <el-dialog title="编辑帖子" :visible.sync="dlg" width="640px" :close-on-click-modal="false">
      <el-form label-width="70px">
        <el-form-item label="标题">
          <el-input v-model.trim="form.title" maxlength="100" />
        </el-form-item>
        <el-form-item label="内容">
          <el-input type="textarea" v-model="form.content" :rows="10" maxlength="10000" show-word-limit />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="dlg = false">取 消</el-button>
        <el-button type="primary" @click="save">确 定</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminThreads',
  data () {
    return {
      list: [], total: 0, page: 1, pages: 1, size: 10, word: '', loading: false,
      dlg: false, delId: 0, msg: '',
      form: { id: 0, title: '', content: '' }
    }
  },
  computed: {
    topCount () { return this.list.filter(t => t.is_top).length },
    fineCount () { return this.list.filter(t => t.is_fine).length }
  },
  mounted () { this.load() },
  methods: {
    search () { this.page = 1; this.load() },
    load () {
      this.loading = true
      api.get('/admin/threads', { params: { page: this.page, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    openDlg (row) {
      this.form = { id: row.id, title: row.title, content: row.content }
      this.dlg = true
    },
    save () {
      api.put(`/threads/${this.form.id}`, { title: this.form.title, content: this.form.content }).then(r => {
        if (r.code === 0) { this.$message.success('已保存'); this.dlg = false; this.load() } else this.$message.error(r.msg)
      })
    },
    toggle (row, field) {
      api.put(`/admin/threads/${row.id}`, { [field]: row[field] ? 0 : 1 }).then(r => {
        if (r.code === 0) this.load()
        else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm(`确定删除《${row.title}》吗？`, '提示', { type: 'warning' }).then(() => {
        api.delete(`/admin/threads/${row.id}`).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>
