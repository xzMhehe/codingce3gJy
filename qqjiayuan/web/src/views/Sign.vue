<template>
  <div>
    <div class="bar">【每日签到】</div>
    <div class="module-content" style="text-align:center;padding:14px 8px">
      <template v-if="signed">
        <div style="font-size:30px;color:#1a9e1a">✔</div>
        <p>今天已签到，连续 <b style="color:#e05a00">{{ consec }}</b> 天，共 <b>{{ totalDays }}</b> 天</p>
        <p style="color:#999">明天继续来哦，连续签到金币更多！</p>
      </template>
      <template v-else>
        <div style="font-size:30px">📅</div>
        <p>今天还没签到</p>
        <p><button class="btn" @click="doSign" :disabled="doing">立 即 签 到</button></p>
        <p style="color:#999">签到+20经验，奖励金币 = 10 + 连续天数（连续7天封顶17金币）</p>
      </template>
      <p v-if="msg" style="color:#c00">{{ msg }}</p>
      <p v-if="okMsg" style="color:#1a9e1a">{{ okMsg }}</p>
    </div>

    <div class="module-title">【签到排行榜】连续天数 TOP10</div>
    <ul class="dtuser" v-if="rank.length">
      <li v-for="(r,i) in rank" :key="r.user_id">
        {{ i+1 }}.<a href="javascript:;" @click="$router.push('/user/'+r.user_id)"><font :color="r.color || '#004299'">{{ r.nickname }}</font></a>
        —— 连续 <b style="color:#e05a00">{{ r.consec }}</b> 天
      </li>
    </ul>
    <div class="module-content" v-else><span class="empty">还没有友友签到</span></div>
    <div class="login-tips">今日全站已有 {{ todayCount }} 位友友签到</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Sign',
  data () {
    return { signed: false, consec: 0, totalDays: 0, rank: [], todayCount: 0, msg: '', okMsg: '', doing: false }
  },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/signin/info').then(r => {
        if (r.code === 0) {
          this.signed = r.data.signed_today
          this.consec = r.data.consec
          this.totalDays = r.data.total_days
          this.rank = r.data.rank
          this.todayCount = r.data.today_count
        }
      })
    },
    doSign () {
      this.doing = true
      api.post('/signin').then(r => {
        this.doing = false
        if (r.code === 0) {
          this.okMsg = '签到成功！连续' + r.data.consec + '天，获得' + r.data.reward + '金币+20经验'
          this.load()
        } else {
          this.msg = r.msg
        }
      })
    }
  }
}
</script>
