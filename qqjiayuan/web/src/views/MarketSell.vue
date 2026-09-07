<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/market')">商店</a>&gt;我的店铺</div>
    <div class="name">我的店铺<br></div>

    <div v-if="!shop.id" class="module-content">
      你还没有店铺，开通后即可发布道具商品买卖。<br>
      <form @submit.prevent="openShop">
        店铺名称:<input type="text" v-model.trim="shopName" maxlength="30" size="15" />
        <input type="submit" value="开通店铺" />
      </form>
    </div>
    <div v-else>
      <div class="module-content">
        店铺:<b>{{ shop.name }}</b> <a href="javascript:;" @click="openShop">[改名]</a><br>
      </div>

      <div class="module-title">发布商品</div>
      <form @submit.prevent="publish">
        名称:<input type="text" v-model.trim="pForm.name" maxlength="60" size="15" /><br>
        分类:
        <select v-model.number="pForm.cat_id">
          <option :value="0">请选择</option>
          <option v-for="c in cats" :key="c.id" :value="c.id">{{ c.name }}</option>
        </select>
        价格(金币):<input type="number" v-model.number="pForm.price" min="1" size="5" />
        库存:<input type="number" v-model.number="pForm.amount" min="1" size="4" /><br>
        说明:<input type="text" v-model.trim="pForm.intro" maxlength="500" size="25" /><br>
        <input type="submit" value="发布" />
      </form>
      <p v-if="tip" style="color:#e05a00;padding:3px 5px">{{ tip }}</p>

      <div class="module-title">我的商品({{ goods.length }})</div>
      <div class="list">
        <div v-for="g in goods" :key="g.id" class="row">
          <a href="javascript:;" @click="$router.push('/market/'+g.id)">{{ g.name }}</a>
          价格<b style="color:#e05a00">{{ g.price }}</b> 库存{{ g.amount }} 销量{{ g.sales }}
          <a href="javascript:;" @click="toggle(g)">{{ g.status === 1 ? '[下架]' : '[上架]' }}</a>
          <a href="javascript:;" @click="delGoods(g.id)">[删除]</a><br>
        </div>
      </div>
      <span v-if="!goods.length" class="empty">还没有发布商品</span>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'MarketSell',
  data () {
    return {
      shop: {}, goods: [], cats: [], tip: '', shopName: '',
      pForm: { name: '', cat_id: 0, price: 1, amount: 1, intro: '' }
    }
  },
  mounted () {
    api.get('/market/categories').then(r => { if (r.code === 0) this.cats = r.data || [] })
    this.load()
  },
  methods: {
    load () {
      api.get('/shop/mine').then(r => {
        if (r.code === 0) { this.shop = r.data.shop || {}; this.goods = r.data.goods || [] }
      })
    },
    openShop () {
      if (!this.shopName) return
      api.post('/shop', { name: this.shopName }).then(r => {
        if (r.code === 0) { this.shopName = ''; this.load() }
      })
    },
    publish () {
      if (!this.pForm.name || !this.pForm.price || !this.pForm.amount) { this.tip = '请填写完整商品信息'; return }
      api.post('/market', this.pForm).then(r => {
        if (r.code === 0) { this.pForm = { name: '', cat_id: 0, price: 1, amount: 1, intro: '' }; this.tip = '发布成功'; this.load() }
        else this.tip = r.msg || '发布失败'
      })
    },
    toggle (g) {
      api.put('/market/' + g.id, { status: g.status === 1 ? 0 : 1 }).then(() => this.load())
    },
    delGoods (id) { api.delete('/market/' + id).then(() => this.load()) }
  }
}
</script>