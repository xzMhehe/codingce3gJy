<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <el-tabs v-model="tab">
        <!-- ================= 玩家科技 ================= -->
        <el-tab-pane label="玩家科技" name="list">
          <div class="td-sub" style="margin-bottom:8px">
            科技归玩家所有、与城市无关；城市需建设「科研中心」后，科技加成才会生效
          </div>
          <div class="toolbar">
            <el-input v-model="word" placeholder="玩家名 / 家园号 / 玩家ID" clearable style="width:200px"
                      @keyup.enter.native="page = 1; load()" />
            <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
            <div class="grow" />
            <el-button type="warning" icon="el-icon-upload2" @click="maxAll">一键满级所有玩家科技</el-button>
            <el-button type="success" icon="el-icon-setting" @click="openSet">设置科技等级</el-button>
            <el-button type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
          </div>
          <el-table :data="list" v-loading="loading" stripe border>
            <el-table-column prop="id" label="ID" width="70" align="center" />
            <el-table-column prop="cfg_name" label="科技" min-width="150" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.cfg_name || ('#' + row.tech_id) }}</span></template>
            </el-table-column>
            <el-table-column label="等级" width="115" align="center">
              <template slot-scope="{row}">
                <span class="lv">Lv.</span>{{ row.level }}<span class="td-muted"> / {{ row.max_level }}</span>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="110" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.status === 1 ? 'warning' : 'success'">{{ row.status_txt }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="剩余时间" width="130" align="center">
              <template slot-scope="{row}">
                <span v-if="row.status === 1" :class="row.remain_sec <= 0 ? 'td-blue' : ''">{{ fmtRemain(row.remain_sec) }}</span>
                <span v-else class="td-muted">—</span>
              </template>
            </el-table-column>
            <el-table-column prop="owner_name" label="归属玩家" min-width="130" show-overflow-tooltip />
            <el-table-column prop="home_num" label="家园号" width="95" align="center" />
            <el-table-column label="操作" width="190" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="改等级" @click="openEdit(row)" />
                <el-button size="mini" type="success" plain icon="el-icon-check" title="立即完成研究"
                           :disabled="row.status !== 1" @click="finish(row)" />
                <el-button size="mini" type="danger" plain icon="el-icon-refresh-left" title="重置该科技" @click="del(row)" />
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

        <!-- ================= 科技配置（可维护） ================= -->
        <el-tab-pane label="科技配置" name="cfg">
          <div class="toolbar">
            <el-input v-model="cfgWord" placeholder="科技名 / ID" clearable style="width:220px"
                      @keyup.enter.native="cfgPage = 1; loadCfgs()" />
            <el-button type="primary" icon="el-icon-search" @click="cfgPage = 1; loadCfgs()">查询</el-button>
            <div class="grow" />
            <el-button type="success" icon="el-icon-plus" @click="openCfgCreate">新增科技</el-button>
            <el-button type="primary" plain icon="el-icon-refresh" @click="loadCfgs">刷新</el-button>
          </div>
          <el-table :data="cfgs" v-loading="loadingCfg" stripe border>
            <el-table-column prop="id" label="ID" width="55" align="center" />
            <el-table-column prop="name" label="科技名" min-width="150" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column label="类型" width="80" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="techTag(row.type)">{{ row.type_name || techTypes[row.type] || '—' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="max_level" label="最高等级" width="88" align="center" />
            <el-table-column prop="pre_building" label="需科研中心" width="95" align="center" />
            <el-table-column label="前置科技" width="100" align="center">
              <template slot-scope="{row}">
                <span v-if="row.pre_tech">#{{ row.pre_tech }}·Lv.{{ row.pre_tech_level }}</span>
                <span v-else class="td-muted">—</span>
              </template>
            </el-table-column>
            <el-table-column label="等级配置" width="88" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.level_count }}</span></template>
            </el-table-column>
            <el-table-column label="已研究" width="85" align="center">
              <template slot-scope="{row}"><span class="td-mono">{{ row.owned_count }}</span></template>
            </el-table-column>
            <el-table-column prop="effect" label="效果" min-width="250" show-overflow-tooltip />
            <el-table-column label="操作" width="200" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="info" plain icon="el-icon-s-operation" title="等级配置" @click="openLvDlg(row)" />
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openCfgEdit(row)" />
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="delCfg(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ cfgTotal }}</b> 条 · 每页 {{ cfgSize }} 条</div>
            <el-pagination v-show="cfgTotal > 0" small background layout="sizes, prev, pager, next, jumper" :total="cfgTotal"
                           :page-size="cfgSize" :current-page="cfgPage" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { cfgPage = p; loadCfgs() }"
                           @size-change="s => { cfgSize = s; cfgPage = 1; loadCfgs() }" />
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 设置玩家科技等级 -->
    <el-dialog title="设置玩家科技等级" :visible.sync="setDlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="玩家ID" required>
          <el-input-number v-model.number="setForm.user_id" :min="1" controls-position="right" style="width:100%" />
        </el-form-item>
        <el-form-item label="科技" required>
          <el-select v-model="setForm.tech_id" filterable style="width:100%">
            <el-option v-for="t in allTechs" :key="t.id" :label="t.id + ' · ' + t.name + '（' + (t.type_name || techTypes[t.type] || '') + '）'" :value="t.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="等级">
          <el-input-number v-model.number="setForm.level" :min="0" controls-position="right" style="width:100%" />
          <span class="td-sub">0 = 未掌握</span>
        </el-form-item>
      </el-form>
      <!-- [说明·不显示在界面] 提示：该玩家科技数据自动落在其主城（科技城），主城无此科技时会自动创建，等级会被自动截断到配置上限 -->
      <div slot="footer">
        <el-button @click="setDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doSet">设 置</el-button>
      </div>
    </el-dialog>

    <!-- 改玩家科技等级 -->
    <el-dialog title="修改科技" :visible.sync="editDlg" width="520px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="科技">
          <span class="td-main">{{ editRow.cfg_name }}（{{ editRow.owner_name }}）</span>
        </el-form-item>
        <el-form-item label="等级">
          <el-input-number v-model.number="editLevel" :min="0" controls-position="right" style="width:100%" />
          <span class="td-sub">最高 {{ editRow.max_level }} 级</span>
        </el-form-item>
        <el-form-item label="研究状态">
          <el-radio-group v-model="editStatus">
            <el-radio :label="0">已掌握</el-radio>
            <el-radio :label="1">研究中</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="editDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doEdit">保 存</el-button>
      </div>
    </el-dialog>

    <!-- 新增 / 编辑 科技配置 -->
    <el-dialog :title="cf.id ? ('编辑科技 · ' + cf.name) : '新增科技'" :visible.sync="cfDlg"
               width="820px" :close-on-click-modal="false">
      <el-form label-width="100px" size="small">
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="科技名" required>
              <el-input v-model="cf.name" maxlength="50" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="类型">
              <el-select v-model.number="cf.type" style="width:100%">
                <el-option label="生产" :value="1" />
                <el-option label="军事" :value="2" />
                <el-option label="辅助" :value="3" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="最高等级">
              <el-input-number v-model.number="cf.max_level" :min="1" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="需科研中心">
              <el-input-number v-model.number="cf.pre_building" :min="0" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="前置科技">
              <el-select v-model.number="cf.pre_tech" filterable clearable placeholder="无" style="width:100%">
                <el-option v-for="t in allTechs" :key="t.id" :label="t.id + ' · ' + t.name" :value="t.id" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="前置等级">
              <el-input-number v-model.number="cf.pre_tech_level" :min="0" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="效果">
          <el-input v-model="cf.effect" maxlength="255" placeholder="例如：粮食产量+10%" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="cf.des" type="textarea" :rows="2" maxlength="500" show-word-limit />
        </el-form-item>
      </el-form>
      <!-- [说明·不显示在界面] 保存后立即生效（后端会重载配置缓存，不用重启） -->
      <div slot="footer">
        <el-button @click="cfDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doCfgSave">保 存</el-button>
      </div>
    </el-dialog>

    <!-- 科技等级配置 -->
    <el-dialog :title="'等级配置 · ' + lvTech.name + '（' + lvRows.length + ' 条）'" :visible.sync="lvDlg"
               width="1080px" top="6vh" :close-on-click-modal="false">
      <div class="toolbar">
        <el-button type="success" size="small" icon="el-icon-plus" @click="openLvCreate">新增等级</el-button>
        <div class="grow" />
        <span class="td-sub">研究耗时单位：秒；四项资源为研究花费</span>
      </div>
      <el-table :data="lvRows" v-loading="loadingLv" size="mini" stripe border max-height="460">
        <el-table-column prop="id" label="ID" width="55" align="center" />
        <el-table-column label="等级" width="60" align="center">
          <template slot-scope="{row}"><span class="lv">Lv.</span>{{ row.level }}</template>
        </el-table-column>
        <el-table-column label="粮" width="90" align="center"><template slot-scope="{row}"><span class="td-mono td-small">{{ fmtN(row.food) }}</span></template></el-table-column>
        <el-table-column label="钢" width="90" align="center"><template slot-scope="{row}"><span class="td-mono td-small">{{ fmtN(row.steel) }}</span></template></el-table-column>
        <el-table-column label="油" width="90" align="center"><template slot-scope="{row}"><span class="td-mono td-small">{{ fmtN(row.oil) }}</span></template></el-table-column>
        <el-table-column label="稀" width="90" align="center"><template slot-scope="{row}"><span class="td-mono td-small">{{ fmtN(row.rare) }}</span></template></el-table-column>
        <el-table-column label="黄金" width="90" align="center"><template slot-scope="{row}"><span class="td-mono td-small">{{ fmtN(row.gold) }}</span></template></el-table-column>
        <el-table-column label="研究(秒)" width="85" align="center"><template slot-scope="{row}"><span class="td-mono">{{ row.research_time }}</span></template></el-table-column>
        <el-table-column prop="effect" label="效果" min-width="250" show-overflow-tooltip />
        <el-table-column label="操作" width="130" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openLvEdit(row)" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="delLv(row)" />
          </template>
        </el-table-column>
      </el-table>
      <div slot="footer">
        <el-button @click="lvDlg = false">关 闭</el-button>
      </div>
    </el-dialog>

    <!-- 新增 / 编辑 科技等级 -->
    <el-dialog :title="lvForm.id ? ('编辑等级 · Lv.' + lvForm.level) : '新增等级'" :visible.sync="lvEditDlg"
               width="880px" append-to-body :close-on-click-modal="false">
      <el-form label-width="100px" size="small">
        <el-row :gutter="12">
          <el-col :span="8">
            <el-form-item label="等级" required>
              <el-input-number v-model.number="lvForm.level" :min="1" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="研究(秒)">
              <el-input-number v-model.number="lvForm.research_time" :min="0" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="8">
            <el-form-item :label="resName('food')">
              <el-input-number v-model.number="lvForm.food" :min="0" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item :label="resName('steel')">
              <el-input-number v-model.number="lvForm.steel" :min="0" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item :label="resName('oil')">
              <el-input-number v-model.number="lvForm.oil" :min="0" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="8">
            <el-form-item :label="resName('rare')">
              <el-input-number v-model.number="lvForm.rare" :min="0" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item :label="resName('gold')">
              <el-input-number v-model.number="lvForm.gold" :min="0" controls-position="right" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="效果">
          <el-input v-model="lvForm.effect" maxlength="255" placeholder="例如：粮食产量+1%" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="lvEditDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="doLvSave">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

