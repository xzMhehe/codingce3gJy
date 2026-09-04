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
    <div class="item">
      <a v-if="page > 1" href="javascript:;" @click="go(page-1)">上页</a><template v-if="page > 1">.</template><a v-if="page < pages" href="javascript:;" @click="go(page+1)">下页</a><template v-if="page < pages">.</template>
      <form style="display:inline" @submit.prevent="go(pageInput)">
        第<input type="text" v-model.number="pageInput" size="2" maxlength="4">页
        <input type="submit" value="前往">
      </form>
      (第<b>{{ page }}</b>/{{ pages }}页/{{ thread.view_count }}阅)<br>
    </div>
    <div class="item">
      [楼主]:<span v-for="b in authorBadges" :key="b.id"><img class="bicon" :src="$pic(b.icon)" :alt="b.name"></span><img class="bicon" v-if="author.priv" :src="'/static/' + author.priv.file" :alt="author.priv.name" :title="author.priv.name"><img class="bicon" v-else-if="author.level_icon" :src="$pic('v'+author.level_icon+'.gif')" alt="等级"><a href="javascript:;" @click="$router.push('/user/'+author.id)"><font :color="author.color">{{ author.nickname }}</font>（{{ author.username }}）</a><br>
      [发帖时间]:{{ fmt(thread.created_at) }}<br>
      <a v-if="isLogin" href="javascript:;" @click="quote(1)">回复</a><template v-if="isLogin">.</template><a href="javascript:;" @click="$router.push('/user/'+author.id)">TA的主页</a><br>
    </div>

    <div class="list">
      <div class="row" v-for="r in replies" :key="r.id">
        {{ r.floor }}楼.{{ r.content }} (<span v-for="b in (r.user?r.user.badges:[])" :key="b.id"><img class="bicon" :src="$pic(b.icon)" :alt="b.name"></span><img class="bicon" v-if="r.user && r.user.priv" :src="'/static/' + r.user.priv.file" :alt="r.user.priv.name" :title="r.user.priv.name" v-else-if="r.user && r.user.level_icon" :src="$pic('v'+r.user.level_icon+'.gif')" alt="等级"><a href="javascript:;" @click="$router.push('/user/'+(r.user?r.user.id:''))"><font :color="r.user?r.user.color:''">{{ r.user?r.user.nickname:'路人' }}</font></a>){{ fmt(r.created_at) }} <a v-if="isLogin" href="javascript:;" @click="quote(r.floor)">回复</a><a v-if="user && r.user && r.user.id === user.id" href="javascript:;" style="color:#c00" @click="delReply(r)">.删除</a><br>
      </div>
    </div>
    <div v-if="!replies.length" class="empty">还没有人回复，来抢沙发！</div>

    <div class="item">
      <a v-if="page > 1" href="javascript:;" @click="go(page-1)">上页</a><template v-if="page > 1">.</template><a v-if="page < pages" href="javascript:;" @click="go(page+1)">下页</a><template v-if="page < pages">.</template>(第<b>{{ page }}</b>/{{ pages }}页)
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
    return { thread: { user: {} }, replies: [], total: 0, page: 1, pages: 1, pageInput: 1, content: '', sending: false, editing: false, editForm: { title: '', content: '' }, editTip: '', editOk: false, saving: false, favored: false }
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
        } else {
          alert(r.msg)
        }
      })
      if (this.isLogin) {
        api.get(`/threads/${id}/favorite-status`).then(r => { if (r.code === 0) this.favored = r.data.favored })
      }
    },
    toggleFav () {
      api.post(`/threads/${this.thread.id}/favorite`).then(r => { if (r.code === 0) this.favored = r.data.favored })
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
