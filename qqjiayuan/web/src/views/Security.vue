<template>
  <div>
    <div class="bar"><a href="javascript:;" @click="$router.push('/home')">家园</a>&gt;安全中心</div>
    <div class="note"></div>
    <div class="module-title">
      <a href="javascript:;" @click="$router.push('/box')">用户中心</a>|安全中心|<a href="javascript:;" @click="$router.push('/wallet')">我的钱包</a>|<a href="javascript:;" @click="$router.push('/rank')">家园排行</a><br>
    </div>
    <div class="module-title">安全中心</div>
    <div class="module-content">
      <p>修改密码：</p>
      <p>原密码：<input type="password" v-model.trim="oldPwd" size="16" maxlength="20"></p>
      <p>新密码：<input type="password" v-model.trim="newPwd" size="16" maxlength="20"></p>
      <p><button class="btn" @click="change">确认修改</button></p>
      <p style="color:#999">新密码 6-20 位，修改后请牢记。</p>
    </div>
    <p style="color:#c00;padding:0 5px" v-if="msg">{{ msg }}</p>
    <p style="color:#1a9e1a;padding:0 5px" v-if="okMsg">{{ okMsg }}</p>
  </div>
</template>

<script>
import api from '../api'

export default {
  name: 'Security',
  data () { return { oldPwd: '', newPwd: '', msg: '', okMsg: '' } },
  methods: {
    change () {
      this.msg = ''; this.okMsg = ''
      if (!this.oldPwd || !this.newPwd) { this.msg = '请填写原密码和新密码'; return }
      api.put('/auth/password', { old: this.oldPwd, new: this.newPwd }).then(r => {
        if (r.code === 0) { this.okMsg = '密码已修改'; this.oldPwd = ''; this.newPwd = '' } else this.msg = r.msg
      })
    }
  }
}
</script>
