<template>
  <div>
    <!-- 在线用户（复刻诺哈 online.html：用户点进个人资料，游客点进 IP 查询；10条/页） -->
    <div class="module-title">【在线用户】</div>

    <div class="list" v-if="rows.length">
      <div class="row00" v-for="(r, i) in rows" :key="r.is_guest ? 'g'+r.ip : 'u'+r.user_id">
        {{ (page - 1) * size + i + 1 }}.<template v-if="!r.is_guest"><a href="javascript:;" @click="$router.push('/user/' + r.user_id)"><font :color="r.color || '#004299'">{{ r.nickname }}</font>({{ r.username }})</a></template>
        <template v-else><a href="javascript:;" @click="$router.push({ path: '/tool/ip', query: { ip: r.ip } })">家园社区游客({{ r.ip }})</a></template>
        [{{ fmt(r.last_active_at) }}]<br>
      </div>
      <div class="pager">
        <a v-if="page > 1" href="javascript:;" @click="go(page - 1)">上页</a>
        第{{ page }}/{{ pages }}页/共{{ total }}条记录
        <a v-if="page < pages" href="javascript:;" @click="go(page + 1)">下页</a><br>
      </div>
    </div>
    <div class="module-content" v-else><span class="empty">暂无在线用户</span></div>

    <div class="bar"><a href="javascript:;" @click="$router.push('/')">广场</a>&gt;在线用户<br></div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Online',
  data () { return { rows: [], page: 1, pages: 1, total: 0, size: 10 } },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    user () { return this.$store.state.user }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      this.page = parseInt(this.$route.query.page) || 1
      api.get('/online', { params: { page: this.page, size: this.size } }).then(r => {
        if (r.code === 0) {
          this.rows = r.data.list || []
          this.total = r.data.total || 0
          this.page = r.data.page || 1
          this.size = r.data.size || 10
          this.pages = Math.max(1, Math.ceil(this.total / this.size))
        }
      })
    },
    go (p) { this.$router.push({ path: '/online', query: { page: p } }) },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t)
      const p = n => (n < 10 ? '0' + n : n)
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds())
    }
  }
}
</script>
