<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <span>二战风云 · 聊天敏感词（独立维护，与社区「黑名单榜」分开）</span>
        <el-button size="mini" type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>
      <div class="toolbar">
        <el-input v-model="word" placeholder="按敏感词搜索" clearable style="width:220px"
                  @keyup.enter.native="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <el-button type="success" icon="el-icon-plus" @click="openAdd">新增敏感词</el-button>
        <el-button type="warning" plain icon="el-icon-upload2" @click="bulkDlg = true">批量导入</el-button>
        <div class="grow" />
        <span class="td-sub">类型：<b>替换</b> = 用替换词覆盖（留空则打 *）；<b>拦截</b> = 直接拒绝发言</span>
      </div>
      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="70" align="center" />
        <el-table-column label="敏感词" min-width="160" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-main">{{ row.word }}</span></template>
        </el-table-column>
        <el-table-column label="替换词" min-width="160" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-mono">{{ row.replace || '（打 * ）' }}</span></template>
        </el-table-column>
        <el-table-column label="类型" width="100" align="center">
          <template slot-scope="{row}">
            <el-tag :type="row.type === 2 ? 'danger' : 'warning'" size="mini">
              {{ row.type === 2 ? '拦截' : '替换' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openEdit(row)" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="remove(row)" />
          </template>
        </el-table-column>
      </el-table>
      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ total }}</b> 条 · 每页 {{ size }} 条</div>
        <el-pagination v-show="total > 0" small background layout="sizes, prev, pager, next, jumper" :total="total" :page-size="size"
                       :current-page="page" :page-sizes="[5, 10, 20, 50, 100]"
                       @current-change="p => { page = p; load() }"
                       @size-change="s => { size = s; page = 1; load() }" />
      </div>
    </el-card>

    <el-dialog :title="formId ? '编辑敏感词' : '新增敏感词'" :visible.sync="dlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="敏感词">
          <el-input v-model="form.word" maxlength="50" placeholder="要过滤的词" />
        </el-form-item>
        <el-form-item label="替换词">
          <el-input v-model="form.replace" maxlength="50" placeholder="留空则用 * 覆盖" />
        </el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model.number="form.type">
            <el-radio :label="1">替换</el-radio>
            <el-radio :label="2">拦截（拒绝发言）</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="dlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保 存</el-button>
      </div>
    </el-dialog>

    <el-dialog title="批量导入敏感词" :visible.sync="bulkDlg" width="640px" :close-on-click-modal="false">
      <div class="td-sub" style="margin-bottom:8px">
        每行一条，格式：<b>词</b> 或 <b>词,替换词</b> 或 <b>词,替换词,类型</b>（类型 1=替换 2=拦截）；<b>#</b> 开头的行忽略。
      </div>
      <el-input type="textarea" v-model="bulkText" :rows="10" placeholder="赌博,***,2&#10;外挂" />
      <div slot="footer">
        <el-button @click="bulkDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doBulk">导 入</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminEzfyWords',
  data () {
    return {
      list: [], total: 0, page: 1, size: 5, word: '', loading: false,
      dlg: false, formId: 0, form: { word: '', replace: '', type: 1 }, saving: false,
      bulkDlg: false, bulkText: ''
    }
  },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/ezfy-word-filters', {
        params: { page: this.page, size: this.size, word: this.word }
      }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list || []; this.total = r.data.total || 0 } else this.$message.error(r.msg)
      })
    },
    openAdd () { this.formId = 0; this.form = { word: '', replace: '', type: 1 }; this.dlg = true },
    openEdit (row) { this.formId = row.id; this.form = { word: row.word, replace: row.replace, type: row.type }; this.dlg = true },
    save () {
      if (!this.form.word.trim()) { this.$message.error('敏感词不能为空'); return }
      this.saving = true
      const done = r => {
        this.saving = false
        if (r.code === 0) { this.dlg = false; this.$message.success(r.data.msg || '已保存'); this.load() }
        else this.$message.error(r.msg)
      }
      if (this.formId) api.put('/admin/ezfy-word-filters/' + this.formId, this.form).then(done)
      else api.post('/admin/ezfy-word-filters', this.form).then(done)
    },
    remove (row) {
      this.$confirm('删除敏感词「' + row.word + '」？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-word-filters/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    doBulk () {
      if (!this.bulkText.trim()) { this.$message.error('请填写要导入的内容'); return }
      this.saving = true
      api.post('/admin/ezfy-word-filters/bulk', { text: this.bulkText }).then(r => {
        this.saving = false
        if (r.code === 0) { this.bulkDlg = false; this.bulkText = ''; this.$message.success(r.data.msg || '导入完成'); this.load() }
        else this.$message.error(r.msg)
      })
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.card-head { display: flex; justify-content: space-between; align-items: center; }
</style>
