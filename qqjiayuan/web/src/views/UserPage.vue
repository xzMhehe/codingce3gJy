<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/')">社区广场</a>&gt;用户信息<br></div>

    <!-- ===== 会员信息 ===== -->
    <div class="module-title">【会员信息】</div>
    <div class="module-content">
      <img :src="avatarImg" style="width:96px;height:96px;object-fit:contain;border-radius:4px" alt="头像"><br>
      社区 I D :{{ u.username || u.id }}<span :style="{ color: u.online ? '#1a9e1a' : '#999' }">({{ u.online ? '在线' : '离线' }})</span><br>
      家园昵称:<a href="javascript:;" @click="$router.push('/user/'+u.id)"><font :color="u.color || '#004299'">{{ u.nickname }}</font></a>
      <template v-if="!isMine">
        (<a href="javascript:;" @click="addFriend">{{ friendState === 'friend' ? '删好友' : (friendState === 'applied' ? '申请中' : '加为好友') }}</a>)
        <template v-if="friendTip"><font color="#1a9e1a"> {{ friendTip }}</font></template>
      </template>
      <template v-else>(<a href="javascript:;" @click="$router.push('/profile')">更新资料</a>)</template><br>
      {{ genderText }}<br>
      生日:{{ birthText }}<span v-if="u.birth_type === 0" class="txt-fade">(农历)</span><br>
      城市:{{ u.city || '—' }}<br>
      故乡:{{ homeText }}　现居:{{ liveText }}<br>
      家族:{{ u.family || '无家可归待收留' }}<br>
      去踩踩:<a href="javascript:;" @click="$router.push('/space/'+u.id)">{{ taWord }}的空间</a>|<a href="javascript:;" @click="goHisHome">{{ taWord }}的家园</a><br>
      论坛成就:{{ u.level_title || '无称号' }}（{{ u.level }}级, exp:{{ u.exp }}）<br>
      家园等级:<img :src="'/static/picture/home_' + (u.gender === 2 ? '2' : '1') + '_' + pad(u.level) + '.gif'" alt="." style="height:14px;vertical-align:-2px">
      (<a href="javascript:;" @click="$router.push('/home-level')">LV{{ u.level }}</a>)<span class="txt-fade"> 升级还需{{ nextNeed }}天</span><br>
      在线时长:{{ hoursText }}<br>
      家园心情:{{ u.mood || '（无）' }}<br>
      个人介绍:{{ u.introduction || '这个人很懒，什么都没留下' }}<br>
      <template v-if="contact">
        <span class="txt-fade">--- 我的联系方式 ---</span><br>
        QQ:{{ contact.qq || '未设置' }}　邮箱:{{ contact.mail || '未设置' }}　手机:{{ contact.phone || '未设置' }}<br>
      </template>
      发帖 {{ u.thread_count }} · 回帖 {{ u.reply_count }} · 签到 {{ u.sign_days }} 天 · 注册 {{ fmt(u.created_at) }}<br>
    </div>

    <!-- ===== 社区成就 ===== -->
    <div class="module-title">社区成就[<a href="javascript:;" @click="$router.push('/achieve')">TA的成就</a>]</div>
    <div class="module-content">
      当前成就点数为：{{ u.achieve || 0 }} 等级为：{{ u.achieve_level || 0 }}<br>
    </div>

    <!-- ===== 婚恋状态 ===== -->
    <div class="module-title">婚恋状态</div>
    <div class="module-content">
      <template v-if="u.partner_name">
        伴侣:<a href="javascript:;" @click="$router.push('/user/'+u.partner_id)"><font :color="u.color || '#004299'">{{ u.partner_name }}</font></a>
        <template v-if="u.baby_name"><br>宝宝:{{ u.baby_name }}</template><br>
      </template>
      <template v-else>
        单身贵族<template v-if="!isMine"> <a href="javascript:;" @click="tip('求婚功能')">求婚</a></template><br>
      </template>
    </div>

    <!-- ===== 贵族身份 ===== -->
    <div class="module-title">贵族身份</div>
    <div class="module-content">
      <template v-if="u.blue_lv > 0">蓝钻<img :src="'/static/picture/noble_2_' + u.blue_lv + '.gif'" :alt="'蓝钻' + u.blue_lv + '级'" style="height:14px;vertical-align:-2px" @error="hideErr"> Lv.{{ u.blue_lv }}</template>
      <template v-if="u.qq_lv > 0">超Q<img :src="'/static/picture/noble_1_' + u.qq_lv + '.gif'" :alt="'超Q' + u.qq_lv + '级'" style="height:14px;vertical-align:-2px" @error="hideErr"> Lv.{{ u.qq_lv }}</template>
      <template v-if="!u.blue_lv && !u.qq_lv">
        <span class="txt-fade">未开通贵族身份，<a href="javascript:;" @click="$router.push('/noble')">去开通</a></span><br>
      </template>
      <template v-if="u.priv"><img :src="'/static/picture/' + u.priv.file" :alt="u.priv.name" :title="u.priv.name" style="height:14px;vertical-align:-2px"></template>
    </div>

    <!-- ===== 业务状态 ===== -->
    <div class="module-title">业务状态</div>
    <div class="module-content">
      <template v-if="u.paid > 0"><img src="/static/picture/liang.gif" alt="靓号" title="靓号"></template>
      <template v-if="u.has_docu"><img src="/static/picture/docu.jpg" alt="身份证" title="实名认证"></template>
      <template v-if="u.has_qq"><img src="/static/picture/qq.jpg" alt="QQ" title="已绑定QQ"></template>
      <template v-if="u.has_phone"><img src="/static/picture/phone.jpg" alt="手机" title="已绑定手机"></template>
      <template v-if="u.has_mail"><img src="/static/picture/email.jpg" alt="邮箱" title="已绑定邮箱"></template>
      <template v-if="!u.paid && !u.has_docu && !u.has_qq && !u.has_phone && !u.has_mail"><span class="txt-fade">暂无业务绑定</span></template><br>
    </div>

    <!-- ===== 社区职务 ===== -->
    <div class="module-title">社区职务</div>
    <div class="module-content" v-if="(u.duties || []).length">
      <div v-for="(d,i) in u.duties" :key="'d'+i">
        {{ i+1 }}.<img :src="'/static/picture/' + d.icon" alt="." style="height:12px;vertical-align:-1px"> {{ d.name }}<br>
      </div>
    </div>
    <div class="module-content" v-else><span class="txt-fade">暂无社区职务</span></div>

    <!-- ===== 荣誉成就 ===== -->
    <div class="module-title">荣誉成就</div>
    <div class="module-content" v-if="(u.badges || []).length">
      <div v-for="(b,i) in u.badges" :key="'b'+b.id">
        {{ i+1 }}.<img class="bicon" :src="'/static/picture/' + b.icon" :alt="b.name" :title="b.name"> {{ b.name }}<br>
      </div>
    </div>
    <div class="module-content" v-else><span class="txt-fade">TA还没有勋章</span></div>

    <!-- ===== 和TA互动 / 个人功能 ===== -->
    <template v-if="!isMine">
      <div class="module-title">【和{{ taWord }}互动】</div>
      <div class="module-content">
        <a href="javascript:;" @click="$router.push('/messages/'+u.id)">给{{ taWord }}家信</a>.<a href="javascript:;" @click="$router.push('/wallet?view=transfer')">转账</a><br>
        <template v-if="friendState === 'friend'">
          <a href="javascript:;" @click="removeFriend" style="color:#c00">删好友</a>
        </template>
        <template v-else>
          <a href="javascript:;" @click="addFriend">加好友</a>
        </template>
        .<a href="javascript:;" @click="toggleBlack" :style="{ color: friendState === 'blacked' ? '#1a9e1a' : '#c00' }">{{ friendState === 'blacked' ? '删黑名单' : '加黑名单' }}</a><br>
      </div>
    </template>
    <template v-else>
      <div class="module-title">【个人功能】</div>
      <div class="module-content">
        <a href="javascript:;" @click="$router.push('/profile')">更新资料</a>.<a href="javascript:;" @click="$router.push('/home')">我的家园</a>.<a href="javascript:;" @click="$router.push('/security')">安全设置</a><br>
      </div>
    </template>

    <!-- ===== 最新帖子 ===== -->
    <div class="module-title">{{ taWord }}的最新帖子</div>
    <ul class="dtuser" v-if="threads.length">
      <li v-for="t in threads" :key="t.id"><a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a> <em>（{{ t.board ? t.board.name : '' }} · {{ t.view_count }}阅/{{ t.reply_count }}回）</em></li>
    </ul>
    <div class="module-content" v-else><span class="empty">{{ taWord }}还没有发过帖子</span></div>

    <div class="bar"><a href="javascript:;" @click="$router.push('/')">社区广场</a>&gt;用户信息<br></div>

    <p style="color:#c00;padding:0 5px" v-if="msg">{{ msg }}</p>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'UserPage',
  data () { return { u: {}, threads: [], friendTip: '', friendState: '', msg: '' } },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    user () { return this.$store.state.user },
    isMine () { return this.user && this.u.id === this.user.id },
    contact () { return this.isMine ? this.u.contact : null },
    taWord () { return (this.u.gender === 2 || this.u.gender === '2') ? '她' : '他' },
    genderText () {
      const g = this.u.gender
      const age = this.u.age || 0
      if (g === 2) return age ? `是位小姐姐今年${age}岁` : '是位小姐姐'
      return age ? `是位小哥哥今年${age}岁` : '是位小哥哥'
    },
    avatarImg () {
      if (this.u.avatar_base64 && this.u.avatar_base64.length > 20) return this.u.avatar_base64
      if (this.u.avatar) return '/static/picture/' + this.u.avatar
      return this.u.gender === 2 ? '/static/picture/0.gif' : '/static/picture/0.gif'
    },
    birthText () {
      if (this.u.solar) return this.u.solar
      const y = this.u.birth_year; const m = this.u.birth_month; const d = this.u.birth_day
      if (y && m && d) return y + '-' + m + '-' + d
      return '未设置'
    },
    homeText () {
      const a = this.u.address || {}
      const s = [a.home_nation, a.home_prov, a.home_city].filter(Boolean).join(' ')
      return s || '—'
    },
    liveText () {
      const a = this.u.address || {}
      const s = [a.live_nation, a.live_prov, a.live_city].filter(Boolean).join(' ')
      return s || '—'
    },
    hoursText () {
      const h = this.u.hours || 0
      if (h < 60) return h + '分钟'
      return Math.floor(h / 60) + '小时' + (h % 60 ? (h % 60) + '分' : '')
    },
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
        if (r.code === 0) { this.u = r.data; this.threads = r.data.threads || [] } else this.msg = r.msg
      })
      this.loadFriendState()
    },
    loadFriendState () {
      if (!this.isLogin || this.isMine) { this.friendState = ''; return }
      api.get('/friends/status/' + this.$route.params.id).then(r => { if (r.code === 0) this.friendState = r.data.state })
    },
    goHisHome () {
      if (this.isMine) { this.$router.push('/home'); return }
      this.$router.push('/space/' + this.u.id)
    },
    addFriend () {
      if (!this.isLogin) { this.$router.push('/login?redirect=' + encodeURIComponent(this.$route.fullPath)); return }
      this.friendTip = '发送中…'
      api.post('/friends', { target_id: this.u.id }).then(r => {
        this.friendTip = r.code === 0 ? (typeof r.data === 'string' ? r.data : '申请已发送') : r.msg
        if (r.code === 0) this.loadFriendState()
      })
    },
    removeFriend () {
      if (!window.confirm('确定删除好友「' + this.u.nickname + '」吗？')) return
      api.delete('/friends/' + this.u.id).then(r => {
        this.friendTip = r.code === 0 ? r.data : r.msg
        if (r.code === 0) this.loadFriendState()
      })
    },
    toggleBlack () {
      if (this.friendState === 'blacked') {
        api.get('/friends/black').then(r => {
          if (r.code !== 0) return
          const row = (r.data || []).find(b => b.friend_id === this.u.id)
          if (!row) { this.loadFriendState(); return }
          api.delete('/friends/black/' + row.id).then(r2 => {
            this.friendTip = r2.code === 0 ? r2.data : r2.msg
            if (r2.code === 0) this.loadFriendState()
          })
        })
        return
      }
      if (!window.confirm('确定把「' + this.u.nickname + '」加入黑名单吗？\n将不再接受对方的信息，并解除好友关系。')) return
      api.post('/friends/black/' + this.u.id).then(r => {
        this.friendTip = r.code === 0 ? r.data : r.msg
        if (r.code === 0) this.loadFriendState()
      })
    },
    tip (title) { this.$router.push('/tip?title=' + encodeURIComponent('家园·' + title)) },
    pad (n) { return '' + (n || 1) },
    hideErr (e) { e.target.style.display = 'none' },
    fmt (t) { return t ? new Date(t).toISOString().slice(0, 10) : '' }
  }
}
</script>