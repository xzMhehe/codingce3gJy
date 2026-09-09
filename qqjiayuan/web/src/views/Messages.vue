<template>
  <div>
    <!-- ===== 信箱首页（对齐诺哈 chat/index.asp：收信箱/发信箱/系统信息） ===== -->
    <template v-if="!peer">
      <div class="bar"><a href="javascript:;" @click="$router.push('/home')">我的地盘</a>&gt;信箱<br></div>

      <div class="module-title">【基本设施】</div>
      <div class="module-content">
        <a href="javascript:;" @click="tab='inbox'">收信箱({{ inboxUnread }})</a><br>
        <a href="javascript:;" @click="tab='outbox'">发信箱</a><br>
        <a href="javascript:;" @click="tab='sys'">系统信息({{ sysUnread }})</a><br>
      </div>

      <!-- 收信箱 -->
      <template v-if="tab === 'inbox'">
        <div class="module-title">【收信箱】<a class="rt" href="javascript:;" @click="loadInbox">刷新</a></div>
        <ul class="dtuser" v-if="inbox.length">
          <li v-for="m in inbox" :key="'i'+m.id">
            <span v-if="!m.is_read" class="tag" style="background:#e05a00">新</span>
            <a href="javascript:;" @click="openChat(m.sender_id, m.sender)"><font :color="m.color || '#004299'">{{ m.sender }}</font></a>
            <span class="txt-fade">（{{ fmt(m.created_at) }}）</span><br>
            <em>{{ m.content.length > 40 ? m.content.slice(0, 40) + '…' : m.content }}</em>
            <a class="rt" href="javascript:;" @click="openChat(m.sender_id, m.sender)">回复&gt;&gt;</a>
          </li>
        </ul>
        <div class="module-content" v-else><span class="empty">您没有收信消息</span></div>
        <div class="module-content" v-if="inboxTotal > inbox.length"><a href="javascript:;" @click="inboxMore">下页&gt;&gt;</a></div>
      </template>

      <!-- 发信箱 -->
      <template v-else-if="tab === 'outbox'">
        <div class="module-title">【发信箱】<a class="rt" href="javascript:;" @click="loadOutbox">刷新</a></div>
        <ul class="dtuser" v-if="outbox.length">
          <li v-for="m in outbox" :key="'o'+m.id">
            <span v-if="m.is_read" class="txt-fade">(已阅)</span><span v-else class="txt-fade">(未阅)</span>
            <a href="javascript:;" @click="openChat(m.receiver_id, m.receiver)"><font :color="m.color || '#004299'">{{ m.receiver }}</font></a>
            <span class="txt-fade">（{{ fmt(m.created_at) }}）</span><br>
            <em>{{ m.content.length > 40 ? m.content.slice(0, 40) + '…' : m.content }}</em>
          </li>
        </ul>
        <div class="module-content" v-else><span class="empty">您没有发出信息</span></div>
        <div class="module-content" v-if="outboxTotal > outbox.length"><a href="javascript:;" @click="outboxMore">下页&gt;&gt;</a></div>
      </template>

      <!-- 系统信息 -->
      <template v-else-if="tab === 'sys'">
        <div class="module-title">【系统信息】<a class="rt" href="javascript:;" @click="loadSys">刷新</a></div>
        <ul class="dtuser" v-if="sys.length">
          <li v-for="n in sys" :key="'n'+n.id">
            <template v-if="n.type === 'friend'"><font color="#e05a00">[好友]</font></template>
            <template v-else-if="n.type === 'reply'"><font color="#1a9e1a">[回复]</font></template>
            <template v-else><font color="#004299">[系统]</font></template>
            {{ n.title }}<span class="txt-fade">（{{ fmt(n.created_at) }}）</span><br>
            <em>{{ n.content }}</em>
            <a class="rt" href="javascript:;" @click="readAll">全部已读</a>
          </li>
        </ul>
        <div class="module-content" v-else><span class="empty">暂无系统信息</span></div>
      </template>

      <!-- 功能导航 -->
      <div class="module-title">【功能导航】</div>
      <div class="module-content">
        <a href="javascript:;" @click="$router.push('/friends')">好友列表</a>.<a href="javascript:;" @click="$router.push('/friends?tab=recent')">最近联系</a>.<a href="javascript:;" @click="$router.push('/search')">好友查找</a><br>
      </div>
      <div class="bar"><a href="javascript:;" @click="$router.push('/home')">我的地盘</a>&gt;信箱<br></div>
    </template>

    <!-- ===== 与某人的往来（对齐诺哈 chat.asp） ===== -->
    <template v-else>
      <div class="bar">
        <a href="javascript:;" @click="$router.push('/messages')">返回信箱</a> | 与 <font :color="peer.color || '#004299'">{{ peer.nickname }}</font> 的往来
      </div>
      <div style="max-height:50vh;overflow-y:auto;padding:4px" ref="box">
        <div v-for="m in msgs" :key="m.id" class="floor-item" style="margin:0 4px">
          <div class="fl">
            <b :style="{color: m.sender_id === myId ? '#1a6fae' : (m.sender ? m.sender.color : '#333')}">
              {{ m.sender ? m.sender.nickname : '?' }}
            </b>
            {{ fmt(m.created_at) }}
          </div>
          <div class="cnt">{{ m.content }}</div>
        </div>
        <div v-if="!msgs.length" class="empty">还没有消息，说点什么吧</div>
      </div>
      <div class="module-content">
        <form @submit.prevent="send">
          <div class="form-item">
            <textarea v-model.trim="content" maxlength="500" style="min-height:60px" placeholder="家信内容（500字以内）"></textarea>
          </div>
          <div class="form-item"><button class="btn" type="submit">发 送</button></div>
        </form>
      </div>
    </template>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Messages',
  data () {
    return {
      tab: 'inbox', peer: null, msgs: [], content: '',
      inbox: [], inboxTotal: 0, inboxPage: 1, inboxUnread: 0,
      outbox: [], outboxTotal: 0, outboxPage: 1,
      sys: [], sysUnread: 0
    }
  },
  computed: { myId () { return this.$store.state.user ? this.$store.state.user.id : 0 } },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  methods: {
    load () {
      const peerId = this.$route.params.peerId
      if (peerId) {
        api.get('/messages/with/' + peerId).then(r => {
          if (r.code === 0) {
            this.peer = r.data.peer
            this.msgs = r.data.list
            this.$nextTick(() => {
              const box = this.$refs.box
              if (box) box.scrollTop = box.scrollHeight
            })
          }
        })
        return
      }
      this.peer = null
      this.loadInbox()
      this.loadOutbox()
      this.loadSys()
    },
    loadInbox () {
      api.get('/messages/inbox', { params: { page: this.inboxPage } }).then(r => {
        if (r.code === 0) { this.inbox = r.data.list; this.inboxTotal = r.data.total }
      })
    },
    inboxMore () { this.inboxPage++; this.loadInbox() },
    loadOutbox () {
      api.get('/messages/outbox', { params: { page: this.outboxPage } }).then(r => {
        if (r.code === 0) { this.outbox = r.data.list; this.outboxTotal = r.data.total }
      })
    },
    outboxMore () { this.outboxPage++; this.loadOutbox() },
    loadSys () {
      api.get('/notifications').then(r => {
        if (r.code === 0) { this.sys = r.data.list || []; this.sysUnread = r.data.unread || 0 }
      })
    },
    readAll () {
      api.post('/notifications/read-all').then(r => { if (r.code === 0) this.loadSys() })
    },
    openChat (uid, name) {
      this.$router.push('/messages/' + uid)
    },
    send () {
      if (!this.content) return
      api.post('/messages', { to: parseInt(this.$route.params.peerId), content: this.content }).then(r => {
        if (r.code === 0) {
          this.content = ''
          this.load()
        } else alert(r.msg)
      })
    },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t)
      const p = n => (n < 10 ? '0' + n : n)
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes())
    }
  }
}
</script>