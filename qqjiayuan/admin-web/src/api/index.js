import axios from 'axios'
import router from '../router'

const api = axios.create({ baseURL: '/api', timeout: 15000 })

api.interceptors.request.use(cfg => {
  const token = localStorage.getItem('jy_admin_token')
  if (token) cfg.headers.Authorization = 'Bearer ' + token
  return cfg
})

api.interceptors.response.use(
  resp => {
    // 被 IP 封禁时后端 302 到 /admin-ui/503.html，axios 跟随重定向后拿到的是 HTML 字符串
    if (typeof resp.data === 'string' && /<html/i.test(resp.data)) {
      window.location.href = '/admin-ui/503.html'
      return { code: 503, msg: '服务不可用', data: null }
    }
    return resp.data
  },
  err => {
    if (err.response && err.response.data) {
      const body = err.response.data
      if (body.code === 401) {
        localStorage.removeItem('jy_admin_token')
        localStorage.removeItem('jy_admin_user')
        if (router.currentRoute.path !== '/login') router.push('/login')
      }
      return Promise.resolve(body)
    }
    return Promise.resolve({ code: 500, msg: '网络开小差了，稍后再试', data: null })
  }
)

export default api
