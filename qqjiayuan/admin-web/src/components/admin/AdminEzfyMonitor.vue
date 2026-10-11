<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-radio-group v-model="preset" size="small" @change="page = 1; load()">
          <el-radio-button label="today">今日</el-radio-button>
          <el-radio-button label="3d">近3天</el-radio-button>
          <el-radio-button label="7d">近7天</el-radio-button>
          <el-radio-button label="custom">自定义</el-radio-button>
        </el-radio-group>
        <el-date-picker v-if="preset === 'custom'" v-model="range" type="daterange"
                        value-format="yyyy-MM-dd" range-separator="至" start-placeholder="开始日期"
                        end-placeholder="结束日期" style="width:260px; margin-left:8px" clearable />
        <el-input v-model="word" placeholder="玩家ID / 账号 / 昵称" clearable style="width:200px; margin-left:8px"
                  @keyup.enter.native="page = 1; load()" />
        <el-select v-model="sort" placeholder="排序" style="width:120px; margin-left:8px" @change="page = 1; load()">
          <el-option label="合计" value="total" />
          <el-option label="造兵" value="train" />
          <el-option label="征服" value="conquer" />
          <el-option label="掠夺" value="raid" />
          <el-option label="召集" value="convene" />
        </el-select>
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
      </div>

      <el-row :gutter="14" v-if="totals" v-loading="loading" style="margin-bottom:14px">
        <el-col :span="5">
          <div class="stat-card">
            <div class="stat-num">{{ fmtBig(totals.train) }}</div>
            <div class="stat-label">造兵次数</div>
          </div>
        </el-col>
        <el-col :span="5">
          <div class="stat-card">
            <div class="stat-num">{{ fmtBig(totals.conquer) }}</div>
            <div class="stat-label">征服次数</div>
          </div>
        </el-col>
        <el-col :span="5">
          <div class="stat-card">
            <div class="stat-num">{{ fmtBig(totals.raid) }}</div>
            <div class="stat-label">掠夺次数</div>
          </div>
        </el-col>
        <el-col :span="5">
          <div class="stat-card stat-money">
            <div class="stat-num">{{ fmtBig(totals.convene) }}</div>
            <div class="stat-label">召集人口<span class="stat-sub"></span></div>
          </div>
        </el-col>
        <el-col :span="4">
          <div class="stat-card">
            <div class="stat-num">{{ fmtBig(totals.players) }}</div>
            <div class="stat-label">参与玩家</div>
          </div>
        </el-col>
      </el-row>

      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column label="归属账号" width="100" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ row.user_id }}</span></template>
        </el-table-column>
        <el-table-column label="玩家" min-width="130" show-overflow-tooltip>
          <template slot-scope="{row}">
            <span class="td-main">{{ row.player_name }}</span>
            <span class="td-gray">({{ row.account }})</span>
          </template>
        </el-table-column>
        <el-table-column prop="train" label="造兵" width="90" align="center" sortable="custom"
                        @sort-change="onSort" />
        <el-table-column prop="conquer" label="征服" width="90" align="center" sortable="custom"
                        @sort-change="onSort" />
        <el-table-column prop="raid" label="掠夺" width="90" align="center" sortable="custom"
                        @sort-change="onSort" />
        <el-table-column prop="convene" label="召集" width="90" align="center" sortable="custom"
                        @sort-change="onSort" />
        <el-table-column prop="total" label="合计" width="110" align="center" sortable="custom"
                        @sort-change="onSort">
          <template slot-scope="{row}"><b>{{ fmtBig(row.total) }}</b></template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, sizes, prev, pager, next" :total="total"
                     :page-size="size" :current-page="page" :page-sizes="[10, 20, 50]"
                     @current-change="p => { page = p; load() }"
                     @size-change="s => { size = s; page = 1; load() }" />
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminEzfyMonitor',
  data () {
    return {
      preset: 'today',
      range: [],
      word: '',
      sort: 'total',
      list: [],
      totals: null,
      total: 0,
      page: 1,
      size: 10,
      loading: false
    }
  },
  mounted () { this.load() },
  methods: {
    // 预设 → 起止日期（含当天）
    dateRange () {
      if (this.preset === 'custom') {
        return this.range && this.range.length === 2 ? this.range : [null, null]
      }
      const today = new Date()
      const fmt = d => d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0') + '-' + String(d.getDate()).padStart(2, '0')
      const end = fmt(today)
      let start
      if (this.preset === '3d') {
        const d = new Date(today); d.setDate(d.getDate() - 2); start = fmt(d)
      } else if (this.preset === '7d') {
        const d = new Date(today); d.setDate(d.getDate() - 6); start = fmt(d)
      } else {
        start = end
      }
      return [start, end]
    },
    onSort ({ prop, order }) {
      if (order === 'ascending' || order === 'descending') {
        this.sort = prop
        this.page = 1
        this.load()
      }
    },
    load () {
      const [start, end] = this.dateRange()
      if (!start || !end) { this.$message.warning('请选择时间范围'); return }
      this.loading = true
      api.get('/admin/ezfy-monitor', {
        params: { start, end, page: this.page, size: this.size, word: this.word, sort: this.sort }
      }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list || []
          this.totals = r.data.totals || { train: 0, conquer: 0, raid: 0, convene: 0, players: 0 }
          this.total = this.totals.players
        } else this.$message.error(r.msg)
      }).catch(() => { this.loading = false })
    },
    fmtBig (n) {
      n = Number(n || 0)
      if (n >= 1e8) return (n / 1e8).toFixed(1) + '亿'
      if (n >= 1e4) return (n / 1e4).toFixed(1) + '万'
      return String(n)
    }
  }
}
</script>
