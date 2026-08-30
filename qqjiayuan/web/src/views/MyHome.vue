<template>
  <div>
    <div class="module-content">
      <b><a href="javascript:;" @click="$router.push('/user/'+u.id)"><font :color="u.color || '#004299'">{{ u.nickname }}</font></a> {{ u.level }}级</b>
      <img v-if="u.priv" :src="'/static/' + u.priv.file" :alt="u.priv.name" :title="u.priv.name">
      <img v-else-if="u.level_icon" :src="$pic('v'+u.level_icon+'.gif')" alt="等级">
      <template v-if="u.avatar"><br><img :src="$pic(u.avatar)" width="60" height="60" style="border-radius:4px;vertical-align:middle" alt="头像"></template>
    </div>
    <div class="write-mood">
      {{ mood ? mood.content : (u.signature || '暂无心情.') }} <a href="javascript:;" @click="$router.push('/mood')">&gt;&gt;</a><br>
    </div>
    <div class="module-content">
      <a href="javascript:;" @click="$router.push('/games')">宅子</a> . <a href="javascript:;" @click="$router.push('/sign')">友友券</a> . <a href="javascript:;" @click="$router.push('/channel/2')">回家</a> . <a href="javascript:;" @click="$router.push('/games')">花仙子</a>
    </div>
    <div class="module-title">
      我的  <a href="javascript:;" @click="$router.push('/notices')"><font color="#FF0000">消息</font></a>  <a href="javascript:;" @click="$router.push('/profile')">设置</a>  <a href="javascript:;" @click="$router.push('/user/'+u.id)">主页</a><br>
    </div>
    <div class="module-title">
      <a href="javascript:;" @click="$router.push('/profile')">我的帖子</a>
    </div>
    <ul class="dtuser" v-if="threads.length">
      <li v-for="t in threads" :key="t.id">
        <a href="javascript:;" @click="$router.push('/thread/'+t.id)">{{ t.title }}</a>
        <em>（{{ t.board ? t.board.name : '' }} · {{ t.view_count }}阅/{{ t.reply_count }}回）</em>
      </li>
    </ul>
    <div class="module-content" v-else><span class="empty">还没发过帖，去<a href="javascript:;" @click="$router.push('/channel/1')">论坛</a>冒个泡</span></div>
    <div class="module-title">
      <a href="javascript:;" @click="$router.push('/friends')">好友来访</a>
    </div>
    <ul class="dtuser" v-if="friends.length">
      <li v-for="f in friends" :key="f.id">
        <a href="javascript:;" @click="$router.push('/user/'+f.id)"><font :color="f.color || '#004299'">{{ f.nickname }}</font></a>
        <a class="rt" href="javascript:;" @click="$router.push('/messages/'+f.id)">家信</a>
      </li>
    </ul>
    <div class="module-content" v-else><span class="empty">还没有好友，去<a href="javascript:;" @click="$router.push('/friends')">好友</a>页添加</span></div>
    <div class="module-content">
      <form @submit.prevent="visit">
        <input type="text" v-model.number="visitId" size="10" maxlength="10" placeholder="输入号码">
        <input type="submit" value="串门">
      </form>
    </div>
    <div class="module-title">
      【功能导航】<br>
    </div>
    <div class="module-content">
      <a href="javascript:;" @click="$router.push('/messages')">家信</a>.<a href="javascript:;" @click="$router.push('/board/4')">婚恋</a>.<a href="javascript:;" @click="$router.push('/channel/1')">论坛</a>.<a href="javascript:;" @click="$router.push('/channel/2')">家族</a>.<a href="javascript:;" @click="$router.push('/sign')">活动</a>.<a href="javascript:;" @click="$router.push('/chat')">聊天室</a>.<a href="javascript:;" @click="$router.push('/nav')">&gt;&gt;</a><br>
      <a href="javascript:;" @click="$router.push('/sign')">任务</a>.<a href="javascript:;" @click="$router.push('/board/17')">反馈</a>.<a href="javascript:;" @click="$router.push('/chat')">秘密</a>.<a href="javascript:;" @click="$router.push('/nav')">黑板墙</a>.<a href="javascript:;" @click="$router.push('/profile')">特权</a>.<a href="javascript:;" @click="$router.push('/find')">靓号</a>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'MyHome',
  data () {
    return { u: {}, threads: [], friends: [], visitId: '', mood: null }
  },
  mounted () { this.load() },
  methods: {
    load () {
      const me = this.$store.state.user
      if (!me) return
      api.get('/users/' + me.id).then(r => {
        if (r.code === 0) {
          this.u = r.data
          this.threads = (r.data.threads || []).slice(0, 5)
        }
      })
      api.get('/friends').then(r => {
        if (r.code === 0) this.friends = r.data.friends.slice(0, 8)
      })
      api.get('/moods/latest').then(r => {
        if (r.code === 0) this.mood = r.data
      })
    },
    visit () {
      if (this.visitId) this.$router.push('/user/' + this.visitId)
    }
  }
}
</script>
