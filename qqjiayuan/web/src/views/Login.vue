<template>
  <div>
    <div class="bar">【社区登录】<br></div>
    <div class="module-content">
      <form @submit.prevent="doLogin">
        家园社区账号:<br><input type="text" v-model.trim="name" maxlength="50" required><br>
        家园社区密码:<br><input type="password" v-model.trim="pass" value="" maxlength="50" required><br>
        <input type="submit" value="确定登录">
      </form>
      <p v-if="err" style="color:#c00">{{ err }}</p>
      <a href="javascript:;" @click="$router.push('/register')">免费注册</a>|<a href="javascript:;" @click="$router.push('/find')">靓号注册</a><br>
      <a href="javascript:;" @click="$router.push('/register')">免费注册</a>|<a href="javascript:;" @click="$router.push('/find')">找回资料</a><br>
      <a href="javascript:;" @click="$router.push('/nav')">注册声明</a>|<a href="javascript:;" @click="$router.push('/channel/4')">客服中心</a><br>
      ----------<br>
      3GQQ.CN(渝ICP备17001534号-2)<br>
      <span class="help-line">演示账号：站长 10000 / admin123；友友 10001~10005 / 123456</span>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Login',
  data () { return { name: '', pass: '', err: '' } },
  methods: {
    doLogin () {
      this.err = ''
      api.post('/auth/login', { name: this.name, password: this.pass }).then(r => {
        if (r.code === 0) {
          this.$store.commit('setUser', { token: r.data.token, user: r.data.user })
          // 再拉取完整资料（含权限）
          api.get('/auth/me').then(m => {
            if (m.code === 0) this.$store.commit('setUser', { user: m.data })
            this.$router.push(this.$route.query.redirect || '/')
          })
        } else {
          this.err = r.msg
        }
      })
    }
  }
}
</script>
