<template>
  <el-container class="layout">
    <!-- 侧边栏 -->
    <el-aside width="230px" class="sidebar" v-if="showChrome">
      <div class="logo">
        <div class="logo-ico">Q</div>
        <div class="logo-txt">
          <div class="t1"> 家园</div>
          <div class="t2">社区管理系统</div>
        </div>
      </div>
      <el-scrollbar wrap-class="scrollbar-wrapper">
        <el-menu :default-active="activeTab" text-color="#a3b1cc"
                 active-text-color="#ffffff" :unique-opened="true" @select="go">
          <el-menu-item index="dashboard">
            <i class="el-icon-data-board"></i><span>数据概览</span>
          </el-menu-item>
          <!-- 会员管理（诺哈 user/：会员列表/证件/联系/地址/密保/日志/推荐） -->
          <el-submenu index="g-user">
            <template slot="title"><i class="el-icon-user"></i><span>会员管理</span></template>
            <el-menu-item index="users"><i class="el-icon-s-custom"></i>会员列表</el-menu-item>
            <el-menu-item index="home"><i class="el-icon-edit-outline"></i>会员资料</el-menu-item>
            <el-menu-item index="userDocu"><i class="el-icon-postcard"></i>会员证件</el-menu-item>
            <el-menu-item index="userContact"><i class="el-icon-phone-outline"></i>会员联系</el-menu-item>
            <el-menu-item index="userAddress"><i class="el-icon-map-location"></i>会员地址</el-menu-item>
            <el-menu-item index="userProtec"><i class="el-icon-key"></i>会员密保</el-menu-item>
            <el-menu-item index="userLogs"><i class="el-icon-date"></i>会员日志</el-menu-item>
            <el-menu-item index="userPhones"><i class="el-icon-mobile-phone"></i>手机审核</el-menu-item>
            <el-menu-item index="invites"><i class="el-icon-share"></i>会员推荐</el-menu-item>
          </el-submenu>
          <!-- 货币管理（诺哈 money/：财务/流水） -->
          <el-submenu index="g-money">
            <template slot="title"><i class="el-icon-coin"></i><span>货币管理</span></template>
            <el-menu-item index="wallet"><i class="el-icon-wallet"></i>会员财务</el-menu-item>
            <el-menu-item index="walletLogs"><i class="el-icon-tickets"></i>货币流水</el-menu-item>
          </el-submenu>
          <!-- 社区管理（诺哈 bbs/：版块/帖子/恢复帖子/黑名单榜） -->
          <el-submenu index="g-bbs">
            <template slot="title"><i class="el-icon-chat-dot-round"></i><span>社区管理</span></template>
            <el-menu-item index="boards"><i class="el-icon-menu"></i>版块管理</el-menu-item>
            <el-menu-item index="boardCategories"><i class="el-icon-collection"></i>版块分类</el-menu-item>
            <el-menu-item index="threads"><i class="el-icon-document"></i>帖子管理</el-menu-item>
            <el-menu-item index="recycle"><i class="el-icon-delete"></i>恢复帖子</el-menu-item>
            <el-menu-item index="wordFilters"><i class="el-icon-remove-outline"></i>黑名单榜</el-menu-item>
          </el-submenu>
          <!-- 内容管理（诺哈 article/ guest/ message/） -->
          <el-submenu index="g-content">
            <template slot="title"><i class="el-icon-notebook-2"></i><span>内容管理</span></template>
            <el-menu-item index="articles"><i class="el-icon-notebook-1"></i>文章管理</el-menu-item>
            <el-menu-item index="guestbook"><i class="el-icon-edit"></i>留言本管理</el-menu-item>
            <el-menu-item index="messages"><i class="el-icon-message"></i>家信管理</el-menu-item>
          </el-submenu>
          <!-- 博客管理（诺哈 blog/） -->
          <el-submenu index="g-blog">
            <template slot="title"><i class="el-icon-s-shop"></i><span>博客管理</span></template>
            <el-menu-item index="spaces"><i class="el-icon-office-building"></i>空间列表</el-menu-item>
          </el-submenu>
          <!-- 家园管理（诺哈 home/：家园列表/家园访客） -->
          <el-submenu index="g-home">
            <template slot="title"><i class="el-icon-house"></i><span>家园管理</span></template>
            <el-menu-item index="homes"><i class="el-icon-s-home"></i>家园列表</el-menu-item>
            <el-menu-item index="visitors"><i class="el-icon-view"></i>家园访客</el-menu-item>
          </el-submenu>
          <!-- 商城管理（诺哈 shop/：商品/订单/评论） -->
          <el-submenu index="g-shop">
            <template slot="title"><i class="el-icon-shopping-cart-2"></i><span>商城管理</span></template>
            <el-menu-item index="shops"><i class="el-icon-s-shop"></i>店铺管理</el-menu-item>
            <el-menu-item index="shopGoods"><i class="el-icon-box"></i>商品管理</el-menu-item>
            <el-menu-item index="shopOrders"><i class="el-icon-s-order"></i>订单管理</el-menu-item>
            <el-menu-item index="shopComments"><i class="el-icon-s-comment"></i>评论管理</el-menu-item>
          </el-submenu>
          <!-- 书城管理（诺哈 book/） -->
          <el-submenu index="g-book">
            <template slot="title"><i class="el-icon-reading"></i><span>书城管理</span></template>
            <el-menu-item index="books"><i class="el-icon-notebook-1"></i>小说列表</el-menu-item>
          </el-submenu>
          <!-- 贵宾会员（诺哈 vip/） -->
          <el-submenu index="g-vip">
            <template slot="title"><i class="el-icon-s-operation"></i><span>贵宾会员</span></template>
            <el-menu-item index="privileges"><i class="el-icon-star-off"></i>特权管理</el-menu-item>
          </el-submenu>
          <!-- 广播管理（诺哈 radio/） -->
          <el-submenu index="g-radio">
            <template slot="title"><i class="el-icon-bell"></i><span>广播管理</span></template>
            <el-menu-item index="announcements"><i class="el-icon-bell"></i>公告广播</el-menu-item>
          </el-submenu>
          <!-- 游戏管理（诺哈 game/，含花园 garden/、勋章 medal/） -->
          <el-submenu index="g-game">
            <template slot="title"><i class="el-icon-magic-stick"></i><span>游戏管理</span></template>
            <el-menu-item index="games"><i class="el-icon-trophy"></i>游戏大厅</el-menu-item>
            <el-menu-item index="badges"><i class="el-icon-medal"></i>勋章管理</el-menu-item>
            <el-menu-item index="gardenActivities"><i class="el-icon-magic-stick"></i>花园活动</el-menu-item>
            <el-menu-item index="gardenSeeds"><i class="el-icon-sunny"></i>花园花种</el-menu-item>
            <el-menu-item index="gardenMaps"><i class="el-icon-picture"></i>花之图谱</el-menu-item>
            <el-menu-item index="gardenMixes"><i class="el-icon-s-cooperation"></i>合成配方</el-menu-item>
            <el-menu-item index="gardenElves"><i class="el-icon-star-on"></i>精灵花册</el-menu-item>
            <el-menu-item index="gardenData"><i class="el-icon-data-analysis"></i>游戏数据管理</el-menu-item>
          </el-submenu>
          <!-- 系统配置（诺哈 config/ manage/ file/） -->
          <el-submenu index="g-config">
            <template slot="title"><i class="el-icon-setting"></i><span>系统配置</span></template>
            <el-menu-item index="siteConfig"><i class="el-icon-s-tools"></i>站点设置</el-menu-item>
            <el-menu-item index="roles"><i class="el-icon-s-check"></i>管理设置</el-menu-item>
            <el-menu-item index="resources"><i class="el-icon-picture-outline"></i>文件管理</el-menu-item>
          </el-submenu>
        </el-menu>
      </el-scrollbar>
    </el-aside>

    <!-- 主区域 -->
    <el-container class="main-wrap">
      <el-header class="header" height="60px" v-if="showChrome">
        <div class="crumb">
          <span class="root">管理系统</span>
          <span class="sep">/</span>
          <span class="cur">{{ currentName }}</span>
        </div>
        <div class="right">
          <el-dropdown @command="onCmd">
            <span class="who">
              <span class="avatar">{{ (user.nickname || 'Q').slice(0, 1) }}</span>
              <span class="nick">{{ user.nickname || '—' }}</span>
              <i class="el-icon-arrow-down" />
            </span>
            <el-dropdown-menu slot="dropdown">
              <el-dropdown-item command="front" icon="el-icon-house">社区前台</el-dropdown-item>
              <el-dropdown-item command="logout" icon="el-icon-switch-button" divided>退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </el-dropdown>
        </div>
      </el-header>
      <el-main class="content">
        <transition name="page-fade" mode="out-in">
          <router-view :key="activeTab" />
        </transition>
      </el-main>
    </el-container>
  </el-container>
