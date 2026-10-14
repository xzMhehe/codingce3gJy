<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <span>风云榜（每周玩家对战战绩）</span>
        <el-button size="mini" type="primary" plain icon="el-icon-refresh" @click="load()">刷新</el-button>
      </div>
      <!-- ★ 2026-10-14 风云榜：按自然周（周一~周日）统计玩家 vs 玩家战斗，任一周可查 -->
      <div class="toolbar">
        <el-select v-model="week" placeholder="选择统计周" style="width:220px" @change="load()">
          <el-option v-for="w in weeks" :key="w" :label="w" :value="w" />
        </el-select>
        <span class="head-hint" style="margin-left:8px">
          榜单按「积分」降序（同积分按灭敌多、胜场多、战损少）。昵称/军团为写榜时快照。
        </span>
      </div>
      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column label="名次" width="70" align="center">
          <template slot-scope="{row}">
            <span class="td-rank" :class="row.rank <= 3 ? 'top' : ''">{{ row.rank }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="玩家" min-width="110" show-overflow-tooltip />
        <el-table-column prop="corps" label="军团" min-width="110" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.corps || '—' }}</template>
        </el-table-column>
        <el-table-column prop="points" label="积分" width="90" align="right">
          <template slot-scope="{row}"><b>{{ row.points.toLocaleString() }}</b></template>
        </el-table-column>
        <el-table-column prop="kills" label="灭敌" width="100" align="right">
          <template slot-scope="{row}">{{ row.kills.toLocaleString() }}</template>
        </el-table-column>
        <el-table-column prop="losses" label="战损比" width="90" align="center">
          <template slot-scope="{row}">
            <el-tooltip :content="'灭敌 ' + row.kills.toLocaleString() + ' ÷ 战损 ' + row.losses.toLocaleString()" placement="top">
              <span>{{ row.losses > 0 ? Math.round((row.kills / row.losses) * 10) / 10 : (row.kills > 0 ? '∞' : '—') }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="battles" label="参战" width="80" align="center" />
        <el-table-column prop="wins" label="胜场" width="80" align="center" />
        <el-table-column label="胜率" width="80" align="center">
          <template slot-scope="{row}">{{ (Math.round(row.win_rate * 10) / 10) }}%</template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

// ★ 2026-10-14 风云榜（每周玩家对战战绩周榜）：管理端查询任一周的玩家对战排名
export default {
  name: 'AdminEzfyPvpRank',
  data () {
    return {
      loading: false,
      week: '',
      weeks: [],
      list: [],
    }
  },
  mounted () {
    this.load()
  },
  methods: {
    load () {
      this.loading = true
      const params = {}
      if (this.week) params.week = this.week
      api.get('/admin/ezfy-pvp-rank', { params }).then(r => {
        if (r.code === 0) {
          this.weeks = r.data.weeks || []
          this.week = r.data.week
          this.list = r.data.ranks || []
          if (this.weeks.indexOf(this.week) < 0 && this.week) this.weeks.unshift(this.week)
        }
      }).finally(() => { this.loading = false })
    }
  }
}
</script>

<style>
  .td-rank { font-weight: 600; color: #909399; }
  .td-rank.top { color: #e6a23c; }
</style>