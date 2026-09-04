<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增活动</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="title" label="标题" min-width="180" show-overflow-tooltip />
        <el-table-column prop="desc" label="说明" min-width="220" show-overflow-tooltip />
        <el-table-column label="状态" width="80">
          <template slot-scope="{row}">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '显示' : '隐藏' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right" header-align="center">
          <template slot-scope="{row}">
            <div class="ops">
              <el-button size="mini" type="primary" plain icon="el-icon-edit" @click="openDlg(row)">编辑</el-button>
              <el-button size="mini" @click="toggle(row)">{{ row.status === 1 ? '隐藏' : '显示' }}</el-button>
              <el-button size="mini" type="danger" plain icon="el-icon-delete" @click="del(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog :title="form.id ? '编辑活动' : '新增活动'" :visible.sync="dlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="80px">
        <el-form-item label="标题">
          <el-input v-model.trim="form.title" maxlength="60" show-word-limit />
        </el-form-item>
        <el-form-item label="说明">
          <el-input type="textarea" v-model="form.desc" :rows="4" maxlength="200" show-word-limit />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="form.status" :active-value="1" :inactive-value="0" active-text="显示" inactive-text="隐藏" />
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
  name: 'AdminGardenActivities',
  data () {
    return { list: [], loading: false, dlg: false, saving: false, form: { id: 0, title: '', desc: '', status: 1 } }
  },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/garden-activities').then(r => {
        this.loading = false
        if (r.code === 0) this.list = r.data
      })
    },
    openDlg (row) {
      this.form = row ? { id: row.id, title: row.title, desc: row.desc, status: row.status } : { id: 0, title: '', desc: '', status: 1 }
      this.dlg = true
    },
    save () {
      if (!this.form.title) { this.$message.warning('请填写标题'); return }
      this.saving = true
      const body = { title: this.form.title, desc: this.form.desc, status: this.form.status }
      const done = () => { this.saving = false; this.dlg = false; this.load() }
      if (this.form.id) {
        api.put('/admin/garden-activities/' + this.form.id, body).then(r => { if (r.code === 0) done(); else this.$message.error(r.msg) })
      } else {
        api.post('/admin/garden-activities', body).then(r => { if (r.code === 0) done(); else this.$message.error(r.msg) })
      }
    },
    toggle (row) {
      api.put('/admin/garden-activities/' + row.id, { title: row.title, desc: row.desc, status: row.status === 1 ? 0 : 1 }).then(() => this.load())
    },
    del (row) {
      api.delete('/admin/garden-activities/' + row.id).then(() => this.load())
    }
  }
}
</script>
