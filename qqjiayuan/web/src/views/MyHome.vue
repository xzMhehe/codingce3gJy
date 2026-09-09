<template>
  <div>
    <!-- 资料卡：夜凌云 1级 [等级][贵族][身份] 三图标 -->
    <div class="module-content unline">
      <b><a href="javascript:;" @click="$router.push('/user/'+u.id)"><font :color="u.color || '#004299'">{{ u.nickname || '我' }}</font></a> {{ u.level || 1 }}级</b>
      <img :src="homeIcon(u)" alt="等级" class="bicon uic" @error="iconErr($event)">
      <img v-if="u.noble > 0" :src="$pic('noble_' + u.noble + '_1.gif')" alt="贵族" class="bicon uic" @error="hideErr($event)">
      <img src="/static/picture/chuping.jpg" alt="身份" class="bicon uic">
    </div>

    <!-- 心情 -->
    <div class="write-mood">{{ mood ? mood : (u.signature || '我的心情很好') }} <a href="javascript:;" @click="$router.push('/mood')">&gt;&gt;</a><br></div>
    <div v-if="todayFirst" class="txt-fade">今日登录+1活跃天。</div>

    <!-- 快捷入口 -->
    <div>
      <a href="javascript:;" @click="$router.push('/space/'+u.id)">宅子</a> . <a href="javascript:;" @click="$router.push('/youquan')">友友券</a> . <a href="javascript:;" @click="$router.push('/home')">回家</a> . <a href="javascript:;" @click="$router.push('/noble')">超Q</a><br>
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

      <!-- 访客 -->
      <div class="module-title">访客(共{{ homeAgg ? homeAgg.visitor_total : 0 }}次)</div>
      <div class="list" v-if="visitors.length">
        <div v-for="(v, i) in visitors" :key="'v'+i" class="row">
          <a href="javascript:;" @click="$router.push('/user/'+v.user_id)"><font :color="v.color || '#004299'">{{ v.nickname || '游客' }}</font></a>({{ ago(v.time) }})<br>
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">还没有访客</span></div>

      <!-- 我的新鲜事 -->
      <div class="module-title">我的新鲜事|<a href="javascript:;" @click="tip('新鲜事设置')">设置</a></div>
      <div class="list" v-if="myNews.length">
        <div v-for="n in myNews" :key="'mn'+n.id" class="row">
          ({{ ago(n.created_at) }})<font :color="n.color || '#004299'">{{ n.nickname }}</font>{{ n.content }}<br>
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">还没有新鲜事</span></div>

      <!-- 好友新鲜事 -->
      <div class="module-title">好友新鲜事</div>
      <div class="list" v-if="friendNews.length">
        <div v-for="n in friendNews" :key="'fn'+n.id" class="row">
          ({{ ago(n.created_at) }})<a href="javascript:;" @click="$router.push('/user/'+n.user_id)"><font :color="n.color || '#004299'">{{ n.nickname }}</font></a>{{ n.content }}<br>
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">还没有好友动态</span></div>

      <form @submit.prevent="visit">
        <input type="text" v-model.number="visitId" maxlength="10" size="10"><input type="submit" value="串门">
      </form>

      <div class="module-title">【功能导航】</div>
      <a href="javascript:;" @click="$router.push('/messages')">家信</a>.<a href="javascript:;" @click="$router.push('/marriage')">婚恋</a>.<a href="javascript:;" @click="$router.push('/channel/1')">论坛</a>.<a href="javascript:;" @click="$router.push('/families')">家族</a>.<a href="javascript:;" @click="$router.push('/activities')">活动</a>.<a href="javascript:;" @click="$router.push('/chat')">聊天室</a>.<a href="javascript:;" @click="$router.push('/home-level')">家园等级</a><br>
      <a href="javascript:;" @click="$router.push('/favorites')">我的收藏({{ favCount }})</a>.<a href="javascript:;" @click="$router.push('/contacts')">通讯录</a>.<a href="javascript:;" @click="$router.push('/invite')">邀请</a>.<a href="javascript:;" @click="$router.push('/guestbook')">留言本</a>.<a href="javascript:;" @click="$router.push('/articles')">文章</a>.<a href="javascript:;" @click="$router.push('/market')">商店</a>.<a href="javascript:;" @click="$router.push('/space/'+u.id)">空间</a><br>
      <a href="javascript:;" @click="$router.push('/noble')">超Q</a>.<a href="javascript:;" @click="$router.push('/wallet')">钱包</a>.<a href="javascript:;" @click="$router.push('/bag')">仓库</a>.<a href="javascript:;" @click="$router.push('/profile')">特权</a>.<a href="javascript:;" @click="$router.push('/find')">靓号</a>.<a href="javascript:;" @click="tip('更多')">&gt;&gt;</a><br>
    </template>

    <!-- ===== 活动 ===== -->
    <template v-if="cur === 'active'">
      <div class="module-title"><img :src="$pic('active.gif')" alt="活动" class="bicon">【最新活动】</div>
      <div v-for="a in plazaActivities" :key="'pa'+a.id">
        <a href="javascript:;" @click="$router.push('/thread/'+a.id)">{{ a.title }}</a>({{ a.reply_count || 0 }}回/{{ a.view_count || 0 }}阅)<br>
      </div>
      <div v-if="!plazaActivities.length" class="module-content"><span class="empty">还没有活动</span></div>
      <a href="javascript:;" @click="$router.push('/activities')">活动专区&gt;&gt;</a><br>

      <div class="module-title"><img src="/static/picture/notice.bmp" alt="公告" class="bicon">【家园公告】</div>
      <div v-for="a in announcements" :key="'an'+a.id"><a href="javascript:;" @click="$router.push('/notices')">{{ a.title }}</a><br></div>
      <a href="javascript:;" @click="$router.push('/notices')">更多公告&gt;&gt;</a><br>
    </template>

    <!-- ===== 帖 ===== -->
    <template v-if="cur === 'post'">
      <!-- 发帖入口（复刻诺哈：发帖.工具箱 风格） -->
      <div class="module-content">
        <a href="javascript:;" @click="$router.push('/post')">发帖</a>.<a href="javascript:;" @click="$router.push('/games')">工具箱</a>.<a href="javascript:;" @click="$router.push('/my-threads')">我的帖子</a>.<a href="javascript:;" @click="$router.push('/search')">搜帖</a><br>
      </div>

      <div class="module-title"><a href="javascript:;" @click="$router.push('/my-threads')">我的帖子({{ threadTotal }})</a>|<a href="javascript:;" @click="$router.push('/profile')">回帖</a>|<a href="javascript:;" @click="$router.push('/favorites')">收藏</a>|<a href="javascript:;" @click="tip('草稿')">草稿</a></div>

      <ul class="dtuser" v-if="threads.length">
        <li v-for="t in threads" :key="'t'+t.id">
          <span v-if="t.is_head" class="tag">[头条]</span><span v-if="t.is_top" class="tag">【顶】</span><a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a>
          <em>（{{ t.board ? t.board.name : '' }} · {{ t.view_count }}阅/{{ t.reply_count }}回）</em>
        </li>
      </ul>
      <div class="text" v-else>你还没有发过帖子呢，点击上方「<a href="javascript:;" @click="$router.push('/post')">发帖</a>」开始吧！<br></div>
      <a v-if="threadTotal > threads.length" href="javascript:;" @click="$router.push('/my-threads')">查看全部({{ threadTotal }})&gt;&gt;</a><br>

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
      <a href="javascript:;" @click="$router.push('/channel/1')">论坛</a>.<a href="javascript:;" @click="$router.push('/search')">搜帖</a>.<a href="javascript:;" @click="$router.push('/favorites')">收藏夹</a><br>
      <a href="javascript:;" @click="$router.push('/threads/hot')">今日热帖</a>.<a href="javascript:;" @click="$router.push('/channel/1')">公共论坛</a><br>
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
      plazaActivities: [],
      myReplies: [], favThreads: [], threadTotal: 0,
      // 诺哈 my_home 聚合
      homeAgg: null, myNews: [], friendNews: [], visitors: [], msgTotal: 0, favCount: 0, todayFirst: false
    }
  },
  mounted () { this.load() },
  methods: {
    load () {
      const me = this.$store.state.user
      if (!me) return
      const id = me.id
      api.get('/users/' + id).then(r => { if (r.code === 0) { this.u = r.data; this.threads = (r.data.threads || []).slice(0, 8) } }).catch(() => {})
      api.get('/my-threads', { params: { page: 1 } }).then(r => {
        if (r.code === 0) { this.threads = r.data.list || []; this.threadTotal = r.data.total || 0 }
      }).catch(() => {})
      api.get('/moods/latest').then(r => {
        if (r.code === 0) {
          // /moods/latest 返回 Mood 对象，取 content 字符串（避免闪现 [object Object]）
          const m = r.data
          this.mood = m ? (typeof m === 'string' ? m : (m.content || '')) : ''
        }
      }).catch(() => {})
      api.get('/friends').then(r => { if (r.code === 0) this.friends = (r.data.friends || []).slice(0, 5) }).catch(() => {})
      api.get('/games').then(r => { if (r.code === 0) this.games = r.data || [] }).catch(() => {})
      api.get('/my-games').then(r => { if (r.code === 0) this.myGames = r.data || [] }).catch(() => {})
      api.get('/plaza').then(r => {
        if (r.code === 0) {
          this.feed = (r.data.dynamics || []).slice(0, 8)
          this.fineThreads = (r.data.fine_threads || [])
          this.commonThreads = (r.data.quick_threads || [])
          this.announcements = (r.data.announcements || [])
          this.mood = this.mood || null
        }
      })
      api.get('/activities').then(r => { if (r.code === 0) this.plazaActivities = (r.data.list || []).slice(0, 5) }).catch(() => {})
      api.get('/space/' + id + '/messages').then(r => { if (r.code === 0) this.msgs = (r.data.list || r.data || []).slice(0, 3) }).catch(() => {})
      api.get('/my-replies').then(r => { if (r.code === 0) this.myReplies = r.data || [] }).catch(() => {})
      api.get('/favorite-threads').then(r => { if (r.code === 0) this.favThreads = r.data || [] }).catch(() => {})
      api.get('/home').then(r => {
        if (r.code === 0) {
          this.homeAgg = r.data
          this.myNews = r.data.my_news || []
          this.friendNews = r.data.friend_news || []
          this.visitors = r.data.visitors || []
          this.msgTotal = r.data.message_total || 0
          this.favCount = r.data.favorite_count || 0
          this.todayFirst = !!r.data.today_first
          if (r.data.mood && r.data.mood.content) this.mood = r.data.mood.content
        }
      }).catch(() => {})
    },
    visit () {
      if (!this.visitId) return
      api.get('/home/visit', { params: { no: this.visitId } }).then(r => {
        if (r.code === 0) this.$router.push('/space/' + r.data.user_id)
      }).catch(() => {})
    },
    brief (s) { s = s || ''; return s.length > 30 ? s.slice(0, 30) + '…' : s },
    homeIcon (u) {
      const lv = Math.max(1, Math.min(50, u.level || 1))
      const sex = (u.gender === 2 || u.gender === '2') ? '2' : '1'
      return this.$pic('home_' + sex + '_' + lv + '.gif')
    },
    iconErr (e) {
      const img = e.target
      if (img.dataset.fallback) { img.src = ''; img.style.visibility = 'hidden'; return }
      img.dataset.fallback = '1'
      img.src = this.$pic('v' + ((this.u && this.u.level) || 1) + '.gif')
    },
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
