<template>
  <div class="garden-admin">
    <!-- 统计卡片 -->
    <el-row :gutter="12" class="stat-row">
      <el-col :span="6">
        <div class="stat-card s-green">
          <div class="stat-ico el-icon-suitcase" />
          <div class="stat-info">
            <div class="stat-num">{{ list.length }}</div>
            <div class="stat-lab">花种总数</div>
          </div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card s-blue">
          <div class="stat-ico el-icon-sell" />
          <div class="stat-info">
            <div class="stat-num">{{ countDtype(0) }}</div>
            <div class="stat-lab">普通(商店)</div>
          </div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card s-purple">
          <div class="stat-ico el-icon-magic-stick" />
          <div class="stat-info">
            <div class="stat-num">{{ countDtype(1) }}</div>
            <div class="stat-lab">特殊(合成)</div>
          </div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card s-orange">
          <div class="stat-ico el-icon-sold-out" />
          <div class="stat-info">
            <div class="stat-num">{{ countStatus(1) }}</div>
            <div class="stat-lab">上架中</div>
          </div>
        </div>
      </el-col>
    </el-row>

    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-select v-model="tyFilter" placeholder="类型" clearable style="width:130px" @change="page = 1">
          <el-option label="普通(商店)" :value="0" />
          <el-option label="特殊(合成)" :value="1" />
        </el-select>
        <el-input v-model.trim="kw" prefix-icon="el-icon-search" placeholder="搜索名称/花语" clearable style="width:220px;margin-left:8px" @input="page = 1" />
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增花种</el-button>
      </div>
      <el-table :data="paged" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column label="种子图" width="86" header-align="center">
          <template slot-scope="{row}">
            <img :src="'/static/picture/garden/' + (row.img || ('s_s_' + row.id + '.gif'))" class="seed-prev" :alt="row.name" />
          </template>
        </el-table-column>
        <el-table-column prop="name" label="名称" min-width="110" />
        <el-table-column label="类型" width="120">
          <template slot-scope="{row}">
            <el-tag :type="row.dtype === 1 ? 'warning' : 'success'" size="mini" effect="light">{{ row.dtype === 1 ? '特殊(合成)' : '普通' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="level" label="等级" width="70" header-align="center" />
        <el-table-column prop="price" label="价格(G币)" width="100" header-align="center">
          <template slot-scope="{row}">
            <span v-if="row.dtype === 1" class="mix-only">—</span>
            <span v-else>{{ row.price }}</span>
          </template>
        </el-table-column>
        <el-table-column label="生长(分)" width="160">
          <template slot-scope="{row}">
            <span class="grow-txt">{{ row.seed }}/{{ row.ling }}/{{ row.buds }}</span>
          </template>
        </el-table-column>
        <el-table-column label="产量" width="80" header-align="center">
          <template slot-scope="{row}">{{ row.less }}-{{ row.more }}</template>
        </el-table-column>
        <el-table-column prop="remark" label="花语" min-width="160" show-overflow-tooltip />
        <el-table-column label="状态" width="80" header-align="center">
          <template slot-scope="{row}">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '上架' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" header-align="center">
          <template slot-scope="{row}">
            <div class="ops">
              <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openDlg(row)">编辑</el-button>
              <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="del(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <div class="pager" v-if="paged.length < filtered.length">
        <el-pagination layout="prev, pager, next" :total="filtered.length" :page-size="pageSize" :current-page.sync="page" background />
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
      kw: '', tyFilter: null, page: 1, pageSize: 12,
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
.stat-row { margin-bottom: 14px; }
.stat-card { display: flex; align-items: center; background: #fff; border-radius: 10px; padding: 14px 16px; border: 1px solid #eef1f5; }
.stat-ico { font-size: 28px; margin-right: 12px; }
.s-green .stat-ico { color: #43a047; }
.s-blue .stat-ico { color: #2e9cd3; }
.s-purple .stat-ico { color: #7b1fa2; }
.s-orange .stat-ico { color: #fb8c00; }
.stat-num { font-size: 22px; font-weight: bold; color: #333; line-height: 1.2; }
.stat-lab { font-size: 12px; color: #999; }
.toolbar { display: flex; align-items: center; margin-bottom: 12px; }
.grow { flex: 1; }
.seed-prev { width: 40px; height: 40px; border-radius: 8px; background: #f5f9fc; border: 1px solid #e3eef8; object-fit: contain; }
.grow-txt { font-size: 12px; color: #666; font-family: Consolas, monospace; }
.mix-only { color: #bbb; }
.help-line { font-size: 12px; color: #999; margin-top: 4px; }
.pager { margin-top: 12px; text-align: right; }
</style>
