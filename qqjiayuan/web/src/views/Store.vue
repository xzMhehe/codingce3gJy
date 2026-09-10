<template>
  <div>
    <!-- 面包屑（复刻诺哈 wap/shop：地盘>财务>店铺街） -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;<a href="javascript:;" @click="$router.push('/box')">用户中心</a>&gt;<a href="javascript:;" @click="go('street')">店铺街</a><template v-if="view === 'shop'">&gt;{{ shop.name }}</template><template v-else-if="view === 'goods'">&gt;商品详情</template><template v-else-if="view === 'buy'">&gt;购买</template><template v-else-if="view === 'orders'">&gt;我的订单</template><template v-else-if="view === 'seller'">&gt;卖家中心</template><br>
    </div>
    <div class="name">店铺街<br></div>

    <!-- 顶部功能行（对齐 shop/index.asp：我的店铺/订单/卖家中心） -->
    <div class="module-content" v-if="view === 'street'">
      <a href="javascript:;" @click="go('my')">我的店铺</a>.<a href="javascript:;" @click="go('orders')">我的订单</a>.<a href="javascript:;" @click="go('seller')">卖家中心</a>
      　<span class="txt-fade">共{{ total }}家店铺</span><br>
    </div>

    <!-- 店铺街列表（对齐 index.asp 店铺列表+搜索） -->
    <template v-if="view === 'street'">
      <div class="module-content">
        搜索店铺：<input type="text" v-model.trim="wd" size="12"> <input type="submit" value="搜索" @click="page = 1; load()">
      </div>
      <div class="module-content">
        <span v-for="(s, i) in shops" :key="s.id">{{ i + 1 + (page - 1) * 10 }}.<a href="javascript:;" @click="goShop(s.id)">{{ s.name }}</a>
          <span class="txt-fade">掌柜:{{ s.nick }} 商品:{{ s.goods_n }}件 开店:{{ s.created_at }}</span><br></span>
        <span v-if="!shops.length" class="empty">暂无店铺。</span>
      </div>
      <div class="module-content" v-if="totalPages > 1">
        <a href="javascript:;" v-if="page < totalPages" @click="load(page + 1)">下页</a><span v-if="page < totalPages && page > 1">.</span><a href="javascript:;" v-if="page > 1" @click="load(page - 1)">上页</a>
        (第<b>{{ page }}</b>/{{ totalPages }}页/共{{ total }}家)<br>
      </div>
    </template>

    <!-- 我的店铺 / 开店（对齐 seller/index.asp） -->
    <template v-else-if="view === 'my'">
      <template v-if="myShop.exists">
        <div class="module-content">
          店铺名：<b>{{ myShop.shop.name }}</b><br>
          开店时间：{{ fmt(myShop.shop.created_at) }}<br>
          [<a href="javascript:;" @click="goShop(myShop.shop.id)">进入店铺</a>].[<a href="javascript:;" @click="go('seller')">卖家中心</a>]<br>
        </div>
      </template>
      <template v-else>
        <div class="module-content">
          您还没有开店，开设自己的店铺就能上架商品出售啦！<br>
          店铺名称：<input type="text" v-model.trim="shopName" size="18" maxlength="30"><br>
          <input type="submit" value="立即开店" @click="createShop">
        </div>
      </template>
    </template>

    <!-- 店铺详情（对齐 shop.asp） -->
    <template v-else-if="view === 'shop' && shop.id">
      <div class="module-content">
        <b>{{ shop.name }}</b><br>
        掌柜：<a href="javascript:;" @click="$router.push('/space/' + shop.user_id)">{{ shop.nick }}</a>　开店：{{ shop.created_at }}<br>
      </div>
      <div class="name">在售商品({{ shop.goods.length }})<br></div>
      <div class="module-content">
        <span v-for="(g, i) in shop.goods" :key="g.id">{{ i + 1 }}.<a href="javascript:;" @click="goGoods(g.id)">{{ g.name }}</a> <b style="color:#e05a00">{{ g.price }}G</b> <span class="txt-fade">库存{{ g.amount }} 销量{{ g.sales }}</span><br></span>
        <span v-if="!shop.goods.length" class="empty">本店暂无在售商品。</span>
      </div>
    </template>

    <!-- 商品详情（对齐 image_list 详情+评论） -->
    <template v-else-if="view === 'goods' && g.goods">
      <div class="module-content">
        <b>{{ g.goods.name }}</b><br>
        价格：<b style="color:#e05a00">{{ g.goods.price }}</b>(G币)　库存：{{ g.goods.amount }}　销量：{{ g.goods.sales }}<br>
        卖家：<a href="javascript:;" @click="goShop(g.shop_id)">{{ g.shop_name }}</a>({{ g.seller_nick }})<br>
        <a href="javascript:;" @click="go('buy')">购买</a><br>
      </div>
      <div class="module-content">----------<br><span class="txt-fade">{{ g.goods.intro }}</span></div>
      <div class="name">商品评价({{ comments.length }})<br></div>
      <div class="module-content">
        <div v-for="m in comments" :key="m.id" style="padding:3px 0">
          <b>{{ m.nick }}</b>：<span :style="{color: dcolor(m.d_type)}">{{ dname(m.d_type) }}</span> {{ m.content }} <span class="txt-fade">({{ m.time_txt }})</span><br>
          <span v-if="m.reply" class="txt-fade">掌柜回复：{{ m.reply }}<br></span>
        </div>
        <span v-if="!comments.length" class="empty">暂无评价。</span>
      </div>
    </template>

    <!-- 购买（对齐 buy.asp：数量+支付密码） -->
    <template v-else-if="view === 'buy' && g.goods">
      <div class="module-content">
        【购买商品】<br>
        <b>{{ g.goods.name }}</b><br>
        价格：<b style="color:#e05a00">{{ g.goods.price }}</b>(G币)　库存：{{ g.goods.amount }}<br>
        购买数量：<input type="text" v-model.number="buyNum" size="4"><br>
        合计：<b style="color:#e05a00">{{ g.goods.price * buyNum }}</b>G币（我的G币：{{ coins }}）<br>
        <template v-if="hasPaypass">
          支付密码：<input type="password" v-model.trim="payPass" size="12"><br>
          <input type="submit" value="确认购买" @click="buy">
        </template>
        <template v-else>您还未设置支付密码，请先到<a href="javascript:;" @click="$router.push('/security')">安全中心</a>设置！</template>
      </div>
    </template>

    <!-- 我的订单（对齐 trade_list.asp：买入/卖出 Tab） -->
    <template v-else-if="view === 'orders'">
      <div class="module-content">
        <a :style="oType === 'buy' ? 'font-weight:bold' : ''" href="javascript:;" @click="oType = 'buy'; loadOrders()">我买入的</a>.
        <a :style="oType === 'sell' ? 'font-weight:bold' : ''" href="javascript:;" @click="oType = 'sell'; loadOrders()">我卖出的</a>
      </div>
      <div class="module-content">
        <div v-for="o in orders" :key="o.id" style="padding:4px 0">
          #{{ o.id }} {{ o.goods_name }}×{{ o.amount }} 合计<b style="color:#e05a00">{{ o.price * o.amount }}G</b><br>
          <span v-if="oType === 'buy'">卖家:{{ o.seller_nick }}</span><span v-else>买家:{{ o.buyer_nick }}</span>
          状态:<b :style="{color: ocolor(o.status)}">{{ oname(o.status) }}</b>
          <span class="txt-fade">{{ fmt(o.created_at) }}</span><br>
          <template v-if="oType === 'buy'">
            <a v-if="o.status === 2" href="javascript:;" @click="oAct(o.id, 'cancel')">取消订单(退款)</a>
            <a v-if="o.status === 3" href="javascript:;" @click="oAct(o.id, 'receive')">确认收货</a>
            <a v-if="o.status === 4 && !o.commented" href="javascript:;" @click="openComment(o)">评价</a><br>
          </template>
          <template v-else>
            <a v-if="o.status === 2" href="javascript:;" @click="oAct(o.id, 'deliver')">发货</a><br>
          </template>
        </div>
        <span v-if="!orders.length" class="empty">暂无订单。</span>
      </div>
    </template>

    <!-- 卖家中心（对齐 seller/：商品管理+发货） -->
    <template v-else-if="view === 'seller'">
      <div class="module-content">
        <a href="javascript:;" @click="go('my')">我的店铺</a>.<a href="javascript:;" @click="go('orders')">订单发货</a><br>
      </div>
      <div class="name">上架新商品<br></div>
      <div class="module-content">
        商品名称：<input type="text" v-model.trim="gForm.name" size="18" maxlength="60"><br>
        价格(G币)：<input type="text" v-model.number="gForm.price" size="8">　库存：<input type="text" v-model.number="gForm.amount" size="6"><br>
        商品介绍：<input type="text" v-model.trim="gForm.intro" size="24" maxlength="200"><br>
        <input type="submit" value="上 架" @click="addGoods">
      </div>
      <div class="name">我的商品({{ myGoods.length }})<br></div>
      <div class="module-content">
        <div v-for="g2 in myGoods" :key="g2.id" style="padding:3px 0">
          {{ g2.name }} <b style="color:#e05a00">{{ g2.price }}G</b> 库存{{ g2.amount }} 销量{{ g2.sales }}
          <span :class="g2.status === 1 ? '' : 'txt-fade'">{{ g2.status === 1 ? '上架中' : '已下架' }}</span><br>
          [<a href="javascript:;" @click="toggleGoods(g2)">{{ g2.status === 1 ? '下架' : '上架' }}</a>].[<a href="javascript:;" @click="delGoods(g2)">删除</a>]
        </div>
        <span v-if="!myGoods.length" class="empty">还没有商品，先上架一件吧。</span>
      </div>
    </template>

    <!-- 评价弹层 -->
    <div class="mask" v-if="cBox" @click.self="cBox = false">
      <div class="panel">
        <div class="name">评价订单 #{{ cOrder.id }} {{ cOrder.goods_name }}</div>
        <div class="row">
          评价类型：<select v-model.number="cForm.d_type">
            <option :value="1">好评</option><option :value="2">中评</option><option :value="3">差评</option>
          </select><br/>
          <textarea v-model.trim="cForm.content" rows="3" style="width:96%" maxlength="300" placeholder="评价内容（300字内）"></textarea><br/>
        </div>
        <a href="javascript:;" @click="submitComment">[提交评价]</a> <a href="javascript:;" @click="cBox = false">[取消]</a>
      </div>
    </div>

    <!-- 提示弹层 -->
    <div class="mask" v-if="tipBox" @click.self="tipBox = false">
      <div class="panel">
        <div class="name">提示</div>
        <div class="row">{{ tipMsg }}</div>
        <a href="javascript:;" @click="tipBox = false">[确 定]</a>
      </div>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Store',
  data () {
    return {
      view: 'street',
      shops: [], total: 0, page: 1, size: 10, wd: '',
      myShop: { exists: false }, shopName: '',
      shop: {}, g: {}, comments: [],
      buyNum: 1, payPass: '', hasPaypass: false, coins: 0,
      oType: 'buy', orders: [],
      myGoods: [], gForm: { name: '', price: 0, amount: 1, intro: '' },
      cBox: false, cOrder: {}, cForm: { d_type: 1, content: '' },
      tipBox: false, tipMsg: ''
    }
  },
  computed: {
    totalPages () { return Math.max(1, Math.ceil(this.total / this.size)) }
  },
  mounted () { this.boot() },
  methods: {
    tip (msg) { this.tipMsg = msg; this.tipBox = true },
    go (v, id) {
      this.view = v
      if (v === 'street') this.load()
      if (v === 'my') this.loadMy()
      if (v === 'orders') { this.loadOrders() }
      if (v === 'seller') this.loadMyGoods()
    },
    boot () {
      this.load()
      api.get('/me/paypass').then(r => { this.hasPaypass = !!(r.code === 0 && r.data && r.data.has_paypass) })
      api.get('/me').then(r => { if (r.code === 0) this.coins = r.data.coins })
    },
    fmt (t) { return t ? String(t).slice(0, 16).replace('T', ' ') : '' },
    // ---- 店铺街 ----
    load (p) {
      if (p) this.page = p
      api.get('/stores', { params: { page: this.page, wd: this.wd || undefined } }).then(r => {
        if (r.code === 0) { this.shops = r.data.list; this.total = r.data.total; this.page = r.data.page }
      })
    },
    // ---- 我的店铺/开店 ----
    loadMy () {
      api.get('/stores/my').then(r => {
        if (r.code === 0) this.myShop = r.data
        else this.myShop = { exists: false }
      })
    },
    createShop () {
      if (!this.shopName) { this.tip('请填写店铺名称'); return }
      api.post('/stores', { name: this.shopName }).then(r => {
        if (r.code === 0) { this.tip(r.data.msg); this.loadMy() } else this.tip(r.msg)
      })
    },
    // ---- 店铺/商品 ----
    goShop (id) {
      api.get('/stores/' + id).then(r => {
        if (r.code === 0) { this.shop = r.data; this.view = 'shop' } else this.tip(r.msg)
      })
    },
    goGoods (id) {
      api.get('/store-goods/' + id).then(r => {
        if (r.code === 0) {
          this.g = r.data
          this.view = 'goods'
          this.buyNum = 1
          api.get('/store-goods/' + id + '/comments').then(rr => { if (rr.code === 0) this.comments = rr.data })
        } else this.tip(r.msg)
      })
    },
    // ---- 购买 ----
    buy () {
      api.post('/store-orders', { goods_id: this.g.goods.id, amount: this.buyNum, pay_pass: this.payPass }).then(r => {
        if (r.code === 0) {
          this.payPass = ''
          this.tip(r.data.msg + '（剩余' + r.data.coins + 'G币）')
          this.coins = r.data.coins
          this.goOrders()
        } else this.tip(r.msg)
      })
    },
    goOrders () {
      this.oType = 'buy'
      this.view = 'orders'
      this.loadOrders()
    },
    // ---- 订单 ----
    loadOrders () {
      api.get('/store-orders', { params: { type: this.oType } }).then(r => {
        if (r.code === 0) this.orders = r.data
      })
    },
    oAct (id, act) {
      api.post('/store-orders/' + id + '/' + act).then(r => {
        this.tip(r.code === 0 ? r.data.msg : r.msg)
        this.loadOrders()
        api.get('/me').then(rr => { if (rr.code === 0) this.coins = rr.data.coins })
      })
    },
    oname (s) { return { 1: '待付款', 2: '待发货', 3: '待收货', 4: '已完成', 5: '已取消' }[s] || s },
    ocolor (s) { return { 1: '#909399', 2: '#e6a23c', 3: '#409eff', 4: '#67c23a', 5: '#909399' }[s] },
    // ---- 评价 ----
    openComment (o) {
      this.cOrder = o
      this.cForm = { d_type: 1, content: '' }
      this.cBox = true
    },
    submitComment () {
      if (!this.cForm.content) { this.tip('请填写评价内容'); return }
      api.post('/store-comments', { order_id: this.cOrder.id, d_type: this.cForm.d_type, content: this.cForm.content }).then(r => {
        if (r.code === 0) { this.cBox = false; this.tip(r.data.msg); this.loadOrders() } else this.tip(r.msg)
      })
    },
    dname (d) { return { 1: '好评', 2: '中评', 3: '差评' }[d] },
    dcolor (d) { return { 1: '#67c23a', 2: '#e6a23c', 3: '#f56c6c' }[d] },
    // ---- 卖家中心 ----
    loadMyGoods () {
      api.get('/store-goods-mine').then(r => { if (r.code === 0) this.myGoods = r.data })
    },
    addGoods () {
      if (!this.gForm.name) { this.tip('请填写商品名称'); return }
      api.post('/store-goods', this.gForm).then(r => {
        if (r.code === 0) { this.tip(r.data.msg); this.gForm = { name: '', price: 0, amount: 1, intro: '' }; this.loadMyGoods() } else this.tip(r.msg)
      })
    },
    toggleGoods (g) {
      api.put('/store-goods/' + g.id, { status: g.status === 1 ? 0 : 1 }).then(r => {
        if (r.code === 0) this.loadMyGoods()
      })
    },
    delGoods (g) {
      if (!confirm('确定删除商品「' + g.name + '」？')) return
      api.delete('/store-goods/' + g.id).then(r => {
        if (r.code === 0) { this.tip(r.data.msg); this.loadMyGoods() } else this.tip(r.msg)
      })
    }
  }
}
</script>
