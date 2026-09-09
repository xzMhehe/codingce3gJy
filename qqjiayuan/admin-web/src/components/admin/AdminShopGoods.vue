<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-input v-model.number="userId" placeholder="按卖家号码过滤" clearable style="width:200px" @keyup.enter.native="search" @clear="search" />
        <el-button type="primary" icon="el-icon-search" @click="search">查询</el-button>
        <div class="grow" />
        <span class="help-line">全部商品（含下架）</span>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="70" />
        <el-table-column prop="name" label="商品名" min-width="180" show-overflow-tooltip />
        <el-table-column prop="seller" label="卖家" min-width="110" />
        <el-table-column prop="price" label="价格" width="90" align="center" />
        <el-table-column prop="amount" label="库存" width="80" align="center" />
        <el-table-column prop="sales" label="销量" width="80" align="center" />
        <el-table-column prop="clicks" label="浏览" width="80" align="center" />
        <el-table-column label="状态" width="90">
          <template slot-scope="{row}">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="mini">{{ row.status === 1 ? '上架' : '下架' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="170" align="center">
          <template slot-scope="{row}">
            <el-button size="mini" @click="toggle(row)">{{ row.status === 1 ? '下架' : '上架' }}</el-button>
            <el-button size="mini" type="danger" plain @click="del(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page"
                     @current-change="p => { page = p; load() }" style="margin-top:14px;text-align:right" />
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminShopGoods',
  data () { return { list: [], total: 0, page: 1, size: 10, userId: '', loading: false } },
  mounted () { this.load() },
  methods: {
    search () { this.page = 1; this.load() },
    load () {
      this.loading = true
      api.get('/admin/shop-goods', { params: { page: this.page, size: this.size, user_id: this.userId || undefined } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    toggle (row) {
      api.put('/admin/shop-goods/' + row.id + '/status', { status: row.status === 1 ? 0 : 1 }).then(r => {
        if (r.code === 0) { this.$message.success('已更新'); this.load() } else this.$message.error(r.msg)
      })
    },
    del (row) {
      this.$confirm(`删除商品「${row.name}」？`, '提示', { type: 'warning' }).then(() => {
        api.delete('/admin/shop-goods/' + row.id).then(r => {
          if (r.code === 0) { this.$message.success('已删除'); this.load() } else this.$message.error(r.msg)
        })
      }).catch(() => {})
    },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>