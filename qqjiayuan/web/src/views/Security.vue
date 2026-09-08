<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;安全中心</div>
    <div class="module-title">
      <a href="javascript:;" @click="$router.push('/box')">用户中心</a>|安全中心|<a href="javascript:;" @click="$router.push('/wallet')">我的钱包</a>|<a href="javascript:;" @click="$router.push('/rank')">家园排行</a><br>
    </div>

    <!-- 登录密码（诺哈 pass.asp） -->
    <div class="module-title">修改登录密码</div>
    <div class="module-content">
      <p>原密码：<input type="password" v-model.trim="oldPwd" size="16" maxlength="20"></p>
      <p>新密码：<input type="password" v-model.trim="newPwd" size="16" maxlength="20"></p>
      <p><button class="btn" @click="change">确认修改</button></p>
      <p style="color:#999">新密码 6-20 位，修改后请牢记。</p>
    </div>

    <!-- 支付密码（诺哈 wap_user_money.pass，独立于登录密码） -->
    <div class="module-title">支付密码 <span class="txt-fade">({{ hasPaypass ? '已设置' : '未设置' }})</span></div>
    <div class="module-content">
      <template v-if="hasPaypass">
        <p>原支付密码：<input type="password" v-model.trim="pay.old" size="16" maxlength="20"></p>
      </template>
      <p>新支付密码：<input type="password" v-model.trim="pay.new" size="16" maxlength="20"></p>
      <p><button class="btn" @click="savePaypass">{{ hasPaypass ? '修改支付密码' : '设置支付密码' }}</button></p>
      <p style="color:#999">支付密码用于商店购物、转账等消费确认，与登录密码相互独立。</p>
    </div>

    <!-- 密保问题（诺哈 protec.asp） -->
    <div class="module-title">密保问题 <span class="txt-fade">({{ hasProtection ? '已设置' : '未设置' }})</span></div>
    <div class="module-content">
      <p>问题：<select v-model.number="prot.issue">
        <option v-for="(q, i) in questions" :key="i" :value="i + 1">{{ q }}</option>
      </select></p>
      <p>答案：<input type="text" v-model.trim="prot.answer" size="16" maxlength="30"></p>
      <p><button class="btn" @click="saveProtection">保存密保</button></p>
    </div>

    <!-- 实名证件（诺哈 docu：设置需登录密码确认） -->
    <div class="module-title">实名证件 <span class="txt-fade">({{ doc.has_doc ? '已认证：' + doc.real_name + ' ' + doc.number : '未认证' }})</span></div>
    <div class="module-content">
      <p>证件类型：<select v-model.number="docForm.type"><option :value="1">身份证</option></select></p>
      <p>真实姓名：<input type="text" v-model.trim="docForm.real_name" size="16" maxlength="30"></p>
      <p>证件号码：<input type="text" v-model.trim="docForm.number" size="20" maxlength="30"></p>
      <p>登录密码：<input type="password" v-model.trim="docForm.password" size="16" maxlength="20"> <span class="txt-fade">(需确认)</span></p>
      <p><button class="btn" @click="saveDocument">保存证件</button></p>
    </div>

    <!-- 登录/操作日志（诺哈 log.asp） -->
    <div class="module-title">最近登录记录</div>
    <div class="list" v-if="logs.length">
      <div v-for="l in logs" :key="l.id" class="row">
        {{ l.action }} <span class="txt-fade">{{ l.ip }} · {{ fmt(l.created_at) }}</span><br>
      </div>
    </div>
    <div class="module-content" v-else><span class="empty">暂无记录</span></div>

    <p style="color:#c00;padding:0 5px" v-if="msg">{{ msg }}</p>
    <p style="color:#1a9e1a;padding:0 5px" v-if="okMsg">{{ okMsg }}</p>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Security',
  data () {
    return {
      oldPwd: '', newPwd: '', msg: '', okMsg: '',
      hasPaypass: false,
      pay: { old: '', new: '' },
      questions: [], hasProtection: false,
      prot: { issue: 1, answer: '' },
      doc: { has_doc: false, real_name: '', number: '' },
      docForm: { type: 1, real_name: '', number: '', password: '' },
      logs: []
    }
  },
  mounted () {
    api.get('/me/paypass').then(r => { if (r.code === 0) this.hasPaypass = r.data.has_paypass })
    api.get('/me/protection').then(r => { if (r.code === 0) { this.questions = r.data.questions || []; this.hasProtection = r.data.has_protection; this.prot.issue = r.data.issue || 1 } })
    api.get('/me/document').then(r => { if (r.code === 0) this.doc = r.data })
    api.get('/me/logs').then(r => { if (r.code === 0) this.logs = r.data.list || [] })
  },
  methods: {
    change () {
      this.msg = ''; this.okMsg = ''
      if (!this.oldPwd || !this.newPwd) { this.msg = '请填写原密码和新密码'; return }
      api.put('/auth/password', { old: this.oldPwd, new: this.newPwd }).then(r => {
        if (r.code === 0) { this.okMsg = '密码已修改'; this.oldPwd = ''; this.newPwd = '' } else this.msg = r.msg
      })
    },
    savePaypass () {
      this.msg = ''; this.okMsg = ''
      if (!this.pay.new) { this.msg = '请填写新支付密码'; return }
      if (this.hasPaypass && !this.pay.old) { this.msg = '请填写原支付密码'; return }
      api.post('/me/paypass', this.pay).then(r => {
        if (r.code === 0) { this.okMsg = '支付密码已保存'; this.hasPaypass = true; this.pay = { old: '', new: '' } } else this.msg = r.msg
      })
    },
    saveProtection () {
      this.msg = ''; this.okMsg = ''
      if (!this.prot.answer) { this.msg = '请填写密保答案'; return }
      api.post('/me/protection', this.prot).then(r => {
        if (r.code === 0) { this.okMsg = '密保已保存'; this.hasProtection = true; this.prot.answer = '' } else this.msg = r.msg
      })
    },
    saveDocument () {
      this.msg = ''; this.okMsg = ''
      if (!this.docForm.real_name || !this.docForm.number || !this.docForm.password) { this.msg = '请填写完整并输入登录密码确认'; return }
      api.post('/me/document', this.docForm).then(r => {
        if (r.code === 0) {
          this.okMsg = '证件已保存'
          this.docForm.password = ''
          api.get('/me/document').then(x => { if (x.code === 0) this.doc = x.data })
        } else this.msg = r.msg
      })
    },
    fmt (t) {
      if (!t) return ''
      const d = new Date(t); const p = n => (n < 10 ? '0' + n : '' + n)
      return d.getFullYear() + '-' + p(d.getMonth() + 1) + '-' + p(d.getDate()) + ' ' + p(d.getHours()) + ':' + p(d.getMinutes())
    }
  }
}
</script>