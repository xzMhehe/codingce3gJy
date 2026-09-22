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
    // url 里若被误填成站内路由（以 / 或 #/ 开头）也按站内路由走，避免开出一个空白页
    play (g) {
      if (g.path) { this.$router.push(g.path); return }
      const u = g.url || ''
      if (u) {
        if (u.charAt(0) === '/' || u.indexOf('#/') === 0) { this.$router.push(u.replace(/^#/, '')); return }
        window.open(u); return
      }
      this.toast('「' + g.name + '」正在建设中，敬请期待')
    },
    tip (title) { this.$router.push('/tip?title=' + encodeURIComponent('游戏·' + title)) }
  }
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
</style>
