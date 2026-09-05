<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-radio-group v-model="status" size="small" @change="load">
          <el-radio-button label="-1">全部</el-radio-button>
          <el-radio-button label="0">待处理</el-radio-button>
          <el-radio-button label="1">已忽略</el-radio-button>
          <el-radio-button label="2">已删内容</el-radio-button>
          <el-radio-button label="3">已封禁</el-radio-button>
        </el-radio-group>
        <div class="grow" />
        <el-tag size="small" type="danger">待处理 {{ pendingCount }}</el-tag>
      </div>

      <el-table :data="list" v-loading="loading" stripe style="width:100%">
        <el-table-column prop="id" label="ID" width="70" header-align="center" />
        <el-table-column label="举报人" width="110" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.reporter ? row.reporter.nickname : ('#' + row.reporter_id) }}</template>
        </el-table-column>
        <el-table-column label="对象" width="90" header-align="center">
          <template slot-scope="{row}">
            <el-tag :type="row.target_type === 'thread' ? 'primary' : 'warning'" size="mini">
              {{ row.target_type === 'thread' ? '帖子' : '回复' }}#{{ row.target_id }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="被举报内容" min-width="200" show-overflow-tooltip>
          <template slot-scope="{row}">
            <span v-if="row.content_status === 0" style="color:#c0c4cc;text-decoration:line-through">[已删除] {{ row.content }}</span>
            <span v-else>{{ row.content }}</span>
          </template>
        </el-table-column>
        <el-table-column label="作者" width="110" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.author ? row.author.nickname : '—' }}</template>
        </el-table-column>
        <el-table-column label="理由" min-width="140" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.reason }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90" header-align="center">
          <template slot-scope="{row}">
            <el-tag :type="statusType(row.status)" size="mini">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="举报时间" width="150">
          <template slot-scope="{row}">{{ fmt(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="230" header-align="center">
          <template slot-scope="{row}">
            <template v-if="row.status === 0">
              <el-button size="mini" type="success" plain @click="handle(row, 'ignore')">忽略</el-button>
              <el-button size="mini" type="warning" plain @click="handle(row, 'delete')">删内容</el-button>
              <el-button size="mini" type="danger" plain @click="handle(row, 'ban')">封作者</el-button>
            </template>
            <span v-else class="txt-fade">{{ row.result || '—' }}</span>
          </template>
        </el-table-column>
      </el-table>

      <div class="pager">
        <el-pagination background layout="prev, pager, next, total" :total="total" :page-size="size"
                       :current-page="page" @current-change="go" />
      </div>
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminReports',
  data () {
    return { status: '0', list: [], total: 0, page: 1, size: 10, loading: false, pendingCount: 0 }
  },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/reports', { params: { status: this.status, page: this.page, size: this.size } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
        }
        this.loadPending()
      })
    },
    loadPending () {
      api.get('/admin/reports', { params: { status: 0, page: 1, size: 1 } }).then(r => {
        if (r.code === 0) this.pendingCount = r.data.total
      })
    },
    go (p) { this.page = p; this.load() },
    handle (row, action) {
      const tips = { ignore: '确定忽略这条举报？', delete: '确定删除被举报内容？', ban: '确定封禁被举报内容的作者？' }
      if (!confirm(tips[action])) return
      api.put(`/admin/reports/${row.id}`, { action, result: '' }).then(r => {
        if (r.code === 0) { this.$message.success('处理完成：' + r.data.result); this.load() }
        else this.$message.error(r.msg)
      })
    },
    statusType (s) { return { 0: 'danger', 1: 'info', 2: 'warning', 3: 'danger' }[s] || 'info' },
    statusText (s) { return { 0: '待处理', 1: '已忽略', 2: '已删内容', 3: '已封禁' }[s] || '未知' },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t)
      const p = n => (n < 10 ? '0' + n : n)
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes())
    }
  }
}
</script>

<style scoped>
.toolbar { display: flex; align-items: center; margin-bottom: 14px; }
.grow { flex: 1; }
.pager { margin-top: 14px; text-align: right; }
.txt-fade { color: #909399; font-size: 12px; }
</style>
