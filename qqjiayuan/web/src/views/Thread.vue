<template>
  <div>
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;<a v-if="thread.board" href="javascript:;" @click="$router.push('/board/'+thread.board.id)">{{ thread.board.name }}</a><br>
    </div>
    <div class="name">
      <template v-if="editing">
        <input type="text" v-model.trim="editForm.title" maxlength="100" style="width:98%;padding:4px;border:1px solid #2e9cd3">
      </template>
      <template v-else>{{ thread.title }}</template>
    </div>
    <div class="title">
      [板块]:<a v-if="thread.board" href="javascript:;" @click="$router.push('/board/'+thread.board.id)">{{ thread.board.name }}</a>
      <template v-if="isLogin"><a href="javascript:;" @click="toggleFav"><font :color="favored ? '#1a9e1a' : '#004299'">{{ favored ? '★已收藏' : '☆收藏' }}</font></a></template>
      <template v-if="canManage"> | <a href="javascript:;" @click="toggle('is_top')">{{ thread.is_top ? '取消置顶' : '置顶' }}</a> | <a href="javascript:;" @click="toggle('is_fine')">{{ thread.is_fine ? '取消精华' : '加精' }}</a></template>
      <template v-if="canManage || mine"> | <a href="javascript:;" @click="startEdit">{{ editing ? '取消编辑' : '编辑' }}</a> | <a href="javascript:;" style="color:#c00" @click="delThread">删除</a></template>
      <br>
    </div>
    <template v-if="editing">
      <div class="module-content">
        编辑正文:<br>
        <textarea v-model="editForm.content" rows="12" style="width:98%"></textarea><br>
        <button class="btn" type="button" @click="saveEdit" :disabled="saving">保存修改</button>
        <button class="btn gray" type="button" @click="editing = false">取消</button>
        <span v-if="editTip" :style="{color: editOk ? '#1a9e1a' : '#c00'}"> {{ editTip }}</span>
      </div>
    </template>
    <template v-else>
      <div class="text" v-html="contentFace"></div>
    </template>
    <div class="line"></div>

    <!-- 互动条（对齐参考站：赞/踩/打赏/送花/分享/收藏/复制链接/举报） -->
    <div class="item interact-bar">
      <a href="javascript:;" @click="vote(1)"><font :color="myVote === 1 ? '#1a9e1a' : '#004299'">[赞{{ likeCount }}]</font></a>
      <a href="javascript:;" @click="vote(-1)"><font :color="myVote === -1 ? '#c00' : '#004299'">[踩{{ dislikeCount }}]</font></a>
      <a href="javascript:;" @click="giftOpen = !giftOpen">[打赏]</a>
      <a href="javascript:;" @click="flowerOpen = !flowerOpen">[送花]</a>
      <a href="javascript:;" @click="doShare(false)">[分享{{ shareCount }}]</a>
      <a href="javascript:;" @click="doShare(true)">[分享到心情]</a>
      <a href="javascript:;" @click="copyLink">[复制本帖链接]</a>
      <a href="javascript:;" @click="openReport('thread', thread.id)">[举报该帖]</a><br>
      <span class="txt-fade" v-if="giftCount || flowerCount">
        <template v-if="giftCount">共收到 {{ giftCount }} 次打赏 {{ giftTotal }} 金币</template><template v-if="giftCount && flowerCount">，</template><template v-if="flowerCount">{{ flowerCount }} 朵花（{{ flowerPeople }} 人送）</template>
      </span>
    </div>

    <!-- 打赏面板 -->
    <div class="module-content" v-if="giftOpen">
      [打赏金币]<br>
      <input type="number" v-model.number="giftCoins" min="1" max="100000" style="width:90px">
      <button class="btn" @click="doGift" :disabled="sending">打赏</button>
      <span class="txt-fade">（金币实时转入楼主账户）</span>
      <div v-if="gifts.length" class="txt-fade">
        <div v-for="g in gifts" :key="g.id">{{ g.sender ? g.sender.nickname : '?' }} 打赏 {{ g.coins }} 金币（{{ fmt(g.created_at) }}）</div>
      </div>
    </div>

    <!-- 送花面板 -->
    <div class="module-content" v-if="flowerOpen">
      [送花] 花：<select v-model="flowerKind" style="width:80px">
        <option v-for="f in flowerKinds" :key="f" :value="f">{{ f }}</option>
      </select>
      <button class="btn" @click="doFlower(99)">99朵</button>
      <button class="btn" @click="doFlower(520)">520朵</button>
      <button class="btn" @click="doFlower(999)">999朵</button>
      <input type="number" v-model.number="flowerNum" min="1" max="999" style="width:70px">
      <button class="btn" @click="doFlower(flowerNum)">自定义</button>
      <a href="javascript:;" @click="$router.push('/games/garden')">去花园种花&gt;&gt;</a><br>
      <div v-if="flowers.length" class="txt-fade">
        <div v-for="f in flowers" :key="f.id">{{ f.sender ? f.sender.nickname : '?' }} 送出 {{ f.count }} 朵{{ f.flower }}（{{ fmt(f.created_at) }}）</div>
      </div>
    </div>

    <!-- 举报面板 -->
    <div class="module-content" v-if="reportOpen">
      [举报{{ reportType === 'thread' ? '该帖' : '该回复' }}]<br>
      <select v-model="reportReason">
        <option value="">选择理由</option>
        <option v-for="r in reportReasons" :key="r" :value="r">{{ r }}</option>
      </select><br>
      或输入自定义理由：<input type="text" v-model.trim="reportReason" maxlength="200" style="width:62%"><br>
      <button class="btn" @click="submitReport">提交举报</button>
      <span v-if="reportMsg" style="color:#1a9e1a">{{ reportMsg }}</span>
    </div>

    <!-- 楼主信息 -->
    <div class="item">
      [楼主]:<span v-for="b in authorBadges" :key="b.id"><img class="bicon" :src="$pic(b.icon)" :alt="b.name"></span><img class="bicon" v-if="author.priv" :src="'/static/' + author.priv.file" :alt="author.priv.name" :title="author.priv.name"><img class="bicon" v-else-if="author.level_icon" :src="$pic('v'+author.level_icon+'.gif')" alt="等级"><a href="javascript:;" @click="$router.push('/user/'+author.id)"><font :color="author.color">{{ author.nickname }}</font>（{{ author.username }}）</a><br>
      [发帖时间]:{{ fmt(thread.created_at) }}<br>
      <a v-if="isLogin" href="javascript:;" @click="quote(1)">回复</a><template v-if="isLogin">.</template><a href="javascript:;" @click="$router.push('/user/'+author.id)">TA的主页</a><br>
    </div>

    <!-- 回帖列表（斑马纹：module-content / deep 交替） -->
    <div class="name">回帖列表 <a class="rt" href="javascript:;" @click="$router.push('/board/'+(thread.board?thread.board.id:''))">返回帖子列表</a><br></div>
    <div class="list">
      <div v-for="(r, i) in replies" :key="r.id" :class="['module-content', i % 2 === 1 ? 'deep' : '']">
        <div class="row">
          {{ r.floor }}楼.{{ r.content }}<br>
          <span v-for="b in (r.user?r.user.badges:[])" :key="b.id"><img class="bicon" :src="$pic(b.icon)" :alt="b.name"></span><img class="bicon" v-if="r.user && r.user.priv" :src="'/static/' + r.user.priv.file" :alt="r.user.priv.name" :title="r.user.priv.name"><img class="bicon" v-else-if="r.user && r.user.level_icon" :src="$pic('v'+r.user.level_icon+'.gif')" alt="等级"> <a href="javascript:;" @click="$router.push('/user/'+(r.user?r.user.id:''))"><font :color="r.user?r.user.color:''">{{ r.user?r.user.nickname:'路人' }}</font></a><i><font color="SlateGray"> {{ fmt(r.created_at) }}</font></i>
          <a v-if="isLogin" href="javascript:;" @click="likeReply(r)">[赞{{ r.like_count || 0 }}]</a>
          <a v-if="isLogin" href="javascript:;" @click="quote(r.floor)">[回复]</a>
          <a v-if="user && r.user && r.user.id === user.id" href="javascript:;" style="color:#c00" @click="delReply(r)">[删除]</a>
          <a v-if="isLogin" href="javascript:;" @click="openReport('reply', r.id)">[举报]</a>
        </div>
      </div>
    </div>
    <div v-if="!replies.length" class="empty">还没有人回复，来抢沙发！</div>

    <div class="item">
      <a v-if="page > 1" href="javascript:;" @click="go(page-1)">上页</a><template v-if="page > 1">.</template><a v-if="page < pages" href="javascript:;" @click="go(page+1)">下页</a><template v-if="page < pages">.</template>
      <form style="display:inline" @submit.prevent="go(pageInput)">
        第<input type="text" v-model.number="pageInput" size="2" maxlength="4">页
        <input type="submit" value="前往">
      </form>
      (第<b>{{ page }}</b>/{{ pages }}页/{{ thread.view_count }}阅)<br>
    </div>

    <div class="module-title">【回复盖楼】</div>
    <div class="module-content" v-if="isLogin">
      <form @submit.prevent="submit">
        [内容]:<br>
        <textarea v-model.trim="content" maxlength="2000"></textarea><br>
        <input type="submit" value="确定回复"> <span class="help-line">回复+5经验+2金币</span>
      </form>
    </div>
    <div class="module-content" v-else>
      <a href="javascript:;" @click="$router.push('/login?redirect='+$route.fullPath)">登陆家园社区</a>后回复盖楼，与友友一起怀旧
    </div>
  </div>
