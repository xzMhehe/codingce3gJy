import Vue from 'vue'
import router from './router'
import store from './store'
import Ezfy from './views/ezfy/Ezfy.vue'

Vue.config.productionTip = false

// ★ 2026-10-17 二战独立部署：独立壳标记，共享的 router 导航守卫靠它区分「家园 SPA」与「二战独立壳」。
//   家园 SPA（index.html）访问二战页 → 守卫整页跳默认二战区服；独立壳（ezfy.html）→ 正常放行渲染。
window.__EZFY_SHELL__ = true

// ★ 2026-10-04 二战独立入口（构建为 dist/ezfy.html，与家园 SPA 完全分离）：
//   · 不 import App.vue / assets/style.css —— 没有家园的顶栏/导航/页脚与全局样式，
//     刷新页面也不会再出现「家园影子」。
//   · 复用家园 router/store/api：登录态存 localStorage，接口鉴权走同一套 token。
//   · 家园相关路径（exitToHome 的「家园」、401 跳登录等）整页跳回家园站点。

// 拦截器：凡不是二战页的导航 → 整页跳回家园站点（避免在独立壳里加载家园页面）
router.beforeEach((to, from, next) => {
  // ★ 2026-10-12 二战独立登录：未登录访问二战页时，主路由守卫会把 /games/ezfy 重定向到 /login。
  //   独立壳里 /login 不显示家园登录（根组件就是 Ezfy.vue）——放行它，由 Ezfy.vue 渲染二战自己的登录界面，
  //   避免被整页弹回家园，也不会和 /games/ezfy 之间来回重定向。
  if (to.path === '/login' || to.path === '/register') {
    return next()
  }
  if (to.path.indexOf('/games/ezfy') !== 0) {
    // ★★ 2026-10-19 修复无限刷新：原实现 `origin + fullPath` 在独立服上是「原地自跳」——
    //   直接打开 http://host:8081/#/（路由 /）会 location.href = origin + '/'，
    //   等于自己刷新自己，页面一直转圈。
    //   独立壳只承载二战游戏，非二战路径统一落到游戏入口，绝不回跳自身。
    window.location.href = window.location.origin + '/#/games/ezfy'
    return
  }
  next()
})

new Vue({
  router,
  store,
  render: h => h(Ezfy)
}).$mount('#app')
