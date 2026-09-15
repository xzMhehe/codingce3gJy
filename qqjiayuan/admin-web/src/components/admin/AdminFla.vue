<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="q.user" placeholder="号码/昵称" size="small" clearable style="width:160px" @keyup.enter.native="search" />
        <el-date-picker v-model="q.date" type="date" value-format="yyyy-MM-dd" placeholder="捐款日期" size="small" style="width:140px" />
        <el-button size="small" type="primary" icon="el-icon-search" @click="search">查询</el-button>
        <el-button size="small" @click="reset">重置</el-button>
        <div class="grow" />
        <el-button size="small" type="success" icon="el-icon-plus" @click="openAdd">手工补录</el-button>
      </div>

      <el-alert type="info" :closable="false" style="margin-bottom:10px">
        福利院·捐款慈善基金：每日捐款价高者上榜，榜首可受全社区膜拜。此处维护全部上榜记录（手工补录不扣 G 币）。
      </el-alert>

      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="用户" min-width="150">
          <template slot-scope="{row}">
            <span :style="{ color: row.color || '' }">{{ row.nickname }}</span>
            <span class="muted">（{{ row.username }}）</span>
          </template>
        </el-table-column>
        <el-table-column prop="amount" label="捐款金额(G币)" width="130" align="right">
          <template slot-scope="{row}">{{ row.amount | num }}</template>
        </el-table-column>
        <el-table-column prop="word" label="捐赠宣言" min-width="200" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.word || '—' }}</template>
        </el-table-column>
        <el-table-column prop="worships" label="膜拜数" width="90" align="center" />
        <el-table-column label="捐款时间" width="170">
          <template slot-scope="{row}">{{ fmt(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="140" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" plain @click="openEdit(row)">编辑</el-button>
            <el-button size="mini" type="danger" plain @click="del(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page"
                     @current-change="p => { page = p; load() }" style="margin-top:14px;text-align:right" />
    </el-card>

    <el-dialog :title="editForm.id ? '编辑上榜记录' : '手工补录上榜记录'" :visible.sync="dlg" width="420px">
      <el-form label-width="90px" size="small">
        <el-form-item v-if="!editForm.id" label="用户">
          <el-input v-model.trim="editForm.username" placeholder="家园号码（填昵称请勾选右侧）" style="width:220px" />
          <el-checkbox v-model="editForm.byNick" style="margin-left:8px">按昵称</el-checkbox>
        </el-form-item>
        <el-form-item v-else label="用户">
          <span>{{ editForm.nickname }}（{{ editForm.username }}）</span>
        </el-form-item>
        <el-form-item label="捐款金额">
          <el-input-number v-model="editForm.amount" :min="1" :max="999999999" :controls="false" style="width:160px" />
          <span class="muted"> G币</span>
        </el-form-item>
        <el-form-item label="捐赠宣言">
          <el-input v-model.trim="editForm.word" type="textarea" :rows="2" maxlength="30" show-word-limit />
        </el-form-item>
        <el-form-item v-if="editForm.id" label="膜拜数">
          <el-input-number v-model="editForm.worships" :min="0" :controls="false" style="width:160px" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button size="small" @click="dlg = false">取 消</el-button>
        <el-button size="small" type="primary" @click="save">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminFla',
  filters: {
    num (v) { return Number(v || 0).toLocaleString() }
  },
  data () {
    return {
      list: [], total: 0, page: 1, size: 10, loading: false,
      q: { user: '', date: '' },
      dlg: false, saving: false,
      editForm: { id: 0, username: '', byNick: false, amount: 500000, word: '', worships: 0, nickname: '' }
    }
  },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      const params = { page: this.page, size: this.size }
      if (this.q.user) params.user = this.q.user
      if (this.q.date) params.date = this.q.date
      api.get('/admin/fla-donations', { params }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    search () { this.page = 1; this.load() },
    reset () { this.q = { user: '', date: '' }; this.search() },
    openAdd () {
      this.editForm = { id: 0, username: '', byNick: false, amount: 500000, word: '', worships: 0, nickname: '' }
      this.dlg = true
    },
    openEdit (row) {
      this.editForm = {
        id: row.id, username: row.username, byNick: false, nickname: row.nickname,
        amount: row.amount, word: row.word || '', worships: row.worships
      }
      this.dlg = true
    },
    save () {
      const f = this.editForm
      if (!f.id && !f.username) { this.$message.warning('请填写家园号码或昵称'); return }
      if (!f.amount || f.amount < 1) { this.$message.warning('捐款金额需大于 0'); return }
      this.saving = true
      const req = f.id
        ? api.put('/admin/fla-donations/' + f.id, { amount: f.amount, word: f.word, worships: f.worships })
        : api.post('/admin/fla-donations', f.byNick ? { nickname: f.username, amount: f.amount, word: f.word } : { username: f.username, amount: f.amount, word: f.word })
      req.then(r => {
        this.saving = false
        if (r.code === 0) {
          this.$message.success(f.id ? '已保存' : '已补录')
          this.dlg = false
          this.load()
        } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm(`删除 ${row.nickname} 的这条上榜记录？（金额 ${row.amount}，连带其膜拜明细）`, '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/fla-donations/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>

<style scoped>
.toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 10px; }
.toolbar .grow { flex: 1; }
.muted { color: #999; }
</style>
