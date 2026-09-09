<template>
  <div>
    <!-- 面包屑 panav -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;<template v-if="board.parent_id"><a href="javascript:;" @click="$router.push('/channel/'+board.parent_id)">{{ parentName }}</a>&gt;</template>{{ board.name }}<br>
    </div>

    <!-- 版块公告（诺哈 notice） -->
    <div class="radio" v-if="board.notice">{{ board.notice }}<br></div>

    <!-- 版主 / 会员制 -->
    <div class="module-content" v-if="moderator || board.members_only">
      <template v-if="moderator">版主：<a href="javascript:;" @click="$router.push('/user/'+moderator.id)"><font :color="moderator.color || '#004299'">{{ moderator.nickname }}</font></a></template>
      <template v-if="board.members_only"><font color="#c00">（本版为会员制，仅成员可发帖）</font></template>
      <br>
    </div>

    <!-- 帖子筛选 tabs（诺哈 所有|精华|新贴） -->
    <div class="title">
      <template v-if="filter === 'fine'"><a href="javascript:;" @click="setQuery({})">所有</a>|精华|<a href="javascript:;" @click="setQuery({sort:'new'})">新贴</a></template>
      <template v-else-if="sort === 'new'">所有|<a href="javascript:;" @click="setQuery({filter:'fine'})">精华</a>|新贴</template>
      <template v-else>所有|<a href="javascript:;" @click="setQuery({filter:'fine'})">精华</a>|<a href="javascript:;" @click="setQuery({sort:'new'})">新贴</a></template>
      <br>
    </div>

    <!-- 头条（诺哈 ForumTopicHead：最新头条帖） -->
    <div class="module-content" v-if="head">
      [头条]<a href="javascript:;" @click="$router.push('/thread/'+head.id)">{{ head.title }}</a><br>
    </div>

    <!-- 帖子列表 -->
    <div class="list">
      <div class="row" v-for="(t, i) in threads" :key="t.id">
        {{ i+1 }}.<template v-if="t.is_head">[头条]</template><template v-if="t.is_top">【顶】</template><template v-if="t.is_fine"><img src="/static/image/fine.gif" alt="精"></template><template v-if="t.is_lock">[锁]</template><template v-if="t.is_notice">[公告]</template><template v-if="t.is_recom">[荐]</template><template v-if="t.type===1">[奖励]</template><template v-if="t.type===2">[踩楼]</template><template v-if="t.type===3">[投票]</template><a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a><br>
        (<span v-for="b in (t.user ? t.user.badges : [])" :key="'b'+b.id"><img class="bicon" :src="$pic(b.icon)" :alt="b.name" :title="b.name"></span>
        <template v-if="t.user && t.user.priv"><img class="bicon" :src="'/static/' + t.user.priv.file" :alt="t.user.priv.name" :title="t.user.priv.name"></template>
        <img class="bicon" v-else-if="t.user && t.user.level_icon" :src="$pic('v'+t.user.level_icon+'.gif')" alt="等级">
        <a href="javascript:;" @click="$router.push('/user/'+(t.user ? t.user.id : ''))"><font :color="t.user ? t.user.color : ''">{{ t.user ? t.user.nickname : '路人' }}</font></a>:
        <a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.reply_count }}</a>回/{{ t.view_count }}阅)<br>
      </div>
      <div v-if="!threads.length" class="row">暂无帖子！<br></div>
    </div>

    <!-- 分页（诺哈 ppage 下页.上页） -->
    <div class="ppage">
      <a v-if="page < pages" href="javascript:;" @click="go(page+1)">下页</a><template v-if="page < pages">.</template>
      <a v-if="page > 1" href="javascript:;" @click="go(page-1)">上页</a><template v-if="page > 1">.</template>
      <br v-if="pages > 1">
    </div>
    <div class="item">
      (第<b>{{ page }}</b>/{{ pages }}页/共{{ total }}帖)<br>
      <form v-if="pages > 1" style="display:inline" @submit.prevent="go(pageInput)">
        第<input type="text" v-model.number="pageInput" size="2" maxlength="4">页 <input type="submit" value="前往">
      </form>
    </div>

    <!-- 操作栏（诺哈 发帖.工具.版主.在线） -->
    <div class="title">
      <a v-if="isLogin && isSub" href="javascript:;" @click="$router.push('/post/'+board.id)">发帖</a><template v-if="isLogin && isSub">.</template>
      <a href="javascript:;" @click="$router.push('/nav')">工具</a>.
      <template v-if="moderator"><a href="javascript:;" @click="$router.push('/user/'+moderator.id)">版主</a>.</template>
      <template v-else-if="canManage"><a href="javascript:;" @click="$router.push('/admin')">版主</a>.</template>
      <a href="javascript:;" @click="$router.push('/search')">在线({{ boardOnline }})</a><br>
    </div>
    <div class="module-content">
      &gt;&gt;<a href="javascript:;" @click="$router.push('/chat')">聊天室</a>.<a href="javascript:;" @click="tip">历史帖子</a><br>
    </div>

    <!-- 广播（诺哈 radio） -->
    <div class="radio">家园社区欢迎你的到来！<br></div>

    <!-- 版块跳转（诺哈 bbs tunnel 快捷跳转） -->
    <div class="item">
      <form @submit.prevent="jump">
        <select v-model.number="jumpId">
          <option value="0">-- 跳转到 --</option>
          <optgroup v-for="ch in channels" :key="ch.id" :label="ch.name">
            <option v-for="s in allSubs(ch)" :key="s.id" :value="s.id">{{ s.name }}</option>
          </optgroup>
        </select>
        <input type="submit" value="跳转">
      </form>
    </div>

    <!-- 面包屑重复 -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;<template v-if="board.parent_id"><a href="javascript:;" @click="$router.push('/channel/'+board.parent_id)">{{ parentName }}</a>&gt;</template>{{ board.name }}<br>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Board',
  data () {
    return {
      board: {}, parentName: '', moderator: null, isMember: false, isSub: false,
      head: null, boardOnline: 0,
      threads: [], total: 0, page: 1, pages: 1, pageInput: 1,
      filter: '', sort: '', channels: [], jumpId: 0
    }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    canManage () { return this.$store.getters.isAdmin }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      const id = this.$route.params.id
      const q = this.$route.query
      this.page = parseInt(q.page || 1)
      this.filter = q.filter || ''
      this.sort = q.sort || ''
      this.pageInput = this.page
      api.get(`/boards/${id}/threads`, { params: { page: this.page, filter: this.filter, sort: this.sort } }).then(r => {
        if (r.code === 0) {
          this.board = r.data.board || {}
          this.isSub = !!(this.board.parent_id)
          this.moderator = r.data.moderator || null
          this.isMember = r.data.is_member || false
          this.head = r.data.head || null
          this.boardOnline = r.data.board_online || 0
          this.threads = r.data.list || []
          this.total = r.data.total || 0
          this.page = r.data.page || 1
          this.pages = Math.max(1, Math.ceil(this.total / (r.data.size || 10)))
          if (this.board.parent_id) {
            api.get('/boards/' + this.board.parent_id).then(x => {
              if (x.code === 0) this.parentName = x.data.name
            })
          }
        }
      })
      api.get('/boards').then(r => { if (r.code === 0) this.channels = r.data })
    },
    setQuery (patch) {
      const q = { page: 1 }
      if (patch.filter) q.filter = patch.filter
      if (patch.sort) q.sort = patch.sort
      this.$router.push({ path: '/board/' + this.$route.params.id, query: q })
    },
    go (p) {
      if (p < 1) p = 1
      if (p > this.pages) p = this.pages
      const q = { page: p }
      if (this.filter) q.filter = this.filter
      if (this.sort) q.sort = this.sort
      this.$router.push({ path: '/board/' + this.$route.params.id, query: q }).catch(() => {})
    },
    jump () {
      if (this.jumpId) this.$router.push('/board/' + this.jumpId)
    },
    allSubs (ch) {
      const arr = (ch.children || []).slice()
      ;(ch.categories || []).forEach(cat => (cat.boards || []).forEach(b => arr.push(b)))
      return arr
    },
    tip () {
      alert('暂无历史帖子，敬请期待')
    }
  }
}
</script>