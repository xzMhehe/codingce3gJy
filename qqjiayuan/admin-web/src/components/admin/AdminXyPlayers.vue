<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="角色名 / 角色ID / 家园号搜索" clearable style="width:240px"
                  @keyup.enter.native="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <el-button type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="id" label="角色ID" width="80" align="center" />
        <el-table-column prop="user_id" label="家园号" width="80" align="center" />
        <el-table-column label="家园昵称" width="110" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.home_nick || '—' }}</template>
        </el-table-column>
        <el-table-column label="角色名" min-width="110" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column label="性别" width="60" align="center">
          <template slot-scope="{row}">{{ sexNames[row.sex] || '男' }}</template>
        </el-table-column>
        <el-table-column label="门派" width="90" align="center">
          <template slot-scope="{row}">{{ sectNames[row.sect] || '—' }}</template>
        </el-table-column>
        <el-table-column label="等级" width="70" align="center">
          <template slot-scope="{row}"><span class="lv">{{ row.level }}</span>级</template>
        </el-table-column>
        <el-table-column label="气血" width="90" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ row.hp }}</span>/{{ row.max_hp }}</template>
        </el-table-column>
        <el-table-column label="银两" width="100" align="center">
          <template slot-scope="{row}"><i class="el-icon-coin td-blue"></i>{{ row.money }}</template>
        </el-table-column>
        <el-table-column label="金豆" width="80" align="center">
          <template slot-scope="{row}"><i class="el-icon-goods td-gold"></i>{{ row.beans }}</template>
        </el-table-column>
        <el-table-column label="VIP" width="70" align="center">
          <template slot-scope="{row}">{{ row.vip }}分</template>
        </el-table-column>
        <el-table-column label="所在地" width="120" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.node_name || '—' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="110" align="center">
          <template slot-scope="{row}">
            <el-tag v-if="isBanned(row)" size="mini" type="danger">封号中</el-tag>
            <el-tag v-else-if="isMuted(row)" size="mini" type="warning">禁言中</el-tag>
            <el-tag v-else size="mini" type="success">正常</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="250" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="info" plain icon="el-icon-view" title="详情" @click="openDetail(row)" />
            <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openEdit(row)" />
            <el-button size="mini" type="success" plain icon="el-icon-present" title="发放" @click="openGrant(row)" />
            <el-button size="mini" type="warning" plain icon="el-icon-lock" title="封号/禁言" @click="openPunish(row)" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删号" @click="del(row)" />
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, sizes, prev, pager, next" :total="total"
                     :page-size="size" :current-page="page" :page-sizes="[10, 20, 50]"
                     @current-change="p => { page = p; load() }"
                     @size-change="s => { size = s; page = 1; load() }" />
    </el-card>

    <!-- 详情（档案+背包+宠物+战斗） -->
    <el-dialog title="角色详情" :visible.sync="detailDlg" width="720px" :close-on-click-modal="false">
      <template v-if="detail">
        <el-descriptions :column="3" size="medium" border>
          <el-descriptions-item label="家园号">{{ detail.player.user_id }}</el-descriptions-item>
          <el-descriptions-item label="家园昵称">{{ detail.home_nick || '—' }}</el-descriptions-item>
          <el-descriptions-item label="角色名">{{ detail.player.name }}</el-descriptions-item>
          <el-descriptions-item label="门派">{{ sectNames[detail.player.sect] }}</el-descriptions-item>
          <el-descriptions-item label="等级">{{ detail.player.level }}</el-descriptions-item>
          <el-descriptions-item label="经验">{{ detail.player.exp }}</el-descriptions-item>
          <el-descriptions-item label="气血">{{ detail.player.hp }}</el-descriptions-item>
          <el-descriptions-item label="法力">{{ detail.player.mp }}</el-descriptions-item>
          <el-descriptions-item label="恶名">{{ detail.player.emz }}</el-descriptions-item>
          <el-descriptions-item label="银两">{{ detail.player.money }}</el-descriptions-item>
          <el-descriptions-item label="存款">{{ detail.player.bank }}</el-descriptions-item>
          <el-descriptions-item label="金豆">{{ detail.player.beans }}</el-descriptions-item>
          <el-descriptions-item label="修炼池">{{ detail.player.xiulian_exp }} (开关{{ detail.player.xiulian_switch === 1 ? '开' : '关' }})</el-descriptions-item>
          <el-descriptions-item label="位置">{{ detail.player.map_x }},{{ detail.player.map_y }}</el-descriptions-item>
          <el-descriptions-item label="VIP">{{ detail.player.vip }}</el-descriptions-item>
        </el-descriptions>
        <div class="sub-title">宠物（{{ detail.pets.length }}）</div>
        <el-table :data="detail.pets" size="mini" border max-height="200">
          <el-table-column prop="id" label="ID" width="70" align="center" />
          <el-table-column prop="name" label="名字" min-width="100" />
          <el-table-column prop="level" label="等级" width="70" align="center" />
          <el-table-column prop="star" label="星级" width="70" align="center" />
          <el-table-column prop="quality" label="品质" width="70" align="center" />
          <el-table-column label="参战" width="70" align="center">
            <template slot-scope="{row}">{{ row.fighting === 1 ? '是' : '否' }}</template>
          </el-table-column>
        </el-table>
        <div class="sub-title">背包（{{ detail.bag.length }} 行）</div>
        <el-table :data="detail.bag" size="mini" border max-height="240">
          <el-table-column prop="id" label="行ID" width="80" align="center" />
          <el-table-column prop="kind" label="类型" width="80" align="center" />
          <el-table-column prop="ref_id" label="引用ID" width="90" align="center" />
          <el-table-column prop="count" label="数量" width="70" align="center" />
          <el-table-column prop="store" label="存放" width="80" align="center" />
        </el-table>
        <div class="sub-title">最近战斗（{{ detail.battles.length }}）</div>
        <el-table :data="detail.battles" size="mini" border max-height="200">
          <el-table-column prop="id" label="ID" width="70" align="center" />
          <el-table-column prop="enemy_name" label="对手" min-width="100" />
          <el-table-column label="结果" width="80" align="center">
            <template slot-scope="{row}">{{ resultNames[row.result] || row.result }}</template>
          </el-table-column>
          <el-table-column prop="exp" label="经验" width="80" align="center" />
          <el-table-column prop="money" label="银两" width="80" align="center" />
        </el-table>
      </template>
      <div slot="footer">
        <el-button @click="detailDlg = false">关 闭</el-button>
      </div>
    </el-dialog>

    <!-- 编辑 -->
    <el-dialog title="编辑角色" :visible.sync="editDlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <div class="form-grid">
          <el-form-item label="角色名">
            <el-input v-model="form.name" maxlength="12" style="width:160px" />
          </el-form-item>
          <el-form-item label="等级">
            <el-input-number v-model.number="form.level" :min="1" :max="9999" />
          </el-form-item>
          <el-form-item label="气血">
            <el-input-number v-model.number="form.hp" :min="0" />
          </el-form-item>
          <el-form-item label="法力">
            <el-input-number v-model.number="form.mp" :min="0" />
          </el-form-item>
          <el-form-item label="银两">
            <el-input-number v-model.number="form.money" :min="0" />
          </el-form-item>
          <el-form-item label="存款">
            <el-input-number v-model.number="form.bank" :min="0" />
          </el-form-item>
          <el-form-item label="金豆">
            <el-input-number v-model.number="form.beans" :min="0" />
          </el-form-item>
          <el-form-item label="恶名">
            <el-input-number v-model.number="form.emz" :min="0" />
          </el-form-item>
          <el-form-item label="VIP(分)">
            <el-input-number v-model.number="form.vip" :min="0" />
          </el-form-item>
          <el-form-item label="坐标">
            <el-input v-model="form.pos" placeholder="dtx,dty" style="width:140px" />
          </el-form-item>
        </div>
      </el-form>
      <div slot="footer">
        <el-button @click="editDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保 存</el-button>
      </div>
    </el-dialog>

    <!-- 发放 -->
    <el-dialog title="发放奖励" :visible.sync="grantDlg" width="520px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <el-form-item label="类型">
          <el-radio-group v-model="grant.kind">
            <el-radio label="item">物品</el-radio>
            <el-radio label="equip">装备</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="grant.kind === 'item' ? '物品ID' : '装备ID'">
          <el-input-number v-model.number="grant.ref_id" :min="1" />
        </el-form-item>
        <el-form-item label="数量">
          <el-input-number v-model.number="grant.count" :min="1" :max="9999" />
        </el-form-item>
        <el-form-item label="同时发银两">
          <el-input-number v-model.number="grant.money" :min="0" />
        </el-form-item>
        <el-form-item label="同时发金豆">
          <el-input-number v-model.number="grant.beans" :min="0" />
        </el-form-item>
      </el-form>
      <em>提示：ID 可在「幻想西游 → 数据管理」中查询</em>
      <div slot="footer">
        <el-button @click="grantDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doGrant">发 放</el-button>
      </div>
    </el-dialog>

    <!-- 封号/禁言 -->
    <el-dialog :title="punish.type === 'ban' ? '封号' : '禁言'" :visible.sync="punishDlg" width="460px" :close-on-click-modal="false">
      <p style="margin:0 0 12px">目标角色：<b>{{ punish.name }}</b>（ID:{{ punish.id }}）</p>
      <el-form label-width="90px">
        <el-form-item label="操作类型">
          <el-radio-group v-model="punish.type" @change="t => { punish.mins = (t === 'ban' ? (isBanned(punish.row) ? 0 : 1440) : (isMuted(punish.row) ? 0 : 1440)) }">
            <el-radio label="ban">封号（禁止登录）</el-radio>
            <el-radio label="mute">禁言（禁止聊天）</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="punish.type === 'ban' ? '封号时长' : '禁言时长'">
          <el-select v-model="punish.mins" style="width:200px">
            <el-option v-for="d in punishDurs" :key="d.v" :label="d.n" :value="d.v" />
          </el-select>
        </el-form-item>
        <el-form-item label="当前状态">
          <el-tag v-if="isBanned(punish.row || {})" size="mini" type="danger">封号中</el-tag>
          <el-tag v-if="isMuted(punish.row || {})" size="mini" type="warning">禁言中</el-tag>
          <el-tag v-if="!isBanned(punish.row || {}) && !isMuted(punish.row || {})" size="mini" type="success">正常</el-tag>
        </el-form-item>
      </el-form>
      <em>提示：选择「解除」可立即恢复该角色的{{ punish.type === 'ban' ? '登录' : '发言' }}权限</em>
      <div slot="footer">
        <el-button @click="punishDlg = false">取 消</el-button>
        <el-button :type="punish.mins === 0 ? 'success' : 'danger'" :loading="saving" @click="doPunish">
          {{ punish.mins === 0 ? '解 除' : '确 定' }}
        </el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminXyPlayers',
  data () {
    return {
      list: [], total: 0, page: 1, size: 10, loading: false, word: '',
      detailDlg: false, detail: null,
      editDlg: false, saving: false, editId: 0, form: {},
      grantDlg: false, grantId: 0, grant: { kind: 'item', ref_id: 1, count: 1, money: 0, beans: 0 },
      punishDlg: false, punish: { type: 'ban', id: 0, name: '', mins: 1440, row: null },
      punishDurs: [
        { n: '解除', v: 0 }, { n: '1小时', v: 60 }, { n: '6小时', v: 360 },
        { n: '1天', v: 1440 }, { n: '3天', v: 4320 }, { n: '7天', v: 10080 },
        { n: '30天', v: 43200 }, { n: '永久', v: -1 }
      ],
      sexNames: { 1: '男', 2: '女' },
      sectNames: { 1: '将军府', 2: '龙宫', 3: '月宫', 4: '方寸山', 5: '普陀山' },
      resultNames: { 2: '胜利', 3: '失败', 4: '逃跑' }
    }
  },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/xy-players', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    openDetail (row) {
      this.detail = null
      this.detailDlg = true
      api.get('/admin/xy-players/' + row.id + '/detail').then(r => {
        if (r.code === 0) this.detail = r.data
        else { this.detailDlg = false; this.$message.error(r.msg) }
      })
    },
    openEdit (row) {
      this.editId = row.id
      this.form = {
        name: row.name, level: row.level, hp: row.hp, mp: row.mp,
        money: row.money, bank: row.bank, beans: row.beans, emz: row.emz, vip: row.vip,
        pos: row.map_x + ',' + row.map_y
      }
      this.editDlg = true
    },
    save () {
      this.saving = true
      const body = {
        name: this.form.name, level: this.form.level, hp: this.form.hp, mp: this.form.mp,
        money: this.form.money, bank: this.form.bank, beans: this.form.beans,
        emz: this.form.emz, vip: this.form.vip
      }
      const parts = String(this.form.pos || '').split(',')
      if (parts.length === 2 && !isNaN(+parts[0]) && !isNaN(+parts[1])) {
        body.map_x = +parts[0]
        body.map_y = +parts[1]
      }
      api.put('/admin/xy-players/' + this.editId, body).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.editDlg = false
          this.$message.success(r.data.msg || '已保存')
          this.load()
        } else this.$message.error(r.msg)
      })
    },
    openGrant (row) {
      this.grantId = row.id
      this.grant = { kind: 'item', ref_id: 1, count: 1, money: 0, beans: 0 }
      this.grantDlg = true
    },
    doGrant () {
      this.saving = true
      api.post('/admin/xy-players/' + this.grantId + '/grant', this.grant).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.grantDlg = false
          this.$message.success(r.data.msg || '已发放')
          this.load()
        } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm('删除角色「' + row.name + '」将同时清除背包/宠物/技能/任务/流水等全部数据，不可恢复！', '危险操作', { type: 'error' }).then(() => {
        api.delete('/admin/xy-players/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    isBanned (row) { return row && row.ban_until > 0 && row.ban_until > Date.now() / 1000 },
    isMuted (row) { return row && row.mute_until > 0 && row.mute_until > Date.now() / 1000 },
    openPunish (row) {
      this.punish = { type: 'ban', id: row.id, name: row.name, mins: this.isBanned(row) ? 0 : 1440, row }
      this.punishDlg = true
    },
    doPunish () {
      this.saving = true
      const url = '/admin/xy-players/' + this.punish.id + (this.punish.type === 'ban' ? '/ban' : '/mute')
      api.post(url, { mins: this.punish.mins }).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.punishDlg = false
          this.$message.success(r.data.msg || '操作成功')
          this.load()
        } else this.$message.error(r.msg)
      })
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.form-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 0 14px; }
.sub-title { font-size: 13px; font-weight: 600; color: #1f2d3d; margin: 12px 0 8px; padding-left: 6px; border-left: 3px solid #409eff; }
</style>
