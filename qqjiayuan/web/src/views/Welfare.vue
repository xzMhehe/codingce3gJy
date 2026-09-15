<template>
  <div>
    <!-- 领取名单页（复刻 福利院领取名单.xhtml） -->
    <template v-if="isClaimsView">
      <div class="name">【福利院今日领取名单】<br /></div>
      今日累计慈善基金支出：{{ fmtNum(claimSum) }}慈善币<br />
      -------------
      <div class="list">
        <template v-if="claimListFull.length">
          <div class="row" v-for="(cl, i) in claimListFull" :key="cl.id">
            {{ (cfPage - 1) * claimSizeFull + i + 1 }}.<a href="javascript:;" @click="goUser(cl.user_id)"><ntext :color="cl.color || '#004299'">{{ cl.nickname }}</ntext></a>今日领慈善福利{{ fmtNum(cl.amount) }}G币<br />{{ fmtTime(cl.created_at) }}
          </div>
          <div class="row pgn">
            <a v-if="cfPage > 1" href="javascript:;" @click="loadClaimsFull(cfPage - 1)">上页</a>
            <span v-if="cfPage > 1"> . </span>
            <a v-if="cfPage * claimSizeFull < claimTotal" href="javascript:;" @click="loadClaimsFull(cfPage + 1)">下页</a><br />
            (第<b>{{ cfPage }}</b>/{{ claimPagesFull }}页/共{{ claimTotal }}条记录)
          </div>
        </template>
        <div class="row" v-else>今日还没有友友领取慈善福利~</div>
      </div>
      ----------
      <br />
      <a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;<a href="javascript:;" @click="$router.push('/fl')">福利院</a>&gt;福利领取名单
    </template>

    <!-- 捐赠名单页（复刻 褔利院捐赠名单.xhtml） -->
    <template v-else-if="isDonationsView">
      <div class="name">【福利院今日捐献名单】<br /></div>
      -------------
      <br />
      今日累计慈善基金收入：{{ fmtNum(donateSum) }}慈善币<br />
      -------------
      <div class="list">
        <template v-if="donateListFull.length">
          <div class="row" v-for="(d, i) in donateListFull" :key="d.id">
            {{ (dfPage - 1) * donateSizeFull + i + 1 }}.[<a href="javascript:;" @click="goUser(d.user_id)"><ntext :color="d.color || '#004299'">{{ d.nickname }}</ntext></a>]今日捐献了{{ fmtNum(d.amount) }}G币,获得{{ d.points }}财富值。{{ fmtTime(d.created_at) }}
          </div>
          <div class="row pgn">
            <a v-if="dfPage > 1" href="javascript:;" @click="loadDonationsFull(dfPage - 1)">上页</a>
            <span v-if="dfPage > 1"> . </span>
            <a v-if="dfPage * donateSizeFull < donateTotal" href="javascript:;" @click="loadDonationsFull(dfPage + 1)">下页</a><br />
            (第<b>{{ dfPage }}</b>/{{ donatePagesFull }}页/共{{ donateTotal }}条记录)
          </div>
        </template>
        <div class="row" v-else>今日还没有友友捐献慈善基金~</div>
      </div>
      ----------
      <br />
      <a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;<a href="javascript:;" @click="$router.push('/fl')">福利院</a>&gt;慈善捐献名单
    </template>

    <!-- 主页（复刻 3GQQ家园褔利院.xhtml） -->
    <template v-else>
      <img src="/static/picture/fly.gif" alt="" /><br />
      <b>慈善基金中心</b><br />
      <div class="line" />
      慈善基金：<b style="color:#e05a00">{{ fmtNum(pool) }}</b>G币<br />
      当前日期：{{ curDate }}<br />
      --------------------<br />
      <template v-if="isLogin">
        <a href="javascript:;" @click="claim">我要领取</a> .<a href="javascript:;" @click="$router.push('/fl?aa=2')">领取名单</a><br />
        <a href="javascript:;" @click="$router.push('/fla')">我要捐赠</a> .<a href="javascript:;" @click="$router.push('/fl?aa=4')">捐赠名单</a><br /><br />
      </template>
      <template v-else>
        <a href="javascript:;" @click="$router.push('/login')">登录</a>后即可领取/捐献慈善福利<br /><br />
      </template>
      --------------------<br />
      <span style="color:red;">今日领取福利需先完成以下任务：</span><br />
      <template v-if="mine">
        1. 发布一篇帖子（<span :style="{color: mine.task_thread >= 1 ? 'green' : 'red'}">{{ mine.task_thread >= 1 ? '已完成' : '未完成' }}</span>）<br />
        2. 完成五条回帖（<span :style="{color: mine.task_reply >= 5 ? 'green' : 'red'}">{{ mine.task_reply }}/5</span>）<br />
        <span v-if="mine.claims_today > 0" class="txt-fade">今日已领取{{ fmtNum(mine.claim_today_sum) }}G币</span><br />
      </template>
      <template v-else>
        1. 发布一篇帖子<br />
        2. 完成五条回帖<br />
      </template>
      <a href="javascript:;" @click="$router.push('/board/5')">去论坛</a><br />
      <span style="color:#c00">{{ msg }}</span>
      <span style="color:#1a9e1a">{{ okMsg }}</span>

      <b>领取动态</b><br />
      <div class="line" />
      <div class="list" id="sec-claims">
        <template v-if="claims.length">
          <div class="row" v-for="(cl, i) in claims" :key="cl.id">
            {{ i + 1 }}.({{ timeAgo(cl.created_at) }})<a href="javascript:;" @click="goUser(cl.user_id)"><ntext :color="cl.color || '#004299'">{{ cl.nickname }}</ntext></a><font color="#0051A4">{{ cl.username ? '(' + cl.username + ')' : '' }}</font>成功领取{{ fmtNum(cl.amount) }}G币！<br />
          </div>
        </template>
        <div class="row" v-else>今日还没有友友领取慈善福利~</div>
      </div>

      <b>捐献动态</b><br />
      <div class="line" />
      <div class="list" id="sec-donations">
        <template v-if="donations.length">
          <div class="row" v-for="(d, i) in donations" :key="d.id">
            {{ i + 1 }}.({{ timeAgo(d.created_at) }})<a href="javascript:;" @click="goUser(d.user_id)"><ntext :color="d.color || '#004299'">{{ d.nickname }}</ntext></a><font color="#0051A4">{{ d.username ? '(' + d.username + ')' : '' }}</font>捐献慈善基金{{ fmtNum(d.amount) }}G币，获得{{ d.points }}财富值！<br />
          </div>
        </template>
        <div class="row" v-else>今日还没有友友捐献慈善基金~</div>
      </div>

      <b>财富排行</b><br />
      <div class="line" />
      <div class="list">
        <template v-if="rank.length">
          <div class="row" v-for="(r, i) in rank" :key="r.user_id">
            NO.{{ i + 1 }} <a href="javascript:;" @click="goUser(r.user_id)"><ntext :color="r.color || '#004299'">{{ r.nickname }}</ntext></a><font color="#0051A4">{{ r.username ? '(' + r.username + ')' : '' }}</font>，累计{{ fmtNum(r.points) }}财富值！<br />
          </div>
        </template>
        <div class="row" v-else>虚位以待，快去捐献慈善基金赢取财富值！</div>
      </div>

      <b>竞价排行</b><br />
      <template v-if="flTop">
        <div class="row">
          <table><tbody><tr>
            <td valign="top" align="center">
              <div style="margin:2px 5px;width:54px;height:54px;border-radius:20%;padding:1px;background-size:100% 100%;"
                   :style="ttBg"></div>
            </td>
            <td style="width:100%;padding-left:0px">
              膜拜：<a href="javascript:;" @click="goUser(flTop.user_id)"><span class="text_effect22"><ntext :color="flTop.color || '#ff0000'">{{ flTop.nickname }}</ntext></span><font color="#0051A4">{{ flTop.username ? '(' + flTop.username + ')' : '' }}</font></a>
              <a href="javascript:;" @click="$router.push('/fla')"><span id="anniu" style="color:#fff;font-size:12px;">我要上榜</span></a><br />
              <em>宣言：{{ flTop.word || '这个颠佬很懒，什么也没有写。' }}</em><br />
              <template v-if="isLogin">
                <a v-if="!flWorshipped && myId !== flTop.user_id" href="javascript:;" @click="worship"><span class="text_effect22">膜拜</span></a>
                <span v-else-if="flWorshipped" class="txt-fade">今日已膜拜</span>
                <a v-else href="javascript:;" @click="$router.push('/fla')"><span class="text_effect22">膜拜</span></a>
                <span>(当前魅力：{{ flTop.worships }})</span>
              </template>
              <template v-else>
                <a href="javascript:;" @click="$router.push('/login')"><span class="text_effect22">膜拜</span></a><span>(当前魅力：{{ flTop.worships }})</span>
              </template>
            </td>
          </tr></tbody></table>
        </div>
      </template>
      <template v-else>
        <div class="row">今日榜单虚位以待，快去<a href="javascript:;" @click="$router.push('/fla')">捐款上榜</a>！</div>
      </template>
      <em><img src="/static/picture/002.gif" alt="" /> 慈善基金池金额越多，玩家领取的慈善币越多。</em><br />
      <em><img src="/static/picture/002.gif" alt=""/> 每日捐献慈善币第一名为可以接受全社区网友<span class="sqm4">膜拜</span>，每日仅可捐献一次。</em><br />

      <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;慈善基金<br /></div>
    </template>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Welfare',
  data () {
    return {
      pool: 0,
      mine: null,
      claims: [], donations: [], rank: [],
      flTop: null, flWorshipped: false,
      claimPage: 1, claimSize: 5,
      claimListFull: [], cfPage: 1, claimSizeFull: 10, claimTotal: 0, claimSum: 0,
      donateListFull: [], dfPage: 1, donateSizeFull: 10, donateTotal: 0, donateSum: 0,
      busy: false, msg: '', okMsg: ''
    }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    myId () { return (this.$store.state.user && this.$store.state.user.id) || 0 },
    isClaimsView () { return this.$route.query.aa === '2' },
    isDonationsView () { return this.$route.query.aa === '4' },
    curDate () {
      const d = new Date()
      return d.getFullYear() + '/' + (d.getMonth() + 1) + '/' + d.getDate()
    },
    claimPagesFull () { return Math.max(1, Math.ceil(this.claimTotal / this.claimSizeFull)) },
    ttBg () {
      const t = this.flTop || {}
      if (t.avatar_base64 && t.avatar_base64.length > 20) {
        return { background: 'url(' + t.avatar_base64 + ') 0 0/100% 100% no-repeat' }
      }
      if (t.avatar) {
        return { background: 'url(/static/picture/' + t.avatar + ') 0 0/100% 100% no-repeat' }
      }
      return { background: '#cfe0f0' }
    }
  },
  mounted () {
    this.load()
    if (this.isClaimsView) { this.loadClaimsFull(1); return }
    if (this.isDonationsView) { this.loadDonationsFull(1); return }
    this.loadClaims(1)
  },
  watch: {
    isClaimsView (v) { if (v) this.loadClaimsFull(1) },
    isDonationsView (v) { if (v) this.loadDonationsFull(1) }
  },
  methods: {
    load () {
      api.get('/welfare').then(r => {
        if (r.code === 0) {
          this.pool = r.data.pool
          this.mine = r.data.mine
          this.claims = r.data.claims || []
          this.donations = r.data.donations || []
          this.rank = r.data.rank || []
          this.flTop = r.data.fl_top
          this.flWorshipped = !!r.data.fl_worshipped
        }
      })
    },
    loadClaims (p) {
      this.claimPage = p
      api.get('/welfare/claims', { params: { page: p, size: this.claimSize } }).then(r => {
        if (r.code === 0) { this.claims = r.data.list }
      })
    },
    loadClaimsFull (p) {
      this.cfPage = p
      api.get('/welfare/claims', { params: { page: p, size: this.claimSizeFull } }).then(r => {
        if (r.code === 0) { this.claimListFull = r.data.list; this.claimTotal = r.data.total; this.claimSum = r.data.sum }
      })
    },
    loadDonationsFull (p) {
      this.dfPage = p
      api.get('/welfare/donations', { params: { page: p, size: this.donateSizeFull } }).then(r => {
        if (r.code === 0) { this.donateListFull = r.data.list; this.donateTotal = r.data.total; this.donateSum = r.data.sum }
      })
    },
    claim () {
      if (!this.isLogin) { this.$router.push('/login'); return }
      this.busy = true; this.msg = ''; this.okMsg = ''
      api.post('/welfare/claim').then(r => {
        this.busy = false
        if (r.code === 0) {
          this.okMsg = '领取成功！获得' + Number(r.data.amount).toLocaleString() + 'G币！'
        } else {
          this.msg = r.msg
        }
        this.load(); this.loadClaims(1)
      }).catch(() => { this.busy = false; this.msg = '网络开小差了，稍后再试' })
    },
    worship () {
      if (!this.isLogin) { this.$router.push('/login'); return }
      api.post('/fla/worship').then(r => {
        if (r.code === 0) {
          this.flWorshipped = true
          if (this.flTop) this.flTop.worships = r.data.worships
          this.msg = ''
          this.okMsg = '膜拜成功！'
        } else this.msg = r.msg
      })
    },
    scrollTo (id) {
      const el = document.getElementById(id)
      if (el) el.scrollIntoView()
    },
    goUser (id) { if (id) this.$router.push('/user/' + id) },
    fmtNum (n) { return Number(n || 0).toLocaleString() },
    fmtTime (t) {
      if (!t) return ''
      return String(t).slice(0, 19).replace('T', ' ')
    },
    timeAgo (t) {
      if (!t) return ''
      const diff = (Date.now() - new Date(String(t).replace(' ', 'T').replace('Z', '')).getTime()) / 1000
      if (diff < 60) return Math.max(1, Math.floor(diff)) + '秒前'
      if (diff < 3600) return Math.floor(diff / 60) + '分前'
      if (diff < 86400) return Math.floor(diff / 3600) + '小时' + Math.floor((diff % 3600) / 60) + '分前'
      return Math.floor(diff / 86400) + '天前'
    }
  }
}
</script>

<style scoped>
.txt-fade { color: #999; font-size: 12px; }
.pgn { color: #999; }
.line { border:none; border-top:1px solid #9FC6EC; margin:2px 0 1px; }
.name { padding-left:3px; line-height:20px; border-bottom:2px solid #9FC6EC; color:#000; font-weight:bold; }
b { margin-top: 4px; }
</style>
