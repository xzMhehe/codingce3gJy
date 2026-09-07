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
        <el-input v-model.trim="kw" prefix-icon="el-icon-search" placeholder="搜索产物种子/材料花" clearable style="width:240px" @input="page = 1" />
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增配方</el-button>
      </div>
      <el-table :data="paged" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column label="产物种子" min-width="180">
          <template slot-scope="{row}">
            <div class="mix-cell">
              <img :src="'/static/picture/garden/' + (row.seed_img || ('s_s_' + row.seed_id + '.gif'))" class="mix-prev" :alt="row.seed_name" />
              <span class="mix-seed">{{ row.seed_name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="配方" min-width="180">
          <template slot-scope="{row}">
            <span class="mix-formula">
              <i class="el-icon-sunny flower-dot" />{{ row.flower }}
              <b class="mix-need">× {{ row.need }}</b>
            </span>
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
      kw: '', page: 1, pageSize: 12,
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
.stat-row { margin-bottom: 14px; }
.stat-card { display: flex; align-items: center; background: #fff; border-radius: 10px; padding: 14px 16px; border: 1px solid #eef1f5; }
.stat-ico { font-size: 28px; margin-right: 12px; }
.s-purple .stat-ico { color: #7b1fa2; }
.s-blue .stat-ico { color: #2e9cd3; }
.s-orange .stat-ico { color: #fb8c00; }
.stat-num { font-size: 22px; font-weight: bold; color: #333; line-height: 1.2; }
.stat-lab { font-size: 12px; color: #999; }
.toolbar { display: flex; align-items: center; margin-bottom: 12px; }
.grow { flex: 1; }
.mix-cell { display: flex; align-items: center; gap: 8px; }
.mix-prev { width: 36px; height: 36px; border-radius: 6px; background: #f5f9fc; border: 1px solid #e3eef8; object-fit: contain; }
.mix-seed { font-size: 13px; color: #333; }
.mix-formula { font-size: 13px; color: #555; }
.flower-dot { color: #e91e63; margin-right: 2px; }
.mix-need { color: #fb8c00; }
.help-line { font-size: 12px; color: #999; margin-top: 4px; }
.pager { margin-top: 12px; text-align: right; }
</style>
