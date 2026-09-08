<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/')">社区</a>&gt;活动</div>
    <div class="name">活动专区<br></div>
    <div class="module-content txt-fade">社区活动帖一览，版主可把帖子设为活动帖参与活动。</div>

    <div class="list">
      <div v-for="(a, i) in list" :key="a.id" class="row">
        {{ (page - 1) * size + i + 1 }}.<img class="bicon" :src="face(a)" alt="." @error="faceErr($event)"><a href="javascript:;" @click="$router.push('/thread/'+a.id)">{{ a.title }}</a>
        <a href="javascript:;" v-if="isLogin && a.user_id === user.id" @click="del(a.id)">[删]</a><br>
        (<font :color="a.color || '#004299'">{{ a.nickname }}</font>:<a href="javascript:;" @click="$router.push('/thread/'+a.id)">{{ a.reply_count }}</a>回/{{ a.view_count }}阅)<br>
      </div>
    </div>
    <span v-if="!list.length" class="empty">暂无活动</span>

    <div v-if="totalPages > 1" class="pager">
      <a href="javascript:;" v-if="page < totalPages" @click="load(page+1)">下页</a>.<a href="javascript:;" v-if="page > 1" @click="load(page-1)">上页</a><br>
      (第<b>{{ page }}</b>/{{ totalPages }}页/共{{ total }}条记录)<br>
      <form @submit.prevent="go">
        第<input type="text" v-model.number="jumpPage" size="2" />页<input type="submit" value="前往" />
      </form>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Activities',
  data () {
    return { list: [], page: 1, size: 10, total: 0, totalPages: 1, jumpPage: 1 }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    user () { return this.$store.state.user || {} }
  },
  mounted () { this.load(1) },
  methods: {
    load (p) {
      this.page = p
      api.get('/activities', { params: { page: p } }).then(r => {
        if (r.code === 0) {
          this.list = r.data.list || []
          this.size = r.data.size
          this.total = r.data.total
          this.totalPages = Math.ceil(r.data.total / r.data.size) || 1
          this.jumpPage = p
        }
      })
    },
    go () {
      const p = Math.max(1, Math.min(this.totalPages, this.jumpPage || 1))
      this.load(p)
    },
    del (id) {
      if (!confirm('确定删除该活动帖？')) return
      api.delete('/threads/' + id).then(() => this.load(this.page))
    },
    face (a) {
      if (a.avatar) return this.$pic(a.avatar)
      return this.$pic('face_' + (a.user_id % 3 + 1) + '.gif')
    },
    faceErr (e) { e.target.style.visibility = 'hidden' }
  }
}
</script>