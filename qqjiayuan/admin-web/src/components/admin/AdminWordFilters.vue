<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增敏感词</el-button>
        <el-button icon="el-icon-upload2" @click="bulkdlg = true">批量导入</el-button>
        <span class="help-line" style="margin-left:10px">替换型：帖子/聊天中出现即替换为占位符；拦截型：聊天直接拒绝发言，帖子进入待审核队列。</span>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" align="center" />
        <el-table-column prop="word" label="敏感词" min-width="180" show-overflow-tooltip />
        <el-table-column prop="replace" label="替换为" min-width="140" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.replace || '***' }}</template>
        </el-table-column>
        <el-table-column label="类型" width="90" align="center">
          <template slot-scope="{row}">
            <el-tag :type="row.type === 2 ? 'danger' : 'info'" size="mini">{{ row.type === 2 ? '拦截' : '替换' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openDlg(row)">编辑</el-button>
      <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="del(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog :title="form.id ? '编辑敏感词' : '新增敏感词'" :visible.sync="dlg" width="440px" :close-on-click-modal="false">
      <el-form label-width="80px">
        <el-form-item label="敏感词">
          <el-input v-model.trim="form.word" maxlength="50" />
        </el-form-item>
        <el-form-item label="替换为">
          <el-input v-model.trim="form.replace" maxlength="50" placeholder="默认 ***" />
        </el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model="form.type">
            <el-radio :label="1">替换</el-radio>
            <el-radio :label="2">拦截</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-alert type="info" :closable="false" style="margin-bottom:0" title="替换型和拦截型对聊天大厅（公共/家族/同城）与幻想西游聊天都生效：替换型把词换为占位符，拦截型直接拒绝发言。帖子里拦截型词走待审核。" />
      </el-form>
      <div slot="footer">
        <el-button @click="dlg = false">取 消</el-button>
        <el-button type="primary" @click="save">确 定</el-button>
      </div>
    </el-dialog>

    <el-dialog title="批量导入敏感词" :visible.sync="bulkdlg" width="520px" :close-on-click-modal="false">
      <el-input type="textarea" v-model="bulkContent" :rows="10" placeholder="每行一个：&#10;垃圾&#10;赌博,*,2支教&#10;天才,大聪明,2&#10;格式：词[,替换词[,类型 1替换/2拦截]]，# 开头的行忽略，重复词自动跳过"></el-input>
      <div class="help-line" style="margin-top:6px">也可用竖线 | 或中文逗号分隔。导入后立即对聊天与帖子生效。</div>
      <div slot="footer">
        <el-button @click="bulkdlg = false">取 消</el-button>
        <el-button type="primary" :loading="bulkLoading" @click="doBulk">导 入</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminWordFilters',
  data () {
    return { list: [], loading: false, dlg: false, form: { id: 0, word: '', replace: '***', type: 1 }, bulkdlg: false, bulkContent: '', bulkLoading: false }
  },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/word-filters').then(r => {
        this.loading = false
        if (r.code === 0) this.list = r.data
        else this.$message.error(r.msg)
      })
    },
    openDlg (row) {
      this.form = row ? { id: row.id, word: row.word, replace: row.replace, type: row.type } : { id: 0, word: '', replace: '***', type: 1 }
      this.dlg = true
    },
    save () {
      if (!this.form.word) { this.$message.error('请填写敏感词'); return }
      const call = this.form.id ? api.put('/admin/word-filters/' + this.form.id, this.form) : api.post('/admin/word-filters', this.form)
      call.then(r => {
        if (r.code === 0) {       this.$message.success('已保存'); this.dlg = false; this.load() } else this.$message.error(r.msg)
      })
    },
    doBulk () {
      if (!this.bulkContent.trim()) { this.$message.error('请填写要导入的内容'); return }
      this.bulkLoading = true
      api.post('/admin/word-filters/bulk', { content: this.bulkContent }).then(r => {
        this.bulkLoading = false
        if (r.code === 0) {
          this.$message.success(`导入完成：新增 ${r.data.added} 个，跳过（重复/为空）${r.data.skipped} 个`)
          this.bulkdlg = false
          this.bulkContent = ''
          this.load()
        } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm(`确定删除敏感词「${row.word}」吗？`, '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/word-filters/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>