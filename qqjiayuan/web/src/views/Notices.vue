<template>
  <div>
    <div class="bar">【消息通知】<a class="rt" href="javascript:;" @click="readAll">全部已读</a></div>
    <ul class="dtuser" v-if="list.length">
      <li v-for="n in list" :key="n.id" :style="{background: n.is_read ? '' : '#FFFDE7'}">
        <span class="tag" :style="{background: colorOf(n.type)}">{{ nameOf(n.type) }}</span>
        <b>{{ n.title }}</b><br>
        {{ n.content }}<br>
        <a v-if="n.type === 'reply'" href="javascript:;" @click="$router.push('/thread/'+n.ref_id)">查看帖子&gt;&gt;</a>
        <a v-else-if="n.type === 'friend'" href="javascript:;" @click="$router.push('/friends')">去处理&gt;&gt;</a>
        <em>{{ fmt(n.created_at) }}</em>
      </li>
    </ul>
    <div class="module-content" v-else><span class="empty">暂无通知</span></div>
    <div class="pager" v-if="pages > 1">
      <a v-if="page > 1" href="javascript:;" @click="go(page-1)">上页</a>
      第{{ page }}/{{ pages }}页
      <a v-if="page < pages" href="javascript:;" @click="go(page+1)">下页</a>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Notices',
  data () { return { list: [], page: 1, pages: 1 } },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/notifications', { params: { page: this.page } }).then(r => {
        if (r.code === 0) {
          this.list = r.data.list
          this.page = r.data.page
          this.pages = Math.max(1, Math.ceil(r.data.total / 20))
          this.$store.commit('setUnread', r.data.unread || 0)
        }
      })
    },
    readAll () {
      api.post('/notifications/read').then(r => {
        if (r.code === 0) this.load()
      })
    },
    go (p) { this.page = p; this.load() },
    nameOf (t) { return { reply: '回复', friend: '好友', system: '系统' }[t] || '通知' },
    colorOf (t) { return { reply: '#71afe3', friend: '#1a9e1a', system: '#d98f00' }[t] || '#999' },
    fmt (t) { return new Date(t).toISOString().slice(0, 16).replace('T', ' ') }
  }
}
</script>
