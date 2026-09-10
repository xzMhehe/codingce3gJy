<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增商品</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column label="图标" width="70" header-align="center">
          <template slot-scope="{row}">
            <img v-if="row.icon" :src="'/static/picture/' + row.icon" style="width:28px;height:28px;vertical-align:middle" :alt="row.name">
            <span v-else class="txt-fade">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="商品" min-width="120" />
        <el-table-column prop="category" label="分类" width="90" />
        <el-table-column prop="price" label="G币" width="80" header-align="center" />
        <el-table-column prop="youquan_price" label="友友券" width="80" header-align="center">
          <template slot-scope="{row}">{{ row.youquan_price || '—' }}</template>
        </el-table-column>
        <el-table-column prop="stock" label="库存" width="70" header-align="center" />
        <el-table-column prop="sales" label="销量" width="70" header-align="center" />
        <el-table-column prop="sort" label="排序" width="70" header-align="center" />
        <el-table-column label="状态" width="80" header-align="center">
          <template slot-scope="{row}"><el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '上架' : '下架' }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="desc" label="说明" min-width="150" show-overflow-tooltip />
        <el-table-column label="操作" width="260" header-align="center">
          <template slot-scope="{row}">
            <el-button size="mini" type="primary" plain @click="openDlg(row)">编辑</el-button>
            <el-button size="mini" @click="toggle(row)">{{ row.status === 1 ? '下架' : '上架' }}</el-button>
            <el-button size="mini" type="danger" plain @click="del(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total"
                     :page-size="size" :current-page="page"
                     @current-change="p => { page = p; load() }"
                     style="margin-top:14px;text-align:right" />
    </el-card>

    <el-dialog :title="form.id ? '编辑商品' : '新增商品'" :visible.sync="dlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="80px">
        <el-form-item label="名称"><el-input v-model.trim="form.name" maxlength="40" /></el-form-item>
        <el-form-item label="分类">
          <el-select v-model="form.category" style="width:200px">
            <el-option v-for="c in categories" :key="c" :label="c" :value="c" />
          </el-select>
        </el-form-item>
        <el-form-item label="金币"><el-input-number v-model="form.price" :min="0" /></el-form-item>
        <el-form-item label="友友券"><el-input-number v-model="form.youquan_price" :min="0" /></el-form-item>
        <el-form-item label="排序"><el-input-number v-model="form.sort" :min="0" /></el-form-item>
        <el-form-item label="图标">
          <el-input v-model.trim="form.icon" maxlength="50" placeholder="static/picture 下的文件名，如 flower_rose.gif">
            <template slot="prepend">/static/picture/</template>
          </el-input>
          <img v-if="form.icon" :src="'/static/picture/' + form.icon" style="width:32px;height:32px;margin-top:6px;vertical-align:middle" :alt="form.icon">
        </el-form-item>
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
  data () { return { list: [], total: 0, page: 1, size: 10, categories: ['鲜花', '道具', '装扮', '特权'], loading: false, dlg: false, form: { id: 0, name: '', category: '道具', icon: '', price: 0, youquan_price: 0, sort: 0, desc: '', status: 1 } } },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/goods', { params: { page: this.page, size: this.size } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list || []
          this.total = r.data.total || 0
          if (r.data.categories && r.data.categories.length) {
            const merged = this.categories.concat(r.data.categories.filter(c => this.categories.indexOf(c) < 0))
            this.categories = merged
          }
        }
      })
    },
    openDlg (row) {
      this.form = row ? { id: row.id, name: row.name, category: row.category, icon: row.icon || '', price: row.price, youquan_price: row.youquan_price || 0, stock: row.stock || 0, sales: row.sales || 0, end_time: row.end_time || null, sort: row.sort || 0, desc: row.desc, status: row.status } : { id: 0, name: '', category: '道具', icon: '', price: 0, youquan_price: 0, stock: 0, sales: 0, end_time: null, sort: 0, desc: '', status: 1 }
      this.dlg = true
    },
    save () {
      if (!this.form.name) { this.$message.warning('请填写商品名'); return }
      const body = { name: this.form.name, category: this.form.category, icon: this.form.icon, price: this.form.price, youquan_price: this.form.youquan_price, stock: this.form.stock, sales: this.form.sales, end_time: this.form.end_time, sort: this.form.sort, desc: this.form.desc, status: this.form.status }
      if (this.form.id) api.put('/admin/goods/' + this.form.id, body).then(r => { if (r.code === 0) { this.dlg = false; this.load() } })
      else api.post('/admin/goods', body).then(r => { if (r.code === 0) { this.dlg = false; this.load() } })
    },
    toggle (row) {
      api.put('/admin/goods/' + row.id, { name: row.name, category: row.category, icon: row.icon || '', price: row.price, youquan_price: row.youquan_price || 0, stock: row.stock || 0, sales: row.sales || 0, end_time: row.end_time || null, sort: row.sort || 0, desc: row.desc, status: row.status === 1 ? 0 : 1 }).then(() => this.load())
    },
    del (row) {
      this.$confirm('确定删除「' + row.name + '」吗？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/goods/' + row.id).then(() => this.load())
      }).catch(() => {})
    }
  }
}
</script>
