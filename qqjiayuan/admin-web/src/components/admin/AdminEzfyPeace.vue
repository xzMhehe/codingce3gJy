<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="玩家昵称 / 用户ID / 游戏号搜索" clearable style="width:250px"
                  @keyup.enter.native="page = 1; load()" />
        <el-checkbox v-model="only" style="margin-left:12px" @change="page = 1; load()">只看在免战</el-checkbox>
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <span class="td-gray" style="font-size:12px">开始免战 = 给该玩家<b>全部城池</b>打免战保护（管理端不受冷却限制）；结束 = 清除全部城池的免战</span>
      </div>
      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="user_id" label="用户ID" width="90" align="center" />
        <!-- ★ 2026-10-26 用户要求「分页家园号列改成游戏ID」：
             游戏号（ezfy_profile.game_uid）才是玩家看得见的号；老数据 game_uid=0 时回落家园号，不留空白 -->
        <el-table-column label="游戏ID" width="110" align="center">
          <template slot-scope="{row}">
            <span class="td-mono">{{ row.game_uid || row.home_num || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="昵称" min-width="150" show-overflow-tooltip>
          <template slot-scope="{row}">
            <span class="td-main">{{ row.nickname || row.home_nick || '—' }}</span>
            <span v-if="row.home_nick && row.home_nick !== row.nickname" class="td-gray">（{{ row.home_nick }}）</span>
          </template>
        </el-table-column>
        <el-table-column label="阵营" width="80" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.camp === 2 ? 'danger' : 'primary'">{{ row.camp_name }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="军功声望" width="100" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ row.prestige }}</span></template>
        </el-table-column>
        <el-table-column label="免战状态" width="110" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.peac_active ? 'warning' : 'success'">{{ row.peace_active ? '免战中' : '未免战' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="免战到期" width="170" align="center">
          <template slot-scope="{row}">{{ row.peace_active ? fmtTime(row.peace_expire_at) : '—' }}</template>
        </el-table-column>
        <el-table-column label="免战城数" width="90" align="center">
          <template slot-scope="{row}">{{ row.peace_active ? row.peace_city_count : '—' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="175" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="warning" plain icon="el-icon-lock" :disabled="row.peace_active"
                       title="开始免战（全城）" @click="openStart(row)" />
            <el-button size="mini" type="danger" plain icon="el-icon-unlock" :disabled="!row.peace_active"
                       title="结束免战（清除全部城池）" @click="end(row)" />
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

    <!-- 开始免战 -->
    <el-dialog :title="'开始免战 —— ' + (targetName || '')" :visible.sync="startDlg" width="460px" :close-on-click-modal="false">
      <div class="td-gray" style="line-height:1.8; margin-bottom:10px">
        将给「{{ targetName }}」的<b>全部 {{ targetCityCount }} 座城池</b>开启免战保护，期间该玩家不会被掠夺/征服。<br/>
        ★ 管理端操作<b>不受玩家免战使用冷却限制</b>，直接生效并覆盖原有到期时间。
      </div>
      <el-form label-width="90px">
        <el-form-item label="生效时长">
          <el-input-number v-model.number="hours" :min="1" :max="720" :step="1" style="width:180px" />
          <span class="td-gray" style="margin-left:8px">小时（1~720）</span>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="startDlg = false">取 消</el-button>
        <el-button type="warning" :loading="saving" @click="doStart">开始免战</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

// ★ 2026-10-12 免战管理：管理免战用户数据（开始/结束免战，管理端不受冷却限制）。
// 列表 / 开始（全城免战） / 结束（清除全部城池免战）。
// 后端接口：/admin/ezfy-peace（列表）、/:id/start（开始，可带 hours）、/:id（DELETE 结束）
export default {
  name: 'AdminEzfyPeace',
  data () {
    return {
      list: [], total: 0, page: 1, size: 10, loading: false,
      word: '', only: false,
      startDlg: false, saving: false, targetId: 0, targetName: '', targetCityCount: 0, hours: 8
    }
  },
  mounted () { this.load() },
  methods: {
    fmtTime (t) { return t ? new Date(t).toLocaleString() : '' },
    load () {
      this.loading = true
      api.get('/admin/ezfy-peace', {
        params: { page: this.page, size: this.size, word: this.word, only: this.only ? '1' : '0' }
      }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list || []
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      }).catch(() => { this.loading = false })
    },
    openStart (row) {
      this.targetId = row.user_id
      this.targetName = row.nickname || row.home_nick || ('玩家' + row.user_id)
      this.targetCityCount = row.city_count
      this.hours = 8
      this.startDlg = true
    },
    doStart () {
      if (!this.hours || this.hours < 1) { this.$message.warning('请填写大于 0 的时长'); return }
      this.$confirm('确认给「' + this.targetName + '」开启免战 ' + this.hours + ' 小时？', '开始免战', { type: 'warning' }).then(() => {
        this.saving = true
        api.post('/admin/ezfy-peace/' + this.targetId + '/start', null, { params: { hours: this.hours } }).then(r => {
          this.saving = false
          if (r.code === 0) {
            this.startDlg = false
            this.$message.success(r.data.msg || '已开启免战')
            this.load()
          } else this.$message.error(r.msg)
        }).catch(() => { this.saving = false })
      }).catch(() => {})
    },
    end (row) {
      this.$confirm('确认结束「' + (row.nickname || row.home_nick || row.user_id) + '」的免战？\n将清除该玩家全部城池的免战保护，之后可被打。', '结束免战', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-peace/' + row.user_id).then(r => {
          if (r.code === 0) {
            this.$message.success(r.data.msg || '已结束免战')
            this.load()
          } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
</style>