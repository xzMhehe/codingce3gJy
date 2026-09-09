<template>
  <div>
    <div><img src="/static/image/youxi.gif" alt="游戏"></div>

    <div class="module-title">社区游戏|<a href="javascript:;" @click="$router.push('/channel/1')">互联网游戏板块</a></div>
    <div class="module-content"><span class="txt-fade">社区小游戏，免费游玩，无任何充值消费</span></div>

    <!-- 我的游戏（对齐诺哈 game_list.asp：上移/下移/移除） -->
    <div class="module-title">【我的游戏】<a href="javascript:;" @click="loadMy">刷新</a></div>
    <div class="module-content" v-if="myGames.length">
      <div v-for="(g,i) in myGames" :key="'m'+g.id" class="row">
        {{ i+1 }}.<a href="javascript:;" @click="play(g)">{{ g.name }}</a>
        <span class="txt-fade">（{{ g.desc || '' }}）</span>
        [<a href="javascript:;" @click="moveGame(g,'up')" :class="{disabled:i===0}">↑</a>
        <a href="javascript:;" @click="moveGame(g,'down')" :class="{disabled:i===myGames.length-1}">↓</a>]
        [<a href="javascript:;" style="color:#c00" @click="removeGame(g.id)">移除</a>]<br>
      </div>
    </div>
    <div class="module-content" v-else><span class="txt-fade">还没有添加游戏，去下面「添加游戏」里挑一个</span></div>

    <!-- 社区游戏（复刻诺哈 game 目录：魔法花园/开心农场/狂抢车位/好友买卖/竞技场/砸金蛋/台球/猜数/六合彩/大富翁/大话吹牛...） -->
    <div class="module-title">【社区游戏】<a href="javascript:;" @click="showAll = !showAll">{{ showAll ? '收起' : '展开' }}</a></div>
    <div class="module-content" v-for="g in comGames" :key="g.id">
      <span class="glogo-wrap">
        <img v-if="g.logo" class="glogo" :src="'/static/image/' + g.logo" :alt="g.name">
        <span v-else class="glogo glogo-text">{{ g.name.slice(0, 2) }}</span>
      </span>
      <b><a href="javascript:;" @click="play(g)">{{ g.name }}</a></b>
      <span class="txt-fade">（{{ g.stars || '' }}）</span>
      <template v-if="isLogin">
        <a href="javascript:;" @click="addGame(g)">[{{ inMy(g.id) ? '已在游戏中' : '添加游戏' }}]</a>
      </template>
      <a href="javascript:;" @click="forum(g)">[论坛]</a>
      <a href="javascript:;" @click="play(g)">[进入游戏]</a><br>
      <span class="txt-fade">{{ g.intro || g.desc }}</span><br>
    </div>
    <div v-if="!comGames.length" class="empty">暂无社区游戏</div>

    <!-- 网络游戏 -->
    <div class="module-title">【网络游戏】</div>
    <div class="module-content" v-for="g in netGames" :key="'n'+g.id">
      <span class="glogo-wrap">
        <img v-if="g.logo" class="glogo" :src="'/static/image/' + g.logo" :alt="g.name">
        <span v-else class="glogo glogo-text">{{ g.name.slice(0, 2) }}</span>
      </span>
      <b><a href="javascript:;" @click="play(g)">{{ g.name }}</a></b>
      <span class="txt-fade">（{{ g.stars || '' }}）</span>
      <template v-if="isLogin"><a href="javascript:;" @click="addGame(g)">[{{ inMy(g.id) ? '已在游戏中' : '添加游戏' }}]</a></template>
      <a href="javascript:;" @click="forum(g)">[论坛]</a>
      <a href="javascript:;" @click="play(g)">[进入游戏]</a><br>
      <span class="txt-fade">{{ g.desc }}</span><br>
    </div>
    <div v-if="!netGames.length" class="empty">暂无网络游戏</div>

    <div class="module-title">【游戏论坛】</div>
    <div class="module-content">
      <a href="javascript:;" @click="tip('游戏综合反馈')">游戏综合反馈</a>. <a href="javascript:;" @click="tip('游戏研发')">游戏研发</a>.<a href="javascript:;" @click="tip('游戏交流')">游戏交流</a><br>
    </div>

    <div class="login-tips">
      <ul>
        <li class="wid"><img :src="$pic('notice.gif')" alt="广告">魔法花园已可玩，其余游戏陆续开放，先去游戏论坛聊聊情怀！</li>
      </ul>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Games',
  data () {
    return { games: [], boards: {}, myGames: [], showAll: true }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    netGames () { return this.games.filter(g => g.category === 'net') },
    comGames () { return this.games.filter(g => g.category !== 'net') }
  },
  mounted () {
    api.get('/games').then(r => { if (r.code === 0) this.games = r.data })
    this.loadMy()
    api.get('/boards').then(r => {
      if (r.code !== 0) return
      const map = {}
      r.data.forEach(ch => (ch.children || []).forEach(b => { map[b.name] = b.id }))
      this.boards = map
    })
  },
  methods: {
    loadMy () {
      if (!this.isLogin) return
      api.get('/my-games').then(r => { if (r.code === 0) this.myGames = r.data })
    },
    inMy (id) { return this.myGames.some(m => m.id === id) },
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
    tip (title) { this.$router.push('/tip?title=' + encodeURIComponent('游戏·' + title)) },
    forum (g) {
      if (g.board_id) {
        this.$router.push('/board/' + g.board_id)
        return
      }
      const id = this.boards[g.name]
      if (id) this.$router.push('/board/' + id)
      else alert('该游戏的论坛还在建设中')
    }
  }
}
</script>

<style scoped>
.glogo-wrap { display: inline-block; }
.glogo {
  width: 32px;
  height: 32px;
  border-width: 0;
  border-radius: 4px;
  margin: 0 4px 0 0;
  vertical-align: middle;
}
.glogo-text {
  background: linear-gradient(135deg, #f8b26a, #d4691e);
  color: #fff;
  font-weight: bold;
  font-size: 12px;
  line-height: 1.2;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  text-shadow: 0 1px 2px rgba(120, 40, 0, .6);
  box-sizing: border-box;
}
.disabled { color: #ccc; pointer-events: none; }
</style>