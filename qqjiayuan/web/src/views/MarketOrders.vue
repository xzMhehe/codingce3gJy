<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/market')">商店</a>&gt;我的订单</div>
    <div class="name">我的订单<br></div>
    <div class="module-title">
      <a href="javascript:;" :style="{ fontWeight: role === 'buy' ? 'bold' : 'normal' }" @click="role='buy';load()">我买到的</a> |
      <a href="javascript:;" :style="{ fontWeight: role === 'sell' ? 'bold' : 'normal' }" @click="role='sell';load()">我卖出的</a>
    </div>
    <div class="list">
      <div v-for="o in orders" :key="o.id" class="row">
        <a href="javascript:;" @click="$router.push('/market/'+o.goods_id)">{{ o.goods_name }}</a> x{{ o.amount }}
        <b style="color:#e05a00">{{ o.price * o.amount }}</b>金币
        <span class="txt-fade">[{{ statusName(o.status) }}]</span>
        <template v-if="role === 'buy'">
          <a href="javascript:;" v-if="o.status === 1" @click="pay(o)">[付款]</a>
          <a href="javascript:;" v-if="o.status === 1 || o.status === 2" @click="cancel(o)">[取消]</a>
          <a href="javascript:;" v-if="o.status === 3" @click="receive(o)">[收货]</a>
          <a href="javascript:;" v-if="o.status === 4" @click="$router.push('/market/'+o.goods_id)">[去评价]</a>
        </template>
        <template v-if="role === 'sell'">
          <a href="javascript:;" v-if="o.status === 2" @click="ship(o)">[发货]</a>
        </template>
        <br>
        <span class="txt-fade">与{{ otherNick(o) }} · {{ fmt(o.created_at) }}</span><br>
      </div>
    </div>
    <span v-if="!orders.length" class="empty">暂无订单</span>
    <p v-if="tip" style="color:#e05a00;padding:3px 5px">{{ tip }}</p>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'MarketOrders',
  data () { return { orders: [], role: 'buy', tip: '' } },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/shop/orders', { params: { role: this.role } }).then(r => {
        if (r.code === 0) this.orders = r.data || []
      })
    },
    pay (o) { api.post('/shop/orders/' + o.id + '/pay').then(r => { if (r.code === 0) this.load(); else this.tip = r.msg }) },
    ship (o) { api.post('/shop/orders/' + o.id + '/ship').then(() => this.load()) },
    receive (o) { api.post('/shop/orders/' + o.id + '/receive').then(() => this.load()) },
    cancel (o) { api.post('/shop/orders/' + o.id + '/cancel').then(() => this.load()) },
    otherNick (o) { return this.role === 'buy' ? o.other_nick : o.other_nick },
    statusName (s) { return { 1: '待付款', 2: '待发货', 3: '待收货', 4: '已完成', 5: '已取消' }[s] || '未知' },
    fmt (t) { return t ? t.slice(0, 16).replace('T', ' ') : '' }
  }
}
</script>