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
      <el-table :data="list" v-loading="loading" stripe style="width:100%">
        <el-table-column prop="id" label="ID" width="80" header-align="center" />
        <el-table-column label="标题" min-width="200" show-overflow-tooltip>
          <template slot-scope="{row}">
            <el-tag v-if="row.is_head" type="warning" size="mini" style="margin-right:4px">头条</el-tag>
            <el-tag v-if="row.is_top" type="danger" size="mini" style="margin-right:4px">顶</el-tag>
            <el-tag v-if="row.is_fine" type="success" size="mini" style="margin-right:4px">精</el-tag>
            <el-tag v-if="row.is_lock" type="info" size="mini" style="margin-right:4px">锁</el-tag>
            <el-tag v-if="row.is_recom" type="warning" size="mini" style="margin-right:4px">荐</el-tag>
            <el-tag v-if="row.is_active" type="danger" size="mini" style="margin-right:4px">活动</el-tag>
            <el-tag v-if="row.type === 1" type="success" size="mini" style="margin-right:4px">奖励</el-tag>
            <el-tag v-if="row.type === 2" type="success" size="mini" style="margin-right:4px">踩楼</el-tag>
            <el-tag v-if="row.type === 3" type="success" size="mini" style="margin-right:4px">投票</el-tag>
            {{ row.title }}
          </template>
        </el-table-column>
        <el-table-column label="板块" min-width="100" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.board ? row.board.name : '—' }}</template>
        </el-table-column>
        <el-table-column label="楼主" min-width="100" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.user ? row.user.nickname : '—' }}</template>
        </el-table-column>
        <el-table-column label="数据" min-width="90" header-align="center">
          <template slot-scope="{row}">{{ row.view_count }}阅/{{ row.reply_count }}回</template>
        </el-table-column>
        <el-table-column label="状态" width="80" align="center">
          <template slot-scope="{row}">
            <el-tag v-if="row.audit_status === 0" type="warning" size="mini">待审核</el-tag>
            <el-tag v-else-if="row.audit_status === 2" type="danger" size="mini">未通过</el-tag>
            <el-tag v-else type="success" size="mini">已发布</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" min-width="360">
          <template slot-scope="{row}">
            <div class="ops">
              <el-button size="mini" type="primary" plain @click="openDlg(row)">编辑</el-button>
              <el-button size="mini" @click="toggle(row, 'is_top')">{{ row.is_top ? '取消置顶' : '置顶' }}</el-button>
              <el-button size="mini" @click="toggle(row, 'is_fine')">{{ row.is_fine ? '取消精华' : '加精' }}</el-button>
              <el-button size="mini" @click="toggle(row, 'is_head')">{{ row.is_head ? '取消头条' : '头条' }}</el-button>
              <el-button size="mini" @click="toggle(row, 'is_lock')">{{ row.is_lock ? '解锁' : '锁定' }}</el-button>
              <el-button size="mini" @click="toggle(row, 'is_recom')">{{ row.is_recom ? '取消推荐' : '推荐' }}</el-button>
              <el-button size="mini" @click="toggle(row, 'is_active')">{{ row.is_active ? '取消活动' : '设活动' }}</el-button>
              <el-button v-if="row.audit_status !== 1" size="mini" type="success" plain @click="audit(row, 1)">通过</el-button>
              <el-button v-if="row.audit_status === 1" size="mini" type="warning" plain @click="audit(row, 0)">转审核</el-button>
              <el-button size="mini" type="danger" plain @click="del(row)">删除</el-button>
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
    audit (row, status) {
      api.put(`/admin/threads/${row.id}`, { audit_status: status }).then(r => {
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
