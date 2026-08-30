<template>
  <div>
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区广场</a>&gt;用户信息<br>
    </div>
    <div class="module-content">
      <img v-if="u.avatar" :src="$pic(u.avatar)" width="150" height="150" alt="."><br>
      社区 I D :{{ u.username }}(<font :color="u.online ? '#1a9e1a' : '#999'">{{ u.online ? '在线' : '离线' }}</font>)<br>
      家园昵称:<a href="javascript:;"><font :color="u.color || '#004299'">{{ u.nickname }}</font></a>(<a href="javascript:;" @click="addFriend">加为好友</a>)<template v-if="friendTip"><font color="#1a9e1a"> {{ friendTip }}</font></template><br>
      是位{{ u.gender === 2 ? '小姐姐' : '小哥哥' }}<br>
      <template v-if="firstBadge">称号:<img class="bicon" :src="$pic(firstBadge.icon)" :alt="firstBadge.name">{{ firstBadge.name }}<br></template>
      去踩踩:<a href="javascript:;" @click="$router.push('/messages/'+u.id)">给他留言</a>|<a href="javascript:;" @click="$router.push('/friends')">我的好友</a><br>
      论坛成就:{{ u.level_title }}（{{ u.level }}级, exp:{{ u.exp }}）<br>
      家园心情:{{ u.signature || '这个人很懒，什么都没留下' }}<br>
      发帖 {{ u.thread_count }} · 回帖 {{ u.reply_count }} · 累计签到 {{ u.sign_days }} 天 · 注册于 {{ fmt(u.created_at) }}<br>
    </div>

    <div class="module-title">
      社区成就[<a href="javascript:;" @click="showAch = !showAch">{{ showAch ? '收起' : 'TA的成就' }}</a>]<br>
    </div>
    <div class="module-content">
      当前成就点数为：{{ u.achieve || 0 }} 等级为：{{ u.achieve_level || 0 }}<br>
      <template v-if="showAch">
        <div class="deep" style="padding:4px;margin:3px 0">
          成就明细（预估）：发帖 {{ u.thread_count }}×3 + 回帖 {{ u.reply_count }}×1 + 签到 {{ u.sign_days }}×2 ≈ {{ estAchieve }} 点<br>
          每100点升1级，当前{{ u.achieve_level || 0 }}级，距下一级还差 {{ 100 - ((u.achieve || 0) % 100) }} 点
        </div>
      </template>
      <span class="help-line">发帖+3、回帖+1、签到+2 成就点，每100点升1级</span>
    </div>

    <div class="module-title">
      婚恋状态<br>
    </div>
    <div class="module-content">
      <template v-if="u.partner_name">
        城堡:<a href="javascript:;" @click="$router.push('/user/'+u.partner_id)"><font color="#ff69b4">{{ u.partner_name }}</font></a><br>
        详情:执子之手，与子偕老<br>
        宝宝:{{ u.baby_name || '　暂无' }}<br>
      </template>
      <template v-else>
        城堡:<span class="help-line">虚位以待，缘分未到</span><br>
        宝宝:{{ u.baby_name || '　暂无' }}<br>
      </template>
    </div>

    <div class="module-title">
      贵族身份<br>
    </div>
    <div class="module-content">
      <template v-if="u.noble > 0">
        <img v-if="u.noble >= 2" :src="$pic('15.gif')" alt="2级"><img v-if="u.noble >= 1" :src="$pic('103.gif')" :alt="u.noble >= 2 ? '1级' : '2级'">
        {{ u.noble >= 2 ? '二级贵族' : '一级贵族' }}
      </template>
      <span v-else class="help-line">暂无贵族身份，请联系管理员册封</span><br>
    </div>

    <div class="module-title">
      业务状态<br>
    </div>
    <div class="module-content">
      <template v-if="u.priv">
        <img :src="'/static/' + u.priv.file" :alt="u.priv.name"> 特权：{{ u.priv.name }}（Lv.{{ u.priv.level }}）
      </template>
      <template v-else>
        <img :src="$pic('v' + (u.level_icon || 1) + '.gif')" alt="等级"> 等级图标 Lv.{{ u.level }}（{{ u.level_title }}）
      </template>
      <br>
    </div>

    <div class="module-title">
      社区职务<br>
    </div>
    <div class="module-content">
      <template v-if="duties.length || realRoles.length">
        <template v-for="(d,i) in duties" :key="'d'+i">{{ i+1 }}.<img class="bicon" :src="$pic(d.icon)" :alt="d.name">{{ d.name }}<br></template>
        <template v-for="(r,i) in realRoles" :key="'r'+r.id">{{ duties.length + i + 1 }}.{{ r.name }} <a href="javascript:;">[+]</a><br></template>
      </template>
      <span v-else class="help-line">暂无职务，社区常年招募管理员与版主</span>
    </div>

    <div class="module-title">
      荣誉成就[马甲]<br>
    </div>
    <div class="module-content">
      <template v-if="medalBadges.length">
        <span v-for="b in medalBadges" :key="b.id" style="margin-right:8px"><img class="bicon" :src="$pic(b.icon)" :alt="b.name">{{ b.name }}</span>
      </template>
      <span v-else class="help-line">还没有获得马甲，多发帖多签到，管理员会看到的！</span><br>
    </div>

    <div class="module-title">
      TA的最新帖子
    </div>
    <ul class="dtuser" v-if="threads.length">
      <li v-for="t in threads" :key="t.id">
        <a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a>
        <em>（{{ t.board ? t.board.name : '' }} · {{ t.view_count }}阅/{{ t.reply_count }}回）</em>
      </li>
    </ul>
    <div class="module-content" v-else><span class="empty">TA还没有发过帖子</span></div>
    <p v-if="tip" class="module-content" style="color:#1a9e1a">{{ tip }}</p>
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/')">社区广场</a>&gt;用户信息<br>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'UserPage',
  data () {
    return { u: {}, threads: [], badges: [], roles: [], duties: [], tip: '', friendTip: '', showAch: false }
  },
  computed: {
    isLogin () { return this.$store.getters.isLogin },
    user () { return this.$store.state.user },
    firstBadge () { return this.badges.length ? this.badges[0] : null },
    dutyIcons () { return ['706.jpg', '704.gif', '3.gif', '501.gif'] },
    medalBadges () { return this.badges.filter(b => this.dutyIcons.indexOf(b.icon) < 0) },
    realRoles () { return this.roles.filter(r => r.code !== 'member') },
    estAchieve () {
      return (this.u.thread_count || 0) * 3 + (this.u.reply_count || 0) + (this.u.sign_days || 0) * 2
    }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/users/' + this.$route.params.id).then(r => {
        if (r.code === 0) {
          this.u = r.data
          this.threads = r.data.threads
          this.badges = r.data.badges || []
          this.roles = r.data.roles || []
          this.duties = r.data.duties || []
        } else alert(r.msg)
      })
    },
    addFriend () {
      // 未登录直接去登录页；已登录在昵称旁给出即时反馈
      if (!this.isLogin) {
        this.$router.push('/login?redirect=' + encodeURIComponent(this.$route.fullPath))
        return
      }
      this.friendTip = '发送中…'
      api.post('/friends', { target_id: this.u.id }).then(r => {
        this.friendTip = r.code === 0 ? (typeof r.data === 'string' ? r.data : '申请已发送') : r.msg
      })
    },
    fmt (t) { return t ? new Date(t).toISOString().slice(0, 10) : '' }
  }
}
</script>
