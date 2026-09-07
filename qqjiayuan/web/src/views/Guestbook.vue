<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;留言本</div>
    <div class="name">留言本<br></div>
    <div class="module-content txt-fade">欢迎在留言本留下你的足迹，支持私密留言（设置密码后仅站长和留言者可查看）。</div>

    <form @submit.prevent="add">
      留言人:<input type="text" v-if="!isLogin" v-model.trim="form.name" maxlength="20" size="10" />
      <span v-else class="txt-fade">{{ myNick }}</span><br>
      <textarea v-model.trim="form.content" rows="3" maxlength="500"></textarea><br>
      <label><input type="checkbox" v-model="form.private" />私密留言</label>
      <input v-if="form.private" type="password" v-model.trim="form.pass" maxlength="20" size="10" placeholder="留言密码" />
      <br><input type="submit" value="留言" />
    </form>
    <p v-if="tip" style="color:#e05a00;padding:3px 5px">{{ tip }}</p>

    <div class="module-title">留言列表(共{{ total }}条)</div>
    <div class="list">
      <div v-for="g in list" :key="g.id" class="row">
        <a href="javascript:;" v-if="g.user_id" @click="$router.push('/user/'+g.user_id)">{{ g.nickname || g.name }}</a>
        <span v-else>{{ g.name }}</span>:
        <template v-if="g.locked">
          <span class="txt-fade">【私密留言】</span>
          <a href="javascript:;" @click="unlock(g)">查看</a>
          <input v-if="g._unlock" type="password" v-model="g._pass" size="8" placeholder="密码" @keyup.enter="doUnlock(g)" />
        </template>
        <template v-else>{{ g.content }}</template>
        <span class="txt-fade">({{ fmt(g.created_at) }})</span>
        <a href="javascript:;" v-if="isAdmin" @click="showReply(g)">[回复]</a>
        <a href="javascript:;" v-if="isAdmin || (isLogin && g.user_id === myId)" @click="del(g.id)">[删除]</a><br>
        <div v-if="g.replies.length" class="txt-fade" style="padding-left:12px">
          <div v-for="r in g.replies" :key="r.id">↳ 站长:{{ r.content }} ({{ fmt(r.created_at) }})</div>
        </div>
        <form v-if="g._showReply" @submit.prevent="doReply(g)">
          <input type="text" v-model.trim="g._replyText" maxlength="300" size="20" placeholder="站长回复" />
          <input type="submit" value="回复" />
        </form>
      </div>
    </div>

    <div v-if="totalPages > 1" class="pager">
      (第<b>{{ page }}</b>/{{ totalPages }}页/共{{ total }}条记录)
      <a href="javascript:;" v-if="page > 1" @click="go(page-1)">&lt;&lt;上一页</a>
      <a href="javascript:;" v-if="page < totalPages" @click="go(page+1)">下一页&gt;&gt;</a>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Guestbook',
  data () {
    return {
      list: [], page: 1, total: 0, totalPages: 1, tip: '',
      form: { name: '', content: '', pass: '', private: false }
    }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    user () { return this.$store.state.user || {} },
    myId () { return this.user.id },
    myNick () { return this.user.nickname },
    isAdmin () { return this.user.perms && this.user.perms.indexOf('admin:access') >= 0 }
  },
  mounted () { this.load() },
  methods: {
    load (page) {
      this.page = page || this.page
      const unlocked = this.list.filter(g => g._unlocked).map(g => g.id).join(',')
      api.get('/guestbook', { params: { page: this.page, unlocked } }).then(r => {
        if (r.code === 0) {
          this.list = (r.data.list || []).map(g => ({ ...g, _unlocked: false, _pass: '', _showReply: false, _replyText: '' }))
          this.total = r.data.total
          this.totalPages = Math.ceil(r.data.total / r.data.size) || 1
        }
      })
    },
    go (p) { this.load(p) },
    add () {
      if (!this.form.content) { this.tip = '留言内容不能为空'; return }
      api.post('/guestbook', {
        name: this.form.name, pass: this.form.private ? this.form.pass : '',
        content: this.form.content
      }).then(r => {
        if (r.code === 0) { this.form = { name: '', content: '', pass: '', private: false }; this.tip = '留言成功'; this.load(1) }
        else this.tip = r.msg || '留言失败'
      }).catch(() => { this.tip = '留言失败，请稍后再试' })
    },
    unlock (g) { this.$set(g, '_unlocked', true) },
    doUnlock (g) {
      api.post('/guestbook/' + g.id + '/unlock', { pass: g._pass }).then(r => {
        if (r.code === 0) { g.content = r.data.content; g.locked = false; this.tip = '' }
        else this.tip = r.msg || '密码不对'
      })
    },
    showReply (g) { this.$set(g, '_showReply', !g._showReply) },
    doReply (g) {
      if (!g._replyText) return
      api.post('/guestbook/' + g.id + '/reply', { content: g._replyText }).then(r => {
        if (r.code === 0) { this.tip = '回复成功'; this.load(this.page) }
      })
    },
    del (id) { api.delete('/guestbook/' + id).then(() => this.load(this.page)) },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t); const p = n => (n < 10 ? '0' + n : '' + n)
      return d.getMonth() + 1 + '/' + d.getDate() + ' ' + p(d.getHours()) + ':' + p(d.getMinutes())
    }
  }
}
</script>