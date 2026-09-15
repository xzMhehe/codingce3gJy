<template>
  <div>
    <div class="module-title">【捐款慈善福利】</div>
    <div class="module-content">
      <em>
        <img src="/static/picture/001.gif" alt=""/>1. 每日捐献慈善币第一名为可以接受全社区网友<span class="sqm4">膜拜</span><br/>
        <img src="/static/picture/001.gif" alt=""/>2. 每人每日仅可捐献一次，价高者上位。<br/>
        <img src="/static/picture/001.gif" alt=""/>3. 捐款时可以发布挑衅宣言（30字以内）。<br/>
        <img src="/static/picture/001.gif" alt=""/>4. <a href="javascript:;" @click="$router.push('/fl')">前往福利院</a>可领取今日慈善福利（需完成发帖/回帖任务）。
      </em>
    </div>

    <div class="module-title">【今日榜首】</div>
    <div class="module-content">
      <template v-if="top">
        <template v-if="top.user_id && myId === top.user_id">
          恭喜！你就是今日慈善榜首，正接受全社区膜拜~<br>
        </template>
        <template v-else>
          榜首：<a href="javascript:;" @click="goUser(top.user_id)"><ntext :color="top.color || '#004299'">{{ top.nickname }}</ntext></a>
          捐款 <b style="color:#e05a00">{{ fmtNum(top.amount) }}</b> G币<br>
        </template>
        <span v-if="top.word" class="txt-fade">宣言：{{ top.word }}</span><br>
        膜拜数：<b>{{ top.worships }}</b>
        <a v-if="isLogin && !worshipped" href="javascript:;" @click="worship">[膜拜]</a>
        <span v-else-if="worshipped" class="txt-fade">(今日已膜拜)</span>
        <span v-else><a href="javascript:;" @click="$router.push('/login')">[登录后膜拜]</a></span>
      </template>
      <template v-else>
        今日还没有友友捐款，榜首位子空着哦~
      </template>
    </div>

    <div class="module-title">【我要捐款】</div>
    <div class="module-content">
      <template v-if="mine">
        你今日已捐款 <b style="color:#e05a00">{{ fmtNum(mine.amount) }}</b> G币<span v-if="mine.word">，宣言：{{ mine.word }}</span><br>
        <span class="txt-fade">每人每日仅可捐献一次，明天再来吧~</span>
      </template>
      <template v-else-if="isLogin">
        捐献慈善金额:
        <input type="number" v-model.number="form.amount" maxlength="9" min="1" size="10"/><br/>
        <div class="word-line">捐赠宣言：</div>
        <textarea v-model.trim="form.word" rows="3" maxlength="30" class="word-box"></textarea><br/>
        <input type="button" class="btn-anniu" value="捐款" :disabled="busy" @click="donate"/>
        <p style="color:#c00" v-if="msg">{{ msg }}</p>
        <p style="color:#1a9e1a" v-if="okMsg">{{ okMsg }}</p>
      </template>
      <template v-else>
        <a href="javascript:;" @click="$router.push('/login')">登录</a>后即可捐款上榜~
      </template>
    </div>

    <div class="module-title">【今日上榜】</div>
    <div class="module-content">
      <template v-if="list.length">
        <div v-for="(d, i) in list" :key="d.id" class="rank-row">
          <div>
            <template v-if="i === 0">
              <img src="/static/picture/001.gif" alt=""/><span class="rank-no"><b style="color:#e05a00">NO.1</b></span>
              <a href="javascript:;" @click="goUser(d.user_id)"><ntext :color="d.color || '#e05a00'">{{ d.nickname }}</ntext></a>
              <span class="rank-amt"><b style="color:#e05a00">{{ fmtNum(d.amount) }}</b>G币</span>
            </template>
            <template v-else>
              <span class="rank-no">NO.{{ i + 1 }}</span>
              <a href="javascript:;" @click="goUser(d.user_id)"><ntext :color="d.color || '#004299'">{{ d.nickname }}</ntext></a>
              <span class="rank-amt">{{ fmtNum(d.amount) }}G币</span>
            </template>
          </div>
          <div v-if="d.word" class="txt-fade rank-sub">「{{ d.word }}」</div>
          <div class="txt-fade rank-sub">膜拜 {{ d.worships }}</div>
        </div>
      </template>
      <template v-else>
        <span class="txt-fade">虚位以待，快来做今日慈善之星！</span>
      </template>
    </div>

    <div class="module-title">
      <a href="javascript:;" @click="$router.push('/home')">家园</a>&gt; <a href="javascript:;" @click="$router.push('/fl')">福利院</a>&gt;捐款慈善基金
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Fla',
  data () {
    return {
      list: [], mine: null, worshipped: false,
      form: { amount: 500000, word: '' },
      msg: '', okMsg: '', busy: false
    }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    myId () { return (this.$store.state.user && this.$store.state.user.id) || 0 },
    top () { return this.list.length ? this.list[0] : null }
  },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/fla').then(r => {
        if (r.code === 0) {
          this.list = r.data.list || []
          this.mine = r.data.mine
          this.worshipped = !!r.data.worshipped
        }
      })
    },
    donate () {
      const amt = Number(this.form.amount)
      if (!amt || amt < 1) { this.msg = '请输入捐款金额'; this.okMsg = ''; return }
      this.busy = true; this.msg = ''; this.okMsg = ''
      api.post('/fla/donate', { amount: amt, word: this.form.word }).then(r => {
        this.busy = false
        if (r.code === 0) {
          this.okMsg = '捐款成功！感谢你为家园慈善事业做出的贡献~'
          this.form.word = ''
          this.load()
        } else {
          this.msg = r.msg
        }
      }).catch(() => { this.busy = false; this.msg = '网络开小差了，稍后再试' })
    },
    worship () {
      api.post('/fla/worship').then(r => {
        if (r.code === 0) { this.worshipped = true; this.load() } else { this.msg = r.msg }
      })
    },
    goUser (id) { if (id) this.$router.push('/user/' + id) },
    fmtNum (n) { return Number(n || 0).toLocaleString() }
  }
}
</script>

<style scoped>
.module-title {
  padding: 5px;
  background: #f5f5f5;
  border-bottom: 1px solid #ddd;
  font-weight: bold;
}
.module-content {
  padding: 10px;
  background: #fff;
}
.txt-fade { color: #999; font-size: 12px; }
.module-content textarea.word-box {
  width: 120px !important;
  max-width: 120px !important;
  margin-left: 6px !important;
}
.word-line { margin: 10px 0 2px; }
.btn-anniu {
  width: 52px;
  height: 26px;
  margin-top: 8px;
  border: none;
  background: #0052d9;
  color: #fff;
  border-radius: 5px;
  font-size: 14px;
  padding: 2px 4px;
  box-shadow: 1px 1px 7px #fb3d3d;
  cursor: pointer;
}
.btn-anniu:disabled { opacity: .6; cursor: default; }
.rank-row {
  padding: 6px 0 7px;
  border-bottom: 1px dotted #dfe8f2;
  line-height: 1.9;
}
.rank-row:last-child { border-bottom: none; }
.rank-no { display: inline-block; min-width: 44px; }
.rank-amt { margin-left: 8px; }
.rank-sub { line-height: 1.6; margin-top: 1px; }
</style>
