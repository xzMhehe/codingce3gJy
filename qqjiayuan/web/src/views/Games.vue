<template>
  <div>
    <!-- 参考：诺哈 /game/index.html 游戏大厅（plist 每行一个游戏 + 娱乐竞猜 + 游戏论坛 + 用户动态） -->
    <div><img src="/static/image/youxi.gif" alt="游戏"></div>

    <div class="module-title">社区游戏|<a href="javascript:;" @click="$router.push('/games/net')">互联网游戏板块</a></div>
    <div class="module-content"><span class="txt-fade">社区小游戏，免费游玩，无任何充值消费</span></div>

    <!-- 我的游戏（诺哈 game_list.asp）：卡片式，右侧上移/下移/移除 -->
    <template v-if="isLogin">
      <div class="module-title">【我的游戏】</div>
      <div v-if="myGames.length" class="g-hall">
        <div class="g-card owned" v-for="(g,i) in myGames" :key="'m'+g.id" @click="play(g)">
          <img v-if="g.logo" class="g-logo" :src="'/static/image/' + g.logo" :alt="g.name">
          <span v-else class="g-tile" :style="tileStyle(g.name)">{{ initial(g.name) }}</span>
          <div class="g-info">
            <div class="g-name">
              {{ g.name }}
              <span class="g-stars">{{ g.stars }}</span>
            </div>
            <div class="g-intro">{{ g.desc }}</div>
          </div>
          <div class="g-right">
            <button class="g-enter" @click.stop="play(g)">进入游戏</button>
            <div class="g-move">
              <button class="g-opt up" :disabled="i===0" title="上移" @click.stop="moveGame(g,'up')">↑</button>
              <button class="g-opt down" :disabled="i===myGames.length-1" title="下移" @click.stop="moveGame(g,'down')">↓</button>
              <button class="g-opt del" title="移除" @click.stop="removeGame(g.id)">移除</button>
            </div>
          </div>
        </div>
      </div>
      <div v-else class="plist"><div class="row00"><span class="txt-fade">还没有添加游戏，去大厅里点「添加」挑一个</span></div></div>
    </template>

    <!-- 社区游戏大厅（诺哈：每行一个游戏，图标 + 名称 + 评分 + 简介） -->
    <div class="module-title">【社区游戏大厅】</div>
    <div v-if="comGames.length" class="g-hall">
      <div class="g-card" v-for="g in comGames" :key="g.id" :class="{ done: hasGo(g), todo: !hasGo(g), owned: isOwned(g) }" @click="play(g)">
        <img v-if="g.logo" class="g-logo" :src="'/static/image/' + g.logo" :alt="g.name">
        <span v-else class="g-tile" :style="tileStyle(g.name)">{{ initial(g.name) }}</span>
        <div class="g-info">
          <div class="g-name">
            {{ g.name }}
            <span class="g-stars">{{ g.stars }}</span>
          </div>
          <div class="g-intro">{{ g.intro || g.desc }}</div>
        </div>
        <div class="g-right">
          <button v-if="isLogin" class="g-add" :class="{ owned: isOwned(g) }" @click.stop="toggleAdd(g)">{{ isOwned(g) ? '已添加' : '添加' }}</button>
        </div>
      </div>
    </div>
    <div v-else class="plist"><div class="row00"><span class="empty">暂无社区游戏</span></div></div>

    <!-- 娱乐竞猜（诺哈：点击进入互联网游戏板块） -->
    <div class="module-title">【<a href="javascript:;" @click="$router.push('/games/net')">娱乐竞猜</a>】</div>
    <div class="plist">
      <div class="row00">
        <a v-for="(label,j) in guessLabels" :key="'gl'+j" class="g-guess" href="javascript:;" @click="playLabel(label)">{{ label }}</a>
      </div>
    </div>

    <!-- 游戏论坛（诺哈：游戏综合反馈/游戏研发/游戏交流） -->
    <div class="module-title"><a href="javascript:;" @click="tip('游戏综合反馈')">游戏综合反馈</a></div>
    <a href="javascript:;" @click="tip('游戏研发')">游戏研发</a><br>
    <a href="javascript:;" @click="tip('游戏交流')">游戏交流</a><br>

    <div class="module-title">【用户动态】</div>

    <!-- 未实现游戏的轻提示 -->
    <transition name="g-toast">
      <div v-if="toastMsg" class="g-toast">{{ toastMsg }}</div>
    </transition>
  </div>
</template>

<script>
import api from '../api'

// 游戏名 → 兜底色块渐变色（无 logo 时渲染首字彩块，避免空图）：[主色, 深一档]
const COLORS = [
  ['#2e9cd3', '#1c6f9c'], ['#e05a00', '#a84100'], ['#1a9e1a', '#116c11'],
  ['#8a5cf5', '#6433c4'], ['#e83a7a', '#b0205b'], ['#1aa3b0', '#117680'],
  ['#c76a2e', '#944a1b'], ['#4f8be0', '#2f5fb0']
]

