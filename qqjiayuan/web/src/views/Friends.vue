<template>
  <div>
    <div class="module-title">我的好友|<a href="javascript:;" @click="tab='recent'">最近联系</a></div>

    <!-- 我的好友 -->
    <template v-if="tab === 'friends'">
      <div class="list">
        <div v-for="f in friends" :key="f.id" class="row">
          <img :src="f.online ? '/static/image/home.gif' : '/static/image/home1.png'" alt="好友">
          <a href="javascript:;" @click="$router.push('/user/'+f.id)"><font :color="f.color || '#004299'">{{ f.nickname }}</font></a>
          [<a href="javascript:;" @click="$router.push('/messages/'+f.id)">发家信</a>]
          <span :style="{ color: f.online ? '#1a9e1a' : '#999' }">{{ f.online ? '在线' : '离线' }}</span>
        </div>
      </div>
      <div class="module-content" v-if="!friends.length"><span class="empty">还没有好友，去添加一个吧</span></div>
      <div class="module-content">(第<b>1</b>/1页/共{{ friends.length }}个好友)</div>

      <div class="module-content">
        <form @submit.prevent="goSearch">
          输入ID号码或昵称搜索好友：
          <input type="text" v-model.trim="word" size="12" maxlength="20">
          <input type="submit" value="搜好友">
        </form>
      </div>
      <div class="module-content">好友图标展示说明：<img src="/static/image/home.gif" alt="." /> 家园在线　<img src="/static/image/home1.png" alt="." /> 家园离线</div>
    </template>

    <!-- 最近联系 -->
    <template v-else>
      <div class="list">
        <div v-for="c in conversations" :key="c.user_id" class="row">
          <img src="/static/image/home.gif" alt="好友">
          <a href="javascript:;" @click="$router.push('/messages/'+c.user_id)"><font :color="c.color || '#004299'">{{ c.nickname }}</font></a>
          <span class="txt-fade">：{{ (c.last_content || '').slice(0, 12) }}</span>
          <span class="txt-fade">（{{ fmtShort(c.last_at) }}）</span>
          [<a href="javascript:;" @click="$router.push('/messages/'+c.user_id)">发家信</a>]
        </div>
      </div>
      <div class="module-content" v-if="!conversations.length"><span class="empty">暂无最近联系</span></div>
    </template>

    <!-- 收到的好友申请 -->
    <div class="module-title">【收到的好友申请】({{ requests.length }})</div>
    <ul class="dtuser" v-if="requests.length">
      <li v-for="r in requests" :key="'r'+r.apply_id">
        <a href="javascript:;" @click="$router.push('/user/'+r.id)"><font :color="r.color || '#004299'">{{ r.nickname }}</font></a>
        申请成为好友（{{ fmt(r.created_at) }}）<br>
        <a href="javascript:;" @click="handle(r.apply_id, 'accept')">同意</a>.<a href="javascript:;" @click="handle(r.apply_id, 'reject')">拒绝</a>
      </li>
    </ul>
    <div class="module-content" v-else><span class="empty">没有待处理的好友申请</span></div>

    <!-- 功能导航 -->
    <div class="module-title">功能导航</div>
    <div class="module-content">
      <a href="javascript:;" @click="$router.push('/inbox')">全部消息</a>|<a href="javascript:;" @click="$router.push('/inbox')">系统消息</a>|<a href="javascript:;" @click="$router.push('/channel/1')">好友动态</a><br>
      <a href="javascript:;" @click="$router.push('/profile')">交友设置</a>|<a href="javascript:;" @click="$router.push('/search')">好友查找</a>|<a href="javascript:;" @click="$router.push('/groups')">分组管理</a>
    </div>

    <p style="color:#c00;padding:0 5px" v-if="msg">{{ msg }}</p>
    <p style="color:#1a9e1a;padding:0 5px" v-if="okMsg">{{ okMsg }}</p>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Friends',
  data () { return { tab: 'friends', friends: [], conversations: [], requests: [], word: '', msg: '', okMsg: '' } },
  mounted () { this.load() },
  methods: {
    load () {
      api.get('/friends').then(r => {
        if (r.code === 0) { this.friends = r.data.friends; this.requests = r.data.requests }
      })
      api.get('/messages/conversations').then(r => { if (r.code === 0) this.conversations = r.data })
    },
    handle (id, action) {
      api.post('/friends/' + id + '/handle', { action }).then(r => { if (r.code === 0) { this.okMsg = '已处理'; this.load() } else this.msg = r.msg })
    },
    goSearch () { if (this.word) this.$router.push('/search?word=' + encodeURIComponent(this.word)) },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t)
      return d.getFullYear() + '/' + (d.getMonth() + 1) + '/' + d.getDate() + ' ' + d.getHours() + ':' + (d.getMinutes() < 10 ? '0' : '') + d.getMinutes()
    },
    fmtShort (t) {
      if (!t) return ''
      const d = new Date(t)
      return d.getMonth() + 1 + '/' + d.getDate() + ' ' + d.getHours() + ':' + (d.getMinutes() < 10 ? '0' : '') + d.getMinutes()
    }
  }
}
</script>
