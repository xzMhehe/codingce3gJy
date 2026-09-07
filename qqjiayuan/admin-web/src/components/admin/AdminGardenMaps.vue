<template>
  <div class="garden-admin">
    <!-- 统计卡片 -->
    <el-row :gutter="12" class="stat-row">
      <el-col :span="6">
        <div class="stat-card s-green">
          <div class="stat-ico el-icon-sunny" />
          <div class="stat-info">
            <div class="stat-num">{{ countBy(0) }}</div>
            <div class="stat-lab">普通</div>
          </div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card s-purple">
          <div class="stat-ico el-icon-star-off" />
          <div class="stat-info">
            <div class="stat-num">{{ countBy(1) }}</div>
            <div class="stat-lab">独特</div>
          </div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card s-red">
          <div class="stat-ico el-icon-medal" />
          <div class="stat-info">
            <div class="stat-num">{{ countBy(2) }}</div>
            <div class="stat-lab">珍稀</div>
          </div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card s-blue">
          <div class="stat-ico el-icon-collection" />
          <div class="stat-info">
            <div class="stat-num">{{ list.length }}</div>
            <div class="stat-lab">图谱总数</div>
          </div>
        </div>
      </el-col>
    </el-row>

    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-select v-model="tyFilter" placeholder="稀有度" clearable style="width:110px" @change="page = 1">
          <el-option label="普通" :value="0" />
          <el-option label="独特" :value="1" />
          <el-option label="珍稀" :value="2" />
        </el-select>
        <el-input v-model.trim="kw" prefix-icon="el-icon-search" placeholder="搜索花名/种子" clearable style="width:220px;margin-left:8px" @input="page = 1" />
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增图鉴花</el-button>
      </div>
      <el-table :data="paged" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column label="图片" width="86" header-align="center">
          <template slot-scope="{row}">
            <img :src="'/static/picture/garden/' + (row.img || ('m_s_' + row.id + '.gif'))" class="map-prev" :alt="row.name" />
          </template>
        </el-table-column>
        <el-table-column prop="name" label="花名" min-width="120" />
        <el-table-column prop="seed_name" label="所属种子" min-width="110" />
        <el-table-column label="稀有度" width="90" header-align="center">
          <template slot-scope="{row}">
            <el-tag :type="['success', 'warning', 'danger'][row.dtype] || 'info'" size="mini">{{ ['普通', '独特', '珍稀'][row.dtype] }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="img" label="图片文件" min-width="120" show-overflow-tooltip />
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

    <el-dialog :title="form.id ? '编辑图鉴花' : '新增图鉴花'" :visible.sync="dlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="90px">
        <el-form-item label="花名">
          <el-input v-model.trim="form.name" maxlength="20" />
        </el-form-item>
        <el-form-item label="所属种子">
          <el-select v-model.number="form.seed_id" filterable placeholder="选择种子" style="width:100%">
            <el-option v-for="s in seeds" :key="s.id" :label="s.name" :value="s.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="稀有度">
          <el-radio-group v-model.number="form.dtype">
            <el-radio :label="0">普通</el-radio>
            <el-radio :label="1">独特</el-radio>
            <el-radio :label="2">珍稀</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="图片文件">
          <div class="img-pick">
            <el-input v-model.trim="form.img" placeholder="如 m_s_21.gif（留空按 ID 取图）" style="flex:1" />
            <img v-if="previewUrl" :src="previewUrl" class="dlg-prev" alt="预览" />
          </div>
          <div class="help-line">图片位于 web/public/static/picture/garden/ 下</div>
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
  name: 'AdminGardenMaps',
  data () {
    return {
      list: [], seeds: [], loading: false, dlg: false, saving: false,
      kw: '', tyFilter: null, page: 1, pageSize: 12,
      form: { id: 0, seed_id: 1, name: '', dtype: 0, img: '' }
    }
  },
  computed: {
    filtered () {
      const k = this.kw.trim().toLowerCase()
      return this.list.filter(x => {
        if (this.tyFilter !== null && x.dtype !== this.tyFilter) return false
        if (k && !(x.name || '').toLowerCase().includes(k) && !(x.seed_name || '').toLowerCase().includes(k)) return false
        return true
      })
    },
    paged () { return this.filtered.slice((this.page - 1) * this.pageSize, this.page * this.pageSize) },
    previewUrl () {
      const f = this.form.img || (this.form.id ? ('m_s_' + this.form.id + '.gif') : '')
      return f ? '/static/picture/garden/' + f : ''
    }
  },
  mounted () { this.load(); this.loadSeeds() },
  methods: {
    countBy (dtype) { return this.list.filter(x => x.dtype === dtype).length },
    load () {
      this.loading = true
      api.get('/admin/garden-maps').then(r => {
        this.loading = false
        if (r.code === 0) this.list = r.data
      })
    },
    loadSeeds () {
      api.get('/admin/garden-seeds').then(r => { if (r.code === 0) this.seeds = r.data })
    },
    openDlg (row) {
      this.form = row ? { id: row.id, seed_id: row.seed_id, name: row.name, dtype: row.dtype, img: row.img || '' } : { id: 0, seed_id: 1, name: '', dtype: 0, img: '' }
      this.dlg = true
    },
    save () {
      if (!this.form.name) { this.$message.warning('请填写花名'); return }
      this.saving = true
      const body = { seed_id: this.form.seed_id, name: this.form.name, dtype: this.form.dtype, img: this.form.img }
      const done = () => { this.saving = false; this.dlg = false; this.load() }
      if (this.form.id) {
        api.put('/admin/garden-maps/' + this.form.id, body).then(r => { if (r.code === 0) done(); else this.$message.error(r.msg) })
      } else {
        api.post('/admin/garden-maps', body).then(r => { if (r.code === 0) done(); else this.$message.error(r.msg) })
      }
    },
    del (row) {
      this.$confirm('删除图鉴「' + row.name + '」将同时清除玩家点亮记录，确认删除？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/garden-maps/' + row.id).then(() => this.load())
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
.s-purple .stat-ico { color: #7b1fa2; }
.s-red .stat-ico { color: #d32f2f; }
.s-blue .stat-ico { color: #2e9cd3; }
.stat-num { font-size: 22px; font-weight: bold; color: #333; line-height: 1.2; }
.stat-lab { font-size: 12px; color: #999; }
.toolbar { display: flex; align-items: center; margin-bottom: 12px; }
.grow { flex: 1; }
.map-prev { width: 44px; height: 44px; border-radius: 8px; background: #f5f9fc; border: 1px solid #e3eef8; object-fit: contain; }
.img-pick { display: flex; align-items: center; gap: 10px; }
.dlg-prev { width: 52px; height: 52px; border-radius: 8px; border: 1px solid #e3eef8; background: #f5f9fc; object-fit: contain; }
.help-line { font-size: 12px; color: #999; margin-top: 4px; }
.pager { margin-top: 12px; text-align: right; }
</style>
