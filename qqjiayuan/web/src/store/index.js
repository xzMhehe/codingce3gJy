import Vue from 'vue'
import Vuex from 'vuex'

Vue.use(Vuex)

export default new Vuex.Store({
  state: {
    token: localStorage.getItem('jy_token') || '',
    user: JSON.parse(localStorage.getItem('jy_user') || 'null'),
    unread: 0
  },
  getters: {
    isLogin: s => !!s.token,
    isAdmin: s => !!(s.user && s.user.perms && s.user.perms.indexOf('admin:access') >= 0)
  },
  mutations: {
    setUser (s, { token, user }) {
      s.token = token || s.token
      s.user = user
      if (token) localStorage.setItem('jy_token', token)
      if (user) localStorage.setItem('jy_user', JSON.stringify(user))
    },
    setUnread (s, n) { s.unread = n },
    logout (s) {
      s.token = ''
      s.user = null
      s.unread = 0
      localStorage.removeItem('jy_token')
      localStorage.removeItem('jy_user')
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
