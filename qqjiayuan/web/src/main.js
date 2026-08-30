import Vue from 'vue'
import App from './App.vue'
import router from './router'
import store from './store'
import './assets/style.css'

Vue.config.productionTip = false

// 演示站图片素材路径（static 已整体拷贝到 public/static）
Vue.prototype.$pic = f => '/static/picture/' + f

new Vue({
  router,
  store,
  render: h => h(App)
}).$mount('#app')
