<template>
  <div class="farm-admin">
    <!-- 风控 / 生成 -->
    <el-card shadow="never" class="box" style="margin-bottom:12px">
      <div class="toolbar" style="flex-wrap:wrap; gap:10px">
        <span class="td-main">人机总数 <b>{{ list.length }}</b> / 上限 {{ max }}</span>
        <el-divider direction="vertical" />
        <span class="td-gray">允许攻击玩家</span>
        <el-switch v-model="attackOn" :active-value="1" :inactive-value="0" @change="saveConfig" />
        <span class="td-gray" style="margin-left:6px">每人机每日出征上限</span>
        <el-input-number v-model.number="atkDailyCap" :min="0" :max="100" size="small" style="width:100px"
                         @change="saveConfig" />
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="createDlg = true">生成人机</el-button>
        <el-button type="danger" plain icon="el-icon-delete" :disabled="!list.length" @click="clearAll">清空全部</el-button>
      </div>
      <div class="td-gray" style="font-size:12px; margin-top:6px">
        · 人机走<b>和真人完全相同的接口</b>（建城/升级/造兵/宣战/出征），不会做出玩家做不到的事；<br/>
        · 每 60 秒一轮、每轮最多 10 个，带租约锁（多进程只有一台真正执行）；<br/>
        · 人机默认<b>不出现在「玩家信息管理」</b>里，只在本页展示。
      </div>
    </el-card>

    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="昵称 / 用户ID 搜索" clearable style="width:220px"
                  @keyup.enter.native="load" @clear="load" />
        <el-button type="primary" icon="el-icon-search" @click="load">查询</el-button>
        <el-button icon="el-icon-refresh" :loading="loading" @click="load">刷新</el-button>
        <div class="grow" />
      </div>
      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="user_id" label="用户ID" width="95" align="center" />
        <el-table-column label="昵称" min-width="120" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-main">{{ row.nickname }}</span></template>
        </el-table-column>
        <el-table-column label="难度" width="100" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="levelTag(row.level)">{{ row.level_text }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="阵营" width="80" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.camp === 2 ? 'danger' : 'primary'">{{ row.camp_name }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="主城" min-width="120" show-overflow-tooltip>
          <template slot-scope="{row}">
            <span v-if="row.city_id">{{ row.city_name }}（{{ row.x }},{{ row.y }}）</span>
            <span v-else class="td-gray">—</span>
          </template>
        </el-table-column>
        <el-table-column label="市政厅" width="80" align="center">
          <template slot-scope="{row}">{{ row.city_level || '—' }}</template>
        </el-table-column>
        <el-table-column label="兵力" width="110" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ fmtNum(row.troop_total) }}</span></template>
        </el-table-column>
        <el-table-column label="声望" width="100" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ fmtNum(row.prestige) }}</span></template>
        </el-table-column>
        <el-table-column label="最后行动" width="170" align="center">
          <template slot-scope="{row}">
            <span v-if="row.last_act_at">{{ fmtTime(row.last_act_at) }}</span>
            <span v-else class="td-gray">未行动</span>
            <div class="td-gray" style="font-size:12px">{{ row.last_action || '' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="今日出征" width="90" align="center">
          <template slot-scope="{row}">{{ row.atk_count_today || 0 }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '运行中' : '已暂停' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="235" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain icon="el-icon-video-play" title="立即行动一轮"
                       @click="actNow(row)" />
            <el-button size="mini" plain icon="el-icon-tickets" title="行为日志" @click="openLogs(row)" />
            <el-button size="mini" plain icon="el-icon-s-operation" title="调难度" @click="openLevel(row)" />
            <el-button size="mini" :type="row.enabled ? 'warning' : 'success'" plain
                       :icon="row.enabled ? 'el-icon-video-pause' : 'el-icon-video-play'"
                       :title="row.enabled ? '暂停' : '启用'" @click="toggle(row)" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="del(row)" />
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 生成人机 -->
    <el-dialog title="生成人机" :visible.sync="createDlg" width="460px" :close-on-click-modal="false">
      <el-form label-width="100px">
        <el-form-item label="数量">
          <el-input-number v-model.number="createCount" :min="1" :max="10" style="width:150px" />
          <span class="td-gray" style="margin-left:8px">单次 1~10 个（剩余名额 {{ max - list.length }}）</span>
        </el-form-item>
        <el-form-item label="难度档">
          <el-radio-group v-model="createLevel">
            <el-radio :label="1">新手陪练</el-radio>
            <el-radio :label="2">普通</el-radio>
            <el-radio :label="3">精英</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="加速开局">
          <el-switch v-model="createBoost" />
          <span class="td-gray" style="margin-left:8px">给一笔启动资源（50万粮/20万钢油稀）</span>
          <div class="td-gray" style="font-size:12px; line-height:1.6">
            不开 = 严格从零开始（1 级市政厅 + 5000 资源），但攒够「召集人口」的 20 万粮要很久；
            开 = 运营初始化，让它当天就能召集人口、造兵、开始发育。
          </div>
        </el-form-item>
      </el-form>
      <div class="td-gray" style="font-size:12px; line-height:1.7; padding-left:100px">
        · 每个都会建家园号 + 二战账号 + 主城，从「1 级市政厅 + 5000 资源」开始自己发育；<br/>
        · 生成后约 1 分钟内开始第一轮行动（懒结算 → 建/升建筑 → 造兵）。
      </div>
      <div slot="footer">
        <el-button @click="createDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doCreate">生 成</el-button>
      </div>
    </el-dialog>

    <!-- 调难度 -->
    <el-dialog :title="'调整难度 —— ' + (cur ? cur.nickname : '')" :visible.sync="levelDlg" width="420px">
      <el-radio-group v-model="levelVal" style="padding-left:20px">
        <el-radio :label="1">新手陪练（慢、防守）</el-radio><br/>
        <el-radio :label="2">普通（均衡）</el-radio><br/>
        <el-radio :label="3">精英（快、侵略性高）</el-radio>
      </el-radio-group>
      <div slot="footer">
        <el-button @click="levelDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="saveLevel">确 定</el-button>
      </div>
    </el-dialog>

    <!-- 行为日志 -->
    <el-dialog :title="'行为日志 —— ' + (cur ? cur.nickname : '全部')" :visible.sync="logsDlg" width="820px">
      <el-table :data="logs" v-loading="logsLoading" size="mini" border max-height="420">
        <el-table-column prop="created_at" label="时间" width="170" align="center" />
        <el-table-column prop="nickname" label="人机" width="120" show-overflow-tooltip />
        <el-table-column prop="action" label="动作" width="110" align="center" />
        <el-table-column label="结果" width="80" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.result ? 'success' : 'info'">{{ row.result ? '成功' : '失败/跳过' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="detail" label="详情" min-width="300" show-overflow-tooltip />
      </el-table>
      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ logTotal }}</b> 条</div>
        <el-pagination v-show="logTotal > 0" small background layout="prev, pager, next" :total="logTotal"
                       :page-size="logSize" :current-page="logPage"
                       @current-change="p => { logPage = p; loadLogs() }" />
      </div>
      <div slot="footer"><el-button @click="logsDlg = false">关 闭</el-button></div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

// ★ 2026-10-17 人机（AI 玩家）管理 —— 与「玩家信息管理」完全独立的模块。
// 列表/生成/清空/启停/难度/行为日志/风控开关。
// 后端：/api/admin/ezfy-bot/*（权限 module:ezfyBot）
export default {
  name: 'AdminEzfyBot',
  data () {
    return {
      list: [], max: 50, loading: false, word: '',
      attackOn: 1, atkDailyCap: 3, saving: false,
      createDlg: false, createCount: 3, createLevel: 2, createBoost: true,
      levelDlg: false, levelVal: 2, cur: null,
      logsDlg: false, logs: [], logTotal: 0, logPage: 1, logSize: 20, logsLoading: false
    }
  },
  mounted () {
    this.load()
    this.loadConfig()
  },
  methods: {
    fmtNum (n) { return n == null ? 0 : Number(n).toLocaleString() },
    fmtTime (t) { return t ? new Date(t).toLocaleString() : '' },
    levelTag (l) { return ({ 1: 'info', 2: '', 3: 'danger' })[l] || '' },
    load () {
      this.loading = true
      api.get('/admin/ezfy-bot/list', { params: { word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list || []
          this.max = r.data.max || 50
          this.attackOn = r.data.attack_on ? 1 : 0
          this.atkDailyCap = r.data.atk_daily_cap || 0
        } else this.$message.error(r.msg)
      }).catch(() => { this.loading = false })
    },
    loadConfig () {
      api.get('/admin/ezfy-bot/config').then(r => {
        if (r.code === 0) {
          this.attackOn = r.data.attack_on ? 1 : 0
          this.atkDailyCap = r.data.atk_daily_cap
        }
      })
    },
    saveConfig () {
      api.post('/admin/ezfy-bot/config', {
        attack_on: this.attackOn, atk_daily_cap: this.atkDailyCap
      }).then(r => {
        if (r.code === 0) this.$message.success('已保存')
        else this.$message.error(r.msg)
      })
    },
    doCreate () {
      this.saving = true
      api.post('/admin/ezfy-bot/create', { count: this.createCount, level: this.createLevel, boost: this.createBoost }).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.createDlg = false
          this.$message.success(r.data.msg || '已生成')
          this.load()
        } else this.$message.error(r.msg)
      }).catch(() => { this.saving = false })
    },
    clearAll () {
      this.$confirm('确认清空全部人机？\n将物理删除每个人机的家园账号与二战全部数据（城池/部队/军官/背包/战报等），不可恢复。',
        '清空人机', { type: 'error' }).then(() => {
        api.post('/admin/ezfy-bot/clear').then(r => {
          if (r.code === 0) {
            this.$message.success(r.data.msg || '已清空')
            this.load()
          } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    toggle (row) {
      api.post('/admin/ezfy-bot/' + row.user_id + '/toggle', { enabled: row.enabled ? 0 : 1 }).then(r => {
        if (r.code === 0) { this.$message.success(r.data.msg || '已更新'); this.load() }
        else this.$message.error(r.msg)
      })
    },
    openLevel (row) {
      this.cur = row
      this.levelVal = row.level
      this.levelDlg = true
    },
    saveLevel () {
      this.saving = true
      api.post('/admin/ezfy-bot/' + this.cur.user_id + '/level', { level: this.levelVal }).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.levelDlg = false
          this.$message.success(r.data.msg || '已调整')
          this.load()
        } else this.$message.error(r.msg)
      }).catch(() => { this.saving = false })
    },
    actNow (row) {
      this.$confirm('让「' + row.nickname + '」立刻执行一轮决策？\n（会真实执行它这轮选中的动作：建/升建筑、造兵、宣战等）',
        '立即行动', { type: 'info' }).then(() => {
        api.post('/admin/ezfy-bot/' + row.user_id + '/act').then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已执行'); this.load() }
          else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    del (row) {
      this.$confirm('确认删除人机「' + row.nickname + '」？\n将物理删除它的家园账号与二战全部数据，不可恢复。',
        '删除人机', { type: 'error' }).then(() => {
        api.delete('/admin/ezfy-bot/' + row.user_id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.load() }
          else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    openLogs (row) {
      this.cur = row
      this.logs = []
      this.logTotal = 0
      this.logPage = 1
      this.logsDlg = true
      this.loadLogs()
    },
    loadLogs () {
      this.logsLoading = true
      api.get('/admin/ezfy-bot/logs', {
        params: { uid: this.cur ? this.cur.user_id : 0, page: this.logPage, size: this.logSize }
      }).then(r => {
        this.logsLoading = false
        if (r.code === 0) {
          this.logs = r.data.list || []
          this.logTotal = r.data.total
        }
      }).catch(() => { this.logsLoading = false })
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
</style>
