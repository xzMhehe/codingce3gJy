<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">我的地盘</a>&gt;帖子</div>
    <div class="module-title">【我的帖子】<a href="javascript:;" @click="$router.push('/post')" style="margin-left:8px;color:#1a9e1a">✎ 发新帖</a></div>
    <div class="list" v-if="threads.length">
      <div class="row" v-for="(t,i) in threads" :key="t.id">
        {{ (page-1)*size + i + 1 }}.<template v-if="t.is_head">[头条]</template><template v-if="t.is_top">【顶】</template><template v-if="t.is_fine">【精】</template>
        <a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a>
        ({{ t.board ? t.board.name : '' }} · <a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.reply_count }}</a>回/{{ t.view_count }}阅)<br>
      </div>
    </div>
    <div class="empty" v-else>你还没有发表帖子。<a href="javascript:;" @click="$router.push('/post')">去发一个吧</a></div>

    <!-- 分页 -->
    <div class="item">
      <a v-if="page > 1" href="javascript:;" @click="go(page-1)">上页</a><template v-if="page > 1">.</template>
      <a v-if="page < pages" href="javascript:;" @click="go(page+1)">下页</a><template v-if="page < pages">.</template>
      <form style="display:inline" @submit.prevent="go(pageInput)">
        第<input type="text" v-model.number="pageInput" size="2" maxlength="4">页
        <input type="submit" value="前往">
      </form>
      (第<b>{{ page }}</b>/{{ pages }}页/共{{ total }}记录)<br>
    </div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">我的地盘</a>&gt;帖子</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'MyThreads',
  data () {
    return { threads: [], total: 0, page: 1, pages: 1, pageInput: 1, size: 10 }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      this.page = parseInt(this.$route.query.page || 1)
      api.get('/my-threads', { params: { page: this.page } }).then(r => {
        if (r.code === 0) {
          this.threads = r.data.list || []
          this.total = r.data.total || 0
          this.page = r.data.page || 1
          this.pageInput = this.page
          this.pages = Math.max(1, Math.ceil(this.total / this.size))
        }
      })
    },
    go (p) {
      if (p < 1) p = 1
      if (p > this.pages) p = this.pages
      this.$router.push('/my-threads?page=' + p)
    }
  }
}
</script>