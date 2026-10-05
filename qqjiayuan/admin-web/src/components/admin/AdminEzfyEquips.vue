<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <!-- ★ 2026-09-27 装备「属性」单独起新菜单「军官装备管理」，
           这里只管属性（三维/战斗属性/部位/品质/系列/强化），价格与库存
           已迁到「数据管理 → 装备道具配置 / 套装装备配置」维护。 -->
      <el-tabs v-model="tab" @tab-click="onTab">
        <!-- ============ 1. 散件装备（set_id=0） ============ -->
        <el-tab-pane label="散件装备" name="loose">
          <div class="toolbar">
            <el-input v-model="eWord" placeholder="装备名 / 部位 / 类型 / ID" clearable style="width:220px"
                      @keyup.enter.native="loadEquips" />
            <el-button type="primary" icon="el-icon-search" @click="loadEquips">查询</el-button>
            <span class="td-sub">散件商城售价 / 库存请到「数据管理 → 装备道具配置」维护</span>
            <div class="grow" />
            <el-button type="success" icon="el-icon-plus" @click="openEquipCreate">新增散件</el-button>
          </div>
          <el-table :data="ePaged" v-loading="loadingE" stripe border>
            <el-table-column prop="id" label="ID" width="55" align="center" />
            <el-table-column prop="name" label="装备名" min-width="170" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column prop="type" label="类型" width="70" align="center" />
            <el-table-column label="部位" width="70" align="center">
              <template slot-scope="{row}">{{ row.slot || row.type }}</template>
            </el-table-column>
            <el-table-column label="品质" width="70" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="tierTag(row.tier)">{{ row.tier_name || tierName(row.tier) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="military" label="军事" width="60" align="center" />
            <el-table-column prop="logistics" label="后勤" width="55" align="center" />
            <el-table-column prop="learning" label="学识" width="55" align="center" />
            <el-table-column label="战斗属性" min-width="190" show-overflow-tooltip>
              <template slot-scope="{row}">{{ equipAttrText(row) }}</template>
            </el-table-column>
            <el-table-column prop="level" label="需求等级" width="80" align="center" />
            <el-table-column prop="series" label="系列" width="90" show-overflow-tooltip>
              <template slot-scope="{row}">
                <span v-if="row.series" class="td-main">{{ row.series }}</span>
                <span v-else class="td-sub">—</span>
              </template>
            </el-table-column>
            <el-table-column label="强化" width="70" align="center">
              <template slot-scope="{row}">
                <span v-if="row.enhance" class="td-mono">{{ row.enhance }}/{{ row.enhance_max }}</span>
                <span v-else class="td-sub">—</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="140" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openEquipEdit(row)" />
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="delEquip(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-bar">
            <div class="pager-info">共 <b>{{ loose.length }}</b> 条 · 每页 {{ eSize }} 条</div>
            <el-pagination v-show="loose.length > 0" small background layout="sizes, prev, pager, next, jumper" :total="loose.length"
                           :page-size="eSize" :current-page="ePage" :page-sizes="[5, 10, 20, 50, 100]"
                           @current-change="p => { ePage = p }"
                           @size-change="s => { eSize = s; ePage = 1 }" />
          </div>
        </el-tab-pane>

        <!-- ============ 2. 套装管理（ezfy_cfg_equip_set + 套装件属性） ============ -->
        <el-tab-pane label="套装管理" name="sets">
          <div class="toolbar">
            <el-input v-model="stWord" placeholder="套装名 / ID" clearable style="width:200px"
                      @keyup.enter.native="loadSets" />
            <el-button type="primary" icon="el-icon-search" @click="loadSets">查询</el-button>
            <span class="td-sub">穿戴同套 N 件即触发套装加成；套装件的商城售价 / 库存到「数据管理 → 装备道具配置」维护</span>
            <div class="grow" />
            <el-button type="success" icon="el-icon-plus" @click="openSetCreate">新增套装</el-button>
          </div>
          <el-table :data="sets" v-loading="loadingSt" stripe border>
            <el-table-column prop="id" label="ID" width="55" align="center" />
            <el-table-column prop="name" label="套装名" min-width="190" show-overflow-tooltip>
              <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
            </el-table-column>
            <el-table-column prop="parts" label="触发件数" width="80" align="center" />
            <el-table-column prop="military" label="军事" width="60" align="center" />
            <el-table-column prop="logistics" label="后勤" width="60" align="center" />
            <el-table-column prop="learning" label="学识" width="60" align="center" />
            <el-table-column label="额外战斗属性" min-width="170" show-overflow-tooltip>
              <template slot-scope="{row}">{{ equipAttrText(row) }}</template>
            </el-table-column>
            <el-table-column label="各件之和（另计）" min-width="170" show-overflow-tooltip>
              <template slot-scope="{row}">
                {{ equipAttrText({
                  dmg: row.piece_sum_dmg, def: row.piece_sum_def, hp: row.piece_sum_hp,
                  move: row.piece_sum_move, crit: row.piece_sum_crit, crit_dmg: row.piece_sum_crit_dmg,
                  military: row.piece_sum_military, logistics: row.piece_sum_logistics, learning: row.piece_sum_learning
                }) }}
              </template>
            </el-table-column>
            <el-table-column label="已配件数" width="80" align="center">
              <template slot-scope="{row}">
                <el-tag size="mini" :type="row.piece_count >= row.parts ? 'success' : 'warning'">{{ row.piece_count }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="sale_count" label="已上架" width="70" align="center" />
            <el-table-column label="操作" width="200" align="center" fixed="right">
              <template slot-scope="{row}">
                <el-button size="mini" type="success" plain icon="el-icon-view" title="查看/编辑套装件" @click="openSetPieces(row)" />
                <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openSetEdit(row)" />
                <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="delSet(row)" />
              </template>
            </el-table-column>
          </el-table>
          <div class="pager-info" style="margin-top:8px">共 <b>{{ sets.length }}</b> 个套装</div>
        </el-tab-pane>
      </el-tabs>

      <!-- ============ 装备属性编辑（散件新增/编辑 + 套装件编辑共用；不含价格/库存/所属套装） ============ -->
      <el-dialog :title="ef.id ? ('编辑装备 · ' + ef.name) : '新增散件'" :visible.sync="eDlg"
                 width="880px" :close-on-click-modal="false">
        <el-form label-width="100px" size="small">
          <el-row :gutter="10">
            <el-col :span="12">
              <el-form-item label="装备名称" required>
                <el-input v-model="ef.name" maxlength="50" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="类型">
                <el-select v-model="ef.type" style="width:100%" :disabled="efSetId > 0">
                  <el-option label="武器" value="武器" />
                  <el-option label="防具" value="防具" />
                  <el-option label="饰品" value="饰品" />
                  <el-option label="珠宝" value="珠宝" />
                  <el-option label="套装" value="套装" />
                </el-select>
                <div v-if="efSetId > 0" class="td-sub" style="margin-top:2px">套装件类型固定为「套装」</div>
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="10">
            <el-col :span="12">
              <el-form-item label="品质">
                <el-select v-model.number="ef.tier" style="width:100%">
                  <el-option label="初级" :value="1" />
                  <el-option label="中级" :value="2" />
                  <el-option label="高级" :value="3" />
                  <el-option label="特殊" :value="4" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="穿戴等级需求">
                <el-input-number v-model.number="ef.level" :min="1" controls-position="right" style="width:100%" />
              </el-form-item>
            </el-col>
          </el-row>
          <el-form-item label="穿戴部位">
            <el-select v-model="ef.slot" filterable allow-create default-first-option style="width:100%"
                       placeholder="留空则用「类型」当部位">
              <el-option v-for="s in slotOptions" :key="s" :label="s" :value="s" />
            </el-select>
          </el-form-item>
          <el-row :gutter="10">
            <el-col :span="8"><el-form-item label="军事加成"><el-input-number v-model.number="ef.military" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
            <el-col :span="8"><el-form-item label="后勤加成"><el-input-number v-model.number="ef.logistics" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
            <el-col :span="8"><el-form-item label="学识加成"><el-input-number v-model.number="ef.learning" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          </el-row>
          <el-form-item label="额外效果">
            <el-input v-model="ef.effect" maxlength="200" placeholder="例如：攻速+40% / 装备+20" />
          </el-form-item>
          <el-row :gutter="10">
            <el-col :span="8">
              <el-form-item label="系列">
                <el-input v-model="ef.series" maxlength="30" placeholder="革命者 / 渡鸦之魂 / 空=散件" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="强化等级">
                <el-input-number v-model.number="ef.enhance" :min="0" controls-position="right" style="width:100%" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="强化上限">
                <el-input-number v-model.number="ef.enhance_max" :min="0" controls-position="right" style="width:100%" />
              </el-form-item>
            </el-col>
          </el-row>
          <el-form-item label="战斗属性(%)">
            <div class="attr-row">
              伤害 <el-input-number v-model.number="ef.dmg" :min="0" size="mini" controls-position="right" style="width:96px" />
              防御 <el-input-number v-model.number="ef.def" :min="0" size="mini" controls-position="right" style="width:96px" />
              生命 <el-input-number v-model.number="ef.hp" :min="0" size="mini" controls-position="right" style="width:96px" />
              移动距离 <el-input-number v-model.number="ef.move" :min="0" size="mini" controls-position="right" style="width:96px" />
              暴击几率 <el-input-number v-model.number="ef.crit" :min="0" size="mini" controls-position="right" style="width:96px" />
              暴击伤害 <el-input-number v-model.number="ef.crit_dmg" :min="0" size="mini" controls-position="right" style="width:96px" />
            </div>
          </el-form-item>
          <el-form-item label="说明">
            <el-input v-model="ef.des" maxlength="200" />
          </el-form-item>
        </el-form>
        <!-- [说明·不显示在界面] 战斗属性单位是百分点：填 125 = +125%。直接进战斗。
            价格 / 库存 / 所属套装在「数据管理」维护，这里不重复。 -->
        <div slot="footer">
          <el-button @click="eDlg = false">取 消</el-button>
          <el-button type="primary" :loading="saving" @click="doEquipSave">保 存</el-button>
        </div>
      </el-dialog>

      <!-- ============ 新增/编辑 装备套装 ============ -->
      <el-dialog :title="stf.id ? ('编辑套装 · ' + stf.name) : '新增套装'" :visible.sync="stDlg"
                 width="720px" :close-on-click-modal="false">
        <el-form label-width="110px" size="small">
          <el-row :gutter="10">
            <el-col :span="14">
              <el-form-item label="套装名称" required>
                <el-input v-model="stf.name" maxlength="100" />
              </el-form-item>
            </el-col>
            <el-col :span="10">
              <el-form-item label="触发件数">
                <el-input-number v-model.number="stf.parts" :min="1" :max="20" controls-position="right" style="width:100%" />
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="10">
            <el-col :span="8"><el-form-item label="军事加成"><el-input-number v-model.number="stf.military" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
            <el-col :span="8"><el-form-item label="后勤加成"><el-input-number v-model.number="stf.logistics" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
            <el-col :span="8"><el-form-item label="学识加成"><el-input-number v-model.number="stf.learning" :min="0" controls-position="right" style="width:100%" /></el-form-item></el-col>
          </el-row>
          <el-form-item label="额外战斗属性(%)">
            <div class="attr-row">
              伤害 <el-input-number v-model.number="stf.dmg" :min="0" size="mini" controls-position="right" style="width:96px" />
              防御 <el-input-number v-model.number="stf.def" :min="0" size="mini" controls-position="right" style="width:96px" />
              生命 <el-input-number v-model.number="stf.hp" :min="0" size="mini" controls-position="right" style="width:96px" />
              移动距离 <el-input-number v-model.number="stf.move" :min="0" size="mini" controls-position="right" style="width:96px" />
              暴击几率 <el-input-number v-model.number="stf.crit" :min="0" size="mini" controls-position="right" style="width:96px" />
              暴击伤害 <el-input-number v-model.number="stf.crit_dmg" :min="0" size="mini" controls-position="right" style="width:96px" />
            </div>
          </el-form-item>
          <el-form-item label="系列">
            <el-input v-model="stf.series" maxlength="30" placeholder="革命者 / 渡鸦之魂 …（可空）" />
          </el-form-item>
          <el-form-item label="套装效果">
            <el-input v-model="stf.effect" maxlength="300" placeholder="例如：9件：攻击+2984，防御+2890" />
          </el-form-item>
          <el-form-item label="说明">
            <el-input v-model="stf.des" maxlength="300" />
          </el-form-item>
        </el-form>
        <!-- [说明·不显示在界面] 穿戴同套 <b>{{ stf.parts || 3 }}</b> 件后，三维加成会直接叠加到军官属性上。 -->
        <div slot="footer">
          <el-button @click="stDlg = false">取 消</el-button>
          <el-button type="primary" :loading="saving" @click="doSetSave">保 存</el-button>
        </div>
      </el-dialog>

      <!-- ============ 套装件列表（查看 + 编辑属性） ============ -->
      <el-dialog :title="'套装件 · ' + setPiecesTitle" :visible.sync="stPiecesDlg" width="860px">
        <el-table :data="setPieces" size="mini" border stripe max-height="460">
          <el-table-column prop="id" label="ID" width="60" align="center" />
          <el-table-column prop="name" label="装备名" min-width="170" show-overflow-tooltip />
          <el-table-column prop="slot" label="部位" width="80" align="center" />
          <el-table-column label="品质" width="70" align="center">
            <template slot-scope="{row}">
              <el-tag size="mini" :type="tierTag(row.tier)">{{ row.tier_name || tierName(row.tier) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="level" label="需求等级" width="80" align="center" />
          <el-table-column prop="military" label="军事" width="60" align="center" />
          <el-table-column prop="logistics" label="后勤" width="60" align="center" />
          <el-table-column prop="learning" label="学识" width="60" align="center" />
          <el-table-column label="战斗属性" min-width="190" show-overflow-tooltip>
            <template slot-scope="{row}">{{ equipAttrText(row) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="90" align="center" fixed="right">
            <template slot-scope="{row}">
              <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑属性" @click="openSetPieceEdit(row)" />
            </template>
          </el-table-column>
        </el-table>
        <div class="pager-info" style="margin-top:8px">
          共 <b>{{ setPieces.length }}</b> 件（少于触发件数就永远触发不了套装效果）；套装件商城售价 / 库存到「数据管理 → 装备道具配置」维护
        </div>
      </el-dialog>
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

// ★ 属性白名单：价格 / 库存 / 所属套装（set_id）都不在这里 —— 那些字段由「数据管理」维护
const E_ATTR_KEYS = ['name', 'type', 'tier', 'military', 'logistics', 'learning', 'level', 'des',
  'slot', 'effect', 'series', 'enhance', 'enhance_max', 'dmg', 'def', 'hp', 'move', 'crit', 'crit_dmg']
const ST_KEYS = ['name', 'parts', 'military', 'logistics', 'learning', 'effect', 'des',
  'series', 'dmg', 'def', 'hp', 'move', 'crit', 'crit_dmg']

export default {
  name: 'AdminEzfyEquips',
  data () {
    return {
      tab: 'loose',
      saving: false,
      // 1 散件装备
      loose: [], loadingE: false, eWord: '', ePage: 1, eSize: 10,
      eDlg: false, ef: {}, efSetId: 0,
      // 2 套装管理
      sets: [], loadingSt: false, stWord: '',
      stDlg: false, stf: {},
      stPiecesDlg: false, setPieces: [], setPiecesTitle: ''
    }
  },
  computed: {
    ePaged () {
      const st = (this.ePage - 1) * this.eSize
      return this.loose.slice(st, st + this.eSize)
    },
    slotOptions () {
      return ['武器', '防具', '饰品', '珠宝', '头盔', '护肩', '胸甲', '腰带', '手套', '战靴', '挂件', '勋章', '左槽', '右槽']
    }
  },
  watch: {
    eWord () { this.ePage = 1 }
  },
  mounted () { this.loadEquips(); this.loadSets() },
  methods: {
    onTab (tab) {
      this.tab = tab.name
      if (this.tab === 'loose') this.loadEquips()
      if (this.tab === 'sets') this.loadSets()
    },
    tierName (t) {
      return ({ 1: '初级', 2: '中级', 3: '高级', 4: '特殊' })[t] || '初级'
    },
    tierTag (t) {
      return ({ 1: 'info', 2: 'primary', 3: 'warning', 4: 'danger' })[t] || 'info'
    },
    // ★ 装备六项战斗属性 + 三维属性的展示文案
    equipAttrText (e) {
      if (!e) return ''
      const parts = []
      if (e.dmg) parts.push('伤害+' + e.dmg + '%')
      if (e.def) parts.push('防御+' + e.def + '%')
      if (e.hp) parts.push('生命+' + e.hp + '%')
      if (e.move) parts.push('移动距离+' + e.move + '%')
      if (e.crit) parts.push('暴击几率+' + e.crit + '%')
      if (e.crit_dmg) parts.push('暴击伤害+' + e.crit_dmg + '%')
      if (e.military) parts.push('军事+' + e.military)
      if (e.logistics) parts.push('后勤+' + e.logistics)
      if (e.learning) parts.push('学识+' + e.learning)
      return parts.length ? parts.join(' ') : '—'
    },
    // ---------- 1 散件装备 ----------
    loadEquips () {
      this.loadingE = true
      // ★ 只看散件（set_id=0 要显式传，不传 = 全部）
      api.get('/admin/ezfy-equipments', { params: { word: this.eWord, set_id: 0 } }).then(r => {
        this.loadingE = false
        if (r.code === 0) this.loose = r.data.list
        else this.$message.error(r.msg)
      })
    },
    openEquipCreate () {
      this.ef = {
        name: '', type: '武器', tier: 1, military: 0, logistics: 0, learning: 0, level: 1, des: '',
        slot: '', effect: '', series: '', enhance: 0, enhance_max: 20,
        dmg: 0, def: 0, hp: 0, move: 0, crit: 0, crit_dmg: 0
      }
      this.efSetId = 0
      this.eDlg = true
    },
    openEquipEdit (row) {
      const f = { id: row.id }
      E_ATTR_KEYS.forEach(k => { f[k] = row[k] })
      this.ef = f
      this.efSetId = row.set_id || 0
      this.eDlg = true
    },
    // 套装件编辑：复用装备属性对话框（type 锁定「套装」，set_id 不回写）
    openSetPieceEdit (row) {
      const f = { id: row.id }
      E_ATTR_KEYS.forEach(k => { f[k] = row[k] })
      if (!f.type) f.type = '套装'
      this.ef = f
      this.efSetId = row.set_id || 1
      this.eDlg = true
    },
    doEquipSave () {
      if (!String(this.ef.name || '').trim()) { this.$message.warning('装备名称不能为空'); return }
      const isNew = !this.ef.id
      const body = {}
      E_ATTR_KEYS.forEach(k => { body[k] = this.ef[k] })
      // ★ 新增默认归到散件（set_id=0），后端默认库存 -1 / 品质初级 / 需求等级 1
      if (isNew) body.set_id = this.efSetId || 0
      this.saving = true
      const req = isNew ? api.post('/admin/ezfy-equipments', body) : api.put('/admin/ezfy-equipments/' + this.ef.id, body)
      req.then(r => {
        this.saving = false
        if (r.code === 0) {
          this.eDlg = false
          this.$message.success(r.data.msg || '已保存')
          this.loadEquips()
          if (this.stPiecesDlg) this.openSetPieces({ name: this.setPiecesTitle, id: (this.ef.set_id || this.efSetId) })
        } else this.$message.error(r.msg)
      })
    },
    delEquip (row) {
      this.$confirm('删除装备配置「' + row.name + '」会同时清掉玩家背包里的 ' + row.owned_count +
        ' 件。确认删除？', '危险操作', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-equipments/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadEquips() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    // ---------- 2 套装管理 ----------
    loadSets () {
      this.loadingSt = true
      api.get('/admin/ezfy-equip-sets', { params: { word: this.stWord } }).then(r => {
        this.loadingSt = false
        if (r.code === 0) this.sets = r.data.list
        else this.$message.error(r.msg)
      })
    },
    openSetCreate () {
      this.stf = {
        name: '', parts: 3, military: 0, logistics: 0, learning: 0, effect: '', des: '',
        series: '', dmg: 0, def: 0, hp: 0, move: 0, crit: 0, crit_dmg: 0
      }
      this.stDlg = true
    },
    openSetEdit (row) {
      const f = { id: row.id }
      ST_KEYS.forEach(k => { f[k] = row[k] })
      this.stf = f
      this.stDlg = true
    },
    doSetSave () {
      if (!String(this.stf.name || '').trim()) { this.$message.warning('套装名称不能为空'); return }
      const isNew = !this.stf.id
      const body = {}
      ST_KEYS.forEach(k => { body[k] = this.stf[k] })
      this.saving = true
      const req = isNew ? api.post('/admin/ezfy-equip-sets', body) : api.put('/admin/ezfy-equip-sets/' + this.stf.id, body)
      req.then(r => {
        this.saving = false
        if (r.code === 0) { this.stDlg = false; this.$message.success(r.data.msg || '已保存'); this.loadSets() }
        else this.$message.error(r.msg)
      })
    },
    delSet (row) {
      this.$confirm('删除套装「' + row.name + '」？该套装下的装备会解除归属（装备本身保留）。',
        '危险操作', { type: 'warning' }).then(() => {
        api.delete('/admin/ezfy-equip-sets/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.loadSets() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    openSetPieces (row) {
      this.setPiecesTitle = row.name
      api.get('/admin/ezfy-equip-sets/' + row.id + '/pieces').then(r => {
        if (r.code === 0) { this.setPieces = r.data.list; this.stPiecesDlg = true } else this.$message.error(r.msg)
      })
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.td-danger { color: #f56c6c; font-weight: 600; }
</style>
