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
  resp => resp.data,
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
