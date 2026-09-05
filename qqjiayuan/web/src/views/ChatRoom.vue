<template>
  <div>
    <div class="bar">
      【{{ isFamily ? '家族聊室' : '社区聊天室' }}】
      <a v-if="isFamily" href="javascript:;" @click="$router.push('/family/'+familyId)">回家族主页</a>
      <a v-else class="rt" href="javascript:;" @click="$router.push('/')">回广场</a>
    </div>
    <div class="login-tips" v-if="isFamily">欢迎来到家族聊室！只有家族成员可以发言（每3秒自动刷新）</div>
    <div class="login-tips" v-else>欢迎来到聊天室！这里是全园友友的公共客厅，请文明发言~（每3秒自动刷新）</div>
    <div style="max-height:55vh;overflow-y:auto;padding:4px" ref="box">
      <div v-for="m in msgs" :key="m.id" class="floor-item">
        <div class="fl">
          <a href="javascript:;" @click="$router.push('/user/'+(m.user?m.user.id:0))"><font :color="m.user?m.user.color:''">{{ m.user?m.user.nickname:'路人' }}</font></a>
          （{{ m.user?m.user.username:'' }}）{{ fmt(m.created_at) }}
        </div>
        <div class="cnt">{{ m.content }}</div>
      </div>
      <div v-if="!msgs.length" class="empty">{{ isFamily ? '家族聊室很安静，来打个招呼吧' : '聊天室很安静，来打破沉默吧' }}</div>
    </div>
    <div class="module-content" v-if="!isFamily || isMember">
      <form @submit.prevent="send">
        <div class="form-item">
          <input type="text" v-model.trim="content" maxlength="500" style="width:100%;padding:6px;border:1px solid #9FC6EC;border-radius:3px" placeholder="说点什么吧…">
        </div>
        <div class="form-item"><button class="btn" type="submit">发 言</button></div>
      </form>
    </div>
    <div class="module-content" v-else>
      <span class="empty">你还不是本家族成员，<a href="javascript:;" @click="$router.push('/family/'+familyId)">去加入后才能发言</a></span>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'ChatRoom',
  data () { return { msgs: [], content: '', timer: null, famRole: '' } },
  computed: {
    familyId () { return parseInt(this.$route.query.family_id || 0) },
    isFamily () { return this.familyId > 0 },
    isMember () { return this.famRole !== '' }
  },
  watch: { 'familyId': 'reload' },
  mounted () { this.reload() },
  beforeDestroy () { clearInterval(this.timer) },
  methods: {
    reload () {
      clearInterval(this.timer)
      this.msgs = []
      this.famRole = ''
      if (this.isFamily) {
        api.get('/families/' + this.familyId).then(r => {
          if (r.code === 0) this.famRole = r.data.my_role || ''
        })
      }
      this.load()
      this.timer = setInterval(this.load, 3000)
    },
    load () {
      const after = this.msgs.length ? this.msgs[this.msgs.length - 1].id : 0
      api.get('/chat', { params: { after, family_id: this.familyId } }).then(r => {
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
      api.post('/chat?family_id=' + this.familyId, { content: this.content }).then(r => {
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
  }
}
</script>
