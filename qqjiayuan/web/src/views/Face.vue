<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;<a href="javascript:;" @click="$router.push('/box')">我的百宝箱</a>&gt;我的头像</div>

    <div class="note"></div>
    <div class="module-title">
      <a href="javascript:;" @click="$router.push('/box')">用户中心</a>|<a href="javascript:;" @click="$router.push('/security')">安全中心</a>|<a href="javascript:;" @click="$router.push('/wallet')">我的钱包</a>|<a href="javascript:;" @click="$router.push('/rank')">家园排行</a><br>
    </div>

    <div class="module-title">【我的头像】</div>
    <div class="module-content">
      当前头像:<br>
      <img v-if="currentFace" :src="currentFace" width="150" height="150" alt=".">
      <span v-else class="txt-fade">未设置</span><br>
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

    <div class="module-title">选项2.取QQ头像变为社区头像</div>
    <div class="module-content">
      输入QQ:<input type="text" v-model.trim="qq" emptyok="true" maxlength="12" style="width:120px">
      <a href="javascript:;" class="ipt-btn-gold-b" :class="{ disabled: !qq }" @click="setQq">确定设置</a>
      <p v-if="qqMsg" style="color:#c00">{{ qqMsg }}</p>
      <p v-if="qqOk" style="color:#1a9e1a">{{ qqOk }}</p>
    </div>

    <div class="module-title">推荐头像（点击图片即可设为头像）</div>
    <div class="module-content">
      <a v-for="f in presets" :key="f.file" href="javascript:;" @click="setPreset(f)" :title="f.name || f.file">
        <img :src="presetSrc(f)" width="60" height="60" style="margin:3px;border:1px solid #eee" alt=".">
      </a>
      <div v-if="!presets.length" class="txt-fade">暂无推荐头像</div>
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
      qq: '', qqMsg: '', qqOk: '',
      presets: []
    }
  },
  computed: {
    currentFace () {
      if (this.face) {
        return this.face.indexOf('data:') === 0 ? this.face : '/static/picture/' + this.face
      }
      return ''
    }
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
      api.get('/avatar/presets').then(r => {
        if (r.code === 0) { this.presets = r.data.list || [] }
      }).catch(() => {})
    },
    presetSrc (f) {
      return f.file.indexOf('db/') === 0 ? '/api/res/' + f.file : '/static/' + f.file
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
      api.post('/avatar/presets', { file: f.file }).then(r => {
        if (r.code === 0) {
          this.loadFace()
          this.okMsg = '已设置推荐头像'
          this.msg = ''
        } else {
          this.msg = r.msg || '设置失败'
        }
      }).catch(() => { this.msg = '设置失败，请重试' })
    },
    setQq () {
      this.qqMsg = ''; this.qqOk = ''
      if (!/^\d{5,12}$/.test(this.qq)) { this.qqMsg = '请输入正确的QQ号（5-12位数字）'; return }
      api.post('/me/avatar/qq', { qq: this.qq }).then(r => {
        if (r.code === 0) {
          this.loadFace()
          this.qqOk = '已获取QQ头像并设为社区头像'
        } else {
          this.qqMsg = r.msg || '获取QQ头像失败'
        }
      }).catch(() => { this.qqMsg = '获取QQ头像失败，请稍后再试' })
    }
  }
}
</script>
