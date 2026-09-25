<template>
  <div class="farm-admin">
    <el-card shadow="never" class="box">
      <div slot="header" class="card-head">
        <!-- ★ 2026-09-25 用户要求：新增「军团商城维护」页面（军团积分兑换商店） -->
        <span>二战风云 · 军团商城维护</span>
        <div>
          <el-button size="mini" type="success" icon="el-icon-plus" @click="openCreate">新增商品</el-button>
          <el-button size="mini" type="primary" plain icon="el-icon-refresh" @click="load">刷新</el-button>
        </div>
      </div>

      <div class="toolbar">
        <el-select v-model="kind" style="width:140px" @change="page = 1; load()">
          <el-option :value="0" label="全部类型" />
          <el-option :value="1" label="资源包" />
          <el-option :value="2" label="道具" />
        </el-select>
        <el-input v-model="keyword" placeholder="按商品名搜索" clearable style="width:220px"
                  @keyup.enter.native="page = 1; load()" />
        <el-button type="primary" icon="el-icon-search" @click="page = 1; load()">查询</el-button>
      </div>

      <el-table :data="list" v-loading="loading" stripe border>
        <el-table-column prop="id" label="ID" width="70" align="center" />
        <el-table-column label="类型" width="90" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.kind === 2 ? 'warning' : 'success'">
              {{ row.kind_name || kindText(row.kind) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="名称" min-width="150" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column label="内容" min-width="220" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-sub">{{ contentText(row) }}</span></template>
        </el-table-column>
        <el-table-column label="价格" width="120" align="right">
          <template slot-scope="{row}"><span class="td-gold">{{ fmtN(row.price) }}</span></template>
        </el-table-column>
        <el-table-column label="每人限购" width="100" align="center">
          <template slot-scope="{row}">
            <span v-if="!row.limit" class="td-muted">不限</span>
            <span v-else class="td-mono">{{ row.limit }}</span>
          </template>
        </el-table-column>
        <el-table-column label="库存 / 已售" width="130" align="center">
          <template slot-scope="{row}">
            <span v-if="row.stock < 0" class="td-green">∞ 不限</span>
            <span v-else class="td-mono">{{ row.stock }}</span>
            <span class="td-sub"> / {{ fmtN(row.sold) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="80" align="center" />
        <el-table-column label="状态" width="90" align="center">
          <template slot-scope="{row}">
            <el-tag size="mini" :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '已上架' : '已下架' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain icon="el-icon-edit" title="编辑" @click="openEdit(row)" />
            <el-button size="mini" :type="row.enabled ? 'warning' : 'success'" plain
                       :icon="row.enabled ? 'el-icon-bottom' : 'el-icon-top'"
                       :title="row.enabled ? '下架' : '上架'" @click="toggle(row)" />
            <el-button size="mini" type="danger" plain icon="el-icon-delete" title="删除" @click="remove(row)" />
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

    <!-- 新增 / 编辑商品 -->
    <el-dialog :title="form.id ? ('编辑商品 #' + form.id) : '新增商品'" :visible.sync="dlg" width="620px" :close-on-click-modal="false">
      <el-form label-width="140px" size="small">
        <el-form-item label="类型" required>
          <el-radio-group v-model="form.kind">
            <el-radio :label="1">资源包</el-radio>
            <el-radio :label="2">道具</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="form.name" maxlength="50" placeholder="商品显示名" style="width:100%" />
        </el-form-item>

        <!-- 资源包：各资源数量 -->
        <template v-if="form.kind === 1">
          <el-form-item label="粮食">
            <el-input-number v-model.number="form.food" :min="0" :step="1000" controls-position="right" style="width:220px" />
          </el-form-item>
          <el-form-item label="钢铁">
            <el-input-number v-model.number="form.steel" :min="0" :step="1000" controls-position="right" style="width:220px" />
          </el-form-item>
          <el-form-item label="石油">
            <el-input-number v-model.number="form.oil" :min="0" :step="1000" controls-position="right" style="width:220px" />
          </el-form-item>
          <el-form-item label="稀矿">
            <el-input-number v-model.number="form.rare" :min="0" :step="1000" controls-position="right" style="width:220px" />
          </el-form-item>
          <el-form-item label="黄金">
            <el-input-number v-model.number="form.gold" :min="0" :step="1000" controls-position="right" style="width:220px" />
            <div class="td-sub" style="margin-top:4px">资源包至少有一项数量大于 0</div>
          </el-form-item>
        </template>

        <!-- 道具：从游戏道具池选择 -->
        <template v-else>
          <el-form-item label="道具" required>
            <el-select v-model="form.item_id" filterable placeholder="从道具池选择" style="width:100%">
              <el-option v-for="it in itemsPool" :key="'it' + it.id"
                         :label="(it.name || '道具') + (it.category_name ? '（' + it.category_name + '）' : '')"
                         :value="it.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="数量" required>
            <el-input-number v-model.number="form.item_count" :min="1" controls-position="right" style="width:220px" />
          </el-form-item>
        </template>

        <el-form-item label="价格（军团积分）" required>
          <el-input-number v-model.number="form.price" :min="0" controls-position="right" style="width:220px" />
        </el-form-item>
        <el-form-item label="每人限购">
          <el-input-number v-model.number="form.limit" :min="0" controls-position="right" style="width:220px" />
          <div class="td-sub" style="margin-top:4px">0 表示不限购</div>
        </el-form-item>
        <el-form-item label="库存">
          <el-input-number v-model.number="form.stock" :min="-1" controls-position="right" style="width:220px" />
          <div class="td-sub" style="margin-top:4px">-1 表示无限库存</div>
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model.number="form.sort" :min="0" controls-position="right" style="width:220px" />
          <div class="td-sub" style="margin-top:4px">数字越小越靠前</div>
        </el-form-item>
        <el-form-item label="是否上架">
          <el-switch v-model="form.enabled" :active-value="1" :inactive-value="0" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="dlg = false">取 消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保 存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

// ★ 2026-09-25 用户要求：资源包字段名 → 中文名（与交易行维护页 RES_FALLBACK 保持一致）
const RES_KEYS = [
  { key: 'food', label: '粮食' },
  { key: 'steel', label: '钢铁' },
  { key: 'oil', label: '石油' },
  { key: 'rare', label: '稀矿' },
  { key: 'gold', label: '黄金' }
]

export default {
  name: 'AdminEzfyCorpsMall',
  data () {
    return {
      list: [], total: 0, page: 1, size: 10, keyword: '', kind: 0,
      loading: false, saving: false,
      dlg: false,
      form: this.blankForm(),
      itemsPool: []
    }
  },
  computed: {
    itemNames () {
      const m = {}
      this.itemsPool.forEach(it => { m[it.id] = it.name })
      return m
    }
  },
  mounted () { this.load(); this.loadItems() },
  methods: {
    blankForm () {
      return {
        id: 0, kind: 1, name: '',
        food: 0, steel: 0, oil: 0, rare: 0, gold: 0,
        item_id: null, item_count: 1,
        price: 0, limit: 0, stock: -1, sort: 0, enabled: 1
      }
    },
    kindText (k) { return k === 2 ? '道具' : '资源包' },
    fmtN (v) {
      if (v === null || v === undefined || v === '') return '0'
      return Number(v).toLocaleString()
    },
    // 内容列：资源包显示各资源数量；道具显示「道具名×数量」
    contentText (row) {
      if (row.kind === 2) {
        const name = this.itemNames[row.item_id] || ('道具#' + row.item_id)
        return name + ' × ' + (row.item_count || 0)
      }
      const parts = RES_KEYS.filter(r => Number(row[r.key]) > 0).map(r => r.label + ' ' + this.fmtN(row[r.key]))
      return parts.length ? parts.join(' / ') : '—'
    },
    loadItems () {
      // 道具池：兼容 {list:[...]} 与 {items:[...]} 两种返回结构
      api.get('/admin/ezfy-corps-mall/items').then(r => {
        if (r.code === 0 && r.data) this.itemsPool = r.data.list || r.data.items || []
      })
    },
    load () {
      this.loading = true
      api.get('/admin/ezfy-corps-mall', {
        params: { page: this.page, size: this.size, kind: this.kind, keyword: this.keyword }
      }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list || []
          this.total = r.data.total || 0
          if (r.data.page) this.page = r.data.page
        } else this.$message.error(r.msg)
      }).catch(() => { this.loading = false })
    },
    openCreate () {
      this.form = this.blankForm()
      this.dlg = true
    },
    openEdit (row) {
      this.form = {
        id: row.id, kind: row.kind, name: row.name,
        food: row.food || 0, steel: row.steel || 0, oil: row.oil || 0, rare: row.rare || 0, gold: row.gold || 0,
        item_id: row.item_id || null, item_count: row.item_count || 1,
        price: row.price || 0, limit: row.limit || 0,
        stock: (row.stock === null || row.stock === undefined) ? -1 : row.stock,
        sort: row.sort || 0, enabled: row.enabled ? 1 : 0
      }
      this.dlg = true
    },
    save () {
      if (!this.form.name || !String(this.form.name).trim()) { this.$message.warning('请填写商品名称'); return }
      if (!(this.form.price >= 0)) { this.$message.warning('价格不能小于 0'); return }
      if (this.form.kind === 1) {
        const sum = Number(this.form.food) + Number(this.form.steel) + Number(this.form.oil) +
          Number(this.form.rare) + Number(this.form.gold)
        if (!(sum > 0)) { this.$message.warning('资源包至少有一项数量大于 0'); return }
      } else {
        if (!this.form.item_id) { this.$message.warning('请选择道具'); return }
        if (!(this.form.item_count > 0)) { this.$message.warning('道具数量必须大于 0'); return }
      }
      const body = {
        kind: this.form.kind,
        name: String(this.form.name).trim(),
        food: Number(this.form.food) || 0,
        steel: Number(this.form.steel) || 0,
        oil: Number(this.form.oil) || 0,
        rare: Number(this.form.rare) || 0,
        gold: Number(this.form.gold) || 0,
        item_id: Number(this.form.item_id) || 0,
        item_count: Number(this.form.item_count) || 0,
        price: Number(this.form.price) || 0,
        limit: Number(this.form.limit) || 0,
        stock: (this.form.stock === null || this.form.stock === undefined || this.form.stock === '') ? -1 : Number(this.form.stock),
        sort: Number(this.form.sort) || 0,
        enabled: this.form.enabled ? 1 : 0
      }
      // 带 id 即更新，不带即新增（契约要求走同一个 POST 接口）
      if (this.form.id) body.id = this.form.id
      this.saving = true
      api.post('/admin/ezfy-corps-mall', body).then(r => {
        this.saving = false
        if (r.code === 0) { this.dlg = false; this.$message.success(r.data.msg || '已保存'); this.load() }
        else this.$message.error(r.msg)
      }).catch(() => { this.saving = false })
    },
    toggle (row) {
      const act = row.enabled ? '下架' : '上架'
      this.$confirm('确定' + act + '商品「' + row.name + '」？', '提示', { type: 'warning' }).then(() => {
        api.post('/admin/ezfy-corps-mall/' + row.id + '/toggle', {}).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || ('已' + act)); this.load() }
          else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    remove (row) {
      this.$confirm('删除商品「' + row.name + '」？删除后不可恢复。', '危险操作', { type: 'error' }).then(() => {
        api.delete('/admin/ezfy-corps-mall/' + row.id).then(r => {
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