const CF_KEYS = ['name', 'type', 'max_level', 'pre_building', 'pre_tech', 'pre_tech_level', 'effect', 'des']
const LV_KEYS = ['tech_id', 'level', 'food', 'steel', 'oil', 'rare', 'gold', 'research_time', 'effect']

function emptyCf () {
  return { id: 0, name: '', type: 3, max_level: 10, pre_building: 0, pre_tech: null, pre_tech_level: 0, effect: '', des: '' }
}
function emptyLv () {
  return { id: 0, tech_id: 0, level: 1, food: 0, steel: 0, oil: 0, rare: 0, gold: 0, research_time: 0, effect: '' }
}

export default {
  name: 'AdminEzfyTechs',
  data () {
    return {
      tab: 'list',
      list: [], total: 0, page: 1, size: 5, loading: false, word: '',
      cfgs: [], cfgTotal: 0, cfgPage: 1, cfgSize: 5, cfgWord: '', loadingCfg: false,
      allTechs: [],
      techTypes: { 1: '生产', 2: '军事', 3: '辅助' },
      resNames: {},
      setDlg: false, setForm: { user_id: 1, tech_id: 0, level: 1 },
      editDlg: false, editRow: {}, editLevel: 0, editStatus: 0,
      cfDlg: false, cf: emptyCf(),
      lvDlg: false, lvTech: {}, lvRows: [], loadingLv: false,
      lvEditDlg: false, lvForm: emptyLv(),
      saving: false
    }
  },
  mounted () { this.load(); this.loadCfgs(); this.loadResNames() },
  methods: {
    techTag (t) {
      return ({ 1: 'success', 2: 'danger', 3: 'primary' })[t] || 'info'
    },
    resName (k) {
      return this.resNames[k] || ({ food: '粮食', steel: '钢铁', oil: '石油', rare: '稀矿', gold: '黄金' })[k]
    },
    fmtN (v) {
      if (v === null || v === undefined) return '—'
      return Number(v).toLocaleString()
    },
    fmtRemain (sec) {
      if (sec <= 0) return '可完成'
      const h = Math.floor(sec / 3600)
      const m = Math.floor((sec % 3600) / 60)
      const s = sec % 60
      if (h > 0) return h + ' 时 ' + m + ' 分'
      if (m > 0) return m + ' 分 ' + s + ' 秒'
      return s + ' 秒'
    },
    // 资源显示名（管理端可改名，前端跟随）
    loadResNames () {
      api.get('/admin/ezfy-res-cfg').then(r => {
        if (r.code === 0) {
          const m = {}
          ;(r.data.list || []).forEach(x => { m[x.key] = x.name })
          this.resNames = m
        }
      }).catch(() => {})
    },
    load () {
      this.loading = true
      api.get('/admin/ezfy-techs', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list
          this.total = r.data.total
          this.page = r.data.page
        } else this.$message.error(r.msg)
      })
    },
    // ---- 科技配置 ----
    loadCfgs () {
      this.loadingCfg = true
      api.get('/admin/ezfy-tech-cfg', { params: { page: this.cfgPage, size: this.cfgSize, word: this.cfgWord } }).then(r => {
        this.loadingCfg = false
        if (r.code === 0) {
          this.cfgs = r.data.list || []
          this.cfgTotal = r.data.total || 0
        } else this.$message.error(r.msg)
      })
    },
    // 下拉用全量科技（不分页）
    loadAllTechs () {
      api.get('/admin/ezfy-tech-cfg', { params: { page: 1, size: 500 } }).then(r => {
        if (r.code === 0) this.allTechs = r.data.list || []
      })
    },
    openCfgCreate () {
      this.cf = emptyCf()
      this.cfDlg = true
    },
    openCfgEdit (row) {
      this.cf = Object.assign(emptyCf(), row)
      this.cfDlg = true
    },
    doCfgSave () {
      if (!String(this.cf.name || '').trim()) { this.$message.warning('请填写科技名'); return }
      const body = {}
      CF_KEYS.forEach(k => { if (this.cf[k] !== null && this.cf[k] !== undefined && this.cf[k] !== '') body[k] = this.cf[k] })
      this.saving = true
      const req = this.cf.id
        ? api.put('/admin/ezfy-tech-cfg/' + this.cf.id, body)
        : api.post('/admin/ezfy-tech-cfg', body)
      req.then(r => {
        this.saving = false
        if (r.code === 0) {
          this.cfDlg = false
          this.$message.success(r.data.msg || '已保存')
          this.loadCfgs(); this.loadAllTechs()
        } else this.$message.error(r.msg)
      })
    },
    delCfg (row) {
      this.$confirm('删除科技「' + row.name + '」会连带删除它的所有等级配置，确认？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-tech-cfg/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadCfgs(); this.loadAllTechs() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ---- 等级配置 ----
    openLvDlg (row) {
      this.lvTech = row
      this.lvDlg = true
      this.loadLevels()
    },
    loadLevels () {
      this.loadingLv = true
      api.get('/admin/ezfy-tech-levels', { params: { tech_id: this.lvTech.id } }).then(r => {
        this.loadingLv = false
        if (r.code === 0) this.lvRows = r.data.list || []
        else this.$message.error(r.msg)
      })
    },
    openLvCreate () {
      this.lvForm = emptyLv()
      this.lvForm.tech_id = this.lvTech.id
      this.lvForm.level = (this.lvRows.length ? Math.max.apply(null, this.lvRows.map(x => x.level)) : 0) + 1
      this.lvEditDlg = true
    },
    openLvEdit (row) {
      this.lvForm = Object.assign(emptyLv(), row)
      this.lvEditDlg = true
    },
    doLvSave () {
      const body = { tech_id: this.lvForm.tech_id }
      LV_KEYS.forEach(k => { if (this.lvForm[k] !== null && this.lvForm[k] !== undefined) body[k] = this.lvForm[k] })
      this.saving = true
      const req = this.lvForm.id
        ? api.put('/admin/ezfy-tech-levels/' + this.lvForm.id, body)
        : api.post('/admin/ezfy-tech-levels', body)
      req.then(r => {
        this.saving = false
        if (r.code === 0) {
          this.lvEditDlg = false
          this.$message.success(r.data.msg || '已保存')
          this.loadLevels(); this.loadCfgs()
        } else this.$message.error(r.msg)
      })
    },
    delLv (row) {
      this.$confirm('删除 Lv.' + row.level + ' 的等级配置？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-tech-levels/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadLevels(); this.loadCfgs() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ---- 玩家科技 ----
    // 一键把所有玩家所有城市的科技升到满级（后端幂等：先去重再 upsert）
    maxAll () {
      this.$confirm('确定把所有玩家的科技**全部升到满级**吗？此操作会覆盖玩家已有的科技等级。',
        '一键满级', { type: 'warning', confirmButtonText: '确定满级', cancelButtonText: '取消' }).then(() => {
        this.saving = true
        api.post('/admin/ezfy-techs/max-all', {}).then(r => {
          this.saving = false
          if (r.code === 0) {
            this.$message.success(r.data.msg || '已满级')
            if (r.data.dup_left > 0) this.$message.warning('检测到仍有重复行 ' + r.data.dup_left + ' 条，请刷新查看')
            this.page = 1
            this.load()
          } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    openSet () {
      if (!this.allTechs.length) this.loadAllTechs()
      this.setForm = { user_id: 1, tech_id: this.allTechs.length ? this.allTechs[0].id : 0, level: 1 }
      this.setDlg = true
    },
    doSet () {
      if (!this.setForm.user_id) { this.$message.warning('请填写玩家ID'); return }
      if (!this.setForm.tech_id) { this.$message.warning('请选择科技'); return }
      this.saving = true
      api.post('/admin/ezfy-techs/set', this.setForm).then(r => {
        this.saving = false
        if (r.code === 0) { this.setDlg = false; this.$message.success(r.data.msg || '已设置'); this.load() }
        else this.$message.error(r.msg)
      })
    },
    openEdit (row) {
      this.editRow = row
      this.editLevel = row.level
      this.editStatus = row.status
      this.editDlg = true
    },
    doEdit () {
      this.saving = true
      api.put('/admin/ezfy-techs/' + this.editRow.id, { level: this.editLevel, status: this.editStatus }).then(r => {
        this.saving = false
        if (r.code === 0) { this.editDlg = false; this.$message.success(r.data.msg || '已保存'); this.load() }
        else this.$message.error(r.msg)
      })
    },
    finish (row) {
      api.post('/admin/ezfy-techs/' + row.id + '/finish').then(r => {
        if (r.code === 0) { this.$message.success(r.data.msg || '已完成'); this.load() } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm('重置后玩家「' + row.owner_name + '」的【' + row.cfg_name + '】将被删除（等级归零），确认？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-techs/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已重置'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  },
  watch: {
    cfgWord () { this.cfgPage = 1 },
    tab (v) { if (v === 'cfg') { this.loadCfgs(); this.loadAllTechs() } }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
</style>
