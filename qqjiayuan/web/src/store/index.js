import Vue from 'vue'
import Vuex from 'vuex'

Vue.use(Vuex)

// ★★ 2026-10-18 跨端口登录态通道（家园 8080 / 二战一区 8081 / 二战二区 8082）：
//   localStorage 按 **origin（协议+域名+端口）** 隔离 —— 家园在 8080 登录，8081 的
//   localStorage 里读不到 jy_token，于是「家园登录了进二战还要再登一次」「只登二战
//   家园没跟着登录」。cookie 只按 **域名+路径** 匹配、**不区分端口** → 拿它做兜底通道。
//   读写都以 localStorage 为主、cookie 为辅（老数据/旧浏览器不至于突然掉线）。
const COOKIE_TOKEN = 'jy_token'
function readCookie (name) {
  const m = document.cookie.match(new RegExp('(?:^|;\\s*)' + name + '=([^;]*)'))
  return m ? decodeURIComponent(m[1]) : ''
}
function writeCookie (name, val) {
  const base = '; path=/; SameSite=Lax'
  if (val) document.cookie = name + '=' + encodeURIComponent(val) + base + '; max-age=' + (7 * 86400)
  else document.cookie = name + '=' + base + '; max-age=0'
}

// ★★ 2026-10-18 从家园 token（JWT）里解出 uid（payload.uid）。
//   为什么不用 `$store.state.user.id`：跨端口时 user 存在**家园源**的 localStorage 里，
//   8081/8082 上 `user` 是空的（只有 cookie 里的 token 过来了）→ 拿不到 uid 就判断不了
//   「家园是不是换人了」，会出现「家园换成 B、二战还停在 A」的串台。
export function jwtUid (t) {
  if (!t) return 0
  try {
    const seg = String(t).split('.')[1]
    if (!seg) return 0
    const b64 = seg.replace(/-/g, '+').replace(/_/g, '/')
    const pad = b64.length % 4 ? '='.repeat(4 - (b64.length % 4)) : ''
    const o = JSON.parse(atob(b64 + pad))
    return (o && o.uid) || 0
  } catch (e) {
    return 0
  }
}

export default new Vuex.Store({
  state: {
    token: localStorage.getItem('jy_token') || readCookie(COOKIE_TOKEN) || '',
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
    isAdmin: s => !!(s.user && s.user.perms && s.user.perms.indexOf('admin:access') >= 0),
    // ★★ 2026-10-18 当前家园账号 uid（从 token 解，跨端口也拿得到）。
    //   用途：二战登录页锁定账号、判断「家园换人了 → 作废旧二战 token」。
    homeUid: s => jwtUid(s.token)
  },
  mutations: {
    // ★★ 2026-10-18 只写 token、不动 user：二战壳收到 URL 带来的 sso_token 时用它
    //   （`Ezfy.vue::ssoFromUrl`）。**原来这里 commit 的是一个根本不存在的 mutation**
    //   → Vuex 只打 warning 不写值 → state.token 还是空 → trySsoLogin() 首行直接 return
    //   → 「家园登录了进二战还要再登录一次」。这个 mutation 是那条链路的必需品。
    setToken (s, token) {
      s.token = token || ''
      if (token) {
        localStorage.setItem('jy_token', token)
        writeCookie(COOKIE_TOKEN, token)
      } else {
        localStorage.removeItem('jy_token')
        writeCookie(COOKIE_TOKEN, '')
      }
    },
    setUser (s, { token, user }) {
      s.token = token || s.token
      s.user = user
      if (token) {
        localStorage.setItem('jy_token', token)
        writeCookie(COOKIE_TOKEN, token)
      }
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
      // ★★ 2026-10-18 家园 token 的跨端口副本（cookie）一并清掉 —— 否则 8081/8082 上的
      //   二战页会从 cookie 里把已退出的家园 token 读回来，形成「退了又自动登回去」的幽灵态。
      writeCookie(COOKIE_TOKEN, '')
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
