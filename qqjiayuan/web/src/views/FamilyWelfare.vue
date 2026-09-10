<template>
  <div>
    <!-- 参考：诺哈 family_welfare（每日分财富/家族签到） -->
    <div class="bar"><a href="javascript:;" @click="$router.push('/families')">家族</a>&gt;每日分财富<br></div>

    <div class="module-title">每日分财富</div>
    <div class="module-content">
      <img src="/static/picture/tree.gif" width="130" height="100" alt="守护树"><br>
      每日分财富 最高100000GB<br>
      <template v-if="fam.my_role">
        <template v-if="fam.signed_today"><span style="color:#1a9e1a">今天已在家族签到 ✓（+20经验 +5G币）</span></template>
        <template v-else><button class="btn" @click="doSign">立即签到分财富</button></template>
      </template>
      <template v-else><span class="empty">加入家族后可参与每日分财富</span></template>
    </div>

    <div class="module-content">
      <a href="javascript:;" @click="$router.push('/family/' + $route.params.id)">返回家族</a><br>
    </div>

    <p v-if="msg" style="color:#c00;padding:0 5px">{{ msg }}</p>
    <p v-if="okMsg" style="color:#1a9e1a;padding:0 5px">{{ okMsg }}</p>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'FamilyWelfare',
  data () {
    return { fam: {}, msg: '', okMsg: '' }
  },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/families/' + this.$route.params.id).then(r => {
        if (r.code === 0) this.fam = r.data
      })
    },
    doSign () {
      this.msg = ''; this.okMsg = ''
      api.post('/families/' + this.$route.params.id + '/signin').then(r => {
        if (r.code === 0) { this.okMsg = '签到成功 +20经验 +5G币，今日财富已分'; this.load() }
        else this.msg = r.msg
      })
    }
  }
}
</script>
