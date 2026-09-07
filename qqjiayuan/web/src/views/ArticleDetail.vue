<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/articles')">文章</a>&gt;正文</div>
    <div class="name">{{ a.title }}<br></div>
    <div v-if="a" class="module-content">
      <span class="txt-fade">作者:{{ a.author }} · 来源:{{ a.source || '家园社区' }} · {{ fmt(a.created_at) }} · {{ a.click }}阅</span><br>
      ----------<br>
      <div style="white-space:pre-wrap">{{ a.content }}</div>
      <br><a href="javascript:;" v-if="isOwner" @click="del">[删除]</a>
    </div>
    <span v-else class="empty">文章不存在</span>

    <div class="module-title">评论({{ comments.length }})</div>
    <div class="list" v-if="comments.length">
      <div v-for="cm in comments" :key="cm.id" class="row">
        <a href="javascript:;" @click="$router.push('/user/'+cm.user_id)"><font :color="cm.color || '#004299'">{{ cm.nickname }}</font></a>:{{ cm.content }} <span class="txt-fade">({{ fmt(cm.created_at) }})</span><br>
      </div>
    </div>
    <span v-else class="empty">还没有评论</span>

    <form v-if="isLogin" @submit.prevent="addComment">
      <input type="text" v-model.trim="commentText" maxlength="300" size="20" placeholder="发表评论" />
      <input type="submit" value="评论" />
    </form>
    <p v-if="tip" style="color:#e05a00;padding:3px 5px">{{ tip }}</p>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'ArticleDetail',
  data () { return { a: null, comments: [], commentText: '', tip: '' } },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    user () { return this.$store.state.user || {} },
    isOwner () { return this.a && this.a.author_id === this.user.id }
  },
  mounted () { this.load() },
  methods: {
    load () {
      const id = this.$route.params.id
      api.get('/articles/' + id).then(r => { if (r.code === 0) this.a = r.data }).catch(() => {})
      api.get('/articles/' + id + '/comments').then(r => { if (r.code === 0) this.comments = r.data }).catch(() => {})
    },
    addComment () {
      if (!this.commentText) return
      api.post('/articles/' + this.$route.params.id + '/comments', { content: this.commentText }).then(r => {
        if (r.code === 0) { this.commentText = ''; this.load() }
        else this.tip = r.msg || '评论失败'
      })
    },
    del () { api.delete('/articles/' + this.$route.params.id).then(() => this.$router.push('/articles')) },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t); const p = n => (n < 10 ? '0' + n : '' + n)
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate())
    }
  }
}
</script>