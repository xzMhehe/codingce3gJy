<template>
  <div class="farm-admin">
    <!-- 全局默认次数 -->
    <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <span>军校免费刷新次数（全局默认）</span>
        <el-button size="mini" type="primary" plain icon="el-icon-refresh" @click="loadAll">刷新</el-button>
      </div>
      <div class="toolbar">
        <span class="td-sub">每日免费刷新次数：</span>
        <el-input-number v-model.number="globalLimit" :min="0" :max="999" controls-position="right" style="width:150px" />
        <el-button type="primary" icon="el-icon-check" :loading="saving" @click="saveGlobal">保存全局默认</el-button>
        <div class="grow" />
        <span class="td-sub">
          玩家覆盖为 0 表示跟随全局默认；次数用完后，用户端可在
          <b>军校直接使用「招生简章」</b>刷新（不用跳背包），每次消耗 1 张
        </span>
      </div>
    </el-card>

    <!-- 玩家覆盖 -->
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="昵称 / 用户ID / 游戏ID" clearable style="width:220px"
                  @keyup.enter.native="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <el-button type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="user_id" label="用户ID" width="80" align="center" />
        <el-table-column prop="game_uid" label="游戏ID" width="100" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ row.game_uid || row.user_id }}</span></template>
        </el-table-column>
        <el-table-column prop="home_num" label="家园号码" width="100" align="center" />
        <el-table-column label="玩家" min-width="130" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-main">{{ row.nickname || row.player_name || '—' }}</span></template>
        </el-table-column>
        <el-table-column label="生效次数/天" width="115" align="center">
          <template slot-scope="{row}">
            <span class="td-mono">{{ row.limit }}</span>
            <el-tag v-if="row.override > 0" size="mini" type="warning" style="margin-left:4px">覆盖</el-tag>
            <el-tag v-else size="mini" type="info" style="margin-left:4px">默认</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="今日已用" width="95" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ row.used_today }}</span></template>
        </el-table-column>
        <el-table-column label="今日剩余" width="95" align="center">
          <template slot-scope="{row}">
            <span :class="row.left_today > 0 ? 'td-mono' : 'td-muted'">{{ row.left_today }}</span>
          </template>
        </el-table-column>
        <el-table-column label="招生简章" width="95" align="center">
          <template slot-scope="{row}">
            <span :class="row.ticket_count > 0 ? 'td-blue' : 'td-muted'">{{ row.ticket_count }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="230" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain icon="el-icon-edit" title="单独设置" @click="openSet(row)" />
            <el-button size="mini" type="warning" plain icon="el-icon-refresh-left" title="重置今日已用次数"
                       @click="resetToday(row)" />
            <el-button size="mini" type="success" plain icon="el-icon-present" title="发招生简章" @click="giveTicket(row)" />
          </template>
        </el-table-column>
      </el-table>
      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ total }}</b> 条 · 每页 {{ size }} 条</div>
        <el-pagination v-show="total > 0" small background layout="sizes, prev, pager, next, jumper" :total="total" :page-size="size"
                       :current-page="page" :page-sizes="[5, 10, 20, 50, 100]"
                       @current-change="p => { page = p; load() }"
                       @size-change="s => { size = s; page = 1; load() }" />
      </div>
    </el-card>

    <!-- 单独设置 -->
    <el-dialog :title="'设置军校刷新次数 · ' + (setRow.nickname || setRow.player_name || setRow.user_id)"
               :visible.sync="setDlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="130px" size="small">
        <el-form-item label="当前生效">
          <span class="td-main">{{ setRow.limit }} 次/天</span>
          <span class="td-sub">（今日已用 {{ setRow.used_today }}，剩余 {{ setRow.left_today }}）</span>
        </el-form-item>
        <el-form-item label="单独设置">
          <el-input-number v-model.number="setValue" :min="0" :max="999" controls-position="right" style="width:100%" />
          <span class="td-sub">填 0 = 跟随全局默认（当前 {{ globalLimit }} 次/天）</span>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="setDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doSet">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminEzfyRecruitLimit',
  data () {
    return {
      globalLimit: 5, saving: false,
      list: [], total: 0, page: 1, size: 5, word: '', loading: false,
      setDlg: false, setRow: {}, setValue: 0
    }
  },
  mounted () { this.loadAll() },
  methods: {
    loadAll () {
      api.get('/admin/ezfy-recruit-limit').then(r => {
        if (r.code === 0) this.globalLimit = r.data.global
      })
      this.load()
    },
    load () {
      this.loading = true
      api.get('/admin/ezfy-recruit-limit-users', {
        params: { page: this.page, size: this.size, word: this.word }
      }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list || []
          this.total = r.data.total || 0
          if (typeof r.data.global === 'number') this.globalLimit = r.data.global
        } else this.$message.error(r.msg)
      })
    },
    saveGlobal () {
      this.saving = true
      api.post('/admin/ezfy-recruit-limit', { value: this.globalLimit }).then(r => {
        this.saving = false
        if (r.code === 0) { this.$message.success(r.data.msg || '已保存'); this.load() } else this.$message.error(r.msg)
      })
    },
    openSet (row) {
      this.setRow = row
      this.setValue = row.override || 0
      this.setDlg = true
    },
    doSet () {
      this.saving = true
      api.put('/admin/ezfy-recruit-limit-users/' + this.setRow.user_id, { value: this.setValue }).then(r => {
        this.saving = false
        if (r.code === 0) { this.setDlg = false; this.$message.success(r.data.msg || '已保存'); this.load() }
        else this.$message.error(r.msg)
      })
    },
    resetToday (row) {
      this.$confirm('把「' + (row.nickname || row.user_id) + '」今日已用的刷新次数清零？', '提示', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-recruit-limit-users/' + row.user_id + '/reset', {}).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已重置'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    giveTicket (row) {
      this.$prompt('发放「招生简章」数量（用户端可在军校直接使用，不占每日次数）', '发放招生简章', {
        inputValue: '1',
        inputPattern: /^[1-9]\d{0,3}$/,
        inputErrorMessage: '请填写 1~9999 的整数'
      }).then(({ value }) => {
        // ★ 发放道具的载荷是 items: [{cfg_id, count}]（13 = 招生简章）
        api.post('/admin/ezfy-players/' + row.user_id + '/grant', { items: [{ cfg_id: 13, count: parseInt(value, 10) }] })
          .then(r => {
            if (r.code === 0) { this.$message.success(r.data.msg || '已发放'); this.load() } else this.$message.error(r.msg)
          })
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.card-head { display: flex; justify-content: space-between; align-items: center; }
</style>
