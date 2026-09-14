<template>
  <div>
    <div class="bar">【找回资料】</div>
    <div class="module-content">
      <template v-if="act === ''">
        【找回资料】<br>
        1.<a href="javascript:;" @click="go('num')">忘记号码</a><br>
        2.<a href="javascript:;" @click="go('pass')">忘记密码</a><br>
      </template>

      <!-- 忘记号码（诺哈 find.asp act=1 → user.asp：凭联系方式/证件查号码） -->
      <template v-else-if="act === 'num'">
        【找回号码】<br>
        <div class="form-item">
          <label>ＱＱ号码:</label>
          <input type="text" v-model.trim="qq" maxlength="20">
          <button class="btn" type="button" @click="findByContact('qq')">确定提交</button>
        </div>
        <div class="form-item">
          <label>电子邮箱:</label>
          <input type="text" v-model.trim="mail" maxlength="50">
          <button class="btn" type="button" @click="findByContact('mail')">确定提交</button>
        </div>
        <div class="form-item">
          <label>手机号码:</label>
          <input type="text" v-model.trim="phone" maxlength="20">
          <button class="btn" type="button" @click="findByContact('phone')">确定提交</button>
        </div>
        <div class="form-item">
          <label>真实姓名:</label>
          <input type="text" v-model.trim="docName" maxlength="10">
        </div>
        <div class="form-item">
          <label>证件号码:</label>
          <input type="text" v-model.trim="docNumber" maxlength="30">
          <button class="btn" type="button" @click="findByDocument">确定提交</button>
        </div>
        <p v-if="numErr" style="color:#c00">{{ numErr }}</p>
        <div v-if="list.length" class="deep" style="border:1px solid #9FC6EC;padding:5px;margin:5px 0">
          <p v-for="u in list" :key="u.id">
            昵称 {{ u.nickname }} —— 家园号码 <b style="color:#c00">{{ u.username }}</b>（{{ u.created_at }} 注册）
          </p>
        </div>
        <p v-else-if="searched" class="empty">没有记录！</p>
      </template>

      <!-- 忘记密码（诺哈 find.asp act=2 → pass.asp：输入号码后四选一） -->
      <template v-else-if="act === 'pass'">
        【找回密码】<br>

        <template v-if="!ch">
          <div class="form-item">
            <label>会员号码:</label>
            <input type="text" v-model.trim="username" maxlength="10">
          </div>
          <div class="form-item"><button class="btn" type="button" @click="loadChannels">下一步</button></div>
          <p v-if="passErr" style="color:#c00">{{ passErr }}</p>
        </template>

        <template v-else>
          <template v-if="way === ''">
            会员号码：<b style="color:#c00">{{ username }}</b>（{{ ch.nickname }}）<br>
            1.<a href="javascript:;" @click="way = 'qq'">通过ＱＱ找回</a><br>
            2.<a href="javascript:;" @click="way = 'mail'">通过邮箱找回</a><br>
            3.<a href="javascript:;" @click="way = 'phone'">通过手机找回</a><br>
            4.<a href="javascript:;" @click="way = 'prot'">通过密保找回</a><br>
          </template>

          <template v-else-if="way === 'qq'">
            <template v-if="!ch.has_qq">
              该号码没有绑定QQ。<a href="javascript:;" @click="back">返回</a><br>
            </template>
            <template v-else>
              您绑定的QQ号码是:{{ ch.qq }}<br>
              <div class="form-item">
                <label>QQ号码:</label>
                <input type="text" v-model.trim="form.qq" maxlength="16">
              </div>
              <div class="form-item"><button class="btn" type="button" @click="doQQ">确定提交</button></div>
            </template>
          </template>

          <template v-else-if="way === 'mail'">
            <template v-if="!ch.has_mail">
              该号码没有绑定邮箱。<a href="javascript:;" @click="back">返回</a><br>
            </template>
            <template v-else>
              您绑定的邮箱是:{{ ch.mail }}<br>
              <div class="form-item">
                <label>您的邮箱:</label>
                <input type="text" v-model.trim="form.mail" maxlength="50">
              </div>
              <div class="form-item">
                <label>新密码:</label>
                <input type="password" v-model.trim="form.newPwd" maxlength="20">
              </div>
              <div class="form-item"><button class="btn" type="button" @click="doMail">确定提交</button></div>
              <p class="help-line">注：本演示站未接入真实邮件服务，邮箱校验通过后直接设置新密码。</p>
            </template>
          </template>

          <template v-else-if="way === 'phone'">
            <template v-if="!ch.has_phone">
              该号码没有绑定手机号码。<a href="javascript:;" @click="back">返回</a><br>
            </template>
            <template v-else>
              请使用手机号码{{ ch.phone }}发送短信：gm#{{ username }}#新密码 到站长手机号码，站长手机号码请联系<a href="javascript:;" @click="$router.push('/kefu')">客服</a>索取。<a href="javascript:;" @click="back">返回</a><br>
            </template>
          </template>

          <template v-else-if="way === 'prot'">
            <template v-if="!ch.has_protection">
              该号码没有设置密码保护。<a href="javascript:;" @click="back">返回</a><br>
            </template>
            <template v-else>
              密保问题:{{ ch.issue }}<br>
              <div class="form-item">
                <label>密保答案:</label>
                <input type="text" v-model.trim="form.answer" maxlength="30">
              </div>
              <div class="form-item"><button class="btn" type="button" @click="doProtection">确定提交</button></div>
            </template>
          </template>

          <p v-if="passErr" style="color:#c00">{{ passErr }}</p>
          <div v-if="okMsg" class="deep" style="border:1px solid #9FC6EC;padding:5px;margin:5px 0">
            <p v-if="okMsg.username">会员号码：<b style="color:#c00">{{ okMsg.username }}</b><br></p>
            <p v-if="okMsg.password">会员密码：<b style="color:#c00">{{ okMsg.password }}</b><br></p>
            <p>操作成功！系统已为你重置密码，请<a href="javascript:;" @click="$router.push('/login')">登录</a>后立即修改密码！</p>
          </div>
          <p v-if="okText" class="deep" style="border:1px solid #9FC6EC;padding:5px;margin:5px 0">{{ okText }}</p>
        </template>
      </template>

      ----------<br>
      <a href="javascript:;" @click="$router.push('/kefu')">客服中心</a>.<a href="javascript:;" @click="go('')">找回资料</a><br>
      <br>
    </div>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Find',
  data () {
    return {
      act: '',
      qq: '', mail: '', phone: '', docName: '', docNumber: '',
      list: [], searched: false, numErr: '',
      username: '', ch: null, way: '',
      form: { qq: '', mail: '', newPwd: '', answer: '' },
      passErr: '', okMsg: null, okText: ''
    }
  },
  watch: {
    '$route.query.act': {
      immediate: true,
      handler (v) {
        this.act = v === 'num' || v === 'pass' ? v : ''
        this.resetNum()
        this.resetPass()
      }
    }
  },
  methods: {
    go (act) {
      this.$router.push(act ? '/find?act=' + act : '/find')
    },
    resetNum () {
      this.qq = this.mail = this.phone = this.docName = this.docNumber = ''
      this.list = []
      this.searched = false
      this.numErr = ''
    },
    resetPass () {
      this.username = ''
      this.ch = null
      this.way = ''
      this.form = { qq: '', mail: '', newPwd: '', answer: '' }
      this.passErr = ''
      this.okMsg = null
      this.okText = ''
    },
    findByContact (field) {
      const params = {}
      if (field === 'qq') { if (!this.qq) { this.numErr = '请输入QQ号码'; return } params.qq = this.qq }
      if (field === 'mail') { if (!this.mail) { this.numErr = '请输入电子邮箱'; return } params.mail = this.mail }
      if (field === 'phone') { if (!this.phone) { this.numErr = '请输入手机号码'; return } params.phone = this.phone }
      this.numErr = ''
      api.get('/auth/find/contact', { params }).then(r => {
        this.searched = true
        if (r.code === 0) { this.list = r.data } else { this.list = []; this.numErr = r.msg }
      })
    },
    findByDocument () {
      if (!this.docName || !this.docNumber) { this.numErr = '请填写真实姓名和证件号码'; return }
      this.numErr = ''
      api.get('/auth/find/document', { params: { name: this.docName, number: this.docNumber } }).then(r => {
        this.searched = true
        if (r.code === 0) { this.list = r.data } else { this.list = []; this.numErr = r.msg }
      })
    },
    loadChannels () {
      this.passErr = ''
      if (!this.username) { this.passErr = '请输入会员号码'; return }
      api.get('/auth/repass/channels', { params: { username: this.username } }).then(r => {
        if (r.code === 0) { this.ch = r.data } else { this.passErr = r.msg }
      })
    },
    back () {
      this.way = ''
      this.passErr = ''
      this.okMsg = null
      this.okText = ''
    },
    doQQ () {
      this.passErr = ''
      if (!this.form.qq) { this.passErr = '请输入QQ号码'; return }
      api.post('/auth/repass/qq', { username: this.username, qq: this.form.qq }).then(r => {
        if (r.code === 0) { this.okMsg = r.data } else { this.passErr = r.msg }
      })
    },
    doMail () {
      this.passErr = ''
      if (!this.form.mail) { this.passErr = '请输入邮箱'; return }
      if (!this.form.newPwd || this.form.newPwd.length < 6) { this.passErr = '新密码需要6-20位'; return }
      api.post('/auth/repass/mail', { username: this.username, mail: this.form.mail, new_password: this.form.newPwd }).then(r => {
        if (r.code === 0) {
          this.okMsg = { username: this.username }
          this.okText = '密码已重置，请牢记并登录后立即修改！'
        } else { this.passErr = r.msg }
      })
    },
    doProtection () {
      this.passErr = ''
      if (!this.form.answer) { this.passErr = '请输入密保答案'; return }
      api.post('/auth/repass/protection', { username: this.username, answer: this.form.answer }).then(r => {
        if (r.code === 0) { this.okMsg = r.data } else { this.passErr = r.msg }
      })
    }
  }
}
</script>