</template>

<script>
import api from '../api'
import { renderFace } from '../utils/qqface'

export default {
  name: 'Thread',
  data () {
    return {
      thread: { user: {}, board: {} }, replies: [], total: 0, page: 1, pages: 1, pageInput: 1, content: '', sending: false,
      editing: false, editForm: { title: '', content: '' }, editTip: '', editOk: false, saving: false, favored: false,
      // 互动
      myVote: 0, likeCount: 0, dislikeCount: 0,
      giftTotal: 0, giftCount: 0, flowerCount: 0, flowerPeople: 0, shareCount: 0,
      flowers: [], gifts: [], likedReplies: {},
      giftOpen: false, giftCoins: 100,
      flowerOpen: false, flowerKind: '玫瑰花', flowerNum: 99,
      flowerKinds: ['玫瑰花', '向日葵', '郁金香', '月光花'],
      reportOpen: false, reportType: 'thread', reportTargetId: 0, reportReason: '', reportMsg: '',
      reportReasons: ['广告/垃圾信息', '人身攻击', '色情低俗', '违法违规', '侵权抄袭', '其他违规'],
      shareTip: ''
    }
  },
  computed: {
    contentFace () { return renderFace(this.thread.content) },
    isLogin () { return this.$store.getters.isLogin },
    user () { return this.$store.state.user },
    author () { return this.thread.user || {} },
    authorBadges () { return this.author.badges || [] },
    mine () { return this.user && this.author && this.user.id === this.author.id },
    canManage () { return this.user && this.user.perms && this.user.perms.indexOf('thread:manage') >= 0 }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    faceHtml (t) { return renderFace(t) },
    load () {
      const id = this.$route.params.id
      const page = parseInt(this.$route.query.page || 1)
      api.get(`/threads/${id}`, { params: { page } }).then(r => {
        if (r.code === 0) {
          this.thread = r.data.thread
          this.replies = r.data.replies
          this.total = r.data.total
          this.page = r.data.page
          this.pageInput = r.data.page
          this.pages = Math.max(1, Math.ceil(r.data.total / r.data.size))
          this.likeCount = r.data.like_count || 0
          this.dislikeCount = r.data.dislike_count || 0
          this.giftTotal = r.data.gift_total || 0
          this.giftCount = r.data.gift_count || 0
          this.flowerCount = r.data.flower_count || 0
          this.flowerPeople = r.data.flower_people || 0
          this.shareCount = r.data.share_count || 0
          this.flowers = r.data.flowers || []
          this.gifts = r.data.gifts || []
          this.myVote = 0
        } else {
          alert(r.msg)
        }
      })
      if (this.isLogin) {
        api.get(`/threads/${id}/favorite-status`).then(r => { if (r.code === 0) this.favored = r.data.favored })
        api.get(`/threads/${id}/interact-status`).then(r => {
          if (r.code === 0) {
            this.myVote = r.data.my_vote || 0
            this.likeCount = r.data.like_count
            this.dislikeCount = r.data.dislike_count
            this.giftTotal = r.data.gift_total
            this.giftCount = r.data.gift_count
            this.flowerCount = r.data.flower_count
            this.flowerPeople = r.data.flower_people
            this.shareCount = r.data.share_count
            this.flowers = r.data.flowers || []
            this.gifts = r.data.gifts || []
          }
        })
      }
    },
    toggleFav () {
      api.post(`/threads/${this.thread.id}/favorite`).then(r => { if (r.code === 0) this.favored = r.data.favored })
    },
    vote (v) {
      if (!this.isLogin) return this.$router.push('/login?redirect=' + this.$route.fullPath)
      const target = this.myVote === v ? 0 : v
      api.post(`/threads/${this.thread.id}/vote`, { value: target }).then(r => {
        if (r.code === 0) {
          this.myVote = r.data.my_vote
          this.likeCount = r.data.like_count
          this.dislikeCount = r.data.dislike_count
        } else alert(r.msg)
      })
    },
    doGift () {
      if (!this.giftCoins || this.giftCoins < 1) return alert('请输入打赏金币数（1~100000）')
      this.sending = true
      api.post(`/threads/${this.thread.id}/gift`, { coins: this.giftCoins }).then(r => {
        this.sending = false
        if (r.code === 0) { alert('打赏成功！'); this.giftOpen = false; this.load() } else alert(r.msg)
      })
    },
    doFlower (n) {
      if (!n || n < 1) return alert('请输入正确的送花数量')
      this.sending = true
      api.post(`/threads/${this.thread.id}/flower`, { flower: this.flowerKind, count: n }).then(r => {
        this.sending = false
        if (r.code === 0) { alert(`送花成功！`); this.flowerOpen = false; this.load() } else alert(r.msg)
      })
    },
    doShare (toMood) {
      if (!this.isLogin) return this.$router.push('/login?redirect=' + this.$route.fullPath)
      api.post(`/threads/${this.thread.id}/share`, { to_mood: !!toMood }).then(r => {
        if (r.code === 0) {
          this.shareCount = r.data.share_count
          alert(toMood ? '已分享到我的心情！' : '分享成功！')
        } else alert(r.msg)
      })
    },
    copyLink () {
      const url = location.origin + '/#/thread/' + this.thread.id
      const done = () => alert('链接已复制：' + url)
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(url).then(done).catch(() => { this.fallbackCopy(url); done() })
      } else { this.fallbackCopy(url); done() }
    },
    fallbackCopy (text) {
      const ta = document.createElement('textarea')
      ta.value = text
      document.body.appendChild(ta)
      ta.select()
      try { document.execCommand('copy') } catch (e) {}
      document.body.removeChild(ta)
    },
    openReport (type, id) {
      if (!this.isLogin) return this.$router.push('/login?redirect=' + this.$route.fullPath)
      this.reportType = type
      this.reportTargetId = id
      this.reportReason = ''
      this.reportMsg = ''
      this.reportOpen = true
    },
    submitReport () {
      if (!this.reportReason) return alert('请选择或填写举报理由')
      api.post('/reports', { target_type: this.reportType, target_id: this.reportTargetId, reason: this.reportReason }).then(r => {
        if (r.code === 0) {
          this.reportMsg = r.data && r.data.duplicated ? '该举报已提交过，管理员会尽快处理' : '举报成功！管理员会尽快处理'
          this.reportOpen = false
          setTimeout(() => { this.reportMsg = '' }, 3000)
        } else alert(r.msg)
      })
    },
    likeReply (r) {
      api.post(`/replies/${r.id}/like`).then(x => {
        if (x.code === 0) {
          r.like_count = x.data.like_count
          this.likedReplies[r.id] = x.data.liked
        } else alert(x.msg)
      })
    },
    quote (floor) {
      this.content = '回复' + floor + '楼：'
      const el = document.querySelector('textarea')
      if (el) { el.scrollIntoView(); el.focus() }
    },
    startEdit () {
      if (this.editing) { this.editing = false; return }
      this.editForm = { title: this.thread.title, content: this.thread.content }
      this.editTip = ''
      this.editing = true
    },
    saveEdit () {
      if (!this.editForm.title || !this.editForm.content) { this.editTip = '标题和内容都不能为空'; this.editOk = false; return }
      this.saving = true
      api.put(`/threads/${this.thread.id}`, { title: this.editForm.title, content: this.editForm.content }).then(r => {
        this.saving = false
        if (r.code === 0) {
          this.editTip = '已保存'
          this.editOk = true
          this.editing = false
          this.load()
        } else {
          this.editTip = r.msg
          this.editOk = false
        }
      })
    },
    submit () {
      if (!this.content) return
      this.sending = true
      api.post(`/threads/${this.$route.params.id}/replies`, { content: this.content }).then(r => {
        this.sending = false
        if (r.code === 0) {
          this.content = ''
          api.get(`/threads/${this.$route.params.id}`, { params: { page: 999999 } }).then(d => {
            if (d.code === 0) this.go(d.data.page)
          })
        } else {
          alert(r.msg)
        }
      })
    },
    go (p) {
      if (p < 1) p = 1
      this.$router.push('/thread/' + this.$route.params.id + '?page=' + p)
    },
    toggle (field) {
      const v = this.thread[field] ? 0 : 1
      api.put(`/admin/threads/${this.thread.id}`, { [field]: v }).then(r => {
        if (r.code === 0) this.thread[field] = v
        else alert(r.msg)
      })
    },
    delThread () {
      if (!confirm('确定删除这篇帖子吗？')) return
      const back = () => {
        this.$router.push(this.thread.board ? '/board/' + this.thread.board.id : '/')
      }
      api.delete(`/threads/${this.thread.id}`).then(r => {
        if (r.code === 0) { alert('已删除'); back() } else {
          api.delete(`/admin/threads/${this.thread.id}`).then(d => {
            if (d.code === 0) { alert('已删除'); back() } else alert(d.msg)
          })
        }
      })
    },
    delReply (r) {
      if (!confirm('确定删除这条回复吗？')) return
      api.delete(`/replies/${r.id}`).then(x => {
        if (x.code === 0) this.load()
        else alert(x.msg)
      })
    },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t)
      const p = n => (n < 10 ? '0' + n : n)
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds())
    }
  }
}
</script>
