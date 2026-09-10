<template>
  <div>
    <!-- 广告位（诺哈 ad001，取最新广播，红字链接） -->
    <div class="ad" v-if="broadcast">*<a href="javascript:;" @click="$router.push('/notices')">{{ broadcast }}</a><br></div>

    <!-- 面包屑 nav01：社区 > 分区 > 版块（城市帖：社区>同城>城市，参考诺哈 topic.asp） -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;
      <template v-if="isCity"><a href="javascript:;" @click="$router.push('/tongcheng')">同城</a>&gt;</template>
      <template v-else-if="thread.board && thread.board.parent_id">
        <a href="javascript:;" @click="$router.push('/channel/'+thread.board.parent_id)">{{ parentName }}</a>&gt;
      </template>
      <a v-if="thread.board" href="javascript:;" @click="$router.push('/board/'+thread.board.id)">{{ thread.board.name }}</a><br>
    </div>

    <!-- 标题 pname -->
    <div class="name">
      <template v-if="editing"><input type="text" v-model.trim="editForm.title" maxlength="50" style="width:90%"></template>
      <template v-else>
        <span v-if="thread.is_head" style="color:#c00">[头条]</span><span v-if="thread.is_top">【顶】</span><span v-if="thread.is_fine">【精】</span><span v-if="thread.is_recom">[荐]</span><span v-if="thread.is_notice">[公告]</span><span v-if="thread.is_active" style="color:#e05a00">[活动]</span>{{ thread.title }}
      </template><br>
    </div>

    <!-- 正文（诺哈：按字数分页 / || 手动分页 / 余下全文 / 全文） -->
    <div class="title" v-if="cpages > 1 && call !== '2'">
      [内容]:<a href="javascript:;" @click="showAll">全文</a>(共{{ cpages }}页)<br>
    </div>
    <div class="text" v-html="renderLine(shownContent)"></div>
    <div class="ppage">
      <a v-if="cpage < cpages" href="javascript:;" @click="goContent(cpage+1)">下页</a><template v-if="cpage < cpages">.</template>
      <a v-if="cpage > 1" href="javascript:;" @click="goContent(cpage-1)">上页</a>
      <template v-if="cpage > 1">.</template>
      <a v-if="cpages-cpage > 1" href="javascript:;" @click="restText">余下全文</a>
      <template v-if="call === '1' || call === '2'">
        <a href="javascript:;" @click="backHome">返回帖子首页</a>
      </template><br>
    </div>
    <div class="item">
      (第<b>{{ cpage }}</b>/{{ cpages }}页/{{ wordCount }}字/{{ viewCount }}阅)<br>
      <form v-if="cpages > 1" style="display:inline" @submit.prevent="goPageInput">
        翻页：第 <input type="text" v-model.number="goInput" size="3" maxlength="6"> 页 <input type="submit" value="前往">
      </form>
    </div>

    <!-- 编辑面板 -->
    <div class="module-content" v-if="editing">
      编辑正文:<br>
      <textarea v-model="editForm.content" rows="10" style="width:98%"></textarea><br>
      <input type="submit" value="保存修改" @click.prevent="saveEdit"> <input type="submit" value="取消" @click.prevent="editing = false">
      <span v-if="editTip" :style="{color: editOk ? '#1a9e1a' : '#c00'}"> {{ editTip }}</span><br>
    </div>

    <!-- 楼主信息卡（诺哈：楼主/时间/分享/勋章/签名/关注） -->
    <div class="item">
      [楼主]:<span v-for="b in authorBadges" :key="'a'+b.id"><img class="bicon" :src="$pic(b.icon)" :alt="b.name" :title="b.name"></span>
      <img class="bicon" v-if="author.blue_lv > 0" :src="$pic('noble_2_' + author.blue_lv + '.gif')" :alt="'蓝钻' + author.blue_lv + '级'" :title="'蓝钻' + author.blue_lv + '级'" @error="hideErr">
      <img class="bicon" v-if="author.qq_lv > 0" :src="$pic('noble_1_' + author.qq_lv + '.gif')" :alt="'超Q' + author.qq_lv + '级'" :title="'超Q' + author.qq_lv + '级'" @error="hideErr">
      <img class="bicon" v-if="!hasNobleId(author) && author.noble > 0" :src="$pic('noble_' + author.noble + '_1.gif')" :alt="'贵族' + (author.noble === 1 ? '一级' : '二级')" @error="hideErr">
      <template v-if="privOf(author)"><img class="bicon" :src="'/static/' + privOf(author).file" :alt="privOf(author).name" :title="privOf(author).name"></template>
      <img class="bicon" v-else-if="author.level_icon" :src="$pic('v'+author.level_icon+'.gif')" alt="等级">
      <a href="javascript:;" @click="$router.push('/user/'+author.id)"><font :color="author.color || '#004299'">{{ author.nickname || '?' }}</font></a>
      <template v-if="isLogin && author.id && !mine">(<a href="javascript:;" @click="$router.push('/messages/'+author.id)">家信</a>)</template><br>
      [时间]:{{ fmt(thread.created_at) }}<br>
      [分享]:<a href="javascript:;" @click="shareWeibo">新浪微博</a>.<a href="javascript:;" @click="shareQzone">QQ空间</a>.<a href="javascript:;" @click="copyTopic">复制链接</a><br>
      [勋章]:<span v-if="authorBadges.length"><span v-for="b in authorBadges" :key="'m'+b.id"><img class="bicon" :src="$pic(b.icon)" :alt="b.name" :title="b.name"> </span></span><span v-else>无</span><br>
      [签名]:{{ author.signature || '（无签名）' }}<br>
      [关注]:<a href="javascript:;" @click="$router.push('/user/'+author.id)">家园</a>.<a href="javascript:;" @click="$router.push('/space/'+author.id)">空间</a>.<a href="javascript:;" @click="$router.push('/user/'+author.id)">帖子</a><br>
    </div>

    <!-- 投票（诺哈 ==投票选项==） -->
    <div class="title" v-if="poll">==投票选项==<br></div>
    <div class="item" v-if="poll" style="background:#FFF6E5">
      <b>{{ poll.question }}</b><template v-if="poll.multiple">（多选）</template><br>
      <div v-for="(o, i) in poll.options" :key="o.id">
        {{ i+1 }}. <label v-if="!pollVoted"><input type="checkbox" v-if="poll.multiple" :value="o.id" v-model="pollSel"><input type="radio" v-else name="pollopt" :value="o.id" v-model="pollSel"> {{ o.name }} ({{ o.votes }})</label>
        <template v-else>{{ o.name }} ({{ o.votes }}) <span style="display:inline-block;width:60px;background:#E3EEF8"><span style="display:block;background:#71afe3;height:8px" :style="{width: percent(o.votes) + '%'}"></span></span></template>
      </div>
      <template v-if="!pollVoted"><input type="submit" value="投" @click.prevent="doVote"></template>
      <template v-else><font color="#1a9e1a">已投票（总 {{ pollTotal }} 票）</font></template>
      <span v-if="pollMsg" style="color:#c00">{{ pollMsg }}</span><br>
    </div>

    <!-- 回帖奖励（诺哈 ==回帖奖励==） -->
    <div class="title" v-if="reward">==回帖奖励==<br></div>
    <div class="item" v-if="reward" style="background:#E6F7E6">
      回帖奖励 +{{ reward.coins }}G币、+{{ reward.exp }}经验<template v-if="reward.limit">（共 {{ reward.limit }} 次，已发 {{ reward.used }} 次）</template><br>
    </div>

    <!-- 踩楼奖励（诺哈 ==踩楼奖励==） -->
    <div class="title" v-if="floors && floors.length">==踩楼奖励==<br></div>
    <div class="item" v-if="floors && floors.length" style="background:#F0E6F7">
      <div v-for="f in floors" :key="f.id">第<b>{{ f.floor }}</b>楼: <b>+{{ f.coins }}</b>G币/{{ f.exp }}经验<template v-if="f.status">（已被{{ f.user_id }}踩中）</template></div>
      <template v-if="floors.length >= 5">>>更多踩楼<br></template>
    </div>

    <!-- 附件下载（诺哈 file.asp） -->
    <div class="title" v-if="attachments && attachments.length">==附件下载==<br></div>
    <div class="item" v-if="attachments && attachments.length" style="background:#E3EEF8">
      <div v-for="a in attachments" :key="a.id">
        <a href="javascript:;" @click="downloadAtt(a)">{{ a.type === 'image' ? '🖼' : '📎' }} {{ a.name }}</a><template v-if="a.price">（{{ a.price }}G币）</template>（{{ a.downloads }}次下载）
      </div>
    </div>

    <!-- 移动 / 分享提示 -->
    <div class="module-content" v-if="moveOpen" style="background:#E3EEF8">
      移动到：<select v-model.number="moveBoardId"><optgroup v-for="ch in channels" :key="ch.id" :label="ch.name"><option v-for="s in allSubBoards(ch)" :key="s.id" :value="s.id">{{ s.name }}</option></optgroup></select>
      <input type="submit" value="确认移动" @click.prevent="doMove"> <span v-if="moveMsg" style="color:#c00">{{ moveMsg }}</span><br>
    </div>
    <div class="module-content" v-if="shareTip" style="background:#E3EEF8">{{ shareTip }}<br></div>

    <!-- 广播（诺哈 radio 黄条） -->
    <div class="radio">家园社区欢迎你的到来！<br></div>

    <!-- 回贴列表（诺哈最后 3 楼 + 置顶回复） -->
    <div class="title">
      回贴列表:<br>
    </div>
    <div class="list">
      <div v-if="sticky" class="row01">
        【顶】<span v-for="b in (sticky.user ? sticky.user.badges : [])" :key="'s'+b.id"><img class="bicon" :src="$pic(b.icon)" :alt="b.name"></span>
        <a href="javascript:;" @click="$router.push('/user/'+(sticky.user ? sticky.user.id : ''))"><font :color="sticky.user ? sticky.user.color : ''">{{ sticky.user ? sticky.user.nickname : '?' }}</font></a><br>
        <span v-html="renderLine(sticky.content)"></span><br>
        [{{ fmt(sticky.created_at) }}]<template v-if="canSticky"> <a href="javascript:;" @click="unsticky">撤顶</a></template><br>
      </div>
      <div v-for="(r, i) in recent" :key="r.id" :class="[sticky ? (i % 2 === 1 ? 'row01' : 'row02') : (i % 2 === 0 ? 'row01' : 'row02')]">
        [{{ r.floor }}楼].<span v-for="b in (r.user ? r.user.badges : [])" :key="'u'+b.id"><img class="bicon" :src="$pic(b.icon)" :alt="b.name"></span>
        <img class="bicon" v-if="r.user && r.user.blue_lv > 0" :src="$pic('noble_2_' + r.user.blue_lv + '.gif')" :alt="'蓝钻' + r.user.blue_lv + '级'" @error="hideErr">
        <img class="bicon" v-if="r.user && r.user.qq_lv > 0" :src="$pic('noble_1_' + r.user.qq_lv + '.gif')" :alt="'超Q' + r.user.qq_lv + '级'" @error="hideErr">
        <template v-if="r.user && privOf(r.user)"><img class="bicon" :src="'/static/' + privOf(r.user).file" :alt="privOf(r.user).name"></template>
        <img class="bicon" v-else-if="r.user && r.user.level_icon" :src="$pic('v'+r.user.level_icon+'.gif')" alt="等级">
        <a href="javascript:;" @click="$router.push('/user/'+(r.user ? r.user.id : ''))"><font :color="r.user ? r.user.color : ''">{{ r.user ? r.user.nickname : '路人' }}</font></a><br>
        <span v-html="renderLine(r.content)"></span><br>
        [{{ fmt(r.created_at) }}] <a href="javascript:;" @click="quote(r)">回复</a>
        <template v-if="canSticky">.<a href="javascript:;" @click="stickyReply(r)">置顶</a></template>
        <template v-if="isLogin && r.user && user.id === r.user.id">.<a href="javascript:;" style="color:#c00" @click="delReply(r)">删除</a></template>
        <template v-if="isLogin && canManageAny && !(r.user && user.id === r.user.id)">.<a href="javascript:;" style="color:#c00" @click="delReply(r)">删除</a></template><br>
      </div>
      <div v-if="!recent.length" class="row01">还没有人回复，来抢沙发！</div>
      <div class="row01"><a href="javascript:;" @click="$router.push('/replies/'+thread.id)">全部回贴({{ total }})</a><br></div>
    </div>

    <!-- 回复区（诺哈 1-120 字 + 插入表情） -->
    <div class="title">
      <template v-if="isLogin && !thread.is_lock">回复该贴(1-120字): <a href="javascript:;" @click="faceOpen = !faceOpen">插入表情</a></template>
      <template v-else>回复:</template>
      <br>
    </div>
    <div v-if="faceOpen" class="module-content" style="background:#E3EEF8">
      <span v-for="f in faces" :key="f"><a href="javascript:;" @click="insertFace(f)" :title="f">{{ faceEmoji(f) }}</a> </span><br>
    </div>
    <div class="module-content" v-if="isLogin && !thread.is_lock">
      <form @submit.prevent="submit">
        <textarea v-model.trim="content" rows="3" maxlength="120"></textarea><br>
        <input type="submit" value="确定回复"> <span class="help-line">回帖+5经验+2G币</span>
        <template v-if="thread.type === 1"><span class="help-line"> 本贴回帖有奖励</span></template><br>
      </form>
    </div>
    <div class="radio" v-else-if="isLogin && thread.is_lock">本贴已锁，仅供查阅。<br></div>
    <div class="module-content" v-else><a href="javascript:;" @click="$router.push('/login?redirect='+$route.fullPath)">登陆家园社区</a>后回复盖楼<br></div>

    <!-- 返回贴子列表 -->
    <div class="item">
      <a href="javascript:;" @click="$router.push('/board/'+(thread.board ? thread.board.id : ''))">返回贴子列表</a><br>
    </div>

    <!-- 贴子管理（诺哈 topic_func：收藏/复制/推荐/管理 + 版主操作） -->
    <div class="item">
      [贴子管理]:<a href="javascript:;" @click="toggleFav"><font :color="favored ? '#1a9e1a' : '#004299'">{{ favored ? '★已收藏' : '收藏' }}</font></a>.<a href="javascript:;" @click="copyTopic">复制</a>.<a href="javascript:;" @click="recommend">推荐</a>.<a href="javascript:;" @click="manageOpen = !manageOpen">管理</a>
      <template v-if="canManageAny">
        .<a href="javascript:;" @click="toggle('is_fine')">{{ thread.is_fine ? '撤精' : '加精' }}</a>
        .<a href="javascript:;" @click="toggle('is_lock')">{{ thread.is_lock ? '解锁' : '锁定' }}</a>
        .<a href="javascript:;" @click="delThread" style="color:#c00">删除</a>
      </template><br>
    </div>

    <!-- 管理面板 -->
    <div class="module-content" v-if="manageOpen" style="background:#E3EEF8">
      [更多管理]<br>
      <template v-if="canManageAny">
        <a href="javascript:;" @click="toggle('is_top')">{{ thread.is_top ? '取消置顶' : '置顶' }}</a>.
        <a href="javascript:;" @click="toggle('is_head')">{{ thread.is_head ? '取消头条' : '头条' }}</a>.
        <a href="javascript:;" @click="toggle('is_recom')">{{ thread.is_recom ? '取消推荐' : '推荐' }}</a>.
        <a href="javascript:;" @click="toggle('is_notice')">{{ thread.is_notice ? '取消公告' : '公告' }}</a>.
        <a href="javascript:;" @click="toggle('is_active')">{{ thread.is_active ? '取消活动' : '设活动' }}</a>.
        <a href="javascript:;" @click="openMove">移动</a>.
        <a href="javascript:;" @click="startEdit">{{ editing ? '取消编辑' : '编辑' }}</a>
        <template v-if="thread.audit_status !== 1">.<a href="javascript:;" @click="audit(true)">审核通过</a>|<a href="javascript:;" @click="audit(false)">拒绝</a></template>
        <br>
      </template>
      <a v-if="mine && !canManageAny" href="javascript:;" @click="startEdit">{{ editing ? '取消编辑' : '编辑' }}</a>
      <template v-if="mine && canManageAny">.<a href="javascript:;" style="color:#c00" @click="delThread">删除</a></template>
    </div>

    <!-- 面包屑重复 -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;
      <template v-if="isCity"><a href="javascript:;" @click="$router.push('/tongcheng')">同城</a>&gt;</template>
      <template v-else-if="thread.board && thread.board.parent_id">
        <a href="javascript:;" @click="$router.push('/channel/'+thread.board.parent_id)">{{ parentName }}</a>&gt;
      </template>
      <a v-if="thread.board" href="javascript:;" @click="$router.push('/board/'+thread.board.id)">{{ thread.board.name }}</a><br>
    </div>
  </div>
