<template>
  <div>
    <div class="bar">【游戏大厅】<a class="rt" href="javascript:;" @click="$router.push('/')">回广场</a></div>
    <div style="background-image:url(/static/image/fresh_1.gif);background-color:#91e09d;background-repeat:no-repeat;height:25px;"></div>
    <div class="bodule-title"> 【网络游戏】 </div>
    <div class="module-content" v-for="g in netGames" :key="g.id">
      <table>
        <tr>
          <th rowspan="3">
            <div v-if="g.logo" class="glogo" :style="{background:'url(/static/image/'+g.logo+') 0 0/100% 100% no-repeat'}"></div>
            <div v-else class="glogo glogo-text">{{ shortName(g.name) }}</div>
          </th>
          <td><a href="javascript:;" @click="play(g)">{{ g.name }}</a>|<a href="javascript:;" @click="forum(g)">游戏论坛</a></td>
        </tr>
        <tr><td>推荐：{{ g.stars }}</td></tr>
        <tr><td>{{ g.desc }}</td></tr>
      </table>
    </div>
    <div v-if="!netGames.length" class="empty">暂无网络游戏</div>

    <div class="bodule-title"><p>【社区游戏】</p></div>
    <div class="module-content" v-for="g in comGames" :key="g.id">
      <table>
        <tr>
          <th rowspan="3">
            <div v-if="g.logo" class="glogo" :style="{background:'url(/static/image/'+g.logo+') 0 0/100% 100% no-repeat'}"></div>
            <div v-else class="glogo glogo-text">{{ shortName(g.name) }}</div>
          </th>
          <td><a href="javascript:;" @click="play(g)">{{ g.name }}</a>|<a href="javascript:;" @click="forum(g)">游戏论坛</a></td>
        </tr>
        <tr><td>推荐：{{ g.stars }}</td></tr>
        <tr><td>{{ g.desc }}</td></tr>
      </table>
    </div>
    <div v-if="!comGames.length" class="empty">暂无社区游戏</div>

    <div class="login-tips">
      <ul>
        <li class="wid"><img :src="$pic('notice.gif')" alt="广告">游戏还在开发中，先到各游戏论坛聊聊情怀！</li>
      </ul>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Games',
  data () {
    return { games: [], boards: {} }
  },
  computed: {
    netGames () { return this.games.filter(g => g.category === 'net') },
    comGames () { return this.games.filter(g => g.category !== 'net') }
  },
  mounted () {
    api.get('/games').then(r => { if (r.code === 0) this.games = r.data })
    api.get('/boards').then(r => {
      if (r.code !== 0) return
      const map = {}
      r.data.forEach(ch => (ch.children || []).forEach(b => { map[b.name] = b.id }))
      this.boards = map
    })
  },
  methods: {
    play (g) {
      if (g.url) {
        window.open(g.url)
        return
      }
      alert('「' + g.name + '」游戏开发中，敬请期待！先去游戏论坛和大家聊聊吧')
    },
    forum (g) {
      if (g.board_id) {
        this.$router.push('/board/' + g.board_id)
        return
      }
      const id = this.boards[g.name] || this.boards[this.shortName(g.name)]
      if (id) this.$router.push('/board/' + id)
      else alert('该游戏的论坛还在建设中')
    },
    shortName (name) {
      // 3GQQ幻想西游 → 幻想西游 / 3GQQ魔法花园 → 魔法花园
      return name.replace(/^3GQQ/, '')
    }
  }
}
</script>

<style scoped>
.glogo {
  width: 65px;
  height: 65px;
  border-width: 0;
  border-radius: 5px;
  margin: 0 2px 5px 0;
  vertical-align: middle;
}
.glogo-text {
  background: linear-gradient(135deg, #f8b26a, #d4691e);
  color: #fff;
  font-weight: bold;
  font-size: 15px;
  line-height: 1.2;
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  text-shadow: 0 1px 2px rgba(120, 40, 0, .6);
  padding: 4px;
  box-sizing: border-box;
}
</style>
