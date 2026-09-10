<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <span class="txt-fade">管理同城客栈下的城市板块</span>
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增城市</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="城市" min-width="160" />
        <el-table-column label="状态" width="100" header-align="center">
          <template slot-scope="{row}"><el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '显示' : '隐藏' }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="80" header-align="center" />
        <el-table-column label="操作" width="220" align="center" fixed="right">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain @click="openDlg(row)">编辑</el-button>
            <el-button size="mini" @click="toggle(row)">{{ row.status === 1 ? '隐藏' : '显示' }}</el-button>
            <el-button size="mini" type="danger" plain @click="del(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog :title="form.id ? '编辑城市' : '新增城市'" :visible.sync="dlg" width="500px" :close-on-click-modal="false">
      <el-form label-width="80px">
        <el-form-item label="城市"><el-input v-model.trim="form.name" maxlength="30" /></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0" /></el-form-item>
      </el-form>
      <div slot="footer"><el-button @click="dlg = false">取 消</el-button><el-button type="primary" @click="save">保 存</el-button></div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminTongcheng',
  data () { return { list: [], loading: false, dlg: false, form: { id: 0, name: '', sort: 0 }, parentId: 0 } },
  mounted () {
    this.load()
    api.get('/boards').then(r => {
      if (r.code === 0) {
        const tc = r.data.find(b => b.name === '同城客栈')
        if (tc) this.parentId = tc.id
      }
    })
  },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/tongcheng').then(r => { this.loading = false; if (r.code === 0) this.list = r.data })
    },
    openDlg (row) { this.form = row ? { id: row.id, name: row.name, sort: row.sort } : { id: 0, name: '', sort: 0 }; this.dlg = true },
    save () {
      if (!this.form.name) { this.$message.warning('请填写城市'); return }
      if (this.form.id) {
        api.put('/admin/boards/' + this.form.id, { name: this.form.name, sort: this.form.sort }).then(r => { if (r.code === 0) { this.dlg = false; this.load() } })
      } else {
        api.post('/admin/boards', { name: this.form.name, parent_id: this.parentId, sort: this.form.sort }).then(r => { if (r.code === 0) { this.dlg = false; this.load() } })
      }
    },
    toggle (row) { api.put('/admin/boards/' + row.id, { name: row.name, sort: row.sort, status: row.status === 1 ? 0 : 1 }).then(() => this.load()) },
    del (row) { api.delete('/admin/boards/' + row.id).then(() => this.load()) }
  }
}
</script>
