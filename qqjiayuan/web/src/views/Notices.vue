<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/')">社区广场</a>&gt;公告通知<br></div>

    <!-- 公告 / 通知 切换 -->
    <div class="module-title">
      <a href="javascript:;" @click="tab='ann'"><font :color="tab === 'ann' ? '#e05a00' : '#004299'">【公告】</font></a>
      <a href="javascript:;" @click="tab='notify'"><font :color="tab === 'notify' ? '#e05a00' : '#004299'">【通知】</font></a>
    </div>

    <!-- ===== 公告（诺哈 wap_notice：公告/广播/活动） ===== -->
    <template v-if="tab === 'ann'">
      <div v-if="anns.length">
        <div v-for="a in anns" :key="a.id">
          <div class="module-title">
            <span :style="{ color: typeColor(a.type) }">{{ typeName(a.type) }}</span>
            <a href="javascript:;" @click="toggleAnn(a)">{{ a.title }}</a>
            <span class="txt-fade">（{{ fmt(a.created_at) }}）</span>
          </div>
          <div class="module-content" v-if="a._open" style="background:#FFFDE7">
            {{ a.content }}<br>
          </div>
        </div>
      </div>
      <div class="module-content" v-else><span class="empty">暂无公告</span></div>
    </template>

    <!-- ===== 通知（消息通知：回复/好友/系统） ===== -->
    <template v-else>
      <div class="module-content">
        <a href="javascript:;" @click="readAll">全部已读</a><br>
      </div>
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
    </template>

    <div class="bar"><a href="javascript:;" @click="$router.push('/')">社区广场</a>&gt;公告通知<br></div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Notices',
  data () { return { tab: 'ann', anns: [], list: [], page: 1, pages: 1 } },
  mounted () {
    if (this.$route.query.tab === 'notify') this.tab = 'notify'
    this.loadAnn()
    this.loadNotify()
  },
  methods: {
    loadAnn () {
      api.get('/announcements').then(r => {
        if (r.code === 0) this.anns = (r.data || []).map(a => ({ ...a, _open: false }))
      })
    },
    loadNotify () {
      api.get('/notifications', { params: { page: this.page } }).then(r => {
        if (r.code === 0) {
          this.list = r.data.list
          this.page = r.data.page
          this.pages = Math.max(1, Math.ceil(r.data.total / 20))
          this.$store.commit('setUnread', r.data.unread || 0)
        }
      })
    },
    toggleAnn (a) { this.$set(a, '_open', !a._open) },
    readAll () {
      api.post('/notifications/read').then(r => { if (r.code === 0) this.loadNotify() })
    },
    go (p) { this.page = p; this.loadNotify() },
    typeName (t) { return { notice: '【公告】', broadcast: '【广播】', activity: '【活动】' }[t] || '【公告】' },
    typeColor (t) { return { notice: '#004299', broadcast: '#1a9e1a', activity: '#e05a00' }[t] || '#004299' },
    nameOf (t) { return { reply: '回复', friend: '好友', system: '系统' }[t] || '通知' },
    colorOf (t) { return { reply: '#71afe3', friend: '#1a9e1a', system: '#d98f00' }[t] || '#999' },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t)
      const p = n => (n < 10 ? '0' + n : n)
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes())
    }
  }
}
</script>