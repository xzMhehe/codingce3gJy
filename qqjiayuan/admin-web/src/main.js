import Vue from 'vue'
import ElementUI from 'element-ui'
import 'element-ui/lib/theme-chalk/index.css'
import App from './App.vue'
import router from './router'
import './style.css'

Vue.config.productionTip = false
Vue.use(ElementUI, { size: 'small' })
// 演示站图片素材路径
Vue.prototype.$pic = f => '/static/picture/' + f

new Vue({
  router,
  render: h => h(App)
}).$mount('#app')
