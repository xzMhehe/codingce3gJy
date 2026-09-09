<template>
  <div>
    <!-- 面包屑 nav01：社区 > 版块 -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;
      <template v-if="board.parent_id"><a href="javascript:;" @click="$router.push('/channel/'+board.parent_id)">{{ boardName }}</a>&gt;</template>
      <a href="javascript:;" @click="$router.push('/board/'+board.id)">{{ board.name }}</a><br>
    </div>

    <!-- 主题 -->
    <div class="name">
      [主题]<a href="javascript:;" @click="$router.push('/thread/'+tid)">{{ title }}</a><br>
    </div>
    <div class="title">
      回复列表：<a href="javascript:;" @click="reload">刷新</a><br>
    </div>

    <!-- 置顶回复（诺哈 wap_topic_reply_apex） -->
    <div class="list">
      <div v-if="sticky" class="row01">
        【顶】<span v-for="b in (sticky.user ? sticky.user.badges : [])" :key="'s'+b.id"><img class="bicon" :src="$pic(b.icon)" :alt="b.name"></span>
        <a href="javascript:;" @click="$router.push('/user/'+(sticky.user ? sticky.user.id : ''))"><font :color="sticky.user ? sticky.user.color : ''">{{ sticky.user ? sticky.user.nickname : '?' }}</font></a><br>
        <span v-html="renderLine(sticky.content)"></span><br>
        [{{ fmt(sticky.created_at) }}]<template v-if="canSticky"> <a href="javascript:;" @click="unsticky">撤顶</a></template><br>
      </div>

      <!-- 回复列表（最新在前） -->
      <div v-for="(r, i) in replies" :key="r.id" :class="[sticky ? (i % 2 === 1 ? 'row01' : 'row02') : (i % 2 === 0 ? 'row01' : 'row02')]">
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
        <template v-if="canManageAny && !(r.user && user.id === r.user.id)">.<a href="javascript:;" style="color:#c00" @click="delReply(r)">删除</a></template><br>
      </div>
      <div v-if="!replies.length" class="row01">暂无回复！<br></div>
    </div>

    <!-- 分页 -->
    <div class="ppage">
      <a v-if="page > 1" href="javascript:;" @click="go(page-1)">上页</a><template v-if="page > 1">.</template>
      <a v-if="page < pages" href="javascript:;" @click="go(page+1)">下页</a><template v-if="page < pages">.</template>
      <br v-if="pages > 1">
    </div>
    <div class="item">
      (第<b>{{ page }}</b>/{{ pages }}页/共{{ total }}条记录)<br>
      <form v-if="pages > 1" style="display:inline" @submit.prevent="go(pageInput)">
        翻页：第 <input type="text" v-model.number="pageInput" size="3" maxlength="6"> 页 <input type="submit" value="前往">
      </form>
    </div>

    <!-- 回复框 / 锁定 -->
    <div class="title">
      <template v-if="isLogin && !isLock">回复该贴(1-120字): <a href="javascript:;" @click="faceOpen = !faceOpen">插入表情</a></template>
      <template v-else>回复:</template>
      <br>
    </div>
    <div v-if="faceOpen" class="module-content" style="background:#E3EEF8">
      <span v-for="f in faces" :key="f"><a href="javascript:;" @click="insertFace(f)" :title="f">{{ faceEmoji(f) }}</a> </span><br>
    </div>
    <div class="module-content" v-if="isLogin && !isLock">
      <form @submit.prevent="submit">
        <textarea v-model.trim="content" rows="3" maxlength="120"></textarea><br>
        <input type="submit" value="确定回复"><br>
      </form>
    </div>
    <div class="radio" v-else-if="isLogin && isLock">本贴已锁，仅供查阅。<br></div>
    <div class="module-content" v-else><a href="javascript:;" @click="$router.push('/login?redirect='+$route.fullPath)">登陆家园社区</a>后回复盖楼<br></div>

    <!-- 返回 -->
    <div class="item">
      <a href="javascript:;" @click="$router.push('/thread/'+tid)">返回[主题]</a><br>
      <a href="javascript:;" @click="$router.push('/board/'+board.id)">返回[帖子列表]</a><br>
    </div>

    <!-- 面包屑重复 -->
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区</a>&gt;
      <template v-if="board.parent_id"><a href="javascript:;" @click="$router.push('/channel/'+board.parent_id)">{{ boardName }}</a>&gt;</template>
      <a href="javascript:;" @click="$router.push('/board/'+board.id)">{{ board.name }}</a><br>
    </div>
  </div>
</template>

<script>
import api from '../api'
import { renderFace, FACE_NAMES } from '../utils/qqface'

