import Vue from 'vue'
import VueRouter from 'vue-router'
import store from '../store'

Vue.use(VueRouter)

const routes = [
  { path: '/', name: 'plaza', component: () => import('../views/Plaza.vue') },
  { path: '/login', name: 'login', component: () => import('../views/Login.vue') },
  { path: '/register', name: 'register', component: () => import('../views/Register.vue') },
  { path: '/find', name: 'find', component: () => import('../views/Find.vue') },
  { path: '/nav', name: 'nav', component: () => import('../views/Nav.vue') },
  { path: '/games', name: 'games', component: () => import('../views/Games.vue') },
  { path: '/home', name: 'myhome', component: () => import('../views/MyHome.vue'), meta: { auth: true } },
  { path: '/mood', name: 'mymood', component: () => import('../views/MyMood.vue'), meta: { auth: true } },
  { path: '/sign', name: 'sign', component: () => import('../views/Sign.vue'), meta: { auth: true } },
  { path: '/channel/:id', name: 'channel', component: () => import('../views/Channel.vue') },
  { path: '/board/:id', name: 'board', component: () => import('../views/Board.vue') },
  { path: '/thread/:id', name: 'thread', component: () => import('../views/Thread.vue') },
  { path: '/post/:boardId', name: 'post', component: () => import('../views/PostEdit.vue'), meta: { auth: true } },
  { path: '/user/:id', name: 'user', component: () => import('../views/UserPage.vue') },
  { path: '/profile', name: 'profile', component: () => import('../views/Profile.vue'), meta: { auth: true } },
  { path: '/friends', name: 'friends', component: () => import('../views/Friends.vue'), meta: { auth: true } },
  { path: '/messages', name: 'messages', component: () => import('../views/Messages.vue'), meta: { auth: true } },
  { path: '/messages/:peerId', name: 'messageChat', component: () => import('../views/Messages.vue'), meta: { auth: true } },
  { path: '/chat', name: 'chat', component: () => import('../views/ChatRoom.vue'), meta: { auth: true } },
  { path: '/space/:userId', name: 'space', component: () => import('../views/Space.vue') },
  { path: '/notices', name: 'notices', component: () => import('../views/Notices.vue'), meta: { auth: true } },
  { path: '/search', name: 'search', component: () => import('../views/Search.vue') },
  { path: '*', redirect: '/' }
]

const router = new VueRouter({ routes })

router.beforeEach((to, from, next) => {
  // 已登录再访问登录/注册页，直接回广场（退出登录后才会放行）
  if ((to.path === '/login' || to.path === '/register') && store.getters.isLogin) {
    return next('/')
  }
  if (to.meta.auth && !store.getters.isLogin) {
    return next('/login?redirect=' + encodeURIComponent(to.fullPath))
  }
  next()
})

export default router
