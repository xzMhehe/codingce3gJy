<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <!-- ★ 2026-10-17 卡顿优化：列表一次拉全量（称号本来就只有几条~几十条），
             搜索与翻页全部在本地完成 —— 点查询/翻页 0 请求、立即出结果。 -->
        <el-input v-model="word" placeholder="称号名称搜索（本地即时过滤）" clearable style="width:240px"
                  @keyup.enter.native="applyFilter" @clear="applyFilter" />
        <el-button type="primary" icon="el-icon-search" @click="applyFilter">查询</el-button>
        <el-button icon="el-icon-refresh" :loading="loading" @click="load">刷新</el-button>
        <div class="grow" />
        <span class="td-gray" style="font-size:12px">
          「天下无双」由<b>风云榜每日积分第一</b>自动获得（有效期取该称号配置）；其他称号由管理员在此授予
        </span>
        <el-button type="success" icon="el-icon-plus" @click="openEdit(null)">新增称号</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="70" align="center" />
        <el-table-column label="称号名称" min-width="170">
          <template slot-scope="{row}">
            <span class="ezfy-title-preview" :class="'t' + row.type">{{ row.name }}</span>
          </template>
        </el-table-column>
        <el-table-column label="类型" width="90" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="typeTag(row.type)">{{ row.type_name }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="valid_text" label="有效期" width="90" align="center" />
        <el-table-column label="称号创建日期" width="170" align="center">
          <template slot-scope="{row}">{{ row.created_at }}</template>
        </el-table-column>
        <el-table-column label="已有称号玩家" width="160" align="center">
          <template slot-scope="{row}">
            <a href="javascript:;" class="link-num" @click="openPlayers(row)">
              {{ row.holder_active }} 人
              <span class="td-gray">（累计 {{ row.holder_total }}）</span>
            </a>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.status ? 'success' : 'info'">{{ row.status ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="备注" min-width="150" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-gray">{{ row.remark || '—' }}</span></template>
        </el-table-column>
        <el-table-column label="操作" width="215" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain icon="el-icon-present" title="授予玩家" @click="openGrant(row)" />
            <el-button size="mini" plain icon="el-icon-user" title="已有称号玩家" @click="openPlayers(row)" />
            <el-button size="mini" plain icon="el-icon-edit" title="编辑称号" @click="openEdit(row)" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除称号" @click="del(row)" />
          </template>
        </el-table-column>
      </el-table>
      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ total }}</b> 条 · 每页 {{ size }} 条</div>
        <el-pagination v-show="total > 0" small background layout="sizes, prev, pager, next, jumper" :total="total"
                       :page-size="size" :current-page="page" :page-sizes="[10, 20, 50, 100]"
                       @current-change="p => { page = p; slicePage() }"
                       @size-change="s => { size = s; page = 1; slicePage() }" />
      </div>
    </el-card>

    <!-- 新增 / 编辑称号 -->
    <el-dialog :title="form.id ? '编辑称号' : '新增称号'" :visible.sync="editDlg" width="520px" :close-on-click-modal="false">
      <el-form label-width="110px">
        <el-form-item label="称号名称">
          <el-input v-model="form.name" maxlength="12" show-word-limit placeholder="如：天下无双、三英战吕布" style="width:260px" />
        </el-form-item>
        <el-form-item label="称号类型">
          <el-select v-model="form.type" style="width:180px">
            <el-option v-for="t in types" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
          <span class="td-gray" style="margin-left:8px">决定玩家端徽章配色</span>
        </el-form-item>
        <el-form-item label="称号有效期">
          <el-input-number v-model.number="form.valid_days" :min="0" :max="3650" :step="1" style="width:160px" />
          <span class="td-gray" style="margin-left:8px">天（填 0 = 永久）</span>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" maxlength="100" placeholder="获得条件/说明，仅后台可见" style="width:330px" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" active-text="启用" inactive-text="停用" />
          <div class="td-gray" style="font-size:12px">停用后：玩家端不再展示该称号，风云榜自动发放也会跳过</div>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="editDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保 存</el-button>
      </div>
    </el-dialog>

    <!-- 授予玩家 -->
    <el-dialog :title="'授予称号 —— ' + (cur ? cur.name : '')" :visible.sync="grantDlg" width="480px" :close-on-click-modal="false">
      <el-form label-width="110px">
        <el-form-item label="玩家">
          <el-input v-model="grantWord" placeholder="用户ID 或 游戏号" style="width:230px" />
          <div class="td-gray" style="font-size:12px">填玩家的用户ID或游戏号（游戏里看到的号码）</div>
        </el-form-item>
        <el-form-item label="有效期">
          <el-input-number v-model.number="grantDays" :min="0" :max="3650" :step="1" style="width:160px" />
          <span class="td-gray" style="margin-left:8px">天（0 = 永久）</span>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="grantDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doGrant">授 予</el-button>
      </div>
    </el-dialog>

    <!-- 已有称号玩家 -->
    <el-dialog :title="'已有称号玩家 —— ' + (cur ? cur.name : '')" :visible.sync="playersDlg" width="860px" :close-on-click-modal="false">
      <div class="toolbar" style="margin-bottom:8px">
        <el-input v-model="playerWord" placeholder="昵称 / 用户ID / 游戏号" clearable style="width:230px"
                  @keyup.enter.native="playerPage = 1; loadPlayers()" />
        <el-checkbox v-model="playerOnly" style="margin-left:12px" @change="playerPage = 1; loadPlayers()">只看生效中</el-checkbox>
        <el-button type="primary" icon="el-icon-search" @click="playerPage = 1; loadPlayers()">查询</el-button>
        <div class="grow" />
        <span class="td-gray" style="font-size:12px">共 {{ playerTotal }} 人</span>
      </div>
      <el-table :data="players" v-loading="playerLoading" stripe border max-height="420">
        <el-table-column prop="user_id" label="用户ID" width="90" align="center" />
        <el-table-column label="游戏号" width="100" align="center">
          <template slot-scope="{row}">{{ row.game_uid || '—' }}</template>
        </el-table-column>
        <el-table-column prop="nickname" label="昵称" min-width="140" show-overflow-tooltip />
        <el-table-column prop="source_name" label="来源" width="130" align="center" />
        <el-table-column prop="created_at" label="获得时间" width="170" align="center" />
        <el-table-column label="到期时间" width="175" align="center">
          <template slot-scope="{row}">
            <span :class="{ 'td-gray': row.expired }">{{ row.expire_text }}</span>
            <el-tag v-if="row.expired" size="mini" type="info" style="margin-left:4px">已过期</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="90" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="danger" plain icon="el-icon-delete" title="撤销该称号" @click="revoke(row)" />
          </template>
        </el-table-column>
      </el-table>
      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ playerTotal }}</b> 条</div>
        <el-pagination v-show="playerTotal > 0" small background layout="prev, pager, next" :total="playerTotal"
                       :page-size="playerSize" :current-page="playerPage"
                       @current-change="p => { playerPage = p; loadPlayers() }" />
      </div>
      <div slot="footer">
        <el-button @click="playersDlg = false">关 闭</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