export default {
  name: 'ReplyList',
  data () {
    return {
      tid: 0, title: '', isLock: 0, board: { name: '' },
      replies: [], sticky: null, canSticky: false, canManageAny: false,
      total: 0, page: 1, pages: 1, pageInput: 1,
      content: '', faceOpen: false
    }
  },
  computed: {
    faces () { return FACE_NAMES },
    isLogin () { return this.$store.getters.isLogin },
    user () { return this.$store.state.user },
    boardName () { return this.board.parent_name || '论坛' }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    renderLine (t) { return renderFace(t || '') },
    load () {
      this.tid = this.$route.params.id
      this.loadData()
    },
    loadData () {
      const p = parseInt(this.$route.query.page || 1)
      api.get(`/threads/${this.tid}/replies`, { params: { page: p } }).then(r => {
        if (r.code !== 0) { alert(r.msg); return }
        this.title = r.data.thread_title || ''
        this.isLock = r.data.is_lock || 0
        this.replies = r.data.replies || []
        this.sticky = r.data.sticky_reply || null
        this.total = r.data.total || 0
        this.page = r.data.page || 1
        this.pageInput = this.page
        this.pages = Math.max(1, Math.ceil(this.total / (r.data.size || 10)))
        // 版块信息（面包屑）
        api.get('/boards/' + r.data.board_id).then(x => {
          if (x.code === 0) this.board = x.data.board || x.data
        }).catch(() => {})
      })
      if (this.isLogin) {
        api.get(`/auth/me`).then(r => {
          if (r.code === 0) {
            const perms = r.data.perms || []
            this.canManageAny = perms.indexOf('thread:manage') >= 0
            // can_sticky：从详情接口取
            api.get(`/threads/${this.tid}`).then(d => {
              if (d.code === 0) this.canSticky = !!d.data.can_sticky
            })
          }
        })
      }
    },
    reload () { this.loadData() },
    go (p) {
      if (p < 1) p = 1
      if (p > this.pages) p = this.pages
      if (p === this.page) { this.loadData(); return }
      this.$router.push('/replies/' + this.tid + '?page=' + p).catch(() => {})
    },
    quote (r) {
      this.content = '回复' + r.floor + '楼：'
      window.scrollTo(0, 99999)
    },
    submit () {
      if (!this.content) return
      api.post(`/threads/${this.tid}/replies`, { content: this.content }).then(r => {
        if (r.code === 0) { this.content = ''; this.go(1) }
        else alert(r.msg)
      })
    },
    stickyReply (r) {
      api.post(`/threads/${this.tid}/sticky-reply`, { reply_id: r.id }).then(r => {
        if (r.code === 0) { alert('置顶成功'); this.reload() } else alert(r.msg)
      })
    },
    unsticky () {
      if (!confirm('确定撤下这条置顶回复吗？')) return
      api.delete(`/threads/${this.tid}/sticky-reply`).then(r => {
        if (r.code === 0) { alert('已撤顶'); this.reload() } else alert(r.msg)
      })
    },
    delReply (r) {
      if (!confirm('确定删除这条回复吗？')) return
      api.delete(`/replies/${r.id}`).then(x => {
        if (x.code === 0) this.reload()
        else alert(x.msg)
      })
    },
    faceEmoji (name) {
      const m = { 微笑: '🙂', 撇嘴: '😖', 色: '😍', 发呆: '😲', 得意: '😏', 流泪: '😢', 害羞: '😳', 闭嘴: '🤐', 睡: '😴', 大哭: '😭', 尴尬: '😬', 发怒: '😡', 调皮: '😜', 呲牙: '😁', 惊讶: '😮', 难过: '😞', 酷: '😎', 冷汗: '😓', 抓狂: '🤯', 吐: '🤮', 偷笑: '🤭', 可爱: '🥰', 白眼: '🙄', 傲慢: '😠', 饥饿: '😋', 困: '😪', 惊恐: '😱', 流汗: '😅', 憨笑: '🤣', 大兵: '💂', 奋斗: '💪', 咒骂: '🤬', 疑问: '❓', 嘘: '🤫', 晕: '😵', 再见: '👋', 擦汗: '🥵', 鼓掌: '👏', 委屈: '🙇', 亲亲: '😘', 可怜: '🥺', 玫瑰: '🌹', 爱心: '❤️', 心碎: '💔', 蛋糕: '🎂', 音乐: '🎵' }
      return m[name] || ''
    },
    insertFace (name) {
      this.content += '/' + name
    },
    privOf (u) {
      const p = u && u.priv
      if (!p) return null
      return p
    },
    hideErr (e) { e.target.style.display = 'none' },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t)
      const p = n => (n < 10 ? '0' + n : n)
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds())
    }
  }
}
</script>