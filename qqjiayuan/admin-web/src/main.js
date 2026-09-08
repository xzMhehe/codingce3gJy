import Vue from 'vue'
import ElementUI from 'element-ui'
import 'element-ui/lib/theme-chalk/index.css'
import App from './App.vue'
import router from './router'
import './style.css'

Vue.config.productionTip = false
Vue.use(ElementUI, { size: 'small' })
// 演示站图片素材路径
// 演示站图片素材路径
// 库存图片（管理端上传，存数据库 base64）：db/xxx.gif → /api/res/db/xxx.gif
Vue.prototype.$pic = f => {
  if (!f) return ''
  if (f.indexOf('db/') === 0) return '/api/res/' + f
  return '/static/picture/' + f
}

new Vue({
  router,
  render: h => h(App)
}).$mount('#app')