</template>

<script>
export default {
  name: 'App',
  data () {
    return {
      user: {},
      names: {
        dashboard: '数据概览',
        users: '会员列表', home: '会员资料', userDocu: '会员证件', userContact: '会员联系', userAddress: '会员地址', userProtec: '会员密保', userLogs: '会员日志', invites: '会员推荐',
        wallet: '会员财务', walletLogs: '货币流水',
        boards: '版块管理', boardCategories: '版块分类', threads: '帖子管理', recycle: '恢复帖子', wordFilters: '黑名单榜',
        articles: '文章管理', guestbook: '留言本管理', messages: '家信管理',
        spaces: '空间列表',
        homes: '家园列表', visitors: '家园访客',
        shops: '店铺管理', shopGoods: '商品管理', shopOrders: '订单管理', shopComments: '评论管理',
        books: '小说列表',
        privileges: '特权管理',
        announcements: '公告广播',
        games: '游戏大厅', badges: '勋章管理', gardenActivities: '花园活动', gardenSeeds: '花园花种', gardenMaps: '花之图谱', gardenMixes: '合成配方', gardenElves: '精灵花册', gardenData: '游戏数据管理',
        siteConfig: '站点设置', roles: '管理设置', resources: '文件管理'
      }
    }
  },
  watch: {
    $route () {
      this.loadUser()
      this.$nextTick(() => {
        const wrap = this.$el && this.$el.querySelector('.content')
        if (wrap) wrap.scrollTop = 0
      })
    }
  },
  created () { this.loadUser() },
  computed: {
    showChrome () { return this.$route.path !== '/login' },
    activeTab () {
      const t = this.$route.query.tab || 'dashboard'
      return this.names[t] ? t : 'dashboard'
    },
    currentName () { return this.names[this.activeTab] || '数据概览' }
  },
  methods: {
    loadUser () {
      this.user = JSON.parse(localStorage.getItem('jy_admin_user') || 'null') || {}
    },
    go (key) {
      this.$router.push({ path: '/', query: { tab: key } }).catch(() => {})
    },
    openFront () { window.open('http://' + location.host + '/') },
    onCmd (cmd) {
      if (cmd === 'front') return this.openFront()
      localStorage.removeItem('jy_admin_token')
      localStorage.removeItem('jy_admin_user')
      this.$router.push('/login')
    }
  }
}
</script>

