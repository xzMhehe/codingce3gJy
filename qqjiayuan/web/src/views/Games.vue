<template>
  <div>
    <!-- 复刻诺哈 /game/index.asp 游戏大厅（演示站 index3.html 原版布局）：
         题图条 + 我的游戏(序号列表) + 网络游戏 + 社区游戏(65px图标表格) + 娱乐竞猜 + 游戏论坛 + 广告条 -->
    <!-- 题图条（演示站 index3.html 原版内联样式：fresh_1.gif + #91e09d 底色，高 25px） -->
    <div style="background-image:url(/static/image/fresh_1.gif);background-repeat:no-repeat;height:25px;"></div>

    <!-- 我的游戏（诺哈 home/game_list.asp：序号 + [x]移除 + ↑↓排序，标题带“添加”） -->
    <template v-if="isLogin">
      <div class="bodule-title">【我的游戏】<a class="g-op" href="javascript:;" @click="showAdd = !showAdd">{{ showAdd ? '收起' : '添加' }}</a></div>
      <div class="module-content">
        <template v-if="myGames.length">
          <template v-for="(g,i) in myGames">
            {{ i + 1 }}.<a href="javascript:;" @click="play(g)">{{ g.name }}</a> [<a href="javascript:;" @click="removeGame(g.id)">x</a>] <a href="javascript:;" @click="moveGame(g,'up')">↑</a> <a href="javascript:;" @click="moveGame(g,'down')">↓</a><br>
          </template>
        </template>
        <template v-else>您没有游戏。</template>
      </div>
      <!-- 添加游戏（诺哈 game_add.asp：序号列出未添加的游戏，点击即加入） -->
      <div v-if="showAdd" class="module-content">
        <template v-for="(g,i) in addable">
          {{ i + 1 }}.<a href="javascript:;" @click="addGame(g)">{{ g.name }}</a><br>
        </template>
        <span v-if="!addable.length" class="txt-fade">游戏大厅的游戏都已添加</span>
      </div>
    </template>

    <!-- 网络游戏（演示站 index3.html：module-content 表格，65px 圆角图标 + 名称|游戏论坛 + 推荐星级 + 简介） -->
    <div class="bodule-title">【网络游戏】</div>
    <div class="module-content g-game" v-for="g in netGames" :key="'n' + g.id">
      <table>
        <tr><th rowspan="3"><div class="g-icon" :style="iconStyle(g)"></div></th>
          <td><a href="javascript:;" @click="play(g)">{{ g.name }}</a>|<a href="javascript:;" @click="goBoard(g)">游戏论坛</a></td></tr>
        <tr><td>推荐：{{ g.stars }}</td></tr>
        <tr><td>{{ g.desc || g.intro }}</td></tr>
      </table>
    </div>
    <div v-if="!netGames.length" class="module-content"><span class="txt-fade">暂无网络游戏</span></div>

    <!-- 社区游戏 -->
    <div class="bodule-title">【社区游戏】</div>
    <div class="module-content g-game" v-for="g in comGames" :key="'c' + g.id">
      <table>
        <tr><th rowspan="3"><div class="g-icon" :style="iconStyle(g)"></div></th>
          <td><a href="javascript:;" @click="play(g)">{{ g.name }}</a>|<a href="javascript:;" @click="goBoard(g)">游戏论坛</a></td></tr>
        <tr><td>推荐：{{ g.stars }}</td></tr>
        <tr><td>{{ g.desc || g.intro }}</td></tr>
      </table>
    </div>
    <div v-if="!comGames.length" class="module-content"><span class="txt-fade">暂无社区游戏</span></div>

    <!-- 娱乐竞猜（诺哈游戏大厅：点击进入互联网游戏板块） -->
    <div class="bodule-title">【<a href="javascript:;" @click="$router.push('/games/net')">娱乐竞猜</a>】</div>
    <div class="module-content">
      <a v-for="(label,j) in guessLabels" :key="'gl' + j" class="g-guess" href="javascript:;" @click="playLabel(label)">{{ label }}</a>
    </div>

    <!-- 游戏论坛（诺哈：游戏综合反馈/游戏研发/游戏交流，均为论坛板块） -->
    <div class="bodule-title">【游戏论坛】</div>
    <div class="module-content">
      <a href="javascript:;" @click="goBoardName('游戏综合反馈')">游戏综合反馈</a>|<a href="javascript:;" @click="goBoardName('游戏研发')">游戏研发</a>|<a href="javascript:;" @click="goBoardName('游戏交流')">游戏交流</a><br>
    </div>

    <!-- 广告条（演示站 index3.html 底部 login-tips） -->
    <div class="login-tips">
      <ul><li class="wid"><img src="/static/picture/notice.gif" alt="广告"><a href="javascript:;" @click="$router.push('/games/hxxy')">[马年新区]古典神话西游！</a></li></ul>
    </div>

    <!-- 未实现游戏的轻提示 -->
    <transition name="g-toast">
      <div v-if="toastMsg" class="g-toast">{{ toastMsg }}</div>
    </transition>

    <!-- ★★ 2026-10-15 进入游戏前的「选择服务器」弹层（区名/区服由管理端游戏大厅维护配置，不写死）。
         点游戏 → 有配区服就先选（哪怕只有一个服也要选），再进入。 -->
    <div v-if="srvDlg.on" class="gs-mask" @click.self="closeSrvDlg">
      <div class="gs-box">
        <div class="gs-title">{{ srvDlg.game ? srvDlg.game.name : '选择服务器' }}</div>
        <div class="gs-sub">请选择要进入的服务器</div>
        <div v-if="srvDlg.loading" class="gs-loading">加载中…</div>
        <!-- ★ 2026-10-15 接口失败不再偷偷关窗直接进（否则"掠影秒进"）：停留显示原因 + 重试 -->
        <div v-else-if="srvDlg.err" class="gs-empty">
          服务器列表加载失败<br>
          <a href="javascript:;" style="color:#2b2b2b;text-decoration:underline;margin-top:6px;display:inline-block;" @click="loadServers">重新加载</a>
        </div>
        <div v-else-if="!srvDlg.list.length" class="gs-empty">该游戏暂无可进入的服务器</div>
        <div v-else class="gs-list">
          <div v-for="s in srvDlg.list" :key="s.id" class="gs-item" @click="enterServer(s)">
            <span class="gs-name">{{ s.name }}</span>
            <span class="gs-in">进入 →</span>
          </div>
        </div>
        <div class="gs-close" @click="closeSrvDlg">关闭</div>
      </div>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Games',
  data () {
    return {
      games: [],
      myGames: [],
      showAdd: false,
      toastMsg: '',
      toastTimer: null,
      // ★★ 2026-10-15 选择服务器弹层（区服由管理端「游戏大厅维护」配置）
      //   err=true 表示列表请求失败（停留给重试，不再"掠影秒进"）
      srvDlg: { on: false, loading: false, err: false, game: null, list: [] },
      // 诺哈原版娱乐竞猜两行：竞猜　农场　江湖　猜球　大话 / 富翁　好友买卖　苹果机　抢车位
      guessLabels: ['竞猜', '农场', '江湖', '猜球', '大话', '富翁', '好友买卖', '苹果机', '抢车位'],
      guessMap: {
        '竞猜': '猜数', '农场': '开心农场', '江湖': '精武堂', '猜球': '台球', '大话': '大话吹牛',
        '富翁': '大富翁', '好友买卖': '好友买卖', '苹果机': '水果乐园', '抢车位': '狂抢车位'
      }
    }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    netGames () { return this.games.filter(g => g.category === 'net') },
    comGames () { return this.games.filter(g => g.category !== 'net') },
    addable () { return this.comGames.filter(g => !this.isOwned(g)) }
  },
  mounted () {
    api.get('/games').then(r => { if (r.code === 0) this.games = r.data })
    this.loadMy()
  },
  beforeDestroy () { clearTimeout(this.toastTimer) },
  methods: {
    loadMy () {
      if (!this.isLogin) return
      api.get('/my-games').then(r => { if (r.code === 0) this.myGames = r.data })
    },
    addGame (g) {
      api.post('/my-games', { game_id: g.id }).then(r => { if (r.code === 0) this.loadMy(); else alert(r.msg) })
    },
    isOwned (g) { return this.myGames.some(m => m.id === g.id) },
    removeGame (id) {
      api.delete('/my-games/' + id).then(r => { if (r.code === 0) this.loadMy() })
    },
    moveGame (g, dir) {
      api.post('/my-games/' + g.id + '/move?dir=' + dir).then(r => {
        if (r.code === 0) this.loadMy(); else alert(r.msg)
      })
    },
    hasGo (g) { return !!(g.path || g.url) },
    // 65px 圆角游戏图标（演示站原版：background-size:100% 100%），无 logo 用问号图兜底
    iconStyle (g) {
      const img = g.logo ? '/static/image/' + g.logo : '/static/image/logo_question.png'
      return { backgroundImage: 'url(' + img + ')' }
    },
    playLabel (label) {
      const name = this.guessMap[label] || label
      const g = this.games.find(x => x.name === name)
      if (g && this.hasGo(g)) { this.play(g); return }
      this.$router.push('/games/net')
    },
    goBoard (g) {
      if (g.board_id) { this.$router.push('/board/' + g.board_id); return }
      this.toast('「' + g.name + '」游戏论坛暂未开放')
    },
    toast (msg) {
      this.toastMsg = msg
      clearTimeout(this.toastTimer)
      this.toastTimer = setTimeout(() => { this.toastMsg = '' }, 2000)
    },
    goBoardName (name) {
      // 「游戏论坛」下的功能性板块：从论坛目录里找到板块 ID 直达
      api.get('/boards').then(r => {
        if (r.code !== 0) return
        for (const ch of (r.data || [])) {
          for (const b of (ch.children || [])) {
            if (b.name === name) { this.$router.push('/board/' + b.id); return }
          }
        }
        this.toast('「' + name + '」板块暂未开放')
      })
    },
    // 进入游戏：内部网址(path)=本站路由，直接跳转；外部网址(url)=外站，新窗口打开
    // ★★ 2026-10-15 改：先查该游戏的区服配置 —— 配了区服（如二战征途）就先弹「选择服务器」
    //   （区名由管理端配置，哪怕只有一个服也要选）；没配区服的维持原逻辑直接进。
    play (g) {
      if (!this.hasGo(g)) { this.toast('「' + g.name + '」正在建设中，敬请期待'); return }
      this.srvDlg.on = true
      this.srvDlg.game = g
      this.loadServers()
    },
    // 拉服务器列表；失败→err=true 停留给重试（不再秒进闪没），成功空列表→直接进（没配区服的普通游戏不受影响）
    loadServers () {
      const g = this.srvDlg.game
      this.srvDlg.loading = true
      this.srvDlg.err = false
      this.srvDlg.list = []
      api.get('/game-servers', { params: { game_id: g.id } }).then(r => {
        if (r.code !== 0) { this.srvDlg.loading = false; this.srvDlg.err = true; return }
        const list = r.data || []
        this.srvDlg.loading = false
        if (!list.length) {
          // 没配区服：维持老行为直接进（二战由后端默认种子保证有区服，不会落这）
          this.srvDlg.on = false
          this.jumpGame(g.path || g.url)
          return
        }
        this.srvDlg.list = list
      }).catch(() => {
        // 接口不存在/后端没重启：停留显示失败 + 重试，绝不"偷偷直接进"
        this.srvDlg.loading = false
        this.srvDlg.err = true
      })
    },
    closeSrvDlg () {
      this.srvDlg.on = false
      this.srvDlg.loading = false
      this.srvDlg.err = false
      this.srvDlg.game = null
      this.srvDlg.list = []
    },
    // 选定区服：host 为空 = 同源本服直接路由；有 host = 跨服整页跳（hash 路由拼 #/ 前缀）
    // ★ 2026-10-15 path 可带 query（如 /games/ezfy?cur=home），统一解析成 {path,query} 再跳
    enterServer (s) {
      const p = (s.path || (this.srvDlg.game && this.srvDlg.game.path) || '')
      if (s.host) {
        window.location.href = s.host.replace(/\/+$/, '') + '/#' + (p.charAt(0) === '/' ? '' : '/') + p
        return
      }
      this.closeSrvDlg()
      if (p) this.$router.push(parseRoute(p))
    },
    jumpGame (u) {
      if (!u) return
      if (u.charAt(0) === '/' || u.indexOf('#/') === 0) { this.$router.push(parseRoute(u.replace(/^#/, ''))); return }
      window.open(u)
    },
    tip (title) { this.$router.push('/tip?title=' + encodeURIComponent('游戏·' + title)) }
  }
}
// 解析「/games/ezfy?cur=home」这类带 query 的站内路由为 vue-router 的 {path, query} 对象
function parseRoute (u) {
  const qi = u.indexOf('?')
  if (qi < 0) return u
  const path = u.slice(0, qi)
  const query = {}
  new URLSearchParams(u.slice(qi + 1)).forEach((v, k) => { query[k] = v })
  return { path, query }
}
</script>

<style scoped>
/* 游戏块（module-content 内表格布局，同演示站原版） */
.g-game table { border-collapse: collapse; }
.g-game th, .g-game td { font-weight: normal; text-align: left; vertical-align: middle; padding: 0; }
/* 65px 圆角图标：width/height 65px、圆角 5px、margin 0 2px 5px 0、垂直居中（原版内联样式）
   等比缩放(contain)居中替代原版 100% 100% 拉伸——横版图（如永恒修仙 283x99）不再压扁 */
.g-icon {
  width: 65px; height: 65px; border-radius: 5px;
  margin: 0 2px 5px 0; vertical-align: middle;
  background-color: #fff;
  background-size: contain; background-position: center; background-repeat: no-repeat;
}

/* 【我的游戏】标题内“添加/收起”操作链接 */
.bodule-title .g-op { float: right; margin-right: 10px; font-weight: normal; }

/* 娱乐竞猜词条间距 */
.g-guess { margin-right: 10px; }

.g-toast {
  position: fixed; left: 50%; bottom: 8%; transform: translateX(-50%);
  z-index: 9; background: rgba(0, 0, 0, .72); color: #fff;
  font-size: 13px; padding: 8px 16px; border-radius: 16px; white-space: nowrap;
}
.g-toast-enter-active, .g-toast-leave-active { transition: opacity .2s; }
.g-toast-enter, .g-toast-leave-to { opacity: 0; }

/* ★★ 2026-10-17 选择服务器弹层（深灰系简约：柔和深灰 + 灰白层次，替代纯黑纯白） */
.gs-mask {
  position: fixed; inset: 0; z-index: 999;
  background: rgba(0, 0, 0, .5);
  display: flex; align-items: center; justify-content: center;
}
.gs-box {
  width: 320px; max-width: 92vw; max-height: 80vh; overflow: auto;
  background: #f8f8f9; border: 1px solid #e0e0e0; border-radius: 10px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, .08);
}
.gs-title {
  background: #2b2b2b;
  padding: 10px 16px; font-weight: bold; color: #fff; text-align: center;
}
.gs-sub { padding: 8px 16px 2px; color: #888; font-size: 12px; }
.gs-list { padding: 10px 16px; }
.gs-item {
  display: flex; justify-content: space-between; align-items: center;
  padding: 10px 12px; margin-bottom: 8px;
  background: #fff; border: 1px solid #e0e0e0; border-radius: 6px;
  cursor: pointer; transition: background .2s, border-color .2s;
}
.gs-item:hover { background: #f2f2f3; border-color: #ccc; }
.gs-name { color: #2b2b2b; font-weight: bold; }
.gs-in { color: #2b2b2b; }
.gs-loading, .gs-empty { padding: 20px 16px; text-align: center; color: #888; }
.gs-close {
  margin: 4px 16px 14px; padding: 7px 0; text-align: center;
  background: #fff; border: 1px solid #2b2b2b; border-radius: 6px;
  color: #2b2b2b; cursor: pointer; transition: background .2s, color .2s;
}
.gs-close:hover { background: #f2f2f3; }
</style>
