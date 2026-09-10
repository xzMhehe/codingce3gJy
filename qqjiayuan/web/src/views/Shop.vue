<template>
  <div>
    <!-- 面包屑（复刻诺哈 shop*.asp：地盘>财务>商店>购买/赠送） -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;<a href="javascript:;" @click="$router.push('/box')">用户中心</a>&gt;<a href="javascript:;" @click="$router.push('/wallet')">我的钱包</a>&gt;<a href="javascript:;" @click="go('list')">商店</a><template v-if="view === 'detail'">&gt;详情</template><template v-else-if="view === 'buy' || view === 'buyOk'">&gt;购买</template><template v-else-if="view === 'send' || view === 'sendOk'">&gt;赠送</template><br>
    </div>
    <div class="name">道具商城<br></div>

    <!-- 列表（复刻诺哈 shop_list.asp：平铺编号列表） -->
    <template v-if="view === 'list'">
      <div class="module-content" v-if="coins >= 0">
        我的G币：<b style="color:#e05a00">{{ coins }}</b>
        <span class="txt-fade">　友友券：{{ youquan }}</span>
        [<a href="javascript:;" @click="$router.push('/bag')">我的仓库</a>]
        [<a href="javascript:;" @click="$router.push('/youquan')">友友券中心</a>]
      </div>
      <div class="module-content">
        <span v-for="(g, i) in goods" :key="g.id">{{ i + 1 + (page - 1) * 10 }}.<a href="javascript:;" @click="go('detail', g.id)">{{ g.name }}</a><br></span>
        <span v-if="!goods.length" class="empty">暂无销售。</span>
      </div>
      <!-- 分页（复刻诺哈：下页.上页 + 跳页输入） -->
      <div class="module-content" v-if="totalPages > 1">
        <a href="javascript:;" v-if="page < totalPages" @click="load(page + 1)">下页</a><span v-if="page < totalPages && page > 1">.</span><a href="javascript:;" v-if="page > 1" @click="load(page - 1)">上页</a>
        (第<b>{{ page }}</b>/{{ totalPages }}页/共{{ total }}记录)<br>
        第<input type="text" v-model.number="jumpPage" size="2">页 <input type="submit" value="前往" @click="jump">
      </div>
      <div class="module-content txt-fade">鲜花类道具在帖子下方【送花】使用，其余道具到仓库中使用。</div>
    </template>

    <!-- 详情（复刻诺哈 shop.asp） -->
    <template v-else-if="view === 'detail' && g">
      <div class="module-content">
        <b>{{ g.name }}</b><br>
        销售价格：<b style="color:#e05a00">{{ g.price }}</b>(G币)<br>
        库存数量：{{ g.stock }}<br>
        销售数量：{{ g.sales }}<br>
        销售时间：{{ fmt(g.add_time) }}<br>
        结束时间：{{ fmt(g.end_time) }}<br>
        <a href="javascript:;" @click="go('buy', g.id)">购买</a>.<a href="javascript:;" @click="go('send', g.id)">赠送</a><br>
      </div>
      <div class="module-content">----------<br><span class="txt-fade">{{ g.desc }}</span></div>
    </template>

    <!-- 购买：数量+支付密码（复刻诺哈 shop_buy.asp，VerifyPayPass） -->
    <template v-else-if="view === 'buy' && g">
      <div class="module-content">
        【购买道具】<br>
        <b>{{ g.name }}</b><br>
        销售价格：<b style="color:#e05a00">{{ g.price }}</b>(G币)<br>
        库存数量：{{ g.stock }}<br>
        购买数量：<input type="text" v-model.number="num" size="4"><br>
        <template v-if="hasPaypass">
          支付密码：<input type="password" v-model.trim="payPass" size="12"><br>
          <input type="submit" value="下一步" @click="toBuyOk">
        </template>
        <template v-else>您还未设置支付密码，请先到<a href="javascript:;" @click="$router.push('/security')">安全中心</a>设置！</template>
      </div>
    </template>

    <!-- 购买确认（复刻诺哈 shop_buy_ok.asp：act=ok 确认页） -->
    <template v-else-if="view === 'buyOk' && g">
      <div class="module-content">
        【购买道具】<br>
        购买数量: {{ num }}<br>
        购买总价: <b style="color:#e05a00">{{ g.price * num }}</b>(G币)<br>
        <input type="submit" value="确定购买" :disabled="submitting" @click="doBuy">
      </div>
    </template>

    <!-- 赠送：号码+数量+支付密码（复刻诺哈 shop_send.asp） -->
    <template v-else-if="view === 'send' && g">
      <div class="module-content">
        【赠送道具】<br>
        <b>{{ g.name }}</b><br>
        销售价格：<b style="color:#e05a00">{{ g.price }}</b>(G币)<br>
        库存数量：{{ g.stock }}<br>
        会员号码：<input type="text" v-model.trim="to" size="8"><br>
        赠送数量：<input type="text" v-model.number="snum" size="4"><br>
        <template v-if="hasPaypass">
          支付密码：<input type="password" v-model.trim="payPass" size="12"><br>
          <input type="submit" value="下一步" @click="toSendOk">
        </template>
        <template v-else>您还未设置支付密码，请先到<a href="javascript:;" @click="$router.push('/security')">安全中心</a>设置！</template>
      </div>
    </template>

    <!-- 赠送确认（复刻诺哈 shop_send_ok.asp 确认页） -->
    <template v-else-if="view === 'sendOk' && pv">
      <div class="module-content">
        【赠送道具】<br>
        赠送会员：<a href="javascript:;" @click="$router.push('/user/' + pv.uid)">{{ pv.nickname }}</a> ({{ pv.username }})<br>
        赠送金额：{{ pv.num }}个({{ pv.name }})<br>
        购买总价：<b style="color:#e05a00">{{ pv.total }}</b>(G币)<br>
        <input type="submit" value="确定赠送" :disabled="submitting" @click="doSend">
      </div>
    </template>

    <!-- 完成 -->
    <template v-else-if="view === 'done'">
      <div class="module-content">{{ doneMsg }}<br>
        [<a href="javascript:;" @click="go('detail', g && g.id)">返回商品</a>]
        [<a href="javascript:;" @click="go('list')">返回商店</a>]
      </div>
    </template>

    <p style="color:#c00;padding:0 5px" v-if="msg">{{ msg }}</p>

    <!-- 底部面包屑 -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;<a href="javascript:;" @click="$router.push('/box')">用户中心</a>&gt;<a href="javascript:;" @click="$router.push('/wallet')">我的钱包</a>&gt;<a href="javascript:;" @click="go('list')">商店</a><br>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Shop',
  data () {
    return {
      view: 'list', goods: [], g: null, pv: null,
      coins: -1, youquan: -1,
      page: 1, total: 0, totalPages: 1, jumpPage: 1,
      num: 1, snum: 1, to: '', payPass: '',
      hasPaypass: null, submitting: false, doneMsg: '',
      msg: ''
    }
  },
  watch: {
    $route: 'sync'
  },
  mounted () { this.sync() },
  methods: {
    sync () {
      const q = this.$route.query
      this.view = q.view || 'list'
      this.msg = ''
      if (this.view === 'list') {
        this.load(q.page ? parseInt(q.page) || 1 : 1)
      } else if (q.id) {
        this.loadGood(parseInt(q.id))
      }
      if ((this.view === 'buy' || this.view === 'send') && this.hasPaypass === null) {
        api.get('/me/paypass').then(r => {
          this.hasPaypass = !!(r.code === 0 && r.data && r.data.has_paypass)
        })
      }
    },
    go (view, id) {
      const q = { view }
      if (id) q.id = id
      this.$router.push({ path: '/shop', query: q })
    },
    load (p) {
      this.page = p
      api.get('/goods', { params: { page: p } }).then(r => {
        if (r.code === 0) {
          this.goods = r.data.list || []
          this.coins = r.data.coins
          this.youquan = r.data.youquan
          this.total = r.data.total || 0
          const size = r.data.size || 10
          this.totalPages = Math.max(1, Math.ceil(this.total / size))
        }
      })
    },
    jump () {
      const p = Math.floor(this.jumpPage || 0)
      if (p >= 1 && p <= this.totalPages) { this.load(p) } else { this.msg = '页码超出范围！' }
    },
    loadGood (id) {
      this.g = null
      this.pv = null
      this.num = 1
      this.snum = 1
      this.payPass = ''
      api.get('/goods/' + id).then(r => {
        if (r.code === 0) { this.g = r.data } else { this.msg = r.msg }
      })
    },
    toBuyOk () {
      if (!(this.num >= 1)) { this.msg = '购买数量错误！'; return }
      if (!this.payPass) { this.msg = '请填写支付密码！'; return }
      this.go('buyOk', this.g.id)
    },
    doBuy () {
      this.submitting = true
      this.msg = ''
      api.post('/goods/' + this.g.id + '/buy', { num: this.num, pay_pass: this.payPass }).then(r => {
        this.submitting = false
        if (r.code === 0) {
          this.coins = r.data.coins
          this.youquan = r.data.youquan
          this.doneMsg = '购买成功！'
          this.payPass = ''
          this.view = 'done'
          this.$router.push({ path: '/shop', query: { view: 'done', id: this.g.id } })
        } else this.msg = r.msg
      })
    },
    toSendOk () {
      if (!this.to) { this.msg = '请填写赠送号码！'; return }
      if (!(this.snum >= 1)) { this.msg = '赠送数量错误！'; return }
      if (!this.payPass) { this.msg = '请填写支付密码！'; return }
      this.msg = ''
      api.get('/goods/' + this.g.id + '/send-preview', { params: { to: this.to, num: this.snum } }).then(r => {
        if (r.code === 0) {
          this.pv = r.data
          this.go('sendOk', this.g.id)
        } else this.msg = r.msg
      })
    },
    doSend () {
      this.submitting = true
      this.msg = ''
      api.post('/goods/' + this.g.id + '/send', { to: this.to, num: this.snum, pay_pass: this.payPass }).then(r => {
        this.submitting = false
        if (r.code === 0) {
          this.coins = r.data.coins
          this.doneMsg = '赠送成功！'
          this.payPass = ''
          this.view = 'done'
          this.$router.push({ path: '/shop', query: { view: 'done', id: this.g.id } })
        } else this.msg = r.msg
      })
    },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t)
      return d.getFullYear() + '/' + (d.getMonth() + 1) + '/' + d.getDate() + ' ' + d.getHours() + ':' + (d.getMinutes() < 10 ? '0' : '') + d.getMinutes()
    }
  }
}
</script>

<style scoped>
.pager, .list { padding: 3px 0; }
</style>
