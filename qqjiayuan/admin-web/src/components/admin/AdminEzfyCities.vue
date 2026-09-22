<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model="word" placeholder="城名 / 玩家昵称 / 用户ID" clearable style="width:240px"
                  @keyup.enter.native="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
        <div class="grow" />
        <el-button type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe border max-height="620">
        <el-table-column prop="id" label="城池ID" width="60" align="center" />
        <el-table-column label="城名" min-width="90" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column label="坐标 / 地形" width="90" align="center">
          <template slot-scope="{row}">
            <div class="td-mono">{{ row.x }},{{ row.y }}</div>
            <div class="td-sub">{{ row.terrain_name }}</div>
          </template>
        </el-table-column>
        <el-table-column label="归属玩家（家园号）" width="120" show-overflow-tooltip>
          <template slot-scope="{row}">
            <span class="td-main">{{ row.player_name || '—' }}</span>
            <span class="td-muted">（{{ row.home_num || '—' }}）</span>
          </template>
        </el-table-column>
        <!-- ★ 用户要求：新增「所在州」，去掉「阵营」列 -->
        <el-table-column label="所在州" width="90" align="center" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-sub">{{ row.continent || '—' }}</span></template>
        </el-table-column>
        <el-table-column prop="city_level" label="市政厅" width="60" align="center" />
        <el-table-column prop="pop" label="人口" width="60" align="center" />
        <!-- ★ 用户要求：资源、建筑/部队 不在列表里罗列，改为点击弹模态框查看 -->
        <el-table-column label="资源" width="80" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="text" @click="openRes(row)">[查看]</el-button>
          </template>
        </el-table-column>
        <el-table-column label="建筑 / 部队" width="110" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="text" @click="openBT(row)">[查看]</el-button>
          </template>
        </el-table-column>
        <el-table-column label="军官/野地" width="76" align="center">
          <template slot-scope="{row}">
            <span class="td-mono">{{ row.officer_num }}/{{ row.wild_num }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="168" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="info" plain icon="el-icon-view" title="详情" @click="openDetail(row)" />
            <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openEdit(row)" />
            <el-button size="mini" type="warning" plain icon="el-icon-refresh-left" title="重置城池" @click="reset(row)" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete" title="拆除" @click="del(row)" />
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

    <!-- 资源明细（点击列表「资源 → 查看」展开） -->
    <el-dialog :title="'资源明细 · ' + (resRow ? resRow.name : '')" :visible.sync="resDlg" width="520px">
      <el-descriptions v-if="resRow" :column="2" size="medium" border>
        <el-descriptions-item label="黄金">{{ fmtN(resRow.gold) }}</el-descriptions-item>
        <el-descriptions-item label="黄金上限">{{ fmtN(resRow.gold_cap) }}</el-descriptions-item>
        <el-descriptions-item label="粮食">{{ fmtN(resRow.food) }}</el-descriptions-item>
        <el-descriptions-item label="粮食上限">{{ fmtN(resRow.food_cap) }}</el-descriptions-item>
        <el-descriptions-item label="钢铁">{{ fmtN(resRow.steel) }}</el-descriptions-item>
        <el-descriptions-item label="钢铁上限">{{ fmtN(resRow.steel_cap) }}</el-descriptions-item>
        <el-descriptions-item label="石油">{{ fmtN(resRow.oil) }}</el-descriptions-item>
        <el-descriptions-item label="石油上限">{{ fmtN(resRow.oil_cap) }}</el-descriptions-item>
        <el-descriptions-item label="稀矿">{{ fmtN(resRow.rare) }}</el-descriptions-item>
        <el-descriptions-item label="稀矿上限">{{ fmtN(resRow.rare_cap) }}</el-descriptions-item>
      </el-descriptions>
      <div slot="footer"><el-button @click="resDlg = false">关 闭</el-button></div>
    </el-dialog>

    <!-- 建筑 / 部队（点击列表「建筑/部队 → 查看」展开） -->
    <el-dialog :title="'建筑 / 部队 · ' + (btRow ? btRow.name : '')" :visible.sync="btDlg" width="720px" v-loading="btLoading">
      <template v-if="btDetail">
        <div class="sub-title">建筑（{{ btDetail.buildings.length }}）</div>
        <el-table :data="btDetail.buildings" size="mini" border max-height="240">
          <el-table-column prop="id" label="ID" width="70" align="center" />
          <el-table-column prop="cfg_name" label="建筑" min-width="120" />
          <el-table-column prop="level" label="等级" width="70" align="center" />
          <el-table-column label="状态" width="90" align="center">
            <template slot-scope="{row}">{{ buildStatus[row.status] }}</template>
          </el-table-column>
        </el-table>
        <div class="sub-title">部队（{{ btDetail.troops.length }}）</div>
        <el-table :data="btDetail.troops" size="mini" border max-height="240">
          <el-table-column prop="id" label="ID" width="70" align="center" />
          <el-table-column prop="cfg_name" label="兵种" min-width="120" />
          <el-table-column prop="count" label="数量" width="120" align="center" />
        </el-table>
      </template>
      <div slot="footer"><el-button @click="btDlg = false">关 闭</el-button></div>
    </el-dialog>

    <!-- 详情 -->
    <el-dialog title="城池详情" :visible.sync="detailDlg" width="860px" :close-on-click-modal="false">
      <template v-if="detail">
        <el-descriptions :column="3" size="medium" border>
          <el-descriptions-item label="城池ID">{{ detail.city.id }}</el-descriptions-item>
          <el-descriptions-item label="城名">{{ detail.city.name }}</el-descriptions-item>
          <el-descriptions-item label="坐标">{{ detail.city.x }},{{ detail.city.y }}（{{ detail.terrain_name }}）</el-descriptions-item>
          <el-descriptions-item label="归属玩家">{{ detail.player_name || '—' }}</el-descriptions-item>
          <el-descriptions-item label="家园号">{{ detail.home_num || '—' }}</el-descriptions-item>
          <el-descriptions-item label="市政厅等级">Lv.{{ detail.city.city_level }}</el-descriptions-item>
          <el-descriptions-item label="民心/民怨">{{ detail.city.feelings }} / {{ detail.city.grievance }}</el-descriptions-item>
          <el-descriptions-item label="税率">{{ detail.city.tax_rate }}%</el-descriptions-item>
          <el-descriptions-item label="人口">{{ detail.city.pop }} / {{ detail.city.pop_max }}</el-descriptions-item>
        </el-descriptions>

        <div class="sub-title">建筑（{{ detail.buildings.length }}）</div>
        <el-table :data="detail.buildings" size="mini" border max-height="200">
          <el-table-column prop="id" label="ID" width="70" align="center" />
          <el-table-column prop="cfg_name" label="建筑" min-width="110" />
          <el-table-column prop="level" label="等级" width="70" align="center" />
          <el-table-column label="状态" width="90" align="center">
            <template slot-scope="{row}">{{ buildStatus[row.status] }}</template>
          </el-table-column>
        </el-table>

        <div class="sub-title">部队（{{ detail.troops.length }}）</div>
        <el-table :data="detail.troops" size="mini" border max-height="200">
          <el-table-column prop="id" label="ID" width="70" align="center" />
          <el-table-column prop="cfg_name" label="兵种" min-width="110" />
          <el-table-column prop="count" label="数量" width="100" align="center" />
        </el-table>

        <div class="sub-title">科技（{{ detail.techs.length }}）</div>
        <el-table :data="detail.techs" size="mini" border max-height="200">
          <el-table-column prop="id" label="ID" width="70" align="center" />
          <el-table-column prop="cfg_name" label="科技" min-width="110" />
          <el-table-column prop="level" label="等级" width="70" align="center" />
          <el-table-column label="状态" width="90" align="center">
            <template slot-scope="{row}">{{ row.status === 1 ? '研究中' : '已掌握' }}</template>
          </el-table-column>
        </el-table>

        <div class="sub-title">军官（{{ detail.officers.length }}）</div>
        <el-table :data="detail.officers" size="mini" border max-height="200">
          <el-table-column prop="id" label="ID" width="70" align="center" />
          <el-table-column prop="name" label="姓名" min-width="100" />
          <el-table-column prop="level" label="等级" width="70" align="center" />
          <el-table-column prop="star" label="星级" width="70" align="center" />
          <el-table-column prop="loyalty" label="忠诚" width="70" align="center" />
        </el-table>

        <div class="sub-title">已占野地（{{ detail.wildlands.length }}）</div>
        <el-table :data="detail.wildlands" size="mini" border max-height="180">
          <el-table-column prop="id" label="ID" width="70" align="center" />
          <el-table-column label="坐标" width="100" align="center">
            <template slot-scope="{row}">{{ row.x }},{{ row.y }}</template>
          </el-table-column>
          <el-table-column prop="level" label="等级" width="70" align="center" />
          <el-table-column label="状态" width="90" align="center">
            <template slot-scope="{row}">{{ row.status === 1 ? '采集中' : '空闲' }}</template>
          </el-table-column>
        </el-table>

        <div class="sub-title">最近出征（{{ detail.orders.length }}）</div>
        <el-table :data="detail.orders" size="mini" border max-height="180">
          <el-table-column prop="id" label="ID" width="70" align="center" />
          <el-table-column label="类型" width="80" align="center">
            <template slot-scope="{row}">{{ orderTypes[row.order_type] || row.order_type }}</template>
          </el-table-column>
          <el-table-column label="目标" width="100" align="center">
            <template slot-scope="{row}">{{ row.target_x }},{{ row.target_y }}</template>
          </el-table-column>
          <el-table-column label="状态" width="90" align="center">
            <template slot-scope="{row}">{{ orderStatus[row.status] || row.status }}</template>
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
    <el-dialog title="编辑城池" :visible.sync="editDlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="110px" size="small">
        <el-form-item label="城名"><el-input v-model="form.name" maxlength="50" style="width:220px" /></el-form-item>
        <el-form-item label="坐标 X / Y">
          <el-input-number v-model.number="form.x" :min="0" controls-position="right" style="width:120px" />
          <el-input-number v-model.number="form.y" :min="0" controls-position="right" style="width:120px;margin-left:8px" />
        </el-form-item>
        <el-form-item label="市政厅等级"><el-input-number v-model.number="form.city_level" :min="1" controls-position="right" /></el-form-item>
        <el-form-item label="民心 / 民怨">
          <el-input-number v-model.number="form.feelings" :min="0" :max="100" controls-position="right" style="width:120px" />
          <el-input-number v-model.number="form.grievance" :min="0" controls-position="right" style="width:120px;margin-left:8px" />
        </el-form-item>
        <el-form-item label="税率 %"><el-input-number v-model.number="form.tax_rate" :min="0" :max="100" controls-position="right" /></el-form-item>
        <el-form-item label="人口 / 上限">
          <el-input-number v-model.number="form.pop" :min="0" controls-position="right" style="width:140px" />
          <el-input-number v-model.number="form.pop_max" :min="0" controls-position="right" style="width:140px;margin-left:8px" />
        </el-form-item>
        <el-divider content-position="left">资源</el-divider>
        <el-form-item label="黄金"><el-input-number v-model.number="form.gold" :min="0" controls-position="right" style="width:180px" /></el-form-item>
        <el-form-item label="粮食"><el-input-number v-model.number="form.food" :min="0" controls-position="right" style="width:180px" /></el-form-item>
        <el-form-item label="钢铁"><el-input-number v-model.number="form.steel" :min="0" controls-position="right" style="width:180px" /></el-form-item>
        <el-form-item label="石油"><el-input-number v-model.number="form.oil" :min="0" controls-position="right" style="width:180px" /></el-form-item>
        <el-form-item label="稀矿"><el-input-number v-model.number="form.rare" :min="0" controls-position="right" style="width:180px" /></el-form-item>
      </el-form>
      <!-- [说明·不显示在界面] 提示：资源直接写入，不受仓储上限限制（GM 专用） -->
      <div slot="footer">
        <el-button @click="editDlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminEzfyCities',
  data () {
    return {
      list: [], total: 0, page: 1, size: 5, loading: false, word: '',
      detailDlg: false, detail: null,
      // ★ 资源 / 建筑·部队 的点击展开模态框
      resDlg: false, resRow: null,
      btDlg: false, btRow: null, btDetail: null, btLoading: false,
      editDlg: false, saving: false, editId: 0, form: {},
      buildStatus: { 0: '空闲', 1: '建造中', 2: '升级中' },
      orderTypes: { 1: '侦查', 2: '掠夺', 3: '征服', 4: '采集', 5: '运输', 6: '增援', 7: '派遣' },
      orderStatus: { 0: '行进中', 1: '驻守中', 2: '返回中', 3: '已完成', 4: '已阵亡' }
    }
  },
  mounted () { this.load() },
  methods: {
    fmtTime (t) { return t ? new Date(t).toLocaleString() : '' },
    // 紧凑大数：不带宽分隔符，保留 1 位小数（表格里用，省宽度）
    fmtCompact (v) {
      if (v === null || v === undefined || v === '') return '—'
      const n = Number(v)
      if (!isFinite(n)) return '—'
      const abs = Math.abs(n)
      const sign = n < 0 ? '-' : ''
      const cut = s => s.replace(/\.0$/, '')
      if (abs < 10000) return sign + String(Math.round(abs))
      if (abs < 1e8) return sign + cut((abs / 1e4).toFixed(1)) + '万'
      if (abs < 1e12) return sign + cut((abs / 1e8).toFixed(1)) + '亿'
      return sign + cut((abs / 1e12).toFixed(1)) + '万亿'
    },
    // 大数加单位（万/亿/万亿），避免一串 10 位数字撑破列宽
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
    fmtN (v) {
      if (v === null || v === undefined) return '—'
      return Number(v).toLocaleString()
    },
    load () {
      this.loading = true
      api.get('/admin/ezfy-cities', { params: { page: this.page, size: this.size, word: this.word } }).then(r => {
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
      api.get('/admin/ezfy-cities/' + row.id).then(r => {
        if (r.code === 0) this.detail = r.data
        else { this.detailDlg = false; this.$message.error(r.msg) }
      })
    },
    // ★ 资源：直接弹框展示（数据已在列表行里）
    openRes (row) {
      this.resRow = row
      this.resDlg = true
    },
    // ★ 建筑 / 部队：点击后才拉详情并弹框，避免列表里罗列一长串
    openBT (row) {
      this.btRow = row
      this.btDetail = null
      this.btDlg = true
      this.btLoading = true
      api.get('/admin/ezfy-cities/' + row.id).then(r => {
        this.btLoading = false
        if (r.code === 0) this.btDetail = r.data
        else { this.btDlg = false; this.$message.error(r.msg) }
      })
    },
    openEdit (row) {
      this.editId = row.id
      const keys = ['name', 'x', 'y', 'city_level', 'feelings', 'grievance', 'tax_rate',
        'pop', 'pop_max', 'gold', 'food', 'steel', 'oil', 'rare']
      const f = {}
      keys.forEach(k => { f[k] = row[k] })
      this.form = f
      this.editDlg = true
    },
    save () {
      this.saving = true
      api.put('/admin/ezfy-cities/' + this.editId, this.form).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.editDlg = false
          this.$message.success(r.data.msg || '已保存')
          this.load()
        } else this.$message.error(r.msg)
      })
    },
    reset (row) {
      this.$confirm('重置将清空该城池的建筑/部队/科技/军官/野地，只保留 1 级市政厅，不可恢复！', '危险操作', { type: 'error' }).then(() => {
        api.post('/admin/ezfy-cities/' + row.id + '/reset').then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已重置'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    del (row) {
      this.$confirm('拆除城池「' + row.name + '」及其全部附属数据，不可恢复！', '危险操作', { type: 'error' }).then(() => {
        api.delete('/admin/ezfy-cities/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已拆除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
.sub-title { font-size: 13px; font-weight: 600; color: #1f2d3d; margin: 14px 0 8px; padding-left: 6px; border-left: 3px solid #409eff; }
</style>
