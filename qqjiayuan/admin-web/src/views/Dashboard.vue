<template>
  <div>
    <!-- ===== 数据概览页 ===== -->
    <template v-if="isOverview">
      <!-- 欢迎横幅 -->
      <div class="overview-hero">
        <div class="hero-left">
          <div class="hero-title">{{ greeting }}，{{ adminName }}！</div>
          <div class="hero-sub">欢迎回到  家园社区管理系统，祝您工作愉快。</div>
        </div>
        <div class="hero-badge">
          <i class="el-icon-date"></i>
          <span>{{ today }}</span>
        </div>
      </div>

      <!-- 统计卡片 -->
      <div class="stat-row">
        <div class="stat-card"><div class="ico ico-users"><i class="el-icon-user"></i></div><div><div class="num">{{ stats.users || 0 }}</div><div class="lab">注册居民</div></div></div>
        <div class="stat-card"><div class="ico ico-online"><i class="el-icon-cpu"></i></div><div><div class="num">{{ stats.online || 0 }}</div><div class="lab">当前在线</div></div></div>
        <div class="stat-card"><div class="ico ico-threads"><i class="el-icon-document"></i></div><div><div class="num">{{ stats.threads || 0 }}</div><div class="lab">帖子</div></div></div>
        <div class="stat-card"><div class="ico ico-replies"><i class="el-icon-chat-dot-square"></i></div><div><div class="num">{{ stats.replies || 0 }}</div><div class="lab">回复</div></div></div>
        <div class="stat-card"><div class="ico ico-boards"><i class="el-icon-menu"></i></div><div><div class="num">{{ stats.boards || 0 }}</div><div class="lab">板块</div></div></div>
      </div>

      <!-- 快捷入口 -->
      <div class="quick-grid">
        <div class="quick-panel">
          <div class="panel-title"><i class="el-icon-s-operation"></i> 快速入口</div>
          <div class="quick-items">
            <div class="quick-item" v-for="q in quickLinks" :key="q.tab" @click="go(q.tab)">
              <div class="qi" :style="{ background: q.bg, color: q.color }"><i :class="q.icon"></i></div>
              <div>
                <div class="qt">{{ q.name }}</div>
                <div class="qd">{{ q.desc }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- ===== 管理面板（仅对应 tab 显示） ===== -->
    <admin-users v-if="tab === 'users'" />
    <admin-home v-else-if="tab === 'home'" />
    <admin-wallet v-else-if="tab === 'wallet'" />
    <admin-wallet-logs v-else-if="tab === 'walletLogs'" />
    <admin-invites v-else-if="tab === 'invites'" />
    <admin-user-phones v-else-if="tab === 'userPhones'" />
    <admin-user-docu v-else-if="tab === 'userDocu'" />
    <admin-user-contacts v-else-if="tab === 'userContact'" />
    <admin-user-addresses v-else-if="tab === 'userAddress'" />
    <admin-user-protections v-else-if="tab === 'userProtec'" />
    <admin-user-logs v-else-if="tab === 'userLogs'" />
    <admin-boards v-else-if="tab === 'boards'" />
    <admin-board-categories v-else-if="tab === 'boardCategories'" />
    <admin-word-filters v-else-if="tab === 'wordFilters'" />
    <admin-threads v-else-if="tab === 'threads'" />
    <admin-thread-recycle v-else-if="tab === 'recycle'" />
    <admin-announcements v-else-if="tab === 'announcements'" />
    <admin-roles v-else-if="tab === 'roles'" />
    <admin-privileges v-else-if="tab === 'privileges'" />
    <admin-games v-else-if="tab === 'games'" />
    <admin-badges v-else-if="tab === 'badges'" />
    <admin-garden-activities v-else-if="tab === 'gardenActivities'" />
    <admin-garden-seeds v-else-if="tab === 'gardenSeeds'" />
    <admin-garden-maps v-else-if="tab === 'gardenMaps'" />
    <admin-garden-mixes v-else-if="tab === 'gardenMixes'" />
    <admin-garden-elves v-else-if="tab === 'gardenElves'" />
    <admin-garden-sign v-else-if="tab === 'gardenSign'" />
    <admin-garden-data v-else-if="tab === 'gardenData'" />
    <admin-farm-seeds v-else-if="tab === 'farmSeeds'" />
    <admin-farm-items v-else-if="tab === 'farmItems'" />
    <admin-farm-data v-else-if="tab === 'farmData'" />
    <admin-park-cars v-else-if="tab === 'parkCars'" />
    <admin-park-data v-else-if="tab === 'parkData'" />
    <admin-resources v-else-if="tab === 'resources'" />
    <admin-spaces v-else-if="tab === 'spaces'" />
    <admin-homes v-else-if="tab === 'homes'" />
    <admin-visitors v-else-if="tab === 'visitors'" />
    <admin-shops v-else-if="tab === 'shops'" />
    <admin-shop-goods v-else-if="tab === 'shopGoods'" />
    <admin-shop-orders v-else-if="tab === 'shopOrders'" />
    <admin-shop-comments v-else-if="tab === 'shopComments'" />
    <admin-site-articles v-else-if="tab === 'articles'" />
    <admin-guestbook v-else-if="tab === 'guestbook'" />
    <admin-messages v-else-if="tab === 'messages'" />
    <admin-books v-else-if="tab === 'books'" />
    <admin-site-config v-else-if="tab === 'siteConfig'" />
  </div>
</template>

<script>
import api from '../api'
import AdminUsers from '../components/admin/AdminUsers.vue'
import AdminHome from '../components/admin/AdminHome.vue'
import AdminWallet from '../components/admin/AdminWallet.vue'
import AdminBoards from '../components/admin/AdminBoards.vue'
import AdminBoardCategories from '../components/admin/AdminBoardCategories.vue'
import AdminWordFilters from '../components/admin/AdminWordFilters.vue'
import AdminThreads from '../components/admin/AdminThreads.vue'
import AdminReports from '../components/admin/AdminReports.vue'
import AdminAnnouncements from '../components/admin/AdminAnnouncements.vue'
import AdminPlazaSections from '../components/admin/AdminPlazaSections.vue'
import AdminFamilies from '../components/admin/AdminFamilies.vue'
import AdminTongcheng from '../components/admin/AdminTongcheng.vue'
import AdminTtou from '../components/admin/AdminTtou.vue'
import AdminRoles from '../components/admin/AdminRoles.vue'
import AdminBadges from '../components/admin/AdminBadges.vue'
import AdminPrivileges from '../components/admin/AdminPrivileges.vue'
import AdminGames from '../components/admin/AdminGames.vue'
import AdminGardenActivities from '../components/admin/AdminGardenActivities.vue'
import AdminGardenSeeds from '../components/admin/AdminGardenSeeds.vue'
import AdminGardenMaps from '../components/admin/AdminGardenMaps.vue'
import AdminGardenMixes from '../components/admin/AdminGardenMixes.vue'
import AdminGardenElves from '../components/admin/AdminGardenElves.vue'
import AdminGardenSign from '../components/admin/AdminGardenSign.vue'
import AdminGardenData from '../components/admin/AdminGardenData.vue'
import AdminFarmSeeds from '../components/admin/AdminFarmSeeds.vue'
import AdminFarmItems from '../components/admin/AdminFarmItems.vue'
import AdminFarmData from '../components/admin/AdminFarmData.vue'
import AdminParkCars from '../components/admin/AdminParkCars.vue'
import AdminParkData from '../components/admin/AdminParkData.vue'
import AdminGoods from '../components/admin/AdminGoods.vue'
import AdminMoneyShop from '../components/admin/AdminMoneyShop.vue'
import AdminResources from '../components/admin/AdminResources.vue'
import AdminSpaces from '../components/admin/AdminSpaces.vue'
import AdminWalletLogs from '../components/admin/AdminWalletLogs.vue'
import AdminInvites from '../components/admin/AdminInvites.vue'
import AdminUserDocu from '../components/admin/AdminUserDocu.vue'
import AdminUserContacts from '../components/admin/AdminUserContacts.vue'
import AdminUserAddresses from '../components/admin/AdminUserAddresses.vue'
import AdminUserProtections from '../components/admin/AdminUserProtections.vue'
import AdminUserLogs from '../components/admin/AdminUserLogs.vue'
import AdminUserPhones from '../components/admin/AdminUserPhones.vue'
import AdminThreadRecycle from '../components/admin/AdminThreadRecycle.vue'
import AdminHomes from '../components/admin/AdminHomes.vue'
import AdminVisitors from '../components/admin/AdminVisitors.vue'
import AdminShops from '../components/admin/AdminShops.vue'
import AdminShopGoods from '../components/admin/AdminShopGoods.vue'
import AdminShopOrders from '../components/admin/AdminShopOrders.vue'
import AdminShopComments from '../components/admin/AdminShopComments.vue'
import AdminSiteArticles from '../components/admin/AdminSiteArticles.vue'
import AdminGuestbook from '../components/admin/AdminGuestbook.vue'
import AdminMessages from '../components/admin/AdminMessages.vue'
import AdminBooks from '../components/admin/AdminBooks.vue'
import AdminSiteConfig from '../components/admin/AdminSiteConfig.vue'

export default {
  name: 'Dashboard',
  components: { AdminUsers, AdminHome, AdminWallet, AdminBoards, AdminBoardCategories, AdminWordFilters, AdminThreads, AdminAnnouncements, AdminRoles, AdminBadges, AdminPrivileges, AdminGames, AdminGardenActivities, AdminGardenSeeds, AdminGardenMaps, AdminGardenMixes, AdminGardenElves, AdminGardenSign, AdminGardenData, AdminFarmSeeds, AdminFarmItems, AdminFarmData, AdminParkCars, AdminParkData, AdminResources, AdminSpaces, AdminWalletLogs, AdminInvites, AdminUserDocu, AdminUserContacts, AdminUserAddresses, AdminUserProtections, AdminUserLogs, AdminUserPhones, AdminThreadRecycle, AdminHomes, AdminVisitors, AdminShops, AdminShopGoods, AdminShopOrders, AdminShopComments, AdminSiteArticles, AdminGuestbook, AdminMessages, AdminBooks, AdminSiteConfig },
  data () {
    return { stats: {}, adminName: '' }
  },
  computed: {
    tab () {
      return this.$route.query.tab || 'dashboard'
    },
    isOverview () {
      return this.tab === 'dashboard'
    },
    greeting () {
      const h = new Date().getHours()
      if (h < 6) return '夜深了'
      if (h < 12) return '早上好'
      if (h < 14) return '中午好'
      if (h < 18) return '下午好'
      return '晚上好'
    },
    today () {
      const d = new Date()
      const week = ['日', '一', '二', '三', '四', '五', '六']
      return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日 星期${week[d.getDay()]}`
    },
    quickLinks () {
      return [
        { tab: 'users', icon: 'el-icon-user', name: '会员列表', desc: '资料/角色/封禁/详情', bg: '#ecf5ff', color: '#409eff' },
        { tab: 'threads', icon: 'el-icon-document', name: '帖子管理', desc: '置顶、加精、活动、审核', bg: '#fdf6ec', color: '#e6a23c' },
        { tab: 'boards', icon: 'el-icon-menu', name: '版块管理', desc: '分区与版块维护', bg: '#f0f9eb', color: '#67c23a' },
        { tab: 'shopGoods', icon: 'el-icon-box', name: '商品管理', desc: '上下架与删除', bg: '#fef0f0', color: '#f56c6c' },
        { tab: 'homes', icon: 'el-icon-s-home', name: '家园列表', desc: '活跃点与家园统计', bg: '#f4f0ff', color: '#7367f0' },
        { tab: 'siteConfig', icon: 'el-icon-s-tools', name: '站点设置', desc: '注册赠送等系统配置', bg: '#f0f9eb', color: '#67c23a' }
      ]
    }
  },
  mounted () {
    this.adminName = (JSON.parse(localStorage.getItem('jy_admin_user') || 'null') || {}).nickname || '管理员'
    this.loadStats()
  },
  methods: {
    loadStats () {
      api.get('/admin/stats').then(r => {
        if (r.code === 0) this.stats = r.data
      })
    },
    go (tab) {
      this.$router.push({ path: '/', query: { tab } }).catch(() => {})
    }
  }
}
</script>

<style scoped>
/* ===== 欢迎横幅 ===== */
.overview-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 26px 30px;
  margin-bottom: 20px;
  border-radius: 14px;
  background: linear-gradient(120deg, #409eff 0%, #5a76f0 60%, #7367f0 100%);
  color: #fff;
  box-shadow: 0 8px 24px rgba(64,158,255,.28);
  overflow: hidden;
}
.hero-title { font-size: 22px; font-weight: 700; letter-spacing: .5px; }
.hero-sub { margin-top: 8px; color: rgba(255,255,255,.82); font-size: 13px; }
.hero-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  border-radius: 30px;
  background: rgba(255,255,255,.16);
  border: 1px solid rgba(255,255,255,.22);
  font-size: 13px;
  white-space: nowrap;
}
.hero-badge i { font-size: 16px; }

/* ===== 快捷入口 ===== */
.quick-grid { margin-top: 4px; }
.quick-panel {
  border-radius: 12px;
  background: #fff;
  border: 1px solid #f0f2f5;
  box-shadow: 0 1px 3px rgba(0,21,41,.04), 0 4px 12px rgba(0,21,41,.03);
  padding: 20px;
}
.panel-title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 16px;
}
.panel-title i { color: #409eff; margin-right: 6px; }
.quick-items { display: grid; grid-template-columns: repeat(auto-fill, minmax(250px, 1fr)); gap: 12px; }
.quick-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px;
  border-radius: 10px;
  background: #f8fafc;
  cursor: pointer;
  transition: all .2s;
  border: 1px solid transparent;
}
.quick-item:hover { background: #fff; border-color: #dbe7fb; box-shadow: 0 4px 12px rgba(64,158,255,.1); transform: translateY(-2px); }
.quick-item .qi {
  width: 40px; height: 40px;
  border-radius: 10px;
  display: grid;
  place-items: center;
  font-size: 19px;
  flex-shrink: 0;
}
.quick-item .qt { font-size: 14px; color: #303133; font-weight: 600; }
.quick-item .qd { font-size: 12px; color: #97a8be; margin-top: 2px; }
</style>
