<template>
  <div>
    <!-- 资料卡：夜凌云 1级 [等级][贵族][身份] 三图标 -->
    <div class="module-content unline">
      <b><a href="javascript:;" @click="$router.push('/user/'+u.id)"><font :color="u.color || '#004299'">{{ u.nickname || '我' }}</font></a> {{ u.level || 1 }}级</b>
      <img :src="$pic('home_' + (u.level || 1) + '_' + (u.level || 1) + '.gif')" alt="等级" class="bicon uic" @error="iconErr($event, u.level)">
      <img v-if="u.noble > 0" :src="$pic('noble_' + u.noble + '_1.gif')" alt="贵族" class="bicon uic" @error="hideErr($event)">
      <img src="/static/picture/chuping.jpg" alt="身份" class="bicon uic">
    </div>

    <!-- 心情 -->
    <div class="write-mood">{{ mood ? mood.content : (u.signature || '我的心情很好') }} <a href="javascript:;" @click="$router.push('/mood')">&gt;&gt;</a><br></div>

    <!-- 快捷入口 -->
    <div>
      <a href="javascript:;" @click="tip('宅子')">宅子</a> . <a href="javascript:;" @click="tip('友友券')">友友券</a> . <a href="javascript:;" @click="$router.push('/home')">回家</a> . <a href="javascript:;" @click="tip('花仙子')">花仙子</a><br>
    </div>

    <!-- tab：我的 活动 帖 书（参考站 module-title 样式） -->
    <div class="module-title">
      <a href="javascript:;" :style="{ fontWeight: cur === 'mine' ? 'bold' : 'normal' }" @click="cur='mine'">我的</a>
      &nbsp;<a href="javascript:;" :style="{ color: cur === 'active' ? '#FF0000' : '#0051A4', fontWeight: cur === 'active' ? 'bold' : 'normal' }" @click="cur='active'">活动</a>
      &nbsp;<a href="javascript:;" @click="cur='post'">帖</a>
      &nbsp;<a href="javascript:;" @click="cur='book'">书</a>
    </div>

    <!-- ===== 我的 ===== -->
    <template v-if="cur === 'mine'">
      <div v-if="myGames.length">正在玩<a href="javascript:;" @click="$router.push('/games')">{{ myGames[0].name }}</a><br></div>
      <div class="list">
        <div v-for="g in myGames" :key="g.id" class="row">
          <a href="javascript:;" @click="$router.push('/games')">{{ g.name }}</a>(1级,{{ g.desc || '可玩' }})<br>
        </div>
      </div>
      <a href="javascript:;" @click="$router.push('/games')">更多我的游戏&gt;&gt;</a><br>
      <a href="javascript:;" @click="$router.push('/games')">添加游戏</a> . <a href="javascript:;" @click="$router.push('/games')">管理游戏</a>

      <div class="module-title"><a href="javascript:;" @click="$router.push('/channel/1')">新鲜事({{ feed.length }})</a> <a href="javascript:;" @click="tip('新鲜事设置')">设置</a></div>
      <div class="list" v-if="feed.length">
        <div v-for="d in feed" :key="'f'+d.id" class="row">
          ({{ ago(d.created_at) }})<a href="javascript:;" @click="$router.push('/user/'+d.user_id)"><font :color="d.color || '#004299'">{{ d.nickname }}</font></a>{{ d.action }}《<a href="javascript:;" @click="$router.push('/thread/'+d.thread_id)">{{ d.title }}</a>》<br>
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">还没有新鲜事</span></div>
      <a href="javascript:;" @click="$router.push('/channel/1')">查看更多&gt;&gt;</a><br>

      <div class="module-title"><a href="javascript:;" @click="$router.push('/space/'+u.id)">留言板</a></div>
      <ul class="dtuser" v-if="msgs.length">
        <li v-for="m in msgs" :key="'m'+m.id"><a href="javascript:;" @click="$router.push('/user/'+m.from_user_id)"><font :color="m.from_color || '#004299'">{{ m.from_nickname || '友友' }}</font></a> <span class="txt-fade">{{ fmt(m.created_at) }}</span><br>{{ m.content }}</li>
      </ul>
      <div class="module-content" v-else><span class="empty">还没有留言</span></div>

      <div class="module-title">好友来访|<a href="javascript:;" @click="$router.push('/friends')">其他访客</a></div>
      <div class="list" v-if="friends.length">
        <div v-for="f in friends" :key="'fr'+f.id" class="row">
          <a href="javascript:;" @click="$router.push('/user/'+f.id)"><font :color="f.color || '#004299'">{{ f.nickname }}</font></a>
          <font color="SlateGray">Lv.{{ f.level }}</font><br>
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">还没有好友来访</span></div>

      <form @submit.prevent="visit">
        <input type="text" v-model.number="visitId" maxlength="10" size="10"><input type="submit" value="串门">
      </form>

      <div class="module-title">【功能导航】</div>
      <a href="javascript:;" @click="$router.push('/messages')">家信</a>.<a href="javascript:;" @click="$router.push('/board/4')">婚恋</a>.<a href="javascript:;" @click="$router.push('/channel/1')">论坛</a>.<a href="javascript:;" @click="$router.push('/families')">家族</a>.<a href="javascript:;" @click="cur='active'">活动</a>.<a href="javascript:;" @click="$router.push('/chat')">聊天室</a>.      <a href="javascript:;" @click="$router.push('/home-level')">家园等级</a>.<a href="javascript:;" @click="tip('更多')">&gt;&gt;</a><br>
      <a href="javascript:;" @click="tip('任务')">任务</a>.<a href="javascript:;" @click="tip('反馈')">反馈</a>.<a href="javascript:;" @click="tip('秘密')">秘密</a>.<a href="javascript:;" @click="tip('黑板墙')">黑板墙</a>.<a href="javascript:;" @click="$router.push('/profile')">特权</a>.<a href="javascript:;" @click="$router.push('/find')">靓号</a><br>
    </template>

    <!-- ===== 活动 ===== -->
    <template v-if="cur === 'active'">
      <div class="module-title"><img src="/static/image/active.gif" alt="活动" class="bicon">【最新活动】</div>
      <div v-for="t in fineThreads" :key="'fa'+t.id"><a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a>({{ t.view_count || 0 }}阅)<br></div>
      <a href="javascript:;" @click="$router.push('/channel/1')">更多活动&gt;&gt;</a>

      <div class="module-title"><img src="/static/image/active.gif" alt="活动" class="bicon">【长期活动】</div>
      <div v-for="t in commonThreads" :key="'lc'+t.id"><a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a>({{ t.view_count || 0 }}阅)<br></div>
      <a href="javascript:;" @click="$router.push('/channel/1')">更多活动&gt;&gt;</a>

      <div class="module-title"><img src="/static/picture/notice.bmp" alt="公告" class="bicon">【家园公告】</div>
      <div v-for="a in announcements" :key="'an'+a.id"><a href="javascript:;" @click="$router.push('/notices')">{{ a.title }}</a><br></div>
      <a href="javascript:;" @click="$router.push('/notices')">更多公告&gt;&gt;</a><br>
    </template>

    <!-- ===== 帖 ===== -->
    <template v-if="cur === 'post'">
      <div class="module-title">我的帖子|<a href="javascript:;" @click="$router.push('/profile')">回帖</a>|<a href="javascript:;" @click="$router.push('/profile')">收藏</a>|<a href="javascript:;" @click="tip('草稿')">草稿</a></div>

      <ul class="dtuser" v-if="threads.length">
        <li v-for="t in threads" :key="'t'+t.id"><a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a> <em>（{{ t.board ? t.board.name : '' }} · {{ t.view_count }}阅/{{ t.reply_count }}回）</em></li>
      </ul>
      <div class="text" v-else>你还没有发过帖子呢，快去<a href="javascript:;" @click="$router.push('/channel/1')">论坛</a>发一个吧！<br></div>

      <div class="module-title">我的回帖</div>
      <ul class="dtuser" v-if="myReplies.length">
        <li v-for="r in myReplies" :key="'r'+r.id"><a href="javascript:;" @click="$router.push('/thread/'+r.thread_id)">回复《{{ (r.thread && r.thread.title) || '帖子' }}》</a><br><em>{{ brief(r.content) }}（{{ ago(r.created_at) }}）</em></li>
      </ul>
      <div class="module-content" v-else><span class="empty">还没有回帖</span></div>

      <div class="module-title">我收藏的帖子</div>
      <ul class="dtuser" v-if="favThreads.length">
        <li v-for="t in favThreads" :key="'ft'+t.id"><a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a> <em>（{{ (t.board ? t.board.name : '') }} · {{ t.view_count || 0 }}阅）</em></li>
      </ul>
      <div class="module-content" v-else><span class="empty">还没有收藏帖子</span></div>

      <div class="module-title">常用地址</div>
      <a href="javascript:;" @click="$router.push('/channel/1')">论坛</a>.<a href="javascript:;" @click="$router.push('/search')">搜帖</a>.<a href="javascript:;" @click="cur='post'">收藏夹</a><br>
      <a href="javascript:;" @click="$router.push('/channel/1')">今日热帖</a>.<a href="javascript:;" @click="$router.push('/channel/1')">公共论坛</a><br>
      <a href="javascript:;" @click="tip('社区服务')">社区服务</a>.<a href="javascript:;" @click="$router.push('/channel/1')">产品论坛</a><br>
    </template>

    <!-- ===== 书（我的书屋） ===== -->
    <template v-if="cur === 'book'">
      <div class="bar"><a href="javascript:;" @click="$router.push('/book')">书城</a>&gt;我的书屋</div>
      <div class="module-title"><a href="javascript:;" @click="tip('我的书架')">我的书架</a></div>
      <span class="txt-fade">暂无书架。</span><br>
      <div class="module-title"><a href="javascript:;" @click="tip('我的收藏')">我的收藏</a></div>
      <span class="txt-fade">你还没有收藏书籍。</span><br>
      <div class="module-title"><a href="javascript:;" @click="tip('我的购买')">我的购买</a></div>
      <span class="txt-fade">你还没有购买的书籍。</span><br>
      <div class="module-title"><a href="javascript:;" @click="tip('我的书评')">我的书评</a></div>
      <span class="txt-fade">你还没有发表书评。</span><br>
      <div class="module-title"><a href="javascript:;" @click="$router.push('/book')">作者专区</a></div>
      <a href="javascript:;" @click="$router.push('/book')">书籍分类</a>.<a href="javascript:;" @click="$router.push('/search')">搜索书籍</a><br>
    </template>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'MyHome',
  data () {
    return {
      cur: 'mine', u: {}, threads: [], friends: [], visitId: '', mood: null,
      games: [], myGames: [], feed: [], msgs: [],
      fineThreads: [], commonThreads: [], announcements: [],
      myReplies: [], favThreads: []
    }
  },
  mounted () { this.load() },
  methods: {
    load () {
      const me = this.$store.state.user
      if (!me) return
      const id = me.id
      api.get('/users/' + id).then(r => { if (r.code === 0) this.u = r.data; this.threads = (r.data.threads || []).slice(0, 8) })
      api.get('/moods/latest').then(r => { if (r.code === 0) this.mood = r.data })
      api.get('/friends').then(r => { if (r.code === 0) this.friends = r.data.friends.slice(0, 5) })
      api.get('/games').then(r => { if (r.code === 0) this.games = r.data })
      api.get('/my-games').then(r => { if (r.code === 0) this.myGames = r.data })
      api.get('/plaza').then(r => {
        if (r.code === 0) {
          this.feed = (r.data.dynamics || []).slice(0, 8)
          this.fineThreads = (r.data.fine_threads || [])
          this.commonThreads = (r.data.quick_threads || [])
          this.announcements = (r.data.announcements || [])
          this.mood = this.mood || null
        }
      })
      api.get('/space/' + id + '/messages').then(r => { if (r.code === 0) this.msgs = (r.data.list || r.data || []).slice(0, 3) }).catch(() => {})
      api.get('/my-replies').then(r => { if (r.code === 0) this.myReplies = r.data }).catch(() => {})
      api.get('/favorite-threads').then(r => { if (r.code === 0) this.favThreads = r.data }).catch(() => {})
    },
    visit () { if (this.visitId) this.$router.push('/user/' + this.visitId) },
    brief (s) { s = s || ''; return s.length > 30 ? s.slice(0, 30) + '…' : s },
    iconErr (e, level) { e.target.src = this.$pic('v' + (level || 1) + '.gif') },
    hideErr (e) { e.target.style.display = 'none' },
    tip (title) { this.$router.push('/tip?title=' + encodeURIComponent('家园·' + title)) },
    ago (t) {
      if (!t) return ''
      const diff = Math.max(0, (Date.now() - new Date(t).getTime()) / 1000)
      if (diff < 60) return Math.floor(diff) + '秒前'
      if (diff < 3600) return Math.floor(diff / 60) + '分钟前'
      if (diff < 86400) return Math.floor(diff / 3600) + '小时前'
      return Math.floor(diff / 86400) + '天前'
    },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t)
      return d.getFullYear() + '/' + (d.getMonth() + 1) + '/' + d.getDate() + ' ' + d.getHours() + ':' + (d.getMinutes() < 10 ? '0' : '') + d.getMinutes()
    }
  }
}
</script>

<style scoped>
.unline { display: flex; align-items: center; flex-wrap: wrap; gap: 3px; }
.unline .bicon.uic { height: 16px; width: 16px; object-fit: contain; }
</style>
