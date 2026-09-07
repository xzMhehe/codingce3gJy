<template>
  <div class="garden-admin">
    <!-- 统计卡片 -->
    <div class="stat-row">
      <div class="stat-card s-green">
        <div class="stat-ico el-icon-suitcase" />
        <div class="stat-info">
          <div class="stat-num">{{ list.length }}</div>
          <div class="stat-lab">花种总数</div>
        </div>
      </div>
      <div class="stat-card s-blue">
        <div class="stat-ico el-icon-sell" />
        <div class="stat-info">
          <div class="stat-num">{{ countDtype(0) }}</div>
          <div class="stat-lab">普通(商店)</div>
        </div>
      </div>
      <div class="stat-card s-purple">
        <div class="stat-ico el-icon-magic-stick" />
        <div class="stat-info">
          <div class="stat-num">{{ countDtype(1) }}</div>
          <div class="stat-lab">特殊(合成)</div>
        </div>
      </div>
      <div class="stat-card s-orange">
        <div class="stat-ico el-icon-sold-out" />
        <div class="stat-info">
          <div class="stat-num">{{ countStatus(1) }}</div>
          <div class="stat-lab">上架中</div>
        </div>
      </div>
    </div>

    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-select v-model="tyFilter" placeholder="类型" clearable style="width:140px" @change="page = 1">
          <el-option label="普通(商店)" :value="0" />
          <el-option label="特殊(合成)" :value="1" />
        </el-select>
        <el-input v-model.trim="kw" prefix-icon="el-icon-search" placeholder="搜索名称/花语" clearable style="width:230px" @input="page = 1" />
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增花种</el-button>
      </div>

      <!-- 按行表格 -->
      <el-table :data="paged" v-loading="loading" stripe border size="medium">
        <el-table-column prop="id" label="ID" width="64" align="center" />
        <el-table-column label="花种" width="110" align="center">
          <template slot-scope="{row}">
            <img :src="'/static/picture/garden/' + (row.img || ('s_s_' + row.id + '.gif'))"
                 class="td-img" :alt="row.name" />
          </template>
        </el-table-column>
        <el-table-column label="名称" min-width="110">
          <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column label="类型" width="88" align="center">
          <template slot-scope="{row}">
            <el-tag :type="row.dtype === 1 ? 'warning' : 'success'" size="mini">{{ row.dtype === 1 ? '特殊' : '普通' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="等级" width="70" align="center">
          <template slot-scope="{row}"><i class="el-icon-medal td-gold"></i>{{ row.level }}</template>
        </el-table-column>
        <el-table-column label="价格(G)" width="88" align="center">
          <template slot-scope="{row}">
            <span v-if="row.dtype === 1" class="td-muted">—</span>
            <span v-else><i class="el-icon-coin td-blue"></i>{{ row.price }}</span>
          </template>
        </el-table-column>
        <el-table-column label="生长(种/苗/蕾 分钟)" width="165" align="center">
          <template slot-scope="{row}"><span class="td-mono">{{ row.seed }}/{{ row.ling }}/{{ row.buds }}</span></template>
        </el-table-column>
        <el-table-column label="产量" width="78" align="center">
          <template slot-scope="{row}"><span class="td-blue">{{ row.less }}-{{ row.more }}</span></template>
        </el-table-column>
        <el-table-column label="花语" min-width="140" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-sub">{{ row.remark || '—' }}</span></template>
        </el-table-column>
        <el-table-column label="状态" width="78" align="center">
          <template slot-scope="{row}">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini" effect="plain">{{ row.status === 1 ? '上架' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="110" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" icon="el-icon-edit" circle title="编辑" @click="openDlg(row)" />
            <el-button size="mini" type="danger" icon="el-icon-delete" circle title="删除" @click="del(row)" />
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页底部栏（始终显示） -->
      <div class="pager-bar">
        <div class="pager-info">共 <b>{{ filtered.length }}</b> 条 · 每页 {{ pageSize }} 条</div>
        <el-pagination
          small
          background
          layout="sizes, prev, pager, next, jumper"
          :total="filtered.length"
          :page-size.sync="pageSize"
          :current-page.sync="page"
          :page-sizes="[5, 10, 20, 50]"
          @size-change="page = 1"
        />
      </div>
    </el-card>

    <el-dialog :title="form.id ? '编辑花种' : '新增花种'" :visible.sync="dlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <el-form-item label="名称">
          <el-input v-model.trim="form.name" maxlength="20" />
        </el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model.number="form.dtype">
            <el-radio :label="0">普通（商店购买）</el-radio>
            <el-radio :label="1">特殊（魔法屋合成）</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="所需等级">
          <el-input-number v-model.number="form.level" :min="1" :max="35" />
        </el-form-item>
        <el-form-item label="价格(G币)" v-if="form.dtype === 0">
          <el-input-number v-model.number="form.price" :min="0" />
        </el-form-item>
        <el-form-item label="生长分钟">
          种子期 <el-input-number v-model.number="form.seed" :min="1" size="small" style="width:90px" />
          花苗期 <el-input-number v-model.number="form.ling" :min="1" size="small" style="width:90px" />
          花蕾期 <el-input-number v-model.number="form.buds" :min="1" size="small" style="width:90px" />
        </el-form-item>
        <el-form-item label="产量范围">
          <el-input-number v-model.number="form.less" :min="1" size="small" style="width:110px" />
          ~
          <el-input-number v-model.number="form.more" :min="1" size="small" style="width:110px" />
        </el-form-item>
        <el-form-item label="花语">
          <el-input v-model.trim="form.remark" maxlength="40" placeholder="如：沉默的爱，勇敢追求幸福。" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model.number="form.status" :active-value="1" :inactive-value="0" active-text="上架" inactive-text="停用" />
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
  name: 'AdminGardenSeeds',
  data () {
    return {
      list: [], loading: false, dlg: false, saving: false,
      kw: '', tyFilter: null, page: 1, pageSize: 5,
      form: { id: 0, name: '', dtype: 0, level: 1, price: 0, seed: 1, ling: 1, buds: 1, less: 2, more: 4, remark: '', status: 1 }
    }
  },
  computed: {
    filtered () {
      const k = this.kw.trim().toLowerCase()
      return this.list.filter(x => {
        if (this.tyFilter !== null && x.dtype !== this.tyFilter) return false
        if (k && !(x.name || '').toLowerCase().includes(k) && !(x.remark || '').toLowerCase().includes(k)) return false
        return true
      })
    },
    paged () { return this.filtered.slice((this.page - 1) * this.pageSize, this.page * this.pageSize) }
  },
  mounted () { this.load() },
  methods: {
    countDtype (d) { return this.list.filter(x => x.dtype === d).length },
    countStatus (s) { return this.list.filter(x => x.status === s).length },
    load () {
      this.loading = true
      api.get('/admin/garden-seeds').then(r => {
        this.loading = false
        if (r.code === 0) this.list = r.data
      })
    },
    openDlg (row) {
      this.form = row ? { ...row } : { id: 0, name: '', dtype: 0, level: 1, price: 0, seed: 1, ling: 1, buds: 1, less: 2, more: 4, remark: '', status: 1 }
      this.dlg = true
    },
    save () {
      if (!this.form.name) { this.$message.warning('请填写花种名称'); return }
      this.saving = true
      const body = { name: this.form.name, dtype: this.form.dtype, level: this.form.level, price: this.form.price, seed: this.form.seed, ling: this.form.ling, buds: this.form.buds, less: this.form.less, more: this.form.more, remark: this.form.remark, status: this.form.status }
      const done = () => { this.saving = false; this.dlg = false; this.load() }
      if (this.form.id) {
        api.put('/admin/garden-seeds/' + this.form.id, body).then(r => { if (r.code === 0) done(); else this.$message.error(r.msg) })
      } else {
        api.post('/admin/garden-seeds', body).then(r => { if (r.code === 0) done(); else this.$message.error(r.msg) })
      }
    },
    del (row) {
      this.$confirm('删除花种将同时清理玩家背包中的该种子，确认删除「' + row.name + '」？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/garden-seeds/' + row.id).then(() => this.load())
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
.stat-row { display: flex; gap: 12px; margin-bottom: 18px; }
.stat-card {
  flex: 1; display: flex; align-items: center; gap: 14px;
  background: #fff; border-radius: 12px; padding: 16px 18px;
  border: 1px solid #eef1f5; box-shadow: 0 2px 8px rgba(18,38,63,.05);
  transition: box-shadow .2s, transform .2s;
}
.stat-card:hover { box-shadow: 0 6px 18px rgba(18,38,63,.09); transform: translateY(-2px); }
.stat-ico {
  width: 46px; height: 46px; border-radius: 12px; flex-shrink: 0;
  display: flex; align-items: center; justify-content: center;
  color: #fff; font-size: 22px;
}
.s-green .stat-ico { background: linear-gradient(135deg,#43a047,#2e7d32); }
.s-blue .stat-ico { background: linear-gradient(135deg,#29b6f6,#0288d1); }
.s-purple .stat-ico { background: linear-gradient(135deg,#ab47bc,#6a1b9a); }
.s-orange .stat-ico { background: linear-gradient(135deg,#ffa726,#ef6c00); }
.stat-info { display: flex; flex-direction: column; }
.stat-num { font-size: 24px; font-weight: 700; color: #1f2d3d; line-height: 1; }
.stat-lab { font-size: 12px; color: #8a9bb0; margin-top: 6px; letter-spacing: .3px; }
.toolbar { display: flex; align-items: center; margin-bottom: 14px; flex-wrap: wrap; gap: 8px; }
.grow { flex: 1; }

/* 表格样式 */
.td-img { width: 52px; height: 52px; object-fit: contain; border-radius: 6px; border: 1px solid #eef2f6; background: #f7fafc; vertical-align: middle; }
.td-main { font-weight: 600; color: #303133; }
.td-sub { color: #8a9bb0; font-size: 13px; }
.td-muted { color: #b6c2d2; }
.td-mono { font-family: Consolas, 'Courier New', monospace; color: #5b6b82; }
.td-blue { color: #2e9cd3; font-weight: 600; }
.td-gold { color: #fb8c00; margin-right: 3px; }
/* 分页 */
.pager-bar { margin-top: 14px; padding-top: 12px; border-top: 1px solid #f0f2f5; display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px; }
.pager-info { font-size: 13px; color: #909399; }
.pager-info b { color: #303133; font-weight: 600; margin: 0 2px; }
.pager-bar >>> .el-pagination { margin: 0; }
</style>