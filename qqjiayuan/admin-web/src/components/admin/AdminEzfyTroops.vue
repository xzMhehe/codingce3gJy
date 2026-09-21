<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <el-tabs v-model="tab" @tab-click="onTab">
        <!-- 玩家部队 -->
        <el-tab-pane label="玩家部队" name="troops">
          <div class="toolbar">
            <el-input v-model="word" placeholder="城名 / 城池ID" clearable style="width:200px"
                      @keyup.enter.native="page = 1; load()" />
            <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
            <div class="grow" />
            <el-button type="success" icon="el-icon-plus" @click="openGrant">增 / 减兵力</el-button>
            <el-button type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
          </div>
          <el-table :data="list" v-loading="loading" stripe border max-height="600">
            <el-table-column prop="id" label="ID" width="70" align="center" />
            <el-table-column prop="cfg_name" label="兵种" min-width="120" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.cfg_name || ('#' + row.troop_id) }}</span></template>
            </el-table-column>
            <el-table-column prop="type_name" label="类型" width="80" align="center" />
            <el-table-column label="数量" width="120" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ fmtN(row.count) }}</span></template>
            </el-table-column>
            <el-table-column label="占用人口" width="100" align="center">
              <template slot-scope="{row}">{{ fmtN(row.total_pop) }}</template>
            </el-table-column>
            <el-table-column prop="city_name" label="所属城池" width="120" show-overflow-tooltip />
            <el-table-column prop="owner_name" label="归属玩家" width="120" show-overflow-tooltip />
            <el-table-column prop="home_num" label="家园号" width="90" align="center" />
            <el-table-column label="操作" width="190" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="改数量" @click="openEdit(row)" />
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="清除" @click="del(row)" />
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
        </el-tab-pane>

        <!-- 兵种配置（可编辑） -->
        <el-tab-pane label="兵种配置" name="cfg">
          <div class="toolbar">
            <el-input v-model="cfgWord" placeholder="兵种名 / 同盟国名 / 轴心国名 / ID" clearable style="width:250px"
                      @keyup.enter.native="loadCfgs" />
            <el-button type="primary" icon="el-icon-search" @click="loadCfgs">查询</el-button>
            <div class="grow" />
            <span class="td-sub">共 {{ cfgs.length }} 种兵种，点「编辑」可改战斗参数与阵营名</span>
            <el-button type="primary" plain icon="el-icon-refresh" @click="loadCfgs">刷新</el-button>
          </div>
          <el-table :data="cfgPaged" v-loading="loadingCfg" stripe border>
            <el-table-column prop="id" label="ID" width="48" align="center" />
            <el-table-column prop="name" label="通用名" min-width="115" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column prop="name_ally" label="同盟国名" min-width="110" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-blue">{{ row.name_ally || '—' }}</span></template>
            </el-table-column>
            <el-table-column prop="name_axis" label="轴心国名" min-width="110" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-red">{{ row.name_axis || '—' }}</span></template>
            </el-table-column>
            <el-table-column prop="type_name" label="类型" width="62" align="center" />
            <el-table-column prop="health" label="生命" width="58" align="center" />
            <el-table-column label="对地" width="52" align="center"><template slot-scope="{row}">{{ row.atk_ground }}</template></el-table-column>
            <el-table-column label="对海" width="52" align="center"><template slot-scope="{row}">{{ row.atk_sea }}</template></el-table-column>
            <el-table-column label="对空" width="52" align="center"><template slot-scope="{row}">{{ row.atk_air }}</template></el-table-column>
            <el-table-column label="对防" width="52" align="center"><template slot-scope="{row}">{{ row.atk_def }}</template></el-table-column>
            <el-table-column prop="defence" label="防御" width="52" align="center" />
            <el-table-column prop="speed" label="速度" width="52" align="center" />
            <el-table-column label="训练(秒)" width="68" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.train_time }}</span></template>
            </el-table-column>
            <el-table-column :label="'造价(' + resShortText + ')'" width="155" align="center">
              <template slot-scope="{row}"><span class="td-mono td-small">{{ row.food }}/{{ row.steel }}/{{ row.oil }}/{{ row.rare }}</span></template>
            </el-table-column>
            <el-table-column label="操作" width="72" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openCfgEdit(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ cfgFiltered.length }}</b> 条 · 每页 {{ cfgSize }} 条</div>
            <el-pagination v-show="cfgFiltered.length > 0" small background layout="sizes, prev, pager, next, jumper" :total="cfgFiltered.length"
                           :page-size="cfgSize" :current-page="cfgPage" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { cfgPage = p }"
                           @size-change="s => { cfgSize = s; cfgPage = 1 }" />
          </div>
        </el-tab-pane>

        <!-- 伤兵 / 逃兵 -->
        <el-tab-pane label="伤兵 / 逃兵" name="wounded">
          <div class="toolbar">
            <el-input v-model="wWord" placeholder="城池ID" clearable style="width:160px"
                      @keyup.enter.native="wPage = 1; loadWounded()" />
            <el-select v-model="wType" style="width:130px" @change="wPage = 1; loadWounded()">
              <el-option label="全部" :value="-1" />
              <el-option label="伤兵" :value="0" />
              <el-option label="逃兵" :value="1" />
            </el-select>
            <el-button type="primary" icon="el-icon-search" @click="wPage = 1; loadWounded()">查询</el-button>
            <div class="grow" />
            <el-button type="primary" plain icon="el-icon-refresh" @click="loadWounded">刷新</el-button>
          </div>
          <el-table :data="wounded" v-loading="loadingW" stripe border max-height="600">
            <el-table-column prop="id" label="ID" width="70" align="center" />
            <el-table-column prop="cfg_name" label="兵种" min-width="120" show-overflow-tooltip />
            <el-table-column label="类型" width="90" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.type === 1 ? 'info' : 'warning'">{{ row.type_name }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="数量" width="120" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ fmtN(row.count) }}</span></template>
            </el-table-column>
            <el-table-column prop="city_name" label="所属城池" width="120" show-overflow-tooltip />
            <el-table-column prop="owner_name" label="归属玩家" width="120" show-overflow-tooltip />
            <el-table-column label="操作" width="120" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="清除" @click="delWounded(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ wTotal }}</b> 条 · 每页 {{ wSize }} 条</div>
            <el-pagination v-show="wTotal > 0" small background layout="sizes, prev, pager, next, jumper" :total="wTotal" :page-size="wSize"
                           :current-page="wPage" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { wPage = p; loadWounded() }"
                           @size-change="s => { wSize = s; wPage = 1; loadWounded() }" />
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 增减兵力 -->
    <el-dialog title="增 / 减兵力" :visible.sync="grantDlg" width="520px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="城池ID" required>
          <el-input-number v-model.number="grantForm.city_id" :min="1" controls-position="right" />
        </el-form-item>
        <el-form-item label="兵种" required>
          <el-select v-model="grantForm.troop_id" filterable style="width:280px">
            <el-option v-for="b in cfgs" :key="b.id" :label="b.id + ' · ' + b.name + '（' + b.type_name + '）'" :value="b.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="数量">
          <el-input-number v-model.number="grantForm.count" :controls="true" :min="-9999999" controls-position="right" />
          <span class="td-sub" style="margin-left:8px">正数增兵，负数扣减</span>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="grantDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doGrant">执 行</el-button>
      </div>
    </el-dialog>

    <!-- 改数量 -->
    <el-dialog title="修改兵力数量" :visible.sync="editDlg" width="440px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="兵种">
          <span class="td-main">{{ editRow.cfg_name }}（{{ editRow.city_name }}）</span>
        </el-form-item>
        <el-form-item label="数量">
          <el-input-number v-model.number="editCount" :min="0" controls-position="right" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="editDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doEdit">保 存</el-button>
      </div>
    </el-dialog>

    <!-- 编辑兵种配置 -->
    <el-dialog :title="'编辑兵种配置 · ' + cfgForm.name" :visible.sync="cfgDlg" width="960px"
               top="5vh" :close-on-click-modal="false">
      <el-form label-width="96px" size="small" class="cfg-form">
        <div class="sec-title">名称与阵营</div>
        <el-row :gutter="10">
          <el-col :span="8">
            <el-form-item label="通用名">
              <el-input v-model="cfgForm.name" maxlength="50" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="同盟国名">
              <el-input v-model="cfgForm.name_ally" maxlength="50" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="轴心国名">
              <el-input v-model="cfgForm.name_axis" maxlength="50" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="10">
          <el-col :span="8">
            <el-form-item label="兵种类型">
              <el-select v-model.number="cfgForm.type" style="width:100%">
                <el-option label="海军" :value="1" />
                <el-option label="陆军" :value="2" />
                <el-option label="空军" :value="3" />
                <el-option label="城防" :value="4" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="图标">
              <el-input v-model="cfgForm.icon" maxlength="50" placeholder="图标文件名（可空）" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="战损修复率%">
              <el-input-number v-model.number="cfgForm.repair_rate" :min="0" :max="100" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>

        <div class="sec-title">战斗属性</div>
        <el-row :gutter="10">
          <el-col :span="8"><el-form-item label="生命"><el-input-number v-model.number="cfgForm.health" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="对地"><el-input-number v-model.number="cfgForm.atk_ground" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="对海"><el-input-number v-model.number="cfgForm.atk_sea" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="对空"><el-input-number v-model.number="cfgForm.atk_air" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="10">
          <el-col :span="8"><el-form-item label="对防"><el-input-number v-model.number="cfgForm.atk_def" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="防御"><el-input-number v-model.number="cfgForm.defence" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="速度"><el-input-number v-model.number="cfgForm.speed" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="射程"><el-input-number v-model.number="cfgForm.attack_range" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="10">
          <el-col :span="8"><el-form-item label="载量"><el-input-number v-model.number="cfgForm.carry" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="人口"><el-input-number v-model.number="cfgForm.pop" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="训练(秒)"><el-input-number v-model.number="cfgForm.train_time" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
        </el-row>

        <div class="sec-title">消耗与维护</div>
        <el-row :gutter="10">
          <el-col :span="8"><el-form-item label="耗粮/时"><el-input-number v-model.number="cfgForm.food_keep" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="耗油/时"><el-input-number v-model.number="cfgForm.oil_keep" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="10">
          <el-col :span="8"><el-form-item label="造价-粮"><el-input-number v-model.number="cfgForm.food" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="造价-钢"><el-input-number v-model.number="cfgForm.steel" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="造价-油"><el-input-number v-model.number="cfgForm.oil" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="造价-稀"><el-input-number v-model.number="cfgForm.rare" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="前置要求">
          <el-input v-model="cfgForm.require" maxlength="500" placeholder="例如：军工厂3级" />
        </el-form-item>
      </el-form>
      <em>保存后立即生效（后端会重载配置缓存，不用重启）</em>
      <div slot="footer">
        <el-button @click="cfgDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doCfgSave">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

