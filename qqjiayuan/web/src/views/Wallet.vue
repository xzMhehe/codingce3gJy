<template>
  <div>
    <div class="module-title">我的钱包</div>
    <div class="module-content plist">
      <div class="row00">当前金币：<b style="color:#e05a00">{{ data.coins || 0 }}</b></div>
      <div class="row00">银行存款：<b style="color:#1a9e1a">{{ data.bank || 0 }}</b></div>
      <div class="row00">累计打工：{{ data.work_total || 0 }} 次</div>
      <div class="row00"><a href="javascript:;" @click="$router.push('/play')">去社区银行 / 打工 / 挖宝 &gt;&gt;</a></div>
    </div>

    <div class="module-title">慈善捐款记录</div>
    <div class="module-content" v-if="data.donations && data.donations.length">
      <div v-for="d in data.donations" :key="d.id">{{ fmt(d.created_at) }} —— 捐款 {{ d.amount }} 金币</div>
    </div>
    <div class="module-content" v-else><span class="empty">暂无捐款记录</span></div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Wallet',
  data () { return { data: {} } },
  mounted () { api.get('/wallet').then(r => { if (r.code === 0) this.data = r.data }) },
  methods: {
    fmt (t) {
      if (!t) return ''
      const d = new Date(t)
      return d.getFullYear() + '/' + (d.getMonth() + 1) + '/' + d.getDate() + ' ' + d.getHours() + ':' + (d.getMinutes() < 10 ? '0' : '') + d.getMinutes()
    }
  }
}
</script>
