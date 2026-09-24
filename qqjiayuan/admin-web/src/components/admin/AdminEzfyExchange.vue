<template>
  <div class="farm-admin">
    <el-tabs v-model="tab">
      <el-tab-pane label="挂单维护" name="orders">
        <!-- 概览 -->
        <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <span>资源交易行 · 概览</span>
        <div>
          <el-button size="mini" type="success" icon="el-icon-plus" @click="openCreate">新增系统挂单</el-button>
          <el-button size="mini" type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
        </div>
      </div>
      <div class="stat-row">
        <div class="stat-card s-green">
          <div class="stat-ico"><i class="el-icon-s-shop" /></div>
          <div class="stat-info">
            <div class="stat-num">{{ systemOn }}</div>
            <div class="stat-lab">在售 · 系统挂单</div>
          </div>
        </div>
        <div class="stat-card s-blue">
          <div class="stat-ico"><i class="el-icon-user" /></div>
          <div class="stat-info">
            <div class="stat-num">{{ playerOn }}</div>
            <div class="stat-lab">在售 · 玩家挂单</div>
          </div>
        </div>
        <div class="stat-card s-purple">
          <div class="stat-ico"><i class="el-icon-sold-out" /></div>
          <div class="stat-info">
            <div class="stat-num">{{ total }}</div>
            <div class="stat-lab">筛选结果总数</div>
          </div>
        </div>
      </div>
      <!-- [说明·不显示在界面] 系统挂单的卖方固定为「系统」，可用<b>黄金</b>或<b>钻石</b>定价；玩家自己挂单只能按黄金计价（在游戏内操作）。
        系统挂单不消耗任何玩家资源，只往交易行里放货。 -->
    </el-card>

    <!-- 挂单列表 -->
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-select v-model="status" style="width:140px" @change="page = 1; load()">
          <el-option label="全部状态" :value="-1" />
          <el-option label="在售" :value="0" />
          <el-option label="已成交" :value="1" />
          <el-option label="已下架" :value="2" />
        </el-select>
        <el-input v-model="word" placeholder="挂单ID / 卖家ID / 卖家昵称" clearable style="width:240px"
                  @keyup.enter.native="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <el-button type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>
      <!-- ★ 列宽按实测内容宽度定：数量/总价/单价 列要装得下千分位大数（如 100,000,000,000,000） -->
      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="75" align="center" />
        <el-table-column label="卖家" min-width="150" show-overflow-tooltip>
          <template slot-scope="{row}">
            <span v-if="row.is_system === 1" class="td-red">系统</span>
            <span v-else class="td-main">{{ row.seller_name || '—' }}（{{ row.seller_id }}）</span>
          </template>
        </el-table-column>
        <el-table-column label="资源" width="95" align="center">
          <template slot-scope="{row}">{{ row.type_name }}</template>
        </el-table-column>
        <el-table-column label="数量" width="150" align="right">
          <template slot-scope="{row}"><span class="td-mono">{{ fmtN(row.es_count) }}</span></template>
        </el-table-column>
        <el-table-column label="库存" width="120" align="center">
          <template slot-scope="{row}">
            <span v-if="row.is_system === 1" class="td-green">∞ 无限</span>
            <span v-else class="td-mono">{{ row.status === 0 ? fmtN(row.es_count) : 0 }}</span>
          </template>
        </el-table-column>
        <el-table-column label="总价" width="150" align="right">
          <template slot-scope="{row}">
            <span :class="row.currency === 2 ? 'td-blue' : 'td-gold'">{{ fmtN(row.total_price) }}</span>
            <span class="td-sub">{{ row.currency_name }}</span>
          </template>
        </el-table-column>
        <el-table-column label="单价" width="130" align="right">
          <template slot-scope="{row}"><span class="td-muted">{{ fmtUnit(row) }}</span></template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="statusTag(row.status)">{{ row.status_name }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="买家ID" width="100" align="center">
          <template slot-scope="{row}">
            <span v-if="row.buyer_id">{{ row.buyer_id }}</span>
            <span v-else class="td-muted">—</span>
          </template>
        </el-table-column>
        <el-table-column label="挂单时间" width="170" align="center">
          <template slot-scope="{row}"><span class="td-sub">{{ fmtTime(row.created_at) }}</span></template>
        </el-table-column>
        <el-table-column label="操作" width="150" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button v-if="row.status === 0" size="mini" type="warning" plain @click="doOff(row)">下架</el-button>
            <el-button size="mini" type="danger" plain @click="doRemove(row)">删除</el-button>
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

    <!-- 新增系统挂单 -->
    <el-dialog title="新增系统挂单（卖方：系统 · 买不完）" :visible.sync="dlg" width="600px" :close-on-click-modal="false">
      <el-form label-width="150px" size="small">
        <el-form-item label="资源包模板">
          <el-select v-model="packId" style="width:100%" placeholder="选一个模板自动填数量/价格，也可选自定义" @change="onPack">
            <el-option :value="0" label="自定义（手动填写）" />
            <el-option v-for="p in templates" :key="p.id" :label="p.name" :value="p.id" />
          </el-select>
          <div class="td-sub" style="margin-top:4px">
            模板只做快速填充，上架前数量/价格仍可改；系统挂单<b>无限库存</b>，玩家可反复购买
          </div>
        </el-form-item>
        <el-form-item label="资源类型" required>
          <el-select v-model="form.es_type" style="width:100%">
            <el-option v-for="(n, t) in resNames" :key="'rt' + t" :label="n" :value="Number(t)" />
          </el-select>
        </el-form-item>
        <el-form-item label="数量（资源数量）" required>
          <el-input-number v-model.number="form.es_count" :min="1" :step="1000" controls-position="right" style="width:100%" />
          <div class="td-sub" style="margin-top:4px">选中模板后仍可二次修改</div>
        </el-form-item>
        <el-form-item label="总价" required>
          <el-input-number v-model.number="form.total_price" :min="1" :step="100" controls-position="right" style="width:100%" />
        </el-form-item>
        <el-form-item label="挂单数量">
          <el-input-number v-model.number="form.repeat" :min="1" :max="500" controls-position="right" style="width:100%" />
          <div class="td-sub" style="margin-top:4px">一次性挂多少单（同资源同数同价），不用一单一单配置</div>
        </el-form-item>
        <el-form-item label="计价货币" required>
          <el-radio-group v-model="form.currency">
            <el-radio :label="1">黄金</el-radio>
            <el-radio :label="2">钻石</el-radio>
          </el-radio-group>
          <div class="td-sub" style="margin-top:4px">
            钻石定价的挂单，玩家需要用档案上的钻石余额购买（钻石由管理员发放）
          </div>
        </el-form-item>
        <el-form-item label="单价预览">
          <span class="td-mono">
            1 {{ form.es_type ? resNames[form.es_type] : '资源' }} ≈
            {{ unitPrice }} {{ form.currency === 2 ? '钻石' : '黄金' }}
          </span>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="dlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doCreate">上 架</el-button>
      </div>
    </el-dialog>
      </el-tab-pane>

      <!-- ================= 维护模版 ================= -->
      <el-tab-pane label="维护模版" name="tpls">
        <el-card shadow="never" class="box">
          <div class="toolbar">
            <span class="td-sub">资源包模板：新增系统挂单时用来快速填充数量/价格（选中后仍可改）</span>
            <div class="grow" />
            <el-button size="mini" type="success" icon="el-icon-plus" @click="openTplCreate">新增模版</el-button>
            <el-button size="mini" type="primary" plain icon="el-icon-refresh" @click="loadTpls">刷新</el-button>
          </div>
          <el-table :data="templates" v-loading="loadingTpl" stripe border>
            <el-table-column prop="id" label="ID" width="70" align="center" />
            <el-table-column prop="name" label="模版名" min-width="160" show-overflow-tooltip />
            <el-table-column prop="type_name" label="资源" width="90" align="center" />
            <el-table-column label="数量" width="150" align="right">
              <template slot-scope="{row}"><span class="td-mono">{{ fmtN(row.es_count) }}</span></template>
            </el-table-column>
            <el-table-column label="总价" width="150" align="right">
              <template slot-scope="{row}">
                <span :class="row.currency === 2 ? 'td-blue' : 'td-gold'">{{ fmtN(row.total_price) }}</span>
                <span class="td-sub">{{ row.currency_name }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="sort_no" label="排序" width="80" align="center" />
            <el-table-column label="操作" width="150" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openTplEdit(row)">编辑</el-button>
                <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="delTpl(row)" />
              </template>
            </el-table-column>
          </el-table>
        </el-card>

        <!-- 新增 / 编辑模版 -->
        <el-dialog :title="tpl.id ? ('编辑模版 #' + tpl.id) : '新增模版'" :visible.sync="tplDlg"
                   width="520px" :close-on-click-modal="false">
          <el-form label-width="110px" size="small">
            <el-form-item label="模版名" required>
              <el-input v-model="tpl.name" maxlength="50" style="width:100%" />
            </el-form-item>
            <el-form-item label="资源类型" required>
              <el-select v-model="tpl.es_type" style="width:100%">
                <el-option v-for="(n, t) in resNames" :key="'tt' + t" :label="n" :value="Number(t)" />
              </el-select>
            </el-form-item>
            <el-form-item label="数量" required>
              <el-input-number v-model.number="tpl.es_count" :min="1" :step="1000" controls-position="right" style="width:100%" />
            </el-form-item>
            <el-form-item label="总价" required>
              <el-input-number v-model.number="tpl.total_price" :min="1" :step="100" controls-position="right" style="width:100%" />
            </el-form-item>
            <el-form-item label="计价货币">
              <el-radio-group v-model="tpl.currency">
                <el-radio :label="1">黄金</el-radio>
                <el-radio :label="2">钻石</el-radio>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="排序">
              <el-input-number v-model.number="tpl.sort_no" :min="0" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-form>
          <div slot="footer">
            <el-button @click="tplDlg = false">取 消</el-button>
            <el-button type="primary" :loading="savingTpl" @click="doTplSave">保 存</el-button>
          </div>
        </el-dialog>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script>
import api from '../../api'

const RES_FALLBACK = { 1: '粮食', 2: '钢铁', 3: '石油', 4: '稀矿' }
// 交易所 es_type → 资源配置表的内部标识（改名后这里要跟着显示新名字）
const ES_KEY = { 1: 'food', 2: 'steel', 3: 'oil', 4: 'rare' }

export default {
  name: 'AdminEzfyExchange',
  data () {
    return {
      list: [], total: 0, page: 1, size: 20, loading: false,
      status: -1, word: '',
      systemOn: 0, playerOn: 0,
      dlg: false, saving: false,
      tab: 'orders',
      packId: 0,
      templates: [], loadingTpl: false, tplDlg: false, savingTpl: false,
      tpl: {},
      form: { es_type: 1, es_count: 10000, total_price: 1000, currency: 1, repeat: 1 },
      resCfgList: []
    }
  },
  computed: {
    resNames () {
      const m = Object.assign({}, RES_FALLBACK)
      const byKey = {}
      this.resCfgList.forEach(x => { if (x.key) byKey[x.key] = x.name })
      Object.keys(ES_KEY).forEach(t => {
        const n = byKey[ES_KEY[t]]
        if (n) m[t] = n
      })
      return m
    },
    unitPrice () {
      const c = Number(this.form.es_count) || 0
      const p = Number(this.form.total_price) || 0
      if (c <= 0) return '—'
      const v = p / c
      return v >= 1 ? v.toFixed(2) : v.toFixed(4)
    }
  },
  mounted () { this.load(); this.loadResCfgs(); this.loadTpls() },
  methods: {
    fmtN (v) {
      if (v === null || v === undefined) return '—'
      return Number(v).toLocaleString()
    },
    // 单价：总价/数量，整数除法会变成 0，所以这里保留小数
    fmtUnit (row) {
      const c = Number(row.es_count) || 0
      const p = Number(row.total_price) || 0
      if (c <= 0) return '—'
      const v = p / c
      if (v >= 1) return v.toFixed(2)
      if (v >= 0.0001) return v.toFixed(4)
      return v.toExponential(2)
    },
    fmtTime (t) {
      if (!t) return '—'
      const d = new Date(t)
      if (isNaN(d.getTime())) return String(t).replace('T', ' ').slice(0, 19)
      const p = n => (n < 10 ? '0' + n : '' + n)
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' +
        p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds())
    },
    statusTag (s) {
      if (s === 0) return 'success'
      if (s === 1) return 'info'
      return 'warning'
    },
    loadResCfgs () {
      api.get('/admin/ezfy-res-cfg').then(r => {
        if (r.code === 0) this.resCfgList = r.data.list || []
      })
    },
    load () {
      this.loading = true
      api.get('/admin/ezfy-exchange', {
        params: { page: this.page, size: this.size, status: this.status, word: this.word }
      }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list || []
          this.total = r.data.total
          this.page = r.data.page
          this.systemOn = r.data.system_on || 0
          this.playerOn = r.data.player_on || 0
        } else this.$message.error(r.msg)
      })
    },
    openCreate () {
      this.packId = 0
      this.form = { es_type: 1, es_count: 10000, total_price: 1000, currency: 1, repeat: 1 }
      this.dlg = true
    },
    // 选中模板（按数值 id 查找，避免对象比较导致的「勾选触发全选」bug）
    onPack (id) {
      if (!id) return
      const t = this.templates.find(x => Number(x.id) === Number(id))
      if (!t) return
      this.form.es_type = t.es_type
      this.form.es_count = t.es_count
      this.form.total_price = t.total_price
      this.form.currency = t.currency || 1
    },
    loadTpls () {
      this.loadingTpl = true
      api.get('/admin/ezfy-exchange-tpls').then(r => {
        this.loadingTpl = false
        if (r.code === 0) this.templates = r.data.list || []
        else this.$message.error(r.msg)
      })
    },
    openTplCreate () {
      this.tpl = { name: '', es_type: 1, es_count: 10000, total_price: 1000, currency: 1, sort_no: 0 }
      this.tplDlg = true
    },
    openTplEdit (row) {
      this.tpl = {
        id: row.id, name: row.name, es_type: row.es_type,
        es_count: row.es_count, total_price: row.total_price,
        currency: row.currency, sort_no: row.sort_no
      }
      this.tplDlg = true
    },
    doTplSave () {
      if (!this.tpl.name || !String(this.tpl.name).trim()) { this.$message.warning('请填写模版名'); return }
      if (!(this.tpl.es_count > 0)) { this.$message.warning('数量必须大于 0'); return }
      if (!(this.tpl.total_price > 0)) { this.$message.warning('总价必须大于 0'); return }
      this.savingTpl = true
      const body = {
        name: String(this.tpl.name).trim(), es_type: this.tpl.es_type,
        es_count: this.tpl.es_count, total_price: this.tpl.total_price,
        currency: this.tpl.currency, sort_no: this.tpl.sort_no
      }
      const req = this.tpl.id
        ? api.put('/admin/ezfy-exchange-tpls/' + this.tpl.id, body)
        : api.post('/admin/ezfy-exchange-tpls', body)
      req.then(r => {
        this.savingTpl = false
        if (r.code === 0) {
          this.tplDlg = false
          this.$message.success(r.data.msg || '模板已保存')
          this.loadTpls()
        } else this.$message.error(r.msg)
      })
    },
    delTpl (row) {
      this.$confirm('确定删除模版「' + row.name + '」？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-exchange-tpls/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '模板已删除'); this.loadTpls() }
          else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    doCreate () {
      if (!(this.form.es_count > 0)) { this.$message.warning('数量必须大于 0'); return }
      if (!(this.form.total_price > 0)) { this.$message.warning('总价必须大于 0'); return }
      this.saving = true
      api.post('/admin/ezfy-exchange', this.form).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.dlg = false
          this.$message.success(r.data.msg || '已上架')
          this.page = 1
          this.load()
        } else this.$message.error(r.msg)
      })
    },
    doOff (row) {
      this.$confirm('确定下架该挂单？玩家挂单会把资源退回卖家城市。', '提示', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-exchange/' + row.id + '/off', {}).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已下架'); this.load() }
          else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    doRemove (row) {
      this.$confirm('确定删除该挂单记录？在售的玩家挂单会先把资源退回卖家。', '警告', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-exchange/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.load() }
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
</style>