const CFG_KEYS = [
  'name', 'name_ally', 'name_axis', 'type', 'icon', 'repair_rate',
  'health', 'atk_ground', 'atk_sea', 'atk_air', 'atk_def', 'defence', 'speed', 'attack_range',
  'carry', 'pop', 'train_time', 'food_keep', 'oil_keep',
  'food', 'steel', 'oil', 'rare', 'require'
]

export default {
  name: 'AdminEzfyTroops',
  data () {
    return {
      tab: 'troops',
      list: [], total: 0, page: 1, size: 5, loading: false, word: '',
      cfgs: [], cfgWord: '', loadingCfg: false, cfgPage: 1, cfgSize: 5,
      resNames: {},
      wounded: [], wTotal: 0, wPage: 1, wSize: 5, loadingW: false, wWord: '', wType: -1,
      grantDlg: false, grantForm: { city_id: 1, troop_id: 0, count: 100 },
      editDlg: false, editRow: {}, editCount: 0,
      cfgDlg: false, cfgForm: {},
      saving: false
    }
  },
  computed: {
    // 资源单字简称（管理端可改名，取自「资源管理 → 资源名称维护」）
    resShortText () {
      const s = this.resNames._short || {}
      return [s.food || '粮', s.steel || '钢', s.oil || '油', s.rare || '稀'].join('/')
    },
    // 兵种配置前端过滤（后端接口一次给全，方便按阵营名搜）
    cfgFiltered () {
      const w = String(this.cfgWord || '').trim().toLowerCase()
      if (!w) return this.cfgs
      return this.cfgs.filter(b =>
        String(b.id) === w ||
        String(b.name || '').toLowerCase().indexOf(w) >= 0 ||
        String(b.name_ally || '').toLowerCase().indexOf(w) >= 0 ||
        String(b.name_axis || '').toLowerCase().indexOf(w) >= 0
      )
    },
    // 兵种配置前端分页（后端一次给全，本地切片）
    cfgPaged () {
      const st = (this.cfgPage - 1) * this.cfgSize
      return this.cfgFiltered.slice(st, st + this.cfgSize)
    }
  },
  watch: {
    // 搜索词变了回到第 1 页
    cfgWord () { this.cfgPage = 1 }
  },
  mounted () { this.load(); this.loadCfgs(); this.loadResNames() },
  methods: {
    fmtN (v) {
      if (v === null || v === undefined) return '—'
      return Number(v).toLocaleString()
    },
    onTab () {
      if (this.tab === 'wounded') this.loadWounded()
    },
    load () {
      this.loading = true
      api.get('/admin/ezfy-troops', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    // 资源显示名（管理端可改名，前端列头跟随）
    loadResNames () {
      api.get('/admin/ezfy-res-cfg').then(r => {
        if (r.code === 0) {
          const m = {}
          ;(r.data.list || []).forEach(x => { m[x.key] = x.name })
          m._short = {}
          ;(r.data.list || []).forEach(x => { m._short[x.key] = x.short })
          this.resNames = m
        }
      }).catch(() => {})
    },
    loadCfgs () {
      this.loadingCfg = true
      api.get('/admin/ezfy-troops-cfg').then(r => {
        this.loadingCfg = false
        if (r.code === 0) this.cfgs = r.data.list
        else this.$message.error(r.msg)
      })
    },
    loadWounded () {
      this.loadingW = true
      api.get('/admin/ezfy-wounded', {
        params: { page: this.wPage, size: this.wSize, city_id: this.wWord, type: this.wType }
      }).then(r => {
        this.loadingW = false
        if (r.code === 0) {
          this.wounded = r.data.list
          this.wTotal = r.data.total
          this.wPage = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    openGrant () {
      this.grantForm = { city_id: 1, troop_id: this.cfgs.length ? this.cfgs[0].id : 0, count: 100 }
      this.grantDlg = true
    },
    doGrant () {
      if (!this.grantForm.troop_id) { this.$message.warning('请选择兵种'); return }
      if (!this.grantForm.count) { this.$message.warning('数量不能为 0'); return }
      this.saving = true
      api.post('/admin/ezfy-troops/grant', this.grantForm).then(r => {
        this.saving = false
        if (r.code === 0) { this.grantDlg = false; this.$message.success(r.data.msg || '已执行'); this.load() }
        else this.$message.error(r.msg)
      })
    },
    openEdit (row) {
      this.editRow = row
      this.editCount = row.count
      this.editDlg = true
    },
    doEdit () {
      this.saving = true
      api.put('/admin/ezfy-troops/' + this.editRow.id, { count: this.editCount }).then(r => {
        this.saving = false
        if (r.code === 0) { this.editDlg = false; this.$message.success(r.data.msg || '已保存'); this.load() }
        else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm('确认清除「' + row.city_name + '」的 ' + row.cfg_name + '？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-troops/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已清除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    delWounded (row) {
      api.delete('/admin/ezfy-wounded/' + row.id).then(r => {
        if (r.code === 0) { this.$message.success(r.data.msg || '已清除'); this.loadWounded() } else this.$message.error(r.msg)
      })
    },
    // ---- 兵种配置编辑 ----
    openCfgEdit (row) {
      const f = {}
      CFG_KEYS.forEach(k => { f[k] = row[k] })
      this.cfgForm = f
      this.cfgDlg = true
    },
    doCfgSave () {
      if (!String(this.cfgForm.name || '').trim()) { this.$message.warning('通用名不能为空'); return }
      this.saving = true
      api.put('/admin/ezfy-troops-cfg/' + this.cfgForm.id, this.cfgForm).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.cfgDlg = false
          this.$message.success(r.data.msg || '已保存')
          this.loadCfgs()
        } else this.$message.error(r.msg)
      })
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.td-red { color: #e6a23c; font-weight: 600; }
.td-small { font-size: 12px; }
.sec-title {
  font-size: 13px; font-weight: 700; color: #1f2d3d;
  margin: 4px 0 10px; padding-left: 7px; border-left: 3px solid #409eff;
}
.cfg-form >>> .el-form-item { margin-bottom: 12px; }
</style>
