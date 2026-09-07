<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;商店</div>
    <div class="name">商店<br></div>
    <div class="module-title">分类:
      <a href="javascript:;" :style="{ fontWeight: !catId ? 'bold' : 'normal' }" @click="catId=0;load(1)">全部</a>
      <template v-for="c in cats">
        <span :key="c.id"> | <a href="javascript:;" :style="{ fontWeight: catId === c.id ? 'bold' : 'normal' }" @click="catId=c.id;load(1)">{{ c.name }}</a></span>
      </template>
    </div>
    <div class="module-content">
      当前G币：<b style="color:#e05a00">{{ myCoins }}</b>
      <a href="javascript:;" @click="$router.push('/market/sell')">我的店铺</a>.<a href="javascript:;" @click="$router.push('/market/orders')">我的订单</a><br>
    </div>

    <div class="list">
      <div v-for="g in list" :key="g.id" class="row">
        <a href="javascript:;" @click="$router.push('/market/'+g.id)">{{ g.name }}</a><br>
        <span class="txt-fade">[{{ g.cat_name || '未分类' }}] 价格:<b style="color:#e05a00">{{ g.price }}</b>(金币) 库存{{ g.amount }} 销量{{ g.sales }} 卖家:{{ g.seller }}</span><br>
      </div>
    </div>
    <span v-if="!list.length" class="empty">还没有商品</span>

    <div v-if="totalPages > 1" class="pager">
      (第<b>{{ page }}</b>/{{ totalPages }}页/共{{ total }}件)
      <a href="javascript:;" v-if="page > 1" @click="load(page-1)">&lt;&lt;上一页</a>
      <a href="javascript:;" v-if="page < totalPages" @click="load(page+1)">下一页&gt;&gt;</a>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Market',
  data () {
    return { list: [], cats: [], catId: 0, page: 1, total: 0, totalPages: 1, myCoins: 0 }
  },
  computed: {
    user () { return this.$store.state.user || {} }
  },
  mounted () {
    this.myCoins = this.user.coins || 0
    api.get('/market/categories').then(r => { if (r.code === 0) this.cats = r.data || [] })
    this.load(1)
  },
  methods: {
    load (p) {
      this.page = p
      api.get('/market', { params: { page: p, cat_id: this.catId } }).then(r => {
        if (r.code === 0) {
          this.list = r.data.list || []
          this.total = r.data.total
          this.totalPages = Math.ceil(r.data.total / r.data.size) || 1
        }
      })
    }
  }
}
</script>