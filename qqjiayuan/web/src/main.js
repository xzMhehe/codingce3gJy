import Vue from 'vue'
import App from './App.vue'
import router from './router'
import store from './store'
import './assets/style.css'

Vue.config.productionTip = false

// 演示站图片素材路径（static 已整体拷贝到 public/static）
// 库存图片（管理端上传，存数据库 base64）：db/xxx.gif → /api/res/db/xxx.gif
Vue.prototype.$pic = f => {
  if (!f) return ''
  if (f.indexOf('db/') === 0) return '/api/res/' + f
  return '/static/picture/' + f
}

new Vue({
  router,
  store,
  render: h => h(App)
}).$mount('#app')
