<template>
  <div>
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/channel/1')">论坛</a>&gt;<a v-if="board.name" href="javascript:;" @click="$router.push('/board/'+board.id)">{{ board.name }}</a>
    </div>
    <div class="module-content">
      <form @submit.prevent="submit">
        主题(2-30字):<br>
        <textarea v-model.trim="title" rows="2" maxlength="30"></textarea><br>
        内容(5-10000字):
        <textarea v-model="content" rows="15" maxlength="10000" ref="contentBox"></textarea><br>
        <button class="btn" type="submit" :disabled="sending">确定发表</button><br><br>
        <button class="btn gray" type="button" @click="showFace = !showFace">表情</button>
        <button class="btn gray" type="button" @click="append('\n')">换行</button>
        <button class="btn gray" type="button" @click="$router.push('/nav')">UBB指南</button>
      </form>
      <div v-if="showFace" class="deep" style="padding:5px;margin-top:5px">
        【选择表情】点击插入<br>
        <button v-for="f in faces" :key="f" type="button" class="face-btn" @click="append('/' + f)">/{{ f }}</button>
      </div>
      <p v-if="err" style="color:#c00">{{ err }}</p>
    </div>
    <div class="module-content">
      如需添加投票/回帖/踩楼奖励等，请先保存为草稿（功能开发中）。<br>
      返回<a href="javascript:;" @click="$router.push('/board/'+(board.id||1))">主题帖列表</a>|<a href="javascript:;" @click="$router.push('/nav')">UBB指南</a><br>
      《<a href="javascript:;" @click="$router.push('/channel/4')">3GQQ家园论坛公约协议</a>》
    </div>
  </div>
</template>

<script>
import api from '../api'
import { FACE_NAMES } from '../utils/qqface'

export default {
  name: 'PostEdit',
  data () {
    return {
      channels: [], board: {},
      title: '', content: '', err: '', sending: false,
      showFace: false, faces: FACE_NAMES
    }
  },
  watch: { '$route.params.boardId': 'loadBoard' },
  mounted () { this.loadBoard() },
  methods: {
    loadBoard () {
      api.get('/boards').then(r => {
        if (r.code === 0) {
          const all = []
          r.data.forEach(ch => (ch.children || []).forEach(b => all.push(b)))
          this.board = all.find(b => b.id === parseInt(this.$route.params.boardId)) || {}
        }
      })
    },
    append (text) {
      this.content += text
      const el = this.$refs.contentBox
      if (el) el.focus()
    },
    submit () {
      this.err = ''
      if (!this.board.id) { this.err = '请选择一个子板块'; return }
      if (this.title.length < 2) { this.err = '主题至少2个字'; return }
      if (this.content.length < 5) { this.err = '内容至少5个字'; return }
      this.sending = true
      api.post(`/boards/${this.board.id}/threads`, { title: this.title, content: this.content }).then(r => {
        this.sending = false
        if (r.code === 0) {
          this.$router.push('/thread/' + r.data.id)
        } else {
          this.err = r.msg
        }
      })
    }
  }
}
</script>
