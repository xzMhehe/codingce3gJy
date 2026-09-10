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
  { path: '/my-news', name: 'myNews', component: () => import('../views/HomeNews.vue'), props: { scope: 'mine' }, meta: { auth: true } },
  { path: '/friend-news', name: 'friendNews', component: () => import('../views/HomeNews.vue'), props: { scope: 'friend' }, meta: { auth: true } },
  { path: '/mood', name: 'mymood', component: () => import('../views/MyMood.vue'), meta: { auth: true } },
  { path: '/sign', name: 'sign', component: () => import('../views/Sign.vue'), meta: { auth: true } },
  { path: '/channel/:id', name: 'channel', component: () => import('../views/Channel.vue') },
  // 社区新帖/社区动态（复刻诺哈 topic_new/topic_reply）
  { path: '/threads/new', name: 'topicNew', component: () => import('../views/TopicNew.vue') },
  { path: '/threads/active', name: 'topicReply', component: () => import('../views/TopicReply.vue') },
  // 便民中心（复刻诺哈 wap/tool）
  { path: '/tool', name: 'tool', component: () => import('../views/Tool.vue') },
  { path: '/tool/:name', name: 'toolPage', component: () => import('../views/ToolPage.vue') },
  { path: '/tongcheng', name: 'tongcheng', component: () => import('../views/CityHome.vue') },
  { path: '/tongcheng/province/:id', name: 'cityList', component: () => import('../views/CityList.vue') },
  { path: '/tongcheng/city/:id', name: 'city', component: () => import('../views/City.vue') },
  { path: '/tongcheng/city/:id/online', name: 'cityOnline', component: () => import('../views/CityOnline.vue') },
  { path: '/board/:id', name: 'board', component: () => import('../views/Board.vue') },
  { path: '/thread/:id', name: 'thread', component: () => import('../views/Thread.vue') },
  { path: '/replies/:id', name: 'replies', component: () => import('../views/ReplyList.vue') },
  { path: '/my-threads', name: 'myThreads', component: () => import('../views/MyThreads.vue'), meta: { auth: true } },
  { path: '/threads/hot', name: 'hotThreads', component: () => import('../views/HotThreads.vue') },
  { path: '/post', name: 'postSelect', component: () => import('../views/PostSelect.vue'), meta: { auth: true } },
  { path: '/post/:boardId', name: 'post', component: () => import('../views/PostEdit.vue'), meta: { auth: true } },
  { path: '/user/:id', name: 'user', component: () => import('../views/UserPage.vue') },
  { path: '/profile', name: 'profile', component: () => import('../views/Profile.vue'), meta: { auth: true } },
  { path: '/friends', name: 'friends', component: () => import('../views/Friends.vue'), meta: { auth: true } },
  { path: '/messages', name: 'messages', component: () => import('../views/Messages.vue'), meta: { auth: true } },
  { path: '/messages/:peerId', name: 'messageChat', component: () => import('../views/Messages.vue'), meta: { auth: true } },
  { path: '/chat', name: 'chat', component: () => import('../views/ChatRoom.vue'), meta: { auth: true } },
  { path: '/play', name: 'play', component: () => import('../views/Play.vue'), meta: { auth: true } },
  { path: '/families', name: 'families', component: () => import('../views/Families.vue') },
  { path: '/families/category/:cat', name: 'familiesCategory', component: () => import('../views/FamiliesCategory.vue') },
  { path: '/families/search/:wd?', name: 'familiesSearch', component: () => import('../views/FamiliesSearch.vue') },
  { path: '/fla', name: 'fla', component: () => import('../views/Fla.vue') },
  { path: '/families/top', name: 'familiesTop', component: () => import('../views/FamiliesTop.vue') },
  { path: '/family/levels', name: 'familyLevels', component: () => import('../views/FamilyLevels.vue') },
  { path: '/family/:id', name: 'family', component: () => import('../views/Family.vue') },
  { path: '/family/:id/forum', name: 'familyForum', component: () => import('../views/FamilyForum.vue') },
  { path: '/family/:id/battle', name: 'familyBattle', component: () => import('../views/FamilyBattle.vue'), meta: { auth: true } },
  { path: '/family/:id/war', name: 'familyWar', component: () => import('../views/FamilyWar.vue'), meta: { auth: true } },
  { path: '/tip', name: 'tip', component: () => import('../views/Tip.vue') },
  { path: '/book', name: 'book', component: () => import('../views/Book.vue') },
  { path: '/book/:id', name: 'bookDetail', component: () => import('../views/BookDetail.vue') },
  { path: '/shelf', name: 'shelf', component: () => import('../views/Shelf.vue'), meta: { auth: true } },
  { path: '/store', name: 'store', component: () => import('../views/Store.vue'), meta: { auth: true } },
  { path: '/inbox', name: 'inbox', component: () => import('../views/Inbox.vue'), meta: { auth: true } },
  { path: '/groups', name: 'groups', component: () => import('../views/Group.vue'), meta: { auth: true } },
  { path: '/games/garden', name: 'garden', component: () => import('../views/Garden.vue'), meta: { auth: true } },
  { path: '/games/farm', name: 'farm', component: () => import('../views/Farm.vue'), meta: { auth: true } },
  { path: '/games/park', name: 'park', component: () => import('../views/Park.vue'), meta: { auth: true } },
  { path: '/noble', name: 'noble', component: () => import('../views/Noble.vue'), meta: { auth: true } },
  { path: '/box', name: 'box', component: () => import('../views/Box.vue'), meta: { auth: true } },
  { path: '/face', name: 'face', component: () => import('../views/Face.vue'), meta: { auth: true } },
  { path: '/shop', name: 'shop', component: () => import('../views/Shop.vue'), meta: { auth: true } },
  { path: '/money-shop', name: 'moneyShop', component: () => import('../views/MoneyShop.vue'), meta: { auth: true } },
  { path: '/bag', name: 'bag', component: () => import('../views/Bag.vue'), meta: { auth: true } },
  { path: '/rank', name: 'rank', component: () => import('../views/Rank.vue') },
  { path: '/activities', name: 'activities', component: () => import('../views/Activities.vue') },
  { path: '/security', name: 'security', component: () => import('../views/Security.vue'), meta: { auth: true } },
  { path: '/wallet', name: 'wallet', component: () => import('../views/Wallet.vue'), meta: { auth: true } },
  { path: '/youquan', name: 'youquan', component: () => import('../views/YouQuan.vue'), meta: { auth: true } },
  { path: '/home-level', name: 'homelevel', component: () => import('../views/HomeLevel.vue'), meta: { auth: true } },
  { path: '/achieve', name: 'achieve', component: () => import('../views/Achieve.vue'), meta: { auth: true } },
  { path: '/marriage', name: 'marriage', component: () => import('../views/Marriage.vue'), meta: { auth: true } },
  { path: '/space/:userId', name: 'space', component: () => import('../views/Space.vue') },
  { path: '/space/article/:id', name: 'blogArticle', component: () => import('../views/BlogArticle.vue') },
  { path: '/contacts', name: 'contacts', component: () => import('../views/Contacts.vue'), meta: { auth: true } },
  { path: '/favorites', name: 'favorites', component: () => import('../views/Favorites.vue'), meta: { auth: true } },
  { path: '/medals', name: 'medals', component: () => import('../views/Medals.vue'), meta: { auth: true } },
  { path: '/invite', name: 'invite', component: () => import('../views/Invite.vue'), meta: { auth: true } },
  { path: '/guestbook', name: 'guestbook', component: () => import('../views/Guestbook.vue') },
  { path: '/articles', name: 'articles', component: () => import('../views/Articles.vue') },
  { path: '/articles/:id', name: 'articleDetail', component: () => import('../views/ArticleDetail.vue') },
  { path: '/articles/edit', name: 'articleEdit', component: () => import('../views/ArticleEdit.vue'), meta: { auth: true } },
  { path: '/market', name: 'market', component: () => import('../views/Market.vue') },
  { path: '/market/:id', name: 'marketItem', component: () => import('../views/MarketItem.vue') },
  { path: '/market/sell', name: 'marketSell', component: () => import('../views/MarketSell.vue'), meta: { auth: true } },
  { path: '/market/orders', name: 'marketOrders', component: () => import('../views/MarketOrders.vue'), meta: { auth: true } },
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
