<template>
  <div>
    <div class="bar">【社区搜索】</div>
    <div class="module-content">
      <form @submit.prevent="go">
        <input type="text" v-model.trim="word" maxlength="30" style="width:60%;padding:5px;border:1px solid #9FC6EC">
        <button class="btn small" type="submit">搜索</button>
      </form>
    </div>
    <template v-if="done">
      <div class="module-title">【找到的帖子】({{ threads.length }})</div>
      <ul class="dtuser" v-if="threads.length">
        <li v-for="t in threads" :key="t.id">
          <a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a>
          <em>（{{ t.board ? t.board.name : '' }} · {{ t.user ? t.user.nickname : '' }}）</em>
        </li>
      </ul>
      <div class="module-content" v-else><span class="empty">没有匹配的帖子</span></div>
      <div class="module-title">【找到的友友】({{ users.length }})</div>
      <ul class="dtuser" v-if="users.length">
        <li v-for="u in users" :key="u.id">
          <a href="javascript:;" @click="$router.push('/user/'+u.id)"><font :color="u.color || '#004299'">{{ u.nickname }}</font></a>
          （家园号码 {{ u.username }}）Lv.{{ u.level }}
        </li>
      </ul>
      <div class="module-content" v-else><span class="empty">没有匹配的友友</span></div>
    </template>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Search',
  data () { return { word: '', threads: [], users: [], done: false } },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      this.word = this.$route.query.word || ''
      if (!this.word) return
      api.get('/search', { params: { word: this.word } }).then(r => {
        if (r.code === 0) {
          this.threads = r.data.threads
          this.users = r.data.users
          this.done = true
        }
      })
    },
    go () {
      if (this.word) this.$router.push('/search?word=' + encodeURIComponent(this.word))
    }
  }
}
</script>
