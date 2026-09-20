<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="玩家昵称 / 用户ID / 家园号搜索" clearable style="width:240px"
                  @keyup.enter.native="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <el-button type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="user_id" label="用户ID" width="70" align="center" />
        <el-table-column prop="home_num" label="家园号" width="75" align="center" />
        <el-table-column label="家园昵称" width="100" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.home_nick || '—' }}</template>
        </el-table-column>
        <el-table-column label="玩家昵称" min-width="100" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-main">{{ row.nickname }}</span></template>
        </el-table-column>
        <el-table-column label="阵营" width="80" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.camp === 2 ? 'danger' : 'primary'">{{ row.camp_name }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="军功声望" width="95" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ row.prestige }}</span></template>
        </el-table-column>
        <el-table-column label="军衔" width="95" align="center">
          <template slot-scope="{row}">{{ row.rank_name }}</template>
        </el-table-column>
        <el-table-column prop="city_count" label="城池" width="65" align="center" />
        <el-table-column label="更新时间" width="150" align="center">
          <template slot-scope="{row}">{{ fmtTime(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="230" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="info" plain icon="el-icon-view" title="详情" @click="openDetail(row)" />
            <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openEdit(row)" />
            <el-button size="mini" type="success" plain icon="el-icon-present" title="发放" @click="openGrant(row)" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删号" @click="del(row)" />
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

    <!-- 详情（档案+城池+背包+军团+最近出征） -->
    <el-dialog title="玩家详情" :visible.sync="detailDlg" width="760px" :close-on-click-modal="false">
      <template v-if="detail">
        <el-descriptions :column="3" size="medium" border>
          <el-descriptions-item label="家园号">{{ detail.home_num || detail.player.user_id }}</el-descriptions-item>
          <el-descriptions-item label="家园昵称">{{ detail.home_nick || '—' }}</el-descriptions-item>
          <el-descriptions-item label="玩家昵称">{{ detail.player.nickname }}</el-descriptions-item>
          <el-descriptions-item label="阵营">{{ detail.camp_name }}</el-descriptions-item>
          <el-descriptions-item label="军功声望">{{ detail.player.prestige }}</el-descriptions-item>
          <el-descriptions-item label="军衔">{{ detail.rank_name }}</el-descriptions-item>
          <el-descriptions-item label="钻石余额">{{ detail.player.diamond || 0 }}</el-descriptions-item>
        </el-descriptions>
        <div class="sub-title">城池（{{ detail.cities.length }}）</div>
        <el-table :data="detail.cities" size="mini" border max-height="220">
          <el-table-column prop="id" label="城池ID" width="80" align="center" />
          <el-table-column prop="name" label="城名" min-width="100" />
          <el-table-column label="坐标" width="90" align="center">
            <template slot-scope="{row}">{{ row.x }},{{ row.y }}</template>
          </el-table-column>
          <el-table-column prop="city_level" label="市政厅" width="80" align="center" />
          <el-table-column prop="gold" label="黄金" width="90" align="center" />
          <el-table-column prop="food" label="粮食" width="90" align="center" />
          <el-table-column prop="steel" label="钢铁" width="90" align="center" />
          <el-table-column prop="oil" label="石油" width="90" align="center" />
          <el-table-column prop="rare" label="稀矿" width="90" align="center" />
        </el-table>
        <div class="sub-title">背包（{{ detail.bag.length }} 行）</div>
        <el-table :data="detail.bag" size="mini" border max-height="200">
          <el-table-column prop="id" label="行ID" width="80" align="center" />
          <el-table-column prop="cfg_id" label="道具ID" width="90" align="center" />
          <el-table-column prop="item_name" label="道具名" min-width="120" />
          <el-table-column prop="count" label="数量" width="90" align="center" />
        </el-table>
        <div class="sub-title">军团（{{ detail.corps.length }}）</div>
        <el-table :data="detail.corps" size="mini" border max-height="160">
          <el-table-column prop="id" label="记录ID" width="90" align="center" />
          <el-table-column prop="corps_name" label="军团名" min-width="120" />
          <el-table-column prop="title" label="职位" width="110" align="center" />
          <el-table-column label="身份" width="90" align="center">
            <template slot-scope="{row}">{{ row.is_leader === 1 ? '军团长' : '成员' }}</template>
          </el-table-column>
        </el-table>
        <div class="sub-title">最近出征（{{ detail.orders.length }}）</div>
        <el-table :data="detail.orders" size="mini" border max-height="200">
          <el-table-column prop="id" label="订单ID" width="90" align="center" />
          <el-table-column label="类型" width="80" align="center">
            <template slot-scope="{row}">{{ typeNames[row.order_type] || row.order_type }}</template>
          </el-table-column>
          <el-table-column label="目标" width="100" align="center">
            <template slot-scope="{row}">{{ row.target_x }},{{ row.target_y }}</template>
          </el-table-column>
          <el-table-column label="状态" width="90" align="center">
            <template slot-scope="{row}">{{ statusNames[row.status] || row.status }}</template>
          </el-table-column>
          <el-table-column label="时间" width="150" align="center">
            <template slot-scope="{row}">{{ fmtTime(row.created_at) }}</template>
          </el-table-column>
        </el-table>
      </template>
      <div slot="footer">
        <el-button @click="detailDlg = false">关 闭</el-button>
      </div>
    </el-dialog>

    <!-- 编辑 -->
    <el-dialog title="编辑玩家" :visible.sync="editDlg" width="480px" :close-on-click-modal="false">
      <el-form label-width="100px">
        <el-form-item label="玩家昵称">
          <el-input v-model="form.nickname" maxlength="20" style="width:200px" />
        </el-form-item>
        <el-form-item label="军功声望">
          <el-input-number v-model.number="form.prestige" :min="0" />
        </el-form-item>
        <el-form-item label="阵营">
          <el-radio-group v-model="form.camp">
            <el-radio :label="1">同盟国</el-radio>
            <el-radio :label="2">轴心国</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="editDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保 存</el-button>
      </div>
    </el-dialog>

    <!-- 发放 -->
    <el-dialog title="发放资源/道具" :visible.sync="grantDlg" width="540px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <el-form-item label="黄金">
          <el-input-number v-model.number="grant.gold" :min="0" :step="1000" />
        </el-form-item>
        <el-form-item label="粮食">
          <el-input-number v-model.number="grant.food" :min="0" :step="1000" />
        </el-form-item>
        <el-form-item label="钢铁">
          <el-input-number v-model.number="grant.steel" :min="0" :step="1000" />
        </el-form-item>
        <el-form-item label="石油">
          <el-input-number v-model.number="grant.oil" :min="0" :step="1000" />
        </el-form-item>
        <el-form-item label="稀矿">
          <el-input-number v-model.number="grant.rare" :min="0" :step="1000" />
        </el-form-item>
        <!-- ★ 第九轮：钻石只能由管理端充值（可负数扣减；玩家端只读余额） -->
        <el-form-item label="钻石">
          <el-input-number v-model.number="grant.diamond" :min="-9999999" :max="9999999" :step="100" />
          <el-button size="mini" type="warning" plain :loading="diamondSaving" @click="doDiamond">充 值</el-button>
          <span class="td-mono" style="margin-left:8px">当前：{{ grantDiamond }}</span>
        </el-form-item>
        <el-form-item label="道具">
          <div class="grant-items">
            <div v-for="(it, i) in grant.items" :key="i" class="grant-item-row">
              <el-input-number v-model.number="it.cfg_id" :min="1" controls-position="right" style="width:120px" />
              <span class="grant-x">×</span>
              <el-input-number v-model.number="it.count" :min="1" :max="9999" controls-position="right" style="width:110px" />
              <el-button type="text" class="danger-btn" @click="grant.items.splice(i, 1)">删除</el-button>
            </div>
            <el-button size="mini" type="primary" plain icon="el-icon-plus" @click="grant.items.push({ cfg_id: 1, count: 1 })">添加道具</el-button>
          </div>
        </el-form-item>
      </el-form>
      <em>提示：资源发放<b>不受主城仓储上限限制</b>（可以超上限堆着）；道具ID 可在「二战风云 → 数据管理」中查询</em>
      <div slot="footer">
        <el-button @click="grantDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doGrant">发 放</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminEzfyPlayers',
  data () {
    return {
      list: [], total: 0, page: 1, size: 5, loading: false, word: '',
      detailDlg: false, detail: null,
      editDlg: false, saving: false, editId: 0, form: {},
      grantDlg: false, grantId: 0, grant: { gold: 0, food: 0, steel: 0, oil: 0, rare: 0, diamond: 0, items: [] },
      diamondSaving: false, grantDiamond: 0,
      typeNames: { 1: '侦查', 2: '掠夺', 3: '征服', 4: '采集', 5: '运输', 6: '增援', 7: '派遣' },
      statusNames: { 0: '行进中', 1: '驻守中', 2: '返回中', 3: '已完成', 4: '已阵亡' }
    }
  },
  mounted () { this.load() },
  methods: {
    fmtTime (t) { return t ? new Date(t).toLocaleString() : '' },
    load () {
      this.loading = true
      api.get('/admin/ezfy-players', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
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
      api.get('/admin/ezfy-players/' + row.user_id + '/detail').then(r => {
        if (r.code === 0) this.detail = r.data
        else { this.detailDlg = false; this.$message.error(r.msg) }
      })
    },
    openEdit (row) {
      this.editId = row.user_id
      this.form = { nickname: row.nickname, prestige: row.prestige, camp: row.camp }
      this.editDlg = true
    },
    save () {
      this.saving = true
      api.put('/admin/ezfy-players/' + this.editId, this.form).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.editDlg = false
          this.$message.success(r.data.msg || '已保存')
          this.load()
        } else this.$message.error(r.msg)
      })
    },
    openGrant (row) {
      this.grantId = row.user_id
      this.grant = { gold: 0, food: 0, steel: 0, oil: 0, rare: 0, diamond: 0, items: [] }
      this.grantDiamond = 0
      this.grantDlg = true
      // 拉一下当前钻石余额（玩家端只读，这里给管理员做参考）
      api.get('/admin/ezfy-players/' + row.user_id + '/detail').then(r => {
        if (r.code === 0 && r.data && r.data.player) this.grantDiamond = r.data.player.diamond || 0
      })
    },
    // ★ 第九轮：钻石充值（独立接口，可正可负；走「钻石充值」公告通知玩家）
    doDiamond () {
      const n = parseInt(this.grant.diamond) || 0
      if (!n) { this.$message.warning('请填写充值数量（可为负数扣减）'); return }
      this.diamondSaving = true
      api.post('/admin/ezfy-players/' + this.grantId + '/diamond', { amount: n, mode: 'add' }).then(r => {
        this.diamondSaving = false
        if (r.code === 0) {
          this.$message.success(r.data.msg || '充值成功')
          this.grantDiamond = r.data.diamond
          this.grant.diamond = 0
        } else this.$message.error(r.msg || '充值失败')
      }).catch(() => { this.diamondSaving = false })
    },
    doGrant () {
      const hasRes = this.grant.gold > 0 || this.grant.food > 0 || this.grant.steel > 0 ||
        this.grant.oil > 0 || this.grant.rare > 0
      if (!hasRes && this.grant.items.length === 0) {
        this.$message.warning('请先填写发放内容')
        return
      }
      this.saving = true
      api.post('/admin/ezfy-players/' + this.grantId + '/grant', this.grant).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.grantDlg = false
          this.$message.success(r.data.msg || '已发放')
          this.load()
        } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm('删除玩家「' + row.nickname + '」将同时清除城池/部队/科技/出征/背包等全部游戏数据，不可恢复！', '危险操作', { type: 'error' }).then(() => {
        api.delete('/admin/ezfy-players/' + row.user_id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.sub-title { font-size: 13px; font-weight: 600; color: #1f2d3d; margin: 12px 0 8px; padding-left: 6px; border-left: 3px solid #409eff; }
.grant-item-row { display: flex; align-items: center; margin-bottom: 6px; }
.grant-x { margin: 0 6px; }
</style>
