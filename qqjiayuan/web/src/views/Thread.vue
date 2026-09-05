<template>
  <div>
    <!-- 面包屑：社区 > 父板块 > 板块 -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;
      <template v-if="thread.board && thread.board.parent_id">
        <a href="javascript:;" @click="$router.push('/channel/'+thread.board.parent_id)">{{ parentName }}</a>&gt;
      </template>
      <a v-if="thread.board" href="javascript:;" @click="$router.push('/board/'+thread.board.id)">{{ thread.board.name }}</a><br>
    </div>

    <!-- 标题 -->
    <div class="name">
      <template v-if="editing">
        <input type="text" v-model.trim="editForm.title" maxlength="100" style="width:96%">
      </template>
      <template v-else>{{ thread.title }}</template><br>
    </div>

    <!-- 板块/收藏/管理 -->
    <div class="title">
      <a v-if="thread.board" href="javascript:;" @click="$router.push('/board/'+thread.board.id)">[{{ thread.board.name }}]</a>
      <template v-if="isLogin"><a href="javascript:;" @click="toggleFav"><font :color="favored ? '#1a9e1a' : '#004299'">{{ favored ? '★已收藏' : '☆收藏' }}</font></a></template>
      <template v-if="canManage"> | <a href="javascript:;" @click="toggle('is_top')">{{ thread.is_top ? '取消置顶' : '置顶' }}</a> | <a href="javascript:;" @click="toggle('is_fine')">{{ thread.is_fine ? '取消精华' : '加精' }}</a></template>
      <template v-if="canManage || mine"> | <a href="javascript:;" @click="startEdit">{{ editing ? '取消编辑' : '编辑' }}</a> | <a href="javascript:;" style="color:#c00" @click="delThread">删除</a></template>
      <br>
    </div>

    <!-- 正文 -->
    <template v-if="editing">
      <div class="module-content">
        编辑正文:<br>
        <textarea v-model="editForm.content" rows="12" style="width:98%"></textarea><br>
        <input type="submit" value="保存修改" @click.prevent="saveEdit"> <input type="submit" value="取消" @click.prevent="editing = false">
        <span v-if="editTip" :style="{color: editOk ? '#1a9e1a' : '#c00'}"> {{ editTip }}</span>
      </div>
    </template>
    <template v-else>
      <div class="text" v-html="contentFace"></div>
    </template>
    <div class="line"></div>

    <!-- 统计：页数/字数/阅读 -->
    <div class="item">
      (第<b>{{ page }}</b>/{{ pages }}页/{{ wordCount }}字/{{ thread.view_count || 0 }}阅)<br>
    </div>

    <!-- 楼主信息 -->
    <div class="item">
      <span v-for="b in authorBadges" :key="b.id"><img class="bicon" :src="$pic(b.icon)" :alt="b.name"></span>
      <img class="bicon" v-if="author.priv" :src="'/static/' + author.priv.file" :alt="author.priv.name" :title="author.priv.name">
      <img class="bicon" v-else-if="author.level_icon" :src="$pic('v'+author.level_icon+'.gif')" alt="等级">
      <a href="javascript:;" @click="$router.push('/user/'+author.id)"><font :color="author.color || '#004299'">{{ author.nickname || '?' }}</font></a>
      <i><font color="SlateGray">{{ fmt(thread.created_at) }}</font></i><br>
    </div>
    <div class="item" v-if="author.city">发表于：<i><font color="SlateGray">{{ author.city }}</font></i> <br></div>

    <!-- 互动区：[评价] -->
    <div class="item">
      [评价] <input type="submit" :value="'赞(' + likeCount + ')'" @click.prevent="vote(1)">
      <input type="submit" :value="'踩(' + dislikeCount + ')'" @click.prevent="vote(-1)">
      <input type="submit" value="打赏" @click.prevent="giftOpen = !giftOpen"><br>
    </div>

    <!-- 打赏面板 -->
    <div class="module-content" v-if="giftOpen" style="background:#E3EEF8">
      打赏金币：<input type="text" v-model.number="giftCoins" size="6"> <input type="submit" value="确认打赏" @click.prevent="doGift">
      <span class="help-line">（金币实时转入楼主账户）</span><br>
      <div v-if="gifts.length" class="txt-fade">
        <div v-for="g in gifts" :key="g.id">{{ g.sender ? g.sender.nickname : '?' }} 打赏 {{ g.coins }} 金币（{{ fmt(g.created_at) }}）</div>
      </div>
    </div>

    <!-- 互动区：[鲜花] -->
    <div class="item">
      [鲜花] ({{ flowerCount }})朵 <a href="javascript:;" @click="flowerLogOpen = !flowerLogOpen">{{ flowerLogOpen ? '收起收花记录' : '查看收花记录' }}</a><br>
    </div>
    <div class="module-content" v-if="flowerLogOpen" style="background:#E3EEF8">
      <div v-if="flowers.length">
        <div v-for="f in flowers" :key="f.id">{{ f.sender ? f.sender.nickname : '?' }} 送出 {{ f.count }} 朵{{ f.flower }}（{{ fmt(f.created_at) }}）</div>
      </div>
      <div v-else class="empty">还没有人送花</div>
    </div>

    <!-- 互动区：[送花] -->
    <div class="item">
      [送花] <input type="submit" value="99朵" @click.prevent="doFlower(99)">
      <input type="submit" value="520朵" @click.prevent="doFlower(520)">
      <input type="submit" value="999朵" @click.prevent="doFlower(999)">
      <input type="text" v-model.number="flowerNum" size="4" @keyup.enter="doFlower(flowerNum)"> <input type="submit" value="自定义" @click.prevent="doFlower(flowerNum)">
      <select v-model="flowerKind" style="margin-left:4px">
        <option v-for="f in flowerKinds" :key="f" :value="f">{{ f }}</option>
      </select>
      <a href="javascript:;" @click="$router.push('/shop')">商城买鲜花&gt;&gt;</a>（从购买的鲜花中扣除）<br>
    </div>

    <!-- 互动区：[分享/收藏/复制/举报] -->
    <div class="item">
      <input type="submit" value="分享" @click.prevent="doShare(false)">
      <a href="javascript:;" @click="toggleFav"><font :color="favored ? '#1a9e1a' : '#004299'">{{ favored ? '★已收藏' : '收藏' }}</font></a>
      .<a href="javascript:;" @click="copyLink">复制本帖链接</a>
      .<a href="javascript:;" @click="openReport('thread', thread.id)">举报该帖</a>
      <input type="submit" value="分享到心情" @click.prevent="doShare(true)" v-if="isLogin"><br>
      <span v-if="shareTip" style="color:#1a9e1a">{{ shareTip }}</span>
      <span v-if="msg" style="color:#c00">{{ msg }}</span>
    </div>

    <!-- 举报面板 -->
    <div class="module-content" v-if="reportOpen" style="background:#E3EEF8">
      举报理由：
      <select v-model="reportReasonSel">
        <option value="">选择理由</option>
        <option v-for="r in reportReasons" :key="r" :value="r">{{ r }}</option>
      </select><br>
      或输入自定义理由：<input type="text" v-model.trim="reportReasonCustom" maxlength="100" style="width:60%"><br>
      <input type="submit" value="提交举报" @click.prevent="submitReport">
      <span v-if="reportMsg" style="color:#1a9e1a">{{ reportMsg }}</span>
    </div>

    <!-- 回帖列表 -->
    <div class="name">
      回帖列表 <a class="rt" href="javascript:;" @click="$router.push('/board/'+(thread.board ? thread.board.id : ''))">返回帖子列表</a><br>
    </div>
    <div class="list">
      <div v-for="(r, i) in replies" :key="r.id" :class="['module-content', i % 2 === 1 ? 'deep' : '']">
        <div class="row">
          {{ r.floor }}楼.{{ r.content }}<br>
          <span v-for="b in (r.user ? r.user.badges : [])" :key="b.id"><img class="bicon" :src="$pic(b.icon)" :alt="b.name"></span>
          <img class="bicon" v-if="r.user && r.user.priv" :src="'/static/' + r.user.priv.file" :alt="r.user.priv.name" :title="r.user.priv.name">
          <img class="bicon" v-else-if="r.user && r.user.level_icon" :src="$pic('v'+r.user.level_icon+'.gif')" alt="等级">
          <a href="javascript:;" @click="$router.push('/user/'+(r.user ? r.user.id : ''))"><font :color="r.user ? r.user.color : ''">{{ r.user ? r.user.nickname : '路人' }}</font></a>
          <i><font color="SlateGray">{{ fmt(r.created_at) }}</font></i>
          <template v-if="isLogin">
            <a href="javascript:;" @click="likeReply(r)">[赞{{ r.like_count || 0 }}]</a>
            <a href="javascript:;" @click="quote(r.floor)">[回复]</a>
            <a href="javascript:;" v-if="user && r.user && r.user.id === user.id" style="color:#c00" @click="delReply(r)">[删除]</a>
            <a href="javascript:;" @click="openReport('reply', r.id)">[举报]</a>
          </template>
          <br>
        </div>
      </div>
    </div>
    <div v-if="!replies.length" class="empty">还没有人回复，来抢沙发！</div>

    <!-- 分页 -->
    <div class="item">
      <a v-if="page > 1" href="javascript:;" @click="go(page-1)">上页</a><template v-if="page > 1">.</template>
      <a v-if="page < pages" href="javascript:;" @click="go(page+1)">下页</a><template v-if="page < pages">.</template>
      <form style="display:inline" @submit.prevent="go(pageInput)">
        第<input type="text" v-model.number="pageInput" size="2" maxlength="4">页
        <input type="submit" value="前往">
      </form>
      (第<b>{{ page }}</b>/{{ pages }}页)<br>
    </div>

    <!-- 回复框（write-mood） -->
    <div class="write-mood">
      <div class="item">
        <form @submit.prevent="submit">
          <template v-if="isLogin">
            <textarea v-model.trim="content" rows="2" style="width:100%"></textarea><br>
            <input type="submit" value="回复"> <span class="help-line">回复+5经验+2金币</span>
          </template>
          <template v-else>
            <a href="javascript:;" @click="$router.push('/login?redirect='+$route.fullPath)">登陆家园社区</a>后回复盖楼
          </template>
        </form>
      </div>
    </div>

    <!-- 全部回帖 -->
    <a href="javascript:;" @click="go(pages)"><b>全部回帖({{ total }})</b></a><br>

    <!-- 面包屑重复 -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;
      <template v-if="thread.board && thread.board.parent_id">
        <a href="javascript:;" @click="$router.push('/channel/'+thread.board.parent_id)">{{ parentName }}</a>&gt;
      </template>
      <a v-if="thread.board" href="javascript:;" @click="$router.push('/board/'+thread.board.id)">{{ thread.board.name }}</a><br>
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
      thread: { user: {}, board: {} },
      replies: [], total: 0, page: 1, pages: 1, pageInput: 1,
      content: '', sending: false,
      editing: false, editForm: { title: '', content: '' }, editTip: '', editOk: false, saving: false,
      favored: false,
      // 互动
      myVote: 0, likeCount: 0, dislikeCount: 0,
      giftTotal: 0, giftCount: 0, flowerCount: 0, flowerPeople: 0, shareCount: 0,
      flowers: [], gifts: [],
      giftOpen: false, giftCoins: 100,
      flowerOpen: false, flowerLogOpen: false, flowerKind: '玫瑰花', flowerNum: 99,
      flowerKinds: ['玫瑰花', '向日葵', '郁金香', '月光花'],
      shareTip: '', msg: '',
      reportOpen: false, reportType: 'thread', reportTargetId: 0,
      reportReasonSel: '', reportReasonCustom: '', reportMsg: '',
      reportReasons: ['广告垃圾', '辱骂攻击', '色情低俗', '造谣传谣', '引战挑事', '其他违规']
    }
  },
  computed: {
    contentFace () { return renderFace(this.thread.content) },
    wordCount () { return (this.thread.content || '').length },
    parentName () {
      const b = this.thread.board
      if (b && b.parent_name) return b.parent_name
      return '论坛'
    },
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
          // 互动统计
          this.likeCount = r.data.like_count || 0
          this.dislikeCount = r.data.dislike_count || 0
          this.giftTotal = r.data.gift_total || 0
          this.giftCount = r.data.gift_count || 0
          this.flowerCount = r.data.flower_count || 0
          this.flowerPeople = r.data.flower_people || 0
          this.shareCount = r.data.share_count || 0
          this.flowers = r.data.flowers || []
          this.gifts = r.data.gifts || []
        } else {
          alert(r.msg)
        }
      })
      if (this.isLogin) {
        api.get(`/threads/${id}/favorite-status`).then(r => { if (r.code === 0) this.favored = r.data.favored })
        api.get(`/threads/${id}/interact-status`).then(r => {
          if (r.code === 0) {
            this.myVote = r.data.my_vote || 0
            this.likeCount = r.data.like_count || 0
            this.dislikeCount = r.data.dislike_count || 0
            this.giftTotal = r.data.gift_total || 0
            this.giftCount = r.data.gift_count || 0
            this.flowerCount = r.data.flower_count || 0
            this.flowerPeople = r.data.flower_people || 0
            this.shareCount = r.data.share_count || 0
            this.flowers = r.data.flowers || []
            this.gifts = r.data.gifts || []
          }
        })
      }
    },
    toggleFav () {
      api.post(`/threads/${this.thread.id}/favorite`).then(r => { if (r.code === 0) this.favored = r.data.favored })
    },
    quote (floor) {
      this.content = '回复' + floor + '楼：'
      const el = document.querySelector('.write-mood textarea')
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
      if (p > this.pages) p = this.pages
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
    // ---- 互动 ----
    vote (v) {
      if (!this.isLogin) { alert('请先登录'); return }
      api.post(`/threads/${this.thread.id}/vote`, { value: v }).then(r => {
        if (r.code === 0) {
          this.myVote = r.data.my_vote
          this.likeCount = r.data.like_count
          this.dislikeCount = r.data.dislike_count
        } else alert(r.msg)
      })
    },
    doGift () {
      if (!this.giftCoins || this.giftCoins < 1) { alert('请输入打赏金额'); return }
      api.post(`/threads/${this.thread.id}/gift`, { coins: this.giftCoins }).then(r => {
        if (r.code === 0) {
          this.giftTotal = r.data.gift_total
          this.giftCount = r.data.gift_count
          this.giftOpen = false
          alert('打赏成功！')
          this.load()
        } else alert(r.msg)
      })
    },
    doFlower (n) {
      if (!this.isLogin) { alert('请先登录'); return }
      const count = Math.max(1, Math.floor(n || 1))
      api.post(`/threads/${this.thread.id}/flower`, { flower: this.flowerKind, count }).then(r => {
        if (r.code === 0) {
          this.flowerCount = r.data.flower_count
          alert('送花成功！')
          this.load()
        } else alert(r.msg)
      })
    },
    doShare (toMood) {
      if (!this.isLogin) { alert('请先登录'); return }
      api.post(`/threads/${this.thread.id}/share`, { to_mood: !!toMood }).then(r => {
        if (r.code === 0) {
          this.shareCount = r.data.share_count
          this.shareTip = toMood ? '已分享到心情！' : '分享成功！'
          setTimeout(() => { this.shareTip = '' }, 2500)
        } else alert(r.msg)
      })
    },
    copyLink () {
      const url = location.origin + location.pathname + '#/thread/' + this.thread.id
      const ta = document.createElement('textarea')
      ta.value = url
      document.body.appendChild(ta)
      ta.select()
      try { document.execCommand('copy') } catch (e) {}
      document.body.removeChild(ta)
      this.shareTip = '链接已复制：' + url
      setTimeout(() => { this.shareTip = '' }, 3000)
    },
    likeReply (r) {
      api.post(`/replies/${r.id}/like`).then(x => {
        if (x.code === 0) r.like_count = x.data.like_count
        else alert(x.msg)
      })
    },
    openReport (type, id) {
      if (!this.isLogin) { alert('请先登录'); return }
      this.reportType = type
      this.reportTargetId = id
      this.reportReasonSel = ''
      this.reportReasonCustom = ''
      this.reportMsg = ''
      this.reportOpen = true
      const el = document.querySelector('.write-mood')
      if (el) el.scrollIntoView()
    },
    submitReport () {
      const reason = this.reportReasonCustom || this.reportReasonSel
      if (!reason) { alert('请选择或填写举报理由'); return }
      api.post('/reports', { target_type: this.reportType, target_id: this.reportTargetId, reason }).then(r => {
        if (r.code === 0) {
          this.reportMsg = r.data.duplicated ? '你已举报过该内容，等待管理员处理' : '举报成功，管理员会尽快处理'
          this.reportOpen = false
        } else alert(r.msg)
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
