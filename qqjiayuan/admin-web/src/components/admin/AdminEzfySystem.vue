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
      <em>提示：开启维护后，玩家将无法进入二战风云（游戏内所有操作均被拦截并显示维护公告）</em>
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
        <el-col :span="6" v-for="c in resCards" :key="c.label">
          <div class="stat-card stat-money">
            <div class="stat-num">{{ c.val }}</div>
            <div class="stat-label">{{ c.label }}</div>
          </div>
        </el-col>
      </el-row>
      <el-row :gutter="14" v-if="stats && stats.tops && stats.tops.length" style="margin-top:14px">
        <el-col :span="12">
          <div class="sub-title">军衔声望排行（TOP10）</div>
          <el-table :data="stats.tops" size="mini" border max-height="300">
            <el-table-column type="index" label="#" width="50" align="center" />
            <el-table-column prop="nickname" label="玩家" min-width="110" show-overflow-tooltip />
            <el-table-column label="阵营" width="90" align="center">
              <template slot-scope="{row}">{{ campNames[row.camp] || '同盟国' }}</template>
            </el-table-column>
            <el-table-column prop="prestige" label="声望" width="90" align="center" />
          </el-table>
        </el-col>
        <el-col :span="12">
          <div class="sub-title">阵营分布</div>
          <el-table :data="campRows" size="mini" border max-height="300">
            <el-table-column prop="name" label="阵营" width="110" align="center" />
            <el-table-column prop="cnt" label="人数" width="90" align="center" />
            <el-table-column label="占比" align="center">
              <template slot-scope="{row}">{{ percent(row.cnt) }}</template>
            </el-table-column>
          </el-table>
        </el-col>
      </el-row>
    </el-card>

    <el-row :gutter="14">
      <!-- 公告管理 -->
      <el-col :span="14">
        <el-card shadow="never" class="box">
          <div slot="header" class="card-head"><span>公告管理（游戏内公告栏 + 世界聊天同步）</span></div>
          <div class="toolbar">
            <el-input v-model="noticeForm.title" placeholder="公告标题" maxlength="100" style="width:200px" />
            <el-checkbox v-model="noticeForm.isTop">置顶</el-checkbox>
          </div>
          <el-input v-model="noticeForm.content" type="textarea" :rows="3" maxlength="2000" show-word-limit
                    placeholder="公告内容" style="margin-bottom:10px" />
          <div style="text-align:right;margin-bottom:10px">
            <el-button type="primary" size="small" :loading="sending" @click="doAnnounce">发 布</el-button>
          </div>
          <el-table :data="notices" v-loading="loadingNotices" size="mini" stripe border max-height="260">
            <el-table-column prop="id" label="ID" width="60" align="center" />
            <el-table-column prop="title" label="标题" min-width="120" show-overflow-tooltip>
              <template slot-scope="{row}">
                <el-tag v-if="row.is_top === 1" size="mini" type="warning" style="margin-right:4px">置顶</el-tag>
                <span class="td-main">{{ row.title }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="content" label="内容" min-width="200" show-overflow-tooltip />
            <el-table-column prop="created_at" label="时间" width="150" />
            <el-table-column label="操作" width="80" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="delNotice(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <!-- 军团管理 -->
      <el-col :span="10">
        <el-card shadow="never" class="box">
          <div slot="header" class="card-head">
            <span>军团管理</span>
            <el-button size="mini" type="primary" plain icon="el-icon-refresh" @click="loadCorps">刷新</el-button>
          </div>
          <div class="toolbar">
            <el-input v-model="corpsWord" placeholder="军团名搜索" clearable style="width:180px"
                      @keyup.enter.native="corpsPage = 1; loadCorps()" />
            <el-button type="primary" size="small" icon="el-icon-search" @click="corpsPage = 1; loadCorps()">查询</el-button>
          </div>
          <el-table :data="corps" v-loading="loadingCorps" size="mini" stripe border max-height="380">
            <el-table-column prop="id" label="ID" width="60" align="center" />
            <el-table-column prop="name" label="军团名" min-width="100" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column prop="leader_name" label="军团长" width="100" show-overflow-tooltip />
            <el-table-column prop="member_count" label="成员" width="60" align="center" />
            <el-table-column prop="created_at" label="创建时间" width="150" />
            <el-table-column label="操作" width="80" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="dissolve(row)">解散</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-pagination background layout="prev, pager, next" :total="corpsTotal"
                         :page-size="corpsSize" :current-page="corpsPage"
                         @current-change="p => { corpsPage = p; loadCorps() }" />
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminEzfySystem',
  data () {
    return {
      stats: null, loadingStats: false,
      srv: { maintenance: false, notice: '', players: 0, cities: 0 },
      srvForm: { on: false, notice: '' },
      loadingSrv: false, savingSrv: false,
      noticeForm: { title: '', content: '', isTop: true },
      notices: [], loadingNotices: false, sending: false,
      corps: [], corpsTotal: 0, corpsPage: 1, corpsSize: 10, corpsWord: '', loadingCorps: false,
      campNames: { 1: '同盟国', 2: '轴心国' }
    }
  },
  computed: {
    statCards () {
      const s = this.stats
      return [
        { label: '玩家总数', val: s.players },
        { label: '今日新建城池', val: s.today_players },
        { label: '城池总数', val: s.cities },
        { label: '军团 / 交易所挂单', val: s.corps + ' / ' + s.exchanges }
      ]
    },
    resCards () {
      const s = this.stats
      return [
        { label: '流通黄金', val: s.gold },
        { label: '粮食总量', val: s.food },
        { label: '钢铁总量', val: s.steel },
        { label: '总声望', val: s.prestige }
      ]
    },
    campRows () {
      if (!this.stats || !this.stats.camps) return []
      return this.stats.camps.map(r => ({ name: this.campNames[r.camp] || ('阵营' + r.camp), cnt: r.cnt }))
    }
  },
  mounted () { this.loadStats(); this.loadServer(); this.loadNotices(); this.loadCorps() },
  methods: {
    loadServer () {
      this.loadingSrv = true
      api.get('/admin/ezfy-server').then(r => {
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
        api.post('/admin/ezfy-server/maintenance', this.srvForm).then(r => {
          this.savingSrv = false
          if (r.code === 0) { this.$message.success(r.data.msg || '已保存'); this.loadServer() }
          else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    loadStats () {
      this.loadingStats = true
      api.get('/admin/ezfy-stats').then(r => {
        this.loadingStats = false
        if (r.code === 0) this.stats = r.data
        else this.$message.error(r.msg)
      })
    },
    percent (cnt) {
      if (!this.stats || !this.stats.players) return '0%'
      return ((cnt / this.stats.players) * 100).toFixed(1) + '%'
    },
    loadNotices () {
      this.loadingNotices = true
      api.get('/admin/ezfy-notices', { params: { page: 1, size: 20 } }).then(r => {
        this.loadingNotices = false
        if (r.code === 0) this.notices = r.data.list
        else this.$message.error(r.msg)
      })
    },
    doAnnounce () {
      if (!this.noticeForm.content.trim()) { this.$message.warning('请输入公告内容'); return }
      this.sending = true
      api.post('/admin/ezfy-announce', {
        title: this.noticeForm.title,
        content: this.noticeForm.content,
        is_top: this.noticeForm.isTop ? 1 : 0
      }).then(r => {
        this.sending = false
        if (r.code === 0) {
          this.$message.success(r.data.msg || '已发布')
          this.noticeForm = { title: '', content: '', isTop: true }
          this.loadNotices()
        } else this.$message.error(r.msg)
      })
    },
    delNotice (row) {
      this.$confirm('确认删除该条公告？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-notices/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadNotices() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    loadCorps () {
      this.loadingCorps = true
      api.get('/admin/ezfy-corps', { params: { page: this.corpsPage, size: this.corpsSize, word: this.corpsWord } }).then(r => {
        this.loadingCorps = false
        if (r.code === 0) {
          this.corps = r.data.list
          this.corpsTotal = r.data.total
          this.corpsPage = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    dissolve (row) {
      this.$confirm('解散军团将清除所有成员与军团聊天记录，确认解散「' + row.name + '」？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-corps/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已解散'); this.loadCorps(); this.loadStats() } else this.$message.error(r.msg)
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
.sub-title { font-size: 13px; font-weight: 600; color: #1f2d3d; margin: 0 0 8px; padding-left: 6px; border-left: 3px solid #409eff; }
</style>
