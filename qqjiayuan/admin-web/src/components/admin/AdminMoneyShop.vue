<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <div class="grow" />
        <el-button type="primary" icon="el-icon-plus" @click="openDlg(null)">新增货币商品</el-button>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column label="卖出" width="130" header-align="center">
          <template slot-scope="{row}">{{ row.money }}（{{ cname(row.mtype) }}）</template>
        </el-table-column>
        <el-table-column label="单价" width="130" header-align="center">
          <template slot-scope="{row}">{{ row.price }}（{{ cname(row.ptype) }}）</template>
        </el-table-column>
        <el-table-column prop="stock" label="库存" width="80" header-align="center" />
        <el-table-column prop="sales" label="销量" width="80" header-align="center" />
        <el-table-column label="结束时间" width="150" header-align="center">
          <template slot-scope="{row}">{{ fmt(row.end_time) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="80" header-align="center">
          <template slot-scope="{row}"><el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '上架' : '下架' }}</el-tag></template>
        </el-table-column>
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

    <el-dialog :title="form.id ? '编辑货币商品' : '新增货币商品'" :visible.sync="dlg" width="560px" :close-on-click-modal="false">
      <el-form label-width="100px">
        <el-form-item label="名称"><el-input v-model.trim="form.name" maxlength="40" /></el-form-item>
        <el-form-item label="卖出货币">
          <el-select v-model="form.mtype" style="width:200px">
            <el-option v-for="m in moneyTypes" :key="m.value" :label="m.label" :value="m.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="卖出数量"><el-input-number v-model="form.money" :min="1" /></el-form-item>
        <el-form-item label="支付货币">
          <el-select v-model="form.ptype" style="width:200px">
            <el-option v-for="m in moneyTypes" :key="m.value" :label="m.label" :value="m.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="单价"><el-input-number v-model="form.price" :min="0" /></el-form-item>
        <el-form-item label="库存"><el-input-number v-model="form.stock" :min="0" /></el-form-item>
        <el-form-item label="销量"><el-input-number v-model="form.sales" :min="0" /></el-form-item>
        <el-form-item label="结束时间">
          <el-date-picker v-model="form.end_time" type="datetime" placeholder="留空=长期有效"
                          value-format="yyyy-MM-ddTHH:mm:ssZ" style="width:220px" />
        </el-form-item>
        <el-form-item label="状态"><el-switch v-model="form.status" :active-value="1" :inactive-value="0" active-text="上架" inactive-text="下架" /></el-form-item>
      </el-form>
      <div slot="footer"><el-button @click="dlg = false">取 消</el-button><el-button type="primary" @click="save">保 存</el-button></div>
    </el-dialog>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminMoneyShop',
  data () {
    return {
      list: [], total: 0, page: 1, size: 10, loading: false, dlg: false,
      moneyTypes: [
        { value: 'coins', label: 'G币' },
        { value: 'yuanbao', label: '元宝' },
        { value: 'jinzuan', label: '金钻' },
        { value: 'youquan', label: '友友券' }
      ],
      form: { id: 0, name: '', mtype: 'coins', money: 1, ptype: 'coins', price: 0, stock: 0, sales: 0, end_time: null, status: 1 }
    }
  },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      api.get('/admin/money-shops', { params: { page: this.page, size: this.size } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.list = r.data.list || []
          this.total = r.data.total || 0
        }
      })
    },
    cname (t) {
      const m = { coins: 'G币', yuanbao: '元宝', jinzuan: '金钻', youquan: '友友券' }
      return m[t] || t
    },
    fmt (t) {
      if (!t) return '长期'
      const d = new Date(t)
      return d.getFullYear() + '/' + (d.getMonth() + 1) + '/' + d.getDate()
    },
    openDlg (row) {
      this.form = row ? { id: row.id, name: row.name, mtype: row.mtype, money: row.money, ptype: row.ptype, price: row.price, stock: row.stock || 0, sales: row.sales || 0, end_time: row.end_time || null, status: row.status } : { id: 0, name: '', mtype: 'coins', money: 1, ptype: 'coins', price: 0, stock: 0, sales: 0, end_time: null, status: 1 }
      this.dlg = true
    },
    save () {
      if (!this.form.name) { this.$message.warning('请填写名称'); return }
      if (this.form.mtype === this.form.ptype) { this.$message.warning('卖出货币与支付货币不能相同'); return }
      const body = { name: this.form.name, mtype: this.form.mtype, money: this.form.money, ptype: this.form.ptype, price: this.form.price, stock: this.form.stock, sales: this.form.sales, end_time: this.form.end_time, status: this.form.status }
      if (this.form.id) api.put('/admin/money-shops/' + this.form.id, body).then(r => { if (r.code === 0) { this.dlg = false; this.load() } })
      else api.post('/admin/money-shops', body).then(r => { if (r.code === 0) { this.dlg = false; this.load() } })
    },
    toggle (row) {
      api.put('/admin/money-shops/' + row.id, { name: row.name, mtype: row.mtype, money: row.money, ptype: row.ptype, price: row.price, stock: row.stock || 0, sales: row.sales || 0, end_time: row.end_time || null, status: row.status === 1 ? 0 : 1 }).then(() => this.load())
    },
    del (row) {
      this.$confirm('确定删除「' + row.name + '」吗？', '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/money-shops/' + row.id).then(() => this.load())
      }).catch(() => {})
    }
  }
}
</script>
