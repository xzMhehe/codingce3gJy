import axios from 'axios'
import store from '../store'
import router from '../router'

const api = axios.create({ baseURL: '/api', timeout: 15000 })

api.interceptors.request.use(cfg => {
  const token = store.state.token
  if (token) cfg.headers.Authorization = 'Bearer ' + token
  return cfg
})

api.interceptors.response.use(
  resp => {
    const body = resp.data
    // ★ 2026-10-02 手机端"进去数据没刷新出来"根因修复：
    //   后端 resp.Unauthorized 返回的是 **HTTP 200 + JSON code 401**（不是 HTTP 401），
    //   原 401 判断写在 axios 错误分支里只对 HTTP 4xx/5xx 生效，业务码 401 永远到不了，
    //   于是登录过期的 token 请求 /view 时前端静默失败，首页全 0、也不跳登录页
    //   （无痕模式是新 token 所以正常）。这里统一处理业务码 401：登出并跳登录页。
    if (body && typeof body === 'object' && body.code === 401) {
      store.commit('logout')
      router.push('/login')
      return body
    }
    // ★ 被 IP 封禁时后端 302 到 /503.html，axios 跟随重定向后拿到 HTML 字符串。
    //   bug 教训：后端 404 时生产模式 NoRoute 也会兜底 index.html（同样是 HTML），
    //   不能见 HTML 就跳 503（点「一键卸下」等新接口在旧二进制上 404 会误跳 503 页），
    //   必须看最终 URL 是否真落在 503.html 上。
    if (typeof body === 'string' && /<html/i.test(body)) {
      const finalURL = resp.request && (resp.request.responseURL || '')
      if (String(finalURL).indexOf('503.html') >= 0) {
        window.location.href = '/503.html'
        return { code: 503, msg: '服务不可用', data: null }
      }
      return { code: 404, msg: '请求的接口不存在或服务异常', data: null }
    }
    return body
  },
  err => {
    if (err.response && err.response.data) {
      const body = err.response.data
      if (body.code === 401) {
        store.commit('logout')
        router.push('/login')
      }
      return Promise.resolve(body)
    }
    return Promise.resolve({ code: 500, msg: '网络开小差了，稍后再试', data: null })
  }
)

export default api
