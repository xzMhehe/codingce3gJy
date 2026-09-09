<template>
  <div id="app-shell">
    <!-- 顶部个人导航（复刻诺哈 Page_Login：号码 家信(N) 家园 空间） -->
    <div class="top_nav">
      <template v-if="isLogin">
        <a href="javascript:;" @click="$router.push('/inbox')"><img src="/static/image/id.gif" alt="号码">{{ user.username }}</a>
        <a href="javascript:;" @click="$router.push('/messages')"><img src="/static/image/message.gif" alt="家信">家信({{ unread }})</a>
        <a href="javascript:;" @click="$router.push('/home')"><img src="/static/image/home.gif" alt="家园">家园</a>
        <a href="javascript:;" @click="$router.push('/space/'+user.id)"><img src="/static/image/blog.gif" alt="空间">空间</a>
        <a href="javascript:;" @click="goNoble"><img src="/static/image/vipqq.jpg" alt="超Q">超Q</a><a href="javascript:;" @click="$router.push('/box')">&gt;&gt;</a><br>
      </template>
      <template v-else>
        您尚未<a href="javascript:;" @click="$router.push('/login')">登录/注册</a><br>
      </template>
    </div>

    <!-- 未登录横幅 -->
    <div class="user-info" v-if="!isLogin">
      <a href="javascript:;" @click="$router.push('/login')">登陆家园社区</a>与好友互动、家族乐斗、最炫魔法花园、武林精武帮战、最牛游戏赢活动豪礼!<a href="javascript:;" @click="$router.push('/register')">注册&gt;&gt;&gt;</a>
    </div>

    <!-- 主导航（复刻诺哈 crumb-nav-large：深蓝条 #71afe3，家园 好友 家族 广场 游戏，当前项 #98d2ff 高亮） -->
    <div class="bar navbar">
      <template v-for="n in navs">
        <span :key="n.name" v-if="isCurrent(n)" class="current">{{ n.name }}</span>
        <a :key="n.name + 'a'" v-else href="javascript:;" @click="$router.push(n.to)">{{ n.name }}</a>
      </template>
      <a class="Rt" v-if="!isLogin" href="javascript:;" @click="$router.push('/login')">登陆</a>
    </div>

    <router-view />

    <!-- 页脚 -->
    <div class="footer">
      <p>
        <a href="javascript:;" @click="$router.push('/')">家园社区</a>-<a href="javascript:;" @click="$router.push('/')">广场</a>-<a href="javascript:;" @click="$router.push('/nav')">导航</a>-<a href="javascript:;" @click="$router.push('/chat')">聊天室</a>-<a href="javascript:;" onclick="window.open('http://'+location.host+'/admin-ui/')">管理</a>-<a v-if="isLogin" href="javascript:;" @click="logoutOut">退出</a><br>
        <template v-if="isLogin"><a href="javascript:;">超Q(0)</a>.<a href="javascript:;" @click="$router.push('/space/'+user.id)">空间({{ spaceCount }})</a>.<a href="javascript:;" @click="$router.push('/messages')">家园({{ unread }})</a>.<a href="javascript:;" @click="$router.push('/notices')">微博(0)</a><br></template>
      </p>
      <p>
        小Q报时：{{ nowText }}<br>
      </p>
    </div>
  </div>
</template>

<script>
import api from './api'

export default {
  name: 'App',
  data () {
    return {
      nowText: '', timer: null, pollTimer: null, spaceCount: 0,
      navs: [
        { name: '家园', to: '/home', keys: ['/home', '/mood', '/sign', '/profile', '/wallet', '/box', '/bag', '/security', '/achieve', '/home-level', '/invite', '/favorites', '/medals', '/guestbook', '/youquan'] },
        { name: '好友', to: '/friends', keys: ['/friends', '/contacts'] },
        { name: '家族', to: '/families', keys: ['/families', '/family', '/fla'] },
        { name: '广场', to: '/', keys: ['/', '/channel', '/board', '/thread', '/replies', '/search', '/rank', '/threads', '/my-threads', '/post'] },
        { name: '游戏', to: '/games', keys: ['/games', '/garden', '/play', '/noble', '/marriage'] }
      ]
    }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    user () { return this.$store.state.user || {} },
    unread () { return this.$store.state.unread }
  },
  mounted () {
    this.tick()
    this.timer = setInterval(this.tick, 1000)
    if (this.isLogin) {
      this.pollUnread()
      this.pollTimer = setInterval(this.pollUnread, 30000)
      this.refreshMe()
      this.loadSpaceCount()
    }
  },
  methods: {
    goNoble () {
      this.$router.push('/noble')
    },
    isCurrent (n) {
      const p = this.$route.path
      for (const k of n.keys) {
        if (k === '/' ? p === '/' : p.indexOf(k) === 0) return true
      }
      return false
    },
    tick () {
      const d = new Date()
      const p = n => (n < 10 ? '0' + n : '' + n)
      this.nowText = d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds())
    },
    loadSpaceCount () {
      api.get('/space/' + this.user.id).then(r => {
        if (r.code === 0 && r.data) {
          this.spaceCount = (r.data.mood_count || 0) + (r.data.article_count || 0)
        }
      }).catch(() => {})
    },
    pollUnread () {
      // 家信(N) 只统计私信未读（诺哈 wap_user_news.home = 未读私信）
      api.get('/messages/unread').then(r => {
        if (r.code === 0) this.$store.commit('setUnread', r.data.unread || 0)
      })
    },
    refreshMe () {
      api.get('/auth/me').then(r => {
        if (r.code === 0) this.$store.commit('setUser', { user: r.data })
      })
    },
    logoutOut () {
      this.$store.commit('logout')
      this.$router.push('/login')
    }
  },
  beforeDestroy () {
    clearInterval(this.timer)
    clearInterval(this.pollTimer)
  }
}
</script>
