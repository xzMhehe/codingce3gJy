<template>
  <div>
    <div class="module-title">
      <a :class="{ cur: cur === 'home' }" href="javascript:;" @click="cur='home'">特权首页</a>
      <a :class="{ cur: cur === 'level' }" href="javascript:;" @click="cur='level'">成长体系</a>
      <a :class="{ cur: cur === 'shop' }" href="javascript:;" @click="cur='shop'">开通特权</a>
      <a :class="{ cur: cur === 'me' }" href="javascript:;" @click="cur='me'">个人中心</a>
    </div>

    <!-- 首页 -->
    <template v-if="cur === 'home'">
      <div>尊贵的蓝钻<img :src="pic(data.blue.icon)" alt="." /><template v-if="data.blue.active">（Lv.{{ data.blue.lv }}）</template><template v-else>（未开通）</template>
        <a href="javascript:;" @click="cur='me'">个人中心</a></div>
      <div>超级QQ<img :src="pic(data.qq.icon)" alt="." /><template v-if="data.qq.active">（Lv.{{ data.qq.lv }}）</template><template v-else>（未开通）</template></div>
      <div class="module-content">
        <p><img src="/static/picture/qs_05.gif" alt="." /> 在线升级提速</p>
        <p><img src="/static/picture/noble_2_1.gif" alt="." /> 酷炫特权图标</p>
        <p><img src="/static/picture/chuping.jpg" alt="." /> 超多游戏特权</p>
      </div>
      <div class="bodule-title"><font color="#FF0000">【特权开通】</font></div>
      <div class="module-content">
        <div v-for="p in data.plans" :key="p.id">{{ p.id }}.<a href="javascript:;" @click="activate(p)">{{ p.name }}({{ p.cost }}金币)</a></div>
      </div>
    </template>

    <!-- 成长体系 -->
    <template v-if="cur === 'level'">
      <div class="bodule-title"><font color="#FF0000">【蓝钻成长体系】</font></div>
      <div class="module-content">
        <div class="row00 noble-hd">
          <span style="display:inline-block;width:30px;text-align:center">等级</span>
          <span style="display:inline-block;width:50px;text-align:center">图标</span>
          <span style="display:inline-block;width:80px;text-align:center">所需成长值</span>
        </div>
        <div v-for="l in data.levels" :key="'b'+l.Lv" class="row00">
          <span style="display:inline-block;width:30px;text-align:center">{{ l.Lv }}</span>
          <span style="display:inline-block;width:50px;text-align:center"><img :src="pic(l.Blue)" alt="." /></span>
          <span style="display:inline-block;width:80px;text-align:center">{{ l.Exp }} 点</span>
        </div>
      </div>

      <div class="bodule-title"><font color="#FF0000">【超Q成长体系】</font></div>
      <div class="module-content">
        <div class="row00 noble-hd">
          <span style="display:inline-block;width:30px;text-align:center">等级</span>
          <span style="display:inline-block;width:50px;text-align:center">图标</span>
          <span style="display:inline-block;width:80px;text-align:center">所需成长值</span>
        </div>
        <div v-for="l in data.levels" :key="'q'+l.Lv" class="row00">
          <span style="display:inline-block;width:30px;text-align:center">{{ l.Lv }}</span>
          <span style="display:inline-block;width:50px;text-align:center"><img :src="pic(l.QQ)" alt="." /></span>
          <span style="display:inline-block;width:80px;text-align:center">{{ l.Exp }} 点</span>
        </div>
      </div>

      <div class="bodule-title"><font color="#FF0000">【每日成长说明】</font></div>
      <div class="module-content">
        每日可获 <font color="#FF0000">{{ data.blue.speed || 10 }}</font> 点成长值。<br>
        在线时长、签到、互动均可加速成长，更多特权敬请期待。<br>
        <a href="javascript:;" @click="cur='shop'">开通/续费贵族</a>.<a href="javascript:;" @click="cur='me'">查看我的成长</a>
      </div>
    </template>

    <!-- 开通特权 -->
    <template v-if="cur === 'shop'">
      <template v-if="!selPlan">
        <div class="bodule-title"><font color="#FF0000">【特权开通】</font></div>
        <div class="module-content">
          <div v-for="(p,i) in data.plans" :key="p.id">{{ i+1 }}.<a href="javascript:;" @click="selPlan=p">{{ p.name }}({{ p.cost }}金币)</a></div>
          <div>(第<b>1</b>/1页/共{{ data.plans.length }}条记录)</div>
        </div>
      </template>
      <template v-else>
        <div class="bodule-title"><font color="#FF0000">☆{{ selPlan.name }}☆</font></div>
        <div class="module-content">
          商店&gt;{{ selPlan.name }}<br>
          贵族类型：{{ selPlan.type === 'blue' ? '蓝钻' : '超Q' }}<br>
          订购价格：{{ selPlan.cost }}金币/月<br>
          成长速度：10点/天<br>
          赠送经验：{{ selPlan.gain }}点<br>
          <a href="javascript:;" @click="activate(selPlan)">购买</a>.<a href="javascript:;" @click="selPlan=null">返回</a><br>
        </div>
      </template>
    </template>

    <!-- 个人中心 -->
    <template v-if="cur === 'me'">
      <div class="bodule-title"><font color="#FF0000">【个人中心】</font></div>
      <div class="module-content">
        <template v-if="data.blue.active">
          尊贵的蓝钻，{{ data.nickname }}<br>
          目前等级：<img :src="pic(data.blue.icon)" :alt="'Lv' + data.blue.lv" /><br>
          等级经验：{{ data.blue.exp }}<br>
          经验速度：{{ data.blue.speed }}点/天 <a href="javascript:;" @click="cur='shop'">提速</a><br>
          开通时间：{{ fmt(data.blue.start) }}<br>
          到期时间：{{ fmt(data.blue.end) }} <a href="javascript:;" @click="cur='shop'">续费</a><br>
        </template>
        <template v-else><span class="txt-fade">尚未开通蓝钻，<a href="javascript:;" @click="cur='shop'">去开通</a></span></template>
      </div>
      <div class="module-content">
        <template v-if="data.qq.active">
          超级QQ，{{ data.nickname }}<br>
          目前等级：<img :src="pic(data.qq.icon)" :alt="'Lv' + data.qq.lv" /><br>
          等级经验：{{ data.qq.exp }}<br>
          经验速度：{{ data.qq.speed }}点/天 <a href="javascript:;" @click="cur='shop'">提速</a><br>
          开通时间：{{ fmt(data.qq.start) }}<br>
          到期时间：{{ fmt(data.qq.end) }} <a href="javascript:;" @click="cur='shop'">续费</a><br>
        </template>
        <template v-else><span class="txt-fade">尚未开通超Q，<a href="javascript:;" @click="cur='shop'">去开通</a></span></template>
      </div>

      <div class="bodule-title"><font color="#FF0000">【蓝钻排行】</font></div>
      <div class="module-content">
        <div v-for="(r,i) in data.blue_rank" :key="'b'+r.user_id">{{ i+1 }}.<a href="javascript:;" @click="$router.push('/user/'+r.user_id)"><font :color="r.color || '#ff0000'">{{ r.nickname }}</font></a>({{ r.exp }}点成长值↑)</div>
        <div v-if="!data.blue_rank || !data.blue_rank.length"><span class="empty">暂无蓝钻友友</span></div>
      </div>
      <div class="bodule-title"><font color="#FF0000">【超Q排行】</font></div>
      <div class="module-content">
        <div v-for="(r,i) in data.qq_rank" :key="'q'+r.user_id">{{ i+1 }}.<a href="javascript:;" @click="$router.push('/user/'+r.user_id)"><font :color="r.color || '#ff0000'">{{ r.nickname }}</font></a>({{ r.exp }}点成长值↑)</div>
        <div v-if="!data.qq_rank || !data.qq_rank.length"><span class="empty">暂无超Q友友</span></div>
      </div>
    </template>

    <p style="color:#c00;padding:0 5px" v-if="msg">{{ msg }}</p>
    <p style="color:#1a9e1a;padding:0 5px" v-if="okMsg">{{ okMsg }}</p>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Noble',
  data () { return { cur: 'home', selPlan: null, data: { plans: [], levels: [], blue_rank: [], qq_rank: [], blue: {}, qq: {} }, msg: '', okMsg: '' } },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/noble').then(r => { if (r.code === 0) this.data = r.data })
    },
    pic (f) { return f ? '/static/picture/' + f : '' },
    activate (p) {
      api.post('/noble/activate', { plan_id: p.id }).then(r => {
        if (r.code === 0) { this.okMsg = '开通成功！' + (p.type === 'blue' ? '蓝钻' : '超Q') + ' +' + p.gain + ' 成长'; this.load() } else this.msg = r.msg
      })
    },
    fmt (t) {
      if (!t) return '—'
      const d = new Date(t)
      return d.getFullYear() + '-' + (d.getMonth() + 1 < 10 ? '0' : '') + (d.getMonth() + 1) + '-' + (d.getDate() < 10 ? '0' : '') + d.getDate() + ' ' + (d.getHours() < 10 ? '0' : '') + d.getHours() + ':' + (d.getMinutes() < 10 ? '0' : '') + d.getMinutes()
    }
  }
}
</script>

<style scoped>
.module-title a { margin-right: 10px; }
.module-title a.cur { color: #c00; font-weight: bold; }
.noble-hd { color:#004299; border-bottom:1px dashed #9FC6EC; padding-bottom:4px; margin-bottom:4px; font-weight:bold; }
</style>
