<template>
  <div>
    <div class="name"><a href="javascript:;" @click="$router.push('/space/'+article.user_id)">日志</a><br></div>
    <div v-if="article" class="module-content">
      <b>{{ article.title }}</b><br>
      <span class="txt-fade">{{ fmt(article.created_at) }} {{ article.atype === 1 ? '[私密]' : '' }}</span><br>
      ----------<br>
      <div style="white-space:pre-wrap">{{ article.content }}</div>
      <br>
      <a href="javascript:;" v-if="isOwner" @click="delArticle">[删除]</a><br>
    </div>
    <span v-else class="empty">日志不存在或已删除</span>

    <div class="module-title">评论({{ comments.length }})</div>
    <div class="list" v-if="comments.length">
      <div v-for="cm in comments" :key="cm.id" class="row">
        <a href="javascript:;" @click="$router.push('/user/'+cm.user_id)"><font :color="cm.color || '#004299'">{{ cm.nickname }}</font></a>:{{ cm.content }} <span class="txt-fade">({{ fmt(cm.created_at) }})</span><br>
      </div>
    </div>
    <span v-else class="empty">还没有评论</span>

    <form v-if="isLogin" @submit.prevent="addComment">
      <input type="text" v-model.trim="commentText" maxlength="300" size="20" placeholder="说点什么…" />
      <input type="submit" value="评论" />
    </form>
    <p v-if="tip" style="color:#e05a00;padding:3px 5px">{{ tip }}</p>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'BlogArticle',
  data () {
    return { article: null, comments: [], commentText: '', tip: '' }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    user () { return this.$store.state.user || {} },
    isOwner () { return this.article && this.article.user_id === this.user.id }
  },
  mounted () { this.load() },
  methods: {
    load () {
      const id = this.$route.params.id
      api.get('/space/article/' + id).then(r => {
        if (r.code === 0) this.article = r.data
      }).catch(() => {})
      api.get('/space/article/' + id + '/comments').then(r => { if (r.code === 0) this.comments = r.data }).catch(() => {})
    },
    addComment () {
      if (!this.commentText) return
      api.post('/space/article/' + this.$route.params.id + '/comment', { content: this.commentText }).then(r => {
        if (r.code === 0) { this.commentText = ''; this.load() }
      })
    },
    delArticle () {
      const uid = this.article.user_id
      api.delete('/space/article/' + this.$route.params.id).then(() => {
        this.tip = '已删除'; this.article = null
        this.$router.push('/space/' + uid)
      })
    },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t); const p = n => (n < 10 ? '0' + n : '' + n)
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes())
    }
  }
}
</script>