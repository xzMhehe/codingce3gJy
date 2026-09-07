<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;通讯录</div>
    <div class="name">通讯录<br></div>
    <div class="module-content">
      设置你的 QQ、邮箱、手机号，方便好友找到你。<br>
    </div>

    <div class="module-title">QQ 号码</div>
    <div class="module-content">
      {{ ct.qq || '未设置' }}
      <a v-if="!ct.qq" href="javascript:;" @click="bindQQ = !bindQQ">[设置]</a>
      <template v-if="bindQQ">
        <form @submit.prevent="saveQQ">
          QQ:<input type="text" v-model.trim="qqForm.qq" maxlength="11" size="12" />
          密码:<input type="password" v-model.trim="qqForm.qq_pass" maxlength="20" size="12" />
          <input type="submit" value="绑定" />
        </form>
      </template>
    </div>

    <div class="module-title">电子邮箱</div>
    <div class="module-content">
      {{ ct.mail || '未设置' }}
      <a v-if="!ct.mail" href="javascript:;" @click="bindMail = !bindMail">[设置]</a>
      <template v-if="bindMail">
        <form @submit.prevent="save('mail')">
          <input type="text" v-model.trim="mailForm" maxlength="50" size="20" placeholder="you@example.com" />
          <input type="submit" value="保存" />
        </form>
      </template>
    </div>

    <div class="module-title">手机号码</div>
    <div class="module-content">
      {{ ct.phone || '未设置' }}
      <a v-if="!ct.phone" href="javascript:;" @click="bindPhone = !bindPhone">[设置]</a>
      <template v-if="bindPhone">
        <form @submit.prevent="save('phone')">
          <input type="text" v-model.trim="phoneForm" maxlength="11" size="12" placeholder="手机号" />
          <input type="submit" value="保存" />
        </form>
      </template>
    </div>

    <p v-if="tip" style="color:#e05a00;padding:3px 5px">{{ tip }}</p>
    <a href="javascript:;" @click="$router.push('/home')">返回家园</a><br>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Contacts',
  data () {
    return {
      ct: {}, tip: '',
      bindQQ: false, bindMail: false, bindPhone: false,
      qqForm: { qq: '', qq_pass: '' }, mailForm: '', phoneForm: ''
    }
  },
  mounted () { this.load() },
  methods: {
    load () { api.get('/contact').then(r => { if (r.code === 0) this.ct = r.data || {} }) },
    saveQQ () {
      api.post('/contact/qq', this.qqForm).then(r => {
        if (r.code === 0) { this.ct.qq = r.data.qq; this.bindQQ = false; this.tip = '绑定成功' }
        else this.tip = r.msg || '绑定失败'
      })
    },
    save (which) {
      const body = { qq: this.ct.qq, mail: this.ct.mail, phone: this.ct.phone }
      if (which === 'mail') body.mail = this.mailForm
      if (which === 'phone') body.phone = this.phoneForm
      api.post('/contact', body).then(r => {
        if (r.code === 0) { this.ct = r.data; this.bindMail = false; this.bindPhone = false; this.tip = '保存成功' }
        else this.tip = r.msg || '保存失败'
      })
    }
  }
}
</script>