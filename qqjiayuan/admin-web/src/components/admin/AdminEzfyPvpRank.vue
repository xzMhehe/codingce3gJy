<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <span>排行榜</span>
      </div>

      <!-- ★ 2026-10-14 用户需求：「风云榜」页改名「排行榜」，下面设 风云榜 / 战力榜 两个 tab -->
      <el-tabs v-model="tab" class="cfg-tabs" @tab-click="onTabChange">
        <el-tab-pane label="风云榜" name="pvp" />
        <el-tab-pane label="战力榜" name="power" />
      </el-tabs>

      <!-- ==================== 风云榜 tab ==================== -->
      <template v-if="tab === 'pvp'">
        <div class="toolbar">
          <!-- ★ 开启/关闭：开启 = 现在这样按周统计；关闭 = 不进行统计（历史保留） -->
          <span class="sw-label">风云榜统计</span>
          <el-switch v-model="enabled" :active-value="1" :inactive-value="0" active-text="开" inactive-text="关"
                     :loading="toggling" @change="onToggle" />
          <span class="td-sub" style="margin-left:6px">
            {{ enabled ? '按自然周统计玩家对战战绩' : '已关闭：不再统计新的玩家对战战绩（历史周榜保留）' }}
          </span>
          <div class="grow" />
          <!-- ★ 2026-10-15 风云榜查询：按游戏ID(user_id)精确查找玩家（清空 = 全量榜单） -->
          <el-input v-model="pvpWord" placeholder="游戏ID" clearable style="width:160px"
                    @keyup.enter.native="doPvpSearch()" />
          <el-button type="primary" icon="el-icon-search" @click="doPvpSearch()">查询</el-button>
          <el-select v-model="week" placeholder="选择统计周" style="width:200px" @change="loadPvp(true)">
            <el-option v-for="w in weeks" :key="w" :label="w" :value="w" />
          </el-select>
        </div>

        <el-table :data="pvpPageList" v-loading="loading" stripe border>
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
          <!-- ★ 操作列：不参与排名（把该玩家从榜单剔除；历史战绩保留，可恢复） -->
          <el-table-column label="操作" width="130" align="center" fixed="right">
            <template slot-scope="{row}">
              <el-button size="mini" type="danger" plain icon="el-icon-remove-outline"
                         title="不参与排名" @click="openExclude(row)">不参与排名</el-button>
            </template>
          </el-table-column>
        </el-table>
        <!-- ★ 2026-10-14 风云榜分页（默认每页 10 条） -->
        <div class="pager-bar">
          <div class="pager-info">共 <b>{{ pvpTotal }}</b> 名 · 榜单按「积分」降序（同积分按灭敌多、胜场多、战损少）。昵称/军团为写榜时快照。</div>
          <el-pagination v-show="pvpTotal > 0" small background layout="sizes, prev, pager, next, jumper"
                         :total="pvpTotal" :page-size="pvpSize" :current-page="pvpPage"
                         :page-sizes="[10, 20, 50, 100]"
                         @current-change="p => { pvpPage = p }"
                         @size-change="s => { pvpSize = s; pvpPage = 1 }" />
        </div>

        <!-- 不参与排名名单 -->
        <div class="panel-title">不参与排名（{{ excluded.length }} 人）</div>
        <el-table :data="excluded" v-loading="loading" stripe border size="mini"
                  :empty-text="'暂无排除玩家'">
          <el-table-column prop="user_id" label="用户ID" width="90" align="center" />
          <el-table-column prop="name" label="玩家" min-width="120" show-overflow-tooltip>
            <template slot-scope="{row}">{{ row.name || '—' }}</template>
          </el-table-column>
          <el-table-column prop="reason" label="原因" min-width="160" show-overflow-tooltip>
            <template slot-scope="{row}">{{ row.reason || '—' }}</template>
          </el-table-column>
          <el-table-column prop="created_at" label="加入时间" width="160" align="center" />
          <el-table-column label="操作" width="110" align="center" fixed="right">
            <template slot-scope="{row}">
              <el-button size="mini" type="success" plain icon="el-icon-refresh-left"
                         title="恢复参与排名" @click="removeExclude(row)">恢复排名</el-button>
            </template>
          </el-table-column>
        </el-table>
      </template>

      <!-- ==================== 战力榜 tab ==================== -->
      <template v-if="tab === 'power'">
        <div class="toolbar">
          <el-input v-model="powerWord" placeholder="昵称 / 用户ID" clearable style="width:220px"
                    @keyup.enter.native="powerPage = 1; loadPower()" />
          <el-button type="primary" icon="el-icon-search" @click="powerPage = 1; loadPower()">查询</el-button>
          <div class="grow" />
          <span class="td-sub">战力 = 科技 + 建筑 + 兵种（权重在「二战系统配置」可调），与游戏内战力榜同口径</span>
        </div>
        <el-table :data="powerList" v-loading="powerLoading" stripe border>
          <el-table-column label="名次" width="70" align="center">
            <template slot-scope="{row}">
              <span class="td-rank" :class="row.rank <= 3 ? 'top' : ''">{{ row.rank }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="user_id" label="用户ID" width="90" align="center" />
          <el-table-column prop="name" label="玩家" min-width="120" show-overflow-tooltip>
            <template slot-scope="{row}">{{ row.name || '—' }}</template>
          </el-table-column>
          <el-table-column prop="city_name" label="主城" min-width="110" show-overflow-tooltip>
            <template slot-scope="{row}">{{ row.city_name || '—' }}</template>
          </el-table-column>
          <el-table-column label="战力" width="110" align="right">
            <template slot-scope="{row}"><b>{{ row.power.toLocaleString() }}</b></template>
          </el-table-column>
          <el-table-column label="科技战力" width="110" align="right">
            <template slot-scope="{row}">{{ row.tech_power.toLocaleString() }}</template>
          </el-table-column>
          <el-table-column label="建筑战力" width="110" align="right">
            <template slot-scope="{row}">{{ row.build_power.toLocaleString() }}</template>
          </el-table-column>
          <el-table-column label="兵种战力" width="110" align="right">
            <template slot-scope="{row}">{{ row.troop_power.toLocaleString() }}</template>
          </el-table-column>
        </el-table>
        <div class="pager-bar">
          <div class="pager-info">共 <b>{{ powerTotal }}</b> 条 · 每页 {{ powerSize }} 条</div>
          <el-pagination v-show="powerTotal > 0" small background layout="sizes, prev, pager, next, jumper"
                         :total="powerTotal" :page-size="powerSize" :current-page="powerPage"
                         :page-sizes="[10, 20, 50, 100]"
                         @current-change="p => { powerPage = p; loadPower() }"
                         @size-change="s => { powerSize = s; powerPage = 1; loadPower() }" />
        </div>
      </template>
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

// ★ 2026-10-14 排行榜（原「风云榜」）：
//   风云榜 tab —— 每周玩家对战战绩周榜（可开关统计、可把玩家设为「不参与排名」）
//   战力榜 tab —— 复用游戏内战力口径（科技 + 建筑 + 兵种），管理端全量分页查看
// ★ 2026-10-15 用户：「排行榜 刷新按钮去掉；风云榜里加个查询按钮能根据游戏ID查询玩家」
//   —— 去掉头部 [刷新]（进页面/切周自动拉数据）；风云榜 tab 加「游戏ID」输入 + [查询]
//      （后端一次性下发整周全量榜单，查询走前端精确过滤，清空回到全量）。
export default {
  name: 'AdminEzfyPvpRank',
  data () {
    return {
      tab: 'pvp',
      // ---- 风云榜 ----
      loading: false,
      toggling: false,
      enabled: 1,
      week: '',
      weeks: [],
      list: [],
      // ★ 2026-10-14 风云榜分页（默认每页 10 条）——后端一次性返回整周榜单，前端切片
      pvpPage: 1,
      pvpSize: 10,
      pvpWord: '', // ★ 2026-10-15 风云榜查询：按游戏ID(user_id)精确查找玩家
      excluded: [],
      // ---- 战力榜 ----
      powerLoading: false,
      powerPage: 1,
      powerSize: 10,
      powerTotal: 0,
      powerWord: '',
      powerList: [],
    }
  },
  mounted () {
    this.loadPvp()
  },
  computed: {
    // ★ 2026-10-14 风云榜分页：后端一次性返回整周榜单，这里按页切片展示
    // ★ 2026-10-15 风云榜查询：填了游戏ID 就只展示该玩家（查不到 = 空），清空回到全量榜单
    pvpShown () {
      if (!this.pvpWord) return this.list
      const uid = Number(this.pvpWord)
      if (isNaN(uid)) return []
      return this.list.filter(r => r.user_id === uid)
    },
    pvpTotal () { return this.pvpShown().length },
    pvpPageList () {
      const start = (this.pvpPage - 1) * this.pvpSize
      return this.pvpShown().slice(start, start + this.pvpSize)
    }
  },
  methods: {
    // ★ 2026-10-15 风云榜查询：按游戏ID精确查找玩家（查不到提示，便于核对输入）
    doPvpSearch () {
      this.pvpPage = 1
      if (!this.pvpWord) return
      const uid = Number(this.pvpWord)
      if (isNaN(uid)) {
        this.$message.warning('请输入数字游戏ID')
        return
      }
      if (!this.list.some(r => r.user_id === uid)) {
        this.$message.info('该周榜单中没有此玩家')
      }
    },
    onTabChange () {
      if (this.tab === 'power' && !this.powerList.length) this.loadPower()
    },
    loadPvp (resetPage) {
      this.loading = true
      // ★ 2026-10-14 切周（resetPage）后回到第 1 页；单纯刷新保留当前页
      if (resetPage === true) this.pvpPage = 1
      const params = {}
      if (this.week) params.week = this.week
      api.get('/admin/ezfy-pvp-rank', { params }).then(r => {
        if (r.code === 0) {
          this.weeks = r.data.weeks || []
          this.week = r.data.week
          this.list = r.data.ranks || []
          this.excluded = r.data.excluded || []
          // 后端 1 = 开 / 0 = 关
          this.enabled = r.data.enabled ? 1 : 0
          if (this.weeks.indexOf(this.week) < 0 && this.week) this.weeks.unshift(this.week)
        } else {
          // ★ 2026-10-15 防「空白页」：接口异常时明确提示，而不是静默一片空白
          this.$message.error(r.msg || '加载排行榜失败')
          this.list = []
          this.excluded = []
        }
      }).catch(() => {
        this.$message.error('加载排行榜失败，请检查后端服务')
        this.list = []
        this.excluded = []
      }).finally(() => { this.loading = false })
    },
    onToggle (val) {
      this.toggling = true
      api.post('/admin/ezfy-pvp-rank/toggle', { on: val }).then(r => {
        if (r.code === 0) this.$message.success(r.data.msg || '已保存')
        else {
          this.$message.error(r.msg)
          this.enabled = val ? 0 : 1 // 失败回滚开关显示
        }
      }).catch(() => { this.enabled = val ? 0 : 1 })
        .finally(() => { this.toggling = false })
    },
    openExclude (row) {
      this.$prompt('把「' + (row.name || row.user_id) + '」设为不参与排名？可填原因（选填）。',
        '不参与排名', { inputPlaceholder: '原因，例如：刷分 / 违规', inputValue: '' })
        .then(({ value }) => {
          api.post('/admin/ezfy-pvp-rank/exclude', { user_id: row.user_id, reason: value || '' }).then(r => {
            if (r.code === 0) {
              this.$message.success(r.data.msg || '已设置')
              this.loadPvp()
            } else this.$message.error(r.msg)
          })
        }).catch(() => {})
    },
    removeExclude (row) {
      this.$confirm('恢复「' + (row.name || row.user_id) + '」参与排名？', '提示', { type: 'warning' }).then(() => {
        // 走排除记录的 id（不是 user_id）—— 接口是按记录主键删的
        api.delete('/admin/ezfy-pvp-rank/exclude/' + row.id).then(r => {
          if (r.code === 0) {
            this.$message.success(r.data.msg || '已恢复')
            this.loadPvp()
          } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    loadPower () {
      this.powerLoading = true
      const params = { page: this.powerPage, size: this.powerSize }
      if (this.powerWord) params.word = this.powerWord
      api.get('/admin/ezfy-power-rank', { params }).then(r => {
        if (r.code === 0) {
          this.powerList = r.data.list || []
          this.powerTotal = r.data.total || 0
        }
      }).finally(() => { this.powerLoading = false })
    }
  }
}
</script>

<style>
  /* ★ 2026-10-14 补齐公共样式（card-head / toolbar / grow / pager-bar / pager-info）——
     之前本文件没 import 也没重定义 `.card-head`，header 里的「排行榜」标题和「刷新」按钮
     因为没有 flex 布局而贴在一起（用户反馈）。 */
  @import './farm-admin.css';
  .td-rank { font-weight: 600; color: #909399; }
  .td-rank.top { color: #e6a23c; }
  .cfg-tabs { margin-bottom: 6px; }
  .sw-label { font-size: 13px; color: #606266; margin-right: 6px; }
  .panel-title { font-size: 14px; font-weight: 600; color: #303133; margin: 16px 0 8px; }
</style>
