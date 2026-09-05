<template>
  <div>
    <div class="name">我的钱包<br></div>

    <!-- 顶部功能导航 -->
    <div class="module-content" style="padding:3px 5px">
      <a href="javascript:;" :class="tab === 'balance' ? 'cur' : ''" @click="tab = 'balance'">[账户余额]</a>
      <a href="javascript:;" :class="tab === 'logs' ? 'cur' : ''" @click="tab = 'logs'">[收支明细]</a>
      <a href="javascript:;" :class="tab === 'help' ? 'cur' : ''" @click="tab = 'help'">[获取G币]</a>
      <a href="javascript:;" @click="$router.push('/shop')">[商城]</a>
      <a href="javascript:;" @click="$router.push('/bag')">[仓库]</a>
    </div>

    <!-- 账户余额 -->
    <div class="module-content plist" v-if="tab === 'balance'">
      <div class="row00"><span class="m-badge" style="background:#d4a017">G币</span><b style="color:#e05a00">{{ data.coins || 0 }}</b>　<span class="txt-fade">家园通用货币</span></div>
      <div class="row00"><span class="m-badge" style="background:#f7c531">元宝</span><b style="color:#e05a00">{{ data.yuanbao || 0 }}</b>　<span class="txt-fade">连续签到7天可得</span></div>
      <div class="row00"><span class="m-badge" style="background:#7f6cf0">金钻</span><b style="color:#7f6cf0">{{ data.jinzuan || 0 }}</b>　<span class="txt-fade">稀有货币，活动可得</span></div>
      <div class="row00"><span class="m-badge" style="background:#2ea44f">友友券</span><b style="color:#2ea44f">{{ data.youquan || 0 }}</b>　<span class="txt-fade">连续签到14天可得</span></div>
      <div class="row00"><span class="m-badge" style="background:#5a7fb8">银行</span><b style="color:#1a9e1a">{{ data.bank || 0 }}</b>　<span class="txt-fade">日息0.5%，每天领一次</span> <a href="javascript:;" @click="$router.push('/play')">[去银行&gt;&gt;]</a></div>
      <div class="row00 txt-fade">累计打工：{{ data.work_total || 0 }} 次　<a href="javascript:;" @click="$router.push('/play')">去打工&gt;&gt;</a></div>
    </div>

    <!-- 收支明细 -->
    <div v-if="tab === 'logs'">
      <div class="module-title">收支明细</div>
      <div class="list" v-if="logs.length">
        <div class="row" v-for="l in logs" :key="l.id" style="padding:4px 2px;border-bottom:1px dotted #dfe8f2">
          <b>{{ l.title }}</b>
          <span :style="{color: l.delta > 0 ? '#1a9e1a' : '#c00'}">{{ l.delta > 0 ? '+' : '' }}{{ l.delta }} {{ curName(l.currency) }}</span>
          <div class="txt-fade">{{ fmt(l.created_at) }}</div>
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">暂无收支记录，去发帖、回帖或打工赚 G币吧～</span></div>
    </div>

    <!-- 获取G币 -->
    <div v-if="tab === 'help'">
      <div class="module-title">G币获取途径</div>
      <div class="module-content">
        <div class="row00">· 每日签到：<a href="javascript:;" @click="$router.push('/sign')">去签到&gt;&gt;</a>（+10~17 G币）</div>
        <div class="row00">· 发布帖子：+5 G币，回复帖子：+2 G币</div>
        <div class="row00">· 打工赚钱：<a href="javascript:;" @click="$router.push('/play')">去打工&gt;&gt;</a>（每天3次，+5~15 G币）</div>
        <div class="row00">· 挖宝：<a href="javascript:;" @click="$router.push('/play')">去挖宝&gt;&gt;</a>（花30 G币搏100 G币）</div>
        <div class="row00">· 家族签到、守护树、慈善排行等更多途径</div>
      </div>
      <div class="module-title">元宝 / 金钻 / 友友券</div>
      <div class="module-content">
        <div class="row00">· 连续签到 7 天：+2 元宝</div>
        <div class="row00">· 连续签到 14 天：+5 友友券</div>
        <div class="row00">· 连续签到 30 天：+1 金钻</div>
        <div class="row00">· 金钻还可通过活动、管理后台发放获得</div>
      </div>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Wallet',
  data () { return { data: {}, logs: [], tab: 'balance' } },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/wallet').then(r => {
        if (r.code === 0) { this.data = r.data; this.logs = r.data.logs || [] }
      })
    },
    curName (c) {
      return { coins: 'G币', yuanbao: '元宝', jinzuan: '金钻', youquan: '友友券', flower: '鲜花' }[c] || c
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
.m-badge {
  display: inline-block;
  min-width: 42px;
  text-align: center;
  color: #fff;
  border-radius: 3px;
  font-size: 12px;
  padding: 1px 5px;
  margin-right: 6px;
  font-weight: bold;
}
.cur { color: #e05a00; font-weight: bold; }
</style>
