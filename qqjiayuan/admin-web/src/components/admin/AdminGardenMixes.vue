<template>
  <div class="garden-admin">
    <!-- 统计卡片 -->
    <el-row :gutter="12" class="stat-row">
      <el-col :span="8">
        <div class="stat-card s-purple">
          <div class="stat-ico el-icon-magic-stick" />
          <div class="stat-info">
            <div class="stat-num">{{ list.length }}</div>
            <div class="stat-lab">配方总数</div>
          </div>
        </div>
      </el-col>
      <el-col :span="8">
        <div class="stat-card s-blue">
          <div class="stat-ico el-icon-sunny" />
          <div class="stat-info">
            <div class="stat-num">{{ seedCount }}</div>
            <div class="stat-lab">涉及产物种子</div>
          </div>
        </div>
      </el-col>
      <el-col :span="8">
        <div class="stat-card s-orange">
          <div class="stat-ico el-icon-folder-opened" />
          <div class="stat-info">
            <div class="stat-num">{{ flowerCount }}</div>
            <div class="stat-lab">涉及材料花</div>
          </div>
        </div>
      </el-col>
    </el-row>

    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model.trim="kw" prefix-icon="el-icon-search" placeholder="搜索产物种子/材料花" clearable style="width:250px" @input="page = 1" />
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增配方</el-button>
      </div>

      <!-- 按行表格 -->
      <el-table :data="paged" v-loading="loading" stripe border size="medium">
        <el-table-column prop="id" label="ID" width="64" align="center" />
        <el-table-column label="材料花" min-width="140">
          <template slot-scope="{row}">
            <span class="mix-flower"><i class="el-icon-sunny"></i>{{ row.flower }}</span>
          </template>
        </el-table-column>
        <el-table-column label="需要数量" width="110" align="center">
          <template slot-scope="{row}"><span class="td-need">× {{ row.need }}</span></template>
        </el-table-column>
        <el-table-column label="合成" width="70" align="center">
          <template slot-scope=""><span class="td-arrow">→</span></template>
        </el-table-column>
        <el-table-column label="产物种子" min-width="150">
          <template slot-scope="{row}">
            <div class="seed-inline">
              <img :src="'/static/picture/garden/' + (row.seed_img || ('s_s_' + row.seed_id + '.gif'))"
                   class="td-img" :alt="row.seed_name" />
              <span class="td-main">{{ row.seed_name }}</span>
            </div>
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

    <el-dialog :title="form.id ? '编辑配方' : '新增配方'" :visible.sync="dlg" width="520px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <el-form-item label="产物种子">
          <el-select v-model.number="form.seed_id" filterable placeholder="选择特殊种子" style="width:100%">
            <el-option v-for="s in specialSeeds" :key="s.id" :label="s.name" :value="s.id" />
          </el-select>
          <div class="help-line">配方产物应为「特殊（魔法屋合成）」类型种子</div>
        </el-form-item>
        <el-form-item label="材料花">
          <el-select v-model="form.flower" filterable allow-create placeholder="输入或选择花朵" style="width:100%">
            <el-option v-for="f in allFlowers" :key="f" :label="f" :value="f" />
          </el-select>
        </el-form-item>
        <el-form-item label="需要数量">
          <el-input-number v-model.number="form.need" :min="1" :max="999" />
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
  name: 'AdminGardenMixes',
  data () {
    return {
      list: [], seeds: [], loading: false, dlg: false, saving: false,
      kw: '', page: 1, pageSize: 5,
      form: { id: 0, seed_id: 0, flower: '', need: 1 }
    }
  },
  computed: {
    specialSeeds () { return this.seeds.filter(s => s.dtype === 1) },
    allFlowers () {
      const set = new Set()
      this.seeds.forEach(s => set.add(s.name))
      this.list.forEach(m => set.add(m.flower))
      return [...set]
    },
    seedCount () { return new Set(this.list.map(x => x.seed_id)).size },
    flowerCount () { return new Set(this.list.map(x => x.flower)).size },
    filtered () {
      const k = this.kw.trim().toLowerCase()
      if (!k) return this.list
      return this.list.filter(x => (x.seed_name || '').toLowerCase().includes(k) || (x.flower || '').toLowerCase().includes(k))
    },
    paged () { return this.filtered.slice((this.page - 1) * this.pageSize, this.page * this.pageSize) }
  },
  mounted () { this.load(); this.loadSeeds() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/garden-mixes').then(r => {
        this.loading = false
        if (r.code === 0) this.list = r.data
      })
    },
    loadSeeds () {
      api.get('/admin/garden-seeds').then(r => { if (r.code === 0) this.seeds = r.data })
    },
    openDlg (row) {
      this.form = row ? { id: row.id, seed_id: row.seed_id, flower: row.flower, need: row.need } : { id: 0, seed_id: 0, flower: '', need: 1 }
      this.dlg = true
    },
    save () {
      if (!this.form.seed_id) { this.$message.warning('请选择产物种子'); return }
      if (!this.form.flower) { this.$message.warning('请填写材料花'); return }
      this.saving = true
      const body = { seed_id: this.form.seed_id, flower: this.form.flower, need: this.form.need }
      const done = () => { this.saving = false; this.dlg = false; this.load() }
      if (this.form.id) {
        api.put('/admin/garden-mixes/' + this.form.id, body).then(r => { if (r.code === 0) done(); else this.$message.error(r.msg) })
      } else {
        api.post('/admin/garden-mixes', body).then(r => { if (r.code === 0) done(); else this.$message.error(r.msg) })
      }
    },
    del (row) {
      this.$confirm('确认删除「' + row.seed_name + ' ← ' + row.flower + '×' + row.need + '」配方？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/garden-mixes/' + row.id).then(() => this.load())
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
.stat-row { margin-bottom: 18px; }
.stat-card {
  display: flex; align-items: center; gap: 14px;
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
.s-purple .stat-ico { background: linear-gradient(135deg,#ab47bc,#6a1b9a); }
.s-blue .stat-ico { background: linear-gradient(135deg,#29b6f6,#0288d1); }
.s-orange .stat-ico { background: linear-gradient(135deg,#ffa726,#ef6c00); }
.stat-info { display: flex; flex-direction: column; }
.stat-num { font-size: 24px; font-weight: 700; color: #1f2d3d; line-height: 1; }
.stat-lab { font-size: 12px; color: #8a9bb0; margin-top: 6px; letter-spacing: .3px; }
.toolbar { display: flex; align-items: center; margin-bottom: 14px; flex-wrap: wrap; gap: 8px; }
.grow { flex: 1; }

/* 表格样式 */
.td-img { width: 46px; height: 46px; object-fit: contain; border-radius: 6px; border: 1px solid #eef2f6; background: #f7fafc; vertical-align: middle; margin-right: 8px; }
.seed-inline { display: flex; align-items: center; }
.td-main { font-weight: 600; color: #303133; }
.mix-flower { color: #d97b2b; font-weight: 600; display: inline-flex; align-items: center; gap: 5px; }
.mix-flower i { color: #fb8c00; }
.td-need { font-weight: 600; color: #7b1fa2; }
.td-arrow { color: #c9b8ab; font-size: 18px; font-weight: 700; }
/* 分页 */
.pager-bar { margin-top: 14px; padding-top: 12px; border-top: 1px solid #f0f2f5; display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px; }
.pager-info { font-size: 13px; color: #909399; }
.pager-info b { color: #303133; font-weight: 600; margin: 0 2px; }
.pager-bar >>> .el-pagination { margin: 0; }
.help-line { font-size: 12px; color: #999; margin-top: 4px; }
</style>