<template>
  <div>
    <!-- 参考：诺哈 /game/index.html 游戏大厅（plist 每行一个游戏 + 娱乐竞猜 + 游戏论坛 + 用户动态） -->
    <div><img src="/static/image/youxi.gif" alt="游戏"></div>

    <div class="module-title">社区游戏|<a href="javascript:;" @click="$router.push('/games/net')">互联网游戏板块</a></div>
    <div class="module-content"><span class="txt-fade">社区小游戏，免费游玩，无任何充值消费</span></div>

    <!-- 我的游戏（诺哈 game_list.asp：上移/下移/移除） -->
    <template v-if="isLogin">
      <div class="module-title">【我的游戏】</div>
      <div class="plist">
        <div class="row00" v-for="(g,i) in myGames" :key="'m'+g.id">
          {{ i+1 }}.<a href="javascript:;" @click="play(g)">{{ g.name }}</a>[<a href="javascript:;" @click="moveGame(g,'up')">上移</a>.<a href="javascript:;" @click="moveGame(g,'down')">下移</a>.<a href="javascript:;" style="color:#c00" @click="removeGame(g.id)">移除</a>]<br>
        </div>
        <div class="row00" v-if="!myGames.length"><span class="txt-fade">还没有添加游戏，去下面「添加游戏」挑一个</span></div>
      </div>

      <div class="module-title">【添加游戏】</div>
      <div class="plist">
        <div class="row00" v-for="g in addable" :key="'a'+g.id">
          <img v-if="g.logo" :src="'/static/image/' + g.logo" width="16" height="16" :alt="g.name"><a href="javascript:;" @click="play(g)">{{ g.name }}</a>.<a href="javascript:;" @click="addGame(g)">[添加]</a><br>
        </div>
        <div class="row00" v-if="!addable.length"><span class="txt-fade">全部游戏都已添加</span></div>
      </div>
    </template>

    <!-- 社区游戏大厅（诺哈：每行一个游戏，图标+名称） -->
    <div class="module-title">【社区游戏大厅】</div>
    <div class="plist">
      <div class="row00" v-for="g in comGames" :key="g.id">
        <img v-if="g.logo" :src="'/static/image/' + g.logo" width="16" height="16" :alt="g.name"><a href="javascript:;" @click="play(g)">{{ g.name }}</a><br>
      </div>
      <div class="row00" v-if="!comGames.length"><span class="empty">暂无社区游戏</span></div>
    </div>

    <!-- 娱乐竞猜（诺哈：点击进入互联网游戏板块） -->
    <div class="module-title">【<a href="javascript:;" @click="$router.push('/games/net')">娱乐竞猜</a>】</div>
    <div class="plist"><div class="row00"></div></div>

    <!-- 游戏论坛（诺哈：游戏综合反馈/游戏研发/游戏交流） -->
    <div class="module-title"><a href="javascript:;" @click="tip('游戏综合反馈')">游戏综合反馈</a></div>
    <a href="javascript:;" @click="tip('游戏研发')">游戏研发</a><br>
    <a href="javascript:;" @click="tip('游戏交流')">游戏交流</a><br>

    <div class="module-title">【用户动态】</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Games',
  data () {
    return { games: [], myGames: [] }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    comGames () { return this.games.filter(g => g.category !== 'net') },
    addable () { return this.comGames.filter(g => !this.myGames.some(m => m.id === g.id)) }
  },
  mounted () {
    api.get('/games').then(r => { if (r.code === 0) this.games = r.data })
    this.loadMy()
  },
  methods: {
    loadMy () {
      if (!this.isLogin) return
      api.get('/my-games').then(r => { if (r.code === 0) this.myGames = r.data })
    },
    addGame (g) {
      api.post('/my-games', { game_id: g.id }).then(r => { if (r.code === 0) this.loadMy(); else alert(r.msg) })
    },
    removeGame (id) {
      api.delete('/my-games/' + id).then(r => { if (r.code === 0) this.loadMy() })
    },
    moveGame (g, dir) {
      api.post('/my-games/' + g.id + '/move?dir=' + dir).then(r => {
        if (r.code === 0) this.loadMy(); else alert(r.msg)
      })
    },
    play (g) {
      if (g.path) {
        this.$router.push(g.path)
        return
      }
      if (g.url) {
        window.open(g.url)
        return
      }
      alert('「' + g.name + '」游戏开发中，敬请期待！先去游戏论坛和大家聊聊吧')
    },
    tip (title) { this.$router.push('/tip?title=' + encodeURIComponent('游戏·' + title)) }
  }
}
</script>
