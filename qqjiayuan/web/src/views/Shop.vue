<template>
  <div>
    <div class="module-title">道具商城</div>
    <div class="module-content"><span class="txt-fade">当前金币：<b style="color:#e05a00">{{ coins }}</b>，社区小道具，自由选购。</span></div>

    <div class="list">
      <div v-for="g in goods" :key="g.id" class="row">
        <a href="javascript:;" @click="buy(g)">{{ g.name }}</a>
        <span class="txt-fade">（{{ g.category }} · {{ g.price }}金币）</span>
        <div class="txt-fade">{{ g.desc }}</div>
        [<a href="javascript:;" @click="buy(g)">购买</a>]
      </div>
    </div>
    <div class="module-content" v-if="!goods.length"><span class="empty">暂无商品</span></div>

    <p style="color:#c00;padding:0 5px" v-if="msg">{{ msg }}</p>
    <p style="color:#1a9e1a;padding:0 5px" v-if="okMsg">{{ okMsg }}</p>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Shop',
  data () { return { goods: [], coins: 0, msg: '', okMsg: '' } },
  mounted () {
    api.get('/goods').then(r => { if (r.code === 0) this.goods = r.data })
    api.get('/auth/me').then(r => { if (r.code === 0) this.coins = r.data.coins || 0 }).catch(() => {})
  },
  methods: {
    buy (g) {
      api.post('/goods/' + g.id + '/buy').then(r => {
        if (r.code === 0) { this.okMsg = '已购买「' + g.name + '」'; this.coins = r.data.coins } else this.msg = r.msg
      })
    }
  }
}
</script>
