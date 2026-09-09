<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">我的地盘</a>&gt;特权<br></div>
    <div class="module-title">
      <a :class="{ cur: cur === 'home' }" href="javascript:;" @click="cur='home'">会员中心</a>
      <a :class="{ cur: cur === 'me' }" href="javascript:;" @click="cur='me'">我的会员</a>
      <a :class="{ cur: cur === 'shop' }" href="javascript:;" @click="cur='shop'">商店</a>
      <a :class="{ cur: cur === 'level' }" href="javascript:;" @click="cur='level'">成长体系</a>
    </div>

    <!-- ===== 会员中心首页（复刻诺哈 vip/index.asp） ===== -->
    <template v-if="cur === 'home'">
      <!-- 会员状态 -->
      <div class="module-content">
        <template v-if="data.blue.active || data.qq.active">
          尊贵的会员，
          <span v-if="data.blue.active">蓝钻<img :src="pic(data.blue.icon)" alt="." />Lv.{{ data.blue.lv }}</span>
          <span v-if="data.blue.active && data.qq.active"> / </span>
          <span v-if="data.qq.active">超Q<img :src="pic(data.qq.icon)" alt="." />Lv.{{ data.qq.lv }}</span>
          <a href="javascript:;" @click="cur='me'">个人中心</a><br>
        </template>
        <template v-else>亲爱的准会员，<a href="javascript:;" @click="cur='shop'">开通会员</a><br></template>
      </div>

      <!-- 购买方案 -->
      <div class="bodule-title"><font color="#FF0000">【购买会员】</font></div>
      <div class="module-content">
        <div v-for="(p,i) in data.plans" :key="p.id">{{ i+1 }}.<a href="javascript:;" @click="openPlan(p)">{{ p.name }}</a><br></div>
      </div>

      <!-- 会员排行 -->
      <div class="bodule-title"><font color="#FF0000">【蓝钻排行】</font></div>
      <div class="module-content">
        <div v-for="(r,i) in data.blue_rank" :key="'b'+r.user_id">{{ i+1 }}.<a href="javascript:;" @click="$router.push('/user/'+r.user_id)"><font :color="r.color || '#ff0000'">{{ r.nickname }}</font></a>(<img :src="pic(blueIcon(r.exp))" alt="." />{{ r.exp }}点)<br></div>
        <div v-if="!data.blue_rank || !data.blue_rank.length"><span class="empty">暂无蓝钻友友</span></div>
      </div>
      <div class="bodule-title"><font color="#FF0000">【超Q排行】</font></div>
      <div class="module-content">
        <div v-for="(r,i) in data.qq_rank" :key="'q'+r.user_id">{{ i+1 }}.<a href="javascript:;" @click="$router.push('/user/'+r.user_id)"><font :color="r.color || '#ff0000'">{{ r.nickname }}</font></a>(<img :src="pic(qqIcon(r.exp))" alt="." />{{ r.exp }}点)<br></div>
        <div v-if="!data.qq_rank || !data.qq_rank.length"><span class="empty">暂无超Q友友</span></div>
      </div>

      <!-- 会员介绍（等级表） -->
      <div class="bodule-title"><font color="#FF0000">【会员介绍】</font></div>
      <div class="module-content">
        <div class="row00 noble-hd">
          <span style="display:inline-block;width:50px;text-align:center">等级</span>
          <span style="display:inline-block;width:50px;text-align:center">图标</span>
          <span style="display:inline-block;width:90px;text-align:center">成长值</span>
        </div>
        <div v-for="l in data.levels" :key="'b'+l.Lv" class="row00">
          <span style="display:inline-block;width:50px;text-align:center">{{ l.Lv }}</span>
          <span style="display:inline-block;width:50px;text-align:center"><img :src="pic(l.Blue)" alt="." /></span>
          <span style="display:inline-block;width:90px;text-align:center">{{ l.Exp }} 点</span>
        </div>
      </div>
    </template>

    <!-- ===== 我的会员（复刻诺哈 my_vip.asp） ===== -->
    <template v-if="cur === 'me'">
      <div class="bodule-title"><font color="#FF0000">【我的会员】</font></div>
      <div class="module-content">
        <template v-if="data.blue.active">
          尊贵的蓝钻，{{ data.nickname }}<br>
          目前等级：<img :src="pic(data.blue.icon)" :alt="'Lv' + data.blue.lv" />({{ data.blue.exp }})<br>
          成长速度：{{ data.blue.speed }}/天 <a href="javascript:;" @click="cur='shop'">提速</a><br>
          开通时间：{{ fmt(data.blue.start) }}<br>
          到期时间：{{ fmt(data.blue.end) }}（剩{{ data.blue.days_left }}天） <a href="javascript:;" @click="cur='shop'">续费</a><br>
        </template>
        <template v-else><span class="txt-fade">尚未开通蓝钻，<a href="javascript:;" @click="cur='shop'">去开通</a></span></template>
      </div>
      <div class="module-content">
        <template v-if="data.qq.active">
          超级QQ，{{ data.nickname }}<br>
          目前等级：<img :src="pic(data.qq.icon)" :alt="'Lv' + data.qq.lv" />({{ data.qq.exp }})<br>
          成长速度：{{ data.qq.speed }}/天 <a href="javascript:;" @click="cur='shop'">提速</a><br>
          开通时间：{{ fmt(data.qq.start) }}<br>
          到期时间：{{ fmt(data.qq.end) }}（剩{{ data.qq.days_left }}天） <a href="javascript:;" @click="cur='shop'">续费</a><br>
        </template>
        <template v-else><span class="txt-fade">尚未开通超Q，<a href="javascript:;" @click="cur='shop'">去开通</a></span></template>
      </div>
      <div class="module-content txt-fade">开通后每天按「成长速度」自动增长成长值；到期后保留等级和成长值30天，期间可续费，超30天将清空重新开始。</div>
    </template>

    <!-- ===== 商店（复刻诺哈 shop_list.asp → shop.asp → shop_buy.asp / shop_send.asp） ===== -->
    <template v-if="cur === 'shop'">
      <template v-if="!selPlan">
        <div class="bodule-title"><font color="#FF0000">【商店中心】</font></div>
        <div class="module-content">
          <div v-for="(p,i) in data.plans" :key="p.id">{{ i+1 }}.<a href="javascript:;" @click="openPlan(p)">{{ p.name }}</a><br></div>
        </div>
      </template>
      <template v-else>
        <div class="module-content">
          <a href="javascript:;" @click="cur='shop'; selPlan=null">商店</a>&gt;{{ selPlan.name }}<br>
          订购价格：{{ selPlan.cost }}G币/{{ selPlan.days }}天<br>
          成长速度：{{ selPlan.speed }}点/天<br>
          成长赠送：{{ selPlan.gain }}点<br>
          <template v-if="selPlan.limit">每号限购：{{ selPlan.limit }}<br></template>
          <template v-if="selPlan.stock">库存数量：{{ selPlan.stock }}<br></template>
          销售数量：{{ selPlan.sales }}<br>
          <a href="javascript:;" @click="buyOpen = !buyOpen">购买</a>.<a href="javascript:;" @click="giftOpen = !giftOpen">赠送</a><br>

          <!-- 购买 -->
          <div v-if="buyOpen" class="module-content deep">
            购买数量（月）：<input type="text" v-model.number="buyNum" size="2" maxlength="2" value="1"><br>
            <input type="submit" value="确定购买" @click="doActivate"><br>
          </div>

          <!-- 赠送 -->
          <div v-if="giftOpen" class="module-content deep">
            对方号码：<input type="text" v-model.trim="giftTo" size="8"><br>
            赠送数量（月）：<input type="text" v-model.number="giftNum" size="2" maxlength="2" value="1"><br>
            <input type="submit" value="确定赠送" @click="doGift"><br>
          </div>
        </div>
      </template>
    </template>

    <!-- ===== 成长体系 ===== -->
    <template v-if="cur === 'level'">
      <div class="bodule-title"><font color="#FF0000">【蓝钻成长体系】</font></div>
      <div class="module-content">
        <div class="row00 noble-hd">
          <span style="display:inline-block;width:50px;text-align:center">等级</span>
          <span style="display:inline-block;width:50px;text-align:center">图标</span>
          <span style="display:inline-block;width:90px;text-align:center">所需成长值</span>
        </div>
        <div v-for="l in data.levels" :key="'b'+l.Lv" class="row00">
          <span style="display:inline-block;width:50px;text-align:center">{{ l.Lv }}</span>
          <span style="display:inline-block;width:50px;text-align:center"><img :src="pic(l.Blue)" alt="." /></span>
          <span style="display:inline-block;width:90px;text-align:center">{{ l.Exp }} 点</span>
        </div>
      </div>
      <div class="bodule-title"><font color="#FF0000">【超Q成长体系】</font></div>
      <div class="module-content">
        <div class="row00 noble-hd">
          <span style="display:inline-block;width:50px;text-align:center">等级</span>
          <span style="display:inline-block;width:50px;text-align:center">图标</span>
          <span style="display:inline-block;width:90px;text-align:center">所需成长值</span>
        </div>
        <div v-for="l in data.levels" :key="'q'+l.Lv" class="row00">
          <span style="display:inline-block;width:50px;text-align:center">{{ l.Lv }}</span>
          <span style="display:inline-block;width:50px;text-align:center"><img :src="pic(l.QQ)" alt="." /></span>
          <span style="display:inline-block;width:90px;text-align:center">{{ l.Exp }} 点</span>
        </div>
      </div>
      <div class="bodule-title"><font color="#FF0000">【每日成长说明】</font></div>
      <div class="module-content">
        会员有效期内每天按「成长速度」自动增长成长值。<br>
        在线时长、签到、互动均可加速成长，更多特权敬请期待。<br>
        <a href="javascript:;" @click="cur='shop'">开通/续费会员</a>.<a href="javascript:;" @click="cur='me'">查看我的成长</a>
      </div>
    </template>

    <p style="color:#c00;padding:0 5px" v-if="msg">{{ msg }}</p>
    <p style="color:#1a9e1a;padding:0 5px" v-if="okMsg">{{ okMsg }}</p>

    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">我的地盘</a>&gt;特权<br></div>
  </div>
