<template>
  <div>
    <div class="name">道具商城<br></div>
    <div class="module-content">
      <span class="txt-fade">当前G币：<b style="color:#e05a00">{{ coins < 0 ? '--' : coins }}</b></span>
      [<a href="javascript:;" @click="$router.push('/bag')">我的仓库</a>]
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
        <span class="txt-fade">（{{ g.category }} · {{ g.price }}G币）</span>
        <div class="txt-fade">{{ g.desc }}</div>
        <span class="txt-fade">数量</span><input type="text" v-model.number="nums[g.id]" size="3" maxlength="3">
        <input type="submit" value="购买" @click="buy(g)">
      </div>
    </div>
    <div class="module-content" v-if="!goods.length"><span class="empty">该分类暂无商品</span></div>

    <p style="color:#c00;padding:0 5px" v-if="msg">{{ msg }}</p>
    <p style="color:#1a9e1a;padding:0 5px" v-if="okMsg">{{ okMsg }}</p>

    <div class="module-content txt-fade">鲜花类道具在帖子下方【送花】使用，其余道具到仓库中使用。</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Shop',
  data () { return { goods: [], all: [], cats: ['全部'], cat: '全部', coins: -1, nums: {}, msg: '', okMsg: '' } },
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
      api.get('/auth/me').then(r => { if (r.code === 0) this.coins = r.data.coins || 0 }).catch(() => {})
    },
    switchCat (c) {
      this.cat = c
      this.goods = c === '全部' ? this.all : this.all.filter(g => g.category === c)
    },
    buy (g) {
      const num = parseInt(this.nums[g.id]) || 1
      this.msg = ''
      this.okMsg = ''
      api.post('/goods/' + g.id + '/buy', { num }).then(r => {
        if (r.code === 0) {
          this.okMsg = '已购买「' + r.data.name + '」×' + r.data.num + '，已放入仓库'
          this.coins = r.data.coins
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
