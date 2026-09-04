<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/')">首页</a>&gt;<a href="javascript:;" @click="$router.push('/book')">书城</a><template v-if="view==='list'">&gt;<a href="javascript:;" @click="showIndex">分类</a>&gt;{{ cate || '全部' }}</template></div>

    <!-- 列表模式 -->
    <template v-if="view==='list'">
      <div class="module-title">《分类书库》{{ cate || '全部' }}</div>
      <div class="module-content" v-if="categoryBooks.length">
        <template v-for="(b,i) in categoryBooks"><span :key="'b'+b.id">{{ i+1 }}.《<a href="javascript:;" @click="$router.push('/book/'+b.id)">{{ b.title }}</a>》(作者:<a href="javascript:;" @click="$router.push('/tip?title='+encodeURIComponent('书城·作者'))">{{ b.author }}</a>)</span><br :key="'br'+b.id"></template>
      </div>
      <div class="module-content" v-else><span class="empty">暂无书籍</span></div>
      <div class="module-content"><a href="javascript:;" @click="showIndex">返回书城&gt;&gt;</a></div>
    </template>

    <!-- 首页模式 -->
    <template v-else>
      <div class="module-title">书城导航</div>
      <div class="module-content">
        <a href="javascript:;" @click="$router.push('/tip?title='+encodeURIComponent('我的书屋'))">书屋</a>　<a href="javascript:;" @click="showIndex">分类</a>　<a href="javascript:;" @click="$router.push('/channel/1')">论坛</a><br>
        <a href="javascript:;" @click="$router.push('/tip?title='+encodeURIComponent('书城排行'))">排行</a>　<a href="javascript:;" @click="$router.push('/tip?title='+encodeURIComponent('新书'))">新书</a>　<a href="javascript:;" @click="$router.push('/tip?title='+encodeURIComponent('连载'))">连载</a><br>
        <form @submit.prevent="doSearch">搜书：<input type="text" v-model.trim="wd" size="12" maxlength="20"><input type="submit" value="搜书"></form>
      </div>

      <div class="module-title">强力推荐</div>
      <div class="module-content" v-for="b in recommend" :key="'r'+b.id">《<a href="javascript:;" @click="$router.push('/book/'+b.id)">{{ b.title }}</a>》（<span class="txt-fade">{{ b.author }} · {{ b.category }} {{ b.status }}</span>）<br></div>

      <div class="module-title">最新连载</div>
      <div class="module-content" v-for="b in latest" :key="'l'+b.id">《<a href="javascript:;" @click="$router.push('/book/'+b.id)">{{ b.title }}</a>》（<span class="txt-fade">{{ b.author }}</span>）<br></div>

      <div class="module-title">新书上架</div>
      <div class="module-content" v-for="b in newest" :key="'n'+b.id" v-if="newest.length">《<a href="javascript:;" @click="$router.push('/book/'+b.id)">{{ b.title }}</a>》（<span class="txt-fade">{{ b.author }}</span>）<br></div>
      <div class="module-content" v-if="!newest.length"><span class="empty">暂无新书</span></div>

      <div class="module-title"><a href="javascript:;" @click="showIndex">分类书库</a></div>
      <div class="module-content">
        <template v-for="ct in categories"><span :key="ct"><a href="javascript:;" @click="openCat(ct)">[{{ ct }}]</a></span><br :key="'c'+ct"></template>
      </div>
    </template>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Book',
  data () { return { view: 'index', cate: '', wd: '', recommend: [], latest: [], newest: [], categories: [], categoryBooks: [] } },
  mounted () { this.load() },
  watch: { '$route': 'load' },
  methods: {
    load () {
      const cate = this.$route.query.cate || ''
      if (cate) { this.view = 'list'; this.cate = cate; this.loadList(cate); return }
      this.view = 'index'
      api.get('/books').then(r => {
        if (r.code === 0) {
          this.recommend = r.data.recommend || []
          this.latest = r.data.latest || []
          this.newest = r.data.newest || []
          this.categories = r.data.categories || []
        }
      })
    },
    loadList (cate) {
      api.get('/books/list?category=' + encodeURIComponent(cate)).then(r => { if (r.code === 0) this.categoryBooks = r.data })
    },
    openCat (ct) { this.$router.push('/book?cate=' + encodeURIComponent(ct)) },
    showIndex () { this.$router.push('/book') },
    doSearch () {
      if (this.wd) {
        this.$router.push('/tip?title=' + encodeURIComponent('搜书·' + this.wd))
      }
    }
  }
}
</script>
