<template>
  <div>
    <div class="bar">
      <a href="javascript:;" @click="$router.push('/channel/1')">论坛</a>&gt;<a v-if="board.name" href="javascript:;" @click="$router.push('/board/'+board.id)">{{ board.name }}</a>&gt;发帖
    </div>
    <div class="module-content">

      <!-- 帖子类型栏（复刻诺哈：普通帖.文件帖.图帖） -->
      <div class="ptype">
        <a href="javascript:;" :class="{ on: type === 0 }" @click="type = 0">普通帖</a>
        <a href="javascript:;" :class="{ on: type === 3 }" @click="type = 3">投票帖</a>
        <a href="javascript:;" :class="{ on: type === 1 }" @click="type = 1">回帖奖励</a>
        <a href="javascript:;" :class="{ on: type === 2 }" @click="type = 2">踩楼帖</a>
        <a href="javascript:;" :class="{ on: type === 5 }" @click="type = 5">图帖</a>
        <a href="javascript:;" :class="{ on: type === 6 }" @click="type = 6">文件帖</a>
      </div>

      <form @submit.prevent="submit">
        <template v-if="type === 1">
          回帖奖励：每次回帖 +<input type="text" v-model.number="rewardCoins" size="4"> G币、+<input type="text" v-model.number="rewardExp" size="4"> 经验，最多 <input type="text" v-model.number="rewardLimit" size="4"> 次（0=不限，每人限1次）<br>
        </template>
        <template v-if="type === 2">
          踩楼奖励：踩中指定楼层得奖励（每行一条：楼层|G币|经验，如 5|8|4）<br>
          <textarea v-model.trim="floorsText" rows="3" style="width:96%"></textarea><br>
        </template>
        <template v-if="type === 3">
          投票标题：<input type="text" v-model.trim="pollQ" maxlength="200" style="width:60%">
          <label><input type="checkbox" v-model="pollMulti"> 多选</label><br>
          投票选项（每行一个）：<br>
          <textarea v-model.trim="pollOptionsText" rows="4" style="width:96%"></textarea><br>
        </template>
        <template v-if="type === 5 || type === 6">
          <template v-if="type === 5">
            从相册选择图片插入（选中的相片会作为图帖内容）：<br>
            <div v-if="photos.length">
              <span v-for="(p,i) in photos" :key="p.id" class="pick">
                <a href="javascript:;" @click="insertPhoto(p)">选择</a>{{ i+1 }}.<img :src="p.url" alt="" class="pickimg"><br>
              </span>
            </div>
            <div v-else class="empty">暂无相片。<a href="javascript:;" @click="$router.push('/space/'+myId)">去空间上传</a><br></div>
          </template>
          <template v-else>
            选择要上传的附件（或填写一个附件链接）：<br>
            <input type="text" v-model.trim="attachName" placeholder="附件名称" style="width:40%">
            <input type="text" v-model.trim="attachUrl" placeholder="附件地址（/api/... 或 http://）" style="width:70%"><br>
            <span class="help-line">（付费下载可选）价格：</span><input type="text" v-model.number="attachPrice" size="4" placeholder="0">
            <span class="help-line">G币</span><br>
          </template>
        </template>

        主题(30字内):<br>
        <textarea v-model.trim="title" rows="2" maxlength="30"></textarea><br>
        内容(25-4000字):<br>
        <textarea v-model="content" rows="15" maxlength="4000" ref="contentBox"></textarea><br>
        <select v-model="draft" style="vertical-align:middle">
          <option :value="0">正常</option>
          <option :value="1">草稿</option>
        </select>
        <input type="submit" class="btn" value="确定发表" :disabled="sending"><br>
        <br>
        <!-- 工具栏（复刻诺哈：换行/表情/粘贴/相片/文件） -->
        <input type="button" class="btn gray" value="换行" @click="append('\r\n')">
        <input type="button" class="btn gray" value="表情" @click="panel='face'">
        <input type="button" class="btn gray" value="粘贴" @click="openPaste">
        <input type="button" class="btn gray" value="相片" @click="openPhoto">
        <input type="button" class="btn gray" value="文件" @click="type = 6">
      </form>

      <!-- 表情面板 -->
      <div v-if="panel === 'face'" class="deep" style="padding:5px;margin-top:5px">
        【选择表情】点击插入（最多5个，已用{{ faceCount }}/5）<br>
        <button v-for="f in faces" :key="f" type="button" class="face-btn" @click="insertFace(f)" :disabled="faceCount >= 5">{{ '/' + f }}</button>
        <br>----------<br>
        <button type="button" class="btn gray" @click="panel=''">返回发帖</button>
      </div>

      <!-- 粘贴面板 -->
      <div v-if="panel === 'paste'" class="deep" style="padding:5px;margin-top:5px">
        【选择粘贴】（我收藏的帖子，插入为链接）<br>
        <span v-for="(f,i) in pasteItems" :key="f.id">
          <form style="display:inline" @submit.prevent="insertPaste(f)">{{ i+1 }}.<a href="javascript:;" @click="$router.push('/thread/'+f.thread_id)">[帖子] {{ (f.thread && f.thread.title) || '帖子' }}</a>
          <input type="submit" value="粘贴" class="btn small"></form><br>
        </span>
        <span v-if="!pasteItems.length" class="empty">暂无收藏，先去收藏几篇帖子吧。</span>
        <br>----------<br>
        <button type="button" class="btn gray" @click="panel=''">返回发帖</button>
      </div>

      <p v-if="err" style="color:#c00">{{ err }}</p>
    </div>
    <div class="module-content">
      版块公告：<span v-html="board.notice ? renderLine(board.notice) : '（无）'"></span><br>
      <a href="javascript:;" @click="$router.push('/board/'+(board.id||1))">返回本版</a><br>
      <a href="javascript:;" @click="$router.push('/')">社区广场</a>&gt;<a href="javascript:;" @click="$router.push('/channel/1')">论坛</a>&gt;发帖<br>
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
      panel: '', faces: FACE_NAMES,
      type: 0, draft: 0,
      rewardCoins: 5, rewardExp: 3, rewardLimit: 10,
      floorsText: '',
      pollQ: '', pollMulti: false, pollOptionsText: '',
      attachName: '', attachUrl: '', attachPrice: 0,
      photos: [], pasteItems: []
    }
  },
  computed: {
    myId () { return this.$store.state.user ? this.$store.state.user.id : 0 },
    faceCount () {
      let n = 0
      this.faces.forEach(f => { n += (this.content.split('/' + f).length - 1) })
      return n
    }
  },
  watch: { '$route.params.boardId': 'loadBoard' },
  mounted () { this.loadBoard() },
  methods: {
    renderLine (t) { return (t || '').replace(/\n/g, '<br>') },
    loadBoard () {
      api.get('/boards').then(r => {
        if (r.code === 0) {
          const all = []
          r.data.forEach(ch => {
            ;(ch.children || []).forEach(b => all.push(b))
            ;(ch.categories || []).forEach(cat => (cat.boards || []).forEach(b => all.push(b)))
          })
          this.board = all.find(b => b.id === parseInt(this.$route.params.boardId)) || {}
        }
      })
    },
    append (text) {
      this.content += text
      const el = this.$refs.contentBox
      if (el) el.focus()
    },
    insertFace (f) {
      if (this.faceCount >= 5) return
      this.append('/' + f)
    },
    openPaste () {
      this.panel = 'paste'
      if (!this.pasteItems.length) {
        api.get('/favorite-threads').then(r => { if (r.code === 0) this.pasteItems = r.data || [] })
      }
    },
    insertPaste (f) {
      this.append('[帖子]' + (f.thread && f.thread.title || '帖子') + ': /thread/' + f.thread_id)
      this.panel = ''
    },
    openPhoto () {
      if (this.type !== 5) this.type = 5
      this.panel = ''
      if (!this.photos.length && this.myId) {
        api.get('/space/' + this.myId + '/albums').then(r => {
          if (r.code === 0) {
            const albums = r.data || []
            this.photos = []
            albums.forEach(a => (a.photos || []).forEach(p => this.photos.push({ id: p.id, url: p.url || '/static/picture/' + p.file })))
          }
        }).catch(() => {})
      }
    },
    insertPhoto (p) {
      this.append('【图】')
      this.panel = ''
    },
    submit () {
      this.err = ''
      if (!this.board.id) { this.err = '请选择一个子板块'; return }
      if (this.title.length < 2) { this.err = '主题至少2个字'; return }
      if (this.content.length < 5) { this.err = '内容至少5个字'; return }
      if (this.type === 5 || this.type === 6) {
        // 图帖/文件帖：把内容转成对应的附件标记
      }
      const payload = { title: this.title, content: this.content, type: (this.type === 5 || this.type === 6) ? 0 : this.type }
      if (this.type === 1) {
        payload.reward_coins = this.rewardCoins
        payload.reward_exp = this.rewardExp
        payload.reward_limit = this.rewardLimit
      }
      if (this.type === 2) {
        const floors = []
        this.floorsText.split('\n').forEach(line => {
          const p = line.split('|')
          if (p.length >= 2 && parseInt(p[0])) {
            floors.push({ floor: parseInt(p[0]), coins: parseInt(p[1]) || 0, exp: parseInt(p[2]) || 0 })
          }
        })
        if (!floors.length) { this.err = '请填写踩楼奖励（格式：楼层|G币|经验）'; return }
        payload.floors = floors
      }
      if (this.type === 3) {
        if (!this.pollQ) { this.err = '请填写投票标题'; return }
        const opts = this.pollOptionsText.split('\n').map(s => s.trim()).filter(Boolean)
        if (opts.length < 2) { this.err = '投票至少2个选项（每行一个）'; return }
        payload.poll_question = this.pollQ
        payload.poll_options = opts
        payload.poll_multiple = this.pollMulti
      }
      if (this.type === 6 && this.attachUrl) {
        payload.attachments = [{ name: this.attachName || this.attachUrl, path: this.attachUrl, size: 0, price: this.attachPrice || 0 }]
      }
      if (this.draft === 1) {
        // 草稿：简化处理，暂提示（后端未建草稿表，先按正常发布）
        this.err = '草稿功能暂未开通，请先正常发表。'
        return
      }
      this.sending = true
      api.post(`/boards/${this.board.id}/threads`, payload).then(r => {
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

<style scoped>
.ptype a { margin-right: 10px; color: #0051A4; }
.ptype a.on { font-weight: bold; color: #c00; }
.pick { display: inline-block; margin: 4px; }
.pickimg { width: 60px; height: 60px; object-fit: cover; vertical-align: middle; border: 1px solid #ccc; }
.btn.small { font-size: 12px; padding: 1px 4px; }
</style>