<template>
  <div>
    <div class="bar">【私信】<a class="rt" href="javascript:;" @click="$router.push('/friends')">好友列表</a></div>

    <!-- 会话列表 -->
    <template v-if="!peer">
      <div class="module-title">【会话列表】</div>
      <ul class="dtuser" v-if="convs.length">
        <li v-for="cv in convs" :key="cv.user_id">
          <a href="javascript:;" @click="$router.push('/messages/'+cv.user_id)"><font :color="cv.color || '#004299'">{{ cv.nickname }}</font></a>
          <span v-if="cv.unread" class="tag" style="background:#e05a00">{{ cv.unread }}未读</span><br>
          <em>{{ cv.last_content }}</em>
          <a class="rt" href="javascript:;" @click="$router.push('/messages/'+cv.user_id)">回复&gt;&gt;</a>
        </li>
      </ul>
      <div class="module-content" v-else><span class="empty">暂无私信，去好友页面给朋友发一条吧</span></div>
    </template>

    <!-- 与某人的对话 -->
    <template v-else>
      <div class="bar">
        <a href="javascript:;" @click="$router.push('/messages')">返回</a> | 与 <font :color="peer.color || '#fff'">{{ peer.nickname }}</font> 的私信
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
            <textarea v-model.trim="content" maxlength="500" style="min-height:60px" placeholder="私信内容（500字以内）"></textarea>
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
  data () { return { convs: [], peer: null, msgs: [], content: '' } },
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
      } else {
        this.peer = null
        api.get('/messages/conversations').then(r => {
          if (r.code === 0) this.convs = r.data
        })
      }
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
      const d = new Date(t)
      const p = n => (n < 10 ? '0' + n : n)
      return p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes())
    }
  }
}
</script>
