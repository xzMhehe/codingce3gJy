<template>
  <div>
    <div class="bar">【社区聊天室】<a class="rt" href="javascript:;" @click="$router.push('/')">回广场</a></div>
    <div class="login-tips">欢迎来到聊天室！这里是全园友友的公共客厅，请文明发言~（每3秒自动刷新）</div>
    <div style="max-height:55vh;overflow-y:auto;padding:4px" ref="box">
      <div v-for="m in msgs" :key="m.id" class="floor-item">
        <div class="fl">
          <a href="javascript:;" @click="$router.push('/user/'+(m.user?m.user.id:0))"><font :color="m.user?m.user.color:''">{{ m.user?m.user.nickname:'路人' }}</font></a>
          （{{ m.user?m.user.username:'' }}）{{ fmt(m.created_at) }}
        </div>
        <div class="cnt">{{ m.content }}</div>
      </div>
      <div v-if="!msgs.length" class="empty">聊天室很安静，来打破沉默吧</div>
    </div>
    <div class="module-content">
      <form @submit.prevent="send">
        <div class="form-item">
          <input type="text" v-model.trim="content" maxlength="500" style="width:100%;padding:6px;border:1px solid #9FC6EC;border-radius:3px" placeholder="说点什么吧…">
        </div>
        <div class="form-item"><button class="btn" type="submit">发 言</button></div>
      </form>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'ChatRoom',
  data () { return { msgs: [], content: '', timer: null } },
  mounted () {
    this.load()
    this.timer = setInterval(this.load, 3000)
  },
  methods: {
    load () {
      const after = this.msgs.length ? this.msgs[this.msgs.length - 1].id : 0
      api.get('/chat', { params: { after } }).then(r => {
        if (r.code === 0 && r.data.length) {
          this.msgs = this.msgs.concat(r.data).slice(-200)
          this.$nextTick(() => {
            const box = this.$refs.box
            if (box) box.scrollTop = box.scrollHeight
          })
        }
      })
    },
    send () {
      if (!this.content) return
      api.post('/chat', { content: this.content }).then(r => {
        if (r.code === 0) {
          this.content = ''
          this.load()
        } else alert(r.msg)
      })
    },
    fmt (t) {
      const d = new Date(t)
      const p = n => (n < 10 ? '0' + n : n)
      return p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds())
    }
  },
  beforeDestroy () { clearInterval(this.timer) }
}
</script>
