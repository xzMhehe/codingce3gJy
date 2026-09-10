<template>
  <div>
    <!-- 面包屑（诺哈 city_list.asp：社区>同城>省份） -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;<a href="javascript:;" @click="$router.push('/tongcheng')">同城</a>&gt;<a href="javascript:;" @click="reload">{{ province.name }}</a><br>
    </div>

    <!-- 简介与统计 -->
    <div class="module-content">{{ province.description }}<br></div>

    <!-- 城市列表（诺哈按编号列出，分页） -->
    <div class="list">
      <div class="row" v-for="(c, i) in cities" :key="'c'+c.id">
        {{ (page-1)*size+i+1 }}.<a href="javascript:;" @click="$router.push('/tongcheng/city/'+c.id)">{{ c.name }}</a><template v-if="c.city_code">({{ c.city_code }})</template> - {{ c.description }}<br>
      </div>
      <div v-if="!cities.length" class="row">暂无记录！<br></div>
    </div>

    <!-- 分页（诺哈 下页.上页） -->
    <div class="ppage">
      <a v-if="page < pages" href="javascript:;" @click="go(page+1)">下页</a><template v-if="page < pages">.</template>
      <a v-if="page > 1" href="javascript:;" @click="go(page-1)">上页</a><template v-if="page > 1">.</template>
      <br v-if="pages > 1">
    </div>
    <div class="item" v-if="pages > 1">
      (第<b>{{ page }}</b>页/共{{ pages }}页/共{{ total }}条记录)<br>
      <form style="display:inline" @submit.prevent="go(pageInput)">
        第<input type="text" v-model.number="pageInput" size="2" maxlength="4">页 <input type="submit" value="前往">
      </form>
    </div>

    <!-- 面包屑重复 -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;<a href="javascript:;" @click="$router.push('/tongcheng')">同城</a>&gt;<a href="javascript:;" @click="reload">{{ province.name }}</a><br>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'CityList',
  data () {
    return { province: {}, cities: [], total: 0, page: 1, pages: 1, size: 10, pageInput: 1 }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      const id = this.$route.params.id
      const q = this.$route.query
      this.page = parseInt(q.page || 1)
      this.pageInput = this.page
      api.get('/tongcheng/province/' + id, { params: { page: this.page } }).then(r => {
        if (r.code !== 0) {
          alert(r.msg || '省份不存在')
          this.$router.push('/tongcheng')
          return
        }
        this.province = r.data.province || {}
        this.cities = r.data.list || []
        this.total = r.data.total
        this.page = r.data.page
        this.size = r.data.size
        this.pages = Math.max(1, Math.ceil(this.total / this.size))
      })
    },
    go (p) {
      if (p < 1 || p > this.pages) return
      this.$router.push({ path: '/tongcheng/province/' + this.$route.params.id, query: { page: p } })
    },
    reload () { this.load() }
  }
}
</script>
