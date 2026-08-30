<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">发布公告</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="类型" width="90">
          <template slot-scope="{row}">
            <el-tag :type="{ notice: '', broadcast: 'warning', activity: 'success' }[row.type]" size="mini">{{ typeName(row.type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="标题" min-width="180" show-overflow-tooltip />
        <el-table-column prop="content" label="内容" min-width="240" show-overflow-tooltip />
        <el-table-column label="状态" width="80">
          <template slot-scope="{row}">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '显示' : '隐藏' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="发布时间" width="150">
          <template slot-scope="{row}">{{ fmt(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="250" fixed="right">
          <template slot-scope="{row}">
            <div class="ops">
              <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openDlg(row)">编辑</el-button>
              <el-button size="mini" @click="toggle(row)">{{ row.status === 1 ? '隐藏' : '显示' }}</el-button>
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

    <!-- 发布 / 编辑公告 模态框 -->
    <el-dialog :title="form.id ? '编辑公告' : '发布公告'" :visible.sync="dlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="80px">
        <el-form-item label="类型">
          <el-radio-group v-model="form.type">
            <el-radio label="notice">公告</el-radio>
            <el-radio label="broadcast">广播</el-radio>
            <el-radio label="activity">活动</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="标题">
          <el-input v-model.trim="form.title" maxlength="100" show-word-limit />
        </el-form-item>
        <el-form-item label="内容">
          <el-input type="textarea" v-model="form.content" :rows="5" maxlength="2000" show-word-limit />
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
  name: 'AdminAnnouncements',
  data () {
    return { list: [], total: 0, page: 1, size: 10, loading: false, dlg: false, delId: 0, msg: '', err: '', form: { id: 0, type: 'notice', title: '', content: '' } }
  },
  mounted () { this.load() },
  methods: {
    typeName (t) { return { notice: '公告', broadcast: '广播', activity: '活动' }[t] || t },
    fmt (t) { return new Date(t).toISOString().slice(0, 16).replace('T', ' ') },
    load () {
      this.loading = true
      api.get('/admin/announcements', { params: { page: this.page, size: this.size } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    openDlg (row) {
      if (row) this.form = { id: row.id, type: row.type, title: row.title, content: row.content }
      else this.form = { id: 0, type: 'notice', title: '', content: '' }
      this.dlg = true
    },
    save () {
      if (!this.form.title) { this.$message.error('标题必填'); return }
      const payload = { type: this.form.type, title: this.form.title, content: this.form.content }
      const call = this.form.id ? api.put('/admin/announcements/' + this.form.id, payload) : api.post('/admin/announcements', payload)
      call.then(r => {
        if (r.code === 0) { this.$message.success('已保存'); this.dlg = false; this.load() } else this.$message.error(r.msg)
      })
    },
    toggle (row) {
      const payload = { type: row.type, title: row.title, content: row.content, status: row.status === 1 ? 0 : 1 }
      api.put('/admin/announcements/' + row.id, payload).then(r => {
        if (r.code === 0) { this.$message.success(row.status === 1 ? '已隐藏' : '已显示'); this.load() } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm(`确定删除「${row.title}」吗？`, '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/announcements/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>