// ★ 2026-10-17 统帅称号（管理端）
// 用户需求：「二战管理端加个模块 统帅称号，称号管理员可添加、称号名称、称号类型、称号有效期、
//          称号创建日期、已有称号玩家」。
// 称号定义增删改查 + 手动授予/撤销 + 「已有称号玩家」列表。
// 玩家端展示（昵称下方、阵营上方）与风云榜每日第一自动发放见后端 ezfy_title.go。
export default {
  name: 'AdminEzfyTitles',
  data () {
    return {
      // all = 服务端一次拉回的全量（≤500），filtered = 本地搜索后的结果，list = 当前页切片
      all: [], filtered: [], list: [], total: 0, page: 1, size: 10, loading: false,
      word: '', types: [],
      saving: false,
      // 新增/编辑
      editDlg: false,
      form: { id: 0, name: '', type: 1, valid_days: 1, remark: '', status: 1 },
      // 授予
      grantDlg: false, grantWord: '', grantDays: 1,
      // 已有称号玩家
      playersDlg: false, players: [], playerTotal: 0, playerPage: 1, playerSize: 10,
      playerLoading: false, playerWord: '', playerOnly: false,
      cur: null
    }
  },
  mounted () { this.load() },
  methods: {
    typeTag (t) {
      return ({ 1: 'danger', 2: 'warning', 3: '', 4: 'info' })[t] || 'info'
    },
    // ★ 2026-10-17 卡顿优化：一次拉全量（服务端已把「列表 + 持有人数」合成一条 SQL），
    //   搜索/翻页都在本地算 —— 只有「进页面」和「刷新」发请求。
    load () {
      this.loading = true
      api.get('/admin/ezfy-titles').then(r => {
        this.loading = false
        if (r.code === 0) {
          this.all = r.data.list || []
          this.types = r.data.types || []
          this.applyFilter()
        } else this.$message.error(r.msg)
      }).catch(() => { this.loading = false })
    },
    // 本地过滤（瞬时）：名称包含关键词即可
    applyFilter () {
      const w = (this.word || '').trim()
      this.filtered = w ? this.all.filter(t => (t.name || '').indexOf(w) >= 0) : this.all
      this.total = this.filtered.length
      this.page = 1
      this.slicePage()
    },
    // 本地切页（瞬时）
    slicePage () {
      const src = this.filtered || this.all
      const start = (this.page - 1) * this.size
      this.list = src.slice(start, start + this.size)
    },
    openEdit (row) {
      if (row) {
        this.form = {
          id: row.id, name: row.name, type: row.type,
          valid_days: row.valid_days, remark: row.remark || '', status: row.status
        }
      } else {
        this.form = { id: 0, name: '', type: 1, valid_days: 1, remark: '', status: 1 }
      }
      this.editDlg = true
    },
    save () {
      const f = this.form
      if (!f.name || !f.name.trim()) { this.$message.warning('请填写称号名称'); return }
      this.saving = true
      api.post('/admin/ezfy-titles', {
        id: f.id, name: f.name.trim(), type: f.type,
        valid_days: f.valid_days, remark: f.remark, status: f.status
      }).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.editDlg = false
          this.$message.success(r.data.msg || '已保存')
          this.load()
        } else this.$message.error(r.msg)
      }).catch(() => { this.saving = false })
    },
    del (row) {
      this.$confirm('确认删除称号「' + row.name + '」？\n将同时清除该称号的全部持有记录（玩家立刻失去该称号），不可恢复。',
        '删除称号', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-titles/' + row.id).then(r => {
          if (r.code === 0) {
            this.$message.success(r.data.msg || '已删除')
            this.load()
          } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    openGrant (row) {
      this.cur = row
      this.grantWord = ''
      this.grantDays = row.valid_days
      this.grantDlg = true
    },
    doGrant () {
      const w = (this.grantWord || '').trim()
      if (!/^\d+$/.test(w) || Number(w) <= 0) { this.$message.warning('请填写玩家用户ID或游戏号（纯数字）'); return }
      this.saving = true
      api.post('/admin/ezfy-titles/' + this.cur.id + '/grant', {
        user_id: Number(w), days: this.grantDays
      }).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.grantDlg = false
          this.$message.success(r.data.msg || '已授予')
          this.load()
        } else this.$message.error(r.msg)
      }).catch(() => { this.saving = false })
    },
    openPlayers (row) {
      this.cur = row
      this.players = []
      this.playerTotal = 0
      this.playerPage = 1
      this.playerWord = ''
      this.playerOnly = false
      this.playersDlg = true
      this.loadPlayers()
    },
    loadPlayers () {
      if (!this.cur) return
      this.playerLoading = true
      api.get('/admin/ezfy-titles/' + this.cur.id + '/players', {
        params: {
          page: this.playerPage, size: this.playerSize,
          word: this.playerWord, only: this.playerOnly ? '1' : '0'
        }
      }).then(r => {
        this.playerLoading = false
        if (r.code === 0) {
          this.players = r.data.list || []
          this.playerTotal = r.data.total
          this.playerPage = r.data.page
        } else this.$message.error(r.msg)
      }).catch(() => { this.playerLoading = false })
    },
    revoke (row) {
      this.$confirm('确认撤销「' + row.nickname + '」的称号「' + this.cur.name + '」？\n撤销后玩家立刻失去该称号。',
        '撤销称号', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-title-grants/' + row.id).then(r => {
          if (r.code === 0) {
            this.$message.success(r.data.msg || '已撤销')
            this.loadPlayers()
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

/* 列表里的称号预览：与玩家端徽章同款配色，管理员一眼看出玩家看到的样子 */
.ezfy-title-preview {
  display: inline-block;
  padding: 0 7px 1px;
  line-height: 19px;
  font-size: 12px;
  font-weight: bold;
  letter-spacing: 2px;
  border-radius: 2px;
  color: #ffe6a8;
  background: linear-gradient(180deg, #b8342a 0%, #8e1f18 100%);
  border: 1px solid #d9a53c;
  text-shadow: 0 1px 1px rgba(0, 0, 0, .5);
}
.ezfy-title-preview.t1 { background: linear-gradient(180deg, #b8342a 0%, #8e1f18 100%); }
.ezfy-title-preview.t2 { background: linear-gradient(180deg, #7a2a86 0%, #4a1055 100%); }
.ezfy-title-preview.t3 { background: linear-gradient(180deg, #1f5f8b 0%, #123a58 100%); }
.ezfy-title-preview.t4 { background: linear-gradient(180deg, #5c6b7a 0%, #37424d 100%); }
.link-num { color: #409eff; text-decoration: none; }
.link-num:hover { text-decoration: underline; }
</style>
