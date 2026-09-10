/**
 * 管理端统一菜单/页面配置（唯一维护点）
 * 新增管理模块：1) 在 components/admin/ 新建组件 2) 在下面菜单树挂一条 { key, name, icon, component }
 * App.vue 的侧边栏/标签导航和 Dashboard.vue 的页面分发都由本文件自动生成，不再手工双写。
 */
import AdminUsers from './components/admin/AdminUsers.vue'
import AdminHome from './components/admin/AdminHome.vue'
import AdminUserDocu from './components/admin/AdminUserDocu.vue'
import AdminUserContacts from './components/admin/AdminUserContacts.vue'
import AdminUserAddresses from './components/admin/AdminUserAddresses.vue'
import AdminUserProtections from './components/admin/AdminUserProtections.vue'
import AdminUserLogs from './components/admin/AdminUserLogs.vue'
import AdminUserPhones from './components/admin/AdminUserPhones.vue'
import AdminInvites from './components/admin/AdminInvites.vue'
import AdminWallet from './components/admin/AdminWallet.vue'
import AdminWalletLogs from './components/admin/AdminWalletLogs.vue'
import AdminBoards from './components/admin/AdminBoards.vue'
import AdminTongcheng from './components/admin/AdminTongcheng.vue'
import AdminBoardCategories from './components/admin/AdminBoardCategories.vue'
import AdminThreads from './components/admin/AdminThreads.vue'
import AdminThreadRecycle from './components/admin/AdminThreadRecycle.vue'
import AdminWordFilters from './components/admin/AdminWordFilters.vue'
import AdminSiteArticles from './components/admin/AdminSiteArticles.vue'
import AdminGuestbook from './components/admin/AdminGuestbook.vue'
import AdminMessages from './components/admin/AdminMessages.vue'
import AdminSpaces from './components/admin/AdminSpaces.vue'
import AdminHomes from './components/admin/AdminHomes.vue'
import AdminVisitors from './components/admin/AdminVisitors.vue'
import AdminShops from './components/admin/AdminShops.vue'
import AdminShopGoods from './components/admin/AdminShopGoods.vue'
import AdminGoods from './components/admin/AdminGoods.vue'
import AdminMoneyShop from './components/admin/AdminMoneyShop.vue'
import AdminShopOrders from './components/admin/AdminShopOrders.vue'
import AdminShopComments from './components/admin/AdminShopComments.vue'
import AdminBooks from './components/admin/AdminBooks.vue'
import AdminBookChapters from './components/admin/AdminBookChapters.vue'
import AdminBookComments from './components/admin/AdminBookComments.vue'
import AdminPrivileges from './components/admin/AdminPrivileges.vue'
import AdminAnnouncements from './components/admin/AdminAnnouncements.vue'
import AdminGames from './components/admin/AdminGames.vue'
import AdminBadges from './components/admin/AdminBadges.vue'
import AdminGardenActivities from './components/admin/AdminGardenActivities.vue'
import AdminGardenSeeds from './components/admin/AdminGardenSeeds.vue'
import AdminGardenMaps from './components/admin/AdminGardenMaps.vue'
import AdminGardenMixes from './components/admin/AdminGardenMixes.vue'
import AdminGardenElves from './components/admin/AdminGardenElves.vue'
import AdminGardenSign from './components/admin/AdminGardenSign.vue'
import AdminGardenData from './components/admin/AdminGardenData.vue'
import AdminFarmSeeds from './components/admin/AdminFarmSeeds.vue'
import AdminFarmItems from './components/admin/AdminFarmItems.vue'
import AdminFarmData from './components/admin/AdminFarmData.vue'
import AdminParkCars from './components/admin/AdminParkCars.vue'
import AdminParkData from './components/admin/AdminParkData.vue'
import AdminSiteConfig from './components/admin/AdminSiteConfig.vue'
import AdminRoles from './components/admin/AdminRoles.vue'
import AdminResources from './components/admin/AdminResources.vue'

