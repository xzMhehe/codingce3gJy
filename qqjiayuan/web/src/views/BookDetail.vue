<template>
  <div>
    <!-- 章节阅读视图（复刻诺哈 wap/book/chapter.asp） -->
    <template v-if="view === 'chapter' && ch.chapter">
      <div class="bar"><a href="javascript:;" @click="view = 'chapters'">目录</a>&gt;<a href="javascript:;" @click="$router.push('/book/' + $route.params.id)">{{ ch.book.title }}</a>&gt;{{ ch.chapter.title }}</div>
      <div class="module-title">{{ ch.chapter.title }}<span v-if="ch.chapter.vip === 1" style="color:#e05a00">[VIP]</span><br></div>
      <div class="module-content chap-body">
        <span v-for="(p, i) in chParas" :key="'p'+i">{{ p }}<br><br></span>
      </div>
      <div class="module-content" style="text-align:center">
        <a href="javascript:;" v-if="ch.prev_id" @click="goChapter(ch.prev_id)">上一章</a><span v-if="ch.prev_id && ch.next_id">　</span><a href="javascript:;" v-if="ch.next_id" @click="goChapter(ch.next_id)">下一章</a><span v-if="!ch.next_id">　</span><a href="javascript:;" @click="view = 'chapters'">返回目录</a><br>
      </div>
      <div class="module-content" style="text-align:center"><span class="txt-fade">{{ ch.book.title }} - {{ ch.chapter.title }}</span><br></div>
    </template>

    <!-- 章节目录（复刻诺哈 chapter_list.asp） -->
    <template v-else-if="view === 'chapters'">
      <div class="bar"><a href="javascript:;" @click="$router.push('/book/' + $route.params.id)">书页</a>&gt;章节目录</div>
      <div class="module-title">《{{ b.title }}》章节目录<br></div>
      <div class="module-content">
        <span v-for="(c, i) in chapters" :key="c.id">{{ i + 1 }}.<a href="javascript:;" @click="goChapter(c.id)">{{ c.title }}</a><span v-if="c.vip === 1" style="color:#e05a00">[VIP]</span><br></span>
        <span v-if="!chapters.length" class="empty">暂无章节，敬请期待。</span>
      </div>
    </template>

    <!-- 书页（复刻诺哈 book.asp） -->
    <template v-else>
      <div class="bar"><a href="javascript:;" @click="$router.push('/book')">书城</a>&gt;{{ b.title || '…' }}</div>
      <div v-if="!loading && !b.id" class="module-content"><span class="empty">书未找到</span></div>

      <template v-if="b.id">
        <div class="module-title">《{{ b.title }}》</div>
        <div class="module-content plist">
          <div class="row00">作者：<a href="javascript:;" @click="$router.push({ path: '/book', query: { wd: b.author } })">{{ b.author }}</a></div>
          <div class="row00">分类：{{ b.category }}　状态：{{ b.status }}　<b style="color:#e05a00">{{ b.rating }}</b></div>
          <div class="row00">点击：{{ b.views }}　评论：{{ comments.length }}</div>
          <div class="row00">简介：{{ b.intro }}</div>
        </div>
        <div class="module-content" style="text-align:center">
          <a href="javascript:;" @click="view = 'chapters'">开始阅读</a>　<a href="javascript:;" @click="addShelf">加入书架</a>　<a href="javascript:;" @click="$router.push('/shelf')">我的书架</a><br>
        </div>

        <!-- 书评（复刻 comment_list.asp / comment_add.asp） -->
        <div class="module-title">书评<br></div>
        <div class="module-content">
          <div v-for="m in comments" :key="m.id" style="padding:3px 0">
            <b>{{ m.nick }}</b>：<span style="color:#e05a00">{{ stars(m.score) }}</span><br>
            {{ m.content }} <span class="txt-fade">({{ m.time_txt }})</span><br>
          </div>
          <span v-if="!comments.length" class="empty">暂无书评，快来抢沙发！</span>
        </div>
        <div class="module-content" v-if="user.id">
          我的评分：
          <select v-model.number="cForm.score" style="width:110px">
            <option :value="5">★★★★★</option><option :value="4">★★★★</option><option :value="3">★★★</option>
            <option :value="2">★★</option><option :value="1">★</option>
          </select><br>
          <textarea v-model.trim="cForm.content" rows="3" style="width:96%" maxlength="300" placeholder="写下你的书评（300字内）"></textarea><br>
          <input type="submit" value="发表书评" @click="submitComment">
        </div>
        <div class="module-content" v-else><a href="javascript:;" @click="$router.push('/login')">登录</a>后可发表书评</div>

        <div class="module-title">同分类推荐</div>
        <div class="module-content" v-for="x in related" :key="'x'+x.id">《<a href="javascript:;" @click="$router.push('/book/'+x.id)">{{ x.title }}</a>》（<span class="txt-fade">{{ x.author }}</span>）<br></div>
      </template>
    </template>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'BookDetail',
  data () {
    return {
      b: {}, related: [], loading: true,
      view: 'book', chapters: [], ch: {}, comments: [],
      cForm: { score: 5, content: '' }
    }
  },
  computed: {
    user () { return this.$store.state.user || {} },
    chParas () {
      return (this.ch.chapter && this.ch.chapter.content ? this.ch.chapter.content : '').split(/\n+/).filter(Boolean)
    }
  },
  mounted () { this.load() },
  watch: { '$route': 'load' },
  methods: {
    stars (n) { return '★★★★★☆☆☆☆☆'.slice(5 - Math.max(1, Math.min(5, n || 5)), 10 - Math.max(1, Math.min(5, n || 5))) },
    load () {
      this.loading = true
      this.view = 'book'
      this.ch = {}
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
      api.get('/books/' + id + '/comments').then(r => { if (r.code === 0) this.comments = r.data })
      api.get('/books/' + id + '/chapters').then(r => { if (r.code === 0) this.chapters = r.data })
    },
    goChapter (cid) {
      api.get('/chapters/' + cid).then(r => {
        if (r.code === 0) { this.ch = r.data; this.view = 'chapter' }
      })
    },
    addShelf () {
      api.post('/books/' + this.$route.params.id + '/shelf').then(r => {
        if (r.code === 0) alert(r.data.msg || '已加入书架')
        else alert(r.msg)
      })
    },
    submitComment () {
      if (!this.cForm.content) { alert('请填写评论内容'); return }
      api.post('/books/' + this.$route.params.id + '/comments', this.cForm).then(r => {
        if (r.code === 0) {
          this.cForm.content = ''
          api.get('/books/' + this.$route.params.id + '/comments').then(rr => { if (rr.code === 0) this.comments = rr.data })
        } else alert(r.msg)
      })
    }
  }
}
</script>
