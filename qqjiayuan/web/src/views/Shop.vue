<template>
  <div>
    <div class="name">道具商城<br></div>
    <div class="module-content">
      <span class="txt-fade">当前G币：<b style="color:#e05a00">{{ coins < 0 ? '--' : coins }}</b></span>
      <span class="txt-fade">　友友券：<b style="color:#e05a00">{{ youquan < 0 ? '--' : youquan }}</b></span>
      [<a href="javascript:;" @click="$router.push('/bag')">我的仓库</a>]
      [<a href="javascript:;" @click="$router.push('/youquan')">友友券中心</a>]
    </div>

    <!-- 分类导航 -->
    <div class="item catnav">
      <a v-for="c in cats" :key="c" href="javascript:;" @click="switchCat(c)" :class="{ cur: cat === c }">{{ c }}</a>
    </div>

    <!-- 商品列表 -->
    <div class="list">
      <div class="row" v-for="g in goods" :key="g.id" style="padding:4px 2px;border-bottom:1px dotted #dfe8f2">
        <img v-if="g.icon" class="gicon" :src="'/static/picture/' + g.icon" :alt="g.name">
        <b>{{ g.name }}</b>
        <span class="txt-fade">（{{ g.category }} · {{ g.price }}G币<template v-if="g.youquan_price"> / {{ g.youquan_price }}友友券</template>）</span>
        <div class="txt-fade">{{ g.desc }}</div>
        <span class="txt-fade">数量</span><input type="text" v-model.number="nums[g.id]" size="3" maxlength="3">
        <template v-if="g.youquan_price > 0">
          <input type="submit" value="G币购买" @click="buy(g, 'coins')">
          <input type="submit" value="友友券购买" @click="buy(g, 'youquan')">
        </template>
        <input v-else type="submit" value="购买" @click="buy(g, 'coins')">
      </div>
    </div>
    <div class="module-content" v-if="!goods.length"><span class="empty">该分类暂无商品</span></div>

    <p style="color:#c00;padding:0 5px" v-if="msg">{{ msg }}</p>
    <p style="color:#1a9e1a;padding:0 5px" v-if="okMsg">{{ okMsg }}</p>

    <div class="module-content txt-fade">鲜花类道具在帖子下方【送花】使用，其余道具到仓库中使用。标有「友友券」价格的商品可用友友券购买。</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Shop',
  data () { return { goods: [], all: [], cats: ['全部'], cat: '全部', coins: -1, youquan: -1, nums: {}, msg: '', okMsg: '' } },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/goods').then(r => {
        if (r.code === 0) {
          this.all = r.data.list || []
          this.goods = this.all
          this.cats = ['全部'].concat(r.data.categories || [])
          this.$forceUpdate()
        }
      })
      api.get('/auth/me').then(r => {
        if (r.code === 0) { this.coins = r.data.coins || 0; this.youquan = r.data.youquan || 0 }
      }).catch(() => {})
    },
    switchCat (c) {
      this.cat = c
      this.goods = c === '全部' ? this.all : this.all.filter(g => g.category === c)
    },
    buy (g, currency) {
      const num = parseInt(this.nums[g.id]) || 1
      this.msg = ''
      this.okMsg = ''
      api.post('/goods/' + g.id + '/buy', { num, currency }).then(r => {
        if (r.code === 0) {
          const payName = currency === 'youquan' ? '友友券' : 'G币'
          this.okMsg = '已用' + payName + '购买「' + r.data.name + '」×' + r.data.num + '，已放入仓库'
          this.coins = r.data.coins
          this.youquan = r.data.youquan
        } else this.msg = r.msg
      })
    }
  }
}
</script>

<style scoped>
.catnav a { margin-right: 8px; color: #0051A4; }
.catnav a.cur { color: #e05a00; font-weight: bold; }
.gicon { width: 30px; height: 30px; vertical-align: middle; margin-right: 3px; }
</style>