// 菜单树：最多两级分组（children 里还可带一层 children，如游戏管理下的各游戏）
export const menu = [
  { key: 'dashboard', name: '数据概览', icon: 'el-icon-data-board' },
  // 会员管理（诺哈 user/）
  {
    key: 'g-user', name: '会员管理', icon: 'el-icon-user',
    children: [
      { key: 'users', name: '会员列表', icon: 'el-icon-s-custom', component: AdminUsers },
      { key: 'home', name: '会员资料', icon: 'el-icon-edit-outline', component: AdminHome },
      { key: 'userDocu', name: '会员证件', icon: 'el-icon-postcard', component: AdminUserDocu },
      { key: 'userContact', name: '会员联系', icon: 'el-icon-phone-outline', component: AdminUserContacts },
      { key: 'userAddress', name: '会员地址', icon: 'el-icon-map-location', component: AdminUserAddresses },
      { key: 'userProtec', name: '会员密保', icon: 'el-icon-key', component: AdminUserProtections },
      { key: 'userLogs', name: '会员日志', icon: 'el-icon-date', component: AdminUserLogs },
      { key: 'userPhones', name: '手机审核', icon: 'el-icon-mobile-phone', component: AdminUserPhones },
      { key: 'invites', name: '会员推荐', icon: 'el-icon-share', component: AdminInvites }
    ]
  },
  // 货币管理（诺哈 money/）
  {
    key: 'g-money', name: '货币管理', icon: 'el-icon-coin',
    children: [
      { key: 'wallet', name: '会员财务', icon: 'el-icon-wallet', component: AdminWallet },
      { key: 'walletLogs', name: '货币流水', icon: 'el-icon-tickets', component: AdminWalletLogs }
    ]
  },
  // 社区管理（诺哈 bbs/）
  {
    key: 'g-bbs', name: '社区管理', icon: 'el-icon-chat-dot-round',
    children: [
      { key: 'boards', name: '版块管理', icon: 'el-icon-menu', component: AdminBoards },
      { key: 'tongcheng', name: '同城管理', icon: 'el-icon-location-outline', component: AdminTongcheng },
      { key: 'boardCategories', name: '版块分类', icon: 'el-icon-collection', component: AdminBoardCategories },
      { key: 'threads', name: '帖子管理', icon: 'el-icon-document', component: AdminThreads },
      { key: 'recycle', name: '恢复帖子', icon: 'el-icon-delete', component: AdminThreadRecycle },
      { key: 'wordFilters', name: '黑名单榜', icon: 'el-icon-remove-outline', component: AdminWordFilters }
    ]
  },
  // 内容管理（诺哈 article/ guest/ message/）
  {
    key: 'g-content', name: '内容管理', icon: 'el-icon-notebook-2',
    children: [
      { key: 'articles', name: '文章管理', icon: 'el-icon-notebook-1', component: AdminSiteArticles },
      { key: 'guestbook', name: '留言本管理', icon: 'el-icon-edit', component: AdminGuestbook },
      { key: 'messages', name: '家信管理', icon: 'el-icon-message', component: AdminMessages }
    ]
  },
  // 博客管理（诺哈 blog/）
  {
    key: 'g-blog', name: '博客管理', icon: 'el-icon-s-shop',
    children: [
      { key: 'spaces', name: '空间列表', icon: 'el-icon-office-building', component: AdminSpaces }
    ]
  },
  // 家园管理（诺哈 home/）
  {
    key: 'g-home', name: '家园管理', icon: 'el-icon-house',
    children: [
      { key: 'homes', name: '家园列表', icon: 'el-icon-s-home', component: AdminHomes },
      { key: 'visitors', name: '家园访客', icon: 'el-icon-view', component: AdminVisitors }
    ]
  },
  // 商城管理（诺哈 shop/ money-shop 道具商城/货币商店）
  {
    key: 'g-shop', name: '商城管理', icon: 'el-icon-shopping-cart-2',
    children: [
      { key: 'shops', name: '店铺管理', icon: 'el-icon-s-shop', component: AdminShops },
      { key: 'shopGoods', name: '商品管理', icon: 'el-icon-box', component: AdminShopGoods },
      { key: 'shopOrders', name: '订单管理', icon: 'el-icon-s-order', component: AdminShopOrders },
      { key: 'shopComments', name: '评论管理', icon: 'el-icon-s-comment', component: AdminShopComments },
      { key: 'goods', name: '道具商城', icon: 'el-icon-goods', component: AdminGoods },
      { key: 'moneyShop', name: '货币商店', icon: 'el-icon-coin', component: AdminMoneyShop }
    ]
  },
  // 书城管理（诺哈 book/）
  {
    key: 'g-book', name: '书城管理', icon: 'el-icon-reading',
    children: [
      { key: 'books', name: '小说列表', icon: 'el-icon-notebook-1', component: AdminBooks },
      { key: 'bookChapters', name: '章节管理', icon: 'el-icon-collection-tag', component: AdminBookChapters },
      { key: 'bookComments', name: '书评管理', icon: 'el-icon-chat-line-square', component: AdminBookComments }
    ]
  },
  // 会员特权（诺哈 vip/）
  {
    key: 'g-vip', name: '会员特权', icon: 'el-icon-s-operation',
    children: [
      { key: 'privileges', name: '特权管理', icon: 'el-icon-star-off', component: AdminPrivileges }
    ]
  },
  // 广播管理（诺哈 radio/）
  {
    key: 'g-radio', name: '广播管理', icon: 'el-icon-bell',
    children: [
      { key: 'announcements', name: '公告广播', icon: 'el-icon-bell', component: AdminAnnouncements }
    ]
  },
  // 游戏管理（诺哈 game/，按游戏分组）
  {
    key: 'g-game', name: '游戏管理', icon: 'el-icon-magic-stick',
    children: [
      { key: 'games', name: '游戏大厅', icon: 'el-icon-trophy', component: AdminGames },
      { key: 'badges', name: '勋章管理', icon: 'el-icon-medal', component: AdminBadges },
      {
        key: 'g-garden', name: '魔法花园', icon: 'el-icon-sunny',
        children: [
          { key: 'gardenActivities', name: '花园活动', icon: 'el-icon-magic-stick', component: AdminGardenActivities },
          { key: 'gardenSeeds', name: '花园花种', icon: 'el-icon-sunny', component: AdminGardenSeeds },
          { key: 'gardenMaps', name: '花之图谱', icon: 'el-icon-picture', component: AdminGardenMaps },
          { key: 'gardenMixes', name: '合成配方', icon: 'el-icon-s-cooperation', component: AdminGardenMixes },
          { key: 'gardenElves', name: '精灵花册', icon: 'el-icon-star-on', component: AdminGardenElves },
          { key: 'gardenSign', name: '签到管理', icon: 'el-icon-date', component: AdminGardenSign },
          { key: 'gardenData', name: '花园数据管理', icon: 'el-icon-data-analysis', component: AdminGardenData }
        ]
      },
      {
        key: 'g-farm', name: '开心农场', icon: 'el-icon-cherry',
        children: [
          { key: 'farmSeeds', name: '农场种子', icon: 'el-icon-suitcase-1', component: AdminFarmSeeds },
          { key: 'farmItems', name: '化肥陷阱', icon: 'el-icon-coin', component: AdminFarmItems },
          { key: 'farmData', name: '农场数据管理', icon: 'el-icon-data-analysis', component: AdminFarmData }
        ]
      },
      {
        key: 'g-park', name: '抢车位', icon: 'el-icon-truck',
        children: [
          { key: 'parkCars', name: '车市车辆', icon: 'el-icon-truck', component: AdminParkCars },
          { key: 'parkData', name: '车位数据管理', icon: 'el-icon-data-analysis', component: AdminParkData }
        ]
      }
    ]
  },
  // 系统配置（诺哈 config/ manage/ file/）
  {
    key: 'g-config', name: '系统配置', icon: 'el-icon-setting',
    children: [
      { key: 'siteConfig', name: '站点设置', icon: 'el-icon-s-tools', component: AdminSiteConfig },
      { key: 'roles', name: '管理设置', icon: 'el-icon-s-check', component: AdminRoles },
      { key: 'resources', name: '文件管理', icon: 'el-icon-picture-outline', component: AdminResources }
    ]
  }
]

// key → 名称（App.vue 标签导航/面包屑用）
export const tabNames = {}

// key → 页面组件（Dashboard.vue 动态分发用）
export const pageMap = {}

;(function walk (items) {
  items.forEach(it => {
    tabNames[it.key] = it.name
    if (it.component) pageMap[it.key] = it.component
    if (it.children) walk(it.children)
  })
})(menu)
