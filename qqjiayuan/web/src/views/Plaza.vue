<template>
  <div>
    <!-- 欢迎 / 在线 / 问候 -->
    <div class="plist" v-if="sec('welcome')">
      <div class="row00">欢迎新童鞋: <a href="javascript:;" @click="$router.push('/user/'+plaza.newest_user.id)">{{ plaza.newest_user.nickname }}</a>回归家园~</div>
      <div class="row00">30分钟内广场上共有<a href="javascript:;"> {{ plaza.online_count }} </a>位小姐姐来来往往！</div>
    </div>
    <div class="module-content" v-if="sec('greeting')"><font color="#ff0000">{{ isLogin ? user.nickname : '游客' }}</font>{{ greeting }}<br></div>

    <!-- 同城推荐 -->
    <template v-if="sec('tongcheng')">累了吗？来<a href="javascript:;" @click="$router.push('/channel/3')">同城客栈</a>透个气吧！<br>
    <span v-if="tcSubs.length"><a href="javascript:;" @click="$router.push('/channel/3')">同城</a> <span v-for="s in tcSubs" :key="'tc'+s.id"><a href="javascript:;" @click="$router.push('/board/'+s.id)">{{ s.name }}</a>.</span></span>
    你可能还会喜欢以下论坛：<br>
    <span v-if="gtSubs.length"><span v-for="s in gtSubs" :key="'gt'+s.id"><a href="javascript:;" @click="$router.push('/board/'+s.id)">{{ s.name }}</a>.</span></span><a href="javascript:;" @click="$router.push('/channel/1')">&gt;&gt;</a><br></template>

    <!-- 家园TV -->
    <div class="module-title" v-if="sec('tv')">家园TV</div>
    <div class="box bline" v-if="sec('tv')">
      <div v-for="t in plaza.fine_threads" :key="'f'+t.id">
        <a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}({{ t.view_count }}阅)</a><br>{{ excerpt(t.content) }}<br>
      </div>
      <a href="javascript:;" @click="$router.push('/channel/1')">更多热点&gt;&gt;</a><br>
    </div>

    <!-- T台秀 -->
    <div class="module-title" v-if="sec('tt')">T台秀</div>
    <div class="row" v-if="sec('tt') && plaza.ttou && plaza.ttou.id">
      <table><tbody><tr>
        <td valign="top" align="center">
          <div class="tt-avatar" :style="ttBg">
            <img v-if="plaza.ttou.priv" :src="'/static/' + plaza.ttou.priv.file" class="tt-priv" alt=".">
            <img v-if="plaza.ttou.level_icon" :src="$pic('v'+plaza.ttou.level_icon+'.gif')" class="tt-level" alt="等级">
            <img src="/static/picture/marksix_1.gif" class="tt-mark" alt="身份">
          </div>
        </td>
        <td style="width:100%;padding-left:2px">
          膜拜：<a href="javascript:;" @click="$router.push('/user/'+plaza.ttou.id)"><font :color="plaza.ttou.color || '#ff0000'">{{ plaza.ttou.nickname }}</font>({{ plaza.ttou.username }})</a>
          <a href="javascript:;" @click="$router.push('/profile')"><em style="color:#fff;font-size:12px;background:#71afe3;border-radius:3px;padding:0 3px">我要上榜</em></a><br>
          <em>宣言：{{ plaza.ttou.signature || '这个佬佬很懒，什么也没有写。' }}</em><br>
        </td>
      </tr></tbody></table>
    </div>
    <a href="javascript:;" @click="$router.push('/user/'+(plaza.ttou ? plaza.ttou.id : ''))" v-if="sec('tt') && plaza.ttou && plaza.ttou.id">我要膜拜</a><br>

    <!-- 欢乐坊（真实入口） -->
    <div class="module-title" v-if="sec('joy')">欢乐坊</div>
    <div class="module-content" v-if="sec('joy')">
      <a href="javascript:;" @click="$router.push('/play')">宅子</a>.<a href="javascript:;" @click="$router.push('/play')">每日星运</a>.<a href="javascript:;" @click="$router.push('/play')">挖宝</a>.<a href="javascript:;" @click="$router.push('/sign')">打卡</a>.<a href="javascript:;" @click="$router.push('/play')">幸运猜数字</a>
    </div>

    <!-- 游乐场（真实入口） -->
    <div class="module-title" v-if="sec('playground')">游乐场</div>
    <div class="module-content" v-if="sec('playground')">
      <a href="javascript:;" @click="$router.push('/chat')">秘密</a>.<a href="javascript:;" @click="$router.push('/play')">打工</a>.<a href="javascript:;" @click="$router.push('/play')">慈善基金</a>.<a href="javascript:;" @click="$router.push('/play')">社区银行</a>.<a href="javascript:;" @click="$router.push('/play')">&gt;&gt;</a><br>
    </div>

    <!-- 最新发帖 -->
    <div class="module-title" v-if="sec('newthread')"><a href="javascript:;" @click="$router.push('/channel/1')">最新发帖</a></div>
    <div class="module-content wid" v-if="sec('newthread')" v-for="(t,i) in plaza.quick_threads" :key="'q'+t.id">{{ i+1 }}.<a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a>({{ t.view_count }}阅/<a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.reply_count }}</a>回)</div>

    <!-- 最新回帖 -->
    <div class="module-title" v-if="sec('newreply')"><a href="javascript:;" @click="$router.push('/channel/1')">最新回帖</a></div>
    <div class="module-content wid" v-if="sec('newreply')" v-for="(t,i) in plaza.active_threads" :key="'at'+t.id">{{ i+1 }}.<a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a>({{ t.view_count }}阅/<a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.reply_count }}</a>回)</div>

    <!-- 公共论坛 / 同城客栈 / 家族天地 -->
    <template v-if="sec('channels') && mainChannels.length">
      <div v-for="ch in mainChannels" :key="'ch'+ch.channel.id">
        <div class="module-title">
          <a href="javascript:;" @click="$router.push('/channel/'+ch.channel.id)">{{ ch.channel.name }}</a>
        </div>
        <div class="module-content">
          <template v-for="(s,i) in ch.subs"><a :key="s.id" href="javascript:;" @click="$router.push('/board/'+s.id)">{{ s.name }}</a>{{ i < ch.subs.length-1 ? '.' : '' }}</template>
          <a href="javascript:;" @click="$router.push('/channel/'+ch.channel.id)"> 更多&gt;&gt;</a><br>
          <div v-for="t in ch.threads" :key="'t'+t.id" class="row00"><a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a>({{ t.view_count }}阅)</div>
        </div>
      </div>
    </template>

    <!-- 聊天大厅 -->
    <div class="module-title" v-if="sec('chat')"><a href="javascript:;" @click="$router.push('/chat')">聊天大厅</a></div>
    <div class="module-content" v-if="sec('chat')">
      <a href="javascript:;" @click="$router.push('/chat')">休闲灌水</a>.<a href="javascript:;" @click="$router.push('/chat')">同城北京</a>.<a href="javascript:;" @click="$router.push('/chat')">世外桃源</a>.<a href="javascript:;" @click="$router.push('/chat')">&gt;&gt;</a><br>
      <div v-for="(m,i) in chats" :key="'cf'+m.id">{{ i+1 }}.<a href="javascript:;" @click="$router.push('/user/'+(m.user?m.user.id:''))"><font :color="m.user?m.user.color:''">{{ m.user?m.user.nickname:'友友' }}</font></a>说:{{ brief(m.content) }} <span class="txt-fade">{{ m.created_at ? fmtShort(m.created_at) : '' }}</span><br></div>
      <form @submit.prevent="sendChat" v-if="isLogin">
        <input type="text" v-model.trim="chatWord" maxlength="100" size="16"><input type="submit" value="发言">
      </form>
    </div>

    <!-- 社区服务 -->
    <div class="box bline" v-if="sec('service')"><div class="module-title"><a href="javascript:;" @click="$router.push('/channel/1')">社区服务</a></div>
      <div class="module-content"><a href="javascript:;" @click="$router.push('/channel/4')">时报</a>.<a href="javascript:;" @click="$router.push('/channel/4')">民报</a>.<a href="javascript:;" @click="$router.push('/channel/1')">新人求助</a>.<a href="javascript:;" @click="$router.push('/channel/4')">更多&gt;&gt;</a></div>
    </div>

    <!-- 用户动态 -->
    <div class="module-title" v-if="sec('dynamics')">用户动态</div>
    <div class="module-content" v-if="sec('dynamics')">
      <div v-for="d in plaza.dynamics" :key="'d'+d.id">({{ ago(d.created_at) }})<a href="javascript:;" @click="$router.push('/user/'+d.user_id)"><font :color="d.color || '#004299'">{{ d.nickname }}</font></a>{{ d.action }}<a href="javascript:;" @click="$router.push('/thread/'+d.thread_id)">《{{ d.title }}》</a></div>
    </div>

    <!-- 搜索 / 更多 -->
    <div class="module-content" v-if="sec('search')"><form @submit.prevent="goSearch">搜搜 <input type="text" v-model.trim="word" class="ipt-txt" size="12" maxlength="30"><input type="submit" value="搜索"></form></div>

    <div class="login-tips" v-if="annPopup"><b>{{ annPopup.title }}</b><br>{{ annPopup.content }}</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Plaza',
  data () {
    return { plaza: { newest_user: {}, fine_threads: [], channels: [], dynamics: [], quick_threads: [], active_threads: [], ttou: {} }, word: '', annPopup: null, chats: [], chatWord: '', sections: {} }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    user () { return this.$store.state.user || {} },
    announcements () { return this.plaza.announcements || [] },
    mainChannels () { return (this.plaza.channels || []).filter(c => ['公共论坛', '同城客栈', '家族大厅'].indexOf(c.channel.name) >= 0) },
    tongcheng () { return (this.plaza.channels || []).find(c => c.channel.name === '同城客栈') },
    gongtan () { return (this.plaza.channels || []).find(c => c.channel.name === '公共论坛') },
    tcSubs () { return this.tongcheng ? this.tongcheng.subs.slice(0, 2) : [] },
    gtSubs () { return this.gongtan ? this.gongtan.subs.slice(0, 3) : [] },
    greeting () {
      const h = new Date().getHours()
      if (h < 6) return '夜深了，早点休息，梦里也能上家园！'
      if (h < 9) return '早上好，签到打卡再吃早饭哦！'
      if (h < 12) return '上午好，摸鱼就来广场浪一浪！'
      if (h < 14) return '中午好，吃饱了来家园消消食！'
      if (h < 18) return '下午好，爱你哟,么么哒~!'
      if (h < 22) return '傍晚好,吃完饭来浪一浪~!'
      return '夜深了，睡前记得签个到！'
    },
    ttBg () {
      const a = this.plaza.ttou && this.plaza.ttou.avatar
      return a ? { background: 'url(/static/picture/' + a + ') 0 0/100% 100% no-repeat' } : { background: '#cfe0f0' }
    }
  },
  mounted () { this.load(); this.loadChat(); this.loadSections() },
  methods: {
    sec (key) { return (this.sections[key] !== undefined ? this.sections[key] : 1) === 1 },
    loadSections () {
      api.get('/plaza-sections').then(r => {
        if (r.code === 0) {
          const m = {}
          ;(r.data || []).forEach(s => { m[s.key] = s.enabled })
          this.sections = m
        }
      })
    },
    load () {
      api.get('/plaza').then(r => { if (r.code === 0) this.plaza = r.data })
    },
    loadChat () {
      api.get('/chat').then(r => { if (r.code === 0) this.chats = (r.data || [] ).slice(-6) })
    },
    sendChat () {
      if (!this.chatWord) return
      api.post('/chat', { content: this.chatWord }).then(r => { if (r.code === 0) { this.chatWord = ''; this.loadChat() } })
    },
    goSearch () { if (this.word) this.$router.push('/search?word=' + encodeURIComponent(this.word)) },
    showAnn (a) { this.annPopup = a },
    brief (s) { s = s || ''; return s.length > 20 ? s.slice(0, 20) + '…' : s },
    excerpt (s) {
      if (!s) return ''
      const t = s.replace(/\s+/g, ' ')
      return t.length > 38 ? t.slice(0, 38) + '...' : t
    },
    ago (t) {
      if (!t) return ''
      const diff = Math.max(0, (Date.now() - new Date(t).getTime()) / 1000)
      if (diff < 60) return Math.floor(diff) + '秒前'
      if (diff < 3600) return Math.floor(diff / 60) + '分钟前'
      if (diff < 86400) return Math.floor(diff / 3600) + '小时前'
      return Math.floor(diff / 86400) + '天前'
    },
    fmtShort (t) {
      if (!t) return ''
      const d = new Date(t)
      return d.getMonth() + 1 + '/' + d.getDate() + ' ' + d.getHours() + ':' + (d.getMinutes() < 10 ? '0' : '') + d.getMinutes()
    }
  }
}
</script>
