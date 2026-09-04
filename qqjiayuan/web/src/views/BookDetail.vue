<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/book')">书城</a>&gt;{{ b.title || '…' }}</div>
    <div v-if="!loading && !b.id" class="module-content"><span class="empty">书未找到</span></div>

    <template v-if="b.id">
      <div class="module-title">《{{ b.title }}》</div>
      <div class="module-content plist">
        <div class="row00">作者：<a href="javascript:;" @click="$router.push('/tip?title='+encodeURIComponent('书城·作者'))">{{ b.author }}</a></div>
        <div class="row00">分类：{{ b.category }}　状态：{{ b.status }}　<b style="color:#e05a00">{{ b.rating }}</b></div>
        <div class="row00">简介：{{ b.intro }}</div>
      </div>
      <div class="module-content" style="text-align:center">
        <a href="javascript:;" @click="$router.push('/tip?title='+encodeURIComponent('开始阅读'))">开始阅读</a>　<a href="javascript:;" @click="$router.push('/tip?title='+encodeURIComponent('加入书架'))">加入书架</a>
      </div>

      <div class="module-title">同分类推荐</div>
      <div class="module-content" v-for="x in related" :key="'x'+x.id">《<a href="javascript:;" @click="$router.push('/book/'+x.id)">{{ x.title }}</a>》（<span class="txt-fade">{{ x.author }}</span>）<br></div>
    </template>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'BookDetail',
  data () { return { b: {}, related: [], loading: true } },
  mounted () { this.load() },
  watch: { '$route': 'load' },
  methods: {
    load () {
      this.loading = true
      const id = this.$route.params.id
      api.get('/books/' + id).then(r => {
        this.loading = false
        if (r.code === 0) {
          this.b = r.data
          api.get('/books/list?category=' + encodeURIComponent(this.b.category)).then(rr => {
            if (rr.code === 0) this.related = rr.data.filter(x => x.id !== this.b.id).slice(0, 5)
          })
        } else this.b = {}
      })
    }
  }
}
</script>
