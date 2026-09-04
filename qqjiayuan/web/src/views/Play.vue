<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/')">首页</a>&gt;<a href="javascript:;" @click="$router.push('/')">广场</a>&gt;欢乐坊 · 游乐场</div>

    <!-- 社区银行 -->
    <div class="module-title">社区银行</div>
    <div class="module-content">
      <p>当前金币：<b style="color:#e05a00">{{ bank.coins || 0 }}</b>　银行存款：<b style="color:#1a9e1a">{{ bank.balance || 0 }}</b></p>
      <p class="txt-fade">今日利息 {{ bank.rate || 0 }} 金币{{ bank.interest_today ? '（已领取）' : '（待领取）' }}，存款按 0.5%/日 计息，可每日领一次。</p>
      <p><button class="btn" :disabled="bank.interest_today || !bank.balance" @click="doInterest">领取今日利息</button></p>
      <p>存入 / 取出金币：
        <input type="text" v-model.number="bankAmt" size="6">
        <button class="btn" @click="bankOp('deposit')">存入</button>
        <button class="btn" @click="bankOp('withdraw')">取出</button>
      </p>
    </div>

    <!-- 打工 -->
    <div class="module-title">打工·赚金币</div>
    <div class="module-content">
      <p class="txt-fade">今日已打工 <b style="color:#e05a00">{{ work.done || 0 }}</b>/{{ work.limit }} 次，剩余 {{ work.left || 0 }} 次，每次随机 5~15 金币。</p>
      <p><button class="btn" @click="doWork">今 日 打 工</button></p>
    </div>

    <!-- 每日星运 -->
    <div class="module-title">每日星运</div>
    <div class="module-content plist">
      <div class="row00" v-if="fortune.luck">
        <b style="color:#e05a00">{{ fortune.star }} {{ fortune.luck }}</b>　（{{ fortune.date }}）
        <br>{{ fortune.opts }}，{{ fortune.word }}
        <br><span class="txt-fade">幸运数字：<b style="color:#004299">{{ fortune.lucky }}</b></span>
      </div>
      <div class="row00" v-else>正在卜卦中…</div>
    </div>

    <!-- 幸运猜数字 -->
    <div class="module-title">幸运猜数字</div>
    <div class="module-content">
      <p class="txt-fade">本期开奖 <b style="color:#e05a00">{{ lastNum || '?' }}</b>，猜「大(≥5)/小(≤4)/单/双」，猜中翻倍返还，猜错金币扣除。当前金币 <b style="color:#e05a00">{{ coins }}</b>。</p>
      <p><button class="btn" :class="{ red: bet === 'big' }" @click="bet = 'big'">大</button>
        <button class="btn" :class="{ red: bet === 'small' }" @click="bet = 'small'">小</button>
        <button class="btn" :class="{ red: bet === 'odd' }" @click="bet = 'odd'">单</button>
        <button class="btn" :class="{ red: bet === 'even' }" @click="bet = 'even'">双</button></p>
      <p>下注金币：<input type="text" v-model.number="lotAmt" size="6">
        <button class="btn" @click="doLottery">开始竞猜</button></p>
    </div>

    <!-- 挖宝 -->
    <div class="module-title">挖宝</div>
    <div class="module-content">
      <p class="txt-fade">花 {{ digCost }} 金币挖一次宝，验证手气！当前金币 <b style="color:#e05a00">{{ coins }}</b>。</p>
      <p><button class="btn" @click="doDig">开 始 挖 宝</button></p>
      <p v-if="digTip" class="row00">{{ digTip }}</p>
    </div>

    <!-- 慈善基金 -->
    <div class="module-title">慈善基金</div>
    <div class="module-content">
      <p class="txt-fade">捐出金币帮助有需要的友友，做好事攒人品。</p>
      <p>捐款：<input type="text" v-model.number="charityAmt" size="6">
        <button class="btn" @click="doCharity">捐 款</button></p>
      <div class="module-title">慈善榜 TOP10</div>
      <div v-for="(r,i) in charityRank" :key="r.user_id">{{ i+1 }}.<a href="javascript:;" @click="$router.push('/user/'+r.user_id)"><font :color="r.color || '#004299'">{{ r.nickname }}</font></a> —— {{ r.total }}</div>
    </div>

    <p v-if="msg" style="color:#c00;padding:0 5px">{{ msg }}</p>
    <p v-if="okMsg" style="color:#1a9e1a;padding:0 5px">{{ okMsg }}</p>

    <div class="bar"><a href="javascript:;" @click="$router.push('/')">首页</a>&gt;<a href="javascript:;" @click="$router.push('/')">广场</a>&gt;欢乐坊 · 游乐场</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Play',
  data () {
    return {
      msg: '', okMsg: '',
      bank: {}, bankAmt: 5,
      work: { done: 0, limit: 3, left: 3 },
      fortune: {}, digCost: 30, digTip: '', charityAmt: 10, charityRank: [],
      bet: 'big', lotAmt: 10, coins: 0, lastNum: 0
    }
  },
  mounted () { this.coins = this.$store.state.user.coins || 0; this.loadAll() },
  methods: {
    loadAll () { this.loadBank(); this.loadWork(); this.loadFortune(); this.loadCharity() },
    loadCharity () { api.get('/charity/rank').then(r => { if (r.code === 0) this.charityRank = r.data }) },
    doDig () {
      api.post('/dig').then(r => {
        if (r.code === 0) { this.digTip = r.data.tip + ' 获得 ' + r.data.reward + ' 金币'; this.coins = r.data.coins } else this.msg = r.msg
      })
    },
    doCharity () {
      if (!this.charityAmt || this.charityAmt < 1) { this.msg = '请输入捐款金额'; return }
      api.post('/charity', { amount: this.charityAmt }).then(r => {
        if (r.code === 0) { this.okMsg = '感谢捐赠 ' + this.charityAmt + ' 金币'; this.charityAmt = 10; this.loadCharity() } else this.msg = r.msg
      })
    },
    loadBank () { api.get('/bank/view').then(r => { if (r.code === 0) this.bank = r.data }) },
    loadWork () { api.get('/work/status').then(r => { if (r.code === 0) this.work = r.data }) },
    loadFortune () { api.get('/fortune').then(r => { if (r.code === 0) this.fortune = r.data }) },
    bankOp (op) {
      this.msg = ''; this.okMsg = ''
      if (!this.bankAmt || this.bankAmt <= 0) { this.msg = '请输入金额'; return }
      api.post('/bank/' + op, { amount: this.bankAmt }).then(r => {
        if (r.code === 0) { this.okMsg = (op === 'deposit' ? '已存入 ' : '已取出 ') + this.bankAmt + ' 金币'; this.bank = r.data }
        else this.msg = r.msg
      })
    },
    doInterest () {
      this.msg = ''; this.okMsg = ''
      api.post('/bank/interest').then(r => {
        if (r.code === 0) { this.okMsg = '获得利息 ' + r.data.rate + ' 金币'; this.loadBank() }
        else this.msg = r.msg
      })
    },
    doWork () {
      this.msg = ''; this.okMsg = ''
      api.post('/work').then(r => {
        if (r.code === 0) { this.okMsg = '打工成功 +' + r.data.reward + ' 金币'; this.work = { done: r.data.done, limit: r.data.limit, left: r.data.limit - r.data.done } }
        else this.msg = r.msg
      })
    },
    doLottery () {
      this.msg = ''; this.okMsg = ''
      if (!this.lotAmt || this.lotAmt <= 0) { this.msg = '请输入下注金币'; return }
      api.post('/lottery', { bet: this.bet, amount: this.lotAmt }).then(r => {
        if (r.code === 0) {
          this.lastNum = r.data.num; this.coins = r.data.coins
          this.okMsg = '开奖 ' + r.data.num + '，' + (r.data.win ? '恭喜猜中！金币 +' + this.lotAmt : '很遗憾，这局没猜中')
        } else this.msg = r.msg
      })
    }
  }
}
</script>
