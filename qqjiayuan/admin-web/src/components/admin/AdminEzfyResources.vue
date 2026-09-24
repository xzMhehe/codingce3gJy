<template>
  <div class="farm-admin">
    <!-- 全服统计 -->
    <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <span>全服资源统计</span>
        <el-button size="mini" type="primary" plain icon="el-icon-refresh" @click="loadSummary">刷新</el-button>
      </div>
      <el-row :gutter="14" v-if="sum" v-loading="loadingSum">
        <el-col :span="6" v-for="c in sumCards" :key="c.label">
          <div class="stat-card">
            <div class="stat-num">{{ fmtBig(c.val) }}</div>
            <div class="stat-label">{{ c.label }}</div>
          </div>
        </el-col>
      </el-row>
      <div v-if="sum && sum.tops && sum.tops.length" class="sub-title">资源总量 TOP10 城池</div>
      <el-table v-if="sum && sum.tops && sum.tops.length" :data="sum.tops" size="mini" border max-height="260">
        <el-table-column type="index" label="#" width="50" align="center" />
        <el-table-column prop="name" label="城池" min-width="150" show-overflow-tooltip />
        <el-table-column prop="user_id" label="玩家ID" width="100" align="center" />
        <el-table-column :label="'资源总量（' + resShortList + '）'" width="200" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ fmtBig(row.total) }}</span></template>
        </el-table-column>
        <el-table-column :label="resName('gold')" width="150" align="center">
          <template slot-scope="{row}"><span class="td-gold">{{ fmtBig(row.gold) }}</span></template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 城池资源列表 -->
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="城名 / 玩家昵称 / 用户ID" clearable style="width:240px"
                  @keyup.enter.native="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <el-button type="success" icon="el-icon-present" @click="openGrant">批量发放</el-button>
        <el-button type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>
      <!-- ★ 列宽按实测内容宽度定：资源列原 108px 装不下「99999.95万亿 / 55万」（需 131px）；
           城名/归属玩家 两个弹性列分摊多余宽度 -->
      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="id" label="城池ID" width="70" align="center" />
        <el-table-column label="城名" min-width="140" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column label="归属玩家（家园号）" min-width="150" show-overflow-tooltip>
          <template slot-scope="{row}">{{ row.player_name || '—' }}（{{ row.home_num || '—' }}）</template>
        </el-table-column>
        <el-table-column :label="resName('gold') + ' / 上限'" width="145" align="center">
          <template slot-scope="{row}">
            <span class="td-gold">{{ fmtBig(row.gold) }}</span>
            <span class="td-muted"> / {{ fmtBig(row.gold_cap) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="resName('food') + ' / 上限'" width="145" align="center">
          <template slot-scope="{row}">{{ fmtBig(row.food) }}<span class="td-muted"> / {{ fmtBig(row.food_cap) }}</span></template>
        </el-table-column>
        <el-table-column :label="resName('steel') + ' / 上限'" width="145" align="center">
          <template slot-scope="{row}">{{ fmtBig(row.steel) }}<span class="td-muted"> / {{ fmtBig(row.steel_cap) }}</span></template>
        </el-table-column>
        <el-table-column :label="resName('oil') + ' / 上限'" width="145" align="center">
          <template slot-scope="{row}">{{ fmtBig(row.oil) }}<span class="td-muted"> / {{ fmtBig(row.oil_cap) }}</span></template>
        </el-table-column>
        <el-table-column :label="resName('rare') + ' / 上限'" width="145" align="center">
          <template slot-scope="{row}">{{ fmtBig(row.rare) }}<span class="td-muted"> / {{ fmtBig(row.rare_cap) }}</span></template>
        </el-table-column>
        <el-table-column label="资源合计" width="130" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ fmtBig(row.total_res) }}</span></template>
        </el-table-column>
        <el-table-column label="操作" width="160" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain @click="openSet(row)">改资源</el-button>
            <el-button size="mini" type="success" plain @click="quickGrant(row)">发资源</el-button>
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

    <!-- 资源名称维护（改名后游戏端 / 管理端展示全部跟随） -->
    <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <span>资源名称维护（改名后游戏端与管理端展示全部跟随）</span>
        <el-button size="mini" type="warning" plain icon="el-icon-refresh-left" @click="resetResCfg">恢复默认</el-button>
      </div>
      <el-table :data="resCfgs" v-loading="loadingRes" size="mini" stripe border>
        <el-table-column prop="id" label="ID" width="60" align="center" />
        <el-table-column prop="key" label="内部标识（勿改）" width="140" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ row.key }}</span></template>
        </el-table-column>
        <el-table-column prop="name" label="显示名" width="180" align="center">
          <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column prop="short" label="单字简称" width="130" align="center">
          <template slot-scope="{row}"><span class="td-blue">{{ row.short }}</span></template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="90" align="center" />
        <el-table-column label="说明" min-width="200" show-overflow-tooltip>
          <template slot-scope="{row}">改名后游戏内资源栏、各类花费、交易所、管理端列头全部跟随</template>
        </el-table-column>
        <el-table-column label="操作" width="110" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain icon="el-icon-edit" title="改名" @click="openResEdit(row)" />
          </template>
        </el-table-column>
      </el-table>
      <!-- [说明·不显示在界面] 提示：这是「预留」能力 —— 后期把「{{ resName('rare') }}」改成别的叫法，只改这里即可，不用改代码 -->
    </el-card>

    <!-- 直接设置资源 -->
    <el-dialog title="直接设置资源（GM 专用，不受仓储上限限制）" :visible.sync="setDlg" width="520px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="城池">
          <span class="td-main">{{ setRow.name }}（ID {{ setRow.id }} · {{ setRow.player_name }}）</span>
        </el-form-item>
        <el-form-item label="资源类型">
          <el-select v-model="setForm.field" style="width:100%">
            <el-option :label="resName('gold')" value="gold" />
            <el-option :label="resName('food')" value="food" />
            <el-option :label="resName('steel')" value="steel" />
            <el-option :label="resName('oil')" value="oil" />
            <el-option :label="resName('rare')" value="rare" />
          </el-select>
        </el-form-item>
        <el-form-item label="数值">
          <el-input-number v-model.number="setForm.value" :min="0" :step="1000" controls-position="right" style="width:100%" />
          <span class="td-sub">当前 {{ fmtBig(setRow[setForm.field]) }}</span>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="setDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doSet">设 置</el-button>
      </div>
    </el-dialog>

    <!-- 批量发放 -->
    <el-dialog title="批量发放资源" :visible.sync="grantDlg" width="600px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="玩家ID列表" required>
          <el-input v-model="grantForm.ids" type="textarea" :rows="3"
                    placeholder="多个用户ID用逗号或空格分隔，例如：10007,10008" />
        </el-form-item>
        <el-form-item :label="resName('gold')"><el-input-number v-model.number="grantForm.gold" :min="-999999999" :step="1000" controls-position="right" style="width:100%" /></el-form-item>
        <el-form-item :label="resName('food')"><el-input-number v-model.number="grantForm.food" :min="-999999999" :step="1000" controls-position="right" style="width:100%" /></el-form-item>
        <el-form-item :label="resName('steel')"><el-input-number v-model.number="grantForm.steel" :min="-999999999" :step="1000" controls-position="right" style="width:100%" /></el-form-item>
        <el-form-item :label="resName('oil')"><el-input-number v-model.number="grantForm.oil" :min="-999999999" :step="1000" controls-position="right" style="width:100%" /></el-form-item>
        <el-form-item :label="resName('rare')"><el-input-number v-model.number="grantForm.rare" :min="-999999999" :step="1000" controls-position="right" style="width:100%" /></el-form-item>
      </el-form>
      <!-- [说明·不显示在界面] 提示：发放<b>不受仓储上限限制</b>（可以超上限堆着，填负数即扣减），并会给玩家发送一条站内通知 -->
      <div slot="footer">
        <el-button @click="grantDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doGrant">发 放</el-button>
      </div>
    </el-dialog>

    <!-- 资源改名 -->
    <el-dialog :title="'修改资源名称 · 当前「' + resForm.name + '」'" :visible.sync="resDlg" width="520px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="内部标识">
          <span class="td-mono">{{ resForm.key }}</span>
          <span class="td-sub">（程序内部用，不要改）</span>
        </el-form-item>
        <el-form-item label="显示名" required>
          <el-input v-model="resForm.name" maxlength="30" placeholder="例如：稀矿 → 秘银" />
        </el-form-item>
        <el-form-item label="单字简称">
          <el-input v-model="resForm.short" maxlength="10" placeholder="用于「粮/钢/油/稀」这类紧凑位置" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model.number="resForm.sort" :min="0" controls-position="right" style="width:100%" />
        </el-form-item>
      </el-form>
      <!-- [说明·不显示在界面] 保存后游戏端资源栏、花费展示、交易所与管理端列头全部跟随新名字 -->
      <div slot="footer">
        <el-button @click="resDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doResSave">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