export default {
  name: 'Games',
  data () {
    return {
      games: [],
      myGames: [],
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
    comGames () { return this.games.filter(g => g.category !== 'net') }
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
    toggleAdd (g) {
      if (this.isOwned(g)) { this.removeGame(g.id); return }
      this.addGame(g)
    },
    removeGame (id) {
      api.delete('/my-games/' + id).then(r => { if (r.code === 0) this.loadMy() })
    },
    moveGame (g, dir) {
      api.post('/my-games/' + g.id + '/move?dir=' + dir).then(r => {
        if (r.code === 0) this.loadMy(); else alert(r.msg)
      })
    },
    hasGo (g) { return !!(g.path || g.url) },
    initial (name) { return (name || '游').slice(0, 1) },
    tileStyle (name) {
      const n = (name || '').charCodeAt(0) || 0
      const pair = COLORS[n % COLORS.length]
      return { background: `linear-gradient(135deg, ${pair[0]}, ${pair[1]})` }
    },
    playLabel (label) {
      const name = this.guessMap[label] || label
      const g = this.games.find(x => x.name === name)
      if (g && this.hasGo(g)) { this.play(g); return }
      this.$router.push('/games/net')
    },
    toast (msg) {
      this.toastMsg = msg
      clearTimeout(this.toastTimer)
      this.toastTimer = setTimeout(() => { this.toastMsg = '' }, 2000)
    },
    play (g) {
      if (g.path) { this.$router.push(g.path); return }
      if (g.url) { window.open(g.url); return }
      this.toast('「' + g.name + '」正在建设中，敬请期待')
    },
    tip (title) { this.$router.push('/tip?title=' + encodeURIComponent('游戏·' + title)) }
  }
}
</script>

<style scoped>
.g-hall { max-width: 360px; padding: 2px 0; }
.g-card {
  display: flex; align-items: center; gap: 6px;
  background: #fff; border: 1px solid #dbe7f3; border-radius: 8px;
  padding: 6px 8px; margin-bottom: 6px; cursor: pointer;
}
.g-card:hover { border-color: #2e9cd3; box-shadow: 0 1px 6px rgba(46,156,211,.18); }
.g-card.todo { opacity: .92; }
.g-card.todo:hover { border-color: #cfd9e3; box-shadow: none; }
.g-logo { width: 40px; height: 40px; border-radius: 8px; object-fit: cover; flex-shrink: 0; }
.g-tile {
  width: 40px; height: 40px; border-radius: 8px; flex-shrink: 0;
  display: inline-flex; align-items: center; justify-content: center;
  color: #fff; font-weight: bold; font-size: 18px;
}
.g-tile.mini { width: 20px; height: 20px; font-size: 12px; border-radius: 4px; vertical-align: -4px; }
.g-info { flex: 1; min-width: 0; }
.g-name { color: #004299; font-weight: bold; font-size: 14px; }
.g-stars { color: #ffaa00; font-size: 12px; margin-left: 4px; }
.g-intro { color: #666; font-size: 12px; margin-top: 2px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.g-right { flex-shrink: 0; display: flex; flex-direction: column; align-items: flex-end; gap: 3px; }
.g-enter {
  flex-shrink: 0; border: none; padding: 3px 12px; font-size: 12px; color: #fff;
  background: #1a9e1a; border-radius: 10px; cursor: pointer;
}
.g-enter:hover { background: #157c15; }
.g-move { display: flex; gap: 4px; align-items: center; }
.g-opt {
  border: 1px solid #d4e0ea; background: #fff; color: #2e6da0;
  font-size: 13px; line-height: 16px; padding: 1px 7px; border-radius: 8px; cursor: pointer;
}
.g-opt:hover { background: #eaf5fc; }
.g-opt:disabled { color: #c5cfd8; cursor: default; background: #f5f7f9; }
.g-opt.del { color: #c0392b; }
.g-opt.del:hover { background: #fdecea; }
.g-add {
  border: 1px solid #2e9cd3; color: #2e9cd3; background: #fff;
  font-size: 12px; line-height: 16px; padding: 1px 8px; border-radius: 10px; cursor: pointer;
}
.g-add:hover { background: #eaf5fc; }
.g-add.owned { border-color: #b9c6d3; color: #8aa3b8; cursor: default; background: #f6f9fb; }
.g-guess { display: inline-block; margin: 2px 10px 2px 0; color: #004299; }
.g-toast {
  position: fixed; left: 50%; bottom: 8%; transform: translateX(-50%);
  z-index: 9; background: rgba(0, 0, 0, .72); color: #fff;
  font-size: 13px; padding: 8px 16px; border-radius: 16px; white-space: nowrap;
}
.g-toast-enter-active, .g-toast-leave-active { transition: opacity .2s; }
.g-toast-enter, .g-toast-leave-to { opacity: 0; }
</style>