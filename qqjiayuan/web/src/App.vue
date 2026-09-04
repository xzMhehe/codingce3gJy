<template>
  <div id="app-shell">
    <!-- 顶部个人导航（同真实  top_nav） -->
    <div class="top_nav">
      <template v-if="isLogin">
        <a href="javascript:;" @click="$router.push('/inbox')"><img src="/static/image/id.gif" alt="家信">{{ user.username }}</a><a href="javascript:;" @click="$router.push('/home')"><img src="/static/image/home.gif" alt="家园">家园({{ unread }})</a><a href="javascript:;" @click="$router.push('/space/'+user.id)"><img src="/static/image/blog.gif" alt="空间">空间({{ spaceCount }})</a><a href="javascript:;" @click="goNoble"><img src="/static/image/vipqq.jpg" alt="超Q">超Q</a><a href="javascript:;" @click="$router.push('/box')">&gt;&gt;</a><br>
      </template>
      <template v-else>
        <a href="javascript:;" @click="$router.push('/login')"><img src="/static/image/id.gif" alt="号码">登陆</a><a href="javascript:;" @click="$router.push('/register')"><img src="/static/image/home.gif" alt="家信">注册</a><a href="javascript:;" @click="$router.push('/find')"><img src="/static/image/blog.gif" alt="空间">找回</a><a href="javascript:;" @click="$router.push('/login')"><img src="/static/image/vipqq.jpg" alt="超Q">超Q</a><a href="javascript:;" @click="$router.push('/nav')">&gt;&gt;</a><br>
      </template>
    </div>

    <div class="user-info" v-else>
      <a href="javascript:;" @click="$router.push('/login')">登陆家园社区</a>与好友互动、家族乐斗、最炫魔法花园、武林精武帮战、最牛游戏赢活动豪礼!<a href="javascript:;" @click="$router.push('/register')">注册&gt;&gt;&gt;</a>
    </div>

    <!-- 主导航（仅 家园 好友 家族 广场 游戏，管理员多一个 管理） -->
    <div class="bar navbar">
      <a href="javascript:;" @click="$router.push(isLogin ? '/home' : '/login')">家园</a>
      <a href="javascript:;" @click="$router.push('/friends')">好友</a>
      <a href="javascript:;" @click="$router.push('/families')">家族</a>
      <a href="javascript:;" @click="$router.push('/')">广场</a>
      <a href="javascript:;" @click="$router.push('/games')">游戏</a>
      <a class="Rt" v-if="isLogin" href="javascript:;" @click="logoutOut">退出</a>
      <a class="Rt" v-else href="javascript:;" @click="$router.push('/login')">登陆</a>
    </div>

    <router-view />

    <!-- 页脚 -->
    <div class="footer">
      <p>
        <a href="javascript:;" @click="$router.push('/')">家园社区</a>-<a href="javascript:;" @click="$router.push('/')">广场</a>-<a href="javascript:;" @click="$router.push('/nav')">导航</a>-<a href="javascript:;" @click="$router.push('/chat')">聊天室</a>-<a href="javascript:;" onclick="window.open('http://'+location.host+'/admin-ui/')">管理</a><br>
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
    return { nowText: '', timer: null, pollTimer: null, spaceCount: 0 }
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
      api.get('/notifications?page=1').then(r => {
        if (r.code === 0 && r.data) this.$store.commit('setUnread', r.data.unread || 0)
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