<style>
/* ===== 布局基础 ===== */
html, body { margin: 0; padding: 0; }
body {
  font-size: 14px; color: #303133; background: #f0f2f5;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
  -webkit-font-smoothing: antialiased;
}
#app { max-width: none; }
.layout { height: 100vh; }

/* ===== 侧边栏 ===== */
.sidebar {
  background: linear-gradient(180deg, #1e2a3a 0%, #253346 100%);
  flex-shrink: 0; height: 100vh; z-index: 30;
  box-shadow: 2px 0 12px rgba(0,0,0,.15);
  display: flex; flex-direction: column; overflow: hidden;
}
.sidebar .logo {
  height: 72px; display: flex; align-items: center; justify-content: center; gap: 12px;
  background: rgba(0,0,0,.2);
  border-bottom: 1px solid rgba(255,255,255,.06);
}
.sidebar .logo-ico {
  width: 38px; height: 38px; border-radius: 10px;
  background: linear-gradient(135deg, #409eff 0%, #7367f0 100%);
  color: #fff; font-weight: 800; font-size: 20px;
  display: flex; align-items: center; justify-content: center;
  box-shadow: 0 4px 12px rgba(64,158,255,.45);
  transition: transform .3s;
}
.sidebar .logo-ico:hover { transform: scale(1.08) rotate(-3deg); }
.sidebar .logo-txt { text-align: left; line-height: 1.3; }
.sidebar .logo-txt .t1 { color: #fff; font-size: 15px; font-weight: 700; letter-spacing: 1px; }
.sidebar .logo-txt .t2 { color: #7c8aa5; font-size: 11px; letter-spacing: 2px; margin-top: 1px; }
.sidebar .scrollbar-wrapper { overflow-x: hidden; }
.sidebar .el-scrollbar { flex: 1; min-height: 0; }
.sidebar .el-scrollbar__wrap { overflow-x: hidden; }
/* 菜单整体留白，纵向收纳 */
.sidebar .el-menu {
  border-right: none; background: transparent !important;
  padding: 6px 8px;
}
/* 一级/二级菜单项通用 */
.sidebar .el-menu-item, .sidebar .el-submenu__title {
  height: 44px; line-height: 44px; background: transparent !important;
  border-radius: 8px; margin-bottom: 2px; position: relative;
  transition: background .2s ease, color .2s ease;
}
.sidebar .el-menu-item i, .sidebar .el-submenu__title i { color: #7c8aa5; transition: color .2s ease; }
/* hover：轻柔高亮，图标变亮 */
.sidebar .el-menu-item:hover, .sidebar .el-submenu__title:hover {
  background: rgba(255,255,255,.07) !important; color: #fff !important;
}
.sidebar .el-menu-item:hover i, .sidebar .el-submenu__title:hover i { color: #b6c4dd; }
/* 选中项：左侧竖向指示条 + 渐变高亮（主流后台风格） */
.sidebar .el-menu-item.is-active {
  background: linear-gradient(90deg, rgba(64,158,255,.32), rgba(64,158,255,.12)) !important;
  color: #fff !important;
  box-shadow: inset 0 0 0 1px rgba(64,158,255,.28);
}
.sidebar .el-menu-item.is-active::before {
  content: ""; position: absolute; left: 0; top: 50%; transform: translateY(-50%);
  width: 4px; height: 20px; border-radius: 4px; background: #409eff;
}
.sidebar .el-menu-item.is-active i { color: #4aa8ff !important; }
/* 父级菜单在其子项选中时保持高亮标题 */
.sidebar .el-submenu.is-active > .el-submenu__title { color: #fff; }
.sidebar .el-submenu.is-active > .el-submenu__title i { color: #4aa8ff; }
/* 二级菜单容器：加深底色，缩进 */
.sidebar .el-submenu .el-menu { background: rgba(0,0,0,.18) !important; }
.sidebar .el-submenu .el-menu .el-menu-item { padding-left: 50px !important; min-width: auto; }
/* 展开箭头：hover 渐变、展开旋转 */
.sidebar .el-submenu__icon-arrow {
  color: #6b7a94; transition: transform .28s ease, color .2s ease;
}
.sidebar .el-submenu.is-opened > .el-submenu__title .el-submenu__icon-arrow {
  transform: rotateZ(180deg);
}

/* ===== 主区域 ===== */
.main-wrap { flex: 1; min-width: 0; display: flex; flex-direction: column; }

/* ===== 顶部导航 ===== */
.header {
  background: #fff; display: flex; align-items: center; padding: 0 28px;
  flex-shrink: 0; height: 60px;
  box-shadow: 0 1px 6px rgba(0,21,41,.06);
  border-bottom: 1px solid #f0f2f5;
}
.header .crumb { display: flex; align-items: center; gap: 8px; font-size: 14px; }
.header .crumb .root { color: #97a8be; }
.header .crumb .sep { color: #d0d6e0; }
.header .crumb .cur { color: #303133; font-weight: 600; }
.header .right { margin-left: auto; display: flex; align-items: center; gap: 16px; }
.header .who {
  display: flex; align-items: center; gap: 8px; cursor: pointer;
  padding: 4px 12px 4px 4px; border-radius: 20px; transition: background .2s;
}
.header .who:hover { background: #f5f7fa; }
.header .who .avatar {
  width: 34px; height: 34px; border-radius: 50%;
  background: linear-gradient(135deg, #409eff, #7367f0);
  color: #fff; display: flex; align-items: center; justify-content: center;
  font-weight: 700; font-size: 14px;
  box-shadow: 0 2px 6px rgba(64,158,255,.3);
}
.header .who .nick { color: #303133; font-weight: 500; font-size: 13px; }

/* ===== 内容区 ===== */
.content { padding: 22px 26px; background: #f0f2f5; flex: 1; min-height: 0; overflow-y: auto; }

/* ===== 概览统计卡 ===== */
.stat-row { display: flex; gap: 16px; flex-wrap: wrap; margin-bottom: 20px; }
.stat-card {
  flex: 1; min-width: 150px; background: #fff; border-radius: 12px; padding: 20px;
  display: flex; align-items: center; gap: 16px;
  box-shadow: 0 1px 3px rgba(0,21,41,.04), 0 4px 12px rgba(0,21,41,.03);
  transition: all .25s ease; border: 1px solid #f0f2f5;
}
.stat-card:hover {
  transform: translateY(-3px);
  box-shadow: 0 6px 20px rgba(0,21,41,.1);
  border-color: #e8ecf1;
}
.stat-card .ico {
  width: 54px; height: 54px; border-radius: 12px; display: flex; align-items: center;
  justify-content: center; font-size: 24px; color: #fff;
  box-shadow: 0 4px 12px rgba(0,0,0,.1);
}
.stat-card .ico i { font-size: 26px; }
.ico-users { background: linear-gradient(135deg, #409eff, #5cadff); }
.ico-online { background: linear-gradient(135deg, #67c23a, #85ce61); }
.ico-threads { background: linear-gradient(135deg, #e6a23c, #f0c060); }
.ico-replies { background: linear-gradient(135deg, #7367f0, #9b8ff5); }
.ico-boards { background: linear-gradient(135deg, #00b8d4, #26c6da); }
.stat-card .num { font-size: 28px; font-weight: 700; color: #303133; line-height: 1.2; font-family: "DIN Alternate", "Segoe UI", sans-serif; }
.stat-card .lab { font-size: 12px; color: #97a8be; margin-top: 3px; letter-spacing: .5px; }

/* ===== 通用 ===== */
.toolbar { margin-bottom: 16px; display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.toolbar .grow { flex: 1; }
img.bicon { vertical-align: middle; margin-right: 2px; }
.preview { max-width: 40px; max-height: 22px; vertical-align: middle; }
.logo-prev { width: 28px; height: 28px; border-radius: 4px; vertical-align: middle; }
.help-line { color: #97a8be; font-size: 12px; }
.empty { padding: 24px 8px; color: #97a8be; text-align: center; }
.el-card.box {
  border: none; border-radius: 10px;
  box-shadow: 0 1px 3px rgba(0,21,41,.04), 0 4px 12px rgba(0,21,41,.03) !important;
  border: 1px solid #f0f2f5;
  transition: box-shadow .2s;
}
.el-card.box:hover { box-shadow: 0 2px 8px rgba(0,21,41,.08), 0 8px 24px rgba(0,21,41,.04) !important; }
.el-table th { background: #fafbfc !important; color: #606266; font-weight: 600; border-bottom: 2px solid #ebeef5; }
.el-table td { border-bottom: 1px solid #f0f2f5 !important; }
.el-pagination { text-align: right; margin-top: 16px; }

/* ===== 概览页附加 ===== */
.overview-grid { display: flex; gap: 16px; flex-wrap: wrap; }
.overview-grid .col { flex: 1; min-width: 300px; }
.quick-item {
  display: flex; align-items: center; gap: 12px; padding: 12px 14px; border-radius: 10px;
  background: #f7f9fc; cursor: pointer; transition: all .2s; margin-bottom: 10px;
  border: 1px solid transparent;
}
.quick-item:hover { background: #ecf5ff; border-color: #d4e8fc; }
.quick-item .qi {
  width: 38px; height: 38px; border-radius: 10px; background: #fff; color: #409eff;
  display: flex; align-items: center; justify-content: center; font-size: 18px;
  box-shadow: 0 2px 6px rgba(0,21,41,.08);
}
.quick-item .qt { font-size: 13px; color: #303133; font-weight: 500; }
.quick-item .qd { font-size: 12px; color: #97a8be; }
</style>
