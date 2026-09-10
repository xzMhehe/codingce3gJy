<template>
  <div>
    <!-- 面包屑（诺哈 topic_reply.asp：社区>动态） -->
    <div class="bar"><a href="javascript:;" @click="$router.push('/')">社区</a>&gt;动态<br></div>

    <div class="list" v-if="list.length">
      <div class="row" v-for="(t, i) in list" :key="t.id">
        {{ startIndex + i }}.<a href="javascript:;" @click="$router.push('/thread/' + t.id)">{{ t.title }}</a><br>
        (<a href="javascript:;" @click="$router.push('/user/' + t.user_id)"><font :color="t.color || ''">{{ t.nickname }}</font></a>:<a href="javascript:;" @click="$router.push('/replies/' + t.id)">{{ t.reply_count }}</a>回/{{ t.view_count }}阅)<br>
      </div>
    </div>
    <div class="module-content" v-else>暂无记录！<br></div>

    <!-- 分页（诺哈原版：下页.上页 / 第X/Y页/共Z条记录 / 前往） -->
    <div class="module-content" v-if="pageCount > 1">
      <a href="javascript:;" v-if="page < pageCount" @click="go(page + 1)">下页</a><template v-if="page < pageCount && page > 1">.</template><a href="javascript:;" v-if="page > 1" @click="go(page - 1)">上页</a><br>
      (第<b>{{ page }}</b>/{{ pageCount }}页/共{{ total }}条记录)<br>
      第<input type="text" v-model.number="jump" maxlength="10" size="2">页
      <button class="btn small" @click="go(jump)">前往</button><br>
    </div>

    <div class="bar"><a href="javascript:;" @click="$router.push('/')">社区</a>&gt;动态<br></div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'TopicReply',
  data () { return { list: [], total: 0, page: 1, pageCount: 1, jump: 1 } },
  computed: { startIndex () { return (this.page - 1) * 10 + 1 } },
  watch: { '$route.query.page': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      this.page = parseInt(this.$route.query.page) || 1
      this.jump = this.page
      api.get('/threads/active', { params: { page: this.page } }).then(r => {
        if (r.code === 0) {
          this.list = r.data.list || []
          this.total = r.data.total
          this.pageCount = r.data.page_count
        }
      })
    },
    go (p) {
      p = parseInt(p)
      if (!p || p < 1 || p > this.pageCount) return
      this.$router.push({ path: '/threads/active', query: { page: p } })
    }
  }
}
</script>
