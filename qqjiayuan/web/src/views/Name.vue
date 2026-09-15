<template>
  <div>
    <!-- ==================== 个性昵称 · 设置页 ==================== -->
    <div class="bar"><a href="javascript:;" @click="$router.push('/box')">我的百宝箱</a>&gt;个性昵称</div>
    <template v-if="cur === 'home'">
      <div class="module-content">
        <template v-if="data.opened">
          昵称颜色：<ntext :color="data.color" style="font-weight:bold">{{ user.nickname || '自己' }}</ntext>（当前）<br>
          有效期剩余：<b>{{ data.days_left }}</b> 天<br>
          个性昵称仅用于<strong>配置昵称颜色</strong>，昵称内容请在「编辑资料」里修改。<br>
          -----------<br>
          <a href="javascript:;" @click="cur='buy'">续费/增加时长</a>.<a href="javascript:;" @click="cur='send'">赠送他人</a>
        </template>
        <template v-else>
          您尚未开通个性昵称功能！<a href="javascript:;" @click="cur='buy'">立即购买</a><br>
          开通即可享受昵称颜色自定义特权！<br>
          您还可以：<a href="javascript:;" @click="cur='send'">赠送他人</a><br>
        </template>
        <br>
        <div class="module-title">【颜色配置】</div>
        <template v-if="data.opened">
          <span v-for="(cc,i) in data.colors" :key="'c'+i">
            <a href="javascript:;" :style="(data.color || '').indexOf(cc) === 0 ? 'border:1px solid #c00' : ''" @click="setColor(cc)"><font :color="cc">{{ cc }}</font></a>.
          </span>
          <br>
          整体单色：<input type="text" v-model.trim="customColor" size="7" maxlength="7" placeholder="#06f">
          <input type="submit" value="确定设置" @click="setCustom"><br>
          <br>
          <div class="module-title">【配色工具】</div>
          推荐方案：<span v-for="(plan,pi) in shownPresets" :key="'ps'+pi">
            <a href="javascript:;" @click="loadSeq(plan)">
              <span v-for="(c,ci) in plan.colors" :key="'d'+ci"><font :color="c">◆</font></span><ntext :color="plan.colors.join(',')">{{ plan.name }}</ntext>
            </a>{{ pi === shownPresets.length - 1 ? '' : ' . ' }}
          </span>.<a href="javascript:;" @click="nextPresets">[换一批]</a><br>
          当前配色：<input type="hidden" /><span v-for="(c,i) in seq" :key="'s'+i">
            <input type="color" :value="toHex6(c)" @change="updSeq(i, $event)">
            <a href="javascript:;" style="color:#c00" @click="seq.splice(i,1)">[删]</a>.
          </span><a href="javascript:;" @click="seq.push('#004C99')">[+加颜色]</a><br>
          <a href="javascript:;" @click="clearSeq">[清空]</a>（颜色少于字数会从头循环）<br>
          预览：<ntext :color="seqColor" style="font-size:16px">{{ user.nickname || '自己' }}</ntext><br>
          <input type="submit" value="确定逐字设置" @click="applySeq"><br>
          <span v-if="msg" style="color:#c00">{{ msg }}</span>
          <span v-if="okMsg" style="color:#1a9e1a">{{ okMsg }}</span>
        </template>
        <template v-else>
          普通用户昵称为默认蓝色。<a href="javascript:;" @click="cur='buy'">开通个性昵称</a>后即可自定义颜色。<br>
        </template>
        <a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;<a href="javascript:;" @click="$router.push('/home')">地盘</a>&gt;个性昵称<br>
      </div>
    </template>

    <!-- ==================== 购买昵称 ==================== -->
    <template v-else-if="cur === 'buy'">
      【购买昵称】<br>
      <div class="module-content">
        购买方式:
        <select v-model.number="buyPid">
          <option disabled value="0">请选择</option>
          <option v-for="p in data.plans" :key="'p'+p.id" :value="p.id">{{ p.name }}</option>
        </select><br>
        购买数量:<input type="text" v-model.number="buyNum" size="3" maxlength="2" value="1"><br>
        <input type="submit" value="确定购买" @click="doBuy"><br>
        <span v-if="buyMsg" style="color:#c00">{{ buyMsg }}</span>
        <span v-if="buyOk" style="color:#1a9e1a">{{ buyOk }}</span>
        <br>
        ----------<br>
        <a href="javascript:;" @click="$router.push('/my-news')">家园</a>&gt;<a href="javascript:;" @click="$router.push('/home')">地盘</a>&gt;<a href="javascript:;" @click="cur='home'">个性昵称</a>&gt;购买<br>
      </div>
    </template>

    <!-- ==================== 赠送昵称 ==================== -->
    <template v-else>
      【赠送昵称】<br>
      <div class="module-content">
        对方号码:<input type="text" v-model.trim="giftTo" size="9"><br>
        赠送方式:
        <select v-model.number="giftPid">
          <option disabled value="0">请选择</option>
          <option v-for="p in data.plans" :key="'g'+p.id" :value="p.id">{{ p.name }}</option>
        </select><br>
        赠送数量:<input type="text" v-model.number="giftNum" size="3" maxlength="2" value="1"><br>
        <input type="submit" value="确定赠送" @click="doSend"><br>
        <span v-if="giftMsg" style="color:#c00">{{ giftMsg }}</span>
        <span v-if="giftOk" style="color:#1a9e1a">{{ giftOk }}</span>
        <br>
        ----------<br>
        <a href="javascript:;" @click="$router.push('/my-news')">家园</a>&gt;<a href="javascript:;" @click="$router.push('/home')">地盘</a>&gt;<a href="javascript:;" @click="cur='home'">个性昵称</a>&gt;赠送他人<br>
      </div>
    </template>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'NameCust',
  data () {
    return {
      cur: 'home',
      data: { opened: false, days_left: 0, color: '#004299', colors: [], plans: [] },
      customColor: '',
      seq: [],
      presetIdx: 0,
      presetCount: 4,
      presets: [
        { name: '炸鱼诱惑', colors: ['#D2B48C', '#800080', '#DAA520', '#FFC0CB', '#EE82EE'] },
        { name: '八彩虹', colors: ['#FF0000', '#FF7F00', '#FFE600', '#00B050', '#3B6AFF', '#800080'] },
        { name: '粉粉少女', colors: ['#FF69B4', '#FF99CC', '#FFB6D9', '#FF8FC0', '#FFC0CB'] },
        { name: '星空紫', colors: ['#800080', '#9B30FF', '#BA55D3', '#C77DFF', '#6A0DAD'] },
        { name: '海洋蓝', colors: ['#004299', '#0074D9', '#3399FF', '#66CCFF', '#001F7A'] },
        { name: '活力橙绿', colors: ['#FF8C00', '#FFC125', '#00B050', '#00CC66', '#FFD700'] },
        { name: '中国红', colors: ['#C00', '#FF4500', '#D98880', '#E74C3C', '#A02020'] },
        { name: '暗夜黑金', colors: ['#000000', '#444444', '#FFD700', '#B8860B', '#666666'] },
        { name: '蜜桃甜心', colors: ['#FFB6C1', '#FF9EAA', '#FF7F7F', '#FFC0CB', '#F98CA5'] },
        { name: '薄荷清爽', colors: ['#7FFFD4', '#40E0D0', '#00CED1', '#20B2AA', '#5F9EA0'] },
        { name: '樱花渐变', colors: ['#FF4E7E', '#FF7EA5', '#FF9FBF', '#FFC0D3', '#FF87A9'] },
        { name: '日出金橙', colors: ['#E06000', '#FF8C1A', '#FFB347', '#FFD37E', '#C0392B'] },
        { name: '深蓝夜空', colors: ['#0B1B4A', '#1F3FA0', '#3B6AFF', '#6FA8FF', '#0A0F2D'] },
        { name: '青柠冰汽', colors: ['#00CC66', '#66FF99', '#CCFF00', '#00B050', '#8DB800'] },
        { name: '香芋紫', colors: ['#B57EDC', '#C39BD3', '#A569BD', '#D2B4DE', '#884EA0'] },
        { name: '海岸线', colors: ['#20B2AA', '#48D1CC', '#7FCFCF', '#2E8B8B', '#66CCC5'] },
        { name: '落日晚霞', colors: ['#FF512F', '#FF7E4A', '#FFB347', '#DD5E89', '#F59E5A'] },
        { name: '森林墨绿', colors: ['#0B6E3A', '#2E8B57', '#4CAF50', '#6FA96F', '#1E5631'] },
        { name: '葡萄美酒', colors: ['#7B1FA2', '#9C27B0', '#BA55D3', '#CE93D8', '#5E1E6B'] },
        { name: '香槟金', colors: ['#B8860B', '#DAA520', '#E8C97A', '#F5DEB3', '#9A7D2E'] },
        { name: '圣诞红绿', colors: ['#C0392B', '#E74C3C', '#2E8B57', '#3CB371', '#8B0000'] },
        { name: '极简灰阶', colors: ['#333333', '#555555', '#777777', '#999999', '#222222'] },
        { name: '冰雪蓝白', colors: ['#5F9EFF', '#87CEEB', '#B0E2FF', '#4682B4', '#A4D3EE'] },
        { name: '墨韵山水', colors: ['#1C1C1C', '#36454F', '#536878', '#708090', '#0F1419'] }
      ],
      msg: '', okMsg: '',
      buyPid: 0, buyNum: 1, buyMsg: '', buyOk: '',
      giftTo: '', giftPid: 0, giftNum: 1, giftMsg: '', giftOk: ''
    }
  },
  computed: {
    user () { return this.$store.state.user || {} },
    seqColor () { return this.seq.join(',') },
    shownPresets () {
      const n = this.presets.length
      const start = (this.presetIdx * this.presetCount) % n
      const out = []
      for (let i = 0; i < Math.min(this.presetCount, n); i++) out.push(this.presets[(start + i) % n])
      return out
    }
  },
  mounted () { this.load() },
  methods: {
    toHex6 (c) {
      c = (c || '').toLowerCase()
      if (/^#[0-9a-f]{6}$/.test(c)) return c
      if (/^#[0-9a-f]{3}$/.test(c)) return '#' + c[1] + c[1] + c[2] + c[2] + c[3] + c[3]
      return '#004299'
    },
    loadSeq (plan) {
      this.seq = plan.colors.slice()
      this.okMsg = '已载入「' + plan.name + '」配色，预览满意后点「确定逐字设置」'
      this.msg = ''
    },
    nextPresets () {
      this.presetIdx = (this.presetIdx + 1) % Math.ceil(this.presets.length / this.presetCount)
    },
    updSeq (i, e) { this.$set(this.seq, i, (e.target.value || '').toLowerCase()) },
    clearSeq () { this.seq = []; this.okMsg = '' },
    async applySeq () {
      this.okMsg = ''
      if (!this.seq.length) { this.msg = '请先添加颜色'; return }
      this.msg = ''
      const r = await api.post('/name/set', { colors: this.seq })
      if (r.code === 0) {
        this.okMsg = r.msg
        this.data.color = r.data.color
        await this.load()
      } else { this.msg = r.msg; this.okMsg = '' }
    },
    async load () {
      const r = await api.get('/name')
      if (r.code === 0) {
        this.data = r.data
        if (r.data.opened) {
          const cs = (r.data.color || '#004299').split(',').map(s => s.trim()).filter(Boolean)
          this.customColor = cs[0]
          this.seq = cs
        }
      }
    },
    async setColor (cc) {
      const r = await api.post('/name/set', { color: cc })
      if (r.code === 0) {
        this.okMsg = r.msg
        this.msg = ''
        this.data.color = cc
        await this.load()
      } else { this.msg = r.msg; this.okMsg = '' }
    },
    setCustom () {
      if (!/^#[0-9a-fA-F]{3,6}$/.test(this.customColor)) { this.msg = '颜色格式不对'; return }
      this.setColor(this.customColor)
    },
    async doBuy () {
      this.buyOk = ''
      const r = await api.post('/name/buy', { pid: this.buyPid, amount: this.buyNum || 0 })
      if (r.code === 0) {
        this.buyOk = r.msg + '花费' + r.data.cost + '元宝'
        this.buyMsg = ''
        await this.load()
        this.cur = 'home'
      } else { this.buyMsg = r.msg }
    },
    async doSend () {
      this.giftOk = ''
      const r = await api.post('/name/send', { to: this.giftTo, pid: this.giftPid, amount: this.giftNum || 0 })
      if (r.code === 0) {
        this.giftOk = r.msg + '已送 ' + r.data.to
        this.giftMsg = ''
        this.giftTo = ''
      } else { this.giftMsg = r.msg }
    }
  }
}
</script>
