import axios from 'axios'
import router from '../router'

const api = axios.create({ baseURL: '/api', timeout: 15000 })

// 登录态失效统一处理：清掉本地缓存并回登录页。
// ★★ 2026-10-21 后端未授权用的是 resp.Unauthorized（HTTP 200 + body.code=401），
//   不会走下面 axios 的 error 分支；两个分支都调用这里，避免「页面提示请重新登录但不跳转」。
function clearAuthAndRedirect () {
  localStorage.removeItem('jy_admin_token')
  localStorage.removeItem('jy_admin_user')
  if (router.currentRoute.path !== '/login') router.push('/login')
}

api.interceptors.request.use(cfg => {
  const token = localStorage.getItem('jy_admin_token')
  if (token) cfg.headers.Authorization = 'Bearer ' + token
  // ★ 2026-10-17 二战拆多区部署：家园服按 X-Ezfy-Zone 请求头反向代理到对应二战区服；
  //   不带该头时后端默认转发第一个区服。全量请求都带上也无害（非代理接口忽略此头）。
  const zone = localStorage.getItem('admin_ezfy_zone')
  if (zone) cfg.headers['X-Ezfy-Zone'] = zone
  return cfg
})

api.interceptors.response.use(
  resp => {
    const body = resp.data
    // ★ 被 IP 封禁时后端 302 到 /admin-ui/503.html，axios 跟随重定向后拿到 HTML 字符串。
    //   bug 教训：后端 404 时生产模式 NoRoute 也会兜底 index.html（同样是 HTML），
    //   不能见 HTML 就跳 503，必须看最终 URL 是否真落在 503.html 上。
    if (typeof body === 'string' && /<html/i.test(body)) {
      const finalURL = resp.request && (resp.request.responseURL || '')
      if (String(finalURL).indexOf('503.html') >= 0) {
        window.location.href = '/admin-ui/503.html'
        return { code: 503, msg: '服务不可用', data: null }
      }
      return { code: 404, msg: '请求的接口不存在或服务异常', data: null }
    }
    // ★★ 2026-10-21 token 失效（HTTP 200 + code:401）→ 清本地登录态并跳登录页
    if (body && body.code === 401) {
      clearAuthAndRedirect()
      return body
    }
    return body
  },
  err => {
    if (err.response && err.response.data) {
      const body = err.response.data
      if (body.code === 401) {
        clearAuthAndRedirect()
      }
      return Promise.resolve(body)
    }
    return Promise.resolve({ code: 500, msg: '网络开小差了，稍后再试', data: null })
  }
)

export default api
