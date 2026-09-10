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
        <!-- 菜单由 ../menu.js 配置驱动，新增模块只改配置文件 -->
        <el-menu :default-active="activeTab" text-color="#a3b1cc"
                 active-text-color="#ffffff" :unique-opened="true" @select="go">
          <template v-for="m in menu">
            <el-submenu v-if="m.children" :key="m.key" :index="m.key">
              <template slot="title"><i :class="m.icon"></i><span>{{ m.name }}</span></template>
              <template v-for="c in m.children">
                <el-submenu v-if="c.children" :key="c.key" :index="c.key">
                  <template slot="title"><i :class="c.icon"></i><span>{{ c.name }}</span></template>
                  <el-menu-item v-for="l in c.children" :key="l.key" :index="l.key">
                    <i :class="l.icon"></i>{{ l.name }}
                  </el-menu-item>
                </el-submenu>
                <el-menu-item v-else :key="c.key" :index="c.key">
                  <i :class="c.icon"></i>{{ c.name }}
                </el-menu-item>
              </template>
            </el-submenu>
            <el-menu-item v-else :key="m.key" :index="m.key">
              <i :class="m.icon"></i><span>{{ m.name }}</span>
            </el-menu-item>
          </template>
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
      <!-- 标签页导航（主流后台 TagsView：点击切换、× 关闭、右键关闭左侧/右侧/全部） -->
      <div class="tags-bar" v-if="showChrome">
        <div class="tags-scroll">
          <span v-for="t in visited" :key="t" class="tag-item" :class="{ active: t === activeTab }"
                @click="go(t)" @contextmenu.prevent="openMenu(t, $event)">
            <i class="dot" />
            {{ names[t] }}
            <i v-if="t !== 'dashboard'" class="el-icon-close" @click.stop="closeTag(t)" />
          </span>
        </div>
        <ul class="ctx-menu" v-if="ctxShow" :style="{ left: ctxX + 'px', top: ctxY + 'px' }">
          <li @click="closeTag(ctxTab)" v-if="ctxTab !== 'dashboard'"><i class="el-icon-close" />关闭</li>
          <li @click="closeOthers(ctxTab)"><i class="el-icon-copy-document" />关闭其他</li>
          <li @click="closeSide('left')"><i class="el-icon-back" />关闭左侧</li>
          <li @click="closeSide('right')"><i class="el-icon-right" />关闭右侧</li>
          <li @click="closeAll()"><i class="el-icon-circle-close" />关闭全部</li>
        </ul>
      </div>
      <el-main class="content">
        <transition name="page-fade" mode="out-in">
          <router-view :key="activeTab" />
        </transition>
      </el-main>
    </el-container>
  </el-container>
</template>

<script>
import { menu, tabNames } from './menu'

export default {
  name: 'App',
  data () {
    return {
      user: {},
      menu,
      visited: ['dashboard'],
      ctxShow: false, ctxX: 0, ctxY: 0, ctxTab: ''
    }
  },
  watch: {
    $route () {
      this.loadUser()
      this.$nextTick(() => {
        const wrap = this.$el && this.$el.querySelector('.content')
        if (wrap) wrap.scrollTop = 0
      })
    },
    activeTab: {
      immediate: true,
      handler (t) {
        if (!this.visited.includes(t)) this.visited.push(t)
      }
    }
  },
  created () { this.loadUser() },
  mounted () {
    document.addEventListener('click', this.hideMenu)
  },
  beforeDestroy () {
    document.removeEventListener('click', this.hideMenu)
  },
  computed: {
    showChrome () { return this.$route.path !== '/login' },
    names () { return tabNames },
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
    // ===== 标签页导航 =====
    openMenu (t, e) {
      this.ctxTab = t
      this.ctxX = e.clientX
      this.ctxY = e.clientY
      this.ctxShow = true
    },
    hideMenu () { this.ctxShow = false },
    closeTag (t) {
      this.ctxShow = false
      if (t === 'dashboard') return
      const i = this.visited.indexOf(t)
      this.visited = this.visited.filter(x => x !== t)
      if (t === this.activeTab) this.go(this.visited[Math.min(i - 1, this.visited.length - 1)] || 'dashboard')
    },
    closeOthers (t) {
      this.ctxShow = false
      // dashboard 永远只保留一个在头部，否则标签栏会出现两个"数据概览"
      this.visited = t === 'dashboard' ? ['dashboard'] : ['dashboard', t]
      if (this.activeTab !== t) this.go(t)
    },
    closeSide (side) {
      this.ctxShow = false
      const i = this.visited.indexOf(this.ctxTab)
      if (i < 0) return
      this.visited = side === 'left'
        ? ['dashboard'].concat(this.visited.slice(i).filter(x => x !== 'dashboard'))
        : this.visited.slice(0, i + 1)
      if (!this.visited.includes(this.activeTab)) this.go(this.ctxTab)
    },
    closeAll () {
      this.ctxShow = false
      this.visited = ['dashboard']
      if (this.activeTab !== 'dashboard') this.go('dashboard')
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

/* ===== 表格规范：所有行强制单行，超出省略并悬停提示 ===== */
.el-table td .cell, .el-table th .cell {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
/* 操作列按钮不换行、间距紧凑，保证一行放下 */
.el-table .cell .el-button + .el-button { margin-left: 6px; }
.el-table .cell .el-button--mini { padding: 6px 8px; }

/* ===== 标签页导航（TagsView） ===== */
.tags-bar {
  position: relative;
  background: #fff;
  border-bottom: 1px solid #e8ebf2;
  padding: 6px 14px;
  flex-shrink: 0;
  z-index: 20;
}
.tags-scroll { display: flex; align-items: center; gap: 6px; overflow-x: auto; scrollbar-width: none; }
.tags-scroll::-webkit-scrollbar { display: none; }
.tag-item {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 4px 10px; border: 1px solid #dcdfe6; border-radius: 3px;
  font-size: 12px; color: #495060; cursor: pointer; white-space: nowrap;
  background: #fff; transition: all .15s; user-select: none;
}
.tag-item .dot { width: 6px; height: 6px; border-radius: 50%; background: #c0c4cc; }
.tag-item:hover { border-color: #b8d4f5; color: #409eff; }
.tag-item.active { background: #409eff; border-color: #409eff; color: #fff; }
.tag-item.active .dot { background: #fff; }
.tag-item .el-icon-close {
  font-size: 12px; border-radius: 50%; width: 14px; height: 14px;
  display: inline-flex; align-items: center; justify-content: center;
  transition: all .15s;
}
.tag-item .el-icon-close:hover { background: rgba(0,0,0,.15); color: #fff; }
.ctx-menu {
  position: fixed; z-index: 4000; margin: 0; padding: 5px 0;
  background: #fff; border-radius: 4px; min-width: 120px;
  box-shadow: 0 4px 16px rgba(0,0,0,.15); list-style: none;
}
.ctx-menu li {
  padding: 7px 16px; font-size: 13px; color: #495060; cursor: pointer;
  display: flex; align-items: center; gap: 8px;
}
.ctx-menu li:hover { background: #ecf5ff; color: #409eff; }

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
