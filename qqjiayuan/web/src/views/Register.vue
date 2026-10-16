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
            <div style="color:#999;font-size:12px;line-height:1.6">2-20个字符，可用 ♥✿★ 等特殊字符，禁止表情符号和纯数字</div>
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
    // ★ 2026-10-16 昵称卡控（与服务端 auth.go 同口径）：2-20 个字符；禁表情；禁纯数字；
    //   主流特殊字符（♥✿★ 等文本符号）不拦。返回错误文案，合法返回空串。
    validNickname (n) {
      const v = (n || '').trim()
      if (!v) return '请输入昵称'
      const len = Array.from(v).length
      if (len < 2) return '昵称至少2个字符'
      if (len > 20) return '昵称最多20个字符'
      // 表情：主流表情区块（1F000-1FAFF 含笑脸/符号/旗标等）+ 变体选择符/ZWJ 表情序列
      if (/[\u{1F000}-\u{1FAFF}\u{200D}\u{FE0F}\u{FE0E}]/u.test(v)) return '昵称不能包含表情符号'
      // 纯数字（含中间夹空白，如 "1 2"）也不允许
      if (/^[0-9\s]+$/.test(v)) return '昵称不能是纯数字'
      return ''
    },
    doReg () {
      this.err = ''
      const nickErr = this.validNickname(this.form.nickname)
      if (nickErr) {
        this.err = nickErr
        return
      }
      if (this.form.password !== this.pass2) {
        this.err = '两次输入的密码不一样哦'
        return
      }
      if (!this.form.captcha) {
        this.err = '请输入验证码'
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
