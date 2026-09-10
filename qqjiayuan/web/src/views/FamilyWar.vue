<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/families')">家族</a>&gt;<a href="javascript:;" @click="$router.push('/family/'+fid)">{{ war.my_name || '我的家族' }}</a>&gt;族斗</div>

    <div v-if="!loading && !war.my_name" class="module-content"><span class="empty">{{ err || '加载中' }}</span></div>

    <template v-if="war.my_name">
      <img src="/static/image/jiazu.gif" alt="族斗" style="max-width:100%;vertical-align:middle"><br>
      我的战斗力：<b style="color:#c00">{{ war.fight }}</b> 功勋值：<b :style="{color: war.merit < 0 ? '#1a9e1a' : '#c00'}">{{ war.merit }}</b><br>

      <div class="module-title">【今日战局】（战斗中，22点结束）</div>
      <div class="module-content">
        [<b style="color:#004299">{{ war.my_name }}</b>]{{ war.my_score }} VS {{ war.enemy_score }}[<a href="javascript:;" @click="goVisit(war.enemy_id)">{{ war.enemy_name }}</a>]<br>
        我的生命力：<template v-for="n in 5"><span v-if="n <= war.life" :key="'hf'+n" style="color:#c00">♥</span><span v-else :key="'he'+n" style="color:#bbb">♥</span></template>
        <a href="javascript:;" @click="doBuyLife">购买</a><br>
        <img src="/static/picture/tree.gif" width="20" height="20" alt="。" style="vertical-align:middle"> 选择攻击对象[<a href="javascript:;" @click="doPk(0)">随机</a>.<a href="javascript:;" @click="load()">换一批</a>]<br>
        <template v-if="war.opponents.length">
          <div class="row00" v-for="(o,i) in war.opponents" :key="'o'+o.user_id">
            [ <a href="javascript:;" @click="doPk(o.user_id)">攻</a> ] <a href="javascript:;" @click="$router.push('/user/'+o.user_id)"><font :color="o.color || '#004299'">{{ o.nickname }}</font></a><br>
          </div>
        </template>
        <template v-else><span class="empty">敌方家族暂无成员可攻击</span><br></template>
      </div>

      <div class="module-title">【世界喊话】</div>
      <div class="module-content">
        <span class="txt-fade">每次进行发言私聊或对话扣除1000G币</span><br>
        <form @submit.prevent="doChat(chatType)">
          <input type="text" v-model.trim="chatContent" maxlength="120" size="14">
          <input type="submit" value="私聊" @click.prevent="doChat('private')">
          . <input type="submit" value="对话" @click.prevent="doChat('chat')">
        </form>
        <template v-if="yid===1">
          家人私聊|<b>家族对话</b><br>
        </template>
        <template v-else>
          <b>家人私聊</b>|<a href="javascript:;" @click="switchYid(1)">家族对话</a><br>
        </template>
        <div class="row00" v-for="c in war.chats" :key="'c'+c.id">
          <span class="txt-fade">({{ ago(c.created_at) }})</span>
          <a href="javascript:;" @click="$router.push('/user/'+c.user_id)"><font :color="c.user && c.user.color || '#004299'">{{ c.user ? c.user.nickname : '神秘友友' }}</font></a>：{{ c.content }}
        </div>
        <div v-if="!war.chats.length"><span class="empty">还没有喊话，来鼓舞士气吧</span></div>
      </div>

      <div class="module-content">
        <span class="txt-fade">规则：每日5点生命力，攻击敌方家人胜率55%+，胜利为本族赢荣誉点</span>
      </div>

      <div class="module-title">【历史战况】</div>
      <div class="module-content">
        <template v-if="war.yesterday && war.yesterday.enemy">
          昨日：{{ war.yesterday.win ? '胜' : '负' }}[{{ war.yesterday.enemy }}]<br>
        </template>
        <template v-else>昨日：暂无战况<br></template>
      </div>

      <div class="module-title">【族斗荣誉榜 | 个人功勋榜】</div>
      <div class="module-content">
        族斗荣誉点：{{ war.my_score }} 排名：第{{ war.my_rank }}名<br>
        <template v-for="(f,i) in war.tops">
          {{ i+1 }} . <a href="javascript:;" @click="$router.push('/family/'+f.id)">{{ f.name }}</a> | {{ f.war_points }}<br>
        </template>
        <br>个人功勋榜（贡献值）：<br>
        <template v-for="(m,i) in war.merit_top">
          {{ i+1 }} . <a href="javascript:;" @click="$router.push('/user/'+m.user_id)"><font :color="m.user && m.user.color || '#004299'">{{ m.user ? m.user.nickname : '友友' }}</font></a> | {{ m.exp }}<br>
        </template>
      </div>
    </template>
    <p v-if="msg" style="color:#c00;padding:0 5px">{{ msg }}</p>
    <p v-if="okMsg" style="color:#1a9e1a;padding:0 5px">{{ okMsg }}</p>
    <div class="bar"><a href="javascript:;" @click="$router.push('/families')">家族</a>&gt;<a href="javascript:;" @click="$router.push('/family/'+fid)">{{ war.my_name || '我的家族' }}</a>&gt;族斗</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'FamilyWar',
  data () {
    return { fid: 0, war: {}, yid: 0, chatContent: '', loading: true, err: '', msg: '', okMsg: '' }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      this.fid = this.$route.params.id
      api.get('/families/' + this.fid + '/war', { params: { yid: this.yid } }).then(r => {
        this.loading = false
        if (r.code === 0) this.war = r.data
        else this.err = r.msg
      })
    },
    switchYid (yid) { this.yid = yid; this.load() },
    goVisit (id) { if (id > 0) this.$router.push('/family/' + id) },
    doPk (userId) {
      this.msg = ''; this.okMsg = ''
      api.post('/families/' + this.fid + '/war/pk/' + userId).then(r => {
        if (r.code === 0) {
          const d = r.data
          this.okMsg = (d.win ? '⚔️ 挑战【' + d.opponent + '】大获全胜，高奏凯歌！家族荣誉点+' + d.gain.points : '挑战【' + d.opponent + '】惜败而归！') + '（剩余生命力' + d.life + '）'
          this.load()
        } else this.msg = r.msg
      })
    },
    doBuyLife () {
      this.msg = ''; this.okMsg = ''
      if (!window.confirm('花费 100 G币购买 1 点生命力？')) return
      api.post('/families/' + this.fid + '/war/life').then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg; this.load() }
        else this.msg = r.msg
      })
    },
    doChat (type) {
      this.msg = ''; this.okMsg = ''
      if (!this.chatContent) { this.msg = '说点什么吧'; return }
      api.post('/families/' + this.fid + '/war/chat', { content: this.chatContent, type: type }).then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg; this.chatContent = ''; this.load() }
        else this.msg = r.msg
      })
    },
    ago (t) {
      if (!t) return ''
      const diff = Math.max(0, (Date.now() - new Date(t).getTime()) / 1000)
      if (diff < 60) return Math.floor(diff) + '秒前'
      if (diff < 3600) return Math.floor(diff / 60) + '分钟前'
      if (diff < 86400) return Math.floor(diff / 3600) + '小时前'
      return Math.floor(diff / 86400) + '天前'
    }
  }
}
</script>
