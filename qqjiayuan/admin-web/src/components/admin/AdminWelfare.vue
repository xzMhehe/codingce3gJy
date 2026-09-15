<template>
  <div>
    <el-card shadow="never" class="box" style="margin-bottom:12px">
      <el-alert type="info" :closable="false" style="margin-bottom:10px">
        福利院·慈善基金：友友完成「发帖 1 篇 + 回帖 5 条」后每日可从基金池领取 G币；捐献 100 G币入池并得 1 财富值。
      </el-alert>
      <div class="stats">
        <div class="stat">
          <div class="stat-label">当前基金池(G币)</div>
          <div class="stat-num" style="color:#e05a00">{{ stats.pool | num }}</div>
        </div>
        <div class="stat"><div class="stat-label">今日领取支出</div><div class="stat-num">{{ stats.today_out | num }}</div></div>
        <div class="stat"><div class="stat-label">今日捐献收入</div><div class="stat-num">{{ stats.today_in | num }}</div></div>
        <div class="stat"><div class="stat-label">累计领取支出</div><div class="stat-num">{{ stats.total_out | num }}</div></div>
        <div class="stat"><div class="stat-label">累计捐献收入</div><div class="stat-num">{{ stats.total_in | num }}</div></div>
        <div class="stat"><div class="stat-label">领取总次数</div><div class="stat-num">{{ stats.claim_count | num }}</div></div>
        <div class="stat"><div class="stat-label">捐献总次数</div><div class="stat-num">{{ stats.donate_count | num }}</div></div>
      </div>
      <div class="toolbar" style="margin-top:12px">
        <el-input-number v-model="poolDelta" :controls="false" size="small" style="width:180px" placeholder="±调整金额" />
        <el-button size="small" type="primary" @click="adjustPool">调整福利池</el-button>
        <el-divider direction="vertical" />
        <el-input-number v-model="poolSet" :controls="false" size="small" style="width:180px" placeholder="直接设置池金额" />
        <el-button size="small" type="warning" @click="setPool">直接设置</el-button>
        <span class="muted">调整=±微调；直接设置=覆盖为该值</span>
        <div class="grow" />
      </div>
      <div class="toolbar" style="margin-top:6px">
        <el-input v-model="createForm.username" size="small" placeholder="家园号码" style="width:150px" />
        <el-checkbox v-model="createForm.byNick" style="margin:0 8px">按昵称</el-checkbox>
        <el-input-number v-model="createForm.amount" :controls="false" size="small" :min="1" style="width:160px" />
        <el-button size="small" type="success" @click="createClaim">补发领取记录（加 G币入池外）</el-button>
      </div>
    </el-card>

    <el-tabs v-model="tab">
      <el-tab-pane label="领取记录" name="claims">
        <div class="toolbar">
          <el-input v-model="cq.user" placeholder="号码/昵称" size="small" clearable style="width:160px" @keyup.enter.native="search" />
          <el-date-picker v-model="cq.date" type="date" value-format="yyyy-MM-dd" placeholder="日期" size="small" style="width:140px" />
          <el-button size="small" type="primary" icon="el-icon-search" @click="search">查询</el-button>
          <el-button size="small" @click="reset">重置</el-button>
          <span class="muted">合计：{{ cSum | num }} G币</span>
        </div>
        <el-table :data="claims" v-loading="loading" stripe>
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column label="用户" min-width="150">
            <template slot-scope="{row}">
              <span :style="{ color: row.color || '' }">{{ row.nickname }}</span>
              <span class="muted">（{{ row.username }}）</span>
            </template>
          </el-table-column>
          <el-table-column prop="amount" label="领取金额(G币)" width="140" align="right">
            <template slot-scope="{row}">{{ row.amount | num }}</template>
          </el-table-column>
          <el-table-column label="领取时间" width="170">
            <template slot-scope="{row}">{{ fmt(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="90" align="center">
            <template slot-scope="{row}">
              <el-button size="mini" type="danger" plain @click="delClaim(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="捐献记录" name="donations">
        <div class="toolbar">
          <el-input v-model="dq.user" placeholder="号码/昵称" size="small" clearable style="width:160px" @keyup.enter.native="search" />
          <el-date-picker v-model="dq.date" type="date" value-format="yyyy-MM-dd" placeholder="日期" size="small" style="width:140px" />
          <el-button size="small" type="primary" icon="el-icon-search" @click="search">查询</el-button>
          <el-button size="small" @click="reset">重置</el-button>
          <span class="muted">合计：{{ dSum | num }} G币</span>
        </div>
        <el-table :data="donations" v-loading="loading" stripe>
          <el-table-column prop="id" label="ID" width="70" />
          <el-table-column label="用户" min-width="150">
            <template slot-scope="{row}">
              <span :style="{ color: row.color || '' }">{{ row.nickname }}</span>
              <span class="muted">（{{ row.username }}）</span>
            </template>
          </el-table-column>
          <el-table-column prop="amount" label="捐献金额(G币)" width="140" align="right">
            <template slot-scope="{row}">{{ row.amount | num }}</template>
          </el-table-column>
          <el-table-column prop="points" label="获得财富值" width="110" align="right" />
          <el-table-column label="捐献时间" width="170">
            <template slot-scope="{row}">{{ fmt(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="90" align="center">
            <template slot-scope="{row}">
              <el-button size="mini" type="danger" plain @click="delDonate(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page"
                   @current-change="p => { page = p; load() }" style="margin-top:14px;text-align:right" />
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminWelfare',
  filters: {
    num (v) { return Number(v || 0).toLocaleString() }
  },
  data () {
    return {
      stats: { pool: 0, today_out: 0, today_in: 0, total_out: 0, total_in: 0, claim_count: 0, donate_count: 0 },
      tab: 'claims',
      claims: [], donations: [], cSum: 0, dSum: 0,
      total: 0, page: 1, size: 10, loading: false,
      cq: { user: '', date: '' },
      dq: { user: '', date: '' },
      poolDelta: 100000,
      poolSet: null,
      createForm: { username: '', byNick: false, amount: 10000 }
    }
  },
  watch: {
    tab () { this.page = 1; this.load() }
  },
  mounted () { this.loadAll() },
  activated () { this.loadAll() },
  methods: {
    loadAll () { this.loadStats(); this.load() },
    loadStats () {
      api.get('/admin/welfare').then(r => {
        if (r.code === 0) this.stats = r.data
        else this.$message.error(r.msg)
      }).catch(() => {})
    },
    load () {
      this.loading = true
      const q = this.tab === 'claims' ? this.cq : this.dq
      const params = { page: this.page, size: this.size }
      if (q.user) params.user = q.user
      if (q.date) params.date = q.date
      const url = this.tab === 'claims' ? '/admin/welfare/claims' : '/admin/welfare/donations'
      api.get(url, { params }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.total = r.data.total
          if (this.tab === 'claims') { this.claims = r.data.list; this.cSum = r.data.sum } else { this.donations = r.data.list; this.dSum = r.data.sum }
        } else this.$message.error(r.msg)
      }).catch(() => { this.loading = false })
    },
    setPool () {
      if (this.poolSet === null || this.poolSet < 0) { this.$message.warning('请输入池金额'); return }
      api.post('/admin/welfare/pool', { set: this.poolSet }).then(r => {
        if (r.code === 0) { this.$message.success('已设置为 ' + Number(r.data.pool).toLocaleString() + ' G币'); this.loadNames() } else this.$message.error(r.msg)
      })
    },
    search () { this.page = 1; this.load() },
    reset () {
      this.cq = { user: '', date: '' }
      this.dq = { user: '', date: '' }
      this.search()
    },
    adjustPool () {
      if (!this.poolDelta) { this.$message.warning('请输入调整金额'); return }
      api.post('/admin/welfare/pool', { delta: this.poolDelta }).then(r => {
        if (r.code === 0) { this.$message.success('已调整，当前福利池 ' + Number(r.data.pool).toLocaleString() + ' G币'); this.loadNames() } else this.$message.error(r.msg)
      })
    },
    createClaim () {
      const f = this.createForm
      if (!f.username) { this.$message.warning('请填写家园号码或昵称'); return }
      if (!f.amount || f.amount < 1) { this.$message.warning('金额需大于 0'); return }
      const payload = f.byNick ? { nickname: f.username, amount: f.amount } : { username: f.username, amount: f.amount }
      api.post('/admin/welfare/claims', payload).then(r => {
        if (r.code === 0) { this.$message.success('已补发'); this.tab = 'claims'; this.load() } else this.$message.error(r.msg)
      })
    },
    delClaim (row) {
      this.$confirm(`作废 ${row.nickname} 的领取记录（金额 ${row.amount}，将扣回其 G币，余额不足则不扣）？`, '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/welfare/claims/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    delDonate (row) {
      this.$confirm(`删除 ${row.nickname} 的捐献记录？将从福利池扣回 ${row.amount} G币并扣减财富值。`, '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/welfare/donations/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' },
    loadNames () { this.loadStats(); this.load() }
  }
}
</script>

<style scoped>
.toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 10px; }
.toolbar .grow { flex: 1; }
.muted { color: #999; }
.stats { display: flex; flex-wrap: wrap; gap: 12px; }
.stat { min-width: 130px; padding: 8px 12px; background: #f7f9fc; border-radius: 4px; }
.stat-label { font-size: 12px; color: #888; margin-bottom: 4px; }
.stat-num { font-size: 16px; font-weight: bold; color: #333; }
</style>
