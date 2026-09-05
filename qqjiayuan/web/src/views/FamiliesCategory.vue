<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/families')">家族</a>&gt;{{ cat }}<br></div>

    <div class="list">
      <div class="row" v-for="(f,i) in pageRows" :key="f.id">
        {{ (page-1)*pageSize + i + 1 }}.<a href="javascript:;" @click="$router.push('/family/'+f.id)">{{ f.name }}</a><br>
      </div>
      <div class="row" v-if="!rows.length"></div>
    </div>
    <div class="module-content" v-if="!rows.length"><span class="empty">该分类下暂无家族</span></div>

    <div class="item" v-if="pages > 1">
      <a v-if="page > 1" href="javascript:;" @click="go(page-1)">上页</a><template v-if="page > 1">.</template><a v-if="page < pages" href="javascript:;" @click="go(page+1)">下页</a><template v-if="page < pages">.</template>
    </div>
    <div class="module-content">(第<b>{{ page }}</b>/{{ pages }}页/共{{ rows.length }}条记录)<br></div>

    <div class="module-content">
      <form @submit.prevent="doSearch">
        <input type="text" v-model.trim="wd" maxlength="30" size="12">
        <input type="submit" value="搜索家族">
      </form>
    </div>

    <div class="bar"><a href="javascript:;" @click="$router.push('/families')">家族</a>&gt;{{ cat }}<br></div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'FamiliesCategory',
  data () { return { rows: [], wd: '', page: 1, pageSize: 10 } },
  computed: {
    cat () { return this.$route.params.cat },
    pages () { return Math.max(1, Math.ceil(this.rows.length / this.pageSize)) },
    pageRows () {
      const start = (this.page - 1) * this.pageSize
      return this.rows.slice(start, start + this.pageSize)
    }
  },
  watch: { '$route.params.cat': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      this.page = parseInt(this.$route.query.page || 1)
      api.get('/families', { params: { category: this.cat } }).then(r => {
        if (r.code === 0) this.rows = r.data
      })
    },
    go (p) { this.page = Math.min(Math.max(1, p), this.pages) },
    doSearch () {
      if (this.wd) this.$router.push('/families/search/' + encodeURIComponent(this.wd))
    }
  }
}
</script>
