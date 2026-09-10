<template>
  <div>
    <!-- 参考：诺哈 topic_active.html 活动中心列表（序号.标题 / 头像 昵称:N回/M阅 / 分页） -->
    <div class="bar"><a href="javascript:;" @click="$router.push('/')">社区广场</a>&gt;活动中心<br></div>

    <div class="list">
      <div v-for="(a, i) in list" :key="a.id" class="row">
        {{ (page - 1) * size + i + 1 }}.<a href="javascript:;" @click="$router.push('/thread/'+a.id)">{{ a.title }}</a><br>
        (<img class="bicon" :src="face(a)" alt="." @error="faceErr($event)"> <font :color="a.color || '#004299'">{{ a.nickname }}</font>:<a href="javascript:;" @click="$router.push('/thread/'+a.id)">{{ a.reply_count }}</a>回/{{ a.view_count }}阅)<br>
      </div>
    </div>
    <span v-if="!list.length" class="empty">暂无活动</span>

    <div v-if="totalPages > 1" class="pager">
      <a href="javascript:;" v-if="page < totalPages" @click="load(page+1)">下页</a>
      <form @submit.prevent="go">
        第<input type="text" v-model.number="jumpPage" size="2" />页
        <input type="submit" value="前往" />
      </form>
      (第<b>{{ page }}</b>/{{ totalPages }}页/共{{ total }}条记录)<br>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'ActivityList',
  data () {
    return { list: [], page: 1, size: 10, total: 0, totalPages: 1, jumpPage: 1 }
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
    face (a) {
      if (a.avatar) return this.$pic(a.avatar)
      return this.$pic('face_' + (a.user_id % 3 + 1) + '.gif')
    },
    faceErr (e) { e.target.style.visibility = 'hidden' }
  }
}
</script>
