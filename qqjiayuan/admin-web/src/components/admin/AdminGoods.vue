<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增商品</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="商品" min-width="140" />
        <el-table-column prop="category" label="分类" width="100" />
        <el-table-column prop="price" label="金币" width="90" header-align="center" />
        <el-table-column label="状态" width="90" header-align="center">
          <template slot-scope="{row}"><el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '上架' : '下架' }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="desc" label="说明" min-width="160" show-overflow-tooltip />
        <el-table-column label="操作" width="270" header-align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain @click="openDlg(row)">编辑</el-button>
            <el-button size="mini" @click="toggle(row)">{{ row.status === 1 ? '下架' : '上架' }}</el-button>
            <el-button size="mini" type="danger" plain @click="del(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog :title="form.id ? '编辑商品' : '新增商品'" :visible.sync="dlg" width="540px" :close-on-click-modal="false">
      <el-form label-width="80px">
        <el-form-item label="名称"><el-input v-model.trim="form.name" maxlength="40" /></el-form-item>
        <el-form-item label="分类"><el-input v-model.trim="form.category" maxlength="20" placeholder="装扮/道具/特权/其他" /></el-form-item>
        <el-form-item label="金币"><el-input-number v-model="form.price" :min="0" /></el-form-item>
        <el-form-item label="说明"><el-input type="textarea" v-model="form.desc" :rows="2" maxlength="200" /></el-form-item>
        <el-form-item label="状态"><el-switch v-model="form.status" :active-value="1" :inactive-value="0" active-text="上架" inactive-text="下架" /></el-form-item>
      </el-form>
      <div slot="footer"><el-button @click="dlg = false">取 消</el-button><el-button type="primary" @click="save">保 存</el-button></div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminGoods',
  data () { return { list: [], loading: false, dlg: false, form: { id: 0, name: '', category: '', price: 0, desc: '', status: 1 } } },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/goods').then(r => { this.loading = false; if (r.code === 0) this.list = r.data })
    },
    openDlg (row) {
      this.form = row ? { id: row.id, name: row.name, category: row.category, price: row.price, desc: row.desc, status: row.status } : { id: 0, name: '', category: '', price: 0, desc: '', status: 1 }
      this.dlg = true
    },
    save () {
      if (!this.form.name) { this.$message.warning('请填写商品名'); return }
      const body = { name: this.form.name, category: this.form.category, price: this.form.price, desc: this.form.desc, status: this.form.status }
      if (this.form.id) api.put('/admin/goods/' + this.form.id, body).then(r => { if (r.code === 0) { this.dlg = false; this.load() } })
      else api.post('/admin/goods', body).then(r => { if (r.code === 0) { this.dlg = false; this.load() } })
    },
    toggle (row) {
      api.put('/admin/goods/' + row.id, { name: row.name, category: row.category, price: row.price, desc: row.desc, status: row.status === 1 ? 0 : 1 }).then(() => this.load())
    },
    del (row) {
      api.delete('/admin/goods/' + row.id).then(() => this.load())
    }
  }
}
</script>
