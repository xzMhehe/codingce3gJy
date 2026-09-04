<template>
  <div>
    <!-- 活动横幅（参考站 note，置于最顶） -->
    <div class="note" v-if="cur === 'garden'">
      <a href="javascript:;" @click="$router.push('/channel/1')">美加墨世界杯【绿茵足球兑换活动】介绍</a><br/>
      <img class="noteico" src="/static/image/vipqq.jpg" alt="." />&nbsp;回家的礼物、一天都不能少(0/7) <a href="javascript:;" @click="$router.push('/channel/1')">签到</a><br/>
    </div>
    <!-- 顶部导航（参考站 .bar）：花园 好友 花房 魔法屋 活动 -->
    <div class="bar garden-nav">
      <span :class="{ cur: cur === 'garden' }">花园</span> <a :class="{ cur: cur === 'friend' }" href="javascript:;" @click="cur='friend'">好友</a> <a :class="{ cur: cur === 'basket' }" href="javascript:;" @click="cur='basket'">花房</a> <a :class="{ cur: cur === 'room' }" href="javascript:;" @click="cur='room'">魔法屋</a> <a :class="{ cur: cur === 'active' }" href="javascript:;" @click="cur='active'">活动</a><img src="/static/picture/hot.gif" alt="." />
    </div>

    <div class="g-main">
      <!-- 花园 -->
      <template v-if="cur === 'garden'">
        <div class="name userline">{{ nick }} <img class="bicon" src="/static/picture/noble_2_1.gif" alt="." />({{ level }}级)</div>
        <div class="module-content deep">{{ level }}级 见习魔法学徒(经验 {{ coins }}/200)</div>
        <div class="module-content">
          精灵花册:<a href="javascript:;" @click="cur='room'">0/10</a><br/>
          <br/>花之图谱:<a href="javascript:;" @click="cur='room'">0/619</a><br/>珍稀:0 独特:0 普通:0<br/>
        </div>
        <div class="plots" v-if="plots.length">
          <img v-for="p in plots" :key="p.index" :src="'/static/picture/' + icon(p)" :title="tip(p)" alt="." />
        </div>
        <br/>
        <div class="module-content">背包中没有魔力道具，无法使用一键功能！</div><br/>
        <div class="name">花盆({{ potNum }})<a href="javascript:;" @click="load">刷新</a></div>
        <div class="list">
          <div class="row">空花盆:{{ emptyCount }} [<a href="javascript:;" @click="openSeed">播种</a>]</div>
          <div v-for="p in plots" :key="'g'+p.index" class="row" v-if="p.status !== 'empty'">
            <template v-if="p.status === 'growing'">{{ p.crop }} 生长中，剩余 {{ remainText(p.remain) }}</template>
            <template v-else>{{ p.crop }} <b style="color:#1a9e1a">已成熟</b> [<a href="javascript:;" @click="harvest(p.index)">收获</a>]</template>
          </div>
        </div>
        <a href="javascript:;" @click="addPot">添置新花盆</a><br/>
        <div class="name">消息</div>
        <div class="list">您没有消息.</div>
        <div class="name">花园设施</div>
        <div class="list">
          <div class="row">公告板:劳动光荣，偷窃可耻！ <a href="javascript:;" @click="demo('公告板')">更改</a></div>
          <div class="row">花园名:{{ nick }}的花园 <a href="javascript:;" @click="demo('花园名')">更改</a></div>
        </div>
        <div class="module-content" v-if="seedBox" style="background:#e3eef8">
          <div>花盆：<select v-model="selPot"><option value="">选空花盆</option><option v-for="p in emptyPlots" :key="p.index" :value="p.index">第 {{ p.index + 1 }} 个</option></select></div>
          <div v-for="c in crops" :key="c.name" class="seedline">
            <a href="javascript:;" @click="plant(c.name)">{{ c.name }}</a>
            <span class="txt-fade">（种子{{ c.seed }} · {{ c.secs }}秒成熟 · 收获{{ c.sell }}）</span><br/>
          </div>
        </div>
      </template>

      <!-- 好友 -->
      <template v-if="cur === 'friend'">
        <div class="list">
          <div v-for="(f, i) in friends" :key="f.id" class="row">{{ i + 1 }}.<a href="javascript:;" @click="$router.push('/user/'+f.id)"><font :color="f.color || '#004299'">{{ f.nickname }}</font></a><br/></div>
        </div>
        <div class="module-content">(第<b>1</b>/1页/共{{ friends.length }}条记录)</div>
        <a href="javascript:;" @click="cur='garden'">花园</a>&gt;好友<br/>
      </template>

      <!-- 花房 -->
      <template v-if="cur === 'basket'">
        <div class="name">【我的花篮|<a href="javascript:;" @click="demo('我的花瓶')">我的花瓶</a>】</div>
        <div class="text" v-if="basket.length"><div v-for="f in basket" :key="f.flower">{{ f.flower }} × {{ f.count }}</div></div>
        <div class="text" v-else>暂无记录！</div>
        <br/>
        <a href="javascript:;" @click="demo('送花')">送花</a>.<a href="javascript:;" @click="cur='room'">合成</a>.<a href="javascript:;" @click="demo('送花记录')">送花记录</a><br/>
        ----------<br/>
        <a href="javascript:;" @click="cur='garden'">花园</a>&gt;花篮<br/>
      </template>

      <!-- 魔法屋 -->
      <template v-if="cur === 'room'">
        <form @submit.prevent=""><input type="text" v-model.trim="wd" maxlength="8" size="8" /><input type="submit" value="搜索" /></form>
        <div class="list">
          <div v-for="s in synList" :key="s.name" class="row">
            {{ s.index }}.<a href="javascript:;" @click="syn(s)">{{ s.name }}</a><br/>
            <span v-for="m in s.mats" :key="m.flower">{{ m.flower }}({{ m.need }}/{{ m.have }})<br/></span>
            <span v-if="s.can">可合成！<a href="javascript:;" @click="syn(s)">[合成]</a></span>
            <span v-else style="color:#999">所需花朵不足，您目前还不能合成。</span>
          </div>
        </div>
        <div class="module-content">(第<b>1</b>/1页/共{{ synList.length }}条记录)</div>
        <a href="javascript:;" @click="cur='garden'">花园</a>&gt;魔法屋<br/>
      </template>

      <!-- 活动 -->
      <template v-if="cur === 'active'">
        <div class="name">追寻远古花园的记忆</div>
        <template v-if="!selAct">
          <div class="module-content">
            <div v-for="a in activities" :key="a.id" style="margin-bottom:6px">
              <a href="javascript:;" @click="selAct = a; amount=1">{{ a.title }}</a><br/>
              <span class="txt-fade">{{ a.desc || '' }}</span>
            </div>
          </div>
          <div class="module-content" v-if="!activities.length"><span class="empty">暂无活动</span></div>
        </template>
        <template v-else>
          【{{ selAct.title }}】<br/>
          <span class="txt-fade">{{ selAct.desc || '' }}</span><br/>
          完成任务需要:<br/>
          <span v-for="n in taskNeeds" :key="n.flower">{{ n.flower }}×{{ n.need }} <span class="txt-fade">(拥有 {{ n.have }})</span><br/></span>
          <br/>完成任务奖励:<br/>
          {{ taskReward }}X1<br/>
          <span>兑换<input type="text" v-model.number="amount" maxlength="1" size="2" />颗。<br/></span>
          <a href="javascript:;" @click="submitActivity">完成任务</a><br/>
          <a href="javascript:;" @click="selAct = null">&lt;&lt;返回活动列表</a><br/>
        </template>
        <br/>
        <a href="javascript:;" @click="cur='garden'">花园</a>&gt;活动&gt;参与任务<br/>
      </template>

      <div class="module-title"><a href="javascript:;" @click="demo('背包')">背包</a>.<a href="javascript:;" @click="demo('商店')">商店</a>.<a href="javascript:;" @click="demo('排行')">排行</a>.<a href="javascript:;" @click="demo('帮助')">帮助</a>.<a href="javascript:;" @click="$router.push('/channel/1')">论坛</a></div>
      <p style="color:#c00;padding:0 5px" v-if="msg">{{ msg }}</p>
      <p style="color:#1a9e1a;padding:0 5px" v-if="okMsg">{{ okMsg }}</p>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Garden',
  data () {
    return {
      cur: 'garden', plots: [], crops: [], coins: 0, level: 1, okMsg: '', msg: '',
      seedBox: false, selPot: '', friends: [], basket: [], synList: [], wd: '', activities: [], selAct: null, amount: 1
    }
  },
  computed: {
    nick () { return (this.$store.state.user || {}).nickname || '夜凌云' },
    potNum () { return this.plots.length },
    emptyCount () { return this.plots.filter(p => p.status === 'empty').length },
    emptyPlots () { return this.plots.filter(p => p.status === 'empty') },
    taskNeeds () {
      const have = {}
      this.basket.forEach(f => { have[f.flower] = f.count })
      return [ { flower: '玫瑰花', need: 6, have: have['玫瑰花'] || 0 }, { flower: '郁金香', need: 6, have: have['郁金香'] || 0 } ]
    },
    taskReward () { return '朝暮盈霄花' }
  },
  mounted () { this.loadAll() },
  methods: {
    loadAll () {
      this.load()
      api.get('/friends').then(r => { if (r.code === 0) this.friends = r.data.friends })
      api.get('/games/garden/basket').then(r => { if (r.code === 0) this.basket = r.data })
      api.get('/games/garden/synlist').then(r => { if (r.code === 0) this.synList = r.data })
      api.get('/garden-activities').then(r => { if (r.code === 0) this.activities = r.data })
    },
    load () {
      api.get('/games/garden/view').then(r => {
        if (r.code === 0) { this.plots = r.data.plots || []; this.crops = r.data.crops || []; this.coins = r.data.coins || 0 }
      })
    },
    icon (p) {
      if (p.status === 'empty') return 'm_s_1.gif'
      if (p.status === 'growing') return 'm_s_3.gif'
      return 'm_s_10.gif'
    },
    tip (p) {
      if (p.status === 'empty') return '空花盆'
      if (p.status === 'growing') return p.crop + ' ' + this.remainText(p.remain)
      return p.crop + ' 可收'
    },
    remainText (s) { if (s <= 0) return '即将成熟'; if (s < 60) return s + '秒'; return Math.floor(s / 60) + '分' },
    openSeed () { this.seedBox = !this.seedBox; this.msg = ''; this.okMsg = '' },
    plant (crop) {
      if (this.selPot === '') { this.msg = '请先选择要播种的花盆'; return }
      api.post('/games/garden/plant', { index: parseInt(this.selPot), crop }).then(r => {
        if (r.code === 0) { this.okMsg = '已种下' + crop; this.seedBox = false; this.selPot = ''; this.load() } else this.msg = r.msg
      })
    },
    harvest (idx) {
      api.post('/games/garden/harvest', { index: idx }).then(r => {
        if (r.code === 0) { this.okMsg = '收获花朵：' + (r.data.flower || '') + ' 金币 +' + r.data.gain; this.load() } else this.msg = r.msg
      })
    },
    addPot () {
      api.post('/games/garden/addpot').then(r => { if (r.code === 0) { this.okMsg = '添置花盆成功，金币 -100'; this.load() } else this.msg = r.msg })
    },
    syn (s) {
      api.post('/games/garden/synthesize', { flower: s.name }).then(r => {
        if (r.code === 0) { this.okMsg = '合成成功，获得「' + s.name + '」'; this.reloadAll() } else this.msg = r.msg
      })
    },
    reloadAll () {
      this.load()
      api.get('/games/garden/basket').then(r => { if (r.code === 0) this.basket = r.data })
      api.get('/games/garden/synlist').then(r => { if (r.code === 0) this.synList = r.data })
    },
    viewAct (a) {
      this.okMsg = ''; this.msg = ''
      this.okMsg = '【' + a.title + '】' + (a.desc || '参与更多敬请期待')
    },
    submitActivity () {
      this.msg = ''; this.okMsg = ''
      if (!this.selAct) return
      const amt = this.amount && this.amount > 0 && this.amount < 10 ? this.amount : 1
      api.post('/games/garden/activity-submit', { id: this.selAct.id, amount: amt }).then(r => {
        if (r.code === 0) { this.okMsg = '任务完成！获得「' + r.data.reward + '」×' + r.data.amount; this.loadAll() } else this.msg = r.msg
      })
    },
    demo (t) { alert('魔法花园 · ' + t + '（演示入口）') }
  }
}
</script>

<style scoped>
.garden-nav a { margin-left: 4px; }
.garden-nav span.cur, .garden-nav a.cur { font-weight: bold; color: #98d2ff; }
.g-main { padding: 0 3px; }
.noteico { width: 14px; height: 14px; vertical-align: -2px; }
.userline { font-size: 14px; }
.plots { padding: 3px 2px; }
.plots img { width: 44px; height: 44px; margin: 2px 4px 2px 0; vertical-align: middle; }
.seedline { padding: 2px 0; }
</style>
