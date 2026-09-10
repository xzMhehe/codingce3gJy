<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/family/'+fid)">世外桃源</a>&gt;乐斗</div>

    <div v-if="!loading && !my.fight && my.fight !== 0" class="module-content"><span class="empty">{{ err || '加载中' }}</span></div>

    <template v-if="ok">
      <img src="/static/image/jiazu.gif" alt="乐斗" style="max-width:100%;vertical-align:middle"><br>
      家人一起来！越斗越开心！<br>
      战斗力:<b style="color:#c00">{{ my.fight }}</b> 功勋值:<b :style="{color: my.merit < 0 ? '#1a9e1a' : '#c00'}">{{ my.merit }}</b><br>
      体力值:{{ my.stamina }} <a href="javascript:;" @click="doFruit">吃果实</a><br>
      今日乐斗:{{ my.ld_count }}/{{ my.ld_limit }}（剩余{{ my.remain }}次）<br>
      <span class="txt-fade">胜{{ my.win_count }}场 / 负{{ my.lose_count }}场</span><br>

      <div class="module-title">
        <template v-if="bt===1">菜鸟</template><template v-else><a href="javascript:;" @click="switchBt(1)">菜鸟</a></template>
        |<template v-if="bt===2">高手</template><template v-else><a href="javascript:;" @click="switchBt(2)">高手</a></template>
        |<template v-if="bt===3">乱斗</template><template v-else><a href="javascript:;" @click="switchBt(3)">乱斗</a></template>
        　<a href="javascript:;" @click="load()">刷新</a>
      </div>
      <div class="module-content" v-if="opponents.length">
        <div class="row00" v-for="o in opponents" :key="'o'+o.user_id">
          <a href="javascript:;" @click="doPk(o)">[斗一斗]</a>　<a href="javascript:;" @click="$router.push('/user/'+o.user_id)"><font :color="o.color || '#004299'">{{ o.nickname }}</font></a>
          <span class="txt-fade">（战斗力{{ o.fight }}）</span><br>
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">该区暂无可挑战的对手，换个区试试</span></div>

      <div class="module-title">乐斗动态</div>
      <div class="module-content" v-if="logs.length">
        <div class="row00" v-for="l in logs" :key="'l'+l.id">
          <span class="txt-fade">({{ ago(l.created_at) }})</span>
          <a href="javascript:;" @click="$router.push('/user/'+l.user_id)"><font :color="l.user && l.user.color || '#004299'">{{ l.user ? l.user.nickname : '神秘友友' }}</font></a>
          {{ l.content }}
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">还没有乐斗动态，来打头阵吧</span></div>

      <div class="module-content"><span class="txt-fade">小系提示：当贡献值达到一定程度，就不能挑战菜鸟区的家人</span></div>
    </template>
    <p v-if="msg" style="color:#c00;padding:0 5px">{{ msg }}</p>
    <p v-if="okMsg" style="color:#1a9e1a;padding:0 5px">{{ okMsg }}</p>
    <div class="bar"><a href="javascript:;" @click="$router.push('/family/'+fid)">{{ famName || '我的家族' }}</a>&gt;乐斗</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'FamilyBattle',
  data () {
    return { fid: 0, famName: '', my: {}, opponents: [], logs: [], bt: 1, loading: true, err: '', msg: '', okMsg: '' }
  },
  computed: {
    ok () { return this.my.fight !== undefined }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      this.loading = true
      this.fid = this.$route.params.id
      api.get('/families/' + this.fid + '/ld', { params: { bt: this.bt } }).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.my = r.data
          this.opponents = r.data.opponents || []
          this.logs = r.data.logs || []
        } else this.err = r.msg
      })
      api.get('/families/mine').then(r => { if (r.code === 0 && r.data) this.famName = r.data.name })
    },
    switchBt (bt) { this.bt = bt; this.load() },
    doPk (o) {
      this.msg = ''; this.okMsg = ''
      api.post('/families/' + this.fid + '/ld/pk/' + o.user_id).then(r => {
        if (r.code === 0) {
          const d = r.data
          this.okMsg = (d.win ? '🏆 战胜了' : '惜败于') + '【' + d.opponent + '】' +
            (d.gain.fight !== undefined ? '，战斗力+1 ' : '') +
            '功勋' + (d.gain.merit >= 0 ? '+' : '') + d.gain.merit +
            (d.gain.coins ? '，G币+' + d.gain.coins : '')
          this.load()
        } else this.msg = r.msg
      })
    },
    doFruit () {
      this.msg = ''; this.okMsg = ''
      api.post('/families/' + this.fid + '/ld/fruit').then(r => {
        if (r.code === 0) { this.okMsg = r.data.msg; this.load() }
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
