<template>
  <div>
    <div class="module-content">
      <img :src="$pic('2.png')" alt="logo"><br>
      欢迎来到家园社区，完善下面的信息注册家园账号，开启社区之旅吧！<a href="javascript:;">专属靓号注册</a><br>
      <form @submit.prevent="doReg">
        <div class="submit">
          <div class="item">
            <div class="text">您的昵称:</div>
            <input type="text" v-model.trim="form.nickname" maxlength="20" required>
          </div>
          <div class="item">
            <div class="text">您的性别:</div>
            <select v-model.number="form.gender">
              <option :value="1">小哥哥</option>
              <option :value="2">小姐姐</option>
            </select>
          </div>
          <div class="item">
            <div class="text">登录密码:</div>
            <input type="password" v-model.trim="form.password" maxlength="20" required>
          </div>
          <div class="item">
            <div class="text">确认密码:</div>
            <input type="password" v-model.trim="pass2" maxlength="20" required>
          </div>
          <button type="button" @click="doReg">提交注册</button>
        </div>
      </form>
      <p v-if="err" style="color:#c00">{{ err }}</p>
      <div v-if="okNo" class="deep" style="padding:8px;margin:6px 0">
        注册成功！你的家园号码是：<b style="color:#c00">{{ okNo }}</b><br>
        已送新人礼包100金币，<a href="javascript:;" @click="$router.push('/login')">快去登陆吧 &gt;&gt;</a>
      </div>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Register',
  data () {
    return { form: { nickname: '', gender: 1, password: '' }, pass2: '', err: '', okNo: '' }
  },
  methods: {
    doReg () {
      this.err = ''
      if (this.form.password !== this.pass2) {
        this.err = '两次输入的密码不一样哦'
        return
      }
      api.post('/auth/register', this.form).then(r => {
        if (r.code === 0) {
          this.okNo = r.data.username
        } else {
          this.err = r.msg
        }
      })
    }
  }
}
</script>
