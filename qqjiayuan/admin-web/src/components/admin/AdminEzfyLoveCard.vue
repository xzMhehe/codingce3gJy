<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <!-- ★ 2026-09-27 用户需求：为爱发电卡（普通 30 天/天领150钻石，高级 30 天/天领200钻石）。
           卡片由管理端发放即生效，玩家在游戏「任务 → 为爱发电卡」tab 每日领取（漏领累加、封顶30天）。 -->
      <div class="toolbar">
        <el-input v-model="word" placeholder="玩家昵称 / 游戏ID" clearable style="width:220px"
                  @keyup.enter.native="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <span class="grow" />
        <el-button type="success" icon="el-icon-present" @click="openGrant">发放为爱发电卡</el-button>
        <!-- ★ 2026-10-05 用户要求：本页已有「查询」按钮（点它就会重新 load），这个「刷新」按钮功能重复、容易误点 → 去掉。 -->
      </div>

      <el-table :data="rows" v-loading="loading" stripe border max-height="620">
        <el-table-column type="index" label="#" width="50" align="center" />
        <el-table-column label="玩家" min-width="160" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-main">{{ row.nickname }}</span></template>
        </el-table-column>
        <el-table-column label="玩家ID" width="90" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ row.user_id }}</span></template>
        </el-table-column>
        <el-table-column label="卡片" min-width="140" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column label="每日钻石" width="90" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ row.daily_diamond }}</span></template>
        </el-table-column>
        <el-table-column label="生效时间" width="160" align="center">
          <template slot-scope="{row}">{{ fmtTime(row.start_time) }}</template>
        </el-table-column>
        <el-table-column label="失效时间" width="160" align="center">
          <template slot-scope="{row}">{{ fmtTime(row.end_time) }}</template>
        </el-table-column>
        <el-table-column label="领取进度" width="150" align="center">
          <template slot-scope="{row}">
            <el-progress :percentage="pct(row)" :stroke-width="10" :format="fmtPct" />
            <span class="td-sub">{{ row.claimed_days }}/{{ row.total_days }} 天 · 剩 {{ row.remaining }}</span>
          </template>
        </el-table-column>
        <el-table-column label="上次领取" width="160" align="center">
          <template slot-scope="{row}">
            <span v-if="row.last_claim_time">{{ fmtTime(row.last_claim_time) }}</span>
            <span v-else class="td-sub">—</span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="发放时间" width="160" align="center">
          <template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="90" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除该卡记录" @click="doDelete(row)" />
          </template>
        </el-table-column>
      </el-table>

      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ total }}</b> 条 · 每页 {{ size }} 条</div>
        <el-pagination v-show="total > 0" small background layout="sizes, prev, pager, next, jumper"
                       :total="total" :page-size="size" :current-page="page" :page-sizes="[10, 20, 50]"
                       @current-change="p => { page = p; load() }"
                       @size-change="s => { size = s; page = 1; load() }" />
      </div>

      <!-- 发放弹窗：按玩家搜索 + 选卡片 + 数量，走通用发放接口（发放即生效创建激活记录） -->
      <el-dialog title="发放为爱发电卡" :visible.sync="grantDlg" width="560px" :close-on-click-modal="false">
        <el-form label-width="90px" size="small">
          <el-form-item label="目标玩家" required>
            <el-input v-model="grantPlayer" placeholder="玩家昵称 / 游戏ID，点下方搜索选择" :disabled="grantUserId > 0"
                      clearable style="width:280px" @clear="grantUserId = 0" />
            <el-button v-if="!grantUserId" size="small" type="primary" plain @click="searchPlayer">搜索</el-button>
            <el-button v-else size="small" @click="grantUserId = 0; grantPlayer = ''">重选</el-button>
          </el-form-item>
          <el-form-item v-if="playerOpts.length" label="选择玩家">
            <el-radio-group v-model="grantUserId" size="small">
              <el-radio-button v-for="p in playerOpts" :key="p.user_id" :label="p.user_id">
                {{ p.nickname }}(ID:{{ p.user_id }})
              </el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="卡片类型" required>
            <el-select v-model="grantCardId" style="width:100%" placeholder="选择要发放的卡片">
              <el-option v-for="o in cardOpts" :key="o.id" :label="o.name" :value="o.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="发放数量">
            <el-input-number v-model="grantCount" :min="1" :max="10" style="width:160px" />
          </el-form-item>
        </el-form>
        <div slot="footer">
          <el-button size="small" @click="grantDlg = false">取 消</el-button>
          <el-button size="small" type="primary" :loading="grantSaving" @click="doGrant">发 放</el-button>
        </div>
      </el-dialog>
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminEzfyLoveCard',
  data () {
    return {
      word: '',
      rows: [],
      total: 0,
      page: 1,
      size: 20,
      loading: false,
      // 发放
      grantDlg: false,
      grantPlayer: '',
      grantUserId: 0,
      playerOpts: [],
      cardOpts: [],
      grantCardId: 0,
      grantCount: 1,
      grantSaving: false
    }
  },
  created () { this.load(); this.loadCardOpts() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/ezfy-love-cards/list', { params: { word: this.word, page: this.page, size: this.size } }).then(r => {
        this.loading = false
        if (r.code !== 0) { this.$message.error(r.msg || '加载失败'); return }
        this.rows = r.data.list || []
        this.total = r.data.total || 0
      }).catch(() => { this.loading = false })
    },
    pct (row) {
      if (!row.total_days) return 0
      return Math.min(100, Math.round(row.claimed_days / row.total_days * 100))
    },
    fmtPct (p) { return p + '%' },
    fmtTime (t) {
      if (!t) return ''
      const d = new Date(t < 1e12 ? t * 1000 : t)
      if (isNaN(d.getTime())) return ''
      const p = n => String(n).padStart(2, '0')
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' +
        p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds())
    },
    // 卡片下拉选项（专属接口，自动带每日钻石数）
    loadCardOpts () {
      api.get('/admin/ezfy-love-cards/options').then(r => {
        const all = (r.code === 0 && r.data.list) ? r.data.list : []
        this.cardOpts = all
      })
    },
    openGrant () {
      this.grantDlg = true
      this.grantPlayer = ''
      this.grantUserId = 0
      this.playerOpts = []
      this.grantCardId = this.cardOpts.length ? this.cardOpts[0].id : 0
      this.grantCount = 1
    },
    searchPlayer () {
      const w = (this.grantPlayer || '').trim()
      if (!w) { this.$message.warning('请输入玩家昵称或游戏ID'); return }
      api.get('/admin/ezfy-item-grant/players', { params: { word: w } }).then(r => {
        if (r.code !== 0) { this.$message.error(r.msg); this.playerOpts = []; return }
        this.playerOpts = r.data.list || []
        if (this.playerOpts.length === 1) this.grantUserId = this.playerOpts[0].user_id
        if (!this.playerOpts.length) this.$message.warning('未找到匹配的玩家')
      })
    },
    doGrant () {
      if (!this.grantUserId) { this.$message.warning('请先搜索并选择目标玩家'); return }
      if (!this.grantCardId) { this.$message.warning('请选择卡片类型'); return }
      this.grantSaving = true
      api.post('/admin/ezfy-item-grant', {
        player: String(this.grantUserId),
        items: [{ cfg_id: this.grantCardId, count: this.grantCount }]
      }).then(r => {
        this.grantSaving = false
        if (r.code === 0) {
          this.$message.success((r.data && r.data.msg) || '发放成功')
          this.grantDlg = false
          this.page = 1
          this.load()
        } else this.$message.error(r.msg || '发放失败')
      }).catch(() => { this.grantSaving = false })
    },
    doDelete (row) {
      this.$confirm('确认删除该玩家的「' + row.name + '」记录？删除后玩家将无法再领取该卡剩余钻石。', '删除确认', {
        type: 'warning',
        confirmButtonText: '删除',
        cancelButtonText: '取消'
      }).then(() => {
        api.delete('/admin/ezfy-love-cards/' + row.id).then(r => {
          if (r.code === 0) {
            this.$message.success((r.data && r.data.msg) || '已删除')
            this.load()
          } else this.$message.error(r.msg || '删除失败')
        })
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
.grow { flex: 1 }
.td-main { font-weight: 600 }
.td-mono { font-family: Menlo, Consolas, monospace; font-size: 12px }
.td-sub { color: #909399; font-size: 12px; display: block; margin-top: 2px }
</style>