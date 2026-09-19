<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="城名 / 城池ID" clearable style="width:200px"
                  @keyup.enter.native="page = 1; load()" />
        <el-select v-model="status" style="width:140px" @change="page = 1; load()">
          <el-option label="全部状态" :value="-1" />
          <el-option label="训练中" :value="0" />
          <el-option label="待领取" :value="1" />
          <el-option label="已完成" :value="2" />
        </el-select>
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <el-button type="success" icon="el-icon-plus" @click="openCreate">新增征兵</el-button>
        <el-button type="warning" icon="el-icon-finished" :loading="finishingAll" @click="finishAll">一键完成</el-button>
        <el-button type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe border max-height="620">
        <el-table-column prop="id" label="ID" width="60" align="center" />
        <el-table-column prop="cfg_name" label="兵种" min-width="105" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-main">{{ row.cfg_name || ('#' + row.troop_id) }}</span></template>
        </el-table-column>
        <el-table-column label="数量" width="100" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ fmtN(row.count) }}</span></template>
        </el-table-column>
        <el-table-column label="占用人口" width="90" align="center">
          <template slot-scope="{row}">{{ fmtN(row.pop_need) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="85" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="statusTag(row.status)">{{ row.status_txt }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="剩余时间" width="100" align="center">
          <template slot-scope="{row}">
            <span :class="row.status === 0 && row.remain_sec <= 0 ? 'td-blue' : ''">{{ fmtRemain(row.remain_sec) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="city_name" label="所属城池" width="105" show-overflow-tooltip />
        <el-table-column prop="owner_name" label="归属玩家" width="100" show-overflow-tooltip />
        <el-table-column prop="home_num" label="家园号" width="80" align="center" />
        <el-table-column label="操作" width="235" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="success" plain :disabled="row.status === 2" @click="finish(row)">完成</el-button>
            <el-button size="mini" type="primary" plain :disabled="row.status === 2" @click="openSpeed(row)">加速</el-button>
            <el-button size="mini" type="danger" plain @click="del(row)">取消</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ total }}</b> 条 · 每页 {{ size }} 条</div>
        <el-pagination small background layout="sizes, prev, pager, next, jumper" :total="total" :page-size="size"
                       :current-page="page" :page-sizes="[10, 20, 50, 100]"
                       @current-change="p => { page = p; load() }"
                       @size-change="s => { size = s; page = 1; load() }" />
      </div>
    </el-card>

    <!-- 新增征兵 -->
    <el-dialog title="新增征兵任务" :visible.sync="createDlg" width="540px" :close-on-click-modal="false">
      <el-form label-width="120px" size="small">
        <el-form-item label="城池ID" required>
          <el-input-number v-model.number="createForm.city_id" :min="1" controls-position="right" />
        </el-form-item>
        <el-form-item label="兵种" required>
          <el-select v-model="createForm.troop_id" filterable style="width:280px">
            <el-option v-for="b in cfgs" :key="b.id" :label="b.id + ' · ' + b.name + '（' + b.type_name + '）'" :value="b.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="数量" required>
          <el-input-number v-model.number="createForm.count" :min="1" controls-position="right" />
        </el-form-item>
        <el-form-item label="耗时（秒）">
          <el-input-number v-model.number="createForm.seconds" :min="0" controls-position="right" />
          <span class="td-sub" style="margin-left:8px">0 = 按兵种配置自动计算</span>
        </el-form-item>
        <el-form-item label="立即完成">
          <el-switch v-model="createForm.instant" />
          <span class="td-sub" style="margin-left:8px">开启后跳过队列，兵力直接入库</span>
        </el-form-item>
      </el-form>
      <em>提示：管理端新增征兵不扣除资源与人口</em>
      <div slot="footer">
        <el-button @click="createDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doCreate">确 定</el-button>
      </div>
    </el-dialog>

    <!-- 加速 -->
    <el-dialog title="征兵加速" :visible.sync="speedDlg" width="420px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="兵种">
          <span class="td-main">{{ speedRow.cfg_name }} × {{ speedRow.count }}（{{ speedRow.city_name }}）</span>
        </el-form-item>
        <el-form-item label="加速分钟">
          <el-input-number v-model.number="speedMinutes" :min="1" :max="1440" controls-position="right" />
        </el-form-item>
        <el-form-item>
          <el-button size="mini" plain @click="speedMinutes = 10">10 分钟</el-button>
          <el-button size="mini" plain @click="speedMinutes = 60">1 小时</el-button>
          <el-button size="mini" plain @click="speedMinutes = 360">6 小时</el-button>
          <el-button size="mini" plain @click="speedMinutes = 1440">24 小时</el-button>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="speedDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doSpeed">加 速</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminEzfyRecruit',
  data () {
    return {
      list: [], total: 0, page: 1, size: 10, loading: false, word: '', status: -1,
      cfgs: [],
      createDlg: false, createForm: { city_id: 1, troop_id: 0, count: 100, seconds: 0, instant: false },
      speedDlg: false, speedRow: {}, speedMinutes: 10,
      finishingAll: false,
      saving: false
    }
  },
  mounted () { this.load(); this.loadCfgs() },
  methods: {
    fmtN (v) {
      if (v === null || v === undefined) return '—'
      return Number(v).toLocaleString()
    },
    statusTag (s) {
      return ({ 0: 'warning', 1: 'primary', 2: 'success' })[s] || 'info'
    },
    fmtRemain (sec) {
      if (sec <= 0) return '可完成'
      const h = Math.floor(sec / 3600)
      const m = Math.floor((sec % 3600) / 60)
      const s = sec % 60
      if (h > 0) return h + ' 时 ' + m + ' 分'
      if (m > 0) return m + ' 分 ' + s + ' 秒'
      return s + ' 秒'
    },
    load () {
      this.loading = true
      api.get('/admin/ezfy-train-queue', {
        params: { page: this.page, size: this.size, word: this.word, status: this.status }
      }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    loadCfgs () {
      api.get('/admin/ezfy-troops-cfg').then(r => {
        if (r.code === 0) this.cfgs = r.data.list
      })
    },
    openCreate () {
      this.createForm = { city_id: 1, troop_id: this.cfgs.length ? this.cfgs[0].id : 0, count: 100, seconds: 0, instant: false }
      this.createDlg = true
    },
    doCreate () {
      if (!this.createForm.troop_id) { this.$message.warning('请选择兵种'); return }
      this.saving = true
      api.post('/admin/ezfy-train-queue', this.createForm).then(r => {
        this.saving = false
        if (r.code === 0) { this.createDlg = false; this.$message.success(r.data.msg || '已入队'); this.load() }
        else this.$message.error(r.msg)
      })
    },
    finish (row) {
      api.post('/admin/ezfy-train-queue/' + row.id + '/finish').then(r => {
        if (r.code === 0) { this.$message.success(r.data.msg || '已完成'); this.load() } else this.$message.error(r.msg)
      })
    },
    // 一键完成全部训练队列（走游戏内结算，兵力正常入库）
    finishAll () {
      this.$confirm('将把全部「训练中」的队列一次性完成，兵力直接入库。确认执行？', '一键完成', { type: 'warning' }).then(() => {
        this.finishingAll = true
        api.post('/admin/ezfy-train-queue/finish-all').then(r => {
          this.finishingAll = false
          if (r.code === 0) { this.$message.success(r.data.msg || '已全部完成'); this.load() } else this.$message.error(r.msg)
        }).catch(() => { this.finishingAll = false })
      }).catch(() => {})
    },
    openSpeed (row) {
      this.speedRow = row
      this.speedMinutes = 10
      this.speedDlg = true
    },
    doSpeed () {
      this.saving = true
      api.post('/admin/ezfy-train-queue/' + this.speedRow.id + '/speed', { minutes: this.speedMinutes }).then(r => {
        this.saving = false
        if (r.code === 0) { this.speedDlg = false; this.$message.success(r.data.msg || '已加速'); this.load() }
        else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm('取消后该征兵任务将作废（兵力不会入库），确认取消？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-train-queue/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已取消'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
</style>
