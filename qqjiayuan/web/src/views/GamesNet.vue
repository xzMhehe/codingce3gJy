<template>
  <div>
    <!-- 参考：诺哈 channel/9701.html 互联网游戏板块（社区游戏|互联网游戏板块 切换 + 娱乐竞猜 plist） -->
    <div><img src="/static/image/youxi.gif" alt="游戏"></div>

    <div class="module-title"><a href="javascript:;" @click="$router.push('/games')">社区游戏</a>|互联网游戏板块</div>
    <br>

    <div class="module-title">【<a href="javascript:;" @click="$router.push('/games')">娱乐竞猜</a>】</div>
    <div class="plist">
      <div class="row00" v-for="(row,i) in rows" :key="'r'+i">
        <a v-for="(label,j) in row" :key="i+'-'+j" href="javascript:;" style="margin-right:1em" @click="playLabel(label)">{{ label }}</a>
      </div>
      <div class="row00" v-if="netGames.length">
        <a v-for="(g,j) in netGames" :key="'ng'+g.id" href="javascript:;" style="margin-right:1em" @click="play(g)">{{ g.name }}</a>
      </div>
    </div>

    <div class="module-title"><a href="javascript:;" @click="tip('游戏综合反馈')">游戏综合反馈</a></div>
    <a href="javascript:;" @click="tip('游戏研发')">游戏研发</a><br>
    <a href="javascript:;" @click="tip('游戏交流')">游戏交流</a><br>

    <div class="module-title">【用户动态】</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'GamesNet',
  data () {
    return {
      games: [],
      // 诺哈原版两行：竞猜　农场　江湖　猜球　大话 / 富翁　好友买卖　苹果机　抢车位
      rows: [
        ['竞猜', '农场', '江湖', '猜球', '大话'],
        ['富翁', '好友买卖', '苹果机', '抢车位']
      ],
      // 板块简称 → 本地游戏名（对应实现的用真名，未实现点击提示开发中）
      map: {
        '竞猜': '猜数', '农场': '开心农场', '江湖': '精武堂', '猜球': '台球', '大话': '大话吹牛',
        '富翁': '大富翁', '好友买卖': '好友买卖', '苹果机': '水果乐园', '抢车位': '狂抢车位'
      }
    }
  },
  computed: {
    netGames () { return this.games.filter(g => g.category === 'net') }
  },
  mounted () {
    api.get('/games').then(r => { if (r.code === 0) this.games = r.data })
  },
  methods: {
    playLabel (label) {
      const name = this.map[label] || label
      const g = this.games.find(x => x.name === name)
      if (g) this.play(g)
      else alert('「' + label + '」游戏开发中，敬请期待！')
    },
    play (g) {
      if (g.path) { this.$router.push(g.path); return }
      if (g.url) { window.open(g.url); return }
      alert('「' + g.name + '」游戏开发中，敬请期待！先去游戏论坛和大家聊聊吧')
    },
    tip (title) { this.$router.push('/tip?title=' + encodeURIComponent('游戏·' + title)) }
  }
}
</script>
