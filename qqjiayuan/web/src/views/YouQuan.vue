<template>
  <div>
    <!-- 面包屑 -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/home')">我的地盘</a>&gt;友友券<br>
    </div>

    <!-- 余额 -->
    <div class="module-content unline">
      <img src="/static/picture/mon03.png" alt=".">友友券余额:<b style="color:#e05a00">{{ data.youquan || 0 }}</b>张<br>
      <span class="txt-fade">友友券可用于商城购买道具、兑换G币、转账给好友。连续签到14天奖励5张。</span><br>
    </div>

    <!-- 功能导航（复刻诺哈 财务中心：领取.兑换.转账.收支） -->
    <div class="module-title">
      <a href="javascript:;" @click="view='home'">友友券中心</a>|
      <a href="javascript:;" @click="view='daily'">领取</a>|
      <a href="javascript:;" @click="view='exchange'">兑换</a>|
      <a href="javascript:;" @click="view='transfer'">转账</a>|
      <a href="javascript:;" @click="view='log'">收支</a><br>
    </div>

    <!-- ===== 首页：获取/使用 ===== -->
    <div v-if="view === 'home'">
      <div class="module-title">【获取友友券】</div>
      <div class="module-content">
        <span v-if="!data.claimed_today"><a href="javascript:;" @click="doDaily">每日免费领取1~3张</a></span>
        <span v-else><font color="#1a9e1a">今日已领取</font></span><br>
        连续签到7天+2元宝，14天+5友友券，30天+1金钻（<a href="javascript:;" @click="$router.push('/sign')">去签到</a>）<br>
        参与社区活动、花园游戏可获得友友券奖励。<br>
      </div>
      <div class="module-title">【使用友友券】</div>
      <div class="module-content">
        <a href="javascript:;" @click="$router.push('/shop')">商城购物</a>（{{ data.youquan_goods || 0 }}种商品支持友友券支付）<br>
        <a href="javascript:;" @click="view='exchange'">兑换G币</a>（{{ data.exchange_rate || 100 }} G币/张）<br>
        <a href="javascript:;" @click="view='transfer'">转账给好友</a>（1%手续费）<br>
      </div>
    </div>

    <!-- ===== 每日领取 ===== -->
    <div v-if="view === 'daily'">
      <div class="module-title">每日免费领取<br></div>
      <div class="module-content">
        <span v-if="!data.claimed_today">今天可领取 <input type="submit" value="点击领取1~3张" @click="doDaily"></span>
        <span v-else><font color="#1a9e1a">今天已经领取过啦，明天再来！</font></span><br>
      </div>
    </div>

    <!-- ===== 兑换G币 ===== -->
    <div v-if="view === 'exchange'">
      <div class="module-title">友友券兑换G币<br></div>
      <div class="module-content">
        汇率：1 友友券 = {{ data.exchange_rate || 100 }} G币<br>
        兑换数量：<input type="text" v-model.number="exchangeNum" size="6"> 张
        <input type="submit" value="确认兑换" @click="doExchange"><br>
        <span class="help-line">兑换后友友券扣除、G币到账（记入收支记录）。</span><br>
      </div>
    </div>

    <!-- ===== 转账 ===== -->
    <div v-if="view === 'transfer'">
      <div class="module-title">友友券转账<br></div>
      <div class="module-content">
        对方号码：<input type="text" v-model.trim="tfTo" size="8"><br>
        转账数量：<input type="text" v-model.number="tfAmount" size="8"> 张<br>
        备&nbsp;&nbsp;&nbsp;&nbsp;注：<input type="text" v-model.trim="tfRemark" size="12"><br>
        <input type="submit" value="确定转账" @click="doTransfer"><br>
        <span class="help-line">说明：转账收取1%手续费；数量在1~1000000之间。</span><br>
      </div>
    </div>

    <!-- ===== 收支记录 ===== -->
    <div v-if="view === 'log'">
      <div class="module-title">友友券收支记录<br></div>
      <div class="module-content" v-if="logs.length">
        <div v-for="(l,i) in logs" :key="l.id">
          {{ i+1 }}.{{ l.title }}:{{ l.delta > 0 ? '+' : '' }}{{ l.delta }}友友券,{{ fmt(l.created_at) }}<br>
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">暂无友友券收支记录，先去签到/领取吧</span></div>
      <div class="module-content txt-fade">(共 {{ logs.length }} 条最近记录)</div>
    </div>

    <p style="color:#1a9e1a;padding:0 5px" v-if="okMsg">{{ okMsg }}</p>
    <p style="color:#c00;padding:0 5px" v-if="msg">{{ msg }}</p>

    <!-- 底部面包屑 -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/home')">我的地盘</a>&gt;友友券<br>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'YouQuan',
  data () {
    return {
      data: {}, logs: [], view: 'home',
      exchangeNum: 10, tfTo: '', tfAmount: null, tfRemark: '',
      okMsg: '', msg: ''
    }
  },
  mounted () {
    this.view = this.$route.query.view || 'home'
    this.load()
  },
  methods: {
    load () {
      api.get('/youquan').then(r => {
        if (r.code === 0) {
          this.data = r.data
          this.logs = r.data.logs || []
        } else {
          this.msg = r.msg
        }
      })
    },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t)
      return d.getFullYear() + '/' + (d.getMonth() + 1) + '/' + d.getDate() + ' ' + d.getHours() + ':' + (d.getMinutes() < 10 ? '0' : '') + d.getMinutes()
    },
    doDaily () {
      api.post('/youquan/daily').then(r => {
        if (r.code === 0) {
          this.okMsg = '领取成功！获得 ' + r.data.gain + ' 张友友券'
          this.msg = ''
          this.load()
        } else { this.msg = r.msg; this.okMsg = '' }
      })
    },
    doExchange () {
      const n = Math.max(1, Math.floor(this.exchangeNum || 0))
      if (!n) { this.msg = '请输入兑换数量'; return }
      api.post('/youquan/exchange', { num: n }).then(r => {
        if (r.code === 0) { this.okMsg = '兑换成功！到账 ' + r.data.gain + ' G币'; this.msg = ''; this.load() } else { this.msg = r.msg; this.okMsg = '' }
      })
    },
    doTransfer () {
      const amt = Math.floor(this.tfAmount || 0)
      if (!this.tfTo || amt <= 0) { this.msg = '请输入对方号码和转账数量'; return }
      api.post('/youquan/transfer', { to: this.tfTo, amount: amt, remark: this.tfRemark }).then(r => {
        if (r.code === 0) { this.okMsg = '转账成功！'; this.msg = ''; this.tfTo = ''; this.tfAmount = null; this.tfRemark = ''; this.load() } else { this.msg = r.msg; this.okMsg = '' }
      })
    }
  }
}
</script>