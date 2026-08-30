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
  }
})
