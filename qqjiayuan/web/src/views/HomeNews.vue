<template>
  <div>
    <!-- 复刻诺哈 my_news.asp / friend_news.asp -->
    <div class="module-title">{{ scope === 'friend' ? '【好友新鲜事】' : '【我的新鲜事】' }}</div>

    <template v-if="list.length">
      <div v-for="(n, i) in list" :key="n.id" class="news-line">
        {{ (page - 1) * pageSize + i + 1 }}.({{ ago(n.created_at) }})<a href="javascript:;" @click="$router.push('/user/'+n.user_id)"><font :color="n.color || '#004299'">{{ n.nickname }}</font></a><template v-if="isThreadNews(n)">{{ n.ntype === 1 ? '发表帖子：' : '回复帖子：' }}《<a href="javascript:;" @click="$router.push('/thread/'+n.ref_id)">{{ newsTitle(n) }}</a>》</template><template v-else>{{ n.content }}</template><br>
      </div>

      <!-- 翻页：下页/上页 + (第X页/共Y页/共Z记录) + 第N页前往表单 -->
      <div class="pager">
        <a v-if="page < pageCount" href="javascript:;" @click="go(page + 1)">下页</a><span v-if="page < pageCount && page > 1">.</span><a v-if="page > 1" href="javascript:;" @click="go(page - 1)">上页</a>
        <template v-if="pageCount > 1">
          <br>(第<b>{{ page }}</b>页/共{{ pageCount }}页/共{{ total }}记录)<br>
          <form class="jump-form" @submit.prevent="go(jump)">
            第<input type="text" v-model.number="jump" size="2" maxlength="10">页 <input type="submit" value="前往">
          </form>
        </template>
      </div>
    </template>
    <div v-else>{{ scope === 'friend' ? '暂无新鲜事。。' : '您没有新鲜事。' }}<br></div>

    <div class="news-foot">
      ----------<br>
      <a href="javascript:;" @click="$router.push('/home')">我的家园</a>&gt;新鲜事<br>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'HomeNews',
  props: {
    // mine=我的新鲜事(my_news.asp) friend=好友新鲜事(friend_news.asp)
    scope: { type: String, default: 'mine' }
  },
  data () {
    return { list: [], total: 0, page: 1, pageSize: 10, jump: 1 }
  },
  computed: {
    pageCount () { return Math.max(1, Math.ceil(this.total / this.pageSize)) }
  },
  watch: {
    '$route' () { this.load() }
  },
  mounted () { this.load() },
  methods: {
    load () {
      const page = parseInt(this.$route.query.page) || 1
      this.page = page < 1 ? 1 : page
      this.jump = this.page
      api.get('/home/news', { params: { scope: this.scope, page: this.page } }).then(r => {
        if (r.code === 0) {
          this.list = r.data.list || []
          this.total = r.data.total || 0
        }
      }).catch(() => {})
    },
    go (p) {
      p = parseInt(p)
      if (!p || p < 1) p = 1
      if (p > this.pageCount) p = this.pageCount
      this.$router.push({ query: { page: p } })
    },
    isThreadNews (n) { return n.ntype === 1 || n.ntype === 2 },
    newsTitle (n) {
      const m = String(n.content || '').match(/《(.+?)》/)
      return m ? m[1] : '帖子已删除'
    },
    ago (t) {
      if (!t) return ''
      const diff = Math.max(0, (Date.now() - new Date(t).getTime()) / 1000)
      if (diff < 60) return Math.floor(diff) + '秒前'
      if (diff < 3600) return Math.floor(diff / 60) + '分钟前'
      if (diff < 86400) return Math.floor(diff / 3600) + '小时前'
      return Math.floor(diff / 86400) + '天前'
    }
  }
}
</script>