</template>

<script>
import api from '../api'
import { renderFace, FACE_NAMES } from '../utils/qqface'

const PAGE_CHARS = 1500

export default {
  name: 'Thread',
  data () {
    return {
      thread: { user: {}, board: {} },
      recent: [], sticky: null, canSticky: false, total: 0,
      poll: null, pollVoted: false, pollTotal: 0, myOptions: [], pollSel: [], pollMsg: '',
      reward: null, floors: [], attachments: [],
      favored: false, broadcast: '',
      content: '', sending: false, faceOpen: false,
      editing: false, editForm: { title: '', content: '' }, editTip: '', editOk: false,
      channels: [], moveOpen: false, moveBoardId: 0, moveMsg: '', manageOpen: false,
      quoteReplyId: 0, shareTip: '', goInput: 1
    }
  },
  computed: {
    contentFace () { return renderFace(this.shownContent) },
    faces () { return FACE_NAMES },
    parentName () {
      const b = this.thread.board
      if (b && b.parent_name) return b.parent_name
      return '论坛'
    },
    isCity () { return !!(this.thread.board && this.thread.board.city_code) },
    isLogin () { return this.$store.getters.isLogin },
    user () { return this.$store.state.user },
    author () { return this.thread.user || {} },
    authorBadges () { return this.author.badges || [] },
    mine () { return this.user && this.author && this.user.id === this.author.id },
    canManage () { return this.user && this.user.perms && this.user.perms.indexOf('thread:manage') >= 0 },
    canMod () {
      return this.isLogin && this.thread.board && this.user && this.thread.board.moderator_id === this.user.id
    },
    canManageAny () { return !!this.canManage || !!this.canMod },
    contentParts () {
      const c = this.thread.content || ''
      if (c.indexOf('||') >= 0) return c.split('||')
      const arr = []
      for (let i = 0; i < c.length; i += PAGE_CHARS) arr.push(c.slice(i, i + PAGE_CHARS))
      return arr.length ? arr : ['']
    },
    cpages () { return this.contentParts.length },
    cpage () {
      return Math.min(Math.max(parseInt(this.$route.query.page || 1), 1), this.cpages)
    },
    call () { return this.$route.query.all || '' },
    shownContent () {
      const parts = this.contentParts
      if (this.call === '2') return parts.join('\n')
      if (this.call === '1') return parts.slice(this.cpage).join('\n')
      return parts[this.cpage - 1] || ''
    },
    wordCount () { return (this.thread.content || '').length },
    viewCount () { return (this.thread.view_count || 0) + 1 }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    renderLine (t) { return renderFace(t || '') },
    load () {
      const id = this.$route.params.id
      api.get(`/threads/${id}`).then(r => {
        if (r.code !== 0) { alert(r.msg); return }
        this.thread = r.data.thread
        this.recent = r.data.recent_replies || []
        this.sticky = r.data.sticky_reply || null
        this.canSticky = !!r.data.can_sticky
        this.total = r.data.total || 0
        this.poll = r.data.poll || null
        this.pollVoted = r.data.poll_voted || false
        this.pollTotal = r.data.poll_total || 0
        this.myOptions = r.data.my_options || []
        this.pollSel = this.poll ? this.poll.options.filter(o => this.myOptions.indexOf(o.id) >= 0).map(o => o.id) : []
        this.reward = r.data.reward || null
        this.floors = r.data.floors || []
        this.attachments = r.data.attachments || []
        this.quoteReplyId = 0
      })
      if (this.isLogin) {
        api.get(`/threads/${id}/favorite-status`).then(r => { if (r.code === 0) this.favored = r.data.favored })
      }
      api.get('/plaza').then(r => {
        if (r.code !== 0) return
        const b = (r.data.broadcasts || [])[0]
        this.broadcast = b ? (b.title || b.content || '') : ''
      })
    },
    // ---- 内容分页 ----
    goContent (p) {
      this.$router.push('/thread/' + this.thread.id + '?page=' + p).catch(() => {})
    },
    goPageInput () {
      const p = parseInt(this.goInput) || 1
      this.goContent(p)
    },
    showAll () {
      this.$router.push('/thread/' + this.thread.id + '?all=2').catch(() => {})
    },
    restText () {
      this.$router.push('/thread/' + this.thread.id + '?page=' + this.cpage + '&all=1').catch(() => {})
    },
    backHome () {
      this.$router.push('/thread/' + this.thread.id).catch(() => {})
    },
    // ---- 回帖 ----
    quote (r) {
      this.content = '回复' + r.floor + '楼：'
      this.quoteReplyId = r.id
      window.scrollTo(0, 99999)
    },
    submit () {
      if (!this.content) return
      this.sending = true
      api.post(`/threads/${this.thread.id}/replies`, { content: this.content, parent_reply_id: this.quoteReplyId || 0 }).then(r => {
        this.sending = false
        if (r.code === 0) { this.content = ''; this.quoteReplyId = 0; this.load() }
        else alert(r.msg)
      })
    },
    stickyReply (r) {
      api.post(`/threads/${this.thread.id}/sticky-reply`, { reply_id: r.id }).then(r => {
        if (r.code === 0) { alert('置顶成功'); this.load() } else alert(r.msg)
      })
    },
    unsticky () {
      if (!confirm('确定撤下这条置顶回复吗？')) return
      api.delete(`/threads/${this.thread.id}/sticky-reply`).then(r => {
        if (r.code === 0) { alert('已撤顶'); this.load() } else alert(r.msg)
      })
    },
    delReply (r) {
      if (!confirm('确定删除这条回复吗？')) return
      api.delete(`/replies/${r.id}`).then(x => {
        if (x.code === 0) this.load()
        else alert(x.msg)
      })
    },
    downloadAtt (a) {
      if (!this.isLogin) { alert('请先登录'); return }
      api.get(`/attachments/${a.id}/download`).then(r => {
        if (r.code === 0) {
          alert(a.price ? '下载成功，已扣除 ' + a.price + ' G币' : '下载成功')
          this.load()
        } else alert(r.msg)
      })
    },
    // ---- 投票 ----
    doVote () {
      if (!this.pollSel.length) { this.pollMsg = '请选择选项'; return }
      const optionIds = Array.isArray(this.pollSel) ? this.pollSel : [this.pollSel]
      api.post(`/threads/${this.thread.id}/poll-vote`, { option_ids: optionIds }).then(r => {
        if (r.code === 0) { alert('投票成功！'); this.load() }
        else { this.pollMsg = r.msg; alert(r.msg) }
      })
    },
    percent (v) {
      const max = Math.max(...(this.poll ? this.poll.options.map(o => o.votes) : [1]), 1)
      return Math.round((v / max) * 100)
    },
    // ---- 贴子管理 ----
    toggleFav () {
      api.post(`/threads/${this.thread.id}/favorite`).then(r => { if (r.code === 0) this.favored = r.data.favored })
    },
    copyTopic () {
      const url = location.origin + location.pathname + '#/thread/' + this.thread.id
      const text = '《' + this.thread.title + '》\n' + (this.thread.content || '').replace(/\s+/g, ' ').slice(0, 200) + '……\n' + url
      this.copyText(text)
    },
    copyText (txt) {
      const ta = document.createElement('textarea')
      ta.value = txt
      document.body.appendChild(ta)
      ta.select()
      try { document.execCommand('copy') } catch (e) {}
      document.body.removeChild(ta)
      this.shareTip = '内容已复制：' + url && location.href
      setTimeout(() => { this.shareTip = '' }, 3000)
    },
    recommend () {
      const to = prompt('输入你要推荐的好友家园号码：')
      if (!to) return
      const url = location.origin + location.pathname + '#/thread/' + this.thread.id
      const content = '向你推荐帖子《' + this.thread.title + '》：' + url
      api.post('/messages', { to: parseInt(to), content }).then(r => {
        if (r.code === 0) alert('已推荐给好友 ' + to)
        else alert(r.msg)
      })
    },
    toggle (field) {
      const v = this.thread[field] ? 0 : 1
      api.post(`/threads/${this.thread.id}/manage`, { field, value: v }).then(r => {
        if (r.code === 0) this.thread[field] = v
        else alert(r.msg)
      })
    },
    audit (approve) {
      if (!confirm(approve ? '确定通过审核？' : '确定拒绝该帖？')) return
      api.post(`/threads/${this.thread.id}/audit`, { approve }).then(r => {
        if (r.code === 0) { this.thread.audit_status = approve ? 1 : 2; alert(approve ? '已通过审核' : '已拒绝') }
        else alert(r.msg)
      })
    },
    startEdit () {
      if (this.editing) { this.editing = false; return }
      this.editForm = { title: this.thread.title, content: this.thread.content }
      this.editTip = ''
      this.editing = true
    },
    saveEdit () {
      if (!this.editForm.title || !this.editForm.content) { this.editTip = '标题和内容不能为空'; this.editOk = false; return }
      api.put(`/threads/${this.thread.id}`, { title: this.editForm.title, content: this.editForm.content }).then(r => {
        if (r.code === 0) {
          this.editTip = '已保存'; this.editOk = true; this.editing = false; this.load()
        } else { this.editTip = r.msg; this.editOk = false }
      })
    },
    delThread () {
      if (!confirm('确定删除这篇帖子吗？')) return
      const back = () => { this.$router.push(this.thread.board ? '/board/' + this.thread.board.id : '/') }
      api.delete(`/threads/${this.thread.id}`).then(r => {
        if (r.code === 0) { alert('已删除'); back() }
        else {
          api.delete(`/admin/threads/${this.thread.id}`).then(d => {
            if (d.code === 0) { alert('已删除'); back() } else alert(d.msg)
          })
        }
      })
    },
    openMove () {
      if (!this.channels.length) api.get('/boards').then(r => { if (r.code === 0) this.channels = r.data })
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
    // ---- 分享 / 表情 / 工具 ----
    faceEmoji (name) {
      const m = { 微笑: '🙂', 撇嘴: '😖', 色: '😍', 发呆: '😲', 得意: '😏', 流泪: '😢', 害羞: '😳', 闭嘴: '🤐', 睡: '😴', 大哭: '😭', 尴尬: '😬', 发怒: '😡', 调皮: '😜', 呲牙: '😁', 惊讶: '😮', 难过: '😞', 酷: '😎', 冷汗: '😓', 抓狂: '🤯', 吐: '🤮', 偷笑: '🤭', 可爱: '🥰', 白眼: '🙄', 傲慢: '😠', 饥饿: '😋', 困: '😪', 惊恐: '😱', 流汗: '😅', 憨笑: '🤣', 大兵: '💂', 奋斗: '💪', 咒骂: '🤬', 疑问: '❓', 嘘: '🤫', 晕: '😵', 再见: '👋', 擦汗: '🥵', 鼓掌: '👏', 委屈: '🙇', 亲亲: '😘', 可怜: '🥺', 玫瑰: '🌹', 爱心: '❤️', 心碎: '💔', 蛋糕: '🎂', 音乐: '🎵' }
      return m[name] || ''
    },
    insertFace (name) {
      this.content += '/' + name
    },
    shareWeibo () {
      window.open('http://service.weibo.com/share/share.php?url=' + encodeURIComponent(location.href))
    },
    shareQzone () {
      const url = encodeURIComponent(location.href)
      const title = encodeURIComponent(this.thread.title || '')
      window.open('http://sns.qzone.qq.com/cgi-bin/qzshare/cgi_qzshare_onekey?url=' + url + '&title=' + title)
    },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t)
      const p = n => (n < 10 ? '0' + n : n)
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds())
    },
    hasNobleId (u) { return !!u && (u.blue_lv > 0 || u.qq_lv > 0) },
    privOf (u) {
      const p = u && u.priv
      if (!p) return null
      const n = p.name || ''
      if (this.hasNobleId(u) && (n.indexOf('蓝钻') === 0 || n.indexOf('超Q') === 0)) return null
      return p
    },
    hideErr (e) { e.target.style.display = 'none' }
  }
}
</script>