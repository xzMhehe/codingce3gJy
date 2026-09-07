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
        <el-select v-model="tyFilter" placeholder="稀有度" clearable style="width:120px" @change="page = 1">
          <el-option label="普通" :value="0" />
          <el-option label="独特" :value="1" />
          <el-option label="珍稀" :value="2" />
        </el-select>
        <el-input v-model.trim="kw" prefix-icon="el-icon-search" placeholder="搜索花名/种子" clearable style="width:230px" @input="page = 1" />
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增图鉴花</el-button>
      </div>

      <!-- 按行表格 -->
      <el-table :data="paged" v-loading="loading" stripe border size="medium">
        <el-table-column prop="id" label="ID" width="64" align="center" />
        <el-table-column label="图片" width="110" align="center">
          <template slot-scope="{row}">
            <img :src="'/static/picture/garden/' + (row.img || ('m_s_' + row.id + '.gif'))"
                 class="td-img" :alt="row.name" />
          </template>
        </el-table-column>
        <el-table-column label="花名" min-width="130">
          <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column label="稀有度" width="100" align="center">
          <template slot-scope="{row}">
            <el-tag :type="['success', 'warning', 'danger'][row.dtype] || 'info'" size="mini">{{ ['普通', '独特', '珍稀'][row.dtype] }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="所属种子" min-width="130">
          <template slot-scope="{row}"><span class="td-sub">{{ row.seed_name || '—' }}</span></template>
        </el-table-column>
        <el-table-column label="图片文件" min-width="160">
          <template slot-scope="{row}"><span class="td-file">{{ row.img || ('m_s_' + row.id + '.gif') }}</span></template>
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
      kw: '', tyFilter: null, page: 1, pageSize: 5,
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
.s-green .stat-ico { background: linear-gradient(135deg,#43a047,#2e7d32); }
.s-purple .stat-ico { background: linear-gradient(135deg,#ab47bc,#6a1b9a); }
.s-red .stat-ico { background: linear-gradient(135deg,#ef5350,#c62828); }
.s-blue .stat-ico { background: linear-gradient(135deg,#29b6f6,#0288d1); }
.stat-info { display: flex; flex-direction: column; }
.stat-num { font-size: 24px; font-weight: 700; color: #1f2d3d; line-height: 1; }
.stat-lab { font-size: 12px; color: #8a9bb0; margin-top: 6px; letter-spacing: .3px; }
.toolbar { display: flex; align-items: center; margin-bottom: 14px; flex-wrap: wrap; gap: 8px; }
.grow { flex: 1; }

/* 表格样式 */
.td-img { width: 52px; height: 52px; object-fit: contain; border-radius: 6px; border: 1px solid #eef2f6; background: #f7fafc; vertical-align: middle; }
.td-main { font-weight: 600; color: #303133; }
.td-sub { color: #8a9bb0; font-size: 13px; }
.td-file { color: #2e9cd3; font-family: Consolas, 'Courier New', monospace; font-size: 13px; }
/* 分页 */
.pager-bar { margin-top: 14px; padding-top: 12px; border-top: 1px solid #f0f2f5; display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px; }
.pager-info { font-size: 13px; color: #909399; }
.pager-info b { color: #303133; font-weight: 600; margin: 0 2px; }
.pager-bar >>> .el-pagination { margin: 0; }
.img-pick { display: flex; align-items: center; gap: 10px; }
.dlg-prev { width: 52px; height: 52px; border-radius: 8px; border: 1px solid #e3eef8; background: #f5f9fc; object-fit: contain; }
.help-line { font-size: 12px; color: #999; margin-top: 4px; }
</style>