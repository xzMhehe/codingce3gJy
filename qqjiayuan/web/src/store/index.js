import Vue from 'vue'
import Vuex from 'vuex'

Vue.use(Vuex)

export default new Vuex.Store({
  state: {
    token: localStorage.getItem('jy_token') || '',
    user: JSON.parse(localStorage.getItem('jy_user') || 'null'),
    // ★ 2026-10-12 二战独立登录：二战专属 token 单独存（与家园 token 彻底分开）。
    //   二战/家园拆开后，mutating 家园 token 不影响二战登录态。
    ezfyToken: localStorage.getItem('ezfy_token') || '',
    // 二战玩家身份（game_id=家园号, nickname），随二战登录写入；家园身份缺失时兜底用。
    ezfyUser: JSON.parse(localStorage.getItem('ezfy_user') || 'null'),
    unread: 0
  },
  getters: {
    isLogin: s => !!s.token,
    // 二战是否已登录（看二战专属 token），与家园登录态相互独立
    isEzfyLogin: s => !!s.ezfyToken,
    isAdmin: s => !!(s.user && s.user.perms && s.user.perms.indexOf('admin:access') >= 0)
  },
  mutations: {
    setUser (s, { token, user }) {
      s.token = token || s.token
      s.user = user
      if (token) localStorage.setItem('jy_token', token)
      if (user) localStorage.setItem('jy_user', JSON.stringify(user))
    },
    // 二战登录：只写 ezfyToken，不碰家园 token
    setEzfyToken (s, token) {
      s.ezfyToken = token || ''
      if (token) localStorage.setItem('ezfy_token', token)
      else localStorage.removeItem('ezfy_token')
    },
    // 二战身份（登录时一并写入）
    setEzfyUser (s, u) {
      s.ezfyUser = u || null
      if (u) localStorage.setItem('ezfy_user', JSON.stringify(u))
      else localStorage.removeItem('ezfy_user')
    },
    setUnread (s, n) { s.unread = n },
    logout (s) {
      // ★★ 2026-10-15 服务端作废：家园 token 和二战 token 都是无状态 JWT —— 只清本地不算数，
      //   脚本/旧页面拿着 token 还能继续打接口。退出前先带当前家园 token 通知服务端
      //   （POST /api/auth/logout，fire-and-forget）：服务端把 users.session_at 和
      //   ezfy_account.session_at 顶到当前 → 两个旧 token 之后全部 401 作废。
      const t = s.token
      if (t) {
        fetch('/api/auth/logout', { method: 'POST', headers: { Authorization: 'Bearer ' + t } }).catch(() => {})
      }
      s.token = ''
      s.user = null
      s.unread = 0
      localStorage.removeItem('jy_token')
      localStorage.removeItem('jy_user')
      // ★ 2026-10-12 家园退出 → 二战一并退出（用户：「家园退了 二战也要退，现在家园退了二战没退」）。
      //   二战 token 是**单独存**的（ezfy_token），只清家园会留下「家园已退、二战还登录着」的孤儿登录态。
      //   家园退出的全部入口都走这个 mutation（页脚「退出」、以及 api 拦截器里家园 401），改这一处即可全覆盖。
      //   ★ 反向不成立：二战登出（ezfyLogout）只清二战，不动家园登录态。
      s.ezfyToken = ''
      s.ezfyUser = null
      localStorage.removeItem('ezfy_token')
      localStorage.removeItem('ezfy_user')
    },
    // 二战登出：只清 ezfyToken 与 ezfyUser
    ezfyLogout (s) {
      s.ezfyToken = ''
      s.ezfyUser = null
      localStorage.removeItem('ezfy_token')
      localStorage.removeItem('ezfy_user')
    }
  },
  actions: {
    // 拉取未读家信数（top_nav 家信(N) = 私信未读，诺哈 wap_user_news.home）
    refreshUnread ({ commit, state }) {
      if (!state.token) return
      const token = state.token
      fetch('/api/messages/unread', { headers: { Authorization: 'Bearer ' + token } })
        .then(r => r.json())
        .then(d => { if (d && d.code === 0) commit('setUnread', (d.data && d.data.unread) || 0) })
        .catch(() => {})
    }
  }
})
