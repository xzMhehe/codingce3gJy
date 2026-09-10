<template>
  <div class="farm-admin">
    <!-- 统计卡片 -->
    <div class="stat-row">
      <div class="stat-card s-green">
        <div class="stat-ico el-icon-truck" />
        <div class="stat-info">
          <div class="stat-num">{{ list.length }}</div>
          <div class="stat-lab">车辆总数</div>
        </div>
      </div>
      <div class="stat-card s-blue">
        <div class="stat-ico el-icon-sell" />
        <div class="stat-info">
          <div class="stat-num">{{ countDtype(1) }}</div>
          <div class="stat-lab">普通车</div>
        </div>
      </div>
      <div class="stat-card s-purple">
        <div class="stat-ico el-icon-magic-stick" />
        <div class="stat-info">
          <div class="stat-num">{{ countDtype(4) }}</div>
          <div class="stat-lab">贵族车</div>
        </div>
      </div>
      <div class="stat-card s-orange">
        <div class="stat-ico el-icon-coin" />
        <div class="stat-info">
          <div class="stat-num">{{ maxPrice() }}</div>
          <div class="stat-lab">最贵车辆(G币)</div>
        </div>
      </div>
    </div>

    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-select v-model="dtypeFilter" placeholder="分类" clearable style="width:140px" @change="page = 1">
          <el-option v-for="(n, d) in dtypeNames" :key="d" :label="n" :value="+d" />
        </el-select>
        <el-input v-model.trim="kw" prefix-icon="el-icon-search" placeholder="搜索车名" clearable style="width:230px" @input="page = 1" />
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增车辆</el-button>
      </div>

      <el-table :data="paged" v-loading="loading" stripe border size="medium">
        <el-table-column prop="id" label="ID" width="64" align="center" />
        <el-table-column label="车名" min-width="120">
          <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column label="分类" width="90" align="center">
          <template slot-scope="{row}">
            <el-tag :type="dtypeTag(row.dtype)" size="mini">{{ dtypeNames[row.dtype] || '未知' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="价格(G)" width="110" align="center">
          <template slot-scope="{row}"><i class="el-icon-coin td-blue"></i>{{ row.price }}</template>
        </el-table-column>
        <el-table-column label="盈利(G/时)" width="100" align="center">
          <template slot-scope="{row}"><span class="td-blue">{{ row.money }}</span></template>
        </el-table-column>
        <el-table-column label="12小时净赚" width="100" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ row.money * 12 * 4 / 5 }}</span></template>
        </el-table-column>
        <el-table-column label="回本(时)" width="85" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ row.money > 0 ? Math.ceil(row.price / (row.money * 4 / 5)) : '—' }}</span></template>
        </el-table-column>
        <el-table-column label="状态" width="80" align="center">
          <template slot-scope="{row}">
            <el-tag :type="row.status === 0 ? 'success' : 'info'" size="mini">{{ row.status === 0 ? '上架' : '下架' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="110" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" icon="el-icon-edit" circle title="编辑" @click="openDlg(row)" />
            <el-button size="mini" type="danger" icon="el-icon-delete" circle title="删除" @click="del(row)" />
          </template>
        </el-table-column>
      </el-table>

      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ filtered.length }}</b> 条 · 每页 {{ pageSize }} 条</div>
        <el-pagination small background layout="sizes, prev, pager, next, jumper"
          :total="filtered.length" :page-size.sync="pageSize" :current-page.sync="page"
          :page-sizes="[5, 10, 20, 50]" @size-change="page = 1" />
      </div>
    </el-card>

    <el-dialog :title="form.id ? '编辑车辆' : '新增车辆'" :visible.sync="dlg" width="520px" :close-on-click-modal="false">
      <el-form label-width="110px">
        <el-form-item label="车名">
          <el-input v-model.trim="form.name" maxlength="30" />
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model.number="form.dtype" style="width:160px">
            <el-option v-for="(n, d) in dtypeNames" :key="d" :label="n" :value="+d" />
          </el-select>
        </el-form-item>
        <el-form-item label="价格(G)">
          <el-input-number v-model.number="form.price" :min="0" />
        </el-form-item>
        <el-form-item label="盈利(G/时)">
          <el-input-number v-model.number="form.money" :min="0" />
          <span class="help-line" style="margin-left:10px">12小时净赚 {{ form.money * 12 * 4 / 5 }} G（含20%税）</span>
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model.number="form.status">
            <el-radio :label="0">上架</el-radio>
            <el-radio :label="1">下架</el-radio>
          </el-radio-group>
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

export default {
  name: 'AdminParkCars',
  data () {
    return {
      list: [], loading: false, dlg: false, saving: false,
      kw: '', dtypeFilter: null, page: 1, pageSize: 10,
      dtypeNames: { 1: '普通车', 2: '高级车', 3: '酷族车', 4: '贵族车', 5: '试驾车' },
      form: { id: 0, name: '', dtype: 1, price: 1000, money: 5, status: 0 }
    }
  },
  computed: {
    filtered () {
      const k = this.kw.trim().toLowerCase()
      return this.list.filter(x =>
        (!k || (x.name || '').toLowerCase().includes(k)) &&
        (!this.dtypeFilter || x.dtype === this.dtypeFilter))
    },
    paged () { return this.filtered.slice((this.page - 1) * this.pageSize, this.page * this.pageSize) }
  },
  mounted () { this.load() },
  methods: {
    countDtype (d) { return this.list.filter(x => x.dtype === d).length },
    maxPrice () { return this.list.reduce((m, x) => Math.max(m, x.price || 0), 0) },
    dtypeTag (d) { return { 1: 'success', 2: '', 3: 'warning', 4: 'danger', 5: 'info' }[d] || 'info' },
    load () {
      this.loading = true
      api.get('/admin/park-cars').then(r => {
        this.loading = false
        if (r.code === 0) this.list = r.data || []
      })
    },
    openDlg (row) {
      this.form = row ? { ...row } : { id: 0, name: '', dtype: 1, price: 1000, money: 5, status: 0 }
      this.dlg = true
    },
    save () {
      if (!this.form.name) { this.$message.warning('请填写车名'); return }
      this.saving = true
      const body = { name: this.form.name, dtype: this.form.dtype, price: this.form.price,
        money: this.form.money, status: this.form.status }
      const done = () => { this.saving = false; this.dlg = false; this.load() }
      const call = this.form.id
        ? api.put('/admin/park-cars/' + this.form.id, body)
        : api.post('/admin/park-cars', body)
      call.then(r => { if (r.code === 0) done(); else { this.saving = false; this.$message.error(r.msg) } })
    },
    del (row) {
      this.$confirm('删除车辆将同时清理用户车库中的该车，确认删除「' + row.name + '」？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/park-cars/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success(r.data.msg || '已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
@import './farm-admin.css';
</style>
