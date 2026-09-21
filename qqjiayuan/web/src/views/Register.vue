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
          <div class="item">
            <div class="text">验证码:</div>
            <input type="text" v-model.trim="form.captcha" maxlength="6" placeholder="输入计算结果" style="width:100px">
            <img v-if="captchaImg" :src="captchaImg" alt="验证码" title="点击切换" style="height:25px;vertical-align:middle;cursor:pointer" @click="loadCaptcha">
            <span style="color:red;font-size:12px;cursor:pointer" @click="loadCaptcha">看不清？点击切换</span>
          </div>
          <button type="button" @click="doReg">提交注册</button>
        </div>
      </form>
      <p v-if="err" style="color:#c00">{{ err }}</p>
      <div v-if="okNo" class="deep" style="padding:8px;margin:6px 0">
        注册成功！你的家园号码是：<b style="color:#c00">{{ okNo }}</b><br>
        已送新人礼包100G币，<a href="javascript:;" @click="$router.push('/login')">快去登陆吧 &gt;&gt;</a>
      </div>
      <div class="line"></div>
      <p>注册前请先阅读<a href="javascript:;" @click="$router.push('/kefu/terms')">《注册声明》</a>，注册即代表同意全部条款。</p>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Register',
  data () {
    return {
      form: { nickname: '', gender: 1, password: '', captcha_id: '', captcha: '' },
      pass2: '',
      captchaImg: '',
      err: '',
      okNo: ''
    }
  },
  created () {
    this.loadCaptcha()
  },
  methods: {
    // 拉一张新的算式验证码（点图片/点“看不清”换一张）
    loadCaptcha () {
      this.form.captcha = ''
      api.get('/auth/captcha').then(r => {
        if (r.code === 0) {
          this.captchaImg = r.data.img
          this.form.captcha_id = r.data.id
        } else {
          this.captchaImg = ''
          this.err = r.msg
        }
      })
    },
    doReg () {
      this.err = ''
      if (this.form.password !== this.pass2) {
        this.err = '两次输入的密码不一样哦'
        return
      }
      if (!this.form.captcha) {
        this.err = '请输入验证码（图片算式的计算结果）'
        return
      }
      api.post('/auth/register', this.form).then(r => {
        if (r.code === 0) {
          this.okNo = r.data.username
        } else {
          this.err = r.msg
          this.loadCaptcha() // 验证码一次性，失败后换一张
        }
      })
    }
  }
}
</script>
