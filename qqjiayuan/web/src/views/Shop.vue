<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;商城<br></div>
    <div class="name">道具商城<br></div>

    <div class="module-content">
      我的G币：<b style="color:#e05a00">{{ coins < 0 ? '--' : coins }}</b>
      <span class="txt-fade">　友友券：<b style="color:#e05a00">{{ youquan < 0 ? '--' : youquan }}</b></span>
      [<a href="javascript:;" @click="$router.push('/bag')">我的仓库</a>]
      [<a href="javascript:;" @click="$router.push('/youquan')">友友券中心</a>]
    </div>

    <!-- 分类导航 -->
    <div class="module-title">分类：
      <a href="javascript:;" @click="switchCat('全部')" :class="{ cur: cat === '全部' }">全部</a>
      <template v-for="(c,i) in cats">
        <span v-if="c !== '全部'" :key="i">|<a href="javascript:;" @click="switchCat(c)" :class="{ cur: cat === c }">{{ c }}</a></span>
      </template>
    </div>

    <!-- 商品列表 -->
    <div class="list">
      <div class="row shop-row" v-for="g in goods" :key="g.id">
        <img v-if="g.icon" class="gicon" :src="'/static/picture/' + g.icon" :alt="g.name" @error="hideErr($event)">
        <span v-else class="gph">{{ g.category.slice(0, 1) }}</span>
        <b>{{ g.name }}</b> <em>[{{ g.category }}]</em><br>
        <span class="txt-fade">{{ g.desc }}</span>
        <div>
          价格：<b style="color:#e05a00">{{ g.price }}</b>G币<template v-if="g.youquan_price > 0"> / <b style="color:#e05a00">{{ g.youquan_price }}</b>友友券</template>
          <span class="txt-fade">　数量</span><input type="text" v-model.number="nums[g.id]" size="2" maxlength="3">
          <template v-if="g.youquan_price > 0">
            <input type="submit" value="G币购买" @click="buy(g, 'coins')">
            <input type="submit" value="友友券购买" @click="buy(g, 'youquan')">
          </template>
          <input v-else type="submit" value="购买" @click="buy(g, 'coins')">
        </div>
      </div>
    </div>
    <div class="module-content" v-if="!goods.length"><span class="empty">该分类暂无商品</span></div>

    <!-- 分页 -->
    <div v-if="totalPages > 1" class="pager">
      (第<b>{{ page }}</b>/{{ totalPages }}页/共{{ total }}件)
      <a href="javascript:;" v-if="page > 1" @click="load(page-1)">&lt;&lt;上一页</a>
      <a href="javascript:;" v-if="page < totalPages" @click="load(page+1)">下一页&gt;&gt;</a>
    </div>

    <p style="color:#c00;padding:0 5px" v-if="msg">{{ msg }}</p>
    <p style="color:#1a9e1a;padding:0 5px" v-if="okMsg">{{ okMsg }}</p>

    <div class="module-content txt-fade">鲜花类道具在帖子下方【送花】使用，其余道具到仓库中使用。标有「友友券」价格的商品可用友友券购买。</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Shop',
  data () {
    return {
      goods: [], cats: ['全部'], cat: '全部',
      coins: -1, youquan: -1,
      page: 1, total: 0, totalPages: 1,
      nums: {}, msg: '', okMsg: ''
    }
  },
  mounted () { this.load(1) },
  methods: {
    load (p) {
      this.page = p
      api.get('/goods', { params: { page: p, cat: this.cat } }).then(r => {
        if (r.code === 0) {
          this.goods = r.data.list || []
          this.cats = ['全部'].concat(r.data.categories || [])
          this.coins = r.data.coins
          this.youquan = r.data.youquan
          this.total = r.data.total || 0
          const size = r.data.size || 12
          this.totalPages = Math.max(1, Math.ceil(this.total / size))
        }
      })
    },
    switchCat (c) {
      this.cat = c
      this.load(1)
    },
    buy (g, currency) {
      const num = Math.max(1, Math.floor(parseInt(this.nums[g.id]) || 1))
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
    },
    hideErr (e) { e.target.style.display = 'none' }
  }
}
</script>

<style scoped>
.module-title a.cur { color: #e05a00; font-weight: bold; }
.shop-row { padding: 5px 3px; }
.shop-row .txt-fade { display: block; }
.shop-row input { margin-right: 2px; }
.gicon { width: 34px; height: 34px; vertical-align: middle; margin-right: 4px; }
.gph { display: inline-block; width: 34px; height: 34px; line-height: 34px; text-align: center; vertical-align: middle; margin-right: 4px; background: #E3EEF8; border: 1px solid #9FC6EC; color: #0051A4; font-weight: bold; font-size: 15px; }
.pager { padding: 5px 3px; line-height: 1.6; }
</style>
