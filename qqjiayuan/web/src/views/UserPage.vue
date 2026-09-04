<template>
  <div>
    <div class="module-title">资料</div>
    <div class="module-content">
      ID:{{ u.username }} <span :style="{ color: u.online ? '#1a9e1a' : '#999' }">（{{ u.online ? '在线' : '离线' }}）</span><br>
      昵称:<a href="javascript:;" @click="$router.push('/user/'+u.id)"><font :color="u.color || '#004299'">{{ u.nickname }}</font></a>
      (<a href="javascript:;" @click="addFriend">加为好友</a>)<template v-if="friendTip"><font color="#1a9e1a"> {{ friendTip }}</font></template><br>
      性别:{{ u.gender === 2 ? '女' : '男' }}<br>
      城市:{{ u.city || '—' }}<br>
      年龄:—<br>
      生日:{{ fmt(u.created_at) }}<br>
    </div>

    <div class="module-title">家园资料</div>
    <div class="module-content">
      等级:<img :src="'/static/picture/home_' + (u.gender === 2 ? '2' : '1') + '_' + pad(u.level) + '.gif'" alt="." style="height:14px;vertical-align:-2px">
      (<a href="javascript:;" @click="$router.push('/home-level')">LV{{ u.level }}</a>)<br>
      <img src="/static/picture/jindu.jpg" alt="." /><br>
      升级还需{{ nextNeed }}天<br>
      触屏版用户:<img src="/static/picture/chuping.jpg" alt="." /><br>
      G币:<a href="javascript:;" @click="$router.push('/wallet')">{{ u.coins }}</a><br>
      成就:<a href="javascript:;" @click="$router.push('/achieve')">{{ u.achieve || 0 }}/0</a><br>
      成就称号：{{ u.level_title || '无' }}<br>
      <a href="javascript:;" @click="$router.push('/user/'+u.id)">&gt;&gt;</a><br>
      所属家族:{{ u.family || '未加入任何家族' }}<br>
      心情:<a href="javascript:;" @click="$router.push('/user/'+u.id)">{{ u.mood || '（无）' }}</a><br>
      爱情状态:{{ u.partner_name || '单身' }}<template v-if="u.partner_name"><a href="javascript:;" @click="$router.push('/user/'+u.partner_id)"> 伴侣</a></template><br>
      个人说明:{{ u.signature || '这个人很懒，什么都没留下' }}<br>
      <a v-if="isMine" href="javascript:;" @click="$router.push('/profile')">更新个人资料</a><br>
      发帖 {{ u.thread_count }} · 回帖 {{ u.reply_count }} · 签到 {{ u.sign_days }} 天 · 注册 {{ fmt(u.created_at) }}<br>
    </div>

    <div class="module-title">TA的最新帖子</div>
    <ul class="dtuser" v-if="threads.length">
      <li v-for="t in threads" :key="t.id"><a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a> <em>（{{ t.board ? t.board.name : '' }} · {{ t.view_count }}阅/{{ t.reply_count }}回）</em></li>
    </ul>
    <div class="module-content" v-else><span class="empty">TA还没有发过帖子</span></div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'UserPage',
  data () { return { u: {}, threads: [], friendTip: '' } },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    user () { return this.$store.state.user },
    isMine () { return this.user && this.u.id === this.user.id },
    nextNeed () {
      const d = this.u.active_days || 0
      const nd = this.u.home_next_days || 0
      if (!nd) return '满级'
      const r = Math.max(0, nd - d)
      return Math.round(r * 10) / 10
    }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/users/' + this.$route.params.id).then(r => {
        if (r.code === 0) { this.u = r.data; this.threads = r.data.threads || [] } else alert(r.msg)
      })
    },
    addFriend () {
      if (!this.isLogin) { this.$router.push('/login?redirect=' + encodeURIComponent(this.$route.fullPath)); return }
      this.friendTip = '发送中…'
      api.post('/friends', { target_id: this.u.id }).then(r => { this.friendTip = r.code === 0 ? (typeof r.data === 'string' ? r.data : '申请已发送') : r.msg })
    },
    pad (n) { return '' + (n || 1) },
    fmt (t) { return t ? new Date(t).toISOString().slice(0, 10) : '' }
  }
}
</script>
