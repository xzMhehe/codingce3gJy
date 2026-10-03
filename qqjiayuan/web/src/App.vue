<template>
  <div id="app-shell">
    <!-- 顶部个人导航（复刻诺哈 Page_Login：号码 家信(N) 家园 空间）
         二战风云是沉浸式游戏页，进入时连同主导航一起隐藏 -->
    <div class="top_nav" >
      <template v-if="isLogin">
        <a href="javascript:;" @click="$router.push('/inbox')"><img :src="idIcon" alt="号码">{{ user.username }}</a>
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

    <!-- 主导航（复刻诺哈 crumb-nav-large：深蓝条 #71afe3，家园 好友 家族 广场 游戏，当前项 #98d2ff 高亮）
         二战风云是沉浸式游戏页，进入时隐藏这条导航 -->
    <div class="bar navbar" >
      <template v-for="n in navs">
        <span :key="n.name" v-if="isCurrent(n)" class="current">{{ n.name }}</span>
        <a :key="n.name + 'a'" v-else href="javascript:;" @click="$router.push(n.to)">{{ n.name }}</a>
      </template>
    </div>

    <router-view />

    <!-- 防抄袭：右键/保存/复制被拦截时的提示 -->
    <div class="anti-copy-toast" v-if="antiTip">{{ antiTip }}</div>

    <!-- 页脚（复刻诺哈 Page_Bottom：家园社区-广场-导航-聊天室-管理-退出 / 超Q.空间.家园.微博 / 小Q报时）
         二战风云是沉浸式游戏页：顶部个人导航、主导航条与页脚全部隐藏
         （2026-09-24 用户要求：去掉二战下面的家园导航，离开游戏走游戏内底部导航的「家园」）。 -->
    <div class="footer" >
      <p>
        <a href="javascript:;" @click="$router.push('/')">家园社区</a>-<a href="javascript:;" @click="$router.push('/')">广场</a>-<a href="javascript:;" @click="$router.push('/nav')">导航</a>-<a href="javascript:;" @click="$router.push('/chat')">聊天室</a>-<a href="javascript:;" @click="goAdmin">管理</a><a v-if="isLogin" href="javascript:;" @click="logoutOut">-退出</a><br>
        <template v-if="isLogin"><a href="javascript:;" @click="$router.push('/noble')">超Q({{ noble }})</a>.<a href="javascript:;" @click="$router.push('/space/'+user.id)">空间({{ spaceCount }})</a>.<a href="javascript:;" @click="$router.push('/messages')">家园({{ unread }})</a>.<a href="javascript:;" @click="$router.push('/notices')">微博({{ noticeUnread }})</a><br></template>
      </p>
      <p>
        小Q报时：{{ nowText }}<br>
        <template v-if="qqGroup">官方QQ群：<a :href="'https://qm.qq.com/cgi-bin/qm/qr?k=&jump_from=&group=' + qqGroup" target="_blank">{{ qqGroup }}</a><br></template>
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
      nowText: '', timer: null, pollTimer: null, spaceCount: 0, noticeUnread: 0, qqGroup: '',
      antiTip: '', antiTimer: null, antiEnabled: false,
      navs: [
        { name: '家园', to: '/home', keys: ['/home', '/mood', '/sign', '/profile', '/wallet', '/bag', '/security', '/achieve', '/home-level', '/invite', '/favorites', '/medals', '/guestbook', '/youquan'] },
        { name: '好友', to: '/friends', keys: ['/friends', '/contacts'] },
        { name: '家族', to: '/families', keys: ['/families', '/family'] },
        { name: '广场', to: '/', keys: ['/', '/channel', '/board', '/thread', '/replies', '/search', '/rank', '/threads', '/my-threads', '/post', '/fla'] },
        { name: '游戏', to: '/games', keys: ['/games', '/garden', '/play', '/marriage'] }
      ]
    }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    user () { return this.$store.state.user || {} },
    unread () { return this.$store.state.unread },
    noble () { return this.user.noble || 0 },
    // 头像前的超Q标志（复刻诺哈 Page_Login）：开通且未过期用 id.gif，未开通/已过期用 idw.gif
    idIcon () {
      const end = this.user.qq_end
      if (end && new Date(end).getTime() > Date.now()) return '/static/image/id.gif'
      return '/static/image/idw.gif'
    }
  },
  mounted () {
    this.tick()
    this.timer = setInterval(this.tick, 1000)
    this.loadSiteInfo()
    if (this.isLogin) {
      this.pollUnread()
      this.pollTimer = setInterval(this.pollUnread, 30000)
      this.refreshMe()
      this.loadSpaceCount()
    }
  },
  methods: {
    // ★ 全站防抄袭（三层）：拦快捷键保存 / 拦右键菜单 / 禁选中复制。
    //   说明：浏览器不允许 JS 改写「另存为」的结果，只能**阻止**保存；
    //   真正防抄靠的是内容走 /api 鉴权、SPA 静态 HTML 拿到的是空壳。
    setupAntiCopy () {
      this._antiKey = (e) => {
        const ctrl = e.ctrlKey || e.metaKey
        const isSave = e.key === 's' || e.key === 'S'
        if (ctrl && isSave) {
          e.preventDefault(); e.stopPropagation()
          this.showAntiTip('本页内容受保护，禁止保存')
          return false
        }
        return true
      }
      this._antiCtx = (e) => {
        e.preventDefault(); e.stopPropagation()
        this.showAntiTip('请勿复制本页内容')
        return false
      }
      this._antiCopy = (e) => {
        e.preventDefault(); e.stopPropagation()
        return false
      }
      window.addEventListener('keydown', this._antiKey, true)
      window.addEventListener('contextmenu', this._antiCtx, true)
      window.addEventListener('copy', this._antiCopy, true)
      // 禁选中（浏览器原生另存菜单/选中依赖 DOM 文本，user-select:none 能挡住普通复制）
      const st = document.createElement('style')
      st.id = 'anti-copy-style'
      st.textContent =
        'html,body,#app{user-select:none;-webkit-user-select:none;-moz-user-select:none;-ms-user-select:none}' +
        'input,textarea{user-select:text;-webkit-user-select:text}'
      document.head.appendChild(st)
    },
    showAntiTip (msg) {
      this.antiTip = msg
      if (this.antiTimer) clearTimeout(this.antiTimer)
      this.antiTimer = setTimeout(() => { this.antiTip = '' }, 2200)
    },
    // 页脚 QQ 群号（后台站点设置 qq_group，空则不展示）
    // ★ 2026-09-30 防复制/防保存网页开关移到站点设置：anti_copy=0 关闭，未配置默认开启
    loadSiteInfo () {
      api.get('/site-info').then(r => {
        if (r.code === 0 && r.data) {
          this.qqGroup = (r.data.qq_group || '').trim()
          this.antiEnabled = (r.data.anti_copy !== '0')
          if (this.antiEnabled) this.setupAntiCopy()
        }
      }).catch(() => {})
    },
    goNoble () {
      this.$router.push('/noble')
    },
    // 页脚"管理"：新窗口打开管理后台（诺哈 Page_Bottom 管理入口）
    goAdmin () {
      window.open(location.origin + '/admin-ui/')
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
      // 家信(N)/家园(N) 只统计私信未读（诺哈 wap_user_news.home = 未读私信）
      api.get('/messages/unread').then(r => {
        if (r.code === 0) this.$store.commit('setUnread', r.data.unread || 0)
      })
      // 微博(N) = 系统通知未读数
      api.get('/notifications').then(r => {
        if (r.code === 0) this.noticeUnread = (r.data && r.data.unread) || 0
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
    if (this.antiTimer) clearTimeout(this.antiTimer)
    if (this._antiKey) window.removeEventListener('keydown', this._antiKey, true)
    if (this._antiCtx) window.removeEventListener('contextmenu', this._antiCtx, true)
    if (this._antiCopy) window.removeEventListener('copy', this._antiCopy, true)
    const st = document.getElementById('anti-copy-style')
    if (st) st.remove()
  }
}
</script>
<style>
/* 防抄袭：右键/保存被拦截时的居中提示条 */
.anti-copy-toast {
  position: fixed;
  left: 50%;
  top: 16px;
  transform: translateX(-50%);
  background: rgba(0, 0, 0, .82);
  color: #ffd83d;
  padding: 8px 18px;
  border-radius: 20px;
  font-size: 13px;
  z-index: 99999;
  pointer-events: none;
  white-space: nowrap;
  box-shadow: 0 2px 8px rgba(0, 0, 0, .3);
}
</style>
