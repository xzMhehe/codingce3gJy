<template>
  <div>
    <!-- 面包屑（诺哈 online.asp：社区>同城>城市>在线老乡） -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;<a href="javascript:;" @click="$router.push('/tongcheng')">同城</a>&gt;<a href="javascript:;" @click="$router.push('/tongcheng/city/'+city.id)">{{ city.name }}</a>&gt;在线老乡<br>
    </div>

    <div class="module-title">【在线老乡】</div>
    <div class="list">
      <div class="row" v-for="(u, i) in users" :key="'u'+u.id">
        {{ (page-1)*size+i+1 }}.<a href="javascript:;" @click="$router.push('/user/'+u.id)"><font :color="u.color">{{ u.nickname }}</font></a>({{ u.username }}) Lv.{{ u.level }} - {{ fmt(u.last_active_at) }}<br>
      </div>
      <div v-if="!users.length" class="row">没有老乡在此，快来占个位！<br></div>
    </div>

    <!-- 分页 -->
    <div class="ppage">
      <a v-if="page < pages" href="javascript:;" @click="go(page+1)">下页</a><template v-if="page < pages">.</template>
      <a v-if="page > 1" href="javascript:;" @click="go(page-1)">上页</a><template v-if="page > 1">.</template>
      <br v-if="pages > 1">
    </div>
    <div class="item" v-if="pages > 1">
      (第<b>{{ page }}</b>/{{ pages }}页/共{{ total }}条记录)<br>
    </div>

    <div class="module-content">
      <a href="javascript:;" @click="$router.push('/tongcheng/city/'+city.id)">返回{{ city.name }}主页</a><br>
    </div>

    <!-- 面包屑重复 -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;<a href="javascript:;" @click="$router.push('/tongcheng')">同城</a>&gt;<a href="javascript:;" @click="$router.push('/tongcheng/city/'+city.id)">{{ city.name }}</a>&gt;在线老乡<br>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'CityOnline',
  data () {
    return { city: {}, users: [], total: 0, page: 1, pages: 1, size: 10 }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      const id = this.$route.params.id
      const q = this.$route.query
      this.page = parseInt(q.page || 1)
      api.get('/tongcheng/city/' + id + '/online', { params: { page: this.page } }).then(r => {
        if (r.code !== 0) {
          alert(r.msg || '该城市不存在')
          this.$router.push('/tongcheng')
          return
        }
        this.city = r.data.city || {}
        this.users = r.data.list || []
        this.total = r.data.total
        this.page = r.data.page
        this.size = r.data.size
        this.pages = Math.max(1, Math.ceil(this.total / this.size))
      })
    },
    go (p) {
      if (p < 1 || p > this.pages) return
      this.$router.push({ path: '/tongcheng/city/' + this.$route.params.id + '/online', query: { page: p } })
    },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t)
      const p = n => (n < 10 ? '0' + n : n)
      return `${p(d.getHours())}:${p(d.getMinutes())}`
    }
  }
}
</script>
