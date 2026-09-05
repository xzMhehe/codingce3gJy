<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;<a href="javascript:;" @click="$router.push('/box')">我的百宝箱</a>&gt;我的头像</div>

    <div class="module-title">【我的头像】</div>
    <div class="module-content">
      当前头像:<br>
      <img :src="currentFace" width="150" height="150" alt="."><br>
    </div>

    <div class="module-title">自定义头像上传:</div>
    <div class="module-content">
      <input type="file" accept="image/*" @change="onFile">
      <span class="txt-fade">（图片尺寸应为 150*150 以上，最大 500KB）</span><br>
      <img v-if="preview" :src="preview" width="80" height="80" style="vertical-align:middle;margin:6px 0" alt="预览">
      <br>
      <a href="javascript:;" class="ipt-btn-gold-b" :class="{ disabled: !preview }" @click="upload">确定上传</a>
      <p v-if="msg" style="color:#c00">{{ msg }}</p>
      <p v-if="okMsg" style="color:#1a9e1a">{{ okMsg }}</p>
    </div>

    <div class="module-title">推荐头像（点击图片即可设为头像）</div>
    <div class="module-content">
      <a v-for="f in presets" :key="f" href="javascript:;" @click="setPreset(f)" :title="f">
        <img :src="'/static/picture/' + f" width="60" height="60" style="margin:3px;border:1px solid #eee" alt=".">
      </a>
      <div v-if="total > 0" style="margin-top:6px">
        <a v-if="page > 1" href="javascript:;" @click="go(page - 1)">上一页</a>
        <span class="txt-fade"> (第 <b>{{ page }}</b>/{{ pages }}页/共{{ total }}条记录) </span>
        <a v-if="page < pages" href="javascript:;" @click="go(page + 1)">下页</a>
      </div>
    </div>

    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;<a href="javascript:;" @click="$router.push('/box')">我的百宝箱</a>&gt;头像&gt;上传</div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Face',
  data () {
    return {
      face: '',        // 当前头像：base64(data URI) 或文件名
      preview: '',     // 待上传预览
      msg: '', okMsg: '',
      presets: [], page: 1, size: 12, total: 0
    }
  },
  computed: {
    currentFace () {
      if (this.face) {
        return this.face.indexOf('data:') === 0 ? this.face : '/static/picture/' + this.face
      }
      return '/static/picture/0.gif'
    },
    pages () { return Math.max(1, Math.ceil(this.total / this.size)) }
  },
  mounted () {
    this.loadFace()
    this.loadPresets()
  },
  methods: {
    loadFace () {
      api.get('/me/avatar').then(r => {
        if (r.code === 0) {
          const b = r.data.avatar_base64
          const a = r.data.avatar
          this.face = (b && b.length > 20) ? b : a
        }
      }).catch(() => {})
    },
    loadPresets () {
      api.get('/avatar/presets', { params: { page: this.page, size: this.size } }).then(r => {
        if (r.code === 0) { this.presets = r.data.list || []; this.total = r.data.total || 0 }
      }).catch(() => {})
    },
    onFile (e) {
      this.msg = ''; this.okMsg = ''
      const file = e.target.files && e.target.files[0]
      if (!file) { this.preview = ''; return }
      if (file.size > 500 * 1024) {
        this.msg = '图片不能超过 500KB，请压缩后上传'
        this.preview = ''
        e.target.value = ''
        return
      }
      const reader = new FileReader()
      reader.onload = ev => { this.preview = ev.target.result }
      reader.readAsDataURL(file)
    },
    upload () {
      if (!this.preview) { this.msg = '请先选择图片'; return }
      api.post('/me/avatar', { data: this.preview }).then(r => {
        if (r.code === 0) {
          this.face = this.preview
          this.okMsg = '上传成功，头像已更新'
          this.msg = ''
          this.preview = ''
        } else {
          this.msg = r.msg || '上传失败'
        }
      }).catch(() => { this.msg = '上传失败，请重试' })
    },
    setPreset (f) {
      api.post('/avatar/presets', { file: f }).then(r => {
        if (r.code === 0) {
          this.face = f
          this.okMsg = '已设置头像：' + f
          this.msg = ''
        } else {
          this.msg = r.msg || '设置失败'
        }
      }).catch(() => { this.msg = '设置失败，请重试' })
    },
    go (p) {
      if (p < 1) return
      this.page = p
      this.loadPresets()
    }
  }
}
</script>