const RES_KEYS = ['gold', 'food', 'steel', 'oil', 'rare']
const RES_FALLBACK = { gold: '黄金', food: '粮食', steel: '钢铁', oil: '石油', rare: '稀矿' }
const RES_SHORT_FALLBACK = { gold: '金', food: '粮', steel: '钢', oil: '油', rare: '稀' }

export default {
  name: 'AdminEzfyResources',
  data () {
    return {
      sum: null, loadingSum: false,
      list: [], total: 0, page: 1, size: 5, loading: false, word: '',
      setDlg: false, setRow: {}, setForm: { field: 'gold', value: 0 },
      grantDlg: false, grantForm: { ids: '', gold: 0, food: 0, steel: 0, oil: 0, rare: 0 },
      resCfgs: [], loadingRes: false,
      resDlg: false, resForm: { id: 0, key: '', name: '', short: '', sort: 0 },
      saving: false
    }
  },
  computed: {
    resNames () {
      const m = Object.assign({}, RES_FALLBACK)
      this.resCfgs.forEach(x => { m[x.key] = x.name })
      return m
    },
    resShort () {
      const m = Object.assign({}, RES_SHORT_FALLBACK)
      this.resCfgs.forEach(x => { m[x.key] = x.short || m[x.key] })
      return m
    },
    resShortList () {
      return RES_KEYS.map(k => this.resShort[k]).join('+')
    },
    sumCards () {
      const s = this.sum
      if (!s) return []
      return [
        { label: '城池总数', val: s.cities },
        { label: '流通' + this.resName('gold'), val: s.gold },
        { label: this.resName('food') + '总量', val: s.food },
        { label: this.resName('steel') + '总量', val: s.steel }
      ]
    }
  },
  mounted () { this.load(); this.loadSummary(); this.loadResCfgs() },
  methods: {
    resName (k) { return this.resNames[k] || RES_FALLBACK[k] || k },
    fmtN (v) {
      if (v === null || v === undefined) return '—'
      return Number(v).toLocaleString()
    },
    // 大数加单位（万/亿/万亿），避免一串 10 位数字看不清
    fmtBig (v) {
      if (v === null || v === undefined || v === '') return '—'
      const n = Number(v)
      if (!isFinite(n)) return '—'
      const abs = Math.abs(n)
      const sign = n < 0 ? '-' : ''
      const cut = s => s.replace(/\.?0+$/, '')
      if (abs < 10000) return sign + abs.toLocaleString()
      if (abs < 1e8) return sign + cut((abs / 1e4).toFixed(2)) + '万'
      if (abs < 1e12) return sign + cut((abs / 1e8).toFixed(2)) + '亿'
      return sign + cut((abs / 1e12).toFixed(2)) + '万亿'
    },
    loadResCfgs () {
      this.loadingRes = true
      api.get('/admin/ezfy-res-cfg').then(r => {
        this.loadingRes = false
        if (r.code === 0) this.resCfgs = r.data.list || []
        else this.$message.error(r.msg)
      })
    },
    openResEdit (row) {
      this.resForm = { id: row.id, key: row.key, name: row.name, short: row.short, sort: row.sort }
      this.resDlg = true
    },
    doResSave () {
      if (!String(this.resForm.name || '').trim()) { this.$message.warning('请填写显示名'); return }
      this.saving = true
      api.put('/admin/ezfy-res-cfg/' + this.resForm.id, {
        name: this.resForm.name, short: this.resForm.short, sort: this.resForm.sort
      }).then(r => {
        this.saving = false
        if (r.code === 0) { this.resDlg = false; this.$message.success(r.data.msg || '已保存'); this.loadResCfgs() }
        else this.$message.error(r.msg)
      })
    },
    resetResCfg () {
      this.$confirm('把资源名称全部恢复为默认（黄金/粮食/钢铁/石油/稀矿）？', '提示', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-res-cfg/reset', {}).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已恢复'); this.loadResCfgs() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    loadSummary () {
      this.loadingSum = true
      api.get('/admin/ezfy-resources-summary').then(r => {
        this.loadingSum = false
        if (r.code === 0) this.sum = r.data
        else this.$message.error(r.msg)
      })
    },
    load () {
      this.loading = true
      api.get('/admin/ezfy-resources', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    openSet (row) {
      this.setRow = row
      this.setForm = { field: 'gold', value: row.gold }
      this.setDlg = true
    },
    doSet () {
      this.saving = true
      api.post('/admin/ezfy-resources/set', {
        city_id: this.setRow.id, field: this.setForm.field, value: this.setForm.value
      }).then(r => {
        this.saving = false
        if (r.code === 0) { this.setDlg = false; this.$message.success(r.data.msg || '已设置'); this.load(); this.loadSummary() }
        else this.$message.error(r.msg)
      })
    },
    openGrant () {
      this.grantForm = { ids: '', gold: 0, food: 0, steel: 0, oil: 0, rare: 0 }
      this.grantDlg = true
    },
    quickGrant (row) {
      this.grantForm = { ids: String(row.user_id), gold: 0, food: 0, steel: 0, oil: 0, rare: 0 }
      this.grantDlg = true
    },
    doGrant () {
      const ids = String(this.grantForm.ids).split(/[\s,，]+/).map(s => parseInt(s, 10)).filter(n => n > 0)
      if (!ids.length) { this.$message.warning('请填写玩家ID'); return }
      const { gold, food, steel, oil, rare } = this.grantForm
      if (!gold && !food && !steel && !oil && !rare) { this.$message.warning('请填写发放数量'); return }
      this.saving = true
      api.post('/admin/ezfy-resources/grant', { user_ids: ids, gold, food, steel, oil, rare }).then(r => {
        this.saving = false
        if (r.code === 0) { this.grantDlg = false; this.$message.success(r.data.msg || '已发放'); this.load(); this.loadSummary() }
        else this.$message.error(r.msg)
      })
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.card-head { display: flex; justify-content: space-between; align-items: center; }
.stat-card { background: #f5f7fa; border-radius: 6px; padding: 14px 10px; text-align: center; margin-bottom: 14px; }
.stat-num { font-size: 20px; font-weight: 700; color: #303133; }
.stat-label { font-size: 12px; color: #909399; margin-top: 4px; }
.sub-title { font-size: 13px; font-weight: 600; color: #1f2d3d; margin: 6px 0 8px; padding-left: 6px; border-left: 3px solid #409eff; }
</style>
