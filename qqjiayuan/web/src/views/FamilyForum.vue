<template>
  <div>
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/families')">家族</a>&gt;<a href="javascript:;" @click="$router.push('/family/'+famId)">{{ fam.name || '…' }}</a>&gt;论坛<br>
    </div>

    <!-- 与家族主页一致的导航条 -->
    <div class="module-title">
      <a href="javascript:;" @click="$router.push('/family/'+famId)">主页</a> | 论坛
      | <a href="javascript:;" @click="$router.push({ path: '/chat', query: { family_id: famId } })">聊室</a>
      | <a href="javascript:;" @click="$router.push('/channel/1')">娱乐</a>
      | <a href="javascript:;" @click="$router.push('/family/'+famId+'#members')">家人</a><br>
    </div>

    <div class="title">
      <a href="javascript:;" @click="setFilter('')">所有</a>|<a href="javascript:;" @click="setFilter('fine')">精华</a>|<a href="javascript:;" @click="setSort('new')">新贴</a><br>
    </div>

    <div class="list">
      <div class="row" v-for="(t,i) in threads" :key="t.id">
        {{ (page-1)*10 + i + 1 }}.<template v-if="t.is_top">【顶】</template><template v-if="t.is_fine">【精】</template><a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a><br>
        (<a href="javascript:;" @click="$router.push('/user/'+(t.user?t.user.id:''))"><font :color="t.user?t.user.color:''">{{ t.user?t.user.nickname:'路人' }}</font></a>:<a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.reply_count }}</a>回/{{ t.view_count }}阅)<br>
      </div>
    </div>
    <div v-if="!threads.length" class="empty">家族论坛还空空的，来抢个沙发吧！</div>

    <div class="item">
      <a v-if="page > 1" href="javascript:;" @click="go(page-1)">上页</a><template v-if="page > 1">.</template><a v-if="page < pages" href="javascript:;" @click="go(page+1)">下页</a><template v-if="page < pages">.</template>
      <form style="display:inline" @submit.prevent="go(pageInput)">
        第<input type="text" v-model.number="pageInput" size="2" maxlength="4">页
        <input type="submit" value="前往">
      </form>
      (第<b>{{ page }}</b>/{{ pages }}页/共{{ total }}帖)
    </div>

    <div class="item">
      <a v-if="isMember && board.id" href="javascript:;" @click="$router.push('/post/'+board.id)">发帖</a><template v-if="isMember && board.id">.</template>
      <a href="javascript:;" @click="$router.push('/family/'+famId)">回家族主页</a><br>
    </div>
    <div class="login-tips" v-if="!isMember">
      <img :src="$pic('0.gif')" alt="广播">只有家族成员才能在家族论坛发帖回帖，浏览不需要加入哦！<br>
    </div>

    <div class="bar">
      <a href="javascript:;" @click="$router.push('/families')">家族</a>&gt;<a href="javascript:;" @click="$router.push('/family/'+famId)">{{ fam.name || '…' }}</a>&gt;论坛<br>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'FamilyForum',
  data () {
    return { fam: {}, board: {}, threads: [], total: 0, page: 1, pages: 1, pageInput: 1 }
  },
  computed: {
    famId () { return this.$route.params.id },
    isLogin () { return this.$store.getters.isLogin },
    isMember () { return !!this.fam.my_role }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      const q = this.$route.query
      api.get('/families/' + this.famId + '/forum', { params: { page: q.page || 1, filter: q.filter || '', sort: q.sort || '' } }).then(r => {
        if (r.code === 0) {
          this.board = r.data.board || {}
          this.threads = r.data.list
          this.total = r.data.total
          this.page = r.data.page
          this.pageInput = r.data.page
          this.pages = Math.max(1, Math.ceil(r.data.total / r.data.size))
        }
      })
      api.get('/families/' + this.famId).then(r => { if (r.code === 0) this.fam = r.data })
    },
    setFilter (f) {
      this.$router.push({ query: { filter: f, sort: this.$route.query.sort || '' } })
    },
    setSort (s) {
      this.$router.push({ query: { filter: this.$route.query.filter || '', sort: s } })
    },
    go (p) {
      p = Math.min(Math.max(1, p || 1), this.pages)
      this.$router.push({ query: { ...this.$route.query, page: p } })
    }
  }
}
</script>
