<template>
  <div>
    <el-card shadow="never" class="box">
      <div class="toolbar">
        <el-select v-model="status" placeholder="全部状态" clearable style="width:140px" @change="search">
          <el-option label="待付款" :value="1" />
          <el-option label="待发货" :value="2" />
          <el-option label="待收货" :value="3" />
          <el-option label="已完成" :value="4" />
          <el-option label="已取消" :value="5" />
        </el-select>
        <div class="grow" />
        <span class="help-line">商店订单（对齐诺哈 商城管理）</span>
      </div>
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="订单号" width="90" />
        <el-table-column prop="goods_name" label="商品" min-width="160" show-overflow-tooltip />
        <el-table-column prop="buyer_nick" label="买家" min-width="110" />
        <el-table-column prop="seller_nick" label="卖家" min-width="110" />
        <el-table-column prop="amount" label="数量" width="70" align="center" />
        <el-table-column label="金额" width="100" align="center">
          <template slot-scope="{row}">{{ row.price * row.amount }} G币</template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template slot-scope="{row}">
            <el-tag :type="statusType(row.status)" size="mini">{{ statusName(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="下单时间" width="170"><template slot-scope="{row}">{{ fmt(row.created_at) }}</template></el-table-column>
      </el-table>
      <el-pagination background layout="total, prev, pager, next" :total="total" :page-size="size" :current-page="page"
                     @current-change="p => { page = p; load() }" style="margin-top:14px;text-align:right" />
    </el-card>
  </div>
</template>

<script>
import api from '../../api'

export default {
  name: 'AdminShopOrders',
  data () { return { list: [], total: 0, page: 1, size: 10, status: null, loading: false } },
  mounted () { this.load() },
  methods: {
    search () { this.page = 1; this.load() },
    load () {
      this.loading = true
      api.get('/admin/shop-orders', { params: { page: this.page, size: this.size, status: this.status || undefined } }).then(r => {
        this.loading = false
        if (r.code === 0) { this.list = r.data.list; this.total = r.data.total } else this.$message.error(r.msg)
      })
    },
    statusName (s) { return { 1: '待付款', 2: '待发货', 3: '待收货', 4: '已完成', 5: '已取消' }[s] || '未知' },
    statusType (s) { return { 1: 'warning', 2: 'primary', 3: 'primary', 4: 'success', 5: 'info' }[s] || 'info' },
    fmt (t) { return t ? String(t).slice(0, 19).replace('T', ' ') : '—' }
  }
}
</script>