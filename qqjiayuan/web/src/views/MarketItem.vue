<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/market')">商店</a>&gt;商品</div>
    <div class="name" v-if="g">{{ g.name }}<br></div>
    <div v-if="g" class="module-content">
      货币种类:G币<br>
      销售价格:<b style="color:#e05a00">{{ g.price }}</b>(金币)<br>
      库存数量:{{ g.amount }}<br>
      评论数量:{{ g.comment }}<br>
      商品分类:{{ catName }}<br>
      商品销量:{{ g.sales }}<br>
      卖家:<a href="javascript:;" @click="$router.push('/user/'+seller.id)"><font :color="seller.color || '#004299'">{{ seller.nickname }}</font></a><br>
      商品说明:{{ g.intro || '暂无说明' }}<br>
    </div>
    <span v-else class="empty">商品不存在或已下架</span>

    <form v-if="g && isLogin && !isMine" @submit.prevent="buy">
      购买数量:<input type="number" v-model.number="amount" min="1" :max="g.amount" size="4" />
      <input type="submit" value="下单" />
    </form>
    <p v-if="tip" style="color:#e05a00;padding:3px 5px">{{ tip }}</p>

    <div class="module-title">成交记录</div>
    <div class="list" v-if="deals.length">
      <div v-for="d in deals" :key="d.id" class="row">
        <span class="txt-fade">{{ d.buyer }} {{ d.price * d.amount }}金币 x{{ d.amount }} {{ fmt(d.created_at) }}</span><br>
      </div>
    </div>
    <span v-else class="empty">还没有成交记录</span>

    <div class="module-title">商品评论</div>
    <div class="list" v-if="comments.length">
      <div v-for="cm in comments" :key="cm.id" class="row">
        [{{ dtypeName(cm.d_type) }}]{{ cm.user }}:{{ cm.content }} <span class="txt-fade">({{ fmt(cm.created_at) }})</span><br>
        <span v-if="cm.reply" class="txt-fade" style="padding-left:12px">↳ 店主:{{ cm.reply }}</span>
        <form v-if="isMine" @submit.prevent="reply(cm)">
          <input type="text" v-model.trim="cm._reply" maxlength="300" size="18" placeholder="店主回复" />
          <input type="submit" value="回复" />
        </form>
      </div>
    </div>
    <span v-else class="empty">还没有评论</span>

    <form v-if="pendingReview" @submit.prevent="comment">
      <div class="module-title">评价(你已收货)</div>
      星级:<select v-model.number="cForm.d_type">
        <option :value="1">好评</option>
        <option :value="2">中评</option>
        <option :value="3">差评</option>
      </select><br>
      内容:<input type="text" v-model.trim="cForm.content" maxlength="300" size="20" />
      <input type="submit" value="评价" />
    </form>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'MarketItem',
  data () {
    return {
      g: null, seller: {}, catName: '', deals: [], comments: [],
      amount: 1, tip: '', orderId: 0, pendingReview: false,
      cForm: { d_type: 1, content: '', order_id: 0 }
    }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    user () { return this.$store.state.user || {} },
    isMine () { return this.g && this.g.user_id === this.user.id }
  },
  mounted () { this.load() },
  methods: {
    load () {
      const id = this.$route.params.id
      api.get('/market/' + id).then(r => {
        if (r.code === 0) {
          this.g = r.data.goods
          this.seller = r.data.seller || {}
          this.catName = r.data.cat_name || '未分类'
          this.deals = r.data.deals || []
          this.comments = (r.data.comments || []).map(cm => ({ ...cm, _reply: '' }))
        }
      }).catch(() => {})
      // 是否有待评价订单
      api.get('/shop/orders').then(r => {
        if (r.code === 0) {
          const done = (r.data || []).find(o => o.goods_id === Number(id) && o.status === 4)
          if (done) { this.pendingReview = true; this.orderId = done.id; this.cForm.order_id = done.id }
        }
      }).catch(() => {})
    },
    buy () {
      if (!this.amount || this.amount < 1) { this.tip = '请填写购买数量'; return }
      api.post('/market/' + this.g.id + '/order', { amount: this.amount }).then(r => {
        if (r.code === 0) {
          this.tip = ''
          if (confirm('下单成功！共需 ' + r.data.total + ' G币，立即付款？')) {
            api.post('/shop/orders/' + r.data.id + '/pay').then(pr => {
              if (pr.code === 0) { this.tip = '付款成功，等待卖家发货'; this.load() }
              else this.tip = pr.msg || '付款失败'
            })
          } else {
            this.$router.push('/market/orders')
          }
        } else this.tip = r.msg || '下单失败'
      }).catch(() => { this.tip = '下单失败，请稍后再试' })
    },
    comment () {
      if (!this.cForm.content) { this.tip = '请填写评价内容'; return }
      api.post('/market/' + this.g.id + '/comments', this.cForm).then(r => {
        if (r.code === 0) { this.tip = '评价成功'; this.pendingReview = false; this.load() }
        else this.tip = r.msg || '评价失败'
      })
    },
    reply (cm) {
      if (!cm._reply) return
      api.post('/shop/comments/' + cm.id + '/reply', { content: cm._reply }).then(r => {
        if (r.code === 0) this.load()
      })
    },
    dtypeName (t) { return { 1: '好评', 2: '中评', 3: '差评' }[t] || '好评' },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t); const p = n => (n < 10 ? '0' + n : '' + n)
      return d.getMonth() + 1 + '/' + d.getDate() + ' ' + p(d.getHours()) + ':' + p(d.getMinutes())
    }
  }
}
</script>