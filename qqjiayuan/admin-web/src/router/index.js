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

router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('jy_admin_token')
  const user = JSON.parse(localStorage.getItem('jy_admin_user') || 'null')
  const hasPerm = user && user.perms && user.perms.indexOf('admin:access') >= 0
  if (to.path !== '/login' && (!token || !hasPerm)) {
    return next('/login')
  }
  if (to.path === '/login' && token && hasPerm) {
    return next('/')
  }
  next()
})

export default router
