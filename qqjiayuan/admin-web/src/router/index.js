import Vue from 'vue'
import VueRouter from 'vue-router'

Vue.use(VueRouter)

const router = new VueRouter({
  routes: [
    { path: '/login', name: 'login', component: () => import('../views/Login.vue') },
    { path: '/', name: 'dashboard', component: () => import('../views/Dashboard.vue') },
    { path: '*', redirect: '/' }
  ]
})

// ---------------------------------------------------------------------------
// 同 web/src/router/index.js：vue-router 3.4 起 push()/replace() 返回 Promise，
// 导航被守卫重定向时会 reject（例如未登录访问受保护页面 → 跳 /login）。
// 这是预期流程，但调用处都没 catch，会在控制台报
//   Redirected when going from "..." to "/login" via a navigation guard.
// 这里在原型上统一吞掉「导航类失败」，其余真实错误照旧抛出。
function isNavFailure (err) {
  if (!err) return false
  if (typeof VueRouter.isNavigationFailure === 'function' && VueRouter.isNavigationFailure(err)) {
    return true
  }
  const name = err.name || ''
  const msg = err.message || ''
  return name === 'NavigationDuplicated' ||
    /Redirected when going from|Navigation (?:cancelled|aborted)|Avoided redundant navigation/.test(msg)
}

function swallowNavFailure (promise) {
  return promise.catch(err => {
    if (isNavFailure(err)) return err
    throw err
  })
}

const rawPush = VueRouter.prototype.push
VueRouter.prototype.push = function push (location, onResolve, onReject) {
  if (onResolve || onReject) return rawPush.call(this, location, onResolve, onReject)
  return swallowNavFailure(rawPush.call(this, location))
}

const rawReplace = VueRouter.prototype.replace
VueRouter.prototype.replace = function replace (location, onResolve, onReject) {
  if (onResolve || onReject) return rawReplace.call(this, location, onResolve, onReject)
  return swallowNavFailure(rawReplace.call(this, location))
}

router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('jy_admin_token')
  const user = JSON.parse(localStorage.getItem('jy_admin_user') || 'null')
  const hasPerm = user && user.perms && user.perms.some(p => p && p.indexOf('module:') === 0)
  if (to.path !== '/login' && (!token || !hasPerm)) {
    return next('/login')
  }
  if (to.path === '/login' && token && hasPerm) {
    return next('/')
  }
  next()
})

export default router
