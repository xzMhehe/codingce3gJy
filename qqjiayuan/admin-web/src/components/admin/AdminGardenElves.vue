<template>
  <div class="garden-admin">
    <!-- 统计卡片 -->
    <el-row :gutter="12" class="stat-row">
      <el-col :span="8">
        <div class="stat-card s-green">
          <div class="stat-ico el-icon-magic-stick" />
          <div class="stat-info">
            <div class="stat-num">{{ total }}</div>
            <div class="stat-lab">精灵总数</div>
          </div>
        </div>
      </el-col>
      <el-col :span="8">
        <div class="stat-card s-blue">
          <div class="stat-ico el-icon-circle-check" />
          <div class="stat-info">
            <div class="stat-num">{{ enabled }}</div>
            <div class="stat-lab">启用中</div>
          </div>
        </div>
      </el-col>
      <el-col :span="8">
        <div class="stat-card s-orange">
          <div class="stat-ico el-icon-collection-tag" />
          <div class="stat-info">
            <div class="stat-num">{{ minNeed }}~{{ maxNeed }}</div>
            <div class="stat-lab">解锁所需图谱(点亮)</div>
          </div>
        </div>
      </el-col>
    </el-row>

    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model.trim="kw" prefix-icon="el-icon-search" placeholder="搜索精灵名称/介绍" clearable style="width:250px" @input="page = 1" />
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增精灵</el-button>
      </div>

      <!-- 按行表格 -->
      <el-table :data="paged" v-loading="loading" stripe border size="medium">
        <el-table-column prop="id" label="ID" width="64" align="center" />
        <el-table-column label="精灵" width="120" align="center">
          <template slot-scope="{row}">
            <img :src="'/static/picture/garden/' + (row.img || 'elf_' + row.id + '.png')"
                 class="td-img" :alt="row.name" />
          </template>
        </el-table-column>
        <el-table-column label="名称" width="140">
          <template slot-scope="{row}"><span class="td-main">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column label="介绍" min-width="180" show-overflow-tooltip>
          <template slot-scope="{row}"><span class="td-sub">{{ row.desc || '—' }}</span></template>
        </el-table-column>
        <el-table-column label="解锁条件" width="150" align="center">
          <template slot-scope="{row}"><span class="td-cond">点亮 {{ row.need_map }} 个图谱</span></template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="80" align="center">
          <template slot-scope="{row}"><span class="td-sort">{{ row.sort }}</span></template>
        </el-table-column>
        <el-table-column label="状态" width="80" align="center">
          <template slot-scope="{row}">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini" effect="plain">{{ row.status === 1 ? '启用' : '停用' }}</el-tag>
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

    <el-dialog :title="form.id ? '编辑精灵' : '新增精灵'" :visible.sync="dlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="130px">
        <el-form-item label="名称">
          <el-input v-model.trim="form.name" maxlength="30" show-word-limit />
        </el-form-item>
        <el-form-item label="介绍">
          <el-input type="textarea" v-model="form.desc" :rows="3" maxlength="200" show-word-limit />
        </el-form-item>
        <el-form-item label="图片文件名">
          <div class="img-pick">
            <el-input v-model.trim="form.img" placeholder="如 elf_1.png，留空按精灵ID取" maxlength="50" style="flex:1" />
            <img v-if="previewUrl" :src="previewUrl" class="dlg-prev" alt="预览" @error="previewErr = true" />
          </div>
          <div class="help-line">图片放在 static/picture/garden/ 下，留空时按 elf_ID.png 取</div>
        </el-form-item>
        <el-form-item label="解锁需点亮图谱">
          <el-input-number v-model="form.need_map" :min="1" :max="619" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort" :min="0" :max="999" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" active-text="启用" inactive-text="停用" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="dlg = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminGardenElves',
  data () {
    return {
      list: [], loading: false, saving: false, dlg: false, form: {},
      kw: '', page: 1, pageSize: 5
    }
  },
  computed: {
    total () { return this.list.length },
    enabled () { return this.list.filter(x => x.status === 1).length },
    minNeed () { return this.list.length ? Math.min(...this.list.map(x => x.need_map)) : 0 },
    maxNeed () { return this.list.length ? Math.max(...this.list.map(x => x.need_map)) : 0 },
    filtered () {
      const k = this.kw.trim().toLowerCase()
      if (!k) return this.list
      return this.list.filter(x => (x.name || '').toLowerCase().includes(k) || (x.desc || '').toLowerCase().includes(k))
    },
    paged () { return this.filtered.slice((this.page - 1) * this.pageSize, this.page * this.pageSize) },
    previewUrl () {
      const f = this.form.img || (this.form.id ? ('elf_' + this.form.id + '.png') : '')
      return f ? '/static/picture/garden/' + f : ''
    }
  },
  created () { this.load() },
  watch: {
    kw () { this.page = 1 }
  },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/garden-elves').then(r => {
        this.loading = false
        if (r.code === 0) this.list = r.data || []
        else this.$message.error(r.msg || '加载失败')
      })
    },
    openDlg (row) {
      this.form = row ? { ...row } : { name: '', desc: '', img: '', need_map: 1, sort: 0, status: 1 }
      this.dlg = true
    },
    save () {
      if (!this.form.name) { this.$message.warning('请填写精灵名称'); return }
      this.saving = true
      const req = { ...this.form }
      const apiCall = req.id ? api.put('/admin/garden-elves/' + req.id, req) : api.post('/admin/garden-elves', req)
      apiCall.then(r => {
        this.saving = false
        if (r.code === 0) { this.$message.success('保存成功'); this.dlg = false; this.load() } else this.$message.error(r.msg || '保存失败')
      })
    },
    del (row) {
      this.$confirm('删除后该精灵点亮记录也会清除，确定删除「' + row.name + '」？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/garden-elves/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg || '删除失败')
        })
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
.s-blue .stat-ico { background: linear-gradient(135deg,#29b6f6,#0288d1); }
.s-orange .stat-ico { background: linear-gradient(135deg,#ffa726,#ef6c00); }
.stat-info { display: flex; flex-direction: column; }
.stat-num { font-size: 24px; font-weight: 700; color: #1f2d3d; line-height: 1; }
.stat-lab { font-size: 12px; color: #8a9bb0; margin-top: 6px; letter-spacing: .3px; }
.toolbar { display: flex; align-items: center; margin-bottom: 14px; flex-wrap: wrap; gap: 8px; }
.grow { flex: 1; }

/* 表格样式 */
.td-img { width: 52px; height: 52px; object-fit: contain; border-radius: 6px; border: 1px solid #ece7f7; background: #f7f5fd; vertical-align: middle; }
.td-main { font-weight: 600; color: #303133; }
.td-sub { color: #8a9bb0; font-size: 13px; }
.td-cond { color: #7367f0; font-weight: 600; }
.td-sort { color: #5b6b82; font-weight: 600; }
/* 分页 */
.pager-bar { margin-top: 14px; padding-top: 12px; border-top: 1px solid #f0f2f5; display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px; }
.pager-info { font-size: 13px; color: #909399; }
.pager-info b { color: #303133; font-weight: 600; margin: 0 2px; }
.pager-bar >>> .el-pagination { margin: 0; }
.img-pick { display: flex; align-items: center; gap: 10px; }
.dlg-prev { width: 52px; height: 52px; border-radius: 8px; border: 1px solid #e3eef8; background: #f5f9fc; object-fit: contain; }
.help-line { font-size: 12px; color: #999; margin-top: 4px; }
</style>