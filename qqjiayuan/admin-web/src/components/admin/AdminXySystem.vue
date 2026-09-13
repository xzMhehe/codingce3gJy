<template>
  <div class="farm-admin">
    <!-- 服务器维护 -->
    <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <span>服务器维护</span>
        <el-tag size="mini" :type="srv.maintenance ? 'danger' : 'success'">{{ srv.maintenance ? '维护中' : '开放中' }}</el-tag>
      </div>
      <el-form label-width="90px" v-loading="loadingSrv">
        <el-form-item label="维护模式">
          <el-switch v-model="srvForm.on" active-text="维护" inactive-text="开放" />
        </el-form-item>
        <el-form-item label="维护公告">
          <el-input v-model="srvForm.notice" type="textarea" :rows="3" maxlength="200" show-word-limit
                    placeholder="维护期间玩家进入游戏将看到此公告" />
        </el-form-item>
      </el-form>
      <div style="text-align:right">
        <el-button size="mini" plain icon="el-icon-refresh" @click="loadServer">刷新状态</el-button>
        <el-button type="danger" :loading="savingSrv" @click="saveServer">{{ srvForm.on ? '开启维护' : '保存并开放' }}</el-button>
      </div>
      <em>提示：开启维护后，所有玩家将无法进入幻想西游（游戏内所有操作均被拦截并显示维护公告）</em>
    </el-card>

    <!-- 游戏统计 -->
    <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <span>游戏统计</span>
        <el-button size="mini" type="primary" plain icon="el-icon-refresh" @click="loadStats">刷新</el-button>
      </div>
      <el-row :gutter="14" v-if="stats" v-loading="loadingStats">
        <el-col :span="6" v-for="c in statCards" :key="c.label">
          <div class="stat-card">
            <div class="stat-num">{{ c.val }}</div>
            <div class="stat-label">{{ c.label }}</div>
          </div>
        </el-col>
      </el-row>
      <el-row :gutter="14" v-if="stats" style="margin-top:14px">
        <el-col :span="6" v-for="c in moneyCards" :key="c.label">
          <div class="stat-card stat-money">
            <div class="stat-num">{{ c.val }}</div>
            <div class="stat-label">{{ c.label }}</div>
          </div>
        </el-col>
      </el-row>
      <template v-if="stats && stats.levels && stats.levels.length">
        <div class="sub-title">等级分布（TOP10）</div>
        <el-table :data="stats.levels" size="mini" border max-height="260">
          <el-table-column prop="level" label="等级" width="90" align="center" />
          <el-table-column label="人数" align="center">
            <template slot-scope="{row}">{{ row.cnt }}</template>
          </el-table-column>
          <el-table-column label="占比" align="center">
            <template slot-scope="{row}">{{ percent(row.cnt) }}</template>
          </el-table-column>
        </el-table>
      </template>
    </el-card>

    <el-row :gutter="14">
      <!-- 发布公告 -->
      <el-col :span="12">
        <el-card shadow="never" class="box">
          <div slot="header" class="card-head"><span>发布系统消息</span></div>
          <el-input v-model="announce" type="textarea" :rows="4" maxlength="200" show-word-limit
                    placeholder="将以「系统」名义发布到游戏聊天频道，玩家在聊天框可见" />
          <div style="margin-top:12px;text-align:right">
            <el-button type="primary" :loading="sending" @click="doAnnounce">发 布</el-button>
          </div>
        </el-card>
      </el-col>
      <!-- 全服发放 -->
      <el-col :span="12">
        <el-card shadow="never" class="box">
          <div slot="header" class="card-head"><span>全服发放</span></div>
          <el-form label-width="90px">
            <el-form-item label="类型">
              <el-radio-group v-model="grantAll.kind" @change="grantAll.kind === 'item' ? 0 : (grantAll.amount = grantAll.amount || 1000)">
                <el-radio label="money">银两</el-radio>
                <el-radio label="beans">金豆</el-radio>
                <el-radio label="item">物品</el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item v-if="grantAll.kind !== 'item'" label="数量">
              <el-input-number v-model.number="grantAll.amount" :min="1" />
            </el-form-item>
            <template v-else>
              <el-form-item label="物品ID">
                <el-input-number v-model.number="grantAll.item_id" :min="1" />
              </el-form-item>
              <el-form-item label="数量">
                <el-input-number v-model.number="grantAll.count" :min="1" :max="9999" />
              </el-form-item>
            </template>
            <el-form-item label="事由">
              <el-input v-model="grantAll.reason" maxlength="50" placeholder="默认：全服奖励" style="width:220px" />
            </el-form-item>
          </el-form>
          <em>提示：物品ID 可在「幻想西游 → 数据管理」中查询；发放直接入背包/钱包</em>
          <div style="margin-top:12px;text-align:right">
            <el-button type="warning" :loading="granting" @click="doGrantAll">全服发放</el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminXySystem',
  data () {
    return {
      stats: null, loadingStats: false,
      announce: '', sending: false,
      grantAll: { kind: 'money', amount: 1000, item_id: 1, count: 1, reason: '' },
      granting: false,
      srv: { maintenance: false, notice: '', players: 0 },
      srvForm: { on: false, notice: '' },
      loadingSrv: false, savingSrv: false
    }
  },
  computed: {
    statCards () {
      const s = this.stats
      return [
        { label: '玩家总数', val: s.players },
        { label: '今日新增', val: s.today_players },
        { label: '战斗总数', val: s.battles },
        { label: '今日战斗', val: s.today_battles }
      ]
    },
    moneyCards () {
      const s = this.stats
      return [
        { label: '流通银两', val: s.money },
        { label: '银行存款', val: s.bank },
        { label: '金豆总量', val: s.beans },
        { label: '帮派 / 宠物', val: s.gangs + ' / ' + s.pets }
      ]
    }
  },
  mounted () { this.loadStats(); this.loadServer() },
  methods: {
    loadServer () {
      this.loadingSrv = true
      api.get('/admin/xy-server').then(r => {
        this.loadingSrv = false
        if (r.code === 0) {
          this.srv = r.data
          this.srvForm = { on: r.data.maintenance, notice: r.data.notice || '' }
        } else this.$message.error(r.msg)
      })
    },
    saveServer () {
      const act = this.srvForm.on ? '开启维护' : '开放服务器'
      this.$confirm('确认' + act + '？', '服务器维护', { type: 'warning' }).then(() => {
        this.savingSrv = true
        api.post('/admin/xy-server/maintenance', this.srvForm).then(r => {
          this.savingSrv = false
          if (r.code === 0) { this.$message.success(r.data.msg || '已保存'); this.loadServer() }
          else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    loadStats () {
      this.loadingStats = true
      api.get('/admin/xy-stats').then(r => {
        this.loadingStats = false
        if (r.code === 0) this.stats = r.data
        else this.$message.error(r.msg)
      })
    },
    percent (cnt) {
      if (!this.stats || !this.stats.players) return '0%'
      return ((cnt / this.stats.players) * 100).toFixed(1) + '%'
    },
    doAnnounce () {
      if (!this.announce.trim()) { this.$message.warning('请输入公告内容'); return }
      this.sending = true
      api.post('/admin/xy-announce', { content: this.announce }).then(r => {
        this.sending = false
        if (r.code === 0) { this.$message.success(r.data.msg || '已发布'); this.announce = '' }
        else this.$message.error(r.msg)
      })
    },
    doGrantAll () {
      const kindName = { money: '银两', beans: '金豆', item: '物品' }[this.grantAll.kind]
      this.$confirm('确认为全服所有玩家发放 ' + kindName + '？该操作不可撤销！', '全服发放', { type: 'warning' }).then(() => {
        this.granting = true
        api.post('/admin/xy-grant-all', this.grantAll).then(r => {
          this.granting = false
          if (r.code === 0) { this.$message.success(r.data.msg || '已发放'); this.loadStats() }
          else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.card-head { display: flex; justify-content: space-between; align-items: center; }
.stat-card { background: #f5f7fa; border-radius: 6px; padding: 14px 10px; text-align: center; }
.stat-money { background: #fdf6ec; }
.stat-num { font-size: 22px; font-weight: 700; color: #303133; }
.stat-label { font-size: 12px; color: #909399; margin-top: 4px; }
.sub-title { font-size: 13px; font-weight: 600; color: #1f2d3d; margin: 16px 0 8px; padding-left: 6px; border-left: 3px solid #409eff; }
</style>
