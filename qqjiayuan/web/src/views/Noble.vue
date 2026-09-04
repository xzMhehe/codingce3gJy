<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;<a href="javascript:;" @click="$router.push('/home')">百宝箱</a>&gt;超级QQ蓝钻</div>

    <div class="module-title"><a href="javascript:;" @click="demo('成长体系')">成长体系</a> <a href="javascript:;" @click="$router.push('/noble')">开通特权</a></div>

    <div>尊贵的{{ title }}<img v-if="data.noble > 0" :src="$pic('noble_' + data.noble + '_1.gif')" alt="." /> <a href="javascript:;" @click="$router.push('/user/'+user.id)">个人中心</a><br></div>

    <div class="module-content">
      <p><img src="/static/picture/qs_05.gif" alt="." /> 在线升级提速<br></p>
      <p><img src="/static/picture/noble_2_1.gif" alt="." /> 酷炫特权图标<br></p>
      <p><img src="/static/picture/chuping.jpg" alt="." /> 超多游戏特权</p>
    </div>

    <div class="bodule-title"><font color="#FF0000">【我的特权】</font></div>
    <div class="module-content">
      当前：{{ title }}（Lv.{{ data.noble_exp }} 成长值）
      <template v-if="!data.noble"><span class="txt-fade">（未开通）</span></template>
    </div>

    <div class="bodule-title"><font color="#FF0000">【特权开通】</font></div>
    <div class="module-content plist">
      <div v-for="p in data.plans" :key="p.id" class="row00">
        {{ p.id }}.<a href="javascript:;" @click="activate(p)">{{ p.name }}（{{ p.cost }}金币，+{{ p.gain }}成长）</a>
      </div>
    </div>

    <div class="bodule-title"><font color="#FF0000">【特权排行】</font></div>
    <div class="module-content">
      <div v-for="(r,i) in data.rank" :key="r.user_id" class="row00">
        {{ i+1 }}.<img v-if="r.noble_exp > 0" :src="$pic('noble_2_1.gif')" alt="." /><a href="javascript:;" @click="$router.push('/user/'+r.user_id)"><font :color="r.color || '#ff0000'">{{ r.nickname }}</font></a>（{{ r.noble_exp }}点成长值↑）
      </div>
      <div v-if="!data.rank.length"><span class="empty">还没有开通超Q的友友</span></div>
    </div>

    <p style="color:#c00;padding:0 5px" v-if="msg">{{ msg }}</p>
    <p style="color:#1a9e1a;padding:0 5px" v-if="okMsg">{{ okMsg }}</p>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Noble',
  data () { return { data: { plans: [], rank: [] }, msg: '', okMsg: '' } },
  computed: {
    user () { return this.$store.state.user || {} },
    title () { return this.data.noble > 1 ? '超Q' : (this.data.noble > 0 ? '蓝钻' : '普通友友') }
  },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/noble').then(r => { if (r.code === 0) this.data = r.data })
    },
    activate (p) {
      api.post('/noble/activate', { plan_id: p.id }).then(r => {
        if (r.code === 0) { this.okMsg = '开通成功！获得 ' + p.gain + ' 点成长值'; this.load() } else this.msg = r.msg
      })
    },
    demo (t) { alert('超Q · ' + t + '（演示入口）') }
  }
}
</script>
