<template>
  <div>
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;<template v-if="board.parent_id"><a href="javascript:;" @click="$router.push('/channel/'+board.parent_id)">{{ parentName }}</a>&gt;</template><a href="javascript:;">{{ board.name }}</a><br>
    </div>
    <div class="title">
      <a href="javascript:;" @click="setFilter('')">所有</a>|<a href="javascript:;" @click="setFilter('fine')">精华</a>|<a href="javascript:;" @click="setSort('new')">新贴</a><br>
    </div>
    <div class="module-content" v-if="board.notice">【版块公告】<span v-html="renderLine(board.notice)"></span><br></div>
    <div class="module-content" v-if="board.members_only"><font color="#c00">本版块为会员制，仅成员可发帖/回帖。</font> <template v-if="isMember">（你是成员）</template><br></div>
    <div class="module-content" v-if="moderator">【版主】<a href="javascript:;" @click="$router.push('/user/'+moderator.id)"><font :color="moderator.color||'#004299'">{{ moderator.nickname }}</font></a><br></div>
    <div class="module-content" v-if="headThread">
      [头条]<a href="javascript:;" @click="$router.push('/thread/'+headThread.id)">{{ headThread.title }}</a><br>
    </div>
    <div class="list">
      <div class="row" v-for="(t,i) in threads" :key="t.id">
        {{ i+1 }}.<template v-if="t.is_head">[头条]</template><template v-if="t.is_top">【顶】</template><template v-if="t.is_fine">【精】</template><template v-if="t.is_lock">[锁]</template><template v-if="t.is_recom">[荐]</template><template v-if="t.is_notice">[公告]</template><template v-if="t.type===1">[奖励]</template><template v-if="t.type===2">[踩楼]</template><template v-if="t.type===3">[投票]</template><a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a><br>
        (<span v-for="b in (t.user?t.user.badges:[])" :key="b.id"><img class="bicon" :src="$pic(b.icon)" :alt="b.name"></span><img class="bicon" v-if="t.user && t.user.priv" :src="'/static/' + t.user.priv.file" :alt="t.user.priv.name" :title="t.user.priv.name"><img class="bicon" v-else-if="t.user && t.user.level_icon" :src="$pic('v'+t.user.level_icon+'.gif')" alt="等级"> <a href="javascript:;" @click="$router.push('/user/'+(t.user?t.user.id:''))"><font :color="t.user?t.user.color:''">{{ t.user?t.user.nickname:'路人' }}</font></a>:<a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.reply_count }}</a>回/{{ t.view_count }}阅)<br>
      </div>
    </div>
    <div v-if="!threads.length" class="empty">这里还空空的，来抢个沙发吧！</div>

    <div class="item">
      <a v-if="page > 1" href="javascript:;" @click="go(page-1)">上页</a><template v-if="page > 1">.</template><a v-if="page < pages" href="javascript:;" @click="go(page+1)">下页</a><template v-if="page < pages">.</template>
      <form style="display:inline" @submit.prevent="go(pageInput)">
        第<input type="text" v-model.number="pageInput" size="2" maxlength="4">页
        <input type="submit" value="前往">
      </form>
      (第<b>{{ page }}</b>/{{ pages }}页/共{{ total }}帖)
    </div>
    <div class="item">
      <a v-if="isLogin && board.parent_id" href="javascript:;" @click="$router.push('/post/'+board.id)">发帖</a><template v-if="isLogin && board.parent_id">.</template><a href="javascript:;" @click="$router.push('/games')">工具箱</a>.<a v-if="canManage" href="javascript:;" @click="$router.push('/admin')">版主</a><template v-if="canManage">.</template><a href="javascript:;" @click="$router.push('/')">在线({{ online }})</a><br>
    </div>
    <div class="login-tips">
      <img :src="$pic('0.gif')" alt="广播">家园社区欢迎你的到来！<br>
    </div>
    <div class="module-content">
      <form @submit.prevent="jump">
        <select v-model.number="jumpId">
          <optgroup v-for="ch in channels" :key="ch.id" :label="ch.name">
            <option v-for="s in allSubs(ch)" :key="s.id" :value="s.id">{{ s.name }}</option>
          </optgroup>
        </select>
        <input type="submit" value="跳转">
      </form>
    </div>
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;<template v-if="board.parent_id"><a href="javascript:;" @click="$router.push('/channel/'+board.parent_id)">{{ parentName }}</a>&gt;</template><a href="javascript:;">{{ board.name }}</a><br>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Board',
  data () {
    return { board: {}, parentName: '', moderator: null, isMember: false, threads: [], total: 0, page: 1, pages: 1, pageInput: 1, filter: '', channels: [], online: 0 }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    canManage () { return this.$store.getters.isAdmin },
    headThread () {
      return this.threads.find(t => t.is_top) || this.threads[0] || null
    }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      const id = this.$route.params.id
      const q = this.$route.query
      this.page = parseInt(q.page || 1)
      this.filter = q.filter || ''
      api.get(`/boards/${id}/threads`, { params: { page: this.page, filter: this.filter, sort: q.sort || '' } }).then(r => {
        if (r.code === 0) {
          this.board = r.data.board
          this.moderator = r.data.moderator || null
          this.isMember = r.data.is_member || false
          this.threads = r.data.list
          this.total = r.data.total
          this.page = r.data.page
          this.pageInput = r.data.page
          this.pages = Math.max(1, Math.ceil(r.data.total / r.data.size))
          if (this.board.parent_id) {
            api.get('/boards/' + this.board.parent_id).then(x => {
              if (x.code === 0) this.parentName = x.data.name
            })
          }
        }
      })
      api.get('/boards').then(r => { if (r.code === 0) this.channels = r.data })
      api.get('/plaza').then(r => { if (r.code === 0) this.online = r.data.online_count })
    },
    setFilter (f) {
      this.$router.push('/board/' + this.$route.params.id + (f ? '?filter=' + f : ''))
    },
    setSort (s) {
      this.$router.push('/board/' + this.$route.params.id + '?sort=' + s)
    },
    go (p) {
      if (p < 1) p = 1
      const q = { page: p }
      if (this.filter) q.filter = this.filter
      this.$router.push({ path: '/board/' + this.$route.params.id, query: q })
    },
    jump () {
      if (this.jumpId) this.$router.push('/board/' + this.jumpId)
    },
    allSubs (ch) {
      const arr = (ch.children || []).slice()
      ;(ch.categories || []).forEach(cat => (cat.boards || []).forEach(b => arr.push(b)))
      return arr
    },
    renderLine (t) { return (t || '').replace(/\n/g, '<br>') }
  }
}
</script>
