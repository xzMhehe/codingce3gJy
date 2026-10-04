import Vue from 'vue'
import router from './router'
import store from './store'
import Ezfy from './views/ezfy/Ezfy.vue'

Vue.config.productionTip = false

// ★ 2026-10-04 二战独立入口（构建为 dist/ezfy.html，与家园 SPA 完全分离）：
//   · 不 import App.vue / assets/style.css —— 没有家园的顶栏/导航/页脚与全局样式，
//     刷新页面也不会再出现「家园影子」。
//   · 复用家园 router/store/api：登录态存 localStorage，接口鉴权走同一套 token。
//   · 家园相关路径（exitToHome 的「家园」、401 跳登录等）整页跳回家园站点。

// 拦截器：凡不是二战页的导航 → 整页跳回家园站点（避免在独立壳里加载家园页面）
router.beforeEach((to, from, next) => {
  if (to.path.indexOf('/games/ezfy') !== 0) {
    window.location.href = window.location.origin + to.fullPath
    return
  }
  next()
})

new Vue({
  router,
  store,
  render: h => h(Ezfy)
}).$mount('#app')
