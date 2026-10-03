<template>
  <div>
    <div class="bar">【社区登录】<br></div>
    <div class="module-content">
      <form @submit.prevent="doLogin">
        家园社区账号:<br><input type="text" v-model.trim="name" maxlength="50" required><br>
        家园社区密码:<br><input type="password" v-model.trim="pass" value="" maxlength="50" required><br>
        <template v-if="needCaptcha">
          验证码:<br>
          <input type="text" v-model.trim="captcha" maxlength="6" placeholder="输入计算结果" style="width:100px" required>
          <img v-if="captchaImg" :src="captchaImg" alt="验证码" title="点击切换" style="height:25px;vertical-align:middle;cursor:pointer" @click="loadCaptcha">
          <span style="color:red;font-size:12px;cursor:pointer" @click="loadCaptcha">看不清？点击切换</span><br>
        </template>
        <input type="submit" value="确定登录">
      </form>
      <p v-if="err" style="color:#c00">{{ err }}</p>
      <a href="javascript:;" @click="$router.push('/register')">免费注册</a>|<a href="javascript:;" @click="$router.push('/kefu')">客服中心</a><br>
      <a href="javascript:;" @click="$router.push('/find')">找回资料</a>|<a href="javascript:;" @click="$router.push('/kefu/terms')">注册声明</a><br>
      ----------<br>
      <br>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Login',
  data () {
    return { name: '', pass: '', captcha: '', captchaId: '', captchaImg: '', needCaptcha: false, err: '' }
  },
  methods: {
    // 拉一张新的算式验证码（点图片/点“看不清”换一张）
    loadCaptcha () {
      this.captcha = ''
      api.get('/auth/captcha').then(r => {
        if (r.code === 0) {
          this.captchaImg = r.data.img
          this.captchaId = r.data.id
        }
      })
    },
    doLogin () {
      this.err = ''
      const body = { name: this.name, password: this.pass }
      if (this.needCaptcha) {
        body.captcha_id = this.captchaId
        body.captcha = this.captcha
      }
      api.post('/auth/login', body).then(r => {
        if (r.code === 0) {
          this.needCaptcha = false
          this.$store.commit('setUser', { token: r.data.token, user: r.data.user })
          // 再拉取完整资料（含权限）
          api.get('/auth/me').then(m => {
            if (m.code === 0) this.$store.commit('setUser', { user: m.data })
            this.$router.push(this.$route.query.redirect || '/')
          })
        } else {
          this.err = r.msg
          if (r.data && r.data.need_captcha) {
            // 密码错误满 3 次，立即弹出验证码框
            this.needCaptcha = true
            this.loadCaptcha()
          } else if (this.needCaptcha) {
            // 验证码一次性，失败后换一张
            this.loadCaptcha()
          }
        }
      })
    }
  }
}
</script>
