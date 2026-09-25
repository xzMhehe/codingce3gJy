<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <!-- ★ 2026-09-25 用户要求：新增「军团宣战维护」页面（军团对军团宣战） -->
        <span>二战风云 · 军团宣战（待生效 / 交战中 / 已结束）</span>
        <el-button size="mini" type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>

      <div class="toolbar">
        <el-input v-model="keyword" placeholder="按军团名搜索" clearable style="width:200px"
                  @keyup.enter.native="page = 1; load()" />
        <el-select v-model="status" style="width:150px" @change="page = 1; load()">
          <el-option :value="0" label="全部状态" />
          <el-option :value="1" label="待生效" />
          <el-option :value="2" label="交战中" />
          <el-option :value="3" label="已结束" />
        </el-select>
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <!-- 后台代宣战：管理员替两个军团直接发起宣战 -->
        <el-button type="success" icon="el-icon-plus" @click="openAdd">后台代宣战</el-button>
        <div class="grow" />
        <!-- ★ 2026-09-25 用户要求「军团宣战维护也加个按钮一键生效」：
             把所有「待生效」的军团宣战立刻推进到「交战中」（与个人宣战的同款按钮同口径） -->
        <el-button type="warning" plain icon="el-icon-alarm-clock" :loading="acting" @click="effectAll">一键生效全部</el-button>
        <!-- 危险操作：一次结束所有进行中（待生效 / 交战中）的军团宣战 -->
        <el-button type="danger" plain icon="el-icon-circle-close" :loading="acting" @click="finishAll">全部结束进行中宣战</el-button>
      </div>

      <div class="td-sub" style="margin:4px 0 10px">
        规则：军团宣战后 <b>12</b> 小时生效，宣战后 <b>48</b> 小时整场结束；生效期间双方军团成员可互相掠夺/征服并获得军团战绩。
        「一键生效」可跳过等待直接开战，「强制结束」立即停战（双方成员恢复和平）。
      </div>

      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="70" align="center" />
        <el-table-column label="宣战方军团" min-width="160" show-overflow-tooltip>
          <template slot-scope="{row}">
            <span class="td-main">{{ row.atk_corps_name || '—' }}</span>
            <span class="td-sub"> #{{ row.atk_corps_id }}</span>
          </template>
        </el-table-column>
        <el-table-column label="应战方军团" min-width="160" show-overflow-tooltip>
          <template slot-scope="{row}">
            <span class="td-main">{{ row.def_corps_name || '—' }}</span>
            <span class="td-sub"> #{{ row.def_corps_id }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110" align="center">
          <template slot-scope="{row}">
            <el-tag :type="statusTag(row.status)" size="mini">{{ row.status_name || statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="声明时间" width="150" align="center">
          <template slot-scope="{row}"><span class="td-sub">{{ fmtTime(row.declare_time) }}</span></template>
        </el-table-column>
        <el-table-column label="生效时间" width="150" align="center">
          <template slot-scope="{row}"><span class="td-sub">{{ fmtTime(row.effect_time) }}</span></template>
        </el-table-column>
        <el-table-column label="结束时间" width="150" align="center">
          <template slot-scope="{row}"><span class="td-sub">{{ fmtTime(row.end_time) }}</span></template>
        </el-table-column>
        <el-table-column label="宣战方战绩" width="100" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ fmtPoint(row.atk_point) }}</span></template>
        </el-table-column>
        <el-table-column label="应战方战绩" width="100" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ fmtPoint(row.def_point) }}</span></template>
        </el-table-column>
        <el-table-column label="剩余" width="110" align="center">
          <template slot-scope="{row}"><span class="td-sub">{{ fmtRemain(row) }}</span></template>
        </el-table-column>
        <el-table-column label="操作" width="210" align="center" fixed="right">
          <template slot-scope="{row}">
            <!-- 「待生效」才可一键生效（跳过 12 小时等待直接开战） -->
            <el-button v-if="row.status === 1" size="mini" type="warning" plain
                       icon="el-icon-alarm-clock" title="一键生效（跳过等待，立即开战）" @click="doEffect(row)" />
            <!-- 进行中（待生效 / 交战中）才可强制结束 -->
            <el-button v-if="row.status !== 3" size="mini" type="success" plain
                       icon="el-icon-circle-close" title="强制结束" @click="doFinish(row)" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete"
                       title="删除记录" @click="remove(row)" />
          </template>
        </el-table-column>
      </el-table>

      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ total }}</b> 条 · 每页 {{ size }} 条</div>
        <el-pagination v-show="total > 0" small background layout="sizes, prev, pager, next, jumper" :total="total" :page-size="size"
                       :current-page="page" :page-sizes="[10, 20, 50, 100]"
                       @current-change="p => { page = p; load() }"
                       @size-change="s => { size = s; page = 1; load() }" />
      </div>
    </el-card>

    <!-- 后台代宣战 -->
    <el-dialog title="后台代宣战" :visible.sync="addDlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="120px" size="small">
        <el-form-item label="宣战方军团" required>
          <el-select v-model="form.atk_corps_id" filterable remote reserve-keyword
                     :remote-method="q => searchCorps('atk', q)" :loading="corpsLoading.atk"
                     placeholder="输入军团名搜索" style="width:100%">
            <el-option v-for="c in corpsOpts.atk" :key="'a' + c.id"
                       :label="(c.name || '军团') + '（ID ' + c.id + '）'" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="应战方军团" required>
          <el-select v-model="form.def_corps_id" filterable remote reserve-keyword
                     :remote-method="q => searchCorps('def', q)" :loading="corpsLoading.def"
                     placeholder="输入军团名搜索" style="width:100%">
            <el-option v-for="c in corpsOpts.def" :key="'d' + c.id"
                       :label="(c.name || '军团') + '（ID ' + c.id + '）'" :value="c.id" />
          </el-select>
        </el-form-item>
        <div class="td-sub">宣战生效/持续时间以游戏内规则为准，后台代宣战只负责发起。</div>
      </el-form>
      <div slot="footer">
        <el-button @click="addDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doAdd">发 起 宣 战</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminEzfyCorpsWars',
  data () {
    return {
      list: [], total: 0, page: 1, size: 10, keyword: '', status: 0,
      loading: false, saving: false, acting: false,
      addDlg: false,
      form: { atk_corps_id: null, def_corps_id: null },
      // 两个军团选择框各自维护候选列表，避免一个搜索后另一个已选项丢失名称
      corpsOpts: { atk: [], def: [] },
      corpsLoading: { atk: false, def: false }
    }
  },
  mounted () { this.load() },
  methods: {
    statusTag (s) {
      if (s === 1) return 'warning'
      if (s === 2) return 'danger'
      return 'info'
    },
    statusText (s) {
      if (s === 1) return '待生效'
      if (s === 2) return '交战中'
      if (s === 3) return '已结束'
      return '—'
    },
    // 毫秒时间戳 → YYYY-MM-DD HH:mm
    fmtTime (t) {
      if (!t) return '—'
      const d = new Date(t)
      if (isNaN(d.getTime())) return String(t)
      const p = n => (n < 10 ? '0' + n : '' + n)
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' +
        p(d.getHours()) + ':' + p(d.getMinutes())
    },
    fmtPoint (v) {
      return (v === null || v === undefined || v === '') ? '—' : v
    },
    // 剩余时长（小时），已结束不显示
    fmtRemain (row) {
      if (row.status === 3) return '—'
      const h = row.remaining_h
      if (h === null || h === undefined || h === '') return '—'
      return h + ' 小时'
    },
    load () {
      this.loading = true
      api.get('/admin/ezfy-corps-wars', {
        params: { page: this.page, size: this.size, status: this.status, keyword: this.keyword }
      }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list || []
          this.total = r.data.total || 0
          if (r.data.page) this.page = r.data.page
        } else this.$message.error(r.msg)
      }).catch(() => { this.loading = false })
    },
    // 复用现有军团列表接口 /admin/ezfy-corps 做下拉候选（后端 size 上限 100，故用关键字远程搜索）
    searchCorps (which, q) {
      this.corpsLoading[which] = true
      api.get('/admin/ezfy-corps', { params: { page: 1, size: 100, word: q || '' } }).then(r => {
        this.corpsLoading[which] = false
        if (r.code === 0) this.corpsOpts[which] = r.data.list || []
        else this.$message.error(r.msg)
      }).catch(() => { this.corpsLoading[which] = false })
    },
    openAdd () {
      this.form = { atk_corps_id: null, def_corps_id: null }
      this.corpsOpts = { atk: [], def: [] }
      this.searchCorps('atk', '')
      this.searchCorps('def', '')
      this.addDlg = true
    },
    doAdd () {
      if (!this.form.atk_corps_id || !this.form.def_corps_id) { this.$message.error('请选择宣战方与应战方军团'); return }
      if (this.form.atk_corps_id === this.form.def_corps_id) { this.$message.error('宣战方与应战方不能是同一个军团'); return }
      this.saving = true
      api.post('/admin/ezfy-corps-wars', {
        atk_corps_id: this.form.atk_corps_id,
        def_corps_id: this.form.def_corps_id
      }).then(r => {
        this.saving = false
        if (r.code === 0) { this.addDlg = false; this.$message.success(r.data.msg || '已发起宣战'); this.load() }
        else this.$message.error(r.msg)
      }).catch(() => { this.saving = false })
    },
    // ★ 2026-09-25 用户要求「一键生效」：单条生效（跳过 12 小时等待，立即开战）
    doEffect (row) {
      this.$confirm('让「' + (row.atk_corps_name || row.atk_corps_id) + ' → ' +
        (row.def_corps_name || row.def_corps_id) + '」立即进入交战状态（跳过等待）？', '一键生效', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-corps-wars/' + row.id + '/effect').then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已生效'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ★ 2026-09-25 用户要求「一键生效」：全部待生效的军团宣战一次性生效
    effectAll () {
      this.$confirm('让当前所有「待生效」的军团宣战立即生效（双方成员马上可以互相掠夺/征服）？',
        '一键生效全部', { type: 'warning' }).then(() => {
        this.acting = true
        api.post('/admin/ezfy-corps-wars/effect-all').then(r => {
          this.acting = false
          if (r.code === 0) { this.$message.success(r.data.msg || '已生效'); this.load() } else this.$message.error(r.msg)
        }).catch(() => { this.acting = false })
      }).catch(() => {})
    },
    doFinish (row) {
      this.$confirm('强制结束「' + (row.atk_corps_name || row.atk_corps_id) + ' → ' +
        (row.def_corps_name || row.def_corps_id) + '」的军团战争？', '强制结束', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-corps-wars/' + row.id + '/finish').then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已结束'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    finishAll () {
      this.$confirm('结束当前全部「进行中」的军团宣战？所有相关军团将恢复和平。', '危险操作', { type: 'error' }).then(() => {
        this.acting = true
        api.post('/admin/ezfy-corps-wars/finish-all').then(r => {
          this.acting = false
          if (r.code === 0) { this.$message.success(r.data.msg || '已结束'); this.load() } else this.$message.error(r.msg)
        }).catch(() => { this.acting = false })
      }).catch(() => {})
    },
    remove (row) {
      this.$confirm('删除军团宣战记录 #' + row.id + '？（仅删除记录，不做其他处理）', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-corps-wars/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.card-head { display: flex; justify-content: space-between; align-items: center; }
</style>