</template>

<script>
import api from '../api'

const LEVELS = [
  { Lv: 1, Exp: 0, Blue: 'noble_2_1.gif', QQ: 'noble_1_1.gif' },
  { Lv: 2, Exp: 600, Blue: 'noble_2_2.gif', QQ: 'noble_1_2.gif' },
  { Lv: 3, Exp: 1800, Blue: 'noble_2_3.gif', QQ: 'noble_1_3.gif' },
  { Lv: 4, Exp: 3600, Blue: 'noble_2_4.gif', QQ: 'noble_1_4.gif' },
  { Lv: 5, Exp: 6000, Blue: 'noble_2_5.gif', QQ: 'noble_1_5.gif' },
  { Lv: 6, Exp: 10800, Blue: 'noble_2_6.gif', QQ: 'noble_1_6.gif' },
  { Lv: 7, Exp: 32400, Blue: 'noble_2_7.gif', QQ: 'noble_1_7.gif' },
  { Lv: 8, Exp: 46800, Blue: 'noble_2_8.gif', QQ: 'noble_1_8.gif' }
]

export default {
  name: 'Noble',
  data () {
    return {
      cur: 'home', selPlan: null, buyOpen: false, giftOpen: false,
      buyNum: 1, giftTo: '', giftNum: 1,
      data: { plans: [], levels: LEVELS, blue_rank: [], qq_rank: [], blue: {}, qq: {} },
      msg: '', okMsg: ''
    }
  },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/noble').then(r => {
        if (r.code === 0) { this.data = r.data; this.data.levels = this.data.levels || LEVELS }
        else this.msg = r.msg
      })
    },
    pic (f) { return f ? '/static/picture/' + f : '' },
    blueIcon (exp) {
      let icon = 'noble_2_1.gif'
      LEVELS.forEach(l => { if (exp >= l.Exp) icon = l.Blue })
      return icon
    },
    qqIcon (exp) {
      let icon = 'noble_1_1.gif'
      LEVELS.forEach(l => { if (exp >= l.Exp) icon = l.QQ })
      return icon
    },
    openPlan (p) {
      this.selPlan = p
      this.cur = 'shop'
      this.buyOpen = false
      this.giftOpen = false
      this.buyNum = 1
      this.giftNum = 1
    },
    doActivate () {
      const num = Math.max(1, Math.floor(this.buyNum || 1))
      api.post('/noble/activate', { plan_id: this.selPlan.id, num }).then(r => {
        if (r.code === 0) {
          this.okMsg = '购买成功！' + (r.data.type === 'blue' ? '蓝钻' : '超Q') + ' +' + r.data.gain + ' 成长，每天+' + r.data.speed + '点'
          this.msg = ''
          this.buyOpen = false
          this.load()
        } else { this.msg = r.msg; this.okMsg = '' }
      })
    },
    doGift () {
      const num = Math.max(1, Math.floor(this.giftNum || 1))
      if (!this.giftTo) { this.msg = '请输入对方号码'; return }
      api.post('/noble/gift', { plan_id: this.selPlan.id, to: this.giftTo, num }).then(r => {
        if (r.code === 0) {
          this.okMsg = '赠送成功！已送「' + r.data.type + '」给 ' + r.data.to
          this.msg = ''
          this.giftOpen = false
          this.giftTo = ''
          this.load()
        } else { this.msg = r.msg; this.okMsg = '' }
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
.deep { padding: 6px; margin-top: 4px; }
</style>