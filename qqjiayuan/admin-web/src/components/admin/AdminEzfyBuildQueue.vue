<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <span>建筑队列（建造中 / 升级中）</span>
        <el-tag size="mini" type="warning">共 {{ total }} 项在队列</el-tag>
      </div>
      <div class="toolbar">
        <el-input v-model="word" placeholder="城名 / 城池ID" clearable style="width:200px"
                  @keyup.enter.native="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <el-button type="success" icon="el-icon-s-claim" @click="finishAll">一键完成全部</el-button>
        <el-button type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe border max-height="620">
        <el-table-column prop="id" label="ID" width="60" align="center" />
        <el-table-column prop="cfg_name" label="建筑" min-width="105" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-main">{{ row.cfg_name || ('#' + row.building_id) }}</span></template>
        </el-table-column>
        <el-table-column label="等级" width="95" align="center">
          <template slot-scope="{row}">
            <span class="lv">Lv.</span>{{ row.level }} <span class="td-muted">→ {{ row.level + 1 }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" type="warning">{{ row.status_txt }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="剩余时间" width="100" align="center">
          <template slot-scope="{row}">
            <span :class="row.remain_sec <= 0 ? 'td-blue' : ''">{{ fmtRemain(row.remain_sec) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="模式" width="95" align="center">
          <template slot-scope="{row}">
            <el-tag v-if="row.chain_build === 1" size="mini" type="danger">一键满级连锁</el-tag>
            <span v-else class="td-sub">单次</span>
          </template>
        </el-table-column>
        <el-table-column prop="city_name" label="所属城池" width="105" show-overflow-tooltip />
        <el-table-column prop="owner_name" label="归属玩家" width="100" show-overflow-tooltip />
        <el-table-column prop="home_num" label="家园号" width="80" align="center" />
        <el-table-column label="操作" width="235" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="success" plain @click="finish(row)">完成</el-button>
            <el-button size="mini" type="primary" plain @click="openSpeed(row)">加速</el-button>
            <el-button size="mini" type="warning" plain @click="cancel(row)">取消</el-button>
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

    <el-dialog title="加速建造" :visible.sync="speedDlg" width="420px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="建筑">
          <span class="td-main">{{ speedRow.cfg_name }}（{{ speedRow.city_name }}）</span>
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
  name: 'AdminEzfyBuildQueue',
  data () {
    return {
      list: [], total: 0, page: 1, size: 10, loading: false, word: '',
      speedDlg: false, saving: false, speedRow: {}, speedMinutes: 10
    }
  },
  mounted () { this.load() },
  methods: {
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
      api.get('/admin/ezfy-build-queue', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    finish (row) {
      api.post('/admin/ezfy-build-queue/' + row.id + '/finish').then(r => {
        if (r.code === 0) { this.$message.success(r.data.msg || '已完成'); this.load() } else this.$message.error(r.msg)
      })
    },
    openSpeed (row) {
      this.speedRow = row
      this.speedMinutes = 10
      this.speedDlg = true
    },
    doSpeed () {
      this.saving = true
      api.post('/admin/ezfy-build-queue/' + this.speedRow.id + '/speed', { minutes: this.speedMinutes }).then(r => {
        this.saving = false
        if (r.code === 0) { this.speedDlg = false; this.$message.success(r.data.msg || '已加速'); this.load() }
        else this.$message.error(r.msg)
      })
    },
    cancel (row) {
      this.$confirm('取消后该建筑的建造/升级进度将丢失（等级不变），确认取消？', '提示', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-build-queue/' + row.id + '/cancel').then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已取消'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    finishAll () {
      if (!this.total) { this.$message.info('当前没有建造中的建筑'); return }
      this.$confirm('将立即完成队列中全部 ' + this.total + ' 个建筑（每个完成一步），确认继续？', '一键完成', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-build-queue/finish-all').then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已全部完成'); this.load() } else this.$message.error(r.msg)
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
