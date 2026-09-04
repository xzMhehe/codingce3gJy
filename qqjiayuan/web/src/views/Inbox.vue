<template>
  <div>
    <div class="module-title">功能导航</div>
    <div class="module-content">
      <a href="javascript:;" @click="setTab('all')">全部消息</a>|<a href="javascript:;" @click="setTab('system')">系统消息</a>|<a href="javascript:;" @click="setTab('friend')">好友动态</a>
    </div>

    <div class="module-title">{{ tabName }}（{{ filtered.length }}）</div>
    <div class="module-content">
      <template v-if="filtered.length">
        <div v-for="m in filtered" :key="m.id">
          【内容】：<a href="javascript:;" @click="openMsg(m)">{{ m.title || m.content }}</a>
          <span class="txt-fade">{{ fmt(m.created_at) }}</span><br>
        </div>
      </template>
      <span v-else class="empty">还没有{{ tabName }}</span>
    </div>

    <div class="module-content">----------<br>(第{{ page }}/{{ pages }}页/共{{ total }}条记录)</div>

    <div class="module-title">【功能导航】</div>
    <div class="module-content">
      <a href="javascript:;" @click="setTab('all')">全部消息</a>|<a href="javascript:;" @click="setTab('system')">系统消息</a>|<a href="javascript:;" @click="setTab('friend')">好友动态</a><br>
      <a href="javascript:;" @click="$router.push('/profile')">交友设置</a>|<a href="javascript:;" @click="$router.push('/friends')">好友查找</a>|<a href="javascript:;" @click="$router.push('/groups')">分组管理</a>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Inbox',
  data () { return { list: [], total: 0, page: 1, pages: 1, tab: 'all', unread: 0 } },
  computed: {
    tabName () { return this.tab === 'system' ? '系统消息' : (this.tab === 'friend' ? '好友动态' : '全部消息') },
    filtered () {
      if (this.tab === 'all') return this.list
      return this.list.filter(m => m.type === this.tab)
    }
  },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/notifications?page=' + this.page).then(r => {
        if (r.code === 0) {
          this.list = (r.data.list || []).slice(0, 20)
          this.total = r.data.total || 0
          this.pages = Math.max(1, Math.ceil(this.total / 20))
          this.unread = r.data.unread || 0
        }
      })
    },
    setTab (t) { this.tab = t },
    openMsg (m) {
      if (m.type === 'friend') this.$router.push('/friends')
      else this.$router.push('/notices')
    },
    tip (title) { this.$router.push('/tip?title=' + encodeURIComponent('家园·' + title)) },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t)
      return d.getFullYear() + '/' + (d.getMonth() + 1) + '/' + d.getDate() + ' ' + d.getHours() + ':' + (d.getMinutes() < 10 ? '0' : '') + d.getMinutes()
    }
  }
}
</script>
