<template>
  <div>
    <!-- 欢迎区（同参考页 plist） -->
    <div class="plist">
      <div class="row00">
        欢迎新童鞋: <a href="javascript:;" @click="$router.push('/user/'+plaza.newest_user.id)">{{ plaza.newest_user.nickname }}</a>加入3GQQ家园~
      </div>
      <div class="row00">
        30分钟内广场上共有<a href="javascript:;"> {{ plaza.online_count }} </a>位小哥哥小姐姐来来往往！<br>
      </div>
    </div>
    <div class="module-content">
      <font color="#ff0000">{{ isLogin ? user.nickname : '游客' }}</font>{{ greeting }}<br>
    </div>

    <div class="login-tips">
      <div v-for="a in announcements" :key="'a'+a.id">
        📢 <a href="javascript:;" @click="showAnn(a)">{{ a.title }}</a>
      </div>
    </div>

    累了吗？来<a href="javascript:;" @click="$router.push('/channel/3')">同城客栈</a>透个气吧！<br>
    <span v-if="tcSubs.length"><a href="javascript:;" @click="$router.push('/channel/3')">同城</a> <span v-for="s in tcSubs" :key="'tc'+s.id"><a href="javascript:;" @click="$router.push('/board/'+s.id)">{{ s.name }}</a>. </span></span>
    你可能还会喜欢以下论坛：<br>
    <span v-if="gtSubs.length"><span v-for="s in gtSubs" :key="'gt'+s.id"><a href="javascript:;" @click="$router.push('/board/'+s.id)">{{ s.name }}</a>. </span></span><a href="javascript:;" @click="$router.push('/channel/1')">&gt;&gt;</a><br>

    <div class="module-content">
      <form @submit.prevent="goSearch">
        社区搜索
        <input type="text" v-model.trim="word" size="15" maxlength="30" value="">
        <input type="submit" value="GO">
      </form>
    </div>

    <!-- 家园TV（社区头条） -->
    <div class="module-title">
      家园TV
    </div>
    <div class="box bline">
      <div v-for="t in plaza.fine_threads" :key="'f'+t.id">
        <a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}({{ t.view_count }}阅)</a><br>{{ excerpt(t.content) }}<br>
      </div>
      <a href="javascript:;" @click="$router.push('/channel/1')">更多热点&gt;&gt;</a><br>
    </div>

    <!-- T台秀 -->
    <div class="module-title">
      T台秀<br>
    </div>
    <div class="row" v-if="plaza.ttou && plaza.ttou.id">
      <table><tbody><tr>
        <td valign="top" align="center">
          <div class="tt-avatar" :style="ttBg">
            <img v-if="plaza.ttou.priv" :src="'/static/' + plaza.ttou.priv.file" class="tt-priv" :alt="plaza.ttou.priv.name">
          </div>
        </td>
        <td style="width:100%;padding-left:2px">
          膜拜：<a href="javascript:;" @click="$router.push('/user/'+plaza.ttou.id)"><font :color="plaza.ttou.color || '#ff0000'">{{ plaza.ttou.nickname }}</font>({{ plaza.ttou.username }})</a>
          <a href="javascript:;" @click="$router.push('/user/'+plaza.ttou.id)"><em style="color:#fff;font-size:12px;background:#71afe3;border-radius:3px;padding:0 3px">我要膜拜</em></a><br>
          <em>宣言：{{ plaza.ttou.signature || '这个佬佬很懒，什么也没有写。' }}</em><br>
        </td>
      </tr></tbody></table>
    </div>

    <!-- 欢乐坊 -->
    <div class="module-title">
      欢乐坊
    </div>
    <div class="module-content">
      <a href="javascript:;" @click="$router.push('/games')">宅子</a>.<a href="javascript:;" @click="$router.push('/sign')">打卡</a>.<a href="javascript:;" @click="$router.push('/sign')">工资</a>.<a href="javascript:;" @click="$router.push('/games')">挖宝</a>.<a href="javascript:;" @click="$router.push('/games')">幸运猜数字</a>
    </div>

    <!-- 游乐场 -->
    <div class="module-title">
      游乐场
    </div>
    <div class="module-content">
      <a href="javascript:;" @click="$router.push('/chat')">秘密</a>.<a href="javascript:;" @click="$router.push('/games')">打工</a>.<a href="javascript:;" @click="$router.push('/channel/4')">慈善基金</a>.<a href="javascript:;" @click="$router.push('/sign')">社区银行</a>.<a href="javascript:;" @click="$router.push('/games')">&gt;&gt;</a><br>
    </div>

    <!-- 最新发帖 -->
    <div class="module-title">
      <a href="javascript:;" @click="$router.push('/channel/1')">最新发帖</a><br>
    </div>
    <div class="module-content wid" v-for="(t,i) in plaza.quick_threads" :key="'q'+t.id">
      {{ i+1 }}.<a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a>({{ t.view_count }}阅/<a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.reply_count }}</a>回)<br>
    </div>

    <!-- 最新回帖 -->
    <div class="module-title">
      <a href="javascript:;" @click="$router.push('/channel/1')">最新回帖</a><br>
    </div>
    <div class="module-content wid" v-for="(t,i) in plaza.active_threads" :key="'at'+t.id">
      {{ i+1 }}.<a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a>({{ t.view_count }}阅/<a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.reply_count }}</a>回)<br>
    </div>

    <!-- 频道 -->
    <div class="module-title" v-for="ch in plaza.channels" :key="'ch'+ch.channel.id">
      <a href="javascript:;" @click="$router.push('/channel/'+ch.channel.id)">{{ ch.channel.name }}</a><br>
    </div>
    <div class="module-content">
      <span v-for="(s,i) in allSubs" :key="'s'+s.id"><a href="javascript:;" @click="$router.push('/board/'+s.id)">{{ s.name }}</a><template v-if="i < allSubs.length-1">.</template></span>
    </div>

    <div class="login-tips" v-if="annPopup">
      <b>{{ annPopup.title }}</b><br>{{ annPopup.content }}
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Plaza',
  data () {
    return {
      plaza: { newest_user: {}, fine_threads: [], channels: [], dynamics: [], quick_threads: [], active_threads: [], ttou: {} },
      word: '', annPopup: null
    }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    user () { return this.$store.state.user || {} },
    announcements () { return this.plaza.announcements || [] },
    tongcheng () { return (this.plaza.channels || []).find(c => c.channel.name === '同城客栈') },
    gongtan () { return (this.plaza.channels || []).find(c => c.channel.name === '公共论坛') },
    tcSubs () { return this.tongcheng ? this.tongcheng.subs.slice(0, 2) : [] },
    gtSubs () { return this.gongtan ? this.gongtan.subs.slice(0, 3) : [] },
    allSubs () {
      const out = []
      ;(this.plaza.channels || []).forEach(ch => (ch.subs || []).forEach(s => out.push(s)))
      return out
    },
    greeting () {
      const h = new Date().getHours()
      if (h < 6) return '夜深了，早点休息，梦里也能上家园！'
      if (h < 9) return '早上好，签到打卡再吃早饭哦！'
      if (h < 12) return '上午好，摸鱼就来广场浪一浪！'
      if (h < 14) return '中午好，吃饱了来家园消消食！'
      if (h < 18) return '下午好，灌灌水赛过活神仙！'
      if (h < 22) return '傍晚好,吃完饭来浪一浪赛过活神仙!'
      return '夜深了，睡前记得签个到！'
    },
    ttBg () {
      const a = this.plaza.ttou && this.plaza.ttou.avatar
      return a ? { background: 'url(/static/picture/' + a + ') 0 0/100% 100% no-repeat' } : { background: '#cfe0f0' }
    }
  },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/plaza').then(r => { if (r.code === 0) this.plaza = r.data })
    },
    goSearch () {
      if (this.word) this.$router.push('/search?word=' + encodeURIComponent(this.word))
    },
    showAnn (a) { this.annPopup = a },
    excerpt (s) {
      if (!s) return ''
      const t = s.replace(/\s+/g, ' ')
      return t.length > 38 ? t.slice(0, 38) + '...' : t
    }
  }
}
</script>
