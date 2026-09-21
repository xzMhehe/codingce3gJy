<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <span>二战风云 · 宣战管理（待生效 / 交战中 / 已结束）</span>
        <el-button size="mini" type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>

      <div class="toolbar">
        <el-input v-model="word" placeholder="按昵称 / 家园号码搜索" clearable style="width:220px"
                  @keyup.enter.native="page = 1; load()" />
        <el-select v-model="status" style="width:150px" @change="page = 1; load()">
          <el-option :value="0" label="全部状态" />
          <el-option :value="1" label="宣战待生效" />
          <el-option :value="2" label="交战中" />
          <el-option :value="3" label="已结束" />
        </el-select>
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <el-button type="success" icon="el-icon-plus" @click="openAdd">新建宣战</el-button>
        <div class="grow" />
        <el-button type="warning" plain icon="el-icon-alarm-clock" :loading="acting" @click="effectAll">一键生效全部</el-button>
        <el-button type="danger" plain icon="el-icon-circle-close" :loading="acting" @click="finishAll">一键完成全部</el-button>
      </div>

      <div class="td-sub" style="margin:4px 0 10px">
        规则：宣战 <b>{{ delayHours }}</b> 小时后自动生效，生效后持续 <b>{{ durationHours }}</b> 小时；
        「一键生效」可跳过等待直接开战，「一键完成」立即结束战争（双方恢复和平）。
      </div>

      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="70" align="center" />
        <el-table-column label="宣战方" min-width="150" show-overflow-tooltip>
          <template slot-scope="{row}">
            <span class="td-main">{{ row.atk_nick || '—' }}</span>
            <span class="td-sub"> #{{ row.atk_num || row.atk_user_id }}</span>
          </template>
        </el-table-column>
        <el-table-column label="被宣战方" min-width="150" show-overflow-tooltip>
          <template slot-scope="{row}">
            <span class="td-main">{{ row.def_nick || '—' }}</span>
            <span class="td-sub"> #{{ row.def_num || row.def_user_id }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110" align="center">
          <template slot-scope="{row}">
            <el-tag :type="statusTag(row.live_status)" size="mini">{{ row.state_text }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="剩余" width="130" align="center">
          <template slot-scope="{row}"><span class="td-sub">{{ row.left_text }}</span></template>
        </el-table-column>
        <el-table-column label="宣战时间" width="150" align="center">
          <template slot-scope="{row}"><span class="td-sub">{{ row.declare_at_text }}</span></template>
        </el-table-column>
        <el-table-column label="生效时间" width="150" align="center">
          <template slot-scope="{row}"><span class="td-sub">{{ row.effect_at_text }}</span></template>
        </el-table-column>
        <el-table-column label="到期时间" width="150" align="center">
          <template slot-scope="{row}"><span class="td-sub">{{ row.expire_at_text }}</span></template>
        </el-table-column>
        <el-table-column label="操作" width="200" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button v-if="row.live_status === 1" size="mini" type="warning" plain
                       icon="el-icon-alarm-clock" title="立即生效（跳过等待）" @click="doEffect(row)" />
            <el-button v-if="row.live_status !== 3" size="mini" type="success" plain
                       icon="el-icon-check" title="一键完成（结束战争）" @click="doFinish(row)" />
            <el-button size="mini" type="primary" plain icon="el-icon-edit"
                       title="调整时间" @click="openEdit(row)" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete"
                       title="删除记录" @click="remove(row)" />
          </template>
        </el-table-column>
      </el-table>

      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ total }}</b> 条 · 每页 {{ size }} 条</div>
        <el-pagination v-show="total > 0" small background layout="sizes, prev, pager, next, jumper" :total="total" :page-size="size"
                       :current-page="page" :page-sizes="[10, 20, 50, 100]"
                       @current-change="p => { page = p; load() }"
                       @size-change="s => { size = s; page = 1; load() }" />
      </div>
    </el-card>

    <el-dialog title="新建宣战" :visible.sync="addDlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="140px" size="small">
        <el-form-item label="宣战方玩家ID">
          <el-input v-model.number="form.atk_user_id" placeholder="家园玩家ID（users.id）" />
        </el-form-item>
        <el-form-item label="被宣战方玩家ID">
          <el-input v-model.number="form.def_user_id" placeholder="家园玩家ID（users.id）" />
        </el-form-item>
        <el-form-item label="立即生效">
          <el-switch v-model="form.instant" />
          <span class="td-sub" style="margin-left:8px">开启则跳过等待，双方马上可以互相掠夺/征服</span>
        </el-form-item>
        <template v-if="!form.instant">
          <el-form-item label="生效延迟(小时)">
            <el-input-number v-model="form.delay_hours" :min="0" :max="720" controls-position="right" />
            <span class="td-sub" style="margin-left:8px">默认 {{ delayHours }} 小时</span>
          </el-form-item>
        </template>
        <el-form-item label="持续(小时)">
          <el-input-number v-model="form.keep_hours" :min="1" :max="8760" controls-position="right" />
          <span class="td-sub" style="margin-left:8px">默认 {{ durationHours }} 小时</span>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="addDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doAdd">创 建</el-button>
      </div>
    </el-dialog>

    <el-dialog title="调整宣战时间" :visible.sync="editDlg" width="520px" :close-on-click-modal="false">
      <el-form label-width="140px" size="small">
        <el-form-item label="生效延迟(小时)">
          <el-input-number v-model="editForm.delay_hours" :min="0" :max="720" controls-position="right" />
          <span class="td-sub" style="margin-left:8px">从现在起算</span>
        </el-form-item>
        <el-form-item label="持续(小时)">
          <el-input-number v-model="editForm.keep_hours" :min="1" :max="8760" controls-position="right" />
          <span class="td-sub" style="margin-left:8px">从生效时刻起算</span>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="editDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doEdit">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminEzfyWars',
  data () {
    return {
      list: [], total: 0, page: 1, size: 10, word: '', status: 0,
      loading: false, saving: false, acting: false,
      delayHours: 24, durationHours: 48,
      addDlg: false,
      form: { atk_user_id: '', def_user_id: '', instant: false, delay_hours: 24, keep_hours: 48 },
      editDlg: false, editId: 0, editForm: { delay_hours: 24, keep_hours: 48 }
    }
  },
  mounted () { this.load() },
  methods: {
    statusTag (s) {
      if (s === 1) return 'warning'
      if (s === 2) return 'danger'
      return 'info'
    },
    load () {
      this.loading = true
      api.get('/admin/ezfy-wars', {
        params: { page: this.page, size: this.size, word: this.word, status: this.status }
      }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list || []
          this.total = r.data.total || 0
          if (r.data.delay_hours) this.delayHours = r.data.delay_hours
          if (r.data.duration_hours) this.durationHours = r.data.duration_hours
        } else this.$message.error(r.msg)
      }).catch(() => { this.loading = false })
    },
    openAdd () {
      this.form = { atk_user_id: '', def_user_id: '', instant: false, delay_hours: this.delayHours, keep_hours: this.durationHours }
      this.addDlg = true
    },
    doAdd () {
      if (!this.form.atk_user_id || !this.form.def_user_id) { this.$message.error('请填写双方玩家ID'); return }
      this.saving = true
      api.post('/admin/ezfy-wars', this.form).then(r => {
        this.saving = false
        if (r.code === 0) { this.addDlg = false; this.$message.success(r.data.msg || '已创建'); this.load() }
        else this.$message.error(r.msg)
      }).catch(() => { this.saving = false })
    },
    openEdit (row) {
      this.editId = row.id
      this.editForm = { delay_hours: 0, keep_hours: this.durationHours }
      this.editDlg = true
    },
    doEdit () {
      this.saving = true
      api.put('/admin/ezfy-wars/' + this.editId, this.editForm).then(r => {
        this.saving = false
        if (r.code === 0) { this.editDlg = false; this.$message.success('已保存'); this.load() }
        else this.$message.error(r.msg)
      }).catch(() => { this.saving = false })
    },
    doEffect (row) {
      this.$confirm('让「' + (row.atk_nick || row.atk_user_id) + ' → ' + (row.def_nick || row.def_user_id) + '」立即进入交战状态？', '一键生效', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-wars/' + row.id + '/effect').then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已生效'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    doFinish (row) {
      this.$confirm('结束「' + (row.atk_nick || row.atk_user_id) + ' → ' + (row.def_nick || row.def_user_id) + '」的战争状态？双方将恢复和平。', '一键完成', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-wars/' + row.id + '/finish').then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已结束'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    effectAll () {
      this.$confirm('让当前所有「待生效」的宣战立即生效？', '一键生效全部', { type: 'warning' }).then(() => {
        this.acting = true
        api.post('/admin/ezfy-wars/effect-all').then(r => {
          this.acting = false
          if (r.code === 0) { this.$message.success(r.data.msg || '已生效'); this.load() } else this.$message.error(r.msg)
        }).catch(() => { this.acting = false })
      }).catch(() => {})
    },
    finishAll () {
      this.$confirm('结束当前全部进行中的宣战？所有相关玩家将恢复和平。', '一键完成全部', { type: 'warning' }).then(() => {
        this.acting = true
        api.post('/admin/ezfy-wars/finish-all').then(r => {
          this.acting = false
          if (r.code === 0) { this.$message.success(r.data.msg || '已结束'); this.load() } else this.$message.error(r.msg)
        }).catch(() => { this.acting = false })
      }).catch(() => {})
    },
    remove (row) {
      this.$confirm('删除宣战记录 #' + row.id + '？（仅删除记录，不做其他处理）', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-wars/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
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
