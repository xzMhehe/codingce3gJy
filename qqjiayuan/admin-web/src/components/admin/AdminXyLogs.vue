<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-radio-group v-model="kind" @change="page = 1; load()">
          <el-radio-button label="battles">战斗流水</el-radio-button>
          <el-radio-button label="wallet">货币流水</el-radio-button>
        </el-radio-group>
        <el-input v-model="word" :placeholder="kind === 'battles' ? '玩家ID / 对手名搜索' : '玩家ID / 事由搜索'"
                  clearable style="width:220px" @keyup.enter.native="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <el-button type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>

      <!-- 战斗流水 -->
      <el-table v-if="kind === 'battles'" :data="list" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="80" align="center" />
        <el-table-column prop="player_name" label="玩家" width="120" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-main">{{ row.player_name || '—' }}</span></template>
        </el-table-column>
        <el-table-column prop="enemy_name" label="对手" min-width="130" show-overflow-tooltip />
        <el-table-column label="类型" width="90" align="center">
          <template slot-scope="{row}">{{ typeNames[row.type] || row.type }}</template>
        </el-table-column>
        <el-table-column label="结果" width="80" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="resultTags[row.result] || 'info'">{{ resultNames[row.result] || row.result }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="round" label="回合" width="70" align="center" />
        <el-table-column prop="exp" label="经验" width="80" align="center" />
        <el-table-column prop="money" label="银两" width="90" align="center" />
        <el-table-column label="掉落" min-width="150" show-overflow-tooltip>
          <template slot-scope="{row}">{{ lootText(row.loot) }}</template>
        </el-table-column>
        <el-table-column label="时间" width="160" align="center">
          <template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="80" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="delBattle(row)" />
          </template>
        </el-table-column>
      </el-table>

      <!-- 货币流水 -->
      <el-table v-else :data="list" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="80" align="center" />
        <el-table-column prop="player_name" label="玩家" width="130" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-main">{{ row.player_name || '—' }}</span></template>
        </el-table-column>
        <el-table-column label="货币" width="90" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="curTags[row.currency] || 'info'">{{ curNames[row.currency] || row.currency }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="变动" width="110" align="center">
          <template slot-scope="{row}">
            <span :class="row.amount >= 0 ? 'td-green' : 'td-red'">{{ row.amount >= 0 ? '+' : '' }}{{ row.amount }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="balance" label="余额" width="110" align="center" />
        <el-table-column prop="reason" label="事由" min-width="160" show-overflow-tooltip />
        <el-table-column label="时间" width="160" align="center">
          <template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template>
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
  name: 'AdminXyLogs',
  data () {
    return {
      kind: 'battles', word: '',
      list: [], total: 0, page: 1, size: 10, loading: false,
      typeNames: { hunt: '打怪', boss: 'BOSS', pvp: '切磋' },
      resultNames: { 2: '胜利', 3: '失败', 4: '逃跑' },
      resultTags: { 2: 'success', 3: 'danger', 4: 'info' },
      curNames: { money: '银两', beans: '金豆', bank: '存款' },
      curTags: { money: 'primary', beans: 'warning', bank: 'success' }
    }
  },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      const url = this.kind === 'battles' ? '/admin/xy-battles' : '/admin/xy-wallet'
      api.get(url, { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    lootText (loot) {
      if (!loot || loot === 'null' || loot === '[]') return '—'
      try {
        const arr = JSON.parse(loot)
        if (!Array.isArray(arr) || !arr.length) return '—'
        return arr.map(i => (i.name || i.n || ('#' + (i.id || '?'))) + '×' + (i.count || i.c || 1)).join('、')
      } catch (e) { return loot }
    },
    fmtTime (t) {
      if (!t) return '—'
      return String(t).replace('T', ' ').slice(0, 19)
    },
    delBattle (row) {
      this.$confirm('删除战斗流水 #' + row.id + '？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/xy-battles/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.td-green { color: #67c23a; font-weight: 600; }
.td-red { color: #f56c6c; font-weight: 600; }
</style>
