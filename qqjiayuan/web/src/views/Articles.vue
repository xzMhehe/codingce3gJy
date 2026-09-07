<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/')">社区</a>&gt;文章</div>
    <div class="name">文章专栏<br></div>
    <div class="module-title">分类:
      <a href="javascript:;" :style="{ fontWeight: !catId ? 'bold' : 'normal' }" @click="catId=0;load(1)">全部</a>
      <template v-for="c in cats">
        <span :key="c.id"> | <a href="javascript:;" :style="{ fontWeight: catId === c.id ? 'bold' : 'normal' }" @click="catId=c.id;load(1)">{{ c.name }}</a></span>
      </template>
    </div>

    <div class="list">
      <div v-for="a in list" :key="a.id" class="row">
        <a href="javascript:;" @click="$router.push('/articles/'+a.id)">{{ a.title }}</a>
        <a href="javascript:;" v-if="isLogin && a.user_id === user.id" @click="del(a.id)">[删]</a><br>
        <span class="txt-fade">[{{ a.cat_name || '未分类' }}] {{ a.author }} · {{ a.click }}阅/{{ a.comment }}评 · {{ fmt(a.created_at) }}</span><br>
      </div>
    </div>
    <span v-if="!list.length" class="empty">还没有文章</span>

    <div v-if="totalPages > 1" class="pager">
      (第<b>{{ page }}</b>/{{ totalPages }}页/共{{ total }}篇)
      <a href="javascript:;" v-if="page > 1" @click="load(page-1)">&lt;&lt;上一页</a>
      <a href="javascript:;" v-if="page < totalPages" @click="load(page+1)">下一页&gt;&gt;</a>
    </div>

    <div class="module-title"><a href="javascript:;" @click="$router.push('/articles/edit')">发表文章</a>.<a href="javascript:;" @click="$router.push('/home')">返回家园</a></div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Articles',
  data () { return { list: [], cats: [], catId: 0, page: 1, total: 0, totalPages: 1 } },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    user () { return this.$store.state.user || {} }
  },
  mounted () { this.loadCats(); this.load(1) },
  methods: {
    loadCats () { api.get('/articles/categories').then(r => { if (r.code === 0) this.cats = r.data || [] }) },
    load (p) {
      this.page = p
      api.get('/articles', { params: { page: p, cat_id: this.catId } }).then(r => {
        if (r.code === 0) {
          this.list = r.data.list || []
          this.total = r.data.total
          this.totalPages = Math.ceil(r.data.total / r.data.size) || 1
        }
      })
    },
    del (id) { api.delete('/articles/' + id).then(() => this.load(this.page)) },
    fmt (t) { return t ? t.slice(0, 10) : '' }
  }
}
</script>