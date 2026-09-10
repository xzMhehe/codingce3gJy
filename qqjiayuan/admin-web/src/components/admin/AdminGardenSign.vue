<template>
  <div class="farm-admin">
    <el-tabs v-model="tab" type="card" @tab-click="onTab">
      <!-- ============ 签到统计 ============ -->
      <el-tab-pane label="签到统计" name="stats">
        <div class="stat-row">
          <div class="stat-card s-blue">
            <div class="stat-ico"><i class="el-icon-edit-outline" /></div>
            <div class="stat-info">
              <div class="stat-num">{{ stats.today_cnt || 0 }}</div>
              <div class="stat-lab">今日签到</div>
            </div>
          </div>
          <div class="stat-card s-gray">
            <div class="stat-ico"><i class="el-icon-date" /></div>
            <div class="stat-info">
              <div class="stat-num">{{ stats.yesterday_cnt || 0 }}</div>
              <div class="stat-lab">昨日签到</div>
            </div>
          </div>
          <div class="stat-card s-purple">
            <div class="stat-ico"><i class="el-icon-calendar" /></div>
            <div class="stat-info">
              <div class="stat-num">{{ stats.week_cnt || 0 }}</div>
              <div class="stat-lab">近7天签到</div>
            </div>
          </div>
          <div class="stat-card s-orange">
            <div class="stat-ico"><i class="el-icon-trophy" /></div>
            <div class="stat-info">
              <div class="stat-num">{{ stats.full_round || 0 }}</div>
              <div class="stat-lab">今日签满7天</div>
            </div>
          </div>
          <div class="stat-card s-green">
            <div class="stat-ico"><i class="el-icon-finished" /></div>
            <div class="stat-info">
              <div class="stat-num">{{ stats.total_cnt || 0 }}</div>
              <div class="stat-lab">累计签到次数</div>
            </div>
          </div>
        </div>

        <el-card shadow="never" class="box">
          <div slot="header" class="box-title">今日奖励发放</div>
          <el-descriptions :column="4" border size="medium">
            <el-descriptions-item label="G币">{{ stats.coins_out || 0 }}</el-descriptions-item>
            <el-descriptions-item label="花园经验">{{ stats.exp_out || 0 }}</el-descriptions-item>
            <el-descriptions-item label="随机花种">{{ stats.seed_out || 0 }}</el-descriptions-item>
            <el-descriptions-item label="元宝">{{ stats.ingot_out || 0 }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-tab-pane>

      <!-- ============ 签到记录 ============ -->
      <el-tab-pane label="签到记录" name="logs">
        <el-card shadow="never" class="box">
          <div class="toolbar">
            <el-input v-model="word" placeholder="家园号 / 昵称搜索" clearable style="width:220px"
                      @keyup.enter.native="page = 1; loadLogs()" />
            <el-button type="primary" icon="el-icon-search" @click="page = 1; loadLogs()">查询</el-button>
            <div class="grow" />
          </div>
          <el-table :data="logs" v-loading="loading" stripe>
            <el-table-column prop="id" label="ID" width="70" />
            <el-table-column prop="uid" label="家园号" width="100" />
            <el-table-column prop="nickname" label="昵称" min-width="130" show-overflow-tooltip />
            <el-table-column prop="sign_date" label="签到日期" width="120" />
            <el-table-column label="星期" width="90" align="center">
              <template slot-scope="{row}">{{ weekName(row.week_day) }}</template>
            </el-table-column>
            <el-table-column label="连续第几天" width="110" align="center">
              <template slot-scope="{row}">
                <el-tag :type="row.day_no === 7 ? 'danger' : row.day_no >= 5 ? 'warning' : 'success'" size="mini">
                  第{{ row.day_no }}天
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
          <div class="pager">
            <el-pagination background layout="prev, pager, next, total" :total="total"
                           :page-size="size" :current-page.sync="page" @current-change="loadLogs" />
          </div>
        </el-card>
      </el-tab-pane>

      <!-- ============ 奖励配置 ============ -->
      <el-tab-pane label="奖励配置" name="rewards">
        <el-card shadow="never" class="box">
          <div class="toolbar">
            <span class="tip">连续第N天的签到奖励；第7天为大奖，签满后进入新一轮。保存后立即生效。</span>
            <div class="grow" />
          </div>
          <el-table :data="rewards" v-loading="rewardLoading" stripe border>
            <el-table-column label="连续天数" width="100" align="center">
              <template slot-scope="{row}">
                <el-tag :type="row.day === 7 ? 'danger' : 'success'" size="medium">第{{ row.day }}天</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="随机花种" width="160" align="center">
              <template slot-scope="{row}">
                <el-input-number v-model="row.seed_n" :min="0" :max="99" size="small" controls-position="right" />
              </template>
            </el-table-column>
            <el-table-column label="G币" width="180" align="center">
              <template slot-scope="{row}">
                <el-input-number v-model="row.coins" :min="0" :max="9999999" size="small" controls-position="right" />
              </template>
            </el-table-column>
            <el-table-column label="花园经验" width="180" align="center">
              <template slot-scope="{row}">
                <el-input-number v-model="row.exp" :min="0" :max="999999" size="small" controls-position="right" />
              </template>
            </el-table-column>
            <el-table-column label="元宝" width="160" align="center">
              <template slot-scope="{row}">
                <el-input-number v-model="row.ingots" :min="0" :max="999" size="small" controls-position="right" />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="110" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" icon="el-icon-check" :loading="row._saving"
                           @click="saveReward(row)">保存</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminGardenSign',
  data () {
    return {
      tab: 'stats',
      // 统计
      stats: {},
      // 记录
      logs: [], total: 0, page: 1, size: 15, loading: false, word: '',
      // 奖励配置
      rewards: [], rewardLoading: false
    }
  },
  mounted () { this.loadStats(); this.loadLogs() },
  methods: {
    onTab () {
      if (this.tab === 'stats') this.loadStats()
      if (this.tab === 'rewards' && this.rewards.length === 0) this.loadRewards()
    },
    weekName (d) {
      return { 1: '周一', 2: '周二', 3: '周三', 4: '周四', 5: '周五', 6: '周六', 7: '周日' }[d] || '-'
    },
    loadStats () {
      api.get('/admin/garden-sign-stats').then(r => {
        if (r.code === 0) this.stats = r.data || {}
        else this.$message.error(r.msg)
      })
    },
    loadLogs () {
      this.loading = true
      api.get('/admin/garden-logs', { params: { type: 'sign', page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.logs = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    loadRewards () {
      this.rewardLoading = true
      api.get('/admin/garden-sign-rewards').then(r => {
        this.rewardLoading = false
        if (r.code === 0) {
          this.rewards = (r.data || []).map(x => ({ ...x, _saving: false }))
        } else this.$message.error(r.msg)
      })
    },
    saveReward (row) {
      row._saving = true
      api.put('/admin/garden-sign-rewards/' + row.day, {
        seed_n: row.seed_n, coins: row.coins, exp: row.exp, ingots: row.ingots
      }).then(r => {
        row._saving = false
        if (r.code === 0) this.$message.success(r.data.msg || '已保存')
        else this.$message.error(r.msg)
      })
    }
  }
}
</script>

<style scoped>
.s-gray .stat-ico { background: linear-gradient(135deg,#b0bec5,#78909c); }
.tip { font-size: 13px; color: #8a9bb0; }
</style>
