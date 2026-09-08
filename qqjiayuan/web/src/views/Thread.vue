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
      <template v-else>
        <template v-if="thread.is_head">[头条]</template><template v-if="thread.is_top">【顶】</template><template v-if="thread.is_fine">【精】</template><template v-if="thread.is_recom">[荐]</template><template v-if="thread.is_notice">[公告]</template>
        {{ thread.title }}
      </template><br>
    </div>

    <!-- 板块/收藏/管理 -->
    <div class="title">
      <a v-if="thread.board" href="javascript:;" @click="$router.push('/board/'+thread.board.id)">[{{ thread.board.name }}]</a>
      <template v-if="thread.is_lock"><font color="#c00">[已锁定]</font></template>
      <template v-if="thread.audit_status === 0"><font color="#c00">[待审核]</font></template>
      <template v-if="thread.audit_status === 2"><font color="#c00">[审核未通过]</font></template>
      <template v-if="isLogin"><a href="javascript:;" @click="toggleFav"><font :color="favored ? '#1a9e1a' : '#004299'">{{ favored ? '★已收藏' : '☆收藏' }}</font></a></template>
      <template v-if="canManage || canMod">
        | <a href="javascript:;" @click="toggle('is_top')">{{ thread.is_top ? '取消置顶' : '置顶' }}</a>
        | <a href="javascript:;" @click="toggle('is_fine')">{{ thread.is_fine ? '取消精华' : '加精' }}</a>
        | <a href="javascript:;" @click="toggle('is_head')">{{ thread.is_head ? '取消头条' : '头条' }}</a>
        | <a href="javascript:;" @click="toggle('is_lock')">{{ thread.is_lock ? '解锁' : '锁定' }}</a>
        | <a href="javascript:;" @click="toggle('is_recom')">{{ thread.is_recom ? '取消推荐' : '推荐' }}</a>
      </template>
      <template v-if="canManage || canMod">
        | <a href="javascript:;" @click="openMove">移动</a>
        <template v-if="thread.audit_status !== 1"><a href="javascript:;" @click="audit(true)">通过</a>|<a href="javascript:;" @click="audit(false)">拒绝</a></template>
      </template>
      <template v-if="canManage || mine"> | <a href="javascript:;" @click="startEdit">{{ editing ? '取消编辑' : '编辑' }}</a> | <a href="javascript:;" style="color:#c00" @click="delThread">删除</a></template>
      <br>
    </div>

    <!-- 移动面板 -->
    <div class="module-content" v-if="moveOpen" style="background:#E3EEF8">
      移动到：
      <select v-model.number="moveBoardId">
        <optgroup v-for="ch in channels" :key="ch.id" :label="ch.name">
          <option v-for="s in allSubBoards(ch)" :key="s.id" :value="s.id">{{ s.name }}</option>
        </optgroup>
      </select>
      <input type="submit" value="确认移动" @click.prevent="doMove">
      <span v-if="moveMsg" style="color:#c00">{{ moveMsg }}</span><br>
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

    <!-- 投票 -->
    <div class="module-content" v-if="poll" style="background:#FFF6E5">
      <b>【投票】{{ poll.question }}</b><template v-if="poll.multiple">（多选）</template><br>
      <div v-for="o in poll.options" :key="o.id">
        <label v-if="!pollVoted">
          <input type="checkbox" v-if="poll.multiple" :value="o.id" v-model="pollSel">
          <input type="radio" v-else name="pollopt" :value="o.id" v-model="pollSel">
          {{ o.name }}（{{ o.votes }}票）
        </label>
        <template v-else>
          {{ o.name }}：<span class="bar-bg" style="display:inline-block;width:80px"><span class="bar-fg" :style="{width: percent(o.votes) + '%'}"></span></span> {{ o.votes }}票
        </template>
      </div>
      <template v-if="!pollVoted">
        <input type="submit" value="投票" @click.prevent="doVote">
      </template>
      <template v-else><font color="#1a9e1a">已投票（总 {{ pollTotal }} 票）</font></template>
      <span v-if="pollMsg" style="color:#c00">{{ pollMsg }}</span><br>
    </div>

    <!-- 回帖奖励 -->
    <div class="module-content" v-if="reward" style="background:#E6F7E6">
      <b>【回帖奖励】</b>每次回帖 +{{ reward.coins }} G币、+{{ reward.exp }} 经验<template v-if="reward.limit">（共 {{ reward.limit }} 次，已发 {{ reward.used }} 次）</template><br>
    </div>

    <!-- 踩楼 -->
    <div class="module-content" v-if="floors && floors.length" style="background:#F0E6F7">
      <b>【踩楼奖励】</b>踩中以下楼层得奖励：<br>
      <span v-for="f in floors" :key="f.id">第{{ f.floor }}楼 +{{ f.coins }}G币/{{ f.exp }}经验<template v-if="f.status">（已被{{ f.user_id }}踩中）</template> </span><br>
    </div>

    <!-- 附件 -->
    <div class="module-content" v-if="attachments && attachments.length" style="background:#E3EEF8">
      <b>【附件】</b><br>
      <span v-for="a in attachments" :key="a.id">
        <a v-if="a.type === 'file'" href="javascript:;" @click="downloadAtt(a)">📎 {{ a.name }}</a>
        <a v-else href="javascript:;" @click="downloadAtt(a)">🖼 {{ a.name }}</a>
        <template v-if="a.price">（{{ a.price }}G币）</template>（{{ a.downloads }}次下载）
        <br>
      </span>
    </div>

    <!-- 置顶回复 -->
    <div class="module-content" v-if="stickyReply" style="background:#FFFDE7">
      <b>【置顶回复】</b>{{ stickyReply.content }} —— {{ stickyReply.user ? stickyReply.user.nickname : '?' }}（{{ fmt(stickyReply.created_at) }}）<br>
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
      打赏G币：<input type="text" v-model.number="giftCoins" size="6"> <input type="submit" value="确认打赏" @click.prevent="doGift">
      <span class="help-line">（G币实时转入楼主账户）</span><br>
      <div v-if="gifts.length" class="txt-fade">
        <div v-for="g in gifts" :key="g.id">{{ g.sender ? g.sender.nickname : '?' }} 打赏 {{ g.coins }} G币（{{ fmt(g.created_at) }}）</div>
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
          {{ r.floor }}楼.<template v-if="r.parent_reply_id"><font color="#c00">[回复{{ parentFloor(r) }}楼]</font></template>{{ r.content }}<br>
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
          <template v-if="isLogin && !thread.is_lock">
            <textarea v-model.trim="content" rows="2" style="width:100%"></textarea><br>
            <input type="submit" value="回复"> <span class="help-line">回复+5经验+2G币</span>
            <template v-if="thread.type === 1"> <span class="help-line">本贴回帖有奖励</span></template>
            <input type="submit" value="顶贴" @click.prevent="doSticky"><span class="help-line">（楼主/版主置顶一条回复）</span>
          </template>
          <template v-else-if="isLogin && thread.is_lock">
            <font color="#c00">本贴已锁定，仅供查阅</font>
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
      reportReasons: ['广告垃圾', '辱骂攻击', '色情低俗', '造谣传谣', '引战挑事', '其他违规'],
      // 论坛新增
      poll: null, pollVoted: false, pollTotal: 0, myOptions: [], pollSel: [], pollMsg: '',
      reward: null, floors: [], stickyReply: null, attachments: [],
      channels: [], moveOpen: false, moveBoardId: 0, moveMsg: '', quoteReplyId: 0
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
    canManage () { return this.user && this.user.perms && this.user.perms.indexOf('thread:manage') >= 0 },
    canMod () {
      // 版主或楼主可管理（置顶回复等）
      return this.isLogin && (this.mine || (this.thread.board && this.user && this.thread.board.moderator_id === this.user.id))
    }
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
          // 论坛新增
          this.poll = r.data.poll || null
          this.pollVoted = r.data.poll_voted || false
          this.pollTotal = r.data.poll_total || 0
          this.myOptions = r.data.my_options || []
          this.pollSel = this.poll ? this.poll.options.filter(o => this.myOptions.indexOf(o.id) >= 0).map(o => o.id) : []
          this.reward = r.data.reward || null
          this.floors = r.data.floors || []
          this.stickyReply = r.data.sticky_reply || null
          this.attachments = r.data.attachments || []
          this.quoteReplyId = 0
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
      const r = this.replies.find(x => x.floor === floor)
      this.quoteReplyId = r ? r.id : 0
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
      api.post(`/threads/${this.$route.params.id}/replies`, { content: this.content, parent_reply_id: this.quoteReplyId || 0 }).then(r => {
        this.sending = false
        if (r.code === 0) {
          this.content = ''
          this.quoteReplyId = 0
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
      api.post(`/threads/${this.thread.id}/manage`, { field, value: v }).then(r => {
        if (r.code === 0) this.thread[field] = v
        else alert(r.msg)
      })
    },
    audit (approve) {
      api.post(`/threads/${this.thread.id}/audit`, { approve }).then(r => {
        if (r.code === 0) { this.thread.audit_status = approve ? 1 : 2; alert(approve ? '已通过审核' : '已拒绝') }
        else alert(r.msg)
      })
    },
    doVote () {
      if (!this.pollSel.length) { this.pollMsg = '请选择选项'; return }
      const optionIds = Array.isArray(this.pollSel) ? this.pollSel : [this.pollSel]
      api.post(`/threads/${this.thread.id}/poll-vote`, { option_ids: optionIds }).then(r => {
        if (r.code === 0) { alert('投票成功！'); this.load() }
        else { this.pollMsg = r.msg; alert(r.msg) }
      })
    },
    doSticky () {
      const content = prompt('输入置顶回复内容：')
      if (!content) return
      api.post(`/threads/${this.thread.id}/sticky-reply`, { content }).then(r => {
        if (r.code === 0) { alert('置顶回复成功'); this.load() }
        else alert(r.msg)
      })
    },
    downloadAtt (a) {
      if (!this.isLogin) { alert('请先登录'); return }
      api.get(`/attachments/${a.id}/download`).then(r => {
        if (r.code === 0) {
          if (a.price) alert('下载成功，已扣除 ' + a.price + ' G币')
          else alert('下载成功')
          this.load()
        } else alert(r.msg)
      })
    },
    openMove () {
      if (!this.channels.length) {
        api.get('/boards').then(r => { if (r.code === 0) this.channels = r.data })
      }
      this.moveOpen = !this.moveOpen
      this.moveMsg = ''
    },
    allSubBoards (ch) {
      const arr = (ch.children || []).slice()
      ;(ch.categories || []).forEach(cat => (cat.boards || []).forEach(b => arr.push(b)))
      return arr
    },
    doMove () {
      if (!this.moveBoardId) { this.moveMsg = '请选择目标版块'; return }
      api.post(`/threads/${this.thread.id}/move`, { board_id: this.moveBoardId }).then(r => {
        if (r.code === 0) { alert('移动成功'); this.moveOpen = false; this.load() }
        else { this.moveMsg = r.msg; alert(r.msg) }
      })
    },
    parentFloor (r) {
      const p = this.replies.find(x => x.id === r.parent_reply_id)
      return p ? p.floor : '?'
    },
    percent (v) {
      const max = Math.max(...(this.poll ? this.poll.options.map(o => o.votes) : [1]), 1)
      return Math.round((v / max) * 100)
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
