<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="家园号 / 昵称搜索" clearable style="width:220px"
                  @keyup.enter.native="page = 1; load()" />
        <el-select v-model="opened" size="small" style="width:120px" @change="page = 1; load()">
          <el-option label="全部用户" value="all" />
          <el-option label="已开通" value="1" />
          <el-option label="未开通" value="0" />
        </el-select>
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <el-button type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="username" label="家园号" width="110" />
        <el-table-column label="昵称" min-width="120" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.nickname || '—' }}</template>
        </el-table-column>
        <el-table-column label="昵称颜色" width="150">
          <template slot-scope="{row}">
            <span v-if="row.multi">
              <font v-for="(c,i) in (row.color||'').toLowerCase().split(',')" :key="i" :color="c">{{ '◆' }}</font>
            </span>
            <span v-else><font :color="row.color || '#004299'">◆</font></span>
            <span style="margin-left:6px;font-size:12px;color:#999">{{ row.color || '默认' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="开通状态" width="100" align="center">
          <template slot-scope="{row}">
            <el-tag :type="row.opened ? 'success' : 'info'" size="mini">{{ row.opened ? '已开通' : '未开通' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="剩余天数" width="90" align="center">
          <template slot-scope="{row}">{{ row.opened ? row.days_left : '—' }}</template>
        </el-table-column>
        <el-table-column label="到期时间" width="160">
          <template slot-scope="{row}">{{ row.name_end ? fmt(row.name_end) : '—' }}</template>
        </el-table-column>
        <el-table-column label="开通时间" width="160">
          <template slot-scope="{row}">{{ row.name_start ? fmt(row.name_start) : '—' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="230" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain icon="el-icon-edit" title="调整" @click="openEdit(row)" />
            <el-button size="mini" type="warning" plain title="颜色重置为默认蓝" @click="resetColor(row)">重置颜色</el-button>
            <el-button size="mini" type="danger" plain icon="el-icon-delete" title="重置开通状态" @click="resetAll(row)" />
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, sizes, prev, pager, next" :total="total"
                     :page-size="size" :current-page="page" :page-sizes="[10, 20, 50]"
                     @current-change="p => { page = p; load() }"
                     @size-change="s => { size = s; page = 1; load() }" />
    </el-card>

    <!-- 调整（续期/到期/颜色） -->
    <el-dialog :title="'调整个性昵称 · ' + (editRow ? (editRow.nickname || editRow.id) : '')" :visible.sync="editDlg" width="540px" :close-on-click-modal="false">
      <el-form label-width="90px" size="small">
        <el-form-item label="续期(天)">
          <el-input-number v-model="editDays" :min="-3650" :max="3650" />
          <div class="form-hint">正数延长，负数缩短；未开通/已过期则从今天起算</div>
        </el-form-item>
        <el-form-item label="到期时间">
          <el-date-picker v-model="editEnd" type="datetime" placeholder="留空则不修改" format="yyyy-MM-dd HH:mm" />
          <div class="form-hint">留空则不修改；想清空到期请用「重置开通状态」按钮</div>
        </el-form-item>
        <el-form-item label="昵称颜色">
          <el-input v-model="editColor" placeholder="单色 #06f / 逐字序列 #D2B48C,#800080,#DAA520" style="width:330px" />
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="editDlg = false">取 消</el-button>
        <el-button type="primary" @click="save">确 定</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminNameConfig',
  data () {
    return {
      word: '', opened: '', page: 1, size: 10, total: 0, list: [],
      loading: false,
      editDlg: false, editRow: null, editColor: '', editEnd: '',
      editDays: 0
    }
  },
  mounted () { this.load() },
  methods: {
    async load () {
      this.loading = true
      const r = await api.get('/admin/name-users', { params: { word: this.word, opened: this.opened, page: this.page, size: this.size } })
      this.loading = false
      if (r.code === 0) {
        this.list = r.data.list
        this.total = r.data.total
      }
    },
    fmt (t) {
      if (!t) return '—'
      const d = new Date(t)
      const p = n => (n < 10 ? '0' : '') + n
      return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
    },
    openEdit (row) {
      this.editRow = row
      this.editColor = (row.color || '').toLowerCase()
      this.editEnd = row.name_end ? new Date(row.name_end) : null
      this.editDays = 0
      this.editDlg = true
    },
    async submit () {
      const body = {}
      if (this.editDays !== 0) body.add_days = this.editDays
      if (this.editColor !== '' && this.editColor !== null) body.color = this.editColor
      if (this.editEnd) body.end = this.fmtDate(this.editEnd)
      const r = await api.put(`/admin/name-users/${this.editRow.id}`, body)
      if (r.code === 0) {
        this.$message.success('设置成功')
        this.editDlg = false
        this.load()
      } else { this.$message.error(r.msg || '设置失败') }
    },
    fmtDate (d) {
      const p = n => (n < 10 ? '0' : '') + n
      return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
    },
    async resetAll (row) {
      this.$confirm(`将重置 ${row.nickname || row.id} 的个性昵称开通状态（颜色恢复默认蓝）？`, '确认', { type: 'warning' }).then(async () => {
        const r = await api.put(`/admin/name-users/${row.id}`, { clear: true, reset_color: true })
        if (r.code === 0) { this.$message.success('重置成功'); this.load() } else { this.$message.error(r.msg) }
      }).catch(() => {})
    },
    async resetColor (row) {
      const r = await api.put(`/admin/name-users/${row.id}`, { reset_color: true })
      if (r.code === 0) { this.$message.success('颜色已重置'); this.load() } else { this.$message.error(r.msg) }
    }
  }
}
</script>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; align-items: center; }
.grow { flex: 1; }
.form-hint { margin-left: 10px; color: #97a8be; font-size: 12px; }
</style>
