/**
 * 管理端统一菜单/页面配置（唯一维护点）
 * 新增管理模块：1) 在 components/admin/ 新建组件 2) 在下面菜单树挂一条 { key, name, icon, component }
 * App.vue 的侧边栏/标签导航和 Dashboard.vue 的页面分发都由本文件自动生成，不再手工双写。
 * perm 为该菜单对应后端接口的权限码（RBAC permissions.code），登录用户的角色未分配该权限则菜单隐藏；
 * 超级管理员（super_admin）或无 perm 的项永远展示。
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
import AdminFamilies from './components/admin/AdminFamilies.vue'
import AdminTtou from './components/admin/AdminTtou.vue'
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
import AdminJwtPlayers from './components/admin/AdminJwtPlayers.vue'
import AdminJwtItems from './components/admin/AdminJwtItems.vue'
import AdminJwtSkills from './components/admin/AdminJwtSkills.vue'
import AdminJwtRecords from './components/admin/AdminJwtRecords.vue'
import AdminJwtData from './components/admin/AdminJwtData.vue'
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
import AdminXyPlayers from './components/admin/AdminXyPlayers.vue'
import AdminXyData from './components/admin/AdminXyData.vue'
import AdminEzfyPlayers from './components/admin/AdminEzfyPlayers.vue'
import AdminEzfyData from './components/admin/AdminEzfyData.vue'
import AdminEzfyLogs from './components/admin/AdminEzfyLogs.vue'
import AdminEzfySystem from './components/admin/AdminEzfySystem.vue'
import AdminEzfyCities from './components/admin/AdminEzfyCities.vue'
import AdminEzfyBuildings from './components/admin/AdminEzfyBuildings.vue'
import AdminEzfyBuildQueue from './components/admin/AdminEzfyBuildQueue.vue'
import AdminEzfyTroops from './components/admin/AdminEzfyTroops.vue'
import AdminEzfyRecruit from './components/admin/AdminEzfyRecruit.vue'
import AdminEzfyOfficers from './components/admin/AdminEzfyOfficers.vue'
import AdminEzfyRecruitLimit from './components/admin/AdminEzfyRecruitLimit.vue'
import AdminEzfyRanks from './components/admin/AdminEzfyRanks.vue'
import AdminEzfyResources from './components/admin/AdminEzfyResources.vue'
import AdminEzfyExchange from './components/admin/AdminEzfyExchange.vue'
import AdminEzfyTechs from './components/admin/AdminEzfyTechs.vue'
import AdminEzfyMap from './components/admin/AdminEzfyMap.vue'
import AdminEzfyCorps from './components/admin/AdminEzfyCorps.vue'
import AdminEzfyPrivchat from './components/admin/AdminEzfyPrivchat.vue'
import AdminEzfyBuildLimit from './components/admin/AdminEzfyBuildLimit.vue'
import AdminEzfyWords from './components/admin/AdminEzfyWords.vue'
import AdminXyLogs from './components/admin/AdminXyLogs.vue'
import AdminXySystem from './components/admin/AdminXySystem.vue'
import AdminSiteConfig from './components/admin/AdminSiteConfig.vue'
import AdminFla from './components/admin/AdminFla.vue'
import AdminWelfare from './components/admin/AdminWelfare.vue'
import AdminRoles from './components/admin/AdminRoles.vue'
import AdminResources from './components/admin/AdminResources.vue'
import AdminName from './components/admin/AdminName.vue'
import AdminMenus from './components/admin/AdminMenus.vue'

// 菜单树：最多两级分组（children 里还可带一层 children，如游戏管理下的各游戏）
export const menu = [
  { key: 'dashboard', name: '数据概览', icon: 'el-icon-data-board', perm: 'module:dashboard' },
  // 会员管理（诺哈 user/）
  {
    key: 'g-user', name: '会员管理', icon: 'el-icon-user',
    children: [
      { key: 'users', name: '会员列表', icon: 'el-icon-s-custom', component: AdminUsers, perm: 'module:users' },
      { key: 'nickName', name: '个性昵称', icon: 'el-icon-brush', component: AdminName, perm: 'module:nickName' },
      { key: 'home', name: '会员资料', icon: 'el-icon-edit-outline', component: AdminHome, perm: 'module:home' },
      { key: 'userDocu', name: '会员证件', icon: 'el-icon-postcard', component: AdminUserDocu, perm: 'module:userDocu' },
      { key: 'userContact', name: '会员联系', icon: 'el-icon-phone-outline', component: AdminUserContacts, perm: 'module:userContact' },
      { key: 'userAddress', name: '会员地址', icon: 'el-icon-map-location', component: AdminUserAddresses, perm: 'module:userAddress' },
      { key: 'userProtec', name: '会员密保', icon: 'el-icon-key', component: AdminUserProtections, perm: 'module:userProtec' },
      { key: 'userLogs', name: '会员日志', icon: 'el-icon-date', component: AdminUserLogs, perm: 'module:userLogs' },
      { key: 'userPhones', name: '手机审核', icon: 'el-icon-mobile-phone', component: AdminUserPhones, perm: 'module:userPhones' },
      { key: 'invites', name: '会员推荐', icon: 'el-icon-share', component: AdminInvites, perm: 'module:invites' },
      { key: 'badges', name: '勋章管理', icon: 'el-icon-medal', component: AdminBadges, perm: 'module:badges' },
    ]
  },
  // 货币管理（诺哈 money/）
  {
    key: 'g-money', name: '货币管理', icon: 'el-icon-coin',
    children: [
      { key: 'wallet', name: '会员财务', icon: 'el-icon-wallet', component: AdminWallet, perm: 'module:wallet' },
      { key: 'walletLogs', name: '货币流水', icon: 'el-icon-tickets', component: AdminWalletLogs, perm: 'module:walletLogs' },
    ]
  },
  // 社区管理（诺哈 bbs/）
  {
    key: 'g-bbs', name: '社区管理', icon: 'el-icon-chat-dot-round',
    children: [
      { key: 'boards', name: '版块管理', icon: 'el-icon-menu', component: AdminBoards, perm: 'module:boards' },
      { key: 'tongcheng', name: '同城管理', icon: 'el-icon-location-outline', component: AdminTongcheng, perm: 'module:tongcheng' },
      { key: 'boardCategories', name: '版块分类', icon: 'el-icon-collection', component: AdminBoardCategories, perm: 'module:boardCategories' },
      { key: 'threads', name: '帖子管理', icon: 'el-icon-document', component: AdminThreads, perm: 'module:threads' },
      { key: 'families', name: '家族管理', icon: 'el-icon-s-cooperation', component: AdminFamilies, perm: 'module:families' },
      { key: 'ttou', name: 'TT头像', icon: 'el-icon-picture-outline', component: AdminTtou, perm: 'module:ttou' },
      { key: 'fla', name: '捐款上榜', icon: 'el-icon-medal-1', component: AdminFla, perm: 'module:fla' },
      { key: 'welfare', name: '福利院', icon: 'el-icon-pie-chart', component: AdminWelfare, perm: 'module:welfare' },
      { key: 'recycle', name: '恢复帖子', icon: 'el-icon-delete', component: AdminThreadRecycle, perm: 'module:recycle' },
      { key: 'wordFilters', name: '敏感词', icon: 'el-icon-remove-outline', component: AdminWordFilters, perm: 'module:wordFilters' },
    ]
  },
  // 内容管理（诺哈 article/ guest/ message/）
  {
    key: 'g-content', name: '内容管理', icon: 'el-icon-notebook-2',
    children: [
      { key: 'articles', name: '文章管理', icon: 'el-icon-notebook-1', component: AdminSiteArticles, perm: 'module:articles' },
      { key: 'guestbook', name: '留言本管理', icon: 'el-icon-edit', component: AdminGuestbook, perm: 'module:guestbook' },
      { key: 'messages', name: '家信管理', icon: 'el-icon-message', component: AdminMessages, perm: 'module:messages' },
    ]
  },
  // 博客管理（诺哈 blog/）
  {
    key: 'g-blog', name: '博客管理', icon: 'el-icon-s-shop',
    children: [
      { key: 'spaces', name: '空间列表', icon: 'el-icon-office-building', component: AdminSpaces, perm: 'module:spaces' },
    ]
  },
  // 家园管理（诺哈 home/）
  {
    key: 'g-home', name: '家园管理', icon: 'el-icon-house',
    children: [
      { key: 'homes', name: '家园列表', icon: 'el-icon-s-home', component: AdminHomes, perm: 'module:homes' },
      { key: 'visitors', name: '家园访客', icon: 'el-icon-view', component: AdminVisitors, perm: 'module:visitors' },
    ]
  },
  // 商城管理（诺哈 shop/ money-shop 道具商城/货币商店）
  {
    key: 'g-shop', name: '商城管理', icon: 'el-icon-shopping-cart-2',
    children: [
      { key: 'shops', name: '店铺管理', icon: 'el-icon-s-shop', component: AdminShops, perm: 'module:shops' },
      { key: 'shopGoods', name: '商品管理', icon: 'el-icon-box', component: AdminShopGoods, perm: 'module:shopGoods' },
      { key: 'shopOrders', name: '订单管理', icon: 'el-icon-s-order', component: AdminShopOrders, perm: 'module:shopOrders' },
      { key: 'shopComments', name: '评论管理', icon: 'el-icon-s-comment', component: AdminShopComments, perm: 'module:shopComments' },
      { key: 'goods', name: '道具商城', icon: 'el-icon-goods', component: AdminGoods, perm: 'module:goods' },
      { key: 'moneyShop', name: '货币商店', icon: 'el-icon-coin', component: AdminMoneyShop, perm: 'module:moneyShop' },
    ]
  },
  // 书城管理（诺哈 book/）
  {
    key: 'g-book', name: '书城管理', icon: 'el-icon-reading',
    children: [
      { key: 'books', name: '小说列表', icon: 'el-icon-notebook-1', component: AdminBooks, perm: 'module:books' },
      { key: 'bookChapters', name: '章节管理', icon: 'el-icon-collection-tag', component: AdminBookChapters, perm: 'module:bookChapters' },
      { key: 'bookComments', name: '书评管理', icon: 'el-icon-chat-line-square', component: AdminBookComments, perm: 'module:bookComments' },
    ]
  },
  // 会员特权（诺哈 vip/）
  {
    key: 'g-vip', name: '会员特权', icon: 'el-icon-s-operation',
    children: [
      { key: 'privileges', name: '特权管理', icon: 'el-icon-star-off', component: AdminPrivileges, perm: 'module:privileges' },
    ]
  },
  // 广播管理（诺哈 radio/）
  {
    key: 'g-radio', name: '广播管理', icon: 'el-icon-bell',
    children: [
      { key: 'announcements', name: '公告广播', icon: 'el-icon-bell', component: AdminAnnouncements, perm: 'module:announcements' },
    ]
  },
  // 游戏管理（诺哈 game/，按游戏分组）
  {
    key: 'g-game', name: '游戏管理', icon: 'el-icon-magic-stick',
    children: [
      { key: 'games', name: '游戏大厅', icon: 'el-icon-trophy', component: AdminGames, perm: 'module:games' },
      {
        key: 'g-garden', name: '魔法花园', icon: 'el-icon-sunny',
        children: [
          { key: 'gardenActivities', name: '花园活动', icon: 'el-icon-magic-stick', component: AdminGardenActivities, perm: 'module:gardenActivities' },
          { key: 'gardenSeeds', name: '花园花种', icon: 'el-icon-sunny', component: AdminGardenSeeds, perm: 'module:gardenSeeds' },
          { key: 'gardenMaps', name: '花之图谱', icon: 'el-icon-picture', component: AdminGardenMaps, perm: 'module:gardenMaps' },
          { key: 'gardenMixes', name: '合成配方', icon: 'el-icon-s-cooperation', component: AdminGardenMixes, perm: 'module:gardenMixes' },
          { key: 'gardenElves', name: '精灵花册', icon: 'el-icon-star-on', component: AdminGardenElves, perm: 'module:gardenElves' },
          { key: 'gardenSign', name: '签到管理', icon: 'el-icon-date', component: AdminGardenSign, perm: 'module:gardenSign' },
          { key: 'gardenData', name: '花园数据管理', icon: 'el-icon-data-analysis', component: AdminGardenData, perm: 'module:gardenData' },
        ]
      },
      {
        key: 'g-farm', name: '开心农场', icon: 'el-icon-cherry',
        children: [
          { key: 'farmSeeds', name: '农场种子', icon: 'el-icon-suitcase-1', component: AdminFarmSeeds, perm: 'module:farmSeeds' },
          { key: 'farmItems', name: '化肥陷阱', icon: 'el-icon-coin', component: AdminFarmItems, perm: 'module:farmItems' },
          { key: 'farmData', name: '农场数据管理', icon: 'el-icon-data-analysis', component: AdminFarmData, perm: 'module:farmData' },
        ]
      },
      {
        key: 'g-park', name: '抢车位', icon: 'el-icon-truck',
        children: [
          { key: 'parkCars', name: '车市车辆', icon: 'el-icon-truck', component: AdminParkCars, perm: 'module:parkCars' },
          { key: 'parkData', name: '车位数据管理', icon: 'el-icon-data-analysis', component: AdminParkData, perm: 'module:parkData' },
        ]
      },
      {
        key: 'g-jwt', name: '精武堂', icon: 'el-icon-s-flag',
        children: [
          { key: 'jwtPlayers', name: '玩家管理', icon: 'el-icon-user', component: AdminJwtPlayers, perm: 'module:jwtPlayers' },
          { key: 'jwtItems', name: '道具管理', icon: 'el-icon-goods', component: AdminJwtItems, perm: 'module:jwtItems' },
          { key: 'jwtSkills', name: '技能管理', icon: 'el-icon-magic-stick', component: AdminJwtSkills, perm: 'module:jwtSkills' },
          { key: 'jwtRecords', name: '比武记录', icon: 'el-icon-trophy', component: AdminJwtRecords, perm: 'module:jwtRecords' },
          { key: 'jwtData', name: '数据管理', icon: 'el-icon-data-analysis', component: AdminJwtData, perm: 'module:jwtData' },
        ]
      },
      {
        key: 'g-xy', name: '幻想西游', icon: 'el-icon-s-custom',
        children: [
          { key: 'xyPlayers', name: '玩家管理', icon: 'el-icon-user', component: AdminXyPlayers, perm: 'module:xyPlayers' },
          { key: 'xyLogs', name: '流水管理', icon: 'el-icon-document', component: AdminXyLogs, perm: 'module:xyLogs' },
          { key: 'xySystem', name: '系统管理', icon: 'el-icon-s-tools', component: AdminXySystem, perm: 'module:xySystem' },
          { key: 'xyData', name: '数据管理', icon: 'el-icon-data-analysis', component: AdminXyData, perm: 'module:xyData' },
        ]
      },
      {
        key: 'g-ezfy', name: '二战风云', icon: 'el-icon-position',
        children: [
          { key: 'ezfyPlayers', name: '玩家信息管理', icon: 'el-icon-user', component: AdminEzfyPlayers, perm: 'module:ezfyPlayers' },
          { key: 'ezfyCities', name: '城市管理', icon: 'el-icon-office-building', component: AdminEzfyCities, perm: 'module:ezfyCities' },
          { key: 'ezfyBuildings', name: '建筑管理', icon: 'el-icon-s-home', component: AdminEzfyBuildings, perm: 'module:ezfyBuildings' },
          { key: 'ezfyBuildQueue', name: '建筑队列管理', icon: 'el-icon-time', component: AdminEzfyBuildQueue, perm: 'module:ezfyBuildQueue' },
          { key: 'ezfyTroops', name: '兵种管理', icon: 'el-icon-s-flag', component: AdminEzfyTroops, perm: 'module:ezfyTroops' },
          { key: 'ezfyRecruit', name: '队伍征兵', icon: 'el-icon-s-promotion', component: AdminEzfyRecruit, perm: 'module:ezfyRecruit' },
          { key: 'ezfyOfficers', name: '军官管理', icon: 'el-icon-medal', component: AdminEzfyOfficers, perm: 'module:ezfyOfficers' },
          { key: 'ezfyRecruitLimit', name: '军校刷新次数', icon: 'el-icon-refresh', component: AdminEzfyRecruitLimit, perm: 'module:ezfyOfficers' },
          { key: 'ezfyRankCfg', name: '军衔维护', icon: 'el-icon-medal', component: AdminEzfyRanks, perm: 'module:ezfyRankCfg' },
          { key: 'ezfyResources', name: '资源管理', icon: 'el-icon-coin', component: AdminEzfyResources, perm: 'module:ezfyResources' },
          { key: 'ezfyExchange', name: '交易行维护', icon: 'el-icon-s-shop', component: AdminEzfyExchange, perm: 'module:ezfyExchange' },
          { key: 'ezfyTechs', name: '科技管理', icon: 'el-icon-cpu', component: AdminEzfyTechs, perm: 'module:ezfyTechs' },
          { key: 'ezfyMap', name: '地图管理', icon: 'el-icon-map-location', component: AdminEzfyMap, perm: 'module:ezfyMap' },
          { key: 'ezfyCorps', name: '军团管理', icon: 'el-icon-s-flag', component: AdminEzfyCorps, perm: 'module:ezfyCorps' },
          { key: 'ezfyPrivchat', name: '私聊管理', icon: 'el-icon-chat-line-square', component: AdminEzfyPrivchat, perm: 'module:ezfyPrivchat' },
          { key: 'ezfyBuildLimit', name: '建筑上限配置', icon: 'el-icon-set-up', component: AdminEzfyBuildLimit, perm: 'module:ezfyBuildLimit' },
          { key: 'ezfyWords', name: '聊天敏感词', icon: 'el-icon-chat-dot-square', component: AdminEzfyWords, perm: 'module:ezfyWords' },
          { key: 'ezfyLogs', name: '流水管理', icon: 'el-icon-document', component: AdminEzfyLogs, perm: 'module:ezfyLogs' },
          { key: 'ezfySystem', name: '系统管理', icon: 'el-icon-s-tools', component: AdminEzfySystem, perm: 'module:ezfySystem' },
          { key: 'ezfyData', name: '数据管理', icon: 'el-icon-data-analysis', component: AdminEzfyData, perm: 'module:ezfyData' },
        ]
      }
    ]
  },
  // 系统配置（诺哈 config/ manage/ file/）
  {
    key: 'g-config', name: '系统配置', icon: 'el-icon-setting',
    children: [
      { key: 'siteConfig', name: '站点设置', icon: 'el-icon-s-tools', component: AdminSiteConfig, perm: 'module:siteConfig' },
      { key: 'roles', name: '管理设置', icon: 'el-icon-s-check', component: AdminRoles, perm: 'module:roles' },
      { key: 'resources', name: '文件管理', icon: 'el-icon-picture-outline', component: AdminResources, perm: 'module:resources' },
      { key: 'menus', name: '菜单维护', icon: 'el-icon-s-operation', component: AdminMenus, perm: 'module:menus' },
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

// 应用菜单覆盖配置：覆盖 name/icon/perm/sort，hidden 过滤，sort 重排同级
export function applyMenuOverrides (items, keyMap) {
  const list = items
    .filter(it => !(keyMap[it.key] && keyMap[it.key].hidden))
    .map((it, i) => {
      const o = keyMap[it.key]
      const node = {
        ...it,
        ...(it.component ? { component: it.component } : {}),
        order: o && o.sort ? o.sort : i + 1
      }
      if (o) {
        if (o.name) node.name = o.name
        if (o.icon) node.icon = o.icon
        if (o.perm) node.perm = o.perm
      }
      if (it.children) node.children = applyMenuOverrides(it.children, keyMap)
      return node
    })
    .filter(it => !it.children || it.children.length)
  list.sort((a, b) => a.order - b.order)
  return list
}

// 深拷贝默认菜单树（菜单维护页/多次合并需隔离节点的 children 引用）
export function cloneMenu () {
  const walk = items => items.map(it => {
    const n = { ...it }
    if (it.children) n.children = walk(it.children)
    return n
  })
  return walk(menu)
}
