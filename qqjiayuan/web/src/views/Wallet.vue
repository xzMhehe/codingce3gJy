<template>
  <div>
    <!-- 面包屑 -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;我的钱包<br>
    </div>
    <div class="note"></div>

    <!-- 用户中心导航 -->
    <div class="module-title">
      <a href="javascript:;" @click="$router.push('/box')">用户中心</a>|<a href="javascript:;" @click="$router.push('/security')">安全中心</a>|我的钱包|<a href="javascript:;" @click="$router.push('/rank')">家园排行</a><br>
    </div>

    <!-- 四币种余额（对齐参考站 mon00~mon03 图标） -->
    <img src="/static/picture/mon00.gif" alt=".">G币余额:{{ data.coins || 0 }}<br>
    <img src="/static/picture/mon01.gif" alt=".">元宝余额:{{ data.yuanbao || 0 }}<a href="javascript:;" @click="goSign">获取</a><br>
    <img src="/static/picture/mon02.gif" alt=".">金钻数量:{{ data.jinzuan || 0 }}点<br>
    <img src="/static/picture/mon03.png" alt=".">友友券数量:{{ data.youquan || 0 }}张<br>

    <!-- G币兑换 -->
    <b>【G币兑换】</b><br>
    1.<a href="javascript:;" @click="exchangeOpen = !exchangeOpen">1000元宝购买1000000币</a><br>
    <div v-if="exchangeOpen" class="module-content">
      <div class="txt-fade">1 元宝 = 1000 G币，兑换后元宝扣除、G币到账（记入收支记录）。</div>
      <input type="text" v-model.number="exchangeNum" size="5" placeholder="元宝数"> 
      <input type="submit" value="兑换" @click="doExchange">
    </div>

    <!-- 功能链接 -->
    <a href="javascript:;" @click="$router.push('/wallet?view=transfer')">G币转账</a> . <a href="javascript:;" @click="$router.push('/wallet?view=log')">收支记录</a><br>
    <a href="javascript:;" @click="$router.push('/play')">社区银行</a> . <a href="javascript:;" @click="$router.push('/play')">G币存取</a><br><br>

    <!-- 收支记录（对齐参考站 money_log 格式） -->
    <div v-if="view === 'log'">
      <div class="module-title">收支记录<br></div>
      <div class="module-content" v-if="logs.length">
        <div v-for="(l, i) in logs" :key="l.id">
          {{ i + 1 }}.{{ l.title }}:{{ l.delta > 0 ? '+' : '' }}{{ l.delta }}{{ l.currency_name }},备注:-,{{ fmt(l.created_at) }}<br>
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">暂无收支记录，去发帖/打工/签到赚G币吧</span></div>
      <div class="module-content txt-fade">(共 {{ logs.length }} 条最近记录)</div>
    </div>

    <!-- G币转账（对齐参考站 trade_add） -->
    <div v-if="view === 'transfer'">
      <div class="module-title">G币转账<br></div>
      <div class="module-content">
        接收号码：<input type="text" v-model.trim="tfTo" size="8"><br>
        转账金额：<input type="text" v-model.number="tfAmount" size="8"> G币<br>
        备注：<input type="text" v-model.trim="tfRemark" size="12"><br>
        <input type="submit" value="转账" @click="doTransfer">
        <p style="color:#1a9e1a" v-if="okMsg">{{ okMsg }}</p>
        <p style="color:#c00" v-if="msg">{{ msg }}</p>
      </div>
    </div>

    <p style="color:#1a9e1a;padding:0 5px" v-if="okMsg && view !== 'transfer'">{{ okMsg }}</p>
    <p style="color:#c00;padding:0 5px" v-if="msg && view !== 'transfer'">{{ msg }}</p>

    <!-- 底部面包屑 -->
    <a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;我的钱包<br>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Wallet',
  data () {
    return {
      data: {}, logs: [], view: 'home',
      exchangeOpen: false, exchangeNum: 1000,
      tfTo: '', tfAmount: null, tfRemark: '',
      okMsg: '', msg: ''
    }
  },
  mounted () {
    this.view = this.$route.query.view || 'home'
    this.load()
  },
  methods: {
    load () {
      api.get('/wallet').then(r => {
        if (r.code === 0) {
          this.data = r.data
          this.logs = r.data.logs || []
        }
      })
    },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t)
      return d.getFullYear() + '/' + (d.getMonth() + 1) + '/' + d.getDate() + ' ' + d.getHours() + ':' + (d.getMinutes() < 10 ? '0' : '') + d.getMinutes()
    },
    goSign () { this.$router.push('/sign') },
    doExchange () {
      const n = Math.max(1, Math.floor(this.exchangeNum || 0))
      if (!n) { this.msg = '请输入兑换的元宝数量'; return }
      api.post('/wallet/exchange', { yuanbao: n }).then(r => {
        if (r.code === 0) { this.okMsg = '兑换成功！到账 ' + r.data.gain + ' G币'; this.msg = ''; this.load() } else { this.msg = r.msg; this.okMsg = '' }
      })
    },
    doTransfer () {
      const amt = Math.floor(this.tfAmount || 0)
      if (!this.tfTo || amt <= 0) { this.msg = '请输入接收号码和转账金额'; return }
      api.post('/wallet/transfer', { to: this.tfTo, amount: amt, remark: this.tfRemark }).then(r => {
        if (r.code === 0) { this.okMsg = '转账成功！'; this.msg = ''; this.tfTo = ''; this.tfAmount = null; this.tfRemark = ''; this.load() } else { this.msg = r.msg; this.okMsg = '' }
      })
    }
  }
}
</script>
