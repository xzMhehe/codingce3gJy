<template>
  <div>
    <!-- ===== 信箱首页（复刻诺哈 chat/index.asp：基本设施 + 功能导航） ===== -->
    <template v-if="!peer && !sendView">
      <div class="bar"><a href="javascript:;" @click="$router.push('/home')">我的地盘</a>&gt;信箱<br></div>

      <div class="name">【基本设施】</div>
      <div class="module-content">
        <a href="javascript:;" @click="tab='inbox'">收信箱({{ inboxUnread }})</a><br>
        <a href="javascript:;" @click="tab='outbox'">发信箱</a><br>
        <a href="javascript:;" @click="tab='sys'">系统信息({{ sysUnread }})</a><br>
      </div>

      <!-- 收信箱（复刻诺哈 inbox.asp） -->
      <template v-if="tab === 'inbox'">
        <div class="name">【收信箱】<a class="rt" href="javascript:;" @click="loadInbox">刷新</a></div>
        <div class="list" v-if="inbox.length">
          <div class="row" v-for="(m, i) in inbox" :key="'i'+m.id">
            {{ inboxPageStart + i }}.<template v-if="!m.is_read"><span class="txt-fade">(新)</span></template>
            <a href="javascript:;" @click="openChat(m.sender_id)">{{ preview(m.content) }}</a><br>
            发信人:<a href="javascript:;" @click="$router.push('/user/'+m.sender_id)"><font :color="m.color || '#004299'">{{ m.sender }}</font>({{ m.sender_id }})</a><br>
            收信时间:{{ fmt(m.created_at) }} [<a href="javascript:;" @click="delMsg(m)">删</a>]<br>
          </div>
        </div>
        <div class="module-content" v-else><span class="empty">您没有收信消息。</span></div>

        <!-- 分页（诺哈：下页.上页 + 第X/Y页/共N记录） -->
        <div class="ppage">
          <a v-if="inboxPage < inboxPages" href="javascript:;" @click="inboxGo(inboxPage+1)">下页</a><template v-if="inboxPage < inboxPages">.</template>
          <a v-if="inboxPage > 1" href="javascript:;" @click="inboxGo(inboxPage-1)">上页</a><template v-if="inboxPage > 1">.</template>
          <br v-if="inboxPages > 1">
        </div>
        <div class="item" v-if="inboxPages > 1">
          (第<b>{{ inboxPage }}</b>/{{ inboxPages }}页/共{{ inboxTotal }}记录)<br>
        </div>
      </template>

      <!-- 发信箱（复刻诺哈 outbox.asp） -->
      <template v-else-if="tab === 'outbox'">
        <div class="name">【发信箱】<a class="rt" href="javascript:;" @click="loadOutbox">刷新</a></div>
        <div class="list" v-if="outbox.length">
          <div class="row" v-for="(m, i) in outbox" :key="'o'+m.id">
            {{ outboxPageStart + i }}.<span class="txt-fade">{{ m.is_read ? '(已阅)' : '(未阅)' }}</span>
            <a href="javascript:;" @click="openChat(m.receiver_id)">{{ preview(m.content) }}</a><br>
            收信人:<a href="javascript:;" @click="$router.push('/user/'+m.receiver_id)"><font :color="m.color || '#004299'">{{ m.receiver }}</font>({{ m.receiver_id }})</a><br>
            发信时间:{{ fmt(m.created_at) }} [<a href="javascript:;" @click="delMsg(m)">删</a>]<br>
          </div>
        </div>
        <div class="module-content" v-else><span class="empty">您没有发出信息。</span></div>

        <div class="ppage">
          <a v-if="outboxPage < outboxPages" href="javascript:;" @click="outboxGo(outboxPage+1)">下页</a><template v-if="outboxPage < outboxPages">.</template>
          <a v-if="outboxPage > 1" href="javascript:;" @click="outboxGo(outboxPage-1)">上页</a><template v-if="outboxPage > 1">.</template>
          <br v-if="outboxPages > 1">
        </div>
        <div class="item" v-if="outboxPages > 1">
          (第<b>{{ outboxPage }}</b>/{{ outboxPages }}页/共{{ outboxTotal }}记录)<br>
        </div>
      </template>

      <!-- 系统信息（复刻诺哈 sysbox.asp） -->
      <template v-else-if="tab === 'sys'">
        <div class="name">【系统信息】<a class="rt" href="javascript:;" @click="loadSys">刷新</a></div>
        <div class="list" v-if="sys.length">
          <div class="row" v-for="n in sys" :key="'n'+n.id">
            <template v-if="n.type === 'friend'"><font color="#e05a00">[好友]</font></template>
            <template v-else-if="n.type === 'reply'"><font color="#1a9e1a">[回复]</font></template>
            <template v-else><font color="#004299">[系统]</font></template>
            {{ n.title }}<span class="txt-fade">（{{ fmt(n.created_at) }}）</span><br>
            {{ n.content }}<br>
          </div>
        </div>
        <div class="module-content" v-else><span class="empty">暂无系统信息。</span></div>
        <div class="module-content" v-if="sysUnread > 0"><a href="javascript:;" @click="readAll">全部标记已读</a><br></div>
      </template>

      <!-- 功能导航（诺哈 chat/index.asp：发信息/清空信箱） -->
      <div class="name">【功能导航】</div>
      <div class="module-content">
        <a href="javascript:;" @click="openSend">发家信</a><br>
        <a href="javascript:;" @click="clearBox('inbox')">清空收信箱</a><br>
        <a href="javascript:;" @click="clearBox('outbox')">清空发信箱</a><br>
        <a href="javascript:;" @click="clearBox('all')">清空所有信息</a><br>
        <a href="javascript:;" @click="$router.push('/friends')">好友列表</a>.<a href="javascript:;" @click="$router.push('/friends?tab=recent')">最近联系</a>.<a href="javascript:;" @click="$router.push('/search')">好友查找</a><br>
      </div>

      <div class="bar"><a href="javascript:;" @click="$router.push('/home')">我的地盘</a>&gt;信箱<br></div>
    </template>

    <!-- ===== 与某人往来（复刻诺哈 chat.asp：昵称(号码) + 刷新 + 消息 + 发送） ===== -->
    <template v-else-if="peer">
      <div class="bar">
        <a href="javascript:;" @click="$router.push('/messages')">我的地盘</a>&gt;<a href="javascript:;" @click="$router.push('/messages')">信箱</a>&gt;聊天<br>
      </div>

      <div class="name">
        <a href="javascript:;" @click="$router.push('/user/'+peer.id)"><font :color="peer.color || '#004299'">{{ peer.nickname }}</font>({{ peer.id }})</a>
        <span class="txt-fade">(往来{{ msgs.length }}条)</span> <a class="rt" href="javascript:;" @click="load">刷新消息</a><br>
      </div>

      <div class="list">
        <div v-for="m in msgs" :key="m.id" class="row">
          <span class="txt-fade">{{ fmt(m.created_at) }}</span><br>
          <b :style="{color: m.sender_id === myId ? '#1a6fae' : (m.sender && m.sender.color ? m.sender.color : '#333')}">
            {{ m.sender_id === myId ? myName : (m.sender ? m.sender.nickname : '系统信息') }}
          </b><br>
          <span class="cnt">{{ m.content }}</span><br>
        </div>
        <div v-if="!msgs.length" class="row"><span class="empty">还没有消息，说点什么吧</span></div>
      </div>

      <div class="module-content">
        <form @submit.prevent="send">
          <textarea v-model.trim="content" rows="3" maxlength="500"></textarea><br>
          <input type="submit" value="发送"><br>
        </form>
      </div>

      <!-- 好友功能（诺哈 chat.asp：好友家园） -->
      <div class="name">【好友功能】</div>
      <div class="module-content">
        <a href="javascript:;" @click="$router.push('/user/'+peer.id)">好友家园</a><br>
      </div>

      <div class="bar">
        <a href="javascript:;" @click="$router.push('/messages')">我的地盘</a>&gt;<a href="javascript:;" @click="$router.push('/messages')">信箱</a>&gt;聊天<br>
      </div>
    </template>

    <!-- ===== 发家信（复刻诺哈 send.asp：输号码→写内容→发送） ===== -->
    <template v-else-if="sendView">
      <div class="bar">
        <a href="javascript:;" @click="$router.push('/home')">我的地盘</a>&gt;<a href="javascript:;" @click="closeSend">信箱</a>&gt;发家信<br>
      </div>

      <div class="name">【发家信】<br></div>
      <div class="module-content">
        <template v-if="!writePeer.id">
          接收人号码：<input type="text" v-model.number="writeTo" size="10"><br>
          <input type="submit" value="下一步" @click="checkPeer"><br>
        </template>
        <template v-else>
          收信人：<a href="javascript:;" @click="$router.push('/user/' + writePeer.id)"><font :color="writePeer.color || '#004299'">{{ writePeer.nickname }}</font>({{ writePeer.id }})</a><br>
          内容：<textarea v-model.trim="writeContent" rows="3" maxlength="500"></textarea><br>
          <input type="submit" value="发 送" @click="sendTo">　<a href="javascript:;" @click="resetSend">重选接收人</a><br>
        </template>
        <a href="javascript:;" @click="closeSend">返回信箱</a><br>
      </div>

      <div class="bar">
        <a href="javascript:;" @click="$router.push('/home')">我的地盘</a>&gt;<a href="javascript:;" @click="closeSend">信箱</a>&gt;发家信<br>
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
      sendView: false, writeTo: '', writePeer: {}, writeContent: '',
      pollTimer: null,
      inbox: [], inboxTotal: 0, inboxPage: 1, inboxPages: 1, inboxUnread: 0,
      outbox: [], outboxTotal: 0, outboxPage: 1, outboxPages: 1,
      sys: [], sysUnread: 0
    }
  },
  computed: {
    myId () { return this.$store.state.user ? this.$store.state.user.id : 0 },
    myName () { return this.$store.state.user ? this.$store.state.user.nickname : '我' },
    inboxPageStart () { return (this.inboxPage - 1) * 10 + 1 },
    outboxPageStart () { return (this.outboxPage - 1) * 10 + 1 }
  },
  watch: { '$route': 'load' },
  mounted () { this.load() },
  beforeDestroy () { this.stopPoll() },
  methods: {
    load () {
      const peerId = this.$route.params.peerId
      if (peerId) {
        api.get('/messages/with/' + peerId).then(r => {
          if (r.code === 0) {
            this.peer = r.data.peer
            const list = r.data.list || []
            // 有新消息才刷新全局未读数（top_nav 家信(N)），避免轮询空转
            if (list.length !== this.msgs.length) this.$store.dispatch('refreshUnread')
            this.msgs = list
          }
        })
        this.startPoll()
        return
      }
      this.stopPoll()
      this.peer = null
      this.loadInbox()
      this.loadOutbox()
      this.loadSys()
    },
    // 聊天视图 5 秒轮询（优化：诺哈需手动刷新，这里自动收信）
    startPoll () {
      this.stopPoll()
      this.pollTimer = setInterval(() => {
        if (this.$route.params.peerId) this.load()
        else this.stopPoll()
      }, 5000)
    },
    stopPoll () {
      if (this.pollTimer) { clearInterval(this.pollTimer); this.pollTimer = null }
    },
    loadInbox () {
      api.get('/messages/inbox', { params: { page: this.inboxPage } }).then(r => {
        if (r.code === 0) {
          this.inbox = r.data.list || []
          this.inboxTotal = r.data.total || 0
          this.inboxPages = Math.max(1, Math.ceil(this.inboxTotal / (r.data.size || 10)))
        }
      })
      // 未读数取全量（私信未读），不用当前页
      api.get('/messages/unread').then(r => { if (r.code === 0) this.inboxUnread = r.data.unread || 0 })
    },
    inboxGo (p) {
      if (p < 1) return
      this.inboxPage = p
      this.loadInbox()
    },
    loadOutbox () {
      api.get('/messages/outbox', { params: { page: this.outboxPage } }).then(r => {
        if (r.code === 0) {
          this.outbox = r.data.list || []
          this.outboxTotal = r.data.total || 0
          this.outboxPages = Math.max(1, Math.ceil(this.outboxTotal / (r.data.size || 10)))
        }
      })
    },
    outboxGo (p) {
      if (p < 1) return
      this.outboxPage = p
      this.loadOutbox()
    },
    loadSys () {
      api.get('/notifications').then(r => {
        if (r.code === 0) { this.sys = r.data.list || []; this.sysUnread = r.data.unread || 0 }
      })
    },
    readAll () {
      // 全部标记已读：通知 + 私信，随后刷新家信未读数
      Promise.all([
        api.post('/notifications/read-all'),
        api.post('/messages/read-all')
      ]).then(() => {
        this.loadSys()
        this.loadInbox()
        this.$store.dispatch('refreshUnread')
      })
    },
    clearBox (box) {
      const names = { inbox: '收信箱', outbox: '发信箱', all: '所有信息' }
      if (!confirm('确定清空' + names[box] + '吗？此操作不可恢复。')) return
      api.post('/messages/clear', { box }).then(r => {
        if (r.code === 0) {
          alert('已清空' + names[box])
          this.loadInbox()
          this.loadOutbox()
          this.$store.dispatch('refreshUnread')
        } else alert(r.msg)
      })
    },
    openChat (uid) {
      this.$router.push('/messages/' + uid)
    },
    send () {
      if (!this.content) return
      api.post('/messages', { to: parseInt(this.$route.params.peerId), content: this.content }).then(r => {
        if (r.code === 0) { this.content = ''; this.load() } else alert(r.msg)
      })
    },
    preview (s) {
      s = s || ''
      return s.length > 15 ? s.slice(0, 15) + '…' : s
    },
    // ---- 发家信（复刻诺哈 send.asp 两步流：输号码→确认收信人→写内容发送） ----
    openSend () {
      this.sendView = true
      this.writeTo = ''
      this.writePeer = {}
      this.writeContent = ''
      this.stopPoll()
    },
    closeSend () { this.sendView = false },
    checkPeer () {
      if (!this.writeTo) { alert('请输入对方号码'); return }
      api.get('/users/' + this.writeTo).then(r => {
        if (r.code === 0) this.writePeer = { id: r.data.id, nickname: r.data.nickname, color: r.data.color }
        else alert(r.msg || '这位友友不存在')
      })
    },
    resetSend () { this.writePeer = {}; this.writeContent = '' },
    sendTo () {
      if (!this.writeContent) { alert('请填写家信内容'); return }
      api.post('/messages', { to: this.writePeer.id, content: this.writeContent }).then(r => {
        if (r.code === 0) {
          alert('家信发送成功！')
          this.resetSend()
          this.sendView = false
          this.loadOutbox()
        } else alert(r.msg)
      })
    },
    // ---- 单条删除（复刻诺哈 message_del.asp） ----
    delMsg (m) {
      if (!confirm('删除这条家信？')) return
      api.delete('/messages/' + m.id).then(r => {
        if (r.code === 0) {
          this.loadInbox()
          this.loadOutbox()
          this.$store.dispatch('refreshUnread')
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