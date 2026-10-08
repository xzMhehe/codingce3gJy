<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="战场ID / 订单ID / 用户ID / 昵称 / 目标名" clearable style="width:250px"
                  @keyup.enter.native="page = 1; load()" />
        <el-select v-model="status" style="width:130px; margin-left:8px" @change="page = 1; load()">
          <el-option label="全部状态" value="-1" />
          <el-option label="进行中" value="1" />
          <el-option label="已结束" value="2" />
        </el-select>
        <el-select v-model="targetType" style="width:130px; margin-left:8px" @change="page = 1; load()">
          <el-option label="全部目标" value="0" />
          <el-option label="野地" value="1" />
          <el-option label="寇城" value="2" />
          <el-option label="玩家城" value="3" />
        </el-select>
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
      </div>
      <el-table :data="list" v-loading="loading" stripe border max-height="560">
        <el-table-column prop="id" label="战场ID" width="85" align="center" />
        <el-table-column prop="order_id" label="订单ID" width="85" align="center" />
        <el-table-column label="攻方" min-width="130" show-overflow-tooltip>
          <template slot-scope="{row}">
            <span class="td-main">{{ row.atk_name || '—' }}</span>
            <span class="td-gray">({{ row.atk_home || row.user_id }})</span>
          </template>
        </el-table-column>
        <el-table-column label="守方" min-width="130" show-overflow-tooltip>
          <template slot-scope="{row}">
            <template v-if="row.def_user_id">
              <span class="td-main">{{ row.def_name || '—' }}</span>
              <span class="td-gray">({{ row.def_home || row.def_user_id }})</span>
            </template>
            <span v-else class="td-gray">AI</span>
          </template>
        </el-table-column>
        <el-table-column label="目标" min-width="170" show-overflow-tooltip>
          <template slot-scope="{row}">
            <el-tag size="mini">{{ row.target_type_name }}</el-tag>
            {{ row.target_name }}（{{ row.target_x }},{{ row.target_y }}）
          </template>
        </el-table-column>
        <el-table-column label="回合" width="75" align="center">
          <template slot-scope="{row}">{{ row.round }}/{{ maxRounds }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.status === 1 ? 'warning' : 'info'">
              {{ row.status === 1 ? '进行中' : '已结束' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="结果" width="90" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="winTag(row.win_name)">{{ row.win_name }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="订单" width="95" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="orderTag(row.order_status)">{{ orderStatusName(row.order_status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="开始时间" width="170" align="center">
          <template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="165" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="info" plain icon="el-icon-view" title="战斗详情" @click="openDetail(row)" />
            <el-button size="mini" type="primary" plain icon="el-icon-right" title="推进一回合"
                       :disabled="row.status !== 1" @click="doTick(row)" />
            <el-button size="mini" type="success" plain icon="el-icon-video-play" title="自动打完"
                       :disabled="row.status !== 1" @click="doAuto(row)" />
            <el-button size="mini" type="warning" plain icon="el-icon-magic-stick" title="强制结算 / 清理卡死"
                       @click="doForce(row)" />
          </template>
        </el-table-column>
      </el-table>
      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ total }}</b> 条 · 每页 {{ size }} 条</div>
        <el-pagination v-show="total > 0" small background layout="sizes, prev, pager, next, jumper" :total="total"
                       :page-size="size" :current-page="page" :page-sizes="[10, 20, 50, 100]"
                       @current-change="p => { page = p; load() }"
                       @size-change="s => { size = s; page = 1; load() }" />
      </div>
    </el-card>

    <!-- ============ 战斗详情 ============ -->
    <el-dialog :title="detailTitle" :visible.sync="detailDlg" width="900px" :close-on-click-modal="false">
      <div v-if="detail" v-loading="detailLoading">
        <el-descriptions :column="3" size="small" border>
          <el-descriptions-item label="战场ID">{{ detail.battle.id }}</el-descriptions-item>
          <el-descriptions-item label="订单ID">{{ detail.battle.order_id || '—' }}</el-descriptions-item>
          <el-descriptions-item label="目标类型">{{ detail.target_type_name }}</el-descriptions-item>
          <el-descriptions-item label="攻方">
            {{ detail.atk_name || '—' }}（{{ detail.atk_home || detail.battle.user_id }}）
          </el-descriptions-item>
          <el-descriptions-item label="守方">
            {{ detail.battle.def_user_id ? (detail.def_name || '—') + '（' + (detail.def_home || detail.battle.def_user_id) + '）' : 'AI' }}
          </el-descriptions-item>
          <el-descriptions-item label="目标">
            {{ detail.battle.target_name }}（{{ detail.battle.target_x }},{{ detail.battle.target_y }}）
          </el-descriptions-item>
          <el-descriptions-item label="回合">{{ detail.battle.round }} / {{ maxRounds }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag size="mini" :type="detail.battle.status === 1 ? 'warning' : 'info'">
              {{ detail.battle.status === 1 ? '进行中' : '已结束' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="结果">
            <el-tag size="mini" :type="winTag(detail.win_name)">{{ detail.win_name }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="订单状态">
            <el-tag size="mini" :type="orderTag(detail.battle.order_status)">{{ orderStatusName(detail.order_status) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="开始时间">{{ fmtTime(detail.battle.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="更新时间">{{ fmtTime(detail.battle.updated_at) }}</el-descriptions-item>
        </el-descriptions>

        <div class="sub-title">双方部队</div>
        <el-table :data="detailUnits" size="mini" border max-height="260">
          <el-table-column prop="side" label="方" width="60" align="center" />
          <el-table-column prop="name" label="兵种" width="110" align="center" />
          <el-table-column label="剩余" width="110" align="center">
            <template slot-scope="{row}"><span class="td-mono">{{ fmtN(row.count) }}</span></template>
          </el-table-column>
          <el-table-column label="初始" width="110" align="center">
            <template slot-scope="{row}"><span class="td-mono">{{ fmtN(row.initial_count) }}</span></template>
          </el-table-column>
          <el-table-column label="损失" width="110" align="center">
            <template slot-scope="{row}">
              <span class="td-mono" :class="{ 'td-danger': row.initial_count - row.count > 0 }">
                {{ fmtN(row.initial_count - row.count) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="pos" label="位置" min-width="90" align="center" />
        </el-table>

        <div class="sub-title">准备回合</div>
        <div class="log-box">{{ (detail.snapshot.head || []).join('\n') || '（无）' }}</div>

        <div class="sub-title">逐回合日志（最新在前）</div>
        <div class="log-box tall">{{ actionsDesc || '（无）' }}</div>
      </div>
      <div slot="footer">
        <el-button @click="detailDlg = false">关 闭</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

// ★ 2026-10-08 管理端「战斗队列管理」：查看战场（双方部队/准备回合/逐回合日志），
//   并可手动推进一回合、一键自动打完、强制结算/清理卡死战场。
//   后端接口：/admin/ezfy-battles（列表）、/:id（详情）、/:id/tick、/:id/auto、/:id/force
export default {
  name: 'AdminEzfyBattles',
  data () {
    return {
      list: [], total: 0, page: 1, size: 10, loading: false,
      word: '', status: '-1', targetType: '0',
      // 战斗回合上限（与后端 ezfyBattleMaxRounds 一致）
      maxRounds: 40,
      detailDlg: false, detail: null, detailLoading: false
    }
  },
  computed: {
    detailTitle () {
      if (!this.detail) return '战斗详情'
      const b = this.detail.battle
      return '战场 #' + b.id + ' · ' + (b.target_name || '') + '（' + b.target_x + ',' + b.target_y + '）'
    },
    // 双方部队合并成一张表（攻方在前）
    detailUnits () {
      if (!this.detail || !this.detail.snapshot) return []
      const snap = this.detail.snapshot
      const out = []
      ;(snap.attackers || []).forEach(u => out.push(Object.assign({ side: '攻' }, u)))
      ;(snap.defenders || []).forEach(u => out.push(Object.assign({ side: '守' }, u)))
      return out
    },
    // 逐回合日志：最新回合在最上（与玩家端指挥室一致）
    actionsDesc () {
      if (!this.detail || !this.detail.snapshot) return ''
      const acts = this.detail.snapshot.actions || []
      return acts.slice().reverse().join('\n')
    }
  },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/ezfy-battles', {
        params: { page: this.page, size: this.size, word: this.word, status: this.status, target_type: this.targetType }
      }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list || []
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      }).catch(() => { this.loading = false })
    },
    openDetail (row) {
      this.detail = null
      this.detailLoading = true
      this.detailDlg = true
      api.get('/admin/ezfy-battles/' + row.id).then(r => {
        this.detailLoading = false
        if (r.code === 0) this.detail = r.data
        else { this.detailDlg = false; this.$message.error(r.msg) }
      }).catch(() => { this.detailLoading = false })
    },
    doTick (row) {
      api.post('/admin/ezfy-battles/' + row.id + '/tick').then(r => {
        if (r.code === 0) {
          this.$message.success(r.data.msg || '已推进')
          this.load()
          if (this.detailDlg) this.openDetail(row)
        } else this.$message.error(r.msg)
      })
    },
    doAuto (row) {
      this.$confirm('确认让 AI 把战场 #' + row.id + ' 直接打完？' +
        '结束后会回写订单并结算（发战报、掠夺、部队返航）。', '自动打完', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-battles/' + row.id + '/auto').then(r => {
          if (r.code === 0) {
            this.$message.success(r.data.msg || '已自动打完')
            this.load()
            if (this.detailDlg) this.openDetail(row)
          } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    doForce (row) {
      this.$confirm('强制结算 / 清理战场 #' + row.id + '？\n' +
        '· 战场已结束但订单卡在「战斗中」→ 回写结果并结算\n' +
        '· 僵尸战场（订单已不在战斗中）→ 直接标记结束', '强制结算 / 清理', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-battles/' + row.id + '/force').then(r => {
          if (r.code === 0) {
            this.$message.success(r.data.msg || '已处理')
            this.load()
            if (this.detailDlg) this.openDetail(row)
          } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // 订单状态（与玩家端军队动态/出征队列同口径；-1 = 订单已不存在）
    orderStatusName (s) {
      switch (s) {
        case -1: return '无订单'
        case 0: return '出征'
        case 1: return '驻守'
        case 2: return '返航'
        case 3: return '常驻驻军'
        case 5: return '战斗中'
        case 6: return '等待'
        case 98: return '结算中'
        default: return '已完成'
      }
    },
    orderTag (s) {
      if (s === 5 || s === 6 || s === 98) return 'warning'
      if (s === -1) return 'info'
      return 'success'
    },
    winTag (name) {
      switch (name) {
        case '攻方胜': return 'success'
        case '守方胜': return 'danger'
        case '平局': return 'info'
        default: return 'warning'
      }
    },
    fmtTime (t) { return t ? new Date(t).toLocaleString() : '' },
    fmtN (n) { return (n === undefined || n === null) ? '' : Number(n).toLocaleString() }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.sub-title { font-size: 13px; font-weight: 600; color: #1f2d3d; margin: 14px 0 8px; padding-left: 6px; border-left: 3px solid #409eff; }
.log-box { max-height: 220px; overflow-y: auto; border: 1px solid #ebeef5; border-radius: 4px; padding: 8px 10px; background: #fafbfc; line-height: 1.7; white-space: pre-wrap; word-break: break-all; font-size: 13px; color: #303133; }
.log-box.tall { max-height: 380px; }
.td-danger { color: #f56c6c; }
</